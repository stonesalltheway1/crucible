package criticalpath

import (
	"context"
	"errors"
	"math"
	"sort"
	"strings"
)

// Band names the four criticality buckets from
// docs/06-research/tier3-trigger-automation.md §"Threshold bands".
type Band string

const (
	BandCold   Band = "cold"
	BandWarm   Band = "warm"
	BandHot    Band = "hot"
	BandMolten Band = "molten"
)

// Score is the per-file classifier output.
type Score struct {
	Path       string       `json:"path"`
	Score      float64      `json:"score"`            // 0..100
	Band       Band         `json:"band"`
	TopSignals []SignalHit  `json:"top_signals"`      // up to 3 strongest contributors
	Features   FileFeatures `json:"features,omitempty"`
}

// SignalHit names one signal that pushed score up (or down) for surfacing
// in PR comments and the override-learning loop.
type SignalHit struct {
	Name        string  `json:"name"`
	Contribution float64 `json:"contribution"` // weighted contribution to the pre-sigmoid sum
	RawValue    float64 `json:"raw_value"`    // 0..1
}

// Weights are the per-signal coefficients. Defaults from the doc; tenants
// override after `crucible calibrate`.
type Weights struct {
	PathPattern      float64 `json:"path_pattern"`
	LLMCategory      float64 `json:"llm_category"`
	FanInCentrality  float64 `json:"fanin_centrality"`
	IncidentMention  float64 `json:"incident_mention"`
	SLOBacking       float64 `json:"slo_backing"`
	ReviewIntensity  float64 `json:"review_intensity"`
	CVEHistory       float64 `json:"cve_history"`
	TestCovGradient  float64 `json:"test_cov_gradient"`
	CommentMarker    float64 `json:"comment_marker"`
	CodeOwnersScore  float64 `json:"codeowners"`
	UIOrTestPenalty  float64 `json:"ui_or_test_penalty"` // applied as negative
}

// DefaultWeights mirrors the doc's weighted sum.
var DefaultWeights = Weights{
	PathPattern:     1.5,
	LLMCategory:     1.2,
	FanInCentrality: 1.0,
	IncidentMention: 0.9,
	SLOBacking:      0.8,
	ReviewIntensity: 0.7,
	CVEHistory:      0.7,
	TestCovGradient: 0.6,
	CommentMarker:   0.5,
	CodeOwnersScore: 0.4,
	UIOrTestPenalty: 0.5, // applied as -W * raw
}

// Thresholds are the band cutoffs.
type Thresholds struct {
	Cold  float64 // < this → Cold
	Warm  float64 // < this → Warm
	Hot   float64 // < this → Hot; ≥ this → Molten
}

// DefaultThresholds: Cold 0–39, Warm 40–59, Hot 60–79, Molten 80–100.
var DefaultThresholds = Thresholds{Cold: 40, Warm: 60, Hot: 80}

// Classifier composes a Featurizer with Weights + Thresholds.
type Classifier struct {
	Featurizer Featurizer
	Weights    Weights
	Thresholds Thresholds
}

// NewClassifier returns a classifier with the doc defaults.
func NewClassifier(featurizer Featurizer) *Classifier {
	return &Classifier{
		Featurizer: featurizer,
		Weights:    DefaultWeights,
		Thresholds: DefaultThresholds,
	}
}

// Classify returns Score for one file.
func (c *Classifier) Classify(ctx context.Context, path string, content []byte) (Score, error) {
	if c.Featurizer == nil {
		return Score{}, errors.New("criticalpath: nil Featurizer")
	}
	f, err := c.Featurizer.Featurize(ctx, path, content)
	if err != nil {
		return Score{}, err
	}
	return c.ScoreFeatures(f), nil
}

// ClassifyMany classifies a batch — used by the dispatcher when fanning
// out over a diff.
func (c *Classifier) ClassifyMany(ctx context.Context, files map[string][]byte) ([]Score, error) {
	if c.Featurizer == nil {
		return nil, errors.New("criticalpath: nil Featurizer")
	}
	keys := make([]string, 0, len(files))
	for k := range files {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]Score, 0, len(keys))
	for _, k := range keys {
		s, err := c.Classify(ctx, k, files[k])
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, nil
}

// ScoreFeatures applies the weighted-sigmoid formula to features.
// Pre-sigmoid sum, then 100 * sigmoid, then band assignment.
func (c *Classifier) ScoreFeatures(f FileFeatures) Score {
	w := c.Weights
	type contrib struct {
		name, axis string
		raw, w     float64
	}
	parts := []contrib{
		{"path_pattern", "", f.PathPatternScore, w.PathPattern},
		{"llm_category", "", f.LLMCategoryScore, w.LLMCategory},
		{"fanin_centrality", "", f.FanInCentrality, w.FanInCentrality},
		{"incident_mention", "", f.IncidentMention, w.IncidentMention},
		{"slo_backing", "", f.SLOBacking, w.SLOBacking},
		{"review_intensity", "", f.ReviewIntensity, w.ReviewIntensity},
		{"cve_history", "", f.CVEHistory, w.CVEHistory},
		{"test_cov_gradient", "", f.TestCovGradient, w.TestCovGradient},
		{"comment_marker", "", f.CommentMarker, w.CommentMarker},
		{"codeowners", "", f.CodeOwnersScore, w.CodeOwnersScore},
	}
	sum := 0.0
	hits := make([]SignalHit, 0, len(parts)+1)
	for _, p := range parts {
		ctrb := p.w * p.raw
		sum += ctrb
		if math.Abs(ctrb) > 1e-9 {
			hits = append(hits, SignalHit{Name: p.name, Contribution: ctrb, RawValue: p.raw})
		}
	}
	// Negative term — UI or test penalty.
	penaltyContribution := -w.UIOrTestPenalty * f.UIOrTestPenalty
	sum += penaltyContribution
	if f.UIOrTestPenalty > 0 {
		hits = append(hits, SignalHit{Name: "ui_or_test_penalty", Contribution: penaltyContribution, RawValue: f.UIOrTestPenalty})
	}

	score := 100.0 * sigmoid(sum)

	// Round to one decimal so two close inputs don't oscillate band-wise.
	score = math.Round(score*10) / 10

	// Sort hits by |contribution| desc, keep top 3.
	sort.Slice(hits, func(i, j int) bool {
		return math.Abs(hits[i].Contribution) > math.Abs(hits[j].Contribution)
	})
	if len(hits) > 3 {
		hits = hits[:3]
	}

	return Score{
		Path:       f.Path,
		Score:      score,
		Band:       c.AssignBand(score),
		TopSignals: hits,
		Features:   f,
	}
}

