package vectorstore

import (
	"context"
	"errors"
	"testing"

	cruciblev1 "github.com/crucible/sdk-go/crucible/v1"

	"github.com/crucible/memory-router/internal/embedding"
)

func TestSearch_RefusesEmptyTenant(t *testing.T) {
	s := NewFake()
	_, err := s.Search(context.Background(), Query{Kind: KindBoth})
	if !errors.Is(err, ErrEmptyTenant) {
		t.Fatalf("want ErrEmptyTenant; got %v", err)
	}
}

func TestSearch_DoesNotReturnOtherTenants(t *testing.T) {
	s := NewFake()
	ctx := context.Background()

	// Tenant A writes a memory.
	memA := cruciblev1.Memory{Kind: cruciblev1.MemEpisodic, Content: "A's secret"}
	vecA := mkVec("ten_a", "A's secret")
	if _, err := s.Write(ctx, memA, vecA, "ten_a", "repo1"); err != nil {
		t.Fatal(err)
	}

	// Tenant B searches with the same vector.
	hits, err := s.Search(ctx, Query{TenantID: "ten_b", Kind: KindBoth, Embedding: vecA, TopK: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 0 {
		t.Fatalf("tenant B must not see tenant A's memory; got %d hits", len(hits))
	}
}

func TestSearch_ReturnsScored(t *testing.T) {
	s := NewFake()
	ctx := context.Background()
	for i, content := range []string{"alpha", "beta", "gamma"} {
		_ = i
		_, _ = s.Write(ctx, cruciblev1.Memory{Kind: cruciblev1.MemEpisodic, Content: content},
			mkVec("ten_a", content), "ten_a", "repo")
	}
	hits, err := s.Search(ctx, Query{
		TenantID:  "ten_a",
		Kind:      KindBoth,
		Embedding: mkVec("ten_a", "alpha"),
		TopK:      2,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 2 {
		t.Fatalf("topK=2; got %d", len(hits))
	}
	if hits[0].Memory.Content != "alpha" {
		t.Fatalf("exact-match should rank first; got %q", hits[0].Memory.Content)
	}
}

func TestReinforceOnAccess_IncrementsCount(t *testing.T) {
	s := NewFake()
	ctx := context.Background()
	id, err := s.Write(ctx, cruciblev1.Memory{Kind: cruciblev1.MemEpisodic, Content: "x"},
		mkVec("ten_a", "x"), "ten_a", "")
	if err != nil {
		t.Fatal(err)
	}
	if err := s.ReinforceOnAccess(ctx, "ten_a", id); err != nil {
		t.Fatal(err)
	}
	hits, _ := s.Search(ctx, Query{TenantID: "ten_a", Kind: KindBoth, Embedding: mkVec("ten_a", "x"), TopK: 1})
	if hits[0].RecallCount != 1 {
		t.Fatalf("recall count should be 1; got %d", hits[0].RecallCount)
	}
}

func TestSearch_KindFilter(t *testing.T) {
	s := NewFake()
	ctx := context.Background()
	_, _ = s.Write(ctx, cruciblev1.Memory{Kind: cruciblev1.MemEpisodic, Content: "ep"},
		mkVec("ten_a", "ep"), "ten_a", "")
	_, _ = s.Write(ctx, cruciblev1.Memory{Kind: cruciblev1.MemSemantic, Content: "sem"},
		mkVec("ten_a", "sem"), "ten_a", "")

	hits, _ := s.Search(ctx, Query{TenantID: "ten_a", Kind: KindEpisodic, Embedding: mkVec("ten_a", "x"), TopK: 10})
	for _, h := range hits {
		if h.Memory.Kind != cruciblev1.MemEpisodic {
			t.Fatalf("kind filter leaked %q", h.Memory.Kind)
		}
	}
}

func mkVec(tenant, content string) embedding.Vector {
	c := embedding.NewFake()
	out, _ := c.Embed(context.Background(), []embedding.Request{{TenantID: tenant, Content: content}})
	return out[0]
}
