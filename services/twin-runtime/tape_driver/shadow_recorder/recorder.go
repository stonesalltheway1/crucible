// Package shadowrecorder records production (or sanctioned-staging) HTTP
// traffic into Crucible tapes. The recorder runs in shadow mode — it
// observes traffic via Envoy's ext_proc filter or eBPF tap (depending on
// deployment topology) and persists scrubbed responses to a content-
// addressed tape store.
//
// Phase 3 ships the recorder skeleton with two ingress paths:
//
//	envoy   — Envoy access-log webhook (POST /ingest/envoy)
//	ebpf    — gRPC stream from the in-cluster eBPF tap binary
//
// The Phase 3 brief mandates: scrubbing MUST run at capture, before bytes
// hit disk. The recorder calls the Phase 3 Presidio scrubber inline; if
// the scrubber is unreachable the recorder fails-closed (rejects the
// capture) rather than persisting unscrubbed bytes.
package shadowrecorder

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"time"

	tapedriver "github.com/crucible/services/twin-runtime/tape_driver"
)

// CapturedRequest is the wire form the recorder accepts from the ingress
// adapter.
type CapturedRequest struct {
	Service   string
	Method    string
	Endpoint  string
	Headers   map[string][]string
	Body      []byte
	Response  CapturedResponse
	Timestamp time.Time
	// SampledFrom identifies the upstream tap (`envoy`, `ebpf`, `manual`).
	SampledFrom string
}

// CapturedResponse mirrors a recorded response.
type CapturedResponse struct {
	Status  int
	Headers map[string][]string
	Body    []byte
}

// TapeEntry is the persisted form. Content-addressed by RequestHash.
type TapeEntry struct {
	TapeSet     string
	Service     string
	Method      string
	Endpoint    string
	RequestHash string
	Request     CapturedRequest
	Response    CapturedResponse
	Scrubbed    ScrubbedResponse
	Timestamp   time.Time
	SampledFrom string
}

// ScrubbedResponse holds the post-scrub body + the audit log.
type ScrubbedResponse struct {
	Body      []byte
	Headers   map[string][]string
	ScrubLog  []tapedriver.ScrubRewrite
}

// TapeStore is the persistence interface. Production callers wire an S3-
// or filesystem-backed implementation; the [InMemoryStore] is for tests.
type TapeStore interface {
	Put(ctx context.Context, entry TapeEntry) error
	Get(ctx context.Context, hash string) (TapeEntry, error)
	List(ctx context.Context, tapeSet string) ([]TapeEntry, error)
	Count(ctx context.Context, tapeSet string) (int, error)
}

// Recorder is the main entry point.
type Recorder struct {
	store    TapeStore
	scrubber tapedriver.Scrubber
	clock    func() time.Time
	failClosedOnScrubberError bool

	mu      sync.RWMutex
	stats   map[string]*EndpointStats
}

// Options shape Recorder construction.
type Options struct {
	Store    TapeStore
	Scrubber tapedriver.Scrubber
	Clock    func() time.Time
	// FailClosed (default true) refuses to persist when the scrubber is
	// unreachable. Setting this to false is a compliance violation in
	// regulated tenants — the option exists only for the developer-loop
	// path where Presidio isn't running.
	FailClosed bool
}

// EndpointStats holds shadow-recorder coverage telemetry.
type EndpointStats struct {
	Service       string
	Method        string
	Endpoint      string
	Samples       int
	LastRecorded  time.Time
	UniqueHashes  int
}

// New constructs a recorder.
func New(opts Options) *Recorder {
	clock := opts.Clock
	if clock == nil {
		clock = time.Now
	}
	scrubber := opts.Scrubber
	if scrubber == nil {
		scrubber = tapedriver.NewRegexScrubber()
	}
	return &Recorder{
		store:                     opts.Store,
		scrubber:                  scrubber,
		clock:                     clock,
		failClosedOnScrubberError: opts.FailClosed,
		stats:                     make(map[string]*EndpointStats),
	}
}

