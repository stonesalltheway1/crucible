package ranker

import (
	"math"
	"testing"
	"time"

	cruciblev1 "github.com/crucible/sdk-go/crucible/v1"
)

func TestCompute_ProceduralOutscoresEpisodicOnEqualSemantic(t *testing.T) {
	now := time.Now()
	w := Default()
	proc := cruciblev1.Memory{Kind: cruciblev1.MemProcedural, Importance: 0.9, LastRecalled: now}
	epis := cruciblev1.Memory{Kind: cruciblev1.MemEpisodic, Importance: 0.9, LastRecalled: now}
	pScore := Compute(proc, 0.6, 0, w)
	eScore := Compute(epis, 0.6, 0, w)
	if pScore.Final <= eScore.Final {
		t.Fatalf("procedural prior should outrank episodic; proc=%.3f epis=%.3f", pScore.Final, eScore.Final)
	}
}

func TestCompute_EbbinghausPenalizesOlderItems(t *testing.T) {
	w := Default()
	recent := cruciblev1.Memory{Kind: cruciblev1.MemEpisodic, Importance: 0.8, LastRecalled: time.Now()}
	old := cruciblev1.Memory{Kind: cruciblev1.MemEpisodic, Importance: 0.8, LastRecalled: time.Now().Add(-60 * 24 * time.Hour)}
	r := Compute(recent, 0.5, 0, w)
	o := Compute(old, 0.5, 0, w)
	if r.Final <= o.Final {
		t.Fatalf("recent must outrank 60-day-old; recent=%.3f old=%.3f", r.Final, o.Final)
	}
}

func TestCompute_NoveltyPenaltyForRepeatRecalls(t *testing.T) {
	w := Default()
	m := cruciblev1.Memory{Kind: cruciblev1.MemEpisodic, Importance: 0.8, LastRecalled: time.Now()}
	fresh := Compute(m, 0.5, 0, w)
	hot := Compute(m, 0.5, 100, w)
	if fresh.Final <= hot.Final {
		t.Fatal("100-recall novelty penalty should outrank 0-recall")
	}
}

func TestCompute_ClampsOutOfRange(t *testing.T) {
	w := Default()
	w.Utility = 10 // weights >1 must not push composite past 1
	m := cruciblev1.Memory{Kind: cruciblev1.MemProcedural, Importance: 1.0, LastRecalled: time.Now()}
	s := Compute(m, 1.0, 0, w)
	if s.Importance.Composite > 1.0 || s.Final > 1.0 {
		t.Fatal("scores must clamp to [0,1] even with high weights")
	}
}

func TestCompute_ZeroLastRecalledGivesZeroRecency(t *testing.T) {
	w := Default()
	m := cruciblev1.Memory{Kind: cruciblev1.MemEpisodic, Importance: 0.8}
	s := Compute(m, 0.5, 0, w)
	if s.Importance.Recency != 0 {
		t.Fatalf("zero LastRecalled should produce recency=0, got %.3f", s.Importance.Recency)
	}
}

func TestCompute_HandlesNaNGracefully(t *testing.T) {
	w := Default()
	m := cruciblev1.Memory{Kind: cruciblev1.MemSemantic, Importance: math.NaN(), LastRecalled: time.Now()}
	s := Compute(m, math.NaN(), 0, w)
	if math.IsNaN(s.Final) {
		t.Fatal("NaN inputs must be clamped, not propagated")
	}
}

func TestThresholdLabel_BoundaryConditions(t *testing.T) {
	tests := []struct {
		v    float64
		want string
	}{
		{0.95, "active"},
		{0.7, "active"},
		{0.5, "suggested"},
		{0.4, "suggested"},
		{0.3, "candidate"},
		{0.25, "candidate"},
		{0.1, "rejected"},
	}
	for _, tc := range tests {
		if got := thresholdLabel(tc.v); got != tc.want {
			t.Fatalf("threshold(%.2f) = %q, want %q", tc.v, got, tc.want)
		}
	}
}
