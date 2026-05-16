// Package tiers implements the per-tier execution logic for the Go
// per-language verifier. Each tier exposes a Run function that takes
// a runtime Context and returns a partially-populated TestReport
// (the caller in main.go fills in Tier/Language/Task identifying
// fields and emits the wire delimiter).
//
// Tier 0 — mutation testing via avito-tech/go-mutesting v2.3.1.
//
// IMPORTANT semantic note about go-mutesting:
//
//   go-mutesting reports each surviving mutant as "PASS" and each
//   killed mutant as "FAIL". This is INVERTED relative to mutmut
//   (Python) and stryker (JS), which report killed=passed-from-the-
//   defender's perspective. We re-invert here so the TestReport's
//   MutationStats.Killed and .Survived match the schema's convention:
//
//     killed   = mutants the test suite caught   (i.e. test failed)
//     survived = mutants the test suite missed   (i.e. test passed)
//
// Threshold note: the brand-promise prompt asks for 0.75. The May
// 2026 research consensus on realistic mutation-score targets for Go
// is closer to 0.60 because go-mutesting's mutator catalog is
// narrower than mutmut/stryker. We hold the line at 0.75 because
// Crucible's verifier ladder over-indexes on Tier 1/3 anyway — Tier
// 0 is a smoke check, not the final word.
package tiers

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/crucible/verifier/pkg/testreport"
)

// Tier0Threshold is the Crucible mutation-score gate for Go (the
// brand-promise rubric value). Survives are flagged below this.
const Tier0Threshold = 0.75

// Tier0RealisticTarget is the documented "what's actually achievable
// with go-mutesting in May 2026" number. Reported in findings so the
// rubric LLM-judge can soften its rejection language when the score
// lands in [RealisticTarget, Tier0Threshold).
const Tier0RealisticTarget = 0.60

// MutationConfig is the Tier 0 runner config.
type MutationConfig struct {
	// WorkDir is the materialised source tree.
	WorkDir string
	// SourcePaths is the diff-scoped Go file list (no test files).
	SourcePaths []string
	// Binary is the go-mutesting CLI; defaults to "go-mutesting".
	Binary string
	// ExtraArgs is appended verbatim (typically "--config=...").
	ExtraArgs []string
	// Threshold overrides Tier0Threshold if non-zero.
	Threshold float64
	// Timeout caps the whole invocation.
	Timeout time.Duration
}

// Tier0Stats is the parsed go-mutesting result, ready to fold into
// MutationStats.
type Tier0Stats struct {
	Killed   int
	Survived int
	Total    int
	Score    float64
	// Survived enumerates each "PASS" line for findings reporting.
	SurvivedMutants []testreport.SurvivedMutant
	MutatedFiles    []string
	RawOutput       string
}

