// Package policy is the OPA-embedded Rego loader for Crucible.
//
// Phase 6 ships the full promotion-policy surface:
//
//   - The default Rego bundle lives in libs/policy/bundles/promotion_default.rego.
//     The compiled engine is exposed via DefaultPromotionEngine.
//   - Per-tenant override modules are layered on top via LoadTenantBundle —
//     they are evaluated against the same input as the default and their
//     decision wins when present.
//   - Every policy bundle (default or tenant) is content-addressed; the
//     resulting hash goes into the PromotionApproval/v1 attestation as
//     `rego_policy_hash`.
//   - SignBundle / VerifyBundle wrap a SignedBundle in a DSSE envelope using
//     the same Signer interface as libs/attestation, so a tenant's policy
//     bundle is itself an attestable artifact.
//
// CRITICAL: This package uses the v1 OPA module path
// `github.com/open-policy-agent/opa/v1/rego`, NOT the legacy
// `github.com/open-policy-agent/opa/rego`. The v1 path is the post-1.0
// canonical path and is required for any 2026 build.
package policy

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"

	rego "github.com/open-policy-agent/opa/v1/rego"
)

// Decision is the outcome of evaluating a policy against an input document.
//
// The default promotion bundle returns a structured doc; Engine.Evaluate maps
// that to this struct. Approver groups and N-of-M live in Decision so the
// approval_router doesn't have to re-read the Rego output.
type Decision struct {
	Allow             bool     `json:"allow"`
	NeedsHuman        bool     `json:"needs_human"`
	Reasons           []string `json:"reasons,omitempty"`
	RequireCodeowner  bool     `json:"require_codeowner"`
	ApproverGroups    []string `json:"approver_groups,omitempty"`
	RequireNApprovers int      `json:"require_n_approvers"`
	AutoApprove       bool     `json:"auto_approve"`
	Trace             any      `json:"trace,omitempty"`
	// Raw is the unflattened evaluation result. The Decision attestation
	// publishes this as `rego_decision_doc` so future-you can replay the
	// audit trail.
	Raw any `json:"raw,omitempty"`
}

// MatchesAllowSelfApprove is a convenience for downstream checks.
func (d Decision) RequiresHuman() bool { return d.NeedsHuman && !d.AutoApprove }

// Engine wraps a prepared rego.PreparedEvalQuery so policies are compiled
// once at load time and evaluated repeatedly.
type Engine struct {
	prepared rego.PreparedEvalQuery
	query    string
	modules  map[string]string
	hash     string
}

// New compiles one or more Rego modules under the given query (e.g.
// "data.crucible.promotion.decision") and returns an Engine ready for
// repeated Evaluate calls.
//
// modules is a map from module name → module source.
func New(ctx context.Context, query string, modules map[string]string) (*Engine, error) {
	if query == "" {
		return nil, errors.New("policy: empty query")
	}
	if len(modules) == 0 {
		return nil, errors.New("policy: no modules provided")
	}
	opts := []func(*rego.Rego){rego.Query(query)}
	for name, src := range modules {
		opts = append(opts, rego.Module(name, src))
	}
	r := rego.New(opts...)
	prepared, err := r.PrepareForEval(ctx)
	if err != nil {
		return nil, fmt.Errorf("policy: prepare: %w", err)
	}
	cloned := make(map[string]string, len(modules))
	for k, v := range modules {
		cloned[k] = v
	}
	return &Engine{
		prepared: prepared,
		query:    query,
		modules:  cloned,
		hash:     HashModules(cloned),
	}, nil
}

// Query returns the prepared eval query string (e.g.
// data.crucible.promotion.decision).
func (e *Engine) Query() string { return e.query }

// PolicyHash is the content-addressed hash of the compiled modules. Goes into
// PromotionApproval/v1's rego_policy_hash field.
func (e *Engine) PolicyHash() string { return e.hash }

// Modules returns a copy of the source modules currently compiled in.
func (e *Engine) Modules() map[string]string {
	out := make(map[string]string, len(e.modules))
	for k, v := range e.modules {
		out[k] = v
	}
	return out
}

// Evaluate runs the prepared query against input and returns a Decision.
//
// The query result may be:
//   - a bool  → only Allow is set.
//   - a map[string]any with the canonical Crucible decision fields.
//
// Anything else returns an error.
func (e *Engine) Evaluate(ctx context.Context, input any) (*Decision, error) {
	rs, err := e.prepared.Eval(ctx, rego.EvalInput(input))
	if err != nil {
		return nil, fmt.Errorf("policy: eval: %w", err)
	}
	if len(rs) == 0 || len(rs[0].Expressions) == 0 {
		return &Decision{Allow: false, NeedsHuman: true, Reasons: []string{"policy: no rule matched"}}, nil
	}

	val := rs[0].Expressions[0].Value
	switch v := val.(type) {
	case bool:
		dec := &Decision{Allow: v, Raw: v}
		if !v {
			dec.NeedsHuman = true
			dec.Reasons = []string{"policy: rule returned false"}
		}
		return dec, nil
	case map[string]any:
		return mapToDecision(v), nil
	default:
		return nil, fmt.Errorf("policy: unexpected eval result type %T", val)
	}
}

func mapToDecision(v map[string]any) *Decision {
	dec := &Decision{Raw: v}
	if allow, ok := v["allow"].(bool); ok {
		dec.Allow = allow
	}
	if nh, ok := v["needs_human"].(bool); ok {
		dec.NeedsHuman = nh
	}
	if reasons, ok := v["reasons"].([]any); ok {
		for _, r := range reasons {
			if s, ok := r.(string); ok {
				dec.Reasons = append(dec.Reasons, s)
			}
		}
	}
	if rc, ok := v["require_codeowner"].(bool); ok {
		dec.RequireCodeowner = rc
	}
	if groups, ok := v["approver_groups"].([]any); ok {
		for _, g := range groups {
			if s, ok := g.(string); ok {
				dec.ApproverGroups = append(dec.ApproverGroups, s)
			}
		}
	}
	if n, ok := v["require_n_approvers"].(float64); ok {
		dec.RequireNApprovers = int(n)
	}
	if n, ok := v["require_n_approvers"].(int); ok {
		dec.RequireNApprovers = n
	}
	if aa, ok := v["auto_approve"].(bool); ok {
		dec.AutoApprove = aa
	}
	if t, ok := v["trace"]; ok {
		dec.Trace = t
	}
	return dec
}

// HashModules returns a stable sha256 of the (sorted by name) module sources.
// This is what goes into PromotionApproval/v1.rego_policy_hash so an auditor
// in 2056 can reproduce the bytes that produced any historical decision.
func HashModules(modules map[string]string) string {
	keys := make([]string, 0, len(modules))
	for k := range modules {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	h := sha256.New()
	for _, k := range keys {
		h.Write([]byte(k))
		h.Write([]byte{0})
		h.Write([]byte(modules[k]))
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))
}
