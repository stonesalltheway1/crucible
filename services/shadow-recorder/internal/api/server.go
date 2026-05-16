// Package api is the HTTP front for crucible-shadow-recorder.
package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/crucible/services/shadow-recorder/internal/coverage"
	"github.com/crucible/services/shadow-recorder/internal/recorder"
	"github.com/crucible/services/shadow-recorder/internal/storage"
	"github.com/crucible/services/shadow-recorder/internal/types"
)

// Config wires the server.
type Config struct {
	Version  string
	Recorder *recorder.Recorder
	Coverage *coverage.Tracker
	Storage  storage.Store
}

// NewServer wires the routes.
func NewServer(cfg Config) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok", "version": cfg.Version})
	})
	mux.HandleFunc("/version", func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprintln(w, cfg.Version)
	})
	mux.HandleFunc("/v1/ingest/envoy", makeIngest(cfg))
	mux.HandleFunc("/v1/ingest/ebpf", makeIngest(cfg))
	mux.HandleFunc("/v1/coverage", func(w http.ResponseWriter, r *http.Request) {
		ten := r.URL.Query().Get("tenant_id")
		if ten == "" {
			ten = r.Header.Get("X-Crucible-Tenant")
		}
		if ten == "" {
			http.Error(w, "tenant_id required", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(cfg.Coverage.AllHosts(ten))
	})
	mux.HandleFunc("/v1/coverage/", func(w http.ResponseWriter, r *http.Request) {
		host := strings.TrimPrefix(r.URL.Path, "/v1/coverage/")
		ten := r.URL.Query().Get("tenant_id")
		if ten == "" || host == "" {
			http.Error(w, "tenant_id + host required", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(cfg.Coverage.HostCoverage(ten, host))
	})
	mux.HandleFunc("/v1/rerecord/run", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		n := cfg.Recorder.RunDueRerecords(context.Background())
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]int{"hosts_refreshed": n})
	})
	mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		c, sf := cfg.Recorder.Stats()
		w.Header().Set("Content-Type", "text/plain; version=0.0.4")
		fmt.Fprintf(w, "# HELP crucible_shadow_captures_total Total captures ingested.\n")
		fmt.Fprintf(w, "# TYPE crucible_shadow_captures_total counter\n")
		fmt.Fprintf(w, "crucible_shadow_captures_total %d\n", c)
		fmt.Fprintf(w, "# HELP crucible_shadow_scrub_failures_total Scrubber-failure events.\n")
		fmt.Fprintf(w, "# TYPE crucible_shadow_scrub_failures_total counter\n")
		fmt.Fprintf(w, "crucible_shadow_scrub_failures_total %d\n", sf)
	})
	return mux
}

func makeIngest(cfg Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var entry types.EnvoyAccessLogEntry
		if err := json.NewDecoder(r.Body).Decode(&entry); err != nil {
			http.Error(w, "bad json: "+err.Error(), http.StatusBadRequest)
			return
		}
		key, err := cfg.Recorder.Ingest(r.Context(), entry)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]string{"key": key})
	}
}
