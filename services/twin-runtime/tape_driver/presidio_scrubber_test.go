package tapedriver

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// fakePresidioServer mirrors the Python service's /scrub contract.
func fakePresidioServer(t *testing.T, requireToken string) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"status":"ready"}`))
	})
	mux.HandleFunc("/scrub", func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if requireToken != "" && auth != "Bearer "+requireToken {
			http.Error(w, "bad token", http.StatusForbidden)
			return
		}
		var body scrubRequestBody
		raw, _ := io.ReadAll(r.Body)
		if err := json.Unmarshal(raw, &body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		// Trivial fake: redact "secret" → "[REDACTED]" and report one rewrite.
		scrubbed := strings.ReplaceAll(body.Payload, "secret", "[REDACTED]")
		resp := scrubResponseBody{Scrubbed: scrubbed}
		resp.Report.TapeSet = body.TapeSet
		resp.Report.ElapsedMs = 1
		resp.Report.Rewrites = append(resp.Report.Rewrites, struct {
			Scrubber      string `json:"scrubber"`
			Field         string `json:"field"`
			BeforeHash    string `json:"before_hash"`
			After         string `json:"after"`
			Operator      string `json:"operator"`
			Algorithm     string `json:"algorithm"`
			Ff3DomainSize int    `json:"ff3_domain_size"`
			TapeSet       string `json:"tape_set"`
			TimestampMs   int64  `json:"timestamp_ms"`
		}{
			Scrubber:   "FAKE_SECRET",
			Field:      "[inline]",
			BeforeHash: "sha256:fake",
			After:      "[REDACTED]",
			Operator:   "REDACT",
		})
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})
	return httptest.NewServer(mux)
}

func TestPresidioScrubberSendsBearerToken(t *testing.T) {
	srv := fakePresidioServer(t, "tok123")
	defer srv.Close()
	t.Setenv(EnvScrubberURL, srv.URL)
	t.Setenv(EnvScrubberToken, "tok123")
	s := NewPresidioScrubber()
	out, report := s.Scrub([]byte("this is a secret payload"))
	require.NotContains(t, string(out), "secret")
	require.Contains(t, string(out), "[REDACTED]")
	require.Len(t, report.Rewrites, 1)
	require.Equal(t, "FAKE_SECRET", report.Rewrites[0].Scrubber)
}

func TestPresidioScrubberFailsOpenToFallback(t *testing.T) {
	// Point at a non-listening endpoint to force a transport error.
	t.Setenv(EnvScrubberURL, "http://127.0.0.1:1")
	t.Setenv(EnvScrubberToken, "")
	s := NewPresidioScrubber()
	out, report := s.Scrub([]byte("email a@b.com"))
	require.NotContains(t, string(out), "a@b.com")
	require.NotEmpty(t, report.Rewrites)
	require.Equal(t, "email", report.Rewrites[0].Scrubber)
}

func TestPresidioScrubberFailClosed(t *testing.T) {
	t.Setenv(EnvScrubberURL, "http://127.0.0.1:1")
	s := NewPresidioScrubber(WithFailClosed())
	out, report := s.Scrub([]byte("email a@b.com"))
	require.Nil(t, out)
	require.Len(t, report.Rewrites, 1)
	require.Equal(t, "presidio-unavailable", report.Rewrites[0].Scrubber)
}

func TestPresidioScrubberAddsJSONContentTypeHint(t *testing.T) {
	var observedContentType string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body scrubRequestBody
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &body)
		observedContentType = body.ContentType
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(scrubResponseBody{Scrubbed: body.Payload})
	}))
	defer srv.Close()
	t.Setenv(EnvScrubberURL, srv.URL)
	t.Setenv(EnvScrubberToken, "")
	s := NewPresidioScrubber()
	_, _ = s.Scrub([]byte(`{"key":"value"}`))
	require.Equal(t, "application/json", observedContentType)
}

func TestPresidioScrubberHealthCheck(t *testing.T) {
	srv := fakePresidioServer(t, "")
	defer srv.Close()
	t.Setenv(EnvScrubberURL, srv.URL)
	t.Setenv(EnvScrubberToken, "")
	s := NewPresidioScrubber()
	require.NoError(t, s.HealthCheck(context.Background()))
}

func TestPresidioScrubberPropagatesNon2xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()
	t.Setenv(EnvScrubberURL, srv.URL)
	t.Setenv(EnvScrubberToken, "")
	s := NewPresidioScrubber(WithFailClosed())
	out, report := s.Scrub([]byte("payload"))
	require.Nil(t, out)
	require.Len(t, report.Rewrites, 1)
}

func TestDefaultScrubberUsesPresidioWhenURLSet(t *testing.T) {
	t.Setenv(EnvScrubberURL, "http://nowhere.invalid")
	d := New()
	_, ok := d.scrubber.(*PresidioScrubber)
	require.True(t, ok, "expected PresidioScrubber when URL configured")
}

func TestDefaultScrubberFallsBackToRegexWhenURLUnset(t *testing.T) {
	t.Setenv(EnvScrubberURL, "")
	d := New()
	_, ok := d.scrubber.(*RegexScrubber)
	require.True(t, ok, "expected RegexScrubber when URL unset")
}
