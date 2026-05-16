// Package ranker computes the A-MAC importance score and the final
// retrieval score that drives re-ranking.
//
// A-MAC = Adaptive Multi-dimensional Admission Control (per the Mem0
// state-of-memory report). The composite is:
//
//   importance = utility * confidence * novelty * recency * content_prior
//
// All factors are in [0, 1]; the composite is clamped to the same range.
// The retrieval final score blends semantic similarity with importance:
//
//   final = (1 - α) * semantic + α * importance
//
// where α defaults to 0.4 — semantic dominates but importance breaks
// ties and surfaces high-confidence procedural rules that may be
// embedding-distant from the literal query.
//
// Ebbinghaus exponential decay implements recency: r(age_d) = exp(-d / τ)
// with τ = 14 days for episodic; τ = 365 days for procedural (procedural
// rules age slowly, episodic memory aggressively).
package ranker

import (
	"math"
	"time"

	memoryspec "github.com/crucible/memory-spec/go"
	cruciblev1 "github.com/crucible/sdk-go/crucible/v1"
)

// Weights configure the A-MAC blend.
type Weights struct {
	// SemanticBlend (α): final = (1-α) * semantic + α * importance.
	SemanticBlend float64

	// Per-factor weights in the importance composite. Default 1.0 each
	// = pure product. Tunable per-tenant (paid-tier feature) without
	// recomputing the index.
	Utility      float64
	Confidence   float64
	Novelty      float64
	Recency      float64
	ContentPrior float64

	// Ebbinghaus τ in days.
	EpisodicTauDays    float64
	ProceduralTauDays  float64
}

// Default returns sensible defaults matching the docs.
func Default() Weights {
	return Weights{
		SemanticBlend:     0.4,
		Utility:           1.0,
		Confidence:        1.0,
		Novelty:           1.0,
		Recency:           1.0,
		ContentPrior:      1.0,
		EpisodicTauDays:   14,
		ProceduralTauDays: 365,
	}
}

// Score is the full breakdown returned for telemetry.
type Score struct {
	Importance memoryspec.AdmissionScore
	Final      float64
}

// ─── Per-factor scorers ─────────────────────────────────────────────────────

// Utility is "how likely is this memory to be useful for the current
// task?". For procedural rules we use the rule's confidence as a proxy
// (a high-confidence rule is generally useful when in scope); for
// episodic we use the historical reinforcement signal carried in
// importance.
func utilityFactor(m cruciblev1.Memory) float64 {
	// Reinforce-on-access — we encode it as a small bump on raw
	// importance to disambiguate cold vs. frequently-recalled items.
	return clamp(m.Importance)
}

// Confidence is the convention's stored confidence (procedural) or 1.0
// for non-procedural memories (we have no other signal).
func confidenceFactor(m cruciblev1.Memory) float64 {
	if m.Kind != cruciblev1.MemProcedural {
		return 1.0
	}
	// Importance carries the calibrated confidence for procedural
	// memories — the distiller writes it that way.
	return clamp(m.Importance)
}

// Novelty is the inverse of how often the same content has been
// recalled. Frequently-recalled items get a small novelty penalty so
// the router doesn't dump the same five rules into every prompt.
func noveltyFactor(recallCount uint32) float64 {
	// 1 / log(2 + n). Caps the penalty so re-recalled items still
	// surface, just lower.
	return 1.0 / math.Log(float64(recallCount)+2.0)
}

// Recency applies Ebbinghaus decay on the time since the memory was
// last reinforced (or written, whichever is later).
func recencyFactor(lastRecalled time.Time, kind cruciblev1.MemoryKind, w Weights) float64 {
	if lastRecalled.IsZero() {
		return 0.0
	}
	tau := w.EpisodicTauDays
	if kind == cruciblev1.MemProcedural {
		tau = w.ProceduralTauDays
	}
	if tau <= 0 {
		tau = 1
	}
	ageDays := time.Since(lastRecalled).Hours() / 24.0
	if ageDays < 0 {
		ageDays = 0
	}
	return math.Exp(-ageDays / tau)
}

// ContentPrior is a per-kind constant boost. Procedural rules get the
// highest prior because they're authored intent; semantic snippets the
// lowest (they're the noisiest input).
func contentPriorFactor(kind cruciblev1.MemoryKind) float64 {
	switch kind {
	case cruciblev1.MemProcedural:
		return 1.0
	case cruciblev1.MemEpisodic:
		return 0.85
	case cruciblev1.MemSemantic:
		return 0.7
	case cruciblev1.MemHot:
		return 0.95
	}
	return 0.5
}

// ─── Compose ────────────────────────────────────────────────────────────────

// Compute returns the full Score for a memory + its semantic similarity
// to the query. recall_count is how many times this memory has been
// surfaced before (drives novelty); pass 0 if unknown.
func Compute(m cruciblev1.Memory, semantic float64, recallCount uint32, w Weights) Score {
	u := utilityFactor(m) * w.Utility
	c := confidenceFactor(m) * w.Confidence
	n := noveltyFactor(recallCount) * w.Novelty
	r := recencyFactor(m.LastRecalled, m.Kind, w) * w.Recency
	p := contentPriorFactor(m.Kind) * w.ContentPrior

	// Clamp each factor to [0,1] post-weighting so weights >1 don't
	// blow past the unit interval; product stays valid.
	u = clamp(u)
	c = clamp(c)
	n = clamp(n)
	r = clamp(r)
	p = clamp(p)

	composite := u * c * n * r * p
	composite = clamp(composite)

	alpha := w.SemanticBlend
	if alpha < 0 {
		alpha = 0
	}
	if alpha > 1 {
		alpha = 1
	}
	final := (1.0-alpha)*clamp(semantic) + alpha*composite

	return Score{
		Importance: memoryspec.AdmissionScore{
			Utility:      u,
			Confidence:   c,
			Novelty:      n,
			Recency:      r,
			ContentPrior: p,
			Composite:    composite,
			Admitted:     composite >= 0.25, // matches surface threshold
			Threshold:    thresholdLabel(composite),
		},
		Final: final,
	}
}

// thresholdLabel maps an importance score to the bucket labels used in
// the cartographer + onboarding UI ("active" / "suggested" / "candidate").
func thresholdLabel(c float64) string {
	switch {
	case c >= 0.7:
		return "active"
	case c >= 0.4:
		return "suggested"
	case c >= 0.25:
		return "candidate"
	}
	return "rejected"
}

func clamp(v float64) float64 {
	switch {
	case math.IsNaN(v):
		return 0
	case v < 0:
		return 0
	case v > 1:
		return 1
	}
	return v
}
