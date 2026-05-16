package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	cruciblev1 "github.com/crucible/sdk-go/crucible/v1"
)

func newFakeServer(t *testing.T, handlers map[string]http.HandlerFunc) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	for pattern, h := range handlers {
		mux.HandleFunc(pattern, h)
	}
	return httptest.NewServer(mux)
}

func TestVersionCommand(t *testing.T) {
	buf := new(bytes.Buffer)
	root := NewRoot()
	root.SetOut(buf)
	root.SetArgs([]string{"version"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), cliVersion) {
		t.Fatalf("expected %s in output, got %q", cliVersion, buf.String())
	}
}

func TestHealthCommand_PrintsServerVersion(t *testing.T) {
	srv := newFakeServer(t, map[string]http.HandlerFunc{
		"GET /healthz": func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"status":  "ok",
				"version": "2026.06.0-phase1-fake",
				"now":     time.Now().Format(time.RFC3339),
			})
		},
	})
	defer srv.Close()

	buf := new(bytes.Buffer)
	root := NewRoot()
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"--endpoint", srv.URL, "health"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "ok") {
		t.Fatalf("expected ok in output, got %q", buf.String())
	}
	if !strings.Contains(buf.String(), "2026.06.0-phase1-fake") {
		t.Fatalf("expected server version in output, got %q", buf.String())
	}
}

func TestTaskNew_RequiresDescription(t *testing.T) {
	buf := new(bytes.Buffer)
	root := NewRoot()
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"task", "new"})
	if err := root.Execute(); err == nil {
		t.Fatal("expected error on missing --description")
	}
}

func TestTaskNew_PostsToServer(t *testing.T) {
	var got map[string]any
	srv := newFakeServer(t, map[string]http.HandlerFunc{
		"POST /v1/tasks": func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewDecoder(r.Body).Decode(&got)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(201)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"task": &cruciblev1.Task{
					ID:          "task_X",
					TenantID:    "ten_test",
					Description: "x",
					Status:      cruciblev1.TaskStatusAwaitingApproval,
				},
			})
		},
	})
	defer srv.Close()

	buf := new(bytes.Buffer)
	root := NewRoot()
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"--endpoint", srv.URL, "task", "new", "--description", "x"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	if got["description"] != "x" {
		t.Fatalf("server didn't receive description: %+v", got)
	}
	if !strings.Contains(buf.String(), "task_X") {
		t.Fatalf("expected task id in output, got %q", buf.String())
	}
}

func TestTaskGet_PrintsPlan(t *testing.T) {
	srv := newFakeServer(t, map[string]http.HandlerFunc{
		"GET /v1/tasks/task_X": func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"task": &cruciblev1.Task{
					ID:     "task_X",
					Status: cruciblev1.TaskStatusAwaitingApproval,
					Plan: &cruciblev1.Plan{
						Description:          "test plan",
						EstimatedCostUsd:     1.5,
						EstimatedDurationMin: 10,
						Complexity:           cruciblev1.ComplexityStandard,
						PlanHash:             strings.Repeat("a", 64),
						RetryBudgetPerStep:   3,
						WallClockBudgetMin:   30,
						Steps: []cruciblev1.PlanStep{
							{Ordinal: 1, Description: "step one", RetryBudget: 3},
						},
					},
				},
			})
		},
	})
	defer srv.Close()

	buf := new(bytes.Buffer)
	root := NewRoot()
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"--endpoint", srv.URL, "plan", "show", "task_X"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, want := range []string{"test plan", "step one", "Plan hash:", "Estimate:"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected %q in output, got %q", want, out)
		}
	}
}

func TestPlanReject_RequiresReason(t *testing.T) {
	buf := new(bytes.Buffer)
	root := NewRoot()
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"plan", "reject", "task_X"})
	if err := root.Execute(); err == nil {
		t.Fatal("expected error on missing --reason")
	}
}

func TestBudgetShow_PrintsSpend(t *testing.T) {
	srv := newFakeServer(t, map[string]http.HandlerFunc{
		"GET /v1/tasks/task_X/budget": func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"budget": &cruciblev1.Budget{
					SpentUsd:             0.25,
					CapUsd:               1.5,
					RetriesUsed:          1,
					RetryCap:             3,
					WallClockUsedSeconds: 30,
					WallClockCapSeconds:  3600,
				},
			})
		},
	})
	defer srv.Close()

	buf := new(bytes.Buffer)
	root := NewRoot()
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"--endpoint", srv.URL, "budget", "show", "task_X"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "$0.2500") {
		t.Fatalf("expected spend in output, got %q", out)
	}
}
