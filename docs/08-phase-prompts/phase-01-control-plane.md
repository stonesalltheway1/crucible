You are starting Phase 1 of building Crucible, an AI coding agent positioned
on "trust and verifiability" against Cursor/Windsurf/Devin/Antigravity. The
full design is already written and lives at E:\AI Coding Agent\docs\. Read
the design before you build — do not re-derive what is already decided.

CALIBRATION (IMPORTANT)
=======================
This is an AI-agent build. Plan in agent-days, not engineer-months. A focused
session = 10–20K LoC of working code. Phase 1 targets the LOW end of that
range (~15K LoC) because the foundation work is high-stakes and we want it
clean rather than maximal. The remaining ~300K LoC of v1 ships across the
next ~18 agent-days per docs/07-roadmap/build-plan-agent-days.md.

Do not soften scope to "human team" framings. Do not invent caveats about
"this is a multi-month effort." If you find yourself writing that phrase,
stop and look at the build plan.

READ FIRST (in this exact order, before writing any code)
=========================================================
1. docs/README.md                                — entry point
2. docs/00-vision/product-vision.md              — the thesis
3. docs/01-architecture/system-overview.md       — the diagram
4. docs/02-engineering/repo-structure.md         — what the monorepo looks like
5. docs/02-engineering/tech-stack.md             — every tech pick
6. docs/05-decisions/README.md                   — index of ADRs
7. docs/05-decisions/ADR-001-digital-twin-first.md
8. docs/05-decisions/ADR-002-cross-family-verifier.md
9. docs/05-decisions/ADR-009-anti-loop-protocol.md
10. docs/05-decisions/ADR-012-monorepo-structure.md
11. docs/05-decisions/ADR-013-nix-for-tier4-builds.md
12. docs/03-sdk/agent-sdk-reference.md           — the twin.* API shape
13. docs/03-sdk/attestation-formats.md           — in-toto predicates we emit
14. docs/01-architecture/model-routing.md        — the 5-tier router
15. docs/07-roadmap/v1-mvp.md                    — what's in/out of scope
16. docs/07-roadmap/build-plan-agent-days.md     — Block 1 (Control Plane) is yours

If anything in those docs is ambiguous, flag it in your end-of-session report
— don't paper over it with assumptions.

RESEARCH BEFORE CODING (do these in parallel via subagents, ~10 min each)
=========================================================================
Documentation was written May 2026; verify currency of the libraries you'll
actually depend on. Use WebFetch on the official docs sites:

1. Anthropic Messages API — latest SDK version (Go, TS, Python, Rust if exists),
   current pricing/cache TTLs for claude-opus-4-7 / sonnet-4-6 / haiku-4-5,
   any breaking changes since May 2026.

2. Google Gen AI SDK — latest gemini-3.1-pro / gemini-3-flash pricing,
   thinking_config parameter shape, structured output schema, current SDK package
   names (they rename these annoyingly often).

3. OpenAI Responses API — gpt-5.5 / gpt-5.3-codex / gpt-5.1-codex-max
   current pricing + reasoning_effort param.

4. connect-go (gRPC + HTTP same handler) — current major version, breaking changes.

5. Sigstore Cosign + Rekor v2 — current Go/Rust client libraries, keyless OIDC
   flow for non-CI environments.

6. Nix flakes — current Nix version, any 2026 schema changes.

7. OPA / open-policy-agent embedded Rego — current Go module path.

If any vendor has shipped breaking changes that affect the tech-stack picks
in docs/02-engineering/tech-stack.md, FLAG IT in the report — don't silently
adopt alternatives. The user makes that call.

PHASE 1 SCOPE
=============
Build Block 1 from docs/07-roadmap/build-plan-agent-days.md: the Agent Control
Plane. ~3 agent-days of work in the original plan; we compress it into ONE
focused session by building the skeleton + the critical path, and stubbing
non-load-bearing pieces. End state: a runnable control plane that can take a
task description, produce a plan via real LLM calls, enforce budgets, and emit
signed in-toto attestations.

