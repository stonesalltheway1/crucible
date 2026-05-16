package dispatcher

import (
	"context"
	"errors"
	"testing"
	"time"

	cruciblev1 "github.com/crucible/sdk-go/crucible/v1"
	"github.com/crucible/verifier/internal/criticalpath"
	"github.com/crucible/verifier/internal/rubric"
	"github.com/crucible/verifier/internal/verification"
	"github.com/crucible/verifier/pkg/testreport"
)

// fakePool returns deterministic reports per (lang, tier).
type fakePool struct {
	results map[string]*testreport.TestReport
	err     map[string]error
}

func (p *fakePool) Submit(_ context.Context, k RunnerKind, req *verification.VerificationRequest) (*testreport.TestReport, error) {
	if e := p.err[k.String()]; e != nil {
		return nil, e
	}
	r := p.results[k.String()]
	if r != nil {
		r.TaskID = req.TaskID
		r.Language = k.Language
		r.Tier = k.Tier
		r.SchemaVersion = testreport.SchemaVersion
	}
	return r, nil
}
func (p *fakePool) Health() error { return nil }

type fakeTier3 struct {
	out *testreport.TestReport
	err error
}

func (f *fakeTier3) Discharge(_ context.Context, req *verification.VerificationRequest, prover string) (*testreport.TestReport, error) {
	if f.err != nil {
		return nil, f.err
	}
	r := f.out
	r.TaskID = req.TaskID
	r.Tier = testreport.TierProof
	r.SchemaVersion = testreport.SchemaVersion
	if r.Proof != nil {
		r.Proof.Prover = prover
	}
	return r, nil
}

type fakeTier4 struct {
	out *testreport.TestReport
	err error
}

func (f *fakeTier4) Verify(_ context.Context, req *verification.VerificationRequest) (*testreport.TestReport, error) {
	if f.err != nil {
		return nil, f.err
	}
	r := f.out
	r.TaskID = req.TaskID
	r.Tier = testreport.TierHonestCI
	r.Language = testreport.LangPolyglot
	r.SchemaVersion = testreport.SchemaVersion
	return r, nil
}

func newRequest() *verification.VerificationRequest {
	return &verification.VerificationRequest{
		TaskID:   "task_disp",
		TenantID: "ten_acme",
		BaseSHA:  "abc",
		Diff: cruciblev1.Diff{
			Files: []cruciblev1.FileChange{
				{Path: "src/auth/oauth.py", Action: cruciblev1.ActionModify, ContentSha256: "0xa"},
			},
		},
		Routing: cruciblev1.Routing{
			ExecutorModel:  "claude-opus-4-7",
			ExecutorVendor: "anthropic",
			VerifierModel:  "gemini-3.1-pro",
			VerifierVendor: "google",
		},
		Languages:         []string{"python"},
		ExecutorSandboxID: "sb_executor_01",
		PerTaskSignals: verification.TaskSignals{
			SelfHostAvailable:     true,
			ScrubberFiredOnAllPII: true,
		},
	}
}

func passingReport(tier testreport.Tier, lang testreport.Language) *testreport.TestReport {
	r := &testreport.TestReport{
		SchemaVersion: testreport.SchemaVersion,
		Tier:          tier,
		Language:      lang,
		Framework:     "fake",
		Verdict:       testreport.VerdictPassed,
		Passed:        true,
		DurationSeconds: 1,
	}
	switch tier {
	case testreport.TierMutation:
		r.Mutation = &testreport.MutationStats{Killed: 10, Survived: 1, Total: 11, Score: 0.91, Threshold: 0.85, DiffScoped: true}
	case testreport.TierPBT:
		r.PBT = &testreport.PBTStats{Iterations: 10000, IterationsMin: 10000}
	case testreport.TierContract:
		r.Contract = &testreport.ContractStats{}
	case testreport.TierHonestCI:
		r.HonestCI = &testreport.HonestCIStats{
			BuilderID: "test", BitIdentical: true, SLSALevel: 3,
			ExecutorRebuildHash: "0xabc", VerifierRebuildHash: "0xabc",
			ScrubberAuditOK: true,
		}
	case testreport.TierProof:
		r.Proof = &testreport.ProofStats{Prover: "dafny", Obligations: 5, Discharged: 5}
	}
	return r
}

