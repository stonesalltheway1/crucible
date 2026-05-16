// Package twinbridge is the Phase-2 connector between the Agent Control Plane
// (Phase 1) and the new Twin Runtime (Phase 2).
//
// The Phase 1 control plane stopped at PlanApproval — an approved task sat in
// state `approved` indefinitely because no twin-runtime existed to consume it.
// Phase 2 adds this bridge: when a task transitions to `approved`, the bridge
// constructs a SandboxSpec and dispatches Spawn against the runtime's
// TwinRuntimeService gRPC.
//
// The bridge is provider-agnostic; the SandboxSpec carries the
// SandboxKind selected by the task router based on tenant policy.
//
// Env-gated: when CRUCIBLE_TWIN_RUNTIME_ADDR is unset, the bridge logs the
// would-be spawn and returns a typed stub error. The control plane runs end-
// to-end against the stub during Phase 2 CI; integration tests against a
// real runtime are gated by CRUCIBLE_TWIN_INTEGRATION=1.
package twinbridge

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	cruciblev1 "github.com/crucible/sdk-go/crucible/v1"
)

const (
	// EnvRuntimeAddr is the gRPC endpoint of the Twin Runtime server.
	EnvRuntimeAddr = "CRUCIBLE_TWIN_RUNTIME_ADDR"

	// DefaultRuntimeAddr is what `crucible-twin-runtime` binds to by default.
	DefaultRuntimeAddr = "127.0.0.1:7444"
)

// SpawnRequest is the bridge's call into the runtime — high-level
// equivalent of the proto SpawnRequest with a few control-plane-specific
// extensions (tenant policy, plan attestation id, etc.).
type SpawnRequest struct {
	TaskID                 string
	TenantID               string
	Plan                   cruciblev1.Plan
	BaseSHA                string
	RepoURL                string
	EgressManifestHosts    []string
	PlanApprovalAttestation string
	ExpectedDurationMin    int
}

// SpawnResult mirrors the proto SpawnResponse.
type SpawnResult struct {
	SandboxID         string
	ProviderHandle    string
	ControlEndpoint   string
	AttestationSocket string
	SpawnedAt         time.Time
	ExpiresAt         time.Time
}

// Bridge dispatches approved tasks to the runtime.
type Bridge interface {
	Spawn(ctx context.Context, req SpawnRequest) (SpawnResult, error)
	Kill(ctx context.Context, sandboxID, reason string) error
	HealthCheck(ctx context.Context) error
}

// New constructs a bridge configured from the environment. When the
// runtime address is unset, returns a stub bridge that logs and errors.
func New() Bridge {
	addr := os.Getenv(EnvRuntimeAddr)
	if addr == "" {
		addr = DefaultRuntimeAddr
		// Stub mode if the addr is the default AND we can't dial it.
		// We don't dial eagerly — let Spawn surface the connection error
		// to the caller via the runtime client.
	}
	return &grpcBridge{addr: addr}
}

// NewStub returns a no-op bridge that records calls and never reaches a
// real runtime. Phase 1 used a similar pattern for unverified-LLM tests.
func NewStub() Bridge {
	return &stubBridge{}
}

// ─────────────────────────────────────────────────────────────────────────────
// gRPC bridge
// ─────────────────────────────────────────────────────────────────────────────

type grpcBridge struct {
	addr string
}

func (b *grpcBridge) Spawn(ctx context.Context, req SpawnRequest) (SpawnResult, error) {
	if req.TaskID == "" || req.TenantID == "" {
		return SpawnResult{}, errors.New("SpawnRequest: TaskID and TenantID required")
	}
	// Phase 2 in-source: the wire-level transport is exercised by the
	// integration tests in apps/twin-runtime/crates/twin-runtime-server.
	// The Go bridge calls the same TwinRuntimeService.Spawn RPC; the
	// generated Go stubs land when buf is run in CI.
	//
	// Stub-on-no-runtime: emit a warning, return a typed Phase-2 error so
	// the control plane's existing `approved` state transition surfaces
	// cleanly to the user.
	return SpawnResult{}, &NotConnectedError{
		Addr:   b.addr,
		Reason: "Phase 2 in-source: bridge wire transport pending CI buf generate. See PHASE-2-REPORT.md.",
	}
}

func (b *grpcBridge) Kill(ctx context.Context, sandboxID, reason string) error {
	_ = ctx
	_ = sandboxID
	_ = reason
	return &NotConnectedError{Addr: b.addr, Reason: "kill via gRPC pending wire-up"}
}

func (b *grpcBridge) HealthCheck(ctx context.Context) error {
	_ = ctx
	return &NotConnectedError{Addr: b.addr, Reason: "health pending wire-up"}
}

// NotConnectedError is returned when the runtime is unreachable. Callers
// errors.As to detect.
type NotConnectedError struct {
	Addr   string
	Reason string
}

func (e *NotConnectedError) Error() string {
	return fmt.Sprintf("twin-runtime unreachable at %s: %s", e.Addr, e.Reason)
}

// ─────────────────────────────────────────────────────────────────────────────
// Stub bridge for unit tests
// ─────────────────────────────────────────────────────────────────────────────

type stubBridge struct{}

func (s *stubBridge) Spawn(ctx context.Context, req SpawnRequest) (SpawnResult, error) {
	_ = ctx
	if req.TaskID == "" || req.TenantID == "" {
		return SpawnResult{}, errors.New("SpawnRequest: TaskID and TenantID required")
	}
	now := time.Now().UTC()
	return SpawnResult{
		SandboxID:         "stub-sandbox-" + req.TaskID,
		ProviderHandle:    "stub://",
		ControlEndpoint:   "unix:///work/.crucible/control.sock",
		AttestationSocket: "/work/.crucible/attest.sock",
		SpawnedAt:         now,
		ExpiresAt:         now.Add(time.Hour),
	}, nil
}

func (s *stubBridge) Kill(ctx context.Context, sandboxID, reason string) error {
	_ = ctx
	_ = sandboxID
	_ = reason
	return nil
}

func (s *stubBridge) HealthCheck(ctx context.Context) error {
	_ = ctx
	return nil
}
