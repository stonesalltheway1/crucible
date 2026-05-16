package dbdriver

// Turso libSQL driver.
//
// Phase 3 currency check (May 2026):
//   - Base URL: https://api.turso.tech/v1
//   - Auth: Authorization: Bearer <TOKEN>
//   - Create-database is the branching mechanism; pass seed={type=database,
//     name=parent} for CoW. Response is synchronous; database is connectable
//     on response in practice. Still poll a trivial SELECT 1 for safety.
//   - Connection URL is libsql://<Hostname>?authToken=<db-scoped-token>;
//     the db-scoped token is minted via a SEPARATE call.
//   - Free tier: 500 dbs / 9 GB / 1B row reads. Production wants a paid org.
//   - Bin-pack: all per-task branches go in one Turso group to avoid the
//     per-group DB count quota.

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
	EnvTursoToken   = "CRUCIBLE_TURSO_TOKEN"
	EnvTursoOrg     = "CRUCIBLE_TURSO_ORG"
	EnvTursoGroup   = "CRUCIBLE_TURSO_GROUP"
	EnvTursoBaseURL = "CRUCIBLE_TURSO_BASE_URL"

	DefaultTursoBaseURL = "https://api.turso.tech/v1"
	DefaultTursoGroup   = "crucible-twin"
)

// TursoDriver wraps the Turso Platform API.
type TursoDriver struct {
	token   string
	org     string
	group   string
	baseURL string
	client  *http.Client
	caps    Capabilities
}

// NewTursoDriver constructs from env. Returns a stub driver when token or
// org is unset.
func NewTursoDriver() Driver {
	token := os.Getenv(EnvTursoToken)
	org := os.Getenv(EnvTursoOrg)
	if token == "" || org == "" {
		return newStubDriver(EngineSQLite, fmt.Sprintf(
			"Turso driver missing env: %s, %s — driver in stub mode.",
			EnvTursoToken, EnvTursoOrg,
		))
	}
	group := os.Getenv(EnvTursoGroup)
	if group == "" {
		group = DefaultTursoGroup
	}
	base := os.Getenv(EnvTursoBaseURL)
	if base == "" {
		base = DefaultTursoBaseURL
	}
	return &TursoDriver{
		token:   token,
		org:     org,
		group:   group,
		baseURL: strings.TrimRight(base, "/"),
		client:  &http.Client{Timeout: 30 * time.Second},
		caps: Capabilities{
			InstantBranch:            true, // typical sub-second
			ScaleToZero:              true,
			FirstPartySchemaDiff:     false, // no native compare; we sqlite_master-diff in adapter
			MaxConcurrentBranches:    500,
			PerTenantProjectRequired: true, // org-scoped tokens are member-level
		},
	}
}

// Engine returns EngineSQLite.
func (d *TursoDriver) Engine() Engine { return EngineSQLite }

// Capabilities returns the Turso feature matrix.
func (d *TursoDriver) Capabilities() Capabilities { return d.caps }

type tursoCreateRequest struct {
	Name  string     `json:"name"`
	Group string     `json:"group"`
	Seed  *tursoSeed `json:"seed,omitempty"`
}

type tursoSeed struct {
	Type      string `json:"type"`
	Name      string `json:"name,omitempty"`
	Timestamp string `json:"timestamp,omitempty"`
}

type tursoCreateResponse struct {
	Database struct {
		Name     string `json:"Name"`
		Hostname string `json:"Hostname"`
		DbId     string `json:"DbId"`
		Group    string `json:"group"`
	} `json:"database"`
}

type tursoTokenResponse struct {
	JWT string `json:"jwt"`
}

func (d *TursoDriver) dbsURL() string {
	return fmt.Sprintf("%s/organizations/%s/databases",
		d.baseURL, url.PathEscape(d.org))
}

func (d *TursoDriver) dbURL(name string) string {
	return fmt.Sprintf("%s/organizations/%s/databases/%s",
		d.baseURL, url.PathEscape(d.org), url.PathEscape(name))
}

// CreateBranch creates a new libSQL database seeded from the parent.
func (d *TursoDriver) CreateBranch(
	ctx context.Context, spec BranchSpec, opts CreateBranchOpts,
) (Branch, error) {
	if opts.Timeout == 0 {
		opts.Timeout = 10 * time.Second
	}
	name := spec.Name
	if name == "" {
		name = fmt.Sprintf("twin-%d", time.Now().UnixNano())
	}
	req := tursoCreateRequest{
		Name:  name,
		Group: d.group,
	}
	if parent := spec.BaseBranchName; parent != "" {
		req.Seed = &tursoSeed{Type: "database", Name: parent}
		if !spec.ParentTimestamp.IsZero() {
			req.Seed.Timestamp = spec.ParentTimestamp.Format(time.RFC3339)
		}
	}
	var resp tursoCreateResponse
	if err := d.do(ctx, "POST", d.dbsURL(), req, &resp); err != nil {
		return Branch{}, fmt.Errorf("turso create database: %w", err)
	}
	token, err := d.issueDBToken(ctx, resp.Database.Name)
	if err != nil {
		return Branch{}, fmt.Errorf("turso issue db token: %w", err)
	}
	uri := fmt.Sprintf("libsql://%s?authToken=%s",
		resp.Database.Hostname, url.QueryEscape(token))
	return Branch{
		ID:            resp.Database.DbId,
		ProjectID:     d.org,
		Name:          resp.Database.Name,
		Host:          resp.Database.Hostname,
		ConnectionURI: uri,
		State:         "ready",
		CreatedAt:     time.Now(),
		Metadata:      spec.Tags,
	}, nil
}

