// Package dispatcher is the verifier daemon's tier-selection state
// machine. It receives a VerificationRequest, asks the critical-path
// classifier for the appropriate tier ladder, fans out per-language
// runners through the process pool, runs the rubric, and assembles the
// final VerifierApproval or VerifierRejection.
//
// State machine:
//
//   intake ─► classify ─► select tiers ─► fan-out runners ─►
//     wait (with per-tier wall-clock budgets) ─► tier3 fallback (if any) ─►
//     rubric ─► sign ─► emit
//
// Each transition emits a DispatchEvent into VerificationResponse.DispatchTrace
// so the dashboard can render lifecycle. The dispatcher's own actions are
// idempotent on TaskID; replays are safe (the deduper at the pool level
// catches dupes).
package dispatcher

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	cruciblev1 "github.com/crucible/sdk-go/crucible/v1"
	"github.com/crucible/verifier/internal/criticalpath"
	"github.com/crucible/verifier/internal/rubric"
	"github.com/crucible/verifier/internal/verification"
	"github.com/crucible/verifier/pkg/testreport"
)

// RunnerKind names the per-language tier runner.
type RunnerKind struct {
	Language testreport.Language
	Tier     testreport.Tier
}

func (k RunnerKind) String() string {
	return fmt.Sprintf("%s/%s", k.Language, k.Tier)
}

// RunnerOutput is what a per-language runner returns from one tier.
type RunnerOutput struct {
	Report *testreport.TestReport
	Err    error
}

// Runner is the abstract per-language runner interface. Production
// runners shell out to the per-language CLI inside the verifier sandbox;
// tests inject fakes.
type Runner interface {
	Run(ctx context.Context, kind RunnerKind, req *verification.VerificationRequest) RunnerOutput
}

// Pool is the abstract process pool the dispatcher dispatches into.
type Pool interface {
	Submit(ctx context.Context, kind RunnerKind, req *verification.VerificationRequest) (*testreport.TestReport, error)
	Health() error
}

// Tier3Adapter is the Tier 3 (formal verification) entry point. We
// segregate it from Pool because Tier 3 has its own dispatcher with
// per-prover state, caching, and timeout/fallback policy.
type Tier3Adapter interface {
	// Discharge attempts the proof. On timeout, returns a report with
	// Proof.TimedOut=true AND FallbackTier="tier_2_5".
	Discharge(ctx context.Context, req *verification.VerificationRequest, prover string) (*testreport.TestReport, error)
}

// Tier4Adapter runs the hermetic-rebuild + SLSA-L3 attestation.
type Tier4Adapter interface {
	Verify(ctx context.Context, req *verification.VerificationRequest) (*testreport.TestReport, error)
}

// Dispatcher orchestrates per-tier dispatch.
type Dispatcher struct {
	Pool       Pool
	Tier3      Tier3Adapter
	Tier4      Tier4Adapter
	Rubric     *rubric.Judge
	Classifier *criticalpath.Classifier

	// MemoryFeaturizer plugs the Phase-5 memory-layer compliance signal
	// into the trust_signal_alignment criterion. Optional — if nil, the
	// rubric runs unchanged (Phase 4 behaviour).
	MemoryFeaturizer *rubric.MemoryComplianceFeaturizer

	// TierBudgets bounds per-tier wall-clock. Defaults from
	// docs/01-architecture/verifier-pipeline.md.
	TierBudgets TierBudgets

	// Now is injectable for deterministic tests.
	Now func() time.Time
}

// TierBudgets defines the per-tier wall-clock budgets.
type TierBudgets struct {
	Tier0 time.Duration
	Tier1 time.Duration
	Tier2 time.Duration
	Tier3 time.Duration
	Tier4 time.Duration
}

// DefaultBudgets matches the verifier-pipeline doc.
var DefaultBudgets = TierBudgets{
	Tier0: 2 * time.Minute,
	Tier1: 15 * time.Minute,
	Tier2: 45 * time.Minute,
	Tier3: 10 * time.Minute, // Dafny default; Lean 30, TLA+ 20 — tier3-adapter handles per-prover.
	Tier4: 30 * time.Minute,
}

