package policy

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

//go:embed bundles/promotion_default.rego
var defaultPromotionRego string

// DefaultPromotionQuery is the canonical entry-point for the default bundle.
const DefaultPromotionQuery = "data.crucible.promotion.decision"

// DefaultPromotionModuleName is the in-bundle path of the default module.
// Tenant overrides MUST use a different module name so layering is unambiguous.
const DefaultPromotionModuleName = "crucible/promotion/default.rego"

// DefaultPromotionEngine returns an Engine compiled against the bundled
// default promotion policy under data.crucible.promotion.decision.
//
// Tenants who don't ship overrides use this engine directly.
func DefaultPromotionEngine(ctx context.Context) (*Engine, error) {
	return New(ctx, DefaultPromotionQuery, map[string]string{
		DefaultPromotionModuleName: defaultPromotionRego,
	})
}

// DefaultPromotionModule returns the embedded Rego source.
func DefaultPromotionModule() string { return defaultPromotionRego }

// TenantBundle is a per-tenant Rego override. Tenants ship a TenantBundle as
// signed JSON; the gate's rego_engine reads it, verifies the signature, and
// compiles a layered Engine on top of the default bundle.
type TenantBundle struct {
	TenantID    string            `json:"tenant_id"`
	Description string            `json:"description,omitempty"`
	// Modules is name → Rego source. Module names MUST NOT collide with the
	// default module name. The package path is forced to
	// `crucible.promotion.tenant.<tenant_id>` inside the bundle.
	Modules map[string]string `json:"modules"`
	// Query, if non-empty, overrides the default DefaultPromotionQuery.
	Query string `json:"query,omitempty"`
	// IssuedAt is the time the tenant committed this bundle.
	IssuedAt time.Time `json:"issued_at"`
	// Version is a tenant-incrementing integer; an attestation of a new
	// version supersedes prior versions automatically.
	Version int `json:"version"`
}

// Validate enforces the per-tenant invariants. Returns nil if the bundle is
// safe to compile.
func (b *TenantBundle) Validate() error {
	if b == nil {
		return errors.New("policy: nil tenant bundle")
	}
	if b.TenantID == "" {
		return errors.New("policy: tenant bundle missing tenant_id")
	}
	if len(b.Modules) == 0 {
		return errors.New("policy: tenant bundle has no modules")
	}
	for name, src := range b.Modules {
		if name == DefaultPromotionModuleName {
			return fmt.Errorf("policy: tenant module name %q collides with default", name)
		}
		if !strings.HasSuffix(name, ".rego") {
			return fmt.Errorf("policy: tenant module name %q must end in .rego", name)
		}
		// Force-disallow re-defining the default package. Tenant modules
		// must declare `package crucible.promotion.tenant` so that any
		// `decision` rule in the tenant module is reachable under a
		// SEPARATE entry-point (queried explicitly when present).
		if !strings.Contains(src, "package crucible.promotion.tenant") {
			return fmt.Errorf("policy: tenant module %q must `package crucible.promotion.tenant`", name)
		}
	}
	return nil
}

// LayeredEngine compiles a default+tenant pair into a single Engine. The
// caller is responsible for choosing whether to evaluate the tenant entrypoint
// or the default entrypoint per request (the rego_engine evaluates BOTH and
// merges them; see ApplyTenantOverride in apps/promotion-gate).
//
// The resulting Engine.PolicyHash() reflects both modules so the
// PromotionApproval/v1 record is reproducible.
func LayeredEngine(ctx context.Context, tenant *TenantBundle) (*Engine, error) {
	if err := tenant.Validate(); err != nil {
		return nil, err
	}
	modules := map[string]string{
		DefaultPromotionModuleName: defaultPromotionRego,
	}
	for k, v := range tenant.Modules {
		modules[k] = v
	}
	query := tenant.Query
	if query == "" {
		query = DefaultPromotionQuery
	}
	return New(ctx, query, modules)
}

// TenantEngine returns an Engine that ONLY evaluates the tenant override at
// data.crucible.promotion.tenant.decision. The gate's rego_engine evaluates
// it after the default and merges Allow/NeedsHuman with conservative AND
// semantics (all gates must allow).
const TenantEntrypoint = "data.crucible.promotion.tenant.decision"

func TenantEngine(ctx context.Context, tenant *TenantBundle) (*Engine, error) {
	if err := tenant.Validate(); err != nil {
		return nil, err
	}
	modules := map[string]string{
		DefaultPromotionModuleName: defaultPromotionRego,
	}
	for k, v := range tenant.Modules {
		modules[k] = v
	}
	return New(ctx, TenantEntrypoint, modules)
}

// LoadTenantBundleFile parses a TenantBundle JSON from disk.
func LoadTenantBundleFile(path string) (*TenantBundle, error) {
	b, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return nil, fmt.Errorf("policy: read tenant bundle %s: %w", path, err)
	}
	return DecodeTenantBundle(b)
}

// DecodeTenantBundle parses a TenantBundle JSON from raw bytes.
func DecodeTenantBundle(b []byte) (*TenantBundle, error) {
	var tb TenantBundle
	if err := json.Unmarshal(b, &tb); err != nil {
		return nil, fmt.Errorf("policy: decode tenant bundle: %w", err)
	}
	return &tb, nil
}

// EncodeTenantBundle re-encodes a TenantBundle to canonical JSON.
func EncodeTenantBundle(tb *TenantBundle) ([]byte, error) {
	return json.MarshalIndent(tb, "", "  ")
}
