package attestation

import (
	"bufio"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	cruciblev1 "github.com/crucible/sdk-go/crucible/v1"
)

// ── BuildStatement ────────────────────────────────────────────────────────

func TestBuildStatement_OK(t *testing.T) {
	digest := SubjectDigest([]byte("hello world"))
	predicate := map[string]any{"path": "/a/b", "action": "modify"}
	stmt, err := BuildStatement(cruciblev1.PredicateWriteAttestation, "task/1/a", digest, predicate)
	if err != nil {
		t.Fatalf("BuildStatement: %v", err)
	}
	if stmt.Type != cruciblev1.PredicateInTotoStatementType {
		t.Fatalf("wrong type: %s", stmt.Type)
	}
	if stmt.PredicateType != cruciblev1.PredicateWriteAttestation {
		t.Fatalf("wrong predicate type: %s", stmt.PredicateType)
	}
	if len(stmt.Subject) != 1 || stmt.Subject[0].Digest["sha256"] == "" {
		t.Fatalf("subject malformed: %+v", stmt.Subject)
	}
	var parsed map[string]any
	if err := json.Unmarshal(stmt.Predicate, &parsed); err != nil {
		t.Fatalf("predicate not valid JSON: %v", err)
	}
	if parsed["path"] != "/a/b" {
		t.Fatalf("predicate round-trip failed: %+v", parsed)
	}
}

func TestBuildStatement_RejectsEmptyType(t *testing.T) {
	if _, err := BuildStatement("", "x", [32]byte{}, "{}"); err == nil {
		t.Fatal("expected error on empty predicate type")
	}
	if _, err := BuildStatement(cruciblev1.PredicateWriteAttestation, "", [32]byte{}, "{}"); err == nil {
		t.Fatal("expected error on empty subject name")
	}
}

// ── LocalEd25519Signer + Verify ───────────────────────────────────────────

func mkSigner(t *testing.T) *LocalEd25519Signer {
	t.Helper()
	dir := t.TempDir()
	s, err := NewLocalEd25519Signer(dir)
	if err != nil {
		t.Fatalf("signer: %v", err)
	}
	return s
}

func TestLocalSigner_KeyPersists(t *testing.T) {
	dir := t.TempDir()
	s1, err := NewLocalEd25519Signer(dir)
	if err != nil {
		t.Fatal(err)
	}
	s2, err := NewLocalEd25519Signer(dir)
	if err != nil {
		t.Fatal(err)
	}
	if s1.KeyID() != s2.KeyID() {
		t.Fatalf("key id changed on reload: %s vs %s", s1.KeyID(), s2.KeyID())
	}
	if !strings.Contains(s1.OidcSubject(), s1.KeyID()) {
		t.Fatalf("subject should embed key id: %s", s1.OidcSubject())
	}
}

func TestSignAndVerify_Roundtrip(t *testing.T) {
	s := mkSigner(t)
	stmt, err := BuildStatement(
		cruciblev1.PredicatePlanProposal, "task/1/plan",
		SubjectDigest([]byte("planhash")),
		map[string]any{"task_id": "task_1", "plan_hash": "abc"},
	)
	if err != nil {
		t.Fatal(err)
	}
	env, err := s.SignStatement(stmt)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	if env.PayloadType != cruciblev1.PredicateDsseEnvelopePayloadType {
		t.Fatalf("wrong payload type: %s", env.PayloadType)
	}
	if len(env.Signatures) != 1 || env.Signatures[0].Sig == "" {
		t.Fatalf("envelope missing signature")
	}
	if err := Verify(env, s.PublicKey()); err != nil {
		t.Fatalf("verify: %v", err)
	}
}

func TestVerify_RejectsTamperedPayload(t *testing.T) {
	s := mkSigner(t)
	stmt, _ := BuildStatement(
		cruciblev1.PredicatePlanProposal, "task/1/plan",
		SubjectDigest([]byte("h")), map[string]any{"task_id": "task_1"},
	)
	env, _ := s.SignStatement(stmt)
	// Mutate the payload after signing.
	env.Payload = env.Payload + "XX"
	if err := Verify(env, s.PublicKey()); err == nil {
		t.Fatal("expected verify failure on tampered payload")
	}
}

