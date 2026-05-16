// Package client is the CLI's HTTP client for the Phase-1 control plane.
//
// Phase 2 swaps the JSON-over-HTTP transport for the connect-go client once
// the generated stubs land. The function signatures here are stable; the
// transport implementation changes underneath.
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	cruciblev1 "github.com/crucible/sdk-go/crucible/v1"
)

// Client is the CLI's view of the control plane.
type Client struct {
	endpoint string
	http     *http.Client
	tenant   string
}

// New constructs a client. endpoint defaults to http://localhost:8080;
// tenant defaults to "single-tenant".
func New(endpoint, tenant string) *Client {
	if endpoint == "" {
		endpoint = "http://localhost:8080"
	}
	if tenant == "" {
		tenant = "single-tenant"
	}
	return &Client{
		endpoint: endpoint,
		http:     &http.Client{Timeout: 60 * time.Second},
		tenant:   tenant,
	}
}

// Endpoint returns the configured base URL.
func (c *Client) Endpoint() string { return c.endpoint }

// Health pings /healthz and returns the parsed response.
func (c *Client) Health(ctx context.Context) (map[string]any, error) {
	var out map[string]any
	if err := c.do(ctx, http.MethodGet, "/healthz", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// SubmitTaskRequest mirrors the api.submitTaskRequest shape.
type SubmitTaskRequest struct {
	Description        string  `json:"description"`
	Repo               string  `json:"repo,omitempty"`
	BaseSha            string  `json:"base_sha,omitempty"`
	TenantID           string  `json:"tenant_id,omitempty"`
	CostCapUSD         float64 `json:"cost_cap_usd,omitempty"`
	WallClockCapMin    uint32  `json:"wall_clock_cap_min,omitempty"`
	RetryCapPerSubgoal uint32  `json:"retry_cap_per_subgoal,omitempty"`
}

// SubmitTask creates a task.
func (c *Client) SubmitTask(ctx context.Context, req SubmitTaskRequest) (*cruciblev1.Task, error) {
	if req.TenantID == "" {
		req.TenantID = c.tenant
	}
	var resp struct{ Task *cruciblev1.Task `json:"task"` }
	if err := c.do(ctx, http.MethodPost, "/v1/tasks", req, &resp); err != nil {
		return nil, err
	}
	return resp.Task, nil
}

// GetTask returns one task.
func (c *Client) GetTask(ctx context.Context, id string) (*cruciblev1.Task, error) {
	var resp struct{ Task *cruciblev1.Task `json:"task"` }
	if err := c.do(ctx, http.MethodGet, "/v1/tasks/"+url.PathEscape(id), nil, &resp); err != nil {
		return nil, err
	}
	return resp.Task, nil
}

// ListTasks returns the most recent tasks for the configured tenant.
func (c *Client) ListTasks(ctx context.Context, limit int) ([]*cruciblev1.Task, error) {
	q := url.Values{}
	q.Set("tenant_id", c.tenant)
	if limit > 0 {
		q.Set("limit", strconv.Itoa(limit))
	}
	var resp struct{ Tasks []*cruciblev1.Task `json:"tasks"` }
	if err := c.do(ctx, http.MethodGet, "/v1/tasks?"+q.Encode(), nil, &resp); err != nil {
		return nil, err
	}
	return resp.Tasks, nil
}

// ApprovePlanRequest is the body for POST /v1/tasks/{id}/approve.
type ApprovePlanRequest struct {
	PlanHash             string  `json:"plan_hash,omitempty"`
	ApproverOidcSubject  string  `json:"approver_oidc_subject,omitempty"`
	CostCapUSD           float64 `json:"cost_cap_usd,omitempty"`
	WallClockCapMin      uint32  `json:"wall_clock_cap_min,omitempty"`
	RetryCapPerSubgoal   uint32  `json:"retry_cap_per_subgoal,omitempty"`
}

// ApprovePlan approves a plan.
func (c *Client) ApprovePlan(ctx context.Context, id string, req ApprovePlanRequest) (*cruciblev1.Task, *cruciblev1.PlanApproval, error) {
	var resp struct {
		Task     *cruciblev1.Task         `json:"task"`
		Approval *cruciblev1.PlanApproval `json:"approval"`
	}
	if err := c.do(ctx, http.MethodPost, "/v1/tasks/"+url.PathEscape(id)+"/approve", req, &resp); err != nil {
		return nil, nil, err
	}
	return resp.Task, resp.Approval, nil
}

// RejectPlanRequest is the body for POST /v1/tasks/{id}/reject.
type RejectPlanRequest struct {
	PlanHash             string `json:"plan_hash,omitempty"`
	RejecterOidcSubject  string `json:"rejecter_oidc_subject,omitempty"`
	Reason               string `json:"reason"`
}

// RejectPlan rejects a plan.
func (c *Client) RejectPlan(ctx context.Context, id string, req RejectPlanRequest) (*cruciblev1.Task, *cruciblev1.PlanRejection, error) {
	var resp struct {
		Task      *cruciblev1.Task          `json:"task"`
		Rejection *cruciblev1.PlanRejection `json:"rejection"`
	}
	if err := c.do(ctx, http.MethodPost, "/v1/tasks/"+url.PathEscape(id)+"/reject", req, &resp); err != nil {
		return nil, nil, err
	}
	return resp.Task, resp.Rejection, nil
}

// GetBudget fetches the live Budget snapshot.
func (c *Client) GetBudget(ctx context.Context, id string) (*cruciblev1.Budget, error) {
	var resp struct{ Budget *cruciblev1.Budget `json:"budget"` }
	if err := c.do(ctx, http.MethodGet, "/v1/tasks/"+url.PathEscape(id)+"/budget", nil, &resp); err != nil {
		return nil, err
	}
	return resp.Budget, nil
}

func (c *Client) do(ctx context.Context, method, path string, body, out any) error {
	var rdr io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("client: marshal: %w", err)
		}
		rdr = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.endpoint+path, rdr)
	if err != nil {
		return fmt.Errorf("client: build request: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "crucible-cli/2026.06.0-phase1")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("client: do request: %w (is the control plane running at %s ?)", err, c.endpoint)
	}
	defer resp.Body.Close()
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("client: read response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// Try to surface the structured error message.
		var apiErr struct {
			Error   string `json:"error"`
			Message string `json:"message"`
		}
		if json.Unmarshal(bodyBytes, &apiErr) == nil && apiErr.Message != "" {
			return fmt.Errorf("server returned %d: %s", resp.StatusCode, apiErr.Message)
		}
		return fmt.Errorf("server returned %d: %s", resp.StatusCode, string(bodyBytes))
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(bodyBytes, out); err != nil {
		return fmt.Errorf("client: decode response: %w", err)
	}
	return nil
}

// SubmitDescribe is a convenience for the smoke test: submits, waits briefly
// for the Plan to be present, returns the final task.
func (c *Client) SubmitDescribe(ctx context.Context, description, repo, baseSHA string) (*cruciblev1.Task, error) {
	task, err := c.SubmitTask(ctx, SubmitTaskRequest{
		Description: description,
		Repo:        repo,
		BaseSha:     baseSHA,
	})
	if err != nil {
		return nil, err
	}
	if task == nil {
		return nil, errors.New("server returned no task body")
	}
	return task, nil
}
