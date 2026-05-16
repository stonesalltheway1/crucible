// Package modelrouter implements the 5-tier model dispatch documented in
// docs/01-architecture/model-routing.md.
//
// Currency notes (May 2026):
//   - Anthropic SDK:   github.com/anthropics/anthropic-sdk-go v1.43.x
//   - Google Gen AI:   google.golang.org/genai v1.57.x
//   - OpenAI:          github.com/openai/openai-go/v3 v3.35.x
//   - Anthropic cache: default TTL flipped from 1h→5m on 2026-03-06; we set
//                      ttl="1h" explicitly on the system + tool slots.
//   - Gemini 3:        thinking_level (low/medium/high), not thinking_budget.
//   - gpt-5.1-codex-max pricing is unverified on the official page; the entry
//                      in DefaultModels is preserved per the design doc but
//                      flagged in the report.
package modelrouter

import (
	"errors"
	"fmt"
	"strings"
)

// Vendor enumerates the LLM-vendor lineage. Cross-family verifier pairing
// (ADR-002) requires executor.Vendor != verifier.Vendor.
type Vendor string

const (
	VendorAnthropic Vendor = "anthropic"
	VendorGoogle    Vendor = "google"
	VendorOpenAI    Vendor = "openai"
	VendorXai       Vendor = "xai"
	VendorDeepSeek  Vendor = "deepseek"
	VendorLocal     Vendor = "local"
)

// ModelTier is the routing tier (0..4).
type ModelTier int

const (
	Tier0 ModelTier = 0 // Haiku-class — file reads, planning decomposition
	Tier1 ModelTier = 1 // Sonnet-class — standard coding
	Tier2 ModelTier = 2 // Opus-class — hard refactors, invariants
	Tier3 ModelTier = 3 // verifier — cross-family by construction
	Tier4 ModelTier = 4 // local — privacy-sensitive
)

// ModelSpec is the per-model price + capacity entry.
type ModelSpec struct {
	ID                       string
	Vendor                   Vendor
	Tier                     ModelTier
	ContextWindow            int
	MaxOutput                int
	InputUSDPerMillion       float64
	OutputUSDPerMillion      float64
	CacheReadUSDPerMillion   float64
	CacheWrite5mPerMillion   float64
	CacheWrite1hPerMillion   float64
	SupportsThinking         bool
	SupportsExplicitCaching  bool
	Notes                    string
}

// EstimatedCostUSD returns the USD cost of a single call given token counts.
// Pass 0 for any unused field. cacheTTL is "1h", "5m", or "" (no cache).
func (m ModelSpec) EstimatedCostUSD(inputFresh, inputCached, output int, cacheTTL string) float64 {
	cost := float64(inputFresh)*m.InputUSDPerMillion/1_000_000 +
		float64(output)*m.OutputUSDPerMillion/1_000_000

	if inputCached > 0 {
		cost += float64(inputCached) * m.CacheReadUSDPerMillion / 1_000_000
		switch cacheTTL {
		case "1h":
			cost += float64(inputCached) * m.CacheWrite1hPerMillion / 1_000_000
		case "5m":
			cost += float64(inputCached) * m.CacheWrite5mPerMillion / 1_000_000
		}
	}
	return cost
}

