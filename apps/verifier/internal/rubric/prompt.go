// Package rubric implements the cross-family LLM-judge that converts a
// VerificationRequest + the per-tier TestReports into the final
// `rubric_score ∈ [0, 1]` plus structured rejection reasons.
//
// Three invariants the package enforces (no exception path lets these
// slip):
//
//  1. The verifier model MUST be from a DIFFERENT vendor lineage than the
//     executor. Same-family pairings are refused at PreparePrompt time
//     and again at Call time — defence in depth.
//
//  2. The prompt MUST NOT contain executor reasoning. Each prompt-render
//     pass routes through verification.AuditNoLeakage to reject any field
//     whose name matches the reasoning denylist (chain_of_thought,
//     scratchpad, agent_trace, etc.).
//
//  3. The rubric MUST consult Phase-3 trust signals:
//       - tape staleness findings  (down-weight)
//       - X-Crucible-Tape disposition histogram  (down-weight synth-* /
//         live-passthrough; reject on miss-blocked)
//       - scrubber AuditLog entries  (require ≥1 per PII tape)
//       - WASM ExecutionReport.usage.trip  (any non-None = finding)
//
//  4. Self-host unavailable is propagated as a signal, NOT failed-open.
//
// The rubric prompt itself is schema-constrained: the model returns a
// strict JSON object that the package decodes; non-conforming output
// triggers a retry (max 2) and then a hard rejection.
package rubric

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/crucible/verifier/internal/verification"
	"github.com/crucible/verifier/pkg/testreport"
)

// Score is the rubric output.
type Score struct {
	Score             float64           `json:"score"`            // 0..1
	Threshold         float64           `json:"threshold"`        // 0..1
	Passed            bool              `json:"passed"`           // Score >= Threshold
	Subscores         map[string]float64 `json:"subscores"`        // per-criterion
	RejectionReasons  []RejectionReason `json:"rejection_reasons,omitempty"`
	Confidence        float64           `json:"confidence"`       // 0..1
	JudgeModel        string            `json:"judge_model"`
	JudgeVendor       string            `json:"judge_vendor"`
	PromptHash        string            `json:"prompt_hash"`
	ResponseTokens    int               `json:"response_tokens,omitempty"`
	InputFreshTokens  int               `json:"input_fresh_tokens,omitempty"`
	InputCachedTokens int               `json:"input_cached_tokens,omitempty"`
	CostUSD           float64           `json:"cost_usd,omitempty"`
}

// RejectionReason mirrors the proto VerifierRejection.RejectionReason.
// Structured so the executor can reflect-and-retry.
type RejectionReason struct {
	Category     string `json:"category"`
	Severity     string `json:"severity"`     // "info" | "warn" | "error"
	Detail       string `json:"detail"`
	File         string `json:"file,omitempty"`
	Line         int    `json:"line,omitempty"`
	SuggestedFix string `json:"suggested_fix,omitempty"`
}

// Criterion is one of the rubric axes the LLM-judge scores.
type Criterion struct {
	Name       string  `json:"name"`
	Weight     float64 `json:"weight"`
	Definition string  `json:"definition"`
}

// DefaultCriteria are the six axes the rubric weighs. Weights sum to 1.0.
var DefaultCriteria = []Criterion{
	{Name: "diff_correctness", Weight: 0.30, Definition: "Does the diff implement the intended behavior change? Is logic sound?"},
	{Name: "test_adequacy", Weight: 0.20, Definition: "Do the tests exercise the new code paths? Do they assert non-trivial properties?"},
	{Name: "spec_consistency", Weight: 0.15, Definition: "Are spec changes consistent with the diff? Are OpenAPI / GraphQL deltas reflected?"},
	{Name: "robustness", Weight: 0.15, Definition: "Are edge cases handled (nil, empty, overflow)? Is error handling structured?"},
	{Name: "security_posture", Weight: 0.10, Definition: "Does the diff introduce auth/authz/crypto/data-flow risks?"},
	{Name: "trust_signal_alignment", Weight: 0.10, Definition: "Are tape-staleness, scrubber-audit, sandbox-trip signals consistent with claimed completion?"},
}

// DefaultThreshold is the score threshold for approval per the brief.
const DefaultThreshold = 0.85

