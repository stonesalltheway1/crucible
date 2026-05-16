package migration

import (
	"context"
	"strings"
	"sync"
	"time"
)

// FakeDriver is the in-memory driver used in tests + dev mode. It records
// every Apply / IntegrityCheck / Rollback call.
type FakeDriver struct {
	mu       sync.Mutex
	applies  []FakeApply
	checks   []IntegrityCheck
	rollback string
	// Optional: fail integrity check N.
	IntegrityFailIdx *int
	ApplyFailScope   *Scope
}

// FakeApply is a recorded apply.
type FakeApply struct {
	Scope Scope
	SQL   string
}

// NewFakeDriver builds a FakeDriver.
func NewFakeDriver() *FakeDriver { return &FakeDriver{} }

// Name implements Driver.Name.
func (d *FakeDriver) Name() string { return "fake" }

// Apply implements Driver.Apply.
func (d *FakeDriver) Apply(_ context.Context, scope Scope, sqlText string) (Result, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.ApplyFailScope != nil && *d.ApplyFailScope == scope {
		return Result{Error: "forced failure on " + string(scope)}, &fakeError{string(scope) + " apply forced to fail"}
	}
	d.applies = append(d.applies, FakeApply{Scope: scope, SQL: sqlText})
	return Result{Affected: int64(strings.Count(sqlText, ";")), DurationMs: 1}, nil
}

// IntegrityCheck implements Driver.IntegrityCheck.
func (d *FakeDriver) IntegrityCheck(_ context.Context, _ Scope, c IntegrityCheck) (Result, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.checks = append(d.checks, c)
	if d.IntegrityFailIdx != nil && *d.IntegrityFailIdx == len(d.checks)-1 {
		return Result{Affected: c.ExpectMax + 1, DurationMs: 1}, nil
	}
	// Hit the expected range.
	want := c.ExpectMin
	if want == 0 {
		want = 1
	}
	return Result{Affected: want, DurationMs: 1}, nil
}

// Rollback implements Driver.Rollback.
func (d *FakeDriver) Rollback(_ context.Context, _ Scope, downSQL string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.rollback = downSQL
	return nil
}

// Applies returns recorded applies (test helper).
func (d *FakeDriver) Applies() []FakeApply {
	d.mu.Lock()
	defer d.mu.Unlock()
	out := make([]FakeApply, len(d.applies))
	copy(out, d.applies)
	return out
}

// RollbackSQL returns the last rollback SQL.
func (d *FakeDriver) RollbackSQL() string {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.rollback
}

type fakeError struct{ s string }

func (e *fakeError) Error() string { return e.s }

var _ time.Time
