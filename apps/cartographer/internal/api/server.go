// Package api is the HTTP front for crucible-cartographer.
//
// Endpoints:
//
//   POST /v1/cartography                — submit a job (returns job_id)
//   GET  /v1/cartography/{id}           — fetch the job result
//   GET  /v1/cartography/{id}/events    — SSE stream for live progress
//   GET  /healthz                        — health probe
//   GET  /version                        — version string
package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/crucible/apps/cartographer/internal/distill"
	"github.com/crucible/apps/cartographer/internal/orchestrator"
	"github.com/crucible/apps/cartographer/internal/oss"
	"github.com/crucible/apps/cartographer/internal/types"
)

// Config wires the server.
type Config struct {
	Version           string
	MemoryRouterAddr  string
	OSSDefaultsLoader *oss.Loader
	LLMClient         *distill.Client
}

// Server is an HTTP handler.
type Server struct {
	cfg    Config
	jobsMu sync.Mutex
	jobs   map[string]*jobState
}

type jobState struct {
	mu       sync.Mutex
	job      types.CartographyJob
	status   types.JobStatus
	result   *types.CartographyResult
	progress []progressEvent
	subs     []chan progressEvent
}

type progressEvent struct {
	Stage    string  `json:"stage"`
	Progress float64 `json:"progress"`
	At       time.Time `json:"at"`
}

// NewServer wires the HTTP routes.
func NewServer(cfg Config) http.Handler {
	srv := &Server{cfg: cfg, jobs: map[string]*jobState{}}
	mux := http.NewServeMux()
	mux.HandleFunc("/", srv.handleRoot)
	mux.HandleFunc("/healthz", srv.handleHealthz)
	mux.HandleFunc("/version", srv.handleVersion)
	mux.HandleFunc("/v1/cartography", srv.handleSubmit)
	mux.HandleFunc("/v1/cartography/", srv.handleByID)
	return mux
}

func (s *Server) handleRoot(w http.ResponseWriter, r *http.Request) {
	// `/` is special — net/http's ServeMux matches it as a catch-all.
	// Distinguish a real visit to `/` from any unmatched route below
	// `/v1/...` so unrecognised endpoints still return 404.
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"service": "crucible-cartographer",
		"version": s.cfg.Version,
		"docs":    "https://github.com/stonesalltheway1/crucible#readme",
		"endpoints": map[string]string{
			"GET  /healthz":                          "liveness probe",
			"GET  /version":                          "version string",
			"POST /v1/cartography":                   "submit a CartographyJob (returns job_id)",
			"GET  /v1/cartography/{job_id}":          "fetch the result or current status",
			"GET  /v1/cartography/{job_id}/events":   "Server-Sent Events progress stream",
		},
	})
}

func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok", "version": s.cfg.Version})
}

func (s *Server) handleVersion(w http.ResponseWriter, r *http.Request) {
	_, _ = io.WriteString(w, s.cfg.Version)
}

func (s *Server) handleSubmit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var job types.CartographyJob
	if err := json.NewDecoder(r.Body).Decode(&job); err != nil {
		http.Error(w, "bad json: "+err.Error(), http.StatusBadRequest)
		return
	}
	if job.TenantID == "" || job.Repo == "" || job.RepoLocalPath == "" {
		http.Error(w, "tenant_id, repo, repo_local_path required", http.StatusBadRequest)
		return
	}
	if job.JobID == "" {
		job.JobID = fmt.Sprintf("carto_%d", time.Now().UnixNano())
	}
	job.EnqueuedAt = time.Now().UTC()

	st := &jobState{
		job:    job,
		status: types.JobStatus{JobID: job.JobID, State: "queued", UpdatedAt: time.Now().UTC()},
	}
	s.jobsMu.Lock()
	s.jobs[job.JobID] = st
	s.jobsMu.Unlock()

	go s.runJob(st)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(map[string]string{"job_id": job.JobID})
}

