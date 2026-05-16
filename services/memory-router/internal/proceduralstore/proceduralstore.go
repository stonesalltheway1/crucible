// Package proceduralstore is the Graphiti abstraction over FalkorDB
// (default backend, per ADR-006). The Neo4j path is documented; both
// implement Store.
//
// Tenant isolation is enforced at the storage tier: each tenant has its
// own named graph `tenant_<sanitized_tenant_id>`. Cross-tenant reads
// require explicitly switching the active graph context — there is no
// "read all tenants" Cypher path through this interface.
//
// Bi-temporal edges follow Graphiti's pattern:
//   :REINFORCED_BY  (recorded_at, valid_from)
//   :VIOLATED_BY    (recorded_at)
//   :SUPERSEDED_BY  (valid_from, valid_to)
//   :APPLIES_TO     (scope as labelled props on the relationship)
package proceduralstore

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	memoryspec "github.com/crucible/memory-spec/go"
	cruciblev1 "github.com/crucible/sdk-go/crucible/v1"

	"github.com/crucible/memory-router/internal/scope"
)

// Store is the Graphiti-shaped procedural-memory persistence interface.
type Store interface {
	// FetchByScope returns Conventions whose scope matches the query,
	// filtered to status='active' or 'suggested'.
	FetchByScope(ctx context.Context, tenantID string, layer memoryspec.MemoryLayer, query memoryspec.ScopeFilter, limit int) ([]memoryspec.Convention, error)

	// FetchByID returns a Convention. Used by twin.memory.conventions
	// and for supersession traversal.
	FetchByID(ctx context.Context, tenantID, conventionID string) (memoryspec.Convention, error)

	// Upsert writes a convention; the procedural store is the authority
	// for procedural memory. The Postgres mirror is upserted by a
	// downstream subscriber. Returns the assigned id.
	Upsert(ctx context.Context, c memoryspec.Convention) (string, error)

	// MarkSuperseded creates a SUPERSEDED_BY edge from old → new and
	// flips old's status to 'superseded' bi-temporally.
	MarkSuperseded(ctx context.Context, tenantID, oldID, newID string) error

	// RecordReinforcement adds a REINFORCED_BY edge from convention →
	// source (typically a pr_comment SourceRef that the distiller
	// observed re-stating the rule).
	RecordReinforcement(ctx context.Context, tenantID, conventionID string, source cruciblev1.SourceRef) error

	// RecordViolation adds a VIOLATED_BY edge. Used by the drift
	// detector counters.
	RecordViolation(ctx context.Context, tenantID, conventionID string, source cruciblev1.SourceRef) error

	// DriftRatio returns (positives_30d, negatives_30d) for a tenant's
	// active conventions. The detector job calls this nightly.
	DriftRatio(ctx context.Context, tenantID, conventionID string) (positives uint32, negatives uint32, err error)

	// ListByTenant streams all active conventions for a tenant. Used
	// by the federation graduation detector + the cartographer for
	// the global_defaults preview at install time.
	ListByTenant(ctx context.Context, tenantID string, layer memoryspec.MemoryLayer) ([]memoryspec.Convention, error)

	Close() error
}

// ErrEmptyTenant is returned when the caller passes an unscoped query.
var ErrEmptyTenant = errors.New("proceduralstore: tenant_id required")

// ErrNotFound is returned when a convention id doesn't exist in the
// tenant's graph.
var ErrNotFound = errors.New("proceduralstore: not found")

// SanitizeGraphName produces a Cypher-safe per-tenant graph name. The
// FalkorDB driver names graphs identifier-style; arbitrary tenant_ids
// must be mapped.
func SanitizeGraphName(tenantID string) string {
	if tenantID == "" {
		return ""
	}
	if tenantID == "global" {
		return "tenant_global"
	}
	var b strings.Builder
	b.Grow(len(tenantID) + 7)
	b.WriteString("tenant_")
	for _, r := range tenantID {
		switch {
		case r >= 'a' && r <= 'z',
			r >= 'A' && r <= 'Z',
			r >= '0' && r <= '9',
			r == '_':
			b.WriteRune(r)
		default:
			b.WriteRune('_')
		}
	}
	return b.String()
}

// ─── In-memory fake ─────────────────────────────────────────────────────────
// Used by unit tests and CRUCIBLE_MEMORY_ROUTER_STUB=1. Implements the
// scope-match + bi-temporal supersession in Go so the contract is
// exercised end-to-end without FalkorDB.

// NewFake constructs an in-memory Store. Safe for concurrent use.
func NewFake() Store {
	return &fake{tenants: map[string]*tenantGraph{}}
}

type tenantGraph struct {
	conventions map[string]memoryspec.Convention
	reinforce   map[string][]cruciblev1.SourceRef
	violate     map[string][]cruciblev1.SourceRef
}

type fake struct {
	mu      sync.RWMutex
	tenants map[string]*tenantGraph
}

func (f *fake) graphFor(tenantID string) *tenantGraph {
	g, ok := f.tenants[tenantID]
	if !ok {
		g = &tenantGraph{
			conventions: map[string]memoryspec.Convention{},
			reinforce:   map[string][]cruciblev1.SourceRef{},
			violate:     map[string][]cruciblev1.SourceRef{},
		}
		f.tenants[tenantID] = g
	}
	return g
}

