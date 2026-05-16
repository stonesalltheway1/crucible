// Package tenantpolicy holds per-tenant configuration: routing overrides,
// privacy/data-residency rules, default budget caps, allowed vendors.
//
// Phase 1 keeps the loader in-memory with sensible defaults. Phase 2 will back
// it with Postgres + Redis cache + a webhook for live reloads.
package tenantpolicy

import (
	"errors"
	"strings"
	"sync"

	"github.com/crucible/control-plane/internal/modelrouter"
)

// Residency narrows which model vendors / regions a tenant is allowed to use.
type Residency string

const (
	ResidencyStandard       Residency = "standard"
	ResidencyEU             Residency = "eu"
	ResidencyHIPAA          Residency = "hipaa"
	ResidencyAirgap         Residency = "airgap"
)

// Policy is one tenant's effective configuration.
type Policy struct {
	TenantID                   string
	Residency                  Residency
	AllowedVendors             []modelrouter.Vendor
	ModelOverrides             map[modelrouter.ModelTier]string // tier → model id
	DefaultCostCapUSD          float64
	DefaultWallClockCapMin     uint32
	DefaultRetryCapPerSubgoal  uint32
	CriticalWallClockCapMin    uint32
}

// Default returns the Phase-1 default policy for a fresh tenant.
func Default(tenantID string) Policy {
	return Policy{
		TenantID:                   tenantID,
		Residency:                  ResidencyStandard,
		AllowedVendors:             []modelrouter.Vendor{modelrouter.VendorAnthropic, modelrouter.VendorGoogle, modelrouter.VendorOpenAI},
		ModelOverrides:             map[modelrouter.ModelTier]string{},
		DefaultCostCapUSD:          2.0,
		DefaultWallClockCapMin:     60,
		DefaultRetryCapPerSubgoal:  3,
		CriticalWallClockCapMin:    240,
	}
}

// AllowsVendor returns true if the policy permits routing to the given vendor.
func (p Policy) AllowsVendor(v modelrouter.Vendor) bool {
	if len(p.AllowedVendors) == 0 {
		return true
	}
	for _, allowed := range p.AllowedVendors {
		if allowed == v {
			return true
		}
	}
	return false
}

// Override returns the tenant's model override for a tier, or empty string.
func (p Policy) Override(t modelrouter.ModelTier) string {
	if p.ModelOverrides == nil {
		return ""
	}
	return p.ModelOverrides[t]
}

// Loader caches per-tenant policies.
type Loader struct {
	mu       sync.RWMutex
	policies map[string]Policy
}

// NewLoader returns an empty loader. Use Set to seed policies; Get falls back
// to Default for unknown tenants.
func NewLoader() *Loader {
	return &Loader{policies: map[string]Policy{}}
}

// Set stores or replaces a tenant policy.
func (l *Loader) Set(p Policy) error {
	if p.TenantID == "" {
		return errors.New("tenantpolicy: empty tenant id")
	}
	for _, v := range p.AllowedVendors {
		if strings.TrimSpace(string(v)) == "" {
			return errors.New("tenantpolicy: empty vendor in AllowedVendors")
		}
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.policies[p.TenantID] = p
	return nil
}

// Get returns the policy for a tenant. Unknown tenants get the default.
func (l *Loader) Get(tenantID string) Policy {
	l.mu.RLock()
	defer l.mu.RUnlock()
	if p, ok := l.policies[tenantID]; ok {
		return p
	}
	return Default(tenantID)
}

// All returns a snapshot of every loaded policy.
func (l *Loader) All() []Policy {
	l.mu.RLock()
	defer l.mu.RUnlock()
	out := make([]Policy, 0, len(l.policies))
	for _, p := range l.policies {
		out = append(out, p)
	}
	return out
}
