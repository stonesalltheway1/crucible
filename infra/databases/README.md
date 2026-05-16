# infra/databases — Phase 5 storage layer

Versioned schemas + RLS for the Phase-5 memory layer.

```
postgres/
  migrations/                  Sequenced SQL migrations (Sqitch-flavored).
    0001_pgvector_extension.sql
    0002_tenant_table.sql
    0003_memory_episodic.sql
    0004_memory_semantic.sql
    0005_conventions.sql
    0006_rls_policies.sql
    0007_distiller_runs.sql
    0008_federation_graduation.sql
  rls/
    set_role.sql               Per-connection `SET ROLE crucible_tenant`
    revoke_public.sql          Deny-by-default scaffold
  indexes/
    diskann.sql                DiskANN params at the 10M+ scale (pgvector 0.9+)
    hnsw.sql                   HNSW params for &lt; 1M tier (fallback)

falkordb/
  graph_init.cypher            Convention/SourceRef/Incident node bootstrap
  indexes.cypher               Per-tenant index definitions + bi-temporal edges
  constraints.cypher           Uniqueness + non-null gates

redis/
  keyspace.md                  Documented prefix scheme + TTL conventions
  scripts/                     Lua scripts for atomic recall-list build

migrations/
  run.sh                       Promotion-flow migration runner — twin first, real second
```

## Eat our own dogfood: migrations run twin-first

Per the Phase-5 brief, **all migrations are promoted via the same Crucible
twin-run-first flow** that customer-authored migrations use. `run.sh` is
the dev wrapper; production migrations land via
`twin.db.migrate(file)` → `twin.verify.tier4` → `twin.promote(...)` for
the Crucible-owned databases (memory-router's pgvector + FalkorDB +
Redis).

## Connection pattern

The memory-router's pgvector connection pool MUST set the per-connection
role via `SET ROLE crucible_tenant_<tenant_id>` at acquire time. The
plan-cache invalidation pitfall from current pgvector multi-tenant
guidance: do NOT rely on `current_setting('crucible.tenant_id')` inside
index predicates — RLS POLICY USING expressions reading `current_user`
are stable across plan reuse.

Connections to the procedural FalkorDB graph use a per-tenant graph
namespace `tenant_<tenant_id>` — there is one graph per tenant, never a
shared graph filtered at query time.

## Backend abstraction

The router speaks pgvector by default and Qdrant as an alternative
(self-host install option). FalkorDB by default, Neo4j as the alternative
(per ADR-006). The Graphiti abstraction layer in
`services/memory-router/internal/proceduralstore` keeps both code paths
behind a single interface.

## Index-tuning crib sheet

| Scale (per-tenant vector count) | Index | Notes |
|---|---|---|
| &lt; 100K | none / IVFFlat | seq scan beats anything else |
| 100K..1M | HNSW (m=16, ef=64) | default for fresh tenants |
| 1M..10M | HNSW (m=32, ef=128) | watch RAM — switch to halfvec |
| &gt; 10M | DiskANN | per pgvector 0.9; RAM stays bounded |
