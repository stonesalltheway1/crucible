package criticalpath

import (
	"context"
	"testing"
)

// labeledExample is a test case from docs/06-research/tier3-trigger-automation.md
// §"Examples". Each test below MUST classify into the documented band.
type labeledExample struct {
	name      string
	path      string
	content   []byte
	wantBand  Band
	wantScore float64 // approximate; we check the band, not exact score
	// Production-signal injections that mimic what the prod featurizers
	// would compute. We hand-set them so the test is self-contained.
	prodIncident     float64
	prodSLOBacking   float64
	prodReview       float64
	prodCVE          float64
	prodTestCov      float64
	prodCodeOwners   float64
	prodCentrality   float64
	llmCategory      string
	llmConfidence    float64
}

// labeledExamples is the canonical doc set. Adding cases here strengthens
// the classifier's ability to ship.
var labeledExamples = []labeledExample{
	{
		name:           "oauth_callback.py — obvious security",
		path:           "src/auth/oauth_callback.py",
		content:        []byte("def callback(req):\n    # validate state, exchange code for token\n    pass\n"),
		wantBand:       BandMolten,
		prodCVE:        1.0,
		prodCodeOwners: 1.0,
		prodCentrality: 0.5,
		llmCategory:    "security",
		llmConfidence:  0.95,
	},
	{
		name:           "refund_engine.go — obvious money",
		path:           "services/billing/refund_engine.go",
		content:        []byte("package billing\nfunc Refund(ctx context.Context, id string) error { return nil }\n"),
		wantBand:       BandMolten,
		prodIncident:   1.0,
		prodSLOBacking: 1.0,
		prodReview:     0.85,
		prodCodeOwners: 1.0,
		llmCategory:    "money",
		llmConfidence:  0.95,
	},
	{
		name:          "MarketingHeroBanner.tsx — obvious UI cold",
		path:          "web/components/MarketingHeroBanner.tsx",
		content:       []byte("export const Hero = () => <div>Welcome!</div>\n"),
		wantBand:      BandCold,
		llmCategory:   "ui",
		llmConfidence: 0.95,
	},
	{
		name:           "retry.ts — load-bearing plumbing with high fan-in",
		path:           "lib/utils/retry.ts",
		content:        []byte("export async function retry<T>(fn: () => Promise<T>, opts?: { tries?: number }): Promise<T> { /* ... */ }\n"),
		wantBand:       BandHot,
		prodCentrality: 0.85, // top-5pc fan-in
		prodReview:     0.5,
		llmCategory:    "plumbing",
		llmConfidence:  0.8,
	},
	{
		name:          "payment_simulator_for_demos.py — adversarial mislabel",
		path:          "tools/payment_simulator_for_demos.py",
		content:       []byte("# Demo simulator only, not production.\nimport random\n"),
		wantBand:      BandCold,
		llmCategory:   "ui", // judge correctly flags as demo
		llmConfidence: 0.9,
	},
}

// stubJudge returns a fixed CategoryAndConfidence for any input.
type stubJudge struct{ cat string; conf float64 }

func (s *stubJudge) Categorise(_ context.Context, _ string, _ []byte) (CategoryAndConfidence, error) {
	return CategoryAndConfidence{Category: s.cat, Confidence: s.conf}, nil
}

type memCache struct{ m map[string]CategoryAndConfidence }

func newMemCache() *memCache { return &memCache{m: map[string]CategoryAndConfidence{}} }
func (c *memCache) Get(k string) (CategoryAndConfidence, bool) {
	v, ok := c.m[k]
	return v, ok
}
func (c *memCache) Put(k string, v CategoryAndConfidence) { c.m[k] = v }

func TestLabeledExamplesClassifyCorrectly(t *testing.T) {
	for _, ex := range labeledExamples {
		t.Run(ex.name, func(t *testing.T) {
			featurizer := buildStackForExample(ex)
			cls := NewClassifier(featurizer)
			s, err := cls.Classify(context.Background(), ex.path, ex.content)
			if err != nil {
				t.Fatalf("Classify: %v", err)
			}
			if s.Band != ex.wantBand {
				t.Fatalf("path=%s: got band=%s S≈%g (top: %v), want band=%s",
					ex.path, s.Band, s.Score, s.TopSignals, ex.wantBand)
			}
		})
	}
}

