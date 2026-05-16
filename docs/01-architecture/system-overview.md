# System Overview

A single-page mental model of Crucible. Each component has its own deep-dive doc in this directory.

## The diagram

```
┌──────────────────────────────────────────────────────────────────────────┐
│                      AGENT CONTROL PLANE (Crucible Core)                 │
│                                                                          │
│   ┌──────────────┐   ┌──────────────┐   ┌──────────────────────────┐   │
│   │ Task Router  │──▶│ Plan Builder │──▶│ Bounded Budget Enforcer  │   │
│   │ (Tier 0)     │   │ (Tier 1/2)   │   │ (cost, time, retry cap)  │   │
│   └──────────────┘   └──────────────┘   └────────────┬─────────────┘   │
│                                                       │                  │
│   ┌──────────────────────────────────────────────────▼───────────────┐ │
│   │              Model Router  (5 tiers, ~12 models)                  │ │
│   └──────────────┬────────────────────────────────────────────────────┘ │
└──────────────────┼───────────────────────────────────────────────────────┘
                   │
                   ▼
┌──────────────────────────────────────────────────────────────────────────┐
│                          TWIN RUNTIME (per task)                         │
│                                                                          │
│  Sandbox: E2B / Firecracker        DB: Neon CoW branch                   │
│  ├ git worktree (depth 1)          ├ instant clone from `main`           │
│  ├ overlayfs upper                 └ scoped DSN, TTL = task              │
│  ├ WASM tool runner                                                      │
│  └ syscall shim ────────────┐      Services: Hoverfly tapes              │
│      ↑                       │      ├ content-addressed                  │
│  egress proxy ←─────────┐    │      └ PII-scrubbed at record             │
│  (Cilium/mitmproxy)     │    │                                           │
│      ↓                  │    │      Secrets: Infisical scoped token      │
│  Destructive Op Gate ───┘    │      ├ TTL = task                         │
│      (cosign-signed)         │      └ vault-only, agent cannot syscall   │
└─────────────────┬────────────┴───────────────────────────────────────────┘
                  │
                  ▼
┌──────────────────────────────────────────────────────────────────────────┐
│                  VERIFIER PIPELINE  (separate process, different model)  │
│                                                                          │
│  Tier 0: mutation-tested unit (mutmut/stryker/cargo-mutants)             │
│  Tier 1: PBT + fuzz (hypothesis/fast-check/proptest/rapid)               │
│  Tier 2: schemathesis contract + DST (Antithesis or in-house)            │
│  Tier 3: Dafny/Lean/TLA+ for @critical paths                             │
│  Tier 4: SLSA-L3 reproducible-build attestation via Sigstore Rekor v2    │
└─────────────────┬────────────────────────────────────────────────────────┘
                  │
                  ▼
┌──────────────────────────────────────────────────────────────────────────┐
│                          MEMORY LAYER  (per-tenant)                      │
│                                                                          │
│  Redis (hot ctx, mins)  pgvector (episodic+semantic, 30–90d)             │
│                         FalkorDB+Graphiti (procedural conventions, ∞)    │
│                         Background Distillation Worker (PR/incident KG)  │
└─────────────────┬────────────────────────────────────────────────────────┘
                  │
                  ▼
┌──────────────────────────────────────────────────────────────────────────┐
│                          PROMOTION CONTRACT                              │
│                                                                          │
│  PromotionBundle → KMS-signed approval → Argo Rollouts canary            │
│                                       → GrowthBook flag + auto-rollback  │
│                                       → in-toto attestation → Rekor      │
└──────────────────────────────────────────────────────────────────────────┘
```

## The six layers and what each owns

### 1. Agent Control Plane

The single entry point. Receives a task description (from IDE, MCP host, REST API, Slack, GitHub issue), routes it through:

- **Task Router** classifies the task (read-only inspection? feature add? refactor? incident response?) and selects a planning tier.
- **Plan Builder** produces a `Plan` artifact — files-touched estimate, cost estimate, time estimate, risk callouts, retry budget — that the user must approve.
- **Bounded Budget Enforcer** runs in-process throughout the task; halts execution if the dollar cap, retry cap, or wall-clock cap is exceeded.
- **Model Router** dispatches every LLM call to the right tier model with per-call cache strategy (see [model-routing.md](model-routing.md)).

