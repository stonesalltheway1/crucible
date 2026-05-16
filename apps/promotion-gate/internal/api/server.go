// Package api is the HTTP surface of the promotion gate.
//
// Endpoints:
//
//   POST /v1/promotions                       — submit a PromotionBundle
//   GET  /v1/promotions/{id}                  — status
//   POST /v1/promotions/{id}/approve          — human approver
//   POST /v1/promotions/{id}/reject           — human approver
//   POST /v1/promotions/{id}/rollback         — admin trigger
//   POST /v1/tenants/{id}/policy              — load signed tenant bundle
//   GET  /healthz                             — health
package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/oklog/ulid/v2"

	"github.com/crucible/policy"
	"github.com/crucible/promotion-gate/internal/approval_router"
	"github.com/crucible/promotion-gate/internal/bundle_validator"
	"github.com/crucible/promotion-gate/internal/delivery_adapter"
	"github.com/crucible/promotion-gate/internal/kms_lease"
	"github.com/crucible/promotion-gate/internal/outcome_watcher"
	"github.com/crucible/promotion-gate/internal/rego_engine"
	"github.com/crucible/promotion-gate/internal/relay"
	cruciblev1 "github.com/crucible/sdk-go/crucible/v1"
)

// Server bundles the gate's components.
type Server struct {
	Logger       *slog.Logger
	Version      string
	Validator    *bundle_validator.Validator
	Rego         *rego_engine.Engine
	Approval     *approval_router.Router
	Leases       *kms_lease.Manager
	Delivery     *delivery_adapter.Pool
	Watcher      *outcome_watcher.Watcher
	Relay        *relay.Client
	State        *State

	// EventSink is an optional webhook publisher. The control plane wires
	// its events.Publisher in here.
	EventSink EventSink
}

// EventSink is the gate's webhook contract.
type EventSink interface {
	Publish(ctx context.Context, eventType string, payload map[string]any) error
}

// Handler returns the HTTP handler.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.handleHealth)
	mux.HandleFunc("POST /v1/promotions", s.handleSubmit)
	mux.HandleFunc("GET /v1/promotions/{id}", s.handleGet)
	mux.HandleFunc("POST /v1/promotions/{id}/approve", s.handleApprove)
	mux.HandleFunc("POST /v1/promotions/{id}/reject", s.handleReject)
	mux.HandleFunc("POST /v1/promotions/{id}/rollback", s.handleRollback)
	mux.HandleFunc("POST /v1/tenants/{id}/policy", s.handleLoadTenantPolicy)
	return s
}

// ServeHTTP implements http.Handler with a tiny logging middleware.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Avoid wrapping w to keep things simple; production wraps for status.
	s.routes().ServeHTTP(w, r)
	s.Logger.LogAttrs(r.Context(), slog.LevelInfo, "http",
		slog.String("method", r.Method),
		slog.String("path", r.URL.Path),
		slog.Duration("elapsed", time.Since(start)),
	)
}

func (s *Server) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.handleHealth)
	mux.HandleFunc("POST /v1/promotions", s.handleSubmit)
	mux.HandleFunc("GET /v1/promotions/{id}", s.handleGet)
	mux.HandleFunc("POST /v1/promotions/{id}/approve", s.handleApprove)
	mux.HandleFunc("POST /v1/promotions/{id}/reject", s.handleReject)
	mux.HandleFunc("POST /v1/promotions/{id}/rollback", s.handleRollback)
	mux.HandleFunc("POST /v1/tenants/{id}/policy", s.handleLoadTenantPolicy)
	return mux
}

// ── State ──────────────────────────────────────────────────────────────────

// State holds the in-flight promotion records.
type State struct {
	mu  sync.RWMutex
	all map[string]*Record
}

// NewState builds an empty State.
func NewState() *State { return &State{all: map[string]*Record{}} }

