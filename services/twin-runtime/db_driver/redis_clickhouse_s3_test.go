package dbdriver

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// ──────────────────────────────────────────────────────────────────────
// Redis
// ──────────────────────────────────────────────────────────────────────

func TestRedisCreateBranchYieldsConsistentURI(t *testing.T) {
	d := NewRedisDriver()
	br, err := d.CreateBranch(context.Background(), BranchSpec{Name: "twin-redis-a"}, CreateBranchOpts{})
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(br.ConnectionURI, "redis://"))
	// Determinism: same name → same port → same URI.
	br2, err := d.CreateBranch(context.Background(), BranchSpec{Name: "twin-redis-a"}, CreateBranchOpts{})
	require.NoError(t, err)
	require.Equal(t, br.ConnectionURI, br2.ConnectionURI)
}

func TestRedisCreateBranchUsesUniquePortPerName(t *testing.T) {
	d := NewRedisDriver()
	a, _ := d.CreateBranch(context.Background(), BranchSpec{Name: "twin-a"}, CreateBranchOpts{})
	b, _ := d.CreateBranch(context.Background(), BranchSpec{Name: "twin-b"}, CreateBranchOpts{})
	require.NotEqual(t, a.ConnectionURI, b.ConnectionURI,
		"different branch names should produce different ports")
}

func TestRedisDeleteIsNoop(t *testing.T) {
	d := NewRedisDriver()
	require.NoError(t, d.DeleteBranch(context.Background(), "twin-anything"))
}

// ──────────────────────────────────────────────────────────────────────
// ClickHouse
// ──────────────────────────────────────────────────────────────────────

func newFakeClickHouse(t *testing.T, createdQueries *[]string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := make([]byte, 4096)
		n, _ := r.Body.Read(buf)
		body := string(buf[:n])
		*createdQueries = append(*createdQueries, body)
		switch {
		case strings.Contains(body, "system.tables WHERE database"):
			_, _ = w.Write([]byte(`{"data":[{"name":"users","create_table_query":"CREATE TABLE users (id Int) ENGINE=MergeTree ORDER BY id"}]}`))
		case strings.Contains(body, "system.databases"):
			_, _ = w.Write([]byte(`{"data":[{"name":"twin_1"},{"name":"untouched"}]}`))
		default:
			_, _ = w.Write([]byte(`OK`))
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestClickHouseCreateBranchClonesEachTable(t *testing.T) {
	var queries []string
	srv := newFakeClickHouse(t, &queries)
	t.Setenv(EnvClickHouseURL, srv.URL)
	t.Setenv(EnvClickHouseSourceDB, "default")
	d := NewClickHouseDriver()
	br, err := d.CreateBranch(context.Background(), BranchSpec{Name: "twin_alpha"}, CreateBranchOpts{})
	require.NoError(t, err)
	require.Equal(t, "twin_alpha", br.Name)
	// CREATE DATABASE + list tables + CLONE AS each.
	require.True(t, anyContains(queries, "CREATE DATABASE"), "missing CREATE DATABASE: %v", queries)
	require.True(t, anyContains(queries, "CLONE AS `default`.`users`"))
	require.Contains(t, br.ConnectionURI, "clickhouse://")
}

func TestClickHouseSchemaDiffDetectsDropped(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := make([]byte, 4096)
		n, _ := r.Body.Read(buf)
		body := string(buf[:n])
		if strings.Contains(body, "WHERE database = 'base'") {
			_, _ = w.Write([]byte(`{"data":[{"name":"users","create_table_query":"CREATE TABLE users (id Int)"}]}`))
			return
		}
		if strings.Contains(body, "WHERE database = 'target'") {
			_, _ = w.Write([]byte(`{"data":[]}`))
			return
		}
		_, _ = w.Write([]byte(`OK`))
	}))
	t.Cleanup(srv.Close)
	t.Setenv(EnvClickHouseURL, srv.URL)
	t.Setenv(EnvClickHouseSourceDB, "default")
	d := NewClickHouseDriver()
	r, err := d.SchemaDiff(context.Background(), "base", "target", "")
	require.NoError(t, err)
	require.True(t, r.HasDestructive)
	require.Contains(t, r.DroppedTables, "users")
}

func TestClickHouseListBranchesFiltersPrefix(t *testing.T) {
	var queries []string
	srv := newFakeClickHouse(t, &queries)
	t.Setenv(EnvClickHouseURL, srv.URL)
	t.Setenv(EnvClickHouseSourceDB, "default")
	d := NewClickHouseDriver()
	out, err := d.ListBranches(context.Background(), "")
	require.NoError(t, err)
	require.Len(t, out, 1)
	require.Equal(t, "twin_1", out[0].Name)
}

