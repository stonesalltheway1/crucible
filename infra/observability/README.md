# Crucible Observability

Production observability stack: Prometheus + Grafana + Loki + Tempo,
the four KPI dashboards from `docs/02-engineering/observability.md`,
alert rules covering RB-01 through RB-15 from
`docs/04-operations/runbooks.md`, and a public SLO status page.

## Layout

```
infra/observability/
├── README.md
├── helm/                       # Sub-charts for the stack
│   ├── prometheus/
│   ├── grafana/
│   ├── loki/
│   └── tempo/
├── dashboards/                 # JSON-as-code Grafana dashboards
│   ├── 01-per-task-economics.json
│   ├── 02-verifier-health.json
│   ├── 03-safety-trust.json
│   └── 04-memory-learning.json
├── alerts/                     # Prometheus alert rules
│   └── crucible-alerts.yaml
├── recording-rules/
│   └── crucible-recording.yaml
└── status-page/                # Cachet config + custom status page
    ├── cachet-values.yaml
    ├── components.yaml
    └── README.md
```

## Install

```bash
helm install observability ./infra/observability/helm \
    --namespace observability --create-namespace
```

## Dashboards

The four KPI dashboards from docs/02-engineering/observability.md:

- **Per-task economics** — median/P95 cost, cache hit rate, verifier
  cost ratio, wall-clock, tokens/dev/day.
- **Verifier health** — Tier 0 mutation kill rate, Tier 1 PBT
  counterexamples, Tier 3 timeout rate, verifier-vs-human disagreement,
  reflect-then-pass rate.
- **Safety / trust** — destructive-op gate firings, egress violations
  (target 0), sandbox escapes (target 0), Rekor publish failures,
  KMS failures, cross-tenant access (target 0).
- **Memory / learning** — procedural-memory writes/tenant/day,
  convention-drift detections/tenant/week, cross-tenant graduations,
  retrieval router p95 latency, token-budget overruns.

## Alerts

`alerts/crucible-alerts.yaml` ships rules for every RB-01 through
RB-15 runbook in docs/04-operations/runbooks.md. The Slack channel
mapping is configured per tenant in the Helm values.

## Public SLO status page

The public status page at `status.crucible.dev` is sourced from this
stack. We use Cachet OSS — see `status-page/`. Air-gap deployments
substitute their own status-page tooling; the Cachet wiring is
optional.
