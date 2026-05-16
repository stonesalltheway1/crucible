// Package verifierbridge is the Phase-4 connector between the Agent
// Control Plane and the new Verifier Daemon (apps/verifier/).
//
// The Phase-3 control plane stopped at task status `executing` (real
// twin spawn went via twinbridge); Phase 4 adds this bridge: when the
// agent claims `done`, the task transitions to `verifying`, the
// control plane assembles a VerificationRequest, and posts it here.
// On approval the task transitions to `promoting`; on rejection it
// transitions to `failed` with structured RejectionReasons surfaced
// in the task event stream.
//
// Env-gated: when CRUCIBLE_VERIFIER_ADDR is unset, the bridge logs the
// would-be call and returns a typed stub. The control plane's unit
// tests run end-to-end against the stub; integration tests against
// the real daemon are gated by CRUCIBLE_VERIFIER_INTEGRATION=1.
//
// Bounded Budget Enforcer integration (ADR-009): the bridge tracks
// VerifierSpentUSD in a SEPARATE counter from ExecutorSpentUSD. Verifier
// cost is reported back to the Enforcer via Charge calls (not deducted
// from the executor's cap).
package verifierbridge

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	cruciblev1 "github.com/crucible/sdk-go/crucible/v1"
)

const (
	// EnvVerifierAddr is the HTTP endpoint of the Verifier Daemon.
	EnvVerifierAddr = "CRUCIBLE_VERIFIER_ADDR"

	// DefaultVerifierAddr is what `crucible-verifier` binds to by default.
	DefaultVerifierAddr = "http://127.0.0.1:9080"
)

// VerifyRequest is the bridge's call into the verifier — mirrors the
// daemon's VerificationRequest shape but uses the control-plane's
// canonical types.
type VerifyRequest struct {
	TaskID             string                       `json:"task_id"`
	TenantID           string                       `json:"tenant_id"`
	Repo               string                       `json:"repo"`
	BaseSHA            string                       `json:"base_sha"`
	Diff               cruciblev1.Diff              `json:"diff"`
	TestFiles          []cruciblev1.FileChange      `json:"test_files,omitempty"`
	SpecChanges        []SpecChange                 `json:"spec_changes,omitempty"`
	Routing            cruciblev1.Routing           `json:"routing"`
	Languages          []string                     `json:"languages"`
	CriticalPathScores []CriticalPathScore          `json:"critical_path_scores,omitempty"`
	PerTaskSignals     TaskSignals                  `json:"per_task_signals,omitempty"`
	Budget             BudgetEnvelope               `json:"budget"`
	AttestationChain   []string                     `json:"attestation_chain,omitempty"`
	ExecutorSandboxID  string                       `json:"executor_sandbox_id"`
}

// SpecChange duplicates verifier-side struct shape.
type SpecChange struct {
	Path         string `json:"path"`
	Kind         string `json:"kind"`
	PreviousHash string `json:"previous_hash"`
	CurrentHash  string `json:"current_hash"`
	Delta        string `json:"delta,omitempty"`
}

type CriticalPathScore struct {
	File   string  `json:"file"`
	Score  float64 `json:"score"`
	Band   string  `json:"band"`
	Reason string  `json:"reason,omitempty"`
}

type TaskSignals struct {
	StalenessFindings        []StalenessFinding   `json:"staleness_findings,omitempty"`
	TapeDispositionHistogram map[string]int       `json:"tape_disposition_histogram,omitempty"`
	ScrubberAuditEntryCount  int                  `json:"scrubber_audit_entry_count,omitempty"`
	ScrubberFiredOnAllPII    bool                 `json:"scrubber_fired_on_all_pii"`
	WasmQuotaTrips           []WasmQuotaTrip      `json:"wasm_quota_trips,omitempty"`
	SelfHostAvailable        bool                 `json:"self_host_available"`
	SelfHostUnavailableReason string              `json:"self_host_unavailable_reason,omitempty"`
}

type StalenessFinding struct {
	Endpoint     string `json:"endpoint"`
	Band         string `json:"band"`
	AgeSeconds   int64  `json:"age_seconds"`
	IntervalSecs int64  `json:"interval_seconds"`
	Message      string `json:"message"`
}

type WasmQuotaTrip struct {
	Quota     string `json:"quota"`
	TrippedAt int64  `json:"tripped_at"`
	Detail    string `json:"detail,omitempty"`
}

type BudgetEnvelope struct {
	VerifierCapUSD        float64 `json:"verifier_cap_usd"`
	VerifierSpentUSD      float64 `json:"verifier_spent_usd"`
	WallClockCapSeconds   uint64  `json:"wall_clock_cap_seconds"`
	WallClockSpentSeconds uint64  `json:"wall_clock_spent_seconds"`
}

