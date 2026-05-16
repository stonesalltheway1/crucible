// Package main is the entry point for crucible-cartographer.
//
// The service receives CartographyJob requests from the control plane,
// runs the day-1 customer experience pipeline on a checked-out repo,
// and emits a CartographyResult (deterministic counts plus inferred
// AGENTS.md). See ../../README.md for the pipeline composition.
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

	"github.com/crucible/apps/cartographer/internal/api"
	"github.com/crucible/apps/cartographer/internal/distill"
	"github.com/crucible/apps/cartographer/internal/oss"
)

const version = "2026.06.0-phase8"

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	addr := envOr("CRUCIBLE_CARTOGRAPHER_LISTEN", ":9420")
	memoryRouter := envOr("CRUCIBLE_MEMORY_ROUTER_ADDR", "")
	distillerAddr := envOr("CRUCIBLE_DISTILLER_ADDR", "")
	ossPath := envOr("CRUCIBLE_OSS_DEFAULTS_DIR", "../../services/memory-router/global_defaults")

	ossLoader, err := oss.NewLoader(ossPath)
	if err != nil {
		// OSS defaults missing is dev-mode tolerable; warn but continue.
		slog.Warn("oss defaults loader degraded", "path", ossPath, "err", err)
	}

	// Distiller routing: prefer the in-cluster distiller service if its
	// address is configured; otherwise fall back to the Anthropic
	// Messages API directly when ANTHROPIC_API_KEY is set; otherwise
	// run in offline mode (deterministic passes only).
	llmCfg := distill.Config{
		Endpoint: distillerAddr,
		Model:    envOr("CRUCIBLE_DISTILLER_MODEL", "claude-haiku-4-5-20251001"),
		Timeout:  5 * time.Minute,
	}
	llmMode := "offline"
	if distillerAddr != "" {
		llmMode = "distiller-service"
	} else if key := os.Getenv("ANTHROPIC_API_KEY"); key != "" {
		llmCfg.APIKey = key
		llmCfg.Provider = distill.ProviderAnthropic
		llmMode = "anthropic-direct"
	} else {
		llmCfg.Provider = distill.ProviderOffline
	}
	llm := distill.NewClient(llmCfg)

	srv := api.NewServer(api.Config{
		Version:           version,
		MemoryRouterAddr:  memoryRouter,
		OSSDefaultsLoader: ossLoader,
		LLMClient:         llm,
	})

	httpServer := &http.Server{
		Addr:              addr,
		Handler:           srv,
		ReadHeaderTimeout: 10 * time.Second,
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	go func() {
		slog.Info("crucible-cartographer listening",
			"addr", addr,
			"version", version,
			"memory_router", memoryRouter,
			"distiller", distillerAddr,
			"oss_defaults_dir", ossPath,
			"llm_mode", llmMode,
			"llm_model", llmCfg.Model,
		)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("listen failed", "err", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelShutdown()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		slog.Warn("graceful shutdown timed out", "err", err)
	}
	fmt.Fprintln(os.Stderr, "crucible-cartographer stopped cleanly")
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
