// Package dbdriver is the Crucible Twin Runtime's database-twin layer.
//
// Phase 2 ships a first-class Neon Postgres driver and typed STUB: errors
// for the other engines (MySQL, SQLite, Mongo). The Driver interface lets
// downstream code remain engine-agnostic while we incrementally fill in
// implementations through Phase 3.
//
// Per the May 2026 currency check, the Neon driver MUST handle the async
// nature of POST /projects/{id}/branches: the response carries
// `current_state: "init"` with a pending operations[] array; callers must
// poll the operations endpoint (or the dedicated /connection_uri endpoint)
// before treating the branch as usable.
//
// All driver methods are env-gated: when CRUCIBLE_NEON_API_KEY is unset,
// New() returns a stub driver that produces typed Phase-2 errors for any
// call. Mirrors the Phase 1 control-plane pattern.
package dbdriver

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// Engine identifies the database engine the driver targets.
type Engine string

// Supported engines. Engines not listed in this enum return a STUB error.
const (
	EnginePostgresNeon  Engine = "postgres-neon"
	EnginePostgresXata  Engine = "postgres-xata"  // Phase 4+ — not in Phase 3
	EnginePostgresDBLab Engine = "postgres-dblab" // Phase 4+ — not in Phase 3
	// Phase 3 engines — all implemented in their own files in this package.
	EngineMySQL      Engine = "mysql-planetscale"
	EngineSQLite     Engine = "sqlite-turso"
	EngineMongo      Engine = "mongo-atlas"
	EngineRedis      Engine = "redis-inproc"
	EngineClickHouse Engine = "clickhouse-clone"
	EngineS3         Engine = "s3-minio"
)

// Driver is the engine-agnostic interface every implementation honours.
type Driver interface {
	// Engine identifies the implementation.
	Engine() Engine

	// Capabilities probe — driven by the Postgres-branching research
	// finding: callers route based on what the underlying engine can do
	// rather than assuming a uniform feature set.
	Capabilities() Capabilities

	// CreateBranch mints a new branch and waits for it to be usable.
	// The returned Branch.ConnectionURI is ready for connect.
	//
	// On Neon, this calls POST /projects/{project_id}/branches then polls
	// the operations endpoint (or GET .../connection_uri) until the
	// create_branch op reports `finished`. The poll deadline is
	// CreateBranchOpts.Timeout (default 10s, see ADR-009 wall-clock cap
	// for context).
	CreateBranch(ctx context.Context, spec BranchSpec, opts CreateBranchOpts) (Branch, error)

	// DeleteBranch is asynchronous on Neon; this method blocks until the
	// suspend_compute + delete_timeline ops are scheduled. Idempotent:
	// deleting a missing branch returns nil.
	DeleteBranch(ctx context.Context, branchID string) error

	// SchemaDiff returns the DDL difference between two branches. On Neon
	// this calls the first-party compare_schema endpoint introduced in
	// 2026 (we deliberately dropped the pg_dump fallback in Phase 2 per
	// the currency-check findings).
	SchemaDiff(ctx context.Context, baseBranchID, targetBranchID, dbName string) (SchemaDiffResult, error)

	// ListBranches returns the runtime's view of branches under a project.
	// Used by the GC reconciler.
	ListBranches(ctx context.Context, projectID string) ([]Branch, error)
}

// Capabilities reports per-engine feature support.
type Capabilities struct {
	// InstantBranch is true when branch-create completes in seconds rather
	// than minutes.
	InstantBranch bool
	// ScaleToZero is true when idle branches suspend automatically (Neon).
	ScaleToZero bool
	// FirstPartySchemaDiff is true when the engine exposes a native
	// branch-vs-branch DDL diff endpoint.
	FirstPartySchemaDiff bool
	// MaxConcurrentBranches is the engine's hard cap per source. Zero
	// means "unbounded as far as we know."
	MaxConcurrentBranches int
	// PerTenantProjectRequired is true when tenant isolation REQUIRES one
	// project per tenant (Neon: branch-prefix RBAC does not exist).
	PerTenantProjectRequired bool
}

// BranchSpec describes the branch we want to create.
type BranchSpec struct {
	ProjectID       string
	BaseBranchID    string
	BaseBranchName  string // used if BaseBranchID is empty; the driver resolves
	ParentLSN       string // optional; pin for deterministic clones
	ParentTimestamp time.Time
	Name            string // optional; auto-generated if empty
	RoleName        string // optional; will create a role if non-empty
	Protected       bool
	// Tags is metadata Neon stores in the branch's `metadata` field.
	Tags map[string]string
}

// CreateBranchOpts tunes the create+poll dance.
type CreateBranchOpts struct {
	// Timeout for the full create+ready cycle. Default 10s.
	Timeout time.Duration
	// PollInterval between operations checks. Default 250ms.
	PollInterval time.Duration
	// CreateRole controls whether a fresh role is provisioned (Neon-only).
	// The role's one-time password is returned in Branch.RolePassword.
	CreateRole bool
}

