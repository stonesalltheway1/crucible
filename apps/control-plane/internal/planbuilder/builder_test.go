package planbuilder

import (
	"context"
	"testing"
	"time"

	"github.com/crucible/attestation"
	"github.com/crucible/control-plane/internal/modelrouter"
	cruciblev1 "github.com/crucible/sdk-go/crucible/v1"
)

// mkAttestSvc builds an attestation.Service backed by tempdir signer + journal.
func mkAttestSvc(t *testing.T) *attestation.Service {
	t.Helper()
	dir := t.TempDir()
	signer, err := attestation.NewLocalEd25519Signer(dir + "/keys")
	if err != nil {
		t.Fatal(err)
	}
	pub, err := attestation.NewLocalJournalPublisher(dir + "/journal.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	svc, err := attestation.NewService(signer, pub)
	if err != nil {
		t.Fatal(err)
	}
	return svc
}

func TestFallbackPlan_StableShape(t *testing.T) {
	task := &cruciblev1.Task{
		ID:          "task_1",
		TenantID:    "ten_a",
		Repo:        "github.com/x/y",
		Description: "test description",
	}
	p := fallbackPlan(task)
	if p.Description != "test description" {
		t.Fatalf("expected description carried through, got %q", p.Description)
	}
	if len(p.Steps) == 0 {
		t.Fatal("fallback plan must have steps")
	}
	if p.RetryBudgetPerStep != 3 {
		t.Fatalf("ADR-009 default of 3 expected, got %d", p.RetryBudgetPerStep)
	}
	if p.EstimatedCostUsd <= 0 {
		t.Fatal("fallback plan must declare a non-zero cost")
	}
}

func TestComputePlanHash_StableAcrossBuildTime(t *testing.T) {
	p1 := &cruciblev1.Plan{Description: "d", EstimatedCostUsd: 1.0, BuiltAt: time.Now()}
	p2 := &cruciblev1.Plan{Description: "d", EstimatedCostUsd: 1.0, BuiltAt: time.Now().Add(time.Hour)}
	h1 := computePlanHash(p1)
	h2 := computePlanHash(p2)
	if h1 != h2 {
		t.Fatalf("plan_hash unstable across BuiltAt: %s vs %s", h1, h2)
	}
}

func TestComputePlanHash_ChangesOnDescriptionChange(t *testing.T) {
	p1 := &cruciblev1.Plan{Description: "a", EstimatedCostUsd: 1.0}
	p2 := &cruciblev1.Plan{Description: "b", EstimatedCostUsd: 1.0}
	if computePlanHash(p1) == computePlanHash(p2) {
		t.Fatal("expected plan_hash to differ on description change")
	}
}

func TestBuild_EmitsPlanProposalAttestation_NoLLM(t *testing.T) {
	svc := mkAttestSvc(t)
	b := New(modelrouter.NewRouter(), svc, "") // no vendor → fallback path
	task := &cruciblev1.Task{
		ID:          "task_99",
		TenantID:    "ten_a",
		Description: "test fallback",
	}
	plan, entry, err := b.Build(context.Background(), task)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if plan == nil {
		t.Fatal("expected plan")
	}
	if entry == nil || entry.UUID == "" {
		t.Fatal("expected attestation entry")
	}
	if !entry.LocalJournalFallback {
		t.Fatal("expected local journal fallback flag")
	}
	if plan.PlanHash == "" {
		t.Fatal("expected plan_hash to be set")
	}
	if plan.TaskID != "task_99" {
		t.Fatalf("expected task id stamped, got %s", plan.TaskID)
	}
}

func TestBuild_RejectsNilTask(t *testing.T) {
	svc := mkAttestSvc(t)
	b := New(nil, svc, "")
	if _, _, err := b.Build(context.Background(), nil); err == nil {
		t.Fatal("expected error on nil task")
	}
}

func TestParsePlanJSON_StripsFencesAndValidates(t *testing.T) {
	in := "```json\n{\"description\":\"d\",\"steps\":[],\"estimated_cost_usd\":1.0,\"estimated_duration_min\":5,\"complexity\":\"standard\",\"retry_budget_per_step\":3,\"wall_clock_budget_min\":30}\n```"
	p, err := parsePlanJSON(in)
	if err != nil {
		t.Fatal(err)
	}
	if p.Description != "d" {
		t.Fatalf("description: %s", p.Description)
	}

	if _, err := parsePlanJSON(`{"steps": []}`); err == nil {
		t.Fatal("expected error on missing description")
	}
}