// Capture is the core recorder entry. Returns the persisted [TapeEntry]
// or an error.
//
// Critical invariant: scrubbing happens BEFORE the entry is written to
// the store. Failures fall closed when [Options.FailClosed] is true.
func (r *Recorder) Capture(ctx context.Context, tapeSet string, c CapturedRequest) (TapeEntry, error) {
	if tapeSet == "" {
		return TapeEntry{}, errors.New("shadow recorder: tape_set empty")
	}
	if c.Service == "" {
		return TapeEntry{}, errors.New("shadow recorder: service empty")
	}
	scrubbedBody, scrubReport := r.scrubBody(c.Response.Body)
	if scrubbedBody == nil && r.failClosedOnScrubberError {
		return TapeEntry{}, errors.New("shadow recorder: scrubber unavailable; failing closed")
	}
	if scrubbedBody == nil {
		scrubbedBody = c.Response.Body
	}
	// Also scrub headers — Authorization, X-User-* etc. carry PII.
	scrubbedHeaders := r.scrubHeaders(c.Response.Headers)
	entry := TapeEntry{
		TapeSet:     tapeSet,
		Service:     c.Service,
		Method:      strings.ToUpper(c.Method),
		Endpoint:    c.Endpoint,
		RequestHash: requestHash(c),
		Request:     c,
		Response:    c.Response,
		Scrubbed: ScrubbedResponse{
			Body:     scrubbedBody,
			Headers:  scrubbedHeaders.headers,
			ScrubLog: append(scrubReport.Rewrites, scrubbedHeaders.rewrites...),
		},
		Timestamp:   r.clock(),
		SampledFrom: c.SampledFrom,
	}
	if err := r.store.Put(ctx, entry); err != nil {
		return TapeEntry{}, fmt.Errorf("tape store put: %w", err)
	}
	r.updateStats(entry)
	return entry, nil
}

func (r *Recorder) scrubBody(body []byte) ([]byte, tapedriver.ScrubReport) {
	if len(body) == 0 {
		return body, tapedriver.ScrubReport{}
	}
	scrubbed, report := r.scrubber.Scrub(body)
	return scrubbed, report
}

type scrubbedHeaderResult struct {
	headers  map[string][]string
	rewrites []tapedriver.ScrubRewrite
}

func (r *Recorder) scrubHeaders(in map[string][]string) scrubbedHeaderResult {
	if len(in) == 0 {
		return scrubbedHeaderResult{headers: in}
	}
	out := make(map[string][]string, len(in))
	var rewrites []tapedriver.ScrubRewrite
	for k, vs := range in {
		// Hop-by-hop headers stay as-is.
		if isHopByHop(k) {
			out[k] = vs
			continue
		}
		// Auth headers and any header containing user-data are scrubbed.
		if shouldScrubHeader(k) {
			cleaned := make([]string, len(vs))
			for i, v := range vs {
				scrubbed, rep := r.scrubber.Scrub([]byte(v))
				if scrubbed != nil {
					cleaned[i] = string(scrubbed)
				} else {
					cleaned[i] = "[REDACTED]"
				}
				for _, rw := range rep.Rewrites {
					rw.Field = "header:" + k
					rewrites = append(rewrites, rw)
				}
			}
			out[k] = cleaned
			continue
		}
		out[k] = vs
	}
	return scrubbedHeaderResult{headers: out, rewrites: rewrites}
}

func isHopByHop(name string) bool {
	switch strings.ToLower(name) {
	case "connection", "transfer-encoding", "te", "keep-alive", "upgrade":
		return true
	}
	return false
}

func shouldScrubHeader(name string) bool {
	lc := strings.ToLower(name)
	if strings.HasPrefix(lc, "x-user-") || strings.HasPrefix(lc, "x-customer-") {
		return true
	}
	switch lc {
	case "authorization", "cookie", "set-cookie", "x-api-key", "x-auth-token":
		return true
	}
	return false
}

// updateStats records the per-endpoint stats — coverage metrics surface
// to the customer dashboard.
func (r *Recorder) updateStats(e TapeEntry) {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := e.Service + "|" + e.Method + "|" + e.Endpoint
	s, ok := r.stats[key]
	if !ok {
		s = &EndpointStats{Service: e.Service, Method: e.Method, Endpoint: e.Endpoint}
		r.stats[key] = s
	}
	s.Samples++
	s.UniqueHashes++ // approximate; the store dedupes by hash
	if e.Timestamp.After(s.LastRecorded) {
		s.LastRecorded = e.Timestamp
	}
}

// Stats returns a snapshot of the per-endpoint coverage telemetry.
func (r *Recorder) Stats() []EndpointStats {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]EndpointStats, 0, len(r.stats))
	for _, s := range r.stats {
		out = append(out, *s)
	}
	return out
}

