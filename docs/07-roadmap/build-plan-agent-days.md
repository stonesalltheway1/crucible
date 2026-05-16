# Build Plan in Agent-Days

The v1 plan, sized in agent-days. **One focused agent-day ≈ 10–20K LoC of working code** (calibration anchor: 350K-LoC stable web app shipped in ~3 months solo with AI agents).

The plan assumes one focused agent in continuous use; multiple agents fan out and compress the calendar further.

## Total v1 scope

**~19 agent-days, ~315K LoC.** Roughly three calendar weeks of continuous focused agent work.

| Block | Agent-days | LoC est. | Critical-path? |
|---|---|---|---|
| Agent Control Plane | 3 | ~50K | yes |
| Twin Runtime | 4 | ~70K | yes (the largest, hardest piece) |
| Verifier Pipeline | 3 | ~50K | yes |
| Memory Layer | 2 | ~35K | yes |
| Promotion Contract | 1 | ~15K | yes |
| Provenance pipeline | 1 | ~15K | yes |
| Agent-facing UX | 3 | ~50K | partial (Web console can land staged) |
| Onboarding / installer | 2 | ~30K | partial |
| **Total** | **~19** | **~315K** | |

## Per-block detail

### Block 1: Agent Control Plane (3 agent-days)

- **Day 1.** Task Router, Plan Builder (gRPC service in Go); model-routing module with 5-tier dispatch; cost-meter telemetry pipeline.
- **Day 2.** Bounded Budget Enforcer (sidecar pattern); retry-cap state machine; per-tenant policy loader.
- **Day 3.** REST + gRPC + MCP server surface; auth integration (Clerk/WorkOS); webhook event publisher.

**Output:** a control plane that accepts task submissions, builds plans, enforces budgets, dispatches to the twin runtime.

### Block 2: Twin Runtime (4 agent-days — the heaviest block)

- **Day 1.** Sandbox driver (E2B integration + raw Firecracker fallback); overlayfs + git worktree wiring; lifecycle management.
- **Day 2.** Neon branch driver; per-engine adapters (MySQL/Turso/Mongo stubbed); Infisical sidecar.
- **Day 3.** Hoverfly tape driver; PII scrubber (Presidio + spaCy + FF3-1); tape decision-tree engine; `X-Crucible-Tape` header logic.
- **Day 4.** **The hardest single piece: syscall shim + destructive-op gate.** Multi-layer enforcement (cmd-line parse + ptrace + eBPF). Egress proxy with Cilium/Tetragon policy. SDK surface (Go, TS, Python, Rust generated from gRPC).

**Output:** twin spawns in <300ms, agent SDK calls work end-to-end, destructive ops route to typed proposals.

### Block 3: Verifier Pipeline (3 agent-days)

- **Day 1.** Per-language Tier 0 + Tier 1 runners for top 6 languages (Python, TS, Rust, Go, Java, Swift). Each is mostly "drive an existing tool" wrapper; integration code is the bulk.
- **Day 2.** Tier 2 schemathesis integration + in-house DST harness (TigerBeetle-style virtualized clock+disk+net for our Postgres+Go stack).
- **Day 3.** Tier 3 dispatcher with Dafny adapter as default; Lean + TLA+ + Kani + Z3 stubs (deferred-load). Tier 4 Nix hermetic-rebuild verifier + SLSA-L3 attestation pipeline.

**Output:** verifier process spins up in a separate sandbox with a different model, runs Tier 0–4 as required, emits `VerifierApproval` or structured rejection.

### Block 4: Memory Layer (2 agent-days)

- **Day 1.** Redis cache; pgvector schema + RLS; FalkorDB + Graphiti integration; multi-signal retrieval router with 7K-token budget enforcement.
- **Day 2.** Background distillation worker (Mem0 hierarchical extraction algorithm); importance scorer + GC; LLM-as-judge filter; convention drift detector.

**Output:** memory layer reads on every plan, writes from distiller and explicit `twin.memory.note`, surfaces conventions to verifier.

### Block 5: Promotion Contract (1 agent-day)

- KMS signing pipeline (AWS KMS, GCP Cloud HSM, YubiHSM adapters).
- Argo Rollouts adapter + AnalysisTemplate generator.
- GrowthBook flag wiring.
- Rego policy evaluation.
- Slack approval bot.

**Output:** verified bundles flow through policy → human approval → KMS lease → canary rollout → final attestation.

### Block 6: Provenance Plumbing (1 agent-day)

- In-toto attestation generators for each predicate type.
- Sigstore Cosign keyless OIDC signing.
- Rekor v2 publisher with local journaling fallback.
- OTel span enrichment with attestation UUIDs.

**Output:** every action emits an attestation; every attestation is verifiable end-to-end.

