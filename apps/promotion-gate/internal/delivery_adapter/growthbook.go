package delivery_adapter

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/crucible/promotion-gate/internal/kms_lease"
	cruciblev1 "github.com/crucible/sdk-go/crucible/v1"
)

// GrowthBookAdapter creates a feature flag scoped to the change at
// promotion-time and steps it from 1 → 100% behind the outcome_watcher.
// Used for serverless / VM-based customers where Argo Rollouts isn't
// available.
type GrowthBookAdapter struct {
	API           string
	CreateFlag    func(ctx context.Context, key, description string) error
	SetFlagWeight func(ctx context.Context, key string, weight uint32) error
	DisableFlag   func(ctx context.Context, key string, reason string) error
}

// Name returns the adapter name.
func (a *GrowthBookAdapter) Name() string { return "growthbook" }

// Start creates the flag, scoped to the promotion id.
func (a *GrowthBookAdapter) Start(ctx context.Context, lease *kms_lease.Lease, bundle *cruciblev1.PromotionBundle) (*Handle, error) {
	if a.CreateFlag == nil {
		return nil, errors.New("growthbook: CreateFlag not wired")
	}
	key := "crucible_" + lease.PromotionID
	if err := a.CreateFlag(ctx, key, "Crucible promotion "+lease.PromotionID); err != nil {
		return nil, err
	}
	return &Handle{
		PromotionID: lease.PromotionID,
		Strategy:    StrategyFeatureFlagOnly,
		Adapter:     a.Name(),
		Resource:    key,
		Steps:       bundle.SuggestedRollout.Steps,
		StartedAt:   time.Now().UTC(),
		LeaseID:     lease.ID,
		BundleHash:  lease.BundleHash,
	}, nil
}

// Promote ramps the flag weight.
func (a *GrowthBookAdapter) Promote(ctx context.Context, h *Handle, nextWeight uint32) error {
	if a.SetFlagWeight == nil {
		return errors.New("growthbook: SetFlagWeight not wired")
	}
	return a.SetFlagWeight(ctx, h.Resource, nextWeight)
}

// Rollback flips the flag to 0%.
func (a *GrowthBookAdapter) Rollback(ctx context.Context, h *Handle, reason string) error {
	if a.DisableFlag == nil {
		return errors.New("growthbook: DisableFlag not wired")
	}
	return a.DisableFlag(ctx, h.Resource, reason)
}

// ── Local growthbook mock for dev ─────────────────────────────────────────

// LocalGrowthBookMock is the in-memory adapter.
type LocalGrowthBookMock struct {
	mu    sync.Mutex
	Flags map[string]*MockFlag
}

// MockFlag is the in-memory flag state.
type MockFlag struct {
	Key      string
	Weight   uint32
	Disabled bool
	Reason   string
}

// NewLocalGrowthBookMock builds the mock.
func NewLocalGrowthBookMock() *LocalGrowthBookMock {
	return &LocalGrowthBookMock{Flags: map[string]*MockFlag{}}
}

// Name returns the adapter name.
func (m *LocalGrowthBookMock) Name() string { return "growthbook" }

// Start implements Adapter.Start.
func (m *LocalGrowthBookMock) Start(_ context.Context, lease *kms_lease.Lease, bundle *cruciblev1.PromotionBundle) (*Handle, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := "crucible_" + lease.PromotionID
	m.Flags[key] = &MockFlag{Key: key}
	return &Handle{
		PromotionID: lease.PromotionID,
		Strategy:    StrategyFeatureFlagOnly,
		Adapter:     m.Name(),
		Resource:    key,
		Steps:       bundle.SuggestedRollout.Steps,
		StartedAt:   time.Now().UTC(),
		LeaseID:     lease.ID,
		BundleHash:  lease.BundleHash,
	}, nil
}

// Promote implements Adapter.Promote.
func (m *LocalGrowthBookMock) Promote(_ context.Context, h *Handle, nextWeight uint32) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	f, ok := m.Flags[h.Resource]
	if !ok {
		return errors.New("mock: flag not found: " + h.Resource)
	}
	if f.Disabled {
		return errors.New("mock: flag already disabled")
	}
	f.Weight = nextWeight
	return nil
}

// Rollback implements Adapter.Rollback.
func (m *LocalGrowthBookMock) Rollback(_ context.Context, h *Handle, reason string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	f, ok := m.Flags[h.Resource]
	if !ok {
		return errors.New("mock: flag not found: " + h.Resource)
	}
	f.Disabled = true
	f.Reason = reason
	f.Weight = 0
	return nil
}
