// Package delivery_adapter hands off an approved promotion to a real-world
// delivery system. Two paths:
//
//   - K8s — Argo Rollouts. We POST an AnalysisRun + Rollout spec generated
//     from the bundle's `suggested_rollout`.
//   - Serverless / VM — GrowthBook feature flags. We create a flag scoped
//     to the change and step it from 1 → 5 → 25 → 100% behind the
//     outcome_watcher's SLO checks.
//
// Both adapters return a `Handle` the outcome_watcher polls until the
// rollout is `landed` or `rolled_back`.
package delivery_adapter

import (
	"context"
	"errors"
	"sort"
	"sync"
	"time"

	"github.com/crucible/promotion-gate/internal/kms_lease"
	cruciblev1 "github.com/crucible/sdk-go/crucible/v1"
)

// Strategy is the rollout-strategy selector pulled from the bundle.
type Strategy string

// Known strategies.
const (
	StrategyCanary          Strategy = "canary"
	StrategyBlueGreen       Strategy = "blue-green"
	StrategyFeatureFlagOnly Strategy = "feature-flag-only"
)

// Handle is what the outcome_watcher polls.
type Handle struct {
	PromotionID     string
	Strategy        Strategy
	Adapter         string // "argo-rollouts" | "growthbook" | "dev-local"
	Resource        string // K8s Rollout name OR feature-flag key
	CurrentWeight   uint32
	Steps           []cruciblev1.SuggestedRolloutStep
	StartedAt       time.Time
	StepIndex       int
	LeaseID         string
	HumanApprovers  []string
	BundleHash      string
	HistoricalSteps []cruciblev1.PromotionOutcomeStep
	mu              sync.Mutex
}

// Snapshot returns a copy of the handle state for read-only consumers.
func (h *Handle) Snapshot() Handle {
	h.mu.Lock()
	defer h.mu.Unlock()
	cp := *h
	cp.HistoricalSteps = append([]cruciblev1.PromotionOutcomeStep{}, h.HistoricalSteps...)
	return cp
}

// AppendStep records a rollout step's outcome on the handle.
func (h *Handle) AppendStep(step cruciblev1.PromotionOutcomeStep) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.HistoricalSteps = append(h.HistoricalSteps, step)
	h.CurrentWeight = step.Weight
	h.StepIndex++
}

// Adapter is the contract: Start kicks off the rollout, Promote moves to
// the next weight, Rollback flips the flag / aborts the rollout.
type Adapter interface {
	Name() string
	Start(ctx context.Context, lease *kms_lease.Lease, bundle *cruciblev1.PromotionBundle) (*Handle, error)
	Promote(ctx context.Context, handle *Handle, nextWeight uint32) error
	Rollback(ctx context.Context, handle *Handle, reason string) error
}

// Pool routes to the correct adapter based on the bundle's
// suggested_rollout.strategy.
type Pool struct {
	adapters map[Strategy]Adapter
	def      Strategy
}

// NewPool constructs a Pool with the given adapters.
func NewPool(adapters map[Strategy]Adapter, def Strategy) *Pool {
	return &Pool{adapters: adapters, def: def}
}

// Start picks an adapter based on the bundle's suggested rollout strategy.
func (p *Pool) Start(ctx context.Context, lease *kms_lease.Lease, bundle *cruciblev1.PromotionBundle) (*Handle, error) {
	strat := strategyForBundle(bundle, p.def)
	a, ok := p.adapters[strat]
	if !ok {
		return nil, errors.New("delivery_adapter: no adapter for strategy " + string(strat))
	}
	handle, err := a.Start(ctx, lease, bundle)
	if err != nil {
		return nil, err
	}
	// Ensure the handle records the chosen strategy + steps.
	handle.Strategy = strat
	if len(handle.Steps) == 0 {
		handle.Steps = bundle.SuggestedRollout.Steps
	}
	sort.SliceStable(handle.Steps, func(i, j int) bool { return handle.Steps[i].Weight < handle.Steps[j].Weight })
	return handle, nil
}

// Promote on the underlying adapter.
func (p *Pool) Promote(ctx context.Context, h *Handle, nextWeight uint32) error {
	a, ok := p.adapters[h.Strategy]
	if !ok {
		return errors.New("delivery_adapter: lost adapter for strategy " + string(h.Strategy))
	}
	return a.Promote(ctx, h, nextWeight)
}

// Rollback on the underlying adapter.
func (p *Pool) Rollback(ctx context.Context, h *Handle, reason string) error {
	a, ok := p.adapters[h.Strategy]
	if !ok {
		return errors.New("delivery_adapter: lost adapter for strategy " + string(h.Strategy))
	}
	return a.Rollback(ctx, h, reason)
}

func strategyForBundle(b *cruciblev1.PromotionBundle, def Strategy) Strategy {
	if len(b.SuggestedRollout.Steps) == 0 {
		return def
	}
	// Phase-6: we don't have an explicit `strategy` field on
	// SuggestedRollout (Phase 1 stored it only on the promotion contract
	// doc); infer from step shape — canary when steps are weight-only,
	// feature-flag-only when single step at 100% with non-zero dwell.
	if len(b.SuggestedRollout.Steps) == 1 && b.SuggestedRollout.Steps[0].Weight >= 100 {
		return StrategyFeatureFlagOnly
	}
	return StrategyCanary
}
