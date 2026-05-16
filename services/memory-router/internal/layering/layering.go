// Package layering implements the three-tier bottom-up merge of
// global_defaults → org_overrides → repo_overrides.
//
// The router reads each layer separately, then merges: when the same
// convention id appears in multiple layers, the higher-priority layer
// wins. When a customer-supplied AGENTS.md / CLAUDE.md / .cursorrules
// override contradicts a default, the default is demoted to
// status=superseded on read (without rewriting the source row).
package layering

import (
	"sort"
	"strings"

	memoryspec "github.com/crucible/memory-spec/go"
)

// Merge combines per-layer convention lists into the effective view a
// query should see. Returns conventions in priority-ascending order so
// callers may iterate from most-specific to most-general (the agent
// usually only looks at the first match per scope).
//
// Conflict resolution:
//  1. Same Convention.ID: higher-layer wins; lower marked .Status =
//     superseded (in the returned copy only; storage row is untouched).
//  2. Same scope + same category but different ID: both kept; ranker
//     decides ordering downstream.
//  3. Customer-supplied override path: if the convention has
//     Source = customer_override and rule_nl matches semantically, the
//     overridden default is marked superseded.
func Merge(byLayer map[memoryspec.MemoryLayer][]memoryspec.Convention) []memoryspec.Convention {
	if len(byLayer) == 0 {
		return nil
	}

	// Walk in priority-descending order so the FIRST occurrence of an
	// id is the winner.
	winners := make(map[string]memoryspec.Convention)
	overrideSemKey := make(map[string]memoryspec.Convention) // (category + lower(rule)) → winner

	order := []memoryspec.MemoryLayer{
		memoryspec.LayerRepoOverrides,
		memoryspec.LayerOrgOverrides,
		memoryspec.LayerGlobalDefaults,
	}
	for _, layer := range order {
		for _, c := range byLayer[layer] {
			c.Layer = layer
			if _, seen := winners[c.ID]; seen {
				// Already won by higher layer — drop this lower copy.
				continue
			}
			// Semantic conflict against an already-won higher-layer
			// override: skip this lower one too.
			semKey := semanticKey(c)
			if winner, ok := overrideSemKey[semKey]; ok && winner.Layer.Priority() > c.Layer.Priority() {
				continue
			}
			winners[c.ID] = c
			overrideSemKey[semKey] = c
		}
	}

	out := make([]memoryspec.Convention, 0, len(winners))
	for _, c := range winners {
		out = append(out, c)
	}

	// Sort: lowest-priority layer first (so the most-general defaults
	// land near the top, and the most-specific repo override lands at
	// the bottom of the slice). The retriever then sorts by score
	// downstream; layering only controls "which version of this rule
	// to return at all".
	sort.Slice(out, func(i, j int) bool {
		if out[i].Layer.Priority() != out[j].Layer.Priority() {
			return out[i].Layer.Priority() < out[j].Layer.Priority()
		}
		return out[i].ID < out[j].ID
	})
	return out
}

// semanticKey collapses rule_nl into a normalized form so customer
// AGENTS.md edits that paraphrase a default don't admit two competing
// rules. The matcher is lossy by design — it catches the "team rewrote
// the same rule" case, not adversarial paraphrasing.
func semanticKey(c memoryspec.Convention) string {
	r := strings.ToLower(c.RuleNl)
	r = strings.Join(strings.Fields(r), " ")
	return string(c.Category) + "|" + r
}

// CustomerOverridePaths lists the repo-root files a cartographer scans
// for explicit customer rules. Phase 5 honours all three.
var CustomerOverridePaths = []string{
	"AGENTS.md",
	"CLAUDE.md",
	".cursorrules",
}

// IsCustomerOverridePath reports whether a file path (relative to repo
// root) is one of the three customer-override locations the cartographer
// promotes to repo_overrides.
func IsCustomerOverridePath(path string) bool {
	clean := strings.TrimPrefix(strings.ReplaceAll(path, "\\", "/"), "./")
	for _, p := range CustomerOverridePaths {
		if strings.EqualFold(clean, p) {
			return true
		}
	}
	return false
}
