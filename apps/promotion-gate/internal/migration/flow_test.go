package migration

import (
	"context"
	"strings"
	"testing"

	"github.com/crucible/promotion-gate/internal/kms_lease"
	cruciblev1 "github.com/crucible/sdk-go/crucible/v1"
)

func mkLease(t *testing.T, action, target string) *kms_lease.Lease {
	t.Helper()
	dir := t.TempDir()
	s, _ := kms_lease.NewDevSigner(dir)
	mgr := kms_lease.New(s, nil)
	l, _ := mgr.MintLease(context.Background(), kms_lease.LeaseRequest{
		PromotionID: "prom_demo",
		BundleHash:  "0x",
		Action:      action,
		ActionTarget: map[string]string{
			"migration_file": target,
		},
	})
	return l
}

func mkPlan() *Plan {
	return &Plan{
		MigrationFile: "db/migrations/20260515_refunds.sql",
		UpSQL:         "CREATE TABLE refunds (id BIGSERIAL PRIMARY KEY);",
		DownSQL:       "DROP TABLE refunds;",
		IntegrityChecks: []IntegrityCheck{
			{Name: "row_count_init", SQL: "SELECT count(*) FROM refunds", ExpectMin: 0, ExpectMax: 0},
		},
	}
}

func mkBundle() *cruciblev1.PromotionBundle {
	return &cruciblev1.PromotionBundle{
		TaskID: "task_demo", DiffHash: "0x", AgentOidcSubject: "a",
		BlastRadius: cruciblev1.BlastRadius{Reversibility: cruciblev1.ReversibilitySnapshot, ImpactScore: 0.3},
	}
}

func TestRun_HappyPath(t *testing.T) {
	d := NewFakeDriver()
	o := New(d)
	lease := mkLease(t, kms_lease.ActionRunMigration, "db/migrations/20260515_refunds.sql")
	out, err := o.Run(context.Background(), mkPlan(), lease, mkBundle())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !out.TwinOK || !out.ShadowOK || !out.RealApplied || !out.IntegrityOK {
		t.Fatalf("expected success across all 3 steps + integrity, got %+v", out)
	}
	if len(d.Applies()) != 3 {
		t.Fatalf("expected 3 applies (twin, shadow, real); got %d", len(d.Applies()))
	}
}

func TestRun_RealApplyFailureTriggersRollback(t *testing.T) {
	d := NewFakeDriver()
	scope := ScopeReal
	d.ApplyFailScope = &scope
	o := New(d)
	lease := mkLease(t, kms_lease.ActionRunMigration, "db/migrations/20260515_refunds.sql")
	out, _ := o.Run(context.Background(), mkPlan(), lease, mkBundle())
	if out.RealApplied {
		t.Fatal("expected RealApplied=false on failure")
	}
	if !out.RolledBack {
		t.Fatal("expected RolledBack=true")
	}
	if d.RollbackSQL() == "" {
		t.Fatal("expected rollback SQL recorded")
	}
}

func TestRun_IntegrityFailureRollsBack(t *testing.T) {
	d := NewFakeDriver()
	idx := 0
	d.IntegrityFailIdx = &idx
	o := New(d)
	plan := mkPlan()
	plan.IntegrityChecks = []IntegrityCheck{
		{Name: "n_refunds_within_initial_window", ExpectMin: 0, ExpectMax: 5},
	}
	lease := mkLease(t, kms_lease.ActionRunMigration, plan.MigrationFile)
	out, err := o.Run(context.Background(), plan, lease, mkBundle())
	if err == nil {
		t.Fatal("expected integrity failure to surface")
	}
	if !out.RolledBack {
		t.Fatal("expected rollback on integrity failure")
	}
	if !strings.Contains(out.Reason, "out of range") {
		t.Fatalf("expected reason on integrity failure, got %q", out.Reason)
	}
}

func TestRun_RejectsWrongLeaseScope(t *testing.T) {
	d := NewFakeDriver()
	o := New(d)
	wrong := mkLease(t, kms_lease.ActionDeployArtifact, "db/migrations/x.sql")
	_, err := o.Run(context.Background(), mkPlan(), wrong, mkBundle())
	if err == nil {
		t.Fatal("expected scope-mismatch error")
	}
}

func TestRun_RefusesDestructiveDDLWithoutDeclaration(t *testing.T) {
	d := NewFakeDriver()
	o := New(d)
	plan := mkPlan()
	plan.UpSQL = "DROP TABLE users_archived;"
	lease := mkLease(t, kms_lease.ActionRunMigration, plan.MigrationFile)
	_, err := o.Run(context.Background(), plan, lease, mkBundle())
	if err == nil {
		t.Fatal("expected destructive-DDL refusal")
	}
}