func TestPRBand_aggregatesAcrossFiles(t *testing.T) {
	featurizer := buildStackForExample(labeledExamples[0]) // oauth_callback
	cls := NewClassifier(featurizer)
	s, _ := cls.Classify(context.Background(), labeledExamples[0].path, labeledExamples[0].content)
	band := cls.PRBand([]Score{s}, 100)
	if band != BandMolten {
		t.Fatalf("PRBand on Molten file should stay Molten, got %s", band)
	}
}

func TestPRBand_threeHotFilesEscalateMolten(t *testing.T) {
	// Build three synthetic "warm-but-not-molten" files.
	mkScore := func(p string, v float64) Score {
		return Score{
			Path:  p,
			Score: v,
			Band:  BandHot,
		}
	}
	cls := NewClassifier(NewPathPatternFeaturizer())
	band := cls.PRBand([]Score{
		mkScore("a", 65),
		mkScore("b", 65),
		mkScore("c", 65),
	}, 100)
	if band != BandMolten {
		t.Fatalf("3 Hot files should escalate to Molten, got %s", band)
	}
}

func TestPRBand_securityTokensWithSize_escalatesHot(t *testing.T) {
	cls := NewClassifier(NewPathPatternFeaturizer())
	s := Score{
		Path:  "src/api/login.go",
		Score: 35,
		Band:  BandCold,
		Features: FileFeatures{
			Path:            "src/api/login.go",
			PathPatternAxes: map[Axis]bool{AxisSecurity: true},
		},
		TopSignals: []SignalHit{
			{Name: "path_pattern", RawValue: 1.0, Contribution: 1.5},
		},
	}
	got := cls.PRBand([]Score{s}, 50)
	if got != BandHot {
		t.Fatalf("security token + 50-line diff should suggest Hot, got %s", got)
	}
}

func TestSigmoid_monotonic(t *testing.T) {
	if sigmoid(-1) >= sigmoid(0) {
		t.Fatalf("sigmoid not monotonic")
	}
	if sigmoid(0) >= sigmoid(1) {
		t.Fatalf("sigmoid not monotonic")
	}
}

// buildStackForExample wires the deterministic featurizers around a
// labeledExample's hand-set production signals.
func buildStackForExample(ex labeledExample) FeaturizerStack {
	stack := FeaturizerStack{NewPathPatternFeaturizer()}
	if ex.llmCategory != "" {
		stack = append(stack, &LLMJudgeFeaturizer{
			Judge: &stubJudge{cat: ex.llmCategory, conf: ex.llmConfidence},
			Cache: newMemCache(),
		})
	}
	stack = append(stack,
		&FanInCentralityFeaturizer{
			Centrality: map[string]float64{ex.path: ex.prodCentrality},
			Top5pcCut:  0.8,
		},
		&CVEFeaturizer{Touched: func() map[string]bool {
			if ex.prodCVE > 0 {
				return map[string]bool{ex.path: true}
			}
			return nil
		}()},
		&CodeOwnersFeaturizer{Owners: codeOwnersFromHint(ex)},
		&ProductionSignalsFeaturizer{
			IncidentMention: map[string]float64{ex.path: ex.prodIncident},
			SLOBacking:      map[string]float64{ex.path: ex.prodSLOBacking},
			ReviewIntensity: map[string]float64{ex.path: ex.prodReview},
			TestCovGradient: map[string]float64{ex.path: ex.prodTestCov},
		},
	)
	return stack
}

func codeOwnersFromHint(ex labeledExample) map[string][]string {
	if ex.prodCodeOwners <= 0 {
		return nil
	}
	// Map every example path to a critical team.
	team := "@security-team"
	if ex.llmCategory == "money" {
		team = "@payments-team"
	}
	return map[string][]string{ex.path: {team}}
}
