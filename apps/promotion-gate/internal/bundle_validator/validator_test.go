package bundle_validator

import (
	"context"
	"strings"
	"testing"
	"time"

	cruciblev1 "github.com/crucible/sdk-go/crucible/v1"
)

func sampleBundle() *cruciblev1.PromotionBundle {
	files := []cruciblev1.FileChange{
		{Path: "api/webhooks.ts", Action: cruciblev1.ActionModify, ContentSha256: "abc"},
	}
	return &cruciblev1.PromotionBundle{
		TaskID:                      "task_demo",
		DiffHash:                    DeriveDiffHash(files),
		VerifierApprovalAttestation: "rekor:ver",
		FilesChanged:                files,
		AgentOidcSubject:            "https://accounts.crucible.dev/agents/x",
		SignedAt:                    time.Now(),
		BlastRadius:                 cruciblev1.BlastRadius{Reversibility: cruciblev1.ReversibilityTrivial, ImpactScore: 0.1},
	}
}

func seededVerifier(diffHash string) *FakeVerifier {
	f := NewFakeVerifier()
	f.Put(&FetchedStatement{
		UUID:          "rekor:ver",
		PredicateType: cruciblev1.PredicateVerifierApproval,
		Predicate: map[string]any{
			"task_id":   "task_demo",
			"diff_hash": diffHash,
			"verdict":   "approved",
		},
	})
	return f
}

func TestValidate_HappyPath(t *testing.T) {
	b := sampleBundle()
	v := New(seededVerifier(b.DiffHash))
	r, err := v.Validate(context.Background(), b)
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if !r.DiffHashOK || !r.VerifierApprovalReferenced || r.AttestationsResolved == 0 {
		t.Fatalf("unexpected result: %+v", r)
	}
}

func TestValidate_RejectsMissingFields(t *testing.T) {
	cases := []func(*cruciblev1.PromotionBundle){
		func(b *cruciblev1.PromotionBundle) { b.TaskID = "" },
		func(b *cruciblev1.PromotionBundle) { b.DiffHash = "" },
		func(b *cruciblev1.PromotionBundle) { b.AgentOidcSubject = "" },
		func(b *cruciblev1.PromotionBundle) { b.VerifierApprovalAttestation = "" },
	}
	for _, m := range cases {
		b := sampleBundle()
		m(b)
		v := New(seededVerifier(b.DiffHash))
		if _, err := v.Validate(context.Background(), b); err == nil {
			t.Fatalf("expected error for mutated bundle: %+v", b)
		}
	}
}

func TestValidate_RejectsDiffHashTamper(t *testing.T) {
	b := sampleBundle()
	b.DiffHash = "0xtampered"
	v := New(seededVerifier(b.DiffHash))
	if _, err := v.Validate(context.Background(), b); err == nil {
		t.Fatal("expected diff_hash mismatch error")
	}
}

func TestValidate_RejectsVerifierApprovalDiffMismatch(t *testing.T) {
	b := sampleBundle()
	f := seededVerifier("0xdifferent")
	v := New(f)
	_, err := v.Validate(context.Background(), b)
	if err == nil || !strings.Contains(err.Error(), "diff_hash") {
		t.Fatalf("expected VerifierApproval.diff_hash mismatch error, got %v", err)
	}
}

func TestValidate_RejectsStaleBundle(t *testing.T) {
	b := sampleBundle()
	b.SignedAt = time.Now().Add(-48 * time.Hour)
	v := New(seededVerifier(b.DiffHash))
	if _, err := v.Validate(context.Background(), b); err == nil || !strings.Contains(err.Error(), "freshness") {
		t.Fatalf("expected freshness error, got %v", err)
	}
}

func TestValidate_RejectsWrongPredicateType(t *testing.T) {
	b := sampleBundle()
	f := NewFakeVerifier()
	f.Put(&FetchedStatement{
		UUID:          "rekor:ver",
		PredicateType: cruciblev1.PredicateWriteAttestation, // wrong type
		Predicate:     map[string]any{"diff_hash": b.DiffHash},
	})
	v := New(f)
	if _, err := v.Validate(context.Background(), b); err == nil {
		t.Fatal("expected wrong-predicate-type error")
	}
}

func TestValidate_RejectsMissingVerifierApprovalUUID(t *testing.T) {
	b := sampleBundle()
	f := NewFakeVerifier() // empty
	v := New(f)
	if _, err := v.Validate(context.Background(), b); err == nil {
		t.Fatal("expected resolve error")
	}
}

func TestDeriveDiffHash_Stable(t *testing.T) {
	files := []cruciblev1.FileChange{
		{Path: "a.go", Action: "modify", ContentSha256: "1"},
		{Path: "b.go", Action: "add", ContentSha256: "2"},
	}
	h1 := DeriveDiffHash(files)
	// Re-order — sorting must give the same hash.
	files = []cruciblev1.FileChange{
		{Path: "b.go", Action: "add", ContentSha256: "2"},
		{Path: "a.go", Action: "modify", ContentSha256: "1"},
	}
	h2 := DeriveDiffHash(files)
	if h1 != h2 {
		t.Fatal("DeriveDiffHash must be order-independent")
	}
}

func TestValidate_HandlesBuildProvenance(t *testing.T) {
	b := sampleBundle()
	b.BuildProvenanceAttestation = "rekor:slsa"
	f := seededVerifier(b.DiffHash)
	f.Put(&FetchedStatement{
		UUID:          "rekor:slsa",
		PredicateType: "https://slsa.dev/provenance/v1",
		Predicate:     map[string]any{"buildDefinition": map[string]any{}},
	})
	v := New(f)
	r, err := v.Validate(context.Background(), b)
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if !r.BuildProvenanceReferenced {
		t.Fatal("expected BuildProvenanceReferenced")
	}
}
