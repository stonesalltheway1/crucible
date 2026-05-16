package store

import (
	"errors"
	"testing"
	"time"

	cruciblev1 "github.com/crucible/sdk-go/crucible/v1"
)

func TestPutAndGet_Roundtrip(t *testing.T) {
	s := New()
	in := &cruciblev1.Task{
		ID:          "task_1",
		TenantID:    "ten_a",
		Repo:        "x/y",
		Description: "demo",
		Status:      cruciblev1.TaskStatusPlanning,
		CreatedAt:   time.Now().UTC(),
	}
	if err := s.Put(in); err != nil {
		t.Fatal(err)
	}
	got, err := s.Get("task_1")
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != in.ID || got.Description != in.Description {
		t.Fatalf("roundtrip mismatch: %+v", got)
	}
	// Mutating the returned copy must not affect the stored task.
	got.Description = "mutated"
	again, _ := s.Get("task_1")
	if again.Description == "mutated" {
		t.Fatal("Get did not return an isolated copy")
	}
}

func TestGet_NotFound(t *testing.T) {
	s := New()
	_, err := s.Get("nope")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestPut_RejectsNilAndEmptyID(t *testing.T) {
	s := New()
	if err := s.Put(nil); err == nil {
		t.Fatal("expected error on nil")
	}
	if err := s.Put(&cruciblev1.Task{}); err == nil {
		t.Fatal("expected error on empty id")
	}
}

func TestUpdate_AppliesUnderLock(t *testing.T) {
	s := New()
	_ = s.Put(&cruciblev1.Task{ID: "task_1", TenantID: "t", Status: cruciblev1.TaskStatusPlanning})
	updated, err := s.Update("task_1", func(t *cruciblev1.Task) error {
		t.Status = cruciblev1.TaskStatusApproved
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Status != cruciblev1.TaskStatusApproved {
		t.Fatalf("expected approved, got %s", updated.Status)
	}
	// And the underlying store reflects the change.
	got, _ := s.Get("task_1")
	if got.Status != cruciblev1.TaskStatusApproved {
		t.Fatalf("update not persisted: %s", got.Status)
	}
}

func TestUpdate_PropagatesError(t *testing.T) {
	s := New()
	_ = s.Put(&cruciblev1.Task{ID: "task_1", TenantID: "t"})
	custom := errors.New("nope")
	_, err := s.Update("task_1", func(t *cruciblev1.Task) error { return custom })
	if !errors.Is(err, custom) {
		t.Fatalf("expected pass-through error, got %v", err)
	}
}

func TestUpdate_NotFound(t *testing.T) {
	s := New()
	_, err := s.Update("nope", func(t *cruciblev1.Task) error { return nil })
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestList_FiltersAndSorts(t *testing.T) {
	s := New()
	now := time.Now().UTC()
	_ = s.Put(&cruciblev1.Task{ID: "a", TenantID: "x", CreatedAt: now.Add(-2 * time.Hour)})
	_ = s.Put(&cruciblev1.Task{ID: "b", TenantID: "x", CreatedAt: now.Add(-1 * time.Hour)})
	_ = s.Put(&cruciblev1.Task{ID: "c", TenantID: "y", CreatedAt: now})

	xs := s.List("x", 0)
	if len(xs) != 2 {
		t.Fatalf("expected 2 tasks for tenant x, got %d", len(xs))
	}
	if xs[0].ID != "b" || xs[1].ID != "a" {
		t.Fatalf("expected newest-first, got %v", []string{xs[0].ID, xs[1].ID})
	}
	all := s.List("", 0)
	if len(all) != 3 {
		t.Fatalf("expected 3 across all tenants, got %d", len(all))
	}
	limited := s.List("", 1)
	if len(limited) != 1 {
		t.Fatalf("expected limit=1 to return 1, got %d", len(limited))
	}
}
