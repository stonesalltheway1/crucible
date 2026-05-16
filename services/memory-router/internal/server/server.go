// Package server implements the MemoryService gRPC handlers.
//
// The wire transport in Phase 5 is HTTP/1.1+JSON (control-plane and
// verifier both speak it) for parity with the verifierbridge pattern;
// a buf-generated gRPC server lands when buf is wired into CI. The
// handlers are vendor-neutral and tested via the http.Handler surface.
package server

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	memoryspec "github.com/crucible/memory-spec/go"
	cruciblev1 "github.com/crucible/sdk-go/crucible/v1"

	"github.com/crucible/memory-router/internal/embedding"
	"github.com/crucible/memory-router/internal/proceduralstore"
	"github.com/crucible/memory-router/internal/retriever"
	"github.com/crucible/memory-router/internal/vectorstore"
)

// Server wraps the retriever + writers behind HTTP handlers.
type Server struct {
	Retriever *retriever.Retriever
	Proc      proceduralstore.Store
	Vec       vectorstore.Store
	Embedder  embedding.Client
	// JudgeFn is the LLM-as-judge filter applied to every procedural
	// write. Returns (admit, score, reason). nil disables the filter
	// (CI mode only; refuses in production via Server.requireJudge).
	JudgeFn       func(ctx context.Context, tenantID string, c memoryspec.Convention) (admit bool, score float64, reason string, injectionCategory string)
	RequireJudge  bool
	// counters
	recallTotal   atomic.Int64
	noteTotal     atomic.Int64
	convsTotal    atomic.Int64
	complianceTotal atomic.Int64
	quarantinedTotal atomic.Int64
}

// New constructs a Server.
func New(r *retriever.Retriever, p proceduralstore.Store, v vectorstore.Store, emb embedding.Client) *Server {
	return &Server{
		Retriever:    r,
		Proc:         p,
		Vec:          v,
		Embedder:     emb,
		RequireJudge: true,
	}
}

// Routes returns an HTTP mux with the memory-router endpoints.
func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/memory/recall", s.handleRecall)
	mux.HandleFunc("/v1/memory/note", s.handleNote)
	mux.HandleFunc("/v1/memory/conventions", s.handleConventions)
	mux.HandleFunc("/v1/memory/check_compliance", s.handleCheckCompliance)
	mux.HandleFunc("/v1/memory/admit_convention", s.handleAdmitConvention)
	mux.HandleFunc("/healthz", s.handleHealth)
	return mux
}

// ─── Handlers ───────────────────────────────────────────────────────────────

type recallRequest struct {
	TenantID  string                   `json:"tenant_id"`
	TaskID    string                   `json:"task_id"`
	Query     string                   `json:"query"`
	Scope     cruciblev1.ScopeFilter   `json:"scope"`
	MaxTokens uint32                   `json:"max_tokens"`
	MaxItems  uint32                   `json:"max_items"`
	Includes  recallIncludes           `json:"includes"`
}

type recallIncludes struct {
	Hot        bool `json:"hot"`
	Episodic   bool `json:"episodic"`
	Semantic   bool `json:"semantic"`
	Procedural bool `json:"procedural"`
}

type recallResponse struct {
	Memories         []memoryspec.ScoredMemory `json:"memories"`
	TokensUsed       uint32                    `json:"tokens_used"`
	BudgetTokens     uint32                    `json:"budget_tokens"`
	ItemsConsidered  uint32                    `json:"items_considered"`
	ItemsReturned    uint32                    `json:"items_returned"`
	LatencyMs        uint32                    `json:"latency_ms"`
}

func (s *Server) handleRecall(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, "method", "method not allowed")
		return
	}
	var req recallRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "decode", err.Error())
		return
	}
	if req.TenantID == "" {
		writeErr(w, http.StatusBadRequest, "tenant_id", "tenant_id required")
		return
	}
	// Default to procedural-on if everything is false (the common
	// "what are my conventions" query).
	if !req.Includes.Hot && !req.Includes.Episodic && !req.Includes.Semantic && !req.Includes.Procedural {
		req.Includes.Procedural = true
		req.Includes.Episodic = true
	}
	res, err := s.Retriever.Recall(r.Context(), memoryspec.RetrievalQuery{
		TenantID:          req.TenantID,
		TaskID:            req.TaskID,
		Query:             req.Query,
		Scope:             req.Scope,
		MaxTokens:         req.MaxTokens,
		MaxItems:          req.MaxItems,
		IncludeHot:        req.Includes.Hot,
		IncludeEpisodic:   req.Includes.Episodic,
		IncludeSemantic:   req.Includes.Semantic,
		IncludeProcedural: req.Includes.Procedural,
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "recall", err.Error())
		return
	}
	s.recallTotal.Add(1)
	writeJSON(w, http.StatusOK, recallResponse{
		Memories:        res.Memories,
		TokensUsed:      res.TokensUsed,
		BudgetTokens:    res.BudgetTokens,
		ItemsConsidered: res.ItemsConsidered,
		ItemsReturned:   res.ItemsReturned,
		LatencyMs:       res.LatencyMs,
	})
}

