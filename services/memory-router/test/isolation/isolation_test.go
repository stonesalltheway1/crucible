// Package isolation runs the 50K-query cross-tenant adversarial test.
//
// The brief mandates: "Cross-tenant isolation: zero leaks in 50,000+
// adversarial random-query tests." This package is the production gate
// for that invariant.
//
// Strategy:
//  1. Seed 8 tenants, each with 100 conventions whose rule_nl carries
//     the tenant_id as a watermark.
//  2. Drive 50,000 random queries: random tenant, random scope, random
//     include flags.
//  3. Assert every returned memory's content does NOT contain any other
//     tenant's watermark.
//  4. Also assert the embedding-batch guard refuses cross-tenant
//     batches.
//
// Test runs with `go test -tags isolation ./test/isolation/...` and
// is part of the Phase-5 ship gate.
package isolation

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	mathrand "math/rand"
	"strings"
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

const (
	numTenants        = 8
	rulesPerTenant    = 100
	memoriesPerTenant = 50
	numQueries        = 50_000
)

type rig struct {
	r       *retriever.Retriever
	tenants []string
}

func setup(t *testing.T) *rig {
	t.Helper()
	hot := hotstore.New(hotstore.NewFake())
	vec := vectorstore.NewFake()
	proc := proceduralstore.NewFake()
	emb := embedding.NewFake()
	r := retriever.New(hot, vec, proc, emb, globaldefaults.NewLoader())

	ctx := context.Background()
	now := time.Now().UTC()

	tenants := make([]string, numTenants)
	for i := range tenants {
		tenants[i] = fmt.Sprintf("ten_isolation_%d", i)
	}

	for _, tenant := range tenants {
		watermark := "WATERMARK_" + tenant + "_" + randHex(t, 8)
		for k := 0; k < rulesPerTenant; k++ {
			_, err := proc.Upsert(ctx, memoryspec.Convention{
				ID:         fmt.Sprintf("conv_%s_%d", tenant, k),
				TenantID:   tenant,
				Layer:      memoryspec.LayerOrgOverrides,
				Scope:      cruciblev1.ScopeFilter{FileGlob: fmt.Sprintf("api/**/*_%d.ts", k%5)},
				RuleNl:     fmt.Sprintf("Rule %d for %s containing %s", k, tenant, watermark),
				Category:   pickCategory(k),
				Status:     memoryspec.StatusActive,
				Confidence: 0.5 + 0.005*float64(k%50),
				ValidFrom:  now,
				WrittenAt:  now,
			})
			if err != nil {
				t.Fatalf("upsert: %v", err)
			}
		}
		// Episodic data per tenant.
		for k := 0; k < memoriesPerTenant; k++ {
			content := fmt.Sprintf("Episodic memory %d for %s [%s]", k, tenant, watermark)
			vecs, _ := emb.Embed(ctx, []embedding.Request{{TenantID: tenant, Content: content}})
			_, err := vec.Write(ctx, cruciblev1.Memory{
				Kind: cruciblev1.MemEpisodic, Content: content,
				Importance: 0.4 + 0.01*float64(k%20),
			}, vecs[0], tenant, "repo/test")
			if err != nil {
				t.Fatalf("vec write: %v", err)
			}
		}
	}

	return &rig{r: r, tenants: tenants}
}

func TestCrossTenantIsolation_50KAdversarialQueries(t *testing.T) {
	if testing.Short() {
		t.Skip("isolation test is long; run without -short to enforce")
	}
	rg := setup(t)
	ctx := context.Background()

	rnd := mathrand.New(mathrand.NewSource(1))
	queryCount := numQueries
	if testing.Short() {
		queryCount = 1000
	}

	leaks := 0
	otherWatermarks := map[string][]string{}
	// Build per-tenant watermark expectations: for each tenant, the
	// list of OTHER tenants' watermark prefixes we must never see.
	for _, t := range rg.tenants {
		for _, other := range rg.tenants {
			if other == t {
				continue
			}
			otherWatermarks[t] = append(otherWatermarks[t], "WATERMARK_"+other)
		}
	}

	for i := 0; i < queryCount; i++ {
		tenant := rg.tenants[rnd.Intn(len(rg.tenants))]
		fg := fmt.Sprintf("api/handlers/oauth_%d.ts", rnd.Intn(5))
		q := memoryspec.RetrievalQuery{
			TenantID:          tenant,
			Query:             "what conventions apply to this file?",
			Scope:             cruciblev1.ScopeFilter{FileGlob: fg},
			IncludeProcedural: rnd.Intn(2) == 0,
			IncludeEpisodic:   rnd.Intn(2) == 0,
			MaxTokens:         2000,
		}
		if !q.IncludeProcedural && !q.IncludeEpisodic {
			q.IncludeProcedural = true
		}
		res, err := rg.r.Recall(ctx, q)
		if err != nil {
			t.Fatalf("recall: %v", err)
		}
		for _, m := range res.Memories {
			content := m.Memory.Content
			for _, otherWatermark := range otherWatermarks[tenant] {
				if strings.Contains(content, otherWatermark) {
					leaks++
					t.Errorf("LEAK: tenant=%s saw %q in: %q", tenant, otherWatermark, content)
				}
			}
		}
	}
	if leaks != 0 {
		t.Fatalf("isolation FAILED: %d cross-tenant leaks across %d adversarial queries", leaks, queryCount)
	}
}

func TestEmbeddingGuard_RefusesCrossTenantBatch(t *testing.T) {
	emb := embedding.NewFake()
	_, err := emb.Embed(context.Background(), []embedding.Request{
		{TenantID: "ten_a", Content: "x"},
		{TenantID: "ten_b", Content: "y"},
	})
	if err == nil {
		t.Fatal("must refuse cross-tenant embedding batch")
	}
}

func randHex(t *testing.T, n int) string {
	t.Helper()
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func pickCategory(i int) memoryspec.ConventionCategory {
	cats := memoryspec.AllCategories()
	return cats[i%len(cats)]
}
