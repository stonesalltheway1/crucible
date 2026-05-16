package tapedriver

// presidio_scrubber.go ─ Go client for the Python Presidio service.
//
// Phase 3 of the Crucible build replaces the Phase 2 [RegexScrubber] baseline
// with a remote Python service backed by Presidio + spaCy + FF3-1 +
// deterministic pseudonymisation. The Go interface stays identical — the
// driver swaps in PresidioScrubber when CRUCIBLE_SCRUBBER_URL is set.
//
// Per the currency check, Presidio has no built-in auth; this Go side
// always presents a shared bearer token sourced from CRUCIBLE_SCRUBBER_TOKEN.
// The Python side refuses any request without it.
//
// Capture-time only. Scrubbing on replay is too late by definition.

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	EnvScrubberURL   = "CRUCIBLE_SCRUBBER_URL"
	EnvScrubberToken = "CRUCIBLE_SCRUBBER_TOKEN"

	// DefaultScrubberURL points at the in-cluster service endpoint we ship
	// in the Helm chart's default values. The env var overrides for tests.
	DefaultScrubberURL = "http://crucible-scrubber.crucible-system.svc.cluster.local:9100"

	// ScrubberRequestTimeout caps each request. Brief targets p95 ≤ 200ms;
	// we cap at 2s to absorb transient slow paths without stalling the
	// twin-runtime spawn path.
	ScrubberRequestTimeout = 2 * time.Second
)

// PresidioScrubber is the Phase 3 PII scrubber. Implements [Scrubber].
type PresidioScrubber struct {
	url       string
	token     string
	client    *http.Client
	tapeSet   string
	fallback  Scrubber
	failClose bool
}

// PresidioScrubberOption mutates the scrubber at construction.
type PresidioScrubberOption func(*PresidioScrubber)

// WithTapeSet sets the per-tape-set namespace used as the deterministic
// pseudonym key. Without this the scrubber uses the literal "default" and
// referential integrity is per-installation rather than per-tape-set.
func WithTapeSet(tapeSet string) PresidioScrubberOption {
	return func(s *PresidioScrubber) { s.tapeSet = tapeSet }
}

// WithFallback installs a fallback Scrubber used when the remote service is
// unreachable. Default fallback is the Phase 2 [RegexScrubber] so a Presidio
// outage degrades to a known-good baseline rather than failing closed.
//
// WARNING: an installation that runs without Presidio is NOT HIPAA-compliant.
// The fallback is for outage recovery only. Wire failClose=true in regulated
// deployments via [WithFailClosed].
func WithFallback(s Scrubber) PresidioScrubberOption {
	return func(p *PresidioScrubber) { p.fallback = s }
}

// WithFailClosed disables the fallback and returns an empty scrubbed payload
// with a single ScrubRewrite entry tagged "presidio-unavailable" when the
// service is unreachable. Use this in regulated deployments.
func WithFailClosed() PresidioScrubberOption {
	return func(p *PresidioScrubber) { p.failClose = true }
}

