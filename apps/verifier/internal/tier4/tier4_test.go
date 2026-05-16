package tier4

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	cruciblev1 "github.com/crucible/sdk-go/crucible/v1"
	"github.com/crucible/verifier/internal/verification"
	"github.com/crucible/verifier/pkg/testreport"
)

func newReq() *verification.VerificationRequest {
	return &verification.VerificationRequest{
		TaskID:   "task_t4",
		TenantID: "ten_acme",
		BaseSHA:  "abc",
		Diff: cruciblev1.Diff{Files: []cruciblev1.FileChange{
			{Path: "x.go", Action: cruciblev1.ActionModify, ContentSha256: "0xa"},
		}},
		ExecutorSandboxID: "sb_executor",
		PerTaskSignals: verification.TaskSignals{
			ScrubberFiredOnAllPII: true,
			ScrubberAuditEntryCount: 0,
		},
	}
}

func newVerifier() *Verifier {
	v := NewVerifier()
	v.Builder = &MemoryBuilder{Hash: "deadbeef"}
	v.Attestor = &MemoryAttestor{}
	v.Differ = MemoryDiffer{}
	v.Now = func() time.Time { return time.Unix(1747300000, 0).UTC() }
	return v
}

func TestVerify_bitIdentical_passes(t *testing.T) {
	req := newReq()
	req.AttestationChain = []string{
		strings.Repeat("d", 4) + "beef" + strings.Repeat("0", 56), // not the right hash
	}
	// Set executor rebuild hash to match the builder's.
	hash := "deadbeef" + strings.Repeat("0", 56)
	req.AttestationChain = []string{hash}
	v := newVerifier()
	v.Builder = &MemoryBuilder{Hash: hash}
	r, err := v.Verify(context.Background(), req)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if r.HonestCI == nil {
		t.Fatalf("nil HonestCI")
	}
	if !r.HonestCI.BitIdentical {
		t.Fatalf("expected bit-identical match")
	}
	if !r.Passed {
		t.Fatalf("expected Passed; got Verdict=%s findings=%+v", r.Verdict, r.Findings)
	}
}

func TestVerify_mismatch_fails(t *testing.T) {
	req := newReq()
	hashA := "aaaa" + strings.Repeat("0", 60)
	hashB := "bbbb" + strings.Repeat("0", 60)
	req.AttestationChain = []string{hashA}
	v := newVerifier()
	v.Builder = &MemoryBuilder{Hash: hashB}
	r, err := v.Verify(context.Background(), req)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if r.HonestCI.BitIdentical {
		t.Fatalf("expected mismatch")
	}
	if r.Passed {
		t.Fatalf("expected Failed; got Verdict=%s", r.Verdict)
	}
	found := false
	for _, f := range r.Findings {
		if f.Category == "honest_ci_mismatch" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected honest_ci_mismatch finding; got %+v", r.Findings)
	}
}

func TestVerify_buildFailure(t *testing.T) {
	req := newReq()
	v := newVerifier()
	v.Builder = &MemoryBuilder{Err: errors.New("nix exploded")}
	r, err := v.Verify(context.Background(), req)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if r.Passed {
		t.Fatalf("expected failure")
	}
	if !strings.Contains(r.Error, "nix exploded") {
		t.Fatalf("expected nix-exploded in report.Error; got %q", r.Error)
	}
}

func TestVerify_rejectsForgedAttestation(t *testing.T) {
	req := newReq()
	hash := "cafe" + strings.Repeat("0", 60)
	req.AttestationChain = []string{hash, "not-a-rekor-uuid-suspicious"}
	v := newVerifier()
	v.Builder = &MemoryBuilder{Hash: hash}
	attestor := &MemoryAttestor{
		Verifies: map[string]VerifyResult{
			"not-a-rekor-uuid-suspicious": {Valid: false, Reasons: []string{"forged-looking entry"}},
		},
	}
	v.Attestor = attestor
	r, err := v.Verify(context.Background(), req)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	found := false
	for _, f := range r.Findings {
		if f.Category == "attestation_invalid" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected attestation_invalid finding for forged entry; got %+v", r.Findings)
	}
	if r.Passed {
		t.Fatalf("forged attestation must reject")
	}
}

func TestVerify_scrubberAuditMissing(t *testing.T) {
	req := newReq()
	hash := "feed" + strings.Repeat("0", 60)
	req.AttestationChain = []string{hash}
	req.PerTaskSignals.ScrubberAuditEntryCount = 3
	req.PerTaskSignals.ScrubberFiredOnAllPII = false
	v := newVerifier()
	v.Builder = &MemoryBuilder{Hash: hash}
	r, err := v.Verify(context.Background(), req)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if r.HonestCI.ScrubberAuditOK {
		t.Fatalf("expected scrubber_audit_ok=false when entries exist but fire-on-all is false")
	}
	if r.Passed {
		t.Fatalf("expected fail when scrubber didn't fire on all PII")
	}
}

func TestSigstoreAttestor_verifyRejectsBareString(t *testing.T) {
	att := &SigstoreAttestor{}
	v, err := att.Verify(context.Background(), "totally-not-a-rekor-uuid")
	if err != nil {
		t.Fatal(err)
	}
	if v.Valid {
		t.Fatalf("non-rekor entry must be rejected")
	}
}

func TestSigstoreAttestor_verifyAcceptsRekorPrefix(t *testing.T) {
	att := &SigstoreAttestor{}
	v, err := att.Verify(context.Background(), "rekor:7d8a2c")
	if err != nil {
		t.Fatal(err)
	}
	if !v.Valid {
		t.Fatalf("rekor:-prefixed entry should validate at Phase-4 surface")
	}
}

func TestExecutorHashHelper(t *testing.T) {
	// 64-hex passes
	if !isHex64(strings.Repeat("a", 64)) {
		t.Fatal("64-hex string should match")
	}
	if isHex64(strings.Repeat("a", 63)) {
		t.Fatal("63-char string must not match")
	}
	if isHex64(strings.Repeat("g", 64)) {
		t.Fatal("non-hex chars must not match")
	}
}

func TestSlsaLevelEqualsThree(t *testing.T) {
	req := newReq()
	hash := strings.Repeat("a", 64)
	req.AttestationChain = []string{hash}
	v := newVerifier()
	v.Builder = &MemoryBuilder{Hash: hash}
	r, _ := v.Verify(context.Background(), req)
	if r.HonestCI.SLSALevel != 3 {
		t.Fatalf("expected SLSA-L3; got %d", r.HonestCI.SLSALevel)
	}
}

// Test that report Validation passes on a successful Tier-4 output.
func TestVerify_outputValidates(t *testing.T) {
	req := newReq()
	hash := strings.Repeat("a", 64)
	req.AttestationChain = []string{hash}
	v := newVerifier()
	v.Builder = &MemoryBuilder{Hash: hash}
	r, _ := v.Verify(context.Background(), req)
	if r.SchemaVersion != testreport.SchemaVersion {
		t.Fatalf("schema_version mismatch")
	}
}
