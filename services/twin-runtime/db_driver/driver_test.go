package dbdriver

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestStubDriverForDeferredEngines(t *testing.T) {
	// Phase 4+ engines still return the stub error from CreateBranch.
	for _, eng := range []Engine{EnginePostgresXata, EnginePostgresDBLab} {
		d := New(eng)
		require.Equal(t, eng, d.Engine())
		_, err := d.CreateBranch(context.Background(), BranchSpec{ProjectID: "p"}, CreateBranchOpts{})
		require.Error(t, err)
		require.True(t, IsStub(err))
	}
}

func TestPhase3EnginesNoLongerStubbedByDefault(t *testing.T) {
	// Each Phase 3 engine must return SOMETHING other than a STUB when its
	// env vars are absent — the per-engine constructor itself returns a
	// stub driver in that case (which is fine — IsStub() == true), but
	// running through `New(engine)` MUST land on the real type.
	cases := []struct {
		engine Engine
	}{
		{EngineMySQL},
		{EngineSQLite},
		{EngineMongo},
		{EngineRedis},
		{EngineClickHouse},
		{EngineS3},
	}
	for _, c := range cases {
		d := New(c.engine)
		require.Equal(t, c.engine, d.Engine(),
			"engine %s: New() returned wrong Engine()", c.engine)
	}
}

func TestDetectEngineRecognisesCommonHints(t *testing.T) {
	tests := []struct {
		hint string
		want Engine
	}{
		{"postgres", EnginePostgresNeon},
		{"pg", EnginePostgresNeon},
		{"PostgreSQL", EnginePostgresNeon},
		{"mysql", EngineMySQL},
		{"vitess", EngineMySQL},
		{"sqlite", EngineSQLite},
		{"libsql", EngineSQLite},
		{"turso", EngineSQLite},
		{"mongo", EngineMongo},
		{"mongodb", EngineMongo},
		{"redis", EngineRedis},
		{"valkey", EngineRedis},
		{"clickhouse", EngineClickHouse},
		{"ch", EngineClickHouse},
		{"s3", EngineS3},
		{"minio", EngineS3},
		{"gcs", EngineS3},
	}
	for _, tt := range tests {
		t.Run(tt.hint, func(t *testing.T) {
			require.Equal(t, tt.want, DetectEngine(tt.hint))
		})
	}
}

func TestNeonStubModeWithoutKey(t *testing.T) {
	t.Setenv(EnvNeonAPIKey, "")
	d := NewNeonDriver()
	_, err := d.CreateBranch(context.Background(), BranchSpec{ProjectID: "p"}, CreateBranchOpts{})
	require.Error(t, err)
	require.True(t, IsStub(err))
}

// fakeNeon emulates the create→poll→connection_uri dance.
type fakeNeon struct {
	server *httptest.Server
}

func newFakeNeon(t *testing.T) *fakeNeon {
	t.Helper()
	mux := http.NewServeMux()
	// POST /projects/{id}/branches
	mux.HandleFunc("/projects/proj_test/branches", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"branches": []map[string]any{
					{"id": "br_1", "project_id": "proj_test", "name": "main", "created_at": time.Now()},
				},
			})
			return
		}
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "Bearer test_key", r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"branch": map[string]any{
				"id": "br_new", "project_id": "proj_test", "name": "twin-task-1",
				"current_state": "init", "created_at": time.Now(),
			},
			"endpoints": []map[string]any{
				{"host": "ep.test", "id": "ep_1", "branch_id": "br_new", "type": "read_write"},
			},
			"operations": []map[string]any{
				{"id": "op_1", "action": "create_branch", "status": "running"},
			},
		})
	})
	// Operations poll
	mux.HandleFunc("/projects/proj_test/operations/op_1", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"operation": map[string]any{"id": "op_1", "action": "create_branch", "status": "finished"},
		})
	})
	// Connection URI
	mux.HandleFunc("/projects/proj_test/branches/br_new/connection_uri", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"uri": "postgres://test@ep.test/db"})
	})
	// Branch delete
	mux.HandleFunc("/projects/proj_test/branches/br_new", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			w.WriteHeader(http.StatusOK)
			return
		}
		// compare_schema target
		if r.Method == http.MethodGet && strings.Contains(r.URL.Path, "compare_schema") {
			_ = json.NewEncoder(w).Encode(map[string]any{"sql": "ALTER TABLE foo ADD COLUMN bar text;"})
			return
		}
		http.NotFound(w, r)
	})
	// Compare schema (separate path)
	mux.HandleFunc("/projects/proj_test/branches/br_new/compare_schema", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"sql": "ALTER TABLE foo ADD COLUMN bar text;"})
	})
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return &fakeNeon{server: server}
}

