// Package suggest produces "good first task" recommendations the
// onboarding flow surfaces to the customer after Cartography.
//
// The brief asks for "a first-task suggestion engine (analyze
// Cartographer output, suggest 3 good-first-tasks specific to the
// customer's codebase)". We pick suggestions that:
//
//   - Touch a small, well-bounded surface.
//   - Are typical "good-first-issue" patterns (rate-limit, idempotency-
//     key, structured-logging conversion, dependency bump, README
//     update).
//   - Map to the conventions the cartographer just learned, so the
//     customer can SEE the inferred conventions paying off on their
//     real diff.
//
// The engine is deterministic and explainable — every suggestion
// carries a `WhySafeFirst` rationale.
package suggest

import (
	"strings"

	"github.com/crucible/apps/cartographer/internal/symbols"
	"github.com/crucible/apps/cartographer/internal/types"
)

// Suggest returns up to N first-task suggestions.
func Suggest(stackPrimary string, idx *symbols.Index, conventions []types.ConventionCandidate, n int) []types.FirstTaskSuggestion {
	if n <= 0 {
		n = 3
	}
	pool := candidatePool(stackPrimary, idx, conventions)
	if len(pool) > n {
		pool = pool[:n]
	}
	return pool
}

func candidatePool(stack string, idx *symbols.Index, cs []types.ConventionCandidate) []types.FirstTaskSuggestion {
	var out []types.FirstTaskSuggestion

	// 1. Webhook idempotency-key check — if we see "webhook" symbols.
	if symbolMatch(idx, "webhook") || symbolMatch(idx, "Webhook") {
		out = append(out, types.FirstTaskSuggestion{
			Title:        "Add an idempotency-key check to your webhook handler.",
			Rationale:    "We saw a webhook handler in your repo. Idempotency-key gating is a standard hardening that's small, well-tested, and visible in production logs.",
			Touches:      []string{"**/webhook*.{ts,go,py,rs,java}", "db/migrations/*"},
			EstUSD:       1.6,
			EstWallMin:   12,
			Complexity:   "small",
			WhySafeFirst: "Single endpoint touched; no migration if you already have an idempotency_keys table; verifier catches double-charge regressions immediately.",
		})
	}

	// 2. Structured-logging conversion — if we have prints in Python or
	// fmt.Printf in Go non-test code.
	if symbolMatch(idx, "print") || stack == "go-services" {
		out = append(out, types.FirstTaskSuggestion{
			Title:        "Convert one module's logging to structured slog/structlog.",
			Rationale:    "Crucible's verifier checks structured-logging conventions; converting one module is the cheapest way to confirm the rubric is dialled in for your team.",
			Touches:      []string{"**/*.{go,py}"},
			EstUSD:       1.4,
			EstWallMin:   10,
			Complexity:   "small",
			WhySafeFirst: "Mechanical refactor; no behaviour change; type-check passes are the gate.",
		})
	}

	// 3. README quickstart update — universal, very safe.
	out = append(out, types.FirstTaskSuggestion{
		Title:        "Refresh your README quickstart against your current setup.",
		Rationale:    "Quickstarts drift the moment a dep or env var changes. We can verify a clean-machine `make quickstart` end-to-end inside the twin.",
		Touches:      []string{"README.md", "Makefile"},
		EstUSD:       0.9,
		EstWallMin:   8,
		Complexity:   "small",
		WhySafeFirst: "Docs change only; verifier runs the quickstart in the twin to confirm correctness.",
	})

	// 4. Cursor-pagination conversion — if we found that convention.
	if convMatch(cs, "cursor pagination") {
		out = append(out, types.FirstTaskSuggestion{
			Title:        "Convert one offset-paginated endpoint to cursor pagination.",
			Rationale:    "Your codebase already prefers cursor pagination; converting one offset endpoint demonstrates the convention applied end-to-end.",
			Touches:      []string{"api/**/*.{go,ts,py}", "db/queries/**/*"},
			EstUSD:       1.8,
			EstWallMin:   18,
			Complexity:   "medium",
			WhySafeFirst: "Single endpoint; existing cursor-pagination helpers reused; verifier ensures response shape unchanged for callers.",
		})
	}

	// 5. Lint-config tightening — universal.
	if convMatch(cs, "strict") || stack == "" {
		out = append(out, types.FirstTaskSuggestion{
			Title:        "Tighten one lint rule and fix the resulting failures.",
			Rationale:    "Quickest way to feel the verifier on your code: pick a low-stakes lint rule that's currently warn-only, raise it to error, fix what falls out.",
			Touches:      []string{"**/*"},
			EstUSD:       1.2,
			EstWallMin:   12,
			Complexity:   "small",
			WhySafeFirst: "Mechanical fix; verifier confirms test suite still passes.",
		})
	}

	return out
}

func symbolMatch(idx *symbols.Index, needle string) bool {
	if idx == nil {
		return false
	}
	for name := range idx.ByName {
		if strings.Contains(name, needle) {
			return true
		}
	}
	return false
}

func convMatch(cs []types.ConventionCandidate, needle string) bool {
	low := strings.ToLower(needle)
	for _, c := range cs {
		if strings.Contains(strings.ToLower(c.RuleNL), low) {
			return true
		}
	}
	return false
}
