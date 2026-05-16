// Package verification defines the central VerificationRequest / Response
// types that flow between the control plane and the verifier daemon. These
// are the inputs to the dispatcher; everything downstream (rubric, runners,
// tier3, tier4) consumes a Request and emits a TestReport or rubric verdict.
//
// The schema is deliberately conservative — every field has a clear
// provenance trail to an attestation. The audit guard in this package
// AUDITS THAT EXECUTOR REASONING IS NEVER PASSED TO THE VERIFIER (the ADR-002
// load-bearing invariant). Any request with a field whose name matches the
// reasoning-deny-list is rejected before any model call.
package verification

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	cruciblev1 "github.com/crucible/sdk-go/crucible/v1"
)

// VerificationRequest is what the control plane sends the verifier daemon
// once the executor agent has claimed `done` on a task.
//
// CRITICAL: this struct lists ONLY the inputs the verifier is allowed to
// see. Adding executor reasoning, internal scratch, or chain-of-thought
// fields here is a design-time error — the audit guard in this package
// will refuse such requests at ingest.
type VerificationRequest struct {
	TaskID   string `json:"task_id"`
	TenantID string `json:"tenant_id"`
	Repo     string `json:"repo"`
	BaseSHA  string `json:"base_sha"`

	// Diff is the cumulative agent-authored diff. The verifier reasons
	// over this, NOT over the agent's reasoning steps.
	Diff cruciblev1.Diff `json:"diff"`

	// Tests is the diff-scoped subset of test files in the same Diff.
	// Surfaced separately for the rubric prompt.
	TestFiles []cruciblev1.FileChange `json:"test_files,omitempty"`

	// SpecChanges enumerates OpenAPI / GraphQL spec deltas the executor
	// produced. The Tier 2 schemathesis runner uses these directly.
	SpecChanges []SpecChange `json:"spec_changes,omitempty"`

	// Routing carries the executor + verifier model identities. The
	// rubric package re-enforces cross-family at call time.
	Routing cruciblev1.Routing `json:"routing"`

	// Languages enumerates the languages touched by the diff. The
	// dispatcher fans out runners accordingly.
	Languages []string `json:"languages"`

	// CriticalPathScores is the per-file score from the critical-path
	// classifier. The dispatcher's tier-selection logic reads these.
	CriticalPathScores []CriticalPathScore `json:"critical_path_scores,omitempty"`

	// PerTaskSignals carries the Phase-3 carry-over signals the rubric
	// must consult. These are TRUST signals — NOT reasoning leakage.
	PerTaskSignals TaskSignals `json:"per_task_signals,omitempty"`

	// Budget is the verifier-side budget envelope (separate from the
	// executor's per ADR-009).
	Budget BudgetEnvelope `json:"budget"`

	// AttestationChain carries the rekor UUIDs the verifier should fold
	// into its final approval / rejection. Per the brief: scrubber
	// AuditLog UUIDs, write attestations, migration attestations.
	AttestationChain []string `json:"attestation_chain,omitempty"`

	// ExecutorSandboxID — the sandbox the executor ran in. The verifier
	// MUST refuse to run inside the same sandbox (ADR-002).
	ExecutorSandboxID string `json:"executor_sandbox_id"`
}

// SpecChange is a single OpenAPI / GraphQL / Avro spec delta in the diff.
type SpecChange struct {
	Path         string `json:"path"`
	Kind         string `json:"kind"`             // "openapi" | "graphql" | "avro" | "proto"
	PreviousHash string `json:"previous_hash"`
	CurrentHash  string `json:"current_hash"`
	Delta        string `json:"delta,omitempty"`  // unified diff
}

// CriticalPathScore is one (file, score, band) triple from the classifier.
type CriticalPathScore struct {
	File   string  `json:"file"`
	Score  float64 `json:"score"`              // 0..100
	Band   string  `json:"band"`               // "cold" | "warm" | "hot" | "molten"
	Reason string  `json:"reason,omitempty"`
}

