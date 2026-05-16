package dbdriver

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

const (
	// EnvNeonAPIKey is the CRUCIBLE_-namespaced env var carrying the Neon
	// API token. Per the memory note we never reuse the user's existing
	// EpsteinExposed project key for Crucible work.
	EnvNeonAPIKey = "CRUCIBLE_NEON_API_KEY"

	// EnvNeonBaseURL overrides the API base (used by tests).
	EnvNeonBaseURL = "CRUCIBLE_NEON_BASE_URL"

	// EnvNeonProjectID is the Crucible-dedicated Neon project. Tenant
	// isolation requires one Neon project per tenant per the May 2026
	// currency check — so in multi-tenant deployments this is per-tenant.
	EnvNeonProjectID = "CRUCIBLE_NEON_PROJECT_ID"

	// DefaultNeonBaseURL — the post-rebrand canonical host. The legacy
	// console.neon.tech still works (308 redirect).
	DefaultNeonBaseURL = "https://console.neon.tech/api/v2"
)

// NeonDriver is the Neon-specific Driver impl.
type NeonDriver struct {
	apiKey  string
	baseURL string
	client  *http.Client
	caps    Capabilities
}

// NewNeonDriver constructs from env. When CRUCIBLE_NEON_API_KEY is unset,
// the returned driver returns *StubError from every method.
func NewNeonDriver() Driver {
	apiKey := os.Getenv(EnvNeonAPIKey)
	if apiKey == "" {
		return newStubDriver(EnginePostgresNeon, fmt.Sprintf(
			"%s unset — driver in stub mode. Set the env var pointing at a Crucible-dedicated Neon project (NEVER reuse another project's token).",
			EnvNeonAPIKey,
		))
	}
	baseURL := os.Getenv(EnvNeonBaseURL)
	if baseURL == "" {
		baseURL = DefaultNeonBaseURL
	}
	return &NeonDriver{
		apiKey:  apiKey,
		baseURL: strings.TrimRight(baseURL, "/"),
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		caps: Capabilities{
			InstantBranch:            true,
			ScaleToZero:              true,
			FirstPartySchemaDiff:     true,
			MaxConcurrentBranches:    0,    // no published hard cap on paid plans
			PerTenantProjectRequired: true, // no branch-prefix RBAC
		},
	}
}

// Engine returns EnginePostgresNeon.
func (d *NeonDriver) Engine() Engine { return EnginePostgresNeon }

// Capabilities returns the Neon feature matrix.
func (d *NeonDriver) Capabilities() Capabilities { return d.caps }

type neonBranchCreateRequest struct {
	Branch    neonBranch       `json:"branch"`
	Endpoints []neonEndpointIn `json:"endpoints"`
}

type neonBranch struct {
	ParentID        string    `json:"parent_id,omitempty"`
	ParentLSN       string    `json:"parent_lsn,omitempty"`
	ParentTimestamp time.Time `json:"parent_timestamp,omitempty"`
	Name            string    `json:"name,omitempty"`
	Protected       bool      `json:"protected,omitempty"`
}

type neonEndpointIn struct {
	Type string `json:"type"`
}

type neonBranchCreateResponse struct {
	Branch struct {
		ID            string `json:"id"`
		ProjectID     string `json:"project_id"`
		ParentID      string `json:"parent_id"`
		ParentLSN     string `json:"parent_lsn"`
		Name          string `json:"name"`
		CurrentState  string `json:"current_state"`
		LogicalSize   int    `json:"logical_size"`
		CreationSource string `json:"creation_source"`
		CreatedAt     time.Time `json:"created_at"`
	} `json:"branch"`
	Endpoints []struct {
		Host     string `json:"host"`
		ID       string `json:"id"`
		BranchID string `json:"branch_id"`
		Type     string `json:"type"`
	} `json:"endpoints"`
	Operations []struct {
		ID     string `json:"id"`
		Action string `json:"action"`
		Status string `json:"status"`
	} `json:"operations"`
}

type neonOperation struct {
	ID     string `json:"id"`
	Action string `json:"action"`
	Status string `json:"status"`
}

type neonOperationResponse struct {
	Operation neonOperation `json:"operation"`
}

type neonConnectionURIResponse struct {
	URI string `json:"uri"`
}

