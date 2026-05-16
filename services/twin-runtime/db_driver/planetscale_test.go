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

func TestPlanetScaleStubModeWithoutEnv(t *testing.T) {
	t.Setenv(EnvPlanetScaleTokenID, "")
	d := NewPlanetScaleDriver()
	_, err := d.CreateBranch(context.Background(), BranchSpec{}, CreateBranchOpts{})
	require.Error(t, err)
	require.True(t, IsStub(err))
}

func newFakePlanetScale(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	branchReady := false
	mux.HandleFunc("/organizations/acme/databases/app/branches", func(w http.ResponseWriter, r *http.Request) {
		// Auth shape: id:token (colon)
		require.Equal(t, "tid:tok", r.Header.Get("Authorization"))
		switch r.Method {
		case http.MethodPost:
			_ = json.NewEncoder(w).Encode(psBranchResponse{
				ID: "br_1", Name: "twin-1", Ready: false, ParentBranch: "main",
			})
		case http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": []psBranchResponse{
					{ID: "br_1", Name: "twin-1", Ready: true, MysqlAddress: "host.ps"},
				},
			})
		}
	})
	mux.HandleFunc("/organizations/acme/databases/app/branches/twin-1", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			w.WriteHeader(http.StatusOK)
			return
		}
		branchReady = !branchReady // flip on each call so first GET=false then true
		_ = json.NewEncoder(w).Encode(psBranchResponse{
			ID: "br_1", Name: "twin-1", Ready: branchReady, MysqlAddress: "host.ps",
		})
	})
	mux.HandleFunc("/organizations/acme/databases/app/branches/twin-1/passwords", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(psPasswordResponse{
			ID: "pw_1", Username: "tw_user", PlainText: "tw_pass",
			Host: "host.ps", DatabaseName: "app",
		})
	})
	mux.HandleFunc("/organizations/acme/databases/app/branches/twin-1/diff", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"diff": []map[string]string{{"raw": "DROP TABLE old_users;"}},
		})
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func TestPlanetScaleCreateBranchPollsThenIssuesPassword(t *testing.T) {
	srv := newFakePlanetScale(t)
	t.Setenv(EnvPlanetScaleTokenID, "tid")
	t.Setenv(EnvPlanetScaleToken, "tok")
	t.Setenv(EnvPlanetScaleOrg, "acme")
	t.Setenv(EnvPlanetScaleDB, "app")
	t.Setenv(EnvPlanetScaleBaseURL, srv.URL)
	d := NewPlanetScaleDriver()
	br, err := d.CreateBranch(context.Background(), BranchSpec{Name: "twin-1"}, CreateBranchOpts{
		Timeout: 2 * time.Second, PollInterval: 10 * time.Millisecond,
	})
	require.NoError(t, err)
	require.Equal(t, "twin-1", br.Name)
	require.Contains(t, br.ConnectionURI, "mysql://tw_user:tw_pass@host.ps/app")
	require.Equal(t, "tw_pass", br.RolePassword)
}

func TestPlanetScaleSchemaDiffFlagsDestructive(t *testing.T) {
	srv := newFakePlanetScale(t)
	t.Setenv(EnvPlanetScaleTokenID, "tid")
	t.Setenv(EnvPlanetScaleToken, "tok")
	t.Setenv(EnvPlanetScaleOrg, "acme")
	t.Setenv(EnvPlanetScaleDB, "app")
	t.Setenv(EnvPlanetScaleBaseURL, srv.URL)
	d := NewPlanetScaleDriver()
	r, err := d.SchemaDiff(context.Background(), "main", "twin-1", "app")
	require.NoError(t, err)
	require.True(t, r.HasDestructive)
	require.True(t, strings.Contains(r.DDL, "DROP TABLE"))
}

func TestPlanetScaleDeleteBranchUsesRecursive(t *testing.T) {
	var observedRecursive bool
	mux := http.NewServeMux()
	mux.HandleFunc("/organizations/acme/databases/app/branches/twin-1", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			if r.URL.Query().Get("recursive") == "true" {
				observedRecursive = true
			}
			w.WriteHeader(http.StatusOK)
		}
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	t.Setenv(EnvPlanetScaleTokenID, "tid")
	t.Setenv(EnvPlanetScaleToken, "tok")
	t.Setenv(EnvPlanetScaleOrg, "acme")
	t.Setenv(EnvPlanetScaleDB, "app")
	t.Setenv(EnvPlanetScaleBaseURL, srv.URL)
	d := NewPlanetScaleDriver()
	require.NoError(t, d.DeleteBranch(context.Background(), "twin-1"))
	require.True(t, observedRecursive, "DELETE should include recursive=true")
}
