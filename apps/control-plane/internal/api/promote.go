// Phase 6: the promotion-side of the API surface.
//
// POST /v1/tasks/{id}/promote — submits the verified bundle to the promotion
// gate. The agent calls this AFTER /verify has flipped the task into
// `promoting`. The handler:
//
//  1. Loads the task; refuses if not in `promoting`.
//  2. Builds the PromotionBundle from the request body + verifier approval
//     attestation ID stored on the task.
//  3. Submits to the gate via the bridge.
//  4. On approval, control plane keeps the task in `promoting` until the
//     gate emits a landed/rolled_back webhook (handled by events publisher).
//  5. On policy denial, transitions task to `failed` with a structured
//     reason exposed to the agent.
//
// Stub behaviour: when `CRUCIBLE_PROMOTION_GATE_ADDR` is unset, the
// endpoint returns 503 and the task remains `promoting` (consistent with
// the pre-Phase-6 contract where promotion was logged-and-succeed).

package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/crucible/control-plane/internal/promotionbridge"
	cruciblev1 "github.com/crucible/sdk-go/crucible/v1"
)

type promoteRequest struct {
	DiffHash                     string                                 `json:"diff_hash"`
	FilesChanged                 []cruciblev1.FileChange                `json:"files_changed"`
	VerifierApprovalAttestation  string                                 `json:"verifier_approval_attestation"`
	BuildProvenanceAttestation   string                                 `json:"build_provenance_attestation,omitempty"`
	RebuildHash                  string                                 `json:"rebuild_hash,omitempty"`
	BlastRadius                  cruciblev1.BlastRadius                 `json:"blast_radius"`
	SuggestedRollout             cruciblev1.SuggestedRollout            `json:"suggested_rollout"`
}

func (s *Server) handlePromoteTask(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if s.PromotionBridge == nil {
		writeErr(w, http.StatusServiceUnavailable, errors.New("promotion gate not wired (set CRUCIBLE_PROMOTION_GATE_ADDR)"))
		return
	}
	var req promoteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, errEmptyBody) {
		writeErr(w, http.StatusBadRequest, fmt.Errorf("invalid JSON: %w", err))
		return
	}
	task, err := s.Store.Update(id, func(t *cruciblev1.Task) error {
		if t.Status != cruciblev1.TaskStatusPromoting {
			return fmt.Errorf("task in state %s; expected promoting (run /verify first)", t.Status)
		}
		return nil
	})
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	bundle := cruciblev1.PromotionBundle{
		TaskID:                      id,
		DiffHash:                    req.DiffHash,
		VerifierApprovalAttestation: req.VerifierApprovalAttestation,
		FilesChanged:                req.FilesChanged,
		BuildProvenanceAttestation:  req.BuildProvenanceAttestation,
		RebuildHash:                 req.RebuildHash,
		BlastRadius:                 req.BlastRadius,
		SuggestedRollout:            req.SuggestedRollout,
		AgentOidcSubject:            s.Attestation.Signer().OidcSubject(),
		SignedAt:                    time.Now().UTC(),
	}
	resp, err := s.PromotionBridge.Submit(r.Context(), promotionbridge.SubmitRequest{
		Bundle:           bundle,
		TenantID:         task.TenantID,
		AgentOidcSubject: bundle.AgentOidcSubject,
	})
	if err != nil {
		if denied, ok := err.(*promotionbridge.PolicyDeniedError); ok {
			// Phase-6 contract: policy denial flips task to failed with
			// a structured reason.
			_, _ = s.Store.Update(id, func(t *cruciblev1.Task) error {
				t.Status = cruciblev1.TaskStatusFailed
				return nil
			})
			writeJSON(w, http.StatusForbidden, map[string]any{
				"task_status": "failed",
				"reason":      denied.Body,
				"error_code":  cruciblev1.ErrPromotionPolicyDenied,
			})
			return
		}
		writeErr(w, http.StatusBadGateway, fmt.Errorf("submit to gate: %w", err))
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]any{
		"task_status":     task.Status,
		"promotion_id":    resp.ID,
		"promotion_status": resp.Status,
		"detail":          resp.Detail,
	})
}