// New returns a Dispatcher with default budgets.
func New(pool Pool, t3 Tier3Adapter, t4 Tier4Adapter, judge *rubric.Judge, cls *criticalpath.Classifier) *Dispatcher {
	return &Dispatcher{
		Pool:        pool,
		Tier3:       t3,
		Tier4:       t4,
		Rubric:      judge,
		Classifier:  cls,
		TierBudgets: DefaultBudgets,
		Now:         time.Now,
	}
}

// Dispatch is the top-level entry. Returns the structured response
// suitable to ship to the control plane.
func (d *Dispatcher) Dispatch(ctx context.Context, req *verification.VerificationRequest) (*verification.VerificationResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	if err := req.AuditNoLeakage(); err != nil {
		return nil, err
	}
	if d.Rubric == nil {
		return nil, errors.New("dispatcher: nil rubric judge")
	}

	resp := &verification.VerificationResponse{
		CostBreakdown: verification.CostBreakdown{
			RunnerSecondsByTier: map[string]float64{},
		},
	}

	// 1. Tier selection from critical-path scores in the request.
	tiers := selectTiers(req)
	appendEventInPlace(&resp.DispatchTrace, d.Now(), "intake", "all", "", fmt.Sprintf("languages=%v tiers=%v", req.Languages, tiers))

	// 2. Fan out tiers + runners.
	reports := d.fanOut(ctx, req, tiers, resp)

	// 3. Tier-3 fallback wiring — if a Tier 3 report came back with
	// TimedOut=true and FallbackTier=tier_2_5, ensure the verifier
	// flagged codeowner-review.
	d.applyTier3Fallback(reports, resp)

	// 4. Rubric.
	score, err := d.Rubric.Score(ctx, req, reports)
	if err != nil {
		return nil, err
	}
	resp.CostBreakdown.RubricUSD = score.CostUSD
	resp.CostBreakdown.TotalUSD = score.CostUSD
	appendEventInPlace(&resp.DispatchTrace, d.Now(), "rubric", "rubric", "",
		fmt.Sprintf("score=%.3f passed=%v reasons=%d", score.Score, score.Passed, len(score.RejectionReasons)))

	// 4b. Phase-5 memory-layer compliance signal — folded into the
	// trust_signal_alignment criterion. Failure here is non-fatal; the
	// Phase-4 trust signals carry the load when memory is unavailable.
	if d.MemoryFeaturizer != nil {
		feats, ferr := d.MemoryFeaturizer.Featurize(ctx, rubric.ComplianceRequest{
			TenantID: req.TenantID,
			TaskID:   req.TaskID,
			Diff:     req.Diff,
		})
		if ferr != nil {
			appendEventInPlace(&resp.DispatchTrace, d.Now(), "memory_compliance", "rubric", "",
				fmt.Sprintf("featurizer error (non-fatal): %v", ferr))
		} else if feats.ConventionsChecked > 0 || feats.WarnViolations > 0 || feats.ErrorViolations > 0 {
			rubric.ApplyToScore(&score, feats)
			appendEventInPlace(&resp.DispatchTrace, d.Now(), "memory_compliance", "rubric", "",
				fmt.Sprintf("conventions_checked=%d warn=%d err=%d trust_delta=%.3f adjusted_score=%.3f",
					feats.ConventionsChecked, feats.WarnViolations, feats.ErrorViolations,
					feats.TrustSignalDelta, score.Score))
		}
	}

	// 5. Compose Approval / Rejection.
	tierResults := composeTierResults(reports)
	if score.Passed {
		resp.Approval = &cruciblev1.VerifierApproval{
			TaskID:               req.TaskID,
			DiffHash:             diffHash(req),
			Verdict:              "approved",
			RubricScore:          score.Score,
			TierResults:          tierResults,
			ExecutorOidcSubject:  "", // filled by emit-layer post-sign
			VerifierOidcSubject:  "",
			ExecutorModel:        req.Routing.ExecutorModel,
			VerifierModel:        req.Routing.VerifierModel,
			SignedAt:             d.Now().UTC(),
		}
	} else {
		reasons := make([]cruciblev1.RejectionReason, 0, len(score.RejectionReasons))
		for _, r := range score.RejectionReasons {
			reasons = append(reasons, cruciblev1.RejectionReason{
				Category:     r.Category,
				Detail:       r.Detail,
				File:         r.File,
				Line:         uint32(r.Line),
				SuggestedFix: r.SuggestedFix,
			})
		}
		resp.Rejection = &cruciblev1.VerifierRejection{
			TaskID:              req.TaskID,
			DiffHash:            diffHash(req),
			Verdict:             "rejected",
			RejectionReasons:    reasons,
			TierResults:         tierResults,
			ExecutorOidcSubject: "",
			VerifierOidcSubject: "",
			SignedAt:            d.Now().UTC(),
		}
	}
	return resp, nil
}

