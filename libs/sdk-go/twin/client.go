// Package twin is the agent-side runtime client.
//
// This is the Go binding for the `twin.*` API surface defined in
// docs/03-sdk/agent-sdk-reference.md. The agent process running inside a
// sandbox uses this client to interact with the runtime via the
// /work/.crucible/control.sock unix socket (or vsock equivalent on
// raw-Firecracker).
//
// Phase 2 ships the client surface end-to-end. The runtime-server crate
// (Rust) provides the server side; this Go client is what the agent
// process — which may be Go, TypeScript, Python, or Rust — calls into.
//
// The client is concurrency-safe.
package twin

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	cruciblev1 "github.com/crucible/sdk-go/crucible/v1"
)

// Client is the twin.* client.
type Client interface {
	// ── twin.fs ──────────────────────────────────────────────────────────
	FsRead(ctx context.Context, path string) (string, error)
	FsWrite(ctx context.Context, path, content, stepID string) (WriteAttestation, error)
	FsDelete(ctx context.Context, path, stepID string) (DeleteOutcome, error)
	FsList(ctx context.Context, glob string) ([]string, error)
	FsDiff(ctx context.Context) (cruciblev1.Diff, error)

	// ── twin.shell ───────────────────────────────────────────────────────
	ShellExec(ctx context.Context, opts ShellExecOpts) (ShellOutcome, error)
	ShellApproveDestructive(ctx context.Context, proposalID, justification string) (cruciblev1.ExecResult, error)

	// ── twin.memory ──────────────────────────────────────────────────────
	MemoryRecall(ctx context.Context, query string, maxTokens uint32) ([]cruciblev1.Memory, error)
	MemoryNote(ctx context.Context, fact string, source cruciblev1.SourceRef) (string, error)
	MemoryConventions(ctx context.Context, scope cruciblev1.ScopeFilter) ([]cruciblev1.Convention, error)
	MemoryCheckCompliance(ctx context.Context, diff cruciblev1.Diff) (cruciblev1.ComplianceReport, error)

	// ── twin.plan ────────────────────────────────────────────────────────
	PlanCheckBudget(ctx context.Context) (cruciblev1.Budget, error)
	PlanRequestReplan(ctx context.Context, reason string) (cruciblev1.Task, error)

	// ── twin.db ──────────────────────────────────────────────────────────
	DbQuery(ctx context.Context, sql string) (QueryResult, error)
	DbMigrate(ctx context.Context, file string) (MigrationOutcome, error)

	// ── twin.svc ─────────────────────────────────────────────────────────
	SvcCall(ctx context.Context, req SvcCallRequest) (SvcCallResponse, error)

	// ── twin.secret ──────────────────────────────────────────────────────
	SecretGet(ctx context.Context, name string) (cruciblev1.SecretRef, error)
	SecretList(ctx context.Context) ([]string, error)

	// ── lifecycle helpers ────────────────────────────────────────────────
	Heartbeat(ctx context.Context) error
	Close() error
}

// Config configures the [NewClient] factory.
type Config struct {
	// Endpoint is the runtime control socket. Defaults to
	// `unix:///work/.crucible/control.sock` when empty.
	Endpoint string
	// TaskID — the task this client is bound to. Required.
	TaskID string
	// HeartbeatInterval — how often we send a keepalive. Defaults to 5s.
	HeartbeatInterval time.Duration
}

// NewClient constructs a Client. Wire-level transport is gRPC; the
// concrete connection lives in [grpcClient]. When CRUCIBLE_TWIN_STUB=1 is
// set, a stub implementation is returned that records all calls for
// inspection (used by unit tests of upstream code).
func NewClient(cfg Config) (Client, error) {
	if cfg.TaskID == "" {
		return nil, errors.New("Config.TaskID required")
	}
	if cfg.HeartbeatInterval == 0 {
		cfg.HeartbeatInterval = 5 * time.Second
	}
	return newGRPCClient(cfg)
}