func TestVerify_RejectsEmptySignatures(t *testing.T) {
	s := mkSigner(t)
	stmt, _ := BuildStatement(
		cruciblev1.PredicatePlanProposal, "x",
		SubjectDigest([]byte("h")), map[string]any{},
	)
	env, _ := s.SignStatement(stmt)
	env.Signatures = nil
	if err := Verify(env, s.PublicKey()); err == nil {
		t.Fatal("expected error on empty signatures")
	}
}

// ── LocalJournalPublisher ─────────────────────────────────────────────────

func mkJournal(t *testing.T) *LocalJournalPublisher {
	t.Helper()
	p, err := NewLocalJournalPublisher(filepath.Join(t.TempDir(), "journal.jsonl"))
	if err != nil {
		t.Fatalf("journal: %v", err)
	}
	return p
}

func TestJournal_PublishAndFetch(t *testing.T) {
	s := mkSigner(t)
	p := mkJournal(t)
	stmt, _ := BuildStatement(cruciblev1.PredicateWriteAttestation, "task/1/a",
		SubjectDigest([]byte("a")), map[string]any{"path": "/a"})
	env, _ := s.SignStatement(stmt)

	ctx := context.Background()
	rec, err := p.Publish(ctx, env)
	if err != nil {
		t.Fatalf("publish: %v", err)
	}
	if !rec.LocalJournalFallback {
		t.Fatal("expected LocalJournalFallback=true")
	}
	if rec.UUID == "" {
		t.Fatal("empty UUID")
	}
	got, err := p.Fetch(ctx, rec.UUID)
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}
	if got.Payload != env.Payload {
		t.Fatal("fetched envelope payload mismatch")
	}
}

func TestJournal_HashChainContinuity(t *testing.T) {
	s := mkSigner(t)
	p := mkJournal(t)
	ctx := context.Background()
	var uuids []string
	for i := 0; i < 5; i++ {
		stmt, _ := BuildStatement(cruciblev1.PredicateWriteAttestation, "task/1",
			SubjectDigest([]byte{byte(i)}), map[string]any{"i": i})
		env, _ := s.SignStatement(stmt)
		rec, err := p.Publish(ctx, env)
		if err != nil {
			t.Fatal(err)
		}
		uuids = append(uuids, rec.UUID)
	}

	// Re-read the journal and assert every entry's Prev matches the previous UUID.
	f, err := os.Open(p.Path())
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	zero := strings.Repeat("0", 64)
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	expectedPrev := zero
	i := 0
	for scanner.Scan() {
		var entry journalEntry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			t.Fatalf("line %d: %v", i, err)
		}
		if entry.Prev != expectedPrev {
			t.Fatalf("line %d: prev=%s, want %s", i, entry.Prev, expectedPrev)
		}
		if entry.UUID != uuids[i] {
			t.Fatalf("line %d: uuid=%s, want %s", i, entry.UUID, uuids[i])
		}
		expectedPrev = entry.UUID
		i++
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("scan: %v", err)
	}
	if i != 5 {
		t.Fatalf("expected 5 entries, got %d", i)
	}
}

func TestJournal_RejectsNilEnvelope(t *testing.T) {
	p := mkJournal(t)
	if _, err := p.Publish(context.Background(), nil); err == nil {
		t.Fatal("expected error on nil envelope")
	}
}

// ── RekorV2Publisher gating ───────────────────────────────────────────────

func TestRekorV2Publisher_GatedByEnv(t *testing.T) {
	t.Setenv("CRUCIBLE_REKOR_PUBLISH", "")
	if _, err := NewRekorV2Publisher("https://example.com", nil); err == nil {
		t.Fatal("expected gating error without CRUCIBLE_REKOR_PUBLISH=1")
	}
	t.Setenv("CRUCIBLE_REKOR_PUBLISH", "1")
	if _, err := NewRekorV2Publisher("https://example.com", nil); err == nil {
		t.Fatal("expected stub error even with env set (impl ships Phase 6)")
	}
}