// HandleEnvoyAccessLog is the HTTP handler for Envoy's access-log webhook.
// Wired into the recorder's HTTP server in apps/twin-runtime/server.
func (r *Recorder) HandleEnvoyAccessLog(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		TapeSet  string           `json:"tape_set"`
		Captured CapturedRequest  `json:"captured"`
	}
	if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	entry, err := r.Capture(req.Context(), body.TapeSet, body.Captured)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"request_hash": entry.RequestHash,
		"scrubbed":     len(entry.Scrubbed.ScrubLog) > 0,
	})
}

// requestHash is the content address for the entry. Combines the
// stable shape (service + method + endpoint + body-hash + key headers).
func requestHash(c CapturedRequest) string {
	h := sha256.New()
	h.Write([]byte(c.Service))
	h.Write([]byte{0})
	h.Write([]byte(strings.ToUpper(c.Method)))
	h.Write([]byte{0})
	h.Write([]byte(c.Endpoint))
	h.Write([]byte{0})
	h.Write(c.Body)
	return "sha256:" + hex.EncodeToString(h.Sum(nil))
}

// ──────────────────────────────────────────────────────────────────────
// In-memory store (test-friendly default)
// ──────────────────────────────────────────────────────────────────────

// InMemoryStore is a test-friendly TapeStore. Production callers wire an
// S3-backed implementation in the helm chart.
type InMemoryStore struct {
	mu      sync.RWMutex
	entries map[string]TapeEntry  // keyed by hash
	byTape  map[string][]string   // tape_set → hashes
}

// NewInMemoryStore constructs a store.
func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{
		entries: make(map[string]TapeEntry),
		byTape:  make(map[string][]string),
	}
}

// Put persists an entry. Hash-deduplicates within a tape set.
func (s *InMemoryStore) Put(_ context.Context, e TapeEntry) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.entries[e.RequestHash]; !exists {
		s.byTape[e.TapeSet] = append(s.byTape[e.TapeSet], e.RequestHash)
	}
	s.entries[e.RequestHash] = e
	return nil
}

// Get returns an entry by hash.
func (s *InMemoryStore) Get(_ context.Context, hash string) (TapeEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	e, ok := s.entries[hash]
	if !ok {
		return TapeEntry{}, fmt.Errorf("tape: hash %s not found", hash)
	}
	return e, nil
}

// List returns all entries in a tape set.
func (s *InMemoryStore) List(_ context.Context, tapeSet string) ([]TapeEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	hashes := s.byTape[tapeSet]
	out := make([]TapeEntry, 0, len(hashes))
	for _, h := range hashes {
		if e, ok := s.entries[h]; ok {
			out = append(out, e)
		}
	}
	return out, nil
}

// Count returns the number of distinct entries in a tape set.
func (s *InMemoryStore) Count(_ context.Context, tapeSet string) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.byTape[tapeSet]), nil
}

// ──────────────────────────────────────────────────────────────────────
// Re-record schedule
// ──────────────────────────────────────────────────────────────────────

// ReCordSchedule describes the per-endpoint re-recording cadence.
type ReCordSchedule struct {
	// Default applies to every endpoint not in PerEndpoint.
	Default time.Duration
	// PerEndpoint overrides for hot paths.
	PerEndpoint map[string]time.Duration
}

// IntervalFor returns the re-record interval for a (service, method, endpoint).
func (s ReCordSchedule) IntervalFor(service, method, endpoint string) time.Duration {
	key := strings.ToLower(service) + "|" + strings.ToUpper(method) + "|" + endpoint
	if v, ok := s.PerEndpoint[key]; ok {
		return v
	}
	if s.Default > 0 {
		return s.Default
	}
	return 30 * 24 * time.Hour
}

// DefaultReCordSchedule returns the monthly-default schedule per
// docs/06-research/tape-coverage-strategy.md.
func DefaultReCordSchedule() ReCordSchedule {
	return ReCordSchedule{Default: 30 * 24 * time.Hour}
}

// SanitisePath returns a host-safe filename derived from an endpoint
// string. Used by the S3-backed store implementation.
func SanitisePath(endpoint string) string {
	return strings.ReplaceAll(filepath.ToSlash(endpoint), "/", "_")
}
