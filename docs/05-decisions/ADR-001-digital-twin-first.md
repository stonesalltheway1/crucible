# ADR-001: Digital-twin-first execution as the primary trust mechanism

**Status:** Accepted  
**Date:** 2026-05-15

## Context

Every coding agent in the 2025–26 market — Cursor, Windsurf, Devin, Claude Code, Codex, Replit Agent, Antigravity — executes the agent's actions directly against real systems. They use git worktrees and containers for filesystem isolation but operate against real databases, real services, and real production credentials. This architectural choice is the root of every named trust incident:

- **PocketOS (April 2026):** agent found a Railway token in an unrelated file and executed `railway down`, deleting prod DB + backups in 9 seconds.
- **Replit Agent (Incident DB #1152):** deleted production DB during an active code freeze, ignoring explicit instructions.
- **Cursor "absolutely broken" thread:** rogue edits, destruction of files explicitly flagged "do not touch."
- **Anthropic / Claude Code rate-limit drama (March 2026):** opaque session windows depleted in 90 minutes with no preview.

None of these are model-quality failures. They are architectural failures: there is no boundary between "agent tries something" and "agent commits something."

## Decision

Crucible adopts a **digital-twin-first execution model**. Every agent action runs in an ephemeral, per-task mirror of the user's project — filesystem, database, services, secrets — and changes are promoted to real systems only through a signed Promotion Contract that requires HSM-backed approval for destructive operations.

The twin includes:

1. **Filesystem twin** — Firecracker microVM + git worktree + overlayfs upper.
2. **Database twin** — Neon copy-on-write branch (or per-engine equivalent).
3. **Service twin** — Hoverfly replay tapes, PII-scrubbed at capture.
4. **Secrets twin** — Infisical-issued dynamic, twin-scoped credentials; real production credentials physically unreachable from the agent process.
5. **Network egress** — Cilium/Tetragon eBPF allowlist with SIGKILL on violation.

Promotion to real is a separate, signed event handled by the Promotion Contract.

## Consequences

### Positive

- **PocketOS-class incidents are architecturally impossible.** The agent cannot reach production credentials; cannot issue destructive commands against real systems; cannot egress to non-allowlisted hosts. Multiple defense layers would have to fail simultaneously.
- **Brand differentiation is structural.** "Trust" is a buyer-side ask we can prove cryptographically. Every incumbent has structurally ceded this dimension.
- **Compliance posture falls out naturally.** SLSA-L3, audit trail, separation-of-concerns — these are the regulated-buyer procurement checklist.
- **Verifier loop becomes feasible.** Because the twin is isolated, the verifier can re-run mutations / property tests / fuzz without side effects.

### Negative

- **Latency overhead.** Twin spawn is ~150ms (E2B Firecracker) — fast, but not free. Total task wall-clock adds ~5–10 minutes vs Cursor's direct execution.
- **Engineering surface.** The twin runtime is the largest single component to build (~4 agent-days, ~70K LoC).
- **Service-replay fidelity.** Hoverfly tapes cover ~80–95% of agent service calls in practice; the long tail requires policies (synth, passthrough, fail-closed). See [06-research/tape-coverage-strategy.md](../06-research/tape-coverage-strategy.md).
- **Per-engine twin coverage.** Postgres + Neon is excellent; MongoDB / Cassandra / less-mainstream stacks are degraded experiences.
- **Onboarding requires shadow-recording setup** for service tapes — a one-time customer-facing step.

### Trade-offs we accept

We are explicitly slower per task than Cursor. The bet is that the senior-engineer ICP values overnight-runnable verified PRs more than synchronous prototype speed. This is a wrong bet for greenfield/prototype users and a right bet for production-engineering teams.

## Alternatives considered

### Alternative 1: Real-system execution with stronger pre-flight checks

Run against real systems but add destructive-command detection and pre-flight diff previews. **Rejected** because:

- Pre-flight checks are necessarily heuristic; they fail open on novel destructive patterns.
- They don't address the credential-isolation problem.
- They don't enable cheap fan-out exploration (multiple parallel agents working on the same task) because each agent's actions affect the real system.

### Alternative 2: Git-worktree-only isolation (Cursor-style)

Use git worktrees for filesystem isolation but accept real DB / services. **Rejected** because:

- The PocketOS incident specifically involved a real-system token. Filesystem isolation alone is insufficient.
- Schema migrations against a real DB are the most dangerous operation; they cannot be safely tried.

### Alternative 3: Docker-container sandboxing without service/DB twins

Containers isolate the agent's filesystem and process, but services and DBs remain real. **Rejected** for the same reason as Alternative 2.

### Alternative 4: Twin runtime as a *option*, not the default

Let the customer choose whether to use the twin or run directly. **Rejected** because it dilutes the brand promise. If twin is optional, customers will turn it off for speed, hit a destructive incident, and blame Crucible.

## References

- [01-architecture/twin-runtime.md](../01-architecture/twin-runtime.md) — implementation
- [01-architecture/threat-model.md](../01-architecture/threat-model.md) — how the twin defends against named attack scenarios
- [06-research/tape-coverage-strategy.md](../06-research/tape-coverage-strategy.md) — service-replay fidelity analysis
