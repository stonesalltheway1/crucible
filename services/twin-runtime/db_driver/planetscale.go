package dbdriver

// PlanetScale MySQL driver.
//
// Phase 3 currency check (May 2026):
//   - Base URL: https://api.planetscale.com/v1
//   - Auth header shape:  Authorization: <SERVICE_TOKEN_ID>:<SERVICE_TOKEN>
//     (a colon — NOT Bearer; PlanetScale's documented exception).
//   - Create-branch is async; response carries ready=false. Poll the
//     GET endpoint for ready=true.
//   - Postgres branching GA'd 2025-09 but uses restore-from-backup, not
//     Neon-style CoW. We default to MySQL (Vitess) here; the Postgres
//     surface is reachable via the Crucible `kind: postgres-planetscale`
//     engine alias (Phase 4+ work).
//   - Connection credentials come from a SEPARATE call:
//       POST .../branches/{name}/passwords
//     Returns per-call host + username/password (rotate-friendly).
//   - Recursive delete shipped 2026-03-27 — we exercise it on cleanup so
//     orphan child-branches from forked tasks don't linger.

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
	EnvPlanetScaleTokenID = "CRUCIBLE_PLANETSCALE_TOKEN_ID"
	EnvPlanetScaleToken   = "CRUCIBLE_PLANETSCALE_TOKEN"
	EnvPlanetScaleOrg     = "CRUCIBLE_PLANETSCALE_ORG"
	EnvPlanetScaleDB      = "CRUCIBLE_PLANETSCALE_DB"
	EnvPlanetScaleBaseURL = "CRUCIBLE_PLANETSCALE_BASE_URL"

	DefaultPlanetScaleBaseURL = "https://api.planetscale.com/v1"
)

// PlanetScaleDriver wraps the PlanetScale REST API.
type PlanetScaleDriver struct {
	tokenID string
	token   string
	org     string
	db      string
	baseURL string
	client  *http.Client
	caps    Capabilities
}

// NewPlanetScaleDriver constructs from env. Returns a stub driver when any
// required env var is unset.
func NewPlanetScaleDriver() Driver {
	tokenID := os.Getenv(EnvPlanetScaleTokenID)
	token := os.Getenv(EnvPlanetScaleToken)
	org := os.Getenv(EnvPlanetScaleOrg)
	db := os.Getenv(EnvPlanetScaleDB)
	missing := []string{}
	if tokenID == "" {
		missing = append(missing, EnvPlanetScaleTokenID)
	}
	if token == "" {
		missing = append(missing, EnvPlanetScaleToken)
	}
	if org == "" {
		missing = append(missing, EnvPlanetScaleOrg)
	}
	if db == "" {
		missing = append(missing, EnvPlanetScaleDB)
	}
	if len(missing) > 0 {
		return newStubDriver(EngineMySQL, fmt.Sprintf(
			"PlanetScale driver missing env: %s — driver in stub mode. "+
				"NEVER reuse another project's token.",
			strings.Join(missing, ", "),
		))
	}
	base := os.Getenv(EnvPlanetScaleBaseURL)
	if base == "" {
		base = DefaultPlanetScaleBaseURL
	}
	return &PlanetScaleDriver{
		tokenID: tokenID,
		token:   token,
		org:     org,
		db:      db,
		baseURL: strings.TrimRight(base, "/"),
		client:  &http.Client{Timeout: 30 * time.Second},
		caps: Capabilities{
			// MySQL branching is CoW-ish (Vitess workspace clone). New branches
			// over a small schema reach `ready` in 1–3s; large schemas trend
			// up but are within the Phase 3 ≤2s p95 budget for the typical
			// case. Postgres branching is restore-from-backup and outside
			// this budget; surface a separate engine for it later.
			InstantBranch:            true,
			ScaleToZero:              false,
			FirstPartySchemaDiff:     true,
			MaxConcurrentBranches:    0,
			PerTenantProjectRequired: true,
		},
	}
}

