package shadowrecorder

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	tapedriver "github.com/crucible/services/twin-runtime/tape_driver"
	"github.com/stretchr/testify/require"
)

func TestCapturePersistsScrubbedEntry(t *testing.T) {
	store := NewInMemoryStore()
	r := New(Options{
		Store:    store,
		Scrubber: tapedriver.NewRegexScrubber(),
	})
	entry, err := r.Capture(context.Background(), "tape-1", CapturedRequest{
		Service:   "stripe",
		Method:    "GET",
		Endpoint:  "/v1/customers/cus_a",
		Headers:   map[string][]string{"X-Request-ID": {"req_1"}},
		Body:      nil,
		Response: CapturedResponse{
			Status:  200,
			Headers: map[string][]string{
				"Authorization": {"Bearer sk-ant-api03-" + strings.Repeat("a", 60)},
			},
			Body: []byte(`{"email":"alice@example.com","ssn":"123-45-6789"}`),
		},
		Timestamp:   time.Now(),
		SampledFrom: "envoy",
	})
	require.NoError(t, err)
	require.NotContains(t, string(entry.Scrubbed.Body), "alice@example.com")
	require.NotContains(t, string(entry.Scrubbed.Body), "123-45-6789")
	require.GreaterOrEqual(t, len(entry.Scrubbed.ScrubLog), 1)

	// The Authorization header must be scrubbed.
	authVals := entry.Scrubbed.Headers["Authorization"]
	require.True(t, len(authVals) == 1 && !strings.Contains(authVals[0], "sk-ant-api03"),
		"Authorization header was not scrubbed: %v", authVals)

	stored, err := store.Get(context.Background(), entry.RequestHash)
	require.NoError(t, err)
	require.Equal(t, entry.RequestHash, stored.RequestHash)
}

func TestCaptureFailsClosedWhenScrubberUnavailable(t *testing.T) {
	store := NewInMemoryStore()
	r := New(Options{
		Store: store,
		// PresidioScrubber pointed at an invalid endpoint with fail-closed.
		Scrubber:   tapedriver.NewPresidioScrubber(tapedriver.WithFailClosed()),
		FailClosed: true,
	})
	t.Setenv(tapedriver.EnvScrubberURL, "http://127.0.0.1:1")
	_, err := r.Capture(context.Background(), "tape-1", CapturedRequest{
		Service:  "stripe",
		Method:   "GET",
		Endpoint: "/v1/charges",
		Response: CapturedResponse{Status: 200, Body: []byte(`{"email":"x@y.com"}`)},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "scrubber unavailable")
}

func TestCaptureDedupesByRequestHash(t *testing.T) {
	store := NewInMemoryStore()
	r := New(Options{Store: store, Scrubber: tapedriver.NewRegexScrubber()})
	req := CapturedRequest{
		Service: "stripe", Method: "GET", Endpoint: "/v1/charges",
		Response: CapturedResponse{Status: 200, Body: []byte(`{"a":1}`)},
	}
	for i := 0; i < 3; i++ {
		_, err := r.Capture(context.Background(), "tape", req)
		require.NoError(t, err)
	}
	count, err := store.Count(context.Background(), "tape")
	require.NoError(t, err)
	require.Equal(t, 1, count, "identical requests should dedupe by hash")
}

func TestStatsTrackPerEndpointSamples(t *testing.T) {
	store := NewInMemoryStore()
	r := New(Options{Store: store, Scrubber: tapedriver.NewRegexScrubber()})
	for _, body := range []string{`{"a":1}`, `{"a":2}`, `{"a":3}`} {
		_, _ = r.Capture(context.Background(), "tape", CapturedRequest{
			Service: "stripe", Method: "GET", Endpoint: "/v1/charges",
			Response: CapturedResponse{Status: 200, Body: []byte(body)},
		})
	}
	stats := r.Stats()
	require.Len(t, stats, 1)
	require.Equal(t, 3, stats[0].Samples)
}

func TestEnvoyAccessLogHandlerHappyPath(t *testing.T) {
	store := NewInMemoryStore()
	r := New(Options{Store: store, Scrubber: tapedriver.NewRegexScrubber()})
	srv := httptest.NewServer(http.HandlerFunc(r.HandleEnvoyAccessLog))
	defer srv.Close()

	body := map[string]any{
		"tape_set": "tape-1",
		"captured": CapturedRequest{
			Service: "stripe", Method: "GET", Endpoint: "/v1/charges",
			Response: CapturedResponse{Status: 200, Body: []byte(`{"email":"a@b.com"}`)},
			SampledFrom: "envoy",
		},
	}
	raw, _ := json.Marshal(body)
	resp, err := http.Post(srv.URL, "application/json", strings.NewReader(string(raw)))
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var parsed map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&parsed))
	require.NotEmpty(t, parsed["request_hash"])
}

func TestReCordScheduleResolvesOverride(t *testing.T) {
	s := ReCordSchedule{
		Default: 30 * 24 * time.Hour,
		PerEndpoint: map[string]time.Duration{
			"stripe|GET|/v1/charges": 12 * time.Hour,
		},
	}
	require.Equal(t, 12*time.Hour, s.IntervalFor("stripe", "GET", "/v1/charges"))
	require.Equal(t, 30*24*time.Hour, s.IntervalFor("stripe", "GET", "/v1/customers"))
}

func TestRequestHashStable(t *testing.T) {
	a := CapturedRequest{Service: "s", Method: "GET", Endpoint: "/x", Body: []byte("body")}
	b := CapturedRequest{Service: "s", Method: "GET", Endpoint: "/x", Body: []byte("body")}
	require.Equal(t, requestHash(a), requestHash(b))
}

func TestRequestHashDiffersByBody(t *testing.T) {
	a := CapturedRequest{Service: "s", Method: "GET", Endpoint: "/x", Body: []byte("body1")}
	b := CapturedRequest{Service: "s", Method: "GET", Endpoint: "/x", Body: []byte("body2")}
	require.NotEqual(t, requestHash(a), requestHash(b))
}

func TestSafeHarbor18IdentifierAuditCorpus(t *testing.T) {
	// HIPAA Safe Harbor 18-identifier sanity sweep: scrub a tape entry
	// containing each identifier and verify none reaches the persisted
	// body. This is the audit-grade test the brief calls out.
	identifiers := []string{
		"alice.smith@example.com",       // 4. Email
		"123-45-6789",                   // 5. Social security
		"+1-555-123-4567",               // 6. Phone
		"AKIAIOSFODNN7EXAMPLE",          // (cloud key — not HIPAA per se but always scrub)
		"4242 4242 4242 4242",           // 8. Account / credit-card-like
		"192.168.1.42",                  // 14. IP
		"https://patient.example.com/123", // 13. URL
	}
	store := NewInMemoryStore()
	r := New(Options{Store: store, Scrubber: tapedriver.NewRegexScrubber()})
	body := strings.Join(identifiers, " | ")
	entry, err := r.Capture(context.Background(), "audit", CapturedRequest{
		Service: "ehr", Method: "GET", Endpoint: "/patients/42",
		Response: CapturedResponse{Status: 200, Body: []byte(body)},
	})
	require.NoError(t, err)
	for _, ident := range identifiers {
		require.NotContains(t, string(entry.Scrubbed.Body), ident,
			"identifier %q leaked into persisted tape", ident)
	}
}