// PromptInput is the structured input to RenderPrompt.
type PromptInput struct {
	Request       *verification.VerificationRequest
	TierReports   []*testreport.TestReport
	Criteria      []Criterion
}

// PromptHashOnly returns the canonical hash of the prompt body without
// invoking the model. Used for cache keys.
func PromptHashOnly(in PromptInput) (string, error) {
	body, err := RenderPrompt(in)
	if err != nil {
		return "", err
	}
	return body.Hash, nil
}

// PromptBody is the rendered prompt + audit metadata.
type PromptBody struct {
	System      string
	User        string
	Schema      map[string]any
	Hash        string
	AuditedAt   int64
}

// RenderPrompt assembles the system + user blocks and the response schema.
// AUDIT-CRITICAL: this function calls AuditNoLeakage on the final
// payload before returning. Any reasoning leak fails the request HERE,
// not at the model call.
func RenderPrompt(in PromptInput) (PromptBody, error) {
	if in.Request == nil {
		return PromptBody{}, errors.New("rubric: nil request")
	}
	if err := in.Request.AuditNoLeakage(); err != nil {
		return PromptBody{}, fmt.Errorf("rubric: pre-audit failed: %w", err)
	}
	criteria := in.Criteria
	if len(criteria) == 0 {
		criteria = DefaultCriteria
	}

	system := buildSystem(criteria, in.Request)
	user := buildUser(in.Request, in.TierReports)
	schema := responseSchema(criteria)

	// Post-audit on the assembled string. The audit guard scans the
	// User body for any of the denylist tokens; we treat the rendered
	// markdown as a single field of the leakage scan.
	audit := map[string]any{
		"system": system,
		"user":   user,
	}
	if err := verification.AuditNoLeakage(audit); err != nil {
		return PromptBody{}, fmt.Errorf("rubric: post-render leak detected: %w", err)
	}

	h := hashStrings(system, user)
	return PromptBody{
		System: system,
		User:   user,
		Schema: schema,
		Hash:   h,
	}, nil
}

func buildSystem(criteria []Criterion, req *verification.VerificationRequest) string {
	var b strings.Builder
	b.WriteString(`You are Crucible's verifier. You audit an AI-authored code diff and the per-tier test reports the verifier pipeline produced for it. Your goal is to decide whether the change should be approved for promotion.

You MUST NOT use any information about the executor agent's internal reasoning. You see ONLY the diff, the test files, the spec changes, the per-tier reports, and the trust signals listed below. The executor's chain-of-thought is intentionally not provided.

Score the change on each criterion in [0.0, 1.0]:

`)
	for _, c := range criteria {
		fmt.Fprintf(&b, "  - %s (weight %.2f): %s\n", c.Name, c.Weight, c.Definition)
	}
	b.WriteString(`
Trust signals you MUST factor into the trust_signal_alignment criterion:
  - Tape staleness: any "stale" or "unrecorded" finding lowers the score; the rubric should reject the change when ≥3 stale findings touch endpoints in the diff.
  - X-Crucible-Tape disposition: "miss-blocked" disposition means the executor's request was refused — the change cannot be trusted; if any tape interaction returned miss-blocked, REJECT and recommend re-plan.
  - "synth-*" or "live-passthrough" dispositions weight the trust signal lower; flag for explicit human review.
  - Scrubber AuditLog: if PII-bearing tapes exist but ScrubberFiredOnAllPII is false, REJECT.
  - WASM quota trips: any non-empty trip list is a finding (Severity: warn).
  - Self-host unavailability: propagate as "self-host unavailable" rejection reason; do NOT treat as approval.

Cross-family invariant: your output explicitly identifies your model family. Calling vendor: `)
	b.WriteString(req.Routing.VerifierVendor)
	b.WriteString(`. Executor vendor: `)
	b.WriteString(req.Routing.ExecutorVendor)
	b.WriteString(`. These MUST differ; if they match, refuse to score and return a hard error.

Respond ONLY with the JSON object described in the response schema. No prose; no markdown fences; no explanation outside the JSON.`)
	return b.String()
}