// TaskSignals folds in the carry-over Phase-3 trust signals: tape
// staleness, tape disposition distribution, scrubber audit fired,
// WASM quota trips, self-host availability.
type TaskSignals struct {
	StalenessFindings        []StalenessFinding   `json:"staleness_findings,omitempty"`
	TapeDispositionHistogram map[string]int       `json:"tape_disposition_histogram,omitempty"`
	ScrubberAuditEntryCount  int                  `json:"scrubber_audit_entry_count,omitempty"`
	ScrubberFiredOnAllPII    bool                 `json:"scrubber_fired_on_all_pii"`
	WasmQuotaTrips           []WasmQuotaTrip      `json:"wasm_quota_trips,omitempty"`
	SelfHostAvailable        bool                 `json:"self_host_available"`
	SelfHostUnavailableReason string              `json:"self_host_unavailable_reason,omitempty"`
}

// StalenessFinding mirrors twin-runtime-staleness::Finding (Rust side).
type StalenessFinding struct {
	Endpoint     string `json:"endpoint"`
	Band         string `json:"band"`             // "fresh" | "aging" | "stale" | "unrecorded"
	AgeSeconds   int64  `json:"age_seconds"`
	IntervalSecs int64  `json:"interval_seconds"`
	Message      string `json:"message"`
}

// WasmQuotaTrip mirrors twin-runtime-wasm::QuotaTrip.
type WasmQuotaTrip struct {
	Quota     string `json:"quota"`             // "wall_clock" | "memory" | "fuel"
	TrippedAt int64  `json:"tripped_at"`        // epoch millis
	Detail    string `json:"detail,omitempty"`
}

// BudgetEnvelope tracks the verifier-side budget — separate from executor.
type BudgetEnvelope struct {
	VerifierCapUSD        float64 `json:"verifier_cap_usd"`
	VerifierSpentUSD      float64 `json:"verifier_spent_usd"`
	WallClockCapSeconds   uint64  `json:"wall_clock_cap_seconds"`
	WallClockSpentSeconds uint64  `json:"wall_clock_spent_seconds"`
}

// VerificationResponse is the verifier's final verdict. Exactly one of
// Approval / Rejection is populated.
type VerificationResponse struct {
	Approval  *cruciblev1.VerifierApproval  `json:"approval,omitempty"`
	Rejection *cruciblev1.VerifierRejection `json:"rejection,omitempty"`

	// Side-channels for diagnostic surfacing — not part of the
	// attestation payload itself.
	DispatchTrace []DispatchEvent `json:"dispatch_trace,omitempty"`
	CostBreakdown CostBreakdown   `json:"cost_breakdown"`
}

// DispatchEvent records a tier's lifecycle for debugging.
type DispatchEvent struct {
	Timestamp   int64  `json:"ts_unix_millis"`
	Phase       string `json:"phase"`           // "dispatched" | "passed" | "failed" | "timed_out" | "fallback_engaged"
	Tier        string `json:"tier"`
	Language    string `json:"language,omitempty"`
	Detail      string `json:"detail,omitempty"`
}

// CostBreakdown attributes verifier cost to its sources for telemetry.
type CostBreakdown struct {
	RubricUSD     float64 `json:"rubric_usd"`
	ClassifierUSD float64 `json:"classifier_usd"`
	RunnerSecondsByTier map[string]float64 `json:"runner_seconds_by_tier"`
	TotalUSD      float64 `json:"total_usd"`
}

// Validate runs all schema-level checks AND the executor-reasoning leak
// audit. Returns an error suitable for surfacing back to the control plane.
func (r *VerificationRequest) Validate() error {
	if r == nil {
		return errors.New("verification: nil request")
	}
	if r.TaskID == "" {
		return errors.New("verification: task_id required")
	}
	if r.TenantID == "" {
		return errors.New("verification: tenant_id required")
	}
	if r.BaseSHA == "" {
		return errors.New("verification: base_sha required")
	}
	if len(r.Diff.Files) == 0 {
		return errors.New("verification: empty diff")
	}
	if r.Routing.ExecutorVendor == "" || r.Routing.VerifierVendor == "" {
		return errors.New("verification: routing must carry both vendor lineages")
	}
	if strings.EqualFold(r.Routing.ExecutorVendor, r.Routing.VerifierVendor) {
		return &SameFamilyError{
			Executor: r.Routing.ExecutorVendor,
			Verifier: r.Routing.VerifierVendor,
		}
	}
	if r.ExecutorSandboxID == "" {
		return errors.New("verification: executor_sandbox_id required (audit trail)")
	}
	return nil
}

