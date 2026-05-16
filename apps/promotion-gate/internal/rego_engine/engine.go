// Package rego_engine drives the embedded OPA policy evaluation.
//
// The engine layers two policies:
//
//   - The default bundle compiled into libs/policy.
//   - An optional per-tenant signed override.
//
// Evaluate runs both and merges the decisions: a promotion is allowed only
// when BOTH layers allow (conservative AND). NeedsHuman is the OR.
//
// Every successful evaluation produces a `decision` document the gate signs
// into a PromotionApproval/v1 attestation via the relay.
package rego_engine

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/crucible/policy"
)

// MergedDecision combines the default + tenant decisions.
type MergedDecision struct {
	Allow             bool            `json:"allow"`
	NeedsHuman        bool            `json:"needs_human"`
	AutoApprove       bool            `json:"auto_approve"`
	RequireCodeowner  bool            `json:"require_codeowner"`
	ApproverGroups    []string        `json:"approver_groups,omitempty"`
	RequireNApprovers int             `json:"require_n_approvers"`
	Reasons           []string        `json:"reasons,omitempty"`
	DefaultDecision   policy.Decision `json:"default_decision"`
	TenantDecision    *policy.Decision `json:"tenant_decision,omitempty"`
	PolicyHash        string          `json:"policy_hash"`
	EvaluatedAt       time.Time       `json:"evaluated_at"`
}

// IsApproved is true when the merged decision is `allow && !needs_human`.
func (m *MergedDecision) IsApproved() bool { return m.Allow && !m.NeedsHuman }

// Engine wraps the default engine plus a tenant cache.
type Engine struct {
	defaultEng *policy.Engine
	tenantMu   sync.RWMutex
	tenantEng  map[string]*policy.Engine // tenant_id → compiled tenant engine
	tenantVer  map[string]int            // tenant_id → bundle version (for cache invalidation)
}

// New builds an Engine with the bundled default policy compiled in.
func New(ctx context.Context) (*Engine, error) {
	def, err := policy.DefaultPromotionEngine(ctx)
	if err != nil {
		return nil, fmt.Errorf("rego_engine: compile default bundle: %w", err)
	}
	return &Engine{
		defaultEng: def,
		tenantEng:  map[string]*policy.Engine{},
		tenantVer:  map[string]int{},
	}, nil
}

// DefaultPolicyHash returns the hash of the compiled default bundle.
func (e *Engine) DefaultPolicyHash() string { return e.defaultEng.PolicyHash() }

// LoadTenant compiles + caches a per-tenant override bundle.
//
// If the bundle's version matches the cached version, this is a no-op.
// Callers MUST pre-verify the SignedTenantBundle via policy.VerifyBundle
// before calling LoadTenant; this function trusts the bytes it gets.
func (e *Engine) LoadTenant(ctx context.Context, tb *policy.TenantBundle) error {
	if tb == nil {
		return errors.New("rego_engine: nil tenant bundle")
	}
	if err := tb.Validate(); err != nil {
		return fmt.Errorf("rego_engine: tenant bundle invalid: %w", err)
	}
	e.tenantMu.RLock()
	if v, ok := e.tenantVer[tb.TenantID]; ok && v >= tb.Version {
		e.tenantMu.RUnlock()
		return nil
	}
	e.tenantMu.RUnlock()
	eng, err := policy.TenantEngine(ctx, tb)
	if err != nil {
		return fmt.Errorf("rego_engine: compile tenant: %w", err)
	}
	e.tenantMu.Lock()
	defer e.tenantMu.Unlock()
	e.tenantEng[tb.TenantID] = eng
	e.tenantVer[tb.TenantID] = tb.Version
	return nil
}

// HasTenant reports whether a tenant policy is currently loaded.
func (e *Engine) HasTenant(tenantID string) bool {
	e.tenantMu.RLock()
	defer e.tenantMu.RUnlock()
	_, ok := e.tenantEng[tenantID]
	return ok
}

// Evaluate runs default + tenant policies and merges.
func (e *Engine) Evaluate(ctx context.Context, input *policy.PromotionInput) (*MergedDecision, error) {
	if input == nil {
		return nil, errors.New("rego_engine: nil input")
	}
	doc, err := inputToMap(input)
	if err != nil {
		return nil, err
	}

	def, err := e.defaultEng.Evaluate(ctx, doc)
	if err != nil {
		return nil, fmt.Errorf("rego_engine: default eval: %w", err)
	}

	merged := &MergedDecision{
		Allow:             def.Allow,
		NeedsHuman:        def.NeedsHuman,
		AutoApprove:       def.AutoApprove,
		RequireCodeowner:  def.RequireCodeowner,
		ApproverGroups:    def.ApproverGroups,
		RequireNApprovers: def.RequireNApprovers,
		Reasons:           append([]string{}, def.Reasons...),
		DefaultDecision:   *def,
		PolicyHash:        e.defaultEng.PolicyHash(),
		EvaluatedAt:       time.Now().UTC(),
	}

	if input.TenantID != "" {
		e.tenantMu.RLock()
		tenantEng := e.tenantEng[input.TenantID]
		e.tenantMu.RUnlock()
		if tenantEng != nil {
			td, err := tenantEng.Evaluate(ctx, doc)
			if err != nil {
				return nil, fmt.Errorf("rego_engine: tenant eval (%s): %w", input.TenantID, err)
			}
			merged.TenantDecision = td
			merged.PolicyHash = tenantEng.PolicyHash()
			merged.Allow = merged.Allow && td.Allow
			merged.NeedsHuman = merged.NeedsHuman || td.NeedsHuman
			merged.AutoApprove = merged.AutoApprove && td.AutoApprove && !td.NeedsHuman
			merged.RequireCodeowner = merged.RequireCodeowner || td.RequireCodeowner
			if td.RequireNApprovers > merged.RequireNApprovers {
				merged.RequireNApprovers = td.RequireNApprovers
			}
			merged.ApproverGroups = mergeApprovers(merged.ApproverGroups, td.ApproverGroups)
			for _, r := range td.Reasons {
				merged.Reasons = append(merged.Reasons, "tenant: "+r)
			}
		}
	}

	return merged, nil
}

func mergeApprovers(a, b []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(a)+len(b))
	for _, x := range append(append([]string{}, a...), b...) {
		if _, ok := seen[x]; ok {
			continue
		}
		seen[x] = struct{}{}
		out = append(out, x)
	}
	return out
}

func inputToMap(p *policy.PromotionInput) (map[string]any, error) {
	b, err := json.Marshal(p)
	if err != nil {
		return nil, fmt.Errorf("rego_engine: marshal input: %w", err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, fmt.Errorf("rego_engine: unmarshal input: %w", err)
	}
	return m, nil
}