func anyContains(s []string, needle string) bool {
	for _, x := range s {
		if strings.Contains(x, needle) {
			return true
		}
	}
	return false
}

// ──────────────────────────────────────────────────────────────────────
// S3 / MinIO
// ──────────────────────────────────────────────────────────────────────

func TestS3StubWithoutEnv(t *testing.T) {
	t.Setenv(EnvS3Endpoint, "")
	d := NewS3Driver()
	_, err := d.CreateBranch(context.Background(), BranchSpec{}, CreateBranchOpts{})
	require.Error(t, err)
	require.True(t, IsStub(err))
}

func newFakeS3(t *testing.T, captured *map[string]string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		(*captured)[r.Method+" "+r.URL.Path] = r.Header.Get("Authorization")
		switch r.Method {
		case http.MethodPut:
			w.WriteHeader(http.StatusOK)
		case http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		case http.MethodGet:
			_, _ = w.Write([]byte(`<?xml version="1.0"?>
<ListAllMyBucketsResult><Buckets>
  <Bucket><Name>twin-a</Name></Bucket>
  <Bucket><Name>untouched</Name></Bucket>
</Buckets></ListAllMyBucketsResult>`))
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestS3CreateBucketSendsSigV2Header(t *testing.T) {
	captured := make(map[string]string)
	srv := newFakeS3(t, &captured)
	t.Setenv(EnvS3Endpoint, srv.URL)
	t.Setenv(EnvS3AccessKey, "AKEY")
	t.Setenv(EnvS3SecretKey, "SECRET")
	d := NewS3Driver()
	br, err := d.CreateBranch(context.Background(), BranchSpec{Name: "twin-test"}, CreateBranchOpts{})
	require.NoError(t, err)
	require.Equal(t, "twin-test", br.Name)
	auth := captured["PUT /twin-test"]
	require.True(t, strings.HasPrefix(auth, "AWS AKEY:"),
		"auth header should be SigV2 AWS-prefixed; got %s", auth)
}

func TestS3ListBranchesFiltersByPrefix(t *testing.T) {
	captured := make(map[string]string)
	srv := newFakeS3(t, &captured)
	t.Setenv(EnvS3Endpoint, srv.URL)
	t.Setenv(EnvS3AccessKey, "AKEY")
	t.Setenv(EnvS3SecretKey, "SECRET")
	d := NewS3Driver()
	out, err := d.ListBranches(context.Background(), "")
	require.NoError(t, err)
	require.Len(t, out, 1)
	require.Equal(t, "twin-a", out[0].Name)
}

func TestS3RcloneSeedCommandShape(t *testing.T) {
	t.Setenv(EnvS3Endpoint, "http://minio:9000")
	t.Setenv(EnvS3AccessKey, "AKEY")
	t.Setenv(EnvS3SecretKey, "SECRET")
	t.Setenv(EnvS3SourceBucket, "prod-data")
	t.Setenv(EnvS3MirrorPrefix, "events/")
	d := NewS3Driver().(*S3Driver)
	cmd := d.rcloneSeedCommand("twin-1")
	require.Contains(t, cmd, "rclone copy src:prod-data/events/")
	require.Contains(t, cmd, "dst:twin-1")
	require.Contains(t, cmd, "--transfers=8")
}

// Phase 3 spawn latency sanity probe. We don't actually spawn anything
// against the real services; the test asserts the driver's local overhead
// for a stub branch is below 100ms — a regression sentinel for the local
// path (the real spawn budget is enforced by the integration tests).
func TestPhase3Drivers_LocalOverhead(t *testing.T) {
	cases := []struct {
		engine    Engine
		setup     func()
		shouldErr bool
	}{
		{EngineRedis, func() {}, false},
	}
	for _, c := range cases {
		t.Run(string(c.engine), func(t *testing.T) {
			c.setup()
			d := New(c.engine)
			start := time.Now()
			_, err := d.CreateBranch(context.Background(), BranchSpec{Name: "perf"}, CreateBranchOpts{})
			elapsed := time.Since(start)
			if c.shouldErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			require.Less(t, elapsed.Milliseconds(), int64(100),
				"local overhead for %s exceeded 100ms (got %s)", c.engine, elapsed)
		})
	}
}
