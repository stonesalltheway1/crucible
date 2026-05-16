package dbdriver

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMongoStubModeWithoutEnv(t *testing.T) {
	t.Setenv(EnvMongoAtlasPublicKey, "")
	d := NewMongoDriver()
	_, err := d.CreateBranch(context.Background(), BranchSpec{}, CreateBranchOpts{})
	require.Error(t, err)
	require.True(t, IsStub(err))
}

func newFakeAtlas(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/groups/g1/clusters/shared", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"name": "shared",
			"connectionStrings": map[string]string{
				"standardSrv": "mongodb+srv://cluster.acme.mongodb.net/?retryWrites=true",
				"standard":    "mongodb://node1.acme,node2.acme/?",
			},
		})
	})
	mux.HandleFunc("/groups/g1/databaseUsers", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			w.WriteHeader(http.StatusCreated)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"results": []map[string]string{
				{"username": "twin_abc-u"},
				{"username": "twin_def-u"},
				{"username": "atlas-admin"},
			},
		})
	})
	mux.HandleFunc("/groups/g1/databaseUsers/admin/twin_abc-u", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			w.WriteHeader(http.StatusNoContent)
		}
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func TestMongoCreateBranchProvisionsScopedUser(t *testing.T) {
	srv := newFakeAtlas(t)
	t.Setenv(EnvMongoAtlasPublicKey, "pub")
	t.Setenv(EnvMongoAtlasPrivateKey, "priv")
	t.Setenv(EnvMongoAtlasGroupID, "g1")
	t.Setenv(EnvMongoAtlasCluster, "shared")
	t.Setenv(EnvMongoAtlasBaseURL, srv.URL)
	d := NewMongoDriver()
	br, err := d.CreateBranch(context.Background(), BranchSpec{Name: "abc"}, CreateBranchOpts{})
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(br.Name, "twin_"))
	require.Contains(t, br.ConnectionURI, "mongodb+srv://")
	require.Contains(t, br.ConnectionURI, br.Name)
	require.Contains(t, br.ConnectionURI, "twin_abc-u")
	require.NotEmpty(t, br.RolePassword)
}

func TestMongoListBranchesFiltersByPrefixAndUserSuffix(t *testing.T) {
	srv := newFakeAtlas(t)
	t.Setenv(EnvMongoAtlasPublicKey, "pub")
	t.Setenv(EnvMongoAtlasPrivateKey, "priv")
	t.Setenv(EnvMongoAtlasGroupID, "g1")
	t.Setenv(EnvMongoAtlasCluster, "shared")
	t.Setenv(EnvMongoAtlasBaseURL, srv.URL)
	d := NewMongoDriver()
	branches, err := d.ListBranches(context.Background(), "")
	require.NoError(t, err)
	require.Len(t, branches, 2)
	require.Equal(t, "twin_abc", branches[0].Name)
}

func TestMongoDeleteBranchIdempotentOn404(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/groups/g1/databaseUsers/admin/twin_missing-u", func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	t.Setenv(EnvMongoAtlasPublicKey, "pub")
	t.Setenv(EnvMongoAtlasPrivateKey, "priv")
	t.Setenv(EnvMongoAtlasGroupID, "g1")
	t.Setenv(EnvMongoAtlasCluster, "shared")
	t.Setenv(EnvMongoAtlasBaseURL, srv.URL)
	d := NewMongoDriver()
	require.NoError(t, d.DeleteBranch(context.Background(), "twin_missing"))
}

func TestInjectAuthIntoSRVPreservesQuery(t *testing.T) {
	uri := injectAuthIntoSRV(
		"mongodb+srv://cluster.acme.net/?retryWrites=true&w=majority",
		"u",
		"p:word",
		"twin_db",
	)
	require.Contains(t, uri, "u:p%3Aword@cluster.acme.net")
	require.Contains(t, uri, "/twin_db")
	require.Contains(t, uri, "?retryWrites=true&w=majority")
}
