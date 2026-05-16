// Package api is the verifier daemon's HTTP server. It exposes the
// twin.verify.* methods the control plane invokes once an executor agent
// claims `done`.
//
// Wire-format: JSON over HTTP. Each method has the shape
//   POST /v1/twin/verify/{tier|bundle}
//   Body: VerificationRequest JSON
//   Resp: VerifierApproval | VerifierRejection JSON (or {error: ...} on 4xx)
//
// We do NOT use gRPC at Phase 4 to keep the dependency graph slim — the
// control plane's existing HTTP client speaks JSON. Phase 6+ migrates
// to gRPC alongside the rest of twin-spec.
package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/crucible/verifier/internal/dispatcher"
	"github.com/crucible/verifier/internal/verification"
)

// Server is the HTTP surface.
type Server struct {
	Dispatcher *dispatcher.Dispatcher
	Logger     *slog.Logger
	Version    string
}

// Handler returns an http.Handler with all routes mounted.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", s.handleHealth)
	mux.HandleFunc("/v1/twin/verify/bundle", s.handleBundle)
	mux.HandleFunc("/v1/twin/verify/audit", s.handleAuditOnly)
	return s.middleware(mux)
}

// middleware wraps every handler with request-ID + panic-recover.
func (s *Server) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if x := recover(); x != nil {
				s.logger().Error("panic in handler", "err", x, "path", r.URL.Path)
				writeError(w, http.StatusInternalServerError, "internal_error", fmt.Sprintf("%v", x))
			}
		}()
		w.Header().Set("X-Crucible-Verifier-Version", s.Version)
		next.ServeHTTP(w, r)
	})
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status":  "ok",
		"version": s.Version,
		"time":    time.Now().UTC(),
	})
}

func (s *Server) handleBundle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "")
		return
	}
	req, err := s.parseRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 65*time.Minute)
	defer cancel()

	start := time.Now()
	resp, err := s.Dispatcher.Dispatch(ctx, req)
	dur := time.Since(start)
	if err != nil {
		s.logger().Error("verify.bundle failed",
			"task_id", req.TaskID, "tenant_id", req.TenantID, "err", err, "dur", dur)
		writeError(w, http.StatusUnprocessableEntity, "verification_failed", err.Error())
		return
	}
	s.logger().Info("verify.bundle done",
		"task_id", req.TaskID,
		"approved", resp.Approval != nil,
		"reasons", reasonCount(resp),
		"dur", dur,
	)
	writeJSON(w, http.StatusOK, resp)
}

// handleAuditOnly runs JUST the executor-reasoning leak audit on a
// candidate request. The control plane uses this to validate its own
// payload shape before submission — cheap defence in depth.
func (s *Server) handleAuditOnly(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "")
		return
	}
	req, err := s.parseRequest(r)
	if err != nil {
		// Distinguish leak from generic parse error: if Validate
		// returned a LeakageError or SameFamilyError, surface as 422.
		if isAuditFailure(err) {
			writeError(w, http.StatusUnprocessableEntity, "audit_failed", err.Error())
			return
		}
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"task_id": req.TaskID,
		"audit":   "ok",
	})
}

func (s *Server) parseRequest(r *http.Request) (*verification.VerificationRequest, error) {
	if r.Body == nil {
		return nil, fmt.Errorf("empty body")
	}
	defer r.Body.Close()
	body, err := io.ReadAll(io.LimitReader(r.Body, 32<<20)) // 32 MiB cap
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}
	// Audit-pass 1: parse into a generic map first and run the
	// reasoning-leak scan there too. This catches fields that aren't
	// declared on VerificationRequest (defence in depth).
	var generic map[string]any
	if err := json.Unmarshal(body, &generic); err != nil {
		return nil, fmt.Errorf("parse request: %w", err)
	}
	if err := verification.AuditNoLeakage(generic); err != nil {
		return nil, err
	}
	var req verification.VerificationRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return nil, fmt.Errorf("decode request: %w", err)
	}
	if err := req.Validate(); err != nil {
		return nil, err
	}
	if err := req.AuditNoLeakage(); err != nil {
		return nil, err
	}
	return &req, nil
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"error": map[string]any{
			"code":    code,
			"message": message,
		},
	})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func reasonCount(resp *verification.VerificationResponse) int {
	if resp == nil || resp.Rejection == nil {
		return 0
	}
	return len(resp.Rejection.RejectionReasons)
}

func isAuditFailure(err error) bool {
	// Defensive substring check; the typed errors are exported on
	// verification but we accept either path.
	s := err.Error()
	return strings.Contains(s, "executor-reasoning leak") ||
		strings.Contains(s, "share vendor lineage")
}

func (s *Server) logger() *slog.Logger {
	if s.Logger != nil {
		return s.Logger
	}
	return slog.Default()
}