// Record is the promotion's lifecycle data.
type Record struct {
	ID            string                                `json:"id"`
	TenantID      string                                `json:"tenant_id"`
	Bundle        cruciblev1.PromotionBundle            `json:"bundle"`
	BundleHash    string                                `json:"bundle_hash"`
	Validation    *bundle_validator.Result              `json:"validation,omitempty"`
	Decision      *rego_engine.MergedDecision           `json:"decision,omitempty"`
	Cohort        *approval_router.Cohort               `json:"cohort,omitempty"`
	Approvals     []approval_router.Approval            `json:"approvals,omitempty"`
	Status        cruciblev1.PromotionStatusKind        `json:"status"`
	Detail        string                                `json:"detail,omitempty"`
	Lease         *kms_lease.Lease                      `json:"lease,omitempty"`
	Handle        *delivery_adapter.Handle              `json:"handle,omitempty"`
	Outcome       *cruciblev1.PromotionOutcomeAttestation `json:"outcome,omitempty"`
	UpdatedAt     time.Time                             `json:"updated_at"`
	BundleRekorUUID  string                             `json:"bundle_rekor_uuid,omitempty"`
	ApprovalRekorUUID string                            `json:"approval_rekor_uuid,omitempty"`
}

// Put stores a record.
func (s *State) Put(r *Record) {
	s.mu.Lock()
	defer s.mu.Unlock()
	r.UpdatedAt = time.Now().UTC()
	s.all[r.ID] = r
}

// Get fetches a record.
func (s *State) Get(id string) (*Record, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	r, ok := s.all[id]
	return r, ok
}

// Mutate locks the record and applies fn.
func (s *State) Mutate(id string, fn func(*Record) error) (*Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, ok := s.all[id]
	if !ok {
		return nil, errors.New("api: record not found")
	}
	if err := fn(r); err != nil {
		return nil, err
	}
	r.UpdatedAt = time.Now().UTC()
	return r, nil
}

// ── handlers ───────────────────────────────────────────────────────────────

type healthResponse struct {
	Status            string    `json:"status"`
	Version           string    `json:"version"`
	Now               time.Time `json:"now"`
	StubPromotion     bool      `json:"stub_promotion"`
	RegoBundleHash    string    `json:"rego_bundle_hash"`
	ActivePromotions  int       `json:"active_promotions"`
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	s.State.mu.RLock()
	defer s.State.mu.RUnlock()
	writeJSON(w, http.StatusOK, healthResponse{
		Status: "ok", Version: s.Version, Now: time.Now().UTC(),
		StubPromotion: false, RegoBundleHash: s.Rego.DefaultPolicyHash(),
		ActivePromotions: len(s.State.all),
	})
}

type submitRequest struct {
	Bundle   cruciblev1.PromotionBundle `json:"bundle"`
	TenantID string                     `json:"tenant_id"`
	// Context enriches the rego input — set by control plane.
	Context policy.PromotionContext `json:"context,omitempty"`
	// Caller-supplied OIDC of the agent worker that built the bundle.
	AgentOidcSubject string `json:"agent_oidc_subject,omitempty"`
	// Optional: pre-resolved CODEOWNERS for the diff.
	CodeOwners policy.CodeOwnerMatch `json:"codeowners,omitempty"`
}

