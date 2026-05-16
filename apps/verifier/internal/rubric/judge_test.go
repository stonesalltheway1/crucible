package rubric

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	cruciblev1 "github.com/crucible/sdk-go/crucible/v1"
	"github.com/crucible/verifier/internal/verification"
	"github.com/crucible/verifier/pkg/testreport"
)

func newRequest() *verification.VerificationRequest {
	return &verification.VerificationRequest{
		TaskID:   "task_01",
		TenantID: "ten_acme",
		BaseSHA:  "abc",
		Diff: cruciblev1.Diff{
			Files: []cruciblev1.FileChange{
				{Path: "api/webhooks.ts", Action: cruciblev1.ActionModify, SizeBytes: 100},
			},
		},
		Routing: cruciblev1.Routing{
			ExecutorModel:  "claude-opus-4-7",
			ExecutorVendor: "anthropic",
			VerifierModel:  "gemini-3.1-pro",
			VerifierVendor: "google",
		},
		ExecutorSandboxID: "sb_executor_01",
		PerTaskSignals: verification.TaskSignals{
			SelfHostAvailable:        true,
			ScrubberFiredOnAllPII:    true,
			TapeDispositionHistogram: map[string]int{"hit-exact": 5},
		},
	}
}

// fakeLLM returns the body bytes; if vendor matches the executor's, the
// test should see an error from the Judge BEFORE reaching Call.
type fakeLLM struct {
	vendor   string
	model    string
	body     string
	callCount int
	err      error
}

func (f *fakeLLM) Vendor() string { return f.vendor }
func (f *fakeLLM) Model() string  { return f.model }
func (f *fakeLLM) Call(_ context.Context, _ CallRequest) (CallResponse, error) {
	f.callCount++
	if f.err != nil {
		return CallResponse{}, f.err
	}
	return CallResponse{
		Body:          []byte(f.body),
		OutputTokens:  len(f.body) / 4,
		CostUSD:       0.001,
		ModelLatency:  10 * time.Millisecond,
	}, nil
}

func goodLLMBody(score float64) string {
	out := Score{
		Score:      score,
		Threshold:  DefaultThreshold,
		Passed:     score >= DefaultThreshold,
		Confidence: 0.92,
		Subscores: map[string]float64{
			"diff_correctness":         score,
			"test_adequacy":            score,
			"spec_consistency":         score,
			"robustness":               score,
			"security_posture":         score,
			"trust_signal_alignment":   score,
		},
		RejectionReasons: []RejectionReason{},
	}
	b, _ := json.Marshal(out)
	return string(b)
}

func TestJudge_refusesSameFamily(t *testing.T) {
	req := newRequest()
	llm := &fakeLLM{vendor: "anthropic", model: "claude-opus-4-7", body: goodLLMBody(0.9)}
	j := NewJudge(llm)
	_, err := j.Score(context.Background(), req, nil)
	if err == nil {
		t.Fatalf("expected SameFamilyError")
	}
	var sfe *verification.SameFamilyError
	if !errors.As(err, &sfe) {
		t.Fatalf("expected SameFamilyError, got %T %v", err, err)
	}
	if llm.callCount != 0 {
		t.Fatalf("Judge called LLM even though same-family; calls=%d", llm.callCount)
	}
}

func TestJudge_respectsBlocklist(t *testing.T) {
	req := newRequest()
	llm := &fakeLLM{vendor: "google", model: "gemini-3.1-pro", body: goodLLMBody(0.9)}
	j := NewJudge(llm)
	j.VendorBlocklist = []string{"google"}
	_, err := j.Score(context.Background(), req, nil)
	if err == nil {
		t.Fatalf("expected blocklist refusal")
	}
}

func TestJudge_acceptsCrossFamily_passing(t *testing.T) {
	req := newRequest()
	llm := &fakeLLM{vendor: "google", model: "gemini-3.1-pro", body: goodLLMBody(0.92)}
	j := NewJudge(llm)
	s, err := j.Score(context.Background(), req, nil)
	if err != nil {
		t.Fatalf("Score: %v", err)
	}
	if !s.Passed {
		t.Fatalf("expected passed; got Score=%v reasons=%v", s.Score, s.RejectionReasons)
	}
	if s.JudgeVendor != "google" {
		t.Fatalf("vendor not recorded")
	}
}

