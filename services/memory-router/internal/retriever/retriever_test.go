package retriever

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	memoryspec "github.com/crucible/memory-spec/go"
	cruciblev1 "github.com/crucible/sdk-go/crucible/v1"

	"github.com/crucible/memory-router/internal/embedding"
	"github.com/crucible/memory-router/internal/globaldefaults"
	"github.com/crucible/memory-router/internal/hotstore"
	"github.com/crucible/memory-router/internal/proceduralstore"
	"github.com/crucible/memory-router/internal/vectorstore"
)

func newRig() *Retriever {
	hot := hotstore.New(hotstore.NewFake())
	return New(hot, vectorstore.NewFake(), proceduralstore.NewFake(), embedding.NewFake(), globaldefaults.NewLoader())
}

func TestRecall_RefusesEmptyTenant(t *testing.T) {
	r := newRig()
	_, err := r.Recall(context.Background(), memoryspec.RetrievalQuery{IncludeHot: true})
	if !errors.Is(err, ErrEmptyTenant) {
		t.Fatalf("want ErrEmptyTenant; got %v", err)
	}
}

func TestRecall_SurfacesProceduralOverEpisodicOnEqualScore(t *testing.T) {
	r := newRig()
	ctx := context.Background()

	// Write one procedural rule.
	now := time.Now().UTC()
	_, _ = r.Proc.Upsert(ctx, memoryspec.Convention{
		ID:         "conv_1",
		TenantID:   "ten_a",
		Layer:      memoryspec.LayerOrgOverrides,
		Scope:      cruciblev1.ScopeFilter{FileGlob: "api/**/*.go"},
		RuleNl:     "Use context.Context as first parameter in async chains.",
		Category:   memoryspec.CatConcurrency,
		Status:     memoryspec.StatusActive,
		Confidence: 0.9,
		ValidFrom:  now,
		WrittenAt:  now,
	})

	// Write one episodic memory with similar content.
	vec, _ := r.Embedder.Embed(ctx, []embedding.Request{{TenantID: "ten_a", Content: "context.Context first arg"}})
	_, _ = r.Vec.Write(ctx, cruciblev1.Memory{Kind: cruciblev1.MemEpisodic, Content: "context.Context first arg", Importance: 0.5},
		vec[0], "ten_a", "")

	res, err := r.Recall(ctx, memoryspec.RetrievalQuery{
		TenantID:          "ten_a",
		Query:             "what's the convention for context?",
		Scope:             cruciblev1.ScopeFilter{FileGlob: "api/handlers/login.go"},
		IncludeProcedural: true,
		IncludeEpisodic:   true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Memories) < 2 {
		t.Fatalf("expected ≥2 memories; got %d", len(res.Memories))
	}
	if res.Memories[0].Memory.Kind != cruciblev1.MemProcedural {
		t.Fatalf("procedural should outrank episodic on equal semantic; got %q first", res.Memories[0].Memory.Kind)
	}
}

func TestRecall_TenantIsolation_NoLeak(t *testing.T) {
	r := newRig()
	ctx := context.Background()
	now := time.Now().UTC()

	for _, tenant := range []string{"ten_a", "ten_b"} {
		_, _ = r.Proc.Upsert(ctx, memoryspec.Convention{
			ID:         "conv_" + tenant,
			TenantID:   tenant,
			Layer:      memoryspec.LayerOrgOverrides,
			Scope:      cruciblev1.ScopeFilter{FileGlob: "api/**/*.go"},
			RuleNl:     "secret " + tenant,
			Category:   memoryspec.CatLogging,
			Status:     memoryspec.StatusActive,
			Confidence: 0.9,
			ValidFrom:  now,
			WrittenAt:  now,
		})
	}

	res, err := r.Recall(ctx, memoryspec.RetrievalQuery{
		TenantID:          "ten_a",
		Scope:             cruciblev1.ScopeFilter{FileGlob: "api/x.go"},
		IncludeProcedural: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range res.Memories {
		if strings.Contains(m.Memory.Content, "ten_b") {
			t.Fatalf("tenant_b leak: %q", m.Memory.Content)
		}
	}
}

func TestRecall_BudgetEnforced(t *testing.T) {
	r := newRig()
	r.BudgetCap = 200 // very small
	ctx := context.Background()
	now := time.Now().UTC()

	// Pack many rules.
	for i := 0; i < 20; i++ {
		id := []byte("conv_x")
		id = append(id, byte('a'+i))
		_, _ = r.Proc.Upsert(ctx, memoryspec.Convention{
			ID:         string(id),
			TenantID:   "ten_a",
			Layer:      memoryspec.LayerOrgOverrides,
			Scope:      cruciblev1.ScopeFilter{Category: "Logging"},
			RuleNl:     strings.Repeat("x", 240), // ~60 tokens each
			Category:   memoryspec.CatLogging,
			Status:     memoryspec.StatusActive,
			Confidence: 0.6,
			ValidFrom:  now,
			WrittenAt:  now,
		})
	}
	res, err := r.Recall(ctx, memoryspec.RetrievalQuery{
		TenantID:          "ten_a",
		Scope:             cruciblev1.ScopeFilter{Category: "Logging"},
		IncludeProcedural: true,
		MaxTokens:         200,
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.TokensUsed > 200 {
		t.Fatalf("budget breach: %d tokens used", res.TokensUsed)
	}
}

func TestRecall_HotLookupAddsPlanWhenTaskIDProvided(t *testing.T) {
	r := newRig()
	ctx := context.Background()
	_ = r.Hot.SetPlan(ctx, "ten_a", "task_1", `{"steps":["read","write"]}`)
	res, err := r.Recall(ctx, memoryspec.RetrievalQuery{
		TenantID:   "ten_a",
		TaskID:     "task_1",
		Query:      "what's our plan",
		IncludeHot: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, m := range res.Memories {
		if m.Memory.Kind == cruciblev1.MemHot {
			found = true
		}
	}
	if !found {
		t.Fatal("hot plan should surface when IncludeHot and a plan exists")
	}
}