// Branch describes a created branch ready for connect.
type Branch struct {
	ID            string
	ProjectID     string
	Name          string
	Host          string
	ConnectionURI string
	State         string
	CreatedAt     time.Time
	RolePassword  string // populated only if CreateRole=true
	Metadata      map[string]string
}

// SchemaDiffResult is the parsed output of a branch-vs-branch DDL compare.
type SchemaDiffResult struct {
	// DDL is the raw migration script that would transform Target → Base.
	DDL string
	// AddedTables, DroppedTables, AlteredTables enumerated for the verifier.
	AddedTables   []string
	DroppedTables []string
	AlteredTables []string
	// HasDestructive is true when any DROP or DELETE-without-WHERE appears.
	HasDestructive bool
}

// New picks the right driver implementation for the engine, reading
// configuration from environment variables prefixed with CRUCIBLE_.
//
// Missing required env vars yield a typed stub driver that returns the
// corresponding error from every method.
func New(engine Engine) Driver {
	switch engine {
	case EnginePostgresNeon:
		return NewNeonDriver()
	case EngineMySQL:
		return NewPlanetScaleDriver()
	case EngineSQLite:
		return NewTursoDriver()
	case EngineMongo:
		return NewMongoDriver()
	case EngineRedis:
		return NewRedisDriver()
	case EngineClickHouse:
		return NewClickHouseDriver()
	case EngineS3:
		return NewS3Driver()
	case EnginePostgresXata, EnginePostgresDBLab:
		return newStubDriver(engine, fmt.Sprintf(
			"engine %q is deferred to Phase 4+.",
			engine,
		))
	default:
		return newStubDriver(engine, fmt.Sprintf("unknown engine %q", engine))
	}
}

// DetectEngine returns the right Engine for a connection-style string. Used
// by the bridge so the control plane can detect engines from the
// `repo.detected_db_engines` field on the task manifest without callers
// having to import the engine string constants.
func DetectEngine(hint string) Engine {
	h := strings.ToLower(strings.TrimSpace(hint))
	switch {
	case strings.HasPrefix(h, "postgres") || strings.HasPrefix(h, "pg"):
		return EnginePostgresNeon
	case strings.HasPrefix(h, "mysql") || strings.HasPrefix(h, "vitess"):
		return EngineMySQL
	case strings.HasPrefix(h, "sqlite") || strings.HasPrefix(h, "libsql") || strings.HasPrefix(h, "turso"):
		return EngineSQLite
	case strings.HasPrefix(h, "mongo"):
		return EngineMongo
	case strings.HasPrefix(h, "redis") || strings.HasPrefix(h, "valkey") || strings.HasPrefix(h, "kv"):
		return EngineRedis
	case strings.HasPrefix(h, "clickhouse") || strings.HasPrefix(h, "ch"):
		return EngineClickHouse
	case strings.HasPrefix(h, "s3") || strings.HasPrefix(h, "minio") || strings.HasPrefix(h, "gcs"):
		return EngineS3
	}
	return Engine(h)
}

type stubDriver struct {
	engine Engine
	msg    string
}

func newStubDriver(engine Engine, msg string) Driver {
	return &stubDriver{engine: engine, msg: msg}
}

func (d *stubDriver) Engine() Engine { return d.engine }

func (d *stubDriver) Capabilities() Capabilities { return Capabilities{} }

func (d *stubDriver) CreateBranch(ctx context.Context, spec BranchSpec, opts CreateBranchOpts) (Branch, error) {
	_ = ctx
	_ = spec
	_ = opts
	return Branch{}, &StubError{Engine: d.engine, Msg: d.msg}
}

func (d *stubDriver) DeleteBranch(ctx context.Context, branchID string) error {
	_ = ctx
	_ = branchID
	return &StubError{Engine: d.engine, Msg: d.msg}
}

func (d *stubDriver) SchemaDiff(ctx context.Context, base, target, db string) (SchemaDiffResult, error) {
	_ = ctx
	_ = base
	_ = target
	_ = db
	return SchemaDiffResult{}, &StubError{Engine: d.engine, Msg: d.msg}
}

func (d *stubDriver) ListBranches(ctx context.Context, projectID string) ([]Branch, error) {
	_ = ctx
	_ = projectID
	return nil, &StubError{Engine: d.engine, Msg: d.msg}
}

// StubError is the typed Phase-2 stub error. Callers can errors.As() to
// detect a deferred-implementation case.
type StubError struct {
	Engine Engine
	Msg    string
}

func (e *StubError) Error() string {
	return fmt.Sprintf("STUB: %s [%s]", e.Msg, e.Engine)
}

// IsStub returns true if err is a *StubError.
func IsStub(err error) bool {
	if err == nil {
		return false
	}
	_, ok := err.(*StubError)
	return ok
}
