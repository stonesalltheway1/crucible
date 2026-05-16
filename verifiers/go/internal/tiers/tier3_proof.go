// Tier 3 — formal verification.
//
// Go has no first-class formal verifier in v1 of Crucible. The
// daemon-side dispatcher handles Tier 3 work itself by routing to the
// Dafny / Lean / TLA+ adapters in apps/verifier/internal/tier3 —
// those operate on the spec artefacts in the diff (.dfy / .lean
// / .tla), not on the Go source.
//
// This runner emits a tool_unavailable TestReport so the dispatcher
// records the dispatch attempt for the per-tenant calibration
// data set, then falls back to the central Tier 3 driver. We do NOT
// fail open: the report's Passed field is false.
package tiers

import (
	"context"
	"time"

	"github.com/crucible/verifier/pkg/testreport"
)

// ProofConfig is a placeholder; Tier 3 takes no Go-runner-side config.
type ProofConfig struct{}

// RunProof emits a deterministic tool_unavailable report. The caller
// in main.go fills task/headers.
func RunProof(_ context.Context, _ ProofConfig) (*testreport.TestReport, error) {
	now := time.Now()
	return &testreport.TestReport{
		SchemaVersion: testreport.SchemaVersion,
		Tier:          testreport.TierProof,
		Language:      testreport.LangGo,
		Framework:     "tier3-dispatch",
		Verdict:       testreport.VerdictToolUnavailable,
		Passed:        false,
		StartedAt:     now,
		FinishedAt:    now,
		Proof: &testreport.ProofStats{
			Prover:   "n/a",
			TimedOut: false,
		},
		Error: "Go runner does not host a formal verifier; the daemon dispatches Tier 3 to the central Dafny/Lean/TLA+ adapters",
	}, nil
}
