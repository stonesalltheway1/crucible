package policy

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	cruciblev1 "github.com/crucible/sdk-go/crucible/v1"
)

// passingInput is the canonical "trivial diff with verifier approval" input.
func passingInput() map[string]any {
	return mustToMap(PromotionInput{
		TaskID:                      "task_demo",
		TenantID:                    "ten_demo",
		DiffHash:                    "0xabc",
		VerifierApprovalAttestation: "rekor:abc",
		BuildProvenanceAttestation:  "rekor:slsa",
		RebuildHash:                 "0xreb",
		AgentOidcSubject:            "https://accounts.crucible.dev/agents/x",
		BlastRadius: PromotionBlastRadius{
			EstimatedImpact: "low",
			Reversibility:   cruciblev1.ReversibilityTrivial,
			ImpactScore:     0.1,
		},
		TierResults: PromotionTierResults{
			Tier0: &TierEntry{Passed: true},
			Tier1: &TierEntry{Passed: true},
			Tier4: &TierEntry{Passed: true},
		},
		Context: PromotionContext{Geo: "us"},
	})
}

func mustToMap(p PromotionInput) map[string]any {
	b, err := json.Marshal(p)
	if err != nil {
		panic(err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		panic(err)
	}
	return m
}

func TestDefaultPromotionEngine_AutoApproveTrivial(t *testing.T) {
	eng, err := DefaultPromotionEngine(context.Background())
	if err != nil {
		t.Fatalf("DefaultPromotionEngine: %v", err)
	}
	dec, err := eng.Evaluate(context.Background(), passingInput())
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if !dec.Allow || dec.NeedsHuman || !dec.AutoApprove {
		t.Fatalf("expected auto-approve, got %+v", dec)
	}
}

func TestDefaultPromotionEngine_RejectsSelfApproval(t *testing.T) {
	eng, err := DefaultPromotionEngine(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	pi := PromotionInput{
		TaskID:                      "t",
		TenantID:                    "ten_demo",
		VerifierApprovalAttestation: "rekor:abc",
		AgentOidcSubject:            "agent_oidc",
		BlastRadius: PromotionBlastRadius{
			EstimatedImpact: "low",
			Reversibility:   cruciblev1.ReversibilityTrivial,
		},
		TierResults: PromotionTierResults{
			Tier0: &TierEntry{Passed: true},
			Tier1: &TierEntry{Passed: true},
			Tier4: &TierEntry{Passed: true},
		},
		Approvals: []ApprovalRecord{
			{ApproverOidcSubject: "agent_oidc", Attestation: "rekor:approval", ApprovedAt: time.Now()},
		},
	}
	dec, err := eng.Evaluate(context.Background(), mustToMap(pi))
	if err != nil {
		t.Fatal(err)
	}
	if dec.Allow {
		t.Fatalf("expected deny on self-approval, got %+v", dec)
	}
	if !containsReason(dec.Reasons, "self-approval forbidden") {
		t.Fatalf("expected self-approval reason, got %v", dec.Reasons)
	}
}

func TestDefaultPromotionEngine_RejectsMergeFreeze(t *testing.T) {
	eng, _ := DefaultPromotionEngine(context.Background())
	pi := passingInput()
	pi["context"] = map[string]any{
		"merge_freeze":        true,
		"merge_freeze_until":  "2026-06-01T00:00:00Z",
		"merge_freeze_reason": "mobile release cut",
	}
	dec, err := eng.Evaluate(context.Background(), pi)
	if err != nil {
		t.Fatal(err)
	}
	if dec.Allow {
		t.Fatalf("expected deny during merge freeze, got %+v", dec)
	}
}

func TestDefaultPromotionEngine_RequiresHumanForSchemaChange(t *testing.T) {
	eng, _ := DefaultPromotionEngine(context.Background())
	pi := passingInput()
	pi["blast_radius"].(map[string]any)["schema_changes"] = []any{
		map[string]any{"file": "db/migrations/x.sql"},
	}
	pi["blast_radius"].(map[string]any)["estimated_impact"] = "medium"
	dec, err := eng.Evaluate(context.Background(), pi)
	if err != nil {
		t.Fatal(err)
	}
	if dec.Allow {
		t.Fatalf("expected needs-human for schema change, got allow=%v", dec.Allow)
	}
	if !dec.NeedsHuman {
		t.Fatalf("expected needs-human, got %+v", dec)
	}
	if dec.RequireNApprovers == 0 {
		t.Fatalf("expected at least 1 approver, got %+v", dec)
	}
}

func TestDefaultPromotionEngine_AllowsSchemaWithHumanApproval(t *testing.T) {
	eng, _ := DefaultPromotionEngine(context.Background())
	pi := passingInput()
	pi["blast_radius"].(map[string]any)["schema_changes"] = []any{
		map[string]any{"file": "db/migrations/x.sql"},
	}
	pi["blast_radius"].(map[string]any)["estimated_impact"] = "medium"
	pi["approvals"] = []any{
		map[string]any{"approver_oidc_subject": "dba@acme", "attestation": "rekor:approval-1"},
	}
	dec, err := eng.Evaluate(context.Background(), pi)
	if err != nil {
		t.Fatal(err)
	}
	if !dec.Allow {
		t.Fatalf("expected allow with human approval, got %+v", dec)
	}
}

func TestDefaultPromotionEngine_RequiresTier4(t *testing.T) {
	eng, _ := DefaultPromotionEngine(context.Background())
	pi := passingInput()
	pi["blast_radius"].(map[string]any)["estimated_impact"] = "high"
	tier := pi["tier_results"].(map[string]any)
	delete(tier, "tier_4")
	dec, err := eng.Evaluate(context.Background(), pi)
	if err != nil {
		t.Fatal(err)
	}
	if dec.Allow {
		t.Fatalf("expected deny without tier4 on non-trivial, got %+v", dec)
	}
	if !containsReason(dec.Reasons, "Tier 4") {
		t.Fatalf("expected tier4 reason, got %v", dec.Reasons)
	}
}

func TestDefaultPromotionEngine_CriticalPathNeedsCodeowner(t *testing.T) {
	eng, _ := DefaultPromotionEngine(context.Background())
	pi := passingInput()
	pi["blast_radius"].(map[string]any)["critical_paths_touched"] = []any{"src/billing/refunds.go"}
	pi["blast_radius"].(map[string]any)["estimated_impact"] = "medium"
	pi["tier_results"].(map[string]any)["tier_3"] = map[string]any{"passed": true}
	pi["codeowners"] = map[string]any{"matched": []any{
		map[string]any{"path_glob": "src/billing/*", "groups": []any{"@payments-leads"}},
	}}
	dec, err := eng.Evaluate(context.Background(), pi)
	if err != nil {
		t.Fatal(err)
	}
	if !dec.RequireCodeowner {
		t.Fatalf("expected codeowner requirement, got %+v", dec)
	}
	// Should require 2 approvers by default for critical path.
	if dec.RequireNApprovers < 2 {
		t.Fatalf("expected ≥2 approvers for critical path, got %d", dec.RequireNApprovers)
	}
}

func TestDefaultPromotionEngine_IrreversibleNeedsHuman(t *testing.T) {
	eng, _ := DefaultPromotionEngine(context.Background())
	pi := passingInput()
	pi["blast_radius"].(map[string]any)["reversibility"] = "irreversible"
	pi["blast_radius"].(map[string]any)["estimated_impact"] = "high"
	dec, err := eng.Evaluate(context.Background(), pi)
	if err != nil {
		t.Fatal(err)
	}
	if dec.Allow {
		t.Fatalf("expected deny for irreversible without human approval, got %+v", dec)
	}
}

func TestDefaultPromotionEngine_MissingVerifier(t *testing.T) {
	eng, _ := DefaultPromotionEngine(context.Background())
	pi := passingInput()
	pi["verifier_approval_attestation"] = ""
	dec, err := eng.Evaluate(context.Background(), pi)
	if err != nil {
		t.Fatal(err)
	}
	if dec.Allow {
		t.Fatalf("expected deny without verifier approval, got %+v", dec)
	}
}

func TestTenantBundle_Validates(t *testing.T) {
	bad := &TenantBundle{TenantID: "ten_demo", Modules: map[string]string{
		"my.rego": "package x\nallow = true",
	}}
	if err := bad.Validate(); err == nil {
		t.Fatal("expected validate error: missing package crucible.promotion.tenant")
	}

	ok := &TenantBundle{TenantID: "ten_demo", Modules: map[string]string{
		"tenant.rego": "package crucible.promotion.tenant\nimport rego.v1\nallow := false",
	}, Version: 1, IssuedAt: time.Now()}
	if err := ok.Validate(); err != nil {
		t.Fatalf("expected ok, got %v", err)
	}
}

func TestSignAndVerifyBundle(t *testing.T) {
	signer, err := NewEd25519Signer("https://accounts.crucible.dev/tenant/ten_demo")
	if err != nil {
		t.Fatal(err)
	}
	tb := &TenantBundle{
		TenantID: "ten_demo",
		Modules: map[string]string{
			"tenant.rego": "package crucible.promotion.tenant\nimport rego.v1\ndecision := {\"allow\": false}",
		},
		Version: 1, IssuedAt: time.Now(),
	}
	env, err := SignBundle(tb, signer)
	if err != nil {
		t.Fatalf("SignBundle: %v", err)
	}
	if env.BundleHash == "" || env.Signature == "" {
		t.Fatal("missing fields in signed bundle")
	}
	got, err := VerifyBundle(env, signer)
	if err != nil {
		t.Fatalf("VerifyBundle: %v", err)
	}
	if got.TenantID != tb.TenantID {
		t.Fatalf("tenant id round-trip mismatch")
	}
	// Tamper.
	env.Bundle.Description = "tampered"
	if _, err := VerifyBundle(env, signer); err == nil {
		t.Fatal("expected verify failure on tamper")
	}
}

func TestLayeredEngine_TenantHardDenyWins(t *testing.T) {
	tb := &TenantBundle{
		TenantID: "ten_demo",
		Modules: map[string]string{
			"tenant.rego": `
package crucible.promotion.tenant
import rego.v1
default decision := {"allow": false, "needs_human": false, "reasons": ["tenant: deny all"], "auto_approve": false, "require_codeowner": false, "approver_groups": [], "require_n_approvers": 0}
`,
		},
		Version:  1,
		IssuedAt: time.Now(),
	}
	eng, err := TenantEngine(context.Background(), tb)
	if err != nil {
		t.Fatal(err)
	}
	dec, err := eng.Evaluate(context.Background(), passingInput())
	if err != nil {
		t.Fatal(err)
	}
	if dec.Allow {
		t.Fatalf("expected tenant policy to deny, got %+v", dec)
	}
}

func TestPolicyHash_StableAcrossRebuild(t *testing.T) {
	modA := map[string]string{"a.rego": "package x\nimport rego.v1\nallow := true"}
	modB := map[string]string{"a.rego": "package x\nimport rego.v1\nallow := true"}
	if HashModules(modA) != HashModules(modB) {
		t.Fatal("hashes should match for identical modules")
	}
	modC := map[string]string{"a.rego": "package x\nimport rego.v1\nallow := false"}
	if HashModules(modA) == HashModules(modC) {
		t.Fatal("hashes should differ when source differs")
	}
}

func TestEvaluate_ParsesBooleanResult(t *testing.T) {
	mod := `package x
import rego.v1
default allow := false
allow if input.go == true`
	eng, err := New(context.Background(), "data.x.allow", map[string]string{"x.rego": mod})
	if err != nil {
		t.Fatal(err)
	}
	dec, err := eng.Evaluate(context.Background(), map[string]any{"go": true})
	if err != nil {
		t.Fatal(err)
	}
	if !dec.Allow {
		t.Fatal("expected allow=true")
	}
	dec, err = eng.Evaluate(context.Background(), map[string]any{"go": false})
	if err != nil {
		t.Fatal(err)
	}
	if dec.Allow {
		t.Fatal("expected allow=false")
	}
}

func TestNew_RejectsEmpty(t *testing.T) {
	if _, err := New(context.Background(), "", map[string]string{"m.rego": "package x"}); err == nil {
		t.Fatal("expected error on empty query")
	}
	if _, err := New(context.Background(), "data.x.allow", map[string]string{}); err == nil {
		t.Fatal("expected error on empty modules")
	}
}

func TestDefaultPromotionModule_ContainsEntryPoint(t *testing.T) {
	mod := DefaultPromotionModule()
	if !strings.Contains(mod, "package crucible.promotion") {
		t.Fatal("default module missing package decl")
	}
	if !strings.Contains(mod, "decision") {
		t.Fatal("default module missing decision rule")
	}
}

func containsReason(reasons []string, substr string) bool {
	for _, r := range reasons {
		if strings.Contains(r, substr) {
			return true
		}
	}
	return false
}
