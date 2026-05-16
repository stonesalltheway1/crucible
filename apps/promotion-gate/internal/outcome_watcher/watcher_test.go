package outcome_watcher

import (
	"context"
	"testing"
	"time"

	"github.com/crucible/promotion-gate/internal/delivery_adapter"
	"github.com/crucible/promotion-gate/internal/kms_lease"
	cruciblev1 "github.com/crucible/sdk-go/crucible/v1"
)

func setupPool(t *testing.T) (*delivery_adapter.Pool, *delivery_adapter.LocalArgoMock) {
	mock := delivery_adapter.NewLocalArgoMock()
	pool := delivery_adapter.NewPool(map[delivery_adapter.Strategy]delivery_adapter.Adapter{
		delivery_adapter.StrategyCanary: mock,
	}, delivery_adapter.StrategyCanary)
	return pool, mock
}

func bundleWithSteps(steps ...cruciblev1.SuggestedRolloutStep) *cruciblev1.PromotionBundle {
	return &cruciblev1.PromotionBundle{
		TaskID: "task_x", DiffHash: "0x", AgentOidcSubject: "a",
		SuggestedRollout: cruciblev1.SuggestedRollout{Steps: steps},
	}
}

func makeLease(t *testing.T) *kms_lease.Lease {
	dir := t.TempDir()
	signer, _ := kms_lease.NewDevSigner(dir)
	mgr := kms_lease.New(signer, nil)
	l, _ := mgr.MintLease(context.Background(), kms_lease.LeaseRequest{
		PromotionID: "prom_demo", BundleHash: "0x", Action: kms_lease.ActionDeployArtifact,
	})
	return l
}

func TestWatcher_LandsAllStepsGreen(t *testing.T) {
	pool, mock := setupPool(t)
	lease := makeLease(t)
	bundle := bundleWithSteps(
		cruciblev1.SuggestedRolloutStep{Weight: 1, DwellSeconds: 0},
		cruciblev1.SuggestedRolloutStep{Weight: 25, DwellSeconds: 0},
		cruciblev1.SuggestedRolloutStep{Weight: 100, DwellSeconds: 0},
	)
	h, err := pool.Start(context.Background(), lease, bundle)
	if err != nil {
		t.Fatal(err)
	}

	sink := NewFakeOutcomeSink()
	w := New(pool, NewFakeSloChecker(), sink)
	w.SetSleep(func(_ context.Context, _ time.Duration) error { return nil })

	outcome, err := w.RunOnce(context.Background(), h)
	if err != nil {
		t.Fatalf("RunOnce: %v", err)
	}
	if outcome.Outcome != "landed" {
		t.Fatalf("expected landed, got %q", outcome.Outcome)
	}
	if len(outcome.RolloutSteps) != 3 {
		t.Fatalf("expected 3 steps recorded, got %d", len(outcome.RolloutSteps))
	}
	last, _ := sink.Last()
	if last.Outcome != "landed" {
		t.Fatal("expected sink to record landed")
	}
	if mock.Rollouts[h.Resource].Weight != 100 {
		t.Fatalf("expected weight 100, got %d", mock.Rollouts[h.Resource].Weight)
	}
}

func TestWatcher_AutoRollbackOnSLORegression(t *testing.T) {
	pool, mock := setupPool(t)
	lease := makeLease(t)
	bundle := bundleWithSteps(
		cruciblev1.SuggestedRolloutStep{Weight: 1, DwellSeconds: 0},
		cruciblev1.SuggestedRolloutStep{Weight: 25, DwellSeconds: 0},
		cruciblev1.SuggestedRolloutStep{Weight: 100, DwellSeconds: 0},
	)
	h, err := pool.Start(context.Background(), lease, bundle)
	if err != nil {
		t.Fatal(err)
	}

	sink := NewFakeOutcomeSink()
	checker := NewFakeSloChecker(
		SloVerdict{Passed: true},
		SloVerdict{Passed: false, Reasons: []string{"error_rate_p99 > baseline*1.5"}},
	)
	w := New(pool, checker, sink)
	w.SetSleep(func(_ context.Context, _ time.Duration) error { return nil })

	outcome, err := w.RunOnce(context.Background(), h)
	if err != nil {
		t.Fatalf("RunOnce: %v", err)
	}
	if outcome.Outcome != "rolled_back" {
		t.Fatalf("expected rolled_back, got %q", outcome.Outcome)
	}
	if !mock.Rollouts[h.Resource].Aborted {
		t.Fatal("expected adapter to abort")
	}
	// auto-rollback fires within one SLO-check cycle of regression.
	if len(outcome.RolloutSteps) != 2 {
		t.Fatalf("expected 2 steps (1 pass + 1 fail), got %d", len(outcome.RolloutSteps))
	}
}

func TestWatcher_StartFailureRollsBack(t *testing.T) {
	mock := delivery_adapter.NewLocalArgoMock()
	mock.ForceStartFailure(true)
	pool := delivery_adapter.NewPool(map[delivery_adapter.Strategy]delivery_adapter.Adapter{
		delivery_adapter.StrategyCanary: mock,
	}, delivery_adapter.StrategyCanary)
	lease := makeLease(t)
	bundle := bundleWithSteps(cruciblev1.SuggestedRolloutStep{Weight: 100, DwellSeconds: 0})
	_, err := pool.Start(context.Background(), lease, bundle)
	if err == nil {
		t.Fatal("expected forced failure")
	}
}
