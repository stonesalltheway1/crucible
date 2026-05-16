// Package testreport is the canonical schema every per-language verifier
// emits when it finishes a tier. It is a strict superset of the
// proto TierResult so the dispatcher can fold N per-language reports
// into the single TierResults that the VerifierApproval predicate carries.
//
// Schema version is pinned to "https://crucible.dev/TestReport/v1" — bumps
// require a 90-day deprecation per twin-spec/schemas/README.md.
package testreport

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"

	cruciblev1 "github.com/crucible/sdk-go/crucible/v1"
)

// PredicateType is the in-toto predicateType URI for every TestReport.
const PredicateType = "https://crucible.dev/TestReport/v1"

// SchemaVersion identifies the report contract. Bumped only on breaking change.
const SchemaVersion = "1"

// Language enumerates the per-language runners.
type Language string

const (
	LangPython     Language = "python"
	LangTypeScript Language = "typescript"
	LangRust       Language = "rust"
	LangGo         Language = "go"
	LangJava       Language = "java"
	LangSwift      Language = "swift"
	LangPolyglot   Language = "polyglot"
)

// Tier mirrors cruciblev1.Tier but is a plain string for runner-native code.
type Tier string

const (
	TierMutation  Tier = "tier_0_mutation"
	TierPBT       Tier = "tier_1_pbt"
	TierContract  Tier = "tier_2_contract"
	TierProof     Tier = "tier_3_proof"
	TierHonestCI  Tier = "tier_4_honest_ci"
)

// Verdict is the runner-local pass/fail signal.
type Verdict string

const (
	VerdictPassed         Verdict = "passed"
	VerdictFailed         Verdict = "failed"
	VerdictTimedOut       Verdict = "timed_out"
	VerdictToolUnavailable Verdict = "tool_unavailable"
	VerdictSkipped        Verdict = "skipped"
)

// TestReport is a single runner+tier+language report. It is JSON-serialised
// over the runner stdio protocol; the dispatcher unmarshals it directly.
//
// The fields here are deliberately a union over all five tiers: each tier
// fills in only the subset that applies (Mutation fills MutationStats,
// PBT fills PBTStats, etc.). The schema is intentionally flat so that
// downstream attestation consumers don't have to switch on tier-specific
// shapes.
type TestReport struct {
	SchemaVersion string    `json:"schema_version"`
	TaskID        string    `json:"task_id"`
	DiffHash      string    `json:"diff_hash"`
	Tier          Tier      `json:"tier"`
	Language      Language  `json:"language"`
	Framework     string    `json:"framework"`
	Verdict       Verdict   `json:"verdict"`
	Passed        bool      `json:"passed"`
	StartedAt     time.Time `json:"started_at"`
	FinishedAt    time.Time `json:"finished_at"`
	DurationSeconds float64 `json:"duration_seconds"`
	WallClockBudgetSeconds float64 `json:"wall_clock_budget_seconds"`

	// Per-tier statistics — exactly one of these MUST be populated for a
	// successful run (tool_unavailable and skipped may leave them zero).
	Mutation *MutationStats `json:"mutation,omitempty"`
	PBT      *PBTStats      `json:"pbt,omitempty"`
	Contract *ContractStats `json:"contract,omitempty"`
	Proof    *ProofStats    `json:"proof,omitempty"`
	HonestCI *HonestCIStats `json:"honest_ci,omitempty"`

	// Findings is the deterministic list of failures attributable to this
	// run. The rubric LLM-judge consumes these to compose
	// VerifierRejection.RejectionReasons.
	Findings []Finding `json:"findings,omitempty"`

	// Tool integrity: the SHA-256 of the tool binary or pinned image
	// digest. Lets compliance auditors check the verifier wasn't running a
	// shadow tool against the diff. Empty when not yet collected.
	ToolDigest string `json:"tool_digest,omitempty"`

	// Reporter identity — the per-language runner binary's name/version
	// and the OIDC subject of the sandbox the runner executed in.
	ReporterID         string `json:"reporter_id"`
	ReporterVersion    string `json:"reporter_version,omitempty"`
	ReporterOidcSubject string `json:"reporter_oidc_subject,omitempty"`

	// Error is populated when Verdict is failed or timed_out and the
	// failure is procedural (tool crashed, output unparsable). For
	// substantive test failures, populate Findings instead.
	Error string `json:"error,omitempty"`
}

// MutationStats — Tier 0 (mutmut / stryker-js / cargo-mutants / go-mutesting / pitest / muter).
type MutationStats struct {
	Killed         int      `json:"killed"`
	Survived       int      `json:"survived"`
	NotCovered     int      `json:"not_covered,omitempty"`
	Timeout        int      `json:"timeout,omitempty"`
	Total          int      `json:"total"`
	Score          float64  `json:"score"`              // Killed / (Killed+Survived)
	Threshold      float64  `json:"threshold"`          // required for pass
	DiffScoped     bool     `json:"diff_scoped"`        // MUST be true in Crucible
	MutatedFiles   []string `json:"mutated_files,omitempty"`
	SurvivedSummary []SurvivedMutant `json:"survived_summary,omitempty"`
}

