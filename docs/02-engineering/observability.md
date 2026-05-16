# Observability

What we instrument, what we measure, what we dashboard, what we alert on.

## Telemetry contract

Every component emits:

- **OpenTelemetry traces** — spans for every meaningful action, with structured attributes.
- **OpenTelemetry metrics** — RED (rate, errors, duration) on every service surface.
- **Structured logs** — JSON-line format, correlation IDs everywhere.
- **In-toto attestations** — separately, the cryptographic audit trail (see [01-architecture/threat-model.md](../01-architecture/threat-model.md)).

OTel spans are exported to:
- **Honeycomb** (SaaS tier)
- **Tempo + Grafana** (self-hosted tier)

Logs:
- **Honeycomb structured events** (SaaS)
- **Loki + Grafana** (self-hosted)

Metrics:
- **Prometheus + Grafana** in both deployments.

## Span attributes (the contract)

Every span carries:

```
task_id            UUID, present on every action in a task
step_id            UUID, sub-step within a task
tenant_id          per-tenant scoping
repo_id            per-repo scoping
agent_oidc_subject Sigstore keyless identity of the agent
model.vendor       anthropic | google | openai | xai | deepseek | ...
model.id           claude-opus-4-7 | gemini-3.1-pro | ...
model.tier         0 | 1 | 2 | 3 | 4
tokens.input.fresh
tokens.input.cached
tokens.output
cost.usd
verifier.role      executor | verifier
tier_result        only on verifier spans
```

Tracing-by-task-id lets us reconstruct the full lifecycle of any task from submission to promotion.

## The four KPI dashboards

### Dashboard 1: Per-task economics

| Metric | Target | Alert if |
|---|---|---|
| Median task cost | ≤ $1.69 | > $2.50 sustained 1h |
| P95 task cost | ≤ $7.00 | > $12.00 sustained 1h |
| Cache hit rate | ≥ 70% | < 60% sustained 2h |
| Verifier cost as % of total | ≤ 10% | > 20% sustained 4h |
| Median task wall-clock | ≤ 15 min | > 30 min sustained 1h |
| Token usage per active dev/day | ≤ 1.5M | > 3M sustained 1d |

Per-task cost > $2.50 sustained means our routing is broken or cache is missing. Cache hit rate < 60% means we are losing GM. Both are alarms-page-the-team.

### Dashboard 2: Verifier health

| Metric | Target | Alert if |
|---|---|---|
| Tier 0 mutation kill rate | ≥ 85% | < 70% |
| Tier 1 PBT counterexample rate | < 5% (most PRs should pass) | > 15% (verifier too strict?) |
| Tier 3 proof timeout rate | < 10% | > 25% (proofs not converging) |
| Verifier disagreement with human reviewer | < 15% (shadow mode) | > 25% |
| Verifier reject → reflect → pass rate | ≥ 70% | < 50% (executor not learning) |

