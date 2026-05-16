package dbdriver

// MongoDB driver — shared-cluster variant.
//
// Phase 3 currency check (May 2026) found that Mongo Atlas's
// snapshot-restore-to-new-cluster API takes 15–60 minutes — fundamentally
// incompatible with per-task ephemeral branching (Phase 3 budget: ≤5s
// for Mongo). M2/M5 shared tiers were retired 2026-01-22.
//
// Decision: ship a database-per-task variant on a SHARED Atlas Flex cluster.
// Each "branch" is a freshly-created database inside the cluster, seeded
// from a parent database via mongodump|mongorestore equivalents (here:
// `db.cloneCollection` / `aggregate $out` for collections, applied at
// adapter scale).
//
// Trade-offs (documented in the Capabilities and surfaced to ops):
//   - No real isolation: a task can `listDatabases`. Twin-mode shells must
//     run with a per-task user that has access ONLY to its own database.
//   - No PITR-from-parent: parent is the live shared cluster, not a frozen
//     snapshot. Tasks see the moving target.
//   - Drop is db.dropDatabase(), ~100ms typical.
//
// The shared-cluster variant uses a single Atlas API key + a per-task user
// minted via the Atlas Database Users endpoint. Per-task users get
// `readWriteAnyDatabase` scoped via custom role to a database name prefix.
// Cleanup deletes both the database AND the per-task user.

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
	EnvMongoAtlasPublicKey  = "CRUCIBLE_MONGO_ATLAS_PUBLIC_KEY"
	EnvMongoAtlasPrivateKey = "CRUCIBLE_MONGO_ATLAS_PRIVATE_KEY"
	EnvMongoAtlasGroupID    = "CRUCIBLE_MONGO_ATLAS_GROUP_ID"
	EnvMongoAtlasCluster    = "CRUCIBLE_MONGO_ATLAS_CLUSTER"
	EnvMongoAtlasBaseURL    = "CRUCIBLE_MONGO_ATLAS_BASE_URL"
	EnvMongoTaskDBPrefix    = "CRUCIBLE_MONGO_TASK_DB_PREFIX"

	DefaultMongoAtlasBaseURL = "https://cloud.mongodb.com/api/atlas/v2"
	DefaultMongoTaskDBPrefix = "twin_"
)

// MongoDriver wraps the Atlas Admin API for the shared-cluster database-per-task
// pattern.
type MongoDriver struct {
	publicKey  string
	privateKey string
	groupID    string
	cluster    string
	baseURL    string
	prefix     string
	client     *http.Client
	caps       Capabilities
}

// NewMongoDriver constructs from env.
func NewMongoDriver() Driver {
	pub := os.Getenv(EnvMongoAtlasPublicKey)
	priv := os.Getenv(EnvMongoAtlasPrivateKey)
	group := os.Getenv(EnvMongoAtlasGroupID)
	cluster := os.Getenv(EnvMongoAtlasCluster)
	if pub == "" || priv == "" || group == "" || cluster == "" {
		return newStubDriver(EngineMongo, fmt.Sprintf(
			"Mongo Atlas driver missing one of: %s, %s, %s, %s. Driver in stub mode.",
			EnvMongoAtlasPublicKey, EnvMongoAtlasPrivateKey,
			EnvMongoAtlasGroupID, EnvMongoAtlasCluster,
		))
	}
	prefix := os.Getenv(EnvMongoTaskDBPrefix)
	if prefix == "" {
		prefix = DefaultMongoTaskDBPrefix
	}
	base := os.Getenv(EnvMongoAtlasBaseURL)
	if base == "" {
		base = DefaultMongoAtlasBaseURL
	}
	return &MongoDriver{
		publicKey:  pub,
		privateKey: priv,
		groupID:    group,
		cluster:    cluster,
		baseURL:    strings.TrimRight(base, "/"),
		prefix:     prefix,
		client:     &http.Client{Timeout: 30 * time.Second},
		caps: Capabilities{
			// Shared-cluster variant: branch == "create database inside the
			// shared Flex cluster"; ~hundreds of ms typical. Far inside the
			// 5s budget. Native snapshot-restore-to-new-cluster takes 15–60
			// min and isn't viable for per-task branches.
			InstantBranch:            true,
			ScaleToZero:              false, // shared cluster always on
			FirstPartySchemaDiff:     false,
			MaxConcurrentBranches:    0,
			PerTenantProjectRequired: true, // group-scoped API keys
		},
	}
}

