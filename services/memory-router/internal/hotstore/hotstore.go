// Package hotstore is the Redis-tier client.
//
// The hot tier holds the agent's working window during a single task:
// current plan, last 50 tool calls, branch state, recall cache. TTL is
// minutes–hours. Per-tenant key scoping via the {tenant_id} hash-tag
// pattern so a tenant's keys land on the same Cluster slot.
package hotstore

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

// Client is the minimal Redis surface the router needs. Implemented by
// the real go-redis client in cmd/main; the stub here lives behind the
// same interface for unit tests and the CRUCIBLE_MEMORY_ROUTER_STUB=1
// mode.
type Client interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key, value string, ttl time.Duration) error
	Del(ctx context.Context, key string) error
	LRange(ctx context.Context, key string, start, stop int64) ([]string, error)
	LPush(ctx context.Context, key string, values ...string) (int64, error)
	LTrim(ctx context.Context, key string, start, stop int64) error
	Expire(ctx context.Context, key string, ttl time.Duration) error
	Publish(ctx context.Context, channel, message string) (int64, error)
	Close() error
}

// Store wraps a Client with tenant-aware key helpers + TTL defaults.
type Store struct {
	rdb Client
	now func() time.Time
}

// New constructs a Store. The Client is injected so the gRPC server's
// main can use a real go-redis Client while tests use the in-memory
// fake at the bottom of this file.
func New(client Client) *Store {
	return &Store{rdb: client, now: time.Now}
}

// TTLs match infra/databases/redis/keyspace.md.
const (
	TTLTaskCtx   = 6 * time.Hour
	TTLPlan      = 24 * time.Hour
	TTLBranch    = 24 * time.Hour
	TTLRecall    = 1 * time.Hour
	TTLToolCalls = 6 * time.Hour
)

// MaxToolCalls is the cap from docs/01-architecture/memory-layer.md.
const MaxToolCalls = 50

// keyTenantTask builds the {tenant_id}-tagged hot-task key.
// The braces are the Cluster hash-tag — keeps all of a tenant's keys
// on the same slot.
func keyTenantTask(tenantID, taskID, suffix string) string {
	// Sanitise: refuse to build keys for empty tenant. The router
	// gateway pre-validates but this is the second wall.
	if tenantID == "" || taskID == "" {
		return ""
	}
	return fmt.Sprintf("crucible:{%s}:task:%s:%s", tenantID, taskID, suffix)
}

func keyRecallCache(tenantID, cacheKey string) string {
	return fmt.Sprintf("crucible:{%s}:recall:%s", tenantID, cacheKey)
}

func keyInvalidatePubSub(tenantID string) string {
	return fmt.Sprintf("crucible:{%s}:invalidate", tenantID)
}

// ErrEmptyTenant is returned when a caller forgets to scope a request.
var ErrEmptyTenant = errors.New("hotstore: tenant_id required")

// SetPlan writes the current Plan JSON for a task.
func (s *Store) SetPlan(ctx context.Context, tenantID, taskID, planJSON string) error {
	k := keyTenantTask(tenantID, taskID, "plan")
	if k == "" {
		return ErrEmptyTenant
	}
	return s.rdb.Set(ctx, k, planJSON, TTLPlan)
}

// GetPlan reads the plan JSON. Returns "" + nil when absent.
func (s *Store) GetPlan(ctx context.Context, tenantID, taskID string) (string, error) {
	k := keyTenantTask(tenantID, taskID, "plan")
	if k == "" {
		return "", ErrEmptyTenant
	}
	v, err := s.rdb.Get(ctx, k)
	if errors.Is(err, ErrNotFound) {
		return "", nil
	}
	return v, err
}

// RecordToolCall pushes a tool-call summary onto the rolling list,
// trimming to MaxToolCalls.
func (s *Store) RecordToolCall(ctx context.Context, tenantID, taskID, summary string) error {
	k := keyTenantTask(tenantID, taskID, "tools_list")
	if k == "" {
		return ErrEmptyTenant
	}
	if _, err := s.rdb.LPush(ctx, k, summary); err != nil {
		return err
	}
	if err := s.rdb.LTrim(ctx, k, 0, MaxToolCalls-1); err != nil {
		return err
	}
	return s.rdb.Expire(ctx, k, TTLToolCalls)
}

// RecentToolCalls returns up to MaxToolCalls items, newest first.
func (s *Store) RecentToolCalls(ctx context.Context, tenantID, taskID string) ([]string, error) {
	k := keyTenantTask(tenantID, taskID, "tools_list")
	if k == "" {
		return nil, ErrEmptyTenant
	}
	return s.rdb.LRange(ctx, k, 0, MaxToolCalls-1)
}

// CacheRecall stores a router-output envelope so identical queries
// resolve from Redis instead of re-running the multi-signal retrieval.
func (s *Store) CacheRecall(ctx context.Context, tenantID, cacheKey, envelope string) error {
	k := keyRecallCache(tenantID, cacheKey)
	if k == "" {
		return ErrEmptyTenant
	}
	return s.rdb.Set(ctx, k, envelope, TTLRecall)
}