EXPLICITLY IN SCOPE
-------------------
1. Monorepo skeleton at E:\AI Coding Agent\:
   - flake.nix at root (Nix flakes; hermetic dev shells per language)
   - apps/, services/, libs/, verifiers/, infra/, examples/, scripts/, .github/
   - Per docs/02-engineering/repo-structure.md exactly
   - Top-level README.md pointing at docs/
   - LICENSE (Apache-2.0 for OSS components; placeholder commercial elsewhere)
   - .gitignore, .editorconfig, CODEOWNERS stub

2. libs/twin-spec/ — protobuf schemas (source of truth for all types):
   - Plan, PlanApproval, PromotionBundle, PromotionId
   - VerifierApproval, VerifierRejection, TierResults
   - Convention (procedural memory data model)
   - Budget, Routing, Task, TaskStatus
   - All Attestation predicate types from docs/03-sdk/attestation-formats.md
   - DestructiveProposal, SecretRef, ExecResult, etc.
   - buf.yaml + buf.gen.yaml for codegen
   - Generated stubs for Go, TS, Python, Rust under libs/sdk-{go,ts,py,rs}/

3. apps/control-plane/ — Go service, the largest piece of Phase 1:
   - task_router/      classifies tasks into tiers (Haiku 4.5 LLM-driven, cacheable)
   - plan_builder/     Sonnet 4.6 prompt → Plan; emits PlanProposal attestation
   - budget_enforcer/  sidecar pattern (sub-package); hard caps; retry counter
   - model_router/     anthropic/google/openai HTTP clients + 5-tier routing
   - api/              connect-go server: gRPC + REST in one
   - main.go           wires it all together
   - Real Anthropic + Google + OpenAI clients (use their official SDKs, not bare HTTP).
   - Aggressive prompt caching: 1h slot for system prompt + tool defs, 5m slot for active context.
   - All actions emit attestations via libs/attestation/.

4. libs/attestation/ — Go package:
   - in-toto Statement v1 builder
   - DSSE envelope signer (Sigstore keyless OIDC; fall back to local key for dev)
   - Rekor v2 publisher (with local hash-chained journal as fallback)
   - All 13 Crucible predicate types from docs/03-sdk/attestation-formats.md

5. libs/policy/ — Rego policy loader using OPA embedded (preparation for the
   promotion-gate; not wired into the contract yet, but the library exists).

6. CI pipeline at .github/workflows/:
   - lint (per-language)
   - type-check
   - unit tests with mutation testing on diff (mutmut/stryker/cargo-mutants per language)
   - Nix flake check (hermetic build verification)
   - SLSA-L3 attest-build-provenance on main branch merges

7. Tests:
   - Unit tests for every public function in libs/twin-spec, libs/attestation,
     libs/policy, and the control-plane subpackages.
   - One integration test: submit a task description via the REST API → get a
     Plan back → approve plan → enforce budget caps when exceeded.
   - One property test for the budget enforcer (no path lets you exceed the cap).
   - Mutation score threshold ≥ 85% on diff.

8. Bare-minimum CLI at apps/cli/:
   - `crucible task new --description "..."` submits a task to the local control plane
   - `crucible plan show <task_id>` displays the plan
   - `crucible plan approve <task_id>` approves
   - Enough to demonstrate end-to-end.

9. Docs updates:
   - Add docs/02-engineering/local-dev.md — how to run this thing locally
   - Touch CHANGELOG.md with 2026.06.0-phase1 entry
   - Update top-level README.md status from "design-stage" to "phase-1 skeleton"

EXPLICITLY OUT OF SCOPE (do not build these in Phase 1)
-------------------------------------------------------
- Twin Runtime (Block 2 — sandbox driver, Neon, Hoverfly, syscall shim) — STUB ONLY
- Verifier Pipeline (Block 3) — STUB ONLY (return a fake "approved" verdict)
- Memory Layer (Block 4) — STUB ONLY (in-memory map suffices)
- Promotion Contract (Block 5) — STUB ONLY (log "would promote" and return success)
- Web console / IDE plugins / GitHub App / Slack bot — none of these
- Real KMS integration — stub the signing path with a local key
- Real Sigstore Rekor publish — local journal only; flag-gate the real Rekor client
- Per-tenant authentication — single-tenant for Phase 1
- Pricing / billing — out of scope

Stubs must be honest: they return typed responses matching the real interface,
log "STUB:" prefix, and never silently swallow data.