func (s *Server) handleSubmit(w http.ResponseWriter, r *http.Request) {
	var req submitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, fmt.Errorf("invalid JSON: %w", err))
		return
	}
	if req.AgentOidcSubject != "" && req.Bundle.AgentOidcSubject == "" {
		req.Bundle.AgentOidcSubject = req.AgentOidcSubject
	}

	// 1. Validate the chain.
	res, err := s.Validator.Validate(r.Context(), &req.Bundle)
	if err != nil {
		writeErr(w, http.StatusUnprocessableEntity, fmt.Errorf("bundle validation: %w", err))
		return
	}

	// 2. Enrich the input for Rego.
	enrich, _ := s.Validator.EnrichInput(r.Context(), &req.Bundle, []string{req.Bundle.VerifierApprovalAttestation})
	input := buildInput(&req, res, enrich)

	// 3. Evaluate Rego.
	dec, err := s.Rego.Evaluate(r.Context(), input)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, fmt.Errorf("rego eval: %w", err))
		return
	}

	// 4. If denied outright, persist + return.
	if !dec.Allow && !dec.NeedsHuman {
		rec := &Record{
			ID:         newPromotionID(),
			TenantID:   req.TenantID,
			Bundle:     req.Bundle,
			Validation: res,
			Decision:   dec,
			Status:     cruciblev1.PromotionRejected,
			Detail:     joinReasons(dec.Reasons),
		}
		s.State.Put(rec)
		s.publish(r.Context(), "task.promotion_rejected", rec)
		writeJSON(w, http.StatusForbidden, rec)
		return
	}

	// 5. Resolve cohort.
	cohort, err := s.Approval.Resolve(dec, &req.Bundle)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, fmt.Errorf("approval resolve: %w", err))
		return
	}

	// 6. Auto-approve path: skip humans, mint lease + delivery.
	rec := &Record{
		ID:         newPromotionID(),
		TenantID:   req.TenantID,
		Bundle:     req.Bundle,
		BundleHash: bundleHashFromValidation(res),
		Validation: res,
		Decision:   dec,
		Cohort:     cohort,
		Status:     cruciblev1.PromotionPendingApproval,
	}
	if dec.AutoApprove || cohort.IsEmpty() {
		rec.Status = cruciblev1.PromotionApproved
	}
	s.State.Put(rec)
	s.publish(r.Context(), "task.promotion_proposed", rec)

	// 7. If auto-approved, immediately proceed.
	if rec.Status == cruciblev1.PromotionApproved {
		if err := s.executeApproved(r.Context(), rec, nil); err != nil {
			rec, _ = s.State.Mutate(rec.ID, func(r *Record) error {
				r.Status = cruciblev1.PromotionRejected
				r.Detail = err.Error()
				return nil
			})
			writeJSON(w, http.StatusInternalServerError, rec)
			return
		}
	}
	writeJSON(w, http.StatusAccepted, rec)
}

func (s *Server) handleGet(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	rec, ok := s.State.Get(id)
	if !ok {
		writeErr(w, http.StatusNotFound, errors.New("not found"))
		return
	}
	writeJSON(w, http.StatusOK, rec)
}

type approveRequest struct {
	ApproverOidcSubject string `json:"approver_oidc_subject"`
	Group               string `json:"group,omitempty"`
	Codeowner           bool   `json:"codeowner,omitempty"`
	Attestation         string `json:"attestation"`
	BundleHashBound     string `json:"bundle_hash_bound"`
}

func (s *Server) handleApprove(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req approveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, fmt.Errorf("invalid JSON: %w", err))
		return
	}
	rec, ok := s.State.Get(id)
	if !ok {
		writeErr(w, http.StatusNotFound, errors.New("not found"))
		return
	}
	// Stale approval check (T2).
	if req.BundleHashBound != "" && req.BundleHashBound != rec.BundleHash && req.BundleHashBound != rec.Bundle.DiffHash {
		writeErr(w, http.StatusConflict, fmt.Errorf("stale approval: bundle_hash_bound=%s != current=%s", req.BundleHashBound, rec.Bundle.DiffHash))
		return
	}
	// Self-approval (T21).
	if req.ApproverOidcSubject == rec.Bundle.AgentOidcSubject {
		writeErr(w, http.StatusForbidden, fmt.Errorf("self-approval forbidden: %s", req.ApproverOidcSubject))
		return
	}

	rec, err := s.State.Mutate(id, func(r *Record) error {
		r.Approvals = append(r.Approvals, approval_router.Approval{
			ApproverOidcSubject: req.ApproverOidcSubject,
			Attestation:         req.Attestation,
			Group:               req.Group,
			Codeowner:           req.Codeowner,
		})
		// Count & flip status if quorum.
		_, ok, _ := s.Approval.CountValid(r.Cohort, r.Bundle.AgentOidcSubject, r.Approvals)
		if ok {
			r.Status = cruciblev1.PromotionApproved
		}
		return nil
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	if rec.Status == cruciblev1.PromotionApproved {
		s.publish(r.Context(), "task.promotion_approved", rec)
		if err := s.executeApproved(r.Context(), rec, nil); err != nil {
			writeErr(w, http.StatusInternalServerError, err)
			return
		}
	}
	writeJSON(w, http.StatusOK, rec)
}

func (s *Server) handleReject(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	rec, err := s.State.Mutate(id, func(r *Record) error {
		r.Status = cruciblev1.PromotionRejected
		r.Detail = "human rejected"
		return nil
	})
	if err != nil {
		writeErr(w, http.StatusNotFound, err)
		return
	}
	s.publish(r.Context(), "task.promotion_rejected", rec)
	writeJSON(w, http.StatusOK, rec)
}

func (s *Server) handleRollback(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	rec, ok := s.State.Get(id)
	if !ok {
		writeErr(w, http.StatusNotFound, errors.New("not found"))
		return
	}
	if rec.Handle == nil {
		writeErr(w, http.StatusBadRequest, errors.New("no delivery handle"))
		return
	}
	if err := s.Delivery.Rollback(r.Context(), rec.Handle, "manual rollback"); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	rec, _ = s.State.Mutate(id, func(r *Record) error {
		r.Status = cruciblev1.PromotionRolledBack
		r.Detail = "manual rollback"
		return nil
	})
	s.publish(r.Context(), "task.promotion_rolled_back", rec)
	writeJSON(w, http.StatusOK, rec)
}

type loadTenantPolicyRequest struct {
	Signed policy.SignedTenantBundle `json:"signed_bundle"`
}

func (s *Server) handleLoadTenantPolicy(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req loadTenantPolicyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	if req.Signed.Bundle.TenantID != id {
		writeErr(w, http.StatusBadRequest, fmt.Errorf("tenant_id mismatch: url=%s bundle=%s", id, req.Signed.Bundle.TenantID))
		return
	}
	// In Phase-6 we accept the bundle as already-verified by the upstream
	// (control plane) before reaching us. Production wires a VerifierBytes
	// from the tenant's registered public key here.
	if err := s.Rego.LoadTenant(r.Context(), &req.Signed.Bundle); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"tenant_id":   id,
		"version":     req.Signed.Bundle.Version,
		"bundle_hash": req.Signed.BundleHash,
	})
}