func TestJudge_hardRejectsMissBlocked(t *testing.T) {
	req := newRequest()
	req.PerTaskSignals.TapeDispositionHistogram["miss-blocked"] = 1
	llm := &fakeLLM{vendor: "google", model: "gemini-3.1-pro", body: goodLLMBody(0.95)}
	j := NewJudge(llm)
	s, err := j.Score(context.Background(), req, nil)
	if err != nil {
		t.Fatal(err)
	}
	if s.Passed {
		t.Fatalf("miss-blocked tape must hard-reject; got Score=%v", s.Score)
	}
	if llm.callCount != 0 {
		t.Fatalf("hard rejection should bypass LLM; calls=%d", llm.callCount)
	}
	if !containsCategory(s.RejectionReasons, "tape_miss_blocked") {
		t.Fatalf("missing tape_miss_blocked reason; got %v", s.RejectionReasons)
	}
}

func TestJudge_hardRejectsStaleTapeOverThreshold(t *testing.T) {
	req := newRequest()
	req.PerTaskSignals.StalenessFindings = []verification.StalenessFinding{
		{Endpoint: "GET /a", Band: "stale"},
		{Endpoint: "GET /b", Band: "stale"},
		{Endpoint: "POST /c", Band: "unrecorded"},
	}
	llm := &fakeLLM{vendor: "google", model: "gemini-3.1-pro", body: goodLLMBody(0.95)}
	j := NewJudge(llm)
	s, err := j.Score(context.Background(), req, nil)
	if err != nil {
		t.Fatal(err)
	}
	if s.Passed {
		t.Fatalf("≥3 stale findings must hard-reject")
	}
	if !containsCategory(s.RejectionReasons, "tape_stale") {
		t.Fatalf("missing tape_stale reason; got %v", s.RejectionReasons)
	}
}

func TestJudge_hardRejectsHonestCIMismatch(t *testing.T) {
	req := newRequest()
	reports := []*testreport.TestReport{
		{
			SchemaVersion: testreport.SchemaVersion,
			TaskID:        "task_01",
			Tier:          testreport.TierHonestCI,
			Language:      testreport.LangPolyglot,
			Verdict:       testreport.VerdictFailed,
			HonestCI: &testreport.HonestCIStats{
				ExecutorRebuildHash: "0xaaaa",
				VerifierRebuildHash: "0xbbbb",
				BitIdentical:        false,
			},
		},
	}
	llm := &fakeLLM{vendor: "google", model: "gemini-3.1-pro", body: goodLLMBody(0.95)}
	j := NewJudge(llm)
	s, err := j.Score(context.Background(), req, reports)
	if err != nil {
		t.Fatal(err)
	}
	if s.Passed {
		t.Fatalf("rebuild mismatch must hard-reject")
	}
	if !containsCategory(s.RejectionReasons, "honest_ci_mismatch") {
		t.Fatalf("missing honest_ci_mismatch reason")
	}
}

func TestJudge_hardRejectsTier3FallbackWithoutCodeownerReq(t *testing.T) {
	req := newRequest()
	reports := []*testreport.TestReport{
		{
			SchemaVersion: testreport.SchemaVersion,
			TaskID:        "task_01",
			Tier:          testreport.TierProof,
			Language:      testreport.LangPython,
			Verdict:       testreport.VerdictTimedOut,
			Proof: &testreport.ProofStats{
				Prover:                   "dafny",
				TimedOut:                 true,
				FallbackTier:             "tier_2_5",
				CodeownerReviewRequired:  false,
			},
		},
	}
	llm := &fakeLLM{vendor: "google", model: "gemini-3.1-pro", body: goodLLMBody(0.95)}
	j := NewJudge(llm)
	s, err := j.Score(context.Background(), req, reports)
	if err != nil {
		t.Fatal(err)
	}
	if s.Passed {
		t.Fatalf("tier3 fallback without codeowner-review must hard-reject")
	}
	if !containsCategory(s.RejectionReasons, "tier3_fallback_missing_review") {
		t.Fatalf("missing tier3_fallback_missing_review reason; got %+v", s.RejectionReasons)
	}
}