// RunMutation invokes go-mutesting against cfg.SourcePaths and parses
// its text output. Returns a partial TestReport (caller fills in
// task/tier/language headers).
//
// When go-mutesting is not on PATH we emit Verdict=tool_unavailable
// rather than failing — the dispatcher will degrade to the coverage+
// LLM-judge fallback documented in verifier-pipeline.md §Tier 0.
func RunMutation(ctx context.Context, cfg MutationConfig) (*testreport.TestReport, error) {
	started := time.Now()

	binary := cfg.Binary
	if binary == "" {
		binary = "go-mutesting"
	}
	threshold := cfg.Threshold
	if threshold == 0 {
		threshold = Tier0Threshold
	}

	report := &testreport.TestReport{
		SchemaVersion: testreport.SchemaVersion,
		Tier:          testreport.TierMutation,
		Language:      testreport.LangGo,
		Framework:     "go-mutesting",
		StartedAt:     started,
		Mutation: &testreport.MutationStats{
			DiffScoped: true,
			Threshold:  threshold,
		},
	}

	if len(cfg.SourcePaths) == 0 {
		report.Verdict = testreport.VerdictSkipped
		report.Passed = true
		report.FinishedAt = time.Now()
		report.DurationSeconds = time.Since(started).Seconds()
		return report, nil
	}

	if _, err := exec.LookPath(binary); err != nil {
		report.Verdict = testreport.VerdictToolUnavailable
		report.Passed = false
		report.Error = fmt.Sprintf("go-mutesting not on PATH: %v", err)
		report.FinishedAt = time.Now()
		report.DurationSeconds = time.Since(started).Seconds()
		return report, nil
	}

	args := []string{}
	args = append(args, cfg.ExtraArgs...)
	args = append(args, cfg.SourcePaths...)

	cctx := ctx
	if cfg.Timeout > 0 {
		var cancel context.CancelFunc
		cctx, cancel = context.WithTimeout(ctx, cfg.Timeout)
		defer cancel()
	}
	cmd := exec.CommandContext(cctx, binary, args...)
	cmd.Dir = cfg.WorkDir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	runErr := cmd.Run()

	parsed := parseMutestingOutput(stdout.Bytes())
	parsed.MutatedFiles = uniqueDirs(cfg.SourcePaths)
	parsed.RawOutput = stdout.String()

	report.Mutation.Killed = parsed.Killed
	report.Mutation.Survived = parsed.Survived
	report.Mutation.Total = parsed.Total
	report.Mutation.Score = parsed.Score
	report.Mutation.MutatedFiles = parsed.MutatedFiles
	report.Mutation.SurvivedSummary = parsed.SurvivedMutants

	report.FinishedAt = time.Now()
	report.DurationSeconds = time.Since(started).Seconds()

	switch {
	case runErr != nil && cctx.Err() == context.DeadlineExceeded:
		report.Verdict = testreport.VerdictTimedOut
		report.Passed = false
		report.Error = fmt.Sprintf("go-mutesting timed out after %s", cfg.Timeout)
	case parsed.Total == 0:
		// No mutants generated. Almost always means the tool found
		// nothing to mutate (e.g. only comments changed). We do NOT
		// fail open: emit tool_unavailable so the dispatcher applies
		// the coverage fallback.
		report.Verdict = testreport.VerdictToolUnavailable
		report.Passed = false
		report.Error = "go-mutesting reported zero mutants (coverage fallback applies)"
		if msg := strings.TrimSpace(stderr.String()); msg != "" {
			report.Error += ": " + truncate(msg, 400)
		}
	case parsed.Score >= threshold:
		report.Verdict = testreport.VerdictPassed
		report.Passed = true
	default:
		report.Verdict = testreport.VerdictFailed
		report.Passed = false
		report.Findings = append(report.Findings, buildMutationFindings(parsed, threshold)...)
	}

	return report, nil
}

// parseMutestingOutput interprets go-mutesting's stdout. The tool
// emits one "PASS"/"FAIL" line per mutant, plus a summary trailer.
// We don't rely on the summary because it has formatting drift
// across versions; we count the per-mutant lines directly.
//
// Sample lines (v2.3.1):
//
//   FAIL "mutant/0" with checksum b6...  ./pool.go: removed condition
//   PASS "mutant/1" with checksum 9a...  ./pool.go:42:6: replaced + with -
//   The mutation score is 0.625 (5 passed, 3 failed, 0 duplicated, 0 skipped, total is 8)
//
// Remember: PASS == survived, FAIL == killed (inverted from mutmut).
func parseMutestingOutput(raw []byte) Tier0Stats {
	var stats Tier0Stats
	lines := strings.Split(string(raw), "\n")

	for _, ln := range lines {
		ln = strings.TrimRight(ln, "\r")
		switch {
		case strings.HasPrefix(ln, "FAIL "):
			stats.Killed++
		case strings.HasPrefix(ln, "PASS "):
			stats.Survived++
			if sm, ok := parseMutantLine(ln); ok {
				stats.SurvivedMutants = append(stats.SurvivedMutants, sm)
			}
		}
	}

	stats.Total = stats.Killed + stats.Survived
	if stats.Total > 0 {
		stats.Score = float64(stats.Killed) / float64(stats.Total)
	}
	return stats
}

