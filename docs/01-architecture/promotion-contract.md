# Promotion Contract

The bridge from twin to real. Every verified change becomes a real-system change only via this contract.

## The contract, in steps

When the agent calls `twin.promote(bundle)`:

```
1. Provenance verification
   ├─ Every in-toto attestation in the bundle is checked against Sigstore Rekor
   ├─ OIDC subjects of all signers are validated against the allowed set
   └─ Bundle is rejected if any attestation is missing or invalid

2. Rego policy evaluation
   ├─ Trivial diffs (Tier 0 verified, no schema change, no critical paths)
   │     → Auto-approve
   ├─ Schema changes, critical-path touches, first-time author-of-area
   │     → Require human approval
   └─ Policy violations (e.g., missing CODEOWNER approval)
         → Reject

3. Human approval (if required)
   ├─ Slack button or web UI
   ├─ Approver signs via Sigstore keyless OIDC
   └─ Approval attestation published to Rekor

4. KMS-signed credential lease
   ├─ AWS KMS / GCP Cloud HSM / YubiHSM signs a single-use credential
   ├─ Scoped to the specific action (deploy this artifact, run this migration)
   ├─ Time-boxed (typical: 5 min)
   └─ Consumed by the deploy pipeline, never returned to the agent

5. Progressive rollout
   ├─ Argo Rollouts canary (Kubernetes) or Flagger (Linkerd/Istio)
   ├─ Traffic mirroring to the new version
   ├─ AnalysisTemplate watches Prometheus SLOs
   ├─ GrowthBook feature flag for fast rollback (millisecond)
   └─ Auto-rollback on SLO regression

6. Final attestation
   ├─ Promotion outcome (success / rolled-back) published to Rekor
   ├─ Procedural memory updated with patterns learned
   └─ Task marked complete
```

## The `PromotionBundle`

The artifact the agent produces and the contract consumes:

```typescript
PromotionBundle {
  task_id: string,
  diff_hash: string,                       // hash of all file changes
  files_changed: { path: string, action: "add" | "modify" | "delete" }[],

  // Verifier output
  verifier_approval: VerifierApproval,     // signed by verifier OIDC
  tier_results: TierResults,

  // Provenance
  attestations: RekorUUID[],               // every action in the task
  build_provenance: SLSAProvenance,        // SLSA-L3 attestation
  rebuild_hash: string,                    // hermetic Nix/Bazel hash

  // Risk & impact
  blast_radius: {
    affected_services: string[],
    affected_endpoints: string[],
    schema_changes: SchemaChange[],
    critical_paths_touched: string[],
    estimated_impact: "low" | "medium" | "high"
  },

  // Deploy plan
  suggested_rollout: {
    strategy: "canary" | "blue-green" | "feature-flag-only",
    canary_percentages: number[],          // e.g. [1, 5, 25, 100]
    dwell_seconds_per_step: number,
    analysis_template_ref: string,         // points to Prometheus rules
    rollback_trigger: string               // e.g. "error_rate_p99 > 0.5%"
  },

  // Signing
  agent_oidc_subject: string,
  signed_at: timestamp,
}
```

## Rego policy structure

The default policy bundle (per-tenant overridable):

```rego
package crucible.promotion

default allow = false
default require_human = false

# Auto-approve trivial
allow {
  input.tier_results.tier_0.passed
  not has_schema_change
  not has_critical_path
  input.blast_radius.estimated_impact == "low"
}

# Schema changes need human approval
require_human {
  has_schema_change
}

# Critical-path changes need human approval AND CODEOWNER signature
require_human {
  has_critical_path
}

require_codeowner {
  has_critical_path
}

has_schema_change {
  count(input.blast_radius.schema_changes) > 0
}

has_critical_path {
  count(input.blast_radius.critical_paths_touched) > 0
}

# Reject if Tier 4 didn't run on production-touching changes
deny[msg] {
  not input.tier_results.tier_4.passed
  input.blast_radius.estimated_impact != "low"
  msg := "Tier 4 reproducible-build attestation required for non-trivial promotions"
}
```

Customers can layer their own rules — e.g., "no promotions during merge freeze," "deploys to prod-eu require EU-based approver," etc.

## Progressive rollout

### Kubernetes (Argo Rollouts)

Default rollout strategy uses Argo Rollouts `AnalysisTemplate` against Prometheus:

