// Package tier3 dispatches formal-verification provers. The default
// prover is Dafny (DafnyPro-style LLM-assisted discharge per POPL 2026).
// Lean 4, TLA+, and Z3 are stubbed with typed errors per ADR-008's
// "v1 default-off for non-Dafny provers" guidance.
//
// Critical invariant: on wall-clock timeout the adapter MUST emit a
// TestReport with:
//   - Proof.TimedOut = true
//   - Proof.FallbackTier = "tier_2_5"
//   - Proof.CodeownerReviewRequired = true
//
// Silent fail-open is a brand-existential bug; the dispatcher's
// applyTier3Fallback re-asserts the requirement, but the adapter is
// the first line of defence.
package tier3

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	cruciblev1 "github.com/crucible/sdk-go/crucible/v1"
	"github.com/crucible/verifier/internal/verification"
	"github.com/crucible/verifier/pkg/testreport"
)

// ProverID enumerates the supported Tier 3 provers.
type ProverID string

const (
	ProverDafny ProverID = "dafny"
	ProverKani  ProverID = "kani"
	ProverLean  ProverID = "lean"
	ProverTLA   ProverID = "tla"
	ProverZ3    ProverID = "z3"
)

// Adapter dispatches the Tier 3 proof against the right prover.
type Adapter struct {
	// Provers maps ProverID → Prover impl. Production wires Dafny;
	// other provers are typed-error stubs.
	Provers map[ProverID]Prover
	// Budgets bounds per-prover wall-clock.
	Budgets ProverBudgets
	// Cache holds partial-proof artifacts keyed by content hash of the
	// (diff, prover) pair. Lets the next PR resume where it left off.
	Cache PartialProofCache
	// Now is injectable for tests.
	Now func() time.Time
}

// Prover is one prover backend.
type Prover interface {
	ID() ProverID
	Discharge(ctx context.Context, req DischargeRequest) (DischargeResult, error)
}

// DischargeRequest is the per-prover input.
type DischargeRequest struct {
	TaskID       string
	BaseSHA      string
	Diff         cruciblev1.Diff
	Spec         string                // path to .dfy / .lean / .tla / .smt2
	WallClockSec int
	PartialProof []byte                // cached partial proof from a prior PR (may be nil)
}

// DischargeResult is the per-prover output.
type DischargeResult struct {
	Obligations            int
	Discharged             int
	TimedOut               bool
	WallClockSeconds       float64
	UnsoundnessHints       []string
	CachedPartial          []byte
	ProofArtifactPath      string
	Findings               []testreport.Finding
}

// ProverBudgets defines per-prover defaults from the doc.
type ProverBudgets struct {
	Dafny time.Duration
	Kani  time.Duration
	Lean  time.Duration
	TLA   time.Duration
	Z3    time.Duration
}

// DefaultBudgets matches docs/01-architecture/verifier-pipeline.md.
var DefaultBudgets = ProverBudgets{
	Dafny: 10 * time.Minute,
	Kani:  10 * time.Minute,
	Lean:  30 * time.Minute,
	TLA:   20 * time.Minute,
	Z3:    5 * time.Minute,
}

// PartialProofCache stores partial proofs keyed by (diff, prover) hash.
type PartialProofCache interface {
	Get(key string) ([]byte, bool)
	Put(key string, value []byte)
}

// NewAdapter returns an Adapter pre-wired with the Dafny default and
// typed-error stubs for the other provers.
func NewAdapter() *Adapter {
	a := &Adapter{
		Provers: map[ProverID]Prover{
			ProverDafny: NewDafnyProver(),
			ProverKani:  newStubProver(ProverKani, "Kani is dispatched via the Rust per-language runner — see verifiers/rust/."),
			ProverLean:  newStubProver(ProverLean, "Lean 4 + LeanCopilot Tier-3 adapter is a v2 Phase-9 feature (ADR-008)."),
			ProverTLA:   newStubProver(ProverTLA, "TLA+ + Apalache Tier-3 adapter is a v2 Phase-9 feature (ADR-008)."),
			ProverZ3:    newStubProver(ProverZ3, "Z3/CVC5 direct dispatch is a v2 Phase-9 feature (ADR-008)."),
		},
		Budgets: DefaultBudgets,
		Cache:   newMemoryCache(),
		Now:     time.Now,
	}
	return a
}

