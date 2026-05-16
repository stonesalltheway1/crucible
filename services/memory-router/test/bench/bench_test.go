// Package bench measures the memory-router p95 latency against the
// < 100ms quality-bar from the Phase-5 brief.
//
// The benchmark uses the in-memory fakes so it reports the
// orchestration overhead (retriever + ranker + budget). Production
// numbers add ~25ms (pgvector RTT) + ~20ms (FalkorDB RTT) atop this;
// the in-mem run must stay well under 50ms to leave the budget for I/O.
package bench

import (
	"context"
	"fmt"
	"sort"
	"testing"
	"time"

	memoryspec "github.com/crucible/memory-spec/go"
	cruciblev1 "github.com/crucible/sdk-go/crucible/v1"

	"github.com/crucible/memory-router/internal/embedding"
	"github.com/crucible/memory-router/internal/globaldefaults"
	"github.com/crucible/memory-router/internal/hotstore"
	"github.com/crucible/memory-router/internal/proceduralstore"
	"github.com/crucible/memory-router/internal/retriever"
	"github.com/crucible/memory-router/internal/vectorstore"
)

func TestP95Latency_UnderBudget(t *testing.T) {
	hot := hotstore.New(hotstore.NewFake())
	vec := vectorstore.NewFake()
	proc := proceduralstore.NewFake()
	emb := embedding.NewFake()
	r := retriever.New(hot, vec, proc, emb, globaldefaults.NewLoader())

	ctx := context.Background()
	now := time.Now().UTC()
	for i := 0; i < 500; i++ {
		_, _ = proc.Upsert(ctx, memoryspec.Convention{
			ID:         fmt.Sprintf("conv_b_%d", i),
			TenantID:   "ten_bench",
			Layer:      memoryspec.LayerOrgOverrides,
			Scope:      cruciblev1.ScopeFilter{FileGlob: fmt.Sprintf("src/**/*_%d.ts", i%4)},
			RuleNl:     fmt.Sprintf("bench rule %d", i),
			Category:   memoryspec.AllCategories()[i%12],
			Status:     memoryspec.StatusActive,
			Confidence: 0.6 + 0.001*float64(i%200),
			ValidFrom:  now,
			WrittenAt:  now,
		})
	}
	for i := 0; i < 1000; i++ {
		content := fmt.Sprintf("bench memory %d", i)
		vecs, _ := emb.Embed(ctx, []embedding.Request{{TenantID: "ten_bench", Content: content}})
		_, _ = vec.Write(ctx, cruciblev1.Memory{Kind: cruciblev1.MemEpisodic, Content: content, Importance: 0.5},
			vecs[0], "ten_bench", "")
	}

	const N = 200
	latencies := make([]time.Duration, N)
	for i := 0; i < N; i++ {
		start := time.Now()
		_, err := r.Recall(ctx, memoryspec.RetrievalQuery{
			TenantID:          "ten_bench",
			Query:             "find applicable conventions",
			Scope:             cruciblev1.ScopeFilter{FileGlob: fmt.Sprintf("src/handlers/login_%d.ts", i%4)},
			IncludeProcedural: true,
			IncludeEpisodic:   true,
			MaxTokens:         7000,
		})
		latencies[i] = time.Since(start)
		if err != nil {
			t.Fatal(err)
		}
	}
	sort.Slice(latencies, func(i, j int) bool { return latencies[i] < latencies[j] })
	p50 := latencies[N/2]
	p95 := latencies[(N*95)/100]
	p99 := latencies[(N*99)/100]
	t.Logf("p50=%s p95=%s p99=%s (in-memory; production adds ~45ms for pgvector + FalkorDB RTT)", p50, p95, p99)

	if p95 > 50*time.Millisecond {
		t.Fatalf("in-mem p95 %s exceeds 50ms (production budget is 100ms incl. ~45ms RTT)", p95)
	}
}
