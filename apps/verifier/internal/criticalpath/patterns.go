// Package criticalpath implements the Tier 3 trigger classifier documented in
// docs/06-research/tier3-trigger-automation.md. It ensembles eight signals
// into one weighted-sigmoid score per file and maps the score to a band
// (Cold / Warm / Hot / Molten).
//
// Architecture:
//
//	[file]                  ┌──── path_pattern_score ─┐
//	  │                     ├──── llm_category_score ─┤
//	  ├── Featurizer ──────►├──── fanin_centrality ───┤
//	  │                     ├──── incident_mention ───┤
//	  │                     ├──── slo_backing  ───────┤
//	  │                     ├──── review_intensity ──►├── Weighted ──► sigmoid → band
//	  │                     ├──── cve_history ────────┤   sum
//	  │                     ├──── test_coverage_grad ─┤
//	  │                     ├──── comment_marker ─────┤
//	  │                     ├──── codeowners ─────────┤
//	  │                     └──── ui_or_test_penalty ─┘
//
// Each signal is a Featurizer; the Classifier is the dependency-injected
// composition. This means tests can inject deterministic featurisers and
// production can wire the LLM / OSV-DB / Datadog adapters.
//
// Source of weights: docs/06-research/tier3-trigger-automation.md §"The
// weighted multi-signal score". Defaults are tenant-overridable via
// `crucible calibrate` (see calibrate/).
package criticalpath

import (
	"regexp"
)

// Axis is one of the six orthogonal criticality dimensions.
type Axis string

const (
	AxisSecurity Axis = "security"
	AxisMoney    Axis = "money"
	AxisData     Axis = "data"
	AxisSafety   Axis = "safety"
	AxisHotPath  Axis = "hotpath"
)

// AxisRegex is a per-axis regex from
// docs/06-research/tier3-trigger-automation.md §"Path-pattern heuristics".
// Each pattern is case-insensitive (compiled with `(?i)`).
type AxisRegex struct {
	Axis    Axis
	Pattern *regexp.Regexp
}

// DefaultAxisPatterns is the canonical pattern set. Mutations require an
// ADR + test refresh, since the labeled examples in the docs depend on
// them.
var DefaultAxisPatterns = []AxisRegex{
	{AxisSecurity, regexp.MustCompile(`(?i)\b(auth[nz]?|oauth|saml|jwt|session|login|signin|password|secret|token|cred|kms|kdf|crypto|cipher|sign|verify|hash|mtls|tls|x509|csrf|cors|sanitiz|escape|validate|permit|rbac|acl|policy|capabilit|sandbox)\b`)},
	{AxisMoney, regexp.MustCompile(`(?i)\b(billing|invoice|payment|payout|refund|charge|subscri|ledger|account(ing)?|balance|currency|fx|tax|vat|gst|stripe|adyen|braintree|paypal|wallet|escrow|settle)\b`)},
	{AxisData, regexp.MustCompile(`(?i)\b(migrat|schema|replicat|snapshot|backup|restore|audit_?log|gdpr|pii|consensus|raft|paxos|leader|quorum|checksum|wal|journal|fsync)\b`)},
	{AxisSafety, regexp.MustCompile(`(?i)\b(asil|sil[1-4]|do178|iec6\d{3}|hipaa|hitrust|fda|iso26262|misra|safety|interlock|estop|failsafe)\b`)},
	{AxisHotPath, regexp.MustCompile(`(?i)\b(hot|fast_?path|inner_?loop|simd|vectoriz|kernel)\b`)},
}

// CommentMarkers — files containing ≥2 of these become candidates.
var CommentMarkers = []*regexp.Regexp{
	regexp.MustCompile(`\bDANGER\b`),
	regexp.MustCompile(`DO NOT TOUCH`),
	regexp.MustCompile(`HERE BE DRAGONS`),
	regexp.MustCompile(`//\s*HACK`),
	regexp.MustCompile(`#\s*HACK`),
	regexp.MustCompile(`FIXME\s+critical`),
	regexp.MustCompile(`XXX\s+security`),
	regexp.MustCompile(`@critical\b`),
	regexp.MustCompile(`@dangerous\b`),
	regexp.MustCompile(`WARNING:`),
	regexp.MustCompile(`TODO\(security\)`),
	regexp.MustCompile(`SECURITY:`),
	regexp.MustCompile(`THREAD-SAFETY:`),
	regexp.MustCompile(`INVARIANT:`),
}

// UIOrTestPathPatterns penalise pure UI/test files via the -0.5 weight term.
var UIOrTestPathPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)(^|/)(test|tests|__tests__|spec|specs)/`),
	regexp.MustCompile(`(?i)_(test|spec)\.(go|py|ts|tsx|js|jsx|rs|java|swift|kt)$`),
	regexp.MustCompile(`(?i)\.(test|spec)\.(ts|tsx|js|jsx)$`),
	regexp.MustCompile(`(?i)\bweb/components/`),
	regexp.MustCompile(`(?i)\bui/(components|stories)/`),
	regexp.MustCompile(`(?i)\bmarketing/`),
	regexp.MustCompile(`(?i)\bdemos?/`),
	regexp.MustCompile(`(?i)_for_demos?\.`),
}

// SimulatorOrSandboxPatterns identify files that the LLM judge MUST
// down-weight: payment_simulator_for_demos.py, fixtures, etc. These
// are the "adversarial mislabel" cases in the doc.
var SimulatorOrSandboxPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)simulator`),
	regexp.MustCompile(`(?i)fixture`),
	regexp.MustCompile(`(?i)mock_`),
	regexp.MustCompile(`(?i)_mock\.`),
	regexp.MustCompile(`(?i)/sandboxes?/`),
}