// mutantLineRE captures the file:line and mutator description from a
// go-mutesting per-mutant line.
//
// We deliberately use a forgiving regex — go-mutesting versions
// occasionally permute their column ordering. If we can't parse the
// detail we fall back to recording just the raw description.
var mutantLineRE = regexp.MustCompile(`(?P<file>[^\s"]+\.go):(?P<line>\d+)(?::\d+)?:\s*(?P<desc>.+)$`)

func parseMutantLine(line string) (testreport.SurvivedMutant, bool) {
	m := mutantLineRE.FindStringSubmatch(line)
	if m == nil {
		// Fallback — still emit a finding so the rubric LLM can see
		// the raw text even when our regex didn't match.
		return testreport.SurvivedMutant{
			File:    "<unparsed>",
			Mutator: strings.TrimSpace(line),
		}, true
	}
	lineNum, _ := strconv.Atoi(m[2])
	desc := strings.TrimSpace(m[3])
	mutator, original, replacement := splitMutatorDesc(desc)
	return testreport.SurvivedMutant{
		File:        filepath.ToSlash(m[1]),
		Line:        lineNum,
		Mutator:     mutator,
		Original:    original,
		Replacement: replacement,
	}, true
}

// splitMutatorDesc extracts a structured (mutator, original,
// replacement) tuple from go-mutesting's free-form description.
// go-mutesting writes descriptions like "replaced + with -" or
// "removed condition", which we lightly parse for findings.
func splitMutatorDesc(desc string) (mutator, original, replacement string) {
	lower := strings.ToLower(desc)
	switch {
	case strings.HasPrefix(lower, "replaced "):
		// "replaced X with Y"
		rest := strings.TrimPrefix(desc, "replaced ")
		if idx := strings.Index(rest, " with "); idx >= 0 {
			return "ReplaceOperator", strings.TrimSpace(rest[:idx]), strings.TrimSpace(rest[idx+len(" with "):])
		}
		return "ReplaceOperator", "", strings.TrimSpace(rest)
	case strings.HasPrefix(lower, "removed "):
		return "RemoveStatement", strings.TrimSpace(strings.TrimPrefix(desc, "removed ")), ""
	case strings.HasPrefix(lower, "negated "):
		return "Negate", strings.TrimSpace(strings.TrimPrefix(desc, "negated ")), "!" + strings.TrimSpace(strings.TrimPrefix(desc, "negated "))
	default:
		return desc, "", ""
	}
}

func buildMutationFindings(parsed Tier0Stats, threshold float64) []testreport.Finding {
	out := make([]testreport.Finding, 0, 1+len(parsed.SurvivedMutants))
	out = append(out, testreport.Finding{
		Category: "mutation_score_below_threshold",
		Severity: "error",
		Detail: fmt.Sprintf(
			"mutation score %.3f < threshold %.2f (realistic Go-tooling target ≈ %.2f). %d of %d mutants survived.",
			parsed.Score, threshold, Tier0RealisticTarget, parsed.Survived, parsed.Total,
		),
		SuggestedFix: "Add focused unit tests that distinguish the surviving mutants below.",
	})
	for _, m := range parsed.SurvivedMutants {
		out = append(out, testreport.Finding{
			Category: "mutation_survived",
			Severity: "warn",
			File:     m.File,
			Line:     m.Line,
			Detail: fmt.Sprintf("mutator=%s original=%q replacement=%q — no test failed against this mutation",
				m.Mutator, m.Original, m.Replacement),
		})
	}
	return out
}

func uniqueDirs(paths []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(paths))
	for _, p := range paths {
		p = filepath.ToSlash(p)
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}
		out = append(out, p)
	}
	return out
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
