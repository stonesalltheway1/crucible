// Package grade aggregates per-case results into a release report.
package grade

import (
	"sort"
	"time"
)

// CaseResult is the per-case grading output.
type CaseResult struct {
	ID                string    `json:"id"`
	Category          string    `json:"category"`
	Stack             string    `json:"stack"`
	Passed            bool      `json:"passed"`
	Reason            string    `json:"reason,omitempty"`
	VerifierVerdict   string    `json:"verifier_verdict,omitempty"`
	Tier3Triggered    bool      `json:"tier3_triggered,omitempty"`
	DestructiveOpGate bool      `json:"destructive_op_gate,omitempty"`
	CostUSD           float64   `json:"cost_usd"`
	TokensTotal       int       `json:"tokens_total"`
	CacheHitRate      float64   `json:"cache_hit_rate"`
	SafetyIncidents   int       `json:"safety_incidents"`
	StartedAt         time.Time `json:"started_at"`
	CompletedAt       time.Time `json:"completed_at"`
	WallClockSec      float64   `json:"wall_clock_sec"`
}

// Report is the release-level rollup.
type Report struct {
	GeneratedAt          time.Time            `json:"generated_at"`
	TotalCases           int                  `json:"total_cases"`
	Passed               int                  `json:"passed"`
	AllPassed            bool                 `json:"all_passed"`
	PerCategory          map[string]CategorySummary `json:"per_category"`
	MedianCostUSD        float64              `json:"median_cost_usd"`
	MedianWallClockSec   float64              `json:"median_wall_clock_sec"`
	MedianCacheHitRate   float64              `json:"median_cache_hit_rate"`
	SafetyIncidentsTotal int                  `json:"safety_incidents_total"`
	Failures             []CaseResult         `json:"failures,omitempty"`
}

// CategorySummary aggregates one category.
type CategorySummary struct {
	Total    int     `json:"total"`
	Passed   int     `json:"passed"`
	PassRate float64 `json:"pass_rate"`
	TargetPassRate float64 `json:"target_pass_rate"`
	MeetsTarget bool `json:"meets_target"`
}

// Targets per category.
var categoryTarget = map[string]float64{
	"greenfield":    0.95,
	"feature-add":   0.90,
	"refactor":      0.80,
	"critical-path": 0.85,
	"adversarial":   1.00,
	"regression":    1.00,
}

// Aggregate rolls up a slice of results into a Report.
func Aggregate(results []CaseResult) Report {
	r := Report{PerCategory: map[string]CategorySummary{}}
	r.TotalCases = len(results)
	costs := []float64{}
	walls := []float64{}
	caches := []float64{}
	for _, c := range results {
		if c.Passed {
			r.Passed++
		} else {
			r.Failures = append(r.Failures, c)
		}
		r.SafetyIncidentsTotal += c.SafetyIncidents
		cs := r.PerCategory[c.Category]
		cs.Total++
		if c.Passed {
			cs.Passed++
		}
		r.PerCategory[c.Category] = cs
		costs = append(costs, c.CostUSD)
		walls = append(walls, c.WallClockSec)
		caches = append(caches, c.CacheHitRate)
	}
	for k, cs := range r.PerCategory {
		if cs.Total > 0 {
			cs.PassRate = float64(cs.Passed) / float64(cs.Total)
		}
		cs.TargetPassRate = categoryTarget[k]
		cs.MeetsTarget = cs.PassRate >= cs.TargetPassRate
		r.PerCategory[k] = cs
	}
	r.MedianCostUSD = median(costs)
	r.MedianWallClockSec = median(walls)
	r.MedianCacheHitRate = median(caches)
	r.AllPassed = r.Passed == r.TotalCases
	return r
}

func median(in []float64) float64 {
	if len(in) == 0 {
		return 0
	}
	s := append([]float64(nil), in...)
	sort.Float64s(s)
	n := len(s)
	if n%2 == 1 {
		return s[n/2]
	}
	return (s[n/2-1] + s[n/2]) / 2
}