### Block 7: Agent-Facing UX (3 agent-days)

- **Day 1.** Web console foundation (Next.js + shadcn + Clerk auth); task dashboard; plan/budget viewer; cost dashboard.
- **Day 2.** Memory browser; convention drift reviewer; approval inbox; SLO dashboard.
- **Day 3.** VS Code extension (~3K LoC); JetBrains plugin (~3K LoC); Zed extension via ACP (~2K LoC); CLI (Go, Cobra-based, ~15K LoC).

**Output:** customers interact with Crucible through their preferred surface.

### Block 8: Onboarding / Installer (2 agent-days)

- **Day 1.** Repo Cartographer (orchestrates Sonnet 4.6 + tree-sitter + lint-config parsers); shadow-traffic recorder; AGENTS.md generator.
- **Day 2.** GitHub App install flow; Slack workspace integration; SaaS sign-up + tenant provisioning; Helm chart for self-hosted; air-gap installer bundle.

**Output:** customer goes from sign-up to first verified PR in < 30 min.

## Calendar shape

Three calendar weeks of continuous focused agent work, with appropriate buffer for inevitable rework:

```
Week 1: Twin Runtime (4d) + Agent Control Plane (3d)
Week 2: Verifier Pipeline (3d) + Memory Layer (2d) + Promotion (1d) + Provenance (1d)
Week 3: Agent-Facing UX (3d) + Onboarding (2d) + buffer
```

Fan-out reduces this further. Three agents working in parallel on Blocks 1, 2, 3 simultaneously compress Week 1 to ~1.5 calendar days of wall-clock.

## What adds time (in agent-days)

These genuinely expand scope:

- **Antithesis SaaS integration vs in-house DST.** +2 agent-days for in-house; +0 if Antithesis (paid).
- **Multi-language verifier coverage beyond Python/TS/Rust/Go.** ~+0.5 agent-day per additional language (Java/Kotlin/Swift/C++/etc.).
- **Self-hosted air-gap installer hardening.** +3 agent-days for full FedRAMP-track install + Sigstore Rekor self-hosted.
- **Real customer onboarding iteration.** Each design partner needs ~2 agent-days of bespoke Cartographer tuning + tape recording assistance until patterns stabilize.
- **Bazel alternative to Nix.** +2 agent-days if customer demand justifies.

## What doesn't add time

These look big but aren't, because the tooling does the heavy lifting:

- **15+ ADRs.** Already drafted; living docs.
- **Webhook event spec.** Standard pattern; ~2 hours wrapped in Block 1.
- **Public OSS releases of verifier harness + tape scrubber.** Split-and-publish from the monorepo; ~3 hours.
- **Public docs site.** Already structured as `docs/`; static site generator builds from MD; ~4 hours.
- **Stripe billing integration.** Stripe SDK + usage-metering; ~6 hours.
- **GitHub App + Slack bot scaffolding.** SDK-driven; ~4 hours each.

## Risks that genuinely slow the build

1. **Syscall shim correctness.** Multi-layer enforcement is fiddly; correctness invariants are the highest-stakes single piece in the codebase. Buffer: +1 day if first implementation has gaps.
2. **Cross-family verifier prompt engineering.** Verifier needs to NOT see executor's reasoning trace; needs to disagree adversarially without being pathologically strict. Buffer: +1 day for prompt iteration.
3. **PII scrubber false negatives.** Adversarial-test PII in real customer data may surface gaps. Buffer: +1 day for scrub-pipeline tuning.
4. **First customer's repo too large for Cartographer.** ~1M-LoC monorepos may exceed v1 limits. Buffer: chunked processing; +1 day.
5. **Sigstore Rekor v2 quirks.** v2 just GA'd; corner cases inevitable. Buffer: +0.5 day for fallback paths.

Adding these buffers: realistic v1 ~22 agent-days, ~350K LoC.

## After v1: what compresses further

- v2 fan-out: most blocks have natural parallelism. v2 features ship in 3–5 agent-days per feature.
- Customer-driven feature requests: 1–2 agent-days each for well-scoped requests.
- Quarterly Helm chart releases: hours of agent-work each, mostly version bumps + changelog.

## How to think about this calendar

The build plan is **honest about agent throughput**, not human-team cadence. Three weeks ≠ three months ≠ a quarter. The same plan in human-team estimation language would read "12 engineer-months" or "two quarters of a 3-person team" — those framings are not just irrelevant, they are misleading. They make ambitious projects look infeasible.

This is a real plan. Execute it.

## References

- [v1-mvp.md](v1-mvp.md) — what's in scope
- [v2-vision.md](v2-vision.md) — what comes next
- [02-engineering/repo-structure.md](../02-engineering/repo-structure.md)
