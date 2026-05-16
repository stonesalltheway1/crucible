package api

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/crucible/apps/cartographer/internal/distill"
	"github.com/crucible/apps/cartographer/internal/types"
)

func writeFile(t *testing.T, p, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestServerSubmitAndPoll(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "go.mod"), "module x")
	writeFile(t, filepath.Join(root, "main.go"), "package main\nfunc main() {}")
	writeFile(t, filepath.Join(root, ".editorconfig"), "[*]\nindent_style=tab\n")

	srv := NewServer(Config{
		Version:   "test",
		LLMClient: distill.NewClient(distill.Config{}),
	})
	ts := httptest.NewServer(srv)
	defer ts.Close()

	body, _ := json.Marshal(types.CartographyJob{
		TenantID: "ten_x", Repo: "acme/svc", RepoLocalPath: root,
	})
	resp, err := http.Post(ts.URL+"/v1/cartography", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("status=%d", resp.StatusCode)
	}
	var sub map[string]string
	_ = json.NewDecoder(resp.Body).Decode(&sub)
	resp.Body.Close()
	id := sub["job_id"]
	if id == "" {
		t.Fatal("no job_id returned")
	}

	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		time.Sleep(50 * time.Millisecond)
		r, err := http.Get(ts.URL + "/v1/cartography/" + id)
		if err != nil {
			t.Fatal(err)
		}
		body, _ := io.ReadAll(r.Body)
		r.Body.Close()
		// Once the job is done the server returns the full
		// CartographyResult (which doesn't have a `state` field). Use
		// the presence of `completed_at` as the done sentinel.
		var done types.CartographyResult
		if err := json.Unmarshal(body, &done); err == nil && !done.CompletedAt.IsZero() {
			return
		}
		var status types.JobStatus
		if err := json.Unmarshal(body, &status); err == nil && status.State != "" {
			if status.State == "done" {
				return
			}
			if status.State == "error" {
				t.Fatalf("job error: %s", status.Error)
			}
		}
	}
	t.Fatal("job did not complete in time")
}

func TestSubmitRejectsBadRequest(t *testing.T) {
	srv := NewServer(Config{})
	ts := httptest.NewServer(srv)
	defer ts.Close()
	resp, err := http.Post(ts.URL+"/v1/cartography", "application/json", bytes.NewReader([]byte(`{}`)))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status=%d", resp.StatusCode)
	}
}

func TestHealthz(t *testing.T) {
	srv := NewServer(Config{Version: "v1"})
	ts := httptest.NewServer(srv)
	defer ts.Close()
	resp, err := http.Get(ts.URL + "/healthz")
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status=%d", resp.StatusCode)
	}
}