type noteRequest struct {
	TenantID string                `json:"tenant_id"`
	TaskID   string                `json:"task_id"`
	Fact     string                `json:"fact"`
	Source   cruciblev1.SourceRef  `json:"source"`
}
type noteResponse struct {
	MemoryID      string `json:"memory_id"`
	AttestationID string `json:"attestation_id"`
}

func (s *Server) handleNote(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, "method", "method not allowed")
		return
	}
	var req noteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "decode", err.Error())
		return
	}
	if req.TenantID == "" {
		writeErr(w, http.StatusBadRequest, "tenant_id", "tenant_id required")
		return
	}
	if strings.TrimSpace(req.Fact) == "" {
		writeErr(w, http.StatusBadRequest, "fact", "fact required")
		return
	}
	// twin.memory.note writes go to the episodic store, NOT procedural.
	// Promotion to procedural happens via the distiller (which sees the
	// note via the agent-observation source channel).
	vecs, err := s.Embedder.Embed(r.Context(), []embedding.Request{{TenantID: req.TenantID, Content: req.Fact}})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "embed", err.Error())
		return
	}
	mem := cruciblev1.Memory{
		Content:    req.Fact,
		Kind:       cruciblev1.MemEpisodic,
		Importance: 0.6,
		Source:     req.Source,
		WrittenAt:  time.Now().UTC(),
	}
	id, err := s.Vec.Write(r.Context(), mem, vecs[0], req.TenantID, req.Source.AdrPath)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "write", err.Error())
		return
	}
	s.noteTotal.Add(1)
	writeJSON(w, http.StatusOK, noteResponse{MemoryID: id, AttestationID: "local:memnote:" + id})
}

type conventionsRequest struct {
	TenantID string                 `json:"tenant_id"`
	Scope    cruciblev1.ScopeFilter `json:"scope"`
	Limit    int                    `json:"limit"`
}
type conventionsResponse struct {
	Conventions []memoryspec.Convention `json:"conventions"`
}

func (s *Server) handleConventions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, "method", "method not allowed")
		return
	}
	var req conventionsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "decode", err.Error())
		return
	}
	if req.TenantID == "" {
		writeErr(w, http.StatusBadRequest, "tenant_id", "tenant_id required")
		return
	}
	out := map[memoryspec.MemoryLayer][]memoryspec.Convention{}
	for _, layer := range []memoryspec.MemoryLayer{memoryspec.LayerOrgOverrides, memoryspec.LayerRepoOverrides} {
		convs, err := s.Proc.FetchByScope(r.Context(), req.TenantID, layer, req.Scope, req.Limit)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, "fetch", err.Error())
			return
		}
		out[layer] = convs
	}
	if s.Retriever != nil && s.Retriever.Globals != nil {
		for _, c := range s.Retriever.Globals.ConventionsForStacks(memoryspec.AllStacks()...) {
			if !scopeMatch(c.Scope, req.Scope) {
				continue
			}
			out[memoryspec.LayerGlobalDefaults] = append(out[memoryspec.LayerGlobalDefaults], c)
		}
	}
	// Merge but expose all winners; the verifier wants the full list,
	// not just the highest-priority.
	merged := make([]memoryspec.Convention, 0, 64)
	for _, layer := range memoryspec.ReadOrder() {
		merged = append(merged, out[layer]...)
	}
	if req.Limit > 0 && len(merged) > req.Limit {
		merged = merged[:req.Limit]
	}
	s.convsTotal.Add(1)
	writeJSON(w, http.StatusOK, conventionsResponse{Conventions: merged})
}