Owns no state of its own; reads the per-tenant memory layer and writes attestations to the provenance pipeline.

### 2. Twin Runtime

The execution surface for everything the agent does. Per-task isolated environment with:

- **Filesystem twin** — Firecracker microVM (via E2B for hosted; raw Firecracker + ZFS for self-hosted) containing a git worktree on the task's base SHA, overlayfs upper for mutations, WASM-sandboxed tool runner, and a syscall shim that intercepts destructive operations.
- **Database twin** — Neon Postgres CoW branch (or per-DB-engine equivalent — PlanetScale for MySQL, Turso for SQLite, snapshot-restore for MongoDB). Created in 1–2 seconds, deleted on task complete.
- **Service twin** — Hoverfly replay tapes for HTTP/gRPC, content-addressed by (service, endpoint, request hash). PII-scrubbed at capture via Presidio + spaCy + FF3-1. See [twin-runtime.md](twin-runtime.md) and [06-research/tape-coverage-strategy.md](../06-research/tape-coverage-strategy.md).
- **Secrets twin** — Infisical-issued dynamic tokens, TTL = task duration, scoped to twin-only resources. Real prod creds live in HSM-backed vault on a separate VPC; agent cannot syscall to it.
- **Network egress** — Cilium/Tetragon eBPF policy with `SIGKILL`-on-violation; per-task manifest declares allowed hosts.

The twin runtime is the load-bearing trust component. If it's compromised, the whole system is.

### 3. Verifier Pipeline

Runs as a separate process, with a **different model family from the executor**, after the agent claims task completion. Four tiers escalate by criticality:

- **Tier 0** — diff-scoped mutation testing on existing unit tests. Default for all changes.
- **Tier 1** — property-based testing + fuzz harness, with both example-based and property tests required (LLM-authored PBT alone catches only 68% of bugs; combined catches 81%).
- **Tier 2** — schemathesis OpenAPI contract testing + deterministic simulation testing for stateful systems. Antithesis on enterprise tier; in-house TigerBeetle-style simulator for OSS tier.
- **Tier 3** — formal verification (Dafny, Lean, TLA+, Kani, Z3) for `@critical` paths, auto-classified by a multi-signal scorer described in [06-research/tier3-trigger-automation.md](../06-research/tier3-trigger-automation.md).
- **Tier 4** — honest CI: hermetic Nix/Bazel rebuild + SLSA-L3 in-toto attestation signed via Sigstore Rekor v2. The verifier independently rebuilds the artifact and compares hashes.

The verifier's sole authority is to issue or withhold a `VerifierApproval`. Without it, the agent's task is not marked complete and no `PromotionBundle` can be generated.

### 4. Memory Layer

Per-tenant, three-store architecture:

- **Redis (hot)** — current task context, last 50 tool calls, active branch state. TTL minutes–hours.
- **pgvector (episodic + semantic)** — session transcripts, retrieved snippets, prior agent decisions. Importance-scored (A-MAC: utility × confidence × novelty × recency). TTL 30–90 days. Row-level security on tenant_id + repo_id.
- **FalkorDB + Graphiti pattern (procedural)** — team conventions, incident patterns, supersession chains, ADR-derived decisions. Bi-temporal edges (valid_from / valid_to). No TTL; lifecycle via `status: active | drifting | superseded`.

A **background distillation worker** runs continuously, ingesting PR review comments, post-mortems, ADRs, and merged code; emitting new convention candidates via Mem0's hierarchical extraction algorithm; merging/rejecting against the existing graph; flagging drift.

Memory is read by the agent on every plan, written by the agent on explicit `twin.memory.note` calls, and reinforced by the distillation worker passively.

### 5. Promotion Contract

The bridge from twin to real. When the agent calls `twin.promote(bundle)`:

1. **Provenance verification** — every in-toto attestation in the bundle is validated against Sigstore Rekor; OIDC subjects checked.
2. **Rego policy** — bundled policies (auto-approve trivial; human-approve schema changes; human-approve critical-path touches) evaluate the bundle and emit Allow / Deny / Require-Human-Approval.
3. **Human approval (if required)** — Slack button or web UI; signed by the approver via Sigstore keyless OIDC.
4. **KMS-signed credential lease** — AWS KMS / GCP Cloud HSM signs a single-use, action-scoped, time-boxed credential. Consumed by the deploy pipeline. Never returned to the agent.
5. **Progressive delivery** — Argo Rollouts canary with traffic mirroring; AnalysisTemplate watches Prometheus SLOs; GrowthBook feature flag for fast rollback.
6. **Final attestation** — promotion result published to Sigstore Rekor.

