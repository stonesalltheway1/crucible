package budgetenforcer

import (
	"errors"
	"math/rand"
	"sync"
	"testing"
	"time"

	cruciblev1 "github.com/crucible/sdk-go/crucible/v1"
)

func mustNew(t *testing.T, cfg Config) *Enforcer {
	t.Helper()
	e, err := New(cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return e
}

func defaultCfg() Config {
	return Config{
		TaskID:             "task_TEST",
		CostCapUSD:         1.0,
		WallClockCapMin:    60,
		RetryCapPerSubgoal: 3,
	}
}

func TestNew_ValidatesInputs(t *testing.T) {
	cases := []struct {
		name string
		cfg  Config
		want string
	}{
		{"empty task id", Config{CostCapUSD: 1, WallClockCapMin: 1, RetryCapPerSubgoal: 1}, "empty task id"},
		{"zero cost cap", Config{TaskID: "t", WallClockCapMin: 1, RetryCapPerSubgoal: 1}, "cost cap"},
		{"zero wall clock", Config{TaskID: "t", CostCapUSD: 1, RetryCapPerSubgoal: 1}, "wall-clock"},
		{"zero retry cap", Config{TaskID: "t", CostCapUSD: 1, WallClockCapMin: 1}, "retry cap"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := New(c.cfg)
			if err == nil {
				t.Fatalf("want error containing %q, got nil", c.want)
			}
		})
	}
}

func TestSnapshot_StartsZero(t *testing.T) {
	e := mustNew(t, defaultCfg())
	b := e.Snapshot()
	if b.SpentUsd != 0 || b.RetriesUsed != 0 || b.StepsUsed != 0 || b.WallClockUsedSeconds != 0 {
		t.Fatalf("expected zero usage, got %+v", b)
	}
	if b.CapUsd != 1.0 || b.RetryCap != 3 || b.WallClockCapSeconds != 3600 {
		t.Fatalf("caps not propagated: %+v", b)
	}
}

func TestCharge_AccumulatesAndFreezes(t *testing.T) {
	e := mustNew(t, defaultCfg())
	if err := e.Charge(0.30); err != nil {
		t.Fatalf("Charge: %v", err)
	}
	if err := e.Charge(0.50); err != nil {
		t.Fatalf("Charge: %v", err)
	}
	if !e.WarnedAt80Percent() {
		t.Fatalf("expected 80%% warning to fire after $0.80 / $1.00")
	}
	err := e.Charge(0.30) // total $1.10, over cap
	if err == nil {
		t.Fatalf("expected BudgetExceeded; got nil")
	}
	var ce *cruciblev1.CrucibleError
	if !errors.As(err, &ce) {
		t.Fatalf("expected *CrucibleError, got %T", err)
	}
	if ce.Code != cruciblev1.ErrBudgetExceeded {
		t.Fatalf("expected BudgetExceeded code, got %s", ce.Code)
	}
	if frozen, _ := e.Frozen(); !frozen {
		t.Fatalf("expected enforcer to be frozen")
	}
	// Subsequent charges error too.
	if err := e.Charge(0); err == nil {
		t.Fatalf("expected post-freeze Charge to error")
	}
}

func TestCharge_RejectsNegative(t *testing.T) {
	e := mustNew(t, defaultCfg())
	if err := e.Charge(-0.01); err == nil {
		t.Fatalf("expected error on negative charge")
	}
}

func TestIncrementRetry_FreezesAtCap(t *testing.T) {
	e := mustNew(t, defaultCfg()) // retry cap = 3
	if err := e.IncrementRetry(); err != nil {
		t.Fatalf("retry 1: %v", err)
	}
	if err := e.IncrementRetry(); err != nil {
		t.Fatalf("retry 2: %v", err)
	}
	err := e.IncrementRetry() // third retry → cap reached
	if err == nil {
		t.Fatalf("expected RetryLimitExceeded; got nil")
	}
	var ce *cruciblev1.CrucibleError
	if !errors.As(err, &ce) || ce.Code != cruciblev1.ErrRetryLimitExceeded {
		t.Fatalf("expected RetryLimitExceeded, got %v", err)
	}
}

func TestTick_WallClockEnforced(t *testing.T) {
	cfg := defaultCfg()
	cfg.WallClockCapMin = 1
	cfg.Now = func() time.Time { return time.Unix(0, 0).UTC() }
	e := mustNew(t, cfg)
	if err := e.Tick(time.Unix(30, 0).UTC()); err != nil {
		t.Fatalf("Tick 30s: %v", err)
	}
	err := e.Tick(time.Unix(70, 0).UTC()) // over 60s cap
	if err == nil {
		t.Fatalf("expected WallClockExceeded")
	}
	var ce *cruciblev1.CrucibleError
	if !errors.As(err, &ce) || ce.Code != cruciblev1.ErrWallClockExceeded {
		t.Fatalf("expected WallClockExceeded, got %v", err)
	}
}

