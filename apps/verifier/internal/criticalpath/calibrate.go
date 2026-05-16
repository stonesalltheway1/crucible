package criticalpath

import (
	"encoding/json"
	"errors"
	"math"
	"sort"
)

// CalibrationLabel is one engineer-labeled example used to fit per-tenant
// weights. Defined in docs/06-research/tier3-trigger-automation.md
// §"Calibration".
type CalibrationLabel struct {
	Path     string          `json:"path"`
	Features FileFeatures    `json:"features"`
	Label    LabelKind       `json:"label"`
	Reason   string          `json:"reason,omitempty"`
	LabelledBy string        `json:"labelled_by,omitempty"`
	LabelledAt int64         `json:"labelled_at,omitempty"`
}

// LabelKind enumerates the engineer's verdict for a sample.
type LabelKind string

const (
	LabelCritical      LabelKind = "critical"
	LabelWarm          LabelKind = "warm"
	LabelCold          LabelKind = "cold"
	LabelNotApplicable LabelKind = "not_applicable"
)

// CalibrationResult is the fitted output of LogisticFit.
type CalibrationResult struct {
	Weights       Weights `json:"weights"`
	Bias          float64 `json:"bias"`
	AccuracyOnSet float64 `json:"accuracy_on_set"`
	PositiveCount int     `json:"positive_count"`
	NegativeCount int     `json:"negative_count"`
	Iterations    int     `json:"iterations"`
}

// LogisticFit fits the weight vector against engineer labels via batch
// gradient descent on the binary cross-entropy loss.
//
// We map labels:
//
//	LabelCritical → y=1
//	LabelWarm     → y=0.66
//	LabelCold     → y=0
//	LabelNotApplicable → skipped
//
// This is a soft-target regression with a sigmoid head — equivalent to
// logistic regression with continuous labels.
func LogisticFit(samples []CalibrationLabel, prior Weights, lr float64, iterations int) (CalibrationResult, error) {
	if len(samples) < 10 {
		return CalibrationResult{}, errors.New("criticalpath/calibrate: need at least 10 labelled samples")
	}
	if lr <= 0 {
		lr = 0.05
	}
	if iterations <= 0 {
		iterations = 200
	}

	type sample struct {
		x      []float64 // [path_pattern, llm, fanin, incident, slo, review, cve, testcov, comment, codeowners, uitest]
		y      float64
	}
	build := func(f FileFeatures) []float64 {
		return []float64{
			f.PathPatternScore,
			f.LLMCategoryScore,
			f.FanInCentrality,
			f.IncidentMention,
			f.SLOBacking,
			f.ReviewIntensity,
			f.CVEHistory,
			f.TestCovGradient,
			f.CommentMarker,
			f.CodeOwnersScore,
			f.UIOrTestPenalty,
		}
	}

	xs := make([]sample, 0, len(samples))
	pos, neg := 0, 0
	for _, s := range samples {
		if s.Label == LabelNotApplicable {
			continue
		}
		var y float64
		switch s.Label {
		case LabelCritical:
			y = 1.0
			pos++
		case LabelWarm:
			y = 0.66
			pos++
		case LabelCold:
			y = 0.0
			neg++
		default:
			continue
		}
		xs = append(xs, sample{x: build(s.Features), y: y})
	}
	if len(xs) == 0 {
		return CalibrationResult{}, errors.New("criticalpath/calibrate: no usable labels")
	}

	// Initialise from prior weights — convert struct to vector.
	w := []float64{
		prior.PathPattern,
		prior.LLMCategory,
		prior.FanInCentrality,
		prior.IncidentMention,
		prior.SLOBacking,
		prior.ReviewIntensity,
		prior.CVEHistory,
		prior.TestCovGradient,
		prior.CommentMarker,
		prior.CodeOwnersScore,
		-prior.UIOrTestPenalty, // signed: ui/test penalty is a negative weight
	}
	bias := 0.0

	// Batch gradient descent. Loss = -[y*log(p) + (1-y)*log(1-p)].
	for it := 0; it < iterations; it++ {
		var grad [11]float64
		var gradBias float64
		for _, s := range xs {
			z := bias
			for j, xj := range s.x {
				z += w[j] * xj
			}
			p := sigmoid(z)
			err := p - s.y
			for j, xj := range s.x {
				grad[j] += err * xj
			}
			gradBias += err
		}
		n := float64(len(xs))
		for j := range w {
			w[j] -= lr * grad[j] / n
		}
		bias -= lr * gradBias / n
	}

	// Map back to Weights — apply asymmetric-cost prior: never let
	// signal weights go below zero (signals are non-decreasing in their
	// associated criticality). UI penalty stays signed.
	clamp := func(v float64) float64 {
		if v < 0 {
			return 0
		}
		return v
	}
	out := Weights{
		PathPattern:     clamp(w[0]),
		LLMCategory:     clamp(w[1]),
		FanInCentrality: clamp(w[2]),
		IncidentMention: clamp(w[3]),
		SLOBacking:      clamp(w[4]),
		ReviewIntensity: clamp(w[5]),
		CVEHistory:      clamp(w[6]),
		TestCovGradient: clamp(w[7]),
		CommentMarker:   clamp(w[8]),
		CodeOwnersScore: clamp(w[9]),
		UIOrTestPenalty: -math.Min(0, w[10]), // negate the (still-negative) coefficient
	}

	// Asymmetric cost prior: bias toward over-escalation. Bump
	// every signal coefficient by 5% of the default so a noisy
	// fit can't drop us below the prior.
	out = blendWithPrior(out, prior, 0.95)

	// Accuracy on training set.
	acc := 0.0
	for _, s := range xs {
		z := bias
		for j, xj := range s.x {
			z += w[j] * xj
		}
		p := sigmoid(z)
		wantPos := s.y >= 0.5
		gotPos := p >= 0.5
		if wantPos == gotPos {
			acc++
		}
	}
	acc /= float64(len(xs))

	return CalibrationResult{
		Weights:       out,
		Bias:          bias,
		AccuracyOnSet: acc,
		PositiveCount: pos,
		NegativeCount: neg,
		Iterations:    iterations,
	}, nil
}

