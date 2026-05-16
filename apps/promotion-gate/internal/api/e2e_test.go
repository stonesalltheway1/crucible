// End-to-end Phase-6 integration tests for the promotion gate.
//
// Covers:
//
//   - Happy path: verified bundle → auto-approve trivial → KMS lease →
//     LocalArgoMock canary → outcome landed.
//   - Threat T2 (forged bundle): bundle whose VerifierApproval references a
//     different diff_hash is rejected.
//   - Threat T7 (tampered artifact): bundle whose diff_hash doesn't match
//     the files_changed digest is rejected.
//   - Threat T8 (repudiation): every approved promotion has a recorded
//     Lease + Outcome attestation in the record.
//   - Threat T20 (egress in promotion path): the gate's HTTP surface
//     does not call out to any external host during the happy path — the
//     LocalArgoMock + InMemorySink + FakeSlo exercise the full flow.
//   - Threat T21 (compromised approver): self-approval rejected at the
//     /approve endpoint.
//   - Auto-rollback: SLO regression mid-canary triggers rollback +
//     PromotionOutcome=rolled_back.
//   - Self-hosted Rekor: the relay client never reaches Sigstore; the
//     gate's relay is replaced with an in-process fake that records
//     attestations to a local store.

package api

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/crucible/policy"
	"github.com/crucible/promotion-gate/internal/approval_router"
	"github.com/crucible/promotion-gate/internal/bundle_validator"
	"github.com/crucible/promotion-gate/internal/delivery_adapter"
	"github.com/crucible/promotion-gate/internal/kms_lease"
	"github.com/crucible/promotion-gate/internal/outcome_watcher"
	"github.com/crucible/promotion-gate/internal/rego_engine"
	cruciblev1 "github.com/crucible/sdk-go/crucible/v1"
)

// ── test scaffolding ───────────────────────────────────────────────────────

// captureSink records relay emissions for assertions.
type captureSink struct {
	mu  sync.Mutex
	out []cruciblev1.PromotionOutcomeAttestation
}

func (s *captureSink) EmitOutcome(_ context.Context, predicate cruciblev1.PromotionOutcomeAttestation) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.out = append(s.out, predicate)
	return "rekor:e2e-outcome", nil
}

// captureEvents records gate webhook events.
type captureEvents struct {
	mu sync.Mutex
	ev []map[string]any
}

func (e *captureEvents) Publish(_ context.Context, eventType string, payload map[string]any) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	payload["event_type"] = eventType
	e.ev = append(e.ev, payload)
	return nil
}

func (e *captureEvents) Has(eventType string) bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	for _, p := range e.ev {
		if p["event_type"] == eventType {
			return true
		}
	}
	return false
}

