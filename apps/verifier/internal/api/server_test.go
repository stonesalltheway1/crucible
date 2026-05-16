package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	cruciblev1 "github.com/crucible/sdk-go/crucible/v1"
	"github.com/crucible/verifier/internal/criticalpath"
	"github.com/crucible/verifier/internal/dispatcher"
	"github.com/crucible/verifier/internal/rubric"
	"github.com/crucible/verifier/internal/verification"
	"github.com/crucible/verifier/pkg/testreport"
)

// stubPool returns one passing report per (lang, tier).
type stubPool struct{}

func (s *stubPool) Submit(_ context.Context, k dispatcher.RunnerKind, _ *verification.VerificationRequest) (*testreport.TestReport, error) {
	r := &testreport.TestReport{
		SchemaVersion: testreport.SchemaVersion,
		Tier:          k.Tier, Language: k.Language, Framework: "stub",
		Verdict: testreport.VerdictPassed, Passed: true,
	}
	switch k.Tier {
	case testreport.TierMutation:
		r.Mutation = &testreport.MutationStats{Killed: 5, Survived: 0, Total: 5, Score: 1, Threshold: 0.85, DiffScoped: true}
	case testreport.TierPBT:
		r.PBT = &testreport.PBTStats{Iterations: 10000, IterationsMin: 10000}
	}
	return r, nil
}
func (s *stubPool) Health() error { return nil }

type stubT4 struct{}

func (s *stubT4) Verify(_ context.Context, req *verification.VerificationRequest) (*testreport.TestReport, error) {
	return &testreport.TestReport{
		SchemaVersion: testreport.SchemaVersion,
		Tier:          testreport.TierHonestCI, Language: testreport.LangPolyglot,
		Verdict: testreport.VerdictPassed, Passed: true,
		HonestCI: &testreport.HonestCIStats{
			BuilderID: "test", BitIdentical: true, SLSALevel: 3,
			ExecutorRebuildHash: "0xa", VerifierRebuildHash: "0xa",
			ScrubberAuditOK: true,
		},
	}, nil
}

func newTestServer() *Server {
	pool := &stubPool{}
	t4 := &stubT4{}
	judge := rubric.NewJudge(rubric.NewHeuristicClient())
	cls := criticalpath.NewClassifier(criticalpath.NewPathPatternFeaturizer())
	disp := dispatcher.New(pool, nil, t4, judge, cls)
	disp.Now = func() time.Time { return time.Unix(1747300000, 0).UTC() }
	return &Server{Dispatcher: disp, Version: "phase4-test"}
}

func newValidBody() []byte {
	req := &verification.VerificationRequest{
		TaskID:   "task_api",
		TenantID: "ten",
		BaseSHA:  "abc",
		Diff: cruciblev1.Diff{Files: []cruciblev1.FileChange{
			{Path: "x.py", Action: cruciblev1.ActionModify, ContentSha256: "0xa"},
		}},
		Routing: cruciblev1.Routing{
			ExecutorVendor: "anthropic", VerifierVendor: "google",
			ExecutorModel: "claude-opus-4-7", VerifierModel: "gemini-3.1-pro",
		},
		Languages:         []string{"python"},
		ExecutorSandboxID: "sb_executor",
		PerTaskSignals: verification.TaskSignals{
			SelfHostAvailable:     true,
			ScrubberFiredOnAllPII: true,
		},
	}
	b, _ := json.Marshal(req)
	return b
}

func TestHandleBundle_passing(t *testing.T) {
	s := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/v1/twin/verify/bundle", bytes.NewReader(newValidBody()))
	s.Handler().ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("status %d body=%s", rec.Code, rec.Body.String())
	}
	var resp verification.VerificationResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if resp.Approval == nil {
		t.Fatalf("expected Approval; got Rejection=%+v", resp.Rejection)
	}
}

func TestHandleBundle_rejectsReasoningLeak(t *testing.T) {
	body := []byte(`{
		"task_id":"x","tenant_id":"y","base_sha":"abc",
		"diff": {"files":[{"path":"a.py","action":"modify","content_sha256":"0xa"}]},
		"routing":{"executor_vendor":"anthropic","verifier_vendor":"google"},
		"executor_sandbox_id":"sb",
		"reasoning":"the agent thought about it deeply"
	}`)
	s := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/v1/twin/verify/bundle", bytes.NewReader(body))
	s.Handler().ServeHTTP(rec, req)
	if rec.Code != 400 {
		t.Fatalf("expected 400 for reasoning leak; got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestHandleBundle_rejectsSameFamily(t *testing.T) {
	body := []byte(`{
		"task_id":"x","tenant_id":"y","base_sha":"abc",
		"diff": {"files":[{"path":"a.py","action":"modify","content_sha256":"0xa"}]},
		"routing":{"executor_vendor":"anthropic","verifier_vendor":"anthropic"},
		"executor_sandbox_id":"sb"
	}`)
	s := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/v1/twin/verify/bundle", bytes.NewReader(body))
	s.Handler().ServeHTTP(rec, req)
	if rec.Code != 400 {
		t.Fatalf("expected 400 for same-family routing; got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestHealthz(t *testing.T) {
	s := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/healthz", nil)
	s.Handler().ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("healthz status %d", rec.Code)
	}
}

func TestHandleAuditOnly_passes(t *testing.T) {
	s := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/v1/twin/verify/audit", bytes.NewReader(newValidBody()))
	req.Header.Set("Content-Type", "application/json")
	s.Handler().ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("audit status %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestHandleBundle_disallowsGet(t *testing.T) {
	s := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/twin/verify/bundle", nil)
	s.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405; got %d", rec.Code)
	}
}
