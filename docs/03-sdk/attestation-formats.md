# Attestation Formats

Every action Crucible takes is captured as an in-toto attestation, signed via Sigstore keyless OIDC, and published to Sigstore Rekor v2. This doc is the schema reference.

## What is an attestation, in our context

```
┌─────────────────────────────────────────────────────────────┐
│  in-toto statement                                          │
│  ┌────────────────────────────────────────────────────────┐ │
│  │  _type:        "https://in-toto.io/Statement/v1"       │ │
│  │  subject:      [{ name, digest }, ...]                 │ │
│  │  predicateType: "https://crucible.dev/<type>/v1"       │ │
│  │  predicate:    <typed payload>                         │ │
│  └────────────────────────────────────────────────────────┘ │
│  signed via DSSE envelope, OIDC subject = agent worker ID   │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
                    Sigstore Rekor v2 entry
                    (returns rekor:uuid, public)
```

The `predicate` is the payload. The `subject` references the thing being attested (file, diff, build, decision). Crucible defines several predicate types, all under the `https://crucible.dev/` namespace.

## Predicate types

### `https://crucible.dev/WriteAttestation/v1`

Emitted on every `twin.fs.write` and `twin.fs.delete`.

```json
{
  "task_id": "task_01HZ...",
  "step_id": "step_03",
  "tenant_id": "ten_...",
  "repo": "github.com/acme/payments",
  "base_sha": "abcd1234...",
  "path": "api/webhooks.ts",
  "action": "modify",
  "content_sha256": "0xabc...",
  "size_bytes": 4823,
  "timestamp": "2026-05-15T18:24:31.012Z",
  "agent_oidc_subject": "https://accounts.crucible.dev/agents/..."
}
```

### `https://crucible.dev/MigrationAttestation/v1`

Emitted on `twin.db.migrate`.

```json
{
  "task_id": "...",
  "tenant_id": "...",
  "migration_file": "db/migrations/20260515_refunds.sql",
  "migration_sha256": "...",
  "schema_diff": {
    "added_tables": ["refunds"],
    "modified_tables": [],
    "dropped_tables": [],
    "added_columns": [],
    "destructive_ddl": false
  },
  "row_count_change": {"refunds": "+0"},
  "applied_at": "...",
  "neon_branch_id": "br_...",
  "agent_oidc_subject": "..."
}
```

### `https://crucible.dev/ServiceCallAttestation/v1`

Emitted on every `twin.svc.call`.

```json
{
  "task_id": "...",
  "tenant_id": "...",
  "service": "stripe",
  "endpoint": "/v1/charges",
  "method": "GET",
  "request_hash": "0x...",
  "response_hash": "0x...",
  "tape_disposition": "hit-exact",
  "x_crucible_tape": "hit-exact",
  "duration_ms": 12,
  "secrets_used": ["stripe_test_key"],
  "agent_oidc_subject": "..."
}
```

### `https://crucible.dev/DestructiveProposal/v1`

Emitted whenever the syscall shim intercepts a destructive command.

```json
{
  "task_id": "...",
  "tenant_id": "...",
  "command": "DROP TABLE users_archived",
  "scope": "twin",
  "justification": "cleaning up unused archive table",
  "blast_radius": {
    "affected_resources": ["table:users_archived"],
    "reversibility": "snapshot",
    "impact_score": 0.4
  },
  "intercepted_at_layer": "syscall-shim",
  "agent_oidc_subject": "..."
}
```

### `https://crucible.dev/DestructiveApproval/v1`

Emitted when a destructive proposal is approved (twin-scoped auto-approval or real-scoped human approval).

```json
{
  "proposal_attestation": "rekor:...",
  "approval_kind": "auto-twin" | "human-real",
  "approver_oidc_subject": "...",
  "approved_at": "...",
  "approval_attestation_id": "..."
}
```

### `https://crucible.dev/TestReport/v1`

Emitted on `twin.test.run` and the per-tier verifier methods.

```json
{
  "task_id": "...",
  "test_kind": "tier_0_mutation" | "tier_1_pbt" | "tier_2_contract" | "tier_3_proof" | "tier_4_honest_ci" | "project_native",
  "framework": "stryker-js",
  "passed": true,
  "stats": {
    "killed": 91,
    "survived": 9,
    "score": 0.91,
    "iterations": 10000,
    "counterexamples": []
  },
  "duration_seconds": 47.3,
  "verifier_model": "gemini-3.1-pro",
  "verifier_oidc_subject": "https://accounts.crucible.dev/verifiers/..."
}
```

### `https://crucible.dev/VerifierApproval/v1` / `VerifierRejection/v1`

The verifier's final verdict for a task.

```json
{
  "task_id": "...",
  "diff_hash": "0x...",
  "verdict": "approved",
  "rubric_score": 0.92,
  "tier_results": {
    "tier_0": { "passed": true, "report_attestation": "rekor:..." },
    "tier_1": { "passed": true, "report_attestation": "rekor:..." },
    "tier_4": { "passed": true, "report_attestation": "rekor:..." }
  },
  "rejection_reasons": [],
  "executor_oidc_subject": "...",
  "verifier_oidc_subject": "...",
  "signed_at": "..."
}
```

### `https://crucible.dev/PlanApproval/v1`

Emitted when a user approves a plan.

```json
{
  "task_id": "...",
  "plan_hash": "0x...",
  "estimated_cost_usd": 1.20,
  "approved_by_oidc": "...",
  "approved_at": "..."
}
```