### 6. Provenance pipeline (cross-cutting)

Every meaningful action — file read, tool call, shell command, test run, plan approval, verifier decision, promotion — emits an in-toto attestation signed via Sigstore keyless OIDC. Attestations are published to Sigstore Rekor v2 (public for SaaS tier; self-hosted Rekor for enterprise). OTel spans are emitted in parallel to Honeycomb/Tempo for observability.

The pipeline produces the audit trail for compliance and the replay log for debugging.

## How a task flows end-to-end

1. **Submit.** User submits task ("add Stripe webhook handler for refund events") via IDE/MCP/REST/Slack.
2. **Plan.** Control Plane builds a Plan; user approves. Plan locked into Bounded Budget Enforcer.
3. **Spawn twin.** Twin Runtime creates sandbox + Neon branch + Hoverfly tape mounts + Infisical scoped token.
4. **Execute.** Agent runs through SDK; every action emits in-toto attestation.
5. **Verify.** Verifier process runs Tier 0/1/2/3 ladder as required, plus Tier 4 reproducible-build check. Emits `VerifierApproval` or `VerifierRejection`.
6. **Bundle.** If approved, control plane produces `PromotionBundle` and presents to user.
7. **Promote.** User approves promotion (or Rego policy auto-approves trivial). KMS signs credential lease. Argo Rollouts executes canary. Auto-rollback on SLO regression.
8. **Land.** Final attestation published. Procedural memory updated with any new patterns learned. Task complete.

Total wall-clock: median task ~5–15 minutes; complex task with Tier 3 verification ~30–60 minutes.

## Deployment topologies

- **SaaS (multi-tenant cloud).** All layers hosted by Crucible. Twin runtimes scheduled on managed Firecracker pool. Memory layer per-tenant isolation via RLS + per-tenant Vectorize-style namespaces.
- **Self-hosted (single-tenant cloud or on-prem).** Customer runs the entire stack in their VPC. Bring-your-own Neon (or self-hosted Postgres + pg_dump branching), bring-your-own Vault, bring-your-own KMS. The orchestrator runs in Kubernetes.
- **Air-gapped.** Same as self-hosted but with offline-installer bundles, self-hosted Rekor, local-model fallback (Llama 4 Scout / DeepSeek V4-Pro). For FedRAMP / defense / banking buyers.

See [04-operations/self-hosted-install.md](../04-operations/self-hosted-install.md) for the install guide.

## Why this architecture

Every layer maps to a specific failure mode in incumbent agents:

| Failure | Layer that prevents it |
|---|---|
| PocketOS-style destructive incidents | Twin Runtime (syscall shim, destructive-op gate, secrets isolation) |
| Hallucinated APIs / fake test pass | Verifier Pipeline (cross-family, four tiers) |
| Infinite loops / token burn | Control Plane (Bounded Budget Enforcer, retry cap) |
| Memory amnesia | Memory Layer (per-tenant procedural graph) |
| Generic AI aesthetic / convention drift | Memory Layer (background distiller learns team taste) |
| No audit trail | Provenance pipeline (signed attestations everywhere) |
| Surprise bills | Control Plane (plan-time cost preview, hard cap) |

The architecture is the brand promise. Every block exists because a specific failure mode in the incumbents demands it.

## What's deliberately not here

- **A new IDE.** Crucible integrates via MCP and ACP into existing IDEs.
- **A new LLM.** Crucible routes to frontier APIs (and local-model fallbacks).
- **A built-in fine-tuning pipeline.** Out of scope for v1.
- **A built-in chat interface.** The IDE is the chat. The web UI is for plan approval, task monitoring, and memory browsing.

See [01-architecture/twin-runtime.md](twin-runtime.md), [verifier-pipeline.md](verifier-pipeline.md), [memory-layer.md](memory-layer.md), [model-routing.md](model-routing.md), [promotion-contract.md](promotion-contract.md), and [threat-model.md](threat-model.md) for component deep-dives.
