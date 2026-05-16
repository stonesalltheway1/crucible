// Package api exposes the Phase-1 REST surface of the Crucible control plane.
//
// Architecture note: docs/02-engineering/tech-stack.md picks connect-go (gRPC
// + REST on the same handler). Phase 1 ships REST-only on net/http because
// generating connect-go stubs requires `buf generate` — which lives in Phase 2.
// The handler signatures here are 1:1 with the ControlPlaneService RPCs in
// libs/twin-spec/proto/crucible/v1/control_plane.proto, so the Phase-2 swap
// is a pure wire-format change with no business-logic edits.
package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/oklog/ulid/v2"

	"github.com/crucible/attestation"
	"github.com/crucible/control-plane/internal/budgetenforcer"
	"github.com/crucible/control-plane/internal/planbuilder"
	"github.com/crucible/control-plane/internal/promotionbridge"
	"github.com/crucible/control-plane/internal/store"
	"github.com/crucible/control-plane/internal/taskrouter"
	"github.com/crucible/control-plane/internal/verifierbridge"
	cruciblev1 "github.com/crucible/sdk-go/crucible/v1"
)

// Server bundles the Phase-1 dependencies plus the Phase-4 verifier
// bridge.
type Server struct {
	Store           *store.Store
	Router          *taskrouter.Router
	PlanBuilder     *planbuilder.Builder
	Budgets         *budgetenforcer.Registry
	Attestation     *attestation.Service
	VerifierBridge  verifierbridge.Bridge   // Phase 4: dispatches to crucible-verifier daemon
	PromotionBridge promotionbridge.Bridge // Phase 6: dispatches to crucible-promotion-gate
	Logger          *slog.Logger
	DefaultTenant   string
	Version         string
}

// Handler returns an http.Handler implementing the REST surface:
//
//   POST /v1/tasks                  → SubmitTask
//   GET  /v1/tasks                  → ListTasks
//   GET  /v1/tasks/{id}             → GetTask
//   POST /v1/tasks/{id}/approve     → ApprovePlan
//   POST /v1/tasks/{id}/reject      → RejectPlan
//   POST /v1/tasks/{id}/replan      → ReplanTask
//   GET  /v1/tasks/{id}/budget      → GetBudget
//   GET  /healthz                   → Health
//
// Phase 2 attaches the same handlers to a connect-go service.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.handleHealth)
	mux.HandleFunc("POST /v1/tasks", s.handleSubmitTask)
	mux.HandleFunc("GET /v1/tasks", s.handleListTasks)
	mux.HandleFunc("GET /v1/tasks/{id}", s.handleGetTask)
	mux.HandleFunc("POST /v1/tasks/{id}/approve", s.handleApprovePlan)
	mux.HandleFunc("POST /v1/tasks/{id}/reject", s.handleRejectPlan)
	mux.HandleFunc("POST /v1/tasks/{id}/replan", s.handleReplanTask)
	mux.HandleFunc("GET /v1/tasks/{id}/budget", s.handleGetBudget)
	mux.HandleFunc("POST /v1/tasks/{id}/verify", s.handleVerifyTask)
	mux.HandleFunc("POST /v1/tasks/{id}/promote", s.handlePromoteTask)
	return logMiddleware(s.Logger, mux)
}

// ── handlers ──────────────────────────────────────────────────────────────

type healthResponse struct {
	Status            string    `json:"status"`
	Version           string    `json:"version"`
	Now               time.Time `json:"now"`
	StubTwinRuntime   bool      `json:"stub_twin_runtime"`
	StubVerifier      bool      `json:"stub_verifier"`
	StubPromotion     bool      `json:"stub_promotion"`
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	stubVerifier := s.VerifierBridge == nil
	if !stubVerifier {
		// Health-probe the verifier daemon; if it's unreachable, surface
		// the same flag so dashboards can see the gap.
		if err := s.VerifierBridge.HealthCheck(r.Context()); err != nil {
			stubVerifier = true
		}
	}
	stubPromotion := s.PromotionBridge == nil
	if !stubPromotion {
		if err := s.PromotionBridge.HealthCheck(r.Context()); err != nil {
			stubPromotion = true
		}
	}
	writeJSON(w, http.StatusOK, healthResponse{
		Status: "ok", Version: s.Version, Now: time.Now().UTC(),
		StubTwinRuntime: true, StubVerifier: stubVerifier, StubPromotion: stubPromotion,
	})
}

