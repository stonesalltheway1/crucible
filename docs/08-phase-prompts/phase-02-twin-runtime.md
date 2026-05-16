You are starting Phase 2 of building Crucible, picking up where Phase 1 left off.

Phase 1 delivered the Agent Control Plane skeleton: protobuf type system,
multi-vendor model router, plan builder, bounded budget enforcer, signed
attestation pipeline, and a minimal CLI. Twin Runtime, Verifier, Memory, and
Promotion Contract were all stubbed with "STUB:" markers.

Phase 2 builds the TWIN RUNTIME — the core trust mechanism of the entire product.
This is the largest and highest-stakes single block in v1. Block 2 was scoped at
4 agent-days (~70K LoC) in the original plan; we compress to ONE focused session
by prioritizing the architectural-trust pieces and deferring the breadth (multi-
engine DB, raw Firecracker, full PII scrubber) to Phase 3.

CALIBRATION
===========
Agent-day throughput, not engineer-time. Phase 2 targets ~20–25K LoC because the
syscall-shim work is high-stakes and we want it right, not maximal. Do not soften
to "human team" framings. The build plan in docs/07-roadmap/build-plan-agent-days.md
is the source of truth.

READ FIRST (in this exact order, before writing code)
=====================================================
1. docs/PHASE-1-REPORT.md                                — what's there, what's stubbed
2. memory/project_crucible_phase1.md                     — Phase 1 handoff context (from C:\Users\Eric\.claude\projects\E--AI-Coding-Agent\memory\)
3. docs/01-architecture/twin-runtime.md                  — the full Twin Runtime spec
4. docs/01-architecture/threat-model.md                  — what the runtime defends against
5. docs/05-decisions/ADR-001-digital-twin-first.md       — why the runtime exists
6. docs/05-decisions/ADR-005-neon-db-branching.md        — Neon driver decisions
7. docs/05-decisions/ADR-007-hoverfly-tape-replay.md     — tape replay decisions
8. docs/05-decisions/ADR-014-infisical-over-vault.md     — secrets layer
9. docs/05-decisions/ADR-015-firecracker-via-e2b.md      — sandbox layer
10. docs/06-research/tape-coverage-strategy.md           — full decision tree for tape hit/miss
11. docs/03-sdk/agent-sdk-reference.md                   — every twin.* method you must implement
12. docs/03-sdk/attestation-formats.md                   — runtime emits 8+ attestation types
13. docs/01-architecture/promotion-contract.md           — what's downstream of the runtime
14. docs/04-operations/runbooks.md (RB-03, RB-04, RB-09) — sandbox-escape, egress, spawn failure scenarios
15. docs/07-roadmap/build-plan-agent-days.md (Block 2)   — your block

If anything conflicts with Phase 1's actual implementation, the docs win UNLESS
Phase 1 flagged a deliberate divergence in PHASE-1-REPORT.md. In that case, the
report wins and you carry the divergence forward consistently.

RESEARCH BEFORE CODING (parallel subagents, ~10 min each)
=========================================================
Docs were May 2026; verify currency for the load-bearing infra. Use WebFetch:

1. E2B SDK — current Go/TS API for sandbox create/exec/snapshot/restore/kill;
   Firecracker version under the hood; pricing model unchanged; any new safety primitives.

2. Neon API — POST /projects/{id}/branches request/response shape; current free
   tier branch quota; cold-start latency benchmarks; any branch-creation gotchas.

3. Hoverfly OSS — current major version; capture+simulate mode flags; tape format
   schema; gRPC support state.

4. Infisical OSS — current dynamic-secrets SDK; sidecar pattern docs; sub-minute
   TTL support; how to scope a token to a single database.

5. Cilium + Tetragon — TracingPolicy syntax for TCP-egress allowlist with
   SIGKILL-on-violation; whether eBPF setup works inside an E2B sandbox or
   requires the orchestrator layer.

6. Firecracker syscall-filter (seccomp-bpf) — current best-practice profiles;
   whether ptrace works inside Firecracker without privilege escalation.

7. Postgres branching alternatives — Xata pivot status (OSS as of Apr 2026 per docs);
   pg_dump/restore latency benchmarks for the self-hosted fallback.

8. mitmproxy current API for the dev-tier egress proxy (we use mitmproxy for
   solo-founder tier per ADR-015; Cilium/Tetragon for production).