### `https://crucible.dev/PromotionBundle/v1`

The bundle submitted to the Promotion Contract.

```json
{
  "task_id": "...",
  "diff_hash": "0x...",
  "verifier_approval_attestation": "rekor:...",
  "files_changed": [...],
  "build_provenance": { "$ref": "SLSA Provenance v1" },
  "rebuild_hash": "0x...",
  "blast_radius": { ... },
  "suggested_rollout": { ... },
  "agent_oidc_subject": "...",
  "signed_at": "..."
}
```

### `https://crucible.dev/PromotionApproval/v1`

Emitted by the Promotion Contract after policy + human approval.

```json
{
  "bundle_attestation": "rekor:...",
  "policy_decision": "auto-approve" | "human-approved",
  "rego_policy_hash": "0x...",
  "rego_decision_doc": { ... },
  "human_approvers_oidc_subjects": ["...", "..."],
  "kms_signing_key_arn": "arn:aws:kms:...",
  "lease_id": "lease_...",
  "approved_at": "..."
}
```

### `https://crucible.dev/PromotionOutcome/v1`

Final outcome of a promotion.

```json
{
  "promotion_id": "prom_...",
  "bundle_attestation": "rekor:...",
  "outcome": "landed" | "rolled_back" | "approval_timeout" | "policy_denied",
  "rollout_steps": [
    { "weight": 1, "dwell_seconds": 300, "slo_check": "passed", "timestamp": "..." },
    ...
  ],
  "final_state": "100% live",
  "rollback_reason": null,
  "completed_at": "..."
}
```

### `https://crucible.dev/MemoryWrite/v1`

Emitted on procedural memory writes (both agent-initiated and distiller-initiated).

```json
{
  "convention_id": "conv_...",
  "tenant_id": "...",
  "scope": { ... },
  "rule_nl": "...",
  "category": "Logging",
  "source_evidence": [{"kind":"pr_comment","pr":...}],
  "confidence": 0.74,
  "judge_score": 0.91,
  "writer_oidc_subject": "...",
  "written_at": "..."
}
```

## Build provenance — SLSA-L3

Crucible's Tier 4 emits SLSA Provenance v1 in addition to our own predicate types. The SLSA predicate is the standard `https://slsa.dev/provenance/v1` schema. Key fields:

```json
{
  "buildDefinition": {
    "buildType": "https://crucible.dev/build/v1",
    "externalParameters": {
      "source": "git+https://github.com/acme/payments@abcd1234",
      "config": "nix flake"
    },
    "internalParameters": {
      "nix_lock_hash": "sha256-..."
    },
    "resolvedDependencies": [
      { "uri": "git+...", "digest": {"sha1":"..."} }
    ]
  },
  "runDetails": {
    "builder": {
      "id": "https://crucible.dev/builders/hermetic-nix/v1",
      "version": { "nix": "2.21.0" }
    },
    "metadata": {
      "invocationId": "task_01HZ...",
      "startedOn": "...",
      "finishedOn": "..."
    },
    "byproducts": [
      { "name": "rebuild_hash", "uri": "...", "digest": {"sha256":"..."} }
    ]
  }
}
```

## Signature format — DSSE

All Crucible attestations use the DSSE (Dead Simple Signing Envelope) format. Sigstore Rekor v2 native support.

```json
{
  "payloadType": "application/vnd.in-toto+json",
  "payload": "<base64-encoded statement>",
  "signatures": [
    {
      "keyid": "",
      "sig": "<base64 signature>",
      "cert": "<base64 x509 cert from Fulcio OIDC issuance>"
    }
  ]
}
```

## Verifying an attestation

```bash
# Fetch attestation by Rekor UUID
crucible attestation get rekor:7d8a2c...

# Verify signature chain
crucible attestation verify rekor:7d8a2c...
  ✓ DSSE signature valid
  ✓ Fulcio cert chains to Sigstore root
  ✓ OIDC subject: https://accounts.crucible.dev/agents/worker-7
  ✓ Predicate type: https://crucible.dev/VerifierApproval/v1
  ✓ Statement subject matches diff hash
  ✓ Rekor inclusion proof valid

# Fetch the full chain for a task
crucible attestation chain task_01HZ...
  rekor:abc... PlanApproval
  rekor:def... WriteAttestation (api/webhooks.ts)
  rekor:ghi... WriteAttestation (db/migrations/20260515_refunds.sql)
  ...
  rekor:xyz... VerifierApproval
  rekor:123... PromotionApproval
  rekor:456... PromotionOutcome (landed)
```

## Self-hosted Rekor

Enterprise self-hosted deployments run their own Rekor instance. The OIDC issuer is the customer's own (configurable). The Fulcio CA root is bundled with the air-gap installer.

Public verification commands transparently work against the customer's self-hosted Rekor — `crucible attestation verify` reads the issuer from the cert and dispatches to the correct log.

## Schema source of truth

All predicate JSON-Schemas live in `libs/twin-spec/schemas/`. They are versioned via the predicate-type URI path (`/v1`, `/v2`, etc.). Breaking changes bump the version + 90-day deprecation. Old versions remain readable indefinitely (Rekor is append-only and immutable).

## Retention

- Sigstore Rekor public log: forever.
- Customer-side mirror: configurable per tenant; default 7 years (matches financial-records retention for the regulated tier).
- Tenant export: full Rekor mirror available via `crucible attestation export` for archival.