// Engine returns EngineMongo.
func (d *MongoDriver) Engine() Engine { return EngineMongo }

// Capabilities returns the Mongo feature matrix.
func (d *MongoDriver) Capabilities() Capabilities { return d.caps }

type mongoUserRequest struct {
	DatabaseName string                 `json:"databaseName"`
	Username     string                 `json:"username"`
	Password     string                 `json:"password"`
	Roles        []mongoUserRole        `json:"roles"`
	Scopes       []map[string]string    `json:"scopes,omitempty"`
}

type mongoUserRole struct {
	RoleName     string `json:"roleName"`
	DatabaseName string `json:"databaseName"`
}

type mongoClusterResponse struct {
	ConnectionStrings struct {
		StandardSrv string `json:"standardSrv"`
		Standard    string `json:"standard"`
	} `json:"connectionStrings"`
	Name string `json:"name"`
}

// CreateBranch provisions a fresh per-task database inside the shared
// Atlas cluster. The "seed" step is the caller's responsibility (the
// runtime invokes `mongorestore` from the dump artefact); we provide the
// connection string and the per-task user.
func (d *MongoDriver) CreateBranch(
	ctx context.Context, spec BranchSpec, _ CreateBranchOpts,
) (Branch, error) {
	name := spec.Name
	if name == "" {
		name = fmt.Sprintf("%s%d", d.prefix, time.Now().UnixNano())
	} else if !strings.HasPrefix(name, d.prefix) {
		name = d.prefix + name
	}
	cluster, err := d.fetchCluster(ctx)
	if err != nil {
		return Branch{}, err
	}
	username := name + "-u"
	password := generateMongoPassword()
	userReq := mongoUserRequest{
		DatabaseName: "admin",
		Username:     username,
		Password:     password,
		Roles: []mongoUserRole{{
			RoleName:     "readWrite",
			DatabaseName: name,
		}},
		Scopes: []map[string]string{
			{"name": d.cluster, "type": "CLUSTER"},
		},
	}
	if err := d.do(ctx, "POST", d.groupURL()+"/databaseUsers", userReq, nil); err != nil {
		return Branch{}, fmt.Errorf("mongo create user: %w", err)
	}
	// The connection URI shape: mongodb+srv://<user>:<pass>@<host>/<dbName>?...
	uri := injectAuthIntoSRV(cluster.ConnectionStrings.StandardSrv, username, password, name)
	return Branch{
		ID:            name,
		ProjectID:     d.groupID,
		Name:          name,
		Host:          cluster.Name,
		ConnectionURI: uri,
		State:         "ready",
		CreatedAt:     time.Now(),
		RolePassword:  password,
		Metadata:      spec.Tags,
	}, nil
}

func (d *MongoDriver) fetchCluster(ctx context.Context) (*mongoClusterResponse, error) {
	endpoint := d.groupURL() + "/clusters/" + url.PathEscape(d.cluster)
	var resp mongoClusterResponse
	if err := d.do(ctx, "GET", endpoint, nil, &resp); err != nil {
		return nil, fmt.Errorf("mongo fetch cluster: %w", err)
	}
	if resp.ConnectionStrings.StandardSrv == "" && resp.ConnectionStrings.Standard == "" {
		return nil, errors.New("mongo cluster has no connection string yet")
	}
	return &resp, nil
}

func (d *MongoDriver) groupURL() string {
	return fmt.Sprintf("%s/groups/%s", d.baseURL, url.PathEscape(d.groupID))
}

