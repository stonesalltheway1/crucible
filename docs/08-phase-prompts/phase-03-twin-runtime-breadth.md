You are starting Phase 3 of building Crucible, filling in the breadth of the
Twin Runtime that Phase 2 deferred.

Phase 2 delivered the architectural-trust core: E2B sandbox, three-layer
syscall shim, destructive-op gate, Neon Postgres driver, basic Hoverfly tape
replay (regex-only PII scrub), Infisical secrets sidecar, egress proxy, full
SDK implementation across Go/TS/Python/Rust, and integration with Phase 1's
control plane. The PocketOS scenario is intercepted by construction.

Phase 3 fills in the breadth: full production-grade PII scrub pipeline,
multi-engine database support, raw Firecracker self-host orchestrator, WASM
tool runner, and shadow-recording mode for tape population. These were
deferred from Phase 2 to keep that session focused on safety-critical pieces.

CALIBRATION
===========
Agent-day throughput. Phase 3 targets ~20K LoC. Most of this is integration
work against mature libraries (Presidio, vendor SDKs, Firecracker SDK) rather
than novel architecture — quality bar is high but the work is well-paved.

READ FIRST
==========
1. docs/PHASE-2-REPORT.md                                — what shipped, what's stubbed
2. memory/project_crucible_phase2.md                     — Phase 2 handoff context
3. docs/01-architecture/twin-runtime.md (§4 service twin tapes, §1 sandbox)
4. docs/05-decisions/ADR-007-hoverfly-tape-replay.md     — PII scrub interface
5. docs/05-decisions/ADR-005-neon-db-branching.md (per-engine equivalents table)
6. docs/05-decisions/ADR-015-firecracker-via-e2b.md (raw Firecracker for self-host)
7. docs/06-research/tape-coverage-strategy.md            — full decision tree including
   schema+llm synth and shadow recording
8. docs/04-operations/self-hosted-install.md             — what air-gap install needs
9. docs/04-operations/runbooks.md RB-09                  — twin spawn failure handling
10. docs/07-roadmap/v1-mvp.md                            — confirm Phase 3 stays in v1 scope

RESEARCH BEFORE CODING (parallel)
=================================
1. Microsoft Presidio — current Analyzer + Anonymizer API; spaCy model versions
   (en_core_web_lg etc.); custom recognizer registration; performance benchmarks
   on JSON payload scrubbing.

2. mysto/python-fpe — FF3-1 implementation status; alternative libraries
   (HashiCorp Vault transform mode, Cryptography library); compliance status.

3. Gretel / SDV / MOSTLY AI — current SDKs for synthetic data augmentation;
   pricing and self-host availability.

4. PlanetScale — current MySQL branching API; Postgres branching status (was
   half-built per May 2026 docs); cold-start latency.

5. Turso — current libSQL branching API + pricing.

6. MongoDB Atlas — snapshot-restore-to-new-cluster API + latency benchmarks.

7. Firecracker — current Rust crate (firecracker-rs or rust-vmm), latest
   container-runtime integrations (containerd-firecracker, kata-runtime
   alternatives); ZFS clone latency benchmarks on Linux 6.x.

8. Wasmtime — current Rust crate version + capability model (WASI preview 2);
   tool-execution sandbox patterns for LLM-generated code (NVIDIA agentic-AI
   sandboxing 2025 paper).

9. Microcks / Stoplight Prism — current LLM Copilot Sample feature for synth
   responses from OpenAPI specs.

Flag any vendor breaking changes that affect the ADR picks.

PHASE 3 SCOPE
=============

EXPLICITLY IN SCOPE
-------------------
1. services/twin-runtime/tape_driver/scrubber/ — full PII pipeline:
   - Presidio Analyzer + Anonymizer integration with default recognizers
     (names, SSN, credit cards, phones, addresses, emails, MRNs)
   - spaCy NER pass on free-text fields (response bodies, log lines)
   - FF3-1 format-preserving encryption for structure-bearing fields
     (credit-card BINs, phone formats, account-number checksums)
   - Deterministic pseudonymization keyed per-tape-set (referential integrity:
     cus_abc123 → cus_zzz789 consistently across all entries)
   - Custom recognizer registration API for tenant-specific PII patterns
   - Scrub audit log: every tape entry records which scrubbers fired and which
     fields were rewritten; queryable for compliance auditors
   - Synthetic augmentation hook (Gretel/SDV/MOSTLY AI) for fields the
     scrubber blanks but downstream code needs realistic-looking values

   Critical: scrubbing happens at CAPTURE, before bytes hit disk. Scrubbing
   on replay is too late.