// Discharge dispatches the right prover by name. On any timeout, the
// returned report has the Tier 2.5 fallback fields set.
func (a *Adapter) Discharge(ctx context.Context, req *verification.VerificationRequest, proverID string) (*testreport.TestReport, error) {
	id := ProverID(strings.ToLower(proverID))
	prover, ok := a.Provers[id]
	if !ok {
		return nil, fmt.Errorf("tier3: unknown prover %q", proverID)
	}
	budget := a.budgetFor(id)
	pctx, cancel := context.WithTimeout(ctx, budget)
	defer cancel()

	start := a.Now()
	// Pull cached partial proof if we have one.
	cacheKey := cacheKey(req, id)
	cached, _ := a.Cache.Get(cacheKey)
	spec := pickSpecPath(req, id)

	dreq := DischargeRequest{
		TaskID:       req.TaskID,
		BaseSHA:      req.BaseSHA,
		Diff:         req.Diff,
		Spec:         spec,
		WallClockSec: int(budget.Seconds()),
		PartialProof: cached,
	}

	res, err := prover.Discharge(pctx, dreq)
	dur := a.Now().Sub(start)

	report := &testreport.TestReport{
		SchemaVersion:          testreport.SchemaVersion,
		TaskID:                 req.TaskID,
		DiffHash:               diffContentHash(req.Diff),
		Tier:                   testreport.TierProof,
		Language:               testreport.LangPolyglot,
		Framework:              string(id),
		StartedAt:              start.UTC(),
		FinishedAt:             a.Now().UTC(),
		DurationSeconds:        dur.Seconds(),
		WallClockBudgetSeconds: budget.Seconds(),
		ReporterID:             "crucible-verifier-tier3",
		ReporterVersion:        "phase4",
	}

	// 1. Hard error (e.g. stub prover) → report Verdict=tool_unavailable.
	if errors.Is(err, ErrProverStubbed) {
		report.Verdict = testreport.VerdictToolUnavailable
		report.Passed = false
		report.Error = err.Error()
		report.Proof = &testreport.ProofStats{Prover: string(id)}
		return report, nil
	}
	if err != nil && !errors.Is(err, context.DeadlineExceeded) {
		report.Verdict = testreport.VerdictFailed
		report.Passed = false
		report.Error = err.Error()
		report.Proof = &testreport.ProofStats{Prover: string(id)}
		return report, nil
	}

	timedOut := errors.Is(err, context.DeadlineExceeded) || res.TimedOut
	report.Proof = &testreport.ProofStats{
		Prover:                   string(id),
		ProofArtifact:            res.ProofArtifactPath,
		Obligations:              res.Obligations,
		Discharged:               res.Discharged,
		TimedOut:                 timedOut,
		WallClockSeconds:         res.WallClockSeconds,
		CachedPartial:            len(res.CachedPartial) > 0,
		UnsoundnessHints:         res.UnsoundnessHints,
	}
	if len(res.CachedPartial) > 0 {
		a.Cache.Put(cacheKey, res.CachedPartial)
	}

	if timedOut {
		// CRITICAL: never silently fail open. Set the Tier 2.5
		// fallback fields and the codeowner-review requirement.
		report.Proof.FallbackTier = "tier_2_5"
		report.Proof.CodeownerReviewRequired = true
		report.Verdict = testreport.VerdictTimedOut
		report.Passed = false
		report.Findings = append(report.Findings, testreport.Finding{
			Category: "tier3_timeout",
			Severity: "error",
			Detail:   fmt.Sprintf("Prover %q timed out after %.0fs; Tier 2.5 fallback engaged. CODEOWNER review required before promotion.", id, budget.Seconds()),
		})
		return report, nil
	}

	if res.Obligations > 0 && res.Discharged == res.Obligations {
		report.Verdict = testreport.VerdictPassed
		report.Passed = true
	} else {
		report.Verdict = testreport.VerdictFailed
		report.Passed = false
		report.Findings = append(report.Findings, testreport.Finding{
			Category: "proof_obligation_undischarged",
			Severity: "error",
			Detail:   fmt.Sprintf("%d of %d obligations remain", res.Obligations-res.Discharged, res.Obligations),
		})
	}
	report.Findings = append(report.Findings, res.Findings...)
	return report, nil
}

// budgetFor returns the per-prover budget.
func (a *Adapter) budgetFor(id ProverID) time.Duration {
	switch id {
	case ProverDafny:
		return a.Budgets.Dafny
	case ProverKani:
		return a.Budgets.Kani
	case ProverLean:
		return a.Budgets.Lean
	case ProverTLA:
		return a.Budgets.TLA
	case ProverZ3:
		return a.Budgets.Z3
	}
	return 5 * time.Minute
}

