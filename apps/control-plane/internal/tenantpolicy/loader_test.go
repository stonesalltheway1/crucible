package tenantpolicy

import (
	"testing"

	"github.com/crucible/control-plane/internal/modelrouter"
)

func TestDefault_HasSaneFields(t *testing.T) {
	p := Default("ten_a")
	if p.TenantID != "ten_a" {
		t.Fatalf("tenant id not propagated: %s", p.TenantID)
	}
	if p.DefaultCostCapUSD <= 0 {
		t.Fatalf("expected positive default cost cap, got %v", p.DefaultCostCapUSD)
	}
	if p.DefaultRetryCapPerSubgoal != 3 {
		t.Fatalf("ADR-009 default of 3 expected, got %d", p.DefaultRetryCapPerSubgoal)
	}
	if !p.AllowsVendor(modelrouter.VendorAnthropic) {
		t.Fatal("default should allow Anthropic")
	}
}

func TestAllowsVendor_DeniesWhenNotInList(t *testing.T) {
	p := Default("ten_a")
	p.AllowedVendors = []modelrouter.Vendor{modelrouter.VendorAnthropic}
	if p.AllowsVendor(modelrouter.VendorOpenAI) {
		t.Fatal("OpenAI must be denied when not in allowlist")
	}
	if !p.AllowsVendor(modelrouter.VendorAnthropic) {
		t.Fatal("Anthropic must be allowed")
	}
}

func TestAllowsVendor_EmptyListAllowsAll(t *testing.T) {
	p := Default("ten_a")
	p.AllowedVendors = nil
	if !p.AllowsVendor(modelrouter.VendorOpenAI) {
		t.Fatal("empty allowlist must allow any vendor")
	}
}

func TestLoaderSetAndGet_FallsBackToDefault(t *testing.T) {
	l := NewLoader()
	p := l.Get("unseen")
	if p.TenantID != "unseen" {
		t.Fatal("Get should return Default for unknown tenant")
	}
	custom := Default("known")
	custom.DefaultCostCapUSD = 99
	if err := l.Set(custom); err != nil {
		t.Fatal(err)
	}
	got := l.Get("known")
	if got.DefaultCostCapUSD != 99 {
		t.Fatalf("expected cap 99, got %v", got.DefaultCostCapUSD)
	}
}

func TestLoader_RejectsEmptyTenant(t *testing.T) {
	l := NewLoader()
	if err := l.Set(Policy{}); err == nil {
		t.Fatal("expected error on empty tenant id")
	}
}

func TestLoader_All(t *testing.T) {
	l := NewLoader()
	_ = l.Set(Default("a"))
	_ = l.Set(Default("b"))
	if got := l.All(); len(got) != 2 {
		t.Fatalf("expected 2 policies, got %d", len(got))
	}
}

func TestOverride(t *testing.T) {
	p := Default("ten_a")
	p.ModelOverrides[modelrouter.Tier1] = "gpt-5.3-codex"
	if p.Override(modelrouter.Tier1) != "gpt-5.3-codex" {
		t.Fatal("override not surfaced")
	}
	if p.Override(modelrouter.Tier2) != "" {
		t.Fatal("expected empty for non-overridden tier")
	}
}