// AssignBand maps a 0..100 score to a Band per Thresholds.
func (c *Classifier) AssignBand(score float64) Band {
	switch {
	case score >= c.Thresholds.Hot:
		return BandMolten
	case score >= c.Thresholds.Warm:
		return BandHot
	case score >= c.Thresholds.Cold:
		return BandWarm
	default:
		return BandCold
	}
}

// PRBand returns the band for an entire PR per the PR-level escalation
// rules in docs/06-research/tier3-trigger-automation.md §"PR-level trigger":
//
//   - Any file with S ≥ 80 → Molten
//   - ≥3 files with S ≥ 60 → Molten
//   - Diff contains security/money tokens AND is ≥40 lines → Hot (suggest Tier3)
//
// `diffLines` is the total added+modified line count of the PR.
func (c *Classifier) PRBand(scores []Score, diffLines int) Band {
	if len(scores) == 0 {
		return BandCold
	}
	hotCount := 0
	highestBand := BandCold
	hasSecurityMoneyToken := false
	for _, s := range scores {
		if s.Score >= c.Thresholds.Hot {
			return BandMolten
		}
		if s.Score >= c.Thresholds.Warm {
			hotCount++
		}
		if rankBand(s.Band) > rankBand(highestBand) {
			highestBand = s.Band
		}
		for _, sig := range s.TopSignals {
			if sig.Name == "path_pattern" && sig.RawValue > 0 {
				// path-pattern matched at least one axis; whether it's
				// security/money requires inspecting features.
				for axis := range s.Features.PathPatternAxes {
					if axis == AxisSecurity || axis == AxisMoney {
						hasSecurityMoneyToken = true
					}
				}
			}
		}
	}
	if hotCount >= 3 {
		return BandMolten
	}
	if hasSecurityMoneyToken && diffLines >= 40 {
		if highestBand == BandCold {
			return BandHot
		}
	}
	return highestBand
}

// rankBand orders bands.
func rankBand(b Band) int {
	switch b {
	case BandMolten:
		return 3
	case BandHot:
		return 2
	case BandWarm:
		return 1
	default:
		return 0
	}
}

// sigmoid is the classic logistic activation.
func sigmoid(x float64) float64 {
	return 1.0 / (1.0 + math.Exp(-x))
}

// AxisSummary aggregates which axes matched in a PR-level scope. The
// rubric prompt uses this to surface "this PR touches security AND money".
func AxisSummary(scores []Score) map[Axis]int {
	out := map[Axis]int{}
	for _, s := range scores {
		for a, hit := range s.Features.PathPatternAxes {
			if hit {
				out[a]++
			}
		}
	}
	return out
}

// FormatExplanation returns a human-readable one-liner for a score.
func FormatExplanation(s Score) string {
	var b strings.Builder
	b.WriteString(s.Path)
	b.WriteString(": ")
	b.WriteString(string(s.Band))
	b.WriteString(" (S≈")
	b.WriteString(formatFloat(s.Score))
	b.WriteString(")")
	if len(s.TopSignals) > 0 {
		b.WriteString(" — top: ")
		for i, h := range s.TopSignals {
			if i > 0 {
				b.WriteString(", ")
			}
			b.WriteString(h.Name)
		}
	}
	return b.String()
}

func formatFloat(f float64) string {
	if f == math.Trunc(f) {
		return strings.TrimSuffix(strings.TrimRight(strings.TrimSuffix(formatFixed(f, 1), "0"), "."), ".")
	}
	return formatFixed(f, 1)
}

func formatFixed(f float64, prec int) string {
	scale := math.Pow10(prec)
	r := math.Round(f*scale) / scale
	// %g doesn't always preserve trailing zeros; use a manual fixed format.
	intPart := int64(math.Floor(r))
	frac := int64(math.Round((r - float64(intPart)) * scale))
	if frac == 0 {
		return intToString(intPart)
	}
	return intToString(intPart) + "." + intToString(frac)
}

func intToString(i int64) string {
	if i == 0 {
		return "0"
	}
	neg := false
	if i < 0 {
		neg = true
		i = -i
	}
	var buf [20]byte
	pos := len(buf)
	for i > 0 {
		pos--
		buf[pos] = byte('0' + i%10)
		i /= 10
	}
	if neg {
		pos--
		buf[pos] = '-'
	}
	return string(buf[pos:])
}