func buildUser(req *verification.VerificationRequest, reports []*testreport.TestReport) string {
	var b strings.Builder
	fmt.Fprintf(&b, "TASK_ID: %s\nTENANT: %s\nBASE_SHA: %s\nEXECUTOR_MODEL: %s\nVERIFIER_MODEL: %s\n\n",
		req.TaskID, req.TenantID, req.BaseSHA, req.Routing.ExecutorModel, req.Routing.VerifierModel)

	// 1. Diff summary
	b.WriteString("=== DIFF SUMMARY ===\n")
	for _, f := range req.Diff.Files {
		fmt.Fprintf(&b, "  %s  %s  (%d bytes)\n", f.Action, f.Path, f.SizeBytes)
	}

	// 2. Critical-path scores
	if len(req.CriticalPathScores) > 0 {
		b.WriteString("\n=== CRITICAL-PATH SCORES ===\n")
		scores := append([]verification.CriticalPathScore{}, req.CriticalPathScores...)
		sort.Slice(scores, func(i, j int) bool { return scores[i].Score > scores[j].Score })
		for _, s := range scores {
			fmt.Fprintf(&b, "  %s  S=%.1f  band=%s\n", s.File, s.Score, s.Band)
		}
	}

	// 3. Spec changes
	if len(req.SpecChanges) > 0 {
		b.WriteString("\n=== SPEC CHANGES ===\n")
		for _, sc := range req.SpecChanges {
			fmt.Fprintf(&b, "  %s (%s)\n    prev=%s\n    curr=%s\n",
				sc.Path, sc.Kind, sc.PreviousHash, sc.CurrentHash)
		}
	}

	// 4. Per-tier reports
	b.WriteString("\n=== TIER REPORTS ===\n")
	for _, r := range reports {
		if r == nil {
			continue
		}
		fmt.Fprintf(&b, "\n  [%s/%s] framework=%s passed=%t verdict=%s duration=%.1fs\n",
			r.Tier, r.Language, r.Framework, r.Passed, r.Verdict, r.DurationSeconds)
		switch {
		case r.Mutation != nil:
			fmt.Fprintf(&b, "    Mutation: killed=%d survived=%d score=%.3f threshold=%.3f\n",
				r.Mutation.Killed, r.Mutation.Survived, r.Mutation.Score, r.Mutation.Threshold)
			for _, m := range r.Mutation.SurvivedSummary {
				fmt.Fprintf(&b, "      survived %s:%d %s\n", m.File, m.Line, m.Mutator)
			}
		case r.PBT != nil:
			fmt.Fprintf(&b, "    PBT: iterations=%d, counterexamples=%d\n",
				r.PBT.Iterations, len(r.PBT.Counterexamples))
			for _, c := range r.PBT.Counterexamples {
				fmt.Fprintf(&b, "      counterexample[%s]: %s\n", c.Property, c.Shrunk)
			}
		case r.Contract != nil:
			fmt.Fprintf(&b, "    Contract: spec=%s violations=%d dst_iters=%d\n",
				r.Contract.SpecPath, len(r.Contract.Violations), r.Contract.DstIterations)
		case r.Proof != nil:
			fmt.Fprintf(&b, "    Proof: prover=%s discharged=%d/%d timed_out=%t fallback=%s\n",
				r.Proof.Prover, r.Proof.Discharged, r.Proof.Obligations, r.Proof.TimedOut, r.Proof.FallbackTier)
		case r.HonestCI != nil:
			fmt.Fprintf(&b, "    HonestCI: bit_identical=%t SLSA=%d scrubber_audit_ok=%t\n",
				r.HonestCI.BitIdentical, r.HonestCI.SLSALevel, r.HonestCI.ScrubberAuditOK)
		}
		for _, f := range r.Findings {
			fmt.Fprintf(&b, "    finding [%s/%s] %s:%d  %s\n",
				f.Category, f.Severity, f.File, f.Line, f.Detail)
		}
	}

	// 5. Trust signals
	b.WriteString("\n=== TRUST SIGNALS ===\n")
	ts := req.PerTaskSignals
	if len(ts.StalenessFindings) > 0 {
		b.WriteString("  StalenessFindings:\n")
		for _, sf := range ts.StalenessFindings {
			fmt.Fprintf(&b, "    %s band=%s age=%ds  %s\n", sf.Endpoint, sf.Band, sf.AgeSeconds, sf.Message)
		}
	}
	if len(ts.TapeDispositionHistogram) > 0 {
		b.WriteString("  TapeDispositionHistogram:\n")
		keys := make([]string, 0, len(ts.TapeDispositionHistogram))
		for k := range ts.TapeDispositionHistogram {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			fmt.Fprintf(&b, "    %-20s %d\n", k, ts.TapeDispositionHistogram[k])
		}
	}
	fmt.Fprintf(&b, "  ScrubberFiredOnAllPII: %t (entries=%d)\n",
		ts.ScrubberFiredOnAllPII, ts.ScrubberAuditEntryCount)
	if len(ts.WasmQuotaTrips) > 0 {
		b.WriteString("  WasmQuotaTrips:\n")
		for _, t := range ts.WasmQuotaTrips {
			fmt.Fprintf(&b, "    quota=%s tripped_at=%d  %s\n", t.Quota, t.TrippedAt, t.Detail)
		}
	}
	fmt.Fprintf(&b, "  SelfHostAvailable: %t  (reason=%q)\n", ts.SelfHostAvailable, ts.SelfHostUnavailableReason)

	return b.String()
}

