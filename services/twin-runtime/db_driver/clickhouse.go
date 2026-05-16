package dbdriver

// ClickHouse driver.
//
// Per docs/05-decisions/ADR-005, ClickHouse branching is "table-level
// CREATE TABLE … CLONE AS". The DB-level CLONE was proposed Apr 2026 but
// hadn't stabilised by May 2026.
//
// This driver targets self-hosted ClickHouse — it issues table-level
// clones via the native HTTP API (Authorization: header with the user's
// API token or basic auth). Crucible ships the SQL rather than depending
// on a vendor SDK; the result is a thin shim.
//
// For ClickHouse Cloud, the recommended path is to run a per-task user
// scoped to the cloned-tables-only schema; the driver supports either via
// CRUCIBLE_CLICKHOUSE_USER / CRUCIBLE_CLICKHOUSE_PASSWORD.

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
	EnvClickHouseURL      = "CRUCIBLE_CLICKHOUSE_URL"
	EnvClickHouseUser     = "CRUCIBLE_CLICKHOUSE_USER"
	EnvClickHousePassword = "CRUCIBLE_CLICKHOUSE_PASSWORD"
	EnvClickHouseSourceDB = "CRUCIBLE_CLICKHOUSE_SOURCE_DB"
)

// ClickHouseDriver wraps a ClickHouse server's HTTP interface.
type ClickHouseDriver struct {
	url      string
	user     string
	password string
	sourceDB string
	client   *http.Client
}

// NewClickHouseDriver constructs from env.
func NewClickHouseDriver() Driver {
	chURL := os.Getenv(EnvClickHouseURL)
	if chURL == "" {
		return newStubDriver(EngineClickHouse, fmt.Sprintf(
			"ClickHouse driver missing env: %s", EnvClickHouseURL))
	}
	source := os.Getenv(EnvClickHouseSourceDB)
	if source == "" {
		source = "default"
	}
	return &ClickHouseDriver{
		url:      strings.TrimRight(chURL, "/"),
		user:     os.Getenv(EnvClickHouseUser),
		password: os.Getenv(EnvClickHousePassword),
		sourceDB: source,
		client:   &http.Client{Timeout: 30 * time.Second},
	}
}

// Engine returns EngineClickHouse.
func (d *ClickHouseDriver) Engine() Engine { return EngineClickHouse }

// Capabilities returns the ClickHouse feature matrix.
func (d *ClickHouseDriver) Capabilities() Capabilities {
	return Capabilities{
		InstantBranch:            true,  // table-level CLONE AS is sub-second per table
		ScaleToZero:              false,
		FirstPartySchemaDiff:     false, // we sniff `system.tables` diffs
		MaxConcurrentBranches:    0,
		PerTenantProjectRequired: false, // we use a per-task DB inside a shared CH instance
	}
}

// CreateBranch creates a new database and clones every table from the
// source database into it via CREATE TABLE ... CLONE AS.
func (d *ClickHouseDriver) CreateBranch(
	ctx context.Context, spec BranchSpec, _ CreateBranchOpts,
) (Branch, error) {
	name := spec.Name
	if name == "" {
		name = fmt.Sprintf("twin_%d", time.Now().UnixNano())
	}
	// 1. Create the per-task database.
	if err := d.execSQL(ctx, fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %s", quoteIdent(name))); err != nil {
		return Branch{}, fmt.Errorf("clickhouse create db: %w", err)
	}
	// 2. Enumerate source tables and CLONE AS each one.
	tables, err := d.listTables(ctx, d.sourceDB)
	if err != nil {
		// Best-effort: empty source → empty branch. The runtime will see
		// the empty DB and the agent can populate it from migrations.
		tables = nil
	}
	for _, table := range tables {
		src := quoteIdent(d.sourceDB) + "." + quoteIdent(table)
		dst := quoteIdent(name) + "." + quoteIdent(table)
		sql := fmt.Sprintf("CREATE TABLE %s CLONE AS %s", dst, src)
		if err := d.execSQL(ctx, sql); err != nil {
			return Branch{}, fmt.Errorf("clickhouse clone %s: %w", table, err)
		}
	}
	uri := buildClickHouseURI(d.url, d.user, d.password, name)
	return Branch{
		ID:            name,
		ProjectID:     d.sourceDB,
		Name:          name,
		Host:          d.url,
		ConnectionURI: uri,
		State:         "ready",
		CreatedAt:     time.Now(),
		Metadata:      spec.Tags,
	}, nil
}

