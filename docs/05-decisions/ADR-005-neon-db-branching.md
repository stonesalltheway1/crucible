# ADR-005: Neon for Postgres copy-on-write branching

**Status:** Accepted  
**Date:** 2026-05-15

## Context

The twin runtime needs a per-task database mirror. The agent must be able to apply migrations and mutate data without touching real production. The mirror must spin up in seconds (or the twin's value evaporates), share storage with the parent (or storage costs explode), and discard cleanly at task end.

Postgres is the most common DB in our target customer base (production-engineering teams of 5–200). MySQL is secondary. Everything else (Mongo, Redis, etc.) is handled per-engine.

## Decision

For Postgres customers: **Neon Postgres branching** is the default twin DB layer.

- `POST /projects/{id}/branches` returns a connection string in 1–2 seconds.
- Copy-on-write at the storage layer; branches share data with parent.
- Branch cost: $0.002/hr (negligible for task duration).
- Storage cost: $0.35/GB-month post-Databricks-acquisition (down from $1.75).
- Cold-start 400–750ms is fine for ephemeral test workloads.

Each project has a **twin-base branch**: a daily snapshot of production with PII scrubbed. Per-task branches are children of twin-base, not children of `main`. This decouples the agent's twin from production database state changes mid-task.

For other engines:

| Engine | Mechanism | Notes |
|---|---|---|
| MySQL | PlanetScale branching | Mature for MySQL; Postgres support still half-built as of May 2026 |
| SQLite / libSQL | Turso branches | Instant per-database CoW |
| MongoDB | Atlas snapshot-restore-to-new-cluster | Minutes (not seconds); acceptable for less-common workload |
| Redis / KV | Fresh `redis-server` inside sandbox | State is small enough to recreate per task |
| S3 | MinIO in sandbox + rclone mirror | Versioning alone insufficient |
| ClickHouse | `CREATE TABLE … CLONE AS` at table level | DB-level clone proposed Apr 2026, not yet stable |

## Consequences

### Positive

- **Branch creation is fast enough to not affect perceived task latency.** 1–2s out of a typical 5–15 minute task is invisible.
- **Marginal storage cost ≈ $0 per twin.** CoW means the branch only diverges from parent for actual writes; for read-heavy tasks the divergence is bytes.
- **Migration verification becomes safe and easy.** Agent applies migration → twin diff vs base → verifier inspects schema delta → no risk to production.
- **Fan-out exploration is cheap.** Multiple parallel twins each get their own branch; no shared-state contention.
- **API surface is clean.** Neon's REST API is one curl call; integration is ~50 LoC.

### Negative

- **Vendor dependency.** Neon is the only meaningful CoW Postgres branching provider as of May 2026. Supabase requires minutes (full project clone). Xata pivoted to OSS but is younger. Self-hosting Neon-equivalent is non-trivial.
- **Per-engine fragmentation.** Not all customers are Postgres. Each non-Postgres engine has a different mechanism with different trade-offs.
- **Twin-base branch staleness.** Daily snapshots may diverge from production by up to 24 hours; tasks against very-recent data may see stale state. Mitigation: customers can request on-demand twin-base refresh.

### Trade-offs we accept

- Customers on Aurora-only, Cassandra, or other unusual stacks get degraded twin DB experience (or none, with explicit per-tenant config). Their workload class is sufficiently atypical we serve them best by being honest about it.
- We pay Neon for the SaaS tier; self-hosted enterprise customers either bring their own Neon installation (uncommon — Neon's self-host story is thin) or accept slower branching via `pg_dump`+`pg_restore` orchestration.

## Alternatives considered

### Alternative 1: Self-hosted Postgres with `pg_dump`/`pg_restore` per task

Use vanilla Postgres + dump-and-restore to create per-task DBs. **Rejected**:

- Restore time is minutes, not seconds. Kills twin perceived latency.
- Storage cost is per-task-full-copy, not CoW. Expensive at scale.
- Migration verification becomes a multi-step orchestration.

### Alternative 2: Postgres logical replication + ephemeral subscribers

Use logical replication to create read replicas; promote ephemeral subscribers for tasks. **Rejected**:

- Subscriber creation is minutes.
- Schema migrations break replication mid-task.
- Complex operational surface.

### Alternative 3: PlanetScale for everything

Single branching vendor regardless of customer's DB. **Rejected**:

- PlanetScale is MySQL-centric; Postgres branching half-built as of May 2026.
- Customer migration to PlanetScale is not feasible as an onboarding step.

### Alternative 4: ZFS / btrfs filesystem-level CoW under self-hosted Postgres

Snapshot the underlying filesystem; mount snapshot as the twin DB's data dir. **Rejected for SaaS**:

- Requires shared infrastructure with weird filesystem-level operational characteristics.
- Hard to multi-tenant safely.

(This *is* the self-hosted enterprise fallback when customers don't run Neon.)

### Alternative 5: Skip the DB twin; mock DB calls

Agent queries against a mock DB. **Rejected**:

- Schema migrations are the most important class of change to test; mocks don't catch them.
- Real query results matter for the agent's reasoning; mocked results lie.

## Open issues

- **Customers without Postgres / MySQL / SQLite / Mongo:** explicit "out of scope for v1; contact us for design partnership" message.
- **On-prem self-hosting:** the air-gapped enterprise tier needs a non-cloud branching story. Likely ZFS-snapshot-based, documented in the operations runbook.
- **Migration rollback testing:** twin can apply migrations forward; testing the *down* migration requires explicit support (currently the customer's responsibility).

## References

- [01-architecture/twin-runtime.md#layer-3-database-twin](../01-architecture/twin-runtime.md)
- Neon pricing: see [ASSETS.md](../ASSETS.md)
