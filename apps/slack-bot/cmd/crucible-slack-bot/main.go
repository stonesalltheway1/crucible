// Command crucible-slack-bot is the Phase-6 approval surface.
package main

import (
	"errors"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/crucible/slack-bot/internal/bot"
)

const version = "2026.06.0-phase6"

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	cfg := bot.Config{
		BindAddr:           getenv("CRUCIBLE_SLACK_BOT_ADDR", ":9280"),
		GateAddr:           getenv("CRUCIBLE_GATE_ADDR", "http://127.0.0.1:9180"),
		RelayAddr:          getenv("CRUCIBLE_RELAY_ADDR", "http://127.0.0.1:9120"),
		SlackBotToken:      getenv("SLACK_BOT_TOKEN", "xoxb-test"),
		SlackSigningSecret: getenv("SLACK_SIGNING_SECRET", "test"),
		ApproversChannel:   getenv("CRUCIBLE_APPROVERS_CHANNEL", "#crucible-approvals"),
	}
	b := bot.New(cfg)
	srv := &http.Server{
		Addr:              cfg.BindAddr,
		Handler:           b.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}
	logger.Info("slack-bot listening", "addr", cfg.BindAddr, "version", version)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Error("slack-bot fatal", "err", err)
		os.Exit(1)
	}
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
