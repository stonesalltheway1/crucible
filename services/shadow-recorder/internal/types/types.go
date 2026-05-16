// Package types defines the wire types for shadow-recorder.
package types

import "time"

// EnvoyAccessLogEntry mirrors the Envoy ALSv3 HTTP record we ingest.
// We accept either Envoy's gRPC ALS (when wired via the gRPC sink) or
// JSON access logs (the default for HTTP ingestion).
type EnvoyAccessLogEntry struct {
	TenantID         string            `json:"tenant_id"`
	UpstreamHost     string            `json:"upstream_host"`
	RequestMethod    string            `json:"request_method"`
	RequestPath      string            `json:"request_path"`
	RequestHeaders   map[string]string `json:"request_headers,omitempty"`
	RequestBody      []byte            `json:"request_body,omitempty"`
	ResponseStatus   int               `json:"response_status"`
	ResponseHeaders  map[string]string `json:"response_headers,omitempty"`
	ResponseBody     []byte            `json:"response_body,omitempty"`
	StartTime        time.Time         `json:"start_time"`
	DurationMillis   int               `json:"duration_millis"`
}

// EBPFTapEntry is the eBPF-tap variant; same shape, different ingest.
type EBPFTapEntry = EnvoyAccessLogEntry

// TapeEntry is what we persist after scrubbing.
type TapeEntry struct {
	TenantID       string            `json:"tenant_id"`
	UpstreamHost   string            `json:"upstream_host"`
	Method         string            `json:"method"`
	Path           string            `json:"path"`
	RequestSig     string            `json:"request_sig"`
	RequestHeaders map[string]string `json:"request_headers,omitempty"`
	RequestBody    []byte            `json:"request_body,omitempty"`
	ResponseStatus int               `json:"response_status"`
	ResponseHeaders map[string]string `json:"response_headers,omitempty"`
	ResponseBody   []byte            `json:"response_body,omitempty"`
	CapturedAt     time.Time         `json:"captured_at"`
	ScrubAuditID   string            `json:"scrub_audit_id"`
}

// EndpointStat is per-endpoint metadata.
type EndpointStat struct {
	TenantID       string    `json:"tenant_id"`
	Host           string    `json:"host"`
	Method         string    `json:"method"`
	PathTemplate   string    `json:"path_template"`
	HitCount       int       `json:"hit_count"`
	LastRecordedAt time.Time `json:"last_recorded_at"`
	NextRecordDue  time.Time `json:"next_record_due"`
}

// HostCoverage rolls EndpointStats up by host.
type HostCoverage struct {
	Host         string         `json:"host"`
	Endpoints    int            `json:"endpoints"`
	TotalHits    int            `json:"total_hits"`
	OldestRecord time.Time      `json:"oldest_record"`
	NewestRecord time.Time      `json:"newest_record"`
	Stats        []EndpointStat `json:"stats,omitempty"`
}
