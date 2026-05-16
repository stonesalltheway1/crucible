You are starting Phase 4 of building Crucible. The control plane (Phase 1)
dispatches tasks; the twin runtime (Phases 2–3) executes them in isolation
with cryptographic provenance. Now we build the layer that decides what's
verified enough to ship: the VERIFIER PIPELINE.

This is the second-most-important block in the entire product after the twin
runtime. Without it, Crucible's brand promise of "verified completion" is just
a claim. With it, every task ships with a cross-family-verified proof that
the agent's work is correct.

CALIBRATION
===========
Phase 4 targets ~25K LoC. Block 3 in the build plan was 3 agent-days; we
compress to one session by parallelizing the per-language Tier 0 + Tier 1
runners. Each language adapter is largely "drive an existing tool" — the
integration code is the bulk, not novel architecture.

READ FIRST
==========
1. docs/PHASE-3-REPORT.md
2. memory/project_crucible_phase3.md
3. docs/01-architecture/verifier-pipeline.md            — the full spec
4. docs/05-decisions/ADR-002-cross-family-verifier.md   — why different families
5. docs/05-decisions/ADR-008-tier3-annotation-default-off.md — when Tier 3 fires
6. docs/06-research/tier3-trigger-automation.md         — the multi-signal classifier
7. docs/01-architecture/model-routing.md (verifier tier) — cross-family pairings
8. docs/03-sdk/agent-sdk-reference.md (twin.verify.*)   — API contracts
9. docs/03-sdk/attestation-formats.md (TestReport, VerifierApproval/Rejection)
10. docs/07-roadmap/build-plan-agent-days.md (Block 3)

RESEARCH BEFORE CODING (parallel)
=================================
1. Hypothesis (Python) — current major version + state-machine API; schemathesis
   workflows for OpenAPI/GraphQL; atheris fuzzer state.

2. fast-check (JS/TS) — current version; @fast-check/vitest + @fast-check/jest
   adapter compatibility; race-condition detector docs.

3. proptest + cargo-mutants + Kani (Rust) — current versions; propproof
   integration with Kani.

4. rapid (Go) — pgregory.net/rapid current; native testing.F fuzz API.

5. Mutation testing — mutmut 4.x (Python), stryker-js (TS), cargo-mutants
   (Rust), pitest (Java), muter (Swift). Per-language diff-scoped modes.

6. Antithesis SaaS — current pricing for DST as a service; alternative:
   TigerBeetle's VOPR + WarpStream's public DST architecture for in-house.

7. DafnyPro — POPL 2026 paper status; Dafny binary version; Laurel auto-
   assertion tool status.

8. Sigstore + SLSA — actions/attest-build-provenance current version; Witness
   for non-GitHub CI; reproducible-build comparison tooling.

9. Cross-vendor LLM routing for verifier — confirm Anthropic→Google or
   Anthropic→OpenAI cache transfer is not a thing (it isn't; you pay full input
   on cross-family).

10. LLM-judge frameworks for verifier rubric scoring — DeepEval, ragas,
    promptfoo current state.

PHASE 4 SCOPE
=============

EXPLICITLY IN SCOPE
-------------------
1. apps/verifier/ — Go daemon orchestrating the per-language verifier processes:
   - dispatcher/      receives VerificationRequest from control plane; selects
                      tiers based on critical-path classifier; fans out to runners
   - process_pool/    spawns per-language verifier processes in isolated sandboxes
                      (separate E2B sandbox from executor; different model family)
   - rubric/          LLM-judge rubric scoring (cross-family verifier rates the
                      diff + tests on 0..1 scale; threshold 0.85 by default)
   - critical_path/   the multi-signal classifier from docs/06-research/
                      tier3-trigger-automation.md
   - api/             gRPC server exposing twin.verify.* to control plane

2. verifiers/python/ — per-language runner:
   - Tier 0: mutmut runner, diff-scoped, threshold 85%
   - Tier 1: hypothesis property-test runner (10K+ iterations); schemathesis
     for OpenAPI; atheris fuzz harness orchestration
   - Tier 2: schemathesis stateful workflows from OpenAPI/GraphQL specs
   - Tier 3: Dafny dispatcher (other provers stubbed)
   - Tier 4: Python-specific reproducible-build verification (Nix-shipped)
   - Outputs match TestReport schema from twin-spec

3. verifiers/typescript/ — same shape:
   - Tier 0: stryker-js
   - Tier 1: fast-check + jsfuzz
   - Tier 2: schemathesis (also works for TS APIs via OpenAPI)
   - Tier 3: dispatch placeholder (no mainstream TS formal verifier in v1)
   - Tier 4: TS reproducible-build verification

4. verifiers/rust/ — same shape:
   - Tier 0: cargo-mutants
   - Tier 1: proptest + cargo-fuzz + cargo-afl
   - Tier 2: schemathesis adapter for Rust API frameworks
   - Tier 3: Kani for unsafe blocks + FFI (propproof integration)
   - Tier 4: Cargo reproducible-build verification

