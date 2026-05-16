package federation

import (
	"testing"
	"time"

	memoryspec "github.com/crucible/memory-spec/go"
	cruciblev1 "github.com/crucible/sdk-go/crucible/v1"
)

func mkConv(tenant, id, rule string, cat memoryspec.ConventionCategory) memoryspec.Convention {
	return memoryspec.Convention{
		ID:         id,
		TenantID:   tenant,
		Layer:      memoryspec.LayerOrgOverrides,
		Scope:      cruciblev1.ScopeFilter{},
		RuleNl:     rule,
		Category:   cat,
		Status:     memoryspec.StatusActive,
		Confidence: 0.8,
		ValidFrom:  time.Now().UTC(),
		WrittenAt:  time.Now().UTC(),
	}
}

func TestAnonymousForm_SameRuleAcrossTenantsHashesIdentically(t *testing.T) {
	_, a := AnonymousForm(memoryspec.CatLogging, "Our team's rule is structured slog calls only.")
	_, b := AnonymousForm(memoryspec.CatLogging, "this project's rule is structured slog calls only")
	if a != b {
		t.Fatalf("paraphrases should hash identically; %q vs %q", a, b)
	}
}

func TestAnonymousForm_DistinctRulesDistinctHashes(t *testing.T) {
	_, a := AnonymousForm(memoryspec.CatLogging, "Structured slog calls only.")
	_, b := AnonymousForm(memoryspec.CatNaming, "Tests end in _test.go.")
	if a == b {
		t.Fatal("distinct rules must have distinct hashes")
	}
}

func TestScan_ThresholdEnforced(t *testing.T) {
	d := New()
	in := map[string][]memoryspec.Convention{
		"ten_1": {mkConv("ten_1", "c1", "structured slog calls only", memoryspec.CatLogging)},
		"ten_2": {mkConv("ten_2", "c2", "structured slog calls only", memoryspec.CatLogging)},
		"ten_3": {mkConv("ten_3", "c3", "structured slog calls only", memoryspec.CatLogging)},
	}
	if got := d.Scan(in, nil); len(got) != 0 {
		t.Fatalf("3 tenants below threshold (5); want 0 candidates, got %d", len(got))
	}
}

func TestScan_ProducesCandidateAtFiveTenants(t *testing.T) {
	d := New()
	in := map[string][]memoryspec.Convention{}
	for i := 1; i <= 5; i++ {
		tn := "ten_" + string(rune('A'+i))
		in[tn] = []memoryspec.Convention{mkConv(tn, "c"+string(rune('A'+i)), "Use slog calls only.", memoryspec.CatLogging)}
	}
	got := d.Scan(in, nil)
	if len(got) != 1 {
		t.Fatalf("want 1 candidate at 5 tenants; got %d", len(got))
	}
	if got[0].DistinctTenantCount != 5 {
		t.Fatalf("tenant count: got %d", got[0].DistinctTenantCount)
	}
	if got[0].Fired {
		t.Fatal("Phase 5 must record, NEVER fire graduations")
	}
}

func TestScan_OptedOutTenantsExcluded(t *testing.T) {
	d := New()
	in := map[string][]memoryspec.Convention{}
	for i := 0; i < 6; i++ {
		tn := "ten_" + string(rune('A'+i))
		in[tn] = []memoryspec.Convention{mkConv(tn, "c"+string(rune('A'+i)), "Use slog calls only.", memoryspec.CatLogging)}
	}
	opt := map[string]bool{"ten_A": true, "ten_B": true}
	got := d.Scan(in, opt)
	if len(got) != 1 {
		t.Fatalf("want 1 candidate; got %d", len(got))
	}
	if got[0].DistinctTenantCount != 4 {
		t.Fatalf("opt-outs should not contribute; got %d", got[0].DistinctTenantCount)
	}
}

func TestAnonymousForm_RedactsLikelyServiceNames(t *testing.T) {
	canon, _ := AnonymousForm(memoryspec.CatSecurityDefaults,
		"Auth middleware must precede AcmeAuth.checkPermissions on every endpoint.")
	if containsCaseInsensitive(canon, "acmeauth") || containsCaseInsensitive(canon, "checkPermissions") {
		t.Fatalf("service-shaped identifiers should be redacted; got %q", canon)
	}
}

func containsCaseInsensitive(haystack, needle string) bool {
	return len(haystack) >= len(needle) && (haystack == needle ||
		indexCaseInsensitive(haystack, needle) >= 0)
}

func indexCaseInsensitive(haystack, needle string) int {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		j := 0
		for ; j < len(needle); j++ {
			a, b := haystack[i+j], needle[j]
			if 'A' <= a && a <= 'Z' {
				a += 'a' - 'A'
			}
			if 'A' <= b && b <= 'Z' {
				b += 'a' - 'A'
			}
			if a != b {
				break
			}
		}
		if j == len(needle) {
			return i
		}
	}
	return -1
}
