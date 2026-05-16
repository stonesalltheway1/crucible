# Phase Prompts

Self-contained session prompts for building Crucible v1 and v2. Each prompt is designed to be pasted as the first message of a fresh session — the agent reads only the prompt and the design docs, then executes.

## How to use

1. Start a new session.
2. Open the prompt file for the next phase.
3. Paste its contents as your first message.
4. The agent reads docs, does currency-check research, builds, and writes an end-of-session report.
5. The report becomes the handoff context for the next phase.

Each prompt assumes the prior phase's end-of-session report exists at `docs/PHASE-N-REPORT.md` and the corresponding memory file at `C:\Users\Eric\.claude\projects\E--AI-Coding-Agent\memory\project_crucible_phaseN.md`.

## v1 phases (~19 agent-days total → ~7–8 focused sessions compressed)

| Phase | Block | What ships | LoC est. |
|---|---|---|---|
| [Phase 1](phase-01-control-plane.md) | Agent Control Plane (Block 1) | Type system, model router, plan builder, budget enforcer, attestation pipeline, CLI | ~15K |
| [Phase 2](phase-02-twin-runtime.md) | Twin Runtime core (Block 2 critical path) | E2B sandbox, syscall shim, destructive-op gate, Neon driver, Hoverfly basic, secrets sidecar, SDK | ~25K |
| [Phase 3](phase-03-twin-runtime-breadth.md) | Twin Runtime breadth (Block 2 fill-in) | Full PII pipeline (Presidio + spaCy + FF3-1), multi-engine DB, raw Firecracker, WASM tool runner | ~20K |
| [Phase 4](phase-04-verifier-pipeline.md) | Verifier Pipeline (Block 3) | Four-tier ladder, cross-family routing, per-language runners, Dafny dispatcher | ~25K |
| [Phase 5](phase-05-memory-layer.md) | Memory Layer (Block 4) | Three-store architecture, distiller, OSS-corpus bootstrap, convention drift detector | ~20K |
| [Phase 6](phase-06-promotion-and-provenance.md) | Promotion Contract + Provenance (Blocks 5+6) | Rego policy, KMS leases, Argo Rollouts, GrowthBook, full Sigstore Rekor publish | ~18K |
| [Phase 7](phase-07-agent-facing-ux.md) | Agent-Facing UX (Block 7) | Web console, IDE plugins, CLI completion, GitHub App, Slack bot | ~25K |
| [Phase 8](phase-08-onboarding-and-v1-launch.md) | Onboarding + v1 final integration (Block 8) | Cartographer, shadow-recording, Helm chart, air-gap installer, Stripe billing, v1 launch criteria validation | ~20K |

**v1 ships at end of Phase 8.** Total: ~168K LoC across 8 focused sessions. The build plan in `docs/07-roadmap/build-plan-agent-days.md` had 19 agent-days; we compress to 8 sessions by parallelizing currency research and using fan-out where applicable.

## v2 phases (signal-driven; ~6-month horizon per `docs/07-roadmap/v2-vision.md`)

| Phase | Pillar | What ships | LoC est. |
|---|---|---|---|
| [Phase 9](phase-09-verifier-deepening.md) | A. Verifier deepening | Custom Crucible verifier model, multi-verifier ensemble, full Tier 3 (Lean/TLA+/Kani/Z3), customer extension API | ~20K |
| [Phase 10](phase-10-memory-deepening.md) | B. Memory deepening | Federation graduations, visual/screenshot memory, voice memory, E2EE with customer KMS | ~18K |
| [Phase 11](phase-11-twin-runtime-deepening.md) | C. Twin runtime deepening | GPU sandbox, mobile twins (iOS/Android), embedded/firmware, multi-region | ~25K |
| [Phase 12](phase-12-pricing-and-specialization.md) | D + E. Pricing + vertical wedge | Complexity-banded Outcome, SLA tier, OSS-maintainer tier, plugin marketplace, Legacy Modernizer OR Autonomous Operator specialization | ~20K |
| [Phase 13](phase-13-operational-hardening.md) | F. Operational hardening | SOC 2 controls tooling, HIPAA SaaS, FedRAMP prep, EU residency | ~12K |
| [Phase 14](phase-14-cross-ide-identity-and-v2-launch.md) | G + v2 launch | Cross-IDE agent identity, v2 integration testing, launch criteria | ~15K |

**v2 ships at end of Phase 14.** Total v2: ~110K LoC. Phase order in v2 is **signal-driven**: prioritize whichever pillar's customer demand surfaces strongest. The order here is a default sequence, not a fixed pipeline.

## Convention notes

- Every prompt starts by reading prior reports + memory. The chain compounds.
- Every prompt forces parallel currency-check research before code (vendor APIs drift).
- Every prompt scopes IN and OUT explicitly so phases don't sprawl.
- Every prompt ends with an end-of-session report that includes the next phase's prompt.
- Every prompt enforces the same quality bar: mutation ≥85% on diff, hermetic Nix build, lints clean, full SDK contract tests pass.
- Every prompt forbids silently swapping library picks — flag and ask.
- Every prompt eats the dogfood: we use Crucible's own verifier ladder on Crucible's code.

## When to skip / reorder

- Skip a phase if its scope has been delivered out-of-band (e.g., a customer wrote a Helm chart in Phase 7 and you don't need to in Phase 8).
- Reorder if customer signal demands (e.g., enterprise pilot wants air-gap before SaaS UX → pull Phase 8 forward).
- Cut scope if blocked (e.g., Antithesis license unsigned → in-house DST harness in Phase 4 only).

Each phase's end-of-session report should flag deferred work explicitly so it doesn't disappear.