2. services/twin-runtime/tape_driver/synth/ — LLM-generated stubs:
   - OpenAPI/proto schema → synthetic response generation
   - Microcks AI Copilot pattern: cheap-tier LLM (Haiku 4.5) augments schema-
     derived Faker output with realistic field values
   - Output marked X-Crucible-Tape: synth-readonly or synth-mutation header
   - Persisted as CANDIDATE tape entry (not auto-promoted) for human review
   - Deterministic state journal for mutation calls (POST/PUT/PATCH/DELETE):
     responses come from spec's default success example, mutations recorded
     in-memory journal that subsequent GETs consult before falling through

3. services/twin-runtime/db_driver/ — fill in per-engine support:
   - MySQL: PlanetScale branching API integration
   - SQLite/libSQL: Turso branching
   - MongoDB: Atlas snapshot-restore-to-new-cluster (slower; acceptable)
   - Redis/KV: per-task fresh redis-server inside sandbox (already simple)
   - ClickHouse: table-level CLONE AS
   - S3: MinIO inside sandbox + rclone mirror prefix
   - Per-engine adapter exposes the same DBTwin interface; runtime selects
     by tenant config + repo's detected DB engines
   - Schema-diff utility extended per engine

4. apps/twin-runtime-self-host/ — raw Firecracker orchestrator:
   - Rust binary, parallel to the E2B-driven apps/twin-runtime
   - firecracker-containerd + containerd integration
   - ZFS dataset per repo; clone-per-task for sub-50ms isolation
   - Pre-warmed pool (configurable, default 20 sandboxes/node)
   - Per-tenant cgroup quotas
   - Network namespaces with Cilium/Tetragon eBPF policy
   - Snapshot-restore latency target: <10ms warm
   - Implements the SandboxProvider interface so all sandbox-using code is
     orchestrator-agnostic

   This is what powers the self-hosted enterprise tier. SaaS keeps E2B.

5. apps/twin-runtime/wasm_runner/ — WASM tool sandbox:
   - Wasmtime embedding (Rust crate)
   - WASI preview 2 capabilities model: no fs/net unless host grants
   - Used for executing LLM-generated tool code at the inner layer (NVIDIA
     agentic-AI sandboxing 2025 pattern)
   - Distinct from the outer microVM sandbox; this is *inside* the sandbox
     for hostile-code-from-the-model containment

6. services/twin-runtime/tape_driver/shadow_recorder/ — population pipeline:
   - Customer points the recorder at staging or sanctioned production traffic
   - eBPF or Envoy taps HTTP/gRPC egress
   - Records to content-addressed tape files keyed by (service, endpoint,
     request_hash)
   - Runs the full scrub pipeline at capture
   - Surfaces tape-coverage metrics back to the customer dashboard
   - Re-record schedule support (default monthly for high-traffic endpoints)

7. Tape staleness detection:
   - Per-endpoint last_recorded timestamp
   - Tape-age warning surfaced in agent's reasoning ("this tape was last
     refreshed 47 days ago")
   - Verifier (built in Phase 4) will lower confidence on stale-tape responses

8. ZFS-based filesystem isolation for self-host:
   - ZFS dataset per project as the lower layer
   - Per-task clone (`zfs clone`) replacing overlayfs upper for the self-host tier
   - Cleanup on task completion

9. Tests:
   - Adversarial scrub-corpus test: 1000+ synthetic PII patterns across formats;
     scrubber must catch ≥ 99% with custom-recognizer extensions for the gap.
   - Per-engine DB twin spawn + migration + read verification.
   - Raw Firecracker spawn-kill loop benchmark — must match E2B's <300ms p95.
   - WASM tool execution containment test: attempt fs/net escape from inside
     a Wasmtime-sandboxed tool; verify denial.
   - Shadow recording end-to-end against a dev-stack with known-PII traffic;
     verify scrubbed tapes pass HIPAA Safe Harbor 18-identifier audit.
   - Tape staleness detection: load a deliberately-aged tape, verify warning fires.

