package coverage

import (
	"testing"
	"time"
)

func TestRecordAndHostCoverage(t *testing.T) {
	tr := New()
	now := time.Now()
	tr.Record("ten_x", "api.acme.com", "GET", "/v1/customers/cus_abc123", now, 30*24*time.Hour)
	tr.Record("ten_x", "api.acme.com", "GET", "/v1/customers/cus_def456", now, 30*24*time.Hour)
	tr.Record("ten_x", "api.acme.com", "POST", "/v1/charges", now, 30*24*time.Hour)

	cov := tr.HostCoverage("ten_x", "api.acme.com")
	if cov.Endpoints != 2 {
		t.Errorf("endpoints=%d want 2 (templates collapse the two cus_ paths)", cov.Endpoints)
	}
	if cov.TotalHits != 3 {
		t.Errorf("hits=%d want 3", cov.TotalHits)
	}
}

func TestPathTemplate(t *testing.T) {
	cases := map[string]string{
		"/v1/customers/cus_abc123":           "/v1/customers/{id}",
		"/v1/charges/123/refunds":            "/v1/charges/{id}/refunds",
		"/users/12345abcdef67890123456":       "/users/{id}",
		"/users/00000000-0000-0000-0000-000000000000": "/users/{id}",
		"/health":                            "/health",
	}
	for in, want := range cases {
		if got := pathTemplate(in); got != want {
			t.Errorf("pathTemplate(%q)=%q want %q", in, got, want)
		}
	}
}

func TestDueRerecords(t *testing.T) {
	tr := New()
	now := time.Now()
	tr.Record("ten", "h", "GET", "/x", now.Add(-31*24*time.Hour), 30*24*time.Hour)
	tr.Record("ten", "h", "GET", "/y", now, 30*24*time.Hour)
	due := tr.DueRerecords(now)
	if len(due) != 1 {
		t.Fatalf("got %d", len(due))
	}
	if due[0].PathTemplate != "/x" {
		t.Errorf("wrong path: %q", due[0].PathTemplate)
	}
}

func TestAllHosts(t *testing.T) {
	tr := New()
	now := time.Now()
	tr.Record("ten", "a", "GET", "/x", now, time.Hour)
	tr.Record("ten", "b", "GET", "/y", now, time.Hour)
	tr.Record("other", "a", "GET", "/z", now, time.Hour)
	out := tr.AllHosts("ten")
	if len(out) != 2 {
		t.Errorf("got %d hosts", len(out))
	}
}