func TestJudge_addsSynthTapeWarning_butStillPasses(t *testing.T) {
	req := newRequest()
	req.PerTaskSignals.TapeDispositionHistogram["synth-readonly"] = 4
	llm := &fakeLLM{vendor: "google", model: "gemini-3.1-pro", body: goodLLMBody(0.92)}
	j := NewJudge(llm)
	s, err := j.Score(context.Background(), req, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !s.Passed {
		t.Fatalf("synth tape is warn-level only; should still pass")
	}
	if !containsCategory(s.RejectionReasons, "synth_tape_used") {
		t.Fatalf("missing synth_tape_used warning; got %v", s.RejectionReasons)
	}
}

func TestJudge_retriesOnParseFailure(t *testing.T) {
	req := newRequest()
	llm := &fakeLLM{vendor: "google", model: "gemini-3.1-pro", body: "not-json"}
	j := NewJudge(llm)
	j.MaxRetries = 1
	_, err := j.Score(context.Background(), req, nil)
	if err == nil {
		t.Fatalf("expected ScoreErr after retries exhausted")
	}
	var se *ScoreErr
	if !errors.As(err, &se) {
		t.Fatalf("expected ScoreErr, got %T", err)
	}
	if se.Attempts != 2 {
		t.Fatalf("expected 2 attempts (1 retry), got %d", se.Attempts)
	}
	if llm.callCount != 2 {
		t.Fatalf("LLM should have been called twice; got %d", llm.callCount)
	}
}

func TestParseResponse_rejectsOutOfRangeScore(t *testing.T) {
	_, err := ParseResponse([]byte(`{"score": 1.5, "threshold": 0.85, "passed": true, "confidence": 0.9, "subscores": {}, "rejection_reasons": []}`), DefaultCriteria)
	if err == nil {
		t.Fatalf("expected error for score > 1")
	}
	_, err = ParseResponse([]byte(`{"score": -0.1, "threshold": 0.85, "passed": true, "confidence": 0.9, "subscores": {}, "rejection_reasons": []}`), DefaultCriteria)
	if err == nil {
		t.Fatalf("expected error for score < 0")
	}
}

func TestParseResponse_rejectsUnknownSubscore(t *testing.T) {
	_, err := ParseResponse([]byte(`{"score":0.9,"threshold":0.85,"passed":true,"confidence":0.9,"subscores":{"bogus":0.5},"rejection_reasons":[]}`), DefaultCriteria)
	if err == nil {
		t.Fatalf("expected error for unknown subscore key")
	}
}

func TestRenderPrompt_excludesReasoningOnPath(t *testing.T) {
	req := newRequest()
	req.Diff.Files = append(req.Diff.Files, cruciblev1.FileChange{
		Path:   "agent_trace/step01.json",
		Action: cruciblev1.ActionAdd,
	})
	_, err := RenderPrompt(PromptInput{Request: req})
	if err == nil {
		t.Fatalf("expected pre-audit failure on reasoning-shaped path")
	}
	if !strings.Contains(err.Error(), "leak") {
		t.Fatalf("expected leak-detection error; got %v", err)
	}
}

func TestRenderPrompt_emitsSystemAndUser(t *testing.T) {
	req := newRequest()
	body, err := RenderPrompt(PromptInput{Request: req, Criteria: DefaultCriteria})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(body.User, "TASK_ID: task_01") {
		t.Fatalf("user prompt missing task id; got %q", body.User[:128])
	}
	if !strings.Contains(body.System, "Cross-family invariant") {
		t.Fatalf("system prompt missing cross-family invariant message")
	}
}

func containsCategory(rr []RejectionReason, cat string) bool {
	for _, r := range rr {
		if r.Category == cat {
			return true
		}
	}
	return false
}
