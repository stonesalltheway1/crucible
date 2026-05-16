// Command crucible-control-plane is the Phase-1 entry point for the Crucible
// Agent Control Plane.
//
// It wires together:
//   - attestation Service           (libs/attestation)
//   - model router                  (anthropic + google + openai)
//   - task router (classifier)
//   - plan builder
//   - budget enforcer registry
//   - cost meter
//   - event publisher (in-memory + optional webhook)
//   - tenant policy loader
//   - in-memory task store
//   - HTTP API (will become connect-go in Phase 2)
//
// Env:
//   ANTHROPIC_API_KEY      — required for real Tier 0/1/2 routing (else heuristic)
//   GOOGLE_API_KEY         — required for verifier-side calls
//   OPENAI_API_KEY         — optional alternate routing
//   CRUCIBLE_LISTEN_ADDR   — defaults to :8080
//   CRUCIBLE_DEFAULT_TENANT— defaults to "single-tenant"
//   CRUCIBLE_WEBHOOK_URL   — optional event sink
//   CRUCIBLE_KEY_DIR       — local Ed25519 key directory
//   CRUCIBLE_JOURNAL_PATH  — local attestation journal
//   CRUCIBLE_COSTLOG_DIR   — per-task cost JSONL directory
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

	"github.com/crucible/attestation"
	"github.com/crucible/control-plane/internal/api"
	"github.com/crucible/control-plane/internal/budgetenforcer"
	"github.com/crucible/control-plane/internal/costmeter"
	"github.com/crucible/control-plane/internal/events"
	"github.com/crucible/control-plane/internal/modelrouter"
	"github.com/crucible/control-plane/internal/planbuilder"
	"github.com/crucible/control-plane/internal/promotionbridge"
	"github.com/crucible/control-plane/internal/store"
	"github.com/crucible/control-plane/internal/taskrouter"
	"github.com/crucible/control-plane/internal/tenantpolicy"
	"github.com/crucible/control-plane/internal/verifierbridge"
)

const version = "2026.06.0-phase6"

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	if err := run(logger); err != nil {
		logger.Error("control-plane: fatal", "err", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// 1. Attestation: local Ed25519 signer + local hash-chained journal.
	signer, err := attestation.NewLocalEd25519Signer(getenv("CRUCIBLE_KEY_DIR", ""))
	if err != nil {
		return fmt.Errorf("attestation signer: %w", err)
	}
	publisher, err := attestation.NewLocalJournalPublisher(getenv("CRUCIBLE_JOURNAL_PATH", ""))
	if err != nil {
		return fmt.Errorf("attestation publisher: %w", err)
	}
	attestSvc, err := attestation.NewService(signer, publisher)
	if err != nil {
		return fmt.Errorf("attestation service: %w", err)
	}
	logger.Info("attestation wired",
		"signer", signer.KeyID(),
		"journal", publisher.Path(),
		"oidc_subject", signer.OidcSubject(),
	)

	// 2. Model router: wire the vendors whose env vars are set.
	mr := modelrouter.NewRouter(
		modelrouter.NewAnthropicClientFromEnv(),
		modelrouter.NewGoogleClientFromEnv(ctx),
		modelrouter.NewOpenAIClientFromEnv(),
	)
	vendors := mr.Vendors()
	if len(vendors) == 0 {
		logger.Warn("no LLM vendor configured — control plane will use heuristic classification and stub plans. " +
			"Set ANTHROPIC_API_KEY for real Tier 0/1/2 routing.")
	} else {
		vs := make([]string, len(vendors))
		for i, v := range vendors {
			vs[i] = string(v)
		}
		logger.Info("LLM vendors wired", "vendors", vs)
	}

	// 3. Task router (classifier) + plan builder.
	router := taskrouter.New(mr, "")
	builder := planbuilder.New(mr, attestSvc, "")

	// 4. Budget enforcers + cost meter.
	enforcers := budgetenforcer.NewRegistry()
	meter, err := costmeter.New(logger, getenv("CRUCIBLE_COSTLOG_DIR", ""), enforcers)
	if err != nil {
		return fmt.Errorf("costmeter: %w", err)
	}
	defer meter.Close()

	// 5. Event publisher: in-memory always; webhook if configured.
	memPub := events.NewInMemory()
	subs := []events.Publisher{memPub}
	if u := getenv("CRUCIBLE_WEBHOOK_URL", ""); u != "" {
		subs = append(subs, events.NewWebhook(u, logger))
		logger.Info("webhook publisher wired", "url", u)
	}
	publisherFanout := events.NewMulti(logger, subs...)

	// 6. Tenant policy loader.
	tenants := tenantpolicy.NewLoader()

	// 7. Task store.
	tasks := store.New()

	// 8. Verifier bridge (Phase 4). When CRUCIBLE_VERIFIER_ADDR is set,
	// we wire the HTTP bridge to crucible-verifier; otherwise the API
	// reports stub_verifier=true and the verify endpoint returns 503.
	var verifierBridge verifierbridge.Bridge
	if addr := os.Getenv(verifierbridge.EnvVerifierAddr); addr != "" {
		verifierBridge = verifierbridge.New()
		logger.Info("verifier bridge wired", "addr", addr)
	} else {
		logger.Warn("CRUCIBLE_VERIFIER_ADDR unset — verify endpoint will return 503. " +
			"Start the crucible-verifier daemon and set CRUCIBLE_VERIFIER_ADDR to enable verification.")
	}

	// 8b. Promotion bridge (Phase 6). When CRUCIBLE_PROMOTION_GATE_ADDR is set,
	// the bridge dispatches to apps/promotion-gate. Otherwise stub_promotion=true.
	promotionBridge := promotionbridge.New()
	if promotionBridge != nil {
		logger.Info("promotion-gate bridge wired", "addr", os.Getenv(promotionbridge.EnvGateAddr))
	} else {
		logger.Warn("CRUCIBLE_PROMOTION_GATE_ADDR unset — promote endpoint will return 503. " +
			"Start crucible-promotion-gate and set the env var to enable promotion.")
	}

	// 9. API.
	server := &api.Server{
		Store:           tasks,
		Router:          router,
		PlanBuilder:     builder,
		Budgets:         enforcers,
		Attestation:     attestSvc,
		VerifierBridge:  verifierBridge,
		PromotionBridge: promotionBridge,
		Logger:          logger,
		DefaultTenant:   getenv("CRUCIBLE_DEFAULT_TENANT", "single-tenant"),
		Version:         version,
	}

	// Silence unused-var warnings for things wired but invoked by stub paths
	// the API doesn't reach yet. They'll be live in Phase 2's runtime loop.
	_ = meter
	_ = publisherFanout
	_ = tenants

	addr := getenv("CRUCIBLE_LISTEN_ADDR", ":8080")
	httpSrv := &http.Server{
		Addr:              addr,
		Handler:           server.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		logger.Info("control-plane listening",
			"addr", addr,
			"version", version,
			"vendors", vendors,
		)
		if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("http server failed", "err", err)
			cancel()
		}
	}()

	<-ctx.Done()
	logger.Info("control-plane shutting down")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	if err := httpSrv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}
	return nil
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
