package distill

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDistillReturnsCandidates(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"output": `[{"category":"Performance defaults","rule":"Use cursor pagination, never offset.","file_glob":"api/**/*.go","rationale":"offset is O(N) per page on Postgres.","evidence_quote":"offset is O(N) on the page"}]`,
		})
	}))
	defer ts.Close()

	c := NewClient(Config{Endpoint: ts.URL, Model: "claude-haiku-4-5-20251001"})
	cands, err := c.Distill(context.Background(), Excerpt{
		Repo: "acme/payments", TenantID: "t",
		SourceChannel: "pr_comment", SourcePath: "pr/123",
		Body: "Please use cursor pagination here — offset is O(N) on the page.",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(cands) != 1 {
		t.Fatalf("got %d", len(cands))
	}
	if cands[0].Category != "Performance defaults" {
		t.Errorf("category=%q", cands[0].Category)
	}
	if cands[0].FileGlob != "api/**/*.go" {
		t.Errorf("file_glob=%q", cands[0].FileGlob)
	}
}

func TestDistillStripsFencedCodeBlock(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"output": "```json\n[]\n```",
		})
	}))
	defer ts.Close()
	c := NewClient(Config{Endpoint: ts.URL})
	got, err := c.Distill(context.Background(), Excerpt{Body: "noise"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty, got %v", got)
	}
}

func TestDistillRejectsInvalidCategory(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"output": `[{"category":"Made-Up-Category","rule":"x","file_glob":"**/*","rationale":"y","evidence_quote":"z"}]`,
		})
	}))
	defer ts.Close()
	c := NewClient(Config{Endpoint: ts.URL})
	got, _ := c.Distill(context.Background(), Excerpt{Body: "x"})
	if len(got) != 0 {
		t.Errorf("invalid category was admitted: %v", got)
	}
}

func TestDistillBatchParallelism(t *testing.T) {
	calls := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		_ = json.NewEncoder(w).Encode(map[string]any{"output": "[]"})
	}))
	defer ts.Close()
	c := NewClient(Config{Endpoint: ts.URL})
	exs := []Excerpt{
		{Body: "a"}, {Body: "b"}, {Body: "c"}, {Body: "d"},
	}
	_, err := c.DistillBatch(context.Background(), exs, 2)
	if err != nil {
		t.Fatal(err)
	}
	if calls != 4 {
		t.Errorf("calls=%d want 4", calls)
	}
}

func TestOfflineFallback(t *testing.T) {
	c := NewClient(Config{Endpoint: ""})
	got, err := c.Distill(context.Background(), Excerpt{Body: "x"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Errorf("offline returned %d candidates", len(got))
	}
}
