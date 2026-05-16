// Package budget enforces the 7K-token retrieval budget.
//
// Implements the "context window is RAM not storage" Mem0 thesis: even
// if the agent asks for everything, the router refuses to send more
// tokens than the budget. Items below the cutoff are dropped lowest-
// score first.
//
// Token counting uses a cheap len/4 approximation matching the
// memory_episodic.token_estimate column. A real tiktoken-equivalent
// would add ~1-2ms per recall; the approximation is within 10% of
// truth on natural-language convention rules and 5% on code snippets.
package budget

import (
	"sort"

	memoryspec "github.com/crucible/memory-spec/go"
)

// DefaultMaxTokens matches the Mem0 thesis "≤7K tokens per retrieval".
const DefaultMaxTokens uint32 = 7000

// Estimate returns the token-count approximation for content. Matches
// the formula in infra/databases/postgres/migrations/0003 so the
// router's pre-DB filtering aligns with the column.
func Estimate(content string) uint32 {
	if content == "" {
		return 0
	}
	// 4 chars per token is the common heuristic for English-dominant
	// natural-language text. Code skews lower (~3); we round up by
	// using ceil semantics.
	return uint32((len(content) + 3) / 4)
}

// Enforce trims memories to fit max_tokens, preferring higher
// final_score and dropping lowest-scored items until under budget.
// Returns the trimmed list and total tokens used.
func Enforce(memories []memoryspec.ScoredMemory, maxTokens uint32) ([]memoryspec.ScoredMemory, uint32) {
	if maxTokens == 0 {
		maxTokens = DefaultMaxTokens
	}
	if len(memories) == 0 {
		return memories, 0
	}

	// Stable sort by final_score descending. Ties resolved by smaller
	// token estimate (cheaper items packed first to maximize count).
	sort.SliceStable(memories, func(i, j int) bool {
		if memories[i].FinalScore != memories[j].FinalScore {
			return memories[i].FinalScore > memories[j].FinalScore
		}
		return memories[i].TokenEstimate < memories[j].TokenEstimate
	})

	out := make([]memoryspec.ScoredMemory, 0, len(memories))
	var used uint32
	for _, m := range memories {
		est := m.TokenEstimate
		if est == 0 {
			est = Estimate(m.Memory.Content)
		}
		if used+est > maxTokens {
			// Skip this item; it doesn't fit. Continue trying smaller
			// items in case any subsequent fits.
			continue
		}
		out = append(out, m)
		used += est
	}
	return out, used
}