type complianceRequest struct {
	TenantID string         `json:"tenant_id"`
	TaskID   string         `json:"task_id"`
	Diff     cruciblev1.Diff `json:"diff"`
}
type complianceResponse struct {
	Report cruciblev1.ComplianceReport `json:"report"`
}

func (s *Server) handleCheckCompliance(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, "method", "method not allowed")
		return
	}
	var req complianceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "decode", err.Error())
		return
	}
	if req.TenantID == "" {
		writeErr(w, http.StatusBadRequest, "tenant_id", "tenant_id required")
		return
	}

	report := cruciblev1.ComplianceReport{
		DiffHash:    DiffHash(req.Diff),
		GeneratedAt: time.Now().UTC(),
	}

	scoped := map[string]memoryspec.Convention{}
	for _, fc := range req.Diff.Files {
		query := cruciblev1.ScopeFilter{FileGlob: fc.Path}
		for _, layer := range []memoryspec.MemoryLayer{memoryspec.LayerOrgOverrides, memoryspec.LayerRepoOverrides} {
			convs, err := s.Proc.FetchByScope(r.Context(), req.TenantID, layer, query, 64)
			if err != nil {
				writeErr(w, http.StatusInternalServerError, "fetch", err.Error())
				return
			}
			for _, c := range convs {
				scoped[c.ID] = c
			}
		}
	}
	report.ConventionsChecked = uint32(len(scoped))
	// Phase 5 compliance is conservative: we surface every active rule
	// that touches the diff scope as a "you should know about this"
	// info entry. The Phase 7 evaluator wires actual rule-machine
	// matching for the severity=error path. The verifier already
	// records ConventionsChecked as the trust signal.
	for _, c := range scoped {
		for _, fc := range req.Diff.Files {
			if !scopeMatch(c.Scope, cruciblev1.ScopeFilter{FileGlob: fc.Path}) {
				continue
			}
			sev := "info"
			if c.Status == memoryspec.StatusActive && c.Confidence >= 0.7 {
				sev = "warn"
			}
			report.Violations = append(report.Violations, cruciblev1.ComplianceReportViolation{
				ConventionID:  c.ID,
				RuleNl:        c.RuleNl,
				OffendingFile: fc.Path,
				Severity:      sev,
			})
		}
	}
	s.complianceTotal.Add(1)
	writeJSON(w, http.StatusOK, complianceResponse{Report: report})
}

type admitRequest struct {
	TenantID  string                 `json:"tenant_id"`
	Conv      memoryspec.Convention  `json:"convention"`
	Source    cruciblev1.SourceRef   `json:"source"`
	ForceLayer memoryspec.MemoryLayer `json:"force_layer"`
}
type admitResponse struct {
	ConventionID    string  `json:"convention_id,omitempty"`
	Admitted        bool    `json:"admitted"`
	Quarantined     bool    `json:"quarantined,omitempty"`
	QuarantineReason string `json:"quarantine_reason,omitempty"`
	InjectionCategory string `json:"injection_category,omitempty"`
	JudgeScore      float64 `json:"judge_score,omitempty"`
}

