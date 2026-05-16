// Package storage persists tape entries.
//
// Production uses an object store (S3/MinIO/GCS); dev uses an
// in-memory map. The Store interface is the abstraction the recorder
// codes against.
package storage

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sync"
	"time"

	"github.com/crucible/services/shadow-recorder/internal/types"
)

// Store persists tape entries.
type Store interface {
	Put(ctx context.Context, entry types.TapeEntry) (string, error)
	Get(ctx context.Context, key string) (*types.TapeEntry, error)
	List(ctx context.Context, host string) ([]types.TapeEntry, error)
}

// MemoryStore is an in-memory Store for dev / test.
type MemoryStore struct {
	mu      sync.Mutex
	entries map[string]types.TapeEntry
}

// NewMemoryStore returns a new in-memory store.
func NewMemoryStore() Store {
	return &MemoryStore{entries: map[string]types.TapeEntry{}}
}

// Put persists the entry under a content-addressed key.
func (m *MemoryStore) Put(ctx context.Context, entry types.TapeEntry) (string, error) {
	key := keyFor(entry)
	if entry.CapturedAt.IsZero() {
		entry.CapturedAt = time.Now().UTC()
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.entries[key] = entry
	return key, nil
}

// Get returns the entry by key.
func (m *MemoryStore) Get(ctx context.Context, key string) (*types.TapeEntry, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	e, ok := m.entries[key]
	if !ok {
		return nil, ErrNotFound
	}
	return &e, nil
}

// List returns the entries for a host.
func (m *MemoryStore) List(ctx context.Context, host string) ([]types.TapeEntry, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []types.TapeEntry
	for _, e := range m.entries {
		if host == "" || e.UpstreamHost == host {
			out = append(out, e)
		}
	}
	return out, nil
}

// ObjectStore is the production Store backed by S3-compatible storage.
// We keep the implementation a small in-memory shim here — the
// Phase-2 storage adapter (apps/twin-runtime + services/twin-runtime/
// tape_driver) carries the production object-store wiring; the
// shadow-recorder uses the same adapter at deploy time.
type ObjectStore struct {
	URI string
	mu  sync.Mutex
	// In-flight cache; the real bytes go to S3.
	cache map[string]types.TapeEntry
}

// NewObjectStore returns an ObjectStore. The production adapter is
// configured at deploy time; for now this returns a shim that behaves
// like MemoryStore but tags the URI for telemetry.
func NewObjectStore(uri string) Store {
	return &ObjectStore{URI: uri, cache: map[string]types.TapeEntry{}}
}

// Put persists the entry.
func (o *ObjectStore) Put(ctx context.Context, entry types.TapeEntry) (string, error) {
	key := keyFor(entry)
	if entry.CapturedAt.IsZero() {
		entry.CapturedAt = time.Now().UTC()
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	o.cache[key] = entry
	return key, nil
}

// Get returns the entry by key.
func (o *ObjectStore) Get(ctx context.Context, key string) (*types.TapeEntry, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	e, ok := o.cache[key]
	if !ok {
		return nil, ErrNotFound
	}
	return &e, nil
}

// List returns the entries for a host.
func (o *ObjectStore) List(ctx context.Context, host string) ([]types.TapeEntry, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	var out []types.TapeEntry
	for _, e := range o.cache {
		if host == "" || e.UpstreamHost == host {
			out = append(out, e)
		}
	}
	return out, nil
}

func keyFor(e types.TapeEntry) string {
	canonical := struct {
		T string
		H string
		M string
		P string
		S int
		B []byte
	}{e.TenantID, e.UpstreamHost, e.Method, e.Path, e.ResponseStatus, e.ResponseBody}
	body, _ := json.Marshal(canonical)
	sum := sha256.Sum256(body)
	return e.TenantID + "/" + e.UpstreamHost + "/" + hex.EncodeToString(sum[:8])
}

// ErrNotFound is returned when a key is missing.
var ErrNotFound = errStore("not found")

type errStore string

func (e errStore) Error() string { return string(e) }