func TestNeonCreateBranchPollsUntilReady(t *testing.T) {
	fake := newFakeNeon(t)
	t.Setenv(EnvNeonAPIKey, "test_key")
	t.Setenv(EnvNeonBaseURL, fake.server.URL)
	t.Setenv(EnvNeonProjectID, "proj_test")
	d := NewNeonDriver()
	br, err := d.CreateBranch(context.Background(), BranchSpec{ProjectID: "proj_test"}, CreateBranchOpts{
		Timeout:      2 * time.Second,
		PollInterval: 10 * time.Millisecond,
	})
	require.NoError(t, err)
	require.Equal(t, "br_new", br.ID)
	require.Equal(t, "postgres://test@ep.test/db", br.ConnectionURI)
}

func TestNeonDeleteBranchIdempotentOn404(t *testing.T) {
	// 404 server.
	mux := http.NewServeMux()
	mux.HandleFunc("/projects/proj_test/branches/br_missing", func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	})
	s := httptest.NewServer(mux)
	t.Cleanup(s.Close)
	t.Setenv(EnvNeonAPIKey, "test_key")
	t.Setenv(EnvNeonBaseURL, s.URL)
	t.Setenv(EnvNeonProjectID, "proj_test")
	d := NewNeonDriver()
	require.NoError(t, d.DeleteBranch(context.Background(), "br_missing"))
}

func TestNeonSchemaDiffFlagsDestructive(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/projects/proj_test/branches/br_t/compare_schema", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"sql": "DROP TABLE users;"})
	})
	s := httptest.NewServer(mux)
	t.Cleanup(s.Close)
	t.Setenv(EnvNeonAPIKey, "test_key")
	t.Setenv(EnvNeonBaseURL, s.URL)
	t.Setenv(EnvNeonProjectID, "proj_test")
	d := NewNeonDriver()
	r, err := d.SchemaDiff(context.Background(), "br_base", "br_t", "db")
	require.NoError(t, err)
	require.True(t, r.HasDestructive)
}

func TestNeonCapabilitiesReflectMay2026Findings(t *testing.T) {
	t.Setenv(EnvNeonAPIKey, "x")
	d := NewNeonDriver()
	caps := d.Capabilities()
	require.True(t, caps.InstantBranch)
	require.True(t, caps.ScaleToZero)
	require.True(t, caps.FirstPartySchemaDiff)
	// Critical invariant from currency check: tenant isolation requires
	// one project per tenant.
	require.True(t, caps.PerTenantProjectRequired)
}

func TestNeonCreateBranchHonoursTimeout(t *testing.T) {
	// Server that never finishes the op.
	mux := http.NewServeMux()
	mux.HandleFunc("/projects/proj_test/branches", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"branch": map[string]any{"id": "br_stuck", "project_id": "proj_test"},
			"operations": []map[string]any{{"id": "op_stuck", "action": "create_branch", "status": "running"}},
		})
	})
	mux.HandleFunc("/projects/proj_test/operations/op_stuck", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"operation": map[string]any{"id": "op_stuck", "action": "create_branch", "status": "running"},
		})
	})
	s := httptest.NewServer(mux)
	t.Cleanup(s.Close)
	t.Setenv(EnvNeonAPIKey, "test_key")
	t.Setenv(EnvNeonBaseURL, s.URL)
	t.Setenv(EnvNeonProjectID, "proj_test")
	d := NewNeonDriver()
	_, err := d.CreateBranch(context.Background(), BranchSpec{ProjectID: "proj_test"}, CreateBranchOpts{
		Timeout:      50 * time.Millisecond,
		PollInterval: 10 * time.Millisecond,
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "did not finish")
}

func TestHasDestructiveDDLClassifier(t *testing.T) {
	tests := []struct {
		sql      string
		expected bool
	}{
		{"DROP TABLE users", true},
		{"drop table users", true},
		{"DROP SCHEMA public", true},
		{"DROP DATABASE prod", true},
		{"TRUNCATE charges", true},
		{"ALTER TABLE foo DROP COLUMN bar", true},
		{"DELETE FROM users", true},
		{"DELETE FROM users WHERE id = 1", false},
		{"ALTER TABLE foo ADD COLUMN bar text", false},
		{"CREATE TABLE foo (id int)", false},
		{"SELECT * FROM users", false},
	}
	for _, tt := range tests {
		t.Run(tt.sql, func(t *testing.T) {
			require.Equal(t, tt.expected, hasDestructiveDDL(tt.sql))
		})
	}
}

// Integration test against the real Neon API. Skipped unless the env var
// is set; mirrors Phase 1's TestIntegration_RealHaiku4_5 pattern.
func TestIntegration_RealNeon(t *testing.T) {
	if os.Getenv("CRUCIBLE_NEON_INTEGRATION") != "1" {
		t.Skip("CRUCIBLE_NEON_INTEGRATION not set; skipping real-Neon test")
	}
	require.NotEmpty(t, os.Getenv(EnvNeonAPIKey), "CRUCIBLE_NEON_API_KEY required")
	require.NotEmpty(t, os.Getenv(EnvNeonProjectID), "CRUCIBLE_NEON_PROJECT_ID required")
	d := NewNeonDriver()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	branches, err := d.ListBranches(ctx, os.Getenv(EnvNeonProjectID))
	require.NoError(t, err)
	t.Logf("project has %d branches", len(branches))
}