// responseSchema is the JSON schema we constrain decoding to. Both
// Anthropic and Google support `response_format`/`responseSchema` so we
// can submit this directly.
func responseSchema(criteria []Criterion) map[string]any {
	props := map[string]any{
		"score":      map[string]any{"type": "number", "minimum": 0, "maximum": 1},
		"threshold":  map[string]any{"type": "number", "minimum": 0, "maximum": 1},
		"passed":     map[string]any{"type": "boolean"},
		"confidence": map[string]any{"type": "number", "minimum": 0, "maximum": 1},
		"subscores": map[string]any{
			"type":                 "object",
			"additionalProperties": map[string]any{"type": "number", "minimum": 0, "maximum": 1},
		},
		"rejection_reasons": map[string]any{
			"type": "array",
			"items": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"category": map[string]any{"type": "string"},
					"severity": map[string]any{"type": "string", "enum": []string{"info", "warn", "error"}},
					"detail":   map[string]any{"type": "string"},
					"file":     map[string]any{"type": "string"},
					"line":     map[string]any{"type": "integer"},
					"suggested_fix": map[string]any{"type": "string"},
				},
				"required": []string{"category", "severity", "detail"},
				"additionalProperties": false,
			},
		},
	}
	required := []string{"score", "passed", "confidence", "subscores", "rejection_reasons"}
	return map[string]any{
		"type":                 "object",
		"properties":           props,
		"required":             required,
		"additionalProperties": false,
	}
}

// hashStrings returns sha256(concatenation) hex.
func hashStrings(parts ...string) string {
	h := newHasher()
	for _, p := range parts {
		h.Write([]byte(p))
		h.Write([]byte{0})
	}
	return h.Hex()
}

// ParseResponse decodes the rubric model's JSON response. Returns an
// error if the body isn't valid structured output; the caller retries
// per the dispatcher policy.
func ParseResponse(body []byte, criteria []Criterion) (Score, error) {
	body = stripFences(body)
	var s Score
	if err := json.Unmarshal(body, &s); err != nil {
		return Score{}, fmt.Errorf("rubric: parse: %w (body=%q)", err, truncate(body, 256))
	}
	if s.Score < 0 || s.Score > 1 {
		return Score{}, fmt.Errorf("rubric: score %v out of [0,1]", s.Score)
	}
	if s.Threshold == 0 {
		s.Threshold = DefaultThreshold
	}
	s.Passed = s.Score >= s.Threshold && len(s.RejectionReasons) == 0
	// Verify subscores conform to criteria — extra keys allowed, missing
	// is fine, but every present key must be a known criterion.
	if len(s.Subscores) > 0 && len(criteria) > 0 {
		valid := map[string]bool{}
		for _, c := range criteria {
			valid[c.Name] = true
		}
		for k := range s.Subscores {
			if !valid[k] {
				return Score{}, fmt.Errorf("rubric: subscore %q not in criteria set", k)
			}
		}
	}
	return s, nil
}

func stripFences(b []byte) []byte {
	s := strings.TrimSpace(string(b))
	if strings.HasPrefix(s, "```") {
		s = strings.TrimPrefix(s, "```json")
		s = strings.TrimPrefix(s, "```")
		s = strings.TrimSuffix(s, "```")
		s = strings.TrimSpace(s)
	}
	return []byte(s)
}

func truncate(b []byte, n int) string {
	if len(b) <= n {
		return string(b)
	}
	return string(b[:n]) + "…"
}