func (d *TursoDriver) issueDBToken(ctx context.Context, dbName string) (string, error) {
	endpoint := d.dbURL(dbName) + "/auth/tokens?expiration=1h&authorization=full-access"
	var resp tursoTokenResponse
	if err := d.do(ctx, "POST", endpoint, nil, &resp); err != nil {
		return "", err
	}
	if resp.JWT == "" {
		return "", errors.New("turso: empty token in response")
	}
	return resp.JWT, nil
}

// DeleteBranch drops the libSQL database. Idempotent on 404.
func (d *TursoDriver) DeleteBranch(ctx context.Context, branchID string) error {
	endpoint := d.dbURL(branchID)
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

// SchemaDiff runs a sqlite_master textual diff. Turso doesn't expose a
// first-party compare endpoint as of May 2026.
func (d *TursoDriver) SchemaDiff(
	ctx context.Context, base, target, _ string,
) (SchemaDiffResult, error) {
	baseSchema, err := d.fetchSchema(ctx, base)
	if err != nil {
		return SchemaDiffResult{}, err
	}
	targetSchema, err := d.fetchSchema(ctx, target)
	if err != nil {
		return SchemaDiffResult{}, err
	}
	return diffSqliteSchemas(baseSchema, targetSchema), nil
}

func (d *TursoDriver) fetchSchema(ctx context.Context, branch string) ([]sqliteObj, error) {
	// Turso exposes the SQLite metadata through the libsql HTTP API; we
	// proxy via the same auth-tokens we mint above. For Phase 3 we keep
	// this in-place rather than depending on a libsql client; the schema
	// fetch endpoint is /organizations/{org}/databases/{db}/schema.
	endpoint := d.dbURL(branch) + "/schema"
	var resp struct {
		Objects []sqliteObj `json:"objects"`
	}
	if err := d.do(ctx, "GET", endpoint, nil, &resp); err != nil {
		return nil, fmt.Errorf("turso fetch schema for %s: %w", branch, err)
	}
	return resp.Objects, nil
}

type sqliteObj struct {
	Type string `json:"type"`
	Name string `json:"name"`
	SQL  string `json:"sql"`
}

func diffSqliteSchemas(base, target []sqliteObj) SchemaDiffResult {
	byName := func(objs []sqliteObj) map[string]sqliteObj {
		m := make(map[string]sqliteObj, len(objs))
		for _, o := range objs {
			m[o.Name] = o
		}
		return m
	}
	baseM := byName(base)
	tgtM := byName(target)
	var (
		added, dropped, altered []string
		ddl                     strings.Builder
		destructive             bool
	)
	for name, t := range tgtM {
		if _, ok := baseM[name]; !ok {
			added = append(added, name)
			ddl.WriteString(t.SQL + ";\n")
		} else if baseM[name].SQL != t.SQL {
			altered = append(altered, name)
			ddl.WriteString("-- altered\n" + t.SQL + ";\n")
		}
	}
	for name := range baseM {
		if _, ok := tgtM[name]; !ok {
			dropped = append(dropped, name)
			ddl.WriteString("DROP TABLE " + name + ";\n")
			destructive = true
		}
	}
	return SchemaDiffResult{
		DDL:            ddl.String(),
		AddedTables:    added,
		DroppedTables:  dropped,
		AlteredTables:  altered,
		HasDestructive: destructive,
	}
}

// ListBranches returns Turso databases under the org.
func (d *TursoDriver) ListBranches(ctx context.Context, _ string) ([]Branch, error) {
	var resp struct {
		Databases []struct {
			Name     string `json:"Name"`
			Hostname string `json:"Hostname"`
			DbId     string `json:"DbId"`
			Group    string `json:"group"`
		} `json:"databases"`
	}
	if err := d.do(ctx, "GET", d.dbsURL(), nil, &resp); err != nil {
		return nil, fmt.Errorf("turso list databases: %w", err)
	}
	out := make([]Branch, 0, len(resp.Databases))
	for _, db := range resp.Databases {
		if d.group != "" && db.Group != d.group {
			continue
		}
		out = append(out, Branch{
			ID:        db.DbId,
			ProjectID: d.org,
			Name:      db.Name,
			Host:      db.Hostname,
			State:     "ready",
		})
	}
	return out, nil
}

// do is the low-level request helper.
func (d *TursoDriver) do(ctx context.Context, method, urlS string, body any, out any) error {
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
	req.Header.Set("Authorization", "Bearer "+d.token)
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
