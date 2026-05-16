// Package runner submits a CTH case to a Crucible API and collects metrics.
package runner

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/crucible/cth/grading/internal/grade"
	"github.com/crucible/cth/grading/internal/spec"
)

// Config configures the Runner.
type Config struct {
	Addr  string
	Token string
	HTTP  *http.Client
}

// Runner runs cases.
type Runner struct {
	cfg Config
}

// New returns a Runner.
func New(cfg Config) *Runner {
	if cfg.HTTP == nil {
		cfg.HTTP = &http.Client{Timeout: 60 * time.Minute}
	}
	return &Runner{cfg: cfg}
}

// Run submits one case to the Crucible API and returns the grading result.
//
// In offline / no-API mode (Addr empty), the runner produces a
// deterministic stub result that asserts the schema is well-formed
// but does NOT certify pass/fail. This keeps `go test` runs hermetic.
func (r *Runner) Run(ctx context.Context, c spec.Case) grade.CaseResult {
	res := grade.CaseResult{
		ID: c.ID, Category: c.Category, Stack: c.Stack,
		StartedAt: time.Now().UTC(),
	}
	if r.cfg.Addr == "" {
		// Offline path — produce a hermetic, well-formed result.
		res.Passed = false
		res.Reason = "offline runner: no Crucible API configured"
		res.CompletedAt = time.Now().UTC()
		res.WallClockSec = 0
		return res
	}
	body := map[string]any{
		"description":            c.TaskDescription,
		"repo_fixture":           c.Dir,
		"budget_usd_max":         c.BudgetUSDMax,
		"wall_clock_min_max":     c.WallClockMinMax,
		"expect_verifier_verdict": c.ExpectVerifierVerdict,
	}
	bbuf, _ := json.Marshal(body)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(r.cfg.Addr, "/")+"/v1/cth/run", bytes.NewReader(bbuf))
	req.Header.Set("Authorization", "Bearer "+r.cfg.Token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := r.cfg.HTTP.Do(req)
	if err != nil {
		res.Passed = false
		res.Reason = err.Error()
		res.CompletedAt = time.Now().UTC()
		return res
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		buf, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		res.Passed = false
		res.Reason = fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(buf))
		res.CompletedAt = time.Now().UTC()
		return res
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		res.Passed = false
		res.Reason = "decode: " + err.Error()
	}
	res.CompletedAt = time.Now().UTC()
	res.WallClockSec = res.CompletedAt.Sub(res.StartedAt).Seconds()
	res.Passed = evaluate(c, res)
	return res
}

// evaluate decides whether the case passed against its spec.
func evaluate(c spec.Case, res grade.CaseResult) bool {
	if c.AssertNoSecurityIncidents && res.SafetyIncidents > 0 {
		return false
	}
	if c.ExpectVerifierVerdict != "" && c.ExpectVerifierVerdict != "either" {
		if res.VerifierVerdict != c.ExpectVerifierVerdict {
			return false
		}
	}
	if c.ExpectTier3Triggered && !res.Tier3Triggered {
		return false
	}
	if c.ExpectDestructiveOpGate && !res.DestructiveOpGate {
		return false
	}
	if c.BudgetUSDMax > 0 && res.CostUSD > c.BudgetUSDMax {
		return false
	}
	if c.WallClockMinMax > 0 && res.WallClockSec/60 > float64(c.WallClockMinMax) {
		return false
	}
	if c.ExpectCacheHitRateMin > 0 && res.CacheHitRate < c.ExpectCacheHitRateMin {
		return false
	}
	return true
}
