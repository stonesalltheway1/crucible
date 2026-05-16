// Package outcome_watcher steps an in-flight rollout through its weight
// schedule, checks SLOs at each dwell, and emits a `PromotionOutcome/v1`
// attestation on land or auto-rollback.
//
// The watcher runs as a background goroutine per active promotion. The
// gate's api/server keeps a handle so /v1/promotions/{id} returns live
// state.
package outcome_watcher

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/crucible/promotion-gate/internal/delivery_adapter"
	cruciblev1 "github.com/crucible/sdk-go/crucible/v1"
)

// SloChecker evaluates whether an SLO is currently green for a given
// rollout. Implementations: PrometheusChecker (real), FakeChecker (tests).
type SloChecker interface {
	Check(ctx context.Context, handle *delivery_adapter.Handle) (SloVerdict, error)
}

// SloVerdict is the structured result of a single SLO check.
type SloVerdict struct {
	Passed  bool     `json:"passed"`
	Reasons []string `json:"reasons,omitempty"`
	Metrics map[string]float64 `json:"metrics,omitempty"`
}

// OutcomeSink emits the final PromotionOutcome/v1 attestation. Backed by
// the attestation relay client in production.
type OutcomeSink interface {
	EmitOutcome(ctx context.Context, predicate cruciblev1.PromotionOutcomeAttestation) (rekorUUID string, err error)
}

// Watcher orchestrates a single promotion's rollout.
type Watcher struct {
	pool   *delivery_adapter.Pool
	slo    SloChecker
	sink   OutcomeSink
	clock  func() time.Time
	sleep  func(ctx context.Context, d time.Duration) error
}

// New builds a Watcher.
func New(pool *delivery_adapter.Pool, slo SloChecker, sink OutcomeSink) *Watcher {
	return &Watcher{
		pool: pool, slo: slo, sink: sink,
		clock: func() time.Time { return time.Now().UTC() },
		sleep: defaultSleep,
	}
}

// SetClock overrides the clock for tests.
func (w *Watcher) SetClock(c func() time.Time) { w.clock = c }

// SetSleep overrides the sleep function for tests (so dwell-seconds become instantaneous).
func (w *Watcher) SetSleep(s func(ctx context.Context, d time.Duration) error) { w.sleep = s }

// RunOnce drives a Handle through its remaining steps. Returns the final
// PromotionOutcome attestation. Blocks until landed or rolled back.
func (w *Watcher) RunOnce(ctx context.Context, h *delivery_adapter.Handle) (cruciblev1.PromotionOutcomeAttestation, error) {
	if h == nil {
		return cruciblev1.PromotionOutcomeAttestation{}, errors.New("outcome_watcher: nil handle")
	}
	if len(h.Steps) == 0 {
		return cruciblev1.PromotionOutcomeAttestation{}, errors.New("outcome_watcher: no rollout steps")
	}
	outcome := cruciblev1.PromotionOutcomeAttestation{
		PromotionID:       h.PromotionID,
		BundleAttestation: "rekor:bundle-" + h.PromotionID, // populated by API layer to a real UUID
	}

	for i, step := range h.Steps {
		if i < h.StepIndex {
			continue
		}
		if err := w.pool.Promote(ctx, h, step.Weight); err != nil {
			outcome.Outcome = "rolled_back"
			outcome.RollbackReason = "promote failed: " + err.Error()
			outcome.CompletedAt = w.clock()
			_ = w.emit(ctx, &outcome)
			return outcome, err
		}

		// Dwell.
		if step.DwellSeconds > 0 {
			if err := w.sleep(ctx, time.Duration(step.DwellSeconds)*time.Second); err != nil {
				outcome.Outcome = "rolled_back"
				outcome.RollbackReason = "dwell interrupted: " + err.Error()
				outcome.CompletedAt = w.clock()
				_ = w.emit(ctx, &outcome)
				return outcome, err
			}
		}

		// SLO check.
		verdict, err := w.slo.Check(ctx, h)
		if err != nil {
			return outcome, fmt.Errorf("outcome_watcher: slo check: %w", err)
		}
		stepRecord := cruciblev1.PromotionOutcomeStep{
			Weight:       step.Weight,
			DwellSeconds: step.DwellSeconds,
			SloCheck:     sloLabel(verdict),
			Timestamp:    w.clock(),
		}
		h.AppendStep(stepRecord)
		outcome.RolloutSteps = append(outcome.RolloutSteps, stepRecord)

		if !verdict.Passed {
			// Auto-rollback fires inside one SLO-check cycle of regression
			// detection.
			if err := w.pool.Rollback(ctx, h, "SLO regression: "+joinReasons(verdict.Reasons)); err != nil {
				outcome.Outcome = "rolled_back"
				outcome.RollbackReason = "rollback err: " + err.Error()
			} else {
				outcome.Outcome = "rolled_back"
				outcome.RollbackReason = joinReasons(verdict.Reasons)
			}
			outcome.CompletedAt = w.clock()
			outcome.FinalState = fmt.Sprintf("%d%% rolled back", step.Weight)
			_ = w.emit(ctx, &outcome)
			return outcome, nil
		}
	}

	outcome.Outcome = "landed"
	outcome.FinalState = "100% live"
	outcome.CompletedAt = w.clock()
	if _, err := w.sink.EmitOutcome(ctx, outcome); err != nil {
		return outcome, fmt.Errorf("outcome_watcher: emit outcome: %w", err)
	}
	return outcome, nil
}

