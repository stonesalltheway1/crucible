// Package promotionbridge is the Phase-6 wiring between the control plane
// and the promotion gate (apps/promotion-gate).
//
// The control plane uses this bridge to:
//
//   - Submit a verified bundle to the gate (after the verifier has signed
//     VerifierApproval and the task transitioned to `promoting`).
//   - Poll the gate for current PromotionStatus.
//   - Receive landed/rolled_back events via the gate's webhook surface
//     (the control plane runs its standard events.Publisher).
//
// Env-gated: when `CRUCIBLE_PROMOTION_GATE_ADDR` is unset, the bridge is
// nil and `stub_promotion=true` continues to be the daemon's health-flag.
package promotionbridge

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

// EnvGateAddr is the env var that wires the bridge.
const EnvGateAddr = "CRUCIBLE_PROMOTION_GATE_ADDR"

// Bridge is the bridge contract.
type Bridge interface {
	HealthCheck(ctx context.Context) error
	Submit(ctx context.Context, req SubmitRequest) (*SubmitResponse, error)
	Status(ctx context.Context, promotionID string) (*StatusResponse, error)
}

// SubmitRequest is the gate-bound bundle envelope.
type SubmitRequest struct {
	Bundle           cruciblev1.PromotionBundle `json:"bundle"`
	TenantID         string                     `json:"tenant_id"`
	AgentOidcSubject string                     `json:"agent_oidc_subject"`
}

// SubmitResponse is the gate's response — minimal fields the control plane needs.
type SubmitResponse struct {
	ID     string                        `json:"id"`
	Status cruciblev1.PromotionStatusKind `json:"status"`
	Detail string                        `json:"detail,omitempty"`
}

// StatusResponse is the gate's GET /v1/promotions/{id} response (subset).
type StatusResponse struct {
	ID     string                        `json:"id"`
	Status cruciblev1.PromotionStatusKind `json:"status"`
	Detail string                        `json:"detail,omitempty"`
	UpdatedAt time.Time                  `json:"updated_at"`
}

// HTTPBridge is the real implementation.
type HTTPBridge struct {
	baseURL string
	client  *http.Client
}

// New builds a bridge from the env. Returns nil + a nil error when the env
// var is unset (allowing the control plane to keep running with
// stub_promotion=true).
func New() Bridge {
	addr := os.Getenv(EnvGateAddr)
	if addr == "" {
		return nil
	}
	return &HTTPBridge{
		baseURL: addr,
		client:  &http.Client{Timeout: 15 * time.Second},
	}
}

// HealthCheck implements Bridge.
func (b *HTTPBridge) HealthCheck(ctx context.Context) error {
	req, _ := http.NewRequestWithContext(ctx, "GET", b.baseURL+"/healthz", nil)
	resp, err := b.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("gate health=%d", resp.StatusCode)
	}
	return nil
}

// Submit implements Bridge.
func (b *HTTPBridge) Submit(ctx context.Context, req SubmitRequest) (*SubmitResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	r, _ := http.NewRequestWithContext(ctx, "POST", b.baseURL+"/v1/promotions", bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	resp, err := b.client.Do(r)
	if err != nil {
		return nil, fmt.Errorf("promotionbridge: submit: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusOK {
		bb, _ := io.ReadAll(resp.Body)
		if resp.StatusCode == http.StatusForbidden {
			return nil, &PolicyDeniedError{Status: resp.StatusCode, Body: string(bb)}
		}
		return nil, fmt.Errorf("promotionbridge: submit status=%d body=%s", resp.StatusCode, string(bb))
	}
	var sr SubmitResponse
	if err := json.NewDecoder(resp.Body).Decode(&sr); err != nil {
		return nil, err
	}
	return &sr, nil
}

// Status implements Bridge.
func (b *HTTPBridge) Status(ctx context.Context, promotionID string) (*StatusResponse, error) {
	r, _ := http.NewRequestWithContext(ctx, "GET", b.baseURL+"/v1/promotions/"+promotionID, nil)
	resp, err := b.client.Do(r)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrNotFound
	}
	if resp.StatusCode != http.StatusOK {
		bb, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("promotionbridge: status=%d body=%s", resp.StatusCode, string(bb))
	}
	var sr StatusResponse
	if err := json.NewDecoder(resp.Body).Decode(&sr); err != nil {
		return nil, err
	}
	return &sr, nil
}

// PolicyDeniedError is returned when the gate's Rego refused the bundle.
type PolicyDeniedError struct {
	Status int
	Body   string
}

func (e *PolicyDeniedError) Error() string {
	return fmt.Sprintf("promotion policy denied: status=%d body=%s", e.Status, e.Body)
}

// ErrNotFound is returned on GET 404.
var ErrNotFound = errors.New("promotionbridge: promotion not found")
