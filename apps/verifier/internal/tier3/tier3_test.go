package tier3

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	cruciblev1 "github.com/crucible/sdk-go/crucible/v1"
	"github.com/crucible/verifier/internal/verification"
	"github.com/crucible/verifier/pkg/testreport"
)

// fakeProver lets tests inject behaviour deterministically.
type fakeProver struct {
	id        ProverID
	obligations int
	discharged  int
	sleep      time.Duration
	err        error
	timedOut   bool
}

func (f *fakeProver) ID() ProverID { return f.id }
func (f *fakeProver) Discharge(ctx context.Context, req DischargeRequest) (DischargeResult, error) {
	if f.sleep > 0 {
		select {
		case <-ctx.Done():
			return DischargeResult{TimedOut: true, Obligations: f.obligations, Discharged: 0}, ctx.Err()
		case <-time.After(f.sleep):
		}
	}
	if f.err != nil {
		return DischargeResult{TimedOut: f.timedOut}, f.err
	}
	return DischargeResult{
		Obligations: f.obligations,
		Discharged:  f.discharged,
		TimedOut:    f.timedOut,
	}, nil
}

func newReq() *verification.VerificationRequest {
	return &verification.VerificationRequest{
		TaskID:  "task_t3",
		BaseSHA: "abc",
		Diff: cruciblev1.Diff{
			Files: []cruciblev1.FileChange{
				{Path: "src/auth/policy.dfy", Action: cruciblev1.ActionModify, ContentSha256: "0xdf"},
			},
		},
	}
}

func TestAdapter_dischargesAllObligations(t *testing.T) {
	a := NewAdapter()
	a.Provers[ProverDafny] = &fakeProver{id: ProverDafny, obligations: 5, discharged: 5}
	a.Budgets.Dafny = time.Second
	report, err := a.Discharge(context.Background(), newReq(), "dafny")
	if err != nil {
		t.Fatalf("Discharge: %v", err)
	}
	if !report.Passed {
		t.Fatalf("expected pass; got %+v", report)
	}
	if report.Proof.TimedOut {
		t.Fatalf("timed_out should be false")
	}
}

func TestAdapter_timeoutSetsFallbackAndCodeownerReview(t *testing.T) {
	a := NewAdapter()
	a.Provers[ProverDafny] = &fakeProver{id: ProverDafny, obligations: 5, sleep: 200 * time.Millisecond}
	a.Budgets.Dafny = 20 * time.Millisecond
	report, err := a.Discharge(context.Background(), newReq(), "dafny")
	if err != nil {
		t.Fatalf("Discharge: %v", err)
	}
	if !report.Proof.TimedOut {
		t.Fatalf("expected timed_out=true")
	}
	if report.Proof.FallbackTier != "tier_2_5" {
		t.Fatalf("expected fallback_tier=tier_2_5; got %q", report.Proof.FallbackTier)
	}
	if !report.Proof.CodeownerReviewRequired {
		t.Fatalf("CRITICAL: timeout MUST set codeowner_review_required=true (Phase-4 brief invariant)")
	}
	if report.Verdict != testreport.VerdictTimedOut {
		t.Fatalf("expected verdict=timed_out; got %q", report.Verdict)
	}
}

func TestAdapter_stubProversReportToolUnavailable(t *testing.T) {
	a := NewAdapter()
	for _, p := range []string{"lean", "tla", "z3"} {
		r, err := a.Discharge(context.Background(), newReq(), p)
		if err != nil {
			t.Fatalf("Discharge(%s): %v", p, err)
		}
		if r.Verdict != testreport.VerdictToolUnavailable {
			t.Fatalf("Discharge(%s) verdict=%q want tool_unavailable", p, r.Verdict)
		}
	}
}

func TestAdapter_unknownProverErrors(t *testing.T) {
	a := NewAdapter()
	_, err := a.Discharge(context.Background(), newReq(), "bogus")
	if err == nil || !strings.Contains(err.Error(), "unknown prover") {
		t.Fatalf("expected unknown-prover error; got %v", err)
	}
}

func TestAdapter_partialProofCacheRoundTrip(t *testing.T) {
	a := NewAdapter()
	fp := &fakeProverWithPartial{id: ProverDafny, obligations: 3, discharged: 2}
	a.Provers[ProverDafny] = fp
	a.Budgets.Dafny = time.Second
	req := newReq()
	// First call seeds the cache.
	_, _ = a.Discharge(context.Background(), req, "dafny")
	// Second call should receive the cached partial.
	fp.assertCachedSeen = true
	_, _ = a.Discharge(context.Background(), req, "dafny")
	if !fp.sawCached {
		t.Fatalf("second call did not see cached partial")
	}
}

type fakeProverWithPartial struct {
	id              ProverID
	obligations     int
	discharged      int
	cached          []byte
	sawCached       bool
	assertCachedSeen bool
}

func (f *fakeProverWithPartial) ID() ProverID { return f.id }
func (f *fakeProverWithPartial) Discharge(_ context.Context, req DischargeRequest) (DischargeResult, error) {
	if len(req.PartialProof) > 0 {
		f.sawCached = true
	}
	return DischargeResult{
		Obligations:   f.obligations,
		Discharged:    f.discharged,
		CachedPartial: []byte("cache-blob"),
	}, nil
}

func TestParseDafnyVerifyOutput(t *testing.T) {
	cases := []struct {
		in              string
		wantOblig       int
		wantDischarged  int
	}{
		{"Dafny program verifier finished with 7 verified, 0 errors\n", 7, 7},
		{"Dafny program verifier finished with 4 verified, 2 errors\n", 6, 4},
		{"Verification finished. Verified: 3. Errors: 1.\n", 4, 3},
		{"compile errors only\n", 0, 0},
	}
	for _, c := range cases {
		o, d := parseDafnyVerifyOutput(c.in)
		if o != c.wantOblig || d != c.wantDischarged {
			t.Fatalf("parse %q: (oblig=%d, disch=%d) want (%d, %d)", c.in, o, d, c.wantOblig, c.wantDischarged)
		}
	}
}

func TestDafnyProver_returnsUnavailableWithoutBinary(t *testing.T) {
	p := &DafnyProver{DafnyBin: "definitely-not-a-real-binary-xyz123"}
	_, err := p.Discharge(context.Background(), DischargeRequest{Spec: "x.dfy"})
	if !errors.Is(err, ErrProverUnavailable) {
		t.Fatalf("expected ErrProverUnavailable; got %v", err)
	}
}
