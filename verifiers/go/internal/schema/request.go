// Package schema re-declares the subset of the verifier-daemon
// VerificationRequest the per-language runner needs. The daemon's
// canonical definition lives at apps/verifier/internal/verification —
// that package is `internal/` so we cannot import it. To prevent
// schema drift we keep this file deliberately narrow: only the fields
// the Go runner reads (TaskID, Diff, TestFiles, SpecChanges, Budget,
// ExecutorSandboxID).
//
// If you add a field to verification.VerificationRequest that the Go
// runner must observe, mirror it here and bump the runner version.
package schema

import (
	cruciblev1 "github.com/crucible/sdk-go/crucible/v1"
)

// VerificationRequest is the runner-side mirror of
// apps/verifier/internal/verification.VerificationRequest.
//
// NOTE: this is intentionally a loose superset — unknown fields on
// stdin are silently ignored by encoding/json so adding fields to the
// daemon's struct without updating this one is forward-compatible.
type VerificationRequest struct {
	TaskID            string             `json:"task_id"`
	TenantID          string             `json:"tenant_id"`
	Repo              string             `json:"repo"`
	BaseSHA           string             `json:"base_sha"`
	Diff              cruciblev1.Diff    `json:"diff"`
	TestFiles         []cruciblev1.FileChange `json:"test_files,omitempty"`
	SpecChanges       []SpecChange       `json:"spec_changes,omitempty"`
	Routing           cruciblev1.Routing `json:"routing"`
	Languages         []string           `json:"languages"`
	Budget            BudgetEnvelope     `json:"budget"`
	AttestationChain  []string           `json:"attestation_chain,omitempty"`
	ExecutorSandboxID string             `json:"executor_sandbox_id"`

	// WorkDir is a runner-local extension: the dispatcher sets it via
	// the sandbox env (CRUCIBLE_VERIFIER_WORKDIR) to a directory where
	// the diff has been materialized. The runner reads this to locate
	// the source tree the tools should operate on. Not present in the
	// daemon-side struct (it's a transport-layer concern).
	WorkDir string `json:"work_dir,omitempty"`
}

// SpecChange mirrors verification.SpecChange.
type SpecChange struct {
	Path         string `json:"path"`
	Kind         string `json:"kind"`
	PreviousHash string `json:"previous_hash"`
	CurrentHash  string `json:"current_hash"`
	Delta        string `json:"delta,omitempty"`
}

// BudgetEnvelope mirrors verification.BudgetEnvelope.
type BudgetEnvelope struct {
	VerifierCapUSD        float64 `json:"verifier_cap_usd"`
	VerifierSpentUSD      float64 `json:"verifier_spent_usd"`
	WallClockCapSeconds   uint64  `json:"wall_clock_cap_seconds"`
	WallClockSpentSeconds uint64  `json:"wall_clock_spent_seconds"`
}