// executeApproved runs steps 4-6 of the contract: mint lease, hand off to
// the delivery adapter, start the watcher.
func (s *Server) executeApproved(ctx context.Context, rec *Record, approverOidc *string) error {
	// 4. KMS lease.
	lease, err := s.Leases.MintLease(ctx, kms_lease.LeaseRequest{
		PromotionID: rec.ID,
		BundleHash:  rec.Bundle.DiffHash,
		Action:      leaseActionForBundle(&rec.Bundle),
		ActionTarget: map[string]string{
			"service": joinAffectedServices(&rec.Bundle),
		},
		OidcSubject: rec.Bundle.AgentOidcSubject,
	})
	if err != nil {
		return fmt.Errorf("mint lease: %w", err)
	}

	// 5. Delivery adapter.
	handle, err := s.Delivery.Start(ctx, lease, &rec.Bundle)
	if err != nil {
		return fmt.Errorf("delivery start: %w", err)
	}

	// Record state.
	s.State.Mutate(rec.ID, func(r *Record) error {
		r.Lease = lease
		r.Handle = handle
		r.Status = cruciblev1.PromotionDeploying
		return nil
	})
	s.publish(ctx, "task.promotion_deploying", rec)

	// 6. Outcome watcher — runs async.
	go func() {
		bg := context.Background()
		outcome, err := s.Watcher.RunOnce(bg, handle)
		_, _ = s.State.Mutate(rec.ID, func(r *Record) error {
			r.Outcome = &outcome
			switch outcome.Outcome {
			case "landed":
				r.Status = cruciblev1.PromotionLanded
			case "rolled_back":
				r.Status = cruciblev1.PromotionRolledBack
				r.Detail = outcome.RollbackReason
			}
			return nil
		})
		if outcome.Outcome == "landed" {
			s.publish(bg, "task.promotion_landed", rec)
		} else {
			s.publish(bg, "task.promotion_rolled_back", rec)
		}
		if err != nil {
			s.Logger.Warn("watcher returned error", "err", err)
		}
	}()

	return nil
}

// ── helpers ────────────────────────────────────────────────────────────────

func newPromotionID() string { return "prom_" + ulid.Make().String() }

func bundleHashFromValidation(r *bundle_validator.Result) string {
	return r.Bundle.DiffHash
}

func leaseActionForBundle(b *cruciblev1.PromotionBundle) string {
	for _, f := range b.FilesChanged {
		if isMigration(f.Path) {
			return kms_lease.ActionRunMigration
		}
	}
	return kms_lease.ActionDeployArtifact
}

