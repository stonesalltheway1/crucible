// Package main is the entry point for crucible-shadow-recorder.
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

	"github.com/crucible/services/shadow-recorder/internal/api"
	"github.com/crucible/services/shadow-recorder/internal/coverage"
	"github.com/crucible/services/shadow-recorder/internal/recorder"
	"github.com/crucible/services/shadow-recorder/internal/scrubber"
	"github.com/crucible/services/shadow-recorder/internal/storage"
)

const version = "2026.06.0-phase8"

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))
	slog.SetDefault(logger)

	addr := envOr("CRUCIBLE_SHADOW_LISTEN", ":9520")
	scrubURL := envOr("CRUCIBLE_SCRUBBER_URL", "")
	objectStoreURI := envOr("CRUCIBLE_SHADOW_OBJECTSTORE", "")
	failClosed := envOr("CRUCIBLE_SHADOW_FAIL_CLOSED", "1") == "1"

	scr := scrubber.NewClient(scrubber.Config{
		Endpoint:   scrubURL,
		FailClosed: failClosed,
	})
	store := storage.NewMemoryStore()
	if objectStoreURI != "" {
		store = storage.NewObjectStore(objectStoreURI)
	}
	cov := coverage.New()
	rec := recorder.New(recorder.Config{
		Scrubber: scr,
		Store:    store,
		Coverage: cov,
		// Per-host re-record schedule defaults to 30 days.
		RerecordEvery: 30 * 24 * time.Hour,
		RetentionDays: 90,
	})

	srv := api.NewServer(api.Config{
		Version:  version,
		Recorder: rec,
		Coverage: cov,
		Storage:  store,
	})

	httpSrv := &http.Server{Addr: addr, Handler: srv, ReadHeaderTimeout: 10 * time.Second}

	// Background cron: re-record schedule scan every hour.
	stopCron := make(chan struct{})
	go cronLoop(rec, stopCron)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	go func() {
		slog.Info("crucible-shadow-recorder listening",
			"addr", addr, "version", version,
			"scrubber", scrubURL, "objectstore", objectStoreURI,
			"fail_closed", failClosed,
		)
		if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("listen failed", "err", err)
			os.Exit(1)
		}
	}()
	<-ctx.Done()
	close(stopCron)
	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelShutdown()
	_ = httpSrv.Shutdown(shutdownCtx)
}

func cronLoop(r *recorder.Recorder, stop <-chan struct{}) {
	t := time.NewTicker(time.Hour)
	defer t.Stop()
	for {
		select {
		case <-stop:
			return
		case <-t.C:
			if n := r.RunDueRerecords(context.Background()); n > 0 {
				slog.Info("re-record cycle complete", "hosts_refreshed", n)
			}
		}
	}
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
