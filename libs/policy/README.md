# policy

Embedded OPA / Rego policy engine that drives the **Phase-6 promotion gate**.

**Import path:** `github.com/open-policy-agent/opa/v1/rego`. The legacy `github.com/open-policy-agent/opa/rego` is deprecated and not carried forward.

## What's here

| File | Purpose |
|---|---|
| `policy.go` | `Engine`, `Decision`, `New`, `HashModules` — the embedded-OPA wrapper. |
| `bundle.go` | `TenantBundle`, `LoadTenantBundleFile`, `LayeredEngine`, `TenantEngine`. |
| `signed_bundle.go` | `SignedTenantBundle`, `SignBundle`, `VerifyBundle`, `Ed25519Signer`. |
| `input.go` | `PromotionInput` — the canonical Rego input doc the gate builds. |
| `bundles/promotion_default.rego` | The default policy bundle (the full spec from `docs/01-architecture/promotion-contract.md`). |
| `bundles/tenant_example.rego` | Reference tenant override (prod-eu geo, billing/ codeowners). |

## Decision shape

The default bundle returns:

```json
{
  "allow":               bool,
  "needs_human":         bool,
  "reasons":             ["..."],
  "require_codeowner":   bool,
  "approver_groups":     ["@platform-team", "@payments-leads"],
  "require_n_approvers": 2,
  "auto_approve":        bool,
  "trace":               { "path": "human.critical_path" }
}
```

The `trace.path` lets the audit log explain which rule fired. Every successful
evaluation produces a signed `PromotionApproval/v1` whose
`rego_policy_hash` field is `Engine.PolicyHash()` (sha256 of the sorted source
modules).

## Usage — default bundle

```go
eng, _ := policy.DefaultPromotionEngine(ctx)
dec, _ := eng.Evaluate(ctx, policy.PromotionInput{
    TaskID: "task_demo", TenantID: "ten_demo",
    VerifierApprovalAttestation: "rekor:abc",
    BlastRadius: policy.PromotionBlastRadius{
        EstimatedImpact: "low",
        Reversibility:   "trivial",
    },
    TierResults: policy.PromotionTierResults{
        Tier0: &policy.TierEntry{Passed: true},
        Tier1: &policy.TierEntry{Passed: true},
        Tier4: &policy.TierEntry{Passed: true},
    },
})
if !dec.Allow {
    log.Println("denied:", dec.Reasons)
}
```

## Usage — tenant override

```go
tb, _ := policy.LoadTenantBundleFile("tenants/acme/policy.json")
eng, _ := policy.LayeredEngine(ctx, tb)
```

A tenant module **MUST** declare `package crucible.promotion.tenant`. Tenant
modules CANNOT redefine the default package. The promotion gate evaluates the
default and the tenant entrypoint and merges with conservative AND semantics.

## Signed bundles

Every tenant override is a signed artifact:

```go
signer, _ := policy.NewEd25519Signer("https://accounts.crucible.dev/tenants/ten_acme")
env, _   := policy.SignBundle(tb, signer)
// distribute env.JSON; gate reads + verifies before compiling
got, err := policy.VerifyBundle(env, signer)
```

In production the `Signer`/`VerifierBytes` adapters wrap the Sigstore keyless
cert chain so the OIDC subject of the tenant admin who edited the bundle is
in the audit log.

## Guardrails enforced by the default bundle

| Guardrail | Phase-6 owner | Where in the rego |
|---|---|---|
| Self-approval forbidden | `deny.self_approval` | agent_oidc must differ from every approver_oidc |
| Merge freeze blocks all non-test promotions | `deny.merge_freeze` | `input.context.merge_freeze == true` |
| Tier-4 required for non-trivial | `deny.tier4_missing` | `estimated_impact != "low"` |
| Irreversible without recorded human approval | `deny.irreversible_without_human` | `reversibility == "irreversible"` |
| Geo-restricted approvers | `deny.geo` | `tenant_overrides.deny_geo` |
| Schema change | `human.schema_change` | `count(schema_changes) > 0` |
| Critical-path touch + CODEOWNER | `human.critical_path` | `count(critical_paths_touched) > 0` |
| High-impact / lossy / snapshot | `human.high_impact` | `estimated_impact == "high"` etc. |

See `bundles/promotion_default.rego` for the source-of-truth bytes.
