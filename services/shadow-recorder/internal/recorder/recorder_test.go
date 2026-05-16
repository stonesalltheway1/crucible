package recorder

import (
	"context"
	"testing"
	"time"

	"github.com/crucible/services/shadow-recorder/internal/coverage"
	"github.com/crucible/services/shadow-recorder/internal/scrubber"
	"github.com/crucible/services/shadow-recorder/internal/storage"
	"github.com/crucible/services/shadow-recorder/internal/types"
)

func TestIngestPersistsAndCounts(t *testing.T) {
	r := New(Config{
		Scrubber: scrubber.NewClient(scrubber.Config{}), // dev passthrough
		Store:    storage.NewMemoryStore(),
		Coverage: coverage.New(),
	})
	key, err := r.Ingest(context.Background(), types.EnvoyAccessLogEntry{
		TenantID: "ten", UpstreamHost: "api.x.com", RequestMethod: "GET",
		RequestPath: "/v1/users/123", ResponseStatus: 200,
		ResponseBody: []byte(`{"id":"123"}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if key == "" {
		t.Fatal("empty key")
	}
	c, _ := r.Stats()
	if c != 1 {
		t.Errorf("captures=%d", c)
	}
}

func TestIngestRejectsIncomplete(t *testing.T) {
	r := New(Config{
		Scrubber: scrubber.NewClient(scrubber.Config{}),
		Store:    storage.NewMemoryStore(),
		Coverage: coverage.New(),
	})
	_, err := r.Ingest(context.Background(), types.EnvoyAccessLogEntry{TenantID: "ten"})
	if err != ErrIncompleteEntry {
		t.Errorf("err=%v", err)
	}
}

func TestIngestFailClosed(t *testing.T) {
	r := New(Config{
		Scrubber: scrubber.NewClient(scrubber.Config{Endpoint: "", FailClosed: true}),
		Store:    storage.NewMemoryStore(),
		Coverage: coverage.New(),
	})
	_, err := r.Ingest(context.Background(), types.EnvoyAccessLogEntry{
		TenantID: "ten", UpstreamHost: "h", RequestMethod: "GET", RequestPath: "/x",
		ResponseStatus: 200,
	})
	if err == nil {
		t.Fatal("expected fail-closed error")
	}
	_, scrubFails := r.Stats()
	if scrubFails == 0 {
		t.Errorf("scrub-failure not counted")
	}
}

func TestRunDueRerecordsCountsCandidates(t *testing.T) {
	cov := coverage.New()
	cov.Record("ten", "h", "GET", "/x", time.Now().Add(-90*24*time.Hour), 30*24*time.Hour)
	r := New(Config{
		Scrubber: scrubber.NewClient(scrubber.Config{}),
		Store:    storage.NewMemoryStore(),
		Coverage: cov,
	})
	got := r.RunDueRerecords(context.Background())
	if got != 1 {
		t.Errorf("got %d", got)
	}
}
