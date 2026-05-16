package grade

import "testing"

func TestAggregateBasics(t *testing.T) {
	results := []CaseResult{
		{ID: "a", Category: "greenfield", Passed: true, CostUSD: 1.5, WallClockSec: 200, CacheHitRate: 0.7},
		{ID: "b", Category: "greenfield", Passed: true, CostUSD: 2.0, WallClockSec: 300, CacheHitRate: 0.8},
		{ID: "c", Category: "adversarial", Passed: true, CostUSD: 0.5},
		{ID: "d", Category: "adversarial", Passed: false, Reason: "missed", SafetyIncidents: 1},
	}
	r := Aggregate(results)
	if r.TotalCases != 4 || r.Passed != 3 {
		t.Errorf("totals: %+v", r)
	}
	if r.AllPassed {
		t.Error("AllPassed should be false")
	}
	if r.SafetyIncidentsTotal != 1 {
		t.Errorf("safety=%d", r.SafetyIncidentsTotal)
	}
	gf := r.PerCategory["greenfield"]
	if gf.PassRate != 1.0 || !gf.MeetsTarget {
		t.Errorf("greenfield: %+v", gf)
	}
	adv := r.PerCategory["adversarial"]
	if adv.MeetsTarget {
		t.Errorf("adversarial 50%% should NOT meet target")
	}
}

func TestMedian(t *testing.T) {
	if median([]float64{1, 2, 3}) != 2 {
		t.Error("median odd")
	}
	if median([]float64{1, 2, 3, 4}) != 2.5 {
		t.Error("median even")
	}
	if median(nil) != 0 {
		t.Error("median nil")
	}
}