5. verifiers/go/ — same shape:
   - Tier 0: go-mutesting + native testing.F fuzz (Go mutation tooling is
     weaker; threshold 75%)
   - Tier 1: pgregory.net/rapid + native fuzz
   - Tier 2: schemathesis adapter
   - Tier 3: dispatch placeholder
   - Tier 4: Go reproducible-build verification (deterministic builds with -trimpath)

6. verifiers/java/ — same shape (jqwik + pitest + JQF) — stubbed if no design
   partner needs it in v1 but interface ready

7. verifiers/swift/ — same shape (swift-testing + muter) — stubbed similarly

8. verifiers/tier3-dafny/ — Dafny adapter:
   - DafnyPro integration (POPL 2026 LLM-assisted verification)
   - Laurel auto-assertion generator
   - Per-function proof obligation discharge
   - Wall-clock budget enforcement: 10 min default, halt-and-fallback to Tier 2.5
   - Cached partial proofs (incremental on next PR)
   - Other Tier 3 tools (Lean 4, TLA+, Z3) — STUB ONLY with typed errors

9. verifiers/tier4-honest-ci/ — reproducible build verifier:
   - Independent Nix rebuild on a second runner (or in-process re-derivation)
   - Bit-identical hash comparison
   - in-toto attestation generation via Sigstore keyless OIDC
   - SLSA-L3 provenance bundle
   - Witness for non-GitHub CI
   - Tekton Chains for K8s-native pipelines
   - Output: HonestCIReport attestation

