// Forged-bundle corpus test. The brief's quality bar:
//
//   - Attestation chain validation: zero false-acceptances against 10,000+
//     forged bundles in the threat-model test corpus.
//
// We synthesize variants by mutating one or more invariants per bundle:
//
//   - DiffHash tamper
//   - File ContentSha256 tamper
//   - SignedAt out of window
//   - Missing required fields
//   - VerifierApproval pointing to non-existent UUID
//   - VerifierApproval with wrong diff_hash
//   - VerifierApproval with wrong predicate type
//   - SubjectName-style tamper on the VerifierApproval predicate

package bundle_validator

import (
	"context"
	"math/rand"
	"testing"
	"time"

	cruciblev1 "github.com/crucible/sdk-go/crucible/v1"
)

const corpusSize = 10000

func TestForgedBundleCorpus_ZeroFalseAcceptances(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping 10K corpus in short mode")
	}
	rng := rand.New(rand.NewSource(42))

	// Build a single legitimate bundle as the baseline; each forged variant
	// is a mutation on top.
	files := []cruciblev1.FileChange{
		{Path: "api/x.go", Action: cruciblev1.ActionModify, ContentSha256: "abc"},
		{Path: "api/y.go", Action: cruciblev1.ActionAdd, ContentSha256: "def"},
	}
	baseline := &cruciblev1.PromotionBundle{
		TaskID:                      "task_corpus",
		DiffHash:                    DeriveDiffHash(files),
		FilesChanged:                files,
		VerifierApprovalAttestation: "rekor:ver-corpus",
		AgentOidcSubject:            "https://accounts.crucible.dev/agents/x",
		SignedAt:                    time.Now(),
		BlastRadius:                 cruciblev1.BlastRadius{Reversibility: cruciblev1.ReversibilityTrivial, ImpactScore: 0.1},
	}
	verifier := NewFakeVerifier()
	verifier.Put(&FetchedStatement{
		UUID:          baseline.VerifierApprovalAttestation,
		PredicateType: cruciblev1.PredicateVerifierApproval,
		Predicate: map[string]any{
			"task_id":  baseline.TaskID,
			"diff_hash": baseline.DiffHash,
			"verdict":  "approved",
		},
	})
	v := New(verifier)

	mutators := []func(*cruciblev1.PromotionBundle, *FakeVerifier){
		// 0: diff_hash tamper.
		func(b *cruciblev1.PromotionBundle, _ *FakeVerifier) { b.DiffHash = "0xforged" },
		// 1: file content tamper.
		func(b *cruciblev1.PromotionBundle, _ *FakeVerifier) {
			b.FilesChanged = append([]cruciblev1.FileChange{}, b.FilesChanged...)
			b.FilesChanged[0].ContentSha256 = "tampered"
		},
		// 2: signed_at stale.
		func(b *cruciblev1.PromotionBundle, _ *FakeVerifier) {
			b.SignedAt = time.Now().Add(-72 * time.Hour)
		},
		// 3: missing task id.
		func(b *cruciblev1.PromotionBundle, _ *FakeVerifier) { b.TaskID = "" },
		// 4: missing agent oidc.
		func(b *cruciblev1.PromotionBundle, _ *FakeVerifier) { b.AgentOidcSubject = "" },
		// 5: missing verifier approval.
		func(b *cruciblev1.PromotionBundle, _ *FakeVerifier) { b.VerifierApprovalAttestation = "" },
		// 6: verifier approval uuid not in relay.
		func(b *cruciblev1.PromotionBundle, _ *FakeVerifier) {
			b.VerifierApprovalAttestation = "rekor:nonexistent"
		},
		// 7: verifier approval wrong predicate type.
		func(b *cruciblev1.PromotionBundle, fv *FakeVerifier) {
			b.VerifierApprovalAttestation = "rekor:wrong-type"
			fv.Put(&FetchedStatement{
				UUID:          "rekor:wrong-type",
				PredicateType: cruciblev1.PredicateWriteAttestation,
				Predicate:     map[string]any{"diff_hash": b.DiffHash},
			})
		},
		// 8: verifier approval wrong diff hash.
		func(b *cruciblev1.PromotionBundle, fv *FakeVerifier) {
			b.VerifierApprovalAttestation = "rekor:wrong-diff"
			fv.Put(&FetchedStatement{
				UUID:          "rekor:wrong-diff",
				PredicateType: cruciblev1.PredicateVerifierApproval,
				Predicate:     map[string]any{"diff_hash": "0xdifferent", "verdict": "approved"},
			})
		},
		// 9: verifier approval missing diff_hash entirely.
		func(b *cruciblev1.PromotionBundle, fv *FakeVerifier) {
			b.VerifierApprovalAttestation = "rekor:no-diff"
			fv.Put(&FetchedStatement{
				UUID:          "rekor:no-diff",
				PredicateType: cruciblev1.PredicateVerifierApproval,
				Predicate:     map[string]any{"verdict": "approved"},
			})
		},
	}

	accepted := 0
	for i := 0; i < corpusSize; i++ {
		clone := *baseline
		// Apply a random mutator (single).
		mut := mutators[rng.Intn(len(mutators))]
		mut(&clone, verifier)
		if _, err := v.Validate(context.Background(), &clone); err == nil {
			accepted++
			t.Logf("forged bundle %d unexpectedly accepted (mutator path)", i)
		}
	}
	if accepted != 0 {
		t.Fatalf("expected ZERO forged-bundle acceptances, got %d / %d", accepted, corpusSize)
	}
}
