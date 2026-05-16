# Product Vision

## One-line

Crucible is the AI engineer that tests every change in a digital twin before touching your real code.

## The problem we exist to fix

By mid-2026 the frontier-feature race in AI coding agents has commoditized: long-horizon autonomous loops, sub-agent orchestration, MCP tool calling, persistent memory, voice input, computer use, and AGENTS.md-style convention files are now table stakes across Cursor, Claude Code, Codex, Windsurf, Devin, Antigravity, GitHub Copilot, and Replit Agent.

Yet the top user pain points have not moved. Across Reddit, Hacker News, GitHub issues, and forum threads in the first half of 2026, the same complaints dominate:

1. **Memory amnesia between sessions** — 68 minutes/day lost to re-orientation per published studies.
2. **Runaway costs** — Uber burned its full-year 2026 Claude Code budget in four months; individuals report $200/day burns from single stuck agent sessions.
3. **Destructive actions without guardrails** — the PocketOS incident (April 24, 2026) saw a Claude-powered agent delete an entire production database plus backups in 9 seconds after finding an API token in an unrelated file.
4. **Hallucinated APIs and "lies about completion"** — agents claim tests pass when they were skipped or mocked; phantom bugs appear in 20–30% of AI-generated codebases.
5. **Infinite explore loops** — Opus 4.6 specifically called out for "thinking loops that burn money with zero output."
6. **Breaks working code / ignores "do not touch"** — rogue edits, destruction of files explicitly flagged.
7. **Large-repo blindness** — agents reinvent helpers, violate layer boundaries, can't see architecture.
8. **Generic AI aesthetic & convention drift** — UI output all looks the same; ignores team's libraries and patterns.
9. **Vibe-coding wall after MVP** — non-tech founders trapped with unmaintainable code.
10. **Rate-limit / fair-use surprises** — Claude Code session drains in 90 minutes; Cursor plans depleted in 4 hours.

Every incumbent treats these as bugs to be patched. **Crucible treats them as a single architectural failure**: agents act directly on real systems with no separation between "try" and "commit."

## What Crucible is, structurally

Three architectural pillars make the failure modes above impossible by construction rather than less likely by patch:

### Pillar 1 — Digital-Twin-First Execution

Every meaningful agent action runs in an ephemeral mirror of the user's project, never on real systems. The mirror includes:

- **Filesystem twin** — git worktree + overlayfs upper inside a Firecracker microVM (via E2B or self-hosted).
- **Database twin** — Neon copy-on-write Postgres branch, instant, scoped to the task.
- **Service twin** — Hoverfly replay tapes of recorded production traffic, PII-scrubbed at capture time, with LLM-generated stubs for cold-start endpoints.
- **Secrets twin** — Infisical-issued dynamic credentials, sub-minute TTL, twin-scoped only. Production credentials live in an HSM-backed vault the agent process literally cannot syscall to.
- **Network egress** — Cilium/Tetragon eBPF policy drops any TCP connection outside the per-task manifest allowlist with `SIGKILL`.

Changes are promoted to real systems only via an explicit `twin.promote(bundle)` call that triggers a signed approval ceremony, KMS-backed credential lease, and Argo Rollouts canary with auto-rollback.

### Pillar 2 — Living Contracts (Verifier Ladder)

The agent cannot mark a task complete without a separate verifier process — running a *different model family* — confirming the change. Verification escalates by criticality:

- **Tier 0** — mutation-tested unit tests on the diff (mutmut, stryker, cargo-mutants). Default for every change.
- **Tier 1** — property-based testing + fuzz (hypothesis, fast-check, proptest, rapid). Default for non-trivial feature work.
- **Tier 2** — schemathesis contract testing + deterministic simulation testing (Antithesis or in-house TigerBeetle-style simulator). For multi-component state.
- **Tier 3** — formal verification (Dafny, Lean 4, TLA+, Z3, Kani). Auto-triggered on `@critical` paths via a multi-signal classifier — see [tier3 trigger automation](../06-research/tier3-trigger-automation.md).
- **Tier 4** — honest CI: reproducible Nix/Bazel rebuild + SLSA-L3 in-toto attestation signed via Sigstore Rekor v2. The agent literally cannot forge a green pipeline.