If anything has changed materially, FLAG IT in the report — do not silently
swap providers without surfacing the decision.

PHASE 2 SCOPE
=============
The Twin Runtime, with the architectural-trust pieces complete and the breadth
pieces staged for Phase 3.

EXPLICITLY IN SCOPE
-------------------
1. apps/twin-runtime/ — Rust service (per tech-stack.md):
   - sandbox/         E2B driver implementing the SandboxProvider interface from twin-spec
   - filesystem/      git worktree + overlayfs orchestration inside the sandbox
   - lifecycle/       spawn, snapshot at checkpoint boundaries, kill, GC for orphans
   - syscall_shim/    THE critical-path piece — see below
   - egress/          allowlist proxy + manifest validation
   - api/             gRPC service exposing the twin.* surface to the control plane

   The Rust choice is per ADR-012; Firecracker integration + syscall perf justifies it.

2. apps/twin-runtime/syscall_shim/ — THE NON-NEGOTIABLE PIECE.
   Multi-layer destructive-op gate, all three layers required:
   a. Command-line parser layer:
      - Lexical analysis of incoming twin.shell.exec commands
      - Pattern matching against destructive operations (full list in
        docs/01-architecture/twin-runtime.md §1.5)
      - Convert matches to DestructiveProposal before any exec
   b. ptrace syscall-filter layer:
      - Runtime interception of unlink/unlinkat, rm-equivalent syscalls
      - Caught syscalls converted to DestructiveProposal events
      - Process suspended pending approval; resumed or killed
   c. eBPF post-exec layer (Tetragon):
      - Defense-in-depth if both above are bypassed
      - SIGKILL-on-violation behavior
      - Logged as security event regardless of approval state

   Twin-scoped destructives auto-approve via the gate; real-scoped require
   downstream Promotion Contract approval (stubbed in Phase 2; wired in Phase 6).

   This must handle the PocketOS scenario from docs/01-architecture/threat-model.md
   §"The PocketOS scenario" — the canonical adversarial test case.

3. libs/sandbox-spec/ — protobuf additions to twin-spec for sandbox primitives:
   - SandboxSpec, Sandbox, SnapshotRef
   - SandboxProvider interface (concrete: E2B; future: raw Firecracker, Daytona)
   - SandboxKillReason enum (clean | TTL | escape-attempt | budget | manual)

4. services/twin-runtime/db_driver/ — Go (separated for vendor SDK pragmatism):
   - Neon REST driver: POST /projects/{id}/branches → connection string
   - Twin-base branch maintenance (daily snapshot of prod, scrubbed)
   - Schema-diff utility (compare twin branch to base)
   - MySQL/SQLite/Mongo support — STUB ONLY (typed errors that say "Phase 3")

5. services/twin-runtime/tape_driver/ — Go:
   - Hoverfly wrapper (subprocess management)
   - Tape mount + content-addressed storage by (service, endpoint, request_hash)
   - Decision tree per docs/06-research/tape-coverage-strategy.md
   - X-Crucible-Tape response header on every replay
   - PII scrubber: REGEX-ONLY in Phase 2 (Presidio + spaCy + FF3-1 are Phase 3).
     The interface must be the full scrub-pipeline shape so Phase 3 can swap
     implementations without changing call sites.

6. services/twin-runtime/secrets_sidecar/ — Go:
   - Infisical client + sidecar daemon pattern
   - Dynamic-secret issuance with sub-minute TTL
   - Egress-proxy integration: $secret(name)$ placeholder substitution at request time
   - Raw secret values NEVER returned to twin.secret.get caller
   - Real prod credentials physically unreachable from the agent process —
     verify via integration test that attempts to read them fail

7. services/twin-runtime/egress_proxy/ — Go:
   - Per-task manifest allowlist
   - Production tier: Cilium+Tetragon TracingPolicy with SIGKILL-on-violation
   - Dev/solo-founder tier: mitmproxy allowlist (simpler, runs userspace)
   - Routes egress to Hoverfly tapes first; falls through to live-allow per manifest

8. SDK implementation — flesh out Phase 1's generated stubs:
   - libs/sdk-go/twin/ — full twin.* implementation calling into the runtime via gRPC
   - libs/sdk-ts/twin/ — same
   - libs/sdk-py/twin/ — same
   - libs/sdk-rs/twin/ — same
   - All four pass the SDK contract property tests from libs/twin-spec/test/

