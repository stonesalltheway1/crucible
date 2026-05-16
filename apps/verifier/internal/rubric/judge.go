package rubric

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/crucible/verifier/internal/verification"
	"github.com/crucible/verifier/pkg/testreport"
)

// LLMClient is the vendor-neutral surface the Judge calls. The verifier
// daemon wires its own modelrouter.Client adapter into it; tests inject
// fakes.
//
// Crucially this interface is INTENTIONALLY MINIMAL — it accepts a
// system/user pair and a schema, returns the response body. It does NOT
// take any opaque "context" or "thinking" field because the rubric must
// never let the verifier model see the executor's thoughts.
type LLMClient interface {
	Vendor() string
	Model() string
	Call(ctx context.Context, req CallRequest) (CallResponse, error)
}

// CallRequest is the LLM-vendor-neutral envelope.
type CallRequest struct {
	System         string
	User           string
	Schema         map[string]any
	MaxOutput      int
	Temperature    float64
}

// CallResponse carries the model's bytes + usage.
type CallResponse struct {
	Body              []byte
	InputFreshTokens  int
	InputCachedTokens int
	OutputTokens      int
	CostUSD           float64
	ModelLatency      time.Duration
}

// Judge runs the rubric against an LLMClient. The Judge enforces the
// cross-family invariant at call time AND the no-reasoning audit on the
// rendered prompt.
type Judge struct {
	Client     LLMClient
	Criteria   []Criterion
	Threshold  float64
	MaxRetries int
	// VendorBlocklist is a defence-in-depth set: the Judge refuses to
	// call any vendor in this list. Wired from the executor's vendor.
	VendorBlocklist []string
}

// NewJudge returns a Judge with defaults.
func NewJudge(client LLMClient) *Judge {
	return &Judge{
		Client:     client,
		Criteria:   DefaultCriteria,
		Threshold:  DefaultThreshold,
		MaxRetries: 2,
	}
}

// ScoreErr signals the Judge couldn't produce a score after retries; the
// dispatcher converts this to a hard rejection.
type ScoreErr struct {
	Cause   string
	Attempts int
	Inner   error
}

func (e *ScoreErr) Error() string {
	return fmt.Sprintf("rubric: %s (%d attempts): %v", e.Cause, e.Attempts, e.Inner)
}

func (e *ScoreErr) Unwrap() error { return e.Inner }

// Score renders the prompt, calls the LLM with retry, parses the response,
// and folds the per-task signals into a final Score.
//
// Pre-call invariants:
//   - request.AuditNoLeakage() must pass (called by RenderPrompt)
//   - judge.Client.Vendor() must NOT be in VendorBlocklist
//   - judge.Client.Vendor() must NOT equal request.Routing.ExecutorVendor
func (j *Judge) Score(ctx context.Context, req *verification.VerificationRequest, reports []*testreport.TestReport) (Score, error) {
	if j.Client == nil {
		return Score{}, errors.New("rubric: nil LLM client")
	}
	vendor := strings.ToLower(j.Client.Vendor())
	exec := strings.ToLower(req.Routing.ExecutorVendor)
	if vendor != "" && exec != "" && vendor == exec {
		return Score{}, &verification.SameFamilyError{
			Executor: req.Routing.ExecutorVendor,
			Verifier: vendor,
		}
	}
	for _, blocked := range j.VendorBlocklist {
		if strings.EqualFold(blocked, vendor) {
			return Score{}, &verification.SameFamilyError{
				Executor: blocked,
				Verifier: vendor,
			}
		}
	}

	// Render — audits twice.
	body, err := RenderPrompt(PromptInput{
		Request:     req,
		TierReports: reports,
		Criteria:    j.Criteria,
	})
	if err != nil {
		return Score{}, err
	}

	// Up-front, deterministic rejections derived from trust signals.
	// We compute these FIRST so a tape-miss-blocked never even reaches
	// the model — the dispatcher gets a structured signal-driven
	// rejection instead.
	hard := hardRejections(req, reports)
	if len(hard) > 0 {
		// Surface a synthetic Score with zero rubric score; the
		// dispatcher records the reasons.
		return Score{
			Score:            0,
			Threshold:        j.Threshold,
			Passed:           false,
			RejectionReasons: hard,
			Confidence:       1.0, // deterministic signal — no LLM uncertainty
			JudgeModel:       j.Client.Model(),
			JudgeVendor:      j.Client.Vendor(),
			PromptHash:       body.Hash,
		}, nil
	}

	// Call the LLM with retry on parse failure.
	var last error
	for attempt := 0; attempt <= j.MaxRetries; attempt++ {
		resp, err := j.Client.Call(ctx, CallRequest{
			System:      body.System,
			User:        body.User,
			Schema:      body.Schema,
			MaxOutput:   1024,
			Temperature: 0.0,
		})
		if err != nil {
			last = err
			continue
		}
		s, err := ParseResponse(resp.Body, j.Criteria)
		if err != nil {
			last = err
			continue
		}
		s.JudgeModel = j.Client.Model()
		s.JudgeVendor = j.Client.Vendor()
		s.PromptHash = body.Hash
		s.InputFreshTokens = resp.InputFreshTokens
		s.InputCachedTokens = resp.InputCachedTokens
		s.ResponseTokens = resp.OutputTokens
		s.CostUSD = resp.CostUSD

		// Fold trust-signal warnings into RejectionReasons even when the
		// LLM passed — warnings stay, errors lower the score.
		s.RejectionReasons = append(s.RejectionReasons, trustSignalWarnings(req)...)

		// Recompute Passed in case warnings were added.
		s.Passed = s.Score >= s.Threshold && countErrors(s.RejectionReasons) == 0
		return s, nil
	}
	return Score{}, &ScoreErr{Cause: "no valid response", Attempts: j.MaxRetries + 1, Inner: last}
}

