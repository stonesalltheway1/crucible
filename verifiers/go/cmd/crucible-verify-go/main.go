// Command crucible-verify-go is the Phase-4 per-language verifier
// runner for Go. The verifier daemon (apps/verifier) spawns this CLI
// in an isolated sandbox, pipes a VerificationRequest JSON object to
// stdin, and reads a TestReport JSON object from stdout, prefixed by
// the wire delimiter `===CRUCIBLE-TESTREPORT===\n`.
//
// All logs go to stderr. The runner never opens network sockets, never
// writes to disk outside of its own temp dirs (the Tier 0/1 tools may,
// but those write into the sandbox-mounted WorkDir), and never reads
// the executor's reasoning trace — see internal/audit.
//
// Usage:
//
//	crucible-verify-go --tier=tier_0_mutation < request.json
//
// Exit code is always 0 unless the runner itself crashed (e.g. stdin
// unreadable, malformed JSON, panic). Substantive failures of the
// tier under test are reported via TestReport.Verdict=failed; the
// process still exits 0 because the dispatcher reads the report.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/crucible/verifier/pkg/testreport"

	"github.com/crucible/verify-go/internal/audit"
	"github.com/crucible/verify-go/internal/diff"
	"github.com/crucible/verify-go/internal/schema"
	"github.com/crucible/verify-go/internal/tiers"
)

// ReporterID identifies this runner binary in TestReport.ReporterID.
const ReporterID = "crucible-verify-go"

// ReporterVersion is bumped in lockstep with the daemon. The runner
// is wire-compatible across patch versions of the daemon; minor
// version bumps must be coordinated.
const ReporterVersion = "0.1.0"

// Wire delimiter consumed by processpool.trimPrelude. We always emit
// it on a line of its own immediately before the JSON body.
const Delimiter = "===CRUCIBLE-TESTREPORT===\n"

func main() {
	tier := flag.String("tier", "", "tier to run: tier_0_mutation|tier_1_pbt|tier_2_contract|tier_3_proof|tier_4_honest_ci")
	workDirFlag := flag.String("work-dir", "", "override the materialised source tree (else read from CRUCIBLE_VERIFIER_WORKDIR or request.work_dir)")
	timeoutFlag := flag.Duration("timeout", 0, "wall-clock budget for the tier (zero means tier default)")
	flag.CommandLine.SetOutput(os.Stderr)
	flag.Parse()

	if *tier == "" {
		fail(testreport.Tier(""), "missing --tier flag")
		return
	}

	raw, err := io.ReadAll(os.Stdin)
	if err != nil {
		fail(testreport.Tier(*tier), fmt.Sprintf("stdin read: %v", err))
		return
	}

	// 1. Audit the parsed map BEFORE binding to the typed struct —
	// this catches denylisted keys that the typed struct would drop
	// silently (encoding/json ignores unknown fields).
	var generic map[string]any
	if err := json.Unmarshal(raw, &generic); err != nil {
		fail(testreport.Tier(*tier), fmt.Sprintf("stdin JSON parse: %v", err))
		return
	}
	if err := audit.NoLeakage(generic); err != nil {
		// LeakageError is structured; record the offending field.
		report := minimalReport(testreport.Tier(*tier), testreport.VerdictToolUnavailable)
		report.Error = err.Error()
		emit(report)
		return
	}

	// 2. Bind to the runner-side mirror struct.
	var req schema.VerificationRequest
	if err := json.Unmarshal(raw, &req); err != nil {
		fail(testreport.Tier(*tier), fmt.Sprintf("VerificationRequest unmarshal: %v", err))
		return
	}

	// 3. Resolve the work-dir (CLI flag > env var > request body).
	workDir := *workDirFlag
	if workDir == "" {
		workDir = os.Getenv("CRUCIBLE_VERIFIER_WORKDIR")
	}
	if workDir == "" {
		workDir = req.WorkDir
	}

	// 4. Filter the diff to Go files.
	files := diff.FilterGo(req.Diff)

	// 5. Dispatch.
	ctx := context.Background()
	report, err := dispatch(ctx, testreport.Tier(*tier), workDir, *timeoutFlag, files, req)
	if err != nil {
		fail(testreport.Tier(*tier), err.Error())
		return
	}

	// 6. Stamp identity fields.
	stampReport(report, &req)

	emit(report)
}

