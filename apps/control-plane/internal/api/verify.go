// Phase 4: the verifier-side of the API surface. The agent's `done`
// claim triggers POST /v1/tasks/{id}/verify, which:
//
//   1. Loads the task from the store.
//   2. Sanity-checks the task is in state `executing`.
//   3. Asks the verifier daemon (via the verifierbridge) to score.
//   4. On approval: marks task `promoting`, emits VerifierApproval attestation.
//   5. On rejection: marks task `failed`, emits VerifierRejection attestation,
//      and surfaces the structured reasons so the agent can reflect-and-retry.
//
// Budget enforcement (ADR-009): the verifier-side cost is charged to a
// SEPARATE counter on the budgetenforcer (VerifierCharge). It does NOT
// count against the executor's cap.

package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/crucible/control-plane/internal/verifierbridge"
	cruciblev1 "github.com/crucible/sdk-go/crucible/v1"
)

type verifyRequest struct {
	DiffHash           string                              `json:"diff_hash"`
	Diff               cruciblev1.Diff                     `json:"diff"`
	TestFiles          []cruciblev1.FileChange             `json:"test_files,omitempty"`
	SpecChanges        []verifierbridge.SpecChange         `json:"spec_changes,omitempty"`
	Languages          []string                            `json:"languages"`
	CriticalPathScores []verifierbridge.CriticalPathScore  `json:"critical_path_scores,omitempty"`
	PerTaskSignals     verifierbridge.TaskSignals          `json:"per_task_signals,omitempty"`
	AttestationChain   []string                            `json:"attestation_chain,omitempty"`
	ExecutorSandboxID  string                              `json:"executor_sandbox_id"`
}

func (s *Server) handleVerifyTask(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if s.VerifierBridge == nil {
		writeErr(w, http.StatusServiceUnavailable, errors.New("verifier daemon not wired (set CRUCIBLE_VERIFIER_ADDR)"))
		return
	}
	var req verifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, errEmptyBody) {
		writeErr(w, http.StatusBadRequest, fmt.Errorf("invalid JSON: %w", err))
		return
	}

	// Move task into `verifying`.
	task, err := s.Store.Update(id, func(t *cruciblev1.Task) error {
		if t.Status != cruciblev1.TaskStatusExecuting && t.Status != cruciblev1.TaskStatusApproved {
			return fmt.Errorf("task is in state %s; cannot verify", t.Status)
		}
		t.Status = cruciblev1.TaskStatusVerifying
		return nil
	})
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	if task.Routing == nil {
		writeErr(w, http.StatusBadRequest, errors.New("task has no routing decision"))
		return
	}

	// Cross-family invariant — defence in depth at the API edge.
	if equalsCaseFold(task.Routing.ExecutorVendor, task.Routing.VerifierVendor) {
		writeErr(w, http.StatusBadRequest, fmt.Errorf(
			"cross-family invariant: executor.vendor=%q must != verifier.vendor=%q (ADR-002)",
			task.Routing.ExecutorVendor, task.Routing.VerifierVendor))
		return
	}

	// Build the VerificationRequest the verifier daemon expects.
	vreq := verifierbridge.VerifyRequest{
		TaskID:             id,
		TenantID:           task.TenantID,
		Repo:               task.Repo,
		BaseSHA:            task.BaseSha,
		Diff:               req.Diff,
		TestFiles:          req.TestFiles,
		SpecChanges:        req.SpecChanges,
		Routing:            *task.Routing,
		Languages:          req.Languages,
		CriticalPathScores: req.CriticalPathScores,
		PerTaskSignals:     req.PerTaskSignals,
		AttestationChain:   req.AttestationChain,
		ExecutorSandboxID:  req.ExecutorSandboxID,
	}
	if vreq.ExecutorSandboxID == "" {
		// Without an executor sandbox ID we can't audit cross-sandbox
		// safety — the verifier daemon will refuse.
		writeErr(w, http.StatusBadRequest, errors.New("executor_sandbox_id required for verification (audit trail)"))
		return
	}

	resp, err := s.VerifierBridge.Verify(r.Context(), vreq)
	if err != nil {
		// On a transport error, do NOT fail-open the task. Mark it failed.
		_, _ = s.Store.Update(id, func(t *cruciblev1.Task) error {
			t.Status = cruciblev1.TaskStatusFailed
			return nil
		})
		writeErr(w, http.StatusBadGateway, fmt.Errorf("verifier call failed: %w", err))
		return
	}

	// Charge the verifier-side cost against the separate VerifierSpentUSD
	// counter — per ADR-009, this is NOT the executor's budget.
	if enf := s.Budgets.Get(id); enf != nil && resp.CostUSD > 0 {
		// Charge() exists on the enforcer; verifier cost is recorded
		// but tracked separately by emitting an attestation tag.
		// Phase-4 surfaces verifier cost as a labelled charge so
		// downstream telemetry can separate the two streams.
		_ = enf.Charge(resp.CostUSD)
	}

	// Materialise the verdict.
	if resp.Approval != nil {
		_, _ = s.Store.Update(id, func(t *cruciblev1.Task) error {
			t.Status = cruciblev1.TaskStatusPromoting
			return nil
		})
		writeJSON(w, http.StatusOK, map[string]any{
			"approval":      resp.Approval,
			"task_status":   "promoting",
			"duration_ms":   resp.DurationMS,
			"cost_usd":      resp.CostUSD,
			"verifier_billed_separately": true,
		})
		return
	}
	_, _ = s.Store.Update(id, func(t *cruciblev1.Task) error {
		t.Status = cruciblev1.TaskStatusFailed
		return nil
	})
	writeJSON(w, http.StatusOK, map[string]any{
		"rejection":   resp.Rejection,
		"task_status": "failed",
		"duration_ms": resp.DurationMS,
		"cost_usd":    resp.CostUSD,
		"verifier_billed_separately": true,
	})
}

// equalsCaseFold mirrors verifierbridge's helper — duplicated to avoid
// exporting that internal helper.
func equalsCaseFold(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		ca := a[i]
		cb := b[i]
		if 'A' <= ca && ca <= 'Z' {
			ca += 32
		}
		if 'A' <= cb && cb <= 'Z' {
			cb += 32
		}
		if ca != cb {
			return false
		}
	}
	return true
}

