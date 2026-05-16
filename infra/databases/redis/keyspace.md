# Redis keyspace conventions — Phase 5 hot memory

The Redis tier is the agent's working window during a single task. TTL
minutes–hours, never long-term storage.

## Prefix scheme

```
crucible:{tenant_id}:task:{task_id}:ctx           → JSON array of {key, content, ts}, list-encoded
crucible:{tenant_id}:task:{task_id}:plan           → JSON of latest Plan
crucible:{tenant_id}:task:{task_id}:tools:{n}      → last 50 tool calls, capped
crucible:{tenant_id}:task:{task_id}:branch         → current branch state
crucible:{tenant_id}:recall:{cache_key}            → 1h-TTL cache of recall router outputs
crucible:{tenant_id}:rate:{kind}                   → token-bucket rate limiters (recall, note)
crucible:bootstrap:bundle:{stack}                  → opaque bytes; the loaded global-defaults JSON for a stack
crucible:cartographer:{job_id}:progress            → CartographerProgress event stream
crucible:cartographer:{job_id}:result              → CartographerScanResult JSON
crucible:distiller:judge_corpus:{shard}            → adversarial-corpus shard for offline catch-rate audit
```

## TTLs

| Prefix | Default TTL | Reinforce-on-access? |
|---|---|---|
| `:task:*:ctx` | 6 h | yes |
| `:task:*:plan` | 24 h | no |
| `:task:*:tools:*` | 6 h | no |
| `:task:*:branch` | 24 h | no |
| `:recall:*` | 1 h | no — invalidated by Convention writes for the tenant |
| `:rate:*` | 60 s | no |
| `:bootstrap:bundle:*` | none | bundles are immutable; loader-managed |
| `:cartographer:*` | 7 d | no |
| `:distiller:judge_corpus:*` | none | corpus is fixture data |

## Per-tenant isolation

Redis Cluster slot computation is tenant-aware: every tenant key includes
`{tenant_id}` as the hash-tag wedge, so per-tenant slot affinity holds and
the router never accidentally fans out a tenant query across slots.

## Atomic ops

The recall-router builds the response with a single `MULTI` block:

```
MULTI
  GET crucible:{tenant_id}:task:{task_id}:plan
  LRANGE crucible:{tenant_id}:task:{task_id}:tools:* 0 49
  GET crucible:{tenant_id}:recall:{cache_key}
EXEC
```

When cache hit, return the cached envelope. When miss, fall through to
pgvector + FalkorDB, then `SETEX` the result with 1h TTL.

## Invalidation

Procedural-memory writes (admission of a new Convention or supersession)
publish to `crucible:invalidate` with payload `{tenant_id}:{cache_key}`.
Router subscribers DEL the affected recall caches.