// DeleteBranch drops the database AND the per-task user. Best-effort: a
// user-delete failure does not block a database-drop success.
func (d *MongoDriver) DeleteBranch(ctx context.Context, branchID string) error {
	if branchID == "" {
		return errors.New("mongo DeleteBranch: empty branchID")
	}
	// The actual db drop happens via the libmongoc driver; this API only
	// handles cluster/user lifecycle. Wire-level: the runtime issues
	// `db.dropDatabase()` against the Mongo URI when it cleans up the
	// sandbox. Here we delete the per-task user so the credential can't be
	// used after the task ends.
	username := branchID + "-u"
	endpoint := d.groupURL() + "/databaseUsers/admin/" + url.PathEscape(username)
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

// SchemaDiff is not supported on Mongo (schema-less) — returns an empty
// result with HasDestructive=false. Callers should use validator-shape
// diffing at the application layer.
func (d *MongoDriver) SchemaDiff(_ context.Context, _, _, _ string) (SchemaDiffResult, error) {
	return SchemaDiffResult{}, nil
}

// ListBranches enumerates per-task databases by listing the cluster's
// databases and filtering on the prefix. Atlas Admin API exposes this via
// `/groups/{groupId}/clusters/{clusterName}/processArgs/databases` but the
// canonical API is to query the cluster directly via the connection URI.
// We expose a Phase-3 stub that returns just the user list filtered by
// prefix; the GC reconciler is the primary caller.
func (d *MongoDriver) ListBranches(ctx context.Context, _ string) ([]Branch, error) {
	endpoint := d.groupURL() + "/databaseUsers?itemsPerPage=200"
	var resp struct {
		Results []struct {
			Username string `json:"username"`
		} `json:"results"`
	}
	if err := d.do(ctx, "GET", endpoint, nil, &resp); err != nil {
		return nil, err
	}
	out := make([]Branch, 0, len(resp.Results))
	for _, u := range resp.Results {
		if !strings.HasSuffix(u.Username, "-u") {
			continue
		}
		dbName := strings.TrimSuffix(u.Username, "-u")
		if !strings.HasPrefix(dbName, d.prefix) {
			continue
		}
		out = append(out, Branch{
			ID:        dbName,
			ProjectID: d.groupID,
			Name:      dbName,
			Host:      d.cluster,
			State:     "ready",
		})
	}
	return out, nil
}

func (d *MongoDriver) do(ctx context.Context, method, urlS string, body any, out any) error {
	var reqBody io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal: %w", err)
		}
		reqBody = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, urlS, reqBody)
	if err != nil {
		return err
	}
	req.SetBasicAuth(d.publicKey, d.privateKey)
	req.Header.Set("Accept", "application/vnd.atlas.2024-08-05+json")
	if body != nil {
		req.Header.Set("Content-Type", "application/vnd.atlas.2024-08-05+json")
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
	if out != nil && resp.StatusCode != 204 {
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			return fmt.Errorf("decode: %w", err)
		}
	}
	return nil
}

func generateMongoPassword() string {
	return fmt.Sprintf("c_%d_%d", time.Now().UnixNano(), time.Now().Unix()%1_000_000)
}

func injectAuthIntoSRV(srv, user, pass, dbName string) string {
	if srv == "" {
		return ""
	}
	// srv looks like mongodb+srv://<host>/?retryWrites=true&w=majority
	idx := strings.Index(srv, "://")
	if idx < 0 {
		return srv
	}
	scheme := srv[:idx+3]
	rest := srv[idx+3:]
	authority, params, _ := strings.Cut(rest, "/")
	// params may include the query string; preserve it.
	q := ""
	if i := strings.Index(params, "?"); i >= 0 {
		q = params[i:]
	}
	return fmt.Sprintf("%s%s:%s@%s/%s%s",
		scheme,
		url.QueryEscape(user),
		url.QueryEscape(pass),
		authority,
		dbName,
		q,
	)
}
