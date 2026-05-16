# Public SLO status page

`status.crucible.dev` — the customer-visible health dashboard.

We use **Cachet** OSS (cachethq.io) as the status-page backend. It
ingests SLO data from Prometheus via a sidecar that translates the
recording rules in `recording-rules/crucible-recording.yaml` into
Cachet metric-points.

## Components

The component list is in `components.yaml`. Each component maps to one
of the five SLOs from `docs/02-engineering/observability.md`.

## Per-tenant SLO dashboards

Pro / Team customers see the public status page only. Enterprise
customers get a per-tenant dashboard via the web-console at
`https://app.crucible.dev/slo` (Phase 7 surface). The data source for
both is the same Prometheus instance.

## Air-gap

Air-gap deployments substitute their own status tooling (typically
Atlassian Statuspage on the customer's side, fed by the same metrics).
The Cachet wiring is optional in the Helm chart.
