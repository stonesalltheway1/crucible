// Package memorybridge is the verifier-side connector to the Phase-5
// memory-router.
//
// The Phase-4 verifier rubric reserves a slot for the
// trust_signal_alignment criterion to consult the memory layer (the
// "Memory as verifier" loop closure from
// docs/01-architecture/memory-layer.md). Phase 5 wires that slot here.
//
// When CRUCIBLE_MEMORY_ROUTER_ADDR is set, the bridge POSTs the diff
// to /v1/memory/check_compliance and surfaces the report. When unset
// the bridge returns an empty report — the verifier degrades gracefully
// to the Phase-4 trust signals, matching the env-gated bridge pattern
// the verifierbridge already uses.
//
// CRITICAL: this bridge MUST NOT receive any executor reasoning. The
// memory-router never sees the reasoning trace — it sees only the diff
// hash + the per-file paths. The audit guard at the verifier ingress
// already enforces that invariant; the bridge is a downstream consumer.
package memorybridge

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	cruciblev1 "github.com/crucible/sdk-go/crucible/v1"
)

const (
	// EnvRouterAddr is the HTTP base URL of the memory-router daemon.
	EnvRouterAddr = "CRUCIBLE_MEMORY_ROUTER_ADDR"
	// DefaultRouterAddr is where the daemon binds locally.
	DefaultRouterAddr = "http://127.0.0.1:8090"
)

// Bridge is the verifier-side memory-router client.
type Bridge interface {
	CheckCompliance(ctx context.Context, req CheckRequest) (cruciblev1.ComplianceReport, error)
	ListConventions(ctx context.Context, req ListRequest) ([]cruciblev1.Convention, error)
	HealthCheck(ctx context.Context) error
}

// CheckRequest is the bridge's input for a compliance check.
type CheckRequest struct {
	TenantID string
	TaskID   string
	Diff     cruciblev1.Diff
}

// ListRequest narrows a conventions lookup. Empty scope returns all
// active conventions for the tenant — used by the rubric to compute the
// "conventions available" trust signal even when the diff is empty.
type ListRequest struct {
	TenantID string
	Scope    cruciblev1.ScopeFilter
	Limit    int
}

// New returns the env-gated bridge. When CRUCIBLE_MEMORY_ROUTER_ADDR is
// empty, returns a stub bridge that emits empty reports (no-op trust
// signal). The verifier degrades gracefully — memory-aware compliance
// becomes a "nothing extra" signal, not a hard fail.
func New() Bridge {
	addr := os.Getenv(EnvRouterAddr)
	if addr == "" {
		return &noop{reason: "CRUCIBLE_MEMORY_ROUTER_ADDR unset; memory compliance trust signal disabled"}
	}
	return &httpBridge{addr: addr, client: &http.Client{Timeout: 5 * time.Second}}
}

// NewStub returns a no-op bridge that always returns empty reports.
// Used in unit tests of the rubric path.
func NewStub() Bridge { return &noop{reason: "stub"} }

// ─── HTTP impl ──────────────────────────────────────────────────────────────

type httpBridge struct {
	addr   string
	client *http.Client
}

type checkReqWire struct {
	TenantID string          `json:"tenant_id"`
	TaskID   string          `json:"task_id"`
	Diff     cruciblev1.Diff `json:"diff"`
}
type checkRespWire struct {
	Report cruciblev1.ComplianceReport `json:"report"`
}

func (b *httpBridge) CheckCompliance(ctx context.Context, req CheckRequest) (cruciblev1.ComplianceReport, error) {
	if req.TenantID == "" {
		return cruciblev1.ComplianceReport{}, errors.New("memorybridge: tenant_id required")
	}
	body, _ := json.Marshal(checkReqWire{TenantID: req.TenantID, TaskID: req.TaskID, Diff: req.Diff})
	r, err := http.NewRequestWithContext(ctx, http.MethodPost, b.addr+"/v1/memory/check_compliance", bytes.NewReader(body))
	if err != nil {
		return cruciblev1.ComplianceReport{}, err
	}
	r.Header.Set("Content-Type", "application/json")
	resp, err := b.client.Do(r)
	if err != nil {
		return cruciblev1.ComplianceReport{}, fmt.Errorf("memorybridge: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return cruciblev1.ComplianceReport{}, fmt.Errorf("memorybridge: HTTP %d", resp.StatusCode)
	}
	var wire checkRespWire
	if err := json.NewDecoder(resp.Body).Decode(&wire); err != nil {
		return cruciblev1.ComplianceReport{}, fmt.Errorf("memorybridge decode: %w", err)
	}
	return wire.Report, nil
}

type listReqWire struct {
	TenantID string                 `json:"tenant_id"`
	Scope    cruciblev1.ScopeFilter `json:"scope"`
	Limit    int                    `json:"limit"`
}
type listRespWire struct {
	Conventions []cruciblev1.Convention `json:"conventions"`
}

func (b *httpBridge) ListConventions(ctx context.Context, req ListRequest) ([]cruciblev1.Convention, error) {
	if req.TenantID == "" {
		return nil, errors.New("memorybridge: tenant_id required")
	}
	body, _ := json.Marshal(listReqWire{TenantID: req.TenantID, Scope: req.Scope, Limit: req.Limit})
	r, err := http.NewRequestWithContext(ctx, http.MethodPost, b.addr+"/v1/memory/conventions", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	r.Header.Set("Content-Type", "application/json")
	resp, err := b.client.Do(r)
	if err != nil {
		return nil, fmt.Errorf("memorybridge: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("memorybridge: HTTP %d", resp.StatusCode)
	}
	var wire listRespWire
	if err := json.NewDecoder(resp.Body).Decode(&wire); err != nil {
		return nil, fmt.Errorf("memorybridge decode: %w", err)
	}
	return wire.Conventions, nil
}

func (b *httpBridge) HealthCheck(ctx context.Context) error {
	r, err := http.NewRequestWithContext(ctx, http.MethodGet, b.addr+"/healthz", nil)
	if err != nil {
		return err
	}
	resp, err := b.client.Do(r)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("memory-router health: HTTP %d", resp.StatusCode)
	}
	return nil
}

// ─── No-op bridge ───────────────────────────────────────────────────────────

type noop struct{ reason string }

func (n *noop) CheckCompliance(ctx context.Context, req CheckRequest) (cruciblev1.ComplianceReport, error) {
	_ = ctx
	_ = req
	return cruciblev1.ComplianceReport{}, nil
}
func (n *noop) ListConventions(ctx context.Context, req ListRequest) ([]cruciblev1.Convention, error) {
	_ = ctx
	_ = req
	return nil, nil
}
func (n *noop) HealthCheck(ctx context.Context) error { return nil }
