// Package budgetenforcer implements ADR-009 — the hard caps on per-task cost,
// retry count, and wall-clock duration.
//
// The Enforcer is a sidecar pattern: model_router calls Charge() on every LLM
// response; task_router calls IncrementRetry() on every retry; periodically
// the runtime calls Tick() to advance the wall-clock. Any cap breach returns
// a *cruciblev1.CrucibleError that the caller must surface — there is no
// silent-deny path.
//
// Concurrency: every method is safe to call from multiple goroutines.
package budgetenforcer

import (
	"errors"
	"fmt"
	"sync"
	"time"

	cruciblev1 "github.com/crucible/sdk-go/crucible/v1"
)

// Enforcer tracks one task's consumption against caps.
type Enforcer struct {
	mu                sync.Mutex
	taskID            string
	costCapUSD        float64
	wallClockCapSec   uint64
	retryCap          uint32
	stepsCap          uint32
	startedAt         time.Time
	spentUSD          float64
	retriesUsed       uint32
	stepsUsed         uint32
	wallClockUsedSec  uint64
	frozen            bool
	frozenReason      string
	warnedAt80Percent bool
}

// Config is the per-task configuration the Enforcer enforces.
type Config struct {
	TaskID                 string
	CostCapUSD             float64
	WallClockCapMin        uint32 // converted internally to seconds
	RetryCapPerSubgoal     uint32
	StepsCap               uint32 // optional hard cap on total step count; 0 means unlimited
	Now                    func() time.Time
}

// New constructs an Enforcer. All caps must be positive.
func New(cfg Config) (*Enforcer, error) {
	if cfg.TaskID == "" {
		return nil, errors.New("budgetenforcer: empty task id")
	}
	if cfg.CostCapUSD <= 0 {
		return nil, errors.New("budgetenforcer: cost cap must be > 0")
	}
	if cfg.WallClockCapMin == 0 {
		return nil, errors.New("budgetenforcer: wall-clock cap must be > 0")
	}
	if cfg.RetryCapPerSubgoal == 0 {
		return nil, errors.New("budgetenforcer: retry cap must be > 0")
	}
	now := cfg.Now
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	return &Enforcer{
		taskID:          cfg.TaskID,
		costCapUSD:      cfg.CostCapUSD,
		wallClockCapSec: uint64(cfg.WallClockCapMin) * 60,
		retryCap:        cfg.RetryCapPerSubgoal,
		stepsCap:        cfg.StepsCap,
		startedAt:       now(),
	}, nil
}

// Snapshot returns the current Budget without mutating state.
func (e *Enforcer) Snapshot() *cruciblev1.Budget {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.snapshotLocked()
}

func (e *Enforcer) snapshotLocked() *cruciblev1.Budget {
	return &cruciblev1.Budget{
		SpentUsd:             e.spentUSD,
		CapUsd:               e.costCapUSD,
		StepsUsed:            e.stepsUsed,
		StepsCap:             e.stepsCap,
		WallClockUsedSeconds: e.wallClockUsedSec,
		WallClockCapSeconds:  e.wallClockCapSec,
		RetriesUsed:          e.retriesUsed,
		RetryCap:             e.retryCap,
	}
}

// Charge adds usd to the task's spend. Returns BudgetExceeded once the cap is
// reached. Once frozen, all subsequent Charge calls return the same error.
//
// Invariant: spentUSD is monotonically non-decreasing.
func (e *Enforcer) Charge(usd float64) error {
	if usd < 0 {
		return fmt.Errorf("budgetenforcer: negative charge %v", usd)
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.frozen {
		return frozenError(e.frozenReason, e.snapshotLocked())
	}
	e.spentUSD += usd
	if !e.warnedAt80Percent && e.spentUSD >= 0.8*e.costCapUSD {
		e.warnedAt80Percent = true
	}
	if e.spentUSD >= e.costCapUSD {
		e.frozen = true
		e.frozenReason = "budget_exceeded"
		return cruciblev1.NewError(
			cruciblev1.ErrBudgetExceeded,
			fmt.Sprintf("budget exceeded: $%.4f / $%.4f", e.spentUSD, e.costCapUSD),
			"call twin.plan.requestReplan to extend or replan",
			false,
		)
	}
	return nil
}

// IncrementRetry bumps the per-task retry counter. Returns RetryLimitExceeded
// once the cap is reached.
func (e *Enforcer) IncrementRetry() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.frozen {
		return frozenError(e.frozenReason, e.snapshotLocked())
	}
	e.retriesUsed++
	if e.retriesUsed >= e.retryCap {
		e.frozen = true
		e.frozenReason = "retry_limit_exceeded"
		return cruciblev1.NewError(
			cruciblev1.ErrRetryLimitExceeded,
			fmt.Sprintf("retry cap reached: %d / %d", e.retriesUsed, e.retryCap),
			"halt and ask for human input via twin.plan.requestReplan",
			false,
		)
	}
	return nil
}