// hardRejections produces deterministic rejections from per-task trust
// signals. These bypass the LLM call entirely.
func hardRejections(req *verification.VerificationRequest, reports []*testreport.TestReport) []RejectionReason {
	var out []RejectionReason
	ts := req.PerTaskSignals

	// 1. miss-blocked tape disposition → re-plan
	if ts.TapeDispositionHistogram["miss-blocked"] > 0 {
		out = append(out, RejectionReason{
			Category: "tape_miss_blocked",
			Severity: "error",
			Detail:   "An egress request was refused (X-Crucible-Tape: miss-blocked). Promotion cannot proceed — re-plan to add the missing service or record the tape.",
			SuggestedFix: "Either record the missing endpoint in shadow-recorder, or amend the plan to avoid the service.",
		})
	}

	// 2. ≥3 stale-tape findings touching diff endpoints → reject
	stale := 0
	for _, f := range ts.StalenessFindings {
		if strings.EqualFold(f.Band, "stale") || strings.EqualFold(f.Band, "unrecorded") {
			stale++
		}
	}
	if stale >= 3 {
		out = append(out, RejectionReason{
			Category: "tape_stale",
			Severity: "error",
			Detail:   fmt.Sprintf("%d stale or unrecorded tape findings — verifier cannot trust the executor's claimed completion.", stale),
			SuggestedFix: "Operator re-records the affected endpoints before promotion.",
		})
	}

	// 3. PII-bearing tape entries without scrubber AuditLog → reject
	if ts.ScrubberAuditEntryCount > 0 && !ts.ScrubberFiredOnAllPII {
		out = append(out, RejectionReason{
			Category: "scrubber_missing_audit",
			Severity: "error",
			Detail:   "PII-bearing tape entries were used but the scrubber AuditLog did not fire on all of them.",
			SuggestedFix: "Re-run with CRUCIBLE_SCRUBBER_FAIL_CLOSED=1 and confirm the audit chain.",
		})
	}

	// 4. WASM quota trip — finding, not necessarily hard reject. Make
	// it a warn unless multiple kinds tripped.
	if len(ts.WasmQuotaTrips) >= 2 {
		var detail strings.Builder
		detail.WriteString("WASM tool runner tripped multiple quotas: ")
		for i, t := range ts.WasmQuotaTrips {
			if i > 0 {
				detail.WriteString(", ")
			}
			detail.WriteString(t.Quota)
		}
		out = append(out, RejectionReason{
			Category: "wasm_quota_trip",
			Severity: "error",
			Detail:   detail.String(),
			SuggestedFix: "Investigate the WASM tool that tripped its budget; verify the tool isn't escaping the sandbox.",
		})
	}

	// 5. Self-host advertised but unavailable — surface but do NOT
	// fail-open. The Phase-3 brief is explicit: "MUST propagate as
	// 'self-host unavailable' rather than fail-open".
	if !ts.SelfHostAvailable && ts.SelfHostUnavailableReason != "" {
		out = append(out, RejectionReason{
			Category: "self_host_unavailable",
			Severity: "warn",
			Detail:   "Self-host orchestrator unreachable: " + ts.SelfHostUnavailableReason,
			SuggestedFix: "If self-host is required for this tenant, halt promotion until the orchestrator is reachable.",
		})
	}

	// 6. Honest-CI bit-identical mismatch on Tier 4 → hard reject.
	for _, r := range reports {
		if r == nil || r.HonestCI == nil {
			continue
		}
		if !r.HonestCI.BitIdentical {
			out = append(out, RejectionReason{
				Category: "honest_ci_mismatch",
				Severity: "error",
				Detail:   fmt.Sprintf("Hermetic rebuild diverged from executor's: %s vs %s", r.HonestCI.ExecutorRebuildHash, r.HonestCI.VerifierRebuildHash),
				SuggestedFix: "Investigate non-determinism in the build (timestamps, $RANDOM, untracked deps).",
			})
		}
	}

	// 7. Tier 3 timeout with fallback engaged but no codeowner-review
	// flag set is a hard error — the brief mandates explicit CODEOWNER
	// review on fallback.
	for _, r := range reports {
		if r == nil || r.Proof == nil {
			continue
		}
		if r.Proof.TimedOut && r.Proof.FallbackTier != "" && !r.Proof.CodeownerReviewRequired {
			out = append(out, RejectionReason{
				Category: "tier3_fallback_missing_review",
				Severity: "error",
				Detail:   fmt.Sprintf("Tier 3 (%s) timed out; fallback %s engaged but codeowner_review_required is false.", r.Proof.Prover, r.Proof.FallbackTier),
				SuggestedFix: "Re-issue request with codeowner_review_required=true and a designated approver in CODEOWNERS.",
			})
		}
	}

	return out
}