// ErrProverStubbed is the typed error stubs return.
var ErrProverStubbed = errors.New("tier3: prover stubbed in v1")

// stubProver returns ErrProverStubbed; reason is recorded for honest
// telemetry.
type stubProver struct {
	id     ProverID
	reason string
}

func newStubProver(id ProverID, reason string) *stubProver { return &stubProver{id: id, reason: reason} }

func (s *stubProver) ID() ProverID { return s.id }

func (s *stubProver) Discharge(_ context.Context, _ DischargeRequest) (DischargeResult, error) {
	return DischargeResult{}, fmt.Errorf("%w: %s", ErrProverStubbed, s.reason)
}

// pickSpecPath chooses the per-prover spec path from the diff. For
// Dafny that's any *.dfy in the diff; for Lean *.lean; etc.
func pickSpecPath(req *verification.VerificationRequest, id ProverID) string {
	want := map[ProverID]string{
		ProverDafny: ".dfy",
		ProverLean:  ".lean",
		ProverTLA:   ".tla",
		ProverZ3:    ".smt2",
	}[id]
	if want == "" {
		return ""
	}
	for _, f := range req.Diff.Files {
		if strings.HasSuffix(strings.ToLower(f.Path), want) {
			return f.Path
		}
	}
	return ""
}

// cacheKey is sha256(diff_hash || prover_id).
func cacheKey(req *verification.VerificationRequest, id ProverID) string {
	h := sha256.New()
	h.Write([]byte(diffContentHash(req.Diff)))
	h.Write([]byte{0})
	h.Write([]byte(id))
	return hex.EncodeToString(h.Sum(nil))
}

// diffContentHash is a deterministic hash over (path, action, content_sha256).
func diffContentHash(d cruciblev1.Diff) string {
	h := sha256.New()
	for _, f := range d.Files {
		h.Write([]byte(f.Path))
		h.Write([]byte{0})
		h.Write([]byte(f.Action))
		h.Write([]byte{0})
		h.Write([]byte(f.ContentSha256))
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))
}

// memoryCache is the default in-process partial-proof cache.
type memoryCache struct {
	mu sync.RWMutex
	m  map[string][]byte
}

func newMemoryCache() *memoryCache { return &memoryCache{m: map[string][]byte{}} }
func (c *memoryCache) Get(k string) ([]byte, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	v, ok := c.m[k]
	return v, ok
}
func (c *memoryCache) Put(k string, v []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.m[k] = v
}

// DafnyProver is the production prover for the default Tier 3 path.
// On a host without `dafny` installed, the prover returns
// ErrProverUnavailable and the adapter records VerdictToolUnavailable.
//
// The DafnyPro POPL 2026 paper publishes a recipe but no released
// model weights / framework; we implement the prompt-driven diff-
// checker + invariant-pruner + hint-augmenter loop ourselves over the
// dafny CLI. Phase 4 ships the orchestration; the LLM-assisted assertion
// generator (Laurel-style) is a follow-up adapter that drops into the
// `LaurelAugmenter` interface.
type DafnyProver struct {
	// DafnyBin is the path to the dafny binary. Defaults to "dafny".
	DafnyBin string
	// Verbose enables stderr logging.
	Verbose bool
	// Augmenter optionally synthesises assertions via an LLM. nil means
	// proof-shape-only (no LLM assistance — best for hermetic CI).
	Augmenter LaurelAugmenter
}

// NewDafnyProver returns a Dafny prover with binary auto-detected.
func NewDafnyProver() *DafnyProver { return &DafnyProver{DafnyBin: "dafny"} }

// ID implements Prover.
func (d *DafnyProver) ID() ProverID { return ProverDafny }

// LaurelAugmenter is the optional LLM-assisted assertion generator.
// Laurel (OOPSLA Apr 2025) achieves 56.6% of DafnyGym; Phase 4 ships
// the interface so the augmenter can be wired in Phase 5+ without a
// breaking change.
type LaurelAugmenter interface {
	Augment(ctx context.Context, spec string, failure string) (string, error)
}

// ErrProverUnavailable is returned when the dafny binary isn't on PATH.
var ErrProverUnavailable = errors.New("tier3/dafny: dafny binary not found on PATH")