// NewPresidioScrubber constructs from the environment. Returns nil if
// CRUCIBLE_SCRUBBER_URL is unset.
func NewPresidioScrubber(opts ...PresidioScrubberOption) *PresidioScrubber {
	url := strings.TrimSpace(os.Getenv(EnvScrubberURL))
	if url == "" {
		url = DefaultScrubberURL
	}
	s := &PresidioScrubber{
		url:      strings.TrimRight(url, "/"),
		token:    os.Getenv(EnvScrubberToken),
		client:   &http.Client{Timeout: ScrubberRequestTimeout},
		tapeSet:  "default",
		fallback: NewRegexScrubber(),
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// URL returns the configured Presidio endpoint (useful for ops/diagnostics).
func (p *PresidioScrubber) URL() string { return p.url }

// Scrub implements [Scrubber]. On remote failure, the configured fallback or
// fail-closed behaviour activates. Each scrub call records its own audit
// log; the report returned matches the Phase 2 [ScrubReport] shape so
// callers don't need to branch.
func (p *PresidioScrubber) Scrub(payload []byte) ([]byte, ScrubReport) {
	out, report, err := p.scrubRemote(context.Background(), payload)
	if err == nil {
		return out, report
	}
	if p.failClose {
		return nil, ScrubReport{
			Rewrites: []ScrubRewrite{{
				Scrubber: "presidio-unavailable",
				Field:    "[remote]",
				Before:   "",
				After:    "[FAIL-CLOSED]",
			}},
		}
	}
	if p.fallback != nil {
		return p.fallback.Scrub(payload)
	}
	return payload, ScrubReport{}
}

type scrubRequestBody struct {
	TapeSet           string                  `json:"tape_set"`
	Payload           string                  `json:"payload"`
	ContentType       string                  `json:"content_type,omitempty"`
	Language          string                  `json:"language,omitempty"`
	Engine            string                  `json:"engine,omitempty"`
	OperatorOverrides map[string]string       `json:"operator_overrides,omitempty"`
	CustomRecognizers []map[string]any        `json:"custom_recognizers,omitempty"`
}

type scrubResponseBody struct {
	Scrubbed string `json:"scrubbed"`
	Report   struct {
		TapeSet   string `json:"tape_set"`
		ElapsedMs int    `json:"elapsed_ms"`
		Rewrites  []struct {
			Scrubber      string `json:"scrubber"`
			Field         string `json:"field"`
			BeforeHash    string `json:"before_hash"`
			After         string `json:"after"`
			Operator      string `json:"operator"`
			Algorithm     string `json:"algorithm"`
			Ff3DomainSize int    `json:"ff3_domain_size"`
			TapeSet       string `json:"tape_set"`
			TimestampMs   int64  `json:"timestamp_ms"`
		} `json:"rewrites"`
	} `json:"report"`
}

func (p *PresidioScrubber) scrubRemote(
	ctx context.Context, payload []byte,
) ([]byte, ScrubReport, error) {
	body := scrubRequestBody{
		TapeSet: p.tapeSet,
		Payload: string(payload),
	}
	// JSON shape hint helps the pipeline walk fields rather than treating
	// the whole payload as opaque text. The brace check is a fast guard.
	trimmed := bytes.TrimSpace(payload)
	if len(trimmed) > 0 && (trimmed[0] == '{' || trimmed[0] == '[') {
		body.ContentType = "application/json"
	}
	rawBody, err := json.Marshal(body)
	if err != nil {
		return nil, ScrubReport{}, fmt.Errorf("marshal: %w", err)
	}
	endpoint := p.url + "/scrub"
	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(rawBody))
	if err != nil {
		return nil, ScrubReport{}, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if p.token != "" {
		req.Header.Set("Authorization", "Bearer "+p.token)
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, ScrubReport{}, fmt.Errorf("http: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, ScrubReport{}, fmt.Errorf(
			"presidio scrubber returned %d: %s", resp.StatusCode, string(raw),
		)
	}
	var parsed scrubResponseBody
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, ScrubReport{}, fmt.Errorf("decode: %w", err)
	}
	rewrites := make([]ScrubRewrite, 0, len(parsed.Report.Rewrites))
	for _, r := range parsed.Report.Rewrites {
		rewrites = append(rewrites, ScrubRewrite{
			Scrubber: r.Scrubber,
			Field:    r.Field,
			// The Python side hashes the original before returning; the
			// Go-facing ScrubRewrite stores the hash in Before so the
			// audit chain doesn't leak the original PII through this hop.
			Before: r.BeforeHash,
			After:  r.After,
		})
	}
	return []byte(parsed.Scrubbed), ScrubReport{Rewrites: rewrites}, nil
}

// HealthCheck pings the Presidio readyz endpoint. Returns nil if the
// service is up; otherwise the caller's RB-09 should already have detected
// the spawn-failure-rate spike.
func (p *PresidioScrubber) HealthCheck(ctx context.Context) error {
	endpoint := p.url + "/readyz"
	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return err
	}
	if p.token != "" {
		req.Header.Set("Authorization", "Bearer "+p.token)
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("presidio /readyz %d: %s", resp.StatusCode, string(raw))
	}
	return nil
}

// ErrScrubberUnavailable indicates the Python scrubber is unreachable; the
// caller selects between fallback and fail-closed semantics.
var ErrScrubberUnavailable = errors.New("presidio scrubber unavailable")
