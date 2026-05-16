# ADR-012: Monorepo with per-component language choices

**Status:** Accepted  
**Date:** 2026-05-15

## Context

Crucible has ~10 distinct services + 4 SDKs + 9 per-language verifier integrations + IDE plugins + CLI + web console. Each has its own natural language fit (Go for orchestration, Rust for sandbox-adjacent perf, Python for LLM-SDK-heavy work, TypeScript for web).

Two coordination decisions:

1. **One repo or many?**
2. **One language for everything, or right-tool-for-each-job?**

## Decision

**One monorepo. Right-tool-per-component.**

- **One repo:** `crucible/` with the full system source.
- **One build system:** Nix flakes default (Bazel alternative).
- **One CI pipeline:** SLSA-L3 attested releases.
- **Per-component language picks:**

| Component | Language | Rationale |
|---|---|---|
| control-plane | Go | Single-binary deploy, gRPC story, predictable GC |
| twin-runtime | Rust | Firecracker integration, syscall shim perf, safety |
| verifier daemon | Go | Orchestrates per-lang processes; gRPC fan-out |
| distiller worker | Python | Best LLM SDK ecosystem; not perf-critical |
| promotion-gate | Go | OPA/Rego, KMS clients |
| web-console | TypeScript (Next.js) | Standard 2026 React stack |
| cli | Go | Cross-platform single binary |
| IDE plugins | TypeScript | Universal IDE plugin language |
| attestation-relay | Rust | Sigstore client mature; perf-critical |
| tape-scrubber | Python | Presidio is Python-native |
| memory-router | Go | Hot-path retrieval; latency-sensitive |
| cost-meter | Go | Hot-path; latency-sensitive |

SDKs published one per supported agent-host language (Go, TS, Python, Rust), all generated from one gRPC schema in `libs/twin-spec/`.

## Consequences

### Positive

- **Single coherent change set across components.** A schema change in `libs/twin-spec/` lands as one PR touching every consumer.
- **Single CI pipeline = single SLSA-L3 attestation surface.** Tier 4 hermetic-rebuild verification across the entire system in one place.
- **Right language per problem.** No "one language for everything" tax (Python doesn't drive the syscall shim; Rust doesn't author LLM extraction code).
- **Cross-component refactors are tractable.** Easier to evolve the architecture without coordination overhead.

### Negative

- **Build-graph complexity.** Bazel or Nix is required to make build times tractable; bare `make` or per-language tooling alone won't scale.
- **Polyglot operational surface.** Engineers need to debug Go and Rust and Python and TypeScript. Mitigation: each component has clear single-language ownership; cross-team rotations encouraged.
- **Onboarding new engineers takes longer.** They learn the repo, not just a service.
- **Repo size grows.** Mitigation: sparse-checkout for component-focused work; clean separation of generated artifacts.

### Trade-offs we accept

We pay polyglot-operational tax in exchange for per-component appropriateness. The team is senior enough to navigate this; the productivity win on the perf-critical components (twin-runtime in Rust vs. Go) is real.

## Alternatives considered

### Alternative 1: Multi-repo (one repo per service)

**Rejected**:

- Cross-service schema changes require N PRs across N repos with manual coordination.
- Tier 4 attestation surface fragments; each repo has its own SLSA chain.
- Branch protection / release coordination becomes per-repo.

### Alternative 2: Monorepo, one language for everything (Go)

**Rejected**:

- Twin-runtime in Go gives up real performance (Rust is 2–4× faster for the syscall shim under load).
- Distiller in Go gives up LLM SDK quality (Python's `anthropic` / `openai` SDKs are months ahead of Go equivalents).
- Web console in Go is not viable (no React).

### Alternative 3: Monorepo, polyglot, but with each component as its own deployable artifact + own CI

**Rejected**:

- Pretends to be a monorepo but operates as multi-repo. Worst of both worlds.

### Alternative 4: Microservices in Kubernetes from day one

We *deploy* as microservices in Kubernetes, but the **source** is monorepo. **Accepted** — this is the actual decision.

## Build system: Nix vs Bazel

- **Nix flakes default** because:
  - Hermetic reproducibility for Tier 4 — bit-identical artifacts mandatory.
  - Better polyglot story than Bazel for our mix (Python + Rust + Go + TS).
  - Air-gap installer cleanly built from Nix.
  - Senior-engineer ICP familiar with Nix.

- **Bazel alternative** for customers whose internal build systems already standardize on Bazel.

## Repository governance

- **CODEOWNERS** per top-level directory; component owner approval required.
- **Conventional Commits** enforced via commitlint pre-merge hook.
- **Semantic versioning is calendar-version-driven for releases, semver-internal for SDKs and protocol schemas.**
- **Branch protection:** main is protected; PRs require Tier 0–4 verification + 1 CODEOWNER + 1 reviewer.

## What lives outside the monorepo

- **Customer-facing OSS releases** (verifier harness, tape-scrub pipeline, cartographer) — built from the monorepo, published to dedicated public OSS repos.
- **Public marketing website + docs site** — separate repo (designers don't need to clone our entire codebase).
- **Customer-supplied integrations / examples** — customer repos.

## References

- [02-engineering/repo-structure.md](../02-engineering/repo-structure.md)
- [ADR-013](ADR-013-nix-for-tier4-builds.md) — Nix specifically