// traceMu serialises DispatchTrace appends across goroutines.
// We use a single package-level mutex because the trace is the only
// shared mutable state during fanOut.
var traceMu sync.Mutex

// fanOut launches all runners for a tier set in parallel; collects results.
func (d *Dispatcher) fanOut(ctx context.Context, req *verification.VerificationRequest, tiers []testreport.Tier, resp *verification.VerificationResponse) []*testreport.TestReport {
	var mu sync.Mutex
	out := make([]*testreport.TestReport, 0, len(tiers)*len(req.Languages))

	var wg sync.WaitGroup
	for _, tier := range tiers {
		for _, langStr := range req.Languages {
			kind := RunnerKind{
				Language: testreport.Language(langStr),
				Tier:     tier,
			}
			wg.Add(1)
			go func(k RunnerKind) {
				defer wg.Done()
				report := d.runOne(ctx, req, k, resp)
				if report == nil {
					return
				}
				mu.Lock()
				out = append(out, report)
				mu.Unlock()
			}(kind)
		}
	}
	wg.Wait()

	// Tier 4 honest-CI is polyglot; runs once per request regardless of
	// language fan-out.
	if hasTier(tiers, testreport.TierHonestCI) && d.Tier4 != nil {
		report := d.runTier4(ctx, req, resp)
		if report != nil {
			mu.Lock()
			out = append(out, report)
			mu.Unlock()
		}
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].Tier != out[j].Tier {
			return out[i].Tier < out[j].Tier
		}
		return out[i].Language < out[j].Language
	})
	return out
}

func (d *Dispatcher) runOne(ctx context.Context, req *verification.VerificationRequest, kind RunnerKind, resp *verification.VerificationResponse) *testreport.TestReport {
	// Skip tier-4 here — it's polyglot.
	if kind.Tier == testreport.TierHonestCI {
		return nil
	}
	appendEventInPlace(&resp.DispatchTrace, d.Now(), "dispatched", string(kind.Tier), string(kind.Language), "")

	var report *testreport.TestReport
	var err error

	switch kind.Tier {
	case testreport.TierProof:
		if d.Tier3 == nil {
			return nil
		}
		prover := chooseProverForLanguage(kind.Language)
		bctx, cancel := withBudget(ctx, d.tierBudget(testreport.TierProof))
		defer cancel()
		report, err = d.Tier3.Discharge(bctx, req, prover)
	default:
		if d.Pool == nil {
			return nil
		}
		bctx, cancel := withBudget(ctx, d.tierBudget(kind.Tier))
		defer cancel()
		report, err = d.Pool.Submit(bctx, kind, req)
	}
	if err != nil {
		appendEventInPlace(&resp.DispatchTrace, d.Now(), "failed", string(kind.Tier), string(kind.Language), err.Error())
		return &testreport.TestReport{
			SchemaVersion: testreport.SchemaVersion,
			TaskID:        req.TaskID,
			Tier:          kind.Tier,
			Language:      kind.Language,
			Verdict:       testreport.VerdictFailed,
			Passed:        false,
			Error:         err.Error(),
		}
	}
	if report != nil {
		resp.CostBreakdown.RunnerSecondsByTier[string(kind.Tier)] += report.DurationSeconds
		phase := "passed"
		if !report.Passed {
			phase = "failed"
		}
		if report.Verdict == testreport.VerdictTimedOut {
			phase = "timed_out"
		}
		appendEventInPlace(&resp.DispatchTrace, d.Now(), phase, string(kind.Tier), string(kind.Language), "")
	}
	return report
}

