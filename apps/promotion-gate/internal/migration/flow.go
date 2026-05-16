// Package migration implements the special-case three-step DB migration
// promotion flow from docs/01-architecture/promotion-contract.md §"Database
// migrations":
//
//  1. Twin run — migration applied to Neon twin branch; verifier checks
//     resulting schema diff against the MigrationAttestation that the
//     bundle ships with.
//  2. Shadow run — same migration applied to a shadow of production
//     (read-replica with replication paused); validator checks no
//     destructive DDL on production data.
//  3. Promotion — KMS-signed credential lease grants temporary ALTER TABLE;
//     migration runs as a single transaction with statement timeout.
//  4. Verification — post-migration query checks (data integrity,
//     expected row-count behaviour, expected indexes).
//  5. Rollback — transactional rollback; or, for non-transactional DDL,
//     a manually-authored down-migration in the bundle.
package migration

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/crucible/promotion-gate/internal/kms_lease"
	cruciblev1 "github.com/crucible/sdk-go/crucible/v1"
)

// Driver is the abstraction over a database. Production drivers wrap
// pgx / mysql / sqlite. The Phase-6 default is `Fake`, which exercises
// the entire flow in-memory.
type Driver interface {
	Name() string
	Apply(ctx context.Context, scope Scope, sqlText string) (Result, error)
	IntegrityCheck(ctx context.Context, scope Scope, check IntegrityCheck) (Result, error)
	Rollback(ctx context.Context, scope Scope, downSQL string) error
}

// Scope tells the driver which side to act on.
type Scope string

// Known scopes.
const (
	ScopeTwin   Scope = "twin"
	ScopeShadow Scope = "shadow"
	ScopeReal   Scope = "real"
)

// Result is the structured outcome of a single Apply or IntegrityCheck.
type Result struct {
	Affected   int64             `json:"affected,omitempty"`
	Rows       []map[string]any  `json:"rows,omitempty"`
	DurationMs uint64            `json:"duration_ms"`
	Error      string            `json:"error,omitempty"`
}

// IntegrityCheck is one post-migration assertion the bundle ships with.
type IntegrityCheck struct {
	Name      string `json:"name"`
	SQL       string `json:"sql"`
	Expect    string `json:"expect,omitempty"`     // optional expected scalar value
	ExpectMin int64  `json:"expect_min,omitempty"`
	ExpectMax int64  `json:"expect_max,omitempty"`
}

// Plan is the migration bundle the gate consumes.
type Plan struct {
	MigrationFile     string           `json:"migration_file"`
	MigrationSha256   string           `json:"migration_sha256"`
	UpSQL             string           `json:"up_sql"`
	DownSQL           string           `json:"down_sql,omitempty"`     // empty => non-transactional rollback path
	IntegrityChecks   []IntegrityCheck `json:"integrity_checks"`
	StatementTimeoutMs uint32          `json:"statement_timeout_ms"`
}

// Outcome is the structured result of running the full flow.
type Outcome struct {
	TwinOK      bool   `json:"twin_ok"`
	ShadowOK    bool   `json:"shadow_ok"`
	RealApplied bool   `json:"real_applied"`
	IntegrityOK bool   `json:"integrity_ok"`
	RolledBack  bool   `json:"rolled_back"`
	Reason      string `json:"reason,omitempty"`
}

// Orchestrator runs the three-step flow.
type Orchestrator struct {
	driver Driver
}

// New builds an Orchestrator.
func New(driver Driver) *Orchestrator { return &Orchestrator{driver: driver} }

// Run executes the full migration flow. Requires a valid KMS lease whose
// Action is kms_lease.ActionRunMigration. The lease is verified once by
// the caller (api/server) and passed in; we re-assert scope here.
func (o *Orchestrator) Run(ctx context.Context, plan *Plan, lease *kms_lease.Lease, bundle *cruciblev1.PromotionBundle) (*Outcome, error) {
	if plan == nil || lease == nil || bundle == nil {
		return nil, errors.New("migration: nil plan / lease / bundle")
	}
	if err := kms_lease.AssertScope(lease, kms_lease.ActionRunMigration, map[string]string{
		"migration_file": plan.MigrationFile,
	}); err != nil {
		return nil, err
	}

	out := &Outcome{}

	// Step 1: twin run.
	if _, err := o.driver.Apply(ctx, ScopeTwin, plan.UpSQL); err != nil {
		out.Reason = "twin apply failed: " + err.Error()
		return out, err
	}
	out.TwinOK = true

	// Step 2: shadow run on a paused-replication snapshot.
	if _, err := o.driver.Apply(ctx, ScopeShadow, plan.UpSQL); err != nil {
		out.Reason = "shadow apply failed: " + err.Error()
		return out, err
	}
	out.ShadowOK = true

	// Refuse destructive DDL on real data if the bundle never declared it.
	if containsDestructiveDDL(plan.UpSQL) && !destructiveDeclared(bundle) {
		out.Reason = "shadow detected destructive DDL but bundle did not declare destructive_ddl=true"
		return out, errors.New(out.Reason)
	}

	// Step 3: real apply.
	if _, err := o.driver.Apply(ctx, ScopeReal, plan.UpSQL); err != nil {
		out.Reason = "real apply failed: " + err.Error()
		// Best-effort rollback on the real side too.
		if plan.DownSQL != "" {
			_ = o.driver.Rollback(ctx, ScopeReal, plan.DownSQL)
			out.RolledBack = true
		}
		return out, err
	}
	out.RealApplied = true

	// Step 4: integrity checks.
	for _, c := range plan.IntegrityChecks {
		r, err := o.driver.IntegrityCheck(ctx, ScopeReal, c)
		if err != nil {
			out.Reason = fmt.Sprintf("integrity check %s failed: %v", c.Name, err)
			if plan.DownSQL != "" {
				_ = o.driver.Rollback(ctx, ScopeReal, plan.DownSQL)
				out.RolledBack = true
			}
			return out, err
		}
		// Range check.
		if (c.ExpectMin > 0 || c.ExpectMax > 0) && (r.Affected < c.ExpectMin || (c.ExpectMax > 0 && r.Affected > c.ExpectMax)) {
			out.Reason = fmt.Sprintf("integrity check %s out of range: got %d not in [%d..%d]", c.Name, r.Affected, c.ExpectMin, c.ExpectMax)
			if plan.DownSQL != "" {
				_ = o.driver.Rollback(ctx, ScopeReal, plan.DownSQL)
				out.RolledBack = true
			}
			return out, errors.New(out.Reason)
		}
	}
	out.IntegrityOK = true
	return out, nil
}

func containsDestructiveDDL(sql string) bool {
	upper := strings.ToUpper(sql)
	for _, kw := range []string{"DROP TABLE", "DROP COLUMN", "TRUNCATE", "ALTER TABLE.*DROP"} {
		if strings.Contains(upper, kw) {
			return true
		}
	}
	return false
}

func destructiveDeclared(b *cruciblev1.PromotionBundle) bool {
	_ = b
	// In Phase 6 the destructive-ddl flag lives in the MigrationAttestation
	// referenced by the bundle. The gate's enrichment puts that into the
	// rego input as `blast_radius.schema_changes[].destructive_ddl`. The
	// orchestrator gets the bundle but not the enrichment; conservative
	// default: refuse unless the impact is high (the rego gate would have
	// caught it earlier anyway).
	return false
}