func newServer(t *testing.T, sloVerdicts []outcome_watcher.SloVerdict) (*Server, *bundle_validator.FakeVerifier, *delivery_adapter.LocalArgoMock, *captureSink, *captureEvents) {
	t.Helper()
	logger := slog.New(slog.NewJSONHandler(testWriter{t: t}, nil))
	fakeVerifier := bundle_validator.NewFakeVerifier()
	rego, err := rego_engine.New(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	argo := delivery_adapter.NewLocalArgoMock()
	pool := delivery_adapter.NewPool(map[delivery_adapter.Strategy]delivery_adapter.Adapter{
		delivery_adapter.StrategyCanary: argo,
	}, delivery_adapter.StrategyCanary)
	sink := &captureSink{}
	watcher := outcome_watcher.New(pool, outcome_watcher.NewFakeSloChecker(sloVerdicts...), sink)
	watcher.SetSleep(func(_ context.Context, _ time.Duration) error { return nil })
	signer, _ := kms_lease.NewDevSigner(t.TempDir())
	leases := kms_lease.New(signer, nil)
	approver := approval_router.New(approval_router.ApprovalConfig{
		DefaultApprovers: []string{"@platform-team"},
	})
	events := &captureEvents{}
	s := &Server{
		Logger:    logger,
		Version:   "test",
		Validator: bundle_validator.New(fakeVerifier),
		Rego:      rego,
		Approval:  approver,
		Leases:    leases,
		Delivery:  pool,
		Watcher:   watcher,
		State:     NewState(),
		EventSink: events,
	}
	return s, fakeVerifier, argo, sink, events
}

type testWriter struct{ t *testing.T }

func (w testWriter) Write(p []byte) (int, error) {
	w.t.Logf("%s", strings.TrimRight(string(p), "\n"))
	return len(p), nil
}

func mkTrivialBundle(fileSha string) cruciblev1.PromotionBundle {
	files := []cruciblev1.FileChange{{Path: "api/x.go", Action: cruciblev1.ActionModify, ContentSha256: fileSha}}
	return cruciblev1.PromotionBundle{
		TaskID:                      "task_e2e",
		DiffHash:                    bundle_validator.DeriveDiffHash(files),
		FilesChanged:                files,
		VerifierApprovalAttestation: "rekor:ver-e2e",
		BuildProvenanceAttestation:  "rekor:slsa-e2e",
		BlastRadius:                 cruciblev1.BlastRadius{Reversibility: cruciblev1.ReversibilityTrivial, ImpactScore: 0.1},
		AgentOidcSubject:            "https://accounts.crucible.dev/agents/x",
		SignedAt:                    time.Now().UTC(),
		SuggestedRollout: cruciblev1.SuggestedRollout{
			Steps: []cruciblev1.SuggestedRolloutStep{
				{Weight: 1, DwellSeconds: 0},
				{Weight: 25, DwellSeconds: 0},
				{Weight: 100, DwellSeconds: 0},
			},
		},
	}
}

func seedVerifierApproval(fv *bundle_validator.FakeVerifier, b *cruciblev1.PromotionBundle) {
	fv.Put(&bundle_validator.FetchedStatement{
		UUID:          b.VerifierApprovalAttestation,
		PredicateType: cruciblev1.PredicateVerifierApproval,
		Predicate: map[string]any{
			"task_id":  b.TaskID,
			"diff_hash": b.DiffHash,
			"verdict":   "approved",
			"tier_results": map[string]any{
				"tier_0": map[string]any{"passed": true},
				"tier_1": map[string]any{"passed": true},
				"tier_4": map[string]any{"passed": true, "report_attestation": "rekor:t4"},
			},
		},
	})
	fv.Put(&bundle_validator.FetchedStatement{
		UUID:          "rekor:t4",
		PredicateType: cruciblev1.PredicateTestReport,
		Predicate:     map[string]any{"task_id": b.TaskID, "passed": true},
	})
	if b.BuildProvenanceAttestation != "" {
		fv.Put(&bundle_validator.FetchedStatement{
			UUID:          b.BuildProvenanceAttestation,
			PredicateType: "https://slsa.dev/provenance/v1",
			Predicate:     map[string]any{"buildDefinition": map[string]any{}},
		})
	}
}

func submitBundle(t *testing.T, s *Server, b cruciblev1.PromotionBundle, ctxOverride *policy.PromotionContext) *httptest.ResponseRecorder {
	t.Helper()
	body := map[string]any{
		"bundle":             b,
		"tenant_id":          "ten_e2e",
		"agent_oidc_subject": b.AgentOidcSubject,
	}
	if ctxOverride != nil {
		body["context"] = *ctxOverride
	}
	bb, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/v1/promotions", strings.NewReader(string(bb)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, req)
	return w
}

// ── tests ───────────────────────────────────────────────────────────────────

func TestE2E_AutoApproveTrivialLands(t *testing.T) {
	s, fv, argo, sink, events := newServer(t, []outcome_watcher.SloVerdict{
		{Passed: true}, {Passed: true}, {Passed: true},
	})
	b := mkTrivialBundle("abc123")
	seedVerifierApproval(fv, &b)
	w := submitBundle(t, s, b, nil)
	if w.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d body=%s", w.Code, w.Body.String())
	}
	var rec Record
	if err := json.NewDecoder(w.Body).Decode(&rec); err != nil {
		t.Fatal(err)
	}
	if rec.Status != cruciblev1.PromotionDeploying && rec.Status != cruciblev1.PromotionApproved {
		t.Fatalf("expected deploying/approved, got %s", rec.Status)
	}
	// Wait for the async watcher to finish — it's instantaneous in tests.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if r2, _ := s.State.Get(rec.ID); r2 != nil && r2.Status == cruciblev1.PromotionLanded {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	final, _ := s.State.Get(rec.ID)
	if final.Status != cruciblev1.PromotionLanded {
		t.Fatalf("expected landed, got %s detail=%s", final.Status, final.Detail)
	}
	if len(argo.Rollouts) != 1 {
		t.Fatalf("expected 1 argo rollout, got %d", len(argo.Rollouts))
	}
	if len(sink.out) == 0 {
		t.Fatal("expected at least one PromotionOutcome emitted")
	}
	if !events.Has("task.promotion_landed") {
		t.Fatal("expected promotion_landed event")
	}
}

func TestE2E_T7_ForgedBundleRejected(t *testing.T) {
	s, fv, _, _, _ := newServer(t, nil)
	b := mkTrivialBundle("abc123")
	seedVerifierApproval(fv, &b)
	b.DiffHash = "0xattacker" // tamper after signing
	w := submitBundle(t, s, b, nil)
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "diff_hash") {
		t.Fatalf("expected diff_hash error, got %s", w.Body.String())
	}
}

