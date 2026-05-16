package verification

import (
	"errors"
	"testing"

	cruciblev1 "github.com/crucible/sdk-go/crucible/v1"
)

func newValidRequest() *VerificationRequest {
	return &VerificationRequest{
		TaskID:   "task_xyz",
		TenantID: "ten_acme",
		BaseSHA:  "abc123",
		Diff: cruciblev1.Diff{
			Files: []cruciblev1.FileChange{
				{Path: "api/webhooks.ts", Action: cruciblev1.ActionModify, ContentSha256: "0xdef"},
			},
			BaseSha: "abc123",
		},
		Routing: cruciblev1.Routing{
			ExecutorModel:  "claude-opus-4-7",
			ExecutorVendor: "anthropic",
			VerifierModel:  "gemini-3.1-pro",
			VerifierVendor: "google",
		},
		Languages:         []string{"typescript"},
		ExecutorSandboxID: "sb_executor_01",
	}
}

func TestValidate_acceptsValidRequest(t *testing.T) {
	r := newValidRequest()
	if err := r.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}
}

func TestValidate_refusesSameFamily(t *testing.T) {
	r := newValidRequest()
	r.Routing.VerifierVendor = "anthropic"
	err := r.Validate()
	if err == nil {
		t.Fatalf("expected SameFamilyError")
	}
	var sfe *SameFamilyError
	if !errors.As(err, &sfe) {
		t.Fatalf("expected SameFamilyError, got %T: %v", err, err)
	}
}

func TestValidate_refusesEmptyDiff(t *testing.T) {
	r := newValidRequest()
	r.Diff.Files = nil
	if err := r.Validate(); err == nil {
		t.Fatalf("expected error for empty diff")
	}
}

func TestValidate_refusesMissingSandboxID(t *testing.T) {
	r := newValidRequest()
	r.ExecutorSandboxID = ""
	if err := r.Validate(); err == nil {
		t.Fatalf("expected error for missing executor_sandbox_id")
	}
}

func TestAuditNoLeakage_detectsTopLevelField(t *testing.T) {
	payload := map[string]any{
		"task_id":  "t1",
		"reasoning": "the model thought about it deeply",
	}
	err := AuditNoLeakage(payload)
	if err == nil {
		t.Fatalf("expected LeakageError for top-level reasoning field")
	}
	var le *LeakageError
	if !errors.As(err, &le) {
		t.Fatalf("expected LeakageError type")
	}
}

func TestAuditNoLeakage_detectsNestedField(t *testing.T) {
	payload := map[string]any{
		"diff": map[string]any{
			"files": []any{
				map[string]any{
					"path":      "api/foo.ts",
					"scratchpad": "internal-only",
				},
			},
		},
	}
	if err := AuditNoLeakage(payload); err == nil {
		t.Fatalf("expected LeakageError for nested scratchpad")
	}
}

func TestAuditNoLeakage_detectsVariantSpelling(t *testing.T) {
	for _, name := range []string{"chain_of_thought", "cot", "thinking_trace", "agent_trace", "executor_trace", "reflection"} {
		payload := map[string]any{"task_id": "t", name: "x"}
		if err := AuditNoLeakage(payload); err == nil {
			t.Fatalf("expected LeakageError for variant %q", name)
		}
	}
}

func TestAuditNoLeakage_clean(t *testing.T) {
	payload := map[string]any{
		"task_id": "t",
		"diff":    map[string]any{"files": []any{}},
	}
	if err := AuditNoLeakage(payload); err != nil {
		t.Fatalf("unexpected leak on clean payload: %v", err)
	}
}

func TestAuditRequest_rejectsReasoningPath(t *testing.T) {
	r := newValidRequest()
	r.Diff.Files = append(r.Diff.Files, cruciblev1.FileChange{
		Path: "agent_trace/step01.json", Action: cruciblev1.ActionAdd,
	})
	if err := r.AuditNoLeakage(); err == nil {
		t.Fatalf("expected LeakageError for reasoning-shaped path")
	}
}

func TestAuditRequest_clean(t *testing.T) {
	r := newValidRequest()
	if err := r.AuditNoLeakage(); err != nil {
		t.Fatalf("clean request flagged: %v", err)
	}
}
