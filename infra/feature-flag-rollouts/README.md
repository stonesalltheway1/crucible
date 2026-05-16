# infra/feature-flag-rollouts

GrowthBook-driven progressive rollouts for non-K8s (serverless / VM)
customers.

The gate's `delivery_adapter` (`growthbook` strategy) drives this:

1. **Flag created** at promotion-time, key `crucible_<promotion-id>`.
2. **Initial rollout**: 1% of users (via GrowthBook's percent rollout rule).
3. **Periodic SLO check** via Prometheus query (same templates as
   Argo Rollouts; `prometheus-query.json` describes them in
   provider-neutral form).
4. **Step up** on clean dwell.
5. **Disable to 0%** on regression (millisecond rollback).

## Files

| File | Purpose |
|---|---|
| `flag-template.json` | The GrowthBook flag definition the gate POSTs to `/api/v1/features`. |
| `prometheus-query.json` | Equivalent SLO queries to the Argo AnalysisTemplate library. |
| `crucible-rollout.json` | The standard rollout schedule: 1 → 5 → 25 → 100 with dwells. |

## Provider-agnostic shape

The flag-template uses GrowthBook's percent-rollout rule:

```json
{
  "id": "crucible_<promotion-id>",
  "type": "boolean",
  "defaultValue": false,
  "rules": [
    {
      "type": "rollout",
      "percent": 0.01,
      "value": true,
      "condition": {"crucible_promotion_id": "<promotion-id>"}
    }
  ]
}
```

LaunchDarkly / Statsig customers swap `flag-template.json` for the
equivalent shape; the gate's `delivery_adapter.GrowthBookAdapter`
abstracts these via `CreateFlag` / `SetFlagWeight` / `DisableFlag`
closures.

## SLO check

The watcher uses the same Prometheus queries as the Argo Rollouts
AnalysisTemplate library. The provider-neutral query catalog is in
`prometheus-query.json`.

## Rollback

On regression, the watcher calls `DisableFlag(reason)` which sets
`defaultValue=false` AND removes the rollout rule. Effect: every SDK
client gets `false` at the next refresh (~millisecond cycle), so
production-side traffic returns to the pre-rollout path.

## Audit

Each flag-create + flag-flip emits a `PromotionOutcome/v1` step entry
referencing the flag key. The gate's relay client persists the entry to
Rekor + the local journal.
