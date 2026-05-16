package twinbridge

import (
	"context"
	"errors"
	"testing"
)

func TestStubBridgeRoundtrip(t *testing.T) {
	b := NewStub()
	r, err := b.Spawn(context.Background(), SpawnRequest{
		TaskID:   "task_test",
		TenantID: "ten_test",
		BaseSHA:  "abc",
		RepoURL:  "https://example.invalid/repo",
	})
	if err != nil {
		t.Fatal(err)
	}
	if r.SandboxID == "" {
		t.Fatal("expected non-empty SandboxID")
	}
	if err := b.Kill(context.Background(), r.SandboxID, "manual"); err != nil {
		t.Fatal(err)
	}
}

func TestStubBridgeRequiresTaskAndTenant(t *testing.T) {
	b := NewStub()
	_, err := b.Spawn(context.Background(), SpawnRequest{})
	if err == nil {
		t.Fatal("expected error for empty request")
	}
}

func TestRealBridgeReturnsNotConnectedWithoutTransport(t *testing.T) {
	b := New()
	_, err := b.Spawn(context.Background(), SpawnRequest{
		TaskID:   "t",
		TenantID: "ten",
	})
	if err == nil {
		t.Fatal("expected NotConnectedError without transport")
	}
	var nce *NotConnectedError
	if !errors.As(err, &nce) {
		t.Fatalf("expected NotConnectedError, got %T: %v", err, err)
	}
}