func newDispatcherForTest(pool Pool, t3 Tier3Adapter, t4 Tier4Adapter, judge *rubric.Judge) *Dispatcher {
	cls := criticalpath.NewClassifier(criticalpath.NewPathPatternFeaturizer())
	d := New(pool, t3, t4, judge, cls)
	d.Now = func() time.Time { return time.Unix(1747300000, 0).UTC() }
	return d
}

func TestDispatch_passingPath(t *testing.T) {
	req := newRequest()
	pool := &fakePool{results: map[string]*testreport.TestReport{
		"python/tier_0_mutation": passingReport(testreport.TierMutation, testreport.LangPython),
		"python/tier_1_pbt":      passingReport(testreport.TierPBT, testreport.LangPython),
	}}
	t4 := &fakeTier4{out: passingReport(testreport.TierHonestCI, testreport.LangPolyglot)}
	judge := rubric.MakeHeuristicJudge(req.Routing.ExecutorVendor)
	d := newDispatcherForTest(pool, nil, t4, judge)

	resp, err := d.Dispatch(context.Background(), req)
	if err != nil {
		t.Fatalf("Dispatch: %v", err)
	}
	if resp.Approval == nil {
		t.Fatalf("expected Approval; got Rejection: %+v", resp.Rejection)
	}
	if resp.Approval.RubricScore == 0 {
		t.Fatalf("rubric score is zero")
	}
	if resp.Approval.TierResults.Tier0 == nil || !resp.Approval.TierResults.Tier0.Passed {
		t.Fatalf("Tier 0 not recorded as passed")
	}
}

func TestDispatch_tier3MoltenEscalation(t *testing.T) {
	req := newRequest()
	req.CriticalPathScores = []verification.CriticalPathScore{
		{File: "src/auth/oauth.py", Score: 92, Band: "molten"},
	}
	pool := &fakePool{results: map[string]*testreport.TestReport{
		"python/tier_0_mutation": passingReport(testreport.TierMutation, testreport.LangPython),
		"python/tier_1_pbt":      passingReport(testreport.TierPBT, testreport.LangPython),
		"python/tier_2_contract": passingReport(testreport.TierContract, testreport.LangPython),
	}}
	t3 := &fakeTier3{out: passingReport(testreport.TierProof, testreport.LangPython)}
	t4 := &fakeTier4{out: passingReport(testreport.TierHonestCI, testreport.LangPolyglot)}
	judge := rubric.MakeHeuristicJudge(req.Routing.ExecutorVendor)
	d := newDispatcherForTest(pool, t3, t4, judge)

	resp, err := d.Dispatch(context.Background(), req)
	if err != nil {
		t.Fatalf("Dispatch: %v", err)
	}
	if resp.Approval == nil {
		t.Fatalf("expected approval; got %+v", resp.Rejection)
	}
	if resp.Approval.TierResults.Tier3 == nil {
		t.Fatalf("Tier 3 should have been dispatched on molten file")
	}
}