9. Attestation emission for every action:
   - WriteAttestation on twin.fs.write
   - MigrationAttestation on twin.db.migrate
   - ServiceCallAttestation on twin.svc.call (with X-Crucible-Tape disposition)
   - DestructiveProposal on shim interception
   - DestructiveApproval on gate decision
   - All signed via libs/attestation (built in Phase 1)
   - All published to local Rekor journal; Sigstore public Rekor flag-gated for dev

10. Integration with Phase 1's control plane:
    - control-plane.task_router now spawns a real twin via the runtime instead of stubbing
    - control-plane.api routes twin.* SDK calls through the runtime
    - End-to-end: submit task → plan → spawn twin → agent SDK calls work → kill twin

11. Tests:
    - Unit tests per usual standards (mutation ≥85%, PBT + EBT)
    - Property tests for syscall_shim: NO destructive pattern can bypass all
      three layers. Generate adversarial command inputs via proptest;
      assert gate fires.
    - Integration test: full PocketOS scenario (agent finds a "Railway token"
      in a file, attempts `railway down`) — must be intercepted, classified,
      blocked. This is the canonical test of the entire architecture.
    - Integration test: spawn twin, twin.fs.write a file, twin.db.query
      against a Neon branch, twin.svc.call against a recorded tape, kill
      twin, verify attestation chain published.
    - Chaos test: kill the sandbox mid-task; verify clean failure + state recovery
      (per docs/runbooks.md RB-09).

12. Docs updates:
    - docs/02-engineering/local-dev.md — how to run the twin runtime locally
      (E2B API key required; document the env vars)
    - CHANGELOG.md → 2026.06.0-phase2 entry
    - Update top-level README.md status

EXPLICITLY OUT OF SCOPE (defer to Phase 3+)
-------------------------------------------
- Raw Firecracker self-hosted orchestrator (E2B only in Phase 2)
- MySQL / SQLite / MongoDB DB twins (typed stubs only)
- Presidio + spaCy + FF3-1 full PII scrub pipeline (regex-only in Phase 2)
- WASM tool runner for inner-layer tool isolation
- LLM-generated stubs for unknown endpoints (the schema+llm decision-tree branch)
- Service trace recording in shadow mode (only replay from pre-existing tapes)
- ZFS-based filesystem isolation for self-host
- Multi-region sandbox orchestration
- GPU-capable twins

Stubs must be honest: typed returns matching the real interface, "STUB:" log
prefix, no silent fakes.

THREAT-MODEL HARD INVARIANTS
============================
These are non-negotiable. If your implementation cannot guarantee all of them,
flag and stop — do not ship broken safety:

1. Agent process cannot syscall to real production credentials. Period.
2. Egress to non-allowlisted hosts is dropped at the kernel (production) or
   userspace (dev). No exceptions.
3. Destructive operations from the agent's shell ALWAYS route through the gate.
   No path bypasses all three enforcement layers.
4. Every action emits an attestation. No "fast path" that skips attestation.
5. Cross-tenant access is impossible by namespace design. Per-tenant isolation
   at sandbox + Neon branch + Infisical scope + memory layer.

The threat model in docs/01-architecture/threat-model.md enumerates the named
adversarial scenarios. Your integration tests should cover at minimum:
- T11 (PocketOS): see scope item 11 above
- T19 (destructive-op gate bypass): the syscall_shim property test
- T20 (egress policy violation): direct test with attempted egress to known
  non-allowlisted host; verify SIGKILL fires

WORKING AGREEMENTS
==================
- Rust for the twin-runtime executable (perf + Firecracker integration); Go for
  the supporting services per ADR-012.
- Build via the existing Nix flake from Phase 1; add Rust toolchain to dev shells.
- Tests use hypothesis/fast-check/proptest/rapid (we eat our own dogfood per
  ADR-002 reasoning applied internally).
- No comments unless explaining WHY. Per docs/02-engineering/testing-strategy.md.
- Conventional Commits enforced. Pre-merge commitlint hook stays on.
- Latest popular dependencies. Currency-check research is the gate; if a doc
  pick has been superseded, FLAG IT, don't silently swap.