// Engine returns EngineMySQL.
func (d *PlanetScaleDriver) Engine() Engine { return EngineMySQL }

// Capabilities returns the PlanetScale feature matrix.
func (d *PlanetScaleDriver) Capabilities() Capabilities { return d.caps }

type psBranchCreateRequest struct {
	Name         string `json:"name"`
	ParentBranch string `json:"parent_branch,omitempty"`
}

type psBranchResponse struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	ParentBranch string `json:"parent_branch"`
	Ready        bool   `json:"ready"`
	HTMLURL      string `json:"html_url"`
	MysqlAddress string `json:"mysql_address"`
	CreatedAt    string `json:"created_at"`
}

type psPasswordRequest struct {
	Name string `json:"name"`
	Role string `json:"role"`
	TTL  int    `json:"ttl,omitempty"`
}

type psPasswordResponse struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	PublicID    string `json:"public_id"`
	PlainText   string `json:"plain_text"`
	Username    string `json:"username"`
	Host        string `json:"access_host_url"`
	DatabaseName string `json:"database_name"`
}

func (d *PlanetScaleDriver) branchURL(branch string) string {
	return fmt.Sprintf(
		"%s/organizations/%s/databases/%s/branches/%s",
		d.baseURL,
		url.PathEscape(d.org),
		url.PathEscape(d.db),
		url.PathEscape(branch),
	)
}

func (d *PlanetScaleDriver) branchesURL() string {
	return fmt.Sprintf(
		"%s/organizations/%s/databases/%s/branches",
		d.baseURL,
		url.PathEscape(d.org),
		url.PathEscape(d.db),
	)
}

// CreateBranch mints a new MySQL branch and waits for ready=true.
//
// On PlanetScale, POST returns immediately with ready=false; we poll the GET
// endpoint at opts.PollInterval (default 250ms) for up to opts.Timeout
// (default 10s). Once ready, we mint a per-branch password pair so the
// returned ConnectionURI is immediately usable.
func (d *PlanetScaleDriver) CreateBranch(
	ctx context.Context, spec BranchSpec, opts CreateBranchOpts,
) (Branch, error) {
	if opts.Timeout == 0 {
		opts.Timeout = 10 * time.Second
	}
	if opts.PollInterval == 0 {
		opts.PollInterval = 250 * time.Millisecond
	}
	name := spec.Name
	if name == "" {
		name = fmt.Sprintf("twin-%d", time.Now().UnixNano())
	}
	parent := spec.BaseBranchName
	if parent == "" {
		parent = "main"
	}
	body := psBranchCreateRequest{Name: name, ParentBranch: parent}
	var created psBranchResponse
	if err := d.do(ctx, "POST", d.branchesURL(), body, &created); err != nil {
		return Branch{}, fmt.Errorf("planetscale create branch: %w", err)
	}
	if err := d.waitForReady(ctx, created.Name, opts); err != nil {
		return Branch{}, err
	}
	pw, err := d.issuePassword(ctx, created.Name, "reader+writer")
	if err != nil {
		return Branch{}, fmt.Errorf("planetscale issue password: %w", err)
	}
	uri := fmt.Sprintf(
		"mysql://%s:%s@%s/%s?ssl-mode=VERIFY_IDENTITY",
		url.QueryEscape(pw.Username),
		url.QueryEscape(pw.PlainText),
		pw.Host,
		url.PathEscape(d.db),
	)
	return Branch{
		ID:            created.ID,
		ProjectID:     d.org + "/" + d.db,
		Name:          created.Name,
		Host:          pw.Host,
		ConnectionURI: uri,
		State:         "ready",
		CreatedAt:     time.Now(),
		RolePassword:  pw.PlainText,
		Metadata:      spec.Tags,
	}, nil
}

