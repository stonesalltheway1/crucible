package rego_engine

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/crucible/policy"
	cruciblev1 "github.com/crucible/sdk-go/crucible/v1"
)

func trivialInput() *policy.PromotionInput {
	return &policy.PromotionInput{
		TaskID:                      "task_demo",
		TenantID:                    "ten_demo",
		DiffHash:                    "0xabc",
		VerifierApprovalAttestation: "rekor:abc",
		AgentOidcSubject:            "https://accounts.crucible.dev/agents/x",
		BlastRadius: policy.PromotionBlastRadius{
			EstimatedImpact: "low",
			Reversibility:   cruciblev1.ReversibilityTrivial,
		},
		TierResults: policy.PromotionTierResults{
			Tier0: &policy.TierEntry{Passed: true},
			Tier1: &policy.TierEntry{Passed: true},
			Tier4: &policy.TierEntry{Passed: true},
		},
	}
}

func TestEvaluate_DefaultAutoApprove(t *testing.T) {
	eng, err := New(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	dec, err := eng.Evaluate(context.Background(), trivialInput())
	if err != nil {
		t.Fatal(err)
	}
	if !dec.IsApproved() {
		t.Fatalf("expected auto-approve, got %+v", dec)
	}
	if dec.PolicyHash == "" {
		t.Fatal("expected policy hash")
	}
}

func TestEvaluate_TenantBlocksDespiteDefaultAllow(t *testing.T) {
	eng, err := New(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	tb := &policy.TenantBundle{
		TenantID: "ten_demo",
		Modules: map[string]string{
			"deny_all.rego": `
package crucible.promotion.tenant
import rego.v1
default decision := {"allow": false, "needs_human": true, "reasons": ["tenant kill switch on"], "auto_approve": false, "require_codeowner": false, "approver_groups": [], "require_n_approvers": 1}
`,
		},
		Version: 1, IssuedAt: time.Now(),
	}
	if err := eng.LoadTenant(context.Background(), tb); err != nil {
		t.Fatal(err)
	}
	dec, err := eng.Evaluate(context.Background(), trivialInput())
	if err != nil {
		t.Fatal(err)
	}
	if dec.Allow {
		t.Fatalf("expected deny via tenant override, got %+v", dec)
	}
	found := false
	for _, r := range dec.Reasons {
		if strings.Contains(r, "tenant kill switch on") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected tenant reason, got %v", dec.Reasons)
	}
}

func TestEvaluate_TenantNotLoadedFallsThroughToDefault(t *testing.T) {
	eng, _ := New(context.Background())
	dec, err := eng.Evaluate(context.Background(), trivialInput())
	if err != nil {
		t.Fatal(err)
	}
	if !dec.Allow {
		t.Fatalf("expected default allow when no tenant override, got %+v", dec)
	}
	if dec.TenantDecision != nil {
		t.Fatal("expected no tenant decision when not loaded")
	}
}

func TestLoadTenant_VersionCaching(t *testing.T) {
	eng, _ := New(context.Background())
	tb := &policy.TenantBundle{
		TenantID: "ten_x",
		Modules: map[string]string{
			"t.rego": `package crucible.promotion.tenant
import rego.v1
default decision := {"allow": true, "needs_human": false, "reasons": [], "auto_approve": true, "require_codeowner": false, "approver_groups": [], "require_n_approvers": 0}`,
		},
		Version: 2, IssuedAt: time.Now(),
	}
	if err := eng.LoadTenant(context.Background(), tb); err != nil {
		t.Fatal(err)
	}
	// Reloading older version is a no-op.
	tb.Version = 1
	if err := eng.LoadTenant(context.Background(), tb); err != nil {
		t.Fatal(err)
	}
	if !eng.HasTenant("ten_x") {
		t.Fatal("tenant should be cached")
	}
}
