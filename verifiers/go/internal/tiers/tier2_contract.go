// Tier 2 — contract testing via schemathesis (pip-installed; same
// shell-out pattern the Python and TypeScript runners use).
//
// schemathesis is language-agnostic: it derives stateful test
// workflows from an OpenAPI 3 / GraphQL spec and replays them against
// a live (sandboxed) service. For Go diffs we only run Tier 2 when
// the request carries SpecChanges; otherwise we emit a "skipped"
// verdict (contract testing isn't applicable to pure-business-logic
// diffs).
//
// Wall-clock budget: 15 min default per verifier-pipeline.md §Tier 2.
package tiers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"

	"github.com/crucible/verifier/pkg/testreport"

	"github.com/crucible/verify-go/internal/schema"
)

// ContractConfig is the Tier 2 runner config.
type ContractConfig struct {
	WorkDir     string
	SpecChanges []schema.SpecChange
	// BaseURL is the in-sandbox service URL schemathesis hits.
	// Empty means schemathesis dry-runs (lint mode only).
	BaseURL string
	Binary  string // defaults to "schemathesis"
	Timeout time.Duration
}

// RunContract invokes schemathesis once per OpenAPI/GraphQL spec
// change in the diff and folds the results into a single ContractStats.
func RunContract(ctx context.Context, cfg ContractConfig) (*testreport.TestReport, error) {
	started := time.Now()

	binary := cfg.Binary
	if binary == "" {
		binary = "schemathesis"
	}

	report := &testreport.TestReport{
		SchemaVersion: testreport.SchemaVersion,
		Tier:          testreport.TierContract,
		Language:      testreport.LangGo,
		Framework:     "schemathesis",
		StartedAt:     started,
		Contract:      &testreport.ContractStats{},
	}

	if len(cfg.SpecChanges) == 0 {
		report.Verdict = testreport.VerdictSkipped
		report.Passed = true
		report.FinishedAt = time.Now()
		report.DurationSeconds = time.Since(started).Seconds()
		return report, nil
	}
	if _, err := exec.LookPath(binary); err != nil {
		report.Verdict = testreport.VerdictToolUnavailable
		report.Passed = false
		report.Error = fmt.Sprintf("schemathesis not on PATH: %v", err)
		report.FinishedAt = time.Now()
		report.DurationSeconds = time.Since(started).Seconds()
		return report, nil
	}

	cctx := ctx
	if cfg.Timeout > 0 {
		var cancel context.CancelFunc
		cctx, cancel = context.WithTimeout(ctx, cfg.Timeout)
		defer cancel()
	}

	allPassed := true
	for _, sc := range cfg.SpecChanges {
		if sc.Kind != "" && sc.Kind != "openapi" && sc.Kind != "graphql" {
			// schemathesis only handles OpenAPI/GraphQL today.
			continue
		}
		args := []string{"run", "--checks", "all", "--output=json"}
		if cfg.BaseURL != "" {
			args = append(args, "--base-url", cfg.BaseURL)
		}
		args = append(args, sc.Path)

		cmd := exec.CommandContext(cctx, binary, args...)
		cmd.Dir = cfg.WorkDir
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		err := cmd.Run()

		violations, parsed := parseSchemathesisJSON(stdout.Bytes())
		report.Contract.SpecPath = sc.Path
		report.Contract.SpecHash = sc.CurrentHash
		report.Contract.StatefulWorkflows += parsed.Workflows
		report.Contract.Checks = mergeStrings(report.Contract.Checks, parsed.Checks)
		report.Contract.Violations = append(report.Contract.Violations, violations...)

		if err != nil {
			allPassed = false
			if cctx.Err() == context.DeadlineExceeded {
				report.Verdict = testreport.VerdictTimedOut
				report.Error = fmt.Sprintf("schemathesis timed out after %s", cfg.Timeout)
				report.Passed = false
				report.FinishedAt = time.Now()
				report.DurationSeconds = time.Since(started).Seconds()
				return report, nil
			}
			if len(violations) == 0 {
				// Non-zero with no violations parsed — schemathesis
				// itself crashed. Surface stderr for triage.
				report.Findings = append(report.Findings, testreport.Finding{
					Category: "contract_runner_error",
					Severity: "error",
					Detail:   "schemathesis exited non-zero: " + truncate(stderr.String(), 400),
				})
			}
		}
	}

	for _, v := range report.Contract.Violations {
		report.Findings = append(report.Findings, testreport.Finding{
			Category: "contract_violation",
			Severity: "error",
			File:     report.Contract.SpecPath,
			Detail:   fmt.Sprintf("%s %s check=%s — %s", v.Method, v.Endpoint, v.Check, v.Detail),
		})
	}

	report.FinishedAt = time.Now()
	report.DurationSeconds = time.Since(started).Seconds()
	if !allPassed || len(report.Contract.Violations) > 0 {
		report.Verdict = testreport.VerdictFailed
		report.Passed = false
	} else {
		report.Verdict = testreport.VerdictPassed
		report.Passed = true
	}
	return report, nil
}

// parsedSchemathesis is the relaxed shape we expect from
// `schemathesis run --output=json`. We don't pin to a strict shape
// because schemathesis's JSON schema has shifted across minor
// versions; we only extract the fields we attach to ContractStats.
type parsedSchemathesis struct {
	Workflows int
	Checks    []string
}

func parseSchemathesisJSON(raw []byte) ([]testreport.ContractViolation, parsedSchemathesis) {
	var generic map[string]any
	if err := json.Unmarshal(raw, &generic); err != nil {
		return nil, parsedSchemathesis{}
	}
	var violations []testreport.ContractViolation
	parsed := parsedSchemathesis{}

	if checks, ok := generic["checks"].([]any); ok {
		for _, c := range checks {
			if s, ok := c.(string); ok {
				parsed.Checks = append(parsed.Checks, s)
			}
		}
	}
	if results, ok := generic["results"].([]any); ok {
		parsed.Workflows = len(results)
		for _, r := range results {
			rm, ok := r.(map[string]any)
			if !ok {
				continue
			}
			endpoint, _ := rm["path"].(string)
			method, _ := rm["method"].(string)
			if errs, ok := rm["errors"].([]any); ok {
				for _, e := range errs {
					em, ok := e.(map[string]any)
					if !ok {
						continue
					}
					check, _ := em["check"].(string)
					detail, _ := em["message"].(string)
					violations = append(violations, testreport.ContractViolation{
						Endpoint: endpoint,
						Method:   method,
						Check:    check,
						Detail:   detail,
					})
				}
			}
		}
	}
	return violations, parsed
}

func mergeStrings(a, b []string) []string {
	seen := map[string]struct{}{}
	out := append([]string{}, a...)
	for _, s := range a {
		seen[s] = struct{}{}
	}
	for _, s := range b {
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}
