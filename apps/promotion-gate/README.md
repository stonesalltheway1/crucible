# crucible-promotion-gate

The Go service that turns a verified `PromotionBundle` into a real-world
deployment. The bridge from twin to real.

```
PromotionBundle (signed)
        │
        ▼
   ┌─────────────────┐
   │ bundle_validator│  attestation chain + subject digests + freshness
   └────────┬────────┘
            ▼
   ┌─────────────────┐
   │   rego_engine   │  embedded OPA — default + tenant policies
   └────────┬────────┘
            ▼
   ┌─────────────────┐
   │ approval_router │  CODEOWNERS / designated / N-of-M
   └────────┬────────┘
            ▼
   ┌─────────────────┐
   │   kms_lease     │  AWS KMS / GCP HSM / YubiHSM — single-use, time-boxed
   └────────┬────────┘
            ▼
   ┌─────────────────┐
   │ delivery_adapter│  Argo Rollouts (K8s) or feature-flag (serverless)
   └────────┬────────┘
            ▼
   ┌─────────────────┐
   │ outcome_watcher │  Prometheus SLO checks; auto-rollback; emit Outcome
   └─────────────────┘
```

## Layout

| Pkg | Purpose |
|---|---|
| `internal/bundle_validator` | Verifies the attestation chain (Rekor UUIDs, hashes, OIDC). |
| `internal/rego_engine`      | Loads default + tenant policies; emits `Decision` and signed `PromotionApproval/v1`. |
| `internal/approval_router`  | Resolves required-approver cohorts from CODEOWNERS + tenant config. |
| `internal/kms_lease`        | AWS KMS / GCP HSM / YubiHSM adapters; mints single-use leases. |
| `internal/delivery_adapter` | Argo Rollouts + GrowthBook feature flag drivers. |
| `internal/outcome_watcher`  | Prometheus SLO checks; auto-rollback; emits `PromotionOutcome/v1`. |
| `internal/migration`        | Three-step DB migration flow (twin → shadow → KMS-leased apply). |
| `internal/api`              | HTTP surface — REST today, gRPC-compatible. |
| `internal/relay`            | Thin HTTP client for the attestation relay. |
| `cmd/crucible-promotion-gate` | Daemon entry-point. |

## API

| Method | Path | Purpose |
|---|---|---|
| POST | `/v1/promotions` | Submit a `PromotionBundle`; returns `PromotionStatus` |
| GET  | `/v1/promotions/{id}` | Fetch current `PromotionStatus` |
| POST | `/v1/promotions/{id}/approve` | Human approver clicks |
| POST | `/v1/promotions/{id}/reject`  | Human approver rejects |
| POST | `/v1/promotions/{id}/rollback` | Manual rollback (admin-only) |
| GET  | `/healthz` | Health probe |

Webhooks fired on every state change: `task.promotion_proposed`,
`task.promotion_approved`, `task.promotion_deploying`,
`task.promotion_canary_dwell`, `task.promotion_landed`,
`task.promotion_rolled_back`.

## Env

| Variable | Purpose | Default |
|---|---|---|
| `CRUCIBLE_GATE_ADDR` | Bind address | `:9180` |
| `CRUCIBLE_RELAY_ADDR` | Attestation relay base URL | `http://127.0.0.1:9120` |
| `CRUCIBLE_KMS_PROVIDER` | `aws` \| `gcp` \| `yubi` \| `dev` | `dev` |
| `CRUCIBLE_KMS_KEY_ARN` | KMS key identifier | unset |
| `CRUCIBLE_TENANT_POLICY_DIR` | Per-tenant signed bundles | `./tenants/` |
| `CRUCIBLE_ARGO_ROLLOUTS_ADDR` | Argo Rollouts API | unset |
| `CRUCIBLE_GROWTHBOOK_ADDR`    | GrowthBook API     | unset |
| `CRUCIBLE_PROMETHEUS_ADDR`    | Prometheus API for SLO checks | unset |
| `CRUCIBLE_SLACK_BOT_ADDR`     | Slack bot for approvals | unset |
| `CRUCIBLE_GATE_DEV_MODE`      | `1` to use FakeKMS + LocalDelivery + FakeProm | unset |

## Mutation score targets

- `rego_engine/`: ≥ 95% mutation kill rate.
- `kms_lease/`:   ≥ 95% mutation kill rate.
- Rest of the gate: ≥ 85%.

## Latency

- Auto-approve trivial: target ≤ 5 s end-to-end (verifier-approved bundle to
  canary-step-1 traffic).
- Human-approval flow: ≤ 30 s of gate-internal latency once the approver
  clicks.
- Attestation chain validation: ≤ 200 ms for a single bundle's full
  chain on hot-cached Rekor data.
