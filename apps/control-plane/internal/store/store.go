// Package store is the Phase-1 in-memory task store. Phase 2 swaps this for
// Postgres-backed persistence with row-level tenant security.
package store

import (
	"errors"
	"sync"
	"time"

	cruciblev1 "github.com/crucible/sdk-go/crucible/v1"
)

// ErrNotFound is returned by Get when the task id is unknown.
var ErrNotFound = errors.New("store: task not found")

// Store is the in-memory task store.
type Store struct {
	mu    sync.RWMutex
	tasks map[string]*cruciblev1.Task
}

// New returns an empty Store.
func New() *Store {
	return &Store{tasks: make(map[string]*cruciblev1.Task)}
}

// Put writes (or overwrites) a task and stamps UpdatedAt.
func (s *Store) Put(task *cruciblev1.Task) error {
	if task == nil {
		return errors.New("store: nil task")
	}
	if task.ID == "" {
		return errors.New("store: task missing id")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	clone := *task
	clone.UpdatedAt = time.Now().UTC()
	s.tasks[task.ID] = &clone
	return nil
}

// Get returns a deep-ish copy of the task; mutations by the caller do not
// affect store state.
func (s *Store) Get(id string) (*cruciblev1.Task, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	t, ok := s.tasks[id]
	if !ok {
		return nil, ErrNotFound
	}
	clone := *t
	return &clone, nil
}

// List returns tasks for a tenant, newest-first. Pass empty tenant to list all.
func (s *Store) List(tenantID string, limit int) []*cruciblev1.Task {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*cruciblev1.Task, 0, len(s.tasks))
	for _, t := range s.tasks {
		if tenantID != "" && t.TenantID != tenantID {
			continue
		}
		clone := *t
		out = append(out, &clone)
	}
	// Sort newest-first by CreatedAt.
	sortByCreatedAtDesc(out)
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out
}

// Update applies fn under the store's lock so callers can mutate the task
// atomically.
func (s *Store) Update(id string, fn func(*cruciblev1.Task) error) (*cruciblev1.Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.tasks[id]
	if !ok {
		return nil, ErrNotFound
	}
	if err := fn(t); err != nil {
		return nil, err
	}
	t.UpdatedAt = time.Now().UTC()
	clone := *t
	return &clone, nil
}

func sortByCreatedAtDesc(ts []*cruciblev1.Task) {
	// insertion sort; small n in Phase 1
	for i := 1; i < len(ts); i++ {
		j := i
		for j > 0 && ts[j-1].CreatedAt.Before(ts[j].CreatedAt) {
			ts[j-1], ts[j] = ts[j], ts[j-1]
			j--
		}
	}
}