// blendWithPrior returns alpha*fitted + (1-alpha)*prior — used to soften
// noisy fits. alpha=1 ignores the prior; alpha=0 keeps only the prior.
func blendWithPrior(fitted, prior Weights, alpha float64) Weights {
	b := func(f, p float64) float64 { return alpha*f + (1-alpha)*p }
	return Weights{
		PathPattern:     b(fitted.PathPattern, prior.PathPattern),
		LLMCategory:     b(fitted.LLMCategory, prior.LLMCategory),
		FanInCentrality: b(fitted.FanInCentrality, prior.FanInCentrality),
		IncidentMention: b(fitted.IncidentMention, prior.IncidentMention),
		SLOBacking:      b(fitted.SLOBacking, prior.SLOBacking),
		ReviewIntensity: b(fitted.ReviewIntensity, prior.ReviewIntensity),
		CVEHistory:      b(fitted.CVEHistory, prior.CVEHistory),
		TestCovGradient: b(fitted.TestCovGradient, prior.TestCovGradient),
		CommentMarker:   b(fitted.CommentMarker, prior.CommentMarker),
		CodeOwnersScore: b(fitted.CodeOwnersScore, prior.CodeOwnersScore),
		UIOrTestPenalty: b(fitted.UIOrTestPenalty, prior.UIOrTestPenalty),
	}
}

// SerializeWeights returns the JSON encoding for storage in .crucible/calibration.toml.
func SerializeWeights(w Weights) ([]byte, error) {
	return json.MarshalIndent(w, "", "  ")
}

// DeserializeWeights reads a Weights JSON blob.
func DeserializeWeights(data []byte) (Weights, error) {
	var w Weights
	if err := json.Unmarshal(data, &w); err != nil {
		return Weights{}, err
	}
	return w, nil
}

// StratifiedSample returns N samples evenly distributed across the score
// bands so the calibration labeling effort hits cold/warm/hot/molten
// proportionally (50 obvious-critical, 50 obvious-non-critical, 100 ambiguous).
func StratifiedSample(scores []Score, n int) []Score {
	if n <= 0 || len(scores) == 0 {
		return nil
	}
	groups := map[Band][]Score{
		BandMolten: nil,
		BandHot:    nil,
		BandWarm:   nil,
		BandCold:   nil,
	}
	for _, s := range scores {
		groups[s.Band] = append(groups[s.Band], s)
	}
	for k := range groups {
		sort.Slice(groups[k], func(i, j int) bool { return groups[k][i].Path < groups[k][j].Path })
	}
	// Allocation: 25%/25%/25%/25% — the calling caller is free to override.
	perBand := n / 4
	out := make([]Score, 0, n)
	for _, b := range []Band{BandMolten, BandHot, BandWarm, BandCold} {
		g := groups[b]
		take := perBand
		if take > len(g) {
			take = len(g)
		}
		out = append(out, g[:take]...)
	}
	// Fill the remainder from whichever band has spare capacity.
	if len(out) < n {
		for _, b := range []Band{BandHot, BandWarm, BandMolten, BandCold} {
			g := groups[b]
			start := perBand
			if start > len(g) {
				start = len(g)
			}
			for _, s := range g[start:] {
				if len(out) >= n {
					return out
				}
				out = append(out, s)
			}
		}
	}
	return out
}