Cross-family verification means executor and verifier disagree on different inputs. Opus 4.7 paired with Gemini 3.1 Pro produces real error decorrelation, not just two passes of the same lineage.

### Pillar 3 — Bounded Plans + Signed Replayable History

Every task starts with a planning contract showing cost, time, files touched, and risks **before** the user approves. The agent has a hard retry budget (3 attempts per subgoal, then halt-and-ask) and a hard dollar budget per task. The Opus-4.6 infinite-explore-loop class of bug becomes architecturally impossible.

Every action — every file read, every tool call, every shell command — is recorded as a signed step in an append-only Sigstore Rekor log. The user can replay, fork from any step, blame any change to a specific decision, and audit the entire history for compliance.

## Who this is for

**Primary:** engineering teams of 5–200 building production systems where correctness matters — fintech, healthtech, infra, B2B SaaS, regulated industries. The "senior engineer hates current agents" demographic. They've felt the pain of Cursor breaking working code, Devin taking days and failing, Claude Code billing surprises, and they will pay a premium for an agent they can actually let run overnight.

**Secondary:** solo founders shipping real revenue businesses (not toys). They need an agent that owns the full SDLC, including post-merge ops, without ever putting their production database one syscall away from `DROP TABLE`.

**Explicitly not for:** greenfield prototyping where speed-of-iteration is the only thing that matters. Cursor, Bolt, and Lovable own that turf and will keep it. Crucible competes one tier up the value chain.

See [target-users.md](target-users.md) for full ICP and persona definitions.

## What success looks like

- A developer can assign Crucible a feature ticket on Friday evening, walk away, and on Monday find a verified PR merged behind a feature flag, with zero hand-wringing about destructive changes or token burn.
- A regulated-industry buyer can deploy Crucible air-gapped, point it at a legacy Rails 4 monolith, and get module-by-module modernization with cryptographic provenance for every line of agent-touched code.
- A senior engineer reviewing a Crucible PR sees the plan, the verifier's report, the property tests, the conventions-applied summary, and the in-toto attestation — and approves in 90 seconds because every claim is independently checkable.

## What we explicitly will not build

- A new IDE. Crucible is editor-agnostic; integrates via MCP and the Agent Client Protocol.
- A new model. Crucible is a routing layer over Anthropic + Google + OpenAI + open-weights frontier models.
- A new vector DB / graph DB / sandbox runtime. Every infrastructure layer is composed from best-in-class commodity components.
- A vibe-coding "build an app from a prompt" surface. That market is saturated, and the trust positioning is incompatible with it.

## How this differs from the alternatives

| Property | Cursor | Devin | Claude Code | **Crucible** |
|---|---|---|---|---|
| Primary execution surface | Real repo | Real cloud IDE | Real repo | **Digital twin** |
| Verification | None (or "tests passed per agent") | Internal | Internal | **Cross-family verifier, mandatory** |
| Destructive ops gate | None | None | None | **Typed proposals, HSM-signed approval** |
| Budget transparency | After-the-fact credit drain | After-the-fact ACUs | Weekly limits, opaque | **Plan-time $/time preview, hard cap** |
| Memory | Cursor Memories (per-user) | Devin Wiki | Skills/AGENTS.md | **Per-tenant procedural graph, learns from PRs** |
| Provenance | None | None | Logged, not signed | **In-toto + Sigstore Rekor v2** |
| Self-host / air-gap | No | No | Limited | **Day-one** |

The unique combination: trust by construction, verified by architecture, priced by outcome, deployable on-prem.

## Brand voice

Anti-vibe-coding, pro-engineering-rigor, dry, evidence-driven. The tagline lives:

> "Cursor lets your agent ship in 9 seconds. Crucible makes sure those 9 seconds don't end your company."

Marketing copy never says "lightning fast" or "10x productivity." It says "verified," "auditable," "reproducible." It cites incidents. It shows attestations. It assumes the reader is a senior engineer who has been burned and is looking for the first AI tool that doesn't ask for blind trust.
