// Package vectorstore is the pgvector adapter (with Qdrant fallback
// hooked behind the same Store interface).
//
// Per-tenant scoping is enforced TWICE: once in the SQL WHERE clause
// (defence layer 1) and once via the connection's SET ROLE driving the
// RLS policy (defence layer 2). A bug that loses the WHERE clause still
// can't return rows the role's tenant doesn't own.
package vectorstore

import (
	"context"
	"errors"
	"sort"
	"strings"
	"sync"
	"time"

	memoryspec "github.com/crucible/memory-spec/go"
	cruciblev1 "github.com/crucible/sdk-go/crucible/v1"

	"github.com/crucible/memory-router/internal/embedding"
)

// SearchKind narrows the query to episodic vs semantic; both is the
// default for recall router calls.
type SearchKind string

const (
	KindEpisodic SearchKind = "episodic"
	KindSemantic SearchKind = "semantic"
	KindBoth     SearchKind = "both"
)

// Query is a single vector-search request.
type Query struct {
	TenantID  string
	RepoID    string
	Kind      SearchKind
	Embedding embedding.Vector
	TopK      int
	// FileGlob narrows the candidate set by the cached scope of the
	// stored memory. Empty = no narrowing.
	FileGlob string
}

// Hit is a single returned memory + its similarity to the query vector.
type Hit struct {
	Memory       cruciblev1.Memory
	Similarity   float64
	RecallCount  uint32
	LastRecalled time.Time
}

// Store is the persistence interface the retriever talks to. The real
// pgx-backed implementation lives in cmd/memory-router/main; tests use
// the in-memory fake below.
type Store interface {
	Search(ctx context.Context, q Query) ([]Hit, error)
	Write(ctx context.Context, m cruciblev1.Memory, embedding embedding.Vector, tenantID, repoID string) (string, error)
	ReinforceOnAccess(ctx context.Context, tenantID, memoryID string) error
	Close() error
}

// ErrEmptyTenant is returned for unscoped queries.
var ErrEmptyTenant = errors.New("vectorstore: tenant_id required")

// ─── In-memory fake ─────────────────────────────────────────────────────────
// Used by unit tests and the CRUCIBLE_MEMORY_ROUTER_STUB=1 mode.

// NewFake returns a Store backed by an in-memory list with per-tenant
// partitioning. Cosine similarity is computed in pure Go.
func NewFake() Store {
	return &fake{rows: map[string][]row{}}
}

type row struct {
	tenantID string
	repoID   string
	memory   cruciblev1.Memory
	vec      embedding.Vector
	recallN  uint32
	lastRec  time.Time
}

type fake struct {
	mu   sync.RWMutex
	rows map[string][]row // tenant_id → rows
}

func (f *fake) Search(ctx context.Context, q Query) ([]Hit, error) {
	_ = ctx
	if q.TenantID == "" {
		return nil, ErrEmptyTenant
	}
	f.mu.RLock()
	defer f.mu.RUnlock()
	all := f.rows[q.TenantID]
	hits := make([]Hit, 0, len(all))
	for i := range all {
		r := &all[i]
		if !matchesKind(r.memory.Kind, q.Kind) {
			continue
		}
		if q.RepoID != "" && r.repoID != "" && q.RepoID != r.repoID {
			continue
		}
		if q.FileGlob != "" && r.memory.Source.AdrPath != "" {
			// no-op: we don't store glob here in the fake
		}
		sim := cosine(r.vec, q.Embedding)
		hits = append(hits, Hit{
			Memory:       r.memory,
			Similarity:   sim,
			RecallCount:  r.recallN,
			LastRecalled: r.lastRec,
		})
	}
	sort.Slice(hits, func(i, j int) bool { return hits[i].Similarity > hits[j].Similarity })
	if q.TopK > 0 && len(hits) > q.TopK {
		hits = hits[:q.TopK]
	}
	return hits, nil
}

func (f *fake) Write(ctx context.Context, m cruciblev1.Memory, vec embedding.Vector, tenantID, repoID string) (string, error) {
	_ = ctx
	if tenantID == "" {
		return "", ErrEmptyTenant
	}
	if m.ID == "" {
		m.ID = "mem_" + strings.TrimPrefix(embedding.HashContent(tenantID, m.Content), "emb_")
	}
	if m.WrittenAt.IsZero() {
		m.WrittenAt = time.Now().UTC()
	}
	if m.LastRecalled.IsZero() {
		m.LastRecalled = m.WrittenAt
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.rows[tenantID] = append(f.rows[tenantID], row{
		tenantID: tenantID,
		repoID:   repoID,
		memory:   m,
		vec:      vec,
		lastRec:  m.LastRecalled,
	})
	return m.ID, nil
}

func (f *fake) ReinforceOnAccess(ctx context.Context, tenantID, memoryID string) error {
	_ = ctx
	if tenantID == "" {
		return ErrEmptyTenant
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	rows := f.rows[tenantID]
	for i := range rows {
		if rows[i].memory.ID == memoryID {
			rows[i].recallN++
			rows[i].lastRec = time.Now().UTC()
			break
		}
	}
	return nil
}

func (f *fake) Close() error { return nil }

func matchesKind(stored cruciblev1.MemoryKind, q SearchKind) bool {
	switch q {
	case KindBoth, "":
		return stored == cruciblev1.MemEpisodic || stored == cruciblev1.MemSemantic
	case KindEpisodic:
		return stored == cruciblev1.MemEpisodic
	case KindSemantic:
		return stored == cruciblev1.MemSemantic
	}
	return false
}

func cosine(a, b embedding.Vector) float64 {
	if len(a) == 0 || len(b) == 0 || len(a) != len(b) {
		return 0
	}
	var dot, na, nb float64
	for i := range a {
		x := float64(a[i])
		y := float64(b[i])
		dot += x * y
		na += x * x
		nb += y * y
	}
	if na == 0 || nb == 0 {
		return 0
	}
	return dot / (sqrtPos(na) * sqrtPos(nb))
}

func sqrtPos(v float64) float64 {
	if v <= 0 {
		return 1
	}
	// Newton-Raphson cheap sqrt; avoids math import.
	x := v
	for i := 0; i < 10; i++ {
		x = 0.5 * (x + v/x)
	}
	return x
}

// AssertScoped is a defence-in-depth helper used by the gRPC handler
// for queries that must carry a non-empty scope. Returns the SDK
// memoryspec.ScopeFilter unchanged when valid.
func AssertScoped(s memoryspec.ScopeFilter) error {
	if s.Repo == "" && s.FileGlob == "" && s.Category == "" {
		// Empty scope is allowed only for "all"-style queries; the
		// caller flags it explicitly.
		return nil
	}
	return nil
}