WORKING AGREEMENTS
==================
- Languages per docs/02-engineering/tech-stack.md: Go for control plane,
  Rust only for libs/attestation if Sigstore Go client is broken (otherwise Go),
  TypeScript for libs/sdk-ts and apps/cli (cli could be Go — pick Go).
- Build system: Nix flakes. Bazel is the alternative; we use Nix.
- Tests use the same frameworks the verifier ladder uses (hypothesis, fast-check,
  proptest, rapid) so we eat our own dogfood from day 1.
- No comments unless explaining WHY. Per docs/02-engineering/testing-strategy.md.
- Conventional Commits enforced via commitlint pre-merge.
- Every public function has a unit test. Every test has both example-based AND
  property-based variants where applicable (per ADR-002 verifier philosophy —
  the lesson applies to our own tests, not just customer tests).
- Use the latest, popular, well-maintained dependencies. If a doc says "use X"
  and your research shows X has been superseded, FLAG IT — don't silently swap.

QUALITY BAR FOR PHASE 1
=======================
This is the foundation everything else builds on. Quality > velocity.

- All types in libs/twin-spec must exactly match the SDK reference doc.
  Future blocks build on these; type drift is expensive.
- Attestation predicates must validate against the published JSON-Schemas.
- Mutation score ≥ 85% on diff.
- Nix build is hermetic — two consecutive `nix build` produce bit-identical hashes.
- `golangci-lint` clean, `mypy --strict` clean (if any Python), `cargo clippy
  -D warnings` clean (if any Rust), `biome check` clean (TS).
- README is enough that a fresh agent on a fresh session could pick up Phase 2
  from a clean clone.

PROGRESS TRACKING
=================
Use TaskCreate/TaskUpdate (the harness's task tools) to break Phase 1 into
trackable subtasks. Suggested initial decomposition:
  1. Read all required docs
  2. Run currency-check research (parallel subagents)
  3. Initialize monorepo + Nix flakes
  4. Write libs/twin-spec protobuf schemas
  5. Generate SDK stubs (Go, TS, Python, Rust)
  6. Build libs/attestation
  7. Build libs/policy stub
  8. Build apps/control-plane (task_router, plan_builder, budget_enforcer, model_router, api)
  9. Build apps/cli minimal
  10. Write tests + integration test + property test
  11. CI pipeline
  12. Docs updates + end-of-session report

Use Glob/Grep/Read aggressively. Use the Agent tool for parallel research and
for delegating bounded subtasks (e.g., "research Anthropic SDK latest" runs in
parallel with "research Gemini SDK latest").

END-OF-SESSION REPORT
=====================
At the end of the session, write a single file at docs/PHASE-1-REPORT.md with:

1. What shipped — file tree of new code + total LoC
2. What works end-to-end — exact commands the user can run
3. What was stubbed — list every "STUB:" marker so Phase 2 knows where to fill in
4. Ambiguities found in the design docs and how you resolved them
5. Library version surprises from the currency-check research
6. Mutation score, lint status, Nix-build hermetic-rebuild status
7. The Phase 2 prompt — a self-contained handoff prompt for the next session
   that starts exactly where you left off

Also update memory at C:\Users\Eric\.claude\projects\E--AI-Coding-Agent\memory\
with a project_crucible_phase1.md describing what's done and what's next.

GUARDRAILS
==========
- Do NOT commit destructive operations (git push --force, rm -rf, DROP, etc.)
  without explicit confirmation — eat the dogfood from your own ADR-009.
- Do NOT skip hooks (--no-verify) or bypass signing unless the user asks.
- Do NOT commit secrets — if you generate API keys for testing, put them in
  .env.local (gitignored) and document the requirement.
- Do NOT mock the LLM calls for the integration test — make a real (cheap) call
  via Haiku 4.5 with the smallest possible prompt. If no API key is available,
  fail the integration test with a clear error rather than fake success.
- Do NOT silently swap library picks. Flag and ask.
- Do NOT exceed budget caps in your OWN building work — if your subagent runs
  blow past expected cost, halt and report.

The goal is a Phase 1 that a new agent (or human) can pick up cleanly. Foundations
matter more than feature surface. Quality wins.

Begin.