QUALITY BAR
===========
- Mutation score ≥ 85% on diff for all new code.
- The syscall_shim package gets ≥ 95% — it's the brand promise.
- Property tests for the gate: 50,000+ iterations of adversarial command
  generation. Zero bypasses.
- Hermetic Nix build of the entire Phase 2 surface. Two consecutive `nix build`
  produce bit-identical hashes.
- Lints clean: clippy (Rust), golangci-lint (Go), biome (TS).
- The full integration test (submit task → spawn twin → SDK calls → kill twin
  → verify attestation chain) runs end-to-end in CI.
- E2E latency budget: twin spawn ≤ 300ms p95 against E2B (per ADR-015).

PROGRESS TRACKING
=================
Decompose with TaskCreate. Suggested initial breakdown:
  1. Read docs + PHASE-1-REPORT
  2. Currency-check research (parallel subagents)
  3. libs/sandbox-spec protobuf additions
  4. Rust scaffolding: apps/twin-runtime + Nix flake update for Rust
  5. sandbox/ — E2B driver
  6. filesystem/ — git worktree + overlayfs
  7. lifecycle/ — spawn/snapshot/kill/GC
  8. syscall_shim/ — the three-layer gate (longest single subtask)
  9. egress/ — proxy + manifest validation
  10. services/twin-runtime/db_driver — Neon driver
  11. services/twin-runtime/tape_driver — Hoverfly basic
  12. services/twin-runtime/secrets_sidecar — Infisical
  13. SDK implementation (Go, TS, Python, Rust)
  14. Attestation emission wire-up
  15. Control plane integration
  16. Tests (incl. the PocketOS adversarial test)
  17. CI updates
  18. Docs + end-of-session report

Use Glob/Grep/Read for code navigation. Use the Agent tool for parallel research
and bounded subtasks. Consider running the three currency-check research streams
on E2B/Neon/Hoverfly in parallel from the start.

END-OF-SESSION REPORT
=====================
Write docs/PHASE-2-REPORT.md with:

1. What shipped — file tree, total LoC added
2. What works end-to-end — exact commands, including the PocketOS-scenario test
3. What's stubbed — every STUB: marker for Phase 3
4. Threat-model invariants — verify all 5 hard invariants from the section above
   are enforced; if any aren't, that's a SHIP BLOCKER
5. Ambiguities found in design docs + how you resolved them
6. Library version surprises
7. Mutation scores per package (syscall_shim must be ≥ 95%)
8. Nix hermetic-rebuild status
9. Twin spawn latency benchmark (target ≤ 300ms p95)
10. The Phase 3 prompt — self-contained handoff for the next session covering:
    full PII scrub pipeline (Presidio + spaCy + FF3-1), multi-engine DB support
    (MySQL/SQLite/Mongo), raw Firecracker self-host, WASM tool runner.
    (The phase-03 prompt template is already at docs/08-phase-prompts/
    phase-03-twin-runtime-breadth.md — your job is to validate it against
    your actual delivery and amend if needed.)

Update memory at C:\Users\Eric\.claude\projects\E--AI-Coding-Agent\memory\
with project_crucible_phase2.md.

GUARDRAILS
==========
- Do NOT bypass the syscall shim during testing. If your tests need to verify
  destructive operations execute correctly post-approval, route through the
  approval flow. The shim is not a thing you mock; it's the thing you trust.
- Do NOT commit secrets. E2B API keys, Anthropic keys, Neon API tokens all live
  in .env.local (gitignored). Document required env vars in local-dev.md.
- Do NOT silently swap library picks. Flag and ask.
- Do NOT skip pre-commit hooks (--no-verify) or bypass signing.
- Do NOT commit destructive operations (git push --force, etc.) without
  explicit confirmation.
- Do NOT exceed E2B sandbox budget in your own building work — your subagents'
  twin spawns count against the shared dev account. Track via cost-meter.
- If you hit ambiguity in the syscall shim threat model, STOP and ask. The
  shim is the brand promise; the cost of getting it wrong dwarfs the cost of
  one back-and-forth.
- If the PocketOS-scenario integration test fails to intercept, that's a SHIP
  BLOCKER. Do not mark Phase 2 complete with that gap.

The Twin Runtime is the architectural pillar of the entire product. Cursor
copied Tab autocomplete and Devin copied autonomous loops, but no incumbent has
built this layer. Build it like you mean it.

Begin.
