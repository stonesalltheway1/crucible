package hotstore

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestStore_RefusesEmptyTenant(t *testing.T) {
	s := New(NewFake())
	ctx := context.Background()
	if err := s.SetPlan(ctx, "", "task_1", "{}"); !errors.Is(err, ErrEmptyTenant) {
		t.Fatalf("want ErrEmptyTenant; got %v", err)
	}
	if _, err := s.GetPlan(ctx, "ten_a", ""); !errors.Is(err, ErrEmptyTenant) {
		t.Fatalf("want ErrEmptyTenant on empty task; got %v", err)
	}
}

func TestStore_TenantHashTagInKey(t *testing.T) {
	k := keyTenantTask("ten_abc", "task_1", "plan")
	if k == "" || k[10] != '{' {
		t.Fatalf("expected hash-tag key; got %q", k)
	}
}

func TestStore_PlanSetGet(t *testing.T) {
	s := New(NewFake())
	ctx := context.Background()
	if err := s.SetPlan(ctx, "ten_a", "task_1", `{"plan":1}`); err != nil {
		t.Fatal(err)
	}
	got, err := s.GetPlan(ctx, "ten_a", "task_1")
	if err != nil {
		t.Fatal(err)
	}
	if got != `{"plan":1}` {
		t.Fatalf("got %q", got)
	}
}

func TestStore_ToolCallsTrimmedTo50(t *testing.T) {
	s := New(NewFake())
	ctx := context.Background()
	for i := 0; i < 60; i++ {
		if err := s.RecordToolCall(ctx, "ten_a", "task_1", "call"); err != nil {
			t.Fatal(err)
		}
	}
	got, err := s.RecentToolCalls(ctx, "ten_a", "task_1")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) > MaxToolCalls {
		t.Fatalf("tool list must be trimmed to %d; got %d", MaxToolCalls, len(got))
	}
}

func TestStore_TenantIsolation(t *testing.T) {
	s := New(NewFake())
	ctx := context.Background()
	if err := s.SetPlan(ctx, "ten_a", "task_1", "A"); err != nil {
		t.Fatal(err)
	}
	if err := s.SetPlan(ctx, "ten_b", "task_1", "B"); err != nil {
		t.Fatal(err)
	}
	a, _ := s.GetPlan(ctx, "ten_a", "task_1")
	b, _ := s.GetPlan(ctx, "ten_b", "task_1")
	if a != "A" || b != "B" {
		t.Fatalf("tenant cross-leak: a=%q b=%q", a, b)
	}
}

func TestStore_PlanTTLExpires(t *testing.T) {
	s := New(NewFake())
	ctx := context.Background()
	if err := s.rdb.Set(ctx, "k", "v", 5*time.Millisecond); err != nil {
		t.Fatal(err)
	}
	time.Sleep(10 * time.Millisecond)
	if _, err := s.rdb.Get(ctx, "k"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("want ErrNotFound after TTL; got %v", err)
	}
}

func TestSanitizeCacheKey_StripsBadChars(t *testing.T) {
	got := SanitizeCacheKey(`hello "world"`)
	if got != "helloworld" {
		t.Fatalf("got %q", got)
	}
}

func TestSanitizeCacheKey_LengthCap(t *testing.T) {
	in := make([]byte, 300)
	for i := range in {
		in[i] = 'a'
	}
	if got := SanitizeCacheKey(string(in)); len(got) != 200 {
		t.Fatalf("cap should be 200; got %d", len(got))
	}
}
