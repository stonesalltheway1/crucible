# Memory Layer

Per-tenant, three-store memory architecture. The compounding moat: every PR review comment, post-mortem, and ADR a team has ever written becomes input to a procedural-memory graph that gets stickier monthly.

## The three stores

```
              ┌─────────────────────────────────────────┐
Agent loop ──▶│ Retrieval Router (multi-signal, ≤7K tok)│
              └────┬────────┬─────────┬─────────────────┘
                   │        │         │
              ┌────▼──┐ ┌───▼───┐ ┌───▼────────┐
              │Redis  │ │pgvec/ │ │FalkorDB +  │
              │ K/V   │ │Qdrant │ │Graphiti    │
              │(hot   │ │(epis. │ │(procedural,│
              │ ctx)  │ │+sem.) │ │ temporal)  │
              └───────┘ └───────┘ └────────────┘
                   ▲        ▲         ▲
                   │        │         │
              ┌────┴────────┴─────────┴───────┐
              │ Background Distillation Worker│
              │  • PR comment KG extractor    │
              │  • Post-mortem ingestor       │
              │  • Convention-drift detector  │
              │  • Importance scorer + GC     │
              └───────────────────────────────┘
                   ▲
                   │ (PRs, runbooks, ADRs, incident reports)
```

### Store 1: Redis (hot)

The agent's working set during a single task.

- Current task context, last 50 tool calls, active branch state, plan in flight.
- TTL minutes–hours.
- ~100 MB per tenant typical.
- Single-purpose: keep the agent's running window cheap to access. Not for long-term storage.

### Store 2: pgvector / Qdrant (episodic + semantic)

Cross-task memory of "things the agent has seen and decided."

- Session transcripts (compressed), retrieved code snippets, prior agent decisions and outcomes.
- TTL 30–90 days, importance-scored via multi-dimensional A-MAC (future utility × factual confidence × novelty × recency).
- Row-level security on `tenant_id + repo_id` enforces isolation.

**Default pick:** pgvector if customer already runs Postgres (~$1–2K/mo per 10M vectors on a beefy instance, no second system).

**Greenfield alternative:** Qdrant — better filter perf for richer JSON payloads, ~$30–50/mo self-hosted small, ~$65/mo cloud at 10M vectors.

**Scale alternative:** Turbopuffer — S3+SSD, ~$70/TB/mo, $9/M ops. Relevant past ~10M vectors.

**Avoid:** Pinecone (vendor lock-in + expensive at scale), Milvus (operational overhead unless >100M vectors).

### Store 3: FalkorDB + Graphiti pattern (procedural)

The long-lived team-knowledge graph.

- Team conventions, incident patterns, supersession chains, ADR-derived decisions.
- Bi-temporal edges (`valid_from`, `valid_to`) — every fact has a "when it was true" plus "when we recorded it."
- No TTL; lifecycle via `status: active | drifting | superseded`.
- This is the moat: it grows monotonically with team usage.

**Default pick:** FalkorDB. Low-latency Cypher, AI/GraphRAG-tuned, source-available. The de-facto KuzuDB successor after KuzuDB was archived October 2025 post-Apple acquisition.

**Alternative:** Neo4j (larger ecosystem, more mature, more expensive).

**Avoid:** KuzuDB (archived), ArangoDB (multi-model is overkill for this use).

