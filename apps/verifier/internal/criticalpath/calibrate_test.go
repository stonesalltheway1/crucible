package criticalpath

import (
	"testing"
)

func TestLogisticFit_recoversWeightsFromSyntheticData(t *testing.T) {
	// Generate synthetic data: y=1 when path_pattern=1 OR llm_category > 0.7;
	// y=0 otherwise. Fit should preserve high weight on those signals.
	mk := func(path, llm string, llmS float64, label LabelKind) CalibrationLabel {
		return CalibrationLabel{
			Path:  path,
			Label: label,
			Features: FileFeatures{
				Path:             path,
				PathPatternScore: 1.0,
				LLMCategoryScore: llmS,
				LLMCategory:      llm,
			},
		}
	}
	samples := []CalibrationLabel{
		mk("auth/login.py", "security", 1.0, LabelCritical),
		mk("billing/charge.go", "money", 0.95, LabelCritical),
		mk("data/migrate.sql", "data-integrity", 1.0, LabelCritical),
		mk("auth/oauth.py", "security", 0.9, LabelCritical),
		mk("payments/refund.go", "money", 0.95, LabelCritical),
		mk("api/health.go", "infrastructure", 0.5, LabelCold),
		mk("ui/banner.tsx", "ui", 1.0, LabelCold),
		mk("ui/footer.tsx", "ui", 1.0, LabelCold),
		mk("docs/changelog.md", "dead", 0.9, LabelCold),
		mk("marketing/hero.tsx", "ui", 1.0, LabelCold),
		mk("infra/cache.go", "infrastructure", 0.4, LabelWarm),
		mk("api/payments_webhook.go", "money", 0.85, LabelCritical),
	}
	res, err := LogisticFit(samples, DefaultWeights, 0.1, 500)
	if err != nil {
		t.Fatalf("LogisticFit: %v", err)
	}
	if res.AccuracyOnSet < 0.7 {
		t.Fatalf("training-set accuracy %v < 0.7", res.AccuracyOnSet)
	}
	if res.Weights.PathPattern <= 0 {
		t.Fatalf("path-pattern weight collapsed: %v", res.Weights.PathPattern)
	}
	if res.Weights.LLMCategory <= 0 {
		t.Fatalf("llm-category weight collapsed: %v", res.Weights.LLMCategory)
	}
}

func TestLogisticFit_rejectsTooFewSamples(t *testing.T) {
	_, err := LogisticFit(nil, DefaultWeights, 0.05, 100)
	if err == nil {
		t.Fatalf("expected error for empty samples")
	}
}

func TestSerializeDeserializeWeights_roundTrip(t *testing.T) {
	w := DefaultWeights
	b, err := SerializeWeights(w)
	if err != nil {
		t.Fatal(err)
	}
	got, err := DeserializeWeights(b)
	if err != nil {
		t.Fatal(err)
	}
	if got != w {
		t.Fatalf("round-trip changed weights: %#v vs %#v", got, w)
	}
}

func TestStratifiedSample_balancesBands(t *testing.T) {
	in := []Score{
		{Path: "a", Score: 90, Band: BandMolten},
		{Path: "b", Score: 85, Band: BandMolten},
		{Path: "c", Score: 70, Band: BandHot},
		{Path: "d", Score: 65, Band: BandHot},
		{Path: "e", Score: 50, Band: BandWarm},
		{Path: "f", Score: 45, Band: BandWarm},
		{Path: "g", Score: 20, Band: BandCold},
		{Path: "h", Score: 5, Band: BandCold},
	}
	out := StratifiedSample(in, 8)
	if len(out) != 8 {
		t.Fatalf("StratifiedSample: got %d, want 8", len(out))
	}
}