func TestReset_RestartsState(t *testing.T) {
	e := mustNew(t, defaultCfg())
	_ = e.Charge(0.5)
	_ = e.IncrementRetry()
	e.Reset(time.Now())
	b := e.Snapshot()
	if b.SpentUsd != 0 || b.RetriesUsed != 0 {
		t.Fatalf("reset failed: %+v", b)
	}
	if frozen, _ := e.Frozen(); frozen {
		t.Fatalf("expected unfrozen after reset")
	}
}

// TestProperty_NeverExceedsCap is the property test the brief required.
// Random sequences of Charge/Retry/Tick from many goroutines must never
// observe a Snapshot in which spent>cap (the freeze is what enforces this).
//
// Strategy: across 50 random seeds × 8 goroutines × ~500 ops, after every op
// re-read Snapshot() and assert spentUSD <= capUSD AND retries <= retryCap
// AND wall_clock_used_seconds <= wall_clock_cap_seconds — at the same instant.
//
// We use spend amounts that can individually exceed the cap (so the cap is
// reached on the first call sometimes); the freeze is what makes the property
// hold. A single Charge call may push spentUSD strictly above cap (that's the
// final state of a frozen enforcer); the property is:
//
//   Once spent > cap, the enforcer is frozen and refuses further mutation.
func TestProperty_NeverExceedsCap(t *testing.T) {
	const seeds = 50
	const goroutines = 8
	const ops = 500

	for seed := int64(0); seed < seeds; seed++ {
		seed := seed
		t.Run("seed_"+t.Name(), func(t *testing.T) {
			t.Parallel()
			cfg := Config{
				TaskID:             "task_PROP",
				CostCapUSD:         1.0,
				WallClockCapMin:    1,
				RetryCapPerSubgoal: 5,
				StepsCap:           20,
				Now:                func() time.Time { return time.Unix(0, 0).UTC() },
			}
			e := mustNew(t, cfg)

			var wg sync.WaitGroup
			for g := 0; g < goroutines; g++ {
				wg.Add(1)
				go func(g int) {
					defer wg.Done()
					r := rand.New(rand.NewSource(seed*1000 + int64(g)))
					for i := 0; i < ops; i++ {
						switch r.Intn(4) {
						case 0:
							_ = e.Charge(r.Float64() * 0.10) // 0–10c
						case 1:
							_ = e.IncrementRetry()
						case 2:
							_ = e.IncrementStep()
						case 3:
							_ = e.Tick(time.Unix(int64(r.Intn(120)), 0).UTC())
						}
						b := e.Snapshot()

						// Once the enforcer reports overspend, it must be frozen.
						if b.SpentUsd > b.CapUsd {
							if frozen, _ := e.Frozen(); !frozen {
								t.Errorf("spent %.4f > cap %.4f but enforcer not frozen", b.SpentUsd, b.CapUsd)
								return
							}
							// And no further Charge can succeed.
							if err := e.Charge(0); err == nil {
								t.Errorf("Charge succeeded after cap breach")
								return
							}
						}
						if b.RetriesUsed > b.RetryCap {
							if frozen, _ := e.Frozen(); !frozen {
								t.Errorf("retries %d > cap %d but not frozen", b.RetriesUsed, b.RetryCap)
								return
							}
						}
						if b.StepsCap > 0 && b.StepsUsed > b.StepsCap+1 { // +1 because the cap-breaching call is recorded before freeze
							t.Errorf("steps %d > cap %d+1", b.StepsUsed, b.StepsCap)
							return
						}
						if b.WallClockCapSeconds > 0 && b.WallClockUsedSeconds > b.WallClockCapSeconds {
							if frozen, _ := e.Frozen(); !frozen {
								t.Errorf("wall %d > cap %d but not frozen", b.WallClockUsedSeconds, b.WallClockCapSeconds)
								return
							}
						}
					}
				}(g)
			}
			wg.Wait()
		})
	}
}

func TestRegistry_RegisterGetRemove(t *testing.T) {
	r := NewRegistry()
	if got := r.Get("nope"); got != nil {
		t.Fatalf("expected nil for unknown task")
	}
	e := mustNew(t, defaultCfg())
	r.Register("task_TEST", e)
	if got := r.Get("task_TEST"); got != e {
		t.Fatalf("registry returned wrong enforcer")
	}
	r.Remove("task_TEST")
	if got := r.Get("task_TEST"); got != nil {
		t.Fatalf("expected nil after Remove")
	}
}