// CreateBranch mints a new Neon branch and waits for it to become usable.
//
// On Neon, POST returns immediately with current_state="init" and a pending
// operations[] array; the connection URI is fetched from a separate endpoint
// once the create_branch op reports `finished`. We poll the operations
// endpoint for up to opts.Timeout (default 10s).
func (d *NeonDriver) CreateBranch(ctx context.Context, spec BranchSpec, opts CreateBranchOpts) (Branch, error) {
	if opts.Timeout == 0 {
		opts.Timeout = 10 * time.Second
	}
	if opts.PollInterval == 0 {
		opts.PollInterval = 250 * time.Millisecond
	}
	projectID := strings.TrimSpace(spec.ProjectID)
	if projectID == "" {
		projectID = os.Getenv(EnvNeonProjectID)
	}
	if projectID == "" {
		return Branch{}, errors.New("ProjectID empty and CRUCIBLE_NEON_PROJECT_ID unset")
	}
	body := neonBranchCreateRequest{
		Branch: neonBranch{
			ParentID:        spec.BaseBranchID,
			ParentLSN:       spec.ParentLSN,
			ParentTimestamp: spec.ParentTimestamp,
			Name:            spec.Name,
			Protected:       spec.Protected,
		},
		Endpoints: []neonEndpointIn{{Type: "read_write"}},
	}
	endpoint := fmt.Sprintf("%s/projects/%s/branches", d.baseURL, url.PathEscape(projectID))
	var resp neonBranchCreateResponse
	if err := d.do(ctx, "POST", endpoint, body, &resp); err != nil {
		return Branch{}, fmt.Errorf("create branch: %w", err)
	}
	// Wait for the create_branch op to finish.
	createOpID := ""
	for _, op := range resp.Operations {
		if op.Action == "create_branch" {
			createOpID = op.ID
			break
		}
	}
	if createOpID != "" {
		if err := d.waitForOp(ctx, projectID, createOpID, opts); err != nil {
			return Branch{}, err
		}
	}
	// Fetch the connection URI; the endpoint returns the full URI once
	// the branch is ready.
	uri, err := d.connectionURI(ctx, projectID, resp.Branch.ID)
	if err != nil {
		return Branch{}, err
	}
	host := ""
	if len(resp.Endpoints) > 0 {
		host = resp.Endpoints[0].Host
	}
	return Branch{
		ID:            resp.Branch.ID,
		ProjectID:     resp.Branch.ProjectID,
		Name:          resp.Branch.Name,
		Host:          host,
		ConnectionURI: uri,
		State:         "ready",
		CreatedAt:     resp.Branch.CreatedAt,
		Metadata:      spec.Tags,
	}, nil
}

func (d *NeonDriver) waitForOp(
	ctx context.Context,
	projectID, opID string,
	opts CreateBranchOpts,
) error {
	deadline := time.Now().Add(opts.Timeout)
	for {
		if time.Now().After(deadline) {
			return fmt.Errorf("create_branch op %s did not finish within %s", opID, opts.Timeout)
		}
		endpoint := fmt.Sprintf("%s/projects/%s/operations/%s",
			d.baseURL, url.PathEscape(projectID), url.PathEscape(opID))
		var resp neonOperationResponse
		if err := d.do(ctx, "GET", endpoint, nil, &resp); err != nil {
			// Transient — retry until deadline.
			time.Sleep(opts.PollInterval)
			continue
		}
		switch resp.Operation.Status {
		case "finished":
			return nil
		case "failed", "error":
			return fmt.Errorf("create_branch op %s failed", opID)
		default:
			time.Sleep(opts.PollInterval)
		}
	}
}

func (d *NeonDriver) connectionURI(ctx context.Context, projectID, branchID string) (string, error) {
	endpoint := fmt.Sprintf("%s/projects/%s/branches/%s/connection_uri",
		d.baseURL, url.PathEscape(projectID), url.PathEscape(branchID))
	var resp neonConnectionURIResponse
	if err := d.do(ctx, "GET", endpoint, nil, &resp); err != nil {
		return "", fmt.Errorf("connection_uri: %w", err)
	}
	if resp.URI == "" {
		return "", errors.New("connection_uri response empty")
	}
	return resp.URI, nil
}

