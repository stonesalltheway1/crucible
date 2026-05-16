// Package embedding wraps the embedding-API client with per-tenant
// guards.
//
// CRITICAL: embeddings are NEVER shared across tenants. The Client
// refuses to embed a batch whose entries don't all carry the same
// tenant_id. The cache (Redis) keys embeddings under {tenant_id}, so a
// cache hit from one tenant is unreachable from another.
package embedding

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
)

// Vector is a 3072-dim float32 slice (anthropic/openai text-embedding-3-large).
type Vector []float32

// Dim is the model dimension. Wired everywhere — DB columns,
// embed-cache shape, similarity math.
const Dim = 3072

// Request is one input to be embedded. tenant_id is mandatory.
type Request struct {
	TenantID string
	Content  string
}

// ErrEmptyTenant is returned when a caller passes an unscoped request.
var ErrEmptyTenant = errors.New("embedding: tenant_id required")

// ErrCrossTenantBatch is returned when Embed sees mixed tenants in a
// single batch. Defense in depth — the caller is supposed to scope per
// tenant.
var ErrCrossTenantBatch = errors.New("embedding: refusing to embed cross-tenant batch")

// Client is the vendor-neutral embedding surface.
type Client interface {
	Embed(ctx context.Context, batch []Request) ([]Vector, error)
	ContentHash(content string) string
	Model() string
	Dimension() int
}

// HashContent is the canonical content-hash for embedding cache keys.
// Tenant_id is folded in so identical content from two tenants gets
// distinct cache slots — embeddings are never shared.
func HashContent(tenantID, content string) string {
	h := sha256.New()
	_, _ = h.Write([]byte(tenantID))
	_, _ = h.Write([]byte{0})
	_, _ = h.Write([]byte(content))
	return "emb_" + hex.EncodeToString(h.Sum(nil))[:32]
}

// EnforceSingleTenant returns ErrCrossTenantBatch if the batch contains
// more than one tenant, or ErrEmptyTenant if any request is unscoped.
// Used by every Client implementation before issuing the upstream call.
func EnforceSingleTenant(batch []Request) error {
	if len(batch) == 0 {
		return nil
	}
	first := batch[0].TenantID
	if first == "" {
		return ErrEmptyTenant
	}
	for _, r := range batch[1:] {
		if r.TenantID == "" {
			return ErrEmptyTenant
		}
		if r.TenantID != first {
			return fmt.Errorf("%w: %q vs %q", ErrCrossTenantBatch, first, r.TenantID)
		}
	}
	return nil
}

// ─── Deterministic fake for tests ───────────────────────────────────────────
//
// Returns a hash-based vector so tests can assert "same content → same
// vector" and "different content → orthogonal-ish vector" without
// hitting a real model. Single-tenant guard still applies; the fake
// enforces it so isolation tests work.

// NewFake returns a Client that fabricates deterministic vectors. Single-
// tenant batch enforcement still applies — tests that need the guard to
// be exercised use this implementation.
func NewFake() Client {
	return &fakeClient{model: "fake-deterministic-3072"}
}

type fakeClient struct{ model string }

func (f *fakeClient) Embed(ctx context.Context, batch []Request) ([]Vector, error) {
	_ = ctx
	if err := EnforceSingleTenant(batch); err != nil {
		return nil, err
	}
	out := make([]Vector, len(batch))
	for i, r := range batch {
		out[i] = pseudoVector(r.TenantID, r.Content)
	}
	return out, nil
}

func (f *fakeClient) ContentHash(content string) string {
	// Note: callers should prefer HashContent(tenant, content); this
	// is only used in narrow places where tenant is bound separately.
	h := sha256.Sum256([]byte(content))
	return "emb_" + hex.EncodeToString(h[:16])
}

func (f *fakeClient) Model() string  { return f.model }
func (f *fakeClient) Dimension() int { return Dim }

// pseudoVector seeds a deterministic per-tenant pseudo-orthogonal
// vector from the content hash. Not for production cosine math; it's
// stable + uniform enough for unit tests to assert ordering invariants.
func pseudoVector(tenantID, content string) Vector {
	h := sha256.Sum256([]byte(tenantID + ":" + content))
	v := make(Vector, Dim)
	for i := 0; i < Dim; i++ {
		// Map each byte (cycled) into [-1, 1]. Cheap, deterministic.
		b := h[i%32]
		v[i] = (float32(b) - 128.0) / 128.0
	}
	return v
}