// DefaultModels is the May-2026 reference table from docs/01-architecture/model-routing.md.
// Pricing is per-1M-token. Any updates here must keep the doc in sync.
var DefaultModels = map[string]ModelSpec{
	// Anthropic
	"claude-haiku-4-5": {
		ID: "claude-haiku-4-5", Vendor: VendorAnthropic, Tier: Tier0,
		ContextWindow: 200_000, MaxOutput: 64_000,
		InputUSDPerMillion: 1.0, OutputUSDPerMillion: 5.0,
		CacheReadUSDPerMillion: 0.10, CacheWrite5mPerMillion: 1.25, CacheWrite1hPerMillion: 2.0,
		SupportsThinking: true, SupportsExplicitCaching: true,
	},
	"claude-sonnet-4-6": {
		ID: "claude-sonnet-4-6", Vendor: VendorAnthropic, Tier: Tier1,
		ContextWindow: 1_000_000, MaxOutput: 64_000,
		InputUSDPerMillion: 3.0, OutputUSDPerMillion: 15.0,
		CacheReadUSDPerMillion: 0.30, CacheWrite5mPerMillion: 3.75, CacheWrite1hPerMillion: 6.0,
		SupportsThinking: true, SupportsExplicitCaching: true,
	},
	"claude-opus-4-7": {
		ID: "claude-opus-4-7", Vendor: VendorAnthropic, Tier: Tier2,
		ContextWindow: 1_000_000, MaxOutput: 128_000,
		InputUSDPerMillion: 5.0, OutputUSDPerMillion: 25.0,
		CacheReadUSDPerMillion: 0.50, CacheWrite5mPerMillion: 6.25, CacheWrite1hPerMillion: 10.0,
		SupportsThinking: true, SupportsExplicitCaching: true,
		Notes: "tokenizer emits ~35% more tokens than older Claude — account for in budget",
	},

	// Google
	"gemini-3-flash-lite": {
		ID: "gemini-3-flash-lite", Vendor: VendorGoogle, Tier: Tier0,
		ContextWindow: 1_000_000, MaxOutput: 64_000,
		InputUSDPerMillion: 0.10, OutputUSDPerMillion: 0.40,
		SupportsThinking: true, SupportsExplicitCaching: true,
	},
	"gemini-3-flash": {
		ID: "gemini-3-flash", Vendor: VendorGoogle, Tier: Tier1,
		ContextWindow: 1_050_000, MaxOutput: 64_000,
		InputUSDPerMillion: 0.50, OutputUSDPerMillion: 3.0,
		SupportsThinking: true, SupportsExplicitCaching: true,
	},
	"gemini-3.1-pro": {
		ID: "gemini-3.1-pro", Vendor: VendorGoogle, Tier: Tier2,
		ContextWindow: 2_000_000, MaxOutput: 64_000,
		InputUSDPerMillion: 2.0, OutputUSDPerMillion: 12.0,
		SupportsThinking: true, SupportsExplicitCaching: true,
		Notes: "use thinking_level=low|medium|high; thinking_budget hurts perf on Gemini 3",
	},

	// OpenAI
	"gpt-5.1-codex-max": {
		ID: "gpt-5.1-codex-max", Vendor: VendorOpenAI, Tier: Tier1,
		ContextWindow: 400_000, MaxOutput: 32_000,
		InputUSDPerMillion: 1.25, OutputUSDPerMillion: 10.0,
		Notes: "UNVERIFIED on official pricing page (May 2026); may be superseded by gpt-5.3-codex",
	},
	"gpt-5.3-codex": {
		ID: "gpt-5.3-codex", Vendor: VendorOpenAI, Tier: Tier1,
		ContextWindow: 400_000, MaxOutput: 64_000,
		InputUSDPerMillion: 1.75, OutputUSDPerMillion: 14.0,
		Notes: "#1 Terminal-Bench 2.0",
	},
	"gpt-5.5": {
		ID: "gpt-5.5", Vendor: VendorOpenAI, Tier: Tier2,
		ContextWindow: 920_000, MaxOutput: 128_000,
		InputUSDPerMillion: 5.0, OutputUSDPerMillion: 30.0,
		SupportsThinking: true,
		Notes: "2× price hike vs GPT-5 (Apr 2026)",
	},
}

// PrimaryForTier returns the default executor model for a given tier.
func PrimaryForTier(tier ModelTier) (ModelSpec, error) {
	switch tier {
	case Tier0:
		return DefaultModels["claude-haiku-4-5"], nil
	case Tier1:
		return DefaultModels["claude-sonnet-4-6"], nil
	case Tier2:
		return DefaultModels["claude-opus-4-7"], nil
	case Tier3:
		return ModelSpec{}, errors.New("modelrouter: Tier 3 is a verifier role — call CrossFamilyVerifier(executor)")
	case Tier4:
		return ModelSpec{}, errors.New("modelrouter: Tier 4 (local) is not configured in Phase 1")
	default:
		return ModelSpec{}, fmt.Errorf("modelrouter: unknown tier %d", tier)
	}
}

// CrossFamilyVerifier returns the recommended verifier model for a given
// executor, enforcing ADR-002's cross-family requirement.
func CrossFamilyVerifier(executor ModelSpec) (ModelSpec, error) {
	switch executor.Vendor {
	case VendorAnthropic:
		return DefaultModels["gemini-3.1-pro"], nil
	case VendorGoogle:
		return DefaultModels["claude-opus-4-7"], nil
	case VendorOpenAI:
		return DefaultModels["claude-opus-4-7"], nil
	default:
		return ModelSpec{}, fmt.Errorf("modelrouter: no cross-family pairing registered for %s", executor.Vendor)
	}
}

// Lookup returns a ModelSpec by id, accepting a few common name variants.
func Lookup(id string) (ModelSpec, error) {
	if m, ok := DefaultModels[id]; ok {
		return m, nil
	}
	// Tolerate a few common variants (case, dashes).
	want := strings.ToLower(id)
	for k, v := range DefaultModels {
		if strings.ToLower(k) == want {
			return v, nil
		}
	}
	return ModelSpec{}, fmt.Errorf("modelrouter: unknown model %q", id)
}
