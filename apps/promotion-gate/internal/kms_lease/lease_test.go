package kms_lease

import (
	"context"
	"strings"
	"testing"
	"time"
)

func newManager(t *testing.T) (*Manager, *DevSigner) {
	t.Helper()
	dir := t.TempDir()
	s, err := NewDevSigner(dir)
	if err != nil {
		t.Fatal(err)
	}
	return New(s, nil), s
}

func TestMintLease_RoundTrip(t *testing.T) {
	m, _ := newManager(t)
	lease, err := m.MintLease(context.Background(), LeaseRequest{
		PromotionID:  "prom_demo",
		BundleHash:   "0xabc",
		Action:       ActionDeployArtifact,
		ActionTarget: map[string]string{"service": "api", "cluster": "prod-eu"},
		OidcSubject:  "https://accounts.crucible.dev/agents/x",
	})
	if err != nil {
		t.Fatalf("MintLease: %v", err)
	}
	if lease.ExpiresAt.Sub(lease.IssuedAt) != DefaultLeaseTTL {
		t.Fatalf("expected TTL=%v, got %v", DefaultLeaseTTL, lease.ExpiresAt.Sub(lease.IssuedAt))
	}
	if lease.Sig == "" {
		t.Fatal("expected signature")
	}
	if err := m.VerifyLease(lease); err != nil {
		t.Fatalf("VerifyLease: %v", err)
	}
}

func TestVerifyLease_RejectsExpired(t *testing.T) {
	m, _ := newManager(t)
	lease, _ := m.MintLease(context.Background(), LeaseRequest{
		PromotionID: "p", BundleHash: "0x", Action: ActionDeployArtifact, TTL: 1 * time.Millisecond,
	})
	time.Sleep(5 * time.Millisecond)
	if err := m.VerifyLease(lease); err == nil {
		t.Fatal("expected expiry error")
	}
}

func TestVerifyLease_RejectsTamperedScope(t *testing.T) {
	m, _ := newManager(t)
	lease, _ := m.MintLease(context.Background(), LeaseRequest{
		PromotionID: "p", BundleHash: "0x", Action: ActionDeployArtifact,
		ActionTarget: map[string]string{"service": "api"},
	})
	lease.ActionTarget["service"] = "billing" // tamper
	if err := m.VerifyLease(lease); err == nil {
		t.Fatal("expected verify failure after tamper")
	}
}

func TestMintLease_IdempotencyConsumed(t *testing.T) {
	m, _ := newManager(t)
	req := LeaseRequest{
		PromotionID: "p", BundleHash: "0xabc", Action: ActionDeployArtifact,
	}
	if _, err := m.MintLease(context.Background(), req); err != nil {
		t.Fatal(err)
	}
	_, err := m.MintLease(context.Background(), req)
	if err == nil {
		t.Fatal("expected idempotency error on replay")
	}
	if !strings.Contains(err.Error(), "already consumed") {
		t.Fatalf("expected already-consumed error, got %v", err)
	}
}

func TestMaxLeaseTTL_Capped(t *testing.T) {
	m, _ := newManager(t)
	lease, err := m.MintLease(context.Background(), LeaseRequest{
		PromotionID: "p", BundleHash: "0x", Action: ActionDeployArtifact,
		TTL: 24 * time.Hour, // attacker-requested huge TTL
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := lease.ExpiresAt.Sub(lease.IssuedAt); got > MaxLeaseTTL {
		t.Fatalf("TTL not capped: %v > %v", got, MaxLeaseTTL)
	}
}

func TestAssertScope(t *testing.T) {
	lease := &Lease{
		Action:       ActionDeployArtifact,
		ActionTarget: map[string]string{"service": "api"},
	}
	if err := AssertScope(lease, ActionDeployArtifact, map[string]string{"service": "api"}); err != nil {
		t.Fatalf("expected scope match, got %v", err)
	}
	if err := AssertScope(lease, ActionRunMigration, nil); err == nil {
		t.Fatal("expected action mismatch")
	}
	if err := AssertScope(lease, ActionDeployArtifact, map[string]string{"service": "billing"}); err == nil {
		t.Fatal("expected target mismatch")
	}
}

func TestRequiresFields(t *testing.T) {
	m, _ := newManager(t)
	_, err := m.MintLease(context.Background(), LeaseRequest{Action: ActionDeployArtifact})
	if err == nil {
		t.Fatal("expected missing-field error")
	}
}
