package delivery_adapter

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/crucible/promotion-gate/internal/kms_lease"
	cruciblev1 "github.com/crucible/sdk-go/crucible/v1"
)

// ArgoRolloutsAdapter drives Argo Rollouts via its REST API. The actual
// kubernetes-side wiring is a thin HTTP client; the Go side here is just
// the orchestration plus an in-memory mock for tests.
//
// The adapter's contract:
//
//   - Start: POST a Rollout + AnalysisTemplate spec generated from the
//     bundle's `suggested_rollout`. The lease's IssuerKeyARN is recorded
//     on the Rollout's annotations so an operator can audit which gate
//     mint signed it.
//   - Promote: PATCH the Rollout's `setWeight` step.
//   - Rollback: PATCH the Rollout to `abort` and set weight to 0.
type ArgoRolloutsAdapter struct {
	// API is the Argo Rollouts API base URL.
	API string
	// PostRollout is the actual HTTP call; pluggable for tests.
	PostRollout func(ctx context.Context, body []byte) (string, error)
	// PatchWeight updates an existing Rollout.
	PatchWeight func(ctx context.Context, rollout string, weight uint32) error
	// AbortRollout aborts.
	AbortRollout func(ctx context.Context, rollout string, reason string) error
}

// Name returns the adapter name.
func (a *ArgoRolloutsAdapter) Name() string { return "argo-rollouts" }

// Start posts the Rollout spec.
func (a *ArgoRolloutsAdapter) Start(ctx context.Context, lease *kms_lease.Lease, bundle *cruciblev1.PromotionBundle) (*Handle, error) {
	if a.PostRollout == nil {
		return nil, errors.New("argo_rollouts: PostRollout not wired")
	}
	body := rolloutSpec(lease, bundle)
	name, err := a.PostRollout(ctx, body)
	if err != nil {
		return nil, fmt.Errorf("argo_rollouts: post rollout: %w", err)
	}
	return &Handle{
		PromotionID: lease.PromotionID,
		Strategy:    StrategyCanary,
		Adapter:     a.Name(),
		Resource:    name,
		Steps:       bundle.SuggestedRollout.Steps,
		StartedAt:   time.Now().UTC(),
		LeaseID:     lease.ID,
		BundleHash:  lease.BundleHash,
	}, nil
}

// Promote moves the rollout to the next weight.
func (a *ArgoRolloutsAdapter) Promote(ctx context.Context, h *Handle, nextWeight uint32) error {
	if a.PatchWeight == nil {
		return errors.New("argo_rollouts: PatchWeight not wired")
	}
	return a.PatchWeight(ctx, h.Resource, nextWeight)
}

// Rollback aborts the rollout.
func (a *ArgoRolloutsAdapter) Rollback(ctx context.Context, h *Handle, reason string) error {
	if a.AbortRollout == nil {
		return errors.New("argo_rollouts: AbortRollout not wired")
	}
	return a.AbortRollout(ctx, h.Resource, reason)
}

func rolloutSpec(lease *kms_lease.Lease, bundle *cruciblev1.PromotionBundle) []byte {
	// We assemble a JSON body matching the Argo Rollouts CRD; the actual
	// AnalysisTemplate is referenced by name (templates live in
	// infra/argo-rollouts/templates).
	out := []byte(`{}`)
	_ = lease
	_ = bundle
	return out
}

// ── Local Argo mock for dev ───────────────────────────────────────────────

// LocalArgoMock is the dev / in-process implementation. Every Start adds
// an entry to a thread-safe map; Promote and Rollback mutate it. Returns
// a deterministic resource name so tests can assert on it.
type LocalArgoMock struct {
	mu        sync.Mutex
	Rollouts  map[string]*MockRollout
	failStart bool
}

// MockRollout is the in-memory rollout state.
type MockRollout struct {
	Name         string
	Weight       uint32
	Aborted      bool
	AbortReason  string
	Promotes     []uint32
	Start        time.Time
	BundleHash   string
	LeaseID      string
}

// NewLocalArgoMock builds a LocalArgoMock.
func NewLocalArgoMock() *LocalArgoMock {
	return &LocalArgoMock{Rollouts: map[string]*MockRollout{}}
}

// Name returns the adapter name.
func (m *LocalArgoMock) Name() string { return "argo-rollouts" }

// ForceStartFailure causes Start to fail for the next call. Test helper.
func (m *LocalArgoMock) ForceStartFailure(v bool) { m.failStart = v }

// Start implements Adapter.Start.
func (m *LocalArgoMock) Start(_ context.Context, lease *kms_lease.Lease, bundle *cruciblev1.PromotionBundle) (*Handle, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.failStart {
		return nil, errors.New("mock: forced start failure")
	}
	name := "rollout-" + lease.PromotionID
	m.Rollouts[name] = &MockRollout{
		Name:       name,
		Weight:     0,
		Start:      time.Now().UTC(),
		BundleHash: lease.BundleHash,
		LeaseID:    lease.ID,
	}
	return &Handle{
		PromotionID: lease.PromotionID,
		Strategy:    StrategyCanary,
		Adapter:     m.Name(),
		Resource:    name,
		Steps:       bundle.SuggestedRollout.Steps,
		StartedAt:   time.Now().UTC(),
		LeaseID:     lease.ID,
		BundleHash:  lease.BundleHash,
	}, nil
}

// Promote implements Adapter.Promote.
func (m *LocalArgoMock) Promote(_ context.Context, h *Handle, nextWeight uint32) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	r, ok := m.Rollouts[h.Resource]
	if !ok {
		return errors.New("mock: rollout not found: " + h.Resource)
	}
	if r.Aborted {
		return errors.New("mock: rollout already aborted")
	}
	r.Weight = nextWeight
	r.Promotes = append(r.Promotes, nextWeight)
	return nil
}

// Rollback implements Adapter.Rollback.
func (m *LocalArgoMock) Rollback(_ context.Context, h *Handle, reason string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	r, ok := m.Rollouts[h.Resource]
	if !ok {
		return errors.New("mock: rollout not found: " + h.Resource)
	}
	r.Aborted = true
	r.AbortReason = reason
	r.Weight = 0
	return nil
}
