// Package retriever orchestrates the multi-signal hybrid recall:
//
//   Redis (hot) + pgvector (epis+sem) + FalkorDB (procedural across 3 layers)
//     → A-MAC re-rank
//       → 7K-token budget enforcement
//         → ScoredMemory[]
//
// The retriever is the only code path that can return memories to a
// caller. It refuses an unscoped (empty tenant_id) call and refuses to
// cross tenants under any circumstance — defence in depth atop the
// underlying RLS + per-graph isolation.
package retriever

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	memoryspec "github.com/crucible/memory-spec/go"
	cruciblev1 "github.com/crucible/sdk-go/crucible/v1"

	"github.com/crucible/memory-router/internal/budget"
	"github.com/crucible/memory-router/internal/embedding"
	"github.com/crucible/memory-router/internal/globaldefaults"
	"github.com/crucible/memory-router/internal/hotstore"
	"github.com/crucible/memory-router/internal/layering"
	"github.com/crucible/memory-router/internal/proceduralstore"
	"github.com/crucible/memory-router/internal/ranker"
	"github.com/crucible/memory-router/internal/vectorstore"
)

// Retriever orchestrates the recall pipeline. All dependencies are
// injected so the gRPC server's main can wire production drivers and
// tests use the in-memory fakes.
type Retriever struct {
	Hot         *hotstore.Store
	Vec         vectorstore.Store
	Proc        proceduralstore.Store
	Embedder    embedding.Client
	Globals     *globaldefaults.Loader
	Weights     ranker.Weights
	BudgetCap   uint32 // default 7000
	DefaultTopK int    // default 32
}

// New constructs a Retriever with default tuning.
func New(hot *hotstore.Store, vec vectorstore.Store, proc proceduralstore.Store, emb embedding.Client, globals *globaldefaults.Loader) *Retriever {
	return &Retriever{
		Hot:         hot,
		Vec:         vec,
		Proc:        proc,
		Embedder:    emb,
		Globals:     globals,
		Weights:     ranker.Default(),
		BudgetCap:   budget.DefaultMaxTokens,
		DefaultTopK: 32,
	}
}

// ErrEmptyTenant is returned for unscoped queries.
var ErrEmptyTenant = errors.New("retriever: tenant_id required")