// ── Service.Emit ──────────────────────────────────────────────────────────

func TestService_EmitRoundtrip(t *testing.T) {
	s := mkSigner(t)
	p := mkJournal(t)
	svc, err := NewService(s, p)
	if err != nil {
		t.Fatal(err)
	}
	pred := map[string]any{
		"task_id":             "task_1",
		"tenant_id":           "ten_a",
		"plan_hash":           strings.Repeat("a", 64),
		"estimated_cost_usd":  1.20,
		"estimated_duration_min": 10,
		"complexity":          "standard",
		"step_count":          3,
		"built_by_oidc":       s.OidcSubject(),
		"built_at":            time.Now().UTC().Format(time.RFC3339),
	}
	rec, err := svc.Emit(context.Background(),
		cruciblev1.PredicatePlanProposal, "task/1/plan",
		[]byte("plan-bytes"), pred,
	)
	if err != nil {
		t.Fatalf("Emit: %v", err)
	}
	env, err := svc.Fetch(context.Background(), rec.UUID)
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if err := Verify(env, s.PublicKey()); err != nil {
		t.Fatalf("Verify: %v", err)
	}
}

func TestNewService_NilArgs(t *testing.T) {
	if _, err := NewService(nil, nil); err == nil {
		t.Fatal("expected error on nil signer")
	}
	if _, err := NewService(mkSigner(t), nil); err == nil {
		t.Fatal("expected error on nil publisher")
	}
}

// ── Schema coverage ───────────────────────────────────────────────────────

func TestAllPredicateTypes_HaveSchemas(t *testing.T) {
	schemas := AllSchemas()
	for _, pt := range cruciblev1.AllPredicateTypes {
		if _, ok := schemas[pt]; !ok {
			t.Errorf("missing schema for predicate %s", pt)
		}
		if _, err := SchemaFor(pt); err != nil {
			t.Errorf("SchemaFor(%s): %v", pt, err)
		}
	}
	if len(cruciblev1.AllPredicateTypes) != 14 {
		t.Fatalf("expected 14 predicate types, got %d", len(cruciblev1.AllPredicateTypes))
	}
	if len(schemas) != 14 {
		t.Fatalf("expected 14 schemas, got %d", len(schemas))
	}
}

func TestValidateRequired_DetectsMissingField(t *testing.T) {
	// Build a payload missing one required field for WriteAttestation.
	bad := map[string]any{
		"task_id":   "task_1",
		"tenant_id": "ten_a",
		"repo":      "r",
		"base_sha":  "abc1234",
		"path":      "/a",
		"action":    "modify",
		// "content_sha256" missing
		"size_bytes":         1,
		"timestamp":          "2026-05-15T10:00:00Z",
		"agent_oidc_subject": "https://x",
	}
	b, _ := json.Marshal(bad)
	err := ValidateRequired(cruciblev1.PredicateWriteAttestation, b)
	if err == nil {
		t.Fatal("expected error for missing required field")
	}
	if !strings.Contains(err.Error(), "content_sha256") {
		t.Fatalf("expected error to name missing field, got %v", err)
	}
}

func TestValidateRequired_AcceptsCompletePayload(t *testing.T) {
	good := map[string]any{
		"task_id":            "task_1",
		"tenant_id":          "ten_a",
		"repo":               "r",
		"base_sha":           "abc1234",
		"path":               "/a",
		"action":             "modify",
		"content_sha256":     strings.Repeat("a", 64),
		"size_bytes":         1,
		"timestamp":          "2026-05-15T10:00:00Z",
		"agent_oidc_subject": "https://x",
	}
	b, _ := json.Marshal(good)
	if err := ValidateRequired(cruciblev1.PredicateWriteAttestation, b); err != nil {
		t.Fatalf("expected no error for complete payload, got %v", err)
	}
}

func TestSchemaFor_UnknownType(t *testing.T) {
	if _, err := SchemaFor("https://crucible.dev/Nope/v1"); err == nil {
		t.Fatal("expected error for unknown type")
	}
}
