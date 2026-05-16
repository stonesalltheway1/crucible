package secretssidecar

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

func TestStubModeWithoutKeys(t *testing.T) {
	t.Setenv(EnvClientID, "")
	t.Setenv(EnvClientSecret, "")
	s := New()
	_, err := s.IssueLease(context.Background(), LeaseRequest{
		Name: "db", VaultPath: "/db", TTL: time.Minute, Scope: ScopeDynamicPG,
	})
	require.Error(t, err)
	require.True(t, IsStub(err))
}

func TestIssueLeaseRejectsBelowFloor(t *testing.T) {
	t.Setenv(EnvClientID, "id")
	t.Setenv(EnvClientSecret, "sec")
	s := New()
	_, err := s.IssueLease(context.Background(), LeaseRequest{
		Name: "db", VaultPath: "/db", TTL: 2 * time.Second, Scope: ScopeDynamicPG,
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "below Infisical floor")
}

func TestIssueLeaseAtFloorAccepted(t *testing.T) {
	srv := newFakeInfisical(t)
	t.Setenv(EnvAPIURL, srv.URL)
	t.Setenv(EnvClientID, "id")
	t.Setenv(EnvClientSecret, "sec")
	t.Setenv(EnvProjectID, "proj")
	s := New()
	lease, err := s.IssueLease(context.Background(), LeaseRequest{
		Name: "db", VaultPath: "/dynamic/db", TTL: MinTTL, Scope: ScopeDynamicPG,
	})
	require.NoError(t, err)
	require.NotEmpty(t, lease.ID)
}

func TestIssueLeaseDoesNotReturnRawValue(t *testing.T) {
	srv := newFakeInfisical(t)
	t.Setenv(EnvAPIURL, srv.URL)
	t.Setenv(EnvClientID, "id")
	t.Setenv(EnvClientSecret, "sec")
	t.Setenv(EnvProjectID, "proj")
	s := New()
	lease, err := s.IssueLease(context.Background(), LeaseRequest{
		Name: "db", VaultPath: "/dynamic/db", TTL: 30 * time.Second, Scope: ScopeDynamicPG,
	})
	require.NoError(t, err)
	// The Lease MUST NOT contain the raw value.
	raw, _ := json.Marshal(lease)
	require.NotContains(t, string(raw), "supersecretpw", "raw credential leaked in lease struct")
	require.NotContains(t, string(raw), "db_user_dyn", "raw credential leaked in lease struct")
}

func TestResolveReturnsValueOnlyAfterIssue(t *testing.T) {
	srv := newFakeInfisical(t)
	t.Setenv(EnvAPIURL, srv.URL)
	t.Setenv(EnvClientID, "id")
	t.Setenv(EnvClientSecret, "sec")
	t.Setenv(EnvProjectID, "proj")
	s := New()
	lease, err := s.IssueLease(context.Background(), LeaseRequest{
		Name: "db", VaultPath: "/dynamic/db", TTL: 30 * time.Second, Scope: ScopeDynamicPG,
	})
	require.NoError(t, err)
	value, err := s.Resolve(context.Background(), SecretRef{LeaseID: lease.ID, Name: "db"})
	require.NoError(t, err)
	require.True(t, strings.Contains(value, "supersecretpw"), "value should carry credential")

	// Unknown lease ID fails.
	_, err = s.Resolve(context.Background(), SecretRef{LeaseID: "ghost", Name: "db"})
	require.Error(t, err)
}

func TestRevokeLeaseIdempotent(t *testing.T) {
	srv := newFakeInfisical(t)
	t.Setenv(EnvAPIURL, srv.URL)
	t.Setenv(EnvClientID, "id")
	t.Setenv(EnvClientSecret, "sec")
	t.Setenv(EnvProjectID, "proj")
	s := New()
	lease, err := s.IssueLease(context.Background(), LeaseRequest{
		Name: "db", VaultPath: "/dynamic/db", TTL: 30 * time.Second, Scope: ScopeDynamicPG,
	})
	require.NoError(t, err)
	require.NoError(t, s.RevokeLease(context.Background(), lease.ID))
	// Second call is idempotent (404 from fake server).
	require.NoError(t, s.RevokeLease(context.Background(), lease.ID))
}

func TestDirectiveByScope(t *testing.T) {
	d := directiveFor(ScopeStatic, "x")
	require.Equal(t, "Bearer %s", d.HeaderFormat)
	d = directiveFor(ScopeDynamicAWS, "x")
	require.Contains(t, d.HeaderFormat, "AWS4")
	d = directiveFor(ScopeDynamicPG, "x")
	require.Equal(t, "$.credentials", d.BodyJSONPath)
}

func newFakeInfisical(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/auth/universal-auth/login", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"accessToken": "test_bearer",
			"expiresIn":   600,
		})
	})
	leases := map[string]bool{}
	mux.HandleFunc("/v1/dynamic-secrets/leases", func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"lease": map[string]any{
				"id":        "lease_test_1",
				"expiresAt": time.Now().Add(2 * time.Minute),
				"data": map[string]string{
					"username": "db_user_dyn",
					"password": "supersecretpw",
				},
			},
		})
		leases["lease_test_1"] = true
	})
	mux.HandleFunc("/v1/dynamic-secrets/leases/lease_test_1", func(w http.ResponseWriter, r *http.Request) {
		if !leases["lease_test_1"] {
			http.NotFound(w, r)
			return
		}
		delete(leases, "lease_test_1")
		w.WriteHeader(http.StatusOK)
	})
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return server
}
