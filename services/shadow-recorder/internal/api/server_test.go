package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/crucible/services/shadow-recorder/internal/coverage"
	"github.com/crucible/services/shadow-recorder/internal/recorder"
	"github.com/crucible/services/shadow-recorder/internal/scrubber"
	"github.com/crucible/services/shadow-recorder/internal/storage"
	"github.com/crucible/services/shadow-recorder/internal/types"
)

func mkServer(t *testing.T) (*httptest.Server, *coverage.Tracker, *recorder.Recorder) {
	t.Helper()
	cov := coverage.New()
	rec := recorder.New(recorder.Config{
		Scrubber: scrubber.NewClient(scrubber.Config{}),
		Store:    storage.NewMemoryStore(),
		Coverage: cov,
	})
	srv := NewServer(Config{Version: "test", Recorder: rec, Coverage: cov, Storage: storage.NewMemoryStore()})
	return httptest.NewServer(srv), cov, rec
}

func TestIngestRoundtrip(t *testing.T) {
	ts, cov, _ := mkServer(t)
	defer ts.Close()

	body, _ := json.Marshal(types.EnvoyAccessLogEntry{
		TenantID: "ten_x", UpstreamHost: "api.acme.com",
		RequestMethod: "GET", RequestPath: "/v1/users/123",
		ResponseStatus: 200, ResponseBody: []byte(`{"id":"123"}`),
	})
	r, err := http.Post(ts.URL+"/v1/ingest/envoy", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	if r.StatusCode != http.StatusCreated {
		t.Errorf("status=%d", r.StatusCode)
	}
	cov2 := cov.HostCoverage("ten_x", "api.acme.com")
	if cov2.Endpoints != 1 {
		t.Errorf("endpoints=%d", cov2.Endpoints)
	}
}

func TestCoverageRequiresTenant(t *testing.T) {
	ts, _, _ := mkServer(t)
	defer ts.Close()
	r, _ := http.Get(ts.URL + "/v1/coverage")
	if r.StatusCode != http.StatusBadRequest {
		t.Errorf("status=%d", r.StatusCode)
	}
}

func TestMetricsExposed(t *testing.T) {
	ts, _, _ := mkServer(t)
	defer ts.Close()
	r, _ := http.Get(ts.URL + "/metrics")
	if r.StatusCode != http.StatusOK {
		t.Fatalf("status=%d", r.StatusCode)
	}
	buf := make([]byte, 4096)
	n, _ := r.Body.Read(buf)
	body := string(buf[:n])
	if !strings.Contains(body, "crucible_shadow_captures_total") {
		t.Errorf("metric missing: %s", body)
	}
}