// GetCachedRecall fetches a previously-cached envelope. "" + nil on miss.
func (s *Store) GetCachedRecall(ctx context.Context, tenantID, cacheKey string) (string, error) {
	k := keyRecallCache(tenantID, cacheKey)
	if k == "" {
		return "", ErrEmptyTenant
	}
	v, err := s.rdb.Get(ctx, k)
	if errors.Is(err, ErrNotFound) {
		return "", nil
	}
	return v, err
}

// InvalidateRecall publishes a pubsub event that subscribers convert
// into DEL on matching cached envelopes. Called by procedural writers
// after Convention admission.
func (s *Store) InvalidateRecall(ctx context.Context, tenantID, cacheKey string) error {
	channel := keyInvalidatePubSub(tenantID)
	if channel == "" {
		return ErrEmptyTenant
	}
	_, err := s.rdb.Publish(ctx, channel, cacheKey)
	return err
}

// SanitizeCacheKey prepares a free-form input string for use in a Redis
// key. Refuses anything containing characters Redis chokes on.
func SanitizeCacheKey(in string) string {
	clean := strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
		case r == '_' || r == '-' || r == '.' || r == ':':
		default:
			return -1
		}
		return r
	}, in)
	if len(clean) > 200 {
		clean = clean[:200]
	}
	return clean
}

// ─── In-memory fake ─────────────────────────────────────────────────────────
// Used by unit tests and the CRUCIBLE_MEMORY_ROUTER_STUB=1 mode.

// ErrNotFound is what fake Get returns when the key is absent. The real
// Redis driver maps redis.Nil to this same sentinel.
var ErrNotFound = errors.New("hotstore: not found")

// NewFake returns a Client backed by an in-memory map. Safe for
// concurrent use.
func NewFake() Client {
	return &fake{kv: map[string]string{}, lists: map[string][]string{}, exp: map[string]time.Time{}}
}

type fake struct {
	mu    sync.RWMutex
	kv    map[string]string
	lists map[string][]string
	exp   map[string]time.Time
}

func (f *fake) Get(ctx context.Context, key string) (string, error) {
	_ = ctx
	f.mu.RLock()
	defer f.mu.RUnlock()
	if t, ok := f.exp[key]; ok && time.Now().After(t) {
		return "", ErrNotFound
	}
	v, ok := f.kv[key]
	if !ok {
		return "", ErrNotFound
	}
	return v, nil
}
func (f *fake) Set(ctx context.Context, key, value string, ttl time.Duration) error {
	_ = ctx
	f.mu.Lock()
	defer f.mu.Unlock()
	f.kv[key] = value
	if ttl > 0 {
		f.exp[key] = time.Now().Add(ttl)
	}
	return nil
}
func (f *fake) Del(ctx context.Context, key string) error {
	_ = ctx
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.kv, key)
	delete(f.lists, key)
	delete(f.exp, key)
	return nil
}
func (f *fake) LRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	_ = ctx
	f.mu.RLock()
	defer f.mu.RUnlock()
	lst := f.lists[key]
	if int64(len(lst)) == 0 {
		return nil, nil
	}
	s, e := int(start), int(stop)
	if s < 0 {
		s = 0
	}
	if e < 0 || e >= len(lst) {
		e = len(lst) - 1
	}
	if s > e {
		return nil, nil
	}
	out := make([]string, e-s+1)
	copy(out, lst[s:e+1])
	return out, nil
}
func (f *fake) LPush(ctx context.Context, key string, values ...string) (int64, error) {
	_ = ctx
	f.mu.Lock()
	defer f.mu.Unlock()
	lst := append([]string{}, values...) // newest at head
	for i, j := 0, len(lst)-1; i < j; i, j = i+1, j-1 {
		lst[i], lst[j] = lst[j], lst[i]
	}
	lst = append(lst, f.lists[key]...)
	f.lists[key] = lst
	return int64(len(lst)), nil
}
func (f *fake) LTrim(ctx context.Context, key string, start, stop int64) error {
	_ = ctx
	f.mu.Lock()
	defer f.mu.Unlock()
	lst := f.lists[key]
	if len(lst) == 0 {
		return nil
	}
	s, e := int(start), int(stop)
	if s < 0 {
		s = 0
	}
	if e < 0 || e >= len(lst) {
		e = len(lst) - 1
	}
	if s > e {
		f.lists[key] = nil
		return nil
	}
	f.lists[key] = append([]string{}, lst[s:e+1]...)
	return nil
}
func (f *fake) Expire(ctx context.Context, key string, ttl time.Duration) error {
	_ = ctx
	f.mu.Lock()
	defer f.mu.Unlock()
	if ttl > 0 {
		f.exp[key] = time.Now().Add(ttl)
	}
	return nil
}
func (f *fake) Publish(ctx context.Context, channel, message string) (int64, error) {
	_ = ctx
	_ = channel
	_ = message
	return 0, nil
}
func (f *fake) Close() error { return nil }