10. Docs updates:
    - docs/02-engineering/local-dev.md — Phase 3 additions (PII scrub setup,
      raw Firecracker dev mode, shadow recorder usage)
    - CHANGELOG.md → 2026.06.0-phase3
    - Update docs/04-operations/self-hosted-install.md — air-gap installer
      now genuinely has all the pieces it claims

EXPLICITLY OUT OF SCOPE (defer to Phase 4+)
-------------------------------------------
- The verifier ladder itself (Phase 4)
- Memory layer (Phase 5)
- Promotion contract real wiring (Phase 6)
- Multi-region twin orchestration (v2)
- GPU twins (v2)

WORKING AGREEMENTS
==================
- Rust for the new orchestrator (apps/twin-runtime-self-host); Go for service-
  layer code (db_driver per-engine adapters, scrubber Python wrapper); Python
  for the Presidio integration (Presidio is Python-native — wrap, don't port).
- Build via the existing Nix flake; verify Presidio's spaCy models are cached
  in the Nix store for hermetic offline builds (Tier 4 still applies to us).
- Tests: hypothesis for the scrub-corpus property tests; proptest for the
  Rust sandbox orchestrator chaos cases.
- No silent library swaps. Currency-check research is the gate.

QUALITY BAR
===========
- PII scrub: ≥ 99% recall against the test PII corpus; surfaces false-negative
  rate honestly in scrub audit log.
- Per-engine DB twin spawn: ≤ 5s for Mongo (slower acceptable), ≤ 2s for MySQL/
  SQLite, matching Phase 2's Postgres latency.
- Raw Firecracker spawn: ≤ 200ms p95 cold (better than E2B because no API hop),
  ≤ 10ms warm via snapshot.
- WASM sandbox: zero successful escape attempts in 10,000 adversarial test runs.
- Mutation score ≥ 85% on diff; scrub pipeline ≥ 90% (compliance-relevant).
- Hermetic Nix builds across all new components.
- Lints clean.

PROGRESS TRACKING
=================
Suggested decomposition:
  1. Read PHASE-2-REPORT + docs
  2. Currency-check research (parallel — 9 streams; consider 3 subagents × 3 streams each)
  3. Presidio + spaCy + FF3-1 scrub pipeline
  4. Synthetic augmentation hooks (Gretel/SDV plug-points)
  5. Shadow recorder + scrub-at-capture wiring
  6. Per-engine DB drivers (4 engines, fan out)
  7. WASM tool runner
  8. Raw Firecracker orchestrator (largest single subtask)
  9. ZFS filesystem isolation for self-host
  10. Tape staleness detection
  11. Tests across all of the above
  12. Docs + end-of-session report

END-OF-SESSION REPORT
=====================
Write docs/PHASE-3-REPORT.md:

1. What shipped — file tree + LoC
2. PII scrub corpus results — recall %, false-negative cases, audit-log examples
3. Per-engine DB spawn benchmarks (all four engines + Postgres baseline)
4. Raw Firecracker benchmark vs E2B
5. WASM sandbox containment test results
6. Stubs and deferred items
7. Library version surprises
8. Mutation scores, hermetic-rebuild status
9. The Phase 4 prompt (verifier pipeline — template at docs/08-phase-prompts/
   phase-04-verifier-pipeline.md; validate and amend)

Update memory: project_crucible_phase3.md.

GUARDRAILS
==========
- Do NOT roll your own PII scrubber. Use Presidio + spaCy as documented; add
  custom recognizers if needed but don't reinvent the NER.
- Do NOT skip the scrub audit log. Compliance buyers can't procure without it.
- Do NOT commit any real captured traffic to the repo. Test corpora are synthetic.
- Do NOT commit the spaCy model weights to git; they live in the Nix store or
  the hermetic build cache.
- Do NOT skip the WASM containment property test. LLM-generated tool code is
  the inner threat model; this is your defense.
- If raw Firecracker integration hits an unexpected wall (e.g., it doesn't run
  in the dev environment), document the gap clearly — DO NOT silently ship a
  half-working orchestrator and claim self-host works.

This phase makes Crucible's enterprise / regulated / air-gapped story real.
Build it like compliance auditors will look at it, because they will.

Begin.
