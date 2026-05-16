# ADR-003: Per-tenant procedural memory as the primary moat

**Status:** Accepted  
**Date:** 2026-05-15

## Context

Models commoditize quarterly. MCP standardized tools. ACP standardizes agent-host protocols. UX features (multi-agent, manager view, etc.) commoditize within months. What doesn't commoditize is **lived experience with a specific team's codebase and conventions.**

The team-conventions problem is universal: every team's PR review comments contain knowledge that takes years for a new engineer to internalize. Existing agents (Cursor Memories, Claude Code Skills, AGENTS.md) require users to *write* the conventions explicitly. This is brittle, manual, and lossy.

Mining PR review comments, post-mortems, and ADRs into a *learned* team-conventions graph is technically tractable in 2026 (Mem0's hierarchical extraction, Graphiti's temporal KG, FalkorDB) but no product has shipped it.

## Decision

Crucible's procedural memory is a per-tenant temporal knowledge graph of team conventions, learned passively from:

- PR review comments (`(commenter, requested_change_type, code_pattern, accepted?)` triples)
- Incident post-mortems (`(trigger → action → outcome)` chains; "never do X" anti-patterns)
- Architecture Decision Records
- Merged code diffs (as implicit positive examples)

Architecture:

- **Backend:** FalkorDB with Graphiti abstraction; bi-temporal edges (valid_from / valid_to).
- **Distillation:** background worker using Mem0's hierarchical extraction algorithm with schema-constrained decoding.
- **Filtering:** LLM-as-judge on every memory write (defense against prompt-injection via PR comments).
- **Decay:** Ebbinghaus exponential on recency; reinforce-on-access; status lifecycle (`active | drifting | superseded`).
- **Federation:** cross-tenant abstractions allowed only when ≥5 tenants agree on a category-form rule.

The memory layer is:

1. **Read** by the agent at plan time and during reasoning.
2. **Read** by the verifier during the compliance check (closing the learning loop).
3. **Written** explicitly by agents via `twin.memory.note`.
4. **Written** passively by the background distiller.

## Consequences

### Positive

- **Compounding stickiness.** Every PR review feeds the graph. Day-30 customer experience materially outperforms day-1 (typical: 91% → 97% convention compliance over four weeks). Switching to a competitor loses 30+ days of learned taste.
- **Solves "generic AI aesthetic / convention drift" complaint** without requiring users to write rules manually.
- **Convention drift detection** as a customer-visible feature — the system surfaces "your convention X is aging" before defects pile up.
- **Onboarding becomes magical.** Cartographer mines existing PR history; day-1 agent already speaks the team's style.
- **Verifier becomes smarter** — checks compliance against team rules, not just generic best practices.

### Negative

- **Cold-start problem.** New customers have no PR history. Mitigation: OSS-derived defaults (Tier A–D corpus, ~400 active rules on a fresh Next.js+FastAPI repo). See [06-research/memory-bootstrap.md](../06-research/memory-bootstrap.md).
- **Prompt-injection attack surface.** PR comments are attacker-controllable. Mitigation: LLM-as-judge filter on every write, plus cross-source agreement threshold.
- **Cross-tenant leakage risk.** Per-tenant isolation everywhere; federation only to anonymized categorical form. Hard requirement; tested.
- **Storage growth.** Procedural memory grows monotonically (no TTL on active conventions). Mitigation: status lifecycle; superseded conventions archived (not deleted) at a fixed retention.

## Alternatives considered

### Alternative 1: User-written rules only (Cursor Memories / AGENTS.md model)

Require users to manually write `.cursorrules` or AGENTS.md. **Rejected**:

- Brittle, manual, lossy.
- Doesn't compound; the file is what it is.
- Users have to be senior enough to know what conventions to write down.

### Alternative 2: Train a per-customer fine-tune

Fine-tune a small model on the customer's PR history. **Rejected for v1**:

- Compute cost is significant.
- Update latency is days/weeks, not real-time.
- Per-tenant model artifacts complicate compliance and storage.
- The Graphiti+FalkorDB approach gets 80% of the benefit at 5% of the operational complexity.

(May revisit for v2 enterprise tier if customer pressure justifies it.)

### Alternative 3: Use vector store only, no graph

Episodic memory in pgvector; skip the graph layer. **Rejected**:

- Conventions have *relationships* (this rule supersedes that one; this rule conflicts with that one; this rule is a refinement of that). Graph structure captures these natively; flat vectors don't.
- Drift detection needs temporal edges; vectors don't model time well.

### Alternative 4: Single global "consensus memory"

Aggregate across tenants into one shared memory. **Rejected**:

- Cross-tenant data leakage.
- Customers explicitly want *their* taste, not the consensus.
- Federation (Alternative-considered-and-accepted) gives the global-common-knowledge benefit without the privacy violation.

## References

- [01-architecture/memory-layer.md](../01-architecture/memory-layer.md)
- [06-research/memory-bootstrap.md](../06-research/memory-bootstrap.md)
- [ADR-006](ADR-006-falkordb-over-alternatives.md)
