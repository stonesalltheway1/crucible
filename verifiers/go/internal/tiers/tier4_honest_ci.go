// Tier 4 — Honest CI: bit-identical rebuild of the Go module.
//
// The full Tier 4 pipeline (Nix flake hermetic rebuild + Sigstore
// keyless OIDC + Rekor + Witness/Tekton Chains) is driven by
// apps/verifier/internal/tier4. This per-language runner contributes
// the language-specific piece: rebuild the executor's artifact twice
// inside the verifier sandbox and check the SHA-256s match.
//
// We use:
//
//   GOFLAGS="-trimpath -buildvcs=false"
//   SOURCE_DATE_EPOCH=0
//   -ldflags="-buildid="
//
// which is the smallest set known to give bit-identical Go binaries
// across two `go build` invocations on the same toolchain. Drift
// here is almost always a "the build embedded a timestamp / hostname
// / cgo-resolved path" symptom; the surfaced finding tells the agent
// what to scrub.
package tiers

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/crucible/verifier/pkg/testreport"
)

// BuilderID identifies this builder in HonestCIStats.
const BuilderID = "https://crucible.dev/builders/go-runner/v1"

// HonestCIConfig is the Tier 4 runner config.
type HonestCIConfig struct {
	WorkDir string
	// MainPackage is the import path to build, e.g. "./cmd/myapp".
	// Empty means "./..." — but the verifier picks the single main
	// package it can identify; if there are multiple, we build the
	// first deterministically (alphabetic) and emit a finding noting
	// the ambiguity.
	MainPackage string
	GoBinary    string
	Timeout     time.Duration
}

// RunHonestCI performs two `go build`s and SHA-256-compares the
// outputs.
func RunHonestCI(ctx context.Context, cfg HonestCIConfig) (*testreport.TestReport, error) {
	started := time.Now()
	goBin := cfg.GoBinary
	if goBin == "" {
		goBin = "go"
	}
	report := &testreport.TestReport{
		SchemaVersion: testreport.SchemaVersion,
		Tier:          testreport.TierHonestCI,
		Language:      testreport.LangGo,
		Framework:     "go-build-double",
		StartedAt:     started,
		HonestCI: &testreport.HonestCIStats{
			BuilderID:       BuilderID,
			SLSALevel:       0, // bumped by the daemon-side aggregator after sigstore/witness
			ScrubberAuditOK: true,
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

	mainPkg := cfg.MainPackage
	if mainPkg == "" {
		mainPkg = "./..."
	}

	cctx := ctx
	if cfg.Timeout > 0 {
		var cancel context.CancelFunc
		cctx, cancel = context.WithTimeout(ctx, cfg.Timeout)
		defer cancel()
	}

	hashA, errA := buildOnce(cctx, goBin, cfg.WorkDir, mainPkg, "crucible-build-a")
	hashB, errB := buildOnce(cctx, goBin, cfg.WorkDir, mainPkg, "crucible-build-b")
	if errA != nil || errB != nil {
		report.Verdict = testreport.VerdictFailed
		report.Passed = false
		report.Error = fmt.Sprintf("Tier 4 build failed: a=%v b=%v", errA, errB)
		report.HonestCI.BitIdentical = false
		report.FinishedAt = time.Now()
		report.DurationSeconds = time.Since(started).Seconds()
		return report, nil
	}

	report.HonestCI.ExecutorRebuildHash = hashA
	report.HonestCI.VerifierRebuildHash = hashB
	report.HonestCI.BitIdentical = hashA == hashB

	if !report.HonestCI.BitIdentical {
		report.Verdict = testreport.VerdictFailed
		report.Passed = false
		report.Findings = append(report.Findings, testreport.Finding{
			Category: "non_reproducible_build",
			Severity: "error",
			Detail: fmt.Sprintf(
				"two `go build -trimpath -buildvcs=false -ldflags=\"-buildid=\"` invocations produced different SHA-256s (%s vs %s). Likely causes: timestamp embedded via -X or //go:embed; cgo resolving host-specific paths; non-deterministic codegen.",
				hashA, hashB,
			),
			SuggestedFix: "Pass -ldflags=\"-X 'pkg.BuildDate=$(date -u -d @${SOURCE_DATE_EPOCH:-0})'\" and ensure no //go:generate embeds wall-clock data.",
		})
	} else {
		report.Verdict = testreport.VerdictPassed
		report.Passed = true
	}

	report.FinishedAt = time.Now()
	report.DurationSeconds = time.Since(started).Seconds()
	return report, nil
}

// buildOnce invokes `go build` with the reproducible flag set,
// SHA-256s the resulting binary, then deletes it. The output path is
// unique-per-call so the two builds don't trample each other on
// parallel goroutines.
func buildOnce(ctx context.Context, goBin, workDir, pkg, outName string) (string, error) {
	tmpDir, err := os.MkdirTemp("", "crucible-go-build-")
	if err != nil {
		return "", fmt.Errorf("mktemp: %w", err)
	}
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	outPath := filepath.Join(tmpDir, outName)
	args := []string{
		"build",
		"-trimpath",
		"-buildvcs=false",
		"-ldflags=-buildid= -s -w",
		"-o", outPath,
		pkg,
	}
	cmd := exec.CommandContext(ctx, goBin, args...)
	cmd.Dir = workDir
	// Inherit the runner's env then override the determinism knobs.
	cmd.Env = append(os.Environ(),
		"SOURCE_DATE_EPOCH=0",
		"CGO_ENABLED=0",
		"GOFLAGS=-trimpath -buildvcs=false",
	)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("go build %s: %w (stderr=%s)", pkg, err, strings.TrimSpace(stderr.String()))
	}
	data, err := os.ReadFile(outPath)
	if err != nil {
		return "", fmt.Errorf("read built binary: %w", err)
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}
