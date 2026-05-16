# infra/argo-rollouts

Argo Rollouts manifests used by the Crucible promotion gate.

Two pieces:

1. **`templates/analysis/`** — `AnalysisTemplate` library. The gate
   references templates by name in the generated `Rollout` spec; the
   template performs the SLO check.
2. **`templates/rollout/`** — `Rollout` strategy templates (1/5/25/100%
   canary with dwell). The gate's `delivery_adapter` patches in the
   bundle's `suggested_rollout` weights before applying.

## AnalysisTemplate library

| Template | Metric | Trigger |
|---|---|---|
| `crucible-slo-error-rate` | `sum(rate(http_requests_total{status=~"5.."}[5m])) / sum(rate(http_requests_total[5m]))` | failure rate > 0.5% |
| `crucible-slo-latency-p95` | `histogram_quantile(0.95, sum(rate(http_request_duration_seconds_bucket[5m])) by (le))` | p95 > baseline × 1.3 |
| `crucible-slo-error-rate-baseline` | same but compared to the baseline service | ratio > 1.5 |
| `crucible-slo-custom` | per-service template the bundle's `suggested_rollout.slo_check` references | custom |

All templates use the standard Prometheus provider. The address is set per
cluster via the `prometheus.address` ConfigMap key.

## Rollout strategy templates

Canary with 4-step ramp + dwells matching the promotion contract's
default:

```
strategy:
  canary:
    steps:
      - setWeight: 1
      - pause: { duration: 5m }
      - analysis: { templateName: crucible-slo-error-rate }
      - setWeight: 5
      - pause: { duration: 10m }
      - analysis: { templateName: crucible-slo-error-rate }
      - setWeight: 25
      - pause: { duration: 30m }
      - analysis: { templateName: crucible-slo-error-rate }
      - setWeight: 100
```

The gate's `delivery_adapter` substitutes the weights + dwell-seconds from
the bundle's `suggested_rollout.steps` field; the analysis-template name
is taken from `suggested_rollout.slo_check`.

## Auto-rollback

The Argo Rollouts controller fires `abortRollout` automatically when an
inline `analysis` step returns `Failed`. The gate's `outcome_watcher`
double-checks by polling the rollout's status and emitting a
`PromotionOutcome/v1` attestation with `outcome=rolled_back` and the
failing template name in `rollback_reason`.

## Install

```bash
kubectl apply -f templates/analysis/
kubectl apply -f templates/rollout/
```

## Cluster requirements

- Argo Rollouts 1.8+.
- Prometheus reachable from the rollouts controller.
- `crucible.io/promotion-id` label propagated through to pod metadata so
  the SLO query can scope per-promotion.