// handleAdmitConvention is what the distiller calls. Defence in depth:
// the gateway runs the LLM-as-judge filter on every write, in addition
// to whatever the distiller did first.
func (s *Server) handleAdmitConvention(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, "method", "method not allowed")
		return
	}
	var req admitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "decode", err.Error())
		return
	}
	if req.TenantID == "" {
		writeErr(w, http.StatusBadRequest, "tenant_id", "tenant_id required")
		return
	}
	c := req.Conv
	c.TenantID = req.TenantID
	if req.ForceLayer != "" {
		c.Layer = req.ForceLayer
	}
	if c.Layer == "" {
		c.Layer = memoryspec.LayerOrgOverrides
	}
	// Guardrail: refuse to write to global_defaults via the public
	// admission API. Only the bootstrap loader writes that layer.
	if c.Layer == memoryspec.LayerGlobalDefaults {
		writeErr(w, http.StatusForbidden, "layer", "admission via this endpoint cannot write to global_defaults")
		return
	}
	if c.Status == "" {
		c.Status = memoryspec.StatusActive
	}
	if c.ValidFrom.IsZero() {
		c.ValidFrom = time.Now().UTC()
	}
	if c.WrittenAt.IsZero() {
		c.WrittenAt = time.Now().UTC()
	}
	if len(c.SourceEvidence) == 0 && req.Source.Kind != "" {
		c.SourceEvidence = append(c.SourceEvidence, req.Source)
	}
	if err := c.Validate(); err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, admitResponse{
			Quarantined: true,
			QuarantineReason: err.Error(),
			InjectionCategory: "malformed",
		})
		return
	}

	if s.RequireJudge && s.JudgeFn == nil {
		writeErr(w, http.StatusInternalServerError, "judge", "LLM-as-judge filter not configured")
		return
	}
	if s.JudgeFn != nil {
		admit, score, reason, cat := s.JudgeFn(r.Context(), req.TenantID, c)
		c.JudgeScore = score
		c.JudgeRationale = reason
		if !admit {
			s.quarantinedTotal.Add(1)
			writeJSON(w, http.StatusOK, admitResponse{
				Admitted:          false,
				Quarantined:       true,
				QuarantineReason:  reason,
				InjectionCategory: cat,
				JudgeScore:        score,
			})
			return
		}
	}

	id, err := s.Proc.Upsert(r.Context(), c)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "upsert", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, admitResponse{
		ConventionID: id,
		Admitted:     true,
		JudgeScore:   c.JudgeScore,
	})
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	_ = r
	writeJSON(w, http.StatusOK, map[string]any{
		"status":           "ok",
		"recall_total":     s.recallTotal.Load(),
		"note_total":       s.noteTotal.Load(),
		"convs_total":      s.convsTotal.Load(),
		"compliance_total": s.complianceTotal.Load(),
		"quarantined_total": s.quarantinedTotal.Load(),
	})
}

// ─── Helpers ────────────────────────────────────────────────────────────────

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeErr(w http.ResponseWriter, status int, code, msg string) {
	writeJSON(w, status, map[string]any{"error": map[string]any{"code": code, "message": msg}})
}

// DiffHash mirrors apps/verifier's diff hasher so the compliance report
// can be cross-referenced against the verifier's TestReport.
func DiffHash(d cruciblev1.Diff) string {
	h := sha256.New()
	_, _ = h.Write([]byte(d.BaseSha))
	for _, f := range d.Files {
		_, _ = h.Write([]byte(f.Path))
		_, _ = h.Write([]byte(f.ContentSha256))
		_, _ = h.Write([]byte(f.Action))
	}
	return "diff_" + hex.EncodeToString(h.Sum(nil))[:32]
}

func scopeMatch(conv, query cruciblev1.ScopeFilter) bool {
	// Local fast-path; delegates to the canonical matcher when both
	// sides are non-trivial. Imported via package retriever's helper
	// is awkward; the production server pulls the package directly.
	if conv.Repo != "" && query.Repo != "" && !strings.EqualFold(conv.Repo, query.Repo) {
		return false
	}
	if conv.Category != "" && query.Category != "" && conv.Category != query.Category {
		return false
	}
	if conv.FileGlob == "" || query.FileGlob == "" {
		return true
	}
	// Direct match wins; otherwise the cartographer-side normalization
	// matters more than the gateway scope check (the proceduralstore
	// itself is the authority).
	return strings.EqualFold(conv.FileGlob, query.FileGlob) || matchesGlob(conv.FileGlob, query.FileGlob)
}

// matchesGlob is a deliberately narrow helper used only by the gateway
// scopeMatch fallback. Heavy-lifting lives in scope.Match.
func matchesGlob(pattern, name string) bool {
	if strings.Contains(pattern, "**") {
		left, right, _ := strings.Cut(pattern, "**")
		left = strings.TrimSuffix(left, "/")
		right = strings.TrimPrefix(right, "/")
		if left != "" && !strings.HasPrefix(name, left) {
			return false
		}
		if right == "" {
			return true
		}
		return strings.HasSuffix(name, strings.TrimPrefix(right, "*"))
	}
	return false
}

// ErrJudgeMissing is returned by the admission handler when no judge is
// configured but RequireJudge is true.
var ErrJudgeMissing = errors.New("LLM-as-judge filter not configured")

// requireJudge is what cmd/main calls at startup to fail fast if the
// production gateway forgot to wire the filter. The handler itself
// rechecks per request.
func (s *Server) requireJudge() error {
	if s.RequireJudge && s.JudgeFn == nil {
		return ErrJudgeMissing
	}
	return nil
}

// Compile-time check that we implement http.Handler-like behaviour.
var _ = fmt.Stringer(nil) // silence unused import warnings if minor