// dispatch routes to the tier handler and returns a partial report.
func dispatch(
	ctx context.Context,
	tier testreport.Tier,
	workDir string,
	timeout time.Duration,
	files diff.FileSet,
	req schema.VerificationRequest,
) (*testreport.TestReport, error) {
	switch tier {
	case testreport.TierMutation:
		return tiers.RunMutation(ctx, tiers.MutationConfig{
			WorkDir:     workDir,
			SourcePaths: files.SourcePaths(),
			Timeout:     coalesce(timeout, 2*time.Minute),
		})
	case testreport.TierPBT:
		return tiers.RunPBT(ctx, tiers.PBTConfig{
			WorkDir:   workDir,
			TestFiles: append(req.TestFiles, files.Test...),
			Packages:  files.Packages(),
			Timeout:   coalesce(timeout, 15*time.Minute),
		})
	case testreport.TierContract:
		return tiers.RunContract(ctx, tiers.ContractConfig{
			WorkDir:     workDir,
			SpecChanges: req.SpecChanges,
			Timeout:     coalesce(timeout, 45*time.Minute),
		})
	case testreport.TierProof:
		return tiers.RunProof(ctx, tiers.ProofConfig{})
	case testreport.TierHonestCI:
		return tiers.RunHonestCI(ctx, tiers.HonestCIConfig{
			WorkDir: workDir,
			Timeout: coalesce(timeout, 30*time.Minute),
		})
	default:
		return nil, fmt.Errorf("unknown tier %q", tier)
	}
}

func coalesce(have, fallback time.Duration) time.Duration {
	if have > 0 {
		return have
	}
	return fallback
}

func stampReport(r *testreport.TestReport, req *schema.VerificationRequest) {
	r.SchemaVersion = testreport.SchemaVersion
	r.TaskID = req.TaskID
	r.Language = testreport.LangGo
	r.ReporterID = ReporterID
	r.ReporterVersion = ReporterVersion
	r.ReporterOidcSubject = os.Getenv("CRUCIBLE_VERIFIER_OIDC_SUBJECT")
	if r.StartedAt.IsZero() {
		r.StartedAt = time.Now()
	}
	if r.FinishedAt.IsZero() {
		r.FinishedAt = time.Now()
	}
	if r.WallClockBudgetSeconds == 0 && req.Budget.WallClockCapSeconds > 0 {
		r.WallClockBudgetSeconds = float64(req.Budget.WallClockCapSeconds)
	}
}

func emit(r *testreport.TestReport) {
	if _, err := os.Stdout.Write([]byte(Delimiter)); err != nil {
		fmt.Fprintf(os.Stderr, "crucible-verify-go: write delimiter: %v\n", err)
		os.Exit(1)
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(r); err != nil {
		fmt.Fprintf(os.Stderr, "crucible-verify-go: encode report: %v\n", err)
		os.Exit(1)
	}
}

func fail(tier testreport.Tier, msg string) {
	fmt.Fprintf(os.Stderr, "crucible-verify-go: %s\n", msg)
	r := minimalReport(tier, testreport.VerdictFailed)
	r.Error = msg
	emit(r)
}

func minimalReport(tier testreport.Tier, verdict testreport.Verdict) *testreport.TestReport {
	now := time.Now()
	if tier == "" {
		// Validate() rejects empty tier; pick a defensible default
		// so the dispatcher gets a parseable envelope.
		tier = testreport.TierMutation
	}
	return &testreport.TestReport{
		SchemaVersion:   testreport.SchemaVersion,
		Tier:            tier,
		Language:        testreport.LangGo,
		Framework:       "crucible-verify-go",
		Verdict:         verdict,
		Passed:          false,
		StartedAt:       now,
		FinishedAt:      now,
		ReporterID:      ReporterID,
		ReporterVersion: ReporterVersion,
	}
}
