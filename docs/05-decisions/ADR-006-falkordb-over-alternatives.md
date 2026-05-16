# ADR-006: FalkorDB for procedural memory graph backend

**Status:** Accepted  
**Date:** 2026-05-15

## Context

The procedural memory layer (ADR-003) requires a graph database for:

- Temporal edges (bi-temporal: valid_from / valid_to + recorded_at).
- Multi-hop traversals (a convention supersedes another; an incident touched a file owned by a team that authored an ADR).
- Per-tenant isolation.
- Low-latency reads (< 100ms p95 for retrieval router calls).
- Cypher (or equivalent) query language familiar to the team.

As of October 2025, KuzuDB — previously a strong candidate — was archived after Apple's acquisition. Remaining viable options:

| Tool | License | Maturity | Latency profile | Notes |
|---|---|---|---|---|
| Neo4j Community | GPLv3 (Community); commercial Enterprise | Mature, huge ecosystem | Single-instance fine; cluster expensive | License complicates redistribution |
| FalkorDB | Source-available | Active development, KuzuDB-successor framing for AI/GraphRAG | Low-latency Cypher, RedisGraph lineage | New but well-funded |
| ArangoDB | Apache-2.0 | Mature multi-model | Reasonable | Multi-model overkill for this use |
| Memgraph | Source-available commercial | Mature | Fast | Pricing less transparent |
| AWS Neptune | Proprietary | Mature | Higher latency for our access pattern | Vendor lock-in |

## Decision

**FalkorDB** is the default graph backend for procedural memory.

- Source-available license; OSS for our needs.
- Cypher query language; familiar to the team.
- Sub-millisecond queries for typical convention retrieval patterns.
- Active development; KuzuDB-successor positioning in the AI/GraphRAG market.
- Low operational overhead; integrates cleanly with Redis-adjacent infrastructure.

Abstraction layer: **Graphiti** (Zep's OSS engine for temporal knowledge graphs). We use Graphiti's data model (bi-temporal edges, episode-based ingestion) regardless of backend, so we can swap if FalkorDB stops being viable.

For customers who prefer Neo4j (large enterprise with existing graph infra), the self-hosted tier supports `backend: neo4j` as a values.yaml option.

## Consequences

### Positive

- **Low latency.** Sub-millisecond Cypher queries support our < 100ms p95 retrieval-router SLO.
- **Cypher familiarity.** Engineers can debug queries directly.
- **OSS license suitable for redistribution.** No GPL pollution in our chart.
- **Graphiti abstraction insulates us.** If FalkorDB falters, we swap backends without rewriting the memory layer.

### Negative

- **Ecosystem smaller than Neo4j.** Fewer plugins, fewer educational resources, fewer hires-with-experience.
- **Single-vendor risk.** FalkorDB is one company. KuzuDB-archive scenario is a real precedent.
- **Some advanced graph algorithms missing.** OK for procedural memory; not OK for graph-algorithm-heavy workloads. Not our use case.

### Trade-offs we accept

We pay the "small ecosystem" tax in exchange for an OSS-redistributable license and low latency. The Graphiti abstraction caps the cost of any future backend swap.

## Alternatives considered

### Alternative 1: Neo4j Community

GPLv3 is the blocker. Our Helm chart and Docker images are redistributed widely; GPL components in a default deployment create downstream license obligations for customers we can't manage. (We could ship Neo4j separately as an opt-in component, but defaults matter.)

Also: Neo4j Enterprise pricing is opaque and expensive; we'd push that cost to customers.

### Alternative 2: KuzuDB

Was the strongest candidate before the October 2025 archive. Apple's acquisition removed it from contention.

### Alternative 3: ArangoDB

Multi-model (document + graph + key-value). **Rejected**:

- Multi-model is overkill — we use the graph features only.
- Operational footprint heavier than FalkorDB.
- Apache-2.0 license is fine, but the operational complexity isn't worth the license benefit.

### Alternative 4: Vector store only (skip the graph)

Use pgvector / Qdrant for everything. **Rejected** in ADR-003; conventions have relational structure that vectors don't capture.

### Alternative 5: Roll our own graph atop Postgres (recursive CTEs)

Persist conventions as rows + edges in Postgres; query with recursive CTEs. **Rejected**:

- Multi-hop queries are slow.
- We'd be reinventing FalkorDB / Neo4j poorly.
- Not worth the engineering cost.

### Alternative 6: Memgraph

Source-available commercial, fast. **Rejected as default** but kept as a "Memgraph as alternative backend" option for customers who prefer it. Pricing less transparent than FalkorDB's.

## Migration path (if FalkorDB doesn't pan out)

Graphiti abstracts the backend. Migration sketch:

1. Set up the new backend (Neo4j / Memgraph / ArangoDB).
2. Use Graphiti's export/import tooling to move tenant graphs.
3. Cut over reads with feature-flag canary.
4. Cut over writes.
5. Decommission FalkorDB.

Estimated effort: ~1 agent-day for the migration code; longer for customer-facing communication and rollout.

## References

- [01-architecture/memory-layer.md](../01-architecture/memory-layer.md)
- [ADR-003](ADR-003-procedural-memory-moat.md)
