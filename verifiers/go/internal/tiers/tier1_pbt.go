// Tier 1 — property-based tests (pgregory.net/rapid v1.3.0) plus
// Go 1.24+ native fuzz (testing.F).
//
// Two phases:
//
//   1. PBT driver:    `go test ./<diff-packages>... -run=^Property -rapid.checks=10000 -count=1`
//      The -rapid.checks flag is read by rapid's testing.T.Run wrapper.
//      We discover diff-scoped packages from the diff (see internal/diff).
//
//   2. Fuzz driver:   for each `func Fuzz<X>(f *testing.F)` declared
//      in a diff-touched test file, `go test -fuzz=Fuzz<X> -fuzztime=15s ./...`.
//      Failing inputs land in testdata/fuzz/Fuzz<X>/ — we surface them
//      as Counterexamples.
//
// Wall-clock budget per verifier-pipeline.md §Tier 1 is 5 min default,
// 15 min max. We default fuzztime to 15s per target so multiple
// targets fit comfortably; callers can override via PBTConfig.FuzzTime.
package tiers

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	cruciblev1 "github.com/crucible/sdk-go/crucible/v1"
	"github.com/crucible/verifier/pkg/testreport"
)

// IterationsMin is the Crucible mandate from verifier-pipeline.md
// (>=10_000). The dispatcher enforces this; we also stamp it into
// PBTStats so the daemon's Validate() catches under-runs.
const IterationsMin = 10000

// DefaultFuzzTime is the per-target fuzz budget. 15s strikes the
// balance between catching shallow crashers and fitting the 5-min
// tier budget when several fuzz targets exist.
const DefaultFuzzTime = 15 * time.Second

// PBTConfig is the Tier 1 runner config.
type PBTConfig struct {
	WorkDir string
	// TestFiles is the diff-scoped subset of *_test.go files (the
	// runner discovers Fuzz targets here).
	TestFiles []cruciblev1.FileChange
	// Packages is the diff-scoped Go package list to pass to `go test`.
	// Empty means "./..." (which is fine but slower).
	Packages []string
	// GoBinary defaults to "go".
	GoBinary string
	// FuzzTime per target. Zero means DefaultFuzzTime.
	FuzzTime time.Duration
	// RapidChecks overrides IterationsMin if non-zero (must still be
	// >= IterationsMin to satisfy the mandate).
	RapidChecks int
	// Timeout caps the whole Tier 1 invocation (PBT phase + all
	// fuzz targets).
	Timeout time.Duration
}