// SameFamilyError is returned when executor and verifier share a vendor —
// the cross-family invariant from ADR-002.
type SameFamilyError struct {
	Executor string
	Verifier string
}

func (e *SameFamilyError) Error() string {
	return fmt.Sprintf("verification: executor and verifier share vendor lineage %q — ADR-002 invariant violated", e.Executor)
}

// LeakageError is returned when the audit guard finds suspicious fields.
type LeakageError struct {
	OffendingField string
	Pattern        string
}

func (e *LeakageError) Error() string {
	return fmt.Sprintf("verification: executor-reasoning leak detected — field %q matched pattern %q (ADR-002 invariant)", e.OffendingField, e.Pattern)
}

// reasoningDenylist is the lower-case substring set the audit guard checks
// against the JSON-tag namespace of any payload reaching a model call.
//
// This list is intentionally aggressive — false-positives are cheaper than
// the brand-existential cost of a leaked reasoning trace.
var reasoningDenylist = []string{
	"reasoning",
	"chain_of_thought",
	"chain-of-thought",
	"cot",
	"thinking_trace",
	"thinking-trace",
	"thoughts",
	"scratchpad",
	"internal_monologue",
	"hidden_state",
	"agent_trace",
	"executor_trace",
	"trajectory",
	"plan_critique",
	"reflection",
}

// AuditNoLeakage scans an arbitrary value's exported field names for
// patterns that suggest executor reasoning. Returns LeakageError on hit.
//
// We accept map[string]any (the typical "render the rubric prompt
// payload" surface) — keys are the field names; values are recursed.
func AuditNoLeakage(payload map[string]any) error {
	return auditMap(payload, "")
}

func auditMap(m map[string]any, prefix string) error {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		full := k
		if prefix != "" {
			full = prefix + "." + k
		}
		lk := strings.ToLower(k)
		for _, deny := range reasoningDenylist {
			if strings.Contains(lk, deny) {
				return &LeakageError{OffendingField: full, Pattern: deny}
			}
		}
		switch v := m[k].(type) {
		case map[string]any:
			if err := auditMap(v, full); err != nil {
				return err
			}
		case []any:
			for i, e := range v {
				if mm, ok := e.(map[string]any); ok {
					if err := auditMap(mm, fmt.Sprintf("%s[%d]", full, i)); err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}

// AuditRequest applies the leak guard to the public-facing fields of a
// VerificationRequest. The struct schema itself is the first line of
// defense (no reasoning fields exist); this guard catches accidental
// inclusion via the AttestationChain / TaskSignals or any future
// extension.
func (r *VerificationRequest) AuditNoLeakage() error {
	// Cheap structural scan — re-marshal through a generic map.
	view := map[string]any{
		"task_id":            r.TaskID,
		"tenant_id":          r.TenantID,
		"base_sha":           r.BaseSHA,
		"executor_sandbox":   r.ExecutorSandboxID,
		"executor_model":     r.Routing.ExecutorModel,
		"verifier_model":     r.Routing.VerifierModel,
	}
	for _, f := range r.Diff.Files {
		if isReasoningPath(f.Path) {
			return &LeakageError{OffendingField: "diff.files." + f.Path, Pattern: "path-pattern"}
		}
	}
	for _, s := range r.AttestationChain {
		if strings.Contains(strings.ToLower(s), "reasoning") ||
			strings.Contains(strings.ToLower(s), "scratchpad") {
			return &LeakageError{OffendingField: "attestation_chain", Pattern: "explicit-naming"}
		}
	}
	return AuditNoLeakage(view)
}

func isReasoningPath(p string) bool {
	pl := strings.ToLower(p)
	for _, deny := range []string{
		".reasoning.", "/reasoning/",
		".cot.", "/cot/",
		"_thinking_", "_scratchpad_",
		"agent_trace", "executor_trace",
	} {
		if strings.Contains(pl, deny) {
			return true
		}
	}
	return false
}
