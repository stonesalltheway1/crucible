// Package console formats the web-console output strings the
// onboarding flow streams over SSE while the cartographer is running.
//
// Format mirrors docs/04-operations/onboarding.md §Stage 2:
//
//   ✓ Indexed 1,247 files across 38 directories.
//   ✓ Detected stack: Next.js 14, FastAPI 0.110, PostgreSQL 16.
//   ✓ Extracted 184 conventions from your existing config + AGENTS.md.
//   ✓ Loaded 312 OSS-derived defaults for your stack.
//   ✓ Inferred 47 additional conventions from your PR review history.
//     - 12 high-confidence (recommended; surfaced active)
//     - 23 medium-confidence (surfaced as suggestions)
//     - 12 low-confidence (stored as candidates; not surfaced yet)
package console

import (
	"fmt"

	"github.com/crucible/apps/cartographer/internal/types"
)

// Lines builds the canonical web-console line set from a result.
func Lines(r *types.CartographyResult) []string {
	if r == nil {
		return nil
	}
	out := []string{
		fmt.Sprintf("✓ Indexed %s files across %d directories.", commas(r.FilesIndexed), r.Directories),
	}
	if r.StackPrimary != "" {
		stackLine := "✓ Detected stack: " + r.StackPrimary
		if len(r.StackSecondary) > 0 {
			stackLine += " (also: "
			for i, s := range r.StackSecondary {
				if i > 0 {
					stackLine += ", "
				}
				stackLine += s
			}
			stackLine += ")"
		}
		out = append(out, stackLine+".")
	}

	configsAndStated := r.ConventionsFromConfigs + r.ConventionsFromAgentsMD + r.ConventionsFromContributing + r.ConventionsFromADRs
	if configsAndStated > 0 {
		out = append(out, fmt.Sprintf("✓ Extracted %d conventions from your existing config + AGENTS.md / ADRs.", configsAndStated))
	}
	if r.ConventionsFromOSSDefaults > 0 {
		out = append(out, fmt.Sprintf("✓ Loaded %d OSS-derived defaults for your stack.", r.ConventionsFromOSSDefaults))
	}

	pr := r.ConventionsFromPRReview + r.ConventionsFromIncidents
	if pr > 0 {
		out = append(out, fmt.Sprintf("✓ Inferred %d additional conventions from your PR review + incident history.", pr))
		out = append(out, fmt.Sprintf("  - %d high-confidence (recommended; surfaced active)", r.HighConfidenceCount))
		out = append(out, fmt.Sprintf("  - %d medium-confidence (surfaced as suggestions)", r.MediumConfidenceCount))
		out = append(out, fmt.Sprintf("  - %d low-confidence (stored as candidates; not surfaced yet)", r.LowConfidenceCount))
	}

	if r.HasCustomerOverride {
		out = append(out, fmt.Sprintf("✓ Found customer override at %s — your rules take precedence.", r.CustomerOverridePath))
	} else {
		out = append(out, "ℹ No AGENTS.md / CLAUDE.md / .cursorrules found — generated an inferred draft for review.")
	}

	if r.WallClockSeconds > 0 {
		out = append(out, fmt.Sprintf("✓ Cartography complete in %.1fs ($%.2f spent).", r.WallClockSeconds, r.UsdSpent))
	}

	out = append(out, "Review at: https://app.crucible.dev/memory")
	return out
}

func commas(n int) string {
	if n < 1000 {
		return itoa(n)
	}
	s := itoa(n)
	out := make([]byte, 0, len(s)+len(s)/3)
	rem := len(s) % 3
	if rem > 0 {
		out = append(out, s[:rem]...)
		if len(s) > rem {
			out = append(out, ',')
		}
	}
	for i := rem; i < len(s); i += 3 {
		out = append(out, s[i:i+3]...)
		if i+3 < len(s) {
			out = append(out, ',')
		}
	}
	return string(out)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	neg := n < 0
	if neg {
		n = -n
	}
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
