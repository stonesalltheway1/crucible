package proceduralstore

import (
	"context"
	"errors"
	"testing"
	"time"

	memoryspec "github.com/crucible/memory-spec/go"
	cruciblev1 "github.com/crucible/sdk-go/crucible/v1"
)

func mkConv(tenant, id, rule string, layer memoryspec.MemoryLayer) memoryspec.Convention {
	now := time.Now().UTC()
	return memoryspec.Convention{
		ID:         id,
		TenantID:   tenant,
		Layer:      layer,
		Scope:      cruciblev1.ScopeFilter{FileGlob: "api/**/*.ts"},
		RuleNl:     rule,
		Category:   memoryspec.CatLogging,
		Status:     memoryspec.StatusActive,
		Confidence: 0.7,
		FirstSeen:  now.Add(-7 * 24 * time.Hour),
		ValidFrom:  now,
		WrittenAt:  now,
	}
}

func TestUpsert_RefusesEmptyTenant(t *testing.T) {
	s := NewFake()
	c := mkConv("", "conv_1", "rule", memoryspec.LayerOrgOverrides)
	_, err := s.Upsert(context.Background(), c)
	if !errors.Is(err, ErrEmptyTenant) {
		t.Fatalf("want ErrEmptyTenant; got %v", err)
	}
}

func TestUpsert_RejectsInvalidCategory(t *testing.T) {
	s := NewFake()
	c := mkConv("ten_a", "conv_1", "rule", memoryspec.LayerOrgOverrides)
	c.Category = "Other"
	if _, err := s.Upsert(context.Background(), c); err == nil {
		t.Fatal("Upsert must validate before persisting")
	}
}

func TestFetchByScope_TenantIsolation(t *testing.T) {
	s := NewFake()
	ctx := context.Background()
	_, _ = s.Upsert(ctx, mkConv("ten_a", "conv_a", "use slog", memoryspec.LayerOrgOverrides))
	_, _ = s.Upsert(ctx, mkConv("ten_b", "conv_b", "use pino", memoryspec.LayerOrgOverrides))

	got, err := s.FetchByScope(ctx, "ten_a", "", cruciblev1.ScopeFilter{FileGlob: "api/handlers/x.ts"}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ID != "conv_a" {
		t.Fatalf("tenant isolation: got %v", got)
	}
}

func TestMarkSuperseded_FlipsStatusAndStampsValidTo(t *testing.T) {
	s := NewFake()
	ctx := context.Background()
	_, _ = s.Upsert(ctx, mkConv("ten_a", "conv_old", "old rule", memoryspec.LayerOrgOverrides))
	_, _ = s.Upsert(ctx, mkConv("ten_a", "conv_new", "new rule", memoryspec.LayerOrgOverrides))

	if err := s.MarkSuperseded(ctx, "ten_a", "conv_old", "conv_new"); err != nil {
		t.Fatal(err)
	}
	old, err := s.FetchByID(ctx, "ten_a", "conv_old")
	if err != nil {
		t.Fatal(err)
	}
	if old.Status != memoryspec.StatusSuperseded {
		t.Fatalf("status should be superseded; got %q", old.Status)
	}
	if old.ValidTo == nil {
		t.Fatal("ValidTo must be stamped on supersession")
	}
}

func TestFetchByScope_OmitsSupersededAndRejected(t *testing.T) {
	s := NewFake()
	ctx := context.Background()
	c := mkConv("ten_a", "conv_1", "rule", memoryspec.LayerOrgOverrides)
	c.Status = memoryspec.StatusSuperseded
	_, _ = s.Upsert(ctx, c)
	got, _ := s.FetchByScope(ctx, "ten_a", "", cruciblev1.ScopeFilter{FileGlob: "api/x.ts"}, 0)
	if len(got) != 0 {
		t.Fatalf("superseded conventions must be filtered out; got %d", len(got))
	}
}

func TestDriftRatio_CountsReinforceVsViolate(t *testing.T) {
	s := NewFake()
	ctx := context.Background()
	_, _ = s.Upsert(ctx, mkConv("ten_a", "conv_1", "rule", memoryspec.LayerOrgOverrides))
	for i := 0; i < 7; i++ {
		_ = s.RecordReinforcement(ctx, "ten_a", "conv_1", cruciblev1.SourceRef{Kind: cruciblev1.SourceRefPrComment, PR: uint64(i)})
	}
	for i := 0; i < 2; i++ {
		_ = s.RecordViolation(ctx, "ten_a", "conv_1", cruciblev1.SourceRef{Kind: cruciblev1.SourceRefPrComment, PR: uint64(100 + i)})
	}
	pos, neg, err := s.DriftRatio(ctx, "ten_a", "conv_1")
	if err != nil {
		t.Fatal(err)
	}
	if pos != 7 || neg != 2 {
		t.Fatalf("drift counts: pos=%d neg=%d", pos, neg)
	}
}

func TestSanitizeGraphName(t *testing.T) {
	cases := map[string]string{
		"ten_abc":           "tenant_ten_abc",
		"acme/payments":     "tenant_acme_payments",
		"global":            "tenant_global",
		"weird;DROP--TABLE": "tenant_weird_DROP__TABLE",
		"":                  "",
	}
	for in, want := range cases {
		if got := SanitizeGraphName(in); got != want {
			t.Fatalf("SanitizeGraphName(%q) = %q, want %q", in, got, want)
		}
	}
}