// IncrementStep bumps the step counter. Returns nil if stepsCap is 0 (unlimited).
func (e *Enforcer) IncrementStep() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.frozen {
		return frozenError(e.frozenReason, e.snapshotLocked())
	}
	e.stepsUsed++
	if e.stepsCap > 0 && e.stepsUsed > e.stepsCap {
		e.frozen = true
		e.frozenReason = "steps_exceeded"
		return cruciblev1.NewError(
			cruciblev1.ErrBudgetExceeded,
			fmt.Sprintf("steps cap exceeded: %d / %d", e.stepsUsed, e.stepsCap),
			"step cap is a defensive bound; re-plan to widen",
			false,
		)
	}
	return nil
}

// Tick advances the wall clock based on elapsed real time since startedAt.
// Returns WallClockExceeded once the cap is reached.
func (e *Enforcer) Tick(now time.Time) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.frozen {
		return frozenError(e.frozenReason, e.snapshotLocked())
	}
	elapsed := uint64(now.Sub(e.startedAt).Seconds())
	if elapsed > e.wallClockUsedSec {
		e.wallClockUsedSec = elapsed
	}
	if e.wallClockUsedSec >= e.wallClockCapSec {
		e.frozen = true
		e.frozenReason = "wall_clock_exceeded"
		return cruciblev1.NewError(
			cruciblev1.ErrWallClockExceeded,
			fmt.Sprintf("wall-clock cap reached: %ds / %ds", e.wallClockUsedSec, e.wallClockCapSec),
			"split the task or extend wall_clock_cap_min",
			false,
		)
	}
	return nil
}

// Frozen returns true if the enforcer has tripped a cap.
func (e *Enforcer) Frozen() (bool, string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.frozen, e.frozenReason
}

// WarnedAt80Percent returns true once cost spend crosses 80% of the cap.
func (e *Enforcer) WarnedAt80Percent() bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.warnedAt80Percent
}

// Reset returns the enforcer to a fresh state with the same caps. Used when
// the user explicitly approves a replan.
func (e *Enforcer) Reset(now time.Time) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.spentUSD = 0
	e.retriesUsed = 0
	e.stepsUsed = 0
	e.wallClockUsedSec = 0
	e.frozen = false
	e.frozenReason = ""
	e.warnedAt80Percent = false
	e.startedAt = now
}

// frozenError builds the CrucibleError returned on attempts to spend more after a freeze.
func frozenError(reason string, b *cruciblev1.Budget) *cruciblev1.CrucibleError {
	var code cruciblev1.ErrorCode
	switch reason {
	case "budget_exceeded":
		code = cruciblev1.ErrBudgetExceeded
	case "retry_limit_exceeded":
		code = cruciblev1.ErrRetryLimitExceeded
	case "wall_clock_exceeded":
		code = cruciblev1.ErrWallClockExceeded
	case "steps_exceeded":
		code = cruciblev1.ErrBudgetExceeded
	default:
		code = cruciblev1.ErrBudgetExceeded
	}
	return cruciblev1.NewError(code, "enforcer frozen: "+reason, "task has hit a cap; re-plan to continue", false)
}

// Registry holds Enforcers per task ID so the API layer can look them up.
type Registry struct {
	mu sync.RWMutex
	m  map[string]*Enforcer
}

// NewRegistry returns an empty registry.
func NewRegistry() *Registry {
	return &Registry{m: make(map[string]*Enforcer)}
}

// Register associates an enforcer with a task id.
func (r *Registry) Register(taskID string, e *Enforcer) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.m[taskID] = e
}

// Get returns the enforcer for a task id, or nil.
func (r *Registry) Get(taskID string) *Enforcer {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.m[taskID]
}

// Remove deletes a registry entry (e.g. after task completion).
func (r *Registry) Remove(taskID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.m, taskID)
}