type submitTaskRequest struct {
	Description       string  `json:"description"`
	Repo              string  `json:"repo"`
	BaseSha           string  `json:"base_sha"`
	TenantID          string  `json:"tenant_id"`
	CostCapUSD        float64 `json:"cost_cap_usd"`
	WallClockCapMin   uint32  `json:"wall_clock_cap_min"`
	RetryCapPerSubgoal uint32 `json:"retry_cap_per_subgoal"`
}

func (s *Server) handleSubmitTask(w http.ResponseWriter, r *http.Request) {
	var req submitTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, fmt.Errorf("invalid JSON: %w", err))
		return
	}
	if req.Description == "" {
		writeErr(w, http.StatusBadRequest, errors.New("description is required"))
		return
	}
	if req.TenantID == "" {
		req.TenantID = s.DefaultTenant
	}
	if req.Repo == "" {
		req.Repo = "(unspecified)"
	}
	if req.BaseSha == "" {
		req.BaseSha = "HEAD"
	}

	now := time.Now().UTC()
	id := "task_" + ulid.Make().String()

	// 1. Classify and route.
	classification, err := s.Router.Classify(r.Context(), req.Description)
	if err != nil {
		s.Logger.WarnContext(r.Context(), "classifier error", "err", err)
	}
	routing, err := s.Router.Route(classification)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, fmt.Errorf("routing: %w", err))
		return
	}

	task := &cruciblev1.Task{
		ID:          id,
		TenantID:    req.TenantID,
		Repo:        req.Repo,
		BaseSha:     req.BaseSha,
		Description: req.Description,
		Status:      cruciblev1.TaskStatusPlanning,
		CreatedAt:   now,
		UpdatedAt:   now,
		SubmittedBy: s.Attestation.Signer().OidcSubject(),
		Routing:     routing,
	}
	if err := s.Store.Put(task); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}

	// 2. Build a Plan.
	plan, _, err := s.PlanBuilder.Build(r.Context(), task)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, fmt.Errorf("plan build: %w", err))
		return
	}

	// 3. Determine the effective caps.
	costCap := req.CostCapUSD
	if costCap == 0 {
		costCap = plan.EstimatedCostUsd * 1.5 // ADR-009 default
		if costCap < 0.10 {
			costCap = 0.10
		}
	}
	wallCap := req.WallClockCapMin
	if wallCap == 0 {
		wallCap = plan.WallClockBudgetMin
		if wallCap == 0 {
			wallCap = 60
		}
	}
	retryCap := req.RetryCapPerSubgoal
	if retryCap == 0 {
		retryCap = 3
	}

	// 4. Spin up a per-task Enforcer.
	enf, err := budgetenforcer.New(budgetenforcer.Config{
		TaskID:             id,
		CostCapUSD:         costCap,
		WallClockCapMin:    wallCap,
		RetryCapPerSubgoal: retryCap,
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, fmt.Errorf("budget: %w", err))
		return
	}
	s.Budgets.Register(id, enf)
	budget := enf.Snapshot()

	updated, err := s.Store.Update(id, func(t *cruciblev1.Task) error {
		t.Plan = plan
		t.Budget = budget
		t.Status = cruciblev1.TaskStatusAwaitingApproval
		return nil
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"task": updated,
	})
}

func (s *Server) handleGetTask(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	t, err := s.Store.Get(id)
	if err != nil {
		writeErr(w, http.StatusNotFound, err)
		return
	}
	if enf := s.Budgets.Get(id); enf != nil {
		t.Budget = enf.Snapshot()
	}
	writeJSON(w, http.StatusOK, map[string]any{"task": t})
}