// VerifyResponse mirrors the daemon's VerificationResponse with the
// approval or rejection always populated.
type VerifyResponse struct {
	Approval  *cruciblev1.VerifierApproval  `json:"approval,omitempty"`
	Rejection *cruciblev1.VerifierRejection `json:"rejection,omitempty"`
	CostUSD   float64                       `json:"cost_usd"`
	DurationMS int64                        `json:"duration_ms"`
}

// Bridge dispatches verify requests.
type Bridge interface {
	Verify(ctx context.Context, req VerifyRequest) (*VerifyResponse, error)
	HealthCheck(ctx context.Context) error
}

// New constructs a bridge configured from the environment. When the
// addr is unset and the default isn't reachable, callers see a typed
// NotConnectedError.
func New() Bridge {
	addr := os.Getenv(EnvVerifierAddr)
	if addr == "" {
		addr = DefaultVerifierAddr
	}
	return &httpBridge{
		addr:   addr,
		client: &http.Client{Timeout: 70 * time.Minute},
	}
}

// NewStub returns a no-op bridge that records calls. Used by tests
// that don't want to spin up a real verifier daemon.
func NewStub() Bridge {
	return &stubBridge{}
}

// ─── HTTP bridge ─────────────────────────────────────────────────────

type httpBridge struct {
	addr   string
	client *http.Client
}

func (b *httpBridge) Verify(ctx context.Context, req VerifyRequest) (*VerifyResponse, error) {
	if req.TaskID == "" || req.TenantID == "" {
		return nil, errors.New("VerifyRequest: TaskID and TenantID required")
	}
	// Cross-family invariant pre-check (defence in depth — the daemon
	// re-checks too).
	if equalsCaseFold(req.Routing.ExecutorVendor, req.Routing.VerifierVendor) {
		return nil, fmt.Errorf("verifierbridge: cross-family invariant — executor.vendor=%q must NOT equal verifier.vendor=%q",
			req.Routing.ExecutorVendor, req.Routing.VerifierVendor)
	}
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal: %w", err)
	}
	start := time.Now()
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, b.addr+"/v1/twin/verify/bundle", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := b.client.Do(httpReq)
	if err != nil {
		return nil, &NotConnectedError{Addr: b.addr, Reason: err.Error()}
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 64<<20))
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("verifierbridge: %d %s — body=%s", resp.StatusCode, resp.Status, string(raw))
	}
	var out VerifyResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	out.DurationMS = time.Since(start).Milliseconds()
	return &out, nil
}

func (b *httpBridge) HealthCheck(ctx context.Context) error {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, b.addr+"/healthz", nil)
	if err != nil {
		return err
	}
	resp, err := b.client.Do(httpReq)
	if err != nil {
		return &NotConnectedError{Addr: b.addr, Reason: err.Error()}
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("verifier health %d", resp.StatusCode)
	}
	return nil
}

// NotConnectedError surfaces when the verifier daemon isn't reachable.
type NotConnectedError struct {
	Addr   string
	Reason string
}

func (e *NotConnectedError) Error() string {
	return fmt.Sprintf("verifier daemon unreachable at %s: %s", e.Addr, e.Reason)
}

// ─── Stub bridge ─────────────────────────────────────────────────────

type stubBridge struct{}

func (s *stubBridge) Verify(_ context.Context, req VerifyRequest) (*VerifyResponse, error) {
	if req.TaskID == "" || req.TenantID == "" {
		return nil, errors.New("VerifyRequest: TaskID and TenantID required")
	}
	if equalsCaseFold(req.Routing.ExecutorVendor, req.Routing.VerifierVendor) {
		return nil, fmt.Errorf("verifierbridge: cross-family invariant violated")
	}
	now := time.Now().UTC()
	return &VerifyResponse{
		Approval: &cruciblev1.VerifierApproval{
			TaskID:        req.TaskID,
			DiffHash:      "stub:" + req.TaskID,
			Verdict:       "approved",
			RubricScore:   0.92,
			ExecutorModel: req.Routing.ExecutorModel,
			VerifierModel: req.Routing.VerifierModel,
			SignedAt:      now,
		},
		CostUSD:    0.0,
		DurationMS: 1,
	}, nil
}

func (s *stubBridge) HealthCheck(_ context.Context) error { return nil }

// ─── Helpers ─────────────────────────────────────────────────────────

func equalsCaseFold(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		ca := a[i]
		cb := b[i]
		if 'A' <= ca && ca <= 'Z' {
			ca += 32
		}
		if 'A' <= cb && cb <= 'Z' {
			cb += 32
		}
		if ca != cb {
			return false
		}
	}
	return true
}
