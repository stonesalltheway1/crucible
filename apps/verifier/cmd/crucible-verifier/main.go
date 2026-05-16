// Command crucible-verifier is the Phase-4 verifier daemon entrypoint.
//
// It wires:
//   - the dispatcher
//   - the critical-path classifier
//   - the cross-family rubric (Gemini 3.1 Pro by default)
//   - the per-language process pool
//   - Tier 3 (Dafny) and Tier 4 (Nix honest-CI)
//   - the HTTP API the control plane calls
//
// Env:
//   CRUCIBLE_VERIFIER_LISTEN_ADDR  — defaults to :9080
//   CRUCIBLE_VERIFIER_KEY_DIR      — local Ed25519 key directory
//   CRUCIBLE_VERIFIER_JOURNAL_PATH — local attestation journal
//   ANTHROPIC_API_KEY              — executor side; the daemon NEVER uses it for rubric calls
//   GOOGLE_API_KEY                 — default verifier rubric model (Gemini 3.1 Pro)
//   OPENAI_API_KEY                 — optional alternate verifier
//   CRUCIBLE_VERIFIER_VENDOR       — override the default verifier vendor (anthropic|google|openai)
//   CRUCIBLE_VERIFIER_HEURISTIC=1  — force heuristic (no LLM) — useful for hermetic CI
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	cruciblev1 "github.com/crucible/sdk-go/crucible/v1"
	"github.com/crucible/verifier/internal/api"
	"github.com/crucible/verifier/internal/criticalpath"
	"github.com/crucible/verifier/internal/dispatcher"
	"github.com/crucible/verifier/internal/memorybridge"
	"github.com/crucible/verifier/internal/processpool"
	"github.com/crucible/verifier/internal/rubric"
	"github.com/crucible/verifier/internal/tier3"
	"github.com/crucible/verifier/internal/tier4"
)

const version = "2026.06.0-phase5"

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	if err := run(logger); err != nil {
		logger.Error("crucible-verifier: fatal", "err", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// 1. Process pool — for Phase 4 ship with an ExecProvider so local
	// dev can drive the per-language CLIs from PATH. Production wires
	// an E2B-backed provider when the daemon runs in the SaaS cluster.
	provider := &processpool.ExecProvider{WorkDir: getenv("CRUCIBLE_VERIFIER_WORKDIR", ".")}
	pool := processpool.NewPool(provider)

	// 2. Tier 3 Dafny adapter (other provers stubbed with typed errors).
	t3 := tier3.NewAdapter()

	// 3. Tier 4 honest-CI verifier.
	t4 := tier4.NewVerifier()

	// 4. Rubric Judge — pick the LLM client.
	judge, err := buildJudge(logger)
	if err != nil {
		return err
	}

	// 5. Critical-path classifier with the offline-only featurizer.
	classifier := criticalpath.NewClassifier(criticalpath.NewPathPatternFeaturizer())

	// 6. Dispatcher.
	disp := dispatcher.New(pool, t3, t4, judge, classifier)

	// 6b. Phase-5 memory-bridge — wires the MemoryComplianceFeaturizer
	// into the rubric's trust_signal_alignment slot. Env-gated via
	// CRUCIBLE_MEMORY_ROUTER_ADDR; when unset the bridge is a no-op and
	// the rubric runs unchanged (Phase 4 behaviour).
	mb := memorybridge.New()
	disp.MemoryFeaturizer = &rubric.MemoryComplianceFeaturizer{
		Bridge:   &memoryBridgeAdapter{mb: mb},
		Disabled: os.Getenv(memorybridge.EnvRouterAddr) == "",
	}
	if disp.MemoryFeaturizer.Disabled {
		logger.Info("memory-bridge: disabled (CRUCIBLE_MEMORY_ROUTER_ADDR unset)")
	} else {
		logger.Info("memory-bridge: configured", "addr", os.Getenv(memorybridge.EnvRouterAddr))
	}

	// 7. HTTP server.
	server := &api.Server{
		Dispatcher: disp,
		Logger:     logger,
		Version:    version,
	}

	addr := getenv("CRUCIBLE_VERIFIER_LISTEN_ADDR", ":9080")
	httpSrv := &http.Server{
		Addr:              addr,
		Handler:           server.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		logger.Info("crucible-verifier listening", "addr", addr, "version", version,
			"judge_vendor", judge.Client.Vendor(), "judge_model", judge.Client.Model())
		if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("http server failed", "err", err)
			cancel()
		}
	}()

	<-ctx.Done()
	logger.Info("crucible-verifier shutting down")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	if err := httpSrv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}
	return nil
}

// buildJudge picks the rubric LLM client based on env. Heuristic fallback
// is the explicit `CRUCIBLE_VERIFIER_HEURISTIC=1` switch.
//
// Phase 4 ships the heuristic by default in CI; the SaaS daemon
// configures GOOGLE_API_KEY at boot and pairs Gemini 3.1 Pro with the
// Anthropic Opus 4.7 executor (the ADR-002 default).
func buildJudge(logger *slog.Logger) (*rubric.Judge, error) {
	if os.Getenv("CRUCIBLE_VERIFIER_HEURISTIC") == "1" {
		logger.Info("rubric: heuristic mode (no LLM)")
		return rubric.NewJudge(rubric.NewHeuristicClient()), nil
	}
	// Phase 4 ships ONLY the heuristic + adapter shells. Production
	// SaaS wires the real client by injecting an LLMClient
	// implementation into the Judge at startup. The cmd binary is
	// intentionally kept hermetic in Phase 4 to keep the operator's
	// install path simple; the modelrouter-adapter is a Phase-5
	// deliverable.
	logger.Info("rubric: heuristic fallback (no model-router adapter wired in Phase 4)")
	return rubric.NewJudge(rubric.NewHeuristicClient()), nil
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

// memoryBridgeAdapter bridges memorybridge.Bridge → rubric.ComplianceClient.
// Lives here (not in memorybridge) because exposing a rubric symbol
// from memorybridge would create an import cycle.
type memoryBridgeAdapter struct{ mb memorybridge.Bridge }

func (a *memoryBridgeAdapter) CheckCompliance(ctx context.Context, req rubric.ComplianceRequest) (cruciblev1.ComplianceReport, error) {
	return a.mb.CheckCompliance(ctx, memorybridge.CheckRequest{
		TenantID: req.TenantID,
		TaskID:   req.TaskID,
		Diff:     req.Diff,
	})
}