func (s *Server) handleListTasks(w http.ResponseWriter, r *http.Request) {
	tenant := r.URL.Query().Get("tenant_id")
	if tenant == "" {
		tenant = s.DefaultTenant
	}
	limit := 50
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}
	tasks := s.Store.List(tenant, limit)
	writeJSON(w, http.StatusOK, map[string]any{"tasks": tasks})
}

type approvePlanRequest struct {
	PlanHash             string  `json:"plan_hash"`
	ApproverOidcSubject  string  `json:"approver_oidc_subject"`
	CostCapUSD           float64 `json:"cost_cap_usd"`
	WallClockCapMin      uint32  `json:"wall_clock_cap_min"`
	RetryCapPerSubgoal   uint32  `json:"retry_cap_per_subgoal"`
}

func (s *Server) handleApprovePlan(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req approvePlanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, errEmptyBody) {
		writeErr(w, http.StatusBadRequest, fmt.Errorf("invalid JSON: %w", err))
		return
	}
	approver := req.ApproverOidcSubject
	if approver == "" {
		approver = s.Attestation.Signer().OidcSubject()
	}

	now := time.Now().UTC()
	var approval *cruciblev1.PlanApproval
	updated, err := s.Store.Update(id, func(t *cruciblev1.Task) error {
		if t.Plan == nil {
			return errors.New("task has no plan")
		}
		if req.PlanHash != "" && req.PlanHash != t.Plan.PlanHash {
			return fmt.Errorf("plan hash mismatch: got %s, expected %s", req.PlanHash, t.Plan.PlanHash)
		}
		if t.Status != cruciblev1.TaskStatusAwaitingApproval &&
			t.Status != cruciblev1.TaskStatusPlanning {
			return fmt.Errorf("task is in state %s; cannot approve", t.Status)
		}
		approval = &cruciblev1.PlanApproval{
			TaskID:              id,
			PlanHash:            t.Plan.PlanHash,
			ApproverOidcSubject: approver,
			ApprovedAt:          now,
			CostCapUsd:          req.CostCapUSD,
			WallClockCapMin:     req.WallClockCapMin,
			RetryCapPerSubgoal:  req.RetryCapPerSubgoal,
		}
		if approval.CostCapUsd == 0 && t.Budget != nil {
			approval.CostCapUsd = t.Budget.CapUsd
		}
		if approval.WallClockCapMin == 0 && t.Plan != nil {
			approval.WallClockCapMin = t.Plan.WallClockBudgetMin
		}
		if approval.RetryCapPerSubgoal == 0 && t.Budget != nil {
			approval.RetryCapPerSubgoal = t.Budget.RetryCap
		}
		t.Status = cruciblev1.TaskStatusApproved
		return nil
	})
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}

	// Emit a PlanApproval attestation. Subject = plan_hash + "approved".
	digestInput := []byte(approval.PlanHash + "/" + approver)
	predicate := cruciblev1.PlanApprovalAttestation{
		TaskID:           id,
		PlanHash:         approval.PlanHash,
		EstimatedCostUsd: updated.Plan.EstimatedCostUsd,
		ApprovedByOidc:   approver,
		ApprovedAt:       now,
	}
	if entry, attErr := s.Attestation.Emit(r.Context(),
		cruciblev1.PredicatePlanApproval,
		fmt.Sprintf("task/%s/plan-approval", id),
		digestInput,
		predicate,
	); attErr == nil {
		approval.AttestationID = entry.UUID
	} else {
		s.Logger.WarnContext(r.Context(), "plan-approval attestation failed", "err", attErr)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"task":     updated,
		"approval": approval,
	})
}

type rejectPlanRequest struct {
	PlanHash             string `json:"plan_hash"`
	RejecterOidcSubject  string `json:"rejecter_oidc_subject"`
	Reason               string `json:"reason"`
}

