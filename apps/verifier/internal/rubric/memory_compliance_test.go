package rubric

import (
	"context"
	"testing"

	cruciblev1 "github.com/crucible/sdk-go/crucible/v1"
)

type stubCompliance struct {
	report cruciblev1.ComplianceReport
	err    error
}

func (s *stubCompliance) CheckCompliance(ctx context.Context, req ComplianceRequest) (cruciblev1.ComplianceReport, error) {
	return s.report, s.err
}

func TestFeaturizer_DisabledReturnsEmpty(t *testing.T) {
	f := MemoryComplianceFeaturizer{Disabled: true}
	got, err := f.Featurize(context.Background(), ComplianceRequest{TenantID: "ten_a"})
	if err != nil {
		t.Fatal(err)
	}
	if got.ConventionsChecked != 0 || len(got.RejectionReasons) != 0 {
		t.Fatal("disabled featurizer should return zero features")
	}
}

func TestFeaturizer_MapsViolationsToReasons(t *testing.T) {
	stub := &stubCompliance{report: cruciblev1.ComplianceReport{
		ConventionsChecked: 3,
		Violations: []cruciblev1.ComplianceReportViolation{
			{ConventionID: "conv_w", RuleNl: "warn rule", OffendingFile: "x.ts", Severity: "warn"},
			{ConventionID: "conv_e", RuleNl: "error rule", OffendingFile: "y.ts", Severity: "error"},
		},
	}}
	f := MemoryComplianceFeaturizer{Bridge: stub}
	feats, err := f.Featurize(context.Background(), ComplianceRequest{TenantID: "ten_a"})
	if err != nil {
		t.Fatal(err)
	}
	if feats.WarnViolations != 1 || feats.ErrorViolations != 1 {
		t.Fatalf("warn=%d err=%d", feats.WarnViolations, feats.ErrorViolations)
	}
	if len(feats.RejectionReasons) != 2 {
		t.Fatalf("want 2 reasons; got %d", len(feats.RejectionReasons))
	}
	// Memory featurizer never escalates to severity=error.
	for _, r := range feats.RejectionReasons {
		if r.Severity == "error" {
			t.Fatalf("memory featurizer must never emit severity=error: %+v", r)
		}
	}
}

func TestApplyToScore_AdjustsTrustSignalDownward(t *testing.T) {
	s := &Score{
		Score:     0.95,
		Threshold: 0.85,
		Passed:    true,
		Subscores: map[string]float64{
			"diff_correctness":       1.0,
			"test_adequacy":          1.0,
			"spec_consistency":       1.0,
			"robustness":             1.0,
			"security_posture":       1.0,
			"trust_signal_alignment": 0.8,
		},
	}
	feats := Features{
		TrustSignalDelta: -0.20,
		WarnViolations:   2,
		ErrorViolations:  1,
	}
	ApplyToScore(s, feats)
	if s.Subscores["trust_signal_alignment"] >= 0.8 {
		t.Fatalf("trust signal not adjusted; got %.3f", s.Subscores["trust_signal_alignment"])
	}
	// Composite must drop too.
	if s.Score >= 0.95 {
		t.Fatalf("composite score not reduced; got %.3f", s.Score)
	}
}

func TestFeaturizer_ErrorPathSurfacesNonFatal(t *testing.T) {
	stub := &stubCompliance{err: context.DeadlineExceeded}
	f := MemoryComplianceFeaturizer{Bridge: stub}
	feats, err := f.Featurize(context.Background(), ComplianceRequest{TenantID: "ten_a"})
	if err == nil {
		t.Fatal("featurizer should return the underlying error")
	}
	if feats.ConventionsChecked != 0 {
		t.Fatal("on error, features must be zero — caller falls back to Phase-4 trust signals")
	}
}
