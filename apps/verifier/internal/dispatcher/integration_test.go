package dispatcher

import (
	"context"
	"sync"
	"testing"

	cruciblev1 "github.com/crucible/sdk-go/crucible/v1"
	"github.com/crucible/verifier/internal/rubric"
	"github.com/crucible/verifier/internal/verification"
	"github.com/crucible/verifier/pkg/testreport"
)

// disagreementLLM produces a deterministic score driven by its `vendor`
// — used to model that Anthropic Opus and Gemini Pro disagree on the
// boundary ~5–10% of the time. We model the disagreement by returning
// different scores for the same prompt depending on which vendor's
// instance is being called.
type disagreementLLM struct {
	vendor   string
	model    string
	scoreMap map[string]float64 // diff_hash → score
}

func (l *disagreementLLM) Vendor() string { return l.vendor }
func (l *disagreementLLM) Model() string  { return l.model }
func (l *disagreementLLM) Call(_ context.Context, req rubric.CallRequest) (rubric.CallResponse, error) {
	// Derive a deterministic score from the prompt hash.
	score := 0.88
	if l.scoreMap != nil {
		for key, v := range l.scoreMap {
			if contains(req.User, key) {
				score = v
				break
			}
		}
	}
	body := []byte(`{"score":` + ftoa(score) + `,"threshold":0.85,"passed":` + boolStr(score >= 0.85) + `,"confidence":0.9,"subscores":{},"rejection_reasons":[]}`)
	return rubric.CallResponse{Body: body, OutputTokens: 100, CostUSD: 0.001}, nil
}

func contains(s, sub string) bool {
	if len(sub) == 0 {
		return true
	}
	if len(sub) > len(s) {
		return false
	}
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func ftoa(f float64) string {
	// Tiny float-to-string for the test fixture; full precision is irrelevant.
	if f >= 1 {
		return "1.0"
	}
	if f <= 0 {
		return "0.0"
	}
	first := int(f * 10)
	second := int((f*100)) % 10
	return "0." + string(rune('0'+first)) + string(rune('0'+second))
}

func boolStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

// TestCrossFamilyDisagreement_recordedAsDistinctVerdicts simulates the
// scenario where the same diff yields different verdicts when paired
// with two different cross-family verifiers. The dispatcher's job is
// to make this measurable, not to resolve the disagreement.
//
// The Phase-4 brief mandates a "cross-family disagreement test: same
// diff verified by Opus and Gemini; verify disagreement detection on
// the ~5–10% case." We mock the LLM and assert that the two verdicts
// are independently visible.
func TestCrossFamilyDisagreement_recordedAsDistinctVerdicts(t *testing.T) {
	req := newRequest()
	req.CriticalPathScores = []verification.CriticalPathScore{
		{File: "src/auth/oauth.py", Score: 65, Band: "hot"},
	}
	pool := &fakePool{results: map[string]*testreport.TestReport{
		"python/tier_0_mutation": passingReport(testreport.TierMutation, testreport.LangPython),
		"python/tier_1_pbt":      passingReport(testreport.TierPBT, testreport.LangPython),
		"python/tier_2_contract": passingReport(testreport.TierContract, testreport.LangPython),
	}}
	t4 := &fakeTier4{out: passingReport(testreport.TierHonestCI, testreport.LangPolyglot)}

	// Score the same diff with two different cross-family verifiers.
	var verdicts sync.Map
	for _, v := range []struct {
		vendor, model string
		execVendor    string
	}{
		{"google", "gemini-3.1-pro", "anthropic"},
		{"anthropic", "claude-opus-4-7", "google"},
	} {
		// Switch the request's executor vendor so each verifier
		// invocation respects the cross-family invariant.
		req.Routing.ExecutorVendor = v.execVendor
		req.Routing.ExecutorModel = "exec-stub"
		req.Routing.VerifierVendor = v.vendor
		req.Routing.VerifierModel = v.model

		llm := &disagreementLLM{
			vendor: v.vendor, model: v.model,
			// 5% disagreement: gemini scores below threshold; opus above.
			scoreMap: map[string]float64{
				"oauth.py": map[string]float64{"google": 0.80, "anthropic": 0.92}[v.vendor],
			},
		}
		j := rubric.NewJudge(llm)
		d := newDispatcherForTest(pool, nil, t4, j)
		resp, err := d.Dispatch(context.Background(), req)
		if err != nil {
			t.Fatalf("Dispatch(%s): %v", v.vendor, err)
		}
		verdicts.Store(v.vendor, resp)
	}

	gemini, _ := verdicts.Load("google")
	opus, _ := verdicts.Load("anthropic")
	gResp := gemini.(*verification.VerificationResponse)
	oResp := opus.(*verification.VerificationResponse)

	// Disagreement: Gemini rejected (0.80 < 0.85), Opus approved (0.92).
	if gResp.Approval != nil {
		t.Fatalf("Gemini should have rejected (score 0.80 < 0.85); got Approval")
	}
	if oResp.Approval == nil {
		t.Fatalf("Opus should have approved (score 0.92 >= 0.85); got Rejection: %+v", oResp.Rejection)
	}
	// Distinct verifier identities recorded in the approval/rejection.
	if oResp.Approval.VerifierModel != "claude-opus-4-7" {
		t.Fatalf("Opus approval should record VerifierModel=claude-opus-4-7; got %q", oResp.Approval.VerifierModel)
	}
	if gResp.Rejection.DiffHash != oResp.Approval.DiffHash {
		t.Fatalf("Both verifiers should agree on diff_hash even when verdicts disagree")
	}
}

// TestNoReasoningEverReachesVerifier asserts the leak audit fires on
// every path that could carry executor reasoning. This is the
// brand-existential invariant.
func TestNoReasoningEverReachesVerifier(t *testing.T) {
	cases := []struct {
		name      string
		mutator   func(*verification.VerificationRequest)
		wantError string
	}{
		{
			name: "path looks like agent_trace/",
			mutator: func(r *verification.VerificationRequest) {
				r.Diff.Files = append(r.Diff.Files, cruciblev1.FileChange{
					Path: "agent_trace/step01.json", Action: cruciblev1.ActionAdd,
				})
			},
			wantError: "leak",
		},
		{
			name: "attestation chain contains 'reasoning'",
			mutator: func(r *verification.VerificationRequest) {
				r.AttestationChain = append(r.AttestationChain, "rekor:abc-reasoning-xyz")
			},
			wantError: "leak",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			req := newRequest()
			c.mutator(req)
			pool := &fakePool{}
			t4 := &fakeTier4{out: passingReport(testreport.TierHonestCI, testreport.LangPolyglot)}
			j := rubric.MakeHeuristicJudge(req.Routing.ExecutorVendor)
			d := newDispatcherForTest(pool, nil, t4, j)
			_, err := d.Dispatch(context.Background(), req)
			if err == nil {
				t.Fatalf("expected leak error; got nil")
			}
		})
	}
}
