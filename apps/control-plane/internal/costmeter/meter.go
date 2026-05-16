// Package costmeter records every LLM call's USD cost into structured spans.
//
// In Phase 1 the meter writes JSON-line events to a per-task log file and also
// debits the per-task Enforcer so caps fire in real time. Phase 2 wires the
// OTel exporter so spans land in Honeycomb/Tempo and aggregate into the
// ClickHouse-backed cost dashboard documented in
// docs/02-engineering/observability.md.
package costmeter

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/crucible/control-plane/internal/budgetenforcer"
	"github.com/crucible/control-plane/internal/modelrouter"
)

// Event is one cost-telemetry record.
type Event struct {
	TaskID              string    `json:"task_id"`
	StepID              string    `json:"step_id,omitempty"`
	TenantID            string    `json:"tenant_id,omitempty"`
	ModelVendor         string    `json:"model.vendor"`
	ModelID             string    `json:"model.id"`
	ModelTier           int       `json:"model.tier"`
	TokensInputFresh    int       `json:"tokens.input.fresh"`
	TokensInputCached   int       `json:"tokens.input.cached"`
	TokensOutput        int       `json:"tokens.output"`
	TokensThinking      int       `json:"tokens.thinking"`
	CostUSD             float64   `json:"cost.usd"`
	CacheHit            bool      `json:"cache.hit"`
	LatencyMs           int64     `json:"latency_ms"`
	At                  time.Time `json:"at"`
}

// Meter records cost events. Logs may be inspected after a run; the running
// enforcer registry receives every charge so caps fire in real time.
type Meter struct {
	logger    *slog.Logger
	dir       string
	enforcers *budgetenforcer.Registry

	mu    sync.Mutex
	files map[string]*os.File
}

// New constructs a Meter. dir defaults to ~/.crucible/costlog/.
func New(logger *slog.Logger, dir string, enforcers *budgetenforcer.Registry) (*Meter, error) {
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("costmeter: locate home dir: %w", err)
		}
		dir = filepath.Join(home, ".crucible", "costlog")
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("costmeter: mkdir: %w", err)
	}
	return &Meter{
		logger:    logger,
		dir:       dir,
		enforcers: enforcers,
		files:     map[string]*os.File{},
	}, nil
}

// RecordResponse builds an Event from a modelrouter Response and the model id,
// writes it to the task's log, charges the enforcer (if registered), and emits
// a structured log line.
//
// Returns the *cruciblev1.CrucibleError produced by enforcer.Charge() when
// the budget cap fires — callers must surface this to the caller; do NOT
// silently retry.
func (m *Meter) RecordResponse(ctx context.Context, taskID, stepID, tenantID string, resp *modelrouter.Response) (*Event, error) {
	if resp == nil {
		return nil, errors.New("costmeter: nil response")
	}
	cost := modelrouter.EstimateCostUSD(resp.Model, resp.Usage)
	spec, _ := modelrouter.Lookup(resp.Model)

	ev := &Event{
		TaskID:            taskID,
		StepID:            stepID,
		TenantID:          tenantID,
		ModelVendor:       string(spec.Vendor),
		ModelID:           resp.Model,
		ModelTier:         int(spec.Tier),
		TokensInputFresh:  resp.Usage.InputTokensFresh,
		TokensInputCached: resp.Usage.CacheReadTokens,
		TokensOutput:      resp.Usage.OutputTokens,
		TokensThinking:    resp.Usage.ThinkingTokens,
		CostUSD:           cost,
		CacheHit:          resp.CacheHit,
		LatencyMs:         resp.Latency.Milliseconds(),
		At:                time.Now().UTC(),
	}

	if err := m.write(taskID, ev); err != nil {
		m.logger.WarnContext(ctx, "costmeter write failed", "err", err, "task", taskID)
	}
	m.logger.InfoContext(ctx, "llm-call",
		"task", taskID, "model", resp.Model, "cost", cost,
		"in_fresh", resp.Usage.InputTokensFresh, "in_cached", resp.Usage.CacheReadTokens,
		"out", resp.Usage.OutputTokens,
	)
	if m.enforcers != nil {
		if enf := m.enforcers.Get(taskID); enf != nil {
			if err := enf.Charge(cost); err != nil {
				return ev, err
			}
		}
	}
	return ev, nil
}

func (m *Meter) write(taskID string, ev *Event) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	f, ok := m.files[taskID]
	if !ok {
		p := filepath.Join(m.dir, taskID+".jsonl")
		var err error
		f, err = os.OpenFile(p, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
		if err != nil {
			return fmt.Errorf("costmeter: open log: %w", err)
		}
		m.files[taskID] = f
	}
	b, err := json.Marshal(ev)
	if err != nil {
		return err
	}
	if _, err := f.Write(append(b, '\n')); err != nil {
		return err
	}
	return nil
}

// Close flushes & closes every open per-task log file.
func (m *Meter) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for id, f := range m.files {
		_ = f.Close()
		delete(m.files, id)
	}
}

// LogPath returns the path of the per-task log file for inspection.
func (m *Meter) LogPath(taskID string) string {
	return filepath.Join(m.dir, taskID+".jsonl")
}
