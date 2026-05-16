package modelrouter

import (
	"context"
	"strings"
	"testing"
)

func TestDefaultModels_AllTiersCovered(t *testing.T) {
	// Each major executor tier must have a primary.
	for _, tier := range []ModelTier{Tier0, Tier1, Tier2} {
		m, err := PrimaryForTier(tier)
		if err != nil {
			t.Fatalf("PrimaryForTier(%d): %v", tier, err)
		}
		if m.ID == "" || m.Vendor == "" {
			t.Fatalf("primary for tier %d missing fields: %+v", tier, m)
		}
	}
}

func TestPrimaryForTier_RejectsTier3And4(t *testing.T) {
	if _, err := PrimaryForTier(Tier3); err == nil {
		t.Fatal("expected error for Tier 3 (verifier role)")
	}
	if _, err := PrimaryForTier(Tier4); err == nil {
		t.Fatal("expected error for Tier 4 (local) in Phase 1")
	}
}

func TestCrossFamilyVerifier_DifferentVendor(t *testing.T) {
	cases := []string{"claude-opus-4-7", "gpt-5.5", "gemini-3.1-pro"}
	for _, id := range cases {
		exec, err := Lookup(id)
		if err != nil {
			t.Fatalf("Lookup(%s): %v", id, err)
		}
		v, err := CrossFamilyVerifier(exec)
		if err != nil {
			t.Fatalf("CrossFamilyVerifier(%s): %v", id, err)
		}
		if v.Vendor == exec.Vendor {
			t.Fatalf("ADR-002 violation: %s → verifier vendor matches (%s)", id, v.Vendor)
		}
	}
}

func TestLookup_CaseTolerant(t *testing.T) {
	m, err := Lookup("Claude-Opus-4-7")
	if err != nil {
		t.Fatalf("case-tolerant lookup failed: %v", err)
	}
	if m.ID != "claude-opus-4-7" {
		t.Fatalf("wrong model returned: %+v", m)
	}
}

func TestLookup_RejectsUnknown(t *testing.T) {
	if _, err := Lookup("not-a-model"); err == nil {
		t.Fatal("expected error for unknown model")
	}
}

func TestCostMath_TierMonotonic(t *testing.T) {
	// Opus output is more expensive than Sonnet output is more expensive than Haiku output.
	haiku := DefaultModels["claude-haiku-4-5"].EstimatedCostUSD(1_000_000, 0, 1_000_000, "")
	sonnet := DefaultModels["claude-sonnet-4-6"].EstimatedCostUSD(1_000_000, 0, 1_000_000, "")
	opus := DefaultModels["claude-opus-4-7"].EstimatedCostUSD(1_000_000, 0, 1_000_000, "")
	if !(haiku < sonnet && sonnet < opus) {
		t.Fatalf("expected monotonic cost: haiku=%.2f sonnet=%.2f opus=%.2f", haiku, sonnet, opus)
	}
}

func TestCostMath_CacheHitCheaperThanFresh(t *testing.T) {
	m := DefaultModels["claude-opus-4-7"]
	fresh := m.EstimatedCostUSD(1_000_000, 0, 0, "")
	cached := m.EstimatedCostUSD(0, 1_000_000, 0, "1h")
	if !(cached < fresh) {
		t.Fatalf("cache must be cheaper than fresh: fresh=%.4f cached=%.4f", fresh, cached)
	}
	cached5m := m.EstimatedCostUSD(0, 1_000_000, 0, "5m")
	if !(cached5m < cached) {
		t.Fatalf("5m cache write should be cheaper than 1h: 5m=%.4f 1h=%.4f", cached5m, cached)
	}
}

func TestEnvVarFor_Anthropic(t *testing.T) {
	if s := envVarFor(VendorAnthropic); s != "ANTHROPIC_API_KEY" {
		t.Fatalf("expected ANTHROPIC_API_KEY, got %s", s)
	}
	if s := envVarFor(VendorOpenAI); s != "OPENAI_API_KEY" {
		t.Fatalf("expected OPENAI_API_KEY, got %s", s)
	}
	if s := envVarFor(VendorGoogle); !strings.Contains(s, "GOOGLE_API_KEY") {
		t.Fatalf("expected GOOGLE_API_KEY mention, got %s", s)
	}
}

func TestEstimateCostUSD_KnownAndUnknown(t *testing.T) {
	u := Usage{InputTokensFresh: 1_000_000, OutputTokens: 500_000}
	if EstimateCostUSD("claude-opus-4-7", u) <= 0 {
		t.Fatal("expected positive cost for opus 1M fresh + 500K out")
	}
	if EstimateCostUSD("does-not-exist", u) != 0 {
		t.Fatal("expected zero cost for unknown model")
	}
}

func TestRouter_RoutesByVendor(t *testing.T) {
	r := NewRouter(fakeClient{v: VendorAnthropic})
	resp, err := r.Call(context.Background(), Request{
		Model:    "claude-haiku-4-5",
		Messages: []Message{{Role: RoleUser, Content: "hi"}},
	})
	if err != nil {
		t.Fatalf("Call: %v", err)
	}
	if resp.Content != "pong" {
		t.Fatalf("expected pong, got %s", resp.Content)
	}
}

func TestRouter_RejectsMissingVendor(t *testing.T) {
	r := NewRouter() // no clients
	_, err := r.Call(context.Background(), Request{
		Model:    "claude-haiku-4-5",
		Messages: []Message{{Role: RoleUser, Content: "hi"}},
	})
	if err == nil {
		t.Fatal("expected error when no client registered for vendor")
	}
}

func TestRouter_RejectsEmptyModel(t *testing.T) {
	r := NewRouter()
	if _, err := r.Call(context.Background(), Request{}); err == nil {
		t.Fatal("expected error on missing model")
	}
}

func TestRouter_Vendors(t *testing.T) {
	r := NewRouter(fakeClient{v: VendorAnthropic}, fakeClient{v: VendorGoogle})
	got := r.Vendors()
	if len(got) != 2 {
		t.Fatalf("expected 2 vendors, got %d", len(got))
	}
}

// fakeClient is a Client double for testing.
type fakeClient struct{ v Vendor }

func (f fakeClient) Vendor() Vendor { return f.v }

func (f fakeClient) Call(_ context.Context, _ Request) (*Response, error) {
	return &Response{
		Content: "pong",
		Usage:   Usage{InputTokensFresh: 5, OutputTokens: 5},
	}, nil
}