func (f *fake) FetchByScope(ctx context.Context, tenantID string, layer memoryspec.MemoryLayer, q memoryspec.ScopeFilter, limit int) ([]memoryspec.Convention, error) {
	_ = ctx
	if tenantID == "" {
		return nil, ErrEmptyTenant
	}
	f.mu.RLock()
	defer f.mu.RUnlock()
	g, ok := f.tenants[tenantID]
	if !ok {
		return nil, nil
	}
	out := make([]memoryspec.Convention, 0, len(g.conventions))
	for _, c := range g.conventions {
		if layer != "" && c.Layer != layer {
			continue
		}
		if c.Status != memoryspec.StatusActive && c.Status != memoryspec.StatusSuggested {
			continue
		}
		if !scope.Match(c.Scope, q) {
			continue
		}
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Confidence > out[j].Confidence })
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (f *fake) FetchByID(ctx context.Context, tenantID, id string) (memoryspec.Convention, error) {
	_ = ctx
	if tenantID == "" {
		return memoryspec.Convention{}, ErrEmptyTenant
	}
	f.mu.RLock()
	defer f.mu.RUnlock()
	g, ok := f.tenants[tenantID]
	if !ok {
		return memoryspec.Convention{}, ErrNotFound
	}
	c, ok := g.conventions[id]
	if !ok {
		return memoryspec.Convention{}, ErrNotFound
	}
	return c, nil
}

func (f *fake) Upsert(ctx context.Context, c memoryspec.Convention) (string, error) {
	_ = ctx
	if c.TenantID == "" {
		return "", ErrEmptyTenant
	}
	if err := c.Validate(); err != nil {
		return "", err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	g := f.graphFor(c.TenantID)
	if c.ID == "" {
		c.ID = fmt.Sprintf("conv_%020d", time.Now().UnixNano())
	}
	if c.ValidFrom.IsZero() {
		c.ValidFrom = time.Now().UTC()
	}
	if c.WrittenAt.IsZero() {
		c.WrittenAt = time.Now().UTC()
	}
	g.conventions[c.ID] = c
	return c.ID, nil
}

func (f *fake) MarkSuperseded(ctx context.Context, tenantID, oldID, newID string) error {
	_ = ctx
	if tenantID == "" {
		return ErrEmptyTenant
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	g, ok := f.tenants[tenantID]
	if !ok {
		return ErrNotFound
	}
	old, ok := g.conventions[oldID]
	if !ok {
		return ErrNotFound
	}
	old.Status = memoryspec.StatusSuperseded
	now := time.Now().UTC()
	old.ValidTo = &now
	old.Supersedes = append(old.Supersedes, newID)
	g.conventions[oldID] = old
	return nil
}

func (f *fake) RecordReinforcement(ctx context.Context, tenantID, conventionID string, src cruciblev1.SourceRef) error {
	_ = ctx
	if tenantID == "" {
		return ErrEmptyTenant
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	g := f.graphFor(tenantID)
	g.reinforce[conventionID] = append(g.reinforce[conventionID], src)
	c, ok := g.conventions[conventionID]
	if ok {
		c.LastReinforced = time.Now().UTC()
		g.conventions[conventionID] = c
	}
	return nil
}

func (f *fake) RecordViolation(ctx context.Context, tenantID, conventionID string, src cruciblev1.SourceRef) error {
	_ = ctx
	if tenantID == "" {
		return ErrEmptyTenant
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	g := f.graphFor(tenantID)
	g.violate[conventionID] = append(g.violate[conventionID], src)
	c, ok := g.conventions[conventionID]
	if ok {
		now := time.Now().UTC()
		c.LastViolated = &now
		g.conventions[conventionID] = c
	}
	return nil
}

func (f *fake) DriftRatio(ctx context.Context, tenantID, conventionID string) (uint32, uint32, error) {
	_ = ctx
	if tenantID == "" {
		return 0, 0, ErrEmptyTenant
	}
	f.mu.RLock()
	defer f.mu.RUnlock()
	g, ok := f.tenants[tenantID]
	if !ok {
		return 0, 0, nil
	}
	cutoff := time.Now().Add(-30 * 24 * time.Hour)
	_ = cutoff // fake counts all-time; production filters by timestamp
	return uint32(len(g.reinforce[conventionID])), uint32(len(g.violate[conventionID])), nil
}

func (f *fake) ListByTenant(ctx context.Context, tenantID string, layer memoryspec.MemoryLayer) ([]memoryspec.Convention, error) {
	_ = ctx
	if tenantID == "" {
		return nil, ErrEmptyTenant
	}
	f.mu.RLock()
	defer f.mu.RUnlock()
	g, ok := f.tenants[tenantID]
	if !ok {
		return nil, nil
	}
	out := make([]memoryspec.Convention, 0, len(g.conventions))
	for _, c := range g.conventions {
		if layer != "" && c.Layer != layer {
			continue
		}
		out = append(out, c)
	}
	return out, nil
}

func (f *fake) Close() error { return nil }