func (d *Dispatcher) runTier4(ctx context.Context, req *verification.VerificationRequest, resp *verification.VerificationResponse) *testreport.TestReport {
	appendEventInPlace(&resp.DispatchTrace, d.Now(), "dispatched", string(testreport.TierHonestCI), "", "")
	bctx, cancel := withBudget(ctx, d.tierBudget(testreport.TierHonestCI))
	defer cancel()
	r, err := d.Tier4.Verify(bctx, req)
	if err != nil {
		appendEventInPlace(&resp.DispatchTrace, d.Now(), "failed", string(testreport.TierHonestCI), "", err.Error())
		return &testreport.TestReport{
			SchemaVersion: testreport.SchemaVersion,
			TaskID:        req.TaskID,
			Tier:          testreport.TierHonestCI,
			Language:      testreport.LangPolyglot,
			Verdict:       testreport.VerdictFailed,
			Passed:        false,
			Error:         err.Error(),
		}
	}
	if r != nil {
		resp.CostBreakdown.RunnerSecondsByTier[string(testreport.TierHonestCI)] += r.DurationSeconds
		appendEventInPlace(&resp.DispatchTrace, d.Now(), phaseFor(r), string(testreport.TierHonestCI), "", "")
	}
	return r
}

// applyTier3Fallback finds any Tier 3 report with TimedOut=true and
// asserts the codeowner-review requirement is set. This is the brief's
// "never silently fail-open on Tier 3 timeout" invariant.
//
// The Tier 3 adapter MUST set codeowner_review_required=true on
// fallback; if it didn't, we synthesise the finding here so the rubric
// hard-rejects.
func (d *Dispatcher) applyTier3Fallback(reports []*testreport.TestReport, resp *verification.VerificationResponse) {
	for _, r := range reports {
		if r == nil || r.Proof == nil {
			continue
		}
		if r.Proof.TimedOut && r.Proof.FallbackTier != "" {
			appendEventInPlace(&resp.DispatchTrace, d.Now(),
				"fallback_engaged", string(r.Tier), string(r.Language),
				fmt.Sprintf("fallback=%s codeowner_required=%t",
					r.Proof.FallbackTier, r.Proof.CodeownerReviewRequired))
			// Defence in depth: enforce here as well.
			if !r.Proof.CodeownerReviewRequired {
				r.Findings = append(r.Findings, testreport.Finding{
					Category: "tier3_fallback_codeowner_required",
					Severity: "error",
					Detail:   "Tier 3 timed out; CODEOWNER review is required before promotion.",
				})
			}
		}
	}
}

// tierBudget returns the per-tier wall-clock cap.
func (d *Dispatcher) tierBudget(t testreport.Tier) time.Duration {
	switch t {
	case testreport.TierMutation:
		return d.TierBudgets.Tier0
	case testreport.TierPBT:
		return d.TierBudgets.Tier1
	case testreport.TierContract:
		return d.TierBudgets.Tier2
	case testreport.TierProof:
		return d.TierBudgets.Tier3
	case testreport.TierHonestCI:
		return d.TierBudgets.Tier4
	}
	return 5 * time.Minute
}

// selectTiers returns the tier set for a given request. Mirrors the
// "additive ladder" doc:
//
//   - Always include Tier 0 (mutation) and Tier 4 (honest-CI).
//   - Tier 1 (PBT) included unless complexity=trivial AND no critical-path hit.
//   - Tier 2 (contract) included when SpecChanges is non-empty OR any
//     CriticalPathScores file is in Hot/Molten.
//   - Tier 3 (proof) included when ANY file is in Molten (score ≥ 80).
func selectTiers(req *verification.VerificationRequest) []testreport.Tier {
	tiers := []testreport.Tier{testreport.TierMutation, testreport.TierHonestCI}

	hasHot := false
	hasMolten := false
	for _, s := range req.CriticalPathScores {
		switch s.Band {
		case "hot":
			hasHot = true
		case "molten":
			hasMolten = true
			hasHot = true
		}
	}

	// Tier 1: include unless task is clearly trivial.
	tiers = append(tiers, testreport.TierPBT)

	// Tier 2: included on spec changes or hot path.
	if len(req.SpecChanges) > 0 || hasHot {
		tiers = append(tiers, testreport.TierContract)
	}

	// Tier 3: only when something is molten.
	if hasMolten {
		tiers = append(tiers, testreport.TierProof)
	}

	// Sort stable: 0,1,2,3,4.
	order := map[testreport.Tier]int{
		testreport.TierMutation: 0,
		testreport.TierPBT:      1,
		testreport.TierContract: 2,
		testreport.TierProof:    3,
		testreport.TierHonestCI: 4,
	}
	sort.Slice(tiers, func(i, j int) bool { return order[tiers[i]] < order[tiers[j]] })
	return tiers
}

