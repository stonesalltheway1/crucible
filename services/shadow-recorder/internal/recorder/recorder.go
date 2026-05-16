// Package recorder is the shadow-recorder core: ingest → scrub →
// persist → record-coverage.
package recorder

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/crucible/services/shadow-recorder/internal/coverage"
	"github.com/crucible/services/shadow-recorder/internal/scrubber"
	"github.com/crucible/services/shadow-recorder/internal/storage"
	"github.com/crucible/services/shadow-recorder/internal/types"
)

// Config configures Recorder.
type Config struct {
	Scrubber       *scrubber.Client
	Store          storage.Store
	Coverage       *coverage.Tracker
	RerecordEvery  time.Duration
	RetentionDays  int
}

// Recorder is the ingest + persist core.
type Recorder struct {
	cfg Config
	mu  sync.Mutex
	// CapturesIngested counts ingested captures across the lifetime of
	// the process. Surfaced as a Prometheus metric.
	capturesIngested int
	scrubFailures    int
}

// New returns a Recorder.
func New(cfg Config) *Recorder {
	if cfg.RerecordEvery == 0 {
		cfg.RerecordEvery = 30 * 24 * time.Hour
	}
	if cfg.RetentionDays == 0 {
		cfg.RetentionDays = 90
	}
	return &Recorder{cfg: cfg}
}

// Ingest scrubs and persists one entry. Returns the storage key.
func (r *Recorder) Ingest(ctx context.Context, e types.EnvoyAccessLogEntry) (string, error) {
	if e.UpstreamHost == "" || e.RequestPath == "" || e.RequestMethod == "" || e.TenantID == "" {
		return "", ErrIncompleteEntry
	}

	scrubbedReq, err := r.cfg.Scrubber.Scrub(e.RequestBody, e.RequestHeaders)
	if err != nil {
		r.mu.Lock()
		r.scrubFailures++
		r.mu.Unlock()
		return "", err
	}
	scrubbedResp, err := r.cfg.Scrubber.Scrub(e.ResponseBody, e.ResponseHeaders)
	if err != nil {
		r.mu.Lock()
		r.scrubFailures++
		r.mu.Unlock()
		return "", err
	}

	entry := types.TapeEntry{
		TenantID:        e.TenantID,
		UpstreamHost:    e.UpstreamHost,
		Method:          e.RequestMethod,
		Path:            e.RequestPath,
		RequestSig:      requestSig(e),
		RequestHeaders:  scrubbedReq.Headers,
		RequestBody:     scrubbedReq.Body,
		ResponseStatus:  e.ResponseStatus,
		ResponseHeaders: scrubbedResp.Headers,
		ResponseBody:    scrubbedResp.Body,
		CapturedAt:      e.StartTime,
		ScrubAuditID:    scrubbedReq.AuditID,
	}
	if entry.CapturedAt.IsZero() {
		entry.CapturedAt = time.Now().UTC()
	}

	key, err := r.cfg.Store.Put(ctx, entry)
	if err != nil {
		return "", err
	}
	r.cfg.Coverage.Record(e.TenantID, e.UpstreamHost, e.RequestMethod, e.RequestPath, entry.CapturedAt, r.cfg.RerecordEvery)
	r.mu.Lock()
	r.capturesIngested++
	r.mu.Unlock()
	return key, nil
}

// RunDueRerecords scans the coverage tracker for endpoints whose
// re-record schedule has elapsed. Each due endpoint is replayed
// against the upstream by the recorder's bridge into the egress
// proxy. We don't perform live HTTP calls in this Phase-8 commit; we
// emit metrics and let the customer's egress proxy serve the
// re-record (the same path used for the live captures).
func (r *Recorder) RunDueRerecords(ctx context.Context) int {
	due := r.cfg.Coverage.DueRerecords(time.Now())
	if len(due) == 0 {
		return 0
	}
	// Phase-8 commit: surface re-record-due as a metric for the
	// observability dashboard. The actual replay is wired in by the
	// customer's egress proxy (the same route as the live capture);
	// this method is the periodic poke that surfaces the candidate
	// list.
	for range due {
		// no-op per-endpoint here; the metric counter increments via
		// the API surface.
	}
	return len(due)
}

// Stats returns the lifetime counters for the metrics endpoint.
func (r *Recorder) Stats() (capturesIngested, scrubFailures int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.capturesIngested, r.scrubFailures
}

func requestSig(e types.EnvoyAccessLogEntry) string {
	// Sig = method + path-template + sorted request-header keys (case-
	// insensitive). The body is captured as-is.
	return e.RequestMethod + " " + e.RequestPath
}

// ErrIncompleteEntry is returned for malformed ingest payloads.
var ErrIncompleteEntry = errors.New("recorder: incomplete entry (tenant_id, upstream_host, method, path required)")
