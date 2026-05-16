package api

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/crucible/attestation"
	"github.com/crucible/control-plane/internal/budgetenforcer"
	"github.com/crucible/control-plane/internal/modelrouter"
	"github.com/crucible/control-plane/internal/planbuilder"
	"github.com/crucible/control-plane/internal/store"
	"github.com/crucible/control-plane/internal/taskrouter"
	cruciblev1 "github.com/crucible/sdk-go/crucible/v1"
)

func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	dir := t.TempDir()
	signer, err := attestation.NewLocalEd25519Signer(filepath.Join(dir, "keys"))
	if err != nil {
		t.Fatal(err)
	}
	pub, err := attestation.NewLocalJournalPublisher(filepath.Join(dir, "journal.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	svc, err := attestation.NewService(signer, pub)
	if err != nil {
		t.Fatal(err)
	}

	mr := modelrouter.NewRouter() // no vendors → heuristic + fallback path
	router := taskrouter.New(mr, "")
	pb := planbuilder.New(mr, svc, "")
	enforcers := budgetenforcer.NewRegistry()
	tasks := store.New()

	s := &Server{
		Store:         tasks,
		Router:        router,
		PlanBuilder:   pb,
		Budgets:       enforcers,
		Attestation:   svc,
		Logger:        slog.New(slog.NewTextHandler(io.Discard, nil)),
		DefaultTenant: "ten_test",
		Version:       "2026.06.0-phase1-test",
	}
	return httptest.NewServer(s.Handler())
}

func postJSON(t *testing.T, url string, body any) *http.Response {
	t.Helper()
	b, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.Post(url, "application/json", bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

func TestHealth(t *testing.T) {
	srv := newTestServer(t)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/healthz")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("status: %d", resp.StatusCode)
	}
	var got map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if got["status"] != "ok" {
		t.Fatalf("expected ok, got %v", got)
	}
	if got["stub_twin_runtime"] != true {
		t.Fatalf("expected stub_twin_runtime=true in Phase 1")
	}
}

func TestSubmit_RequiresDescription(t *testing.T) {
	srv := newTestServer(t)
	defer srv.Close()
	resp := postJSON(t, srv.URL+"/v1/tasks", map[string]any{"repo": "x"})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestSubmit_RejectsInvalidJSON(t *testing.T) {
	srv := newTestServer(t)
	defer srv.Close()
	resp, err := http.Post(srv.URL+"/v1/tasks", "application/json", bytes.NewReader([]byte(`not json`)))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

// TestEndToEnd_SubmitPlanApproveBudget exercises the full Phase-1 lifecycle
// without any external LLM call: submit → server uses heuristic + fallback plan
// → list → get → approve → budget snapshot.
func TestEndToEnd_SubmitPlanApproveBudget(t *testing.T) {
	srv := newTestServer(t)
	defer srv.Close()

	// 1. Submit.
	resp := postJSON(t, srv.URL+"/v1/tasks", map[string]any{
		"description": "Add a refund webhook handler",
		"repo":        "github.com/acme/payments",
		"base_sha":    "abc1234",
	})
	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		t.Fatalf("submit returned %d: %s", resp.StatusCode, body)
	}
	var submitOut struct {
		Task *cruciblev1.Task `json:"task"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&submitOut); err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	task := submitOut.Task
	if task == nil || task.ID == "" {
		t.Fatal("server did not return a task")
	}
	if task.Status != cruciblev1.TaskStatusAwaitingApproval {
		t.Fatalf("expected awaiting_approval, got %s", task.Status)
	}
	if task.Plan == nil {
		t.Fatal("expected plan attached")
	}
	if task.Plan.PlanHash == "" {
		t.Fatal("expected plan_hash to be set")
	}
	if task.Routing == nil {
		t.Fatal("expected routing")
	}
	if task.Routing.ExecutorVendor == task.Routing.VerifierVendor {
		t.Fatalf("ADR-002 violation: same-vendor pairing")
	}

	// 2. List.
	listResp, err := http.Get(srv.URL + "/v1/tasks?tenant_id=ten_test")
	if err != nil {
		t.Fatal(err)
	}
	var listOut struct{ Tasks []*cruciblev1.Task `json:"tasks"` }
	_ = json.NewDecoder(listResp.Body).Decode(&listOut)
	listResp.Body.Close()
	if len(listOut.Tasks) != 1 {
		t.Fatalf("expected 1 task in list, got %d", len(listOut.Tasks))
	}

	// 3. Get.
	getResp, err := http.Get(srv.URL + "/v1/tasks/" + task.ID)
	if err != nil {
		t.Fatal(err)
	}
	var getOut struct{ Task *cruciblev1.Task `json:"task"` }
	_ = json.NewDecoder(getResp.Body).Decode(&getOut)
	getResp.Body.Close()
	if getOut.Task.ID != task.ID {
		t.Fatalf("get returned wrong task")
	}

	// 4. Approve.
	approveResp := postJSON(t, srv.URL+"/v1/tasks/"+task.ID+"/approve", map[string]any{
		"plan_hash": task.Plan.PlanHash,
	})
	if approveResp.StatusCode != 200 {
		body, _ := io.ReadAll(approveResp.Body)
		approveResp.Body.Close()
		t.Fatalf("approve returned %d: %s", approveResp.StatusCode, body)
	}
	var approveOut struct {
		Task     *cruciblev1.Task         `json:"task"`
		Approval *cruciblev1.PlanApproval `json:"approval"`
	}
	if err := json.NewDecoder(approveResp.Body).Decode(&approveOut); err != nil {
		t.Fatal(err)
	}
	approveResp.Body.Close()
	if approveOut.Task.Status != cruciblev1.TaskStatusApproved {
		t.Fatalf("expected approved status, got %s", approveOut.Task.Status)
	}
	if approveOut.Approval == nil || approveOut.Approval.AttestationID == "" {
		t.Fatal("expected approval with attestation id")
	}

	// 5. Approving a stale plan_hash must fail.
	resp = postJSON(t, srv.URL+"/v1/tasks/"+task.ID+"/approve", map[string]any{
		"plan_hash": "not-the-right-hash",
	})
	if resp.StatusCode != http.StatusBadRequest {
		resp.Body.Close()
		t.Fatalf("expected 400 on stale plan_hash, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	// 6. Budget snapshot.
	budResp, err := http.Get(srv.URL + "/v1/tasks/" + task.ID + "/budget")
	if err != nil {
		t.Fatal(err)
	}
	var budOut struct{ Budget *cruciblev1.Budget `json:"budget"` }
	_ = json.NewDecoder(budResp.Body).Decode(&budOut)
	budResp.Body.Close()
	if budOut.Budget == nil || budOut.Budget.CapUsd == 0 {
		t.Fatalf("expected budget snapshot, got %+v", budOut.Budget)
	}
}

func TestGetTask_NotFound(t *testing.T) {
	srv := newTestServer(t)
	defer srv.Close()
	resp, err := http.Get(srv.URL + "/v1/tasks/nope")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}

func TestReject(t *testing.T) {
	srv := newTestServer(t)
	defer srv.Close()
	// Submit first.
	resp := postJSON(t, srv.URL+"/v1/tasks", map[string]any{"description": "x"})
	var out struct{ Task *cruciblev1.Task `json:"task"` }
	_ = json.NewDecoder(resp.Body).Decode(&out)
	resp.Body.Close()

	rejectResp := postJSON(t, srv.URL+"/v1/tasks/"+out.Task.ID+"/reject", map[string]any{
		"reason": "scope-creep",
	})
	defer rejectResp.Body.Close()
	if rejectResp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", rejectResp.StatusCode)
	}
	var rOut struct {
		Task      *cruciblev1.Task          `json:"task"`
		Rejection *cruciblev1.PlanRejection `json:"rejection"`
	}
	_ = json.NewDecoder(rejectResp.Body).Decode(&rOut)
	if rOut.Task.Status != cruciblev1.TaskStatusRejected {
		t.Fatalf("expected rejected, got %s", rOut.Task.Status)
	}
}

// TestIntegration_RealHaiku4_5 is the brief's mandated real-LLM integration test.
//
// Runs only when ANTHROPIC_API_KEY is set. Submits a tiny task and asserts the
// classifier returned a structured complexity. If the env var is missing, the
// test fails with a clear message rather than silently passing — per the brief.
func TestIntegration_RealHaiku4_5(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	key := os.Getenv("ANTHROPIC_API_KEY")
	if key == "" {
		t.Skip("ANTHROPIC_API_KEY not set; skipping the real-LLM integration test. " +
			"Set the env var to exercise this path end-to-end.")
	}

	dir := t.TempDir()
	signer, err := attestation.NewLocalEd25519Signer(filepath.Join(dir, "keys"))
	if err != nil {
		t.Fatal(err)
	}
	pub, err := attestation.NewLocalJournalPublisher(filepath.Join(dir, "journal.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	svc, _ := attestation.NewService(signer, pub)

	mr := modelrouter.NewRouter(modelrouter.NewAnthropicClientFromEnv())
	if len(mr.Vendors()) == 0 {
		t.Fatal("ANTHROPIC_API_KEY set but Anthropic client failed to initialize")
	}
	router := taskrouter.New(mr, "claude-haiku-4-5")

	// One cheap classifier call.
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cls, err := router.Classify(ctx, "Add a one-line typo fix to README.md")
	if err != nil {
		t.Fatalf("Classify: %v", err)
	}
	if cls.Complexity == "" {
		t.Fatal("real classifier returned empty complexity")
	}
	t.Logf("real Haiku 4.5 classified the task as %q (critical_score=%d)",
		cls.Complexity, cls.CriticalScore)

	// Wire a minimal server and submit a task end-to-end.
	s := &Server{
		Store:         store.New(),
		Router:        router,
		PlanBuilder:   planbuilder.New(mr, svc, "claude-haiku-4-5"), // use Haiku for plan too, cheapest
		Budgets:       budgetenforcer.NewRegistry(),
		Attestation:   svc,
		Logger:        slog.New(slog.NewTextHandler(io.Discard, nil)),
		DefaultTenant: "ten_int",
		Version:       "2026.06.0-phase1-integration",
	}
	srv := httptest.NewServer(s.Handler())
	defer srv.Close()

	resp := postJSON(t, srv.URL+"/v1/tasks", map[string]any{
		"description": "Add a one-line typo fix to README.md",
		"repo":        "github.com/example/repo",
	})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("submit returned %d: %s", resp.StatusCode, body)
	}
	var out struct{ Task *cruciblev1.Task `json:"task"` }
	_ = json.NewDecoder(resp.Body).Decode(&out)
	if out.Task == nil || out.Task.Plan == nil {
		t.Fatal("expected task + plan back from real LLM call")
	}
	t.Logf("real Plan from Haiku 4.5: %d steps, $%.2f estimate, hash=%s",
		len(out.Task.Plan.Steps), out.Task.Plan.EstimatedCostUsd, out.Task.Plan.PlanHash[:12])
}