// RunPBT executes the PBT then fuzz phases and returns a partial
// TestReport (caller fills task/headers).
func RunPBT(ctx context.Context, cfg PBTConfig) (*testreport.TestReport, error) {
	started := time.Now()

	goBin := cfg.GoBinary
	if goBin == "" {
		goBin = "go"
	}
	checks := cfg.RapidChecks
	if checks < IterationsMin {
		checks = IterationsMin
	}
	fuzzTime := cfg.FuzzTime
	if fuzzTime <= 0 {
		fuzzTime = DefaultFuzzTime
	}

	report := &testreport.TestReport{
		SchemaVersion: testreport.SchemaVersion,
		Tier:          testreport.TierPBT,
		Language:      testreport.LangGo,
		Framework:     "rapid+testing.F",
		StartedAt:     started,
		PBT: &testreport.PBTStats{
			Iterations:    checks,
			IterationsMin: IterationsMin,
		},
	}

	if _, err := exec.LookPath(goBin); err != nil {
		report.Verdict = testreport.VerdictToolUnavailable
		report.Passed = false
		report.Error = fmt.Sprintf("go toolchain not on PATH: %v", err)
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

	// --- Phase 1: rapid PBT driver ---
	pbtPasses, pbtFindings, pbtProps := runRapidPhase(cctx, goBin, cfg, checks)
	report.PBT.Properties = pbtProps
	report.Findings = append(report.Findings, pbtFindings...)

	// --- Phase 2: native fuzz ---
	targets := discoverFuzzTargets(cfg)
	fuzzPasses := true
	for _, t := range targets {
		fr := runFuzzTarget(cctx, goBin, cfg, t, fuzzTime)
		report.PBT.FuzzCorpusSize += fr.CorpusSize
		report.PBT.FuzzNewSeeds += fr.NewSeeds
		report.PBT.FuzzCrashes += fr.Crashes
		report.PBT.Counterexamples = append(report.PBT.Counterexamples, fr.Counterexamples...)
		if fr.Crashes > 0 || len(fr.Counterexamples) > 0 {
			fuzzPasses = false
			report.Findings = append(report.Findings, fr.Findings...)
		}
	}

	report.FinishedAt = time.Now()
	report.DurationSeconds = time.Since(started).Seconds()

	switch {
	case cctx.Err() == context.DeadlineExceeded:
		report.Verdict = testreport.VerdictTimedOut
		report.Passed = false
		report.Error = fmt.Sprintf("Tier 1 timed out after %s", cfg.Timeout)
	case !pbtPasses || !fuzzPasses:
		report.Verdict = testreport.VerdictFailed
		report.Passed = false
	default:
		report.Verdict = testreport.VerdictPassed
		report.Passed = true
	}
	return report, nil
}

// rapidPropertyRE matches `func PropertyXxx(t *testing.T)` (or
// `func (s *Suite) PropertyXxx(...)`-style) — we use the function
// name `Property*` as the gating convention. The brief specifies
// `-run=Property` to scope the rapid driver to property tests.
var rapidPropertyRE = regexp.MustCompile(`func\s+(Property[A-Za-z0-9_]*)\s*\(`)

// runRapidPhase invokes `go test -run=^Property -rapid.checks=N`. We
// scope to cfg.Packages if any, else ./...
func runRapidPhase(ctx context.Context, goBin string, cfg PBTConfig, checks int) (passed bool, findings []testreport.Finding, properties []string) {
	pkgs := cfg.Packages
	if len(pkgs) == 0 {
		pkgs = []string{"./..."}
	}
	properties = discoverProperties(cfg)

	args := []string{"test"}
	args = append(args, pkgs...)
	args = append(args,
		"-run=^Property",
		fmt.Sprintf("-rapid.checks=%d", checks),
		"-count=1",
		"-v",
	)

	cmd := exec.CommandContext(ctx, goBin, args...)
	cmd.Dir = cfg.WorkDir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err == nil {
		return true, nil, properties
	}

	// rapid emits "--- FAIL: PropertyXxx" followed by a shrunken
	// counterexample on subsequent lines. We extract one Finding per
	// failing property.
	findings = parseRapidFailures(stdout.String() + "\n" + stderr.String())
	if len(findings) == 0 {
		findings = []testreport.Finding{{
			Category: "property_failed",
			Severity: "error",
			Detail:   "go test (rapid phase) returned non-zero but no property name parseable from output",
		}}
	}
	return false, findings, properties
}

// parseRapidFailures scans `go test -v` output for rapid-style
// failures. Each "--- FAIL: PropertyXxx" begins a stanza; we capture
// the immediately following "    Falsified after N tests" / "    [seed]"
// lines as the counterexample summary.
func parseRapidFailures(out string) []testreport.Finding {
	var findings []testreport.Finding
	sc := bufio.NewScanner(strings.NewReader(out))
	sc.Buffer(make([]byte, 1<<16), 1<<22)
	var current *testreport.Finding
	for sc.Scan() {
		line := sc.Text()
		if m := failPropertyRE.FindStringSubmatch(line); m != nil {
			if current != nil {
				findings = append(findings, *current)
			}
			current = &testreport.Finding{
				Category: "property_failed",
				Severity: "error",
				Detail:   m[1],
			}
			continue
		}
		if current != nil {
			// Collect indented context lines under the failing property.
			if strings.HasPrefix(line, "    ") || strings.HasPrefix(line, "\t") {
				current.Detail += "\n" + strings.TrimSpace(line)
			}
		}
	}
	if current != nil {
		findings = append(findings, *current)
	}
	return findings
}

var failPropertyRE = regexp.MustCompile(`^\s*---\s+FAIL:\s+(Property[A-Za-z0-9_]+)`)

// discoverProperties parses cfg.TestFiles for `func PropertyXxx(` —
// returns the property names so the report lists what was exercised.
// Best-effort; falls back to scanning the materialised workdir if the
// FileChange.Content is empty (the dispatcher sometimes ships
// just the path).
func discoverProperties(cfg PBTConfig) []string {
	seen := map[string]struct{}{}
	out := []string{}
	addMatches := func(content string) {
		for _, m := range rapidPropertyRE.FindAllStringSubmatch(content, -1) {
			if _, ok := seen[m[1]]; ok {
				continue
			}
			seen[m[1]] = struct{}{}
			out = append(out, m[1])
		}
	}
	for _, f := range cfg.TestFiles {
		if f.Content != "" {
			addMatches(f.Content)
			continue
		}
		if cfg.WorkDir != "" {
			data, err := os.ReadFile(filepath.Join(cfg.WorkDir, filepath.FromSlash(f.Path)))
			if err == nil {
				addMatches(string(data))
			}
		}
	}
	sort.Strings(out)
	return out
}

// --- Fuzz phase ---------------------------------------------------

// fuzzTarget is one (test-file, FuzzXxx) declaration discovered in
// the diff.
type fuzzTarget struct {
	Package string // import path relative to WorkDir, e.g. "./mypkg"
	Name    string // "FuzzParseURL"
}

var fuzzFuncRE = regexp.MustCompile(`func\s+(Fuzz[A-Za-z0-9_]+)\s*\(\s*[a-zA-Z_][a-zA-Z0-9_]*\s+\*testing\.F\s*\)`)

func discoverFuzzTargets(cfg PBTConfig) []fuzzTarget {
	seen := map[string]struct{}{}
	out := []fuzzTarget{}
	for _, f := range cfg.TestFiles {
		pkg := packageOf(f.Path)
		content := f.Content
		if content == "" && cfg.WorkDir != "" {
			data, err := os.ReadFile(filepath.Join(cfg.WorkDir, filepath.FromSlash(f.Path)))
			if err == nil {
				content = string(data)
			}
		}
		for _, m := range fuzzFuncRE.FindAllStringSubmatch(content, -1) {
			key := pkg + ":" + m[1]
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			out = append(out, fuzzTarget{Package: pkg, Name: m[1]})
		}
	}
	return out
}

// packageOf returns the `./dir` import-path for a file relative to
// the module root.
func packageOf(p string) string {
	dir := filepath.ToSlash(filepath.Dir(p))
	if dir == "" || dir == "." {
		return "."
	}
	return "./" + dir
}

// fuzzResult is the parsed outcome of one fuzz target invocation.
type fuzzResult struct {
	CorpusSize      int
	NewSeeds        int
	Crashes         int
	Counterexamples []testreport.Counterexample
	Findings        []testreport.Finding
}

// runFuzzTarget invokes `go test -fuzz=FuzzXxx$ -fuzztime=Ns ./pkg`.
// On a crash, Go writes the failing input to
// testdata/fuzz/FuzzXxx/<id> — we read those back and surface them
// as Counterexamples.
func runFuzzTarget(ctx context.Context, goBin string, cfg PBTConfig, t fuzzTarget, fuzzTime time.Duration) fuzzResult {
	args := []string{
		"test",
		t.Package,
		"-run=^$", // disable normal tests; only run the fuzz engine
		"-fuzz=^" + t.Name + "$",
		fmt.Sprintf("-fuzztime=%s", fuzzTime),
		"-count=1",
	}
	cmd := exec.CommandContext(ctx, goBin, args...)
	cmd.Dir = cfg.WorkDir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	out := stdout.String() + "\n" + stderr.String()

	result := fuzzResult{}
	// CorpusSize counts each "new interesting:" emission; NewSeeds is
	// the same signal projected with the captured integer (we only
	// need the count here, not the per-line int — the daemon's
	// telemetry pipeline materialises the histogram).
	result.CorpusSize = countLineMatches(out, fuzzNewInputsRE)
	result.NewSeeds = result.CorpusSize

	if err == nil {
		return result
	}

	// On failure, Go prints "Failing input written to testdata/fuzz/FuzzXxx/<id>".
	for _, m := range failingInputRE.FindAllStringSubmatch(out, -1) {
		path := filepath.Join(cfg.WorkDir, filepath.FromSlash(m[1]))
		content, readErr := os.ReadFile(path)
		seed := "<unread>"
		if readErr == nil {
			seed = string(content)
			if len(seed) > 400 {
				seed = seed[:400] + "..."
			}
		}
		result.Crashes++
		result.Counterexamples = append(result.Counterexamples, testreport.Counterexample{
			Property: t.Name,
			Shrunk:   seed,
			Seed:     m[1],
		})
		result.Findings = append(result.Findings, testreport.Finding{
			Category: "fuzz_crash",
			Severity: "error",
			File:     m[1],
			Detail:   fmt.Sprintf("fuzz target %s crashed; failing input persisted at %s", t.Name, m[1]),
		})
	}
	if result.Crashes == 0 {
		// Non-zero exit but no failing-input marker — surface as a
		// generic failure so the dispatcher still rejects.
		result.Crashes++
		result.Findings = append(result.Findings, testreport.Finding{
			Category: "fuzz_failed",
			Severity: "error",
			Detail:   "go test -fuzz=" + t.Name + " returned non-zero without producing a failing-input marker: " + truncate(strings.TrimSpace(out), 400),
		})
	}
	return result
}

var (
	failingInputRE  = regexp.MustCompile(`Failing input written to (testdata/fuzz/[^\s]+)`)
	fuzzNewInputsRE = regexp.MustCompile(`new interesting: \d+`)
)

func countLineMatches(s string, re *regexp.Regexp) int {
	matches := re.FindAllString(s, -1)
	return len(matches)
}