```yaml
strategy:
  canary:
    steps:
      - setWeight: 1
      - pause: { duration: 5m }
      - analysis:
          templates:
            - templateName: crucible-slo-check
      - setWeight: 5
      - pause: { duration: 10m }
      - analysis: { ... }
      - setWeight: 25
      - pause: { duration: 30m }
      - analysis: { ... }
      - setWeight: 100

analysisRunMetadata:
  rollout: from-crucible
  task_id: <task_id>
```

Auto-rollback on:
- Error rate p99 > pre-rollout baseline × 1.5
- Latency p95 > baseline × 1.3
- Custom rules per service (defined in the task manifest)

### Non-K8s (serverless / VM-based)

Feature-flag-driven rollouts via GrowthBook:

- Flag created at promotion time, scoped to the change.
- Initial rollout: 1% of users.
- Periodic SLO check via Prometheus query (configurable per service).
- Step up percentage on clean dwell.
- Flag flip to 0% on regression (millisecond rollback).

### Database migrations

Special handling:

1. **Twin run** — migration applied to Neon twin branch; verifier checks resulting schema diff against expected.
2. **Shadow run** — same migration applied to a shadow of production (read-replica with replication paused), verifier checks no destructive DDL on production data.
3. **Promotion** — KMS-signed credential lease grants temporary `ALTER TABLE` permission; migration runs as a single transaction with statement timeout.
4. **Verification** — post-migration query checks (data integrity, row counts, expected indexes).
5. **Rollback** — if any check fails, transaction rolls back. For non-transactional DDL (e.g., MySQL pre-8.0), a manually-authored down-migration is required as part of the bundle.

## KMS signing

The "unseal ceremony" for destructive prod actions:

```
1. Approver clicks Slack button or web UI button
2. Sigstore keyless OIDC issues a short-lived cert for the approver
3. Crucible signs the action request with the OIDC cert
4. AWS KMS / GCP Cloud HSM / YubiHSM verifies the cert + signs a credential lease
5. Credential lease:
   - scoped to the specific action (e.g., "deploy artifact X to service Y")
   - time-boxed (5 minutes default)
   - single-use (idempotency key consumed on first use)
   - NEVER returned to the agent process
6. Deploy pipeline consumes lease, executes action, returns result
7. Lease automatically expires
```

For air-gapped / on-prem deployments, KMS is replaced by an on-prem HSM (e.g., Thales Luna, YubiHSM, AWS CloudHSM standalone).

## Approval routing

Who approves what is configured per-tenant:

```yaml
# tenant approval policy
default_approvers: ["@platform-team"]
overrides:
  - matches:
      schema_changes: true
    approvers: ["@dba-team"]
  - matches:
      critical_paths_touched: ["src/billing/*"]
    approvers: ["@payments-leads"]
    require_codeowner: true
  - matches:
      blast_radius.estimated_impact: "high"
    approvers: ["@on-call", "@eng-leadership"]
    require_n_approvers: 2
```

All approvals are signed and published to Rekor. Audit log is therefore queryable: "show me all critical-path deploys in the last 30 days and who approved each."

## Failure handling

### Promotion bundle rejected at policy gate

Returned to agent with structured rejection. Agent surfaces to user. If recoverable (e.g., missing CODEOWNER signature), user can add the missing approval and retry. If not, the change is held in the bundle store for the user to amend.

### Approval timeout

Configurable per-tenant. Default: bundle expires after 24 hours of waiting for approval. User can extend or refresh.

### Canary regression

Auto-rollback fires. Bundle marked `rolled_back`. Procedural memory records the failure pattern. Agent receives a structured "rollback report" with the SLO that triggered it, the diff, and the regression metrics — usable input for a retry task.

### Partial promotion (e.g., 2 of 3 services deployed, 3rd fails)

The promotion contract is atomic per-bundle. If any sub-deploy fails, **all** deploys in the bundle roll back. This is the difference between "deploy script" and "promotion contract."

## What we explicitly will not allow

- **Direct production access from the agent process.** Ever. The only path is through the KMS-signed credential lease.
- **Approval bypass for "emergencies."** If something is on fire, an approver clicks the button. Bypass paths are how trust dies.
- **Self-approval.** An agent that proposes a promotion cannot approve it. Different OIDC subjects required.
- **Stale approvals.** An approval is valid for one specific bundle hash. Any diff change invalidates it.

These are non-negotiable architectural invariants. See [threat-model.md](threat-model.md).