func (d *PlanetScaleDriver) waitForReady(
	ctx context.Context, branch string, opts CreateBranchOpts,
) error {
	deadline := time.Now().Add(opts.Timeout)
	for {
		if time.Now().After(deadline) {
			return fmt.Errorf("planetscale branch %s not ready within %s", branch, opts.Timeout)
		}
		var b psBranchResponse
		if err := d.do(ctx, "GET", d.branchURL(branch), nil, &b); err == nil && b.Ready {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(opts.PollInterval):
		}
	}
}

func (d *PlanetScaleDriver) issuePassword(
	ctx context.Context, branch, role string,
) (*psPasswordResponse, error) {
	endpoint := d.branchURL(branch) + "/passwords"
	body := psPasswordRequest{
		Name: fmt.Sprintf("crucible-%d", time.Now().UnixNano()),
		Role: role,
		TTL:  3600,
	}
	var pw psPasswordResponse
	if err := d.do(ctx, "POST", endpoint, body, &pw); err != nil {
		return nil, err
	}
	if pw.PlainText == "" {
		return nil, errors.New("planetscale: empty password in response")
	}
	return &pw, nil
}

// DeleteBranch removes the branch (recursive). 2026-03-27 added the
// recursive flag; we always pass it so child branches from fan-out
// exploration get cleaned up too.
func (d *PlanetScaleDriver) DeleteBranch(ctx context.Context, branchID string) error {
	endpoint := d.branchURL(branchID) + "?recursive=true"
	err := d.do(ctx, "DELETE", endpoint, nil, nil)
	if err == nil {
		return nil
	}
	var httpErr *HTTPError
	if errors.As(err, &httpErr) && httpErr.Status == http.StatusNotFound {
		return nil
	}
	return err
}

// SchemaDiff uses the PlanetScale schema-diff endpoint
// (`GET /branches/{branch}/diff`). Returns the DDL between base and target.
func (d *PlanetScaleDriver) SchemaDiff(
	ctx context.Context, base, target, db string,
) (SchemaDiffResult, error) {
	endpoint := fmt.Sprintf(
		"%s/diff?source_branch=%s",
		d.branchURL(target),
		url.QueryEscape(base),
	)
	var resp struct {
		Diff []struct {
			Raw string `json:"raw"`
		} `json:"diff"`
	}
	if err := d.do(ctx, "GET", endpoint, nil, &resp); err != nil {
		return SchemaDiffResult{}, fmt.Errorf("planetscale schema diff: %w", err)
	}
	var sb strings.Builder
	for _, e := range resp.Diff {
		sb.WriteString(e.Raw)
		sb.WriteString("\n")
	}
	ddl := sb.String()
	return SchemaDiffResult{
		DDL:            ddl,
		HasDestructive: hasDestructiveDDL(ddl),
	}, nil
}

// ListBranches returns the project's branches.
func (d *PlanetScaleDriver) ListBranches(ctx context.Context, _ string) ([]Branch, error) {
	var resp struct {
		Data []psBranchResponse `json:"data"`
	}
	if err := d.do(ctx, "GET", d.branchesURL(), nil, &resp); err != nil {
		return nil, fmt.Errorf("planetscale list branches: %w", err)
	}
	out := make([]Branch, 0, len(resp.Data))
	for _, b := range resp.Data {
		out = append(out, Branch{
			ID:        b.ID,
			ProjectID: d.org + "/" + d.db,
			Name:      b.Name,
			Host:      b.MysqlAddress,
			State:     readyState(b.Ready),
		})
	}
	return out, nil
}

func readyState(ready bool) string {
	if ready {
		return "ready"
	}
	return "initializing"
}

// do is the low-level request helper.
func (d *PlanetScaleDriver) do(ctx context.Context, method, urlS string, body any, out any) error {
	var reqBody io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal request: %w", err)
		}
		reqBody = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, urlS, reqBody)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	// PlanetScale auth: id:token (colon), NOT Bearer.
	req.Header.Set("Authorization", d.tokenID+":"+d.token)
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
		return &HTTPError{Status: resp.StatusCode, Body: string(raw), URL: urlS}
	}
	if out != nil {
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
	}
	return nil
}
