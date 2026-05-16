# services/shadow-recorder

The Phase-8 standalone shadow-recorder service. Hooks into the
customer's staging environment via an egress proxy or eBPF tap, records
production-like traffic for a 7-day default window, full-PII-scrubs at
capture, exposes coverage metrics, and runs the monthly re-record
schedule.

This service is the sibling of `services/twin-runtime/tape_driver/shadow_recorder`
(Phase 3, in-tree to the twin runtime). The Phase-3 recorder is the
in-tape-driver capture path that runs alongside the twin; this Phase-8
service is the standalone deployment customers point their staging
ingress at.

## Endpoints

- `POST /v1/ingest/envoy` — Envoy access-log sink (the customer's edge
  Envoy ALS streams here).
- `POST /v1/ingest/ebpf` — eBPF tap sink (Phase-3 recorder + raw
  Cilium HTTP metrics).
- `GET  /v1/coverage` — coverage metrics: per-endpoint last-recorded
  timestamps, per-host hit counts, tape-population %.
- `GET  /v1/coverage/{host}` — host-scoped coverage detail.
- `POST /v1/rerecord/run` — kick the monthly re-record schedule
  manually (cron also fires it).
- `GET  /healthz`, `GET  /version`.

## Storage

Tapes are written to the configured object store (S3/MinIO/GCS) under
`<bucket>/tapes/<tenant>/<host>/<sha>/`. Metadata + per-endpoint
last-recorded timestamps are kept in Postgres (uses the same DB as the
control plane in dev; a per-tenant schema in production).

## PII scrub

The recorder calls the Phase-3 scrub service
(`services/twin-runtime/tape_driver/scrubber`) at capture time. We
fail-closed when the scrubber is unreachable to honour the brief's
"production deployments REQUIRE the Presidio service" guarantee.

## Coverage dashboard

The service exposes a Prometheus metrics surface (`/metrics`) the
Phase-8 observability stack scrapes. The four KPI dashboards
(`infra/observability/dashboards/`) include a per-tenant tape-
coverage panel sourced from these metrics.

## Re-record schedule

Default: monthly. Configurable per host. Cron-driven. Each scheduled
run replays the previous capture and refreshes the tape if upstream
service signatures changed (response-shape diff > threshold).

## Build + test

```bash
cd services/shadow-recorder
go build ./...
go test ./...
```

## Local run

```bash
export CRUCIBLE_SHADOW_LISTEN=:9520
export CRUCIBLE_SCRUBBER_URL=http://127.0.0.1:9100
export CRUCIBLE_SHADOW_OBJECTSTORE=s3://crucible-tapes-dev
export CRUCIBLE_SHADOW_DB_DSN=postgres://localhost/crucible_shadow
crucible-shadow-recorder
```