type SurvivedMutant struct {
	File      string `json:"file"`
	Line      int    `json:"line"`
	Mutator   string `json:"mutator"`             // e.g. "BooleanLiteral", "ArithmeticOperator"
	Original  string `json:"original,omitempty"`
	Replacement string `json:"replacement,omitempty"`
}

// PBTStats — Tier 1 (hypothesis / fast-check / proptest / rapid / jqwik).
type PBTStats struct {
	Iterations      int      `json:"iterations"`
	IterationsMin   int      `json:"iterations_min"` // Crucible mandate: ≥10_000
	Properties      []string `json:"properties,omitempty"`
	Counterexamples []Counterexample `json:"counterexamples,omitempty"`
	FuzzCorpusSize  int      `json:"fuzz_corpus_size,omitempty"`
	FuzzNewSeeds    int      `json:"fuzz_new_seeds,omitempty"`
	FuzzCrashes     int      `json:"fuzz_crashes,omitempty"`
}

type Counterexample struct {
	Property  string `json:"property"`
	Shrunk    string `json:"shrunk"`          // human-readable counter-example
	Seed      string `json:"seed,omitempty"`  // for reproducibility
	StackHint string `json:"stack_hint,omitempty"`
}

// ContractStats — Tier 2 (schemathesis / DST runner).
type ContractStats struct {
	SpecPath           string   `json:"spec_path,omitempty"`         // openapi.yaml / schema.graphql
	SpecHash           string   `json:"spec_hash,omitempty"`
	StatefulWorkflows  int      `json:"stateful_workflows,omitempty"`
	Checks             []string `json:"checks,omitempty"`
	Violations         []ContractViolation `json:"violations,omitempty"`
	DstIterations      int      `json:"dst_iterations,omitempty"`
	DstReplayID        string   `json:"dst_replay_id,omitempty"`
	DstFailingSchedule string   `json:"dst_failing_schedule,omitempty"`
}

type ContractViolation struct {
	Endpoint   string `json:"endpoint"`
	Method     string `json:"method"`
	Check      string `json:"check"`              // e.g. "response_schema_conformance"
	Detail     string `json:"detail"`
	Reproducer string `json:"reproducer,omitempty"`
}

// ProofStats — Tier 3 (Dafny / Lean / TLA+ / Kani / Z3).
type ProofStats struct {
	Prover         string   `json:"prover"`                // "dafny" | "kani" | "lean" | "tla" | "z3"
	ProofArtifact  string   `json:"proof_artifact,omitempty"` // path to .dfy / .lean / .tla
	Obligations    int      `json:"obligations,omitempty"`
	Discharged     int      `json:"discharged,omitempty"`
	TimedOut       bool     `json:"timed_out"`
	WallClockSeconds float64 `json:"wall_clock_seconds,omitempty"`
	CachedPartial  bool     `json:"cached_partial,omitempty"`
	FallbackTier   string   `json:"fallback_tier,omitempty"` // "tier_2_5" when timed out
	CodeownerReviewRequired bool `json:"codeowner_review_required,omitempty"`
	UnsoundnessHints []string `json:"unsoundness_hints,omitempty"`
}

// HonestCIStats — Tier 4 (Nix hermetic rebuild + SLSA-L3).
type HonestCIStats struct {
	BuilderID            string   `json:"builder_id"`             // e.g. "https://crucible.dev/builders/hermetic-nix/v1"
	NixFlakeHash         string   `json:"nix_flake_hash,omitempty"`
	NixLockHash          string   `json:"nix_lock_hash,omitempty"`
	ExecutorRebuildHash  string   `json:"executor_rebuild_hash"`
	VerifierRebuildHash  string   `json:"verifier_rebuild_hash"`
	BitIdentical         bool     `json:"bit_identical"`
	SLSALevel            int      `json:"slsa_level"`             // 0..4
	InTotoStatementHash  string   `json:"in_toto_statement_hash,omitempty"`
	FulcioCertHash       string   `json:"fulcio_cert_hash,omitempty"`
	RekorUUID            string   `json:"rekor_uuid,omitempty"`
	WitnessAttestation   string   `json:"witness_attestation,omitempty"`
	TektonChainsRef      string   `json:"tekton_chains_ref,omitempty"`
	DiffoscopeReport     string   `json:"diffoscope_report,omitempty"` // local-journal blob id
	ScrubberAuditOK      bool     `json:"scrubber_audit_ok"`
	ScrubberAuditEntries int      `json:"scrubber_audit_entries,omitempty"`
}

// Finding is the canonical structured-error format. Each Finding maps
// 1:1 onto a RejectionReason in VerifierRejection.
type Finding struct {
	Category     string `json:"category"`     // "mutation_survived" | "property_failed" | ...
	Severity     string `json:"severity"`     // "info" | "warn" | "error"
	File         string `json:"file,omitempty"`
	Line         int    `json:"line,omitempty"`
	Detail       string `json:"detail"`
	SuggestedFix string `json:"suggested_fix,omitempty"`
}