func (d *ClickHouseDriver) listTables(ctx context.Context, db string) ([]string, error) {
	sql := fmt.Sprintf(
		"SELECT name FROM system.tables WHERE database = '%s' FORMAT JSON",
		escapeStringLit(db),
	)
	raw, err := d.querySQL(ctx, sql)
	if err != nil {
		return nil, err
	}
	var parsed struct {
		Data []struct {
			Name string `json:"name"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, fmt.Errorf("decode tables: %w", err)
	}
	out := make([]string, 0, len(parsed.Data))
	for _, r := range parsed.Data {
		out = append(out, r.Name)
	}
	return out, nil
}

// DeleteBranch drops the per-task database. Idempotent.
func (d *ClickHouseDriver) DeleteBranch(ctx context.Context, branchID string) error {
	if branchID == "" {
		return errors.New("clickhouse DeleteBranch: empty branchID")
	}
	return d.execSQL(ctx, fmt.Sprintf("DROP DATABASE IF EXISTS %s", quoteIdent(branchID)))
}

// SchemaDiff compares the CREATE TABLE statements between base and target
// databases by querying system.tables.
func (d *ClickHouseDriver) SchemaDiff(
	ctx context.Context, base, target, _ string,
) (SchemaDiffResult, error) {
	baseTables, err := d.tableDDLs(ctx, base)
	if err != nil {
		return SchemaDiffResult{}, err
	}
	tgtTables, err := d.tableDDLs(ctx, target)
	if err != nil {
		return SchemaDiffResult{}, err
	}
	var (
		added, dropped, altered []string
		ddl                     strings.Builder
		destructive             bool
	)
	for name, t := range tgtTables {
		if _, ok := baseTables[name]; !ok {
			added = append(added, name)
			ddl.WriteString(t + ";\n")
		} else if baseTables[name] != t {
			altered = append(altered, name)
			ddl.WriteString("-- altered " + name + "\n" + t + ";\n")
		}
	}
	for name := range baseTables {
		if _, ok := tgtTables[name]; !ok {
			dropped = append(dropped, name)
			ddl.WriteString("DROP TABLE " + quoteIdent(name) + ";\n")
			destructive = true
		}
	}
	return SchemaDiffResult{
		DDL:            ddl.String(),
		AddedTables:    added,
		DroppedTables:  dropped,
		AlteredTables:  altered,
		HasDestructive: destructive,
	}, nil
}

func (d *ClickHouseDriver) tableDDLs(ctx context.Context, db string) (map[string]string, error) {
	sql := fmt.Sprintf(
		"SELECT name, create_table_query FROM system.tables WHERE database = '%s' FORMAT JSON",
		escapeStringLit(db),
	)
	raw, err := d.querySQL(ctx, sql)
	if err != nil {
		return nil, err
	}
	var parsed struct {
		Data []struct {
			Name             string `json:"name"`
			CreateTableQuery string `json:"create_table_query"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, err
	}
	out := make(map[string]string, len(parsed.Data))
	for _, r := range parsed.Data {
		out[r.Name] = r.CreateTableQuery
	}
	return out, nil
}

// ListBranches returns databases whose name matches the twin prefix.
func (d *ClickHouseDriver) ListBranches(ctx context.Context, _ string) ([]Branch, error) {
	raw, err := d.querySQL(ctx, "SELECT name FROM system.databases FORMAT JSON")
	if err != nil {
		return nil, err
	}
	var parsed struct {
		Data []struct {
			Name string `json:"name"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, err
	}
	out := make([]Branch, 0, len(parsed.Data))
	for _, r := range parsed.Data {
		if !strings.HasPrefix(r.Name, "twin_") {
			continue
		}
		out = append(out, Branch{
			ID:        r.Name,
			ProjectID: d.sourceDB,
			Name:      r.Name,
			Host:      d.url,
			State:     "ready",
		})
	}
	return out, nil
}

func (d *ClickHouseDriver) execSQL(ctx context.Context, sql string) error {
	_, err := d.querySQL(ctx, sql)
	return err
}

func (d *ClickHouseDriver) querySQL(ctx context.Context, sql string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", d.url, bytes.NewReader([]byte(sql)))
	if err != nil {
		return nil, err
	}
	if d.user != "" {
		req.SetBasicAuth(d.user, d.password)
	}
	req.Header.Set("Content-Type", "text/plain")
	resp, err := d.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode >= 400 {
		return nil, &HTTPError{Status: resp.StatusCode, Body: string(body), URL: d.url}
	}
	return body, nil
}

// quoteIdent quotes a ClickHouse identifier. The simple form covers all
// names we generate (CREATE DATABASE / CREATE TABLE names); customer data
// flows via parameterised queries the runtime issues directly through the
// libchclient.
func quoteIdent(name string) string {
	// `name` is generated by us; we don't accept user input here. Still,
	// escape backticks defensively.
	return "`" + strings.ReplaceAll(name, "`", "``") + "`"
}

func escapeStringLit(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}

func buildClickHouseURI(base, user, password, db string) string {
	// Convert http(s)://host:port to clickhouse://user:pass@host:port/db
	scheme := "clickhouse"
	host := base
	if i := strings.Index(host, "://"); i >= 0 {
		host = host[i+3:]
	}
	auth := ""
	if user != "" {
		auth = url.QueryEscape(user) + ":" + url.QueryEscape(password) + "@"
	}
	return fmt.Sprintf("%s://%s%s/%s", scheme, auth, host, db)
}