func (s *Server) handleRejectPlan(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req rejectPlanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, errEmptyBody) {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	rejecter := req.RejecterOidcSubject
	if rejecter == "" {
		rejecter = s.Attestation.Signer().OidcSubject()
	}
	now := time.Now().UTC()
	var rej *cruciblev1.PlanRejection
	updated, err := s.Store.Update(id, func(t *cruciblev1.Task) error {
		if t.Plan == nil {
			return errors.New("task has no plan")
		}
		if req.PlanHash != "" && req.PlanHash != t.Plan.PlanHash {
			return fmt.Errorf("plan hash mismatch: got %s, expected %s", req.PlanHash, t.Plan.PlanHash)
		}
		rej = &cruciblev1.PlanRejection{
			TaskID:              id,
			PlanHash:            t.Plan.PlanHash,
			Reason:              req.Reason,
			RejecterOidcSubject: rejecter,
			RejectedAt:          now,
		}
		t.Status = cruciblev1.TaskStatusRejected
		return nil
	})
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"task": updated, "rejection": rej})
}

type replanRequest struct {
	Reason          string  `json:"reason"`
	CostCapUSD      float64 `json:"cost_cap_usd"`
	WallClockCapMin uint32  `json:"wall_clock_cap_min"`
}

func (s *Server) handleReplanTask(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req replanRequest
	_ = json.NewDecoder(r.Body).Decode(&req)

	task, err := s.Store.Get(id)
	if err != nil {
		writeErr(w, http.StatusNotFound, err)
		return
	}
	plan, _, err := s.PlanBuilder.Build(r.Context(), task)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	if req.CostCapUSD == 0 {
		req.CostCapUSD = plan.EstimatedCostUsd * 1.5
		if req.CostCapUSD < 0.10 {
			req.CostCapUSD = 0.10
		}
	}
	if req.WallClockCapMin == 0 {
		req.WallClockCapMin = plan.WallClockBudgetMin
		if req.WallClockCapMin == 0 {
			req.WallClockCapMin = 60
		}
	}
	enf, err := budgetenforcer.New(budgetenforcer.Config{
		TaskID:             id,
		CostCapUSD:         req.CostCapUSD,
		WallClockCapMin:    req.WallClockCapMin,
		RetryCapPerSubgoal: 3,
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	s.Budgets.Register(id, enf)
	budget := enf.Snapshot()
	updated, err := s.Store.Update(id, func(t *cruciblev1.Task) error {
		t.Plan = plan
		t.Budget = budget
		t.Status = cruciblev1.TaskStatusAwaitingApproval
		return nil
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"task": updated, "reason": req.Reason})
}

func (s *Server) handleGetBudget(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	enf := s.Budgets.Get(id)
	if enf == nil {
		t, err := s.Store.Get(id)
		if err != nil {
			writeErr(w, http.StatusNotFound, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"budget": t.Budget})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"budget": enf.Snapshot()})
}

// ── helpers ───────────────────────────────────────────────────────────────

var errEmptyBody = errors.New("empty body")

func writeJSON(w http.ResponseWriter, code int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		// Headers are already written; nothing useful to do but log.
		_ = err
	}
}

func writeErr(w http.ResponseWriter, code int, err error) {
	writeJSON(w, code, map[string]any{
		"error":   http.StatusText(code),
		"message": err.Error(),
	})
}

func logMiddleware(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rid := ulid.Make().String()
		ctx := context.WithValue(r.Context(), ctxKeyRequestID{}, rid)
		ww := &statusRecorder{ResponseWriter: w, code: 200}
		next.ServeHTTP(ww, r.WithContext(ctx))
		logger.InfoContext(ctx, "http",
			"rid", rid,
			"method", r.Method,
			"path", r.URL.Path,
			"status", ww.code,
			"duration_ms", time.Since(start).Milliseconds(),
		)
	})
}

type statusRecorder struct {
	http.ResponseWriter
	code int
}

func (s *statusRecorder) WriteHeader(c int) {
	s.code = c
	s.ResponseWriter.WriteHeader(c)
}

type ctxKeyRequestID struct{}