func (w *Watcher) emit(ctx context.Context, outcome *cruciblev1.PromotionOutcomeAttestation) error {
	if w.sink == nil {
		return nil
	}
	_, err := w.sink.EmitOutcome(ctx, *outcome)
	return err
}

func sloLabel(v SloVerdict) string {
	if v.Passed {
		return "passed"
	}
	return "failed"
}

func joinReasons(rs []string) string {
	out := ""
	for i, r := range rs {
		if i > 0 {
			out += "; "
		}
		out += r
	}
	if out == "" {
		out = "SLO regression"
	}
	return out
}

func defaultSleep(ctx context.Context, d time.Duration) error {
	select {
	case <-time.After(d):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// ── helpers ────────────────────────────────────────────────────────────────

// FakeSloChecker is the test double. Public so the gate-level integration
// tests can drive it.
type FakeSloChecker struct {
	mu       sync.Mutex
	Verdicts []SloVerdict
	idx      int
}

// NewFakeSloChecker builds a FakeSloChecker.
func NewFakeSloChecker(verdicts ...SloVerdict) *FakeSloChecker {
	return &FakeSloChecker{Verdicts: verdicts}
}

// Check implements SloChecker.
func (f *FakeSloChecker) Check(_ context.Context, _ *delivery_adapter.Handle) (SloVerdict, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.idx >= len(f.Verdicts) {
		return SloVerdict{Passed: true}, nil
	}
	v := f.Verdicts[f.idx]
	f.idx++
	return v, nil
}

// FakeOutcomeSink records emissions.
type FakeOutcomeSink struct {
	mu  sync.Mutex
	out []cruciblev1.PromotionOutcomeAttestation
}

// NewFakeOutcomeSink builds a FakeOutcomeSink.
func NewFakeOutcomeSink() *FakeOutcomeSink {
	return &FakeOutcomeSink{}
}

// EmitOutcome implements OutcomeSink.
func (f *FakeOutcomeSink) EmitOutcome(_ context.Context, predicate cruciblev1.PromotionOutcomeAttestation) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.out = append(f.out, predicate)
	return "rekor:fake-outcome", nil
}

// Last returns the most-recent emission.
func (f *FakeOutcomeSink) Last() (cruciblev1.PromotionOutcomeAttestation, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.out) == 0 {
		return cruciblev1.PromotionOutcomeAttestation{}, false
	}
	return f.out[len(f.out)-1], true
}
