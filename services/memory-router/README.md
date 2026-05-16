# memory-router (Phase 5 hot-path)

The Go daemon that serves `twin.memory.*` for every agent task.

```
                 ┌──── Redis (hot)  ─────────────┐
RetrievalQuery ──┼──── pgvector (epis. + sem.) ──┼── A-MAC re-rank ── 7K budget ── ScoredMemory[]
                 └──── FalkorDB (procedural) ────┘
                          ↑
                          │ writes (admission, judge, drift)
                          │
                  Distiller (Python, async, separate process)
```

## Architectural invariants

1. **Per-tenant scoping enforced server-side.** Every query starts with a
   `tenant_id`; the router refuses to issue a downstream query without
   one. The pgvector connection pool sets `SET ROLE crucible_tenant_<id>`
   at acquire time so RLS is the secondary defence even if a code bug
   removes the WHERE clause.

2. **Three-tier layering, bottom-up read.**
   `global_defaults` → `org_overrides` → `repo_overrides`. The router
   merges results by Convention ID — higher-priority layer wins.

3. **Embeddings never cross tenants.** Cache keys include the tenant_id;
   the embedding-service client refuses to call with a cross-tenant
   payload mix.

4. **AGENTS.md / CLAUDE.md / .cursorrules win.** Customer-supplied
   override files materialise into `repo_overrides` at cartographer time
   and outrank any inherited rule.

5. **7K-token budget enforced.** The router refuses to return more
   tokens than the budget; over-quota items are dropped lowest-scored
   first.

## Packages

```
cmd/memory-router/main.go         binary entrypoint, wires the daemon
cmd/cartographer/main.go          installer-side per-repo mining CLI
internal/
  server/                         gRPC handlers (MemoryService)
  retriever/                      multi-signal hybrid retrieval orchestrator
  ranker/                         A-MAC importance scoring + re-rank
  budget/                         7K-token enforcement, tiktoken-equivalent
  scope/                          ScopeFilter normalization + match
  layering/                       global/org/repo merge
  hotstore/                       Redis client + recall envelope builder
  vectorstore/                    pgvector + Qdrant adapters
  proceduralstore/                Graphiti abstraction over FalkorDB / Neo4j
  globaldefaults/                 per-stack bundle loader
  federation/                     ≥5-tenant graduation candidate detector
  embedding/                      pluggable embedding client + per-tenant guard
  cartographer/                   one-shot repo scanner (also runs in /cmd)
  config/                         env-driven config loader
metrics/                          Prom counters, histograms; p95 latency observed here
test/
  isolation/                      cross-tenant adversarial random-query test
  golden/                         retrieval-output snapshots
global_defaults/                  per-stack JSON bundles (built by oss-corpus-bootstrap)
```

## p95 latency budget — < 100ms

|  Stage  | Budget |
|---|---|
| Redis lookup + cache check | 5 ms |
| Embedding (cached or computed) | 30 ms |
| pgvector top-K + RLS | 25 ms |
| FalkorDB scope-traversal | 20 ms |
| A-MAC re-rank + budget enforcement | 5 ms |
| gRPC wire + buffer copy | 10 ms |
| **Total** | **95 ms** |

The benchmark harness in `test/bench_latency_test.go` enforces this gate.

## Cartographer

`cmd/cartographer` is the installer-side one-shot. It runs inside the
customer infrastructure boundary — same isolation as a twin — and writes
its output into the `repo_overrides` layer of the memory-router. See
the Stage-2 Cartography flow in `docs/04-operations/onboarding.md`.