// Discharge invokes `dafny verify <spec>`; on failure it (optionally)
// asks the Augmenter to propose extra assertions and re-tries.
// Wall-clock is governed by the context passed in by the adapter.
func (d *DafnyProver) Discharge(ctx context.Context, req DischargeRequest) (DischargeResult, error) {
	if req.Spec == "" {
		// No .dfy spec in the diff — nothing to prove.
		return DischargeResult{
			Obligations: 0,
			Discharged:  0,
		}, nil
	}
	bin := d.DafnyBin
	if bin == "" {
		bin = "dafny"
	}
	if _, err := exec.LookPath(bin); err != nil {
		return DischargeResult{}, ErrProverUnavailable
	}

	// First pass: vanilla verify.
	out, dur, err := d.runOnce(ctx, bin, req.Spec)
	res := DischargeResult{
		WallClockSeconds:  dur.Seconds(),
		ProofArtifactPath: req.Spec,
	}
	res.Obligations, res.Discharged = parseDafnyVerifyOutput(out)

	// Context-cancellation surfaces as TimedOut.
	if ctx.Err() != nil {
		res.TimedOut = true
		return res, ctx.Err()
	}

	if err == nil && res.Discharged == res.Obligations && res.Obligations > 0 {
		return res, nil
	}

	// If the Augmenter is wired, attempt a single retry with
	// LLM-proposed assertions. Phase-4 caps to one retry to bound cost;
	// Phase-5+ may iterate.
	if d.Augmenter != nil && res.Discharged < res.Obligations {
		fix, _ := d.Augmenter.Augment(ctx, req.Spec, out)
		if fix != "" {
			// Augmenter returned a candidate file path with extra
			// assertions; re-run verify on it. We do NOT mutate the
			// caller's spec — the augmented file lives in a temp dir.
			out2, dur2, err2 := d.runOnce(ctx, bin, fix)
			res.WallClockSeconds += dur2.Seconds()
			obl2, dis2 := parseDafnyVerifyOutput(out2)
			if obl2 > 0 && dis2 == obl2 {
				res.Obligations = obl2
				res.Discharged = dis2
				return res, nil
			}
			if err2 != nil {
				res.UnsoundnessHints = append(res.UnsoundnessHints,
					fmt.Sprintf("retry with augmented assertions failed: %v", err2))
			}
		}
	}

	if err != nil {
		// Distinguish timeout from other failures.
		if errors.Is(err, context.DeadlineExceeded) {
			res.TimedOut = true
		}
		return res, err
	}
	return res, nil
}

// runOnce invokes `dafny verify <spec>`. Captures stdout+stderr.
func (d *DafnyProver) runOnce(ctx context.Context, bin, spec string) (string, time.Duration, error) {
	start := time.Now()
	args := []string{"verify", "--no-verify-runtime"}
	// On Dafny 4.11+, the modern subcommand is `verify`. Older versions
	// used flags only; we tolerate both.
	if !filepath.IsAbs(spec) {
		// Pass the spec as-is; dafny resolves relative to cwd.
	}
	args = append(args, spec)
	cmd := exec.CommandContext(ctx, bin, args...)
	out, err := cmd.CombinedOutput()
	dur := time.Since(start)
	return string(out), dur, err
}

// parseDafnyVerifyOutput extracts (obligations, discharged) from Dafny's
// summary lines. Dafny 4.x prints "Dafny program verifier finished with
// N verified, M errors". The 'verified' field counts discharged
// obligations; 'errors' is the undischarged count. If we can't parse,
// we surface (0, 0) — the adapter records "tool_unavailable"-shaped
// non-discharge rather than synthesising false discharges.
func parseDafnyVerifyOutput(out string) (obligations, discharged int) {
	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(line, "Dafny program verifier finished with") {
			var verified, errs int
			// Format: "...with %d verified, %d errors"
			_, _ = fmt.Sscanf(line, "Dafny program verifier finished with %d verified, %d errors", &verified, &errs)
			discharged = verified
			obligations = verified + errs
			return
		}
		// Newer dafny variants use "Verification finished. Verified: %d. Errors: %d."
		if strings.HasPrefix(strings.TrimSpace(line), "Verification finished") {
			var v, e int
			_, _ = fmt.Sscanf(line, "Verification finished. Verified: %d. Errors: %d.", &v, &e)
			if v > 0 || e > 0 {
				discharged = v
				obligations = v + e
				return
			}
		}
	}
	return 0, 0
}