// DeleteBranch is async on Neon; we issue DELETE and treat 404 as "already
// gone" (idempotent).
func (d *NeonDriver) DeleteBranch(ctx context.Context, branchID string) error {
	projectID := os.Getenv(EnvNeonProjectID)
	if projectID == "" {
		return errors.New("CRUCIBLE_NEON_PROJECT_ID unset")
	}
	endpoint := fmt.Sprintf("%s/projects/%s/branches/%s",
		d.baseURL, url.PathEscape(projectID), url.PathEscape(branchID))
	err := d.do(ctx, "DELETE", endpoint, nil, nil)
	if err == nil {
		return nil
	}
	var httpErr *HTTPError
	if errors.As(err, &httpErr) && httpErr.Status == http.StatusNotFound {
		return nil // idempotent
	}
	return err
}

// SchemaDiff uses the Neon first-party compare_schema endpoint.
func (d *NeonDriver) SchemaDiff(ctx context.Context, base, target, db string) (SchemaDiffResult, error) {
	projectID := os.Getenv(EnvNeonProjectID)
	if projectID == "" {
		return SchemaDiffResult{}, errors.New("CRUCIBLE_NEON_PROJECT_ID unset")
	}
	endpoint := fmt.Sprintf(
		"%s/projects/%s/branches/%s/compare_schema?base_branch_id=%s&db_name=%s",
		d.baseURL,
		url.PathEscape(projectID),
		url.PathEscape(target),
		url.QueryEscape(base),
		url.QueryEscape(db),
	)
	var resp struct {
		SQL string `json:"sql"`
	}
	if err := d.do(ctx, "GET", endpoint, nil, &resp); err != nil {
		return SchemaDiffResult{}, fmt.Errorf("compare_schema: %w", err)
	}
	return SchemaDiffResult{
		DDL:            resp.SQL,
		HasDestructive: hasDestructiveDDL(resp.SQL),
	}, nil
}

// ListBranches enumerates branches for the project.
func (d *NeonDriver) ListBranches(ctx context.Context, projectID string) ([]Branch, error) {
	if projectID == "" {
		projectID = os.Getenv(EnvNeonProjectID)
	}
	endpoint := fmt.Sprintf("%s/projects/%s/branches",
		d.baseURL, url.PathEscape(projectID))
	var resp struct {
		Branches []struct {
			ID         string    `json:"id"`
			ProjectID  string    `json:"project_id"`
			Name       string    `json:"name"`
			CreatedAt  time.Time `json:"created_at"`
			Endpoints  []struct {
				Host string `json:"host"`
			} `json:"endpoints"`
		} `json:"branches"`
	}
	if err := d.do(ctx, "GET", endpoint, nil, &resp); err != nil {
		return nil, fmt.Errorf("list branches: %w", err)
	}
	out := make([]Branch, 0, len(resp.Branches))
	for _, b := range resp.Branches {
		host := ""
		if len(b.Endpoints) > 0 {
			host = b.Endpoints[0].Host
		}
		out = append(out, Branch{
			ID:        b.ID,
			ProjectID: b.ProjectID,
			Name:      b.Name,
			Host:      host,
			CreatedAt: b.CreatedAt,
		})
	}
	return out, nil
}

// do is the low-level request helper.
func (d *NeonDriver) do(ctx context.Context, method, url_ string, body any, out any) error {
	var reqBody io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal request: %w", err)
		}
		reqBody = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, url_, reqBody)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+d.apiKey)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("http: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return &HTTPError{Status: resp.StatusCode, Body: string(raw), URL: url_}
	}
	if out != nil {
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
	}
	return nil
}

// HTTPError is returned for non-2xx Neon responses.
type HTTPError struct {
	Status int
	Body   string
	URL    string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("neon http %d on %s: %s", e.Status, e.URL, e.Body)
}

// hasDestructiveDDL returns true if the SQL contains DROP TABLE / DROP COLUMN
// / TRUNCATE / DELETE-without-WHERE.
func hasDestructiveDDL(sql string) bool {
	upper := strings.ToUpper(sql)
	if strings.Contains(upper, "DROP TABLE") ||
		strings.Contains(upper, "DROP DATABASE") ||
		strings.Contains(upper, "DROP SCHEMA") ||
		strings.Contains(upper, "TRUNCATE") {
		return true
	}
	if strings.Contains(upper, "DROP COLUMN") {
		return true
	}
	// DELETE without WHERE is destructive — but DELETE WHERE is fine.
	if strings.Contains(upper, "DELETE FROM ") && !strings.Contains(upper, "WHERE ") {
		return true
	}
	return false
}