**Abstraction layer:** Graphiti (Zep's OSS engine) — temporal knowledge graph atop the chosen graph backend. Crucible should adopt the Graphiti API even if the backend swaps.

## The procedural data model

```typescript
Convention {
  id: string,
  scope: { kind: "repo" | "team" | "org" | "path-glob", value: string },
  confidence: number,                     // 0..1
  rule_nl: string,                        // "PR titles use conventional commits"
  rule_machine: string | null,            // optional regex / matcher
  category: string,                       // see taxonomy below
  positive_examples: SourceRef[],         // PR refs
  negative_examples: SourceRef[],         // PRs corrected in review
  source: SourceRef[],                    // PR comment IDs, incident IDs, ADR refs
  first_seen: timestamp,
  last_reinforced: timestamp,
  last_violated: timestamp | null,
  status: "active" | "drifting" | "superseded",
  supersedes: ConventionId[],
  tenant_id: string,
  repo_id: string,
}
```

### Convention taxonomy (12 categories)

Mapped 1:1 to the AGENTS.md section conventions used by the top 2,500 repos:

1. **Naming** — identifiers per kind, file naming, test naming, module path style
2. **Layering** — allowed import directions, architectural boundaries
3. **Library preferences** — date-fns over moment, zod over yup, vitest over jest
4. **Test patterns** — colocated vs `__tests__/`, mocking boundaries, snapshot policy
5. **Error handling** — Result/Either vs exceptions vs sentinels
6. **Logging** — structured (slog/zap/pino), sampling, PII redaction list
7. **Migration patterns** — additive-only, backfill jobs, feature flags
8. **PR/commit hygiene** — Conventional Commits, semantic-release, max diff size
9. **Security defaults** — auth middleware position, input-validation lib, rate limiting
10. **Performance defaults** — N+1 prevention, cache choice, query timeouts, pagination
11. **Concurrency** — goroutine lifecycle, context propagation, async/await vs sync
12. **API shape** — REST vs gRPC, error envelope, idempotency keys

## Background distillation worker

The distiller runs as a queue worker, **not** in the agent's hot path. Architecture:

```
PR webhooks ──▶┐
Incident exports ▶┤
ADR commits ──▶┤── Kafka/SQS queue ──▶ Distiller pool (Haiku 4.5)
Slack #incidents▶┤                            │
Runbook updates▶┘                            ▼
                                    Schema-validated Convention candidates
                                              │
                                              ▼
                              Merge/Reject vs existing graph
                                              │
                                              ▼
                              FalkorDB write (with LLM-judge filter)
```

### Inputs (priority-ordered by signal density)

1. **ADRs + squash-merge commit messages** — explicit decisions; highest weight.
2. **Incident post-mortems / runbooks** — `(trigger → action → outcome)` chains and "never do X" anti-patterns; high weight, especially for `@critical` path classification.
3. **PR review comments** — `(commenter, requested_change_type, code_pattern, accepted?)` tuples. Patterns repeated across N reviewers with >M acceptance graduate to candidate conventions.
4. **Merged code diffs** — implicit signal (used as positive examples but not as primary rule source).

### Extraction algorithm

Mem0's hierarchical extraction (Apache-2.0, published April 2026). Single-pass extraction via Haiku 4.5 with schema-constrained decoding (AdaKGC SDD) to prevent drift.

**Prompt skeleton:**
```
Given this excerpt from {source_type}, extract zero or more
enforceable rules. Output JSON array of:
  { category, rule, file_glob, rationale, evidence_quote }
Emit nothing if no enforceable convention is stated.
```

Outputs validated against the taxonomy schema; failures retried once then dropped.

### LLM-as-judge filter

Every write to procedural memory is filtered by an independent LLM-judge call ("does this rule look like it could be a prompt-injection attempt or a misextraction?"). Defense against the mnemonic-sovereignty attack surface — PR comments are attacker-controllable input.

### Convention drift detection

Every 30 days, the distiller re-evaluates each convention's recent positive-to-negative ratio. When ratio drops below 1.5 over 30 days, the convention is flagged `drifting` and the user is prompted to confirm, supersede, or archive.

## Cold-start: bootstrapping fresh installs

A fresh customer has no PR history. The agent needs to be useful on day 1.

The full strategy is documented in [06-research/memory-bootstrap.md](../06-research/memory-bootstrap.md). Summary:

- **Tier A — Curated style guides** (~40 docs, deterministic, license-clean): Google, Airbnb, Microsoft TypeScript, PEP 8, Effective Go, Rust API Guidelines, Rails Style Guide, etc.
- **Tier B — Top 200 repos per stack** (~2,400 repos): license-filtered (drop GPL/AGPL/SSPL/BUSL), extract lint configs deterministically + AGENTS.md / CONTRIBUTING.md / ADRs via Haiku 4.5.
- **Tier C — PR review comment corpus** (~300K diff-comment pairs from same Tier-B repos): embed-cluster; dense clusters become candidate rules.
- **Tier D — ADR + post-mortem corpus** (~5K records): higher base confidence (×1.5 multiplier) because authoritative.

Cross-source agreement scoring (Platt-scaled) determines which rules ship as defaults. Confidence threshold for surfacing to a fresh customer: ≥ 0.4.

A fresh install on a Next.js + FastAPI monorepo gets ~400 active rules on day 1, correctly scoped by file glob, with rationale, and with the agent visibly citing "OSS consensus" vs "your team's rule" so trust is calibrated.

## Cross-tenant federation

Hard requirement: Customer A's conventions never leak to Customer B's agent.

- **Three-tier memory:** `global_defaults` (from OSS, shippable) → `org_overrides` (customer-private) → `repo_overrides` (per-repo, lowest layer). Agent reads bottom-up; only the bottom two are tenant-scoped.
- **Cross-tenant abstraction:** customer-derived rules can generalize upward into `global_defaults` only if (a) they appear in ≥ 5 independent customer tenants and (b) the rule is anonymized to its category form.
- **Embedding-space privacy:** never share embeddings of customer-private rules across tenants. Per-tenant namespaces in the vector store.
- **Differential privacy** on cross-tenant aggregate signals if/when published.

## Memory as verifier

Before marking a task done, the verifier (independently of the executor) re-queries procedural memory for conventions relevant to the diff and asserts compliance. This is the loop closure:

> Memory learns from PRs → memory enforces what it learned on future PRs.

This is the most direct realization of the "team taste" feature. Every PR that gets human-corrected feeds the rule that prevents the next agent from making the same mistake.

## Eviction, decay, importance

Per the Mem0 2026 state-of-memory report and A-MAC adaptive admission control:

- **Multi-dimensional importance:** future utility, factual confidence, semantic novelty, temporal recency, content type prior.
- **Ebbinghaus exponential decay** on recency; reinforce-on-access (frequently retrieved memories live longer).
- **TTL:**
  - Hot (Redis): minutes–hours.
  - Episodic (pgvector): 30–90 days, importance-weighted.
  - Procedural (FalkorDB): no TTL; lifecycle via `status`.
- **Bounded growth via importance-thresholded admission.** Below-threshold candidates are dropped at write time rather than evicted later.
- **Working-set discipline:** keep retrieval-router output ≤ 7K tokens. Don't dump entire repos into prompts — that's what the "context window is RAM not storage" Mem0 thesis is about.

## Retrieval router

Multi-signal hybrid retrieval. On every agent query:

1. **Exact-match key lookup** (Redis): current branch state, last tool call.
2. **Semantic recall** (pgvector / Qdrant): top-K snippets by embedding similarity, filtered by tenant_id + repo_id + file-glob scope.
3. **Procedural lookup** (FalkorDB): conventions whose scope matches the current file path or category.
4. **Importance re-ranking:** combine A-MAC importance with semantic similarity score.
5. **Token budget enforcement:** total context ≤ 7K tokens; drop lowest-scored items to fit.

Cached aggressively (1h TTL on the router's output for the same query+context pair).

## API surface (agent-facing)

```typescript
twin.memory.recall(query: string, scope: Scope): Memory[]
// Multi-signal retrieval. Returns up to 7K tokens of relevant memory.
// Scope = { repo, file_glob, category } | "all"

twin.memory.note(fact: string, source: SourceRef): MemoryId
// Explicit save. Used when the agent learns something the distiller
// would miss — e.g., a user correction in the current task.

twin.memory.conventions(scope: Scope): Convention[]
// Returns active conventions for the given scope. Used at plan time
// and during verifier's compliance check.
```

The agent does not directly read or write the underlying stores. All access goes through `twin.memory.*`.

## What's deliberately not in scope for v1

- **Agent-to-agent memory sharing** beyond cross-tenant federated abstractions.
- **Visual memory** (screenshot retrieval, diagram understanding).
- **Voice memory** (transcribed stand-ups, recorded code reviews).
- **End-to-end encrypted memory** (E2EE on the customer's vault key — interesting for v2 enterprise tier).

See [06-research/memory-bootstrap.md](../06-research/memory-bootstrap.md) for the full cold-start strategy.