func isMigration(p string) bool {
	return contains(p, "/migrations/") || contains(p, "/migrate/")
}

func joinAffectedServices(b *cruciblev1.PromotionBundle) string {
	if len(b.FilesChanged) == 0 {
		return ""
	}
	return b.FilesChanged[0].Path
}

func buildInput(req *submitRequest, res *bundle_validator.Result, enrich map[string]any) *policy.PromotionInput {
	br := policy.PromotionBlastRadius{
		Reversibility:    req.Bundle.BlastRadius.Reversibility,
		ImpactScore:      req.Bundle.BlastRadius.ImpactScore,
		EstimatedImpact:  estimatedImpactFromScore(req.Bundle.BlastRadius.ImpactScore),
	}
	if s, ok := enrich["schema_changes"].([]any); ok && len(s) > 0 {
		for _, x := range s {
			if m, ok := x.(map[string]any); ok {
				br.SchemaChanges = append(br.SchemaChanges, policy.SchemaChangeEntry{
					File:           asStr(m["file"]),
					DestructiveDDL: asBool(m["destructive_ddl"]),
				})
			}
		}
	}
	if c, ok := enrich["critical_paths_touched"].([]any); ok {
		for _, x := range c {
			if s, ok := x.(string); ok {
				br.CriticalPathsTouched = append(br.CriticalPathsTouched, s)
			}
		}
	}
	tr := policy.PromotionTierResults{
		Tier0: &policy.TierEntry{Passed: true},
		Tier1: &policy.TierEntry{Passed: true},
		Tier4: &policy.TierEntry{Passed: res.BuildProvenanceReferenced},
	}
	if len(br.CriticalPathsTouched) > 0 {
		tr.Tier3 = &policy.TierEntry{Passed: true} // verifier wouldn't have approved otherwise
	}
	return &policy.PromotionInput{
		TaskID:                      req.Bundle.TaskID,
		TenantID:                    req.TenantID,
		DiffHash:                    req.Bundle.DiffHash,
		FilesChanged:                req.Bundle.FilesChanged,
		VerifierApprovalAttestation: req.Bundle.VerifierApprovalAttestation,
		BuildProvenanceAttestation:  req.Bundle.BuildProvenanceAttestation,
		RebuildHash:                 req.Bundle.RebuildHash,
		BlastRadius:                 br,
		SuggestedRollout:            req.Bundle.SuggestedRollout,
		TierResults:                 tr,
		AgentOidcSubject:            req.Bundle.AgentOidcSubject,
		Context:                     req.Context,
		CodeOwners:                  req.CodeOwners,
	}
}

func estimatedImpactFromScore(score float64) string {
	switch {
	case score >= 0.66:
		return "high"
	case score >= 0.33:
		return "medium"
	default:
		return "low"
	}
}

func asStr(v any) string {
	s, _ := v.(string)
	return s
}

func asBool(v any) bool {
	b, _ := v.(bool)
	return b
}

func contains(s, sub string) bool {
	if len(sub) == 0 {
		return true
	}
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func joinReasons(rs []string) string {
	out := ""
	for i, r := range rs {
		if i > 0 {
			out += "; "
		}
		out += r
	}
	return out
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeErr(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]any{"error": err.Error()})
}

func (s *Server) publish(ctx context.Context, eventType string, rec *Record) {
	if s.EventSink == nil {
		return
	}
	if err := s.EventSink.Publish(ctx, eventType, map[string]any{
		"event_type":    eventType,
		"promotion_id":  rec.ID,
		"task_id":       rec.Bundle.TaskID,
		"tenant_id":     rec.TenantID,
		"status":        rec.Status,
		"detail":        rec.Detail,
		"agent_oidc":    rec.Bundle.AgentOidcSubject,
		"approver_oidc": approverOIDCs(rec),
		"timestamp":     time.Now().UTC(),
	}); err != nil {
		s.Logger.Warn("publish event failed", "event", eventType, "err", err)
	}
}

func approverOIDCs(r *Record) []string {
	out := []string{}
	for _, a := range r.Approvals {
		out = append(out, a.ApproverOidcSubject)
	}
	return out
}