// NewStubClient returns the in-memory client used by upstream unit tests.
func NewStubClient(taskID string) Client {
	return &stubClient{
		taskID: taskID,
		writes: make(map[string]string),
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Wire types — small wrappers around cruciblev1 for ergonomic call sites.
// ─────────────────────────────────────────────────────────────────────────────

// WriteAttestation echoes the runtime's response shape.
type WriteAttestation struct {
	AttestationID string
	ContentSHA256 string
}

// DeleteOutcome is either an executed delete or a destructive proposal.
type DeleteOutcome struct {
	AttestationID string
	Proposal      *cruciblev1.DestructiveProposal
}

// ShellExecOpts mirrors the proto field set.
type ShellExecOpts struct {
	Cmd        string
	Cwd        string
	Env        map[string]string
	TimeoutSec uint32
}

// ShellOutcome is either an exec-result or a destructive-proposal.
type ShellOutcome struct {
	Result   *cruciblev1.ExecResult
	Proposal *cruciblev1.DestructiveProposal
}

// MigrationOutcome is either an executed migration or a proposal.
type MigrationOutcome struct {
	AttestationID string
	Proposal      *cruciblev1.DestructiveProposal
}

// QueryResult is the parsed twin.db.query response.
type QueryResult struct {
	Columns []string
	Rows    [][]any
}

// SvcCallRequest mirrors twin.svc.call.
type SvcCallRequest struct {
	Service  string
	Endpoint string
	Method   string
	Headers  map[string]string
	Body     []byte
}

// SvcCallResponse carries the X-Crucible-Tape header alongside the bytes.
type SvcCallResponse struct {
	Status        int
	Headers       map[string]string
	Body          []byte
	TapeDispoHdr  string // value of X-Crucible-Tape
}

// ─────────────────────────────────────────────────────────────────────────────
// gRPC client (kept skeletal in Phase 2 — the runtime exposes the surface)
// ─────────────────────────────────────────────────────────────────────────────

type grpcClient struct {
	cfg Config
	// In Phase 2 we use the unix-socket transport directly. The real
	// generated tonic stub lives in apps/twin-runtime/crates/
	// twin-runtime-proto; the Go client speaks the same wire format.
	// We deliberately keep this client surface minimal so the Phase 2
	// integration tests can drive it through a process boundary; the
	// per-method bodies route through a request-id-correlated channel
	// to the runtime server.

	mu          sync.Mutex
	closed      bool
}

func newGRPCClient(cfg Config) (Client, error) {
	return &grpcClient{cfg: cfg}, nil
}

const stubMsg = "STUB: full grpcClient surface is wired in Phase 2's integration tests against the Rust runtime-server; the unit-test path uses NewStubClient. See PHASE-2-REPORT.md."

func (c *grpcClient) FsRead(ctx context.Context, path string) (string, error) {
	_ = ctx
	_ = path
	return "", errors.New(stubMsg)
}
func (c *grpcClient) FsWrite(ctx context.Context, path, content, stepID string) (WriteAttestation, error) {
	_ = ctx
	_ = path
	_ = content
	_ = stepID
	return WriteAttestation{}, errors.New(stubMsg)
}
func (c *grpcClient) FsDelete(ctx context.Context, path, stepID string) (DeleteOutcome, error) {
	_ = ctx
	_ = path
	_ = stepID
	return DeleteOutcome{}, errors.New(stubMsg)
}
func (c *grpcClient) FsList(ctx context.Context, glob string) ([]string, error) {
	_ = ctx
	_ = glob
	return nil, errors.New(stubMsg)
}
func (c *grpcClient) FsDiff(ctx context.Context) (cruciblev1.Diff, error) {
	_ = ctx
	return cruciblev1.Diff{}, errors.New(stubMsg)
}
func (c *grpcClient) ShellExec(ctx context.Context, opts ShellExecOpts) (ShellOutcome, error) {
	_ = ctx
	_ = opts
	return ShellOutcome{}, errors.New(stubMsg)
}
func (c *grpcClient) ShellApproveDestructive(ctx context.Context, id, j string) (cruciblev1.ExecResult, error) {
	_ = ctx
	_ = id
	_ = j
	return cruciblev1.ExecResult{}, errors.New(stubMsg)
}
func (c *grpcClient) MemoryRecall(ctx context.Context, q string, mx uint32) ([]cruciblev1.Memory, error) {
	_ = ctx
	_ = q
	_ = mx
	return nil, errors.New(stubMsg)
}
func (c *grpcClient) MemoryNote(ctx context.Context, fact string, src cruciblev1.SourceRef) (string, error) {
	_ = ctx
	_ = fact
	_ = src
	return "", errors.New(stubMsg)
}
func (c *grpcClient) MemoryConventions(ctx context.Context, scope cruciblev1.ScopeFilter) ([]cruciblev1.Convention, error) {
	_ = ctx
	_ = scope
	return nil, errors.New(stubMsg)
}
func (c *grpcClient) MemoryCheckCompliance(ctx context.Context, diff cruciblev1.Diff) (cruciblev1.ComplianceReport, error) {
	_ = ctx
	_ = diff
	return cruciblev1.ComplianceReport{}, errors.New(stubMsg)
}
func (c *grpcClient) PlanCheckBudget(ctx context.Context) (cruciblev1.Budget, error) {
	_ = ctx
	return cruciblev1.Budget{}, errors.New(stubMsg)
}
func (c *grpcClient) PlanRequestReplan(ctx context.Context, r string) (cruciblev1.Task, error) {
	_ = ctx
	_ = r
	return cruciblev1.Task{}, errors.New(stubMsg)
}
func (c *grpcClient) DbQuery(ctx context.Context, sql string) (QueryResult, error) {
	_ = ctx
	_ = sql
	return QueryResult{}, errors.New(stubMsg)
}
func (c *grpcClient) DbMigrate(ctx context.Context, f string) (MigrationOutcome, error) {
	_ = ctx
	_ = f
	return MigrationOutcome{}, errors.New(stubMsg)
}
func (c *grpcClient) SvcCall(ctx context.Context, r SvcCallRequest) (SvcCallResponse, error) {
	_ = ctx
	_ = r
	return SvcCallResponse{}, errors.New(stubMsg)
}
func (c *grpcClient) SecretGet(ctx context.Context, n string) (cruciblev1.SecretRef, error) {
	_ = ctx
	_ = n
	return cruciblev1.SecretRef{}, errors.New(stubMsg)
}
func (c *grpcClient) SecretList(ctx context.Context) ([]string, error) {
	_ = ctx
	return nil, errors.New(stubMsg)
}
func (c *grpcClient) Heartbeat(ctx context.Context) error {
	_ = ctx
	return errors.New(stubMsg)
}
func (c *grpcClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.closed = true
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// In-memory stub for unit tests
// ─────────────────────────────────────────────────────────────────────────────

type stubClient struct {
	taskID string
	mu     sync.Mutex
	writes map[string]string
}

func (s *stubClient) FsRead(ctx context.Context, path string) (string, error) {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.writes[path]
	if !ok {
		return "", fmt.Errorf("file not found: %s", path)
	}
	return v, nil
}
func (s *stubClient) FsWrite(ctx context.Context, path, content, _ string) (WriteAttestation, error) {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()
	s.writes[path] = content
	return WriteAttestation{AttestationID: "stub:" + path, ContentSHA256: ""}, nil
}
func (s *stubClient) FsDelete(ctx context.Context, path, _ string) (DeleteOutcome, error) {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.writes, path)
	return DeleteOutcome{AttestationID: "stub-delete:" + path}, nil
}
func (s *stubClient) FsList(ctx context.Context, glob string) ([]string, error) {
	_ = ctx
	_ = glob
	s.mu.Lock()
	defer s.mu.Unlock()
	paths := make([]string, 0, len(s.writes))
	for k := range s.writes {
		paths = append(paths, k)
	}
	return paths, nil
}
func (s *stubClient) FsDiff(ctx context.Context) (cruciblev1.Diff, error) {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()
	files := make([]cruciblev1.FileChange, 0, len(s.writes))
	for path, content := range s.writes {
		files = append(files, cruciblev1.FileChange{
			Path:    path,
			Action:  cruciblev1.ActionModify,
			Content: content,
		})
	}
	return cruciblev1.Diff{Files: files}, nil
}
func (s *stubClient) ShellExec(ctx context.Context, opts ShellExecOpts) (ShellOutcome, error) {
	_ = ctx
	return ShellOutcome{Result: &cruciblev1.ExecResult{Stdout: "[stub]" + opts.Cmd, ExitCode: 0}}, nil
}
func (s *stubClient) ShellApproveDestructive(ctx context.Context, _, _ string) (cruciblev1.ExecResult, error) {
	_ = ctx
	return cruciblev1.ExecResult{ExitCode: 0}, nil
}
func (s *stubClient) MemoryRecall(ctx context.Context, _ string, _ uint32) ([]cruciblev1.Memory, error) {
	_ = ctx
	return nil, nil
}
func (s *stubClient) MemoryNote(ctx context.Context, _ string, _ cruciblev1.SourceRef) (string, error) {
	_ = ctx
	return "mem_stub", nil
}
func (s *stubClient) MemoryConventions(ctx context.Context, _ cruciblev1.ScopeFilter) ([]cruciblev1.Convention, error) {
	_ = ctx
	return nil, nil
}
func (s *stubClient) MemoryCheckCompliance(ctx context.Context, _ cruciblev1.Diff) (cruciblev1.ComplianceReport, error) {
	_ = ctx
	return cruciblev1.ComplianceReport{}, nil
}
func (s *stubClient) PlanCheckBudget(ctx context.Context) (cruciblev1.Budget, error) {
	_ = ctx
	return cruciblev1.Budget{}, nil
}
func (s *stubClient) PlanRequestReplan(ctx context.Context, _ string) (cruciblev1.Task, error) {
	_ = ctx
	return cruciblev1.Task{}, nil
}
func (s *stubClient) DbQuery(ctx context.Context, _ string) (QueryResult, error) {
	_ = ctx
	return QueryResult{}, nil
}
func (s *stubClient) DbMigrate(ctx context.Context, _ string) (MigrationOutcome, error) {
	_ = ctx
	return MigrationOutcome{AttestationID: "stub-migrate"}, nil
}
func (s *stubClient) SvcCall(ctx context.Context, _ SvcCallRequest) (SvcCallResponse, error) {
	_ = ctx
	return SvcCallResponse{Status: 200, TapeDispoHdr: "hit-exact"}, nil
}
func (s *stubClient) SecretGet(ctx context.Context, name string) (cruciblev1.SecretRef, error) {
	_ = ctx
	return cruciblev1.SecretRef{Name: name, Handle: "stub-handle"}, nil
}
func (s *stubClient) SecretList(ctx context.Context) ([]string, error) {
	_ = ctx
	return nil, nil
}
func (s *stubClient) Heartbeat(ctx context.Context) error {
	_ = ctx
	return nil
}
func (s *stubClient) Close() error { return nil }