The disagreement-with-human-reviewer metric is the most important signal for verifier quality. We want it low (verifier matches human judgment) but not zero (some genuine humans disagree on style; that's noise).

### Dashboard 3: Safety / trust

| Metric | Target | Alert if |
|---|---|---|
| Destructive-op gate firings | tracked, not capped | n/a (informational) |
| Twin-scoped destructives (auto-approved) | tracked | n/a |
| Real-scoped destructives requiring approval | tracked | n/a |
| Egress policy violations | 0 | > 0 (immediate page) |
| Sandbox escape attempts | 0 | > 0 (P0 security incident) |
| Sigstore Rekor publish failures | 0 | > 0 (audit trail gap; page) |
| KMS signing failures | 0 | > 0 (P1) |
| Cross-tenant memory access attempts | 0 | > 0 (P0 isolation breach) |

The 0-target metrics are the safety floor. Any non-zero count is a paging event.

### Dashboard 4: Memory / learning

| Metric | Target | Alert if |
|---|---|---|
| Procedural memory writes per tenant per day | tracked | growth stalls (informational) |
| Convention drift detections per tenant per week | tracked | spike > 10x baseline |
| Cross-tenant abstraction graduations per week | tracked | n/a |
| Memory retrieval router p95 latency | < 100ms | > 250ms |
| Memory retrieval token-budget overruns | < 1% of calls | > 5% |

The procedural-memory write rate is a leading indicator of customer engagement. Stalls mean the customer's PR review activity isn't reaching us — likely an integration broken.

## Standard alerts (SaaS tier)

Critical alerts (page on-call immediately):

- Any 0-target safety metric > 0.
- Cache hit rate < 50% for 30 min.
- Median task cost > $5 for 30 min.
- Sigstore Rekor or KMS unreachable > 5 min.
- Twin-runtime spawn failure rate > 2% for 10 min.
- Promotion-gate evaluation latency p95 > 5s for 10 min.

Warning alerts (Slack notify, not page):

- Cache hit rate 50–60% for 1h.
- Median task cost $3–$5 sustained.
- Verifier disagreement-with-human > 20% over 24h.
- Tier 3 proof timeout rate > 20% over 24h.

## Self-hosted alerting

Self-hosted customers receive a default Prometheus alert pack matching the above, parameterized by their tenant config. They wire it to their own PagerDuty/Opsgenie/whatever.

## Cost telemetry storage

OTel spans → Honeycomb (SaaS) for ad-hoc analysis + Honeycomb-Triggers for alerts.

Long-term cost analytics → ClickHouse cluster with daily rollups. Per-tenant per-model per-day token+dollar aggregates retained 13 months for SOC-2 audit + customer billing reconciliation.

## SLOs we publish to customers

```yaml
slo:
  task_completion_within_estimate:
    objective: 90%
    window: 30d
    description: "Tasks complete within the wall-clock and cost estimate shown in the plan."
  
  promotion_canary_success:
    objective: 99.5%
    window: 30d
    description: "Verified promotions pass canary without rollback."
  
  verifier_decision_within_15min:
    objective: 95%
    window: 30d
    description: "Tier 0+1 verification completes within 15 minutes."
  
  control_plane_availability:
    objective: 99.9%
    window: 30d
    description: "Control plane API responsive (excluding planned maintenance)."
  
  attestation_publish_success:
    objective: 99.99%
    window: 30d
    description: "All in-toto attestations successfully published to Rekor."
```

Customers can subscribe to SLO status via our public status page; enterprise tier gets a per-customer dashboard.

## Customer-facing observability

Each tenant's web console exposes:

- **Their own task timeline** — every task they've run, cost, verifier result, promotion outcome.
- **Their own cost dashboard** — per-developer, per-repo, per-day.
- **Their own memory browser** — view active conventions, drifting conventions, supersession history.
- **Their own attestation viewer** — Rekor UUIDs and content of any attestation.
- **Their own SLO dashboard** — relative to our published SLOs.

The web console is part of the product surface, not a separate observability bolt-on.

## What we don't expose externally

- Per-customer-aggregate metrics (cost telemetry, etc.) are internal-only.
- Cross-customer benchmark comparisons are not surfaced.
- Internal verifier disagreement rates are not surfaced (they're noisy and easy to misinterpret).

The exception: a public, transparent quarterly "Crucible Trust Report" with anonymized aggregates — cache hit rate distribution, median task cost, verifier-vs-human agreement, safety incidents (zero, hopefully). This is the brand investment.

## Tooling-stack rationale

- **OpenTelemetry** as the wire protocol → vendor-neutrality. Customers can swap exporters.
- **Honeycomb** for SaaS-tier hot analysis → speed of ad-hoc query is critical; their pricing scales with us.
- **Prometheus + Grafana** for self-host → universal standard; customers already have it.
- **ClickHouse for long-term aggregates** → cheap columnar storage, fast queries on billions of spans.
- **Loki for logs** → cheap, multi-tenant, integrates with Grafana.
- **Sentry for errors** → standard tool; customers know it.

We don't roll our own observability infra. Use the commodity.
