package verifierbridge

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	cruciblev1 "github.com/crucible/sdk-go/crucible/v1"
)

func validRequest() VerifyRequest {
	return VerifyRequest{
		TaskID:   "t1",
		TenantID: "ten",
		BaseSHA:  "abc",
		Diff: cruciblev1.Diff{Files: []cruciblev1.FileChange{
			{Path: "a.py", Action: cruciblev1.ActionModify, ContentSha256: "0xa"},
		}},
		Routing: cruciblev1.Routing{
			ExecutorVendor: "anthropic", VerifierVendor: "google",
			ExecutorModel: "claude-opus-4-7", VerifierModel: "gemini-3.1-pro",
		},
		Languages:         []string{"python"},
		ExecutorSandboxID: "sb_exec",
	}
}

func TestVerify_stubApproves(t *testing.T) {
	b := NewStub()
	resp, err := b.Verify(context.Background(), validRequest())
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if resp.Approval == nil {
		t.Fatalf("expected approval")
	}
}

func TestVerify_stubRefusesSameFamily(t *testing.T) {
	req := validRequest()
	req.Routing.VerifierVendor = "anthropic"
	b := NewStub()
	_, err := b.Verify(context.Background(), req)
	if err == nil {
		t.Fatalf("expected error on same-family")
	}
}

func TestVerify_httpBridge_relaysApproval(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/twin/verify/bundle" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		_, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"approval": map[string]any{
				"task_id":      "t1",
				"diff_hash":    "0xabc",
				"verdict":      "approved",
				"rubric_score": 0.91,
			},
		})
	}))
	defer ts.Close()
	b := &httpBridge{addr: ts.URL, client: ts.Client()}
	resp, err := b.Verify(context.Background(), validRequest())
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if resp.Approval == nil || resp.Approval.RubricScore < 0.9 {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestVerify_httpBridge_surfacesRejection(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"rejection": map[string]any{
				"task_id":   "t1",
				"diff_hash": "0xabc",
				"verdict":   "rejected",
				"rejection_reasons": []map[string]any{
					{"category": "mutation_survived", "detail": "x"},
				},
			},
		})
	}))
	defer ts.Close()
	b := &httpBridge{addr: ts.URL, client: ts.Client()}
	resp, err := b.Verify(context.Background(), validRequest())
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if resp.Rejection == nil {
		t.Fatalf("expected rejection")
	}
}

func TestVerify_httpBridge_unreachable_returnsTypedError(t *testing.T) {
	b := &httpBridge{addr: "http://127.0.0.1:1", client: http.DefaultClient}
	_, err := b.Verify(context.Background(), validRequest())
	if err == nil {
		t.Fatalf("expected NotConnectedError")
	}
	var nce *NotConnectedError
	if !errors.As(err, &nce) {
		t.Fatalf("expected NotConnectedError; got %T %v", err, err)
	}
}

func TestVerify_httpBridge_serverError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(500)
		_, _ = w.Write([]byte(`{"error": {"code": "internal_error", "message": "oops"}}`))
	}))
	defer ts.Close()
	b := &httpBridge{addr: ts.URL, client: ts.Client()}
	_, err := b.Verify(context.Background(), validRequest())
	if err == nil {
		t.Fatalf("expected error on 500")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Fatalf("error should mention 500; got %v", err)
	}
}