// Validate enforces the invariants the dispatcher relies on.
func (r *TestReport) Validate() error {
	if r == nil {
		return errors.New("testreport: nil")
	}
	if r.SchemaVersion != SchemaVersion {
		return fmt.Errorf("testreport: schema_version %q != %q", r.SchemaVersion, SchemaVersion)
	}
	if r.TaskID == "" {
		return errors.New("testreport: task_id required")
	}
	if r.Tier == "" {
		return errors.New("testreport: tier required")
	}
	if r.Language == "" {
		return errors.New("testreport: language required")
	}
	switch r.Tier {
	case TierMutation, TierPBT, TierContract, TierProof, TierHonestCI:
	default:
		return fmt.Errorf("testreport: unknown tier %q", r.Tier)
	}
	if r.DurationSeconds < 0 {
		return errors.New("testreport: negative duration")
	}
	if r.Tier == TierMutation && r.Mutation != nil {
		if !r.Mutation.DiffScoped {
			return errors.New("testreport: mutation report not diff-scoped (Crucible mandate)")
		}
		if r.Mutation.Total > 0 && r.Mutation.Killed+r.Mutation.Survived > r.Mutation.Total {
			return errors.New("testreport: mutation killed+survived > total")
		}
	}
	if r.Tier == TierPBT && r.PBT != nil {
		if r.PBT.IterationsMin > 0 && r.PBT.Iterations < r.PBT.IterationsMin {
			return fmt.Errorf("testreport: PBT iterations %d < required %d",
				r.PBT.Iterations, r.PBT.IterationsMin)
		}
	}
	return nil
}

// CanonicalJSON returns a stable, sorted-key JSON encoding suitable for hashing
// and attestation. Maps are sorted; arrays preserve order; whitespace is removed.
func (r *TestReport) CanonicalJSON() ([]byte, error) {
	// json.Marshal already emits sorted keys for map[string]X but Go structs
	// preserve field order. Round-trip through map[string]any to canonicalize
	// any nested map types defensively.
	raw, err := json.Marshal(r)
	if err != nil {
		return nil, err
	}
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return nil, err
	}
	return marshalCanonical(v)
}

// ContentHash returns the hex-encoded SHA-256 of CanonicalJSON.
func (r *TestReport) ContentHash() (string, error) {
	b, err := r.CanonicalJSON()
	if err != nil {
		return "", err
	}
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:]), nil
}

// AsTierResult folds this TestReport into the proto TierResult so the
// dispatcher can populate VerifierApproval.TierResults.
func (r *TestReport) AsTierResult() cruciblev1.TierResult {
	tr := cruciblev1.TierResult{
		Tier:            cruciblev1.Tier(r.Tier),
		Passed:          r.Passed,
		Framework:       r.Framework,
		DurationSeconds: uint64(r.DurationSeconds),
		Error:           r.Error,
	}
	switch {
	case r.Mutation != nil:
		tr.Score = r.Mutation.Score
	case r.PBT != nil:
		if r.PBT.Iterations > 0 {
			tr.Score = 1.0 - float64(len(r.PBT.Counterexamples))/float64(r.PBT.Iterations)
		}
	case r.Proof != nil:
		if r.Proof.Obligations > 0 {
			tr.Score = float64(r.Proof.Discharged) / float64(r.Proof.Obligations)
		}
	case r.HonestCI != nil:
		if r.HonestCI.BitIdentical {
			tr.Score = 1.0
		}
	}
	return tr
}

// MergeFindings concatenates and de-duplicates findings keyed by (Category,File,Line,Detail).
func MergeFindings(reports ...*TestReport) []Finding {
	type key struct {
		category, file, detail string
		line                   int
	}
	seen := map[key]bool{}
	out := make([]Finding, 0)
	for _, r := range reports {
		if r == nil {
			continue
		}
		for _, f := range r.Findings {
			k := key{f.Category, f.File, f.Detail, f.Line}
			if seen[k] {
				continue
			}
			seen[k] = true
			out = append(out, f)
		}
	}
	return out
}

// marshalCanonical encodes v with sorted object keys for hashing.
func marshalCanonical(v any) ([]byte, error) {
	switch t := v.(type) {
	case map[string]any:
		keys := make([]string, 0, len(t))
		for k := range t {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		buf := []byte{'{'}
		for i, k := range keys {
			if i > 0 {
				buf = append(buf, ',')
			}
			kb, _ := json.Marshal(k)
			buf = append(buf, kb...)
			buf = append(buf, ':')
			vb, err := marshalCanonical(t[k])
			if err != nil {
				return nil, err
			}
			buf = append(buf, vb...)
		}
		buf = append(buf, '}')
		return buf, nil
	case []any:
		buf := []byte{'['}
		for i, e := range t {
			if i > 0 {
				buf = append(buf, ',')
			}
			eb, err := marshalCanonical(e)
			if err != nil {
				return nil, err
			}
			buf = append(buf, eb...)
		}
		buf = append(buf, ']')
		return buf, nil
	default:
		return json.Marshal(t)
	}
}
