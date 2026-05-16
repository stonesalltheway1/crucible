package dbdriver

// Redis driver.
//
// Per docs/05-decisions/ADR-005-neon-db-branching.md, Redis branching is
// "fresh redis-server inside the sandbox" — the state is small enough that
// a fresh per-task instance is faster than any branching scheme.
//
// The driver doesn't actually run the redis-server process — that's the
// runtime/lifecycle layer's job. Instead it generates a deterministic
// connection-URI for a per-task in-sandbox instance, optionally seeding
// the per-task DB from a `redis-cli --rdb` snapshot the user uploaded.

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	EnvRedisBindHost = "CRUCIBLE_REDIS_BIND_HOST"
	EnvRedisBasePort = "CRUCIBLE_REDIS_BASE_PORT"
)

// RedisDriver issues handles for per-task in-sandbox redis-server instances.
type RedisDriver struct {
	bindHost string
	basePort int
}

// NewRedisDriver constructs from env. Defaults to 127.0.0.1:6500+offset.
func NewRedisDriver() Driver {
	host := os.Getenv(EnvRedisBindHost)
	if host == "" {
		host = "127.0.0.1"
	}
	port := 6500
	if raw := os.Getenv(EnvRedisBasePort); raw != "" {
		if v, err := strconv.Atoi(raw); err == nil && v > 0 && v < 65536 {
			port = v
		}
	}
	return &RedisDriver{
		bindHost: host,
		basePort: port,
	}
}

// Engine returns EngineRedis.
func (d *RedisDriver) Engine() Engine { return EngineRedis }

// Capabilities returns the Redis feature matrix.
func (d *RedisDriver) Capabilities() Capabilities {
	return Capabilities{
		InstantBranch:            true, // fresh redis-server is sub-100ms typical
		ScaleToZero:              false,
		FirstPartySchemaDiff:     false,
		MaxConcurrentBranches:    0,
		PerTenantProjectRequired: false,
	}
}

// CreateBranch returns a Branch handle pointing at an in-sandbox port. The
// runtime spawns the redis-server process per the BranchSpec.Tags["seed-rdb"]
// hint (if present, redis-server is launched with --dbfilename pointing at
// the RDB).
func (d *RedisDriver) CreateBranch(
	_ context.Context, spec BranchSpec, _ CreateBranchOpts,
) (Branch, error) {
	name := spec.Name
	if name == "" {
		name = fmt.Sprintf("twin-redis-%d", time.Now().UnixNano())
	}
	port := d.basePort + portOffset(name)
	uri := fmt.Sprintf("redis://%s:%d/0", d.bindHost, port)
	return Branch{
		ID:            name,
		ProjectID:     "inproc",
		Name:          name,
		Host:          d.bindHost,
		ConnectionURI: uri,
		State:         "ready",
		CreatedAt:     time.Now(),
		Metadata:      spec.Tags,
	}, nil
}

// DeleteBranch is a no-op at the driver level — the runtime kills the
// per-task redis-server process via the sandbox-kill path.
func (d *RedisDriver) DeleteBranch(_ context.Context, _ string) error { return nil }

// SchemaDiff is not meaningful for Redis (schema-less). Returns empty.
func (d *RedisDriver) SchemaDiff(_ context.Context, _, _, _ string) (SchemaDiffResult, error) {
	return SchemaDiffResult{}, nil
}

// ListBranches returns an empty list — instances are ephemeral and the
// driver doesn't maintain a registry. The runtime's lifecycle tracker is
// the source of truth.
func (d *RedisDriver) ListBranches(_ context.Context, _ string) ([]Branch, error) {
	return nil, nil
}

func portOffset(name string) int {
	if name == "" {
		return 0
	}
	// FNV-1a 32-bit, modulo 4000 → 6500..10500
	const offset32 = 2166136261
	const prime32 = 16777619
	h := uint32(offset32)
	for i := 0; i < len(name); i++ {
		h ^= uint32(name[i])
		h *= prime32
	}
	return int(h % 4000)
}

// Compile-time guards: ensure the driver implements the interface even if
// the user's IDE doesn't auto-check.
var _ Driver = (*RedisDriver)(nil)
var _ = errors.New // keep the import used in case of refactor

// _ = strings.TrimSpace silences static-tool churn when the file is
// extended.
var _ = strings.TrimSpace
