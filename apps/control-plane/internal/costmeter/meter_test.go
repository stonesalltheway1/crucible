package costmeter

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/crucible/control-plane/internal/budgetenforcer"
	"github.com/crucible/control-plane/internal/modelrouter"
)

func newMeter(t *testing.T, registry *budgetenforcer.Registry) *Meter {
	t.Helper()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	m, err := New(logger, t.TempDir(), registry)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(m.Close)
	return m
}

func TestRecordResponse_WritesLogAndChargesEnforcer(t *testing.T) {
	reg := budgetenforcer.NewRegistry()
	enf, _ := budgetenforcer.New(budgetenforcer.Config{
		TaskID: "task_M", CostCapUSD: 10, WallClockCapMin: 60, RetryCapPerSubgoal: 3,
	})
	reg.Register("task_M", enf)

	m := newMeter(t, reg)
	resp := &modelrouter.Response{
		Model: "claude-haiku-4-5",
		Usage: modelrouter.Usage{InputTokensFresh: 1000, OutputTokens: 500},
		Latency: 50 * time.Millisecond,
	}
	ev, err := m.RecordResponse(context.Background(), "task_M", "step_1", "ten_a", resp)
	if err != nil {
		t.Fatalf("RecordResponse: %v", err)
	}
	if ev.CostUSD <= 0 {
		t.Fatalf("expected positive cost, got %v", ev.CostUSD)
	}
	// Enforcer should have debited.
	b := enf.Snapshot()
	if b.SpentUsd == 0 {
		t.Fatal("expected enforcer to be charged")
	}

	// Log file should have one line.
	p := m.LogPath("task_M")
	f, err := os.Open(p)
	if err != nil {
		t.Fatalf("open log: %v", err)
	}
	defer f.Close()
	count := 0
	for sc := bufio.NewScanner(f); sc.Scan(); count++ {
		var e Event
		if err := json.Unmarshal(sc.Bytes(), &e); err != nil {
			t.Fatalf("log line not valid JSON: %v", err)
		}
		if e.TaskID != "task_M" {
			t.Fatalf("wrong task id in log: %s", e.TaskID)
		}
	}
	if count != 1 {
		t.Fatalf("expected 1 log line, got %d", count)
	}
}

func TestRecordResponse_NilRejected(t *testing.T) {
	m := newMeter(t, nil)
	if _, err := m.RecordResponse(context.Background(), "t", "", "", nil); err == nil {
		t.Fatal("expected error on nil response")
	}
}

func TestRecordResponse_NoEnforcerNoError(t *testing.T) {
	m := newMeter(t, budgetenforcer.NewRegistry())
	resp := &modelrouter.Response{
		Model: "claude-haiku-4-5",
		Usage: modelrouter.Usage{InputTokensFresh: 1000, OutputTokens: 500},
	}
	if _, err := m.RecordResponse(context.Background(), "task_unknown", "", "", resp); err != nil {
		t.Fatalf("expected no error without enforcer, got %v", err)
	}
}

func TestLogPath_DerivedFromDir(t *testing.T) {
	dir := t.TempDir()
	m, err := New(slog.New(slog.NewTextHandler(io.Discard, nil)), dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	want := filepath.Join(dir, "task_X.jsonl")
	if got := m.LogPath("task_X"); got != want {
		t.Fatalf("LogPath: got %s want %s", got, want)
	}
}
