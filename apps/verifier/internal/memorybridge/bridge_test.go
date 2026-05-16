package memorybridge

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	cruciblev1 "github.com/crucible/sdk-go/crucible/v1"
)

func TestCheckCompliance_RefusesEmptyTenant(t *testing.T) {
	b := &httpBridge{addr: "http://nope", client: http.DefaultClient}
	_, err := b.CheckCompliance(context.Background(), CheckRequest{})
	if err == nil || !strings.Contains(err.Error(), "tenant_id") {
		t.Fatalf("want tenant_id error; got %v", err)
	}
}

func TestCheckCompliance_HappyPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/memory/check_compliance" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(checkRespWire{
			Report: cruciblev1.ComplianceReport{
				DiffHash:           "h",
				ConventionsChecked: 7,
			},
		})
	}))
	defer srv.Close()

	b := &httpBridge{addr: srv.URL, client: http.DefaultClient}
	rep, err := b.CheckCompliance(context.Background(), CheckRequest{
		TenantID: "ten_a",
		Diff:     cruciblev1.Diff{Files: []cruciblev1.FileChange{{Path: "a.go"}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if rep.ConventionsChecked != 7 {
		t.Fatalf("got %d", rep.ConventionsChecked)
	}
}

func TestNoopBridge_EmptyReport(t *testing.T) {
	b := NewStub()
	rep, err := b.CheckCompliance(context.Background(), CheckRequest{TenantID: "ten_a"})
	if err != nil {
		t.Fatal(err)
	}
	if rep.ConventionsChecked != 0 || len(rep.Violations) != 0 {
		t.Fatal("noop bridge must return empty report")
	}
}

func TestNew_StubsWhenEnvUnset(t *testing.T) {
	t.Setenv(EnvRouterAddr, "")
	b := New()
	if _, ok := b.(*noop); !ok {
		t.Fatalf("expected noop bridge when env unset; got %T", b)
	}
}

func TestListConventions_SendsScope(t *testing.T) {
	var capturedBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := make([]byte, r.ContentLength)
		_, _ = r.Body.Read(buf)
		capturedBody = buf
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(listRespWire{Conventions: nil})
	}))
	defer srv.Close()

	b := &httpBridge{addr: srv.URL, client: http.DefaultClient}
	_, _ = b.ListConventions(context.Background(), ListRequest{
		TenantID: "ten_a",
		Scope:    cruciblev1.ScopeFilter{FileGlob: "api/**/*.ts"},
	})
	if !strings.Contains(string(capturedBody), "api/**/*.ts") {
		t.Fatalf("scope not propagated; body=%q", capturedBody)
	}
}