func TestDispatch_tier3TimeoutEscalatesCodeownerReq(t *testing.T) {
	req := newRequest()
	req.CriticalPathScores = []verification.CriticalPathScore{
		{File: "src/auth/oauth.py", Score: 92, Band: "molten"},
	}
	pool := &fakePool{results: map[string]*testreport.TestReport{
		"python/tier_0_mutation": passingReport(testreport.TierMutation, testreport.LangPython),
		"python/tier_1_pbt":      passingReport(testreport.TierPBT, testreport.LangPython),
		"python/tier_2_contract": passingReport(testreport.TierContract, testreport.LangPython),
	}}
	timedOut := &testreport.TestReport{
		SchemaVersion: testreport.SchemaVersion,
		Tier:          testreport.TierProof,
		Language:      testreport.LangPython,
		Verdict:       testreport.VerdictTimedOut,
		Passed:        false,
		Proof: &testreport.ProofStats{
			Prover:                  "dafny",
			TimedOut:                true,
			FallbackTier:            "tier_2_5",
			CodeownerReviewRequired: false, // deliberately false — dispatcher MUST enforce
		},
	}
	t3 := &fakeTier3{out: timedOut}
	t4 := &fakeTier4{out: passingReport(testreport.TierHonestCI, testreport.LangPolyglot)}
	judge := rubric.MakeHeuristicJudge(req.Routing.ExecutorVendor)
	d := newDispatcherForTest(pool, t3, t4, judge)

	resp, err := d.Dispatch(context.Background(), req)
	if err != nil {
		t.Fatalf("Dispatch: %v", err)
	}
	if resp.Approval != nil {
		t.Fatalf("Tier 3 timeout without codeowner-review must hard-reject; got Approval")
	}
	if resp.Rejection == nil {
		t.Fatalf("expected Rejection")
	}
	found := false
	for _, r := range resp.Rejection.RejectionReasons {
		if r.Category == "tier3_fallback_missing_review" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected tier3_fallback_missing_review reason; got %+v", resp.Rejection.RejectionReasons)
	}
}

func TestDispatch_hardRejectsOnPoolError(t *testing.T) {
	req := newRequest()
	pool := &fakePool{
		results: map[string]*testreport.TestReport{
			"python/tier_1_pbt": passingReport(testreport.TierPBT, testreport.LangPython),
		},
		err: map[string]error{
			"python/tier_0_mutation": errors.New("pool: image pull failed"),
		},
	}
	t4 := &fakeTier4{out: passingReport(testreport.TierHonestCI, testreport.LangPolyglot)}
	judge := rubric.MakeHeuristicJudge(req.Routing.ExecutorVendor)
	d := newDispatcherForTest(pool, nil, t4, judge)

	resp, err := d.Dispatch(context.Background(), req)
	if err != nil {
		t.Fatalf("Dispatch: %v", err)
	}
	// The Tier 0 should now be a failed report — score should be lower
	// and rubric should not approve.
	if resp.Approval != nil {
		t.Fatalf("expected rejection on pool error; got Approval")
	}
}

func TestDispatch_refusesSameFamily(t *testing.T) {
	req := newRequest()
	req.Routing.VerifierVendor = "anthropic" // same as executor
	pool := &fakePool{}
	t4 := &fakeTier4{out: passingReport(testreport.TierHonestCI, testreport.LangPolyglot)}
	judge := rubric.MakeHeuristicJudge(req.Routing.ExecutorVendor)
	d := newDispatcherForTest(pool, nil, t4, judge)

	_, err := d.Dispatch(context.Background(), req)
	if err == nil {
		t.Fatalf("expected SameFamilyError; got nil")
	}
	var sfe *verification.SameFamilyError
	if !errors.As(err, &sfe) {
		t.Fatalf("expected SameFamilyError, got %T %v", err, err)
	}
}

func TestSelectTiers_alwaysIncludesT0AndT4(t *testing.T) {
	r := newRequest()
	got := selectTiers(r)
	want := map[testreport.Tier]bool{testreport.TierMutation: true, testreport.TierHonestCI: true}
	for k := range want {
		if !hasTier(got, k) {
			t.Fatalf("selectTiers missing %s", k)
		}
	}
}

func TestSelectTiers_specChangeAddsContract(t *testing.T) {
	r := newRequest()
	r.SpecChanges = []verification.SpecChange{{Path: "openapi.yaml", Kind: "openapi", PreviousHash: "a", CurrentHash: "b"}}
	got := selectTiers(r)
	if !hasTier(got, testreport.TierContract) {
		t.Fatalf("spec change should add Tier 2 contract")
	}
}

func TestSelectTiers_moltenAddsProof(t *testing.T) {
	r := newRequest()
	r.CriticalPathScores = []verification.CriticalPathScore{{File: "x", Score: 91, Band: "molten"}}
	got := selectTiers(r)
	if !hasTier(got, testreport.TierProof) {
		t.Fatalf("molten should add Tier 3 proof")
	}
}
