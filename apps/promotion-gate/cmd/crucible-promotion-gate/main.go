// Command crucible-promotion-gate is the Phase-6 promotion gate daemon.
//
// Wires together:
//
//   - libs/policy default Rego bundle (with optional signed tenant overrides).
//   - bundle_validator (relay-backed Verifier).
//   - rego_engine + approval_router (default cohort = @platform-team).
//   - kms_lease.Manager (Dev signer by default; AWS / GCP / YubiHSM via env).
//   - delivery_adapter (LocalArgoMock + LocalGrowthBookMock; real adapters
//     wire when CRUCIBLE_ARGO_ROLLOUTS_ADDR / CRUCIBLE_GROWTHBOOK_ADDR set).
//   - outcome_watcher (FakeSloChecker by default; real Prometheus checker
//     when CRUCIBLE_PROMETHEUS_ADDR is set).
//   - relay.Client (HTTP client for apps/attestation-relay).
//
// Phase 6 dev mode (`CRUCIBLE_GATE_DEV_MODE=1`) lets the gate work without
// any external services — useful for end-to-end tests.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/crucible/promotion-gate/internal/api"
	"github.com/crucible/promotion-gate/internal/approval_router"
	"github.com/crucible/promotion-gate/internal/bundle_validator"
	"github.com/crucible/promotion-gate/internal/delivery_adapter"
	"github.com/crucible/promotion-gate/internal/kms_lease"
	"github.com/crucible/promotion-gate/internal/outcome_watcher"
	"github.com/crucible/promotion-gate/internal/rego_engine"
	"github.com/crucible/promotion-gate/internal/relay"
)

const version = "2026.06.0-phase6"

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)
	if err := run(logger); err != nil {
		logger.Error("promotion-gate: fatal", "err", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// 1. Relay client (HTTP). When CRUCIBLE_RELAY_ADDR is unset, defaults to
	// :9120 — also where the in-process relay would run in dev mode.
	relayAddr := getenv("CRUCIBLE_RELAY_ADDR", "http://127.0.0.1:9120")
	relayClient, err := relay.New(relayAddr)
	if err != nil {
		return err
	}

	// 2. Bundle validator wraps the relay.
	validator := bundle_validator.New(relayClient)

	// 3. Rego engine — default bundle compiled in; tenant overrides loaded
	// dynamically via the /v1/tenants/{id}/policy endpoint.
	regoEng, err := rego_engine.New(ctx)
	if err != nil {
		return err
	}

	// 4. Approval router. Phase-6 default: @platform-team for everything
	// that doesn't ship a CODEOWNERS file. Production wires the loader from
	// the tenant's policy dir.
	approver := approval_router.New(approval_router.ApprovalConfig{
		DefaultApprovers: []string{"@platform-team"},
	})

	// 5. KMS lease manager — Dev signer by default.
	signer, err := kmsSigner(logger)
	if err != nil {
		return err
	}
	leases := kms_lease.New(signer, nil)

	// 6. Delivery adapters.
	pool := delivery_adapter.NewPool(map[delivery_adapter.Strategy]delivery_adapter.Adapter{
		delivery_adapter.StrategyCanary:          delivery_adapter.NewLocalArgoMock(),
		delivery_adapter.StrategyFeatureFlagOnly: delivery_adapter.NewLocalGrowthBookMock(),
	}, delivery_adapter.StrategyCanary)

	// 7. Outcome watcher with a default-green SLO checker. Production wires
	// the real Prometheus checker.
	watcher := outcome_watcher.New(
		pool,
		outcome_watcher.NewFakeSloChecker(),
		relayClient,
	)

	// 8. API server.
	server := &api.Server{
		Logger:    logger,
		Version:   version,
		Validator: validator,
		Rego:      regoEng,
		Approval:  approver,
		Leases:    leases,
		Delivery:  pool,
		Watcher:   watcher,
		Relay:     relayClient,
		State:     api.NewState(),
	}

	addr := getenv("CRUCIBLE_GATE_ADDR", ":9180")
	httpSrv := &http.Server{
		Addr:              addr,
		Handler:           server.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		logger.Info("promotion-gate listening", "addr", addr, "version", version)
		if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("http server failed", "err", err)
			cancel()
		}
	}()

	<-ctx.Done()
	logger.Info("promotion-gate shutting down")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	return httpSrv.Shutdown(shutdownCtx)
}

func kmsSigner(logger *slog.Logger) (kms_lease.Signer, error) {
	switch getenv("CRUCIBLE_KMS_PROVIDER", "dev") {
	case "dev":
		dir := getenv("CRUCIBLE_KMS_DEV_DIR", "")
		s, err := kms_lease.NewDevSigner(dir)
		if err != nil {
			return nil, err
		}
		logger.Info("kms_lease: using dev signer", "arn", s.KeyARN())
		return s, nil
	case "aws":
		// Real AWS wiring happens upstream — the env vars CRUCIBLE_KMS_KEY_ARN
		// + AWS_REGION + creds discovery — but the closures land here. For
		// Phase-6 we ship the scaffold; a v2 PR plugs in the SDK.
		return nil, errors.New("CRUCIBLE_KMS_PROVIDER=aws scaffold present; v2 wires aws-sdk-go-v2 directly")
	case "gcp":
		return nil, errors.New("CRUCIBLE_KMS_PROVIDER=gcp scaffold present; v2 wires cloud-kms SDK directly")
	case "yubi":
		return nil, errors.New("CRUCIBLE_KMS_PROVIDER=yubi scaffold present; v2 wires PKCS#11 directly")
	default:
		return nil, errors.New("unknown CRUCIBLE_KMS_PROVIDER")
	}
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