10. apps/verifier/critical_path/ — multi-signal classifier:
    - Path-pattern regex matchers (SECURITY/MONEY/DATA/SAFETY/HOTPATH from
      docs/06-research/tier3-trigger-automation.md)
    - Import-graph centrality via tree-sitter + per-language symbol resolvers
      (pyan, jdeps, go-callvis, ts-morph)
    - CVE-touched-file detection via git log + OSV-DB
    - Comment-marker scanner (DANGER, // HACK, etc.)
    - LLM-judge classifier (Haiku 4.5, content-hash cached)
    - Production-signal integrations: postmortems, SLO data, PR-review intensity
      — stub these initially; wire actual sources via tenant config later
    - Weighted sigmoid score → Cold/Warm/Hot/Molten band assignment
    - `crucible calibrate` subcommand for per-tenant weight fitting (200-file
      stratified-sample labeling + logistic regression)

11. apps/verifier/rubric/ — LLM-judge rubric:
    - Cross-family executor pairing (Opus 4.7 ↔ Gemini 3.1 Pro by default,
      configurable per tenant)
    - Verifier sees only diff + tests + spec changes + OpenAPI delta —
      explicitly NOT the executor's reasoning trace
    - Rubric prompt structured for schema-constrained decoding
    - rubric_score ∈ [0, 1]; threshold 0.85 default; tenant-tunable
    - Rejection reasons structured for the executor to reflect-and-retry

12. Wire verifier into control plane:
    - control-plane.task_router now invokes verifier after agent claims done
    - Bounded Budget Enforcer tracks verifier-side cost separately
    - Approved bundles flow to promotion contract (Phase 6 will real-wire it;
      Phase 4 keeps the stub but emits the right attestations)

13. Tests:
    - Per-language: each runner is exercised against a fixture project with
      both correct and deliberately-buggy diffs; verifier must approve
      correct and reject buggy.
    - Cross-family disagreement test: same diff verified by Opus and Gemini;
      verify disagreement detection on the ~5–10% case.
    - Critical-path classifier: labeled test set per docs/06-research/
      tier3-trigger-automation.md "Examples" section (oauth_callback.py,
      refund_engine.go, MarketingHeroBanner.tsx, retry.ts, payment_simulator).
      All must classify correctly.
    - Tier 3 timeout fallback: deliberately-impossible Dafny proof; verify
      Tier 2.5 fallback fires (PBT + mutation + CODEOWNER review requirement).
    - Tier 4 reproducible-build: deliberately-non-deterministic build (e.g.,
      `date` in output); verify failure.
    - Honest CI: forged attestation rejected.

14. Docs updates:
    - docs/02-engineering/local-dev.md — Phase 4 additions (per-language
      verifier env, Dafny install, Antithesis credentials if using SaaS)
    - CHANGELOG.md → 2026.06.0-phase4
    - Add docs/04-operations/runbooks.md RB-07 (verifier disagreement) and
      RB-10 (Tier 3 timeout rate) to local runbook tooling

PHASE 3 CARRY-OVER WIRING (added when Phase 3 closed; do this in Phase 4)
-------------------------------------------------------------------------
- The verifier rubric MUST consult `twin-runtime-staleness::Tracker::report()`
  per task and weight responses served from `Stale` or `Unrecorded` tapes
  lower. The tracker is at apps/twin-runtime/crates/twin-runtime-staleness/.
- Tape responses carry `X-Crucible-Tape: hit-exact | hit-template |
  synth-readonly | synth-mutation | synth-candidate | live-passthrough |
  miss-blocked`. The rubric MUST use this disposition as a trust signal:
  `synth-*` and `live-passthrough` weight lower; `miss-blocked` should
  cause re-plan rather than verification.
- The Presidio scrubber's per-tape AuditLog is part of the per-task
  attestation chain the Tier 4 verifier consults — confirm at least one
  scrubber fired on each PII-bearing tape entry before promotion.
- The WASM tool runner's `ExecutionReport.usage.trip` field signals any
  quota that the inner-layer sandbox tripped during the task. The
  verifier rubric should treat any non-None trip as a finding.
- Self-host tier: the orchestrator scaffold at apps/twin-runtime-self-host/
  is gated by the `linux-firecracker` Cargo feature. The verifier daemon
  MUST NOT assume the orchestrator is reachable; the bridge already
  returns a typed PhaseStub when the feature is off, which the verifier
  must propagate as "self-host unavailable" rather than fail-open.

EXPLICITLY OUT OF SCOPE (defer to Phase 5+ or v2)
-------------------------------------------------
- Memory-as-verifier compliance check (needs Phase 5 memory layer)
- Multi-verifier ensemble for high-stakes promotions (v2 Phase 9)
- Lean 4 + LeanCopilot Tier 3 adapter (v2 Phase 9)
- TLA+ + Apalache Tier 3 adapter (v2 Phase 9)
- Z3/CVC5 direct dispatch (v2 Phase 9)
- Custom in-house Crucible verifier model (v2 Phase 9)
- Customer-defined verifier extension API (v2 Phase 9)
- Antithesis full SaaS integration if the customer hasn't licensed it
  (in-house DST is the OSS-tier path; ship that; SaaS wiring is a flag)

WORKING AGREEMENTS
==================
- Go for the verifier daemon + dispatcher (orchestration, gRPC).
- Per-language runners written in their native language (Python for Python
  tools, etc.) because driving the per-language tooling from outside the
  language is painful.
- All runner outputs match a single typed TestReport schema.
- Verifier sandbox is a SEPARATE E2B sandbox from the executor — different
  state, different model, different egress policy. ADR-002 invariant.
- Cross-family routing is configured per tenant; default Opus 4.7 ↔ Gemini
  3.1 Pro pairing.

QUALITY BAR
===========
- Mutation score ≥ 85% on diff for verifier-daemon code; ≥ 90% on rubric/
  and critical_path/ (these are the trust pieces).
- Per-language runner correctness: each tier rejects ≥ 95% of deliberately-
  buggy fixtures and accepts ≥ 98% of correct fixtures.
- Cross-family verifier ALWAYS uses a different vendor lineage than executor.
  No path lets them collide.
- Verifier process can NEVER read the executor's reasoning trace. Audit this.
- Tier 4 reproducible-build: independent rebuild must bit-identical-match.
- Hermetic Nix builds for all new components.

PROGRESS TRACKING
=================
Suggested decomposition:
  1. Read docs + PHASE-3-REPORT
  2. Currency-check research (10 parallel streams; consider 3-4 subagents)
  3. apps/verifier/ scaffolding + gRPC API
  4. critical_path/ multi-signal classifier (the multi-signal scorer)
  5. rubric/ LLM-judge with cross-family routing
  6. dispatcher/ + process_pool/ (fan out per-language)
  7. Per-language runners (Python, TS, Rust, Go — parallel via subagents)
  8. Java + Swift runner stubs
  9. Tier 3 Dafny adapter
  10. Tier 4 honest CI verifier
  11. Wire into control plane
  12. Tests (the fixture-project test suite is the heaviest piece)
  13. Docs + end-of-session report

END-OF-SESSION REPORT
=====================
docs/PHASE-4-REPORT.md:

1. What shipped — file tree + LoC per package
2. Per-language tier coverage matrix (which tiers ship for which languages)
3. Critical-path classifier accuracy on the labeled test set
4. Cross-family pairing test results (disagreement rate)
5. Tier 4 honest-CI: any reproducible-build gaps in our own build
6. Verifier-cost benchmark (verification phase as % of total task cost;
   target: ≤ 10%)
7. Stubs and deferred items
8. The Phase 5 prompt (memory layer — template at docs/08-phase-prompts/
   phase-05-memory-layer.md)

Update memory: project_crucible_phase4.md.

GUARDRAILS
==========
- Do NOT let the verifier model see the executor's reasoning. Audit the
  request payload before each verifier call.
- Do NOT let same-family pairings ship. Refuse the pairing at the routing
  layer if executor and verifier share a vendor.
- Do NOT skip mutation testing on the rubric/ package. The verifier verifying
  itself is the meta-level we must get right.
- Do NOT bill verifier-process time against the executor's budget. Separate
  accounting per ADR-009.
- Do NOT silently fail-open on Tier 3 timeout. The fallback is Tier 2.5 with
  EXPLICIT CODEOWNER-review requirement; never just "passed".
- Do NOT mark Phase 4 complete if the critical-path classifier fails any of
  the named example test cases in docs/06-research/tier3-trigger-automation.md.

The verifier is what turns Crucible's "trust" claim from marketing into a
checkable property. Build it like compliance auditors will read it, because
they will.

Begin.