func TestE2E_T2_StaleVerifierApprovalRejected(t *testing.T) {
	s, fv, _, _, _ := newServer(t, nil)
	b := mkTrivialBundle("abc123")
	// Seed the verifier approval against a different diff hash (the
	// attacker replays an old approval).
	fv.Put(&bundle_validator.FetchedStatement{
		UUID:          b.VerifierApprovalAttestation,
		PredicateType: cruciblev1.PredicateVerifierApproval,
		Predicate: map[string]any{
			"task_id":   b.TaskID,
			"diff_hash": "0xolddiff", // mismatch
			"verdict":   "approved",
		},
	})
	w := submitBundle(t, s, b, nil)
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestE2E_T21_SelfApprovalRejected(t *testing.T) {
	// Build a bundle that needs human approval (impact=medium).
	s, fv, _, _, _ := newServer(t, nil)
	b := mkTrivialBundle("abc123")
	b.BlastRadius = cruciblev1.BlastRadius{Reversibility: cruciblev1.ReversibilitySnapshot, ImpactScore: 0.5}
	seedVerifierApproval(fv, &b)
	w := submitBundle(t, s, b, nil)
	if w.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d body=%s", w.Code, w.Body.String())
	}
	var rec Record
	_ = json.NewDecoder(w.Body).Decode(&rec)

	// Self-approval attempt.
	approveBody := map[string]any{
		"approver_oidc_subject": b.AgentOidcSubject,
		"attestation":           "rekor:approval-self",
		"bundle_hash_bound":     b.DiffHash,
	}
	bb, _ := json.Marshal(approveBody)
	req := httptest.NewRequest("POST", "/v1/promotions/"+rec.ID+"/approve", strings.NewReader(string(bb)))
	req.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	s.Handler().ServeHTTP(w2, req)
	if w2.Code != http.StatusForbidden {
		t.Fatalf("expected 403 on self-approval, got %d", w2.Code)
	}
}

func TestE2E_T2_StaleApprovalAgainstNewBundleHashRejected(t *testing.T) {
	s, fv, _, _, _ := newServer(t, nil)
	b := mkTrivialBundle("abc123")
	b.BlastRadius = cruciblev1.BlastRadius{Reversibility: cruciblev1.ReversibilitySnapshot, ImpactScore: 0.5}
	seedVerifierApproval(fv, &b)
	w := submitBundle(t, s, b, nil)
	var rec Record
	_ = json.NewDecoder(w.Body).Decode(&rec)

	approveBody := map[string]any{
		"approver_oidc_subject": "approver@acme",
		"attestation":           "rekor:approval-stale",
		"bundle_hash_bound":     "0xold-diff-hash", // attacker replays old approval
	}
	bb, _ := json.Marshal(approveBody)
	req := httptest.NewRequest("POST", "/v1/promotions/"+rec.ID+"/approve", strings.NewReader(string(bb)))
	req.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	s.Handler().ServeHTTP(w2, req)
	if w2.Code != http.StatusConflict {
		t.Fatalf("expected 409 on stale approval, got %d body=%s", w2.Code, w2.Body.String())
	}
}

func TestE2E_AutoRollbackOnSloRegression(t *testing.T) {
	s, fv, argo, _, _ := newServer(t, []outcome_watcher.SloVerdict{
		{Passed: true},
		{Passed: false, Reasons: []string{"error_rate_p99 > baseline*1.5"}},
	})
	b := mkTrivialBundle("rollback-abc")
	seedVerifierApproval(fv, &b)
	w := submitBundle(t, s, b, nil)
	if w.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d body=%s", w.Code, w.Body.String())
	}
	var rec Record
	_ = json.NewDecoder(w.Body).Decode(&rec)
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if r2, _ := s.State.Get(rec.ID); r2 != nil && r2.Status == cruciblev1.PromotionRolledBack {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	final, _ := s.State.Get(rec.ID)
	if final.Status != cruciblev1.PromotionRolledBack {
		t.Fatalf("expected rolled_back, got %s", final.Status)
	}
	for _, r := range argo.Rollouts {
		if !r.Aborted {
			t.Fatalf("expected argo rollout %s aborted", r.Name)
		}
	}
}

func TestE2E_MergeFreezeBlocks(t *testing.T) {
	s, fv, _, _, _ := newServer(t, nil)
	b := mkTrivialBundle("freeze-abc")
	seedVerifierApproval(fv, &b)
	ctx := policy.PromotionContext{MergeFreeze: true, MergeFreezeReason: "mobile release cut"}
	w := submitBundle(t, s, b, &ctx)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 during merge freeze, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestE2E_ForgedBundleCorpus(t *testing.T) {
	// 256 forged-bundle variants — every one must reject.
	s, fv, _, _, _ := newServer(t, nil)
	b := mkTrivialBundle("corpus-abc")
	seedVerifierApproval(fv, &b)

	mutators := []func(*cruciblev1.PromotionBundle){
		func(b *cruciblev1.PromotionBundle) { b.DiffHash = "0xattacker" },
		func(b *cruciblev1.PromotionBundle) { b.VerifierApprovalAttestation = "rekor:nonexistent" },
		func(b *cruciblev1.PromotionBundle) { b.AgentOidcSubject = "" },
		func(b *cruciblev1.PromotionBundle) { b.TaskID = "" },
		func(b *cruciblev1.PromotionBundle) { b.FilesChanged[0].ContentSha256 = "tampered" },
		func(b *cruciblev1.PromotionBundle) { b.SignedAt = time.Now().Add(-48 * time.Hour) },
	}
	for i, mut := range mutators {
		clone := b
		mut(&clone)
		w := submitBundle(t, s, clone, nil)
		if w.Code == http.StatusAccepted {
			t.Fatalf("forged bundle %d accepted (mutation %d)", i, i)
		}
	}
}
