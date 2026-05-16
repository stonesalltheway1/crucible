package rubric

import (
	"context"
	"fmt"
	"strings"

	cruciblev1 "github.com/crucible/sdk-go/crucible/v1"
)

// MemoryComplianceFeaturizer queries the Phase-5 memory layer for
// conventions touching the diff scope, then translates the resulting
// ComplianceReport into RejectionReason entries that feed the
// trust_signal_alignment criterion.
//
// This is the Phase-5 wiring for the "Memory as verifier" loop closure
// (docs/01-architecture/memory-layer.md §"Memory as verifier"). The
// Phase-4 rubric reserved the slot; Phase-5 fills it.
//
// CRITICAL: this code path NEVER sees executor reasoning. The diff +
// per-file paths is the entire input; the memorybridge passes nothing
// else. The audit guard at the rubric ingress already enforces that —
// this featurizer is downstream of that guard.
type MemoryComplianceFeaturizer struct {
	// Bridge is the memorybridge.Bridge — interface-typed here to
	// avoid an import cycle from rubric → memorybridge → rubric.
	Bridge ComplianceClient
	// Disabled short-circuits the featurizer in CI when the bridge is
	// unwired. The Phase-5 cmd/main flips this based on the env var.
	Disabled bool
}

// ComplianceClient is the minimal surface MemoryComplianceFeaturizer
// uses. memorybridge.Bridge satisfies it. Tests inject in-process fakes.
type ComplianceClient interface {
	CheckCompliance(ctx context.Context, req ComplianceRequest) (cruciblev1.ComplianceReport, error)
}

// ComplianceRequest mirrors memorybridge.CheckRequest but lives here to
// avoid the cycle.
type ComplianceRequest struct {
	TenantID string
	TaskID   string
	Diff     cruciblev1.Diff
}

// Features holds the trust-signal contribution from the memory layer.
// trustSignalDelta is added to the trust_signal_alignment criterion;
// negative deltas dock the score, positive deltas (currently unused)
// would raise it.
type Features struct {
	ConventionsChecked uint32
	WarnViolations     uint32
	ErrorViolations    uint32
	RejectionReasons   []RejectionReason
	TrustSignalDelta   float64
}

// Featurize calls the memory-router and translates the report.
//
// Severity mapping:
//   - severity=info  → not surfaced (the rubric already has too much info)
//   - severity=warn  → RejectionReason severity="info", trust signal -0.05 per
//   - severity=error → RejectionReason severity="warn", trust signal -0.20 per
//
// The featurizer never produces a severity="error" reason. Memory-layer
// violations are advisory until the Phase-7 rule_machine matcher lands;
// a Phase-5 false-positive in the convention scope must never block a
// promotion.
func (f *MemoryComplianceFeaturizer) Featurize(ctx context.Context, req ComplianceRequest) (Features, error) {
	if f.Disabled || f.Bridge == nil {
		return Features{}, nil
	}
	report, err := f.Bridge.CheckCompliance(ctx, req)
	if err != nil {
		// Non-fatal — log via the returned features, do not fail the
		// verifier. The Phase-4 trust signals carry the load if memory
		// is unavailable.
		return Features{}, fmt.Errorf("memory-compliance featurizer: %w", err)
	}
	feats := Features{ConventionsChecked: report.ConventionsChecked}
	var seen = map[string]bool{}
	for _, v := range report.Violations {
		key := v.ConventionID + "::" + v.OffendingFile
		if seen[key] {
			continue
		}
		seen[key] = true
		switch strings.ToLower(v.Severity) {
		case "warn":
			feats.WarnViolations++
			feats.TrustSignalDelta -= 0.05
			feats.RejectionReasons = append(feats.RejectionReasons, RejectionReason{
				Category:     "memory_convention_warn",
				Severity:     "info",
				Detail:       fmt.Sprintf("Convention %q applies to %s: %s", v.ConventionID, v.OffendingFile, v.RuleNl),
				SuggestedFix: "Confirm the diff respects the team convention.",
			})
		case "error":
			feats.ErrorViolations++
			feats.TrustSignalDelta -= 0.20
			feats.RejectionReasons = append(feats.RejectionReasons, RejectionReason{
				Category:     "memory_convention_violation",
				Severity:     "warn",
				Detail:       fmt.Sprintf("High-confidence convention %q likely violated in %s: %s", v.ConventionID, v.OffendingFile, v.RuleNl),
				SuggestedFix: "Review the rule and confirm; if the rule is stale, drift-review it in the web console.",
			})
		}
	}
	// Floor the delta so a noisy convention set doesn't zero out the
	// criterion. The rubric weight on trust_signal_alignment is 0.10
	// so -0.20 max corresponds to a -0.02 contribution to the final
	// score — order-of-magnitude correct for an advisory signal.
	if feats.TrustSignalDelta < -0.20 {
		feats.TrustSignalDelta = -0.20
	}
	return feats, nil
}

// ApplyToScore folds the featurizer's delta into a Score in place. The
// caller invokes this after Judge.Score returns successfully.
func ApplyToScore(s *Score, feats Features) {
	if s == nil {
		return
	}
	if s.Subscores == nil {
		s.Subscores = map[string]float64{}
	}
	base := s.Subscores["trust_signal_alignment"]
	adjusted := base + feats.TrustSignalDelta
	if adjusted < 0 {
		adjusted = 0
	}
	if adjusted > 1 {
		adjusted = 1
	}
	s.Subscores["trust_signal_alignment"] = adjusted
	s.RejectionReasons = append(s.RejectionReasons, feats.RejectionReasons...)
	// Recompute composite from subscores.
	s.Score = recomputeScore(s.Subscores, DefaultCriteria)
	s.Passed = s.Score >= s.Threshold && countErrors(s.RejectionReasons) == 0
}

func recomputeScore(subscores map[string]float64, weights []Criterion) float64 {
	var sum float64
	for _, c := range weights {
		sum += c.Weight * subscores[c.Name]
	}
	return sum
}