func hasTier(tiers []testreport.Tier, t testreport.Tier) bool {
	for _, x := range tiers {
		if x == t {
			return true
		}
	}
	return false
}

// chooseProverForLanguage picks a Tier 3 prover per language. Dafny is
// the universal default; Kani for Rust. Other languages fall back to
// Dafny via embedding-translation (out of scope for v1 — return Dafny
// and let the adapter dispatch the no-op-stub if it doesn't know how
// to translate).
func chooseProverForLanguage(lang testreport.Language) string {
	switch lang {
	case testreport.LangRust:
		return "kani"
	case testreport.LangPython, testreport.LangGo, testreport.LangTypeScript, testreport.LangJava:
		return "dafny"
	default:
		return "dafny"
	}
}

func phaseFor(r *testreport.TestReport) string {
	if r.Passed {
		return "passed"
	}
	if r.Verdict == testreport.VerdictTimedOut {
		return "timed_out"
	}
	return "failed"
}

// withBudget returns a child context with the budget timeout, or a
// no-op cancel func when timeout is zero.
func withBudget(ctx context.Context, d time.Duration) (context.Context, context.CancelFunc) {
	if d <= 0 {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, d)
}

// composeTierResults folds the per-language reports into the proto
// TierResults shape that VerifierApproval carries. When multiple
// languages report for the same tier, we take the worst per-tier
// outcome (any non-pass demotes the tier to non-pass) so the auditor
// sees the conservative aggregate.
func composeTierResults(reports []*testreport.TestReport) cruciblev1.TierResults {
	var tr cruciblev1.TierResults
	for _, r := range reports {
		if r == nil {
			continue
		}
		result := r.AsTierResult()
		switch r.Tier {
		case testreport.TierMutation:
			tr.Tier0 = mergeTierResult(tr.Tier0, &result)
		case testreport.TierPBT:
			tr.Tier1 = mergeTierResult(tr.Tier1, &result)
		case testreport.TierContract:
			tr.Tier2 = mergeTierResult(tr.Tier2, &result)
		case testreport.TierProof:
			tr.Tier3 = mergeTierResult(tr.Tier3, &result)
		case testreport.TierHonestCI:
			tr.Tier4 = mergeTierResult(tr.Tier4, &result)
		}
	}
	return tr
}

func mergeTierResult(a *cruciblev1.TierResult, b *cruciblev1.TierResult) *cruciblev1.TierResult {
	if a == nil {
		return b
	}
	if b == nil {
		return a
	}
	out := *a
	// Worst-case merge — `passed` is AND of inputs; score is min.
	out.Passed = a.Passed && b.Passed
	if b.Score < a.Score || a.Score == 0 {
		out.Score = b.Score
	}
	if b.Error != "" && a.Error == "" {
		out.Error = b.Error
	}
	out.DurationSeconds = a.DurationSeconds + b.DurationSeconds
	return &out
}

// diffHash returns the deterministic hash of the FileChange list.
// Real-world: hash content_sha256 per file in canonical order.
func diffHash(req *verification.VerificationRequest) string {
	files := append([]cruciblev1.FileChange{}, req.Diff.Files...)
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	h := newDiffHasher()
	for _, f := range files {
		h.WriteString(string(f.Action))
		h.WriteByte(0)
		h.WriteString(f.Path)
		h.WriteByte(0)
		h.WriteString(f.ContentSha256)
		h.WriteByte(0)
	}
	return h.Hex()
}

// appendEventInPlace appends to *trace under the package mutex so
// concurrent fan-out goroutines don't race on resp.DispatchTrace.
func appendEventInPlace(trace *[]verification.DispatchEvent, now time.Time, phase, tier, lang, detail string) {
	traceMu.Lock()
	defer traceMu.Unlock()
	*trace = append(*trace, verification.DispatchEvent{
		Timestamp: now.UnixMilli(),
		Phase:     phase,
		Tier:     tier,
		Language: lang,
		Detail:   detail,
	})
}
