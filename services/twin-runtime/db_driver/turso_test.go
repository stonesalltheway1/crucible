package dbdriver

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestTursoStubModeWithoutToken(t *testing.T) {
	t.Setenv(EnvTursoToken, "")
	d := NewTursoDriver()
	_, err := d.CreateBranch(context.Background(), BranchSpec{}, CreateBranchOpts{})
	require.Error(t, err)
	require.True(t, IsStub(err))
}

func newFakeTurso(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/organizations/acme/databases", func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "Bearer tok", r.Header.Get("Authorization"))
		switch r.Method {
		case http.MethodPost:
			var body tursoCreateRequest
			raw, _ := readAll(r.Body)
			_ = json.Unmarshal(raw, &body)
			require.NotNil(t, body.Seed)
			require.Equal(t, "database", body.Seed.Type)
			require.Equal(t, "main", body.Seed.Name)
			resp := tursoCreateResponse{}
			resp.Database.Name = body.Name
			resp.Database.Hostname = body.Name + ".turso.io"
			resp.Database.DbId = "db_1"
			resp.Database.Group = body.Group
			_ = json.NewEncoder(w).Encode(resp)
		case http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]any{
				"databases": []map[string]string{
					{"Name": "twin-1", "Hostname": "twin-1.turso.io", "DbId": "db_1", "group": "crucible-twin"},
					{"Name": "untouched", "Hostname": "untouched.turso.io", "DbId": "db_2", "group": "other"},
				},
			})
		}
	})
	mux.HandleFunc("/organizations/acme/databases/twin-1/auth/tokens", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(tursoTokenResponse{JWT: "eyJ.fake.jwt"})
	})
	mux.HandleFunc("/organizations/acme/databases/twin-1/schema", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"objects": []sqliteObj{
				{Type: "table", Name: "users", SQL: "CREATE TABLE users (id INTEGER)"},
				{Type: "table", Name: "carts", SQL: "CREATE TABLE carts (id INTEGER)"},
			},
		})
	})
	mux.HandleFunc("/organizations/acme/databases/main/schema", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"objects": []sqliteObj{
				{Type: "table", Name: "users", SQL: "CREATE TABLE users (id INTEGER)"},
				{Type: "table", Name: "old", SQL: "CREATE TABLE old (id INTEGER)"},
			},
		})
	})
	mux.HandleFunc("/organizations/acme/databases/twin-1", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			w.WriteHeader(http.StatusOK)
		}
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func readAll(r interface{ Read(p []byte) (int, error) }) ([]byte, error) {
	buf := make([]byte, 0, 1024)
	tmp := make([]byte, 256)
	for {
		n, err := r.Read(tmp)
		buf = append(buf, tmp[:n]...)
		if err != nil {
			if err.Error() == "EOF" {
				return buf, nil
			}
			return buf, err
		}
	}
}

func TestTursoCreateBranchSeedsAndIssuesToken(t *testing.T) {
	srv := newFakeTurso(t)
	t.Setenv(EnvTursoToken, "tok")
	t.Setenv(EnvTursoOrg, "acme")
	t.Setenv(EnvTursoBaseURL, srv.URL)
	d := NewTursoDriver()
	br, err := d.CreateBranch(context.Background(), BranchSpec{
		Name: "twin-1", BaseBranchName: "main",
	}, CreateBranchOpts{Timeout: 2 * time.Second})
	require.NoError(t, err)
	require.Contains(t, br.ConnectionURI, "libsql://")
	require.Contains(t, br.ConnectionURI, "authToken=eyJ.fake.jwt")
	require.Equal(t, "twin-1.turso.io", br.Host)
}

func TestTursoSchemaDiffDetectsDroppedTable(t *testing.T) {
	srv := newFakeTurso(t)
	t.Setenv(EnvTursoToken, "tok")
	t.Setenv(EnvTursoOrg, "acme")
	t.Setenv(EnvTursoBaseURL, srv.URL)
	d := NewTursoDriver()
	r, err := d.SchemaDiff(context.Background(), "main", "twin-1", "")
	require.NoError(t, err)
	require.True(t, r.HasDestructive)
	require.Contains(t, r.DroppedTables, "old")
	require.Contains(t, r.AddedTables, "carts")
	require.True(t, strings.Contains(r.DDL, "DROP TABLE old"))
}

func TestTursoListBranchesFiltersByGroup(t *testing.T) {
	srv := newFakeTurso(t)
	t.Setenv(EnvTursoToken, "tok")
	t.Setenv(EnvTursoOrg, "acme")
	t.Setenv(EnvTursoBaseURL, srv.URL)
	d := NewTursoDriver()
	branches, err := d.ListBranches(context.Background(), "")
	require.NoError(t, err)
	require.Len(t, branches, 1)
	require.Equal(t, "twin-1", branches[0].Name)
}
