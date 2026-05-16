// Package scrubber wraps the Phase-3 scrubber HTTP service.
//
// Production deployments REQUIRE the scrubber to be reachable. The
// FailClosed flag matches docs/04-operations/self-hosted-install.md
// `tapeScrubber.failClosed` value: regulated tenants must run with
// failClosed=true so a scrubber outage cannot leak unscrubbed bytes
// to disk.
package scrubber

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Config configures the client.
type Config struct {
	Endpoint   string // empty disables scrubbing (dev-only)
	Token      string
	FailClosed bool
	Timeout    time.Duration
}

// Client is the scrubber HTTP client.
type Client struct {
	cfg  Config
	http *http.Client
}

// NewClient builds a Client.
func NewClient(cfg Config) *Client {
	to := cfg.Timeout
	if to == 0 {
		to = 5 * time.Second
	}
	return &Client{cfg: cfg, http: &http.Client{Timeout: to}}
}

// ScrubResult mirrors the Phase-3 service response.
type ScrubResult struct {
	Body         []byte            `json:"body"`
	Headers      map[string]string `json:"headers,omitempty"`
	AuditID      string            `json:"audit_id"`
	ScrubbersFired []string        `json:"scrubbers_fired,omitempty"`
}

// ErrScrubberUnavailable is returned when the scrubber is not reachable
// and FailClosed is set.
var ErrScrubberUnavailable = errors.New("scrubber: unavailable and fail-closed")

// Scrub sends the body through the scrubber.
//
// Behaviour:
//   - Endpoint == "" + FailClosed=false: dev-mode passthrough.
//   - Endpoint == "" + FailClosed=true:  return ErrScrubberUnavailable.
//   - Endpoint set + reachable:           return scrubbed body.
//   - Endpoint set + unreachable + FailClosed=true: return error.
//   - Endpoint set + unreachable + FailClosed=false: return original body
//     (logged; not safe for production).
func (c *Client) Scrub(body []byte, headers map[string]string) (*ScrubResult, error) {
	if c.cfg.Endpoint == "" {
		if c.cfg.FailClosed {
			return nil, ErrScrubberUnavailable
		}
		return &ScrubResult{Body: body, Headers: headers, AuditID: "dev-passthrough"}, nil
	}
	req := map[string]any{
		"body":    string(body),
		"headers": headers,
	}
	buf, _ := json.Marshal(req)
	httpReq, _ := http.NewRequest(http.MethodPost, c.cfg.Endpoint+"/scrub", bytes.NewReader(buf))
	httpReq.Header.Set("Content-Type", "application/json")
	if c.cfg.Token != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.cfg.Token)
	}
	resp, err := c.http.Do(httpReq)
	if err != nil {
		if c.cfg.FailClosed {
			return nil, fmt.Errorf("scrubber unreachable (fail-closed): %w", err)
		}
		return &ScrubResult{Body: body, Headers: headers, AuditID: "fail-soft"}, nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		bb, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		if c.cfg.FailClosed {
			return nil, fmt.Errorf("scrubber HTTP %d (fail-closed): %s", resp.StatusCode, string(bb))
		}
		return &ScrubResult{Body: body, Headers: headers, AuditID: "fail-soft"}, nil
	}
	var res ScrubResult
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}
	return &res, nil
}