// trustSignalWarnings produces warn-level rejection reasons that don't
// hard-reject but DO surface in PR comments.
func trustSignalWarnings(req *verification.VerificationRequest) []RejectionReason {
	var out []RejectionReason
	ts := req.PerTaskSignals
	if h := ts.TapeDispositionHistogram; h != nil {
		if h["synth-readonly"]+h["synth-mutation"]+h["synth-candidate"] > 0 {
			out = append(out, RejectionReason{
				Category: "synth_tape_used",
				Severity: "warn",
				Detail:   fmt.Sprintf("Synthetic tape responses were used (%d). Trust signal weights down.", h["synth-readonly"]+h["synth-mutation"]+h["synth-candidate"]),
			})
		}
		if h["live-passthrough"] > 0 {
			out = append(out, RejectionReason{
				Category: "live_passthrough",
				Severity: "warn",
				Detail:   fmt.Sprintf("%d live-passthrough responses were observed.", h["live-passthrough"]),
			})
		}
	}
	for _, f := range ts.StalenessFindings {
		if strings.EqualFold(f.Band, "aging") {
			out = append(out, RejectionReason{
				Category: "tape_aging",
				Severity: "info",
				Detail:   fmt.Sprintf("Tape for %s is aging (age=%ds).", f.Endpoint, f.AgeSeconds),
			})
		}
	}
	if len(ts.WasmQuotaTrips) == 1 {
		t := ts.WasmQuotaTrips[0]
		out = append(out, RejectionReason{
			Category: "wasm_quota_trip",
			Severity: "warn",
			Detail:   fmt.Sprintf("WASM tool tripped quota=%s", t.Quota),
		})
	}
	return out
}

func countErrors(reasons []RejectionReason) int {
	n := 0
	for _, r := range reasons {
		if r.Severity == "error" {
			n++
		}
	}
	return n
}