// Recall executes the hybrid retrieval. Returns the budget-enforced,
// A-MAC re-ranked list of ScoredMemories.
func (r *Retriever) Recall(ctx context.Context, q memoryspec.RetrievalQuery) (memoryspec.RetrievalResult, error) {
	start := time.Now()
	if q.TenantID == "" {
		return memoryspec.RetrievalResult{}, ErrEmptyTenant
	}
	if q.MaxTokens == 0 {
		q.MaxTokens = r.BudgetCap
	}
	if q.MaxItems == 0 {
		q.MaxItems = uint32(r.DefaultTopK)
	}

	// 1. Embed the query (single-tenant guard inside embedder).
	var qvec embedding.Vector
	if (q.IncludeEpisodic || q.IncludeSemantic) && r.Embedder != nil {
		vecs, err := r.Embedder.Embed(ctx, []embedding.Request{{TenantID: q.TenantID, Content: q.Query}})
		if err != nil {
			return memoryspec.RetrievalResult{}, fmt.Errorf("embed: %w", err)
		}
		qvec = vecs[0]
	}

	considered := 0
	all := make([]memoryspec.ScoredMemory, 0, 64)

	// 2. Procedural lookup across layers (bottom-up read).
	procByLayer := map[memoryspec.MemoryLayer][]memoryspec.Convention{}
	if q.IncludeProcedural && r.Proc != nil {
		for _, layer := range []memoryspec.MemoryLayer{memoryspec.LayerOrgOverrides, memoryspec.LayerRepoOverrides} {
			convs, err := r.Proc.FetchByScope(ctx, q.TenantID, layer, q.Scope, int(q.MaxItems))
			if err != nil {
				return memoryspec.RetrievalResult{}, fmt.Errorf("procedural %s: %w", layer, err)
			}
			procByLayer[layer] = convs
			considered += len(convs)
		}
		// Always pull global_defaults via the in-process loader (faster than DB).
		if r.Globals != nil {
			// Stack auto-detection is the cartographer's job; in-band
			// we use category-based pulling — every category every time.
			for _, c := range r.Globals.ConventionsForStacks(memoryspec.AllStacks()...) {
				procByLayer[memoryspec.LayerGlobalDefaults] = append(procByLayer[memoryspec.LayerGlobalDefaults], c)
			}
			considered += len(procByLayer[memoryspec.LayerGlobalDefaults])
		}

		merged := layering.Merge(procByLayer)
		for _, c := range merged {
			mem := conventionToMemory(c)
			score := ranker.Compute(mem, conventionSemanticHint(c, q), 0, r.Weights)
			all = append(all, memoryspec.ScoredMemory{
				Memory:          mem,
				Layer:           c.Layer,
				SemanticScore:   conventionSemanticHint(c, q),
				ImportanceScore: score.Importance.Composite,
				FinalScore:      score.Final,
				TokenEstimate:   budget.Estimate(mem.Content),
			})
		}
	}

	// 3. Episodic / semantic vector lookup.
	if (q.IncludeEpisodic || q.IncludeSemantic) && r.Vec != nil && qvec != nil {
		kind := vectorstore.KindBoth
		if q.IncludeEpisodic && !q.IncludeSemantic {
			kind = vectorstore.KindEpisodic
		}
		if !q.IncludeEpisodic && q.IncludeSemantic {
			kind = vectorstore.KindSemantic
		}
		hits, err := r.Vec.Search(ctx, vectorstore.Query{
			TenantID:  q.TenantID,
			RepoID:    q.Scope.Repo,
			Kind:      kind,
			Embedding: qvec,
			TopK:      int(q.MaxItems),
			FileGlob:  q.Scope.FileGlob,
		})
		if err != nil {
			return memoryspec.RetrievalResult{}, fmt.Errorf("vec search: %w", err)
		}
		considered += len(hits)
		for _, h := range hits {
			h.Memory.LastRecalled = h.LastRecalled
			score := ranker.Compute(h.Memory, h.Similarity, h.RecallCount, r.Weights)
			all = append(all, memoryspec.ScoredMemory{
				Memory:          h.Memory,
				Layer:           memoryspec.LayerOrgOverrides, // episodic+semantic live tenant-private
				SemanticScore:   h.Similarity,
				ImportanceScore: score.Importance.Composite,
				FinalScore:      score.Final,
				TokenEstimate:   budget.Estimate(h.Memory.Content),
			})
		}
	}

	// 4. Hot lookup (Redis); typically returns the current plan +
	// recent tool calls. Compose into one synthetic Memory if non-empty.
	if q.IncludeHot && r.Hot != nil && q.TaskID != "" {
		if planJSON, err := r.Hot.GetPlan(ctx, q.TenantID, q.TaskID); err == nil && planJSON != "" {
			mem := cruciblev1.Memory{
				ID:           "hot_plan_" + q.TaskID,
				Kind:         cruciblev1.MemHot,
				Content:      planJSON,
				Importance:   0.9,
				LastRecalled: time.Now().UTC(),
			}
			score := ranker.Compute(mem, 1.0, 0, r.Weights)
			all = append(all, memoryspec.ScoredMemory{
				Memory:          mem,
				Layer:           memoryspec.LayerOrgOverrides,
				SemanticScore:   1.0,
				ImportanceScore: score.Importance.Composite,
				FinalScore:      score.Final,
				TokenEstimate:   budget.Estimate(mem.Content),
			})
			considered++
		}
	}

	// 5. A-MAC re-rank + budget enforcement.
	sort.Slice(all, func(i, j int) bool { return all[i].FinalScore > all[j].FinalScore })
	trimmed, used := budget.Enforce(all, q.MaxTokens)

	// 6. Reinforce-on-access for the episodic/semantic items we returned.
	go func(items []memoryspec.ScoredMemory, tenantID string) {
		bg, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		for _, m := range items {
			if m.Memory.Kind == cruciblev1.MemEpisodic || m.Memory.Kind == cruciblev1.MemSemantic {
				_ = r.Vec.ReinforceOnAccess(bg, tenantID, m.Memory.ID)
			}
		}
	}(trimmed, q.TenantID)

	return memoryspec.RetrievalResult{
		Memories:         trimmed,
		TokensUsed:       used,
		BudgetTokens:     q.MaxTokens,
		ItemsConsidered:  uint32(considered),
		ItemsReturned:    uint32(len(trimmed)),
		LatencyMs:        uint32(time.Since(start).Milliseconds()),
	}, nil
}

// conventionToMemory adapts a Convention to a Memory for unified
// ranking. Procedural memories carry the convention's rule_nl as
// content and the confidence as importance.
func conventionToMemory(c memoryspec.Convention) cruciblev1.Memory {
	return cruciblev1.Memory{
		ID:           c.ID,
		Content:      c.RuleNl,
		Kind:         cruciblev1.MemProcedural,
		Importance:   c.Confidence,
		WrittenAt:    c.WrittenAt,
		LastRecalled: c.LastReinforced,
	}
}

// conventionSemanticHint is a coarse proxy for "how well does this
// convention match the query" without re-embedding the rule. Used as
// the semantic_score input to the A-MAC rank. The cartographer + the
// procedural store both pre-compute the rule's embedding at admission;
// the production hot-path looks it up rather than recomputing.
func conventionSemanticHint(c memoryspec.Convention, q memoryspec.RetrievalQuery) float64 {
	if q.Scope.Category != "" && q.Scope.Category == string(c.Category) {
		return 1.0
	}
	if q.Scope.FileGlob != "" && c.Scope.FileGlob != "" {
		if c.Scope.FileGlob == q.Scope.FileGlob {
			return 0.85
		}
	}
	return 0.55
}