func (s *Server) handleByID(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/v1/cartography/")
	id = strings.TrimSuffix(id, "/events")
	id = strings.TrimSuffix(id, "/")
	id = strings.TrimSpace(id)
	if id == "" {
		http.NotFound(w, r)
		return
	}
	s.jobsMu.Lock()
	st, ok := s.jobs[id]
	s.jobsMu.Unlock()
	if !ok {
		http.NotFound(w, r)
		return
	}
	if strings.HasSuffix(r.URL.Path, "/events") {
		s.handleEvents(w, r, st)
		return
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	if st.result != nil {
		_ = json.NewEncoder(w).Encode(st.result)
		return
	}
	_ = json.NewEncoder(w).Encode(st.status)
}

func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request, st *jobState) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache, no-transform")
	w.Header().Set("Connection", "keep-alive")

	ch := make(chan progressEvent, 32)
	st.mu.Lock()
	for _, e := range st.progress {
		ch <- e
	}
	st.subs = append(st.subs, ch)
	resultDone := st.result != nil
	st.mu.Unlock()

	defer func() {
		st.mu.Lock()
		for i, c := range st.subs {
			if c == ch {
				st.subs = append(st.subs[:i], st.subs[i+1:]...)
				break
			}
		}
		st.mu.Unlock()
	}()

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case e, alive := <-ch:
			if !alive {
				return
			}
			payload, _ := json.Marshal(e)
			fmt.Fprintf(w, "event: progress\ndata: %s\n\n", payload)
			flusher.Flush()
			if e.Stage == "done" || e.Stage == "error" {
				return
			}
		case <-time.After(15 * time.Second):
			fmt.Fprintf(w, ": keep-alive\n\n")
			flusher.Flush()
			if resultDone {
				return
			}
		}
	}
}

func (s *Server) runJob(st *jobState) {
	st.mu.Lock()
	st.status.State = "running"
	st.status.UpdatedAt = time.Now().UTC()
	st.mu.Unlock()

	progress := func(stage string, frac float64) {
		ev := progressEvent{Stage: stage, Progress: frac, At: time.Now().UTC()}
		st.mu.Lock()
		st.progress = append(st.progress, ev)
		st.status.Stage = stage
		st.status.StageProgress = frac
		st.status.UpdatedAt = ev.At
		subs := append([]chan progressEvent(nil), st.subs...)
		st.mu.Unlock()
		for _, c := range subs {
			select {
			case c <- ev:
			default: // slow subscriber; drop frame.
			}
		}
	}

	deps := orchestrator.Deps{
		OSS: s.cfg.OSSDefaultsLoader,
		LLM: s.cfg.LLMClient,
	}
	// Resolve GitHub token from the secret-ref envelope. In production
	// the control plane sidecars Infisical and resolves the secret;
	// for dev we fall back to the GITHUB_TOKEN env var.
	if ref := st.job.GitHubTokenSecretRef; ref != "" {
		deps.GitHubToken = resolveSecret(ref)
	}

	ctx, cancel := context.WithTimeout(context.Background(), wallClockBudget(st.job))
	defer cancel()
	res, err := orchestrator.Run(ctx, st.job, deps, progress)
	st.mu.Lock()
	defer st.mu.Unlock()
	if err != nil {
		st.status.State = "error"
		st.status.Error = err.Error()
		st.status.UpdatedAt = time.Now().UTC()
		// Tell every subscriber.
		for _, c := range st.subs {
			select {
			case c <- progressEvent{Stage: "error", Progress: 1, At: time.Now().UTC()}:
			default:
			}
		}
		return
	}
	st.result = res
	st.status.State = "done"
	st.status.Stage = "done"
	st.status.StageProgress = 1
	st.status.UpdatedAt = time.Now().UTC()
}

func wallClockBudget(job types.CartographyJob) time.Duration {
	if job.WallClockBudget == "" {
		return 30 * time.Minute
	}
	if d, err := time.ParseDuration(job.WallClockBudget); err == nil {
		return d
	}
	return 30 * time.Minute
}

// resolveSecret is a placeholder for the Infisical / KMS integration.
// In dev / test the env var path is sufficient. In production the
// control plane sidecars Infisical and resolves the secret BEFORE
// passing the job to cartographer; the secret-ref envelope is here
// for forward compatibility.
func resolveSecret(ref string) string {
	if strings.HasPrefix(ref, "env:") {
		return getEnv(strings.TrimPrefix(ref, "env:"))
	}
	if strings.HasPrefix(ref, "infisical://") {
		// The control plane should have resolved this. If we get
		// here, the secret was not pre-resolved — fail-soft and
		// return empty.
		return ""
	}
	return ref
}

func getEnv(name string) string {
	v, _ := envLookup(name)
	return v
}

// envLookup is split into its own var so tests can monkey-patch.
var envLookup = func(name string) (string, bool) {
	v, ok := lookupEnvImpl(name)
	if !ok {
		return "", false
	}
	return v, true
}

// Errors.
var (
	ErrJobNotFound = errors.New("api: job not found")
)
