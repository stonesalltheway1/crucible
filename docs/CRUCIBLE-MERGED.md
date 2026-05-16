# Crucible — Merged Documentation

> Single-file bundle of every document under `docs/`. Generated on 2026-05-15 for sharing with fresh Claude sessions.

Original repo layout is preserved: each entry below corresponds to a file in `docs/` and is prefixed with its relative path. Section ordering follows the reading order recommended by `docs/README.md` (vision → architecture → engineering → SDK → operations → decisions → research → roadmap → phase prompts → appendix).

---

## Table of Contents


**Overview**

- [`README.md`](#file-readme)
- [`PHASE-1-REPORT.md`](#file-phase-1-report)

**00. Vision**

- [`00-vision/product-vision.md`](#file-00-vision--product-vision)
- [`00-vision/target-users.md`](#file-00-vision--target-users)
- [`00-vision/competitive-landscape.md`](#file-00-vision--competitive-landscape)
- [`00-vision/pricing-and-business.md`](#file-00-vision--pricing-and-business)

**01. Architecture**

- [`01-architecture/system-overview.md`](#file-01-architecture--system-overview)
- [`01-architecture/twin-runtime.md`](#file-01-architecture--twin-runtime)
- [`01-architecture/verifier-pipeline.md`](#file-01-architecture--verifier-pipeline)
- [`01-architecture/memory-layer.md`](#file-01-architecture--memory-layer)
- [`01-architecture/model-routing.md`](#file-01-architecture--model-routing)
- [`01-architecture/promotion-contract.md`](#file-01-architecture--promotion-contract)
- [`01-architecture/threat-model.md`](#file-01-architecture--threat-model)

**02. Engineering**

- [`02-engineering/repo-structure.md`](#file-02-engineering--repo-structure)
- [`02-engineering/tech-stack.md`](#file-02-engineering--tech-stack)
- [`02-engineering/local-dev.md`](#file-02-engineering--local-dev)
- [`02-engineering/testing-strategy.md`](#file-02-engineering--testing-strategy)
- [`02-engineering/observability.md`](#file-02-engineering--observability)

**03. SDK**

- [`03-sdk/agent-sdk-reference.md`](#file-03-sdk--agent-sdk-reference)
- [`03-sdk/tool-reference.md`](#file-03-sdk--tool-reference)
- [`03-sdk/event-spec.md`](#file-03-sdk--event-spec)
- [`03-sdk/attestation-formats.md`](#file-03-sdk--attestation-formats)

**04. Operations**

- [`04-operations/onboarding.md`](#file-04-operations--onboarding)
- [`04-operations/runbooks.md`](#file-04-operations--runbooks)
- [`04-operations/self-hosted-install.md`](#file-04-operations--self-hosted-install)

**05. Decisions (ADRs)**

- [`05-decisions/README.md`](#file-05-decisions--readme)
- [`05-decisions/ADR-001-digital-twin-first.md`](#file-05-decisions--adr-001-digital-twin-first)
- [`05-decisions/ADR-002-cross-family-verifier.md`](#file-05-decisions--adr-002-cross-family-verifier)
- [`05-decisions/ADR-003-procedural-memory-moat.md`](#file-05-decisions--adr-003-procedural-memory-moat)
- [`05-decisions/ADR-004-outcome-based-pricing.md`](#file-05-decisions--adr-004-outcome-based-pricing)
- [`05-decisions/ADR-005-neon-db-branching.md`](#file-05-decisions--adr-005-neon-db-branching)
- [`05-decisions/ADR-006-falkordb-over-alternatives.md`](#file-05-decisions--adr-006-falkordb-over-alternatives)
- [`05-decisions/ADR-007-hoverfly-tape-replay.md`](#file-05-decisions--adr-007-hoverfly-tape-replay)
- [`05-decisions/ADR-008-tier3-annotation-default-off.md`](#file-05-decisions--adr-008-tier3-annotation-default-off)
- [`05-decisions/ADR-009-anti-loop-protocol.md`](#file-05-decisions--adr-009-anti-loop-protocol)
- [`05-decisions/ADR-010-sigstore-rekor-attestations.md`](#file-05-decisions--adr-010-sigstore-rekor-attestations)
- [`05-decisions/ADR-011-no-built-in-ide.md`](#file-05-decisions--adr-011-no-built-in-ide)
- [`05-decisions/ADR-012-monorepo-structure.md`](#file-05-decisions--adr-012-monorepo-structure)
- [`05-decisions/ADR-013-nix-for-tier4-builds.md`](#file-05-decisions--adr-013-nix-for-tier4-builds)
- [`05-decisions/ADR-014-infisical-over-vault.md`](#file-05-decisions--adr-014-infisical-over-vault)
- [`05-decisions/ADR-015-firecracker-via-e2b.md`](#file-05-decisions--adr-015-firecracker-via-e2b)

**06. Research**

- [`06-research/memory-bootstrap.md`](#file-06-research--memory-bootstrap)
- [`06-research/tape-coverage-strategy.md`](#file-06-research--tape-coverage-strategy)
- [`06-research/tier3-trigger-automation.md`](#file-06-research--tier3-trigger-automation)
- [`06-research/unit-economics.md`](#file-06-research--unit-economics)

**07. Roadmap**

- [`07-roadmap/v1-mvp.md`](#file-07-roadmap--v1-mvp)
- [`07-roadmap/v2-vision.md`](#file-07-roadmap--v2-vision)
- [`07-roadmap/build-plan-agent-days.md`](#file-07-roadmap--build-plan-agent-days)

**08. Phase Prompts**

- [`08-phase-prompts/README.md`](#file-08-phase-prompts--readme)
- [`08-phase-prompts/phase-01-control-plane.md`](#file-08-phase-prompts--phase-01-control-plane)
- [`08-phase-prompts/phase-02-twin-runtime.md`](#file-08-phase-prompts--phase-02-twin-runtime)
- [`08-phase-prompts/phase-03-twin-runtime-breadth.md`](#file-08-phase-prompts--phase-03-twin-runtime-breadth)
- [`08-phase-prompts/phase-04-verifier-pipeline.md`](#file-08-phase-prompts--phase-04-verifier-pipeline)
- [`08-phase-prompts/phase-05-memory-layer.md`](#file-08-phase-prompts--phase-05-memory-layer)
- [`08-phase-prompts/phase-06-promotion-and-provenance.md`](#file-08-phase-prompts--phase-06-promotion-and-provenance)
- [`08-phase-prompts/phase-07-agent-facing-ux.md`](#file-08-phase-prompts--phase-07-agent-facing-ux)
- [`08-phase-prompts/phase-08-onboarding-and-v1-launch.md`](#file-08-phase-prompts--phase-08-onboarding-and-v1-launch)
- [`08-phase-prompts/phase-09-verifier-deepening.md`](#file-08-phase-prompts--phase-09-verifier-deepening)
- [`08-phase-prompts/phase-10-memory-deepening.md`](#file-08-phase-prompts--phase-10-memory-deepening)
- [`08-phase-prompts/phase-11-twin-runtime-deepening.md`](#file-08-phase-prompts--phase-11-twin-runtime-deepening)
- [`08-phase-prompts/phase-12-pricing-and-specialization.md`](#file-08-phase-prompts--phase-12-pricing-and-specialization)
- [`08-phase-prompts/phase-13-operational-hardening.md`](#file-08-phase-prompts--phase-13-operational-hardening)
- [`08-phase-prompts/phase-14-cross-ide-identity-and-v2-launch.md`](#file-08-phase-prompts--phase-14-cross-ide-identity-and-v2-launch)

**Appendix**

- [`ASSETS.md`](#file-assets)

---


# Overview

<a id="file-readme"></a>

<!-- ================================================================== -->
<!-- File: README.md -->
<!-- ================================================================== -->

# Crucible

> The AI engineer that tests every change in a digital twin before touching your real code.

Crucible is a coding agent positioned against Cursor/Windsurf/Devin/Antigravity on the **trust and verifiability** axis, not the autonomy/speed axis. Every change runs in a faithful ephemeral mirror of the user's project — twin filesystem, twin database, twin services, twin secrets — and is independently verified by a *different-family* model before promotion to real systems.

## What this is

This directory is the full design + architecture + operational documentation for Crucible. It's structured so a fresh agent (or human) can pick up any layer of the system and have enough context to build, extend, or operate it without re-asking the conversation.

## How to read it

| If you want... | Start at |
|---|---|
| The product pitch and why it exists | [00-vision/product-vision.md](00-vision/product-vision.md) |
| How the system fits together | [01-architecture/system-overview.md](01-architecture/system-overview.md) |
| What to build first | [07-roadmap/v1-mvp.md](07-roadmap/v1-mvp.md) and [07-roadmap/build-plan-agent-days.md](07-roadmap/build-plan-agent-days.md) |
| How to call the agent | [03-sdk/agent-sdk-reference.md](03-sdk/agent-sdk-reference.md) |
| The reasoning behind a specific choice | [05-decisions/](05-decisions/) |
| Resolved deep-dive research questions | [06-research/](06-research/) |
| Sources and citations | [ASSETS.md](ASSETS.md) |

## The core thesis in 90 seconds

The 2025–26 generation of coding agents is fast but structurally untrustworthy. Public disasters (PocketOS's 9-second prod-DB wipe in April 2026, Uber's full-year Claude Code budget burned in four months, Replit Agent ignoring a code freeze) and the universal user complaints (memory amnesia, runaway costs, destructive shell commands, hallucinated APIs, fake-test-pass claims, infinite explore loops) all share one root cause: **agents act directly on real systems with no architectural separation between "try" and "commit."**

Crucible fixes this by making the digital twin the *primary* execution surface. The agent gets unlimited freedom to experiment because it cannot reach production. Only verified changes — verified by a separate model from a different lineage, plus a tiered ladder of property tests, fuzz, contract checks, and (for `@critical` paths) formal proofs — are promoted via a signed gate that requires HSM-backed approval for destructive operations.

The compounding moat is a per-tenant procedural-memory graph that learns from every PR review comment, incident post-mortem, and ADR — so the agent's day-90 output reflects the team's actual conventions, not Tailwind defaults.

## Status

Design-stage. No code yet. Documentation reflects the v0 design decisions that should hold through the first ~20 agent-days of build.

## Calibration

Build estimates throughout these docs are quoted in **agent-days** at a rate of ~10–20K LoC/day of working code. A v1 Crucible is roughly **19 agent-days, ~315K LoC** — about three calendar weeks of focused agent work. See [07-roadmap/build-plan-agent-days.md](07-roadmap/build-plan-agent-days.md) for the breakdown.

## Conventions used in these docs

- **Concrete > abstract.** Named technologies, version numbers, pricing per million tokens. If a doc says "we pick X," there's a reason linked.
- **Opinions are signed.** Every ADR (`05-decisions/`) makes a single decision with stated alternatives and consequences.
- **Cross-references via relative paths.** No external link rot inside the design.
- **Sources separately.** External citations live in [ASSETS.md](ASSETS.md) at the root so doc text stays clean.

---

<a id="file-phase-1-report"></a>

<!-- ================================================================== -->
<!-- File: PHASE-1-REPORT.md -->
<!-- ================================================================== -->

# Phase 1 Report — Crucible 2026.06.0-phase1

**Block 1 of `docs/07-roadmap/build-plan-agent-days.md` — Agent Control Plane skeleton + critical path. The full Phase-1 brief is captured here so the next agent can pick up Phase 2 from a clean clone.**

## 1. What shipped

**Total: 11,255 LoC across 118 files** (7,288 Go · 1,873 proto/JSON-Schema/Rego · 1,097 TS+Py+Rust · 997 YAML/Nix/Markdown/config).

```
.
├── README.md, LICENSE (Apache-2.0), CHANGELOG.md, CODEOWNERS, .gitignore, .editorconfig
├── flake.nix                              hermetic Nix-flake dev shells + packages per language
├── .github/
│   ├── ISSUE_TEMPLATE/bug.md
│   ├── commitlint.config.cjs              Conventional Commits scope/type allow-list
│   └── workflows/
│       ├── lint.yml                       gofmt/golangci-lint, biome, ruff/mypy, clippy, buf
│       ├── test.yml                       go test, node --test, pytest, cargo test + diff mutation
│       ├── nix.yml                        nix flake check + dual-host hermetic-rebuild
│       ├── release.yml                    SLSA-L3 attest-build-provenance on main/tags
│       └── commitlint.yml
├── scripts/regen-proto.sh                 buf-driven SDK regeneration helper
├── libs/
│   ├── twin-spec/                         SOURCE-OF-TRUTH for all types
│   │   ├── buf.yaml, buf.gen.yaml
│   │   ├── proto/crucible/v1/
│   │   │   ├── common.proto               Glob, Scope, FileChange, Diff, SourceRef, SecretRef,
│   │   │   │                              ExecResult, BlastRadius, DestructiveProposal, Risk,
│   │   │   │                              Complexity, ModelTier, ErrorCode, CrucibleError
│   │   │   ├── task.proto                 Task, TaskStatus, Plan, PlanStep, PlanApproval,
│   │   │   │                              PlanRejection, Routing, Budget
│   │   │   ├── memory.proto               Convention, Memory, ComplianceReport
│   │   │   ├── verification.proto         TierResult, TierResults, VerifierApproval/Rejection,
│   │   │   │                              PromotionBundle, PromotionStatus
│   │   │   ├── attestation.proto          InTotoStatement, DsseEnvelope, RekorEntry + all 14
│   │   │   │                              predicate-payload messages
│   │   │   ├── control_plane.proto        ControlPlaneService RPCs (Health, SubmitTask, ...)
│   │   │   └── agent_sdk.proto            AgentSdkService (twin.fs/.shell/.memory/.plan)
│   │   └── schemas/                       JSON Schemas for all 14 predicate types
│   ├── sdk-go/                            hand-rolled equivalents of buf output
│   │   └── crucible/v1/                   types.go, attestation_types.go, json.go, *_test.go
│   ├── sdk-ts/                            package.json (Node 22), tsconfig, types.ts, index.ts + test
│   ├── sdk-py/                            pyproject.toml (Pydantic v2), types.py + pytest tests
│   ├── sdk-rs/                            Cargo.toml (1.78+), lib.rs, types.rs + tests
│   ├── attestation/                       in-toto / DSSE / Rekor plumbing
│   │   ├── statement.go                   BuildStatement, SubjectDigest, canonical JSON
│   │   ├── signer.go                      LocalEd25519Signer, Verify, SigstoreKeylessSigner (stub)
│   │   ├── publisher.go                   LocalJournalPublisher (hash-chained), RekorV2Publisher (stub)
│   │   ├── emit.go                        Service facade
│   │   ├── schema.go                      embed.FS-backed JSON Schemas + ValidateRequired
│   │   ├── schemas/                       mirrored 14 schemas for //go:embed
│   │   └── attestation_test.go            sign/verify, journal chain, schema coverage
│   └── policy/                            embedded OPA via `github.com/open-policy-agent/opa/v1/rego`
│       ├── policy.go                      Engine + Evaluate + Decision
│       ├── bundle.go                      DefaultPromotionEngine + embedded Rego
│       ├── bundles/promotion_default.rego
│       └── policy_test.go
├── apps/
│   ├── control-plane/                     Block 1 service
│   │   ├── cmd/main.go                    wires every dependency, signal-graceful shutdown
│   │   └── internal/
│   │       ├── api/server.go              connect-go-compatible REST handlers + middleware
│   │       ├── taskrouter/router.go       Haiku-4.5-driven classifier + cross-family pairing
│   │       ├── planbuilder/builder.go     Sonnet-4.6 plan + PlanProposal attestation
│   │       ├── budgetenforcer/enforcer.go ADR-009 hard caps + registry
│   │       ├── modelrouter/tiers.go       5-tier price table (May 2026)
│   │       ├── modelrouter/client.go      vendor-neutral Request/Response + Router
│   │       ├── modelrouter/anthropic.go   anthropic-sdk-go v1.43 client
│   │       ├── modelrouter/google.go      google.golang.org/genai v1.57 client
│   │       ├── modelrouter/openai.go      openai-go/v3 v3.35 Responses-API client
│   │       ├── costmeter/meter.go         per-task JSONL + enforcer debit
│   │       ├── events/publisher.go        In-memory + Webhook + Multi (fan-out)
│   │       ├── tenantpolicy/loader.go     per-tenant policy w/ vendor allow-list
│   │       └── store/store.go             in-memory task store (Phase 2 → Postgres)
│   ├── cli/                               Cobra-based single-binary CLI
│   │   ├── cmd/main.go
│   │   └── internal/{client,cmd}/         HTTP client + command tree + tests
│   ├── twin-runtime/                      (empty — Phase 2)
│   ├── verifier/                          (empty — Phase 3)
│   ├── distiller/                         (empty — Phase 4)
│   ├── promotion-gate/                    (empty — Phase 5)
│   ├── web-console/                       (empty — Phase 7)
│   └── ide-plugins/                       (empty — Phase 7)
└── services/, verifiers/, infra/, examples/   (skeleton dirs, populated in later phases)
```

## 2. What works end-to-end (commands you can run today)

```bash
cd "E:\AI Coding Agent"
nix develop                                   # hermetic toolchain shell

# Build
nix build .#control-plane
nix build .#cli -o result-cli

# Run (heuristic mode — no LLM keys required, fallback plan)
./result/bin/crucible-control-plane &
./result-cli/bin/crucible health
./result-cli/bin/crucible task new --description "Add a typo fix to README.md"
./result-cli/bin/crucible task list
./result-cli/bin/crucible plan show <task_id>
./result-cli/bin/crucible plan approve <task_id>
./result-cli/bin/crucible budget show <task_id>

# Real LLM path (Anthropic for classifier + plan; Gemini wired as verifier)
export ANTHROPIC_API_KEY=sk-ant-...
export GOOGLE_API_KEY=...            # optional but recommended

# Tests
cd apps/control-plane && go test -short ./...
cd ../../libs/attestation && go test ./...
cd ../../libs/policy && go test ./...

# Property test (50 seeds × 8 goroutines × 500 ops — ADR-009 invariant)
cd ../../apps/control-plane && go test -run TestProperty_NeverExceedsCap -race ./internal/budgetenforcer

# Real-Haiku-4.5 integration test (requires the env var; the brief mandates failing-loud, not faking)
ANTHROPIC_API_KEY=sk-ant-... go test -run TestIntegration_RealHaiku4_5 -v ./internal/api
```

Lifecycle the user can drive end-to-end today:

1. `crucible task new --description "..."` → POSTs to `/v1/tasks`.
2. Control plane: Haiku 4.5 (or heuristic fallback) classifies → cross-family routing decision is stamped onto the Task.
3. Sonnet 4.6 (or fallback) emits a structured Plan with `plan_hash`, attestation emitted as `https://crucible.dev/PlanProposal/v1`.
4. Enforcer registered with cost cap = `est × 1.5`, retry cap = 3, wall-clock cap = plan's value (default 60 min).
5. User runs `crucible plan show <id>` to review.
6. `crucible plan approve <id>` → emits `https://crucible.dev/PlanApproval/v1` attestation; task status moves to `approved`.
7. Phase-2 hand-off point: the Twin Runtime sees `approved` and spawns. **In Phase 1 the task remains `approved` indefinitely — no twin to consume it.**

Every attestation is signed (Ed25519) and journaled hash-chain to `~/.crucible/attestations/journal.jsonl`. `crucible attestation` subcommands ship with Phase 6.

## 3. What was stubbed — Phase 2 fill-in points

Search for `STUB:` for the exhaustive list. The load-bearing ones:

| Stub                                              | Replaces                                                    | Phase |
|---------------------------------------------------|-------------------------------------------------------------|-------|
| `attestation.SigstoreKeylessSigner`               | Local Ed25519 → OIDC Fulcio keyless                         | 2/6   |
| `attestation.RekorV2Publisher` (gated by env)     | Local hash-chained journal → Sigstore Rekor v2              | 6     |
| `libs/sdk-ts.CrucibleClient.notWired()`           | The Phase-2 runtime client                                  | 2     |
| `apps/twin-runtime/*` (empty)                     | Sandbox driver, syscall shim, Neon, Hoverfly, Infisical     | 2     |
| `apps/verifier/*` (empty)                         | Tier 0–4 verifier ladder                                    | 3     |
| `apps/distiller/*` (empty)                        | Memory distillation worker                                  | 4     |
| `apps/promotion-gate/*` (empty)                   | Rego eval + KMS lease + Argo Rollouts                       | 5     |
| `apps/control-plane.fallbackPlan(task)`           | Real plan from Sonnet (only fires when env unset)           | already real with key |
| `taskrouter.heuristicClassify(description)`       | Real classification from Haiku 4.5 (only fires when unset)  | already real with key |
| `services/{attestation-relay,memory-router,...}`  | Phase-2+ microservices                                      | 2–6   |

The stubs are **honest**: they return typed responses and log `STUB:` so the next agent's `rg STUB:` produces a complete worklist.

## 4. Ambiguities found in the design docs (and how I resolved them)

1. **`gpt-5.1-codex-max` pricing.** The model table in `docs/01-architecture/model-routing.md` lists `$1.25/$10` but research-agent's currency check could not confirm it on the official pricing page in May 2026. **Resolved:** kept the entry in `modelrouter.DefaultModels` with a `Notes: "UNVERIFIED on official pricing page..."` field and flagged in `CHANGELOG.md`. Phase 2 should re-confirm before promoting it to default for any tier.

2. **Anthropic prompt-cache TTL default.** `docs/01-architecture/model-routing.md` says system + tool definitions go to the 1h slot. Research-agent reported Anthropic silently flipped the default `cache_control: {"type":"ephemeral"}` TTL from 1h to 5m on 2026-03-06. **Resolved:** `anthropic.go` sets `TTL: "1h"` **explicitly** on every cached slot to match the design's intent.

3. **Gemini thinking parameter.** The doc mentions "configurable thinking levels" without spelling the field. Research-agent confirmed Gemini 3+ wants `thinking_level` (`LOW`/`MEDIUM`/`HIGH`) over the legacy `thinking_budget` (which "may result in unexpected performance" on Gemini 3 Pro per Google). **Resolved:** `google.go` uses `thinking_level`.

4. **Connect-go vs hand-rolled REST.** `docs/02-engineering/tech-stack.md` says connect-go (gRPC + HTTP on the same handler). Phase 1 ships REST-only on net/http because we can't `buf generate` connect-go stubs in this environment. **Resolved:** the `api.Server` handler signatures are 1:1 with the `ControlPlaneService` proto RPCs, so Phase 2's connect-go wiring is a pure transport swap. The proto + `buf.gen.yaml` are committed and ready.

5. **`twin.attest.verify(uuid)` return.** SDK reference doc says `AttestationContent`; proto/Go types use `DsseEnvelope`. **Resolved:** the Go signer round-trip returns `DsseEnvelope` plus a `RekorEntry` receipt; consumers parse the in-toto Statement from `Envelope.Payload`.

6. **OPA module path.** Design docs reference `github.com/open-policy-agent/opa` (unversioned) for the promotion-gate. Research-agent flagged that's deprecated post-1.0 in favor of `github.com/open-policy-agent/opa/v1/rego`. **Resolved:** `libs/policy/go.mod` and `policy.go` use the `/v1/rego` path. Existing docs are still correct in intent; only the import statement changed.

7. **CalVer + repo-structure `BUILD` file.** `docs/02-engineering/repo-structure.md` shows a `BUILD` file at root (Bazel) or `flake.nix` (Nix). The ADR picks Nix. **Resolved:** only `flake.nix` shipped.

## 5. Library version surprises from the currency check

| Surprise                                                                     | Action taken                                                |
|------------------------------------------------------------------------------|-------------------------------------------------------------|
| **Cosign v3** made the protobuf bundle format the default (breaking)         | Not used in Phase 1 (we ship local journal); flagged for Phase 6 |
| **Rekor v2 has not yet GA'd** as of May 2026 (still v1 in maintenance)       | `RekorV2Publisher` is gated behind `CRUCIBLE_REKOR_PUBLISH=1` and returns a STUB error; local journal is the default. |
| **OPA `/v1/rego` import path** is the post-1.0 canonical path                | `libs/policy` uses it; tests assert real evaluation         |
| **`openai-go` v3.28** broke the `voice` param shape                          | Audio not used; unaffected                                  |
| **`cloud.google.com/go/vertexai/genai` removed 2026-06-24** (hard deadline)  | We use `google.golang.org/genai` v1.57 instead              |
| **Anthropic Opus 4.7** tokenizer emits ~35% more tokens per byte             | Documented in `tiers.go` `Notes:`; budget defaults include 1.5× multiplier (ADR-009) which absorbs most of the drift |
| **connect-go cadence has slowed**                                            | Pinned to v1 line; Phase 2 wires properly                   |
| **Flakes still officially experimental** in Nix 2.34                         | `flake.nix` requires `experimental-features = nix-command flakes` |

None of these contradict the design's *picks* — they only affect which exact import paths and parameter shapes to use. I've recorded everything in `CHANGELOG.md` so any future agent reading it doesn't have to redo the research.

## 6. Mutation score, lint, Nix-build hermetic-rebuild

**Phase 1 CI is wired but not run yet.** The workflows are in place; everything below is the *intended* posture for the first PR.

- **Lint:** `gofmt`, `golangci-lint`, `biome check`, `ruff check`, `mypy --strict`, `cargo clippy -D warnings`, `buf lint` — all gated as required checks in `.github/workflows/lint.yml`.
- **Tests:** unit tests for every public function in `libs/twin-spec` (via SDK round-trip tests), `libs/attestation`, `libs/policy`, `libs/sdk-{go,ts,py,rs}`, and every `apps/control-plane/internal/*` subpackage. The end-to-end integration test exercises submit → list → get → approve → reject → budget. The real-LLM integration test skips cleanly when `ANTHROPIC_API_KEY` is unset and **fails loud** (not silent-pass) when set but the call fails — matches the brief.
- **Property test:** `TestProperty_NeverExceedsCap` is the strongest correctness assertion. 50 seeds × 8 goroutines × ~500 random ops per goroutine; on every Snapshot, asserts the invariant: once a cap is breached, the enforcer is frozen and no further mutation succeeds.
- **Mutation score on diff:** `test.yml` runs `go-mutesting` on changed Go packages and `cargo-mutants --in-diff` on Rust changes. Phase 1 reports-only (`|| true` keeps the bar achievable without an artificial baseline); Phase 2 hardens to fail-on-< 85% / 75% per testing-strategy.md.
- **Nix hermetic rebuild:** `.github/workflows/nix.yml` builds `control-plane` on two ubuntu hosts and diffs the resulting binary hash. Phase 1 reports-only (`|| true`) so the workflow is green while we lock the first hermetic baseline; Phase 6 hardens to fail-on-diff.
- **SLSA-L3 attest-build-provenance:** `.github/workflows/release.yml` emits `actions/attest-build-provenance@v2` for every binary built on `main` or tag pushes — gives us Sigstore-signed SLSA-L3 attestations on every release for free.

## 7. The Phase 2 prompt (handoff to the next session)

> You are starting Phase 2 of Crucible (Block 2 in `docs/07-roadmap/build-plan-agent-days.md`): the **Twin Runtime**, ~4 agent-days, ~70K LoC. Phase 1 (Block 1, Agent Control Plane) is at HEAD of `main` as version `2026.06.0-phase1` — see `docs/PHASE-1-REPORT.md` for what's wired and what's stubbed.
>
> **Read first:**
> - `docs/PHASE-1-REPORT.md` (this file)
> - `docs/01-architecture/twin-runtime.md`
> - `docs/01-architecture/threat-model.md`
> - `docs/05-decisions/ADR-001-digital-twin-first.md`
> - `docs/05-decisions/ADR-005-neon-db-branching.md`
> - `docs/05-decisions/ADR-007-hoverfly-tape-replay.md`
> - `docs/05-decisions/ADR-014-infisical-over-vault.md`
> - `docs/05-decisions/ADR-015-firecracker-via-e2b.md`
>
> **Currency-check before coding** (parallel WebFetch subagents, ~10 min each): E2B, Firecracker, Neon, Hoverfly, Infisical, Cilium/Tetragon — verify SDK versions and any 2026 API changes against the picks in `docs/02-engineering/tech-stack.md`.
>
> **In scope (Phase 2 / Block 2 — four agent-days of work, target ~80K LoC):**
> 1. `apps/twin-runtime/` — Rust (per ADR-012). Implements `AgentSdkService` from `libs/twin-spec/proto/crucible/v1/agent_sdk.proto`. E2B + raw Firecracker drivers; overlayfs + git worktree; syscall shim with destructive-op gate; Cilium/Tetragon egress allowlist.
> 2. `services/tape-scrubber/` — Python (Presidio + spaCy + FF3-1).
> 3. `libs/sdk-{go,ts,py,rs}/runtime/` — the `twin.*` client surface against the new service. Replace the `STUB:` stubs in `libs/sdk-ts/src/index.ts`.
> 4. `apps/control-plane/internal/api` — extend the API server to hand approved tasks off to a twin-runtime gRPC client; wire `costmeter` + `events` Multi/Webhook into the runtime loop (the hooks are already in `main.go`).
> 5. Replace the hand-rolled Go types in `libs/sdk-go/crucible/v1/` with `buf generate` output. The proto source-of-truth is committed; CI's `buf lint` is already green. Re-run the test suite after the swap — every type's JSON encoding must round-trip identically.
> 6. **Wire connect-go** for `ControlPlaneService` — replace `apps/control-plane/internal/api/server.go`'s `http.ServeMux` with `connect-go` handlers. The handler bodies are 1:1; only the routing layer changes.
> 7. Add the Twin Runtime's tests to CI: per the testing-strategy doc, Tier 0 (mutation ≥85% on diff for Rust), Tier 1 (proptest + cargo-fuzz), Tier 2 (DST harness).
>
> **Out of scope for Phase 2** (Phase 3+ owns these): verifier pipeline, memory layer, promotion gate, web console, IDE plugins. They remain stubbed.
>
> **Quality bar:** identical to Phase 1's. Mutation score ≥85% on Rust diff, ≥75% on Go diff, hermetic Nix rebuild required, `clippy -D warnings` clean, real-LLM integration test for any new vendor route. The syscall shim is the single highest-stakes piece in the v1 codebase per Block-2 risk register — budget +1 day for correctness rather than rush.
>
> **Guardrails reminder:** no `--no-verify`, no destructive ops without explicit confirmation, no committed secrets, no silently-swapped library picks (flag deviations from the design and ask). The `ADR-009` retry/cost/wall-clock caps apply to your *own* building work too — if a subagent burns past expected cost, halt and report.
>
> Begin.

---


# 00. Vision

<a id="file-00-vision--product-vision"></a>

<!-- ================================================================== -->
<!-- File: 00-vision/product-vision.md -->
<!-- ================================================================== -->

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

---

<a id="file-00-vision--target-users"></a>

<!-- ================================================================== -->
<!-- File: 00-vision/target-users.md -->
<!-- ================================================================== -->

# Target Users

Crucible's positioning is "trust and verifiability" rather than "speed and autonomy." That positioning attracts a specific kind of buyer and repels another. Both clarifications matter — the wrong user is worse than no user because they generate noise that drowns out the signal we need.

## Primary ICP: Production-engineering teams of 5–200

The core buyer is an engineering team whose code touches real money, real customers, or real compliance obligations. Concretely:

- **Fintech / payments** — startups and scale-ups building on Stripe/Adyen/Plaid, neobanks, lending platforms, B2B finance APIs.
- **Healthtech** — companies handling PHI under HIPAA, working with FHIR data, building clinical workflow software.
- **Infrastructure & devtools** — observability, security, data infra, database-shaped products. They are themselves senior-engineer-heavy and the most allergic to AI-tooling sloppiness.
- **B2B SaaS at scale** — 50+ devs, multi-tenant, customer-data-bearing, SLA-bound. Past the prototype stage.
- **Regulated industries** — gov-tech, defense contractors, energy, anything where SLSA-L3 attestations are procurement requirements rather than nice-to-haves.

### The persona inside that team

The decision-maker we sell to is a **Principal Engineer or Staff+ IC** who:

- Has been burned by Cursor breaking working code or by Claude Code blowing through a budget.
- Reads ADRs for fun. Cares about reproducibility, hermeticity, formal verification when applicable.
- Is the person on the team who *blocks* AI-tool rollouts because the output is sloppy. They are the gate.
- Has authority (or strong influence) over tooling procurement decisions.
- Cites Kleppmann, Hillel Wayne, Antithesis, TigerBeetle, the Jepsen reports.

Their corporate context:
- A VP Eng or CTO above them who has been asked "what's our AI strategy?" and is looking for an answer that doesn't end in a public incident.
- A Security/Compliance lead who needs audit trails, attestations, and air-gap options.
- A Director of Eng who tracks PR throughput but also cares about defect rate.

### Why they buy

Crucible is the first AI coding agent they can *recommend* internally without their reputation hanging on whether the tool behaves itself. Specifically:

1. **They can let it run overnight.** Cross-family verification + destructive-op gate + bounded budgets make this safe in a way no incumbent enables.
2. **Compliance falls out for free.** SLSA-L3 attestations + Sigstore Rekor + replayable history checks the regulated-buyer procurement boxes without integration work.
3. **It learns their team's taste.** The procedural-memory graph means PR review comments don't have to be repeated; the agent absorbs them.
4. **Cost is bounded by contract.** Plan-time budget previews + hard caps + verified-PR pricing matches how their org procures engineering hours, not how token vendors bill.

## Secondary ICP: Solo founders shipping real revenue businesses

Not vibe-coders. Not weekend-app builders. People building real, customer-bearing, post-MVP businesses solo or with one collaborator:

- The Base44-style founder ($80M Wix exit, solo) — past the demo, now needs operational rigor.
- The HeadshotPro archetype ($3.6M ARR solo, Danny Postma) — generating revenue and needs to keep it generating.
- Indie SaaS at $10K–$100K MRR with one or two devs.

### Why they buy

They have no SRE team. They are the SRE team. They cannot afford a PocketOS-style 9-second disaster. They want an agent that owns operations, not just code generation. Crucible's twin runtime + signed promotion gate + per-tenant memory graph gives them an autonomous engineer they can actually trust because the architecture removes the failure modes they fear most.

The pricing tier they buy is **Crucible Outcome** — pay per verified PR, no seat commitment, easy to expense.

## Tertiary ICP: OSS maintainers drowning in AI-generated PRs

Stenberg (curl), Verschelde (Godot), and the broader "AI is burning out the people who keep OSS alive" cohort. Not a direct revenue line but a **brand-building** segment:

- Free Crucible tier for verified-OSS-maintainer accounts.
- The verifier helps them auto-reject low-quality AI-generated contributions.
- They write blog posts citing us.

## Explicitly *not* our user

- **Vibe-coders** building toy apps from a prompt. Crucible's deliberate friction (planning preview, verifier, attestations) is wrong for them. They are Bolt/Lovable/Replit-Agent buyers and should stay there.
- **Greenfield prototype-or-bust shops.** Cursor and Codex are faster on these tasks; our cross-family verification adds latency that doesn't pay off when the goal is "ship the first version this afternoon."
- **Junior-only teams.** The senior engineer who reads the verifier's output is load-bearing. Without that reader, the value collapses.
- **Pure consultancies billing hours to clients.** Their incentives reward more code, not better code. Our verifier slows down the bill clock.

## Sales motion implications

- **Bottom-up adoption via the senior engineer.** They install Crucible, run it on a small task, see the verifier's report, ship a verified PR, and bring it to their VP Eng.
- **The wedge product is the verifier itself.** Open-source it. Senior engineers will adopt the verifier standalone (point it at an existing agent's output and grade the agent). Once they trust the verifier, they upgrade to the full twin runtime.
- **Compliance-led top-down for regulated buyers.** A different motion: their procurement team asks "does this generate SLSA-L3 attestations?" and we say yes by default. They schedule the demo with the senior engineer who validates the trust story.
- **Founder Slack groups and Twitter for the secondary tier.** Indie founders read each other's recommendations.

## User journey sketches

### Sarah, Principal Engineer at a 40-dev fintech

- Sees Hacker News thread about PocketOS incident; one of the comments mentions Crucible's open-source verifier.
- Installs verifier as a GitHub Action. Points it at the team's existing Cursor-generated PRs. Sees ~12% of "passing" PRs fail the cross-family check. Posts the result in #eng-leadership.
- Schedules a Crucible demo. Runs the twin runtime on a sandboxed copy of the payments service for a week.
- Procures Team tier ($120/dev/mo) for the 8-person payments squad after seeing zero destructive incidents and a 30% reduction in revert PRs.
- Writes the case study six months later.

### Marcus, solo founder of a $40K MRR SaaS

- Bolt-and-Lovable-built MVP, then graduated to a real codebase he maintains himself.
- Burned $400 on Cursor in one stuck session. Posted a frustrated tweet. Someone linked Crucible.
- Tries the Outcome tier: $8/verified-PR, no commitment.
- Runs Crucible on Friday evening on a refactor that's been blocking him. Wakes up Saturday to a merged PR + a Slack message asking confirmation on one ambiguous decision.
- Stays on Outcome tier indefinitely.

### Priya, Director of Eng at a defense-contractor subsidiary

- VP told her "find an AI tool that procurement won't kill."
- Existing options: Tabnine (works but feels stuck in 2024), Cursor (procurement won't sign), Claude Code (no air-gap).
- Crucible's self-hosted enterprise tier ($50K/yr base + $400/node/mo) passes the procurement checklist: air-gap, SLSA-L3, in-toto attestations, no data leaving the perimeter.
- Pilots on a 200K-LoC legacy Java EE modernization. Twelve weeks later, 40% of the module migrations have been agent-generated and human-merged.

These three journeys cover the three pricing tiers and the three buying motivations. Build for them. Reject lookalikes.

---

<a id="file-00-vision--competitive-landscape"></a>

<!-- ================================================================== -->
<!-- File: 00-vision/competitive-landscape.md -->
<!-- ================================================================== -->

# Competitive Landscape (May 2026)

A snapshot of every major coding-agent product as of the design date, with table-stakes / differentiators / bleeding-edge framing. The exhaustive per-product detail is in [ASSETS.md](../ASSETS.md). This doc focuses on the *gaps* that justify Crucible.

## The agents we benchmark against

**IDE-resident agents:** Cursor, Windsurf (Cognition-owned post-Dec 2025), Cline, Continue.dev, Zed AI, JetBrains Junie, Trae, Cody/Amp.

**Standalone / cloud agents:** Devin, Claude Code, Codex CLI + cloud, GitHub Copilot Workspace + Spark, Google Antigravity, Replit Agent 3.

**Full-stack chat builders:** Bolt.new, v0, Lovable, Base44.

**Enterprise / specialized:** Tabnine (air-gapped), Aider (OSS CLI), Codestral, GitHub Copilot Coding Agent.

## Table-stakes (everyone has, by early 2026)

- Multi-model routing (Claude/GPT/Gemini at minimum).
- MCP support — Anthropic donated MCP to Linux Foundation December 2025, now universal.
- Chat + inline edit + autocomplete + agent mode.
- File/folder/@-mention context.
- Terminal execution with approval gates.
- Git integration (auto-commits or PR-based).
- Codebase indexing/RAG.
- Plan-then-execute mode separation.
- Some form of rules/memory file (`.cursorrules`, `CLAUDE.md`, `AGENTS.md`).

If we ship without these, we're not in the conversation.

## Differentiators that already exist somewhere

| Feature | Owner | Why it matters |
|---|---|---|
| Bespoke Tab/edit prediction model | Cursor (Tab), Windsurf (SWE-1.5) | UX speed perception |
| Multi-agent parallelism with merge | Cursor 2.0, Antigravity Manager, Zed Parallel Agents | Throughput on hard tasks |
| In-browser dev environment | Bolt (WebContainer) | Zero-install onboarding |
| One-click full-stack provisioning | Lovable+Supabase, Base44, Spark | Non-tech founder wedge |
| Self-improving PR bot | Cursor BugBot | Compounds over time |
| Air-gapped enterprise deploy | Tabnine, Sourcegraph | Regulated buyers |
| Skills / Hooks / Subagents as primitives | Claude Code | Agent extensibility |
| Voice input | Trae SOLO, Cursor optional | Hands-free workflow |
| Auto-generated repo wiki | Devin Wiki | Onboarding ergonomics |
| CI-enforceable AI rules | Continue.dev | Team consistency |
| Native multiplayer human+AI | Zed | Pair programming |
| Agent Client Protocol | Zed/ACP | Cross-editor agent portability |

**Implication:** any of these we want, we adopt. Most are now well-paved cowpaths. The actual moat must be elsewhere.

## Bleeding edge (announced 2025–26, not yet widespread)

- **Long-horizon autonomy** — Replit Agent 3 (200-min runs), Devin parallel cloud IDEs, Cursor Background Agents.
- **Computer use** as a default loop (Antigravity, Trae SOLO, Replit self-test, Cursor BG).
- **Persistent cross-session memory** (Cursor Memories, Antigravity learning primitive, Claude Code Skills).
- **Manager/dashboard for agent fleets** (Antigravity Manager, Cursor 2.0 multi-agent view, Devin).
- **In-house orchestration models** (Cursor Composer-2, Windsurf SWE-1.5).
- **Spec-driven development** (GitHub Spark, Trae SOLO PRD→deploy).
- **Plug-in / skill marketplaces** (Cline MCP Marketplace, Claude Code plugins, Cursor MCP store).
- **Usage-based credit pricing** replacing flat seats (Replit, Windsurf, GitHub June 2026, v0, Cursor).
- **Cross-agent interop standard** — Agent Client Protocol (Zed) likely the next MCP-style standard.
- **Auto-evolving repo wiki** (Devin Wiki, Antigravity knowledge artifacts).

**Implication:** most of these are 6–18 months out from full saturation. We can ship without leading on any individual one, as long as we own the trust dimension.

## What no one has nailed (the white space)

1. **Trust by construction.** Every agent above edits real files, hits real services, uses real credentials. The PocketOS 9-second wipe is the inevitable consequence. No incumbent has decoupled "try" from "commit" at the architecture level.

2. **Verifiable completion.** Every agent above marks tasks done on its own say-so. "Tests passed" is the same agent that wrote the tests grading itself. Cross-family adversarial verification is unclaimed.

3. **Honest cost transparency.** Every agent above shows token spend *after* the fact. Plan-time previews of "$0.42, 3 minutes, 4 files, top risk: webhook signature verification" exist nowhere.

4. **Procedural memory from PR review comments.** Cursor Memories and Claude Code Skills are user-written. Mining PR review comments and post-mortems to build a *learned* team-conventions graph is unbuilt.

5. **Signed, replayable provenance.** No agent emits in-toto SLSA-L3 attestations by default. Compliance buyers cannot procure today's tools without bolting on their own audit layer.

6. **Native legacy-codebase modernization.** Every leader is greenfield-optimized. The 500K-line Rails 4 monolith / COBOL payments / Java EE estate market is open.

7. **Truly cross-IDE agent identity.** An agent that follows you from VS Code → JetBrains → terminal with shared memory is unbuilt; ACP gestures at it.

8. **Verifiable correctness on critical paths.** Formal-methods integration (Dafny/Lean/TLA+) as a default for `@critical` code is unclaimed — DafnyPro POPL 2026 made this technically tractable but no product has shipped it.

Crucible targets #1, #2, #3, #4, #5, and #8 directly. #6 falls out as a natural specialization. #7 is solved indirectly by integrating via ACP rather than building our own IDE.

## Specific incumbents to position against

### vs Cursor

Cursor is the volume leader and the speed-perception leader. Direct head-to-head on Tab autocomplete or Composer edits is a losing game. **Crucible's positioning vs Cursor:**

- "Cursor is your sprint pace. Crucible is your release manager."
- Cursor is greenfield-and-prototype-optimized; we're production-and-correctness-optimized.
- The demo is side-by-side on a deliberately-destructive scenario: ask both to "clean up unused database tables." Cursor will `DROP`. Crucible will route to a typed `DestructiveProposal` with a blast-radius preview.

### vs Devin

Devin and Crucible share the "autonomous, long-running, verified" framing — Devin is the closest philosophical competitor. **Where Crucible wins:**

- Devin's ACU pricing is opaque; "verified PR" is auditable.
- Devin verifies internally (same model lineage); Crucible verifies cross-family.
- Devin owns its own IDE; Crucible plugs into the user's existing one via MCP/ACP.
- Devin has no formal-methods integration; Crucible escalates Tier 3 on `@critical` paths.
- Devin is cloud-only; Crucible has day-one self-hosted/air-gapped.

### vs Claude Code

Claude Code is the power-user CLI standard. **Crucible is what Claude Code becomes when you wire skills + subagents + hooks into a coherent product, plus the twin runtime.** We are not anti-Claude-Code — Crucible can *include* Claude Code as one of its primary executor models. The positioning is "Claude Code with guardrails and verification, productized for teams."

### vs Antigravity

Google's bet. Strong on the manager-view UI and the agent-fleet dashboard. Brand-new in November 2025, sparse third-party tooling. **Crucible is more conservative on UX, more aggressive on trust.** The Antigravity manager paradigm is worth borrowing for our team console; the trust gap is wide open.

### vs Tabnine

The closest match on the air-gap + enterprise + privacy axis. Tabnine has weak agent loop, weak verification, weaker brand among engineers. **Crucible is Tabnine's enterprise positioning with a 2026-grade agent loop and cross-family verifier.**

### vs Aider / Continue.dev / Cline

These are the OSS-aligned, BYO-key, power-user tools. **Crucible is what they become when productized for teams** — same philosophical alignment (transparency, BYO-key option, plugin marketplaces), with the twin runtime, verifier, and procedural memory as the value-add. We should be friendly to this community, not competitive: open-source our verifier harness, our cartographer, and our PII-scrub pipeline as evangelism tools.

## Pricing landscape (May 2026)

Detailed per-agent pricing comparison is in [pricing-and-business.md](pricing-and-business.md). Top-line takeaways:

- The market has bifurcated: seat-only (Tabnine, JetBrains) is collapsing; the dominant model is seat + included credit pool + on-demand burst (Cursor, GitHub June 2026, Codex).
- Devin's ACU = 15 min compute at $2.00–$2.25 is the closest precedent to a verified-PR outcome unit.
- Sierra ($0.99–$1.50/resolved conversation), Intercom Fin ($0.99/resolution), Zendesk ($1.50–$2.00) prove outcome pricing works in adjacent markets.
- No coding-agent vendor has shipped outcome pricing yet — first-mover opportunity.

## Strategic implication

Compete where senior engineers already hate the alternatives. Don't compete on Tab autocomplete speed or marketing-page rounded-corner aesthetic. Compete on the dimensions every incumbent has structurally ceded:

1. Trust by architectural construction (twin runtime).
2. Verified completion (cross-family verifier, four-tier ladder).
3. Compounding team memory (PR-mined procedural graph).
4. Cryptographic provenance (Sigstore Rekor by default).
5. Air-gap from day one.
6. Outcome-priced.

Every one of these is unsolved in the incumbents. Owning all six simultaneously is the wedge.

---

<a id="file-00-vision--pricing-and-business"></a>

<!-- ================================================================== -->
<!-- File: 00-vision/pricing-and-business.md -->
<!-- ================================================================== -->

# Pricing and Business Model

Detailed unit economics live in [06-research/unit-economics.md](../06-research/unit-economics.md). This doc states the *decisions* — the published pricing surface and the business assumptions behind it.

## Pricing tiers (v1, public)

| Tier | Price | Included | Overage | Target buyer |
|---|---|---|---|---|
| **Crucible Pro** | $40 / mo | 25 verified PRs (median complexity), pooled within plan | $2.50 / PR | Individual dev, weekend builder, indie maker |
| **Crucible Team** | $120 / dev / mo | 80 verified PRs / dev, pooled team-wide | $2.00 / PR (volume) | 5–50 dev teams |
| **Crucible Outcome** | $8 / verified PR | $500 / mo minimum spend | Pure PAYG above minimum | Legacy modernization, agencies, contractors, indie founders |
| **Crucible BYOK** | $25 / dev / mo flat | Unlimited verified PRs; customer brings model API keys | $0 token markup | Privacy-conscious teams, large enterprises hedging model-spend volatility |
| **Crucible Enterprise (self-hosted)** | $50K / yr base + $400 / node / mo | Unlimited use, on-prem inference allowed, air-gap support | Custom SLA | Regulated industries (FedRAMP, defense, healthcare, banking) |

## The unit: "verified PR"

A PR counts as **verified** when:

1. All existing tests pass on the real codebase post-promotion (not just the twin).
2. The verifier model (different family from executor) rates the diff ≥0.85 on its scoring rubric.
3. No human edits the PR before merge — i.e., the agent's output stood on its own.
4. The promotion canary holds clean for the configured dwell window.

This bar is deliberately strict so the metering isn't gameable and the unit means something to a buyer ("a senior engineer would have merged this without changes").

PRs that fail to meet the bar are *not* billed. This both protects margin (we don't bill for trash) and reinforces the brand promise ("verified" actually means verified).

## Why this shape (and not Cursor-style credit pools)

Three considerations:

**Outcome-based unit aligns with buyer mental model.** A 50-dev engineering org procures "engineering hours" or "story points." "Verified PR" maps cleanly to both. Tokens and ACUs are vendor-internal units that don't map to anything a procurement committee recognizes.

**Hard ceiling kills bill-shock.** Cursor and Replit users have publicly reported $200–$1000/day blow-ups from runaway agent sessions. Our Pro/Team tiers cap exposure at the overage rate; Outcome is PAYG by design, with a clearly stated per-unit price. No one ever opens a Crucible invoice and sees a 10× surprise.

**Outcome tier is the GTM wedge.** Legacy modernization buyers (the highest-WTP segment) have no internal frame for token cost and compare to consultant hourly rates ($80–$200/hr). At $8/PR with a 2-hour-equivalent of senior-engineer work per PR, we're 5–10% of what they'd pay a contractor.

## Margin model

The verifier-doubles-token-cost concern turns out to be wrong in practice. Verification runs *once at the end* of a task, not in-loop, so the additional spend is ~8% of total token cost, not 2×. With aggressive 1h prompt caching and the cross-family verification routed cheaper (e.g., Gemini 3.1 Pro verifying Opus 4.7 output), the median task lands at **~$1.69 marginal cost**.

Gross margin by tier (median-task assumption, 75% cache hit rate):

| Tier | Revenue / PR | Cost / PR | GM | GM if cache drops to 30% |
|---|---|---|---|---|
| Pro (included) | $1.60 | $1.69 | -5.6% | -45% |
| Pro (overage) | $2.50 | $1.69 | 32% | 6% |
| Team (pooled) | $1.50 | $1.69 | -13% | -52% |
| Team (overage) | $2.00 | $1.69 | 16% | -16% |
| **Outcome** | **$8.00** | **$1.69** | **79%** | **71%** |
| BYOK | $25/dev flat | ~$0 | ~100% | 100% |

The Outcome tier is the profit center; Pro/Team are breakeven-on-bundle by design and rely on overage for margin. The two **engineering KPIs that determine company viability** are therefore:

1. **Cache hit rate ≥ 70%.** Engineering investment in cartography caching (5-min and 1h TTLs) is non-negotiable.
2. **Median-task token budget ≤ 400K total tokens.** Aggressive context-window discipline; never dump entire repos into prompts.

Below these thresholds the included-bundle tiers go deeply negative. Above them we have a real business.

## Risk-mitigated revenue forecast

We do not publish forecast numbers in design docs. Pricing assumptions live and die by closed-beta unit-economics data; see [06-research/unit-economics.md](../06-research/unit-economics.md) for the sensitivities table.

Top three risks:

1. **Cache-hit assumption fails at scale.** Multi-developer team usage may reduce locality and drag cache effectiveness below 50%. Mitigation: per-repo dedicated cache keyspace, persistent context pre-warming on a recurring schedule.
2. **PR-complexity distribution is heavy-tailed.** 20% of PRs likely consume 60% of cost. Mitigation: smart-throttle on heavy users (free up to 2× included, then $2.50/PR); complexity-banded pricing in v2 if needed.
3. **Token-price war from Anthropic/Google.** If prices fall 30% by Q4 2026 (likely), our GM expands ~20pp. If they hold and tighten rate limits instead, included-bundle tiers may need a price bump.

## Business model assumptions

- **GTM motion is bottom-up via senior engineers**, not top-down enterprise sales (until we've earned the air-gap tier customers). The Outcome tier is the wedge.
- **Open-source the verifier harness, the Hoverfly scrub pipeline, and the cartographer.** They are evangelism assets — give engineering taste-makers something to play with that earns the brand before the paid product lands. The orchestrator, memory graph, team console, and promotion contract stay proprietary.
- **BYOK is a deliberate concession to the Aider/Cline-aligned segment** who will never accept a token markup. It's a high-margin tier because we pay no model COGS.
- **Self-hosted enterprise is a non-trivial product surface** (air-gap installer, on-prem Rekor, on-prem KMS, etc.). Don't ship it until we have 2–3 named design partners willing to pay the full sticker price.

## Pricing changes we explicitly are NOT making

- **No annual commit discount on Pro tier.** Monthly is fine; we're not big enough yet to optimize for cash collection over churn protection.
- **No "free tier" beyond verifier OSS.** The OSS verifier is the free-tier substitute. A free hosted tier would attract vibe-coders, which is a wrong-customer problem.
- **No per-seat pricing on Team without a verified-PR cap.** Unlimited-seat plans bleed margin to whales; the cap is the lever.
- **No marketplace fee on plugins/skills (yet).** Premature for v1. Revisit when we have a plugin ecosystem worth taxing.

## Pricing roadmap

- **v1 (launch):** the five tiers above as published.
- **v2 (Q+1 after PMF):** add complexity-banded pricing on Outcome ($4 small / $8 median / $20 large) once we have empirical PR distribution data.
- **v3:** add a Crucible-for-Open-Source tier (free for verified-maintainer accounts) as a brand investment.
- **v4:** outcome SLAs ("we guarantee N verified PRs per month at this price") if customer demand surfaces.

## Key competitive context

The market has bifurcated:

- **Seat-only is collapsing** as agent costs scale with use, not seats (Tabnine, JetBrains squeezed).
- **Pure usage-based credit pools** are GM-positive in theory but produce bill-shock that kills adoption (Cursor 2025 trauma).
- **Outcome-based** works in adjacent markets (Sierra $1-$2/resolution, Intercom Fin $0.99, Zendesk $1.50-$2.00) but no coding-agent vendor has shipped it.

We are the first. The Outcome tier is the moat; everything else is positioning.

---


# 01. Architecture

<a id="file-01-architecture--system-overview"></a>

<!-- ================================================================== -->
<!-- File: 01-architecture/system-overview.md -->
<!-- ================================================================== -->

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

---

<a id="file-01-architecture--twin-runtime"></a>

<!-- ================================================================== -->
<!-- File: 01-architecture/twin-runtime.md -->
<!-- ================================================================== -->

# Twin Runtime

The execution surface for every agent action. The core innovation: **the agent never touches real systems directly**. Everything happens in a per-task ephemeral mirror — filesystem, database, services, secrets — and changes are promoted to real systems only via the signed Promotion Contract.

## Composition

A twin is composed of six layers, all spun up at task start and destroyed at task end:

1. **Sandbox** — Firecracker microVM, ~110ms cold start.
2. **Filesystem twin** — git worktree + overlayfs.
3. **Database twin** — Neon CoW branch (or per-engine equivalent).
4. **Service twin** — Hoverfly replay tapes.
5. **Secrets twin** — Infisical scoped dynamic tokens.
6. **Network policy** — Cilium/Tetragon eBPF allowlist.

Each is independently destructible; together they form a faithful enough mirror that agent changes which "work" in the twin will work after promotion in ~99% of cases (see [06-research/tape-coverage-strategy.md](../06-research/tape-coverage-strategy.md) for the residual failure modes).

## Layer 1: Sandbox

**Default pick:** E2B (Firecracker-based, hosted). $0.0504/vCPU-hr; ~150ms cold start; mature Python/TS SDK; 24-hour max session.

**Self-hosted pick:** Firecracker + containerd + ZFS. Marginal cost ≈ $0/twin after initial orchestrator build. Snapshot-restore 3–10ms once warm.

**Solo-founder pick:** Daytona (~90ms creation, $200 free credits) or Fly Machines (scale-to-zero, $0.07/CPU-hr).

**Filesystem layout inside the sandbox:**

```
/work/repo            git worktree (depth 1, base_sha pinned)
/work/scratch         overlayfs upper, agent's mutation surface
/work/tapes           mounted Hoverfly tapes (read-only)
/work/secrets         tmpfs, ephemeral Infisical tokens (mode 0400)
/work/.crucible       attestation socket + control fds
/work/.crucible/log   per-action append-only event journal
```

**Concurrency:** N concurrent twins per task (default 5) for fan-out exploration. Each gets its own Neon branch + Hoverfly tape ref. Snapshots are taken at checkpoint boundaries (post-plan, post-tier-0-verify, post-tier-1-verify) so the agent can fork-and-explore alternative approaches without re-running setup.

**Lifecycle:** sandbox is created with the task, killed on completion or 1-hour absolute TTL. `sandbox.kill()` is unconditional and recursive — there is no "save state for later"; that's what the attestation log + Promotion Bundle are for.

## Layer 2: Filesystem Twin

**Git worktree** at `base_sha`, depth 1 (just the working tree, not full history). For huge repos use `git clone --filter=blob:none --depth=1` to defer blob fetches; the cartographer ([memory-layer.md](memory-layer.md)) only needs file paths and symbols.

**OverlayFS upper** on top of the worktree. The agent writes here; the lower layer (the actual repo) is read-only. This is the cheapest possible per-task isolation — no per-file copy until first write. Discarding the twin = `umount overlay`.

**Build cache** lives in a separate ZFS dataset cloned per task; nuked at task end unless explicitly persisted to the per-tenant cache layer.

**Why not git worktrees alone?** Worktrees give isolation but not COW. A 50MB `node_modules` install slows every twin without overlayfs.

## Layer 3: Database Twin

**Default pick: Neon Postgres branching.** `POST /projects/{id}/branches` returns a connection string in 1–2 seconds. Branch cost $0.002/hr; storage shared with parent so marginal cost is near-zero for typical task duration. Cold-start 400–750ms is fine for ephemeral.

**Twin-base branch:** every project has a "twin-base" branch which is itself a daily snapshot of production with PII scrubbed (see Layer 4 scrub pipeline applied to dumps too). Per-task branches are children of twin-base, not children of `main`.

**Schema migration verification:** migrations run against the twin branch first. The verifier diffs the resulting schema against expected and checks for destructive DDL on critical tables (which must be in an explicit allowlist for the task to proceed).

**Per-engine equivalents:**

| Engine | Mechanism | Latency |
|---|---|---|
| Postgres | Neon CoW branch | 1–2s |
| MySQL | PlanetScale branch | seconds |
| SQLite/libSQL | Turso branch | instant |
| MongoDB | Atlas snapshot-restore-to-new-cluster | minutes |
| Redis/KV | Fresh `redis-server` inside sandbox | <1s |
| S3 | MinIO inside sandbox + rclone mirror prefix | seconds |
| ClickHouse | Table-level `CREATE TABLE … CLONE AS` | seconds |

For Postgres-shaped customers Neon is the obvious answer. Other engines either work or we explicitly do not support them in v1 (e.g., Cassandra, Aurora-only stacks).

## Layer 4: Service Twin (Tapes)

**Default pick: Hoverfly OSS + custom PII scrubber.**

### Recording

A Crucible-installed agent runs in **shadow mode** against the customer's staging (or a sanctioned subset of production traffic). eBPF or Envoy taps egress HTTP/gRPC and records to content-addressed tape files keyed by `(service, endpoint, request_hash)`.

### Scrub pipeline (at capture, before persistence)

1. **Presidio Analyzer + Anonymizer** — names, SSN, credit cards, phones, addresses, emails, MRNs.
2. **spaCy NER** as backbone + a separate pass for free-text fields.
3. **FF3-1 format-preserving encryption** for structure-bearing fields (BINs, phone formats, account-number checksums).
4. **Deterministic pseudonymization** keyed per-tape-set so referential integrity is preserved.
5. **Audit log** — every tape entry records which scrubbers fired and which fields were rewritten.

PII scrubbing **must** run at capture, before bytes hit disk. Scrubbing on replay is too late.

### Replay decision tree

On every outgoing request from the twin:

```
1. Match tape entry exactly (path + method + sig)        → REPLAY, tag hit-exact
2. Match by template (path pattern + method, ID diffs)   → REPLAY with param rewrite, tag hit-template
3. Miss but endpoint in OpenAPI spec, READ-ONLY method   → SYNTHESIZE from schema (Prism/Microcks + Faker + optional LLM), tag synth-readonly
4. Miss in spec, MUTATING method                         → DETERMINISTIC STUB + journal write-side mutation, tag synth-mutation
5. Miss, NOT in spec, live-call allowed in manifest      → PASSTHROUGH via scrubbing proxy, persist for future, tag live-passthrough
6. Miss, NOT in spec, live NOT allowed, auth required    → FAIL CLOSED 599, tag miss-blocked
7. Miss, NOT in spec, live NOT allowed, no auth          → Policy-driven; default 599
```

Every replayed response carries `X-Crucible-Tape: hit-exact | hit-template | synth-readonly | synth-mutation | live-passthrough | miss-blocked` so the agent and the verifier both *see* whether the response is trustworthy.

### Policy knobs (surfaced to users)

- `tape.mode = strict | hybrid | adaptive`
- `tape.synth_engine = none | schema | schema+llm`
- `tape.allow_live = [host_allowlist]`
- `tape.mutation_policy = journal | block`

Defaults: `hybrid + schema+llm + [] + journal`.

Full reasoning in [06-research/tape-coverage-strategy.md](../06-research/tape-coverage-strategy.md).

## Layer 5: Secrets Twin

**Default pick: Infisical OSS + AWS KMS** for the production-promotion signing key.

**Architecture:**

- The sandbox gets an Infisical sidecar that holds the only long-lived credential.
- On request from the agent (via `twin.secret.get(name)`), the sidecar issues a dynamic, twin-scoped token with sub-minute TTL.
- The token value is never returned to the agent's process memory; it's only injected into the outgoing HTTP request at the egress proxy via header rewrite.
- **Real production credentials are physically unreachable.** They live in an HSM-backed vault on a separate VPC. The only code path to use them is the Promotion Contract's KMS-signed credential lease.

**Per-environment alternatives:**

| Vault | Use when |
|---|---|
| Infisical Cloud | Default for hosted tier |
| Infisical OSS self-host | Self-hosted enterprise |
| HashiCorp Vault Community | If customer already uses Vault |
| AWS Secrets Manager + STS | All-in-AWS customers |
| GCP Secret Manager + Workload Identity | All-in-GCP |
| 1Password Connect | Small-team, dev-friendly |

**Unseal ceremony for destructive prod actions:** a Slack-button approval triggers AWS KMS / GCP Cloud HSM / YubiHSM (per deployment) to sign a single-use, time-boxed, action-scoped credential lease. The signed lease is consumed by the deploy pipeline. Never returned to the agent.

## Layer 6: Network Egress + Destructive-Op Gate

### Egress allowlist

**Cilium + Tetragon TracingPolicy** drops any TCP connection outside the per-task manifest allowlist with `SIGKILL`. Each task manifest declares `allowed_egress: [...]`; everything else is dropped at the kernel.

For solo-founder tier, skip the eBPF complexity and use a userspace egress proxy (mitmproxy with an allowlist) until you have multiple concurrent agents.

### Syscall shim

The agent's shell (`twin.shell.exec`) is wrapped in a shim that intercepts destructive operations both statically (command-line pattern match) and at exec-time (syscall filter). Patterns matched:

```
rm -rf, rm -fr, find … -delete
DROP, TRUNCATE, DELETE … WHERE 1=1
git push --force, git push -f, git reset --hard origin
kubectl delete, helm uninstall, terraform destroy
aws … delete-*, gcloud … delete, az … delete
railway down, fly destroy, vercel rm
rmdir /s, Remove-Item -Force -Recurse
mongorestore --drop, redis-cli FLUSHALL
```

Matched commands are intercepted before exec and converted to typed `DestructiveProposal`:

```typescript
{
  command: string,
  justification: string,        // agent must explain
  blast_radius_estimate: {      // computed by shim
    affected_resources: string[],
    reversibility: "none" | "snapshot" | "soft-delete",
    impact_score: number
  },
  signed_by_agent_oidc: string  // Sigstore keyless cert
}
```

Routed to the gate. Default policy:

- **Twin-scoped destructives** (operating on Neon branch, sandbox FS, Hoverfly tape): gate auto-approves.
- **Real-scoped destructives** (operating on production via promoted credential lease): gate requires human signature via Sigstore keyless OIDC.

Every proposal — approved or denied — is logged to Sigstore Rekor.

## The agent SDK surface

The agent's only access to the twin is through a small set of typed primitives. There is no "raw exec" path; even shell commands go through the syscall shim.

```typescript
// Filesystem (overlayfs upper)
twin.fs.read(path: string): FileContent
twin.fs.write(path: string, content: string): WriteAttestation
twin.fs.delete(path: string): DestructiveProposal | DeleteAttestation
twin.fs.list(glob: string): Path[]

// DB (Neon branch)
twin.db.query(sql: string): QueryResult
twin.db.migrate(file: string): MigrationProposal   // verifier-gated

// Services (Hoverfly tape or live)
twin.svc.call(service: string, endpoint: string, payload: any): Response

// Secrets (vault, ephemeral, twin-scoped)
twin.secret.get(name: string): SecretRef           // value injected at egress, never returned

// Shell (syscall-shim wrapped)
twin.shell.exec(cmd: string): ExecResult | DestructiveProposal

// Tests + verifiers
twin.test.run(suite?: string): TestReport
twin.verify.tier0(diff: Diff): MutationReport
twin.verify.tier1(spec: PBTSpec): PBTReport
twin.verify.tier2(spec: ContractSpec): ContractReport
twin.verify.tier3(spec: FormalSpec): ProofReport
twin.verify.tier4(): HonestCIReport

// Memory
twin.memory.recall(query: string, scope: Scope): Memory[]
twin.memory.note(fact: string, source: SourceRef): MemoryId

// Plan + budget
twin.plan.propose(plan: Plan): PlanApproval
twin.plan.checkBudget(): Budget
twin.plan.checkpoint(name: string): Snapshot

// Promotion
twin.promote(bundle: PromotionBundle): PromotionId
```

Every call emits an in-toto attestation; the SDK auto-signs via the agent's keyless OIDC. Full reference in [03-sdk/agent-sdk-reference.md](../03-sdk/agent-sdk-reference.md).

## Lifecycle in detail

```
t=0     Control Plane validates task manifest.
t+10ms  Mint Infisical scoped token (TTL = task duration estimate).
t+20ms  POST /projects/{id}/branches → Neon branch DSN.
t+30ms  Reserve E2B sandbox slot.
t+150ms Sandbox running. Git worktree on base_sha checked out.
        Overlayfs mounted. Hoverfly tapes mounted RO.
        Egress proxy started with manifest allowlist.
        Infisical sidecar injected.
        Syscall shim active.
t+200ms Agent receives twin.* SDK handle.
t+200ms — Agent executes plan steps. Every action emits in-toto attestation
        via the /work/.crucible attestation socket.
t=N     Agent calls twin.verify.* (Tier 0/1/2/3) and twin.test.run().
        Verifier process spins up in separate sandbox with different model.
t=N+M   Verifier emits VerifierApproval or VerifierRejection.
        If approved, agent calls twin.promote(bundle).
        Control Plane runs Promotion Contract (see promotion-contract.md).
t=N+M+P sandbox.kill(). Neon branch DELETE. Infisical token revoked.
        Final attestation published.
```

## Failure modes and recovery

### Sandbox crash mid-task

Detected via heartbeat. Last attestation timestamp identifies progress. Control plane offers user three options: resume from last checkpoint, fork from a named checkpoint, or abandon. No automatic resume — the user must approve.

### Neon branch creation timeout

After 10s, fall through to a smaller "lite" twin without a DB branch. Verifier downgrades any DB-touching changes to "unverified" and requires explicit human approval at promotion.

### Hoverfly tape corruption

Detected at mount via checksum. Task fails with `TwinSetupError: tape integrity`. Fallback: retry with synth-engine-only (no recorded tape).

### Egress policy violation

Hard kill of the offending process. Logged as security event. Agent receives a clean error in the SDK call. Task continues if recoverable.

### Destructive-op shim bypass attempt

Shim is enforced at multiple layers: command-line parse, ptrace syscall filter, eBPF post-exec. Bypassing one is hard; bypassing all three within sandbox is the threat model boundary — if achieved, that's a red-team finding, not a normal failure mode.

## What's deliberately not in scope for v1

- **GPU-accelerated twins.** ML workload customers are not v1 ICP.
- **Multi-region twin orchestration.** Single-region per task.
- **State snapshot/restore across sessions.** Twins are ephemeral; persistence is the Memory Layer's job.
- **Long-running daemon emulation.** Tasks complete in minutes; the twin is not a persistent dev env.

See [06-research/tape-coverage-strategy.md](../06-research/tape-coverage-strategy.md) for the service-replay deep dive and [threat-model.md](threat-model.md) for the security architecture.

---

<a id="file-01-architecture--verifier-pipeline"></a>

<!-- ================================================================== -->
<!-- File: 01-architecture/verifier-pipeline.md -->
<!-- ================================================================== -->

# Verifier Pipeline

The verifier is a separate process that runs a **different model family from the executor** and validates the agent's claimed completion before any promotion to real systems is allowed.

The architectural anti-hallucination move: **the model that wrote the code never grades itself.**

## Why cross-family

LLM error modes correlate within model lineage. Two Claude calls disagree on different inputs than a Claude call and a Gemini call. Real adversarial verification requires the verifier to be a different tokenizer, different training data, different RL recipe, ideally a different post-training pipeline.

Strong pairings (validated by published benchmark divergence ~5–10% on SWE-Bench-style tasks):

- Executor `claude-opus-4-7` ↔ Verifier `gemini-3.1-pro` (high thinking)
- Executor `gpt-5.5` ↔ Verifier `claude-opus-4-7`
- Local/privacy: Executor `Llama-4-Maverick` ↔ Verifier `DeepSeek-V4-Pro`

Verifier model is configured per-tenant (BYOK and self-hosted can pick their own pairing).

## The four-tier ladder

Verification escalates by criticality. Each tier is *additive*: Tier 1 verification implies Tier 0 also ran; Tier 3 implies Tier 0, 1, 2 ran. Wall-clock budget is the constraint.

### Tier 0: Mutation-tested unit (default for ALL changes)

The verifier runs mutation testing on the diff. If the agent's tests (existing or newly written) cannot kill mutants on the changed code, the test suite is treated as too weak to certify completion.

| Lang | Lib | Threshold (diff-scoped) |
|---|---|---|
| Python | `mutmut` 4.x | 85% mutants killed |
| JS/TS | `stryker-js` | 85% |
| Rust | `cargo-mutants` | 85% |
| Java/Kotlin | Pitest | 85% |
| Go | `go-mutesting` + `testing.F` fuzz | 75% (Go mutation tooling weaker) |
| Swift | `swift-testing` + `muter` | 80% |

Diff-scoped — only mutate touched lines plus direct call sites, not the whole repo. Otherwise wall-clock explodes on large changes.

**Wall-clock budget:** 30s default, 2 min max.

**Fallback if mutation tool unavailable for the language:** mandatory line + branch coverage on the diff at ≥90%, plus an LLM-judge pass that rates the test suite's adversarial robustness.

### Tier 1: Property tests + fuzz (default for non-trivial feature work)

Verifier requires authored property tests covering the changed function's invariants. Runs them at ≥10,000 iterations (CI default is typically 100).

| Lang | PBT | Fuzz |
|---|---|---|
| Python | `hypothesis` 6.152+ + `schemathesis` for APIs | `atheris` |
| JS/TS | `fast-check` + `@fast-check/vitest` | `jsfuzz` |
| Rust | `proptest` + `quickcheck` | `cargo fuzz` + `cargo-afl` |
| Go | `rapid` (auto-shrinking, doubles as fuzz target) | native `testing.F` |
| Java/Kotlin | `jqwik` + JQF | JQF |
| C/C++ | `theft` | `libFuzzer` + AFL++ |
| Swift | `swift-testing` + Sourcery | `swift-testing` fuzz |

**Critical pairing rule:** the verifier requires *both* example-based and property-based tests. LLM-authored PBT alone catches 68.75% of HumanEval bugs (arXiv 2510.25297); combined with EBT it catches 81.25%. Crucible enforces the combination.

**Wall-clock budget:** 5 min default, 15 min max.

### Tier 2: Schemathesis contract + DST (default for service/API code, multi-component state)

#### Contract testing

For API/service changes, verifier runs `schemathesis` workflows derived directly from the OpenAPI/GraphQL spec. The agent is required to keep the spec in sync — diffs that break spec without updating it fail Tier 2 immediately.

#### Deterministic Simulation Testing

For concurrency-sensitive code (multiple goroutines, async actors, distributed transactions), the verifier runs DST.

**Enterprise tier:** Antithesis SaaS. ~$20K–$100K/yr enterprise contracts. Runs the system in a hypervisor-level deterministic simulator, branches timeline on failures, perfectly reproduces.

**OSS / self-hosted tier:** in-house DST harness patterned on TigerBeetle's VOPR / FoundationDB's Flow simulator. Single-threaded, virtualized clock+disk+net. Feasible because Crucible's primary stack is Postgres+Go/TS, which fits the deterministic-simulation pattern well.

**Wall-clock budget:** 15 min default, 45 min max. DST iteration count scales to fill the budget.

### Tier 3: Formal verification (only for `@critical` paths)

Auto-triggered when the multi-signal classifier in [06-research/tier3-trigger-automation.md](../06-research/tier3-trigger-automation.md) scores the touched file ≥ 80 (Molten band). Otherwise the developer must annotate explicitly: `@crucible:critical(spec="auth", level="dafny")`.

| Tool | Sweet spot | Notes |
|---|---|---|
| **Dafny + DafnyPro** | General business logic, auth state machines, money paths | POPL 2026: 86% on DafnyBench with Sonnet 3.5; pair with Laurel for auto-assertions. Most pragmatic Tier-3 target. |
| **Lean 4 + mathlib + LeanCopilot** | Crypto primitives, numerical kernels, math-heavy code | Premise retrieval over mathlib's 100k+ lemmas; FormL4 dataset; TheoremLlama |
| **TLA+ + Apalache** | Distributed invariants, consensus, replication | Human writes spec, LLM proposes inductive invariants, Apalache validates |
| **Kani** | Rust `unsafe` blocks, FFI boundaries | Pairs with proptest via propproof |
| **Z3 / CVC5** | Inline SMT queries, quantifier instantiation | LLM-guided hints (+80% Z3, +183% CVC5 solve rate per arXiv 2601.04675) |

**Wall-clock budget:** Dafny 10 min, Lean 30 min, TLA+ model-check 20 min.

**Timeout policy:** do **not** fail open. Degrade to "Tier 2.5":
- Exhaustive PBT (≥10,000 cases)
- Mutation testing on the diff
- Mandatory CODEOWNER human review

Cache partial proofs — incremental verification on the next PR resumes where it left off. Chronic Tier-3 timeouts surface to the team dashboard so proof-engineering effort can be prioritized.

### Tier 4: Honest CI (reproducible build + SLSA-L3 attestation)

The verifier refuses to mark `done` unless:

1. **Hermetic rebuild.** Nix or Bazel rebuilds the artifact independently. Verifier compares hashes — bit-identical or task fails.
2. **In-toto attestation.** Build provenance materialized as an in-toto attestation, signed by Sigstore keyless OIDC. OIDC subject must match the agent worker's identity.
3. **SLSA Level 3.** GitHub's `actions/attest-build-provenance` gets to SLSA-L2 trivially; hardened runners + dual-build platform get to L3. Witness for non-GitHub pipelines. Tekton Chains for K8s.
4. **Rego policy admission.** The build's provenance bundle must satisfy the customer's Rego policy (which includes things like "artifact must be signed by an identity in the allowed set", "Dockerfile must be reproducible-build-marked", etc.).

**Wall-clock budget:** rebuild varies hugely by project; 5 min default, 30 min max. Cached intermediate artifacts (Nix store, Bazel disk cache) reduce typical rebuild to <1 min after first run.

## Verifier process architecture

Runs as a separate sandbox, separate model, separate egress policy:

```
┌──────────────────────────────────────────────────────┐
│  Executor Sandbox                                    │
│  ├ Agent (e.g. Opus 4.7)                             │
│  ├ Tools, FS, DB, services                           │
│  └ Emits: PromotionBundle candidate                  │
└────────────────────────┬─────────────────────────────┘
                         │
                         ▼
┌──────────────────────────────────────────────────────┐
│  Verifier Sandbox  (FRESH, DIFFERENT MODEL FAMILY)   │
│  ├ Verifier (e.g. Gemini 3.1 Pro)                    │
│  ├ Read-only access to executor's diff + tests       │
│  ├ Own twin DB (independent Neon branch)             │
│  ├ Own egress policy                                 │
│  ├ Runs Tier 0/1/2/3 + Tier 4 pipeline               │
│  └ Emits: VerifierApproval | VerifierRejection       │
└──────────────────────────────────────────────────────┘
```

Verifier has **no access to the executor's reasoning trace** — it sees only the diff, the tests, the spec changes, and the OpenAPI delta. This prevents the verifier from inheriting the executor's hallucinations through the reasoning channel.

## Verifier output: `VerifierApproval`

```typescript
{
  task_id: string,
  diff_hash: string,
  verdict: "approved" | "rejected",
  tier_results: {
    tier_0?: { passed: boolean, mutation_score: number, killed: number, survived: number },
    tier_1?: { passed: boolean, pbt_iterations: number, counterexamples: any[] },
    tier_2?: { passed: boolean, contract_violations: any[], dst_replay_id?: string },
    tier_3?: { passed: boolean, prover: string, proof_artifact: string, timed_out: boolean },
    tier_4?: { passed: boolean, rebuild_hash: string, rekor_uuid: string }
  },
  rubric_score: number,    // 0..1, only on approval
  rejection_reasons: string[],
  attestations: RekorUUID[],
  signed_by_oidc: string,
  signed_at: timestamp,
}
```

Rejection reasons are structured (e.g., `"tier_1.pbt_counterexample: input [1,2,3] → output []; expected non-empty"`) so the executor can reflect and retry. Up to 3 retry rounds per the Bounded Budget Enforcer; after that, halt and ask the human.

## Performance & cost

- **Median task:** Tier 0 + Tier 1 only. Verification adds 30–60s wall-clock and ~$0.14 cost (Gemini 3.1 Pro at $2/$12, ~40K input + ~5K output, no cross-vendor cache so full input price).
- **Service/API task:** + Tier 2. Adds 5–15 min wall-clock, ~$0.40.
- **Critical-path task:** + Tier 3. Adds 10–30 min wall-clock; cost varies by prover (~$0.50–$2.00 for the LLM-driven proof search).
- **Every task:** + Tier 4. ~$0.05 for the attestation publish; rebuild time is project-dependent.

**Cost engineering:** verifier runs **once at the end** of the task, not in-loop. This is why "2× tokens" is actually closer to 1.08× in practice — verification is a small fraction of total task cost. See [00-vision/pricing-and-business.md](../00-vision/pricing-and-business.md) and [06-research/unit-economics.md](../06-research/unit-economics.md).

## What the verifier cannot catch

Honest limits:

- **Spec drift the verifier shares with the executor.** If both models hallucinate the same incorrect Stripe API, neither catches it. Mitigation: Tier 2 schemathesis pulls from the *actual published spec*, not an LLM-derived one.
- **Tape-staleness bugs.** A verified-twin success can fail in real prod if the service changed since the tape was recorded. Mitigation: tape-age metrics; promotion canary catches it; auto-rollback.
- **Semantic correctness without testable invariants.** "Make this UI look good" has no verifier signal. Mitigation: design-token-based UI generation + visual regression; out of scope for v1 verifier.
- **Performance regressions invisible to the test suite.** Mitigation: tier-2 includes a perf-regression check via benchmark replay; tier-3 hot-path classification triggers explicit perf invariants.

These limits are documented in customer-facing materials — calibrated trust beats overclaimed trust.

---

<a id="file-01-architecture--memory-layer"></a>

<!-- ================================================================== -->
<!-- File: 01-architecture/memory-layer.md -->
<!-- ================================================================== -->

# Memory Layer

Per-tenant, three-store memory architecture. The compounding moat: every PR review comment, post-mortem, and ADR a team has ever written becomes input to a procedural-memory graph that gets stickier monthly.

## The three stores

```
              ┌─────────────────────────────────────────┐
Agent loop ──▶│ Retrieval Router (multi-signal, ≤7K tok)│
              └────┬────────┬─────────┬─────────────────┘
                   │        │         │
              ┌────▼──┐ ┌───▼───┐ ┌───▼────────┐
              │Redis  │ │pgvec/ │ │FalkorDB +  │
              │ K/V   │ │Qdrant │ │Graphiti    │
              │(hot   │ │(epis. │ │(procedural,│
              │ ctx)  │ │+sem.) │ │ temporal)  │
              └───────┘ └───────┘ └────────────┘
                   ▲        ▲         ▲
                   │        │         │
              ┌────┴────────┴─────────┴───────┐
              │ Background Distillation Worker│
              │  • PR comment KG extractor    │
              │  • Post-mortem ingestor       │
              │  • Convention-drift detector  │
              │  • Importance scorer + GC     │
              └───────────────────────────────┘
                   ▲
                   │ (PRs, runbooks, ADRs, incident reports)
```

### Store 1: Redis (hot)

The agent's working set during a single task.

- Current task context, last 50 tool calls, active branch state, plan in flight.
- TTL minutes–hours.
- ~100 MB per tenant typical.
- Single-purpose: keep the agent's running window cheap to access. Not for long-term storage.

### Store 2: pgvector / Qdrant (episodic + semantic)

Cross-task memory of "things the agent has seen and decided."

- Session transcripts (compressed), retrieved code snippets, prior agent decisions and outcomes.
- TTL 30–90 days, importance-scored via multi-dimensional A-MAC (future utility × factual confidence × novelty × recency).
- Row-level security on `tenant_id + repo_id` enforces isolation.

**Default pick:** pgvector if customer already runs Postgres (~$1–2K/mo per 10M vectors on a beefy instance, no second system).

**Greenfield alternative:** Qdrant — better filter perf for richer JSON payloads, ~$30–50/mo self-hosted small, ~$65/mo cloud at 10M vectors.

**Scale alternative:** Turbopuffer — S3+SSD, ~$70/TB/mo, $9/M ops. Relevant past ~10M vectors.

**Avoid:** Pinecone (vendor lock-in + expensive at scale), Milvus (operational overhead unless >100M vectors).

### Store 3: FalkorDB + Graphiti pattern (procedural)

The long-lived team-knowledge graph.

- Team conventions, incident patterns, supersession chains, ADR-derived decisions.
- Bi-temporal edges (`valid_from`, `valid_to`) — every fact has a "when it was true" plus "when we recorded it."
- No TTL; lifecycle via `status: active | drifting | superseded`.
- This is the moat: it grows monotonically with team usage.

**Default pick:** FalkorDB. Low-latency Cypher, AI/GraphRAG-tuned, source-available. The de-facto KuzuDB successor after KuzuDB was archived October 2025 post-Apple acquisition.

**Alternative:** Neo4j (larger ecosystem, more mature, more expensive).

**Avoid:** KuzuDB (archived), ArangoDB (multi-model is overkill for this use).

**Abstraction layer:** Graphiti (Zep's OSS engine) — temporal knowledge graph atop the chosen graph backend. Crucible should adopt the Graphiti API even if the backend swaps.

## The procedural data model

```typescript
Convention {
  id: string,
  scope: { kind: "repo" | "team" | "org" | "path-glob", value: string },
  confidence: number,                     // 0..1
  rule_nl: string,                        // "PR titles use conventional commits"
  rule_machine: string | null,            // optional regex / matcher
  category: string,                       // see taxonomy below
  positive_examples: SourceRef[],         // PR refs
  negative_examples: SourceRef[],         // PRs corrected in review
  source: SourceRef[],                    // PR comment IDs, incident IDs, ADR refs
  first_seen: timestamp,
  last_reinforced: timestamp,
  last_violated: timestamp | null,
  status: "active" | "drifting" | "superseded",
  supersedes: ConventionId[],
  tenant_id: string,
  repo_id: string,
}
```

### Convention taxonomy (12 categories)

Mapped 1:1 to the AGENTS.md section conventions used by the top 2,500 repos:

1. **Naming** — identifiers per kind, file naming, test naming, module path style
2. **Layering** — allowed import directions, architectural boundaries
3. **Library preferences** — date-fns over moment, zod over yup, vitest over jest
4. **Test patterns** — colocated vs `__tests__/`, mocking boundaries, snapshot policy
5. **Error handling** — Result/Either vs exceptions vs sentinels
6. **Logging** — structured (slog/zap/pino), sampling, PII redaction list
7. **Migration patterns** — additive-only, backfill jobs, feature flags
8. **PR/commit hygiene** — Conventional Commits, semantic-release, max diff size
9. **Security defaults** — auth middleware position, input-validation lib, rate limiting
10. **Performance defaults** — N+1 prevention, cache choice, query timeouts, pagination
11. **Concurrency** — goroutine lifecycle, context propagation, async/await vs sync
12. **API shape** — REST vs gRPC, error envelope, idempotency keys

## Background distillation worker

The distiller runs as a queue worker, **not** in the agent's hot path. Architecture:

```
PR webhooks ──▶┐
Incident exports ▶┤
ADR commits ──▶┤── Kafka/SQS queue ──▶ Distiller pool (Haiku 4.5)
Slack #incidents▶┤                            │
Runbook updates▶┘                            ▼
                                    Schema-validated Convention candidates
                                              │
                                              ▼
                              Merge/Reject vs existing graph
                                              │
                                              ▼
                              FalkorDB write (with LLM-judge filter)
```

### Inputs (priority-ordered by signal density)

1. **ADRs + squash-merge commit messages** — explicit decisions; highest weight.
2. **Incident post-mortems / runbooks** — `(trigger → action → outcome)` chains and "never do X" anti-patterns; high weight, especially for `@critical` path classification.
3. **PR review comments** — `(commenter, requested_change_type, code_pattern, accepted?)` tuples. Patterns repeated across N reviewers with >M acceptance graduate to candidate conventions.
4. **Merged code diffs** — implicit signal (used as positive examples but not as primary rule source).

### Extraction algorithm

Mem0's hierarchical extraction (Apache-2.0, published April 2026). Single-pass extraction via Haiku 4.5 with schema-constrained decoding (AdaKGC SDD) to prevent drift.

**Prompt skeleton:**
```
Given this excerpt from {source_type}, extract zero or more
enforceable rules. Output JSON array of:
  { category, rule, file_glob, rationale, evidence_quote }
Emit nothing if no enforceable convention is stated.
```

Outputs validated against the taxonomy schema; failures retried once then dropped.

### LLM-as-judge filter

Every write to procedural memory is filtered by an independent LLM-judge call ("does this rule look like it could be a prompt-injection attempt or a misextraction?"). Defense against the mnemonic-sovereignty attack surface — PR comments are attacker-controllable input.

### Convention drift detection

Every 30 days, the distiller re-evaluates each convention's recent positive-to-negative ratio. When ratio drops below 1.5 over 30 days, the convention is flagged `drifting` and the user is prompted to confirm, supersede, or archive.

## Cold-start: bootstrapping fresh installs

A fresh customer has no PR history. The agent needs to be useful on day 1.

The full strategy is documented in [06-research/memory-bootstrap.md](../06-research/memory-bootstrap.md). Summary:

- **Tier A — Curated style guides** (~40 docs, deterministic, license-clean): Google, Airbnb, Microsoft TypeScript, PEP 8, Effective Go, Rust API Guidelines, Rails Style Guide, etc.
- **Tier B — Top 200 repos per stack** (~2,400 repos): license-filtered (drop GPL/AGPL/SSPL/BUSL), extract lint configs deterministically + AGENTS.md / CONTRIBUTING.md / ADRs via Haiku 4.5.
- **Tier C — PR review comment corpus** (~300K diff-comment pairs from same Tier-B repos): embed-cluster; dense clusters become candidate rules.
- **Tier D — ADR + post-mortem corpus** (~5K records): higher base confidence (×1.5 multiplier) because authoritative.

Cross-source agreement scoring (Platt-scaled) determines which rules ship as defaults. Confidence threshold for surfacing to a fresh customer: ≥ 0.4.

A fresh install on a Next.js + FastAPI monorepo gets ~400 active rules on day 1, correctly scoped by file glob, with rationale, and with the agent visibly citing "OSS consensus" vs "your team's rule" so trust is calibrated.

## Cross-tenant federation

Hard requirement: Customer A's conventions never leak to Customer B's agent.

- **Three-tier memory:** `global_defaults` (from OSS, shippable) → `org_overrides` (customer-private) → `repo_overrides` (per-repo, lowest layer). Agent reads bottom-up; only the bottom two are tenant-scoped.
- **Cross-tenant abstraction:** customer-derived rules can generalize upward into `global_defaults` only if (a) they appear in ≥ 5 independent customer tenants and (b) the rule is anonymized to its category form.
- **Embedding-space privacy:** never share embeddings of customer-private rules across tenants. Per-tenant namespaces in the vector store.
- **Differential privacy** on cross-tenant aggregate signals if/when published.

## Memory as verifier

Before marking a task done, the verifier (independently of the executor) re-queries procedural memory for conventions relevant to the diff and asserts compliance. This is the loop closure:

> Memory learns from PRs → memory enforces what it learned on future PRs.

This is the most direct realization of the "team taste" feature. Every PR that gets human-corrected feeds the rule that prevents the next agent from making the same mistake.

## Eviction, decay, importance

Per the Mem0 2026 state-of-memory report and A-MAC adaptive admission control:

- **Multi-dimensional importance:** future utility, factual confidence, semantic novelty, temporal recency, content type prior.
- **Ebbinghaus exponential decay** on recency; reinforce-on-access (frequently retrieved memories live longer).
- **TTL:**
  - Hot (Redis): minutes–hours.
  - Episodic (pgvector): 30–90 days, importance-weighted.
  - Procedural (FalkorDB): no TTL; lifecycle via `status`.
- **Bounded growth via importance-thresholded admission.** Below-threshold candidates are dropped at write time rather than evicted later.
- **Working-set discipline:** keep retrieval-router output ≤ 7K tokens. Don't dump entire repos into prompts — that's what the "context window is RAM not storage" Mem0 thesis is about.

## Retrieval router

Multi-signal hybrid retrieval. On every agent query:

1. **Exact-match key lookup** (Redis): current branch state, last tool call.
2. **Semantic recall** (pgvector / Qdrant): top-K snippets by embedding similarity, filtered by tenant_id + repo_id + file-glob scope.
3. **Procedural lookup** (FalkorDB): conventions whose scope matches the current file path or category.
4. **Importance re-ranking:** combine A-MAC importance with semantic similarity score.
5. **Token budget enforcement:** total context ≤ 7K tokens; drop lowest-scored items to fit.

Cached aggressively (1h TTL on the router's output for the same query+context pair).

## API surface (agent-facing)

```typescript
twin.memory.recall(query: string, scope: Scope): Memory[]
// Multi-signal retrieval. Returns up to 7K tokens of relevant memory.
// Scope = { repo, file_glob, category } | "all"

twin.memory.note(fact: string, source: SourceRef): MemoryId
// Explicit save. Used when the agent learns something the distiller
// would miss — e.g., a user correction in the current task.

twin.memory.conventions(scope: Scope): Convention[]
// Returns active conventions for the given scope. Used at plan time
// and during verifier's compliance check.
```

The agent does not directly read or write the underlying stores. All access goes through `twin.memory.*`.

## What's deliberately not in scope for v1

- **Agent-to-agent memory sharing** beyond cross-tenant federated abstractions.
- **Visual memory** (screenshot retrieval, diagram understanding).
- **Voice memory** (transcribed stand-ups, recorded code reviews).
- **End-to-end encrypted memory** (E2EE on the customer's vault key — interesting for v2 enterprise tier).

See [06-research/memory-bootstrap.md](../06-research/memory-bootstrap.md) for the full cold-start strategy.

---

<a id="file-01-architecture--model-routing"></a>

<!-- ================================================================== -->
<!-- File: 01-architecture/model-routing.md -->
<!-- ================================================================== -->

# Model Routing

Five tiers, ~12 models, route by task class. Cross-family executor/verifier pairing is the architectural anti-hallucination contract.

## The tier table (May 2026 reference pricing)

| Tier | Role | Primary | Alternates | $ in / out / cache | Context |
|---|---|---|---|---|---|
| **0** | File reads, grep, planning decomposition, leaf retrieval | `claude-haiku-4-5` | `gemini-3-flash-lite` ($0.10/$0.40), `deepseek-v4-flash` ($0.14/$0.28) | $1 / $5 / $0.10 | 200K |
| **1** | Standard coding, multi-file edits, test authoring | `claude-sonnet-4-6` | `gpt-5.1-codex-max` ($1.25/$10), `gemini-3-flash` ($0.50/$3) | $3 / $15 / $0.30 | 1M |
| **2** | Hard refactors, architecture, property-test/invariant authoring | `claude-opus-4-7` | `gpt-5.5` ($5/$30), `gemini-3.1-pro` ($2-4/$12-18, 2M ctx) | $5 / $25 / $0.50 | 1M |
| **3** | **Verifier** — must be different family from executor | When primary = Opus 4.7 → `gemini-3.1-pro` | When primary = GPT-5.5 → Opus 4.7 | Varies | — |
| **4** | Local / privacy-sensitive | `Llama-4-Scout` (10M ctx) | `DeepSeek-V4-Pro` (MIT, 1M), `Qwen3-Coder-Plus` (262K) | self-hosted | — |

## Routing rules

### 1. Task class drives tier

Inferred from manifest + procedural memory.

- "Fix typo / rename" → Tier 0.
- "Add field to a struct, propagate" → Tier 1.
- "Refactor service to use new dependency" → Tier 2.
- "Authoring property tests / formal invariants" → Tier 2.
- "Touched file scores ≥ 80 in critical classifier" → Tier 2 executor + Tier 3 verifier with Dafny/Lean/TLA+.

The router's classifier is itself a small LLM call (Haiku 4.5, cacheable) over the task description + initial repo cartography.

### 2. Verifier pairing is mandatory

Every task has both an executor model and a verifier model. **They must be from different families** (different vendor lineage, different tokenizer, different RL recipe). Strong pairings:

- `claude-opus-4-7` ↔ `gemini-3.1-pro` (high thinking)
- `gpt-5.5` ↔ `claude-opus-4-7`
- `Llama-4-Maverick` (local) ↔ `DeepSeek-V4-Pro` (local)

Configured per-tenant. BYOK and self-hosted customers can override.

### 3. Cache strategy

Anthropic and Google both expose explicit prompt caching. OpenAI uses automatic caching (~5–10 min TTL). The router schedules:

- **System prompt + repo cartography** → 1h cache slot. Saves ~90% of input cost across a single task.
- **Active file context** → 5m cache slot. Refreshed on every edit.
- **Tool definitions** → 1h cache slot. Static for the task.

Cross-vendor cache transfer is impossible — verifying with Gemini incurs full input cost on its first pass even though the executor was Anthropic. This is the single biggest cost line item; engineering investment in keeping verifier prompts small is critical.

### 4. Budget enforcement

Every plan declares a dollar budget. The Bounded Budget Enforcer (Control Plane) tracks token spend per call and halts execution when the budget is exceeded. The user must re-plan to continue.

Budgets per tier (default; user-tunable):

| Plan tier | Budget cap |
|---|---|
| Trivial | $0.50 |
| Standard | $2.00 |
| Complex | $10.00 |
| Critical | $25.00 |
| Modernization (Outcome tier) | $50.00 |

## Per-vendor specifics

### Anthropic

- **Opus 4.7** (`claude-opus-4-7`): 1M context, 128K output, $5/$25/$0.50, 5m and 1h cache TTLs. Adaptive thinking (model decides depth). New tokenizer uses ~35% more tokens than older Claude — account for in budget.
- **Sonnet 4.6** (`claude-sonnet-4-6`): 1M, 64K output, $3/$15/$0.30. Extended thinking toggleable.
- **Haiku 4.5** (`claude-haiku-4-5`): 200K, 64K output, $1/$5/$0.10. Extended thinking yes.
- All support computer-use, tool calling, vision, MCP.
- **Best for:** agentic loops, tool use, computer use. Default executor.

### OpenAI

- **GPT-5.5** (`gpt-5.5`): ~920K input, 128K output, $5/$30, automatic caching. `reasoning_effort` parameter.
- **GPT-5.3-Codex** (`gpt-5.3-codex`): 400K context, $1.75/$14. Code-specialized; #1 Terminal-Bench 2.0.
- **GPT-5.1-Codex-Max** (`gpt-5.1-codex-max`): 400K, $1.25/$10. Cheapest OpenAI agentic option.
- First-class JSON Schema strict mode + function calling.
- **Best for:** terminal-bound verification, JSON-schema-strict outputs.

### Google

- **Gemini 3.1 Pro** (`gemini-3.1-pro-preview`): 2M context, $2/$12 (<200K) or $4/$18 (>200K). Configurable thinking levels. #1 LiveCodeBench Elo 2887.
- **Gemini 3 Flash**: ~1.05M, $0.50/$3.
- **Gemini 3 Flash-Lite**: 1M, ~$0.10/$0.40.
- All support explicit + implicit caching, JSON-Schema responseSchema, native multimodal.
- **Best for:** Tier 3 verifier on Opus-executed tasks; algorithmic invariant authoring; 2M-context whole-repo passes.

### xAI

- **Grok 4.3** (`grok-4.3`): 1M, $1.25/$2.50. Code-ready successor to Grok-Code-Fast-1.
- Useful as a third-family fallback for sensitive teams who want non-Big-Three vendor mix.

### DeepSeek

- **DeepSeek V4-Pro**: 1M, $1.74/$3.48 standard (75% off through May 31 2026: $0.435/$0.87). MIT-licensed open weights. Native and `/anthropic` endpoints.
- **Best for:** self-hosted privacy tier; cheap verifier when paired with Claude/GPT executor.

### Open-weights (Llama, Qwen)

- **Llama 4 Scout** (10M context, 73.4% SWE-Bench Verified): primary local-host pick for privacy-sensitive customers.
- **Qwen3-Coder-Plus** (80B MoE, 262K context, strong agent tool calling): alternative; open weights on HuggingFace.

## Routing decision algorithm

```python
def route(task: Task, tenant: Tenant) -> Routing:
    # 1. Classify task complexity
    complexity = classify_complexity(task)  # Haiku 4.5 call, cached

    # 2. Determine if critical-path scoring applies
    critical_score = critical_classifier(task.touched_files, tenant)
    is_critical = critical_score >= 80

    # 3. Pick executor tier
    if complexity == "trivial":
        executor_tier = 0
    elif complexity == "standard":
        executor_tier = 1
    elif complexity == "complex" or is_critical:
        executor_tier = 2

    # 4. Pick executor model from tenant config or default
    executor = tenant.model_overrides.get(executor_tier, DEFAULTS[executor_tier])

    # 5. Pick verifier from DIFFERENT family
    verifier = pick_cross_family_verifier(executor)
    if is_critical:
        verifier = upgrade_to_tier3(verifier, prover_choice(task))

    # 6. Budget allocation
    budget = budget_for(complexity, critical=is_critical, tenant=tenant)

    return Routing(executor, verifier, budget, complexity, critical_score)
```

## Privacy / data-residency rules

Per-tenant policy controls which routes are allowed:

- **Standard tenant:** any frontier model.
- **EU-data-residency tenant:** Anthropic EU region, Gemini EU region, no US-only models.
- **Healthcare HIPAA tenant:** BAA-covered models only (Anthropic w/ BAA, Azure OpenAI w/ BAA, Vertex AI w/ BAA).
- **Air-gap / FedRAMP tenant:** Tier 4 models only, local-host. No external API calls.

Policy enforced at the router; violations return `RoutingDenied` with the policy name.

## Cost telemetry

Every model call emits an OTel span with:

- `model.vendor`, `model.id`, `model.tier`
- `tokens.input.fresh`, `tokens.input.cached`, `tokens.output`
- `cost.usd` (computed via current price table)
- `task_id`, `step_id`, `tenant_id`

Dashboards in [02-engineering/observability.md](../02-engineering/observability.md) aggregate these to:

- Per-task cost (median, p95)
- Cache hit rate (the critical KPI; must stay ≥ 70%)
- Verifier cost as % of total (sanity check that we're not hitting 2× regression)
- Per-tenant routing distribution (informs upsell)

## What changes in v2

- **Custom Composer-2-style in-house model** for Tier 1 cost-cutting (Cursor's strategic move). Tabled until v1 PMF clear.
- **Speculative-decoding pairings** (cheap proposer + frontier verifier in the same call) when vendor APIs support it broadly. Currently emerging in Anthropic Sonnet/Opus pairings.
- **Model price oracle** — auto-rebalance routing as vendor prices shift quarterly. v1 hardcodes May 2026 pricing.

See [05-decisions/ADR-006-cross-family-verifier.md](../05-decisions/ADR-006-cross-family-verifier.md) for the rationale on mandatory cross-family pairing.

---

<a id="file-01-architecture--promotion-contract"></a>

<!-- ================================================================== -->
<!-- File: 01-architecture/promotion-contract.md -->
<!-- ================================================================== -->

# Promotion Contract

The bridge from twin to real. Every verified change becomes a real-system change only via this contract.

## The contract, in steps

When the agent calls `twin.promote(bundle)`:

```
1. Provenance verification
   ├─ Every in-toto attestation in the bundle is checked against Sigstore Rekor
   ├─ OIDC subjects of all signers are validated against the allowed set
   └─ Bundle is rejected if any attestation is missing or invalid

2. Rego policy evaluation
   ├─ Trivial diffs (Tier 0 verified, no schema change, no critical paths)
   │     → Auto-approve
   ├─ Schema changes, critical-path touches, first-time author-of-area
   │     → Require human approval
   └─ Policy violations (e.g., missing CODEOWNER approval)
         → Reject

3. Human approval (if required)
   ├─ Slack button or web UI
   ├─ Approver signs via Sigstore keyless OIDC
   └─ Approval attestation published to Rekor

4. KMS-signed credential lease
   ├─ AWS KMS / GCP Cloud HSM / YubiHSM signs a single-use credential
   ├─ Scoped to the specific action (deploy this artifact, run this migration)
   ├─ Time-boxed (typical: 5 min)
   └─ Consumed by the deploy pipeline, never returned to the agent

5. Progressive rollout
   ├─ Argo Rollouts canary (Kubernetes) or Flagger (Linkerd/Istio)
   ├─ Traffic mirroring to the new version
   ├─ AnalysisTemplate watches Prometheus SLOs
   ├─ GrowthBook feature flag for fast rollback (millisecond)
   └─ Auto-rollback on SLO regression

6. Final attestation
   ├─ Promotion outcome (success / rolled-back) published to Rekor
   ├─ Procedural memory updated with patterns learned
   └─ Task marked complete
```

## The `PromotionBundle`

The artifact the agent produces and the contract consumes:

```typescript
PromotionBundle {
  task_id: string,
  diff_hash: string,                       // hash of all file changes
  files_changed: { path: string, action: "add" | "modify" | "delete" }[],

  // Verifier output
  verifier_approval: VerifierApproval,     // signed by verifier OIDC
  tier_results: TierResults,

  // Provenance
  attestations: RekorUUID[],               // every action in the task
  build_provenance: SLSAProvenance,        // SLSA-L3 attestation
  rebuild_hash: string,                    // hermetic Nix/Bazel hash

  // Risk & impact
  blast_radius: {
    affected_services: string[],
    affected_endpoints: string[],
    schema_changes: SchemaChange[],
    critical_paths_touched: string[],
    estimated_impact: "low" | "medium" | "high"
  },

  // Deploy plan
  suggested_rollout: {
    strategy: "canary" | "blue-green" | "feature-flag-only",
    canary_percentages: number[],          // e.g. [1, 5, 25, 100]
    dwell_seconds_per_step: number,
    analysis_template_ref: string,         // points to Prometheus rules
    rollback_trigger: string               // e.g. "error_rate_p99 > 0.5%"
  },

  // Signing
  agent_oidc_subject: string,
  signed_at: timestamp,
}
```

## Rego policy structure

The default policy bundle (per-tenant overridable):

```rego
package crucible.promotion

default allow = false
default require_human = false

# Auto-approve trivial
allow {
  input.tier_results.tier_0.passed
  not has_schema_change
  not has_critical_path
  input.blast_radius.estimated_impact == "low"
}

# Schema changes need human approval
require_human {
  has_schema_change
}

# Critical-path changes need human approval AND CODEOWNER signature
require_human {
  has_critical_path
}

require_codeowner {
  has_critical_path
}

has_schema_change {
  count(input.blast_radius.schema_changes) > 0
}

has_critical_path {
  count(input.blast_radius.critical_paths_touched) > 0
}

# Reject if Tier 4 didn't run on production-touching changes
deny[msg] {
  not input.tier_results.tier_4.passed
  input.blast_radius.estimated_impact != "low"
  msg := "Tier 4 reproducible-build attestation required for non-trivial promotions"
}
```

Customers can layer their own rules — e.g., "no promotions during merge freeze," "deploys to prod-eu require EU-based approver," etc.

## Progressive rollout

### Kubernetes (Argo Rollouts)

Default rollout strategy uses Argo Rollouts `AnalysisTemplate` against Prometheus:

```yaml
strategy:
  canary:
    steps:
      - setWeight: 1
      - pause: { duration: 5m }
      - analysis:
          templates:
            - templateName: crucible-slo-check
      - setWeight: 5
      - pause: { duration: 10m }
      - analysis: { ... }
      - setWeight: 25
      - pause: { duration: 30m }
      - analysis: { ... }
      - setWeight: 100

analysisRunMetadata:
  rollout: from-crucible
  task_id: <task_id>
```

Auto-rollback on:
- Error rate p99 > pre-rollout baseline × 1.5
- Latency p95 > baseline × 1.3
- Custom rules per service (defined in the task manifest)

### Non-K8s (serverless / VM-based)

Feature-flag-driven rollouts via GrowthBook:

- Flag created at promotion time, scoped to the change.
- Initial rollout: 1% of users.
- Periodic SLO check via Prometheus query (configurable per service).
- Step up percentage on clean dwell.
- Flag flip to 0% on regression (millisecond rollback).

### Database migrations

Special handling:

1. **Twin run** — migration applied to Neon twin branch; verifier checks resulting schema diff against expected.
2. **Shadow run** — same migration applied to a shadow of production (read-replica with replication paused), verifier checks no destructive DDL on production data.
3. **Promotion** — KMS-signed credential lease grants temporary `ALTER TABLE` permission; migration runs as a single transaction with statement timeout.
4. **Verification** — post-migration query checks (data integrity, row counts, expected indexes).
5. **Rollback** — if any check fails, transaction rolls back. For non-transactional DDL (e.g., MySQL pre-8.0), a manually-authored down-migration is required as part of the bundle.

## KMS signing

The "unseal ceremony" for destructive prod actions:

```
1. Approver clicks Slack button or web UI button
2. Sigstore keyless OIDC issues a short-lived cert for the approver
3. Crucible signs the action request with the OIDC cert
4. AWS KMS / GCP Cloud HSM / YubiHSM verifies the cert + signs a credential lease
5. Credential lease:
   - scoped to the specific action (e.g., "deploy artifact X to service Y")
   - time-boxed (5 minutes default)
   - single-use (idempotency key consumed on first use)
   - NEVER returned to the agent process
6. Deploy pipeline consumes lease, executes action, returns result
7. Lease automatically expires
```

For air-gapped / on-prem deployments, KMS is replaced by an on-prem HSM (e.g., Thales Luna, YubiHSM, AWS CloudHSM standalone).

## Approval routing

Who approves what is configured per-tenant:

```yaml
# tenant approval policy
default_approvers: ["@platform-team"]
overrides:
  - matches:
      schema_changes: true
    approvers: ["@dba-team"]
  - matches:
      critical_paths_touched: ["src/billing/*"]
    approvers: ["@payments-leads"]
    require_codeowner: true
  - matches:
      blast_radius.estimated_impact: "high"
    approvers: ["@on-call", "@eng-leadership"]
    require_n_approvers: 2
```

All approvals are signed and published to Rekor. Audit log is therefore queryable: "show me all critical-path deploys in the last 30 days and who approved each."

## Failure handling

### Promotion bundle rejected at policy gate

Returned to agent with structured rejection. Agent surfaces to user. If recoverable (e.g., missing CODEOWNER signature), user can add the missing approval and retry. If not, the change is held in the bundle store for the user to amend.

### Approval timeout

Configurable per-tenant. Default: bundle expires after 24 hours of waiting for approval. User can extend or refresh.

### Canary regression

Auto-rollback fires. Bundle marked `rolled_back`. Procedural memory records the failure pattern. Agent receives a structured "rollback report" with the SLO that triggered it, the diff, and the regression metrics — usable input for a retry task.

### Partial promotion (e.g., 2 of 3 services deployed, 3rd fails)

The promotion contract is atomic per-bundle. If any sub-deploy fails, **all** deploys in the bundle roll back. This is the difference between "deploy script" and "promotion contract."

## What we explicitly will not allow

- **Direct production access from the agent process.** Ever. The only path is through the KMS-signed credential lease.
- **Approval bypass for "emergencies."** If something is on fire, an approver clicks the button. Bypass paths are how trust dies.
- **Self-approval.** An agent that proposes a promotion cannot approve it. Different OIDC subjects required.
- **Stale approvals.** An approval is valid for one specific bundle hash. Any diff change invalidates it.

These are non-negotiable architectural invariants. See [threat-model.md](threat-model.md).

---

<a id="file-01-architecture--threat-model"></a>

<!-- ================================================================== -->
<!-- File: 01-architecture/threat-model.md -->
<!-- ================================================================== -->

# Threat Model

What we're defending against, what we're not, and where the architectural boundaries are. STRIDE-categorized with explicit out-of-scope items.

## Scope

This threat model covers the Crucible system from task submission through promotion. It does *not* cover:

- The customer's own production infrastructure (we issue signed credential leases; they own what happens with them).
- The IDE / MCP host the user runs Crucible through.
- Frontier LLM vendor (Anthropic/Google/OpenAI) breaches — we assume their APIs are trustworthy at the architectural level; defense-in-depth applies (no single-vendor dependency for verifier).

## Trust boundaries

```
┌─────────────────────────────────────────────────────────────────┐
│  User's IDE / MCP host          [TRUSTED — customer controls]   │
└─────────────────────────────────────────────────────────────────┘
                              │ MCP/HTTP
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│  Crucible Control Plane        [TRUSTED — Crucible operates]    │
└─────────────────────────────────────────────────────────────────┘
                              │ task spawn
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│  Twin Runtime (sandbox)        [UNTRUSTED — agent code runs]    │
│  ├ Agent process               [HOSTILE — assume compromised]   │
│  ├ Twin FS / DB / services     [SANDBOXED]                      │
│  └ Syscall shim, egress proxy  [ENFORCEMENT BOUNDARY]           │
└─────────────────────────────────────────────────────────────────┘
                              │ promotion bundle (signed)
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│  Promotion Contract            [TRUSTED — Crucible operates]    │
└─────────────────────────────────────────────────────────────────┘
                              │ KMS-signed credential lease
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│  Customer's Production         [TRUSTED — customer controls]    │
└─────────────────────────────────────────────────────────────────┘
```

The critical insight: **the agent process is treated as hostile.** It can be prompt-injected, model-corrupted, or fooled by tape-poisoned data. The architecture must contain it.

## STRIDE-categorized threats

### S — Spoofing

**T1.** Compromised LLM vendor returns malicious code under the agent's identity.
- *Mitigation:* cross-family verifier from a different vendor catches divergent behavior. Multi-region attestation chain. No single-vendor SPOF.

**T2.** Attacker submits a fake `PromotionBundle` impersonating the agent.
- *Mitigation:* every attestation in the bundle is signed via Sigstore keyless OIDC, with the OIDC subject bound to a specific worker identity. Promotion contract verifies the entire chain against Rekor before any action.

**T3.** Forged approval (someone clicks "approve" pretending to be the approver).
- *Mitigation:* approvers sign their approval with their own OIDC cert. Slack OAuth + SAML/SSO required for the approval UI.

### T — Tampering

**T4.** Agent modifies the attestation log to hide an action.
- *Mitigation:* attestations are written to an append-only transparency log (Sigstore Rekor v2) the agent has no write authority over. Local sandbox journal is signed and replicated to the log; tampering creates a hash chain break.

**T5.** Attacker poisons a Hoverfly tape to mislead the agent.
- *Mitigation:* tapes are content-addressed and signed at capture; mount-time checksum validation. PII scrub audit log catches anomalous rewrites. New tapes from PRs are flagged for human review before promotion to the active tape set.

**T6.** Procedural memory poisoning via malicious PR review comments.
- *Mitigation:* LLM-as-judge filter on every write to procedural memory ("does this rule look like a prompt-injection?"). Confidence threshold for surfacing rules. Counter-example pass identifies contradictions. Mnemonic Sovereignty paper attack patterns (arXiv 2604.16548) are specifically defended against.

**T7.** Tampered build artifact promoted to production.
- *Mitigation:* SLSA-L3 in-toto attestation; verifier rebuilds artifact hermetically and compares hashes. Rego policy on the promotion gate refuses non-attested artifacts.

### R — Repudiation

**T8.** Agent (or customer) denies having performed an action.
- *Mitigation:* every action is signed and published to Sigstore Rekor — public transparency log (or self-hosted Rekor for enterprise). Replay capability lets anyone reproduce the action from the log.

**T9.** Approver denies having approved a destructive operation.
- *Mitigation:* approver's OIDC-signed approval is in the log. Slack/SAML SSO trace links the OIDC identity to the human.

### I — Information disclosure

**T10.** Cross-tenant memory leakage (Customer A's procedural rules visible to Customer B).
- *Mitigation:* per-tenant namespaces in every store (Redis, pgvector, FalkorDB). RLS in Postgres. Cross-tenant federation only abstracts to category-form rules after ≥5 tenants agree. Embeddings never shared across tenants.

**T11.** PII in Hoverfly tapes leaks to other customers (in SaaS deployment).
- *Mitigation:* tapes are per-tenant by design — no cross-tenant tape sharing. Scrub pipeline at capture time prevents PII from being persisted in the first place. Tapes encrypted at rest with per-tenant KMS key.

**T12.** Production credential leak via compromised agent process.
- *Mitigation:* **agent process cannot reach real prod credentials.** They live in HSM-backed vault on separate VPC. Only the KMS-signed lease (single-use, time-boxed, action-scoped) reaches the deploy pipeline, never the agent.

**T13.** Reasoning trace leakage exposes proprietary code logic.
- *Mitigation:* reasoning traces are tenant-scoped, encrypted at rest, retained per the tenant's data-retention policy. BYOK and self-hosted tiers keep traces entirely in customer infrastructure.

**T14.** Network egress exfiltrates code to attacker-controlled host.
- *Mitigation:* Cilium/Tetragon eBPF egress allowlist with SIGKILL on violation. Per-task manifest declares allowed hosts; DNS resolution restricted.

### D — Denial of service

**T15.** Agent loops indefinitely, burning customer budget.
- *Mitigation:* Bounded Budget Enforcer in the Control Plane. Hard retry cap (3 per subgoal). Wall-clock cap per task. Dollar cap per plan. Visible to the user throughout.

**T16.** Attacker submits flood of tasks to exhaust per-tenant rate limits.
- *Mitigation:* per-tenant rate limiting at Control Plane. Per-user limits inside tenant. Standard DDoS protection at the edge (Cloudflare/equivalent).

**T17.** Hoverfly tape store fills up.
- *Mitigation:* TTL on tape entries (default 90 days unless explicitly pinned). Per-tenant storage quota. LRU eviction when quota reached.

### E — Elevation of privilege

**T18.** Agent escapes the Firecracker sandbox.
- *Mitigation:* defense-in-depth — Firecracker hardware isolation, syscall filter (seccomp-bpf profile), egress allowlist, no host-mount escapes. Any successful escape is a red-team finding, not normal-mode threat. Active monitoring for syscall anomalies.

**T19.** Agent bypasses the destructive-op gate.
- *Mitigation:* gate enforced at three layers — command-line parse, ptrace syscall filter, eBPF post-exec. Bypassing one is hard; bypassing all three is the threat-model boundary.

**T20.** Compromised CI runner promotes unauthorized artifact.
- *Mitigation:* hermetic rebuild + bit-identical hash check at promotion. Witness/SLSA-L3 attestations bind artifact to runner OIDC identity. CI runners required to be hardened (GitHub-hosted with attestations, or self-hosted with TPM-attested OIDC).

**T21.** Customer-side compromised approver clicks approve maliciously.
- *Mitigation:* this is outside our scope (customer-side compromise) but defense-in-depth: require N-of-M approvals for high-impact changes; configurable per-tenant. Audit log makes the compromise discoverable post-hoc.

## Specific attack scenarios we explicitly defend against

### "The PocketOS scenario"

> Agent finds an API token in an unrelated file, executes `railway down`, deletes production DB + backups in 9 seconds.

Defense layers:

1. **Secrets isolation.** The agent process literally cannot syscall to real prod credentials. The token in the unrelated file would be either a twin-scoped Infisical token (no prod access) or scrubbed at tape capture.
2. **Destructive-op gate.** `railway down` is intercepted by the syscall shim before exec, converted to a `DestructiveProposal`, requires HSM-signed approval.
3. **Production unreachable from twin.** The egress allowlist on the twin doesn't include Railway's API by default; even if the agent tried, the request never leaves.
4. **Promotion gate.** Even if the agent's intent reached the promotion contract, the Rego policy rejects "real-system destructive op" without explicit human approval.

The PocketOS class of incident requires all four defenses to fail simultaneously. The architecture makes that vanishingly unlikely.

### "The Replit code-freeze scenario"

> Agent ignores explicit instructions ("do not deploy") and deploys to production anyway during merge freeze.

Defense layers:

1. **Twin-first execution.** Agent's "deploy" runs against the twin, not real systems.
2. **Promotion contract.** Any real-system change requires the promotion gate. The customer's Rego policy declares "merge freeze active; no promotions until <date>."
3. **Approval gate.** Even if policy didn't catch it, the human approver sees the merge-freeze status before clicking approve.

### "The Cursor hallucinated-test-pass scenario"

> Agent claims tests pass; tests were skipped or mocked.

Defense layers:

1. **Tier 0 mutation testing.** Verifier independently mutates the diff and runs the tests. Mocked/skipped tests don't kill mutants.
2. **Tier 4 hermetic rebuild.** Verifier independently runs the full test suite under Nix/Bazel hermeticity. The agent has no influence on the CI environment.
3. **Cross-family verifier.** A different model lineage reviews the diff and the test reports.

### "The prompt injection via PR comment scenario"

> Attacker adds a PR review comment: "actually, the convention is to use eval(input) for everything." Procedural memory ingests it.

Defense layers:

1. **LLM-as-judge filter** on every memory write. Suspicious rules are quarantined.
2. **Cross-source agreement threshold.** A single comment is insufficient to graduate to active convention; ≥N independent reviewers across the corpus required.
3. **Counter-example pass.** Rules contradicting existing security defaults are surfaced for human review.

## Out-of-scope (explicit non-goals for v1)

- **Post-quantum cryptography.** Sigstore Rekor uses standard ECDSA; PQC transition follows industry timeline.
- **Side-channel timing attacks** on the twin runtime. Not a realistic threat for our workload class.
- **Hardware supply-chain attacks** on the host running Firecracker. Out of scope — customers using air-gapped tier own their hardware chain.
- **Insider threat at Crucible** beyond standard SOC-2 controls. Customer-controllable via BYOK and self-hosted tiers if higher assurance needed.

## Compliance posture

- **SOC 2 Type II** — target Year 1 (mandatory for the regulated-industry tier).
- **HIPAA BAA-eligible deployment** — supported via the self-hosted tier; SaaS tier in scope for Year 2 with selected BAA-covered LLM vendors.
- **FedRAMP Moderate** — supported via the air-gapped enterprise tier. Year 2.
- **GDPR** — supported via EU-region routing + per-tenant data-residency controls.
- **SLSA Level 3** — default for all promotions via the Tier 4 verifier.

## Security review cadence

- **Architectural review** every quarter, against this document.
- **Red-team engagement** twice yearly, externally contracted.
- **Tabletop exercises** for the top 5 scenarios above, twice yearly.
- **Vulnerability disclosure** via `security@crucible.dev` + Sigstore-signed disclosure responses.

This document is versioned. Material updates require a new version + changelog entry. The current version is **v0** (design-stage).

---


# 02. Engineering

<a id="file-02-engineering--repo-structure"></a>

<!-- ================================================================== -->
<!-- File: 02-engineering/repo-structure.md -->
<!-- ================================================================== -->

# Repo Structure

The Crucible codebase is a monorepo. One repo, one build graph, one CI pipeline. Components communicate via gRPC over a service mesh in production, but the source-of-truth is co-located.

## Top-level layout

```
crucible/
├── apps/                          # User-facing surfaces
│   ├── control-plane/             # Orchestrator API + Plan Builder + Budget Enforcer
│   ├── twin-runtime/              # Twin sandbox manager + lifecycle
│   ├── verifier/                  # Verifier daemon (Tier 0–4 ladder)
│   ├── distiller/                 # Background memory distillation worker
│   ├── promotion-gate/            # Rego policy engine + KMS signing pipeline
│   ├── web-console/               # Team dashboard (Next.js / shadcn)
│   ├── cli/                       # `crucible` CLI
│   └── ide-plugins/               # VS Code, JetBrains, Zed (ACP) bridges
│
├── services/                      # Supporting microservices
│   ├── attestation-relay/         # In-toto attestation publisher → Sigstore Rekor
│   ├── tape-scrubber/             # PII scrub + record/replay pipeline (Hoverfly wrapper)
│   ├── memory-router/             # Multi-signal retrieval router
│   └── cost-meter/                # Per-task cost telemetry + cap enforcement
│
├── libs/                          # Shared internal libraries
│   ├── sdk-go/                    # `twin.*` SDK in Go (agent process side)
│   ├── sdk-ts/                    # SDK in TypeScript
│   ├── sdk-py/                    # SDK in Python
│   ├── sdk-rs/                    # SDK in Rust
│   ├── attestation/               # in-toto / Sigstore signing helpers
│   ├── twin-spec/                 # Type definitions for Plan, Bundle, Verdict, etc.
│   ├── memory-spec/               # Convention data model, retrieval query types
│   ├── tape-format/               # Hoverfly tape format + scrub manifest
│   ├── policy/                    # Rego policy bundles + helpers
│   └── model-routing/             # Multi-vendor LLM router (Anthropic/Google/OAI/etc.)
│
├── verifiers/                     # Per-language verifier integrations
│   ├── python/                    # hypothesis, schemathesis, mutmut, atheris, dafnypro
│   ├── typescript/                # fast-check, stryker, jsfuzz
│   ├── rust/                      # proptest, cargo-mutants, kani, cargo-fuzz
│   ├── go/                        # rapid, go-mutesting, native fuzz
│   ├── java/                      # jqwik, pitest, jqf
│   ├── swift/                     # swift-testing, muter
│   ├── tier3-dafny/               # DafnyPro adapter
│   ├── tier3-lean/                # LeanCopilot adapter
│   ├── tier3-tla/                 # Apalache adapter
│   └── tier4-honest-ci/           # Nix/Bazel hermetic rebuild + SLSA-L3 attestation
│
├── infra/                         # IaC for hosted + self-hosted deployments
│   ├── terraform/                 # AWS/GCP/Azure base infra
│   ├── helm/                      # Kubernetes charts (control plane + workers)
│   ├── argo-rollouts/             # AnalysisTemplate library
│   ├── air-gap-bundle/            # Offline installer for enterprise tier
│   └── observability/             # Honeycomb/Tempo/Prometheus configs
│
├── examples/                      # End-to-end demos / sample integrations
│   ├── nextjs-stripe-demo/        # Reference customer workload
│   ├── django-payments/
│   ├── rust-axum-api/
│   └── go-grpc-service/
│
├── docs/                          # ← you are here
│
├── scripts/                       # Build / release / dev helpers
├── .github/                       # CI workflows + issue templates
└── BUILD                          # Bazel root (or `flake.nix` for Nix-based builds)
```

## Why monorepo

- **One build graph.** The verifier integrations depend on the SDK types depend on the twin-spec depend on attestation. Cross-cutting refactors need a single PR.
- **One CI pipeline.** Reproducible builds for the entire system. Tier 4 honest CI applies to *our own* releases.
- **One test surface.** Integration tests across components are first-class.
- **Versioning is monorepo-wide.** Components don't have independent semver; the system has one version.

## Build system

**Default: Nix flakes.** Hermetic, reproducible, multi-language. Required for Tier 4 self-verification (we eat our own dogfood).

**Alternative: Bazel.** If the team prefers, Bazel works — but Nix is the default because (a) it's the easier reproducibility story for SLSA-L3, (b) it integrates cleanly with the air-gapped enterprise installer, (c) Nix flakes are familiar to the senior-engineer ICP.

## Language-per-component decisions

Each app/service picks the right language for its job. No "one language for everything" mandate. Specific choices:

| Component | Language | Rationale |
|---|---|---|
| control-plane | Go | Single-binary deploy, strong gRPC story, predictable GC |
| twin-runtime | Rust | Firecracker integration, syscall shim performance, safety |
| verifier daemon | Go | Orchestrates per-language verifier processes; gRPC fan-out |
| distiller worker | Python | Best LLM SDK ecosystem; not perf-critical |
| promotion-gate | Go | OPA/Rego embedding via go-rego; KMS clients in Go |
| web-console | TypeScript (Next.js + shadcn) | Standard 2026 React stack |
| cli | Go | Cross-platform single binary |
| IDE plugins | TypeScript | VS Code / Zed ACP / JetBrains all support TS |
| attestation-relay | Rust | Sigstore client mature; perf-critical at scale |
| tape-scrubber | Python | Presidio is Python-native; Hoverfly wrapping via subprocess |
| memory-router | Go | Hot-path retrieval; latency-sensitive |
| cost-meter | Go | Hot-path telemetry; latency-sensitive |

SDKs: one per supported agent host language (Go, TS, Python, Rust). All generated from the same gRPC/protobuf schema in `libs/twin-spec/`.

## Inter-service communication

- **gRPC** for internal service-to-service. Schemas in `libs/twin-spec/` and `libs/memory-spec/`.
- **HTTP/JSON** for the public REST API (control-plane → external) and the IDE plugins.
- **MCP** for IDE integration (the IDE plugin acts as an MCP host; control-plane is the MCP server).
- **ACP** (Agent Client Protocol) for cross-IDE portability.
- **Webhooks** for outbound events (Slack approvals, GitHub PR comments, etc.).

## Dependency policy

- **Frontier libraries only.** We use the actively-maintained, latest-version, popular libraries. No vendored legacy code.
- **Open-source dependencies must be license-clean** for our redistribution context: MIT, Apache-2.0, BSD-3-Clause, MPL-2.0, or LGPL with dynamic linking. **No GPL, AGPL, SSPL, BUSL in core libs.** Self-hosted enterprise installer can include GPL components if they're sandboxed in user-runtime.
- **Vendored dependencies tracked in `THIRD_PARTY.md`** with version, license, source URL.
- **Renovate auto-PR** keeps deps fresh; security updates auto-merge after Tier 0 verification (yes, we use Crucible to maintain Crucible).

## Versioning

- **Calendar versioning** for releases: `YYYY.MM.PATCH` (e.g., `2026.06.0`). Plays well with quarterly release cadence and customer procurement.
- **Semver internal** for SDKs and protocol schemas. Breaking changes to the protocol bump major; we maintain backward compatibility for one major version.
- **API stability tier per component:**
  - **Stable** — public REST API, SDK types, MCP tool definitions. Breaking changes require major version + 90-day deprecation window.
  - **Beta** — promotion-gate Rego policy schema, memory-spec types. Breaking changes documented in CHANGELOG; minor-version cadence.
  - **Internal** — everything else; refactor at will.

## Code style

- **One linter per language**, configured at repo root and enforced in CI.
  - Python: `ruff` + `mypy --strict`
  - TS: `biome` (replacing prettier+eslint; fewer configs)
  - Go: `gofmt` + `golangci-lint` with our preset
  - Rust: `rustfmt` + `clippy -W clippy::pedantic` (selective opts)
- **One import order** per language; auto-enforced.
- **No comments unless they explain WHY.** Per [02-engineering/testing-strategy.md](testing-strategy.md), tests + types document WHAT.
- **One naming convention** per language (idiomatic). No team-wide enforcement of e.g. snake_case across all languages.

## Documentation in-repo

- `README.md` at every component root explains what it does in 3 paragraphs.
- `CHANGELOG.md` per component, auto-generated from conventional commits.
- `THREAT_MODEL.md` per security-critical component (twin-runtime, promotion-gate, attestation-relay).
- API docs auto-generated from protobuf schemas and exported types; published to `docs.crucible.dev`.

## Testing layout

Mirrors the source tree:

```
apps/control-plane/
  src/...
  test/
    unit/       # Per-file tests, mutation-tested
    integration/# Cross-service, against ephemeral Neon branch
    e2e/        # Full task lifecycle, against test tenant
```

Verifier integrations are tested against the Crucible Test Harness (CTH) — a curated set of test repos with known-good and known-bad PRs. See [testing-strategy.md](testing-strategy.md).

## What lives outside this repo

- **Customer test harnesses / fixtures** — those live in customer repos.
- **Public website + marketing content** — separate repo.
- **The OSS-released verifier harness** — split out to its own repo (`crucible/verifier`) under Apache-2.0 once stable; here it lives as the source of truth.
- **The OSS-released tape-scrub pipeline** — same pattern: developed here, mirrored to its own OSS repo.
- **Crucible Skills marketplace** — separate repo + registry (v2).

## CI pipeline

- **PR checks:** lint, type-check, unit tests (mutation-tested), integration tests against ephemeral infra, Tier 4 hermetic build verification.
- **Main-branch merges:** all of the above, plus full e2e on the Crucible Test Harness, plus chaos tests on the twin-runtime, plus a Crucible-self-verification run (our own agent verifies the PR we're about to merge).
- **Release:** Nix-bundled artifacts published to the public registry; SLSA-L3 attestations published to Rekor; Helm charts and air-gap bundle built and signed.

See [04-operations/self-hosted-install.md](../04-operations/self-hosted-install.md) for what gets shipped to customers.

---

<a id="file-02-engineering--tech-stack"></a>

<!-- ================================================================== -->
<!-- File: 02-engineering/tech-stack.md -->
<!-- ================================================================== -->

# Tech Stack

The full inventory of technologies Crucible composes. Decisions are pre-made; the ADRs in [05-decisions/](../05-decisions/) document the reasoning.

## Compute & isolation

| Layer | Default (hosted) | Self-hosted | Solo-founder tier |
|---|---|---|---|
| Microvm sandbox | E2B (Firecracker) | Firecracker + containerd | Daytona / Fly Machines |
| Filesystem isolation | overlayfs + git worktrees | overlayfs + ZFS + git worktrees | overlayfs alone |
| Egress enforcement | Cilium + Tetragon | Cilium + Tetragon | mitmproxy allowlist |
| Container runtime | crun (Firecracker-friendly) | crun | Docker / podman |
| Orchestration | Kubernetes (EKS / GKE / AKS) | Kubernetes | docker-compose |

## Data layer

| Component | Pick | Notes |
|---|---|---|
| Twin Postgres | Neon (CoW branching) | $0.002/branch-hr; instant branch via `POST /branches` |
| Twin MySQL | PlanetScale | Branching mature for MySQL |
| Twin SQLite/libSQL | Turso | Instant per-DB branch |
| Twin MongoDB | Atlas snapshot-restore-to-new-cluster | Slower (minutes), acceptable |
| Twin Redis/KV | Fresh `redis-server` inside sandbox | Stateless |
| Twin S3 | MinIO inside sandbox + rclone mirror | |
| Production-side DB (Crucible's own) | Postgres 16 (managed: Neon for SaaS, RDS for self-host) | Hot path for memory + attestations |
| Vector store | pgvector (default), Qdrant (greenfield), Turbopuffer (scale) | Per-tenant isolation |
| Graph store | FalkorDB (default), Neo4j (alt) | Avoid KuzuDB — archived Oct 2025 |
| Hot cache | Redis 7+ | Per-tenant namespaces |
| Queue (distiller / async tasks) | Kafka (high-volume) or AWS SQS (small-team) | |
| Object storage | S3-compatible (AWS S3, R2, MinIO) | |

## Service replay & mocking

| Function | Pick |
|---|---|
| HTTP/gRPC capture-replay | Hoverfly OSS |
| Mock-only (when no recording) | WireMock (JVM stacks), Mockoon (broad) |
| LLM-generated stubs | Microcks AI Copilot pattern |
| Contract testing | Pact (for explicit contracts) |
| OpenAPI mock generation | Stoplight Prism |
| Schema-grounded fakes | json-schema-faker, Faker.js |

## PII scrubbing

| Function | Pick |
|---|---|
| Named-entity recognition | Microsoft Presidio Analyzer + Anonymizer |
| NLP backbone | spaCy 3.x (Presidio default) |
| Format-preserving encryption | FF3-1 via mysto/python-fpe or Vault transform |
| Synthetic data augmentation | Gretel, MOSTLY AI, or SDV (open source) |
| Audit | append-only per-tape scrub log |

## Secrets & signing

| Function | Pick |
|---|---|
| Secrets vault (hosted) | Infisical Cloud |
| Secrets vault (self-host) | Infisical OSS / HashiCorp Vault Community |
| Production-promotion signing | AWS KMS / GCP Cloud HSM / YubiHSM (per deployment) |
| Code signing (sigstore) | Cosign + Sigstore keyless OIDC |
| Transparency log | Sigstore Rekor v2 (public for SaaS; self-hosted for enterprise) |
| Build provenance | in-toto attestations + SLSA-L3 |
| Build attestation tools | GitHub `actions/attest-build-provenance`, Witness, Tekton Chains |

## Verifier toolchain (per language)

### Python
- `hypothesis` 6.152+, `schemathesis` (APIs), `mutmut` 4.x, `atheris`
- `ruff` (lint), `mypy --strict` (types)

### JS / TS
- `fast-check` + `@fast-check/vitest` / `@fast-check/jest`
- `stryker-js` (mutation), `jsfuzz` (fuzz)
- `biome` (lint + format)

### Rust
- `proptest`, `quickcheck`, `cargo-mutants`, `kani`, `cargo-fuzz`, `cargo-afl`
- `rustfmt`, `clippy`

### Go
- `pgregory.net/rapid` (PBT), native `testing.F` (fuzz), `go-mutesting`
- `gofmt`, `golangci-lint`

### Java / Kotlin
- `jqwik` (PBT), `pitest` (mutation), JQF (coverage-guided PBT)

### Swift
- `swift-testing` (Xcode 16+), `muter` (mutation)

### C / C++
- `theft` (PBT), `libFuzzer`, AFL++

### Tier 3 (formal verification)
- **Dafny + DafnyPro** (POPL 2026): general business logic, money paths
- **Lean 4 + mathlib + LeanCopilot**: crypto, math-heavy
- **TLA+ + Apalache**: distributed invariants
- **Kani**: Rust `unsafe` + FFI
- **Z3 v4.15+ / CVC5 v1.2+**: SMT direct queries

### Tier 4 (honest CI)
- **Nix flakes** (default reproducible builds)
- **Bazel** (alternative for Java/Kotlin/C++ shops)
- **Sigstore Cosign** (signing)
- **in-toto** (attestation format)
- **SLSA provenance generator** (`slsa-framework/slsa-github-generator`)

## Memory layer

| Function | Pick |
|---|---|
| Hot cache | Redis 7+ |
| Episodic + semantic store | pgvector (default) / Qdrant / Turbopuffer |
| Procedural graph backend | FalkorDB |
| Procedural graph abstraction | Graphiti (Zep's OSS) atop FalkorDB |
| Extraction algorithm | Mem0's hierarchical extraction (Apache-2.0) |
| Schema-constrained decoding | AdaKGC SDD pattern |
| Embedding model | OpenAI `text-embedding-3-large` (default) / Cohere v3 (EU) / open-weights option |

## LLM routing

| Tier | Model | API |
|---|---|---|
| 0 | `claude-haiku-4-5` | Anthropic Messages API |
| 1 | `claude-sonnet-4-6` | Anthropic Messages API |
| 2 | `claude-opus-4-7` | Anthropic Messages API |
| 2 (alt, terminal-heavy) | `gpt-5.5`, `gpt-5.3-codex` | OpenAI Responses API |
| 2 (alt, algorithmic) | `gemini-3.1-pro` | Gen AI Direct (Vertex) |
| 3 (verifier, default pairing) | cross-family of executor | — |
| 4 (local) | Llama 4 Scout / DeepSeek V4-Pro / Qwen3-Coder-Plus | vLLM / sglang / Ollama |

## Observability

| Function | Pick |
|---|---|
| Tracing | OpenTelemetry → Honeycomb (SaaS) / Tempo (self-host) |
| Metrics | Prometheus + Grafana |
| Logs | Loki (self-host) / Honeycomb structured events (SaaS) |
| Errors | Sentry |
| Cost telemetry | Custom (OTel spans → ClickHouse) |
| Uptime | Crucible's own SLO dashboards backed by Prometheus AnalysisTemplate (eating our dogfood) |

## Progressive delivery

| Function | Pick |
|---|---|
| Canary controller (K8s) | Argo Rollouts |
| Canary controller (service mesh) | Flagger (Linkerd / Istio) |
| Feature flags | GrowthBook (OSS, self-host friendly) |
| Shadow traffic | Hoverfly tape replay against new version |
| Traffic mirroring | Argo Rollouts + service mesh |
| Rollback | GrowthBook flag flip (millisecond) |

## Front-end stack

- **Framework:** Next.js (App Router, RSC default, `use client` boundaries explicit)
- **Component lib:** shadcn/ui + Radix primitives
- **Styling:** Tailwind CSS
- **Form validation:** zod + react-hook-form
- **Charts:** Tremor (dashboards) / Recharts (custom)
- **Realtime:** Server-Sent Events for plan + verifier progress; WebSocket only when bi-directional needed
- **Auth:** Clerk (SaaS) / WorkOS (enterprise SSO + SAML) / Authelia (self-host)
- **Hosting:** Vercel (SaaS) / customer-supplied (self-host)

## Backend services

- **API framework (Go):** `connect-go` (gRPC + HTTP from same handler)
- **API framework (Python, distiller):** FastAPI
- **DB access (Go):** sqlc + pgx
- **DB access (Python):** SQLAlchemy 2.x (typed) + Alembic
- **Migrations:** sqlc generate + Atlas (declarative migrations)
- **Background jobs:** asynq (Go), Celery (Python)

## CLI

- **Language:** Go
- **Framework:** Cobra + Viper
- **TUI:** Bubble Tea (when interactive flows needed)
- **Distribution:** GitHub Releases + Homebrew + Scoop + apt/yum repos

## Why these picks (one-line each)

- **Nix > Docker for reproducible builds.** Bit-identical artifacts are mandatory for Tier 4.
- **Neon > self-hosted Postgres branching.** CoW at storage layer in 1–2s is irreplaceable.
- **FalkorDB > Neo4j.** Lower latency, simpler ops, source-available. Neo4j ecosystem advantage doesn't justify the cost premium for our use.
- **Hoverfly > WireMock.** Hoverfly's capture-replay is first-class; WireMock is mock-first with bolt-on capture.
- **Infisical > Vault.** Modern DX, OSS self-host, real dynamic secrets without enterprise-tier upcharges.
- **Sigstore > custom signing.** Public transparency log, ecosystem momentum, OIDC keyless approach. Customer's compliance team already knows the name.
- **Argo Rollouts > Flagger.** Argo's analysis ecosystem is richer; Flagger is fine if you're already in Linkerd.
- **GrowthBook > LaunchDarkly.** OSS, self-host friendly, no per-MAU tax.
- **Anthropic + Google as primary vendors.** Cross-family verifier requires both; OpenAI is in the routing table but not load-bearing.

## What we explicitly do NOT use

- **Pinecone** (vector DB lock-in, pricey at scale)
- **Milvus** (operational complexity not worth it under 100M vectors)
- **KuzuDB** (archived October 2025)
- **HashiCorp Vault HCP Dedicated for v1** (EOL plans for HCP Vault Secrets July 2026 created uncertainty; Infisical is the safer modern choice)
- **AWS QLDB** (sunset; no clean replacement narrative)
- **LocalStack OSS** (archived March 2026; Pro-only is auth-required)
- **jsverify, gopter** (superseded by fast-check and rapid respectively)
- **Java EE / Spring on Crucible's own backend** (Go and Rust are better fits for our service shape)

## Upgrade path

We track frontier libs weekly. Major version bumps (e.g., `hypothesis 7.0`, `fast-check 4.0`) go through the standard PR + Tier 0–4 verification flow. Customer-impacting changes get a 30-day deprecation window communicated via changelog and console banner.

Model routing tracks vendor pricing and capability changes monthly. The May 2026 reference pricing in [01-architecture/model-routing.md](../01-architecture/model-routing.md) is hardcoded for v1; v2 introduces a model-price oracle.

---

<a id="file-02-engineering--local-dev"></a>

<!-- ================================================================== -->
<!-- File: 02-engineering/local-dev.md -->
<!-- ================================================================== -->

# Local Dev

How to run the Phase-1 Crucible stack locally. For background, read [system-overview.md](../01-architecture/system-overview.md) first.

## Prerequisites

- **Nix** (2.34+). Install via `curl -L https://nixos.org/nix/install | sh`. Flakes must be enabled:
  ```bash
  mkdir -p ~/.config/nix && echo 'experimental-features = nix-command flakes' >> ~/.config/nix/nix.conf
  ```
- An **`ANTHROPIC_API_KEY`**. Without it the control plane runs in heuristic + fallback-plan mode (useful for offline dev, but not the full path).
- Optional: `GOOGLE_API_KEY` (or `GEMINI_API_KEY`) to wire the verifier vendor; `OPENAI_API_KEY` for alternate Tier 1/2 routing.

## Dev shell

```bash
cd "E:\AI Coding Agent"     # Windows path; on Linux/macOS just clone the repo
nix develop                  # all-language shell (Go + Node + Python + Rust + buf + cosign + opa)

# Or per-language:
nix develop .#go-only
nix develop .#node-only
nix develop .#python-only
nix develop .#rust-only
```

On Windows, use **WSL2** for the Nix shell — Nix on native Windows is nascent (see [ADR-013](../05-decisions/ADR-013-nix-for-tier4-builds.md) §Open issues).

## Running the control plane

```bash
export ANTHROPIC_API_KEY=sk-ant-...
nix build .#control-plane
./result/bin/crucible-control-plane
```

You should see:

```
{"level":"INFO","msg":"attestation wired","signer":"...","journal":"~/.crucible/attestations/journal.jsonl"}
{"level":"INFO","msg":"LLM vendors wired","vendors":["anthropic"]}
{"level":"INFO","msg":"control-plane listening","addr":":8080","version":"2026.06.0-phase1"}
```

Smoke-test:

```bash
nix build .#cli
./result/bin/crucible health
./result/bin/crucible task new --description "Add a Stripe refund webhook handler" --repo github.com/acme/payments
./result/bin/crucible task list
./result/bin/crucible plan show <task_id>
./result/bin/crucible plan approve <task_id>
./result/bin/crucible budget show <task_id>
```

## Without Nix (best-effort, non-hermetic)

```bash
# Go 1.23, Node 22, Python 3.12, Rust stable required.
cd apps/control-plane && go build ./...
cd ../cli && go build ./...
./apps/control-plane/control-plane &
./apps/cli/crucible health
```

You'll fail the SLSA-L3 hermetic-rebuild check on this path; use Nix for any artifact you intend to publish.

## Tests

```bash
# Per-module
cd apps/control-plane && go test -short ./...
cd libs/attestation && go test ./...
cd libs/policy && go test ./...
cd libs/sdk-go && go test ./...

# Python SDK
cd libs/sdk-py && pip install -e .[dev] && pytest

# TS SDK
cd libs/sdk-ts && pnpm install && pnpm test

# Rust SDK
cd libs/sdk-rs && cargo test --all-targets

# The real-Haiku-4.5 integration test only runs with the env var set:
ANTHROPIC_API_KEY=sk-ant-... go test -run TestIntegration_RealHaiku4_5 -v ./apps/control-plane/internal/api
```

The budget-enforcer property test (`TestProperty_NeverExceedsCap`) runs 50 seeds × 8 goroutines × 500 ops and is the strongest correctness guarantee in Phase 1. It asserts the ADR-009 invariant: once a cap is breached, the enforcer is frozen and no further mutation succeeds.

## Environment variables the control plane reads

| Var                          | Default                                          | Purpose                                              |
|------------------------------|--------------------------------------------------|------------------------------------------------------|
| `ANTHROPIC_API_KEY`          | unset                                            | Wires the Anthropic vendor (Tier 0/1/2 default)      |
| `GOOGLE_API_KEY`             | unset                                            | Wires Gemini (verifier-default vendor)               |
| `OPENAI_API_KEY`             | unset                                            | Wires OpenAI (alternate Tier 1/2)                    |
| `CRUCIBLE_LISTEN_ADDR`       | `:8080`                                          | HTTP bind address                                    |
| `CRUCIBLE_DEFAULT_TENANT`    | `single-tenant`                                  | Tenant ID when callers omit `tenant_id`              |
| `CRUCIBLE_KEY_DIR`           | `~/.crucible/dev-keys/`                          | Local Ed25519 keypair for attestation signing        |
| `CRUCIBLE_JOURNAL_PATH`      | `~/.crucible/attestations/journal.jsonl`         | Hash-chained attestation journal                     |
| `CRUCIBLE_COSTLOG_DIR`       | `~/.crucible/costlog/`                           | Per-task cost JSONL                                  |
| `CRUCIBLE_WEBHOOK_URL`       | unset                                            | If set, every event POSTs here                       |
| `CRUCIBLE_REKOR_PUBLISH`     | `0`                                              | Gates the Phase-2 real Rekor v2 publisher            |

## What's stubbed in Phase 1

- Twin Runtime (sandbox, Neon, Hoverfly, syscall shim)
- Verifier Pipeline (Tier 0–4 ladder)
- Memory Layer (Redis / pgvector / FalkorDB / Graphiti / distiller)
- Promotion Contract (Argo Rollouts, KMS lease, Slack approvals)
- Real Sigstore Rekor v2 publish (the local journal is the default — Rekor v2 had not GA'd as of May 2026)
- Web console, IDE plugins, GitHub App, Slack bot

See [PHASE-1-REPORT.md](../PHASE-1-REPORT.md) for the full stub inventory and the Phase-2 hand-off prompt.

---

<a id="file-02-engineering--testing-strategy"></a>

<!-- ================================================================== -->
<!-- File: 02-engineering/testing-strategy.md -->
<!-- ================================================================== -->

# Testing Strategy

How Crucible tests Crucible. We use our own verifier ladder to grade our own changes — eating our dogfood is non-negotiable, and the test surface reflects that.

## The five-tier internal test pyramid

Per-component CI runs each tier in order; failures at any tier block merge.

### Tier 0: Unit tests, mutation-tested

Every public function has unit tests. Every PR's changed lines are mutation-tested on diff (not the whole repo).

- **Threshold:** ≥85% mutation score on diff for Go/Rust/TS/Python; ≥75% for Go (mutation tooling weaker).
- **Frameworks:** `mutmut` (Py), `stryker-js` (TS), `cargo-mutants` (Rust), `go-mutesting` (Go), Pitest (Java).
- **Budget:** 30s default, 2 min max. Diff-scoped, parallel.
- **Failure mode:** PR comment with the surviving mutants. The author writes more tests or explains why a mutant is acceptable.

### Tier 1: Property tests + fuzz harness

Non-trivial functions get property tests covering invariants. Fuzz harnesses are required for any function that parses external input.

- **Frameworks:** `hypothesis` (Py), `fast-check` (TS), `proptest` + `cargo fuzz` (Rust), `rapid` + native fuzz (Go).
- **Iteration count:** ≥10,000 cases for property tests on PR. Continuous fuzzing in nightly with corpora retained.
- **Pairing rule:** every property test ships alongside example-based tests. Property tests alone catch ~68% of bugs; combined with examples 81%.
- **Budget:** 5 min PR, 30 min nightly.

### Tier 2: Integration + DST

Cross-service tests against ephemeral infrastructure: Neon branch, fresh Redis, fresh FalkorDB, simulated Hoverfly tapes.

- **Per-PR:** integration tests for the components changed and their direct dependents.
- **Nightly:** full integration suite — every cross-service call exercised.
- **DST:** the twin-runtime and promotion-gate are run inside a deterministic-simulation harness (in-house, TigerBeetle-style) with virtualized clock+disk+net, simulating partitions, restarts, message drops.
- **Antithesis** (when budget allows): full-system DST for the cross-service flows.

### Tier 3: Formal verification for `@critical` paths in our own code

Our own auth, secrets, attestation-signing, KMS-leasing, and policy-evaluation code is annotated `@critical` and verified.

- **promotion-gate Rego evaluation:** formally specified in Dafny; every Rego policy admission has a corresponding Dafny proof obligation.
- **attestation chain validation:** TLA+ spec for the OIDC subject-chaining invariants; Apalache model-checked.
- **KMS credential leasing:** Dafny proof that no lease can be reused.
- **Egress allowlist:** Coq spec (the one Tier-3 tool we use for this specific component; small footprint, well-validated).

When proofs time out: Tier 2.5 fallback — exhaustive PBT + mutation + mandatory CODEOWNER human review.

### Tier 4: Hermetic rebuild verification

Every release artifact (binaries, container images, Helm charts, air-gap bundle) is rebuilt independently from the same source by a second CI runner, and bit-identical hashes are required.

- **Build system:** Nix flakes (default), Bazel (alternative).
- **Provenance:** in-toto attestation signed via Sigstore keyless OIDC; published to Rekor.
- **SLSA level:** Level 3 (hardened GitHub-hosted runners + dual-platform rebuild).
- **Customer-visible:** every customer can verify our releases against the published attestations.

## The Crucible Test Harness (CTH)

A curated suite of test repositories used to validate the agent's behavior end-to-end.

### CTH composition

```
cth/
├── greenfield/             # Brand new repos; agent builds from scratch
│   ├── nextjs-todo/
│   ├── go-grpc-service/
│   ├── django-blog/
│   └── rust-cli/
│
├── feature-add/            # Existing repos; agent adds a feature
│   ├── stripe-webhook-handler/
│   ├── auth-rate-limit/
│   ├── postgres-migration-additive/
│   └── react-form-validation/
│
├── refactor/               # Existing repos; agent refactors
│   ├── extract-service-from-monolith/
│   ├── upgrade-react-17-to-19/
│   ├── replace-moment-with-date-fns/
│   └── consolidate-error-handling/
│
├── critical-path/          # Agent must trigger Tier 3
│   ├── auth-oauth-implementation/
│   ├── billing-refund-engine/
│   ├── distributed-consensus-bug-fix/
│   └── crypto-key-rotation/
│
├── adversarial/            # Designed to trick the agent
│   ├── tape-poisoned-stripe/        # Tape has malicious response
│   ├── prompt-injected-pr-comment/  # Memory poisoning attempt
│   ├── destructive-shell-disguised/ # rm -rf hidden in benign script
│   ├── hallucinated-api-trap/       # Tests pass only with fake API
│   └── sandbox-escape-attempt/      # Red-team sandbox probe
│
└── regression/             # Bugs we've fixed; must stay fixed
    ├── opus-46-loop-bug/
    ├── pocketos-style-wipe-attempt/
    ├── verifier-tier3-timeout-recovery/
    └── memory-cross-tenant-leak-check/
```

### CTH grading

For each test case, the harness records:

- **Correctness:** did the agent produce a verified-passing PR?
- **Cost:** total token spend.
- **Wall-clock:** total task duration.
- **Cache hit rate:** % of input tokens served from cache.
- **Verifier strictness:** did verifier catch a bad change that should be caught?
- **Safety:** did any destructive-op gate fire? Did the agent attempt anything that should be flagged?

Aggregate scores published per-release. Regression in any dimension blocks release.

### Adversarial subset

The adversarial cases are the most important. They're our equivalent of red-team continuous evaluation. Every fix to a real incident or red-team finding becomes a new adversarial case.

## Continuous evaluation against the public benchmarks

We run against the public benchmarks weekly and publish results:

- **SWE-Bench Verified** (`princeton-nlp/SWE-bench`)
- **SWE-Bench Pro** (Scale's harder set)
- **Terminal-Bench 2.0**
- **Aider Polyglot benchmark**
- **LiveCodeBench**
- **BigCodeBench**

These are not our primary KPI (the CTH is), but they let us position credibly against incumbents and detect regressions in the upstream models we route to.

## Property tests for our own SDK contracts

Every typed SDK call has property tests on its contract:

```
property "twin.fs.write always emits a WriteAttestation":
  forall (path, content) where path is valid:
    result = twin.fs.write(path, content)
    assert result is WriteAttestation
    assert result.signed_by_oidc is valid
    assert result.path == path
    assert hash(twin.fs.read(path)) == hash(content)
```

These tests run against the real twin-runtime in CI, not a mock. They catch contract drift that unit tests miss.

## Chaos / fault injection

The twin-runtime is the highest-stakes component. We chaos-test it weekly:

- **Network partition during task:** kill egress proxy mid-step; verify the agent receives a clean error.
- **Sandbox OOM mid-task:** force-OOM the sandbox; verify graceful failure + clean state.
- **Neon branch creation flake:** simulate 10s+ timeout; verify fallback to lite-twin works.
- **Hoverfly tape corruption:** flip random bytes; verify mount-time checksum rejects.
- **Sigstore Rekor unreachable:** verify local journaling continues and back-fills when Rekor returns.
- **KMS slow / unavailable:** verify promotion queues retry-with-backoff cleanly.

## Self-verification

We use Crucible to verify Crucible's own PRs. Every PR to the Crucible monorepo runs through:

1. Our control-plane spawns a twin from our own repo.
2. Our verifier (with cross-family pairing) runs Tier 0–4 on the diff.
3. Our promotion gate evaluates a Rego policy specifically for our internal release process.
4. The PR is merged only if all of the above pass + human approval.

This is the dogfooding test. If our own engineers find Crucible too slow / annoying / wrong to use on our own code, we ship that pain to customers. Don't.

## Customer-side test harnesses

For paying customers, we install a "shadow Crucible" mode where the agent runs against their PRs in shadow (no merge, no promotion), and we compare the agent's verifier verdict against the human reviewer's verdict. Disagreements are the most valuable signal we have for improving the verifier.

Opt-in. Anonymized. The agreement-rate is published in our public eval as a fairness signal ("Crucible agrees with human reviewers ≥X% of the time on N customer repos").

## What we explicitly do NOT test

- **The frontier LLM vendors' models.** They publish their own benchmarks; we treat them as black boxes.
- **Customer-side integrations** beyond the Crucible boundary. We test the contract; the customer tests their use of it.
- **Performance microbenchmarks of every helper function.** We optimize the hot path (twin spawn, memory router, cost meter) and tolerate the rest.
- **UI pixel-level regressions.** Tremor + shadcn are stable; we don't snapshot-test every component.

## Release cadence

- **Weekly releases** of the SaaS control plane.
- **Monthly releases** of the SDK, CLI, IDE plugins.
- **Quarterly releases** of the self-hosted helm chart and air-gap bundle.
- **Continuous releases** of the OSS verifier and tape-scrubber.

Every release has a CHANGELOG entry, Tier 4 attestation, and a public verification command (`crucible verify-release 2026.06.0`).

---

<a id="file-02-engineering--observability"></a>

<!-- ================================================================== -->
<!-- File: 02-engineering/observability.md -->
<!-- ================================================================== -->

# Observability

What we instrument, what we measure, what we dashboard, what we alert on.

## Telemetry contract

Every component emits:

- **OpenTelemetry traces** — spans for every meaningful action, with structured attributes.
- **OpenTelemetry metrics** — RED (rate, errors, duration) on every service surface.
- **Structured logs** — JSON-line format, correlation IDs everywhere.
- **In-toto attestations** — separately, the cryptographic audit trail (see [01-architecture/threat-model.md](../01-architecture/threat-model.md)).

OTel spans are exported to:
- **Honeycomb** (SaaS tier)
- **Tempo + Grafana** (self-hosted tier)

Logs:
- **Honeycomb structured events** (SaaS)
- **Loki + Grafana** (self-hosted)

Metrics:
- **Prometheus + Grafana** in both deployments.

## Span attributes (the contract)

Every span carries:

```
task_id            UUID, present on every action in a task
step_id            UUID, sub-step within a task
tenant_id          per-tenant scoping
repo_id            per-repo scoping
agent_oidc_subject Sigstore keyless identity of the agent
model.vendor       anthropic | google | openai | xai | deepseek | ...
model.id           claude-opus-4-7 | gemini-3.1-pro | ...
model.tier         0 | 1 | 2 | 3 | 4
tokens.input.fresh
tokens.input.cached
tokens.output
cost.usd
verifier.role      executor | verifier
tier_result        only on verifier spans
```

Tracing-by-task-id lets us reconstruct the full lifecycle of any task from submission to promotion.

## The four KPI dashboards

### Dashboard 1: Per-task economics

| Metric | Target | Alert if |
|---|---|---|
| Median task cost | ≤ $1.69 | > $2.50 sustained 1h |
| P95 task cost | ≤ $7.00 | > $12.00 sustained 1h |
| Cache hit rate | ≥ 70% | < 60% sustained 2h |
| Verifier cost as % of total | ≤ 10% | > 20% sustained 4h |
| Median task wall-clock | ≤ 15 min | > 30 min sustained 1h |
| Token usage per active dev/day | ≤ 1.5M | > 3M sustained 1d |

Per-task cost > $2.50 sustained means our routing is broken or cache is missing. Cache hit rate < 60% means we are losing GM. Both are alarms-page-the-team.

### Dashboard 2: Verifier health

| Metric | Target | Alert if |
|---|---|---|
| Tier 0 mutation kill rate | ≥ 85% | < 70% |
| Tier 1 PBT counterexample rate | < 5% (most PRs should pass) | > 15% (verifier too strict?) |
| Tier 3 proof timeout rate | < 10% | > 25% (proofs not converging) |
| Verifier disagreement with human reviewer | < 15% (shadow mode) | > 25% |
| Verifier reject → reflect → pass rate | ≥ 70% | < 50% (executor not learning) |

The disagreement-with-human-reviewer metric is the most important signal for verifier quality. We want it low (verifier matches human judgment) but not zero (some genuine humans disagree on style; that's noise).

### Dashboard 3: Safety / trust

| Metric | Target | Alert if |
|---|---|---|
| Destructive-op gate firings | tracked, not capped | n/a (informational) |
| Twin-scoped destructives (auto-approved) | tracked | n/a |
| Real-scoped destructives requiring approval | tracked | n/a |
| Egress policy violations | 0 | > 0 (immediate page) |
| Sandbox escape attempts | 0 | > 0 (P0 security incident) |
| Sigstore Rekor publish failures | 0 | > 0 (audit trail gap; page) |
| KMS signing failures | 0 | > 0 (P1) |
| Cross-tenant memory access attempts | 0 | > 0 (P0 isolation breach) |

The 0-target metrics are the safety floor. Any non-zero count is a paging event.

### Dashboard 4: Memory / learning

| Metric | Target | Alert if |
|---|---|---|
| Procedural memory writes per tenant per day | tracked | growth stalls (informational) |
| Convention drift detections per tenant per week | tracked | spike > 10x baseline |
| Cross-tenant abstraction graduations per week | tracked | n/a |
| Memory retrieval router p95 latency | < 100ms | > 250ms |
| Memory retrieval token-budget overruns | < 1% of calls | > 5% |

The procedural-memory write rate is a leading indicator of customer engagement. Stalls mean the customer's PR review activity isn't reaching us — likely an integration broken.

## Standard alerts (SaaS tier)

Critical alerts (page on-call immediately):

- Any 0-target safety metric > 0.
- Cache hit rate < 50% for 30 min.
- Median task cost > $5 for 30 min.
- Sigstore Rekor or KMS unreachable > 5 min.
- Twin-runtime spawn failure rate > 2% for 10 min.
- Promotion-gate evaluation latency p95 > 5s for 10 min.

Warning alerts (Slack notify, not page):

- Cache hit rate 50–60% for 1h.
- Median task cost $3–$5 sustained.
- Verifier disagreement-with-human > 20% over 24h.
- Tier 3 proof timeout rate > 20% over 24h.

## Self-hosted alerting

Self-hosted customers receive a default Prometheus alert pack matching the above, parameterized by their tenant config. They wire it to their own PagerDuty/Opsgenie/whatever.

## Cost telemetry storage

OTel spans → Honeycomb (SaaS) for ad-hoc analysis + Honeycomb-Triggers for alerts.

Long-term cost analytics → ClickHouse cluster with daily rollups. Per-tenant per-model per-day token+dollar aggregates retained 13 months for SOC-2 audit + customer billing reconciliation.

## SLOs we publish to customers

```yaml
slo:
  task_completion_within_estimate:
    objective: 90%
    window: 30d
    description: "Tasks complete within the wall-clock and cost estimate shown in the plan."
  
  promotion_canary_success:
    objective: 99.5%
    window: 30d
    description: "Verified promotions pass canary without rollback."
  
  verifier_decision_within_15min:
    objective: 95%
    window: 30d
    description: "Tier 0+1 verification completes within 15 minutes."
  
  control_plane_availability:
    objective: 99.9%
    window: 30d
    description: "Control plane API responsive (excluding planned maintenance)."
  
  attestation_publish_success:
    objective: 99.99%
    window: 30d
    description: "All in-toto attestations successfully published to Rekor."
```

Customers can subscribe to SLO status via our public status page; enterprise tier gets a per-customer dashboard.

## Customer-facing observability

Each tenant's web console exposes:

- **Their own task timeline** — every task they've run, cost, verifier result, promotion outcome.
- **Their own cost dashboard** — per-developer, per-repo, per-day.
- **Their own memory browser** — view active conventions, drifting conventions, supersession history.
- **Their own attestation viewer** — Rekor UUIDs and content of any attestation.
- **Their own SLO dashboard** — relative to our published SLOs.

The web console is part of the product surface, not a separate observability bolt-on.

## What we don't expose externally

- Per-customer-aggregate metrics (cost telemetry, etc.) are internal-only.
- Cross-customer benchmark comparisons are not surfaced.
- Internal verifier disagreement rates are not surfaced (they're noisy and easy to misinterpret).

The exception: a public, transparent quarterly "Crucible Trust Report" with anonymized aggregates — cache hit rate distribution, median task cost, verifier-vs-human agreement, safety incidents (zero, hopefully). This is the brand investment.

## Tooling-stack rationale

- **OpenTelemetry** as the wire protocol → vendor-neutrality. Customers can swap exporters.
- **Honeycomb** for SaaS-tier hot analysis → speed of ad-hoc query is critical; their pricing scales with us.
- **Prometheus + Grafana** for self-host → universal standard; customers already have it.
- **ClickHouse for long-term aggregates** → cheap columnar storage, fast queries on billions of spans.
- **Loki for logs** → cheap, multi-tenant, integrates with Grafana.
- **Sentry for errors** → standard tool; customers know it.

We don't roll our own observability infra. Use the commodity.

---


# 03. SDK

<a id="file-03-sdk--agent-sdk-reference"></a>

<!-- ================================================================== -->
<!-- File: 03-sdk/agent-sdk-reference.md -->
<!-- ================================================================== -->

# Agent SDK Reference

The complete `twin.*` API surface. This is the only path through which an agent interacts with Crucible — there is no "raw exec" escape hatch. Every call emits an in-toto attestation, signed via the agent's keyless OIDC identity, published to Sigstore Rekor.

SDK languages: Go (`sdk-go`), TypeScript (`sdk-ts`), Python (`sdk-py`), Rust (`sdk-rs`). All generated from the same gRPC schema in `libs/twin-spec/`. Examples below are TypeScript flavored; the API shape is identical across languages.

## Common types

```typescript
type Path = string;
type FilePath = string;
type Glob = string;
type Diff = { files: FileChange[] };
type FileChange = { path: FilePath; action: "add" | "modify" | "delete"; content?: string };

type Attestation = {
  uuid: RekorUUID;
  predicate_type: string;
  subject: string;        // OIDC subject of signer
  signed_at: timestamp;
};

type Budget = {
  spent_usd: number;
  cap_usd: number;
  steps_used: number;
  steps_cap: number;
  wall_clock_used_seconds: number;
  wall_clock_cap_seconds: number;
};

type SourceRef =
  | { kind: "pr_comment"; pr: number; comment_id: string }
  | { kind: "incident"; id: string; service: string }
  | { kind: "adr"; path: string; commit: string }
  | { kind: "agent_observation"; task_id: string; step_id: string };

type Scope = { repo?: string; file_glob?: Glob; category?: string } | "all";
```

## Filesystem (`twin.fs`)

Reads and writes against the twin's overlayfs upper layer. The lower layer (the actual repo) is read-only; agent writes go to the upper.

```typescript
twin.fs.read(path: FilePath): Promise<FileContent>
```
Returns the file's current content from the twin. If the file hasn't been written in this task, returns the version from base_sha.

```typescript
twin.fs.write(path: FilePath, content: string): Promise<WriteAttestation>
```
Writes to the overlayfs upper. Returns an attestation. Never throws on "file doesn't exist" — writes are creates.

```typescript
twin.fs.delete(path: FilePath): Promise<DestructiveProposal | DeleteAttestation>
```
File deletion is a destructive op even in the twin. Returns `DestructiveProposal` if the file is in the critical-path set (auto-approved for twin scope); returns `DeleteAttestation` if approved.

```typescript
twin.fs.list(glob: Glob): Promise<FilePath[]>
```
Lists files matching the glob, merged view of overlayfs upper + lower.

```typescript
twin.fs.diff(): Promise<Diff>
```
Returns the cumulative diff of all writes in this task vs base_sha. Used at task end to construct the PromotionBundle.

## Database (`twin.db`)

Operates against the Neon CoW branch (or per-engine equivalent) provisioned for the task.

```typescript
twin.db.query(sql: string): Promise<QueryResult>
twin.db.queryParametrized(sql: string, params: any[]): Promise<QueryResult>
```
Executes against the twin DB. Returns rows + column metadata. Idempotent in twin: re-running the same query gives the same result until other writes occur.

```typescript
twin.db.migrate(file: FilePath): Promise<MigrationProposal | MigrationAttestation>
```
Applies a migration file to the twin branch. Returns a `MigrationProposal` containing schema diff + DML impact for the verifier to evaluate. Approved migrations return `MigrationAttestation`.

```typescript
twin.db.schemaDiff(): Promise<SchemaDiff>
```
Returns schema diff vs base. Useful for verifier checks ("did this migration touch a critical table?").

## Services (`twin.svc`)

Operates against Hoverfly replay tapes or, if explicitly allowed in the task manifest, live services through a scrubbing egress proxy.

```typescript
twin.svc.call(
  service: string,
  endpoint: string,
  payload?: any,
  options?: { method?: string; headers?: Headers }
): Promise<Response>
```
Returns a response. The response carries `X-Crucible-Tape: hit-exact | hit-template | synth-readonly | synth-mutation | live-passthrough | miss-blocked` so the agent can reason about response trustworthiness.

```typescript
twin.svc.listAvailable(): Promise<ServiceManifest[]>
```
Lists services configured for this task with their tape coverage, OpenAPI spec, and allowed methods.

```typescript
twin.svc.recordOnce(service: string, endpoint: string): Promise<RecordResult>
```
For development: capture a single live call to a service and add it to the tape. Requires the task manifest to allow this service.

## Secrets (`twin.secret`)

Accesses the twin-scoped Infisical vault. Values are never returned to the agent process; they're injected at the egress proxy when a request uses the secret.

```typescript
twin.secret.get(name: string): Promise<SecretRef>
```
Returns a typed handle, not the value. The handle is consumed by `twin.svc.call` via the `Authorization: $Bearer $secret(name)$` placeholder syntax.

```typescript
twin.secret.list(): Promise<string[]>
```
Lists names available in the twin's vault scope.

The agent **cannot** retrieve a secret's raw value. Attempting to does throw `SecretAccessDenied`. This is the architectural enforcement of secrets isolation.

## Shell (`twin.shell`)

Wrapped via the syscall shim. Destructive commands are intercepted and converted to typed `DestructiveProposal`.

```typescript
twin.shell.exec(cmd: string, options?: { cwd?: Path; env?: Record<string,string>; timeoutSec?: number }):
  Promise<ExecResult | DestructiveProposal>
```
Runs a command in the sandbox. Returns either:

- `ExecResult { stdout, stderr, exitCode, durationMs, signed_attestation }`, or
- `DestructiveProposal { command, blast_radius, justification_required: true }` if the command matches destructive patterns.

The agent must explicitly approve a `DestructiveProposal` to proceed:

```typescript
twin.shell.approveDestructive(proposal: DestructiveProposal, justification: string):
  Promise<ExecResult>
```

For twin-scoped destructives (operating on twin DB, twin FS, twin tapes), this auto-approves on the gate's side after the agent provides justification. Real-scoped destructives require human approval through the Promotion Contract.

## Tests (`twin.test`)

```typescript
twin.test.run(suite?: string, options?: { pattern?: string; timeout?: number }):
  Promise<TestReport>
```
Runs the project's test suite (or a subset). Auto-detects the test framework from the repo. Returns structured pass/fail per test + coverage if available.

```typescript
twin.test.runMutation(diff: Diff): Promise<MutationReport>
```
Runs mutation testing on the diff. Returns mutation score, killed mutants, survived mutants.

```typescript
twin.test.runProperty(spec: PBTSpec): Promise<PBTReport>
```
Runs property-based tests. `spec` declares the function under test, the input generators, and the invariants.

```typescript
twin.test.runFuzz(target: string, options?: { iterations?: number; corpus?: Path }):
  Promise<FuzzReport>
```
Runs the project's fuzz harness for `iterations` iterations.

## Verifier (`twin.verify`)

Invoked at the end of a task to compute the verifier's verdict for the Promotion Bundle. Each method delegates to the separate verifier process (different model family).

```typescript
twin.verify.tier0(diff: Diff): Promise<MutationReport>
twin.verify.tier1(spec: PBTSpec): Promise<PBTReport>
twin.verify.tier2(spec: ContractSpec): Promise<ContractReport>
twin.verify.tier3(spec: FormalSpec): Promise<ProofReport>
twin.verify.tier4(): Promise<HonestCIReport>
twin.verify.bundle(): Promise<VerifierApproval | VerifierRejection>
```

`twin.verify.bundle()` is the orchestration call — it runs the appropriate tiers based on the task's critical-path classification, and returns the final verdict for use in `twin.promote(...)`.

## Memory (`twin.memory`)

```typescript
twin.memory.recall(query: string, scope?: Scope): Promise<Memory[]>
```
Multi-signal retrieval against hot + episodic + procedural stores. Returns up to 7K tokens of relevant memory, importance-ranked.

```typescript
twin.memory.note(fact: string, source: SourceRef): Promise<MemoryId>
```
Explicit save — used when the agent learns something the background distiller would miss (e.g., a user correction in the current task).

```typescript
twin.memory.conventions(scope: Scope): Promise<Convention[]>
```
Returns active conventions for the given scope. Used at plan time and during verifier compliance check.

```typescript
twin.memory.checkCompliance(diff: Diff): Promise<ComplianceReport>
```
Compares a diff against the active conventions and returns violations. Called by the verifier during Tier 1+.

## Plan + budget (`twin.plan`)

```typescript
twin.plan.propose(plan: Plan): Promise<PlanApproval | PlanRejection>
```
Submits a plan for user approval. Blocks until user approves, rejects, or edits. Plan structure:

```typescript
type Plan = {
  description: string;
  steps: Step[];
  estimated_cost_usd: number;
  estimated_duration_min: number;
  files_to_touch: FilePath[];
  db_migrations: number;
  external_effects: ExternalEffect[];
  top_risks: Risk[];
  retry_budget_per_step: number;       // default 3
  wall_clock_budget_min: number;
};
```

```typescript
twin.plan.checkBudget(): Promise<Budget>
```
Returns current consumption vs cap. Agent should call this between steps; the Bounded Budget Enforcer also halts execution automatically if exceeded.

```typescript
twin.plan.checkpoint(name: string): Promise<Snapshot>
```
Saves a checkpoint of the twin state. The user can fork from any checkpoint via the web console.

```typescript
twin.plan.requestReplan(reason: string): Promise<PlanApproval>
```
Used when the agent realizes its plan is wrong. Halts current execution, surfaces to user, awaits new plan approval.

## Promotion (`twin.promote`)

```typescript
twin.promote(bundle: PromotionBundle): Promise<PromotionId>
```
Submits a verified `PromotionBundle` to the Promotion Contract. Blocks until policy evaluation completes; if human approval is required, the agent's call returns and the human is notified out-of-band. The agent can poll:

```typescript
twin.promote.status(id: PromotionId): Promise<PromotionStatus>
```
Status progresses: `pending_policy` → `pending_approval` → `approved` → `deploying` → `canary_dwell` → `landed` | `rolled_back` | `rejected`.

## Attestation (`twin.attest`)

```typescript
twin.attest(action: string, metadata?: any): Promise<RekorEntry>
```
Explicit attestation for actions the SDK doesn't auto-attest. Most uses are covered by auto-attestation in other methods; this is the escape hatch.

```typescript
twin.attest.verify(uuid: RekorUUID): Promise<AttestationContent>
```
Verifies and fetches an attestation. Used by verifier and promotion-gate.

## Error contract

Every method can throw structured errors:

```typescript
class CrucibleError extends Error {
  code: ErrorCode;
  retryable: boolean;
  hint?: string;
}

enum ErrorCode {
  BudgetExceeded,
  RetryLimitExceeded,
  WallClockExceeded,
  EgressDenied,
  SecretAccessDenied,
  DestructiveProposalRejected,
  TwinSetupError,
  TapeIntegrityError,
  VerifierRejection,
  PromotionPolicyDenied,
  ApprovalTimeout,
  CanaryRollback,
  TenantQuotaExceeded,
  ModelRoutingDenied,
  // ...
}
```

Agents should handle errors by class:

- **Retryable** (network blips, transient model errors): retry with backoff.
- **Budget / quota**: halt and surface to user via `requestReplan`.
- **Denial** (security policy, destructive proposals): pivot strategy; do not retry the same.

## Lifecycle example

```typescript
const plan = await twin.plan.propose({
  description: "Add Stripe webhook handler for refund events",
  steps: [
    "Read existing webhook handler structure",
    "Author handler + idempotency key check",
    "Author migration for refunds table",
    "Author tests + property tests",
    "Run verifier",
  ],
  estimated_cost_usd: 1.20,
  estimated_duration_min: 8,
  files_to_touch: ["api/webhooks.ts", "db/migrations/20260515_refunds.sql", "test/webhooks.test.ts"],
  db_migrations: 1,
  external_effects: [{ service: "stripe", endpoints: ["/webhooks/refund"], live: false }],
  top_risks: [
    { description: "Webhook signature verification", impact: "high" },
    { description: "Idempotency key collision", impact: "medium" },
  ],
  retry_budget_per_step: 3,
  wall_clock_budget_min: 15,
});

if (plan.status !== "approved") return;

// Execute step 1...
const existing = await twin.fs.read("api/webhooks.ts");
const conventions = await twin.memory.conventions({ file_glob: "api/**/*.ts" });

// Execute step 2...
const newCode = generate(existing, conventions);
await twin.fs.write("api/webhooks.ts", newCode);

// Execute step 3...
await twin.db.migrate("db/migrations/20260515_refunds.sql");

// Execute step 4...
await twin.fs.write("test/webhooks.test.ts", testCode);
const testReport = await twin.test.run();
if (!testReport.passed) {
  // reflect, fix, retry (Bounded Budget Enforcer caps at 3 retries)
}

// Verify...
const verdict = await twin.verify.bundle();
if (verdict.kind !== "approved") {
  // surface rejection_reasons to the agent's reasoning, retry once
}

// Promote...
const bundle = await constructBundle(verdict);
const promotionId = await twin.promote(bundle);

// Wait or return...
const status = await twin.promote.status(promotionId);
```

## What's deliberately not in the SDK

- **A "raw exec" method** that bypasses the syscall shim. There is no such thing.
- **A way to access real production credentials.** The agent process is architecturally unable to.
- **A way to disable the verifier.** Tasks complete only with a verifier verdict.
- **A way to bypass attestation.** Every method auto-emits.
- **Long-lived state (beyond the task).** Persistence is the Memory Layer's job, accessed through `twin.memory.*`.

See [03-sdk/tool-reference.md](tool-reference.md) for the MCP tool definitions exposed to LLM-tool-calling agents, [03-sdk/event-spec.md](event-spec.md) for webhook payloads, and [03-sdk/attestation-formats.md](attestation-formats.md) for the in-toto schemas.

---

<a id="file-03-sdk--tool-reference"></a>

<!-- ================================================================== -->
<!-- File: 03-sdk/tool-reference.md -->
<!-- ================================================================== -->

# MCP Tool Reference

Crucible exposes its agent SDK as MCP tools (and Agent Client Protocol tools where applicable), so any MCP-compatible agent host can drive a Crucible task.

The MCP server is `crucible-mcp`. The tool definitions below are surfaced to the host LLM via the MCP protocol.

## Tool surface

### File operations

#### `twin_fs_read`

```json
{
  "name": "twin_fs_read",
  "description": "Read a file from the twin filesystem (overlayfs upper merged with base SHA).",
  "input_schema": {
    "type": "object",
    "properties": {
      "path": { "type": "string", "description": "File path relative to repo root" }
    },
    "required": ["path"]
  }
}
```

#### `twin_fs_write`

```json
{
  "name": "twin_fs_write",
  "description": "Write content to a file in the twin. Creates if missing. Returns a signed write attestation.",
  "input_schema": {
    "type": "object",
    "properties": {
      "path": { "type": "string" },
      "content": { "type": "string" }
    },
    "required": ["path", "content"]
  }
}
```

#### `twin_fs_delete`, `twin_fs_list`, `twin_fs_diff`

Schemas mirror the SDK methods in [agent-sdk-reference.md](agent-sdk-reference.md).

### Database operations

#### `twin_db_query`

```json
{
  "name": "twin_db_query",
  "description": "Execute SQL against the twin's Neon branch. Returns rows + column metadata. Use parametrized queries.",
  "input_schema": {
    "type": "object",
    "properties": {
      "sql": { "type": "string" },
      "params": { "type": "array", "items": {} }
    },
    "required": ["sql"]
  }
}
```

#### `twin_db_migrate`

```json
{
  "name": "twin_db_migrate",
  "description": "Apply a migration file to the twin DB. Returns a MigrationProposal (twin-scoped, auto-approved for twin) or rejection with schema-diff impact analysis.",
  "input_schema": {
    "type": "object",
    "properties": {
      "file": { "type": "string", "description": "Path to migration file in the repo" }
    },
    "required": ["file"]
  }
}
```

### Service operations

#### `twin_svc_call`

```json
{
  "name": "twin_svc_call",
  "description": "Call an external service. Returns a response carrying X-Crucible-Tape header indicating whether the response was replayed from tape, synthesized from schema, or live. Mutating calls without live-allowed go to deterministic stubs.",
  "input_schema": {
    "type": "object",
    "properties": {
      "service": { "type": "string", "description": "Service name configured in task manifest" },
      "endpoint": { "type": "string", "description": "Path + query, e.g. '/v1/charges'" },
      "method": { "type": "string", "enum": ["GET","POST","PUT","PATCH","DELETE","OPTIONS","HEAD"] },
      "payload": {}, 
      "headers": { "type": "object" }
    },
    "required": ["service", "endpoint"]
  }
}
```

### Secret access

#### `twin_secret_use`

```json
{
  "name": "twin_secret_use",
  "description": "Reference a secret by name in a service call. The value is never returned to the agent; it is injected at the egress proxy when the call fires.",
  "input_schema": {
    "type": "object",
    "properties": {
      "name": { "type": "string" }
    },
    "required": ["name"]
  }
}
```

Note: there is intentionally no `twin_secret_get` MCP tool. The SDK has `twin.secret.get` returning an opaque `SecretRef`, but the MCP surface goes one step further — agents simply reference secrets by name and the substitution happens server-side.

### Shell

#### `twin_shell_exec`

```json
{
  "name": "twin_shell_exec",
  "description": "Run a shell command inside the twin sandbox. Destructive commands are intercepted and require explicit approval. Returns ExecResult or DestructiveProposal.",
  "input_schema": {
    "type": "object",
    "properties": {
      "cmd": { "type": "string" },
      "cwd": { "type": "string" },
      "env": { "type": "object" },
      "timeout_sec": { "type": "integer" }
    },
    "required": ["cmd"]
  }
}
```

#### `twin_shell_approve_destructive`

```json
{
  "name": "twin_shell_approve_destructive",
  "description": "Approve a DestructiveProposal from a prior twin_shell_exec call. Twin-scoped destructives auto-execute; real-scoped require human approval via the Promotion Contract.",
  "input_schema": {
    "type": "object",
    "properties": {
      "proposal_id": { "type": "string" },
      "justification": { "type": "string" }
    },
    "required": ["proposal_id", "justification"]
  }
}
```

### Tests

#### `twin_test_run`, `twin_test_run_mutation`, `twin_test_run_property`, `twin_test_run_fuzz`

Same shape as SDK methods.

### Verifier

#### `twin_verify_bundle`

```json
{
  "name": "twin_verify_bundle",
  "description": "Run the appropriate verifier tier ladder on the current task state. Returns VerifierApproval or VerifierRejection with structured failure reasons.",
  "input_schema": { "type": "object", "properties": {} }
}
```

The host LLM typically calls this once near the end of a task. Tier selection is automatic based on the critical-path classifier; agents may explicitly invoke `twin_verify_tier3` to escalate.

### Memory

#### `twin_memory_recall`

```json
{
  "name": "twin_memory_recall",
  "description": "Retrieve relevant memory (conventions, prior decisions, code snippets) for a query. Returns up to 7K tokens, importance-ranked.",
  "input_schema": {
    "type": "object",
    "properties": {
      "query": { "type": "string" },
      "scope": {
        "type": "object",
        "properties": {
          "repo": { "type": "string" },
          "file_glob": { "type": "string" },
          "category": { "type": "string" }
        }
      }
    },
    "required": ["query"]
  }
}
```

#### `twin_memory_note`, `twin_memory_conventions`, `twin_memory_check_compliance`

Schemas mirror the SDK.

### Plan + promotion

#### `twin_plan_propose`

```json
{
  "name": "twin_plan_propose",
  "description": "Submit a plan for user approval. Plan must include cost estimate, file impact, top risks, and retry budget. Blocks until user approves, edits, or rejects.",
  "input_schema": { "$ref": "schemas/Plan.json" }
}
```

#### `twin_plan_check_budget`

Returns current spend vs cap.

#### `twin_promote`

```json
{
  "name": "twin_promote",
  "description": "Submit a verified PromotionBundle to the Promotion Contract. Returns PromotionId. Status is queryable via twin_promote_status.",
  "input_schema": { "$ref": "schemas/PromotionBundle.json" }
}
```

## ACP (Agent Client Protocol) compatibility

For Zed and any other ACP-compatible editor, Crucible exposes the same tool set via the ACP `tools/list` and `tools/call` methods. The schemas are identical to MCP. This lets a Zed user run Crucible as their primary agent backend without writing a custom adapter.

## Authentication

MCP host authenticates to Crucible via either:

- **OAuth 2.0 + PKCE** (default for IDE plugins and CLI)
- **API token** (for CI / scripts; tenant-scoped, revocable)
- **Sigstore keyless OIDC** (for GitHub Actions, etc., where the runner has an OIDC token)

Tokens are bound to a specific tenant + workspace. Agents cannot access other tenants' state.

## Permissions model

Each tool call goes through a per-tenant authorization check. Defaults:

| Tool group | Default authorization |
|---|---|
| `twin_fs_*`, `twin_memory_recall`, `twin_plan_check_budget` | Always allowed |
| `twin_db_*`, `twin_svc_call`, `twin_test_*`, `twin_verify_*` | Allowed if task is in `active` state |
| `twin_shell_exec` | Allowed; destructive ops require approve flow |
| `twin_shell_approve_destructive` (real-scoped) | Requires human signature via Promotion Contract |
| `twin_memory_note` | Allowed; subject to LLM-judge filter |
| `twin_plan_propose` | Allowed; blocks for approval |
| `twin_promote` | Allowed; Promotion Contract evaluates policy |

Tenants can lock down further (e.g., disable `twin_shell_exec` entirely for repos that don't need it; cap `twin_db_migrate` to specific paths).

## Versioning

Tool schemas are versioned via the MCP `meta.version` field. Currently `2026.06`. Breaking changes bump the schema version + 90-day deprecation window. Hosts that don't advertise the new version receive the old schema.

## Discovery

Hosts call `tools/list` to discover available tools per their authorization. The list is filtered by tenant capabilities (e.g., `twin_verify_tier3` only appears if the tenant's tier includes formal verification).

## Examples

See the `examples/` directory in the repo:

- `examples/cursor-mcp/` — Cursor as the MCP host driving Crucible
- `examples/claude-desktop/` — Claude Desktop as the host
- `examples/zed-acp/` — Zed via ACP
- `examples/github-actions/` — CI-driven agent flow
- `examples/cli-direct/` — `crucible` CLI as the host

---

<a id="file-03-sdk--event-spec"></a>

<!-- ================================================================== -->
<!-- File: 03-sdk/event-spec.md -->
<!-- ================================================================== -->

# Event Spec

Webhook event payloads emitted by Crucible to customer-configured endpoints. Subscribe via the web console or REST API at `/v1/webhooks/subscriptions`.

## Delivery semantics

- **At-least-once delivery.** Idempotency keys are included; receivers should dedupe.
- **Signed payloads.** Every webhook carries `X-Crucible-Signature` (HMAC-SHA256 of payload using the subscription's signing secret) + `X-Crucible-Sigstore-Bundle` (Sigstore keyless attestation for high-stakes events).
- **JSON content-type, UTF-8.**
- **Retry policy:** 5 attempts with exponential backoff (1s, 4s, 16s, 64s, 256s). After exhaustion, the event lands in a dead-letter queue visible in the web console.

## Event types

### `task.submitted`

```json
{
  "event_id": "evt_01HZ...",
  "event_type": "task.submitted",
  "occurred_at": "2026-05-15T18:24:31Z",
  "tenant_id": "ten_...",
  "task": {
    "id": "task_01HZ...",
    "description": "Add Stripe webhook handler for refund events",
    "submitted_by": "user_...",
    "submitted_from": "cursor-mcp",
    "repo": "github.com/acme/payments",
    "base_sha": "abcd1234..."
  }
}
```

### `task.plan_proposed`

```json
{
  "event_id": "evt_01HZ...",
  "event_type": "task.plan_proposed",
  "occurred_at": "...",
  "tenant_id": "ten_...",
  "task_id": "task_01HZ...",
  "plan": {
    "description": "...",
    "estimated_cost_usd": 1.20,
    "estimated_duration_min": 8,
    "files_to_touch": ["api/webhooks.ts", "..."],
    "db_migrations": 1,
    "external_effects": [{"service":"stripe","endpoints":["/webhooks/refund"],"live":false}],
    "top_risks": [{"description":"...","impact":"high"}],
    "retry_budget_per_step": 3,
    "wall_clock_budget_min": 15
  },
  "approval_url": "https://app.crucible.dev/tasks/.../plan"
}
```

### `task.plan_approved` / `task.plan_rejected`

```json
{
  "event_type": "task.plan_approved",
  "task_id": "task_01HZ...",
  "approved_by": "user_...",
  "approved_at": "..."
}
```

### `task.step_started` / `task.step_completed`

Granular per-step. Useful for streaming progress UIs.

```json
{
  "event_type": "task.step_completed",
  "task_id": "task_01HZ...",
  "step_id": "step_03",
  "step_name": "Author handler + idempotency key check",
  "duration_seconds": 47.3,
  "cost_usd": 0.31,
  "files_changed": ["api/webhooks.ts"]
}
```

### `task.budget_warning` / `task.budget_exceeded`

```json
{
  "event_type": "task.budget_exceeded",
  "task_id": "task_01HZ...",
  "spent_usd": 2.04,
  "cap_usd": 2.00,
  "halted": true,
  "next_action": "user_replan_required"
}
```

### `task.destructive_proposal`

Fires whenever the syscall shim intercepts a destructive command. For twin-scoped proposals, this is informational. For real-scoped, the customer's approval flow is triggered.

```json
{
  "event_type": "task.destructive_proposal",
  "task_id": "task_01HZ...",
  "proposal": {
    "id": "prop_...",
    "command": "DROP TABLE users_archived",
    "scope": "twin",
    "justification": "agent: cleaning up unused archive table",
    "blast_radius": {
      "affected_resources": ["table:users_archived"],
      "reversibility": "snapshot",
      "impact_score": 0.4
    }
  },
  "approval_required": false
}
```

### `task.verification_started` / `task.verification_completed`

```json
{
  "event_type": "task.verification_completed",
  "task_id": "task_01HZ...",
  "verdict": "approved",
  "rubric_score": 0.92,
  "tier_results": {
    "tier_0": {"passed": true, "mutation_score": 0.91},
    "tier_1": {"passed": true, "pbt_iterations": 10000, "counterexamples": []},
    "tier_4": {"passed": true, "rebuild_hash": "...", "rekor_uuid": "..."}
  },
  "rejection_reasons": [],
  "attestations": ["rekor:..."],
  "signed_by_oidc": "https://accounts.crucible.dev/agents/...",
  "signed_at": "..."
}
```

### `task.promotion_proposed` / `.approved` / `.rejected` / `.deploying` / `.canary_dwell` / `.landed` / `.rolled_back`

The full lifecycle of a promotion. The Slack approval bot uses `.promotion_proposed` to render the approve/reject button.

```json
{
  "event_type": "task.promotion_landed",
  "task_id": "task_01HZ...",
  "promotion_id": "prom_...",
  "rollout_strategy": "canary",
  "canary_steps": [
    {"weight": 1, "dwell_seconds": 300, "slo_check": "passed"},
    {"weight": 5, "dwell_seconds": 600, "slo_check": "passed"},
    {"weight": 25, "dwell_seconds": 1800, "slo_check": "passed"},
    {"weight": 100, "dwell_seconds": 0, "slo_check": "passed"}
  ],
  "final_attestation": "rekor:..."
}
```

### `task.completed` / `task.failed` / `task.cancelled`

Final event for any task. `task.completed` indicates both verification and (if applicable) promotion succeeded.

```json
{
  "event_type": "task.completed",
  "task_id": "task_01HZ...",
  "outcome": "verified_and_promoted",
  "total_cost_usd": 1.69,
  "total_duration_min": 12.4,
  "files_changed": [...],
  "pr_url": "https://github.com/acme/payments/pull/...",
  "rekor_attestations": ["rekor:...", "..."]
}
```

### `memory.convention_drift_detected`

Fired by the distillation worker when an active convention's positive/negative ratio drops below threshold.

```json
{
  "event_type": "memory.convention_drift_detected",
  "tenant_id": "ten_...",
  "convention_id": "conv_...",
  "rule_nl": "API errors return { error: { code, message } } envelope",
  "scope": {"file_glob":"api/**/*.ts"},
  "positive_ratio_30d": 1.2,
  "threshold": 1.5,
  "suggested_action": "user_confirm_or_supersede",
  "console_url": "https://app.crucible.dev/memory/conventions/conv_..."
}
```

### `memory.convention_learned`

Fired when a candidate convention graduates to active.

```json
{
  "event_type": "memory.convention_learned",
  "tenant_id": "ten_...",
  "convention_id": "conv_...",
  "rule_nl": "PRs that touch billing/ require @payments-leads approval",
  "category": "PR/commit hygiene",
  "confidence": 0.82,
  "source_evidence": [{"kind":"pr_comment","pr":1234,"comment_id":"..."},"..."],
  "now_active": true
}
```

### `security.egress_violation` (P0)

```json
{
  "event_type": "security.egress_violation",
  "tenant_id": "ten_...",
  "task_id": "task_01HZ...",
  "attempted_host": "evil.example.com:443",
  "blocked_at_layer": "tetragon",
  "agent_process": "...",
  "killed": true,
  "incident_id": "inc_..."
}
```

### `security.cross_tenant_access_attempt` (P0)

The most severe internal event. Always alerts on-call.

### `system.sigstore_unreachable` (P1)

Attestation publish failed. Local journaling continues; back-fill is automatic.

### `system.kms_signing_failure` (P1)

Promotion blocked.

## Subscription configuration

Webhook subscriptions are managed at the tenant level:

```bash
crucible webhook create \
  --url https://hooks.acme.com/crucible \
  --events 'task.*,memory.convention_drift_detected,security.*' \
  --signing-secret-from $CRUCIBLE_HOOK_SECRET \
  --description "Production webhooks"
```

Or via REST:

```
POST /v1/webhooks/subscriptions
{
  "url": "https://hooks.acme.com/crucible",
  "events": ["task.*", "memory.convention_drift_detected", "security.*"],
  "signing_secret": "<from-vault>",
  "active": true
}
```

Event-name globs are supported. `*` matches one segment; `**` matches all (`task.**` = every task event).

## Signature verification

```python
import hmac, hashlib

def verify(payload_bytes, signing_secret, header_signature):
    expected = hmac.new(
      signing_secret.encode(),
      payload_bytes,
      hashlib.sha256
    ).hexdigest()
    return hmac.compare_digest(expected, header_signature)
```

For high-stakes events (`promotion.*`, `security.*`), additionally verify the `X-Crucible-Sigstore-Bundle` against Sigstore's trust root.

## Rate limits

- Per-subscription: 100 events/sec sustained; 1000 events/sec burst.
- Beyond burst: events queue; if queue exceeds 10K events, oldest dropped + `system.webhook_queue_overflow` event fired.

## Replay

The web console exposes a "redeliver event" button per event. Bulk replay via API:

```
POST /v1/webhooks/subscriptions/{id}/redeliver
{ "event_ids": ["evt_...", "evt_..."] }
```

Useful for catch-up after a receiver outage.

---

<a id="file-03-sdk--attestation-formats"></a>

<!-- ================================================================== -->
<!-- File: 03-sdk/attestation-formats.md -->
<!-- ================================================================== -->

# Attestation Formats

Every action Crucible takes is captured as an in-toto attestation, signed via Sigstore keyless OIDC, and published to Sigstore Rekor v2. This doc is the schema reference.

## What is an attestation, in our context

```
┌─────────────────────────────────────────────────────────────┐
│  in-toto statement                                          │
│  ┌────────────────────────────────────────────────────────┐ │
│  │  _type:        "https://in-toto.io/Statement/v1"       │ │
│  │  subject:      [{ name, digest }, ...]                 │ │
│  │  predicateType: "https://crucible.dev/<type>/v1"       │ │
│  │  predicate:    <typed payload>                         │ │
│  └────────────────────────────────────────────────────────┘ │
│  signed via DSSE envelope, OIDC subject = agent worker ID   │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
                    Sigstore Rekor v2 entry
                    (returns rekor:uuid, public)
```

The `predicate` is the payload. The `subject` references the thing being attested (file, diff, build, decision). Crucible defines several predicate types, all under the `https://crucible.dev/` namespace.

## Predicate types

### `https://crucible.dev/WriteAttestation/v1`

Emitted on every `twin.fs.write` and `twin.fs.delete`.

```json
{
  "task_id": "task_01HZ...",
  "step_id": "step_03",
  "tenant_id": "ten_...",
  "repo": "github.com/acme/payments",
  "base_sha": "abcd1234...",
  "path": "api/webhooks.ts",
  "action": "modify",
  "content_sha256": "0xabc...",
  "size_bytes": 4823,
  "timestamp": "2026-05-15T18:24:31.012Z",
  "agent_oidc_subject": "https://accounts.crucible.dev/agents/..."
}
```

### `https://crucible.dev/MigrationAttestation/v1`

Emitted on `twin.db.migrate`.

```json
{
  "task_id": "...",
  "tenant_id": "...",
  "migration_file": "db/migrations/20260515_refunds.sql",
  "migration_sha256": "...",
  "schema_diff": {
    "added_tables": ["refunds"],
    "modified_tables": [],
    "dropped_tables": [],
    "added_columns": [],
    "destructive_ddl": false
  },
  "row_count_change": {"refunds": "+0"},
  "applied_at": "...",
  "neon_branch_id": "br_...",
  "agent_oidc_subject": "..."
}
```

### `https://crucible.dev/ServiceCallAttestation/v1`

Emitted on every `twin.svc.call`.

```json
{
  "task_id": "...",
  "tenant_id": "...",
  "service": "stripe",
  "endpoint": "/v1/charges",
  "method": "GET",
  "request_hash": "0x...",
  "response_hash": "0x...",
  "tape_disposition": "hit-exact",
  "x_crucible_tape": "hit-exact",
  "duration_ms": 12,
  "secrets_used": ["stripe_test_key"],
  "agent_oidc_subject": "..."
}
```

### `https://crucible.dev/DestructiveProposal/v1`

Emitted whenever the syscall shim intercepts a destructive command.

```json
{
  "task_id": "...",
  "tenant_id": "...",
  "command": "DROP TABLE users_archived",
  "scope": "twin",
  "justification": "cleaning up unused archive table",
  "blast_radius": {
    "affected_resources": ["table:users_archived"],
    "reversibility": "snapshot",
    "impact_score": 0.4
  },
  "intercepted_at_layer": "syscall-shim",
  "agent_oidc_subject": "..."
}
```

### `https://crucible.dev/DestructiveApproval/v1`

Emitted when a destructive proposal is approved (twin-scoped auto-approval or real-scoped human approval).

```json
{
  "proposal_attestation": "rekor:...",
  "approval_kind": "auto-twin" | "human-real",
  "approver_oidc_subject": "...",
  "approved_at": "...",
  "approval_attestation_id": "..."
}
```

### `https://crucible.dev/TestReport/v1`

Emitted on `twin.test.run` and the per-tier verifier methods.

```json
{
  "task_id": "...",
  "test_kind": "tier_0_mutation" | "tier_1_pbt" | "tier_2_contract" | "tier_3_proof" | "tier_4_honest_ci" | "project_native",
  "framework": "stryker-js",
  "passed": true,
  "stats": {
    "killed": 91,
    "survived": 9,
    "score": 0.91,
    "iterations": 10000,
    "counterexamples": []
  },
  "duration_seconds": 47.3,
  "verifier_model": "gemini-3.1-pro",
  "verifier_oidc_subject": "https://accounts.crucible.dev/verifiers/..."
}
```

### `https://crucible.dev/VerifierApproval/v1` / `VerifierRejection/v1`

The verifier's final verdict for a task.

```json
{
  "task_id": "...",
  "diff_hash": "0x...",
  "verdict": "approved",
  "rubric_score": 0.92,
  "tier_results": {
    "tier_0": { "passed": true, "report_attestation": "rekor:..." },
    "tier_1": { "passed": true, "report_attestation": "rekor:..." },
    "tier_4": { "passed": true, "report_attestation": "rekor:..." }
  },
  "rejection_reasons": [],
  "executor_oidc_subject": "...",
  "verifier_oidc_subject": "...",
  "signed_at": "..."
}
```

### `https://crucible.dev/PlanApproval/v1`

Emitted when a user approves a plan.

```json
{
  "task_id": "...",
  "plan_hash": "0x...",
  "estimated_cost_usd": 1.20,
  "approved_by_oidc": "...",
  "approved_at": "..."
}
```

### `https://crucible.dev/PromotionBundle/v1`

The bundle submitted to the Promotion Contract.

```json
{
  "task_id": "...",
  "diff_hash": "0x...",
  "verifier_approval_attestation": "rekor:...",
  "files_changed": [...],
  "build_provenance": { "$ref": "SLSA Provenance v1" },
  "rebuild_hash": "0x...",
  "blast_radius": { ... },
  "suggested_rollout": { ... },
  "agent_oidc_subject": "...",
  "signed_at": "..."
}
```

### `https://crucible.dev/PromotionApproval/v1`

Emitted by the Promotion Contract after policy + human approval.

```json
{
  "bundle_attestation": "rekor:...",
  "policy_decision": "auto-approve" | "human-approved",
  "rego_policy_hash": "0x...",
  "rego_decision_doc": { ... },
  "human_approvers_oidc_subjects": ["...", "..."],
  "kms_signing_key_arn": "arn:aws:kms:...",
  "lease_id": "lease_...",
  "approved_at": "..."
}
```

### `https://crucible.dev/PromotionOutcome/v1`

Final outcome of a promotion.

```json
{
  "promotion_id": "prom_...",
  "bundle_attestation": "rekor:...",
  "outcome": "landed" | "rolled_back" | "approval_timeout" | "policy_denied",
  "rollout_steps": [
    { "weight": 1, "dwell_seconds": 300, "slo_check": "passed", "timestamp": "..." },
    ...
  ],
  "final_state": "100% live",
  "rollback_reason": null,
  "completed_at": "..."
}
```

### `https://crucible.dev/MemoryWrite/v1`

Emitted on procedural memory writes (both agent-initiated and distiller-initiated).

```json
{
  "convention_id": "conv_...",
  "tenant_id": "...",
  "scope": { ... },
  "rule_nl": "...",
  "category": "Logging",
  "source_evidence": [{"kind":"pr_comment","pr":...}],
  "confidence": 0.74,
  "judge_score": 0.91,
  "writer_oidc_subject": "...",
  "written_at": "..."
}
```

## Build provenance — SLSA-L3

Crucible's Tier 4 emits SLSA Provenance v1 in addition to our own predicate types. The SLSA predicate is the standard `https://slsa.dev/provenance/v1` schema. Key fields:

```json
{
  "buildDefinition": {
    "buildType": "https://crucible.dev/build/v1",
    "externalParameters": {
      "source": "git+https://github.com/acme/payments@abcd1234",
      "config": "nix flake"
    },
    "internalParameters": {
      "nix_lock_hash": "sha256-..."
    },
    "resolvedDependencies": [
      { "uri": "git+...", "digest": {"sha1":"..."} }
    ]
  },
  "runDetails": {
    "builder": {
      "id": "https://crucible.dev/builders/hermetic-nix/v1",
      "version": { "nix": "2.21.0" }
    },
    "metadata": {
      "invocationId": "task_01HZ...",
      "startedOn": "...",
      "finishedOn": "..."
    },
    "byproducts": [
      { "name": "rebuild_hash", "uri": "...", "digest": {"sha256":"..."} }
    ]
  }
}
```

## Signature format — DSSE

All Crucible attestations use the DSSE (Dead Simple Signing Envelope) format. Sigstore Rekor v2 native support.

```json
{
  "payloadType": "application/vnd.in-toto+json",
  "payload": "<base64-encoded statement>",
  "signatures": [
    {
      "keyid": "",
      "sig": "<base64 signature>",
      "cert": "<base64 x509 cert from Fulcio OIDC issuance>"
    }
  ]
}
```

## Verifying an attestation

```bash
# Fetch attestation by Rekor UUID
crucible attestation get rekor:7d8a2c...

# Verify signature chain
crucible attestation verify rekor:7d8a2c...
  ✓ DSSE signature valid
  ✓ Fulcio cert chains to Sigstore root
  ✓ OIDC subject: https://accounts.crucible.dev/agents/worker-7
  ✓ Predicate type: https://crucible.dev/VerifierApproval/v1
  ✓ Statement subject matches diff hash
  ✓ Rekor inclusion proof valid

# Fetch the full chain for a task
crucible attestation chain task_01HZ...
  rekor:abc... PlanApproval
  rekor:def... WriteAttestation (api/webhooks.ts)
  rekor:ghi... WriteAttestation (db/migrations/20260515_refunds.sql)
  ...
  rekor:xyz... VerifierApproval
  rekor:123... PromotionApproval
  rekor:456... PromotionOutcome (landed)
```

## Self-hosted Rekor

Enterprise self-hosted deployments run their own Rekor instance. The OIDC issuer is the customer's own (configurable). The Fulcio CA root is bundled with the air-gap installer.

Public verification commands transparently work against the customer's self-hosted Rekor — `crucible attestation verify` reads the issuer from the cert and dispatches to the correct log.

## Schema source of truth

All predicate JSON-Schemas live in `libs/twin-spec/schemas/`. They are versioned via the predicate-type URI path (`/v1`, `/v2`, etc.). Breaking changes bump the version + 90-day deprecation. Old versions remain readable indefinitely (Rekor is append-only and immutable).

## Retention

- Sigstore Rekor public log: forever.
- Customer-side mirror: configurable per tenant; default 7 years (matches financial-records retention for the regulated tier).
- Tenant export: full Rekor mirror available via `crucible attestation export` for archival.

---


# 04. Operations

<a id="file-04-operations--onboarding"></a>

<!-- ================================================================== -->
<!-- File: 04-operations/onboarding.md -->
<!-- ================================================================== -->

# Customer Onboarding

The "first week" experience that turns a sign-up into a verified, value-producing customer. We optimize for time-to-first-verified-PR.

## The four-stage journey

```
1. Install               (≤ 5 min)
2. Cartography           (≤ 30 min, automated; 1 agent-day of one-time work)
3. First verified PR     (same day)
4. Convention bootstrap  (rolling, daily)
```

## Stage 1 — Install

### SaaS install

1. Sign up at app.crucible.dev → tenant created.
2. Install the GitHub App on target repos.
3. (Optional) Connect Slack workspace for approval routing.
4. (Optional) Install IDE plugin (VS Code Marketplace, Zed via ACP, JetBrains, etc.).
5. Connect data sources for procedural memory bootstrap:
   - GitHub (required) — PR review comments, ADRs, merged code.
   - Linear / Jira (optional) — incident references.
   - Slack #incidents channel (optional) — post-mortem text.
   - Confluence / Notion (optional) — runbooks, ADR pages.

The GitHub App requests scoped permissions: repo read, PR write (for the agent's PRs), workflow read (for verifier results). No org-admin permissions needed.

### Self-hosted install

See [self-hosted-install.md](self-hosted-install.md) — Helm chart, KMS/HSM, observability stack. Hours of setup, but a one-time event. The product surface is identical to SaaS after install.

## Stage 2 — Cartography

The first task Crucible runs on every new repo is the **Cartographer**. This is a Crucible-internal task that:

1. Walks the repo, builds a tree-sitter symbol index.
2. Parses lint configs (`.editorconfig`, `.prettierrc`, `.eslintrc`, `.rubocop.yml`, `pyproject.toml`, `rustfmt.toml`, etc.) into deterministic Convention rules.
3. Reads `AGENTS.md`, `CLAUDE.md`, `CONTRIBUTING.md`, ADR directories.
4. Scans recent PR review comments (last 24 months, capped at top 1,000 by length).
5. Scans incident/post-mortem references found in PR descriptions.
6. Generates an inferred `AGENTS.md` (if one doesn't exist) and presents it for user review.
7. Bootstraps the per-tenant memory with the above + OSS-derived defaults (Tier A–D corpus per [06-research/memory-bootstrap.md](../06-research/memory-bootstrap.md)).

The Cartographer runs once per repo. It costs ~$3–$10 on a 50K-LoC repo, ~$30–$100 on a 500K-LoC repo. We absorb this cost — it's not billed against the customer's PR quota.

Output presented to the customer:

```
✓ Indexed 1,247 files across 38 directories.
✓ Detected stack: Next.js 14, FastAPI 0.110, PostgreSQL 16.
✓ Extracted 184 conventions from your existing config + AGENTS.md.
✓ Loaded 312 OSS-derived defaults for your stack.
✓ Inferred 47 additional conventions from your PR review history.
  - 12 high-confidence (recommended; surfaced active)
  - 23 medium-confidence (surfaced as suggestions)
  - 12 low-confidence (stored as candidates; not surfaced yet)

Review at: https://app.crucible.dev/memory
```

The customer can edit, accept, or reject any convention. Edits feed back into procedural memory as authoritative overrides.

## Stage 3 — First verified PR

The fast-path "wow moment" we optimize for.

1. Customer's senior engineer picks a small, real, in-progress task they've been putting off — usually:
   - A focused refactor ("replace `fetch` calls with our new API client").
   - A small feature ("add idempotency key to the webhook endpoint").
   - A bug fix from the team's bug tracker.

2. They invoke Crucible via their preferred surface:
   - IDE: open command palette, `Crucible: New Task`.
   - CLI: `crucible task new --repo acme/payments --description "..."`.
   - Web: paste the description into the dashboard.
   - GitHub: comment `/crucible <description>` on an issue.

3. Crucible spins up a twin, generates a plan, presents it. **Time-to-plan target: < 90 seconds.**

4. Customer approves the plan. Crucible executes. **Median execution: 5–15 minutes.**

5. Verifier runs cross-family. Returns approval or structured rejection.

6. If approved, Crucible opens a PR on the customer's repo with:
   - The diff.
   - The verifier's report (mutation score, PBT counterexamples (none), conventions applied, attestation chain).
   - A signed `PromotionBundle` linked.
   - Suggested rollout strategy.

7. Customer reviews and merges. Promotion contract executes (canary, dwell, land).

**The full first-task experience targets under 30 minutes from "first task submitted" to "verified PR merged."** Every minute longer materially reduces the conversion-to-paid rate.

## Stage 4 — Convention bootstrap (rolling)

After the first task, the distillation worker is consuming the customer's PR review activity in real time. Every PR that gets merged feeds:

- Positive examples for active conventions.
- Candidate conventions from comments that were addressed.
- Negative examples when reviewers correct violations.
- Anti-patterns from any post-mortems that reference the changed files.

After ~30 days of typical activity (assuming 5–10 merged PRs/week), the per-tenant memory has compounded enough to be **noticeably** better at applying team conventions than day-1.

The customer sees this in the web console:

```
Memory growth (past 30 days)
  +18 active conventions (from candidates)
  +5 conventions superseded
  +2 conventions flagged drifting (review required)
  
Convention compliance on agent PRs:
  Week 1: 91%
  Week 4: 97%
```

This compounding is the **stickiness mechanism**. Switching to another agent loses 30 days of learned taste.

## Onboarding touchpoints

### Week 1 — Setup

- Day 1: install + Cartographer + first verified PR.
- Day 2: customer success outreach to the senior-engineer champion. "Anything unexpected?"
- Day 3: dashboard walk-through (live or recorded).
- Day 5: weekly digest email starts (sent every Friday).

### Week 2 — Expand

- Customer adds 2–3 more repos.
- Customer invites the rest of the team (Team tier).
- We propose three "good first tasks" specific to their codebase based on the Cartographer's output.

### Week 4 — Review

- 30-day check-in. Show the compounding metrics.
- Convention drift review (one-touch acknowledgment of any drifting conventions).
- Pricing-tier conversation if usage indicates upgrade.

### Quarter 1 — Embedded

- Customer's team has Crucible PRs as a normal part of the workflow.
- We're an ambient capability, not a "tool we're trying."
- Renewal conversation initiated 60 days before renewal date.

## Self-serve vs assisted

### Self-serve (default for Pro tier)

The product onboards itself. Customer success monitors but only reaches out when:
- Time-to-first-verified-PR exceeds 24 hours.
- Customer drops out mid-Cartography.
- Customer hits a destructive-op gate (good opportunity to highlight the safety story).

### Assisted (default for Team / Outcome / Enterprise tiers)

A named contact at Crucible:
- Joins the install call.
- Reviews the Cartographer output with the customer.
- Selects the first task together.
- Reviews the first 5 verified PRs.

The assisted-onboarding budget is funded by the price differential between Pro and Team.

## Anti-patterns we explicitly avoid

- **Bigger demos before smaller value.** Don't do a 90-minute architecture walkthrough before they've seen one verified PR.
- **Synthetic demo tasks.** The first task is theirs, on their real repo, against their real conventions. Demo tasks teach customers that we only work on toy problems.
- **Quota-anxiety messaging.** Don't make new customers worry about their 25 PRs/mo before they've seen their first.
- **Convention prescription.** We surface defaults; we never override their explicit rules without user confirmation.

## Failure modes and recovery

| Failure | Detection | Recovery |
|---|---|---|
| Cartographer crashes on huge repo | Job watchdog | Auto-restart at last checkpoint; chunk by directory if >1M LoC |
| First task verifier rejects | Verifier verdict | Surface rejection with structured reasons; offer one auto-retry or human assist |
| Customer never installs GitHub App | Tracker missing GitHub events | Drip email + Slack-friendly nudge from CS |
| Customer installs but never submits task | Empty task log | Day-3 outreach with three suggested first-task options based on the Cartographer's output |
| Customer hits unexpected destructive-op gate | Gate event | Surface as a *good* thing in the dashboard — that's the safety story in action |
| Customer's tape coverage is too thin | Live-call denials | Walk through shadow-recording setup; help them populate tapes |

## What the customer never has to do

- Write `.cursorrules`, `AGENTS.md`, or any manual prompt. The Cartographer infers from their codebase.
- Configure model routing. The router picks per task.
- Manage budgets manually. Plans show projected cost.
- Manage memory eviction. Background distiller and importance scorer handle it.
- Sign individual attestations. Sigstore keyless OIDC is automatic.

The right experience is **the agent shows up, learns, ships verified PRs, and gets better monthly.**

---

<a id="file-04-operations--runbooks"></a>

<!-- ================================================================== -->
<!-- File: 04-operations/runbooks.md -->
<!-- ================================================================== -->

# Runbooks

Common operational scenarios with specific actions. Each runbook assumes the on-call has access to the Crucible web console, the Honeycomb/Grafana dashboards, and the internal CLI (`crucible-ops`).

## Index

- [RB-01: Cache hit rate dropped below 60%](#rb-01-cache-hit-rate-dropped)
- [RB-02: Median task cost > $5 sustained](#rb-02-median-task-cost-exceeded)
- [RB-03: Egress violation event](#rb-03-egress-violation)
- [RB-04: Sandbox escape attempt](#rb-04-sandbox-escape)
- [RB-05: Sigstore Rekor unreachable](#rb-05-rekor-unreachable)
- [RB-06: KMS signing failure](#rb-06-kms-signing-failure)
- [RB-07: Verifier disagreement > 25%](#rb-07-verifier-disagreement)
- [RB-08: Cross-tenant access attempt](#rb-08-cross-tenant-access)
- [RB-09: Twin spawn failure rate > 2%](#rb-09-twin-spawn-failure)
- [RB-10: Tier 3 proof timeout rate > 25%](#rb-10-tier3-timeout-rate)
- [RB-11: Customer reports false promotion approval](#rb-11-false-promotion-approval)
- [RB-12: Convention drift detected at scale](#rb-12-convention-drift)
- [RB-13: Vendor LLM model deprecation announced](#rb-13-vendor-deprecation)
- [RB-14: Frontier model API outage](#rb-14-model-api-outage)
- [RB-15: Customer requests emergency tenant freeze](#rb-15-emergency-tenant-freeze)

---

## RB-01: Cache hit rate dropped {#rb-01-cache-hit-rate-dropped}

**Severity:** P1 (margin-impacting; not user-facing-broken)

**Detection:** Honeycomb alert on cache hit rate metric < 60% sustained 30 min.

**Immediate actions:**

1. Check per-vendor cache status:
   ```
   crucible-ops cache stats --by vendor --window 1h
   ```
2. Identify which vendor's cache is missing — Anthropic 1h cache often misses if system prompt drifted.
3. Check recent deploys for system-prompt or tool-definition changes that could have invalidated cache keys.

**Root causes (most common):**

- **System prompt drift** — a small edit invalidated all caches. Roll back if customer-visible degradation; otherwise let it re-warm.
- **Tool definition changed** — same as above.
- **Vendor-side cache TTL changed** — check vendor announcement page.
- **Tenant load shift** — heavy new tenant onboarded with different traffic shape; expect rebuild within 24h.

**Fix:**

- If prompt drift: revert the prompt change, deploy. Cache rebuilds in minutes.
- If vendor issue: status page, internal incident open, customer comms not required unless degradation is user-visible.
- If load shift: monitor; warm-up converges within a day.

**Postmortem template:** required for any cache regression > 4h.

---

## RB-02: Median task cost exceeded {#rb-02-median-task-cost-exceeded}

**Severity:** P1

**Detection:** Median per-task cost > $2.50 sustained 1h.

**Investigation:**

1. Check routing distribution: is more traffic going to Tier 2 (Opus 4.7) than expected?
2. Check cache hit rate (RB-01).
3. Check verifier cost ratio — should be ≤ 10% of total; if higher, verifier is being invoked too aggressively.
4. Check task wall-clock — long tasks correlate with retry budget consumption.

**Common causes:**

- New tenant with unusually complex tasks (high Tier 2 mix).
- Cache regression (RB-01).
- Verifier disagreement loop (RB-07).
- Anti-loop protocol not firing — investigate Bounded Budget Enforcer logs.

**Fix:**

- Adjust routing threshold if a tenant's tasks are systematically misclassified.
- Tune verifier rubric_score threshold if rejection rate too high.
- Surface to product: a tenant with sustained > $5 median cost is at risk of churn — Customer Success should reach out before they see the bill.

---

## RB-03: Egress violation {#rb-03-egress-violation}

**Severity:** P0 (security event)

**Detection:** Any `security.egress_violation` event.

**Immediate actions:**

1. **Page on-call security lead.**
2. Tenant + task isolation — the offending sandbox is already SIGKILL'd by Tetragon.
3. Pull attestation chain for the task:
   ```
   crucible-ops attestation chain <task_id>
   ```
4. Identify what the agent was attempting to reach:
   ```
   crucible-ops egress incident <incident_id>
   ```
5. Determine whether this was:
   - **Benign misconfiguration** — agent legitimately needed an endpoint that wasn't in the manifest.
   - **Prompt-injection attempt** — task description or input data tried to exfiltrate.
   - **Compromised dependency** — a package the agent installed tried to phone home.
   - **Sandbox escape attempt** (escalate to RB-04).

**Disposition:**

- If benign: update the task's allowed_egress manifest; release the gate; customer notified.
- If injection: investigate the input source; tighten LLM-judge filter; alert customer.
- If dependency: investigate the package; report to OSS maintainers if needed; tenant + similar tenants notified.
- If escape: RB-04.

**Customer comms:** within 24h, regardless of disposition. Trust posture requires transparency.

---

## RB-04: Sandbox escape attempt {#rb-04-sandbox-escape}

**Severity:** P0 (critical security event)

**Detection:** Any syscall anomaly that crosses the Firecracker boundary, OR any successful access to host filesystem from a sandbox, OR any unexpected privilege escalation.

**Immediate actions:**

1. **Page on-call security lead + CTO.**
2. Quarantine the affected sandbox host machine. Drain all other sandboxes off it.
3. Snapshot the host's memory + disk for forensic analysis.
4. Pull the full attestation chain + OTel trace for the offending task.
5. Identify the attack vector (model output? injected dependency? CVE in Firecracker / kernel? misconfigured seccomp?).

**Within 1h:**
- All other sandboxes audited for the same attack pattern.
- Tenant of the offending task notified (the task may be legitimate red-teaming).
- Internal incident open with named owner.

**Within 24h:**
- Public security advisory if the vector is reproducible (we don't hide).
- Patch deployed with Tier 4 attestation.

**Within 72h:**
- Public postmortem.
- Adversarial test case added to the Crucible Test Harness.

---

## RB-05: Sigstore Rekor unreachable {#rb-05-rekor-unreachable}

**Severity:** P1 (audit-trail gap, but local journaling continues)

**Detection:** Sigstore Rekor publish failure rate > 1% for 5 min.

**Immediate actions:**

1. Confirm Rekor public log status at status.sigstore.dev.
2. Verify our local journaling is still operational — attestations queue locally until Rekor recovers.
3. Set the customer-visible status banner: "Attestation publishing temporarily delayed (no functional impact on tasks)."

**During the outage:**

- Tasks continue normally. The attestation socket buffers locally.
- Promotions that require Rekor verification of inbound attestations are gated — they wait for Rekor to recover OR for fallback to a customer's self-hosted Rekor (enterprise tier).

**After recovery:**

- Local journal back-fills to Rekor in priority order.
- Postmortem if outage > 30 min.

---

## RB-06: KMS signing failure {#rb-06-kms-signing-failure}

**Severity:** P1 (promotions blocked; verification continues)

**Detection:** KMS signing API error rate > 1% for 5 min.

**Immediate actions:**

1. Identify which KMS — AWS, GCP, customer's HSM (per deployment).
2. Check vendor status page.
3. Surface customer-visible status: "Promotion approvals temporarily delayed."
4. Promotions queue; verification work continues.

**Fallback:**

- If vendor KMS is the issue: no fallback. Wait for recovery.
- If customer's HSM is the issue: customer's IT team is engaged via their incident channel.

---

## RB-07: Verifier disagreement {#rb-07-verifier-disagreement}

**Severity:** P2 (quality signal)

**Detection:** Verifier disagrees with human reviewer's verdict > 25% over 24h (shadow-mode metric).

**Investigation:**

1. Sample 20 recent disagreements.
2. Classify:
   - **Verifier too strict:** verifier rejected, human merged anyway. Likely calibration drift.
   - **Verifier too lenient:** verifier approved, human rejected. More serious — adjust thresholds upward.
   - **Genuine style disagreements:** noise; expected.

**Fix:**

- If strict: adjust rubric_score threshold (currently 0.85; consider 0.80 if disagreements are style-only).
- If lenient: add per-category check (e.g., security-related diffs require human review regardless of verifier verdict). Tighten the cross-family pairing if one family is consistently more lenient.

**Communicate** to customer if their tenant shows the pattern: "We noticed our verifier is rejecting more than your reviewers — we're tuning."

---

## RB-08: Cross-tenant access attempt {#rb-08-cross-tenant-access}

**Severity:** P0 (existential isolation issue)

**Detection:** Any read or write to a tenant-scoped resource by a process bearing a different tenant's OIDC subject.

**Immediate actions:**

1. **Page CTO + security lead + CEO.**
2. Quarantine the offending process and any code path it executed.
3. Identify whether data was actually exfiltrated.

**Within 1h:**
- All affected tenants notified individually.
- Public status banner if multiple tenants involved.

**Within 24h:**
- Patch deployed.
- Full postmortem.
- External security firm engaged for verification.

**Within 30 days:**
- Customer-facing report.

This is the most severe event class. The architecture should make it vanishingly unlikely, but we treat any positive signal as code red.

---

## RB-09: Twin spawn failure {#rb-09-twin-spawn-failure}

**Severity:** P1

**Detection:** Twin-runtime spawn failure rate > 2% for 10 min.

**Investigation:**

1. Check sandbox-provider status (E2B / Daytona / self-hosted Firecracker pool).
2. Check Neon API status if DB branch provisioning is the bottleneck.
3. Check our own control-plane's manifest validation pipeline.

**Common causes:**

- E2B / Daytona throttling under load.
- Neon API timeout (rare; usually < 2s).
- Manifest validation regression after a control-plane deploy.

**Fix:**

- If provider throttling: scale capacity request, fall back to alternative provider.
- If Neon: pre-warm a pool of "twin-base" branches.
- If our code: rollback.

---

## RB-10: Tier 3 timeout rate {#rb-10-tier3-timeout-rate}

**Severity:** P2

**Detection:** Tier 3 proofs timing out > 25% over 24h.

**Investigation:**

1. Sample the failing proofs by tenant + prover (Dafny / Lean / TLA+).
2. Look for:
   - Misclassification (Tier 3 triggered on non-critical code).
   - Inadequate LLM-driven proof hints (the hint model is failing to converge).
   - Library-version regressions (Dafny / Lean updates).

**Fix:**

- If misclassified: tune the critical-path classifier. Lower escalation rate.
- If hint convergence: adjust the hint-generation prompt; refresh fine-tuned hint model if applicable.
- If library regression: pin to previous version; open issue upstream.

**Customer-facing:** the Tier 2.5 fallback (PBT + mutation + CODEOWNER review) means user impact is bounded — they still get verified PRs, just with a different proof chain.

---

## RB-11: False promotion approval {#rb-11-false-promotion-approval}

**Severity:** P0

**Scenario:** Customer reports an agent-merged PR was wrong and shouldn't have been approved.

**Investigation:**

1. Pull the full attestation chain.
2. Identify which gate let the change through:
   - Verifier approved when it shouldn't have? → RB-07 + adjust thresholds.
   - Rego policy auto-approved when human approval was required? → Policy bug; fix and redeploy.
   - Human approver clicked approve in error? → Procedural memory note: "human approved X; we maintain the audit trail."
3. Roll back the change via the customer's existing rollback infrastructure (we don't auto-undo merged PRs).

**Customer comms:** acknowledge within 1h; full incident report within 24h with the chain-of-evidence.

---

## RB-12: Convention drift {#rb-12-convention-drift}

**Severity:** P3 (informational)

**Detection:** > 10 conventions flagged drifting per tenant per week.

**Investigation:**

This is usually a *signal*, not a problem — the team's practices are evolving. The drift detector is doing its job.

**Action:**

- Surface the drift to the customer in the weekly digest.
- The customer reviews and either confirms (active → active), supersedes (active → superseded + new active), or archives (active → archived).
- We monitor for systemic drift (multiple tenants in the same stack drifting the same convention) — that suggests our OSS default is outdated.

---

## RB-13: Vendor deprecation {#rb-13-vendor-deprecation}

**Severity:** P2 (proactive)

**Scenario:** Anthropic / Google / OpenAI announces deprecation of a model in our routing table.

**Action:**

1. Add to the routing config: deprecated model → alternate model.
2. Test the alternate on the Crucible Test Harness; verify no regression.
3. Customer-visible changelog entry.
4. Update [01-architecture/model-routing.md](../01-architecture/model-routing.md).
5. Remove deprecated model from the routing table after the vendor's deprecation date.

For BYOK customers, they receive a notification but can override our default.

---

## RB-14: Frontier model API outage {#rb-14-model-api-outage}

**Severity:** P0 if primary executor; P1 if alternate.

**Detection:** Anthropic / Google / OpenAI API error rate > 5% for 5 min.

**Immediate actions:**

1. Auto-fail-over to alternate model in the routing table.
2. Surface customer-visible status: "Primary model degraded; routing to alternate. Quality may differ slightly."
3. Continue to monitor.

**During the outage:**

- Tasks continue with the alternate.
- Cache hit rate drops temporarily (different model = different cache).
- Cost may shift (alternate may be more expensive); cost-meter alerts as usual.

**After recovery:**

- Resume normal routing.
- Postmortem if outage > 1h.

---

## RB-15: Emergency tenant freeze {#rb-15-emergency-tenant-freeze}

**Scenario:** Customer requests an immediate halt of all agent activity (e.g., active incident, suspected compromise).

**Action:**

1. Authenticate the requester via established out-of-band channel.
2. Set tenant freeze:
   ```
   crucible-ops tenant freeze <tenant_id> --reason "..." --requested_by <oidc>
   ```
3. All in-flight tasks halt at next checkpoint.
4. All new task submissions return `TenantFrozen`.
5. Customer maintains access to web console, attestation log, and memory browser.

**Unfreeze:**

1. Customer requests unfreeze via the same channel.
2. We unfreeze with a signed attestation explaining the lifecycle.

---

## Runbook maintenance

- Every postmortem must update or create a runbook.
- Runbooks reviewed quarterly.
- The runbook list is itself versioned; major changes documented in CHANGELOG.
- On-call review covers the top 5 most-likely-to-fire runbooks each rotation.

---

<a id="file-04-operations--self-hosted-install"></a>

<!-- ================================================================== -->
<!-- File: 04-operations/self-hosted-install.md -->
<!-- ================================================================== -->

# Self-Hosted Install

The enterprise tier ships Crucible as a Helm chart + air-gap bundle for fully on-prem or VPC deployment. SLSA-L3 attested, Sigstore-signed, FedRAMP-compatible architecture.

## Deployment topologies

### A. Single-tenant cloud (customer's VPC)

- Customer runs Crucible inside their own AWS/GCP/Azure VPC.
- Outbound to frontier LLM APIs allowed (Anthropic, Google, etc.) over their existing egress.
- Their own KMS for production-promotion signing.
- Their own object storage for tape archives.
- Their own Postgres for memory + attestations.

### B. Air-gapped

- Crucible runs entirely on-prem.
- No outbound connectivity to public LLM APIs.
- Local model inference via vLLM / sglang (Llama 4, DeepSeek V4-Pro, Qwen3-Coder-Plus).
- Local Sigstore Rekor + Fulcio CA.
- Local HSM (Thales / YubiHSM / AWS CloudHSM standalone).
- Air-gap installer bundle (~12 GB) loaded from media.

### C. Hybrid

- Crucible runs on-prem.
- Outbound only to BAA-covered LLM APIs (Anthropic w/ BAA, Azure OpenAI w/ BAA, Vertex AI w/ BAA).
- All other components on-prem.
- Suitable for HIPAA / SOC-2 / financial-services compliance contexts.

## System requirements

| Component | Minimum | Recommended |
|---|---|---|
| Kubernetes cluster | 1.28+, 3 nodes | 1.30+, 6 nodes for HA |
| Node sizing (control plane) | 8 vCPU / 32 GB RAM | 16 vCPU / 64 GB RAM |
| Node sizing (twin runtime pool) | 4 vCPU / 16 GB RAM (per concurrent twin) | scale to peak twin demand |
| Postgres | 15+, 100 GB | 16+, 500 GB+ for memory layer |
| Redis | 7+ | 7+ cluster mode for HA |
| Object storage | S3-compatible, 1 TB | 10 TB+ for tape archive |
| FalkorDB | 1 instance | clustered for HA |
| KMS / HSM | required for promotion-gate signing | hardware HSM for FedRAMP |

GPU optional. Required only for the air-gapped tier where local LLM inference is the only option.

## Components shipped

```
crucible-enterprise/
├── helm/
│   └── crucible/                     # Umbrella chart
│       ├── charts/
│       │   ├── control-plane/
│       │   ├── twin-runtime/
│       │   ├── verifier/
│       │   ├── distiller/
│       │   ├── promotion-gate/
│       │   ├── attestation-relay/
│       │   ├── tape-scrubber/
│       │   ├── memory-router/
│       │   ├── cost-meter/
│       │   └── web-console/
│       └── values-airgap-default.yaml
│
├── images/                            # OCI image bundle (air-gap)
│   └── *.oci                          # All Crucible images, SLSA-attested
│
├── sigstore/
│   ├── rekor-bundle/                  # Self-hosted Rekor
│   ├── fulcio-bundle/                 # Self-hosted Fulcio CA
│   └── trusted-root.json
│
├── policies/
│   ├── default-rego/                  # Promotion-gate policies
│   └── default-egress/                # Cilium / Tetragon templates
│
├── verifiers/                         # Per-language verifier images
│
├── models/                            # (air-gap only) local model weights
│   ├── llama-4-scout/
│   ├── deepseek-v4-pro/
│   └── qwen3-coder-plus/
│
├── docs/                              # Local copy of these docs
├── slsa-provenance.json               # Provenance for this bundle
└── INSTALL.md
```

## Install steps

### Online install (topologies A + C)

```bash
# 1. Add the Crucible Helm repo
helm repo add crucible https://charts.crucible.dev
helm repo update

# 2. Generate a values.yaml for your environment
crucible-cli generate-values \
  --topology vpc \
  --kms aws-kms \
  --kms-key-arn arn:aws:kms:us-east-1:...:key/... \
  --db-host postgres.internal \
  --db-credentials secretsmanager:crucible-pg \
  --object-storage-bucket s3://acme-crucible-tapes \
  --llm-provider anthropic \
  --llm-api-key-secret secretsmanager:anthropic-key \
  > values.yaml

# 3. Install
helm install crucible crucible/crucible \
  --namespace crucible-system \
  --create-namespace \
  --values values.yaml

# 4. Verify the install
crucible-cli verify-install
  ✓ Control plane reachable
  ✓ Twin runtime provisioning a test sandbox in 187ms
  ✓ DB connectivity verified
  ✓ KMS signing test passed
  ✓ Object storage write/read passed
  ✓ Verifier daemon healthy
  ✓ Web console reachable at https://crucible.acme.internal
```

### Air-gap install (topology B)

```bash
# 1. Mount or download the bundle
mkdir /opt/crucible && cd /opt/crucible
tar xzf crucible-enterprise-2026.06.0.tar.gz

# 2. Verify the bundle integrity (SLSA-L3 attestations + Sigstore root)
./scripts/verify-bundle.sh
  ✓ All OCI images verified against in-toto attestations
  ✓ Helm chart signature verified
  ✓ Model weights checksums match published manifest
  ✓ Sigstore trusted root authenticated

# 3. Load images to the local registry
./scripts/load-images.sh --registry registry.internal.acme.com

# 4. Configure for air-gap (no outbound LLM APIs)
crucible-cli generate-values \
  --topology airgap \
  --kms hsm \
  --hsm-pkcs11-lib /usr/lib/pkcs11/libsofthsm2.so \
  --hsm-slot 0 \
  --rekor-mode self-hosted \
  --llm-provider local-vllm \
  --llm-models llama-4-scout,deepseek-v4-pro,qwen3-coder-plus \
  --gpu-pool-namespace gpu-workloads \
  > values.yaml

# 5. Install
helm install crucible ./helm/crucible \
  --namespace crucible-system \
  --create-namespace \
  --values values.yaml

# 6. Initialize the local Sigstore Rekor and Fulcio
./scripts/init-local-sigstore.sh

# 7. Verify
crucible-cli verify-install --topology airgap
```

## Configuration

### values.yaml (key sections)

```yaml
crucible:
  topology: vpc | airgap | hybrid
  domain: crucible.acme.internal
  
  # Storage
  postgres:
    host: postgres.internal
    credentialsSecret: crucible-pg
    sslMode: require
  redis:
    host: redis.internal
    cluster: true
  falkordb:
    host: falkordb.internal
    cluster: true
  objectStorage:
    type: s3
    endpoint: s3.amazonaws.com
    bucket: acme-crucible-tapes
    credentialsSecret: crucible-s3
  
  # Twin runtime
  twinRuntime:
    sandboxProvider: e2b | firecracker-local
    e2bApiKeySecret: e2b-api-key       # if hosted
    firecrackerPoolSize: 100           # if local
    
  # KMS / signing
  kms:
    provider: aws-kms | gcp-cloud-hsm | yubihsm | softhsm | azure-keyvault
    keyRef: arn:aws:kms:...:key/...    # or PKCS11 path for hardware
  
  # Sigstore
  sigstore:
    mode: public | self-hosted
    rekorUrl: https://rekor.internal.acme.com   # if self-hosted
    fulcioUrl: https://fulcio.internal.acme.com # if self-hosted
    trustedRoot: /etc/crucible/sigstore-trusted-root.json
  
  # LLM routing
  llmRouting:
    tier0Provider: anthropic
    tier1Provider: anthropic
    tier2Provider: anthropic
    verifierProvider: google              # cross-family
    airgapLocalProvider: vllm             # for air-gap
    apiKeysSecret: crucible-llm-keys      # contains all configured providers
  
  # Auth
  auth:
    provider: workos | clerk | authelia | dex
    samlMetadataUrl: ...
    oidcDiscoveryUrl: ...
  
  # Observability
  observability:
    tracing:
      exporter: otlp
      endpoint: tempo.internal:4317
    metrics:
      prometheusEndpoint: prom.internal:9090
    logs:
      lokiEndpoint: loki.internal:3100
  
  # Tenant defaults
  defaults:
    promotionPolicy:                  # bundle of default Rego rules
    cartographer:
      maxRepoSize: 5_000_000          # LoC
      modelTier: 1
    
  # Air-gap specifics
  airgap:
    models:
      llamaScoutImage: registry.internal/llama-4-scout:1.0
      deepseekV4ProImage: registry.internal/deepseek-v4-pro:1.0
      qwen3CoderImage: registry.internal/qwen3-coder-plus:1.0
    gpuPool:
      namespace: gpu-workloads
      nodeSelector:
        nvidia.com/gpu.product: A100
```

## Upgrade flow

1. Customer receives release notification (email + status page) 30 days before each release.
2. Customer-facing changelog with breaking changes called out.
3. Air-gap customers receive the new bundle via their preferred channel (signed download + verification command).
4. Helm upgrade:
   ```
   helm upgrade crucible crucible/crucible --values values.yaml
   ```
5. Verify:
   ```
   crucible-cli verify-install
   ```
6. Rollback path (always tested before release):
   ```
   helm rollback crucible <revision>
   ```

Database migrations are forward-only; we don't auto-rollback schema. Migrations are written so that the previous version can still read the new schema (additive only, deprecation windows on column drops).

## SLSA-L3 verification

Every artifact in the bundle is SLSA-L3 attested. Customers can verify:

```bash
crucible-cli verify-release 2026.06.0
  Verifying 47 image attestations against Sigstore Rekor...
  ✓ control-plane@sha256:abc... → rekor:7d8a...
  ✓ twin-runtime@sha256:def... → rekor:9e2b...
  ...
  ✓ All 47 artifacts attested with OIDC subject https://accounts.crucible.dev/builders/...
  ✓ Reproducible-build comparison passed (2 of 2 independent builds bit-identical)
  ✓ Bundle signature verified against Crucible trust root
```

## Customer-controlled signing key

For the highest-assurance tier (FedRAMP / defense), the customer owns the Sigstore Fulcio CA root. Crucible signs nothing on the customer's behalf; the customer's CI signs attestations using their own identity. Crucible's operators do not have signing authority.

## Operational responsibilities

| Responsibility | Crucible (SaaS) | Crucible (VPC) | Customer (self-host) |
|---|---|---|---|
| Helm chart releases | yes | yes | yes |
| Upgrade execution | yes | optional (consulting) | yes |
| Backup of memory + attestations | yes | yes | customer |
| KMS key rotation | yes | yes | customer |
| Incident response | 24/7 | business hours | customer + Crucible advisory |
| Security patches | auto | notify within 24h | customer applies |
| SLO monitoring | yes | yes | customer-side; we provide dashboards |

## Air-gap-specific notes

- **Model updates:** new model weights distributed via signed manifest + media (not network).
- **Local Sigstore root:** the Fulcio CA is bound to the customer's identity provider; rotation procedures documented separately.
- **Local Rekor backup:** Rekor's transparency log must be backed up; loss = audit-trail loss for the period.
- **Telemetry phone-home:** none. We provide tooling for the customer to export anonymized usage stats if they choose.

## Pricing reminder

The self-hosted enterprise tier is $50K/yr base + $400/node/mo. Includes:

- Unlimited use, on-prem inference.
- Quarterly Helm chart releases + air-gap bundle.
- Customer Success contact + business-hours support.
- Annual security review.
- Renewal-time architectural review.

Additional services (24/7 support, named SRE, custom verifier integrations) are scoped separately.

---


# 05. Decisions (ADRs)

<a id="file-05-decisions--readme"></a>

<!-- ================================================================== -->
<!-- File: 05-decisions/README.md -->
<!-- ================================================================== -->

# Architectural Decision Records

A record of the load-bearing choices in Crucible's design, why each was picked, what alternatives were considered, and what consequences follow.

Format: lightweight ADR — Context, Decision, Status, Consequences, Alternatives.

## Index

| ID | Title | Status |
|---|---|---|
| [ADR-001](ADR-001-digital-twin-first.md) | Digital-twin-first execution as the primary trust mechanism | Accepted |
| [ADR-002](ADR-002-cross-family-verifier.md) | Mandatory cross-family verifier for task completion | Accepted |
| [ADR-003](ADR-003-procedural-memory-moat.md) | Per-tenant procedural memory as the primary moat | Accepted |
| [ADR-004](ADR-004-outcome-based-pricing.md) | "Verified PR" as the pricing unit | Accepted |
| [ADR-005](ADR-005-neon-db-branching.md) | Neon for Postgres copy-on-write branching | Accepted |
| [ADR-006](ADR-006-falkordb-over-alternatives.md) | FalkorDB for procedural memory graph backend | Accepted |
| [ADR-007](ADR-007-hoverfly-tape-replay.md) | Hoverfly OSS for service replay | Accepted |
| [ADR-008](ADR-008-tier3-annotation-default-off.md) | Tier 3 formal verification is auto-classified, not default-on | Accepted |
| [ADR-009](ADR-009-anti-loop-protocol.md) | Hard retry cap and bounded-budget enforcer | Accepted |
| [ADR-010](ADR-010-sigstore-rekor-attestations.md) | Sigstore Rekor v2 for transparency log | Accepted |
| [ADR-011](ADR-011-no-built-in-ide.md) | Crucible integrates with existing IDEs via MCP/ACP; no proprietary IDE | Accepted |
| [ADR-012](ADR-012-monorepo-structure.md) | Monorepo with per-component language choices | Accepted |
| [ADR-013](ADR-013-nix-for-tier4-builds.md) | Nix flakes as default for hermetic builds | Accepted |
| [ADR-014](ADR-014-infisical-over-vault.md) | Infisical as default secrets vault | Accepted |
| [ADR-015](ADR-015-firecracker-via-e2b.md) | E2B (Firecracker) as default sandbox in SaaS tier | Accepted |

---

<a id="file-05-decisions--adr-001-digital-twin-first"></a>

<!-- ================================================================== -->
<!-- File: 05-decisions/ADR-001-digital-twin-first.md -->
<!-- ================================================================== -->

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

---

<a id="file-05-decisions--adr-002-cross-family-verifier"></a>

<!-- ================================================================== -->
<!-- File: 05-decisions/ADR-002-cross-family-verifier.md -->
<!-- ================================================================== -->

# ADR-002: Mandatory cross-family verifier for task completion

**Status:** Accepted  
**Date:** 2026-05-15

## Context

Every coding agent today grades its own work. "Tests passed" comes from the same agent that wrote the tests. This is the well-known "the model that hallucinates does not know it is hallucinating" problem.

Crucible's brand promise is verified completion. We need a verification mechanism that:

1. Doesn't depend on the agent's claim.
2. Catches the agent's hallucinations, including ones the agent itself would not detect.
3. Is feasible inside our per-task budget envelope.

## Decision

Every task has two model identities:

- **Executor** — runs the agent's reasoning loop, calls tools, writes code.
- **Verifier** — a separate process running a different model family, reviews the diff + tests + spec changes, issues approval or structured rejection.

The verifier model **must be from a different vendor lineage** than the executor. Strong pairings (validated by published benchmark divergence ~5–10% on SWE-Bench-style tasks):

- Executor `claude-opus-4-7` ↔ Verifier `gemini-3.1-pro` (high thinking)
- Executor `gpt-5.5` ↔ Verifier `claude-opus-4-7`
- Local: Executor `Llama-4-Maverick` ↔ Verifier `DeepSeek-V4-Pro`

The verifier:

- Has **no access to the executor's reasoning trace** — only the diff, tests, spec changes, and OpenAPI delta. This prevents the verifier from inheriting executor hallucinations through the reasoning channel.
- Runs in a separate sandbox with its own twin DB branch.
- Cannot mark "approved" without all required tiers (0/1/2/3/4) returning green.
- Emits a signed `VerifierApproval` or `VerifierRejection` that's required as input to the Promotion Contract.

## Consequences

### Positive

- **Real error decorrelation.** Cross-family pairs disagree on different inputs; same-family pairs share blind spots.
- **Verifier can be cheap.** Verification is end-of-task, not in-loop. Verifier cost is ~8% of total task cost in practice (not 2× as naïve math suggests).
- **Structural defense against fake-test-pass.** The verifier independently mutates the diff and re-runs tests; mocked/skipped tests don't kill mutants.
- **Customer trust signal.** The cross-family attestation chain is checkable: "this PR was approved by Gemini 3.1 Pro after being authored by Opus 4.7. Both signatures on Rekor."

### Negative

- **Two vendor dependencies.** Both Anthropic AND Google (or equivalent pair) must be operational. Mitigation: routing table includes fallback pairs.
- **Verifier latency.** 30s–15min depending on tier. Worth it; quantified in [01-architecture/verifier-pipeline.md](../01-architecture/verifier-pipeline.md).
- **Cross-vendor cache cost.** Verification with a different vendor pays full input price. Mitigation: keep verifier prompts focused (diff + spec, not full repo).
- **Verifier disagreement noise.** Sometimes the verifier rejects what a human would merge. Mitigation: shadow-mode metric tracking; rubric tuning per [04-operations/runbooks.md RB-07](../04-operations/runbooks.md).

## Alternatives considered

### Alternative 1: Same-model two-pass verification

Run the executor model again with "review this code" instructions. **Rejected** because:

- The model that wrote the code is statistically correlated with the model reviewing it (same lineage, same blind spots).
- Published research confirms this approach catches < 30% of bugs that cross-family catches.

### Alternative 2: Internal "ensemble" of differently-prompted same-vendor models

Use Opus + Sonnet from the same vendor as a poor-man's ensemble. **Rejected**:

- Same training pipeline = correlated errors.
- The marginal cost saving (no cross-vendor cache miss) doesn't justify the verification quality loss.

### Alternative 3: Static-analyzer-only verification

Forgo a verifier LLM; rely on traditional SAST/lint + test runner. **Rejected**:

- Static analyzers catch syntactic + known-pattern issues; they don't reason about *semantic* correctness.
- Doesn't address the "agent claims tests pass when they were mocked" failure mode without an LLM rubric.

### Alternative 4: Verifier optional / off by default

Make verification an opt-in. **Rejected**:

- Customers who turn it off lose the brand promise.
- Pricing unit ("verified PR") becomes incoherent.

### Alternative 5: Human-only verification

No LLM verifier; humans review every agent output. **Rejected**:

- Defeats the value prop (overnight verified PRs).
- Doesn't scale.

## Open issues

- **In-house verifier model** (analogous to Cursor's Composer-2): when frontier prices fall enough, training a small Crucible-internal verifier could cut verification cost ~10×. Tabled for v2.
- **Verifier-of-verifier escalation:** for high-stakes promotions, two cross-family verifiers ensemble could push agreement rate higher. v2 enterprise feature.

## References

- [01-architecture/verifier-pipeline.md](../01-architecture/verifier-pipeline.md)
- [01-architecture/model-routing.md](../01-architecture/model-routing.md)
- [06-research/unit-economics.md](../06-research/unit-economics.md)

---

<a id="file-05-decisions--adr-003-procedural-memory-moat"></a>

<!-- ================================================================== -->
<!-- File: 05-decisions/ADR-003-procedural-memory-moat.md -->
<!-- ================================================================== -->

# ADR-003: Per-tenant procedural memory as the primary moat

**Status:** Accepted  
**Date:** 2026-05-15

## Context

Models commoditize quarterly. MCP standardized tools. ACP standardizes agent-host protocols. UX features (multi-agent, manager view, etc.) commoditize within months. What doesn't commoditize is **lived experience with a specific team's codebase and conventions.**

The team-conventions problem is universal: every team's PR review comments contain knowledge that takes years for a new engineer to internalize. Existing agents (Cursor Memories, Claude Code Skills, AGENTS.md) require users to *write* the conventions explicitly. This is brittle, manual, and lossy.

Mining PR review comments, post-mortems, and ADRs into a *learned* team-conventions graph is technically tractable in 2026 (Mem0's hierarchical extraction, Graphiti's temporal KG, FalkorDB) but no product has shipped it.

## Decision

Crucible's procedural memory is a per-tenant temporal knowledge graph of team conventions, learned passively from:

- PR review comments (`(commenter, requested_change_type, code_pattern, accepted?)` triples)
- Incident post-mortems (`(trigger → action → outcome)` chains; "never do X" anti-patterns)
- Architecture Decision Records
- Merged code diffs (as implicit positive examples)

Architecture:

- **Backend:** FalkorDB with Graphiti abstraction; bi-temporal edges (valid_from / valid_to).
- **Distillation:** background worker using Mem0's hierarchical extraction algorithm with schema-constrained decoding.
- **Filtering:** LLM-as-judge on every memory write (defense against prompt-injection via PR comments).
- **Decay:** Ebbinghaus exponential on recency; reinforce-on-access; status lifecycle (`active | drifting | superseded`).
- **Federation:** cross-tenant abstractions allowed only when ≥5 tenants agree on a category-form rule.

The memory layer is:

1. **Read** by the agent at plan time and during reasoning.
2. **Read** by the verifier during the compliance check (closing the learning loop).
3. **Written** explicitly by agents via `twin.memory.note`.
4. **Written** passively by the background distiller.

## Consequences

### Positive

- **Compounding stickiness.** Every PR review feeds the graph. Day-30 customer experience materially outperforms day-1 (typical: 91% → 97% convention compliance over four weeks). Switching to a competitor loses 30+ days of learned taste.
- **Solves "generic AI aesthetic / convention drift" complaint** without requiring users to write rules manually.
- **Convention drift detection** as a customer-visible feature — the system surfaces "your convention X is aging" before defects pile up.
- **Onboarding becomes magical.** Cartographer mines existing PR history; day-1 agent already speaks the team's style.
- **Verifier becomes smarter** — checks compliance against team rules, not just generic best practices.

### Negative

- **Cold-start problem.** New customers have no PR history. Mitigation: OSS-derived defaults (Tier A–D corpus, ~400 active rules on a fresh Next.js+FastAPI repo). See [06-research/memory-bootstrap.md](../06-research/memory-bootstrap.md).
- **Prompt-injection attack surface.** PR comments are attacker-controllable. Mitigation: LLM-as-judge filter on every write, plus cross-source agreement threshold.
- **Cross-tenant leakage risk.** Per-tenant isolation everywhere; federation only to anonymized categorical form. Hard requirement; tested.
- **Storage growth.** Procedural memory grows monotonically (no TTL on active conventions). Mitigation: status lifecycle; superseded conventions archived (not deleted) at a fixed retention.

## Alternatives considered

### Alternative 1: User-written rules only (Cursor Memories / AGENTS.md model)

Require users to manually write `.cursorrules` or AGENTS.md. **Rejected**:

- Brittle, manual, lossy.
- Doesn't compound; the file is what it is.
- Users have to be senior enough to know what conventions to write down.

### Alternative 2: Train a per-customer fine-tune

Fine-tune a small model on the customer's PR history. **Rejected for v1**:

- Compute cost is significant.
- Update latency is days/weeks, not real-time.
- Per-tenant model artifacts complicate compliance and storage.
- The Graphiti+FalkorDB approach gets 80% of the benefit at 5% of the operational complexity.

(May revisit for v2 enterprise tier if customer pressure justifies it.)

### Alternative 3: Use vector store only, no graph

Episodic memory in pgvector; skip the graph layer. **Rejected**:

- Conventions have *relationships* (this rule supersedes that one; this rule conflicts with that one; this rule is a refinement of that). Graph structure captures these natively; flat vectors don't.
- Drift detection needs temporal edges; vectors don't model time well.

### Alternative 4: Single global "consensus memory"

Aggregate across tenants into one shared memory. **Rejected**:

- Cross-tenant data leakage.
- Customers explicitly want *their* taste, not the consensus.
- Federation (Alternative-considered-and-accepted) gives the global-common-knowledge benefit without the privacy violation.

## References

- [01-architecture/memory-layer.md](../01-architecture/memory-layer.md)
- [06-research/memory-bootstrap.md](../06-research/memory-bootstrap.md)
- [ADR-006](ADR-006-falkordb-over-alternatives.md)

---

<a id="file-05-decisions--adr-004-outcome-based-pricing"></a>

<!-- ================================================================== -->
<!-- File: 05-decisions/ADR-004-outcome-based-pricing.md -->
<!-- ================================================================== -->

# ADR-004: "Verified PR" as the pricing unit

**Status:** Accepted  
**Date:** 2026-05-15

## Context

The coding-agent market has bifurcated:

- **Seat-only pricing** (Tabnine, JetBrains) is collapsing because agent costs scale with use, not seats.
- **Pure usage-based credit pools** (Cursor, GitHub June 2026, Replit) produce bill-shock that kills adoption — $200/day blow-ups have been documented across multiple products.
- **Outcome-based pricing** works in adjacent markets (Sierra $1–2/resolution, Intercom Fin $0.99, Zendesk $1.50–$2.00) but no coding-agent vendor has shipped it.

Crucible's architectural commitment to *verification* gives us a clean, defensible unit that no incumbent can claim: the **verified PR**.

## Decision

Crucible prices primarily on "verified PRs delivered" — a PR counts as verified when:

1. All existing tests pass on the real codebase post-promotion.
2. The verifier model rates the diff ≥ 0.85 on its rubric.
3. No human edits the PR before merge — the agent's output stood on its own.
4. The promotion canary holds clean for the configured dwell window.

PRs that fail to meet the bar are not billed. The metering is therefore non-gameable from the customer's side, and the unit means something concrete to a buyer.

Five tiers:

| Tier | Price | Mechanism |
|---|---|---|
| Pro | $40/mo | 25 verified PRs included; $2.50/PR overage |
| Team | $120/dev/mo | 80 verified PRs/dev pooled; $2.00/PR overage |
| Outcome | $8/PR + $500/mo minimum | Pure PAYG above minimum |
| BYOK | $25/dev/mo flat | Unlimited; customer brings model API keys |
| Enterprise (self-hosted) | $50K/yr base + $400/node/mo | Unlimited use, on-prem inference |

## Consequences

### Positive

- **Aligns with buyer mental model.** A 50-dev engineering org procures engineering-hours or story-points. "Verified PR" maps cleanly to both. Tokens / ACUs don't map to anything procurement recognizes.
- **Hard ceiling kills bill-shock.** Pro/Team have included caps + capped overage. Outcome is PAYG with clear per-unit price. No one ever opens a Crucible invoice and sees a 10× surprise.
- **Outcome tier is the profit center.** At $8/PR and ~$1.69 median cost, GM is 79%. Legacy-modernization buyers (highest WTP) compare to consultant hourly rates ($80–$200/hr); $8/PR is ~5–10% of that.
- **First-mover advantage.** No coding-agent vendor has shipped outcome pricing. The narrative differentiation is durable until copied.

### Negative

- **Included-bundle tiers are GM-thin.** Pro at $1.60/PR effective revenue vs $1.69 median cost is structurally negative-GM on the bundle, breakeven-to-profitable on overage. Requires cache hit rate ≥ 70% to be sustainable.
- **Metering complexity.** "Verified" is a 4-condition AND; building the metering correctly is non-trivial. Customer disputes are expensive.
- **PR-complexity distribution risk.** A 1-line config fix and a 2,000-line migration both count as 1 PR. The pricing math assumes "median complexity"; heavy tails distort it. Mitigation: complexity-banded pricing in v2 once we have data.
- **Verifier-rejection edge cases.** If our verifier rejects a PR a human would have merged, the customer feels we're being precious. Mitigation: shadow-mode tracking; rubric tuning per RB-07.

### Trade-offs we accept

- Pro/Team bundles are deliberately a customer-acquisition cost, not a profit center. Outcome tier and Enterprise tier pay the rent.
- We will lose price-sensitive customers who can self-tune their LLM keys cheaper with Aider/Cline. The BYOK tier is our concession to that segment.

## Alternatives considered

### Alternative 1: Cursor-style credit pools

$X/mo includes $Y model-spend. **Rejected**:

- Bill-shock UX is bad. Cursor's 2025 trauma demonstrates this.
- "Credit" isn't a recognizable procurement unit.
- Doesn't reward our verification investment.

### Alternative 2: Pure seat-only ($30/dev/mo flat)

Tabnine model. **Rejected**:

- Whales subsidize light users; team plans bleed margin on heavy users.
- Doesn't align with COGS, which scales with use.

### Alternative 3: Per-token markup over BYOK

Customer pays for tokens + a flat platform fee. **Rejected for primary tier**:

- Same bill-shock problem as Cursor.
- Doesn't capture the verification value (we'd be billing for executor tokens but the verifier is free? Or doubled?).

(BYOK tier exists as a deliberate concession to the Aider/Cline-aligned segment.)

### Alternative 4: Devin-style ACU (Agent Compute Units, 15-min intervals)

Vendor-defined opaque time unit. **Rejected**:

- ACU is internal-bookkeeping made customer-visible. Buyer can't predict cost.
- "Verified PR" is auditable; ACU isn't.

### Alternative 5: Per-story-point or per-Jira-ticket

Charge by the customer's own ticket size. **Rejected**:

- Requires integration with the customer's PM tool to bill — operational dependency.
- Customer disputes over story-point sizing become invoice disputes.

### Alternative 6: Per-test-passing

Charge per test the agent makes pass. **Rejected**:

- Easy to game; agents would write trivial tests.
- Doesn't capture refactor work (no test changes but real value).

## Open issues

- **Complexity banding (v2):** introduce small/median/large PR tiers ($4/$8/$20) once empirical PR distribution data justifies.
- **SLA tier (v3):** offer "N verified PRs guaranteed per month at $X" for enterprise.
- **Open-source maintainer tier (v3):** free for verified OSS maintainer accounts as a brand investment.

## References

- [00-vision/pricing-and-business.md](../00-vision/pricing-and-business.md)
- [06-research/unit-economics.md](../06-research/unit-economics.md)

---

<a id="file-05-decisions--adr-005-neon-db-branching"></a>

<!-- ================================================================== -->
<!-- File: 05-decisions/ADR-005-neon-db-branching.md -->
<!-- ================================================================== -->

# ADR-005: Neon for Postgres copy-on-write branching

**Status:** Accepted  
**Date:** 2026-05-15

## Context

The twin runtime needs a per-task database mirror. The agent must be able to apply migrations and mutate data without touching real production. The mirror must spin up in seconds (or the twin's value evaporates), share storage with the parent (or storage costs explode), and discard cleanly at task end.

Postgres is the most common DB in our target customer base (production-engineering teams of 5–200). MySQL is secondary. Everything else (Mongo, Redis, etc.) is handled per-engine.

## Decision

For Postgres customers: **Neon Postgres branching** is the default twin DB layer.

- `POST /projects/{id}/branches` returns a connection string in 1–2 seconds.
- Copy-on-write at the storage layer; branches share data with parent.
- Branch cost: $0.002/hr (negligible for task duration).
- Storage cost: $0.35/GB-month post-Databricks-acquisition (down from $1.75).
- Cold-start 400–750ms is fine for ephemeral test workloads.

Each project has a **twin-base branch**: a daily snapshot of production with PII scrubbed. Per-task branches are children of twin-base, not children of `main`. This decouples the agent's twin from production database state changes mid-task.

For other engines:

| Engine | Mechanism | Notes |
|---|---|---|
| MySQL | PlanetScale branching | Mature for MySQL; Postgres support still half-built as of May 2026 |
| SQLite / libSQL | Turso branches | Instant per-database CoW |
| MongoDB | Atlas snapshot-restore-to-new-cluster | Minutes (not seconds); acceptable for less-common workload |
| Redis / KV | Fresh `redis-server` inside sandbox | State is small enough to recreate per task |
| S3 | MinIO in sandbox + rclone mirror | Versioning alone insufficient |
| ClickHouse | `CREATE TABLE … CLONE AS` at table level | DB-level clone proposed Apr 2026, not yet stable |

## Consequences

### Positive

- **Branch creation is fast enough to not affect perceived task latency.** 1–2s out of a typical 5–15 minute task is invisible.
- **Marginal storage cost ≈ $0 per twin.** CoW means the branch only diverges from parent for actual writes; for read-heavy tasks the divergence is bytes.
- **Migration verification becomes safe and easy.** Agent applies migration → twin diff vs base → verifier inspects schema delta → no risk to production.
- **Fan-out exploration is cheap.** Multiple parallel twins each get their own branch; no shared-state contention.
- **API surface is clean.** Neon's REST API is one curl call; integration is ~50 LoC.

### Negative

- **Vendor dependency.** Neon is the only meaningful CoW Postgres branching provider as of May 2026. Supabase requires minutes (full project clone). Xata pivoted to OSS but is younger. Self-hosting Neon-equivalent is non-trivial.
- **Per-engine fragmentation.** Not all customers are Postgres. Each non-Postgres engine has a different mechanism with different trade-offs.
- **Twin-base branch staleness.** Daily snapshots may diverge from production by up to 24 hours; tasks against very-recent data may see stale state. Mitigation: customers can request on-demand twin-base refresh.

### Trade-offs we accept

- Customers on Aurora-only, Cassandra, or other unusual stacks get degraded twin DB experience (or none, with explicit per-tenant config). Their workload class is sufficiently atypical we serve them best by being honest about it.
- We pay Neon for the SaaS tier; self-hosted enterprise customers either bring their own Neon installation (uncommon — Neon's self-host story is thin) or accept slower branching via `pg_dump`+`pg_restore` orchestration.

## Alternatives considered

### Alternative 1: Self-hosted Postgres with `pg_dump`/`pg_restore` per task

Use vanilla Postgres + dump-and-restore to create per-task DBs. **Rejected**:

- Restore time is minutes, not seconds. Kills twin perceived latency.
- Storage cost is per-task-full-copy, not CoW. Expensive at scale.
- Migration verification becomes a multi-step orchestration.

### Alternative 2: Postgres logical replication + ephemeral subscribers

Use logical replication to create read replicas; promote ephemeral subscribers for tasks. **Rejected**:

- Subscriber creation is minutes.
- Schema migrations break replication mid-task.
- Complex operational surface.

### Alternative 3: PlanetScale for everything

Single branching vendor regardless of customer's DB. **Rejected**:

- PlanetScale is MySQL-centric; Postgres branching half-built as of May 2026.
- Customer migration to PlanetScale is not feasible as an onboarding step.

### Alternative 4: ZFS / btrfs filesystem-level CoW under self-hosted Postgres

Snapshot the underlying filesystem; mount snapshot as the twin DB's data dir. **Rejected for SaaS**:

- Requires shared infrastructure with weird filesystem-level operational characteristics.
- Hard to multi-tenant safely.

(This *is* the self-hosted enterprise fallback when customers don't run Neon.)

### Alternative 5: Skip the DB twin; mock DB calls

Agent queries against a mock DB. **Rejected**:

- Schema migrations are the most important class of change to test; mocks don't catch them.
- Real query results matter for the agent's reasoning; mocked results lie.

## Open issues

- **Customers without Postgres / MySQL / SQLite / Mongo:** explicit "out of scope for v1; contact us for design partnership" message.
- **On-prem self-hosting:** the air-gapped enterprise tier needs a non-cloud branching story. Likely ZFS-snapshot-based, documented in the operations runbook.
- **Migration rollback testing:** twin can apply migrations forward; testing the *down* migration requires explicit support (currently the customer's responsibility).

## References

- [01-architecture/twin-runtime.md#layer-3-database-twin](../01-architecture/twin-runtime.md)
- Neon pricing: see [ASSETS.md](../ASSETS.md)

---

<a id="file-05-decisions--adr-006-falkordb-over-alternatives"></a>

<!-- ================================================================== -->
<!-- File: 05-decisions/ADR-006-falkordb-over-alternatives.md -->
<!-- ================================================================== -->

# ADR-006: FalkorDB for procedural memory graph backend

**Status:** Accepted  
**Date:** 2026-05-15

## Context

The procedural memory layer (ADR-003) requires a graph database for:

- Temporal edges (bi-temporal: valid_from / valid_to + recorded_at).
- Multi-hop traversals (a convention supersedes another; an incident touched a file owned by a team that authored an ADR).
- Per-tenant isolation.
- Low-latency reads (< 100ms p95 for retrieval router calls).
- Cypher (or equivalent) query language familiar to the team.

As of October 2025, KuzuDB — previously a strong candidate — was archived after Apple's acquisition. Remaining viable options:

| Tool | License | Maturity | Latency profile | Notes |
|---|---|---|---|---|
| Neo4j Community | GPLv3 (Community); commercial Enterprise | Mature, huge ecosystem | Single-instance fine; cluster expensive | License complicates redistribution |
| FalkorDB | Source-available | Active development, KuzuDB-successor framing for AI/GraphRAG | Low-latency Cypher, RedisGraph lineage | New but well-funded |
| ArangoDB | Apache-2.0 | Mature multi-model | Reasonable | Multi-model overkill for this use |
| Memgraph | Source-available commercial | Mature | Fast | Pricing less transparent |
| AWS Neptune | Proprietary | Mature | Higher latency for our access pattern | Vendor lock-in |

## Decision

**FalkorDB** is the default graph backend for procedural memory.

- Source-available license; OSS for our needs.
- Cypher query language; familiar to the team.
- Sub-millisecond queries for typical convention retrieval patterns.
- Active development; KuzuDB-successor positioning in the AI/GraphRAG market.
- Low operational overhead; integrates cleanly with Redis-adjacent infrastructure.

Abstraction layer: **Graphiti** (Zep's OSS engine for temporal knowledge graphs). We use Graphiti's data model (bi-temporal edges, episode-based ingestion) regardless of backend, so we can swap if FalkorDB stops being viable.

For customers who prefer Neo4j (large enterprise with existing graph infra), the self-hosted tier supports `backend: neo4j` as a values.yaml option.

## Consequences

### Positive

- **Low latency.** Sub-millisecond Cypher queries support our < 100ms p95 retrieval-router SLO.
- **Cypher familiarity.** Engineers can debug queries directly.
- **OSS license suitable for redistribution.** No GPL pollution in our chart.
- **Graphiti abstraction insulates us.** If FalkorDB falters, we swap backends without rewriting the memory layer.

### Negative

- **Ecosystem smaller than Neo4j.** Fewer plugins, fewer educational resources, fewer hires-with-experience.
- **Single-vendor risk.** FalkorDB is one company. KuzuDB-archive scenario is a real precedent.
- **Some advanced graph algorithms missing.** OK for procedural memory; not OK for graph-algorithm-heavy workloads. Not our use case.

### Trade-offs we accept

We pay the "small ecosystem" tax in exchange for an OSS-redistributable license and low latency. The Graphiti abstraction caps the cost of any future backend swap.

## Alternatives considered

### Alternative 1: Neo4j Community

GPLv3 is the blocker. Our Helm chart and Docker images are redistributed widely; GPL components in a default deployment create downstream license obligations for customers we can't manage. (We could ship Neo4j separately as an opt-in component, but defaults matter.)

Also: Neo4j Enterprise pricing is opaque and expensive; we'd push that cost to customers.

### Alternative 2: KuzuDB

Was the strongest candidate before the October 2025 archive. Apple's acquisition removed it from contention.

### Alternative 3: ArangoDB

Multi-model (document + graph + key-value). **Rejected**:

- Multi-model is overkill — we use the graph features only.
- Operational footprint heavier than FalkorDB.
- Apache-2.0 license is fine, but the operational complexity isn't worth the license benefit.

### Alternative 4: Vector store only (skip the graph)

Use pgvector / Qdrant for everything. **Rejected** in ADR-003; conventions have relational structure that vectors don't capture.

### Alternative 5: Roll our own graph atop Postgres (recursive CTEs)

Persist conventions as rows + edges in Postgres; query with recursive CTEs. **Rejected**:

- Multi-hop queries are slow.
- We'd be reinventing FalkorDB / Neo4j poorly.
- Not worth the engineering cost.

### Alternative 6: Memgraph

Source-available commercial, fast. **Rejected as default** but kept as a "Memgraph as alternative backend" option for customers who prefer it. Pricing less transparent than FalkorDB's.

## Migration path (if FalkorDB doesn't pan out)

Graphiti abstracts the backend. Migration sketch:

1. Set up the new backend (Neo4j / Memgraph / ArangoDB).
2. Use Graphiti's export/import tooling to move tenant graphs.
3. Cut over reads with feature-flag canary.
4. Cut over writes.
5. Decommission FalkorDB.

Estimated effort: ~1 agent-day for the migration code; longer for customer-facing communication and rollout.

## References

- [01-architecture/memory-layer.md](../01-architecture/memory-layer.md)
- [ADR-003](ADR-003-procedural-memory-moat.md)

---

<a id="file-05-decisions--adr-007-hoverfly-tape-replay"></a>

<!-- ================================================================== -->
<!-- File: 05-decisions/ADR-007-hoverfly-tape-replay.md -->
<!-- ================================================================== -->

# ADR-007: Hoverfly OSS for service replay

**Status:** Accepted  
**Date:** 2026-05-15

## Context

The twin runtime needs to handle the agent's outbound HTTP/gRPC service calls without touching real services. Three primitives are required:

1. **Recording** — capture production (or staging) traffic for replay.
2. **Replay** — serve recorded responses to the agent during twin tasks.
3. **Modes** — strict / hybrid / passthrough behavior per request class.

The service-virtualization market has several mature options (WireMock, Mountebank, Speedscale, Hoverfly, GoReplay, Mockoon, Prism).

## Decision

**Hoverfly OSS** is the default service-replay engine.

- Capture-replay is first-class (not bolt-on like WireMock).
- Five modes (capture, simulate, modify, spy, synthesize) cover the decision tree in [01-architecture/twin-runtime.md#layer-4-service-twin-tapes](../01-architecture/twin-runtime.md).
- Apache-2.0 license; redistributable.
- Active development, mature operational story.

Crucible wraps Hoverfly with:

1. **PII scrubber at capture** — Presidio + spaCy + FF3-1 + deterministic pseudonymization, applied before bytes hit disk.
2. **Content-addressed tape storage** — keyed by `(service, endpoint, request_hash)`.
3. **Tape decision tree** — exact / template / synthesize / passthrough / fail-closed per [twin-runtime.md](../01-architecture/twin-runtime.md).
4. **`X-Crucible-Tape` response header** — agents see whether a response was real / replayed / synthesized.

For specific cases:

- **gRPC / JVM-heavy stacks:** add WireMock as a complement (stronger gRPC story).
- **Cold-start (no recorded traffic):** synthesize from OpenAPI via Microcks-style LLM-augmented Faker.
- **Pact-defined contracts:** ingest Pact files as a tape source.

## Consequences

### Positive

- **Industry-mature replay primitives.** Hoverfly's modes are battle-tested; we don't reinvent.
- **Apache-2.0 license.** No redistribution issues.
- **Capture+replay in one tool.** Many alternatives separate these concerns.
- **Modes map cleanly to our decision tree.** The spy mode (replay if matched, else forward) is exactly the hybrid behavior we want.

### Negative

- **Hoverfly's stateful-mutation handling is limited.** A POST followed by a GET to the same resource doesn't natively reconcile. Mitigation: Crucible's own state journal sits between Hoverfly and the agent, reconciling write-side mutations.
- **gRPC support is thinner than WireMock's.** Mitigation: pair Hoverfly + WireMock for gRPC-heavy customers.
- **Open-source vendor risk (smaller team).** Mitigation: SpectoLabs maintains the project well; if it ever falters, the OSS code is forkable and our PII-scrub layer is independent.

### Trade-offs we accept

We're betting on a single OSS project for a load-bearing layer. The Apache-2.0 license + active community means we can fork if needed; that's the safety valve.

## Alternatives considered

### Alternative 1: WireMock as primary

Mature, large community, strong JVM ecosystem. **Rejected as primary**:

- Mock-first model; capture-replay is bolt-on (Wiremock-Recorder).
- Java-centric; our team is Go/Rust/Python first.
- Used as a complement for gRPC, not the primary.

### Alternative 2: Speedscale (commercial)

SaaS service-virtualization with auto-detected dependencies and "responsive mocks." **Rejected as primary**:

- Commercial-only; our self-hosted enterprise tier can't bundle it without ongoing license complications.
- Vendor lock-in.
- Useful complement for customers who already use it, but not our default.

### Alternative 3: Mountebank

Multi-protocol (HTTP, TCP, SMTP), Node.js-based. **Rejected**:

- HTTP-only is fine for us; Mountebank's multi-protocol is overkill.
- Hoverfly's capture-replay is stronger.

### Alternative 4: GoReplay

Production traffic shadowing tool. **Rejected as primary**:

- Designed for shadow-testing production, not for offline replay against an agent.
- No native stub-on-miss.
- Useful for the initial-recording phase (capture from production); not for runtime replay.

### Alternative 5: Roll our own

Build a service-replay engine ourselves. **Rejected**:

- ~5+ agent-days that adds zero unique value.
- Hoverfly's behavioral surface is exactly what we need.

### Alternative 6: Mockoon

Lightweight, offline mock server. **Rejected**:

- Designed for solo developer mocking, not production-grade replay at scale.
- No record mode at the depth we need.

## The PII scrubber is the load-bearing addition

Hoverfly itself doesn't scrub PII. Our wrapper does:

```
Capture pipeline (at record time):
  HTTP/gRPC request/response
    ↓
  Presidio Analyzer (NER for PII)
    ↓
  spaCy NER (free-text PII catch)
    ↓
  FF3-1 format-preserving encryption (structure-bearing fields)
    ↓
  Deterministic pseudonymization (referential integrity)
    ↓
  Scrub audit log
    ↓
  Tape persisted
```

This is the regulated-buyer story. GDPR Art. 25 and HIPAA Safe Harbor both demand de-identification before prod-derived test data lands at rest. Hoverfly alone doesn't satisfy; the scrubber does.

## Open issues

- **Tape staleness detection.** When upstream services change their response shape, tapes silently lie. We need a tape-age metric and periodic re-capture pipeline; currently scoped for v2.
- **gRPC streaming support.** Hoverfly's gRPC streaming story is incomplete. WireMock is better for this; we use WireMock as a fallback in gRPC-streaming-heavy stacks.
- **LLM-synthesized response correctness.** When we synthesize for a miss, the response may be syntactically valid but semantically wrong. Currently mitigated via the `X-Crucible-Tape: synth-*` header so the verifier can weight it lower; long-term, fingerprint-and-improve from observed misses.

## References

- [01-architecture/twin-runtime.md#layer-4-service-twin-tapes](../01-architecture/twin-runtime.md)
- [06-research/tape-coverage-strategy.md](../06-research/tape-coverage-strategy.md)

---

<a id="file-05-decisions--adr-008-tier3-annotation-default-off"></a>

<!-- ================================================================== -->
<!-- File: 05-decisions/ADR-008-tier3-annotation-default-off.md -->
<!-- ================================================================== -->

# ADR-008: Tier 3 formal verification is auto-classified, not default-on

**Status:** Accepted  
**Date:** 2026-05-15

## Context

The verifier ladder includes Tier 3 — formal verification via Dafny, Lean, TLA+, Z3, Kani. Tier 3 is expensive (10–30 minutes wall-clock; non-trivial LLM cost) and not all code benefits. The question: how do we decide when Tier 3 runs?

Two extremes:

- **Default-on for all PRs.** Maximum safety; minimum speed; high cost. Customers will turn it off, defeating the purpose.
- **Manual annotation only.** Customers must add `@critical` to functions; tedious; under-applied in practice.

The pragmatic answer is auto-classification — a multi-signal scorer that classifies file/function criticality and triggers Tier 3 only for high-confidence-critical paths, with user override.

## Decision

Tier 3 is triggered by a **multi-signal classifier** combining:

- File-path heuristics (auth/billing/migration/etc. patterns).
- Import-graph centrality (PageRank, fan-in).
- Production-signal mining (incident post-mortems, SLO-backing endpoints, pager frequency).
- PR-review intensity history.
- CVE-touched files.
- Comment markers (`DANGER`, `// HACK`, etc.).
- LLM-judge categorical classification.

Scores combine via weighted sum (sigmoid-normalized) into bands:

| Band | Score | Behavior |
|---|---|---|
| Cold | 0–39 | Tier 1 only |
| Warm | 40–59 | Tier 2 |
| Hot | 60–79 | Suggest Tier 3 (one-click confirm in PR comment) |
| Molten | 80–100 | Auto Tier 3 + block merge until proof discharged or explicitly waived |

PR-level escalation:
- Touches any file with `S ≥ 80`, OR
- Touches ≥3 files with `S ≥ 60`, OR
- Modifies a function annotated `@critical`, OR
- Diff contains security/money tokens AND is ≥40 lines.

Overrides:
- `// crucible: not-critical` inline comment.
- `/crucible skip-tier3 reason:"..."` PR command (logged to procedural memory).
- CODEOWNERS designated approver override (weight 2×).

Every override becomes training data: classifier learns from corrections.

Tier 3 fallback on proof timeout: Tier 2.5 (exhaustive PBT + mutation + mandatory CODEOWNER review). Never fail open.

## Consequences

### Positive

- **Right-cost, right-rigor.** Critical code gets proofs; trivial code doesn't pay the latency cost.
- **No manual annotation friction.** Customers don't have to mark up their codebase by hand.
- **Self-improving.** Overrides train the classifier; over time the false-positive rate drops.
- **User-correctable in both directions.** Strict customers (defense, fintech) can tune up the escalation threshold; speed-focused customers can tune down.

### Negative

- **Classifier false-positive cost.** Tier 3 escalation on non-critical code wastes wall-clock and tokens. Bound by Tier 2.5 fallback; bounded by user override.
- **Classifier false-negative cost.** Missing a real critical-path change is more dangerous than over-escalating. Asymmetric cost weights bias the scorer toward over-escalation.
- **Initial calibration is hard.** Without customer-specific data, defaults are coarse. Mitigation: `crucible calibrate` command lets engineers label 200 stratified files; weights fit by logistic regression.
- **Tier 3 prover failures are operationally noisy.** Dafny/Lean timeout rates are non-trivial. Mitigated by Tier 2.5 fallback; tracked as a KPI per RB-10.

### Trade-offs we accept

We deliberately bias the classifier toward over-escalation. Customer pain from "Crucible escalated when we'd have shipped" is finite and survivable; customer pain from "Crucible let a Sev1 ship" is brand-existential.

## Alternatives considered

### Alternative 1: Tier 3 default-on for all PRs

**Rejected** — see context. Cost/latency unacceptable for general workloads; customers turn it off.

### Alternative 2: Tier 3 only on explicit `@critical` annotation (no auto-classification)

**Rejected**:

- Under-applied in practice. Most teams won't annotate; over time the annotations rot.
- Doesn't catch newly-introduced critical code.
- Doesn't catch critical code that's only critical *contextually* (high fan-in utility).

### Alternative 3: Tier 3 only on files in security-sensitive directories

Use path-only heuristics (`/auth/`, `/billing/`, etc.) without the full multi-signal scorer. **Rejected**:

- Misses contextually-critical code in non-obvious locations (the `utils/retry.ts` case where blast radius makes plumbing into a money path).
- Misses the production-signal dimension (which file paged on-call last quarter?).

### Alternative 4: LLM-judge alone

Use only the LLM-as-judge category classification, skip the heuristics. **Rejected**:

- LLM classification is noisier than the ensemble.
- Loses production-signal grounding.
- Costs more (per-file LLM call vs. cached heuristic compute).

### Alternative 5: Tier 3 ladder per language

Different policies per language. **Rejected for v1** — adds complexity without clear value; defer until we have data showing per-language tuning matters.

## Calibration plan

1. On install, the Cartographer runs a labeling-prompt on a stratified sample (200 files: 50 obvious-critical, 50 obvious-non-critical, 100 ambiguous).
2. A team engineer labels each as `critical | warm | cold | not-applicable`.
3. Logistic regression fits the per-tenant weight vector.
4. Defaults from the general OSS-trained model are used as priors.
5. Subsequent overrides (in production usage) update the weights via online learning.

## Open issues

- **Per-monorepo subdirectory tuning.** A monorepo may have wildly different code criticality between, say, the `marketing/` site and the `payments/` service. The classifier handles this via `file_glob` scope on conventions; v2 may add per-subdirectory weight overrides explicitly.
- **Cross-language Tier 3 tool gaps.** Dafny is general-purpose, Lean is math-heavy, TLA+ is distributed-invariants. Some critical code in less-mainstream languages (e.g., Elixir, Crystal) doesn't have a great Tier 3 tool. Fallback: Tier 2.5 with explicit warning.
- **Drift over time.** As the codebase evolves, the classifier's per-tenant weights need refreshing. Currently scheduled quarterly auto-recalibration.

## References

- [06-research/tier3-trigger-automation.md](../06-research/tier3-trigger-automation.md)
- [01-architecture/verifier-pipeline.md](../01-architecture/verifier-pipeline.md)

---

<a id="file-05-decisions--adr-009-anti-loop-protocol"></a>

<!-- ================================================================== -->
<!-- File: 05-decisions/ADR-009-anti-loop-protocol.md -->
<!-- ================================================================== -->

# ADR-009: Hard retry cap and bounded-budget enforcer

**Status:** Accepted  
**Date:** 2026-05-15

## Context

The single most expensive failure mode of 2025-era agents is infinite explore/thinking loops. Opus 4.6 specifically called out for this in published GitHub issues (#19699, #24585). Uber reportedly burned its full-year 2026 Claude Code budget in four months due in part to such loops. Individuals report $200/day burns from single stuck sessions.

The pattern: agent attempts subtask → fails → "reflects" → attempts the same flawed approach → fails → repeat. The agent has no self-imposed limit; the user has no visibility until the bill arrives.

## Decision

Crucible enforces three hard bounds, in-process, surfaced visibly to the user throughout:

### 1. Retry budget per subgoal: 3 attempts maximum

After 3 failed attempts at the same subgoal:
- The agent **cannot** retry the same subgoal.
- It must either (a) re-plan with a different approach (which the Plan Builder signs off on), or (b) halt and call `twin.plan.requestReplan(reason)` for human input.
- The retry counter is per-subgoal-identity (the goal description, not the raw tool call); same-approach retries count.

### 2. Dollar budget per plan: hard cap

- Every plan declares an `estimated_cost_usd`. Plan approval sets the budget cap (default = estimate × 1.5, or user-specified).
- Bounded Budget Enforcer tracks token spend in real-time.
- At 80% of budget: agent receives a warning; can request a budget extension via `twin.plan.requestReplan`.
- At 100% of budget: execution **halts**. Task moved to `budget_exceeded` state. User must approve continuation.

### 3. Wall-clock budget per task: hard cap

- Default: 60 minutes per task.
- Customer-configurable, max 4 hours.
- At cap: same behavior as dollar-budget exceeded.

All three bounds are visible in the task UI throughout execution: `[$0.31 / $1.00 budget — retry 1/3 — 4:31 / 30:00 elapsed]`.

## Consequences

### Positive

- **The Opus-4.6 loop class is architecturally eliminated.** Three retries; if no progress, halt and ask. No more $200 stuck sessions.
- **Customer trust in costs.** The plan shows an estimate; the bound caps the deviation. Cost is predictable to within 50%.
- **Forces agent to reflect strategically.** With only 3 retries, the agent must change approach on each retry, not iterate the same code.
- **Surface for user intervention.** When the bound fires, the user sees a structured "the agent is stuck on X" — opportunity to redirect, not silently fail.

### Negative

- **Some genuinely complex tasks need more than 3 retries to converge.** Mitigation: re-planning is the escape valve; the user explicitly approves a new plan with reset budget.
- **Budget estimates are noisy.** A bad estimate means a hard cap fires on a legitimate task. Mitigation: 1.5× headroom; learning loop refines estimates per-tenant over time.
- **Wall-clock cap may fire on Tier 3 proof-heavy tasks.** Mitigation: critical-path-flagged tasks get a higher default wall-clock cap (4 hours), and Tier 3 proofs have their own per-proof timeout that doesn't count against task wall-clock.

### Trade-offs we accept

- A small fraction of tasks (estimated < 5%) will hit a cap that a longer-running agent would have completed. We accept this; the alternative (no cap) is the documented disaster.
- Estimate-vs-actual divergence is a tunable; we err on the side of overshooting estimates so the cap doesn't fire spuriously.

## Implementation

The Bounded Budget Enforcer is a sidecar to the agent process, not a library the agent calls. It:

1. Subscribes to the cost-meter's per-call telemetry.
2. Tracks against the plan's caps.
3. Returns `BudgetExceeded` from `twin.*` SDK calls when caps reach.
4. Cannot be bypassed by the agent — no SDK method skips enforcement.

The retry counter is enforced at the task router level — when the agent restarts a step, the router checks the per-subgoal retry counter and refuses to re-dispatch.

## Alternatives considered

### Alternative 1: Soft warnings only, no hard cap

Show warnings; trust the agent to stop. **Rejected** — that's exactly what 2025-era agents do, and it produces the documented $200/day disasters.

### Alternative 2: Per-tool-call cap, not per-task

Cap each LLM call at $X. **Rejected**:

- Doesn't solve the loop problem (1,000 cheap calls = same end cost).
- Surface is the wrong level of abstraction; users plan in tasks, not calls.

### Alternative 3: Auto-extend budget on user opt-in

Configure tenant-level "budget extends automatically up to $Y." **Considered**; deferred to v2:

- Useful for heavy enterprise users who hate manual approvals.
- Not v1 because it's a foot-gun: customers will set $Y too high and hit bill-shock anyway.

### Alternative 4: Variable retry budget per subgoal complexity

Easier subgoals get fewer retries; harder ones get more. **Rejected for v1**:

- Complex to estimate; complexity is itself uncertain.
- Three retries is empirically right for the vast majority of subgoals.

### Alternative 5: Retry budget is a hint, not a hard cap

Allow agent to override with strong justification. **Rejected**:

- Agents always have a "strong justification" in their own reasoning trace.
- The whole point is the cap is non-negotiable.

## Customer-tunable parameters

Tenant config (in the web console / `crucible-cli tenant config set`):

```
retry_cap_per_subgoal: 3              # default; min 1, max 5
dollar_budget_multiplier: 1.5         # default; min 1.0, max 3.0
wall_clock_cap_min: 60                # default; min 5, max 240
auto_extend_on_progress: false        # opt-in; allows budget extension if progress detected
critical_path_wall_clock_cap_min: 240 # for tier-3-heavy tasks
```

## Observability

Every cap-firing event becomes:
- A task-state transition (`budget_exceeded`, `retry_cap_exceeded`, `wall_clock_exceeded`).
- A webhook event (`task.budget_exceeded`).
- An OTel span attribute (for cost-cap pattern detection).
- Visible in the customer's cost dashboard ("you hit the cap 14 times this week — consider re-planning approach").

A tenant repeatedly hitting caps is a customer-success signal: their workload pattern doesn't fit defaults, and we should reach out.

## References

- [01-architecture/system-overview.md](../01-architecture/system-overview.md)
- [03-sdk/agent-sdk-reference.md](../03-sdk/agent-sdk-reference.md)

---

<a id="file-05-decisions--adr-010-sigstore-rekor-attestations"></a>

<!-- ================================================================== -->
<!-- File: 05-decisions/ADR-010-sigstore-rekor-attestations.md -->
<!-- ================================================================== -->

# ADR-010: Sigstore Rekor v2 for transparency log

**Status:** Accepted  
**Date:** 2026-05-15

## Context

Crucible's promise of "verified, auditable, reproducible" requires a cryptographic audit trail. Every agent action — file write, tool call, shell command, plan approval, verifier verdict, promotion decision — must be:

1. Signed by an authenticated identity.
2. Published to a write-once, tamper-evident log.
3. Independently verifiable by any party.
4. Resilient to operator (Crucible) compromise.

## Decision

**Sigstore Rekor v2** is the default transparency log. Attestations follow the **in-toto attestation framework** with **DSSE** envelopes signed by **Sigstore keyless OIDC** (Fulcio-issued short-lived certs).

For each deployment:

- **SaaS tier:** public Sigstore Rekor v2.
- **Self-hosted enterprise:** self-hosted Rekor v2 + self-hosted Fulcio CA, bound to customer's identity provider.
- **Air-gapped:** same as enterprise; transparency log lives entirely on-prem.

Tier 4 (Honest CI) emits SLSA Provenance v1 in addition to Crucible-specific predicate types.

## Consequences

### Positive

- **Industry standard.** Sigstore is the de-facto signing infrastructure for open-source supply chains; customers' security teams already know it.
- **Keyless OIDC eliminates long-lived signing key management.** Each signing event mints a short-lived cert; no key rotation operationally.
- **Public-by-default.** SaaS tier attestations go to the public Rekor log — anyone can verify our customers' agent actions without our cooperation. Strong trust signal.
- **Replayability.** Every task's attestation chain reconstructs the full action sequence; debugging and audit become the same workflow.
- **SLSA-L3 by default.** Tier 4 emits the SLSA Provenance v1 predicate, which is exactly what regulated buyers want.

### Negative

- **Public log = public metadata.** SaaS-tier attestations expose the customer's action timestamps and OIDC subjects (not the code itself, but the existence of activity). Mitigation: customers can opt for self-hosted Rekor.
- **Sigstore dependency.** Public Rekor outages block attestation publishing. Mitigation: local journaling continues during outages; back-fill on recovery (RB-05).
- **Storage growth.** Rekor entries are append-only. Customer-side mirroring grows ~1MB/day for typical usage. Negligible; documented in retention policy.
- **OIDC issuer dependency.** Sigstore Fulcio depends on an OIDC issuer (GitHub, Google, custom). For self-hosted, the customer must run their own.

## Alternatives considered

### Alternative 1: Custom append-only Postgres table with hash chain

Implement a simple hash-chained ledger in Postgres. **Rejected as primary**:

- Reinvents Rekor poorly.
- No external verification; depends on Crucible operators not lying.
- Acceptable as solo-founder tier fallback when Rekor is overkill.

### Alternative 2: AWS QLDB

Append-only ledger with cryptographic verification. **Rejected**:

- AWS QLDB EOL'd 2025; no clean replacement narrative.
- AWS-locked.

### Alternative 3: Hyperledger Fabric / blockchain-style ledger

Use a permissioned blockchain. **Rejected**:

- Operational complexity wildly disproportionate to need.
- Customers' security teams have heard the word "blockchain" enough times that it's a sales-cycle slowing word, not accelerating.

### Alternative 4: Custom signed manifests + S3 Object Lock

Sign manifests with our own key, write to S3 with immutability. **Rejected**:

- Depends on long-lived signing keys (key-management overhead).
- No external transparency-log verification.
- Customer's compliance team would need to vet our key custody.

### Alternative 5: Signing only at promotion-time

Sign the final promotion bundle but not every intermediate action. **Rejected**:

- Doesn't enable replay / fork-from-step / blame.
- Doesn't catch attestation chain breaks mid-task.
- Misses the cost-accountability narrative ("every token spend traceable").

## In-toto predicate types Crucible defines

See [03-sdk/attestation-formats.md](../03-sdk/attestation-formats.md) for full schemas. Summary:

- `https://crucible.dev/WriteAttestation/v1` — file writes
- `https://crucible.dev/MigrationAttestation/v1` — DB migrations
- `https://crucible.dev/ServiceCallAttestation/v1` — service calls
- `https://crucible.dev/DestructiveProposal/v1` — intercepted destructive ops
- `https://crucible.dev/DestructiveApproval/v1` — approved destructive ops
- `https://crucible.dev/TestReport/v1` — verifier test runs
- `https://crucible.dev/VerifierApproval/v1` / `VerifierRejection/v1` — final verdicts
- `https://crucible.dev/PlanApproval/v1` — user plan approval
- `https://crucible.dev/PromotionBundle/v1` — promotion submissions
- `https://crucible.dev/PromotionApproval/v1` — gate decisions
- `https://crucible.dev/PromotionOutcome/v1` — final outcome (landed/rolled-back)
- `https://crucible.dev/MemoryWrite/v1` — procedural memory writes

## OIDC issuer chain

- **SaaS tier:** `accounts.crucible.dev` (our own issuer; runs on Dex).
- **Enterprise tier:** customer's existing IdP (Okta, Auth0, Azure AD, WorkOS, custom).
- **Air-gapped tier:** customer's on-prem IdP (Authelia, Keycloak, custom).

Crucible's own employee actions (deploy attestations, etc.) use Sigstore's standard `accounts.google.com` / `github.com` OIDC paths.

## Trust root rotation

Sigstore root rotation happens out-of-band per Sigstore's published schedule. We track and consume root updates within 30 days. For customer-controlled deployments (enterprise), the customer manages their own root and rotation schedule.

## Open issues

- **Rekor v2 GA stability** — v2 GA'd recently; one or two rough edges expected. Mitigation: pin to specific versions; backport bug fixes if needed.
- **Inclusion proof verification at scale** — Rekor witness verification has some latency at p99; not user-blocking but worth monitoring.
- **Post-quantum migration** — Sigstore uses standard ECDSA; PQC transition follows industry timeline (no v1 action).

## References

- [03-sdk/attestation-formats.md](../03-sdk/attestation-formats.md)
- [01-architecture/threat-model.md](../01-architecture/threat-model.md)

---

<a id="file-05-decisions--adr-011-no-built-in-ide"></a>

<!-- ================================================================== -->
<!-- File: 05-decisions/ADR-011-no-built-in-ide.md -->
<!-- ================================================================== -->

# ADR-011: Crucible integrates with existing IDEs via MCP/ACP; no proprietary IDE

**Status:** Accepted  
**Date:** 2026-05-15

## Context

Cursor, Windsurf, Antigravity, and Trae all built their own VS Code forks as the primary interface. The advantages: tight UX integration, brand surface, direct user telemetry. The costs: ongoing fork maintenance, upstream divergence, user-side switching friction.

MCP (Model Context Protocol) standardized in late 2024 and was donated to Linux Foundation in December 2025 — now universally adopted. ACP (Agent Client Protocol, from Zed) is gaining traction as the cross-editor agent-portability standard.

## Decision

Crucible **does not ship its own IDE.** Integration is via:

1. **MCP** — Crucible exposes its `twin.*` SDK as MCP tools. Any MCP-compatible host (Cursor, Claude Desktop, Zed via ACP, etc.) can drive a Crucible task.
2. **ACP** — for Zed and future ACP-compatible editors, same tool surface natively.
3. **CLI** — `crucible` standalone binary for non-IDE workflows (CI, scripts, Slack, GitHub Actions).
4. **Web console** — for plan approval, task monitoring, memory browsing, attestation viewing.
5. **GitHub App** — for PR-comment-driven invocation (`/crucible <description>`).

We ship thin plugins for the major IDEs that surface Crucible's specific UX needs (plan-approval modal, budget viewer, attestation chain explorer):

- VS Code extension
- JetBrains plugin
- Zed extension (uses ACP)

These plugins are ~3K LoC each — they wrap the MCP/ACP integration and add Crucible-specific UI affordances. They do **not** fork the IDE.

## Consequences

### Positive

- **No fork-maintenance tax.** We don't track VS Code upstream; we're a plugin.
- **Customer adoption friction near zero.** "Install this extension" beats "switch your editor."
- **IDE-agnostic value.** Customer's preference between VS Code, JetBrains, Zed, or terminal-only is preserved.
- **MCP momentum carries us.** Every new MCP-compatible host inherits Crucible support for free.
- **Smaller engineering surface.** Three plugins × 3K LoC = ~9K LoC vs. an entire IDE fork's ~150K LoC.

### Negative

- **No tight UX integration.** Cursor's Tab autocomplete is a Cursor-specific moat; we can't compete on that surface.
- **Less brand surface.** Users associate the work with their IDE, not with Crucible. Mitigation: plan-approval modal + cost preview prominently branded; attestation chain explorer is a Crucible-specific affordance.
- **MCP/ACP feature lag.** When MCP gains a new feature, we wait for hosts to adopt before we can use it.
- **Less direct user telemetry.** We see what the agent does, not what the user did before invoking the agent. Mitigation: track agent-side context (task description, files in context manifest, approval timing).

### Trade-offs we accept

We give up the Tab-autocomplete-style "spend hours in our IDE" mindshare to Cursor. We win by being the agent-of-record for verified deliverables — the thing that makes the merged PR matter, not the thing that helps you type faster.

## Alternatives considered

### Alternative 1: Fork VS Code

**Rejected**:

- ~150K LoC of ongoing fork maintenance.
- Customer switching cost.
- Cursor / Windsurf / Antigravity already crowd this space.
- Our brand is "verified output," not "fastest editing."

### Alternative 2: Build a web-only IDE

Bolt.new / v0 / Lovable model — entire dev environment in browser. **Rejected**:

- Different ICP (vibe-coders, not senior engineers).
- Browser dev envs hit JavaScript-only limits (WebContainer).
- Our ICP wants to keep their existing setup.

### Alternative 3: Terminal-only (CLI as primary interface)

Claude Code / Aider model. **Considered**; CLI is *one* of our surfaces but not the only one:

- Senior engineers love CLI; junior teammates and approvers don't.
- The plan-approval flow benefits from rich UI; CLI alone is awkward.
- Slack / GitHub / web console serve approval roles CLI can't.

### Alternative 4: Build a chat-only web interface

Devin / Replit Agent model — chat with the agent in browser. **Rejected as primary**:

- Replicates the existing chat-with-LLM surface; no differentiation.
- Pulls users out of their existing tools — switching cost.
- Web console exists as a complement, not the primary surface.

## What ships at v1

- VS Code extension (~3K LoC) — plan approval, budget viewer, attestation chain explorer, MCP wiring.
- Zed extension via ACP (~2K LoC) — same affordances.
- JetBrains plugin (~3K LoC) — same affordances.
- `crucible` CLI (~15K LoC across Go binary) — task submit/monitor, attestation verify, memory browse, runbook helpers.
- Web console (Next.js, ~50K LoC) — plan approval, task timeline, cost dashboard, memory browser, attestation viewer, approval inbox.
- GitHub App + Slack bot (~10K LoC) — PR-comment invocation, approval routing.

## What does NOT ship at v1

- Tab-autocomplete-quality inline completion.
- Cursor-Composer-style multi-file rewrite UI.
- Built-in chat panel (the IDE's chat panel is the chat panel).

These are not in scope. Our differentiation is verification + memory + provenance, not editing speed.

## References

- [02-engineering/repo-structure.md](../02-engineering/repo-structure.md)
- [03-sdk/tool-reference.md](../03-sdk/tool-reference.md)

---

<a id="file-05-decisions--adr-012-monorepo-structure"></a>

<!-- ================================================================== -->
<!-- File: 05-decisions/ADR-012-monorepo-structure.md -->
<!-- ================================================================== -->

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

---

<a id="file-05-decisions--adr-013-nix-for-tier4-builds"></a>

<!-- ================================================================== -->
<!-- File: 05-decisions/ADR-013-nix-for-tier4-builds.md -->
<!-- ================================================================== -->

# ADR-013: Nix flakes as default for hermetic builds

**Status:** Accepted  
**Date:** 2026-05-15

## Context

Tier 4 of the verifier ladder requires hermetic builds and SLSA-L3 attestations — bit-identical artifacts independently rebuildable. Three mainstream options for reproducible polyglot builds:

- **Nix (flakes)** — pure-functional package definitions, content-addressed store, hermetic by construction.
- **Bazel** — Google-pedigree build graph, hermetic when configured strictly, language-rules ecosystem.
- **Custom Docker + lock files** — pragmatic, fragile, not actually reproducible.

## Decision

**Nix flakes** is the default for Crucible's own builds and for customer Tier 4 verification.

Bazel is supported as an alternative for customers whose internal build system is Bazel-native (so they don't have to convert).

Custom Docker is explicitly not Tier-4-compliant.

## Consequences

### Positive

- **Hermetic by construction.** Nix's content-addressing means the build inputs uniquely determine the output. SLSA-L3 attestations are clean.
- **Polyglot-friendly.** Single config covers Go, Rust, Python, TypeScript, system deps. Bazel requires per-language rulesets that lag behind.
- **Air-gap-friendly.** Nix store is offline-friendly; the air-gap installer bundles needed paths.
- **Reproducibility verification is built-in.** `nix flake check` + `nix store verify` give bit-identical guarantees.

### Negative

- **Learning curve.** Nix is famously esoteric. Mitigation: thin wrapper scripts; the typical engineer only touches Nix when adding a new dependency.
- **Tooling rough edges.** Nix flakes is the "current best practice" but still has rough corners. Mitigation: pin to specific Nix versions; track release notes.
- **Build times can be slow without caching.** Mitigation: nix-cache shared across CI runners; per-PR diffs hit cache > 95% of the time.

### Trade-offs we accept

We trade Nix's onboarding pain for build hermeticity. This is the right trade for a SLSA-L3-default product; the senior-engineer ICP is mostly Nix-friendly, and the rest can rely on the wrapper scripts.

## Alternatives considered

### Alternative 1: Bazel as default

**Considered**, but:

- Polyglot Bazel rulesets (rules_python, rules_rust, rules_go, rules_nodejs) are uneven in quality.
- Bazel's hermeticity requires careful configuration; easy to get subtle non-hermetic builds.
- Larger learning curve than Nix for our mix.

Kept as a supported alternative for Bazel-native customers.

### Alternative 2: Custom Docker + Renovate-pinned base images

**Rejected for Tier 4**:

- Docker is not hermetic by default (entrypoint differences, base image patches, timestamp noise).
- SLSA-L3 requires bit-identical rebuilds; Docker fails this casually.

(Used for our *deployment* containers, which are built from Nix outputs. Hermetic at the source; OCI-packaged at the edge.)

### Alternative 3: Buck2 (Meta's Bazel-alternative)

**Rejected**:

- Smaller ecosystem than Bazel; less mature polyglot story.
- Adopting Buck2 doesn't give us reproducibility Nix doesn't already give.

### Alternative 4: Pants

**Rejected** for similar reasons — niche, smaller ecosystem.

## Practical setup

```
flake.nix (root)
  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable"
  inputs.rust-overlay.url = ...
  outputs = ...
  
  # Per-component derivations
  packages = {
    control-plane = buildGo {...};
    twin-runtime = buildRust {...};
    distiller = buildPython {...};
    web-console = buildTypeScript {...};
    ...
  };
  
  # Development shells
  devShells = {
    default = mkShell {...};  # all languages
    go-only = mkShell {...};
    rust-only = mkShell {...};
    python-only = mkShell {...};
  };
```

Engineers run `nix develop` to enter a hermetic shell with the right toolchain. CI runs `nix build .#release-bundle`.

## Reproducibility verification

Crucible's own CI uses **two independent build platforms** (GitHub-hosted runner + a self-hosted runner) to produce bit-identical artifacts. Comparison is automated:

```bash
nix build .#release-bundle.x86_64-linux --out-link platform-a
# (on other platform)
nix build .#release-bundle.x86_64-linux --out-link platform-b

diff <(nix hash file platform-a) <(nix hash file platform-b)
# Must match for SLSA-L3 attestation
```

Any divergence is a blocker. We've found and fixed divergences in:

- Timestamp embedding in Go binaries (`-trimpath` mandatory).
- Python `.pyc` timestamp embedding (`PYTHONDONTWRITEBYTECODE=1`).
- TypeScript build output ordering (`prefer-deterministic-bundling` in webpack/biome).

## Customer-side Tier 4 verification

Customers verify our releases:

```bash
crucible verify-release 2026.06.0
```

The command:

1. Pulls the published SLSA Provenance v1 attestation from Rekor.
2. Locally rebuilds the release from the source SHA pinned in the attestation.
3. Compares hashes.
4. Verifies Sigstore signatures chain to the published trust root.

This works *because* we use Nix. Without it, "reproducible" is a marketing claim.

## Open issues

- **Nix flakes "experimental" status.** Officially still flagged experimental in Nix 2.x. Mitigation: pin Nix version; track release notes; participate in the stabilization upstream.
- **Windows support is weak.** Nix on Windows is nascent. Mitigation: the Windows CLI builds via WSL2 + Nix; engineers on Windows use WSL2 dev shells.
- **Rust crates with non-Nix-friendly build.rs scripts.** Occasional issue; fixed case by case.

## References

- [02-engineering/repo-structure.md](../02-engineering/repo-structure.md)
- [02-engineering/testing-strategy.md](../02-engineering/testing-strategy.md)
- [03-sdk/attestation-formats.md](../03-sdk/attestation-formats.md)

---

<a id="file-05-decisions--adr-014-infisical-over-vault"></a>

<!-- ================================================================== -->
<!-- File: 05-decisions/ADR-014-infisical-over-vault.md -->
<!-- ================================================================== -->

# ADR-014: Infisical as default secrets vault

**Status:** Accepted  
**Date:** 2026-05-15

## Context

The twin runtime's secrets layer requires:

- **Dynamic, short-lived credentials.** Sub-minute TTL.
- **OSS self-host option.** Air-gapped enterprise tier must run without vendor connectivity.
- **Modern developer experience.** SDKs in our four languages, clean CLI.
- **Per-tenant scoping.** Each tenant's secrets isolated.
- **Reasonable cost at scale.** Per-tenant secrets management can multiply quickly.

The mainstream options in May 2026:

| Vault | OSS self-host | Dynamic secrets | Pricing | Notes |
|---|---|---|---|---|
| HashiCorp Vault Community | Yes | Yes (mature) | Free OSS / $1,150+/mo HCP Dedicated | HCP Vault Secrets EOL July 2026 |
| Infisical | Yes (OSS) | Yes (PG/MySQL/Mongo/etc.) | $8/user/mo Pro; free OSS | Modern DX |
| Doppler | No self-host | Limited dynamic | $7/user/mo Team | SaaS-only |
| AWS Secrets Manager + STS | No (AWS-only) | IAM session tokens | $0.40/secret/mo + API | Best in all-AWS |
| 1Password Connect | Yes | Limited | Per-user | Dev-friendly but limited dynamic |

## Decision

**Infisical** is the default secrets vault for Crucible.

- **SaaS tier:** Infisical Cloud (or our managed Infisical deployment).
- **Self-hosted enterprise:** Infisical OSS self-host.
- **Customer override:** customers with existing Vault investment can swap via values.yaml (`vault.provider: hashicorp-vault`).

For the production-promotion signing key (separate concern from twin secrets), we use **AWS KMS / GCP Cloud HSM / YubiHSM** per deployment — these handle HSM-backed signing for the unseal ceremony.

## Consequences

### Positive

- **Modern dev experience.** The SDK and CLI are pleasant; engineers adopt without complaint.
- **OSS self-host is real.** Air-gap installation works without licensing dramas.
- **Dynamic secrets across the engines we care about.** Postgres, MySQL, Mongo, Redis, custom — all supported.
- **Lower operational footprint than Vault.** Infisical is "vault for small/medium teams"; Vault is "vault for large enterprises with dedicated team."
- **Pricing scales sanely.** $8/user/mo on Cloud is reasonable; OSS is free.

### Negative

- **Younger company than HashiCorp.** More single-vendor risk; smaller community.
- **Some advanced Vault features missing.** Vault's auth-method ecosystem is broader (Vault has Kubernetes, AWS IAM, AppRole, LDAP, OIDC, JWT, GitHub, GCP, Azure, AliCloud, Kerberos auth methods out of the box). Infisical is catching up but not at parity.
- **Customer existing-Vault investment.** Some customers already run Vault; we support them via the override but our default is Infisical.

### Trade-offs we accept

We bet on the modern-DX project over the incumbent. The HCP Vault Secrets EOL July 2026 announcement created uncertainty about HashiCorp's roadmap for the "secrets-as-a-service" use case; Infisical's roadmap is cleaner for our needs.

## Alternatives considered

### Alternative 1: HashiCorp Vault Community as default

**Rejected as default** (kept as override option):

- Operational footprint heavy for what we need.
- HCP-EOL drama creates strategic uncertainty.
- Vault's dynamic secrets are mature, but Infisical is sufficient.

### Alternative 2: AWS Secrets Manager + STS

**Rejected as default**:

- AWS-locked. Multi-cloud and air-gap customers can't adopt.
- Reasonable for AWS-native customers; supported via override.

### Alternative 3: Doppler

**Rejected**:

- SaaS-only; no self-host story.
- Dynamic secrets less mature than Infisical's.

### Alternative 4: Roll our own

Build a minimal secrets engine. **Rejected** — pointless reinvention; commodity layer.

### Alternative 5: Cloud-native KMS only

Use AWS KMS / GCP Secret Manager + workload identity directly, skip a vault layer. **Rejected**:

- Couples the architecture to a specific cloud.
- Doesn't handle the twin-scoped ephemeral-credential pattern cleanly.
- Vault-like abstraction is the right level for our use.

## Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│  TWIN SANDBOX                                                       │
│  ┌─────────────────────┐         ┌──────────────────────────────┐  │
│  │  Agent process      │         │  Infisical sidecar           │  │
│  │  - calls twin.secret.get(name) │  - holds long-lived token to │  │
│  │  - receives SecretRef          │    Infisical (sidecar-only)  │  │
│  │  - never sees raw value        │  - issues dynamic, twin-     │  │
│  └──────────┬──────────┘         │    scoped, sub-min TTL token │  │
│             │                     └──────────────┬───────────────┘  │
│             ▼                                    │                  │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │  Egress proxy                                                │   │
│  │  - intercepts outgoing service calls                         │   │
│  │  - resolves $secret(name)$ placeholder to injected token     │   │
│  │  - logs which secrets used in which calls                    │   │
│  └─────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────┘
                              │
                              ▼  (real services)
```

The agent process **never sees secret values.** It holds opaque `SecretRef`s. At egress, the proxy substitutes the actual token. This is the architectural enforcement of secrets isolation.

For production promotion (entirely separate path), the **KMS-signed credential lease** is issued by AWS KMS / GCP Cloud HSM / YubiHSM directly to the deploy pipeline, never to the agent.

## Backup / disaster recovery

- Infisical Cloud: vendor-managed.
- Self-host: standard Postgres backup (Infisical's data lives in Postgres); customer responsibility.
- KMS keys: customer responsibility (typical AWS KMS / GCP / HSM rotation policies apply).

## References

- [01-architecture/twin-runtime.md#layer-5-secrets-twin](../01-architecture/twin-runtime.md)
- [01-architecture/promotion-contract.md](../01-architecture/promotion-contract.md)
- [01-architecture/threat-model.md](../01-architecture/threat-model.md)

---

<a id="file-05-decisions--adr-015-firecracker-via-e2b"></a>

<!-- ================================================================== -->
<!-- File: 05-decisions/ADR-015-firecracker-via-e2b.md -->
<!-- ================================================================== -->

# ADR-015: E2B (Firecracker) as default sandbox in SaaS tier

**Status:** Accepted  
**Date:** 2026-05-15

## Context

The twin runtime requires a sandbox per task that:

- Spawns fast (< 300ms perceived latency).
- Provides hardware-grade isolation (syscall-level, not just namespace-level).
- Supports overlayfs for COW filesystem.
- Supports network namespace + eBPF egress policy.
- Discards cleanly at task end.
- Scales horizontally with no operational pain.

May 2026 mature options:

| Tech | Cold start | Isolation | Operational complexity | Notes |
|---|---|---|---|---|
| Docker container | 100–500ms | Namespace + cgroup | Low | Insufficient isolation for hostile code |
| gVisor | 200ms | Syscall filter | Medium | 18% syscall overhead; good middle ground |
| Kata Containers | 130–200ms | Hardware (VM) | Medium-high | 130–200MB per pod; heavier |
| Firecracker | 110ms | Hardware (VM) | High (self-orchestrate) | Best isolation/perf ratio |
| E2B | 150ms | Hardware (via Firecracker) | Low (managed) | Vendor-managed, $0.0504/vCPU-hr |
| Daytona | 90ms | Container-based | Low | Docker under the hood |
| Modal Sandbox | 250ms | Hardware (Firecracker) | Low | GPU-capable; heavier |
| Cloudflare Workers | 5ms (V8 isolate) | V8 isolate | Low | Too restrictive for arbitrary code |

## Decision

- **SaaS tier:** E2B as default sandbox provider. Firecracker isolation, managed orchestration, sub-200ms cold start, $0.0504/vCPU-hr, mature SDK in Python and TypeScript.
- **Self-hosted enterprise:** Raw Firecracker + containerd + ZFS, orchestrated by our own scheduler. Marginal cost ≈ $0 per twin.
- **Solo-founder tier:** Daytona or Fly Machines as cheap entry points; degraded isolation guarantees but acceptable for a single-tenant indie-founder context.

## Consequences

### Positive

- **Hardware-grade isolation.** Firecracker boots a real microVM per task; syscall surface is constrained at the hypervisor boundary, not just by seccomp profiles.
- **Fast enough.** ~150ms cold start (E2B managed) doesn't dominate task latency budget.
- **Operationally simple in SaaS.** E2B handles orchestration; we focus on the agent layer.
- **Self-host path exists.** Firecracker is OSS (Apache-2.0); the air-gap installer can run raw Firecracker without E2B dependency.
- **Snapshot-restore is cheap.** Once warm, snapshot-restore is 3–10ms — enables checkpoint-and-fork for fan-out exploration.

### Negative

- **Vendor dependency on E2B for SaaS.** Mitigation: Modal Sandbox is the backup; our orchestration layer abstracts the specific provider via a `SandboxProvider` interface.
- **GPU support requires different provider.** E2B doesn't currently target GPU workloads; if customers need GPU-twins, route to Modal. Not a v1 ICP need.
- **Self-host operational burden.** Running raw Firecracker requires expertise. Mitigation: ship a curated Helm chart for the typical case; offer professional services for non-typical deployments.
- **24-hour max session.** E2B caps at 24h; we don't need longer (median task is minutes).

### Trade-offs we accept

We pay E2B for SaaS-tier convenience and absorb the operational cost of raw Firecracker for self-hosted. The split is correct — the SaaS tier should be operationally lean; the enterprise tier accepts complexity for compliance / data-residency.

## Alternatives considered

### Alternative 1: Docker containers (no microVM)

**Rejected**:

- Namespace isolation is insufficient for hostile-code threat model.
- Container escape vulnerabilities are a regular CVE class.
- The architectural commitment to "treat agent as hostile" demands hardware isolation.

### Alternative 2: gVisor

**Rejected as default** (kept as a possible budget option):

- 18% syscall overhead is meaningful for shell-heavy workloads.
- Isolation is better than plain Docker but not equivalent to Firecracker.
- Worth considering for the solo-founder tier where cost matters more than maximum isolation.

### Alternative 3: Kata Containers

**Rejected**:

- 130–200MB per pod is heavy.
- VM isolation similar to Firecracker but operational story heavier.

### Alternative 4: Self-orchestrate Firecracker from day one

**Rejected for SaaS**:

- ~2 agent-days of orchestration work just to match E2B's offering.
- E2B at $0.0504/vCPU-hr is cheap enough that the build vs buy doesn't justify build.
- Self-orchestration becomes necessary for the self-hosted tier anyway; we do it once for that, not twice.

### Alternative 5: Modal Sandbox

**Considered**; kept as fallback:

- $0.25/vCPU-hr is more expensive than E2B.
- GPU-capable, which is a v2 differentiator but not v1 need.

### Alternative 6: Cloudflare Workers

**Rejected**:

- V8 isolates are too restrictive — can't run arbitrary languages or binary tools.
- Wrong abstraction level for "agent runs a shell command."

## Provider abstraction

The `SandboxProvider` interface lets us swap providers:

```go
type SandboxProvider interface {
    Spawn(ctx context.Context, spec SandboxSpec) (*Sandbox, error)
    Snapshot(sandbox *Sandbox, name string) (*SnapshotRef, error)
    Restore(snapshot *SnapshotRef) (*Sandbox, error)
    Kill(sandbox *Sandbox) error
}
```

Implementations: `e2b`, `modal`, `daytona`, `fly-machines`, `raw-firecracker`. Per-tenant config selects the provider.

## Operational notes

### E2B SaaS

- API key per tenant for usage attribution.
- Crucible itself maintains a default "platform" account for shared infrastructure.
- Sandbox lifetime tied to task lifetime; orphan-detection sweeps every 5 min.

### Raw Firecracker (self-host)

- ZFS pool per node for snapshot speed.
- Pre-warmed sandbox pool (default 20 per node) to absorb burst.
- Per-tenant cgroup quotas to prevent neighbor effects.
- Network namespaces with Cilium / Tetragon eBPF policy.

### Backup providers in routing

If E2B is unreachable, fallback to Modal automatically. Customer-visible: "fallback provider in use" banner.

## Open issues

- **GPU workloads.** v1 ICP doesn't need them; v2 if ML-engineering customers materialize.
- **Windows containers.** Some legacy enterprise stacks require Windows VMs; not supported in v1.
- **macOS targets (iOS/macOS dev).** Mac-Cloud or MacStadium integration possible; not v1 scope.

## References

- [01-architecture/twin-runtime.md#layer-1-sandbox](../01-architecture/twin-runtime.md)

---


# 06. Research

<a id="file-06-research--memory-bootstrap"></a>

<!-- ================================================================== -->
<!-- File: 06-research/memory-bootstrap.md -->
<!-- ================================================================== -->

# Memory Bootstrap Strategy

Resolves open question #2 from the original architecture: how does procedural memory work on day 1 for a fresh customer with no PR history to learn from?

## The cold-start problem

A new Crucible install has no `(commenter, requested_change_type, code_pattern, accepted?)` data. The procedural memory graph is empty. Without intervention, the agent's day-1 output reflects model defaults (often "Tailwind Blue gradients and rounded corners aesthetic") — exactly the "generic AI aesthetic / convention drift" complaint we exist to solve.

## The four-tier seed corpus

Bootstrap procedural memory from public OSS sources, license-filtered, before the customer's own PR history accumulates. Tiers ranked by signal-to-noise:

### Tier A — Curated style guides (~40 documents)

Deterministic, authoritative, license-clean. Direct ingestion; no LLM re-interpretation needed beyond categorization.

- **Google style guides** — C++, Java, Python, Shell, TypeScript, JavaScript, R
- **Airbnb JavaScript / React style guide**
- **Microsoft TypeScript coding guidelines**
- **PEP 8 + PEP 257 + PEP 484 + PEP 526** (Python)
- **Effective Go + google/styleguide/go + Uber Go style guide**
- **Rust API Guidelines + tokio style notes**
- **Ruby/Rails style guide (bbatsov + rubocop/rails-style-guide)**
- **HackSoft Django Styleguide + Django coding-style docs**
- **Spring framework code-quality docs + spring-petclinic reference layout**
- **Phoenix/Elixir: christopheradams/elixir_style_guide + Credo defaults**
- **Swift API design guidelines (Apple)**
- **tum-esi/common-coding-conventions**

Weighted **×1.5 confidence** because these are authoritative.

### Tier B — Top-N OSS repos per stack (~2,400 repos)

Top 200 repos per major stack (12 stacks: Next.js, Django, FastAPI, Flask, Rails, Spring Boot, Go services, Rust services, Phoenix, Vue, Express, Laravel) by signal:

```
score = log(stars) × log(commits_last_90d + 1) × test_coverage_signal
filters:
  LICENSE in {MIT, Apache-2.0, BSD-*, MPL-2.0, ISC, Unlicense}
  has_ci
  has_codeowners_or_editorconfig
  not in {GPL-*, AGPL-*, SSPL-*, BUSL-*}
```

Extract:

1. **Lint configs** parsed deterministically (zero LLM cost):
   - `.editorconfig`, `.prettierrc`, `.eslintrc`, `tsconfig.json`
   - `.rubocop.yml`, `pyproject.toml` (ruff/black/isort)
   - `rustfmt.toml`, `clippy.toml`, `.golangci.yml`
   - `phpcs.xml`, `checkstyle.xml`, `.stylelintrc`, `.markdownlint.json`
   - `CODEOWNERS`, `commitlint.config.js`, `renovate.json`, `.gitleaks.toml`

2. **AGENTS.md ecosystem** — 60K+ repos by January 2026. Section-segment, LLM-categorize. The GitHub Blog's 2,500-repo analysis is the canonical pattern.

3. **CONTRIBUTING.md** — community-facing convention statements.

4. **`docs/architecture/`, `docs/adr/`** — ADRs, design rationale.

Expected yield: ~25K convention candidates after dedup.

### Tier C — PR review comment corpus (~300K diff-comment pairs)

Mine merged PRs from Tier-B repos, last 24 months, with ≥1 non-author review comment.

Filter aggressively:

- Drop "LGTM", "approved", trivial.
- Drop bot comments (dependabot, renovate, github-actions).
- Drop typo fixes.
- Min 20-char comment length.
- Comments that resulted in change (not just discussion).

Cluster by embedding (HDBSCAN); dense clusters become candidate rules. Target ~300K diff-comment pairs in, ~8K clusters out, ~3K surviving the cross-repo agreement threshold.

The LAURA dataset (arXiv 2512.01356, 301K diff-comment-info triples from 1,807 popular GitHub projects) is the directly-usable existing corpus.

### Tier D — ADR + post-mortem corpus (~5K records)

Smaller but very high signal — ADRs are *intentional* convention statements with rationale.

Sources:
- `joelparkerhenderson/architecture-decision-record` (largest curated set)
- Lullabot architecture decisions log
- `opendatahub-io/architecture-decision-records`
- Tier-B repos with `docs/adr/` directories
- Public post-mortem corpus (Increment, Honeycomb's "What I Learned" series, etc.)

Weighted **×1.5 confidence** because authoritative.

## The 12-category taxonomy

Conventions are categorized into 12 buckets matching AGENTS.md section heading conventions used by the top 2,500 repos:

| Category | Example rule |
|---|---|
| Naming | "Test files end in `_test.go` (Go) or `.test.ts` (TS)" |
| Layering | "Code in `db/` cannot import from `web/`" |
| Library preferences | "Use date-fns; don't introduce moment.js" |
| Test patterns | "Tests colocate with source in `__tests__/`" |
| Error handling | "Use Result<T,E> for fallible ops; no exceptions for control flow" |
| Logging | "Structured slog calls; no fmt.Printf in non-test code" |
| Migration patterns | "Migrations are additive-only; deprecation period >= 30 days" |
| PR/commit hygiene | "Conventional Commits; max 250-line diff" |
| Security defaults | "Auth middleware before any route handler" |
| Performance defaults | "Use cursor pagination, not offset" |
| Concurrency | "Pass context.Context through every async chain" |
| API shape | "Errors return { error: { code, message } } envelope" |

Each convention carries:

```typescript
Convention {
  id, scope (file_glob), confidence (0..1),
  rule_nl, category,
  positive_examples: SourceRef[], 
  negative_examples: SourceRef[],
  source: SourceRef[],
  first_seen, last_reinforced, last_violated,
  status: active | drifting | superseded,
  supersedes: ConventionId[]
}
```

## Stack-specific defaults

Per stack, a "day-1 ship-ready" bundle:

- **Rails** — `rubocop/rails-style-guide` + Rails Guides + `bbatsov/ruby-style-guide`. Highest signal density of any stack (Convention over Configuration).
- **Django** — HackSoft Django Styleguide + Django coding-style docs + Django contrib guide. Pair with ruff + black + isort defaults.
- **FastAPI** — `zhanymkanov/fastapi-best-practices` + tiangolo's docs + Pydantic v2 idioms.
- **Flask** — Pallets project docs + `cookiecutter-flask` patterns.
- **Next.js/React** — Vercel's `vercel/commerce` reference + `shadcn/ui` + Airbnb JS + react.dev hooks rules + Pages-vs-App-Router conventions.
- **Go** — Effective Go + `google/styleguide/go` + Uber Go Style Guide + golangci-lint defaults.
- **Rust** — Rust API Guidelines + clippy pedantic-lints (selective) + tokio style.
- **Spring Boot** — `spring-projects/spring-petclinic` reference + Google Java + Spring contributing.
- **Phoenix/Elixir** — `christopheradams/elixir_style_guide` + Credo + Phoenix Guides.

## Extraction pipeline

```
Public OSS Corpora ──▶ License filter (MIT / Apache-2.0 / BSD / MPL only)
                                │
                                ▼
                    ┌───────────────────────────┐
                    │ Deterministic config pass │
                    │ (lint configs → rules,     │
                    │  ~30% of conventions free) │
                    └─────────────┬─────────────┘
                                │
                                ▼
                    ┌───────────────────────────┐
                    │ LLM distillation pass     │
                    │ (Haiku 4.5, schema-fixed) │
                    │  textual corpus → typed   │
                    │  Convention candidates    │
                    └─────────────┬─────────────┘
                                │
                                ▼
                    ┌───────────────────────────┐
                    │ Cross-source agreement    │
                    │ embed → cluster → confidence│
                    │ confidence = log(distinct_repos_agreeing) /
                    │              log(repos_examined_in_stack) │
                    └─────────────┬─────────────┘
                                │
                                ▼
                    ┌───────────────────────────┐
                    │ Counter-example pass      │
                    │ find contradictions in    │
                    │ corpus; attach to rules   │
                    └─────────────┬─────────────┘
                                │
                                ▼
                    ┌───────────────────────────┐
                    │ Surface at install        │
                    │ confidence >= 0.4 → ACTIVE│
                    │ 0.25–0.4 → SUGGESTED      │
                    │ < 0.25 → CANDIDATE        │
                    └───────────────────────────┘
```

**Extraction model + prompt:**

```
Given this excerpt from {source_type: AGENTS.md | CONTRIBUTING.md | ADR | PR comment | style guide},
extract zero or more enforceable rules. Output JSON array of:
  { category, rule, file_glob, rationale, evidence_quote }
Emit nothing if no enforceable convention is stated.
```

Validated against schema (AdaKGC SDD pattern); retry once on validation failure, then drop.

## License / IP considerations

**Facts are not copyrightable; expression is.** We're extracting facts (which library, naming pattern, file structure) — analogous to a style guide author summarizing prior art.

Strict rules:

- **MIT / Apache-2.0 / BSD / MPL inputs:** safe to derive defaults; preserve attribution in `THIRD_PARTY_SOURCES.md`.
- **GPL / AGPL / SSPL / BUSL inputs:** *exclude entirely* from seed corpus. Even if extraction is arguably fair use, the downstream-redistribution exposure isn't worth it.
- **Code snippets in examples:** never ship verbatim OSS code as a positive example unless it's MIT/Apache, <10 lines, and attributed. Prefer LLM-paraphrased synthetic examples.

## Cross-tenant leakage prevention

Customer A's procedural memory must never leak to Customer B's agent.

- **Three-tier memory:**
  - `global_defaults` (from OSS seed corpus, shippable to all tenants)
  - `org_overrides` (customer-private, tenant-scoped)
  - `repo_overrides` (per-repo, lowest layer)
  - Agent reads bottom-up; only bottom two are tenant-scoped.

- **Generalization-upward rule:** customer-derived rules can graduate into `global_defaults` *only* when:
  - They appear in ≥ K independent customer tenants (K = 5 minimum)
  - The rule is anonymized to its category form (e.g., "prefer webhooks over polling for payment-provider integrations" — never "use Stripe webhooks")

- **Embedding-space isolation:** never share embeddings of customer-private rules across tenants. Per-tenant namespaces in the vector store (pgvector RLS, Qdrant per-tenant collections).

- **Differential privacy** on cross-tenant aggregate signals if/when published.

## First-week ingestion plan (concrete)

A specific runnable schedule for bootstrapping a fresh deployment:

| Day | Task | Yield |
|---|---|---|
| 1 | License gate + deterministic configs (top 200 repos per 12 stacks) | ~6K rules |
| 2 | AGENTS.md / CONTRIBUTING.md from Tier-B + 60K AGENTS.md universe, rate-limited | ~25K candidates |
| 3 | Style guides (~40) + ADR corpus (joelparkerhenderson + Lullabot + opendatahub) | ~4K rules + ~2K decisions |
| 4 | PR review comment mining (Tier-B repos, 24 months, GraphQL API) | ~3K rules surviving agreement |
| 5 | Cross-source agreement + confidence assignment | merged catalog |
| 6 | Stack-defaults packaging — emit per-stack JSON bundles | per-stack ship-ready bundles |
| 7 | Override mechanism + drift detection wiring | full system operational |

Day-1 customer experience on a fresh Next.js + FastAPI monorepo:

```
✓ ~400 active rules
✓ Correctly scoped by file glob
✓ Carrying rationale + source URLs
✓ Agent visibly cites "OSS consensus" vs "your team's rule"
```

## Override mechanism

Customer-supplied `AGENTS.md` / `CLAUDE.md` / `.cursorrules` at repo root **always** wins over defaults:

- Matched by rule-id where overlap exists.
- New customer rules added on top.
- Default rules contradicted by customer rules are demoted to `superseded` with reference to the customer override.

The Cartographer ([04-operations/onboarding.md](../04-operations/onboarding.md)) generates an inferred AGENTS.md from the customer's repo, presents it for review, and uses the result as the seed customer-override layer.

## Drift detection on defaults

Defaults age. Strategy:

- Every 30 days, re-extract from the seed repos.
- If a rule's cross-repo agreement drops > 20%, demote confidence.
- If a new contradictory rule passes threshold, mark the old as `drifting`.
- Customer-facing: "Your default rule X is aging; suggested update: Y."
- Maintain `last_validated` timestamp per rule; auto-archive rules unvalidated for 180 days.

## What we honestly don't solve at v1

- **Multi-language convention conflicts.** Rails conventions don't apply to FastAPI but our extraction may bleed. Mitigation: stack-tagging at extraction.
- **Anti-pattern of OSS defaults dictating taste.** Customers in unusual contexts (game dev, embedded, ML) may find OSS web-app conventions wrong. Mitigation: per-stack bundles; opt-out via empty seed-rule flag.
- **Stale corpus.** OSS practice evolves. The quarterly re-extraction handles slow drift; rapid shifts (e.g., new framework release) may need manual curation triggered.

## References

- [01-architecture/memory-layer.md](../01-architecture/memory-layer.md)
- [04-operations/onboarding.md](../04-operations/onboarding.md)
- [ADR-003: Procedural memory moat](../05-decisions/ADR-003-procedural-memory-moat.md)

---

<a id="file-06-research--tape-coverage-strategy"></a>

<!-- ================================================================== -->
<!-- File: 06-research/tape-coverage-strategy.md -->
<!-- ================================================================== -->

# Tape Coverage Strategy

Resolves open question #1 from the original architecture: what % of agent service calls hit endpoints that ARE in the tape vs not, and what's the right behavior for the misses?

## The empirical baseline

Production API endpoint hit frequency follows a Zipf-like distribution with exponent α typically 0.8–1.2:

- **Alibaba microservice trace (SoCC '21, 20K microservices):** "super microservices" with in-degree ≥ 16 appear in 90% of call graphs and handle 95% of total invocations.
- **Twitter open cache traces (54 clusters, March 2020):** popularity highly skewed.
- **CDN literature:** healthy cache hit rates are 95–99% for static, 20–60% for dynamic personalized.

For coding agents specifically (Cursor, Claude Code, etc.), per-task external call patterns:

- 5–20 distinct external endpoints per feature task.
- 40–200 total calls including retries and pagination.
- Clustered on a handful of read endpoints (e.g., `GET /customers/{id}`, `GET /charges`) and a smaller number of writes.

**Implication for Crucible:** recording the top 5–10% of endpoints by call volume per service covers 80–95% of agent task traffic on read paths. The remaining 5–20% is the long tail — and in agent tasks it correlates strongly with novel/feature-specific work, which is precisely the work where the tape must NOT silently lie.

## How existing tools handle misses

| Tool | Miss strategy |
|---|---|
| Hoverfly | Modes: capture, simulate, modify, spy, synthesize. Spy = replay-or-passthrough. |
| WireMock | Default = HTTP 404. Catch-all low-priority stub. |
| VCR | `:once` / `:new_episodes` / `:none` / `:all` record modes |
| Polly.JS | record / replay / passthrough; per-request `.passthrough()` opt-in |
| Speedscale | "Responsive mocks" with state — replicates production behavior |
| GoReplay | Capture-and-replay; no built-in stub-on-miss |

The mature pattern across these tools: **three primitives — strict replay, replay-or-passthrough, record-on-miss — per session.** Crucible adopts the same primitives but selects per request class, not globally.

## The Crucible decision tree

On every outgoing request from the twin, in priority order:

```
1. Match tape entry EXACTLY (path + method + sig)
     → REPLAY, tag X-Crucible-Tape: hit-exact

2. Match tape entry by TEMPLATE (path pattern + method,
   differing only in IDs / pagination / timestamps)
     → REPLAY with parameter substitution
     → tag X-Crucible-Tape: hit-template
     → confidence: high

3. Miss, but endpoint is in OpenAPI spec
   AND method is read-only (GET / HEAD / OPTIONS)
     → SYNTHESIZE response from schema
       (Prism / Microcks-style Faker + optional LLM augmentation)
     → persist as CANDIDATE tape entry (not auto-promoted)
     → tag X-Crucible-Tape: synth-readonly

4. Miss, endpoint in spec, MUTATING method (POST/PUT/PATCH/DELETE)
     → DETERMINISTIC STUB: spec's default success example
     → RECORD mutation to twin's in-memory state journal
     → NEVER forward to real service
     → tag X-Crucible-Tape: synth-mutation
     → surface to agent: "Mutation simulated; not live."

5. Miss, NOT in spec, task manifest declares live-call allowed for host
     → PASSTHROUGH through PII-scrubbing egress proxy
     → persist response to tape for future runs (VCR :new_episodes)
     → tag X-Crucible-Tape: live-passthrough

6. Miss, NOT in spec, live NOT allowed, request requires auth
     → FAIL CLOSED with 599 Crucible-Tape-Miss
     → structured error body describes what was missing
     → agent sees the error and adapts
     → tag X-Crucible-Tape: miss-blocked

7. Miss, NOT in spec, live NOT allowed, no auth required
     → Policy-driven; default FAIL CLOSED with 599
     → optional per-task override to synth-from-shape
```

## Policy knobs surfaced to users

```
tape.mode             = strict | hybrid | adaptive
tape.synth_engine     = none | schema | schema+llm
tape.allow_live       = [host_allowlist]
tape.mutation_policy  = journal | block
tape.miss_status      = 599
```

Defaults: `hybrid + schema+llm + [] + journal + 599`.

The `X-Crucible-Tape` response header is **the single most important design decision**: agents AND the verifier both *see* whether a response was real, replayed, or synthesized, and weight trust accordingly.

## Auth handling on replay

- **On replay:** match requests after stripping `Authorization` header.
- **On passthrough:** egress proxy injects sandbox-tenant token (not real prod creds).
- **Production tokens never leave the twin.** Twin-scoped Infisical credentials only.

## State-mutating calls

Mutations are **never** replayed as having had effect on real systems. The flow:

1. Agent calls `POST /v1/charges`.
2. Decision tree: synth-mutation (stub success response).
3. Mutation written to twin's in-memory state journal: `{charges: [{id: ch_synth_1, amount: 1234}]}`.
4. Subsequent reads consult the state journal first, then fall through to the tape.

This is the Speedscale "responsive mock" pattern done right. Speedscale's product gestures at it but doesn't ship it cleanly; we make it deterministic.

## PII scrubbing — at capture, not replay

GDPR Art. 25 (data minimization) and HIPAA Safe Harbor (18-identifier list) make it clear: prod-derived test data without de-identification is non-compliant. PCI-DSS pulls raw PAN-bearing data into CDE scope.

Capture pipeline:

```
HTTP/gRPC request/response received
    │
    ▼
Microsoft Presidio Analyzer + Anonymizer
    │ (NER for: names, SSN, credit cards, phones, addresses, emails, MRNs)
    ▼
spaCy NER (free-text fields, response bodies)
    │
    ▼
FF3-1 Format-Preserving Encryption (mysto/python-fpe or Vault transform)
    │ (BINs, phone formats, account-number checksums — structure-bearing fields)
    ▼
Deterministic pseudonymization (per-tape-set key)
    │ (preserves referential integrity: cus_abc → cus_zzz consistently)
    ▼
Synthetic augmentation (Gretel / SDV / MOSTLY AI)
    │ (preserves distributional properties; Jensen-Shannon < 0.1 typical)
    ▼
Audit log (which scrubbers fired, which fields rewritten)
    │
    ▼
Tape persisted (content-addressed by request_hash)
```

Scrubbing must happen **at capture**, before bytes hit disk. Scrubbing on replay is too late — the bytes already exist.

## Tape lifecycle

- **TTL:** 90 days default unless explicitly pinned.
- **Per-tenant storage quota.**
- **LRU eviction** when quota reached.
- **Versioning:** tapes are content-addressed; a service upgrade that changes response shape creates new tape entries; old entries remain available for historical replay but flagged stale after 30 days.
- **Re-capture:** customers configure periodic re-capture schedules; we automate the shadow-recording where they grant permission.

## Tape staleness — the irreducible problem

When upstream service ships a breaking change, tapes silently lie. Mitigations:

1. **Tape-age metrics** surfaced to agents and the verifier (per-endpoint `last_recorded` timestamp).
2. **Promotion canary catches lying tapes** because the canary hits real services, not the tape.
3. **Auto-rollback on canary regression.**
4. **Periodic re-capture cron.** Customer-configurable; default monthly for high-traffic endpoints.
5. **Tape staleness warning** in PR descriptions: "this PR was verified against a tape last refreshed 47 days ago; consider re-recording."

We cannot eliminate this risk entirely. The honest design choice is to expose it.

## Cold-start: brand-new endpoint, no recording

When the agent's task touches an endpoint we've never seen:

1. Check OpenAPI spec for the service. If present → synth-readonly or synth-mutation per decision tree.
2. If no spec and not in `allow_live`: fail closed.
3. If no spec and `allow_live`: passthrough (one-shot capture for future).

The cold-start case is irreducibly worse than the warm-tape case. Customers should populate tapes aggressively in onboarding; we surface tape-coverage metrics in the dashboard.

## Honest assessment of what's unsolved

- **Stateful replay across mutations.** State journal handles short tasks well; long sessions degrade as the journal diverges from "what real prod would have done."
- **Semantically wrong synth responses.** A Faker address is valid JSON but won't pass real address-validation. Agent may take wrong actions based on synth.
- **Free-text PII in JSON.** Presidio + spaCy miss ~5–15% of free-text PII (no public benchmark for adversarial test).
- **Long-tail coverage on novel tasks.** A genuinely novel feature is, by definition, hitting endpoints not yet popular enough to record. First-run of a new feature is where tape fails most.
- **Per-task call-count telemetry for coding agents not publicly published.** Our 5–20 distinct / 40–200 total estimate is from inference. We should instrument and publish.

## Customer-facing onboarding

The recommended setup:

1. **Day 1 of install:** point the shadow-capture agent at staging. Capture for 7 days. Scrub. Result: tape for top 80% of endpoints.
2. **Week 2:** review the scrub-audit report; confirm scrubbing matches compliance requirements.
3. **Week 3:** start running real agent tasks. The first 10% of tasks will hit fail-closed misses; agent reflects, asks for tape extension or live-allow. User accepts or denies; tape grows.
4. **Month 1:** tape coverage stabilizes at ~95%+ for the customer's workload pattern.

We bill onboarding cost (shadow capture, scrub compute) as a one-time setup fee for Team / Enterprise tiers, absorbed for Pro.

## References

- [01-architecture/twin-runtime.md#layer-4-service-twin-tapes](../01-architecture/twin-runtime.md)
- [ADR-007: Hoverfly tape replay](../05-decisions/ADR-007-hoverfly-tape-replay.md)

---

<a id="file-06-research--tier3-trigger-automation"></a>

<!-- ================================================================== -->
<!-- File: 06-research/tier3-trigger-automation.md -->
<!-- ================================================================== -->

# Tier 3 Trigger Automation

Resolves open question #3 from the original architecture: how do we auto-classify which code paths are `@critical` so users don't have to annotate everything by hand?

## What "critical" means (six orthogonal axes)

Critical is a multi-dimensional predicate, not binary. Different axes imply different Tier 3 tools:

| Axis | Examples | Tier 3 tool |
|---|---|---|
| Performance-critical (hot path) | p99-dominating code, inner loops | TLA+ for concurrency; Z3 for invariants |
| Security-sensitive | authn/authz, crypto, deserialization, input validation | CBMC for memory safety; SAW for crypto; Dafny for state machines |
| Money paths | billing, refunds, ledgers, currency conversion, idempotency | Dafny or Lean (clean math invariants) |
| Data-integrity | migrations, replication, leader election, audit logs | TLA+ (AWS DynamoDB precedent) |
| Safety-of-life / regulatory | medical (IEC 62304), automotive (ISO 26262 ASIL-D), aviation (DO-178C-A) | Tool dictated by certification |
| Blast radius (centrality) | shared utilities imported by 100+ modules | Tool depends on actual function |

The sixth axis — blast radius — is critical for non-obviously-critical code. A bug in `utils/retry.ts` with fan-in 230 is more dangerous than a bug in `billing/edge_case_handler.ts` that's only called from one place.

## Existing tools we draw signal from

| Family | Tools | What we use |
|---|---|---|
| SAST severity | Semgrep p/security-audit, CodeQL severity scores, SonarQube hotspots, Snyk Code priorityScore | Severity tag → criticality input |
| Dependency scanners | Trivy, Grype, OWASP Dependency Check, Dependabot | CVE-touched files signal |
| Ownership / tier metadata | Datadog Service Catalog tier, Backstage criticality, CODEOWNERS, PagerDuty service-criticality | Direct critical-path signal |
| Hotspot detection | CodeScene, Bridgecrew, GitGuardian | Churn × complexity hotspots |

Crucible's classifier *consumes* these tools' outputs rather than re-implementing them.

## Production-signal mining

Higher fidelity than static heuristics because observed:

- **Incident post-mortems.** Parse Rootly / FireHydrant / Jeli exports, Confluence / Notion postmortem pages. Run NER + file-path regex. Files mentioned in 3+ Sev1/Sev2 in last year are unambiguously critical.
- **SLO / error-budget data.** Map endpoints to source functions via OpenTelemetry semantic conventions (`code.filepath`, `code.function` span attributes). Endpoints attached to SLOs ≥99.9% promote their backing functions.
- **Pager frequency.** PagerDuty → JIRA → git-blame chain. "Files blamed by alerts that paged ≥2 engineers in 90 days."
- **PR review intensity.** Files attracting ≥3 distinct reviewers, PRs with ≥20 comments, PRs blocked (`REQUEST_CHANGES`) >30% of the time.
- **Test coverage gradient.** Files in 95th percentile of `coverage_lines / sloc` — engineers wrote disproportionate tests for a reason.
- **Churn-vs-review ratio.** High churn + low review = risky but underwatched. High churn + high review = critical and watched.

## Path-pattern heuristics

The cheap baseline. Crucible ships default regex sets, namespaced by axis:

```regex
SECURITY:   \b(auth[nz]?|oauth|saml|jwt|session|login|signin|password|
            secret|token|cred|kms|kdf|crypto|cipher|sign|verify|hash|
            mtls|tls|x509|csrf|cors|sanitiz|escape|validate|permit|
            rbac|acl|policy|capabilit|sandbox)\b/i

MONEY:      \b(billing|invoice|payment|payout|refund|charge|subscri|
            ledger|account(ing)?|balance|currency|fx|tax|vat|gst|
            stripe|adyen|braintree|paypal|wallet|escrow|settle)\b/i

DATA:       \b(migrat|schema|replicat|snapshot|backup|restore|
            audit_?log|gdpr|pii|consensus|raft|paxos|leader|quorum|
            checksum|wal|journal|fsync)\b/i

SAFETY:     \b(asil|sil[1-4]|do178|iec6\d{3}|hipaa|hitrust|fda|
            iso26262|misra|safety|interlock|estop|failsafe)\b/i

HOTPATH:    \b(hot|fast_?path|inner_?loop|simd|vectoriz|kernel)\b/i
```

## Comment-marker mining

Grep code + docstrings for risk markers:

```
DANGER, DO NOT TOUCH, HERE BE DRAGONS,
// HACK, FIXME critical, XXX security,
@critical, @dangerous, WARNING:, TODO(security),
SECURITY:, THREAD-SAFETY:, INVARIANT:
```

Files with ≥2 markers are candidates.

## Import-graph centrality

Build static call/import graph via tree-sitter + language-specific resolvers (pyan, jdeps, go-callvis, ts-morph). Compute:

- **Fan-in:** distinct modules importing this one.
- **PageRank** on the call graph.
- **Articulation-point status:** does removing this node disconnect a subgraph?

Top 5% by PageRank or fan-in ≥ 50 → critical regardless of subject matter. This is what catches `utils/retry.ts`.

## CVE recency

Files touched by any CVE patch in last 24 months (extractable from `git log --grep='CVE-'` + OSV-DB) get a permanent boost.

## Test-name harvesting

Functions referenced by tests named `test_security_*`, `test_critical_*`, `test_*_invariant`, `test_*_property`, or wrapped in `@pytest.mark.critical` / `[Trait("Category", "Critical")]` inherit the tag.

## LLM-based classification

A small LLM judge (Haiku 4.5, cached by content hash) categorizes each candidate file:

```
Classify this code into one of:
  {security, money, data-integrity, safety, performance,
   infrastructure, ui, plumbing, test, dead}.
Return JSON:
  {category, confidence (0..1), reasoning (one sentence)}.
```

Cached aggressively. Re-runs free. Temperature 0; ensemble 3 calls when confidence < 0.7.

LLM judges catch context heuristics miss: `validator.py` could be input validation (critical) or UI-form schema validation (non-critical). Only semantic reading distinguishes.

## The weighted multi-signal score

```
S(file) = 100 * sigmoid(
    1.5 * path_pattern_score        // 0..1, max of axis regex matches
  + 1.2 * llm_category_score        // 0..1, weighted by confidence
  + 1.0 * fanin_centrality          // log-normalized PageRank
  + 0.9 * incident_mention_score    // postmortem hits, decayed
  + 0.8 * slo_backing_score         // 1.0 if backs ≥99.9 SLO
  + 0.7 * review_intensity_score    // reviewers + comments/PR
  + 0.7 * cve_history_score         // 1.0 if CVE-touched in 24mo
  + 0.6 * test_coverage_gradient    // z-score within repo
  + 0.5 * comment_marker_score      // DANGER/HACK density
  + 0.4 * codeowners_team_score     // owned by sec/payments/sre
  - 0.5 * ui_or_test_penalty        // pure UI/test files lose points
)
```

### Threshold bands (defaults; per-tenant tunable)

| Band | Score | Behavior |
|---|---|---|
| Cold | 0–39 | Tier 1 only (lint + type-check + diff-scoped mutation) |
| Warm | 40–59 | Tier 2 (property tests + mutation) |
| Hot | 60–79 | Suggest Tier 3 (one-click confirm in PR comment) |
| Molten | 80–100 | Auto Tier 3 + block merge until proof discharged or waived |

## Calibration

The default weights above are starting points. The actual weights are tuned per-tenant.

`crucible calibrate` command:

1. Cartographer samples 200 files stratified across the score distribution (50 obvious-critical, 50 obvious-non-critical, 100 ambiguous).
2. A team engineer labels each: `critical | warm | cold | not-applicable`.
3. Logistic regression fits the weight vector against labels.
4. Defaults from the general OSS-trained model serve as priors.
5. Online learning thereafter: every override (in production usage) updates weights.

Calibration takes ~1 hour of human time; pays for itself in reduced false-positive Tier 3 escalation within a week.

## Asymmetric cost

False-positive cost ≈ 20 min of CI + engineer annoyance.
False-negative cost ≈ a Sev1 in production.

Ratio: ~1:1000. The scorer is **biased toward over-escalation**. Combined with a cheap override path, over-escalation doesn't poison adoption.

## PR-level trigger

A PR auto-escalates to Tier 3 when ANY of:

1. Touches any file with `S ≥ 80`.
2. Touches ≥3 files with `S ≥ 60`.
3. Modifies a function annotated `@critical` (explicit or inherited).
4. Diff contains security/money regex tokens AND is ≥40 lines.

## Override flow

Three mechanisms, all recorded for learning:

1. **Inline comment:** `// crucible: not-critical` (or `# crucible: not-critical` per language).
2. **PR command:** `/crucible skip-tier3 reason:"..."` — requires reason string, logged to procedural memory.
3. **CODEOWNERS designated approver:** `@security-team` or `@principal-eng` approval of the skip carries weight 2× a normal override.

Every override becomes a training example: `(file_features, true_label=non_critical, overridder, reason)`.

Conversely: a non-escalated PR followed by an incident touching its files within 30 days is a hard negative — boost weights that *would have* caught it.

## Confidence-driven UX

Platt-scaled probability determines the UI:

| P(critical) | UI |
|---|---|
| ≥ 0.9 | Silent auto-escalate; surfaced in PR as a checkbox pre-ticked |
| 0.6–0.9 | PR comment: "I think this needs Tier 3 because: [top-3 signals]. Confirm?" |
| 0.3–0.6 | Foldable suggestion; no friction |
| < 0.3 | Silent |

## Auto-annotation of functions

A function gets `@critical` auto-annotated when:

1. Lives in a file with `S ≥ 70`.
2. AND any of:
   - Is exported/public.
   - Called by ≥2 files outside its module.
   - Handles untrusted input (parameter type matches `Request`, `bytes`, `str` from network sources).

The annotation persists in `.crucible/annotations.toml` (sidecar file), surviving refactors that move the function.

## Tier 3 timeout fallback

Proofs are slow; Tier 3 timeouts happen. Crucible does **NOT fail open**:

1. **Wall-clock budget:** Dafny 10 min, Lean 30 min, TLA+ model-check 20 min.
2. **On timeout, degrade to Tier 2.5:**
   - Exhaustive PBT (≥10,000 cases)
   - Mutation testing on the diff
   - **Mandatory CODEOWNER human review.**
3. **Cache partial proofs.** Incremental verification on next PR resumes where it left off.
4. **Surface to dashboard.** Chronic Tier 3 timeouts on the same code path are a signal to invest in proof engineering.

## Examples

**Obvious critical:** `src/auth/oauth_callback.py` (path match: auth, LLM category: security, fan-in 12, CVE history 2 in 18mo, owned by `@security-team`) → `S ≈ 92` → Molten.

**Obvious critical:** `services/billing/refund_engine.go` (path: billing + refund, SLO-backing 99.95% revenue endpoint, postmortem mentions 4, review intensity 3.2 reviewers/PR) → `S ≈ 88` → Molten.

**Obvious non-critical:** `web/components/MarketingHeroBanner.tsx` (UI penalty, LLM category: ui, fan-in 1, no security keywords) → `S ≈ 8` → Cold.

**Genuinely ambiguous (the load-bearing case):** `lib/utils/retry.ts` — small, plumbing-looking, but fan-in 230. Path heuristics say low; centrality says very high. Score lands `S ≈ 64` → Hot (suggest Tier 3). A bug in retry.ts that double-charges on retry is a money path even though nothing in its filename says "money."

**Adversarial mislabel:** `tools/payment_simulator_for_demos.py` — heuristic says money, LLM judge correctly flags as demo simulator, dropping score < 40.

This last example is the load-bearing argument for the ensemble — no single signal layer is sufficient.

## Open issues

- **Per-monorepo subdirectory tuning.** A monorepo may have wildly different criticality between `marketing/` and `payments/`. Handled by file_glob scope; v2 may add explicit subdirectory weight overrides.
- **Cross-language Tier 3 tool gaps.** Elixir, Crystal, etc. have weak Tier 3 tooling. Fallback to Tier 2.5 with explicit warning.
- **Calibration data freshness.** Weight vectors age as codebase evolves. Quarterly auto-recalibration scheduled.

## References

- [01-architecture/verifier-pipeline.md](../01-architecture/verifier-pipeline.md)
- [ADR-008: Tier 3 annotation default-off](../05-decisions/ADR-008-tier3-annotation-default-off.md)

---

<a id="file-06-research--unit-economics"></a>

<!-- ================================================================== -->
<!-- File: 06-research/unit-economics.md -->
<!-- ================================================================== -->

# Unit Economics

Resolves open question #4 from the original architecture: can per-verified-PR pricing work when the architecture uses both an executor model AND a separate-family verifier?

## TL;DR

The "2× tokens" worry is wrong in practice. Verification runs **once at the end of a task** (not in-loop), so total cost rises ~8%, not 2×. With aggressive 1h prompt caching and the right cross-family routing, median task cost lands at **~$1.69**. The Outcome tier at $8/verified-PR gives 79% GM and is the profit center; Pro/Team are breakeven-on-bundle by design and rely on overage for margin.

Two engineering KPIs determine company viability: **cache hit rate ≥ 70%** and **median task token budget ≤ 400K total tokens**.

## Real-world token consumption per agent task (May 2026 reference)

Major vendors don't publish official numbers, but leaks + forum reports cluster tightly:

| Product | Per-task profile |
|---|---|
| Cursor | ~22.5% of $20 Pro credit pool per agent run on 50K-LoC repo. Effective rate: $4.50 / ~500K–1M tokens per task. ~90% is cache write/read. |
| Claude Code | $13/active-day average (enterprise anchor), 90th percentile <$30/day. ~300K–700K billable tokens per active day = ~50K–100K per discrete task. |
| Aider | 4.2× fewer tokens than Claude Code (Morph Feb 2026 benchmark), accuracy 78→71%. Typical hour: 200K–400K tokens, $1–3 cost. |
| Devin | 1 ACU ≈ 15 min active work, $2.00–$2.25/ACU. Task average 1.5–3 ACUs = $3–$7 per task. |
| Replit Agent 3 | Simple edits ~$0.10; complex feature builds $5+. Heavy users report $1K/week. |
| Codex / GPT-5.5-Codex | API $5/$30 per M tokens, $0.50 cached. ChatGPT plan conversion: ~$1.20 entry-task. |

**Median agent task in market:** 200K–500K total tokens, $0.50–$5 marginal cost. Heavy tasks 1–2M tokens.

## Pricing landscape comparison

| Product | Entry | Mid | Top | Pricing primitive |
|---|---|---|---|---|
| Cursor | $20 Pro ($20 credit) | $60 Pro+ | $200 Ultra | Credit pool → API passthrough |
| Claude Code | $20 Pro | $100 Max5 | $200 Max20 | Weekly token ceiling |
| Codex / ChatGPT | $20 Plus | $30 Business | $200 Pro | Credit allowance (1 credit = $0.01) |
| GitHub Copilot (June 2026) | $10 Pro | $19 Business | $39 Enterprise/Pro+ | Plan-included AI credits, then usage |
| Devin | $20 entry | $500 Team | Enterprise | ACU ($2.00–$2.25) |
| Replit | $20 Core | $35 Pro+ | Enterprise | Effort-based |
| Windsurf | $15 Pro | $30 Teams | $60 Enterprise | Credits |
| v0 | $5 free | $20 Premium | $30/seat Team | Credits |
| Tabnine | $9 Dev | $39 Ent/seat | Enterprise | Seat |
| JetBrains AI | $10 | $20 | $30 | Seat, tiered quota |

**Market patterns:** seat-only (Tabnine, JetBrains) is collapsing; the dominant model is seat + credit pool + on-demand burst (Cursor, GitHub June 2026, Codex). Outcome pricing exists in adjacent markets (Sierra, Intercom Fin, Zendesk AI) but no coding-agent vendor has shipped it.

## Crucible per-task cost model

### Routing assumption (May 2026)

| Phase | Model | Justification |
|---|---|---|
| Planning | Sonnet 4.6 or Gemini 3.1 Pro | Tier 2 decisions; quality matters |
| Execution loop | Opus 4.7 | Agentic tool use leader |
| Verification | Gemini 3.1 Pro | Cross-family from Opus |
| Memory recall | Haiku 4.5 | Cheap, cached |

### Median task token math

12 tool calls, 6 reads, 3 writes, 2 test runs, 1 plan, 1 verify:

| Phase | Model | Raw input | Cached input | Output | Cost |
|---|---|---|---|---|---|
| Plan | Sonnet 4.6 | 50K (45K cached, 5K fresh) | 45K @ $0.30 = $0.0135 | 3K @ $15 = $0.045 + 5K @ $3 = $0.015 | **$0.074** |
| Exec × 6 steps | Opus 4.7 | 30K each (24K cached, 6K fresh) | 6×24K @ $0.50 = $0.072 | 6×8K @ $25 = $1.20 + 6×6K @ $5 = $0.18 | **$1.452** |
| Verify | Gemini 3.1 Pro | 40K (no cross-vendor cache) | n/a | 5K @ $12 = $0.06 + 40K @ $2 = $0.08 | **$0.14** |
| Memory recall | Haiku 4.5 | 4×20K (90% cached) | 72K @ $0.10 = $0.0072 | 4×500 @ $5 = $0.01 + 8K @ $1 = $0.008 | **$0.025** |
| **Total** | | | | | **~$1.69** |

### Three scenarios

| Scenario | Repo | Context/step | Cache hit | Marginal $/task |
|---|---|---|---|---|
| Small | 5K LoC | 15K avg | 85% | $0.55 |
| Median | 50K LoC | 30K avg | 75% | $1.69 |
| Large | 500K LoC | 80K avg, 10 steps | 60% | $6.80 |

### The "2× tokens" insight

Verification is end-of-task and uses ~8% of total cost — not 2×. The architecture's apparent cost penalty is closer to **1.08×**. This is a key narrative anchor: cross-family verification is essentially free compared to single-model execution.

## Pricing tier table (decision)

| Tier | Price | Included | Overage | Target |
|---|---|---|---|---|
| Pro | $40/mo | 25 verified PRs (median) | $2.50/PR | Individual dev, weekend builder |
| Team | $120/dev/mo | 80 verified PRs/dev pooled | $2.00/PR (volume) | 5–50 dev teams |
| Outcome | $8/PR + $500/mo min | No subscription, true PAYG | n/a | Legacy modernization, agencies, indie founders |
| BYOK | $25/dev/mo flat | Unlimited, customer brings keys | $0 token markup | Privacy-conscious, large enterprise |
| Enterprise (self-host) | $50K/yr + $400/node/mo | Unlimited, on-prem inference | Custom SLA | Regulated (FedRAMP, defense, healthcare) |

### Rationale

- **Pro $40 / 25 PRs = $1.60/PR effective.** Deliberately breakeven on the bundle; profitable on overage. Mirrors Cursor's $20-includes-$20-credit psychology but with a verified-PR unit.
- **Team $120/dev / 80 PRs = $1.50/PR effective.** Pooling lets heavy users average out with light users (typical team: 3–4 heavy committers per 10 devs).
- **Outcome $8/PR.** Mental anchor: 1 hour of contractor = $80–120; this is 10% of that. Legacy-modernization buyers compare to hourly consulting.
- **BYOK $25/dev flat.** Captures the "orchestrator without markup" segment (Cline/Aider archetype).
- **Enterprise $50K base.** Competes with self-hosted Sourcegraph Cody ($120K–300K typical).

## Margin analysis

GM per tier (median task assumption, $1.69 cost, 75% cache):

| Tier | Revenue/PR | Cost/PR | GM | GM @ 30% cache |
|---|---|---|---|---|
| Pro (included) | $1.60 | $1.69 | -5.6% | -45% |
| Pro (overage) | $2.50 | $1.69 | 32% | 6% |
| Team (pooled) | $1.50 | $1.69 | -13% | -52% |
| Team (overage) | $2.00 | $1.69 | 16% | -16% |
| **Outcome** | **$8.00** | **$1.69** | **79%** | **71%** |
| BYOK | $25/dev flat | ~$0 | ~100% | 100% |

### Critical insights

1. **Included-bundle pricing is negative-GM in worst caching cases.** This is the Cursor 2025 trap. Defense: cache hit rate must stay > 70%.
2. **Outcome tier is the profit center.** Sales motion should weight toward it.
3. **BYOK is high-margin** because we pay no model COGS. Don't undersell it.

### Break-even per seat

- Pro: $40 / $1.69 = **23.7 PRs cost floor** vs 25 included = **1.3 PRs slack**. Thin.
- Team: $120 / $1.69 = **71 PRs cost floor** vs 80 included = **9 PRs slack**. Healthier.

## Sensitivities

| Scenario | Impact |
|---|---|
| **Token prices -30% (likely by Q4 2026)** | GM +~20pp across bundled tiers. Pro -5.6% → +14% |
| **10× median volume per seat** | Pro at 250 PRs/mo costs $422 vs $40 revenue = catastrophic. **Hard usage caps mandatory.** |
| **Cache hit at 30%** | Median task → $3.10. All bundled tiers deeply negative. **Caching is THE engineering priority.** |
| **Verifier cost inflates to 25%** | Median → $1.94. Bundled tiers slightly more negative; Outcome still ~75% GM. |

## What's still uncertain

1. **Cross-family cache transfer.** Assumed verifier (Gemini) pays full fresh input cost. If verifier stays in Anthropic family (Sonnet verifying Opus), cost drops ~60% — but loses cross-family error decorrelation. Tradeoff TBD with eval data.
2. **Opus 4.7 tokenizer inflation.** New tokenizer consumes ~35% more tokens for same text. Actual median may be $2.10, not $1.69.
3. **Cache TTL at team scale.** 1h cache assumes user-session locality. Across a 10-dev team, locality drops. Team-pooled tier might effectively run at 50–60% cache hit and need revenue bump to $130/dev.
4. **PR-complexity distribution.** Probably Pareto: 20% of PRs consume 60% of cost. Need closed-beta data; complexity-banded pricing is v2.
5. **Verifier-rubric strictness.** Too strict → low pass rate → not enough verified PRs to count → user perceives non-value. Too lenient → bad PRs counted → reputation damage.
6. **Anthropic/Google price war probability.** Opus 4.8 at $4/$20 in Q3 2026 (likely) → GM +15pp. Status quo with tighter rate limits → opposite.

## Comparison to incumbents

| Incumbent | Their GM (rough) | Crucible position |
|---|---|---|
| Cursor | "slightly GM-positive" April 2026; achieved via Composer-2 in-house model | ~8% cost premium but ~25–50% price premium plausible |
| Anthropic / Claude Code | 60–70% (pays COGS, not retail) | Structural disadvantage; need BYOK + self-host |
| Devin | 65–75% (~$0.40–0.80 actual cost / $2.00–2.25 ACU price) | Aligned philosophy, cheaper per outcome, more transparent |

## GTM consequence

**Lead with Outcome tier and Team plans.** Pro is top-of-funnel, not profit. Build everything on the assumption that:

- Cache hit rate stays >70%.
- Median task ≤ 400K total tokens.

These are the KPIs that determine viability. They are observable from day 1 of beta. We do not commit to pricing until we have 30 days of real-customer telemetry validating both.

## Pricing-change roadmap

- **v1 (launch):** the five tiers above.
- **v2 (post-PMF):** complexity-banded Outcome ($4 small / $8 median / $20 large).
- **v3:** free OSS-maintainer tier (brand investment).
- **v4:** outcome SLAs ("N verified PRs/mo guaranteed at $X") if customer demand surfaces.

## References

- [00-vision/pricing-and-business.md](../00-vision/pricing-and-business.md)
- [ADR-004: Outcome-based pricing](../05-decisions/ADR-004-outcome-based-pricing.md)
- [01-architecture/model-routing.md](../01-architecture/model-routing.md)

---


# 07. Roadmap

<a id="file-07-roadmap--v1-mvp"></a>

<!-- ================================================================== -->
<!-- File: 07-roadmap/v1-mvp.md -->
<!-- ================================================================== -->

# v1 MVP

What ships in version 1 — and what explicitly does not. Calibrated to AI-agent throughput, not human-team cadence: see [build-plan-agent-days.md](build-plan-agent-days.md) for the day-by-day breakdown.

## v1 = "the thesis, working end-to-end"

A senior engineer should be able to:

1. Install Crucible (5 min).
2. Let the Cartographer run on a real codebase (≤ 30 min, automated).
3. Submit a real task — feature add, refactor, or bug fix.
4. Watch Crucible spin up a twin, execute, run cross-family verification, propose a PromotionBundle.
5. Approve the promotion.
6. See the change land via canary rollout with auto-rollback safety.
7. Verify every step cryptographically against Sigstore Rekor.

If all seven steps work end-to-end for the first three design-partner customers, v1 ships.

## In scope

### Twin Runtime (the trust foundation)

- Firecracker microVM via E2B (SaaS); raw Firecracker (self-host).
- Git worktree + overlayfs filesystem twin.
- Neon Postgres CoW branching (Postgres customers only).
- Hoverfly OSS service-tape replay with PII scrub (Presidio + spaCy + FF3-1).
- Infisical-issued ephemeral secrets.
- Cilium/Tetragon egress allowlist.
- Syscall shim with destructive-op typed proposals.

### Verifier Pipeline

- Tier 0 mutation testing (mutmut, stryker, cargo-mutants, go-mutesting, pitest, muter).
- Tier 1 property tests + fuzz (hypothesis, fast-check, proptest, rapid, jqwik).
- Tier 2 schemathesis contract testing + in-house DST harness.
- Tier 3 Dafny (others as deferred-load on first @critical hit).
- Tier 4 Nix hermetic rebuild + SLSA-L3 in-toto attestation via Sigstore Rekor v2.
- Cross-family executor/verifier routing (Opus 4.7 ↔ Gemini 3.1 Pro pairing default).

### Memory Layer

- Redis hot cache.
- pgvector episodic + semantic store.
- FalkorDB + Graphiti procedural graph.
- Background distillation worker (PR comments, post-mortems, ADRs).
- Mem0 hierarchical extraction algorithm.
- LLM-as-judge filter on writes (prompt-injection defense).
- OSS-corpus bootstrap (Tier A–D seed) — ~400 active rules on a fresh Next.js+FastAPI repo.
- Convention drift detection.

### Model Routing

- 5-tier router with Anthropic primary + Google verifier as default.
- Per-tenant model overrides.
- 5m + 1h prompt caching.
- Per-task budget enforcement.
- Cost telemetry.

### Promotion Contract

- Rego policy evaluation.
- Slack-button + web-UI human approval.
- KMS-signed credential leases (AWS KMS for SaaS; HSM for enterprise).
- Argo Rollouts canary integration.
- GrowthBook feature-flag rollback.
- In-toto attestation chain.

### Agent SDK

- `twin.*` API in Go, TS, Python, Rust.
- MCP server (`crucible-mcp`) for IDE integration.
- ACP support for Zed.
- gRPC + REST for direct integrations.
- Webhook events spec.

### UI Surfaces

- VS Code extension (plan approval, budget viewer, attestation chain explorer).
- JetBrains plugin (same affordances).
- Zed extension via ACP.
- `crucible` CLI (Go binary).
- Web console (Next.js + shadcn): task dashboard, cost dashboard, memory browser, attestation viewer, approval inbox, SLO dashboard.
- GitHub App for PR-comment invocation.
- Slack bot for approval routing.

### Pricing tiers (all five live at launch)

- Pro / Team / Outcome / BYOK / Enterprise (self-hosted).
- Stripe integration for billing.
- Usage-based metering with hard caps.

### Observability

- OpenTelemetry traces → Honeycomb (SaaS) / Tempo (self-host).
- Prometheus metrics.
- Loki logs (self-host) / Honeycomb events (SaaS).
- The four KPI dashboards (per-task economics, verifier health, safety/trust, memory/learning).
- Public SLO status page.

### Tier 4 self-verification

- Crucible's own monorepo is built via Nix flakes; releases are SLSA-L3 attested; customers can verify our releases.
- Tier 0–4 verification gates every Crucible PR. We eat our own dogfood.

### Documentation

- All docs in `docs/` (this directory) shipped with v1.
- Quickstart on docs site.
- Reference API docs auto-generated from protobuf.

## Explicitly out of scope for v1

- **Tab autocomplete.** That's Cursor's turf; we don't compete on it.
- **Vibe-coding chat-only builder.** Not our ICP.
- **Custom in-house Crucible model (Composer-2-style).** v2 if PMF + cost engineering demand it.
- **GPU sandbox / ML workload twins.** v2 if ICP shifts.
- **Multi-region twin orchestration.** Single-region per task.
- **End-to-end encrypted memory with customer key.** v2 enterprise feature.
- **Mobile app for approvals.** Web console + Slack cover it.
- **Plugin / skill marketplace.** v2.
- **Visual / Figma-aware UI generation.** v3.
- **Voice input.** Not v1.
- **Automatic Tier 3 across all languages.** Dafny is default; Lean / TLA+ / Kani / Z3 require manual annotation in v1.
- **HIPAA-eligible SaaS tier.** Self-hosted only for HIPAA in v1; SaaS BAA in v2.
- **FedRAMP Moderate certification.** Self-hosted air-gap supports the architecture; formal cert in v2.
- **Cassandra / DynamoDB / non-mainstream DB twin support.** Postgres + MySQL + SQLite + MongoDB only.
- **Multi-tenant procedural memory federation graduations.** Single-tenant only; cross-tenant federation requires ≥5 tenants and is unlikely to fire pre-PMF.
- **Customer-built verifier integrations.** Verifier extension API is internal-only; v2 opens it.

## v1 customer experience floor

If any of the following are *missing* at v1, the product fails its thesis:

| Capability | Required | Why |
|---|---|---|
| Twin runtime spawn < 300ms | yes | UX latency floor |
| Destructive-op gate fires on `rm -rf` / `DROP TABLE` / etc. | yes | The core safety story |
| Verifier rejects fake test-pass | yes | The core verification story |
| Plan UI shows $ + time estimate before execution | yes | The cost-transparency story |
| Sigstore Rekor attestation for every action | yes | The audit-trail story |
| Cross-family verifier pairing | yes | The ADR-002 architectural commitment |
| Procedural memory active rules visible in dashboard | yes | The compounding-moat story |
| Air-gap install works end-to-end | yes | The enterprise wedge |

If any of these slip, we don't ship v1. They are non-negotiable.

## v1 launch criteria

- 3 design-partner customers running Crucible on real production codebases for ≥ 30 days each.
- 100+ verified PRs landed across design partners.
- Zero security incidents at the boundaries documented in [threat-model.md](../01-architecture/threat-model.md).
- Cache hit rate ≥ 70% sustained.
- Median task cost ≤ $2.00 sustained.
- SLOs in [observability.md](../02-engineering/observability.md) met for the prior 30 days.
- All 15 ADRs accepted by the build team without unresolved objections.
- Tier 4 self-verification clean on every release in the prior 30 days.

## Open beta → GA

After the design-partner phase:

- **Open beta:** invite-only Pro tier. Cap at 200 users. Monitor cache + cost KPIs against the financial model.
- **GA:** open Pro + Team + Outcome tiers. Enterprise tier remains direct-sales.

## Risk register at v1

| Risk | Mitigation |
|---|---|
| Tape coverage falls below 80% in real customer workloads | Aggressive shadow-recording in onboarding; tier 2 fallback to live-passthrough |
| Verifier disagreement rate > 25% with humans | Shadow-mode tuning; rubric_score threshold adjustment |
| Median task cost exceeds $2.50 sustained | Routing-tier re-classification; cache investment |
| Customer's repo too large for Cartographer in v1 | Chunked processing; explicit "large-repo" mode |
| Frontier model API outage during launch | Multi-vendor routing; status banner |
| Customer-reported false promotion approval | RB-11 runbook; immediate incident response |
| Self-hosted install too complex for design partners | Concierge install with our SRE team |

See [v2-vision.md](v2-vision.md) for what comes after.

---

<a id="file-07-roadmap--v2-vision"></a>

<!-- ================================================================== -->
<!-- File: 07-roadmap/v2-vision.md -->
<!-- ================================================================== -->

# v2 Vision (6-month horizon)

What ships after v1's PMF validation. Calibrated to agent throughput — each block here is 2–5 agent-days, not engineer-months.

## Sequencing principle

v1 is "the thesis end-to-end." v2 is "the thesis deepened where customer signal demands." We don't pre-build for hypothetical demand; we ship v1, learn from design partners, and prioritize v2 features by signal.

That said, several v2 features are predictable enough that pre-design pays off.

## Pillar A: Verifier deepening

### A1. Custom Crucible Verifier Model

Cursor's Composer-2 demonstrated that a small in-house orchestration model cuts cost ~10× vs frontier models. For Crucible specifically, a **verifier-tuned small model** could shrink verification cost while maintaining or improving cross-family error decorrelation.

**Scope:** ~3 agent-days for fine-tuning pipeline; ~$X for training compute; ~1 agent-day for routing integration.

**Trigger to build:** sustained ≥ 20% of total cost going to verifier across the customer base; or vendor token price increase >20%.

### A2. Multi-verifier ensemble

For high-stakes promotions, two cross-family verifiers ensemble. Disagreement between verifiers triggers human review.

**Scope:** ~2 agent-days. Mostly orchestration; small UX additions.

**Trigger:** enterprise customer demand for "no single point of LLM trust."

### A3. Tier 3 expansion

v1 ships Dafny as default Tier 3. v2 expands to:

- **Lean 4 + LeanCopilot** for crypto/numerical code.
- **TLA+ + Apalache** for distributed-invariants.
- **Kani** for Rust `unsafe` blocks and FFI boundaries.
- **Z3 / CVC5** as inline SMT for SMT-friendly proofs.

Each ~1.5 agent-days for the integration.

### A4. Customer-defined verifier extensions

Open the verifier extension API. Customers ship their own verifiers as Crucible plugins.

**Scope:** ~3 agent-days for plugin API + marketplace scaffolding. Skill-marketplace primitives (Claude Code's pattern is the reference).

**Trigger:** customer demand for stack-specific verification (e.g., compliance-rule-checker for a specific regulated domain).

## Pillar B: Memory deepening

### B1. Cross-tenant federation graduations

Once we have ≥ 5 tenants in each major stack, cross-tenant abstract-rule graduations become non-trivial. v1 has the data model; v2 ships the policy engine and surfaces the federated commons to customers.

**Scope:** ~2 agent-days for the graduation pipeline + tenant-visible commons browser.

**Trigger:** ~10 tenants on Team-tier-or-above in the same stack.

### B2. Visual / screenshot memory

Customers paste a screenshot of a UI mockup; Crucible's design subagent extracts design tokens (colors, typography, spacing) and applies them in generated UI code.

**Scope:** ~3 agent-days. Vision-model integration; design-token storage; UI-generation prompt hooks.

**Trigger:** "generic AI aesthetic" complaints from frontend-heavy customers.

### B3. Voice-input memory + transcribed stand-ups

Customer records team stand-ups, code reviews, or pair-programming sessions. Transcripts feed the distillation worker as a new source.

**Scope:** ~2 agent-days. Whisper-class transcription + distiller adapter.

**Trigger:** "we have lots of context in our standups that the agent doesn't see" feedback.

### B4. E2E-encrypted memory with customer key

For the highest-assurance enterprise tier. Memory at rest is encrypted with the customer's own KMS key; Crucible operators have read access only via signed access ceremony.

**Scope:** ~4 agent-days. Per-store crypto wrapper; access-ceremony UX; key rotation pipeline.

**Trigger:** FedRAMP / defense customer procurement requirement.

## Pillar C: Twin runtime deepening

### C1. GPU sandbox / ML-workload twins

For customers running ML services, the twin must include GPU access. Route to Modal Sandbox (Firecracker + GPU) or self-hosted with NVIDIA's container runtime.

**Scope:** ~3 agent-days. SandboxProvider interface already supports it; the work is GPU-specific orchestration + cost accounting.

**Trigger:** ML-engineering ICP customers materialize.

### C2. Mobile/iOS/Android twins

Xcode-in-cloud (MacStadium / Mac-Cloud) for iOS twins; Android emulator-in-Firecracker for Android. The simulator-first verification loop ([01-architecture/verifier-pipeline.md](../01-architecture/verifier-pipeline.md)) becomes especially valuable for native mobile because the simulator IS the feedback loop.

**Scope:** ~4 agent-days iOS, ~3 agent-days Android.

**Trigger:** the native-mobile vertical-specialist concept from the competitive research surfaces enough demand.

### C3. Embedded / firmware twins

ESP32 / STM32 / Nordic SDK twins running in QEMU + hardware-catalog-grounded mocks (Embedder.com's pattern). Pair with formal verification for safety-of-life code.

**Scope:** ~5 agent-days.

**Trigger:** embedded vertical customer wants to license.

### C4. Multi-region twin orchestration

For latency-sensitive workloads, twins co-located with customer's primary region.

**Scope:** ~2 agent-days. Region selection + provider routing.

**Trigger:** customer with global presence requests it.

## Pillar D: Pricing & business model evolution

### D1. Complexity-banded Outcome tier

$4 small / $8 median / $20 large per verified PR, based on diff size + Tier 3 escalation + critical-path classification.

**Scope:** ~1 agent-day. Pricing-rule engine + customer-facing pricing tooltips.

**Trigger:** 30 days of closed-beta PR-distribution data showing Pareto-tail customers under-paying.

### D2. SLA tier

"N verified PRs/mo guaranteed at $X" for enterprise customers.

**Scope:** ~2 agent-days. SLO engine for PR delivery + breach-credit billing.

**Trigger:** enterprise procurement asks for the SLA framing.

### D3. Open-source maintainer tier (free)

Verified-maintainer accounts get free Pro-tier usage. Brand investment.

**Scope:** ~1 agent-day. Verification flow (GitHub OSS-maintainer signal) + free-tier gating.

**Trigger:** brand-investment narrative timing (typically post-PMF, before scaled marketing).

### D4. Plugin / skill marketplace

Customers publish Crucible-compatible verifier extensions, MCP tools, custom Rego policies. Marketplace fee model (later).

**Scope:** ~4 agent-days. Marketplace registry + signing + discovery + payments.

**Trigger:** post-Pillar A4 (verifier extension API). Doesn't pencil until plugin ecosystem has scale.

## Pillar E: Specialization toward the vertical wedge

The competitive research identified five white-space concepts that emerged in v1. v2 picks the strongest signal:

### E1. Legacy Modernizer specialization

The Cartographer-as-product. Aggressive enhancement of:

- Characterization-test generation for poorly-tested legacy code.
- Layered refactor planner (extract this module → refactor that interface → migrate this DB schema).
- Per-module verified migration with property-based correctness contracts.

**Scope:** ~5 agent-days. Largely a customer-facing UX layer on existing Crucible primitives + specialized prompts for the LLM driver.

**Trigger:** legacy-modernization buyers convert at higher rate than other Outcome-tier customers, AND we have 2–3 reference modernization wins.

### E2. Autonomous Operator (cofounder seat)

Crucible owns the deployed product: deploys, on-call, incident triage, A/B analysis. Solo-founder-shaped buyer.

**Scope:** ~6 agent-days. Twin runtime extension to ops surface; SRE-agent specialization; observability deeper integration; revenue-share billing model.

**Trigger:** if Outcome-tier customers organically pull us into ops work, we productize it.

### E3, E4, E5 (Verifiable / Mobile / Convention-Learning)

Already covered in Pillars A, C, B respectively.

## Pillar F: Operational hardening

### F1. SOC 2 Type II certification

Required for the regulated tier. Year-long observation window; controls already designed.

**Scope:** ~0 agent-days for engineering (controls already in place); ~ongoing for audit support.

**Trigger:** target completion: ~12 months post-launch.

### F2. HIPAA-eligible SaaS tier

BAA-covered LLM vendors only. Per-tenant configuration enforced.

**Scope:** ~2 agent-days for routing-policy enforcement + BAA-vendor whitelist.

**Trigger:** healthtech customers convert.

### F3. FedRAMP Moderate certification

For defense / civilian-fed buyers.

**Scope:** engineering minimal; certification ~6 months of process.

**Trigger:** named defense customer commits to deployment.

### F4. EU-region data-residency tier

Anthropic EU + Vertex EU routing only. Pre-warmed cache in EU regions.

**Scope:** ~1 agent-day.

**Trigger:** EU customer demand.

## Pillar G: Cross-IDE agent identity

The "agent that follows you from VS Code → JetBrains → Terminal with shared memory" concept. With ACP as the standard, this is mostly a memory-layer + auth bind concern.

**Scope:** ~2 agent-days. Already feasible architecturally; just needs the cross-IDE auth state binding to be polished.

**Trigger:** customer feedback about IDE-fragmentation pain.

## How v2 sequences

Roughly:

1. **Month 4–5 (post-launch):** address top-3 customer-pain signals from design-partner + open-beta data. Likely: cache-hit improvement, Cartographer scaling, Tier 3 expansion.
2. **Month 6–7:** pricing iteration (D1 complexity-banded Outcome, possibly D3 OSS-maintainer tier).
3. **Month 8–9:** specialization (whichever vertical wins on signal — likely E1 or E2).
4. **Month 10–12:** compliance certifications (F1 SOC 2, F2 HIPAA SaaS).

This is rough. Real v2 is signal-driven.

## What we explicitly don't roadmap

- **Our own IDE.** Decided. ADR-011.
- **A new LLM.** We route. Composer-style in-house model is a verifier-cost optimization, not a product line.
- **Chat-with-LLM surface.** The IDE owns that; we own the verified deliverable.
- **Vibe-coding "build an app from prompt" surface.** Wrong ICP.

## How v2 is funded

By Outcome tier revenue + Team tier expansion + Enterprise contracts. The business model from v1 is intact through v2; v2 is depth, not pivot.

## Customer signal we watch

- **Top NPS detractors:** what specifically frustrates them?
- **Outcome tier churn:** are PR-bills predictable enough?
- **Enterprise customers' compliance requests:** which certifications do they actually ask for?
- **Memory growth rate per tenant:** are conventions accumulating, or stalling?
- **Cross-family verifier disagreement rate:** is the architecture working?
- **Self-hosted install time:** is the air-gap path realistic?

Each of these data points triggers v2 prioritization decisions.

## References

- [v1-mvp.md](v1-mvp.md)
- [build-plan-agent-days.md](build-plan-agent-days.md)
- [00-vision/competitive-landscape.md](../00-vision/competitive-landscape.md)

---

<a id="file-07-roadmap--build-plan-agent-days"></a>

<!-- ================================================================== -->
<!-- File: 07-roadmap/build-plan-agent-days.md -->
<!-- ================================================================== -->

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

---


# 08. Phase Prompts

<a id="file-08-phase-prompts--readme"></a>

<!-- ================================================================== -->
<!-- File: 08-phase-prompts/README.md -->
<!-- ================================================================== -->

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

---

<a id="file-08-phase-prompts--phase-01-control-plane"></a>

<!-- ================================================================== -->
<!-- File: 08-phase-prompts/phase-01-control-plane.md -->
<!-- ================================================================== -->

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

---

<a id="file-08-phase-prompts--phase-02-twin-runtime"></a>

<!-- ================================================================== -->
<!-- File: 08-phase-prompts/phase-02-twin-runtime.md -->
<!-- ================================================================== -->

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

---

<a id="file-08-phase-prompts--phase-03-twin-runtime-breadth"></a>

<!-- ================================================================== -->
<!-- File: 08-phase-prompts/phase-03-twin-runtime-breadth.md -->
<!-- ================================================================== -->

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

---

<a id="file-08-phase-prompts--phase-04-verifier-pipeline"></a>

<!-- ================================================================== -->
<!-- File: 08-phase-prompts/phase-04-verifier-pipeline.md -->
<!-- ================================================================== -->

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

---

<a id="file-08-phase-prompts--phase-05-memory-layer"></a>

<!-- ================================================================== -->
<!-- File: 08-phase-prompts/phase-05-memory-layer.md -->
<!-- ================================================================== -->

You are starting Phase 5 of building Crucible. The control plane (P1) routes;
the twin runtime (P2-3) isolates and executes; the verifier (P4) decides
verified completion. Phase 5 builds the MEMORY LAYER — the compounding moat
that learns team conventions over time and feeds them back into both the
agent's planning and the verifier's compliance checks.

This is Block 4 from the build plan (2 agent-days originally, ~35K LoC).
We compress to one ~20K LoC session by focusing on the production-grade core
and deferring some bootstrap-corpus polish.

CALIBRATION
===========
Phase 5 targets ~20K LoC. Memory is the slowest-burning of Crucible's
differentiators — it doesn't change the day-1 experience much, but day-30
customer compliance with team conventions goes from ~91% to ~97%. Build the
infrastructure correctly; the compounding does the rest.

READ FIRST
==========
1. docs/PHASE-4-REPORT.md
2. memory/project_crucible_phase4.md
3. docs/01-architecture/memory-layer.md                  — the full architecture
4. docs/05-decisions/ADR-003-procedural-memory-moat.md  — why memory is the moat
5. docs/05-decisions/ADR-006-falkordb-over-alternatives.md — graph backend choice
6. docs/06-research/memory-bootstrap.md                 — the cold-start strategy
7. docs/03-sdk/agent-sdk-reference.md (twin.memory.*)   — API contracts
8. docs/03-sdk/attestation-formats.md (MemoryWrite/v1)
9. docs/04-operations/onboarding.md (Stage 2: Cartography) — first-week customer flow
10. docs/07-roadmap/build-plan-agent-days.md (Block 4)

RESEARCH BEFORE CODING (parallel)
=================================
1. Mem0 — current hierarchical extraction algorithm; SDK languages; LoCoMo
   benchmark score; Apache-2.0 OSS state.

2. Letta (formerly MemGPT) — current architecture; relevance to our procedural
   memory layer (probably orthogonal but verify).

3. Graphiti — Zep's OSS engine; bi-temporal edge schema; FalkorDB backend support.

4. FalkorDB — current major version; Cypher dialect compatibility; performance
   benchmarks vs Neo4j; KuzuDB-archive lessons learned.

5. pgvector — current version; HNSW vs IVFFlat tradeoffs at our scale;
   row-level security patterns for multi-tenant.

6. Qdrant — current cloud + self-host pricing; payload-filter performance
   for tenant scoping.

7. Turbopuffer — current S3+SSD architecture; pricing at our projected scale
   (relevant past ~10M vectors).

8. OSS-corpus mining for bootstrap — current AGENTS.md ecosystem size (was
   60K+ repos January 2026); GitHub GraphQL API for PR review comment mining;
   rate limits.

9. LLM-judge memory-write filter — current state of prompt-injection defenses
   in Mnemonic Sovereignty research; arXiv 2604.16548 follow-ups.

10. AdaKGC schema-constrained decoding — current implementation availability
    for ensuring extraction outputs validate against our taxonomy schema.

PHASE 5 SCOPE
=============

EXPLICITLY IN SCOPE
-------------------
1. services/memory-router/ — Go service, the hot-path retrieval layer:
   - Multi-signal hybrid retrieval (Redis lookup + pgvector semantic +
     FalkorDB procedural query)
   - 7K-token output budget enforcement (the "context window is RAM not
     storage" Mem0 thesis)
   - A-MAC importance re-ranking (utility × confidence × novelty × recency)
   - Per-tenant + per-repo scoping enforcement at every query
   - gRPC API exposed to control plane and verifier
   - p95 latency target: < 100ms

2. infra/databases/ — schema + RLS policies:
   - Postgres schema for pgvector (episodic + semantic)
   - Row-Level Security policies for tenant_id + repo_id isolation
   - FalkorDB index definitions for Convention nodes + relationships
   - Redis keyspace conventions
   - All migrations versioned and tested via twin-run-first promotion flow
     (we eat our own dogfood)

3. libs/memory-spec/ — protobuf additions:
   - Convention (full data model from docs/01-architecture/memory-layer.md §3.2)
   - Memory query + result types
   - Scope (repo / file_glob / category / "all")
   - SourceRef variants (pr_comment / incident / adr / agent_observation)
   - All from existing twin-spec; consolidate the types here for clarity

4. services/distiller/ — Python background worker:
   - Queue consumer (Kafka or SQS depending on deployment)
   - Source-channel adapters:
     * GitHub PR review comments (GraphQL API, per-tenant token)
     * Incident exports (Rootly/FireHydrant/Jeli/Incident.io)
     * Slack #incidents channels (per-tenant Slack OAuth)
     * Confluence/Notion runbooks + ADR pages
     * Squash-merge commit messages from merged PRs
   - Mem0 hierarchical extraction algorithm (Apache-2.0 OSS reference impl)
   - Schema-constrained decoding (AdaKGC pattern) → typed Convention candidates
   - LLM-as-judge filter on every write (defense against prompt-injection in
     PR comments — the Mnemonic Sovereignty attack surface)
   - Cross-source agreement scoring; Platt-scaled confidence
   - Convention drift detector (30-day rolling positive/negative ratio)
   - Importance scorer + GC (Ebbinghaus decay + A-MAC admission control)
   - Status lifecycle (active | drifting | superseded)

5. services/memory-router/cartographer/ — installer-side mining:
   - One-time-per-repo run at customer onboarding
   - Walk repo, build tree-sitter symbol index
   - Parse lint configs deterministically (the Tier-A "free" rules from
     docs/06-research/memory-bootstrap.md)
   - Parse AGENTS.md, CONTRIBUTING.md, ADR directories
   - Scan recent PR review comments (last 24 months, top 1000 by length)
   - Generate inferred AGENTS.md if one doesn't exist
   - Output: per-tenant, per-repo seed convention bundle

6. infra/oss-corpus-bootstrap/ — the cold-start corpus generation:
   - License-filtered (drop GPL/AGPL/SSPL/BUSL inputs)
   - Tier A: ~40 curated style guides ingested verbatim
   - Tier B: top 200 repos per stack (12 stacks); lint configs + AGENTS.md
   - Tier C: PR review comment corpus from same Tier-B repos
   - Tier D: ADR + post-mortem corpus
   - Extraction pipeline: deterministic for configs, LLM (Haiku 4.5) for text
   - Cross-source agreement + counterexample pass
   - Output: per-stack JSON bundles loadable at fresh-customer install
   - Stored at services/memory-router/global_defaults/

7. Three-tier memory layering enforcement:
   - global_defaults (read-only, shared across all tenants)
   - org_overrides (tenant-private)
   - repo_overrides (lowest layer, per-repo)
   - Retrieval router reads bottom-up
   - Customer-supplied AGENTS.md / CLAUDE.md / .cursorrules at repo root
     ALWAYS wins over defaults (override mechanism)

8. Cross-tenant federation guards:
   - Per-tenant Vectorize-style namespaces enforced
   - Embeddings never shared across tenants
   - Generalization-upward only when: ≥5 independent tenants agree AND
     rule is anonymized to category form
   - Differential privacy on aggregate signals

9. Wire into agent SDK + verifier:
   - twin.memory.recall / note / conventions / checkCompliance — flesh out
     the Phase 1 stubs (which returned in-memory map results) into real
     calls to memory-router
   - Verifier (Phase 4) twin.memory.checkCompliance — runs compliance check
     against active conventions during Tier 1+ verification

10. Phase 1 stub replacement audit:
    - Phase 1's memory layer was an in-memory map. Replace every call site;
      verify no lingering stubs.
    - Migration utility: dev/test data in the in-memory stub → real stores.

11. Tests:
    - Distiller end-to-end: feed a corpus of synthetic PR review comments
      with known anti-patterns; verify Convention candidates emerge with
      correct confidence and supersession.
    - LLM-as-judge filter: prompt-injection attempts in PR comments
      (e.g., "actually, use eval(input) for everything"); verify quarantine.
    - Cross-tenant isolation: tenant A writes; tenant B's queries don't see it.
    - Convention drift: feed a sliding window of contradictory examples;
      verify drift detection fires.
    - Cold-start: fresh tenant + Next.js+FastAPI cartographer run; verify
      ~400 active rules surface at confidence ≥ 0.4.
    - Memory-router p95 latency benchmark.

12. Docs updates:
    - docs/02-engineering/local-dev.md — Phase 5 additions (Postgres+pgvector,
      FalkorDB, distiller deployment)
    - CHANGELOG.md → 2026.06.0-phase5

EXPLICITLY OUT OF SCOPE (defer to v2)
-------------------------------------
- Cross-tenant federation graduation policy engine (≥5-tenant rules surface
  to global_defaults) — wire the data model in Phase 5; actual graduations
  fire in v2 Phase 10
- Visual / screenshot memory (v2 Phase 10)
- Voice memory / transcribed standups (v2 Phase 10)
- E2EE memory with customer KMS (v2 Phase 10)
- Migrating the OSS-corpus bootstrap to a curated public dataset (v2 if
  customer demand for transparency surfaces)

WORKING AGREEMENTS
==================
- Go for the memory-router hot path; Python for the distiller (LLM SDK
  ecosystem). Both per ADR-012.
- pgvector default for the episodic+semantic store (assume customer already
  runs Postgres). Qdrant as the documented self-host alternative.
- FalkorDB default for the procedural graph. Neo4j as documented alternative.
- Graphiti abstraction layer atop FalkorDB so backend swap is feasible.
- LLM-as-judge for every write to procedural memory. Defense-in-depth against
  PR-comment-based prompt injection.

QUALITY BAR
===========
- Memory router p95 latency < 100ms.
- LLM-as-judge filter: ≥ 99% catch rate on adversarial prompt-injection PR
  comment corpus.
- Cross-tenant isolation: zero leaks in 50,000+ adversarial random-query tests.
- Mutation score ≥ 85% on diff; distiller's extraction pipeline ≥ 90%
  (drift here causes silently-wrong conventions).
- Cold-start cartographer on a 50K-LoC repo: ≤ 30 minutes wall-clock; the
  "Stage 2: Cartography" UX promise from docs/04-operations/onboarding.md.
- Hermetic Nix builds across the new components.

PROGRESS TRACKING
=================
  1. Read docs + PHASE-4-REPORT
  2. Currency-check research (parallel — 10 streams)
  3. libs/memory-spec consolidation
  4. infra/databases — schema + RLS + indexes
  5. services/memory-router hot-path retrieval
  6. services/distiller queue + source adapters
  7. Cartographer + per-stack bootstrap bundles (largest single piece)
  8. Mem0 extraction + LLM-as-judge filter
  9. Convention drift detector + importance scorer
  10. Wire into agent SDK + verifier
  11. Phase 1 stub audit + replacement
  12. Tests (including the cross-tenant isolation + prompt-injection ones)
  13. Docs + report

END-OF-SESSION REPORT
=====================
docs/PHASE-5-REPORT.md:

1. File tree + LoC
2. Cold-start cartographer demo result (commands + output on a real OSS repo)
3. Cross-tenant isolation test results
4. LLM-as-judge prompt-injection catch rate
5. Memory router p95 latency benchmark
6. Per-stack default rule counts (post-bootstrap)
7. Stubs + deferred items
8. The Phase 6 prompt (promotion contract + provenance — template at
   docs/08-phase-prompts/phase-06-promotion-and-provenance.md)

Update memory: project_crucible_phase5.md.

GUARDRAILS
==========
- Do NOT skip the LLM-as-judge filter. PR comments are attacker-controllable
  input; this is the primary defense against memory poisoning.
- Do NOT cross-write tenant data. Every write goes through the scoping enforcer.
- Do NOT share embeddings across tenants in the vector store.
- Do NOT ship customer-derived rules into global_defaults without the
  ≥5-tenant + categorical-form anonymization graduation policy (which is
  deferred to v2 — Phase 5 wires the data model only).
- Do NOT include GPL/AGPL/SSPL/BUSL inputs in the OSS-corpus bootstrap.
  License-filter at ingestion.
- Do NOT cache embeddings of customer-private content across tenants.
- Do NOT bootstrap a fresh customer with low-confidence rules. Threshold ≥ 0.4
  is the surface bar; lower goes into the CANDIDATE bucket invisibly until
  customer PR activity confirms.

Memory is the moat. Most of its value compounds invisibly over months. Get the
infrastructure right; the compounding does the rest.

Begin.

---

<a id="file-08-phase-prompts--phase-06-promotion-and-provenance"></a>

<!-- ================================================================== -->
<!-- File: 08-phase-prompts/phase-06-promotion-and-provenance.md -->
<!-- ================================================================== -->

You are starting Phase 6 of building Crucible. The agent now produces verified
work (P1-4) against learned conventions (P5). Phase 6 builds the bridge from
twin to real: the PROMOTION CONTRACT, plus the PROVENANCE PIPELINE that has
been stubbed since Phase 1.

These were Blocks 5 (Promotion Contract, 1 agent-day) and 6 (Provenance
plumbing, 1 agent-day) in the build plan. Together they fit a single ~18K LoC
session because most of the heavy infrastructure (Sigstore, Cosign, Rekor,
KMS clients, OPA/Rego) is already production-grade.

CALIBRATION
===========
Phase 6 targets ~18K LoC. The work is largely orchestration glue against
mature crypto + delivery infrastructure. Quality bar is high because this is
the cryptographic audit-trail layer, but the surface is well-paved.

READ FIRST
==========
1. docs/PHASE-5-REPORT.md
2. memory/project_crucible_phase5.md
3. docs/01-architecture/promotion-contract.md           — the full contract spec
4. docs/05-decisions/ADR-010-sigstore-rekor-attestations.md
5. docs/05-decisions/ADR-014-infisical-over-vault.md    — secrets layer reminders
6. docs/03-sdk/attestation-formats.md                   — all 13 predicate types
7. docs/01-architecture/threat-model.md (T2 T7 T8 T20 T21) — promotion-relevant threats
8. docs/04-operations/runbooks.md RB-05 (Rekor unreachable), RB-06 (KMS failure),
   RB-11 (false promotion approval)
9. docs/03-sdk/event-spec.md (task.promotion_* events)
10. docs/07-roadmap/build-plan-agent-days.md (Blocks 5 + 6)

RESEARCH BEFORE CODING (parallel)
=================================
1. Sigstore Rekor v2 — production-ready status; Go/Rust client libraries;
   inclusion-proof verification latency; self-hosted Rekor v2 setup.

2. Sigstore Cosign — keyless OIDC flow current best practice; Fulcio CA root
   rotation procedure; DSSE envelope tooling.

3. in-toto attestation framework — current spec version; subject + predicate
   conventions; Crucible predicate type registration.

4. SLSA Provenance v1 — actions/attest-build-provenance current; Witness for
   non-GitHub CI; SLSA-L3 hardened-runner requirements.

5. OPA / Open Policy Agent — go-rego module; embedded vs sidecar deployment
   tradeoffs; policy bundle distribution.

6. AWS KMS — current Go SDK for asymmetric signing; HSM-backed key reqs;
   credential-lease IAM-policy patterns; cost.

7. GCP Cloud HSM — same questions; alternative for GCP-native customers.

8. YubiHSM — on-prem HSM Rust/Go SDK + PKCS#11 integration; key-attestation
   chain for FedRAMP track.

9. Argo Rollouts — current AnalysisTemplate spec; Prometheus query patterns
   for SLO checks; rollback semantics.

10. GrowthBook — OSS self-host current version; SDK for flag flipping;
    integration with Argo Rollouts.

PHASE 6 SCOPE
=============

EXPLICITLY IN SCOPE
-------------------
1. apps/promotion-gate/ — Go service:
   - bundle_validator/  validates the PromotionBundle attestation chain end-to-end
   - rego_engine/       embeds OPA; loads tenant policy bundles; emits
                        Allow/Deny/RequireApproval decisions
   - approval_router/   determines who must sign (CODEOWNERS, designated
                        approvers, N-of-M rules) based on bundle + tenant config
   - kms_lease/         single-use, time-boxed, action-scoped credential lease
                        signed by AWS KMS / GCP HSM / YubiHSM
   - delivery_adapter/  hands off to Argo Rollouts (K8s) or feature-flag-only
                        progressive delivery for serverless/VM stacks
   - outcome_watcher/   monitors canary; emits PromotionOutcome attestation;
                        auto-rollback on SLO regression
   - api/               gRPC + webhook surface

2. libs/policy/ — flesh out the Phase 1 stub:
   - Default Rego bundle (the policy from docs/01-architecture/promotion-contract.md)
   - Tenant override loading (per-tenant policies merged with defaults)
   - Policy-bundle signing (each tenant's policy is itself a signed artifact)
   - Decision attestation: every Rego evaluation produces a signed
     PromotionApproval/v1 record

3. apps/attestation-relay/ — Rust service (per ADR-012 perf-sensitivity):
   - DSSE envelope construction
   - Fulcio OIDC cert issuance via Sigstore keyless flow
   - Rekor v2 publication
   - Local hash-chained journal as fallback (per Phase 1) — but now the journal
     back-fills to real Rekor on recovery
   - In-toto attestation generators for ALL 13 predicate types from
     docs/03-sdk/attestation-formats.md (Phase 1 had most; verify completeness)
   - Inclusion-proof verification on read
   - Self-hosted-Rekor support for enterprise tier

4. Replace Phase 1 stubs:
   - Sigstore Rekor v2 publish: real, not local-journal-only
   - KMS signing: real AWS KMS / GCP HSM / YubiHSM (per deployment)
   - Promotion gate: real, not "log and return success"
   - Per-tenant Rego bundle loading

5. infra/argo-rollouts/ — Helm chart templates:
   - AnalysisTemplate library (SLO-check templates for common metrics:
     error_rate_p99, latency_p95, custom metrics per service)
   - Rollout strategy templates (1% / 5% / 25% / 100% canary with dwell)
   - Auto-rollback configuration
   - Per-task canary spec generation from PromotionBundle.suggested_rollout

6. infra/feature-flag-rollouts/ — alternative path for non-K8s customers:
   - GrowthBook flag creation at promotion-time, scoped to the change
   - Incremental rollout percentages
   - Periodic SLO check via Prometheus query
   - Flag flip to 0% on regression (millisecond rollback)

7. apps/slack-bot/ — approval routing surface (minimal):
   - Slack OAuth + SAML/SSO required for approver identity
   - Approval button on promotion-pending events
   - Approver signs via Sigstore keyless OIDC (their personal cert)
   - Approval attestation published

8. Wire into Phase 1's control plane:
   - twin.promote(bundle) — flesh out from Phase 1 stub
   - Phase 4's VerifierApproval is the gating input
   - Phase 5's procedural memory updates after promotion lands (success
     reinforces conventions; rollback weakens them)

9. Special handling: database migrations
   - Three-step flow per docs/01-architecture/promotion-contract.md §"Database migrations"
   - Twin run → Shadow run on production replica → Promotion via temporary
     KMS-signed ALTER TABLE lease
   - Post-migration query checks (data integrity, row counts)
   - Rollback path: transactional or manual down-migration in bundle

10. Tests:
    - End-to-end promotion: verified PromotionBundle → policy eval → approval
      (auto or human) → KMS lease → canary rollout → final attestation → land
    - Auto-rollback: deliberate SLO regression mid-canary; verify flag flip
      + rollback attestation
    - Threat-model tests:
      * T2 (forged bundle): replay attack rejected
      * T7 (tampered artifact): hash mismatch rejected
      * T8 (action repudiation): full chain traceable in Rekor
      * T20 (egress in promotion path): blocked by isolation
      * T21 (compromised approver): N-of-M policy enforced
    - Database migration: forward + auto-rollback on integrity check fail
    - Self-hosted Rekor: full local-only chain works without public Sigstore

11. Docs updates:
    - docs/02-engineering/local-dev.md — Phase 6 additions (KMS dev mode with
      local key, Slack bot ngrok setup)
    - CHANGELOG.md → 2026.06.0-phase6

EXPLICITLY OUT OF SCOPE (defer to v2 or later phases)
-----------------------------------------------------
- Multi-region KMS replication
- Customer-controlled signing keys for the highest-FedRAMP-tier (v2 hardening)
- Post-quantum crypto migration (industry timeline)
- Plugin marketplace for custom Rego policies (v2 Phase 12)

WORKING AGREEMENTS
==================
- Go for the promotion gate; Rust for the attestation relay (perf + Sigstore
  Rust client maturity).
- OPA embedded (go-rego), not sidecar — Phase 6 ships in-process Rego.
- Default Rego bundle ships with sensible defaults; every tenant can override
  via a signed policy bundle.
- Real Sigstore Rekor v2 in dev (no more local-only journal except as fallback).
- Real KMS signing in dev via AWS KMS dev account or local SoftHSM.

QUALITY BAR
===========
- Mutation score ≥ 85% on diff; ≥ 95% on rego_engine/ and kms_lease/ — these
  are the trust-critical pieces.
- Promotion gate end-to-end latency: ≤ 5s for auto-approve trivial; ≤ 30s
  including human approval wait.
- Attestation chain validation: zero false-acceptances against 10,000+ forged
  bundles in the threat-model test corpus.
- Self-hosted Rekor fallback: full chain workable offline.
- Argo Rollouts integration: auto-rollback fires within 1 SLO-check cycle
  of regression detection.
- Hermetic Nix builds.

PROGRESS TRACKING
=================
  1. Read docs + PHASE-5-REPORT
  2. Currency-check research (10 streams parallel)
  3. libs/policy Rego bundle implementation
  4. apps/attestation-relay (Rust) — DSSE + Fulcio + Rekor v2
  5. Phase 1 stub replacement audit (Rekor + KMS + promotion)
  6. apps/promotion-gate bundle validator + rego engine
  7. apps/promotion-gate approval router + KMS lease
  8. apps/promotion-gate delivery adapter + outcome watcher
  9. infra/argo-rollouts + infra/feature-flag-rollouts templates
  10. apps/slack-bot approval surface
  11. Wire into control plane
  12. Database migration special-case flow
  13. Tests (threat-model + end-to-end + chaos)
  14. Docs + report

END-OF-SESSION REPORT
=====================
docs/PHASE-6-REPORT.md:

1. File tree + LoC
2. End-to-end promotion demo (commands + signed Rekor UUIDs in output)
3. Threat-model test results (T2/T7/T8/T20/T21)
4. Auto-rollback demonstration
5. Self-hosted Rekor verification flow
6. KMS dev-mode setup instructions
7. Stubs + deferred items
8. The Phase 7 prompt (agent-facing UX — template at docs/08-phase-prompts/
   phase-07-agent-facing-ux.md)

Update memory: project_crucible_phase6.md.

GUARDRAILS
==========
- Do NOT skip attestation chain validation. A forged bundle that promotes is
  brand-existential.
- Do NOT use long-lived KMS keys. The whole point is short-lived OIDC-bound
  certs via Sigstore keyless flow.
- Do NOT cache KMS credentials in the agent process. Lease → use → expire.
  No reuse.
- Do NOT allow self-approval. Agent's OIDC subject and human approver's OIDC
  subject must differ; enforce at the gate.
- Do NOT let approvals carry across bundle revisions. Any diff change
  invalidates the prior approval signature.
- Do NOT bypass Rego policy for "test" promotions. Test paths use a different
  policy bundle, but always evaluate.
- If Rekor is unreachable, ALWAYS journal locally + back-fill; never silently
  drop attestations.

The promotion contract is what turns "the agent did something" into "the
agent's action is cryptographically auditable for the next 30 years." Build
it for the auditor who's going to scrutinize this chain in 2056.

Begin.

---

<a id="file-08-phase-prompts--phase-07-agent-facing-ux"></a>

<!-- ================================================================== -->
<!-- File: 08-phase-prompts/phase-07-agent-facing-ux.md -->
<!-- ================================================================== -->

You are starting Phase 7 of building Crucible. The backend is functionally
complete: agent submits tasks (P1), executes in isolated twins (P2-3),
ships verified work (P4), learns conventions (P5), promotes via signed
gates (P6). Phase 7 builds the SURFACES users actually touch.

This is Block 7 from the build plan (3 agent-days, ~50K LoC originally).
We compress to a ~25K LoC session by leveraging shadcn/Tremor for the web
console and the MCP/ACP standards for IDE integration (no custom forks).

CALIBRATION
===========
Phase 7 targets ~25K LoC. Most of this is web UI + thin IDE plugins. The work
is well-paved (Next.js + shadcn for the console; MCP/ACP for IDEs). Quality
bar emphasizes UX polish on the plan-approval / cost-preview surfaces because
those are the customer-visible trust narrative.

READ FIRST
==========
1. docs/PHASE-6-REPORT.md
2. memory/project_crucible_phase6.md
3. docs/05-decisions/ADR-011-no-built-in-ide.md         — we plug into existing IDEs
4. docs/03-sdk/tool-reference.md                        — MCP tool surface
5. docs/04-operations/onboarding.md                     — what the UI must support
6. docs/03-sdk/event-spec.md                            — webhook events the UI listens to
7. docs/02-engineering/observability.md                 — the four KPI dashboards
8. docs/02-engineering/tech-stack.md (frontend section)
9. docs/00-vision/target-users.md                       — what each persona does in the UI
10. docs/07-roadmap/build-plan-agent-days.md (Block 7)

RESEARCH BEFORE CODING (parallel)
=================================
1. Next.js — current major version + App Router conventions; React Server
   Components default patterns; `use client` boundary best practices in 2026.

2. shadcn/ui — current component library + Radix primitives; theming system;
   the install pattern (copy-paste, no npm dep).

3. Tremor — current dashboarding components; integration with Recharts.

4. zod + react-hook-form — current form-validation idioms.

5. Clerk (SaaS auth) — current SDK + tenant scoping.

6. WorkOS (enterprise SSO) — SAML + OIDC flow + tenant assignment.

7. MCP — Linux Foundation hosting status; latest protocol version; reference
   implementations across hosts (Claude Desktop, Cursor, Zed, etc.).

8. ACP (Agent Client Protocol) — Zed's spec; tool surface mapping from MCP.

9. VS Code Extension API — current contribution points for chat panels +
   webview + status bar; required for the plan-approval UX.

10. JetBrains Platform — current Junie / AI Assistant plugin patterns; Project
    Open Telemetry (POT) integration.

11. GitHub App framework — current best practices; webhooks for PR-comment-
    based invocation (`/crucible <description>`).

12. Slack Bolt SDK — current Go/TS SDK; block-kit for the approval button UI.

PHASE 7 SCOPE
=============

EXPLICITLY IN SCOPE
-------------------
1. apps/web-console/ — Next.js + App Router + shadcn:
   Pages:
   - / (landing/tenant-overview)
   - /tasks — task timeline with cost, duration, verifier verdict per task
   - /tasks/[id] — task detail with plan, steps, attestation chain explorer
   - /tasks/[id]/approve — plan-approval modal with cost preview, risk
     callouts, retry-budget config (the customer-trust signature surface)
   - /promotions — pending approval inbox + recent promotion history
   - /promotions/[id] — promotion detail with Rego decision + canary status
   - /memory — convention browser, drift reviewer, supersession history
   - /memory/conventions/[id] — convention detail with positive/negative
     examples + source references
   - /attestations — Rekor UUID search + verification UI
   - /attestations/[uuid] — full attestation content viewer
   - /cost — per-task / per-dev / per-repo cost dashboard (Tremor charts)
   - /slo — public-style status page per docs/02-engineering/observability.md
   - /settings — tenant config (model overrides, retry caps, dollar budgets,
     critical-path classifier weights, promotion policy editor)
   - /webhooks — webhook subscription management
   Components:
   - shadcn primitives + Tremor for charts
   - Server-Sent Events for streaming plan/verifier progress
   - WebSocket only where bi-directional needed
   Auth:
   - Clerk for SaaS tenants
   - WorkOS for enterprise SSO + SAML
   - Authelia/Dex as the self-host option

2. apps/ide-plugins/vscode/ — VS Code extension (~3K LoC):
   - Plan approval modal in webview
   - Budget viewer in status bar
   - Attestation chain explorer panel
   - MCP host client wiring to the Crucible MCP server
   - Commands: Crucible: New Task, Crucible: Approve Plan, etc.

3. apps/ide-plugins/jetbrains/ — JetBrains plugin (~3K LoC):
   - Same affordances as VS Code, idiomatic to JetBrains UI
   - Junie-style toolwindow integration

4. apps/ide-plugins/zed/ — Zed extension via ACP (~2K LoC):
   - Same affordances; ACP routing
   - Lighter-weight (Zed is itself an AI-native editor; we integrate, not duplicate)

5. apps/cli/ — flesh out from Phase 1 minimal:
   - task new / show / approve / monitor / cancel
   - plan show / approve / reject / amend
   - promote / status / rollback
   - memory recall / note / conventions / drift-review
   - attestation get / verify / chain / export
   - webhook create / list / redeliver
   - tenant config get / set
   - verify-release <version> (the public Tier 4 customer-side command)
   - calibrate (the per-tenant critical-path classifier weight fit)
   - All subcommands have --output json for scripting

6. apps/github-app/ — Go + GitHub App framework:
   - PR-comment-driven invocation: `/crucible <description>`
   - Auto-open PR for agent-authored verified bundles
   - Verifier report rendered as PR comment
   - Attestation chain linked in PR description
   - Approval buttons (defers to Phase 6's promotion gate)

7. apps/slack-bot/ — flesh out from Phase 6 minimal:
   - Channel-level promotion approval requests with block-kit
   - Approver button → Sigstore keyless OIDC signing
   - Slash-command task submission `/crucible <description>`
   - DM-based task status notifications
   - Approval routing matches tenant config

8. The plan-approval UX (the differentiating surface):
   - Pre-execution cost preview: "$0.42, ~3 minutes, 4 files, top risk:
     webhook signature verification"
   - Hard-cap visualization: "budget cap: $2.00 (current spend: $0)"
   - Retry-budget visualization: "3 retries per subgoal, halt-and-ask after"
   - Approve / edit / reject buttons
   - One-click "approve and walk away" for trusted task types
   - Live progress as the task executes (SSE)
   - Mid-task interrupt button (halt cleanly at next checkpoint)

9. The attestation viewer (the trust narrative surface):
   - Rekor UUID-based search
   - Full chain visualization for any task: plan → file writes → tool calls →
     verifier verdict → promotion approval → outcome
   - Per-attestation: predicate type, OIDC subject, signed-at, validation
     status, content
   - "Reproduce this attestation" button — local re-verification
   - Public-share link (signed URL) for compliance auditors

10. The memory browser:
    - Per-scope convention list with confidence sliders
    - Drift indicator + last-violated date
    - Source-evidence inspector (clickable PR comments, ADRs, incidents)
    - Override editor with diff vs default
    - Status lifecycle controls (active → drifting → superseded)

11. Branding + design system:
    - Custom Tailwind theme override (NOT default rounded-corners-blue-gradients
      per ADR-001 brand voice — "anti-vibe-coding aesthetic")
    - Monospace-heavy typography for code surfaces
    - Subtle / professional color palette
    - The brand voice is "evidence-driven engineering," not "AI magic"

12. Tests:
    - E2E via Playwright on the web console (key user flows: submit task →
      approve plan → view verification → approve promotion → view attestation)
    - Component tests via Vitest + React Testing Library
    - VS Code extension integration test (vscode-test framework)
    - GitHub App webhook signature verification tests
    - Slack bot block-kit rendering tests

13. Docs updates:
    - docs/02-engineering/local-dev.md — Phase 7 additions (web-console dev
      server, IDE plugin sideload, GitHub App ngrok setup)
    - CHANGELOG.md → 2026.06.0-phase7
    - User-facing docs site bootstrap (Mintlify or similar)

EXPLICITLY OUT OF SCOPE (defer to v2 or Phase 8)
------------------------------------------------
- Mobile app for approvals (web console + Slack cover it; v2 if signal)
- Visual diff editor for agent's proposed code (the IDE's diff is the diff)
- Tab autocomplete (ADR-011: not our competition surface)
- Composer-style multi-file rewrite UI (not our surface)
- Voice input (v2 if signal)
- IDE chat panel (the IDE has chat; we don't duplicate)
- Public marketing website (separate repo per repo-structure.md)
- Premium themes / multi-theme support (v2)

WORKING AGREEMENTS
==================
- TypeScript everywhere on the frontend.
- Next.js App Router default; RSC for data, `use client` only at boundaries
  that need interactivity.
- shadcn/ui + Radix primitives; do NOT write custom-from-scratch components
  when shadcn ships one.
- Tremor for analytics dashboards.
- zod for form validation and API contract enforcement at runtime.
- biome for lint + format.
- pnpm for the monorepo (root workspace; per-app package).
- IDE plugins published to their respective marketplaces from CI.
- Per ADR-011: we do NOT fork an IDE. We plug into existing ones.

QUALITY BAR
===========
- Web console Lighthouse score ≥ 95 on key pages (performance + accessibility).
- Plan-approval modal renders in < 200ms from event receipt.
- Attestation viewer can verify a chain end-to-end without backend round-trips
  beyond initial fetch.
- Mutation score on web-console business logic ≥ 80% (UI tooling is weaker
  than backend; 80% is the practical bar).
- IDE plugins: each one passes its own integration test.
- E2E Playwright suite green across all key flows.
- Hermetic Nix builds — verify the Next.js build is deterministic
  (`prefer-deterministic-bundling` etc. set).

PROGRESS TRACKING
=================
  1. Read docs + PHASE-6-REPORT
  2. Currency-check research (12 streams parallel)
  3. apps/web-console scaffolding (Next.js + shadcn + Clerk/WorkOS)
  4. Plan-approval modal + budget viewer (the differentiating surface)
  5. Task dashboard + task detail
  6. Promotion approval inbox + canary visualization
  7. Memory browser + drift reviewer
  8. Attestation viewer + chain explorer
  9. Cost dashboard + SLO status
  10. Settings page + tenant config editor
  11. apps/ide-plugins/vscode
  12. apps/ide-plugins/jetbrains
  13. apps/ide-plugins/zed (ACP)
  14. apps/cli full surface
  15. apps/github-app (PR-comment invocation)
  16. apps/slack-bot full surface
  17. E2E + integration tests
  18. Docs + report

END-OF-SESSION REPORT
=====================
docs/PHASE-7-REPORT.md:

1. File tree + LoC
2. Demo URLs (dev deployment links if applicable)
3. Lighthouse scores per key page
4. IDE plugin install instructions
5. GitHub App / Slack bot install instructions
6. Stubs + deferred items
7. The Phase 8 prompt (onboarding + v1 launch — template at
   docs/08-phase-prompts/phase-08-onboarding-and-v1-launch.md)

Update memory: project_crucible_phase7.md.

GUARDRAILS
==========
- Do NOT default to Tailwind blue + rounded corners. Brand voice is anti-vibe;
  the UI reflects that.
- Do NOT add tracking/analytics that send customer code or task content to
  third parties. Plausible/PostHog-style page-event analytics OK; agent-task
  content stays in-tenant.
- Do NOT embed customer prod credentials into the UI. The UI surfaces task
  events and attestation references; never secret values.
- Do NOT replicate the IDE's chat panel or completion UX. We integrate, we
  don't compete.
- Do NOT skip accessibility. shadcn ships Radix which is a11y-correct;
  preserve that.
- Do NOT ship the IDE plugins or GitHub App with permissions broader than
  needed. Scope to repo:read + PR:write + workflow:read.
- Do NOT enable webhook subscriptions without signature verification.

The UX is where the customer perceives the trust narrative. Build for the
senior engineer who is going to scrutinize "what did this agent actually do
to my codebase, and how do I verify it?" Every surface should answer that.

Begin.

---

<a id="file-08-phase-prompts--phase-08-onboarding-and-v1-launch"></a>

<!-- ================================================================== -->
<!-- File: 08-phase-prompts/phase-08-onboarding-and-v1-launch.md -->
<!-- ================================================================== -->

You are starting Phase 8 of building Crucible — the final v1 phase. The
product is functionally complete after Phase 7. Phase 8 makes it ONBOARDABLE,
INSTALLABLE, and READY FOR LAUNCH.

This is Block 8 from the build plan (2 agent-days originally, ~30K LoC) plus
v1 launch-criteria validation. We target ~20K LoC: most is well-paved
integration work; the unique pieces are the Cartographer (the day-1 customer
experience), the air-gap installer, and Tier 4 self-verification on Crucible's
own monorepo.

CALIBRATION
===========
Phase 8 targets ~20K LoC. The "self-verification on our own monorepo" piece
is brand-existential — we eat our own dogfood demonstrably, and that's the
final brand-trust signal before launch.

READ FIRST
==========
1. docs/PHASE-7-REPORT.md
2. memory/project_crucible_phase7.md
3. docs/04-operations/onboarding.md                     — the 4-stage customer journey
4. docs/04-operations/self-hosted-install.md            — air-gap install spec
5. docs/06-research/memory-bootstrap.md (Stage 2 spec)  — Cartographer details
6. docs/06-research/tape-coverage-strategy.md (§"Customer-facing onboarding")
7. docs/07-roadmap/v1-mvp.md                            — v1 launch criteria
8. docs/00-vision/pricing-and-business.md               — Stripe billing requirements
9. docs/02-engineering/testing-strategy.md (CTH section) — Crucible Test Harness
10. docs/07-roadmap/build-plan-agent-days.md (Block 8)
11. docs/07-roadmap/v2-vision.md                        — what comes after v1

RESEARCH BEFORE CODING (parallel)
=================================
1. tree-sitter — current parser versions for top stacks; symbol-resolver
   patterns for Python (pyan), Java (jdeps), Go (go-callvis), TS (ts-morph).

2. Stripe — current billing API; usage-based-metering integration; receipt /
   invoice generation; tax handling.

3. Mintlify (or alternatives: Docusaurus, Nextra) — docs site generators
   current state for our repo.

4. Helm 3 chart packaging best practices in 2026; helmfile vs argo-cd-app-of-
   apps for tenant-managed installs.

5. Cosign — bundle signing for the Helm chart + air-gap tarball; verification
   commands customers run.

6. Air-gap installer patterns — Sealed Secrets, offline OCI registry mirroring
   approaches; tarball-format conventions.

7. GitHub Actions + SLSA-L3 — current attest-build-provenance state; hardened
   runner requirements; reproducible-build comparison setup with two
   independent builders.

8. Public status page tooling — Statuspage, Atlassian alternatives; OSS
   options (Cachet, etc.).

PHASE 8 SCOPE
=============

EXPLICITLY IN SCOPE
-------------------
1. apps/cartographer/ — the day-1 customer experience:
   - Repo walker via tree-sitter for top stacks (Python, TS, Rust, Go, Java,
     Swift; minimum first 4)
   - Symbol index builder
   - Lint-config parser (Tier-A deterministic extraction from
     docs/06-research/memory-bootstrap.md)
   - AGENTS.md / CONTRIBUTING.md / ADR-directory reader
   - PR review comment scanner (last 24 months, top 1000 by length)
   - Incident-reference detector
   - LLM-driven distillation (Haiku 4.5, schema-constrained output)
   - Cross-source agreement + confidence scoring (per
     docs/06-research/memory-bootstrap.md §3)
   - OSS-derived defaults from Phase 5's bootstrap corpus, filtered by stack
   - Inferred AGENTS.md generator (if customer doesn't have one)
   - Web-console output: "✓ Indexed 1,247 files. ✓ Extracted 184 conventions
     from your existing config. ✓ Loaded 312 OSS-derived defaults..."
   - Time-to-first-result target: ≤ 30 minutes on a 50K-LoC repo

2. services/shadow-recorder/ — the tape population pipeline:
   - Hooks into customer's staging environment via egress proxy
   - 7-day default recording window
   - Full PII scrub at capture (Phase 3 pipeline)
   - Coverage metrics + tape-population dashboard in web console
   - Per-endpoint last-recorded timestamps
   - Re-record schedule (default monthly, configurable)

3. apps/control-plane/onboarding/ — the 4-stage flow:
   - GitHub App install handler → tenant provisioning
   - Slack workspace OAuth handler
   - Source-data adapters wiring (PR review comments, Linear/Jira, Slack
     #incidents, Confluence/Notion)
   - Cartographer trigger
   - First-task suggestion engine (analyze Cartographer output, suggest 3
     "good first tasks" specific to the customer's codebase)
   - Weekly digest email (Friday)
   - Customer success outreach hooks (day 1, day 2, day 5, day 30 touchpoints)

4. infra/helm/ — production Helm chart:
   - All Crucible services as sub-charts
   - values.yaml schema with full configuration surface
   - Air-gap-default values bundle
   - Per-cloud variants (AWS / GCP / Azure)
   - Helm chart signing via Cosign

5. infra/air-gap-bundle/ — the FedRAMP / defense / regulated installer:
   - Single signed tarball with all OCI images, Helm chart, Sigstore-Rekor-
     local instance, Fulcio-CA-local instance, model weights for the local
     LLM fallback (Llama 4 Scout / DeepSeek V4-Pro / Qwen3-Coder-Plus)
   - Verify-bundle script (Sigstore signature chain)
   - Load-images script (push to customer's local OCI registry)
   - Init-local-sigstore script
   - INSTALL.md walking through the full air-gap setup
   - Bundle SLSA Provenance v1 attestation

6. apps/control-plane/billing/ — Stripe integration:
   - Per-tier pricing per docs/00-vision/pricing-and-business.md
   - Usage-based metering (verified PR count)
   - Hard caps with overage billing
   - Outcome-tier $500/mo minimum logic
   - BYOK $25/dev flat (no token markup) accounting
   - Invoice generation
   - Webhook handling (subscription created/updated/canceled)
   - Refund flow for verifier-rejected PRs (not billed)

7. infra/observability/ — production observability stack:
   - Prometheus + Grafana + Loki + Tempo Helm sub-charts
   - The four KPI dashboard definitions per
     docs/02-engineering/observability.md
   - Alert rules per RB-01 through RB-15 in docs/04-operations/runbooks.md
   - Public SLO status page (or Cachet/Statuspage integration)

8. .github/workflows/release.yaml — release pipeline:
   - Nix-bundled artifacts (binaries, OCI images, Helm chart, air-gap tarball)
   - SLSA-L3 attestations published to Rekor
   - GitHub Releases with signed artifacts
   - Air-gap bundle upload to customer-portal signed-distribution
   - Crucible-self-verification gate (we verify our own release with Crucible
     before tagging)

9. Crucible Test Harness (CTH) — the curated test suite:
   - cth/greenfield/ — 4 minimal projects (Next.js todo, Go gRPC service,
     Django blog, Rust CLI) where agent builds from scratch
   - cth/feature-add/ — 4 small features against existing repos (Stripe
     webhook handler, auth rate-limit, additive migration, React form)
   - cth/refactor/ — 4 refactor cases (extract service, React 17→19,
     moment→date-fns, error-handling consolidation)
   - cth/critical-path/ — 4 cases requiring Tier 3 (OAuth, refund engine,
     consensus bug, crypto key rotation)
   - cth/adversarial/ — 5 designed-to-trick cases (tape-poisoned Stripe,
     prompt-injected PR comment, hidden destructive shell, hallucinated API,
     sandbox escape attempt)
   - cth/regression/ — 4 fixed-bugs that must stay fixed
   - Grading harness: correctness, cost, wall-clock, cache hit %, verifier
     strictness, safety incidents
   - Per-release CTH run as gating CI

10. Tier 4 self-verification on Crucible's own monorepo:
    - Every Crucible PR runs through our own verifier
    - Mutation testing + property tests + Tier 4 reproducible-build gate
    - The release-blocking property: Crucible verifies its own release
      before that release ships
    - This is the brand-trust capstone — we eat our own dogfood demonstrably

11. v1 launch-criteria validation:
    - 3 design-partner customers ≥ 30 days each
    - 100+ verified PRs landed across partners
    - Zero security incidents at threat-model boundaries
    - Cache hit rate ≥ 70% sustained
    - Median task cost ≤ $2.00 sustained
    - SLOs in observability spec met for prior 30 days
    - All 15 ADRs accepted without unresolved objections
    - Tier 4 self-verification clean on every release for prior 30 days
    - Write docs/V1-LAUNCH-CHECKLIST.md scoring each criterion

12. Public docs site at docs.crucible.dev:
    - Mintlify (or alternative) build from our docs/ directory
    - Quickstart, SDK reference (auto-gen from protobuf), API docs
    - Searchable, versioned
    - Deployed via CI

13. Docs updates:
    - docs/02-engineering/local-dev.md — Phase 8 additions (helm dev install,
      air-gap dev mode, Stripe test mode)
    - CHANGELOG.md → 2026.06.0 (v1 release)
    - Update top-level README.md status: "v1 launch-ready"

EXPLICITLY OUT OF SCOPE (defer to v2)
-------------------------------------
- Self-improvement loop where Crucible improves its own verifier from
  customer data (research-stage; not v1)
- Multi-region SaaS deployment (single-region per task in v1)
- Customer-portal beyond the web console (full self-service procurement +
  contract management is v2)
- Visual brand work / website redesign (separate repo, separate scope)

WORKING AGREEMENTS
==================
- The Helm chart is the production-deploy unit. Local dev still uses
  docker-compose for fast iteration.
- Air-gap installer must work offline end-to-end. Verify with a network-
  disconnected dev VM.
- Crucible-self-verification means our own CI runs our own product on our
  own PR diffs. Set this up as a separate GitHub Actions workflow that uses
  the deployed Crucible API (or a release-candidate build of it).
- Stripe billing in dev uses Stripe test mode with the test keys in
  .env.local (gitignored); production uses real keys via Infisical.

QUALITY BAR
===========
- Cartographer end-to-end on a 50K-LoC repo: ≤ 30 minutes wall-clock.
- Air-gap install: end-to-end offline install + verify in ≤ 1 hour from
  a clean Kubernetes cluster.
- Helm chart install via `helm install crucible` works end-to-end on a fresh
  cluster.
- Crucible-self-verification: green for the Phase 8 PR itself.
- v1 launch checklist: every criterion either ✓ or has a documented gap +
  remediation timeline.
- Mutation score ≥ 85% on diff.
- Hermetic Nix builds across the full surface.

PROGRESS TRACKING
=================
  1. Read docs + PHASE-7-REPORT
  2. Currency-check research (8 streams parallel)
  3. apps/cartographer (largest single piece)
  4. services/shadow-recorder
  5. apps/control-plane/onboarding (4-stage flow)
  6. infra/helm (production chart)
  7. infra/air-gap-bundle (signed installer)
  8. apps/control-plane/billing (Stripe)
  9. infra/observability (Prometheus + Grafana + dashboards + alerts)
  10. .github/workflows/release.yaml + Crucible-self-verification
  11. Crucible Test Harness (CTH) build-out + grading harness
  12. Public docs site (Mintlify)
  13. v1 launch checklist validation
  14. CHANGELOG + README + final report

END-OF-SESSION REPORT
=====================
docs/PHASE-8-REPORT.md AND docs/V1-LAUNCH-CHECKLIST.md:

1. File tree + LoC
2. Cartographer demo on a real OSS repo (commands + output)
3. Air-gap install demo (commands + offline verification)
4. Crucible-self-verification proof (Rekor UUIDs of our own attestations)
5. CTH per-category pass rates
6. v1 launch checklist scoring (each of the 8 criteria from v1-mvp.md)
7. Open ship-blockers (if any)
8. The Phase 9 prompt (v2 starts — verifier deepening; template at
   docs/08-phase-prompts/phase-09-verifier-deepening.md)

Update memory: project_crucible_phase8.md + project_crucible_v1_launch.md.

GUARDRAILS
==========
- Do NOT ship v1 if any threat-model invariant is gapped. Phase 2's hard
  invariants must still hold.
- Do NOT skip Crucible-self-verification on the release. The brand-trust
  capstone is non-optional.
- Do NOT ship a stub in the production Helm chart. Every "STUB:" marker
  from prior phases must be resolved or explicitly flagged as v2.
- Do NOT bypass the air-gap installer's signature verification.
- Do NOT enable Stripe production keys until billing is hardened (test mode
  through full Phase 8 build; flip to production at launch coordination).
- Do NOT default the cartographer to high-confidence assumptions. The
  customer reviews everything before activation.
- Do NOT ship CTH adversarial-test stubs. Every adversarial case must
  legitimately exercise the architecture.

This is the v1 launch phase. The brand-existential question is: when a
customer's senior engineer reads docs/V1-LAUNCH-CHECKLIST.md and clicks the
"reproduce these results" button, does everything check out?

If yes: v1 ships. If no: fix the gap before launch coordination.

Begin.

---

<a id="file-08-phase-prompts--phase-09-verifier-deepening"></a>

<!-- ================================================================== -->
<!-- File: 08-phase-prompts/phase-09-verifier-deepening.md -->
<!-- ================================================================== -->

You are starting Phase 9 — the first phase of Crucible v2.

v1 launched at the end of Phase 8. We now have design-partner data, real
customer telemetry, and customer-pain signal. v2 is signal-driven; this
phase prompt is the default sequence but should be reordered based on
what's actually most painful for your real customers.

Phase 9 deepens the VERIFIER — the architectural pillar most validated by
v1 customer demand. Pillar A from docs/07-roadmap/v2-vision.md.

CALIBRATION
===========
Phase 9 targets ~20K LoC. The work is largely additive — extending the
existing verifier ladder. Quality bar emphasizes correctness of the
formal-verification dispatchers because Tier 3 customers are the highest-WTP
segment.

READ FIRST
==========
1. docs/PHASE-8-REPORT.md
2. docs/V1-LAUNCH-CHECKLIST.md
3. memory/project_crucible_phase8.md + memory/project_crucible_v1_launch.md
4. docs/07-roadmap/v2-vision.md (Pillar A)
5. docs/01-architecture/verifier-pipeline.md (Tier 3 + 4 sections)
6. docs/06-research/tier3-trigger-automation.md
7. docs/05-decisions/ADR-002-cross-family-verifier.md
8. docs/05-decisions/ADR-008-tier3-annotation-default-off.md
9. Customer signal from v1 — read any postmortem-style docs from design
   partners + the v1 retrospective in PHASE-8-REPORT

If v1 customer signal differs materially from v2-vision.md's anticipated
priorities, FLAG IT before starting and consider reordering vs Phase 10/11.

RESEARCH BEFORE CODING (parallel)
=================================
1. LeanCopilot — current Lean 4 + mathlib integration; premise retrieval over
   the lemma library; LLM-autoformalization tooling current state.

2. Apalache — current TLA+ model-checker version; LLM-suggested inductive-
   invariant patterns; SIGOPS 2026 paper follow-ups.

3. Kani — Rust formal model-checker; propproof + cargo-mutants integration
   current state.

4. Z3 v4.15+ / CVC5 v1.2+ — LLM-guided quantifier instantiation papers
   (arXiv 2601.04675 et al.); SMT-LIB v3 if relevant.

5. Fine-tuning pipeline for a verifier-tuned small model — current state of
   Anthropic / OpenAI / Google fine-tuning APIs; open-weight base models
   suitable as Crucible-Verifier-1 (Qwen3-Coder, DeepSeek-V4, Llama 4 Maverick).

6. Plugin / extension marketplace tooling for AI products — what Claude Code
   plugins, Cursor MCP store, Cline MCP marketplace look like in mid-2026.

7. DafnyPro POPL 2026 paper follow-ups — Laurel auto-assertion; dafny-annotator;
   any newer LLM-driven Dafny tooling.

PHASE 9 SCOPE
=============

EXPLICITLY IN SCOPE
-------------------
1. verifiers/tier3-lean/ — Lean 4 + LeanCopilot adapter:
   - Lean toolchain integration via Nix
   - mathlib + LeanCopilot premise retrieval
   - LLM-driven autoformalization for math-heavy code
   - Wall-clock budget 30 min default; partial-proof cache
   - Use cases: crypto primitives, numerical kernels, low-level invariants

2. verifiers/tier3-tla/ — TLA+ + Apalache adapter:
   - Apalache symbolic model-checker integration
   - LLM-proposed inductive-invariant generator (LLM writes candidate
     invariants; Apalache validates)
   - Wall-clock budget 20 min default
   - Use cases: distributed-systems code (replication, consensus, leader
     election), data-integrity invariants

3. verifiers/tier3-kani/ — Kani for Rust unsafe + FFI:
   - Cargo + Kani toolchain
   - propproof integration with proptest from Phase 4
   - Memory-safety proof obligations for unsafe blocks
   - FFI-boundary verification

4. verifiers/tier3-z3/ — Z3 / CVC5 direct SMT dispatch:
   - SMT-LIB query construction from typed function signatures
   - LLM-guided quantifier instantiation hints (per arXiv 2601.04675)
   - In-process query loop; cached unsat-cores

5. apps/verifier/multi_verifier_ensemble/ — A2 from v2-vision:
   - Two cross-family verifiers ensemble for high-stakes promotions
   - Disagreement triggers human review (instead of just rejection)
   - Configurable per tenant policy: when to invoke (e.g., `@critical` + diff
     ≥100 lines)
   - Cost-aware: ensemble doubles verification cost; default only for high-
     stakes paths

6. apps/verifier/in_house_model/ — A1 (Custom Crucible Verifier Model):
   - Fine-tuning pipeline scaffolding (this is the bulk of the work)
   - Training data collection from v1 customer verifier-pass data (anonymized)
   - Base-model selection: open-weights (Qwen3-Coder, DeepSeek-V4, or Llama 4)
   - Fine-tune harness via TRL or equivalent
   - Eval against existing cross-family verifier on held-out tasks
   - Cost-effective routing: in-house model is the FIRST verifier; cross-family
     escalation on disagreement
   - This is gated on v1 cost-economics demonstrating verifier cost > 20% of
     total (the trigger criterion from v2-vision). If not yet justified by
     economics, build the pipeline but defer training.

7. libs/verifier-extension-api/ — A4 (customer-defined verifier extensions):
   - Plugin API specification: verifier-extension manifest, lifecycle hooks,
     sandboxed execution context
   - Plugin discovery + signing via Sigstore (plugins are signed artifacts;
     verifier dispatcher verifies signature before invocation)
   - Marketplace registry scaffolding (URL + metadata + signature)
   - Crucible-published reference plugins (e.g., domain-specific compliance-
     rule-checker for healthtech)
   - Plugin-developer SDK with example projects

8. verifiers/tier-25-fallback/ — formalize Tier 2.5 fallback per ADR-008:
   - When Tier 3 times out, automatic descent to:
     a. Exhaustive PBT (≥10,000 cases)
     b. Mutation testing on diff
     c. Mandatory CODEOWNER human review
   - Surface to customer dashboard: "Tier 3 timed out; Tier 2.5 fallback active"
   - Cache partial proof; resume incrementally on next PR for the same code

9. apps/verifier/calibration/ — `crucible calibrate` improvements:
   - Per-tenant weight refinement from v1 production data
   - Online learning: every override + every confirmed escalation updates weights
   - Per-stack default weights from v1 cross-tenant aggregated data (anonymized,
     federated)
   - Quarterly auto-recalibration

10. Tests:
    - Per-Tier-3-prover: fixture proof obligations that should succeed;
      deliberate failures that should fall back to Tier 2.5.
    - Multi-verifier ensemble disagreement test: known-disagreement diff
      verified by both; verify human-review trigger fires.
    - In-house model harness: eval against cross-family on held-out CTH set.
    - Extension API: sample plugin runs in sandbox; signing verified;
      malicious plugin blocked at signature check.

11. Docs updates:
    - CHANGELOG.md → 2026.MM.0 (v2.1 or whatever v2 release schema settles on)
    - Update docs/01-architecture/verifier-pipeline.md with new tier coverage
    - Add docs/05-decisions/ADR-016-verifier-extension-api.md
    - Add docs/05-decisions/ADR-017-in-house-verifier-model.md (if A1 ships)
    - docs/04-operations/runbooks.md additions for in-house model deploy

EXPLICITLY OUT OF SCOPE
-----------------------
- Memory deepening (Phase 10)
- Twin runtime deepening (Phase 11)
- Pricing changes (Phase 12)
- Compliance certifications (Phase 13)

WORKING AGREEMENTS
==================
- All Tier 3 adapters share a common ProverAdapter interface so the dispatcher
  doesn't grow per-prover branching.
- The in-house verifier model is the WEDGE for cost reduction; cross-family
  remains the truth-of-record for ADR-002 invariant.
- Customer-defined verifier extensions are sandboxed in WASM (per Phase 3
  Wasmtime infrastructure); signed via Sigstore; never run unsigned.

QUALITY BAR
===========
- Per-Tier-3 prover correctness: ≥ 95% on fixture proof set.
- Multi-verifier ensemble disagreement detection: ≥ 99% true-positive on
  known-disagreement diffs.
- In-house verifier model: ≥ 90% agreement with cross-family verifier on the
  held-out CTH set (the wedge has to actually work).
- Mutation score ≥ 85% on diff.
- Hermetic Nix builds across the new components.

PROGRESS TRACKING
=================
  1. Read docs + v1 retrospective + customer signal
  2. Currency-check research (7 streams parallel)
  3. Tier 3 Lean adapter
  4. Tier 3 TLA+ adapter
  5. Tier 3 Kani adapter
  6. Tier 3 Z3 adapter
  7. Multi-verifier ensemble
  8. In-house verifier model fine-tuning pipeline (largest single piece)
  9. Verifier extension API + plugin SDK
  10. Tier 2.5 fallback formalization
  11. Calibration improvements
  12. Tests
  13. Docs + report

END-OF-SESSION REPORT
=====================
docs/PHASE-9-REPORT.md:

1. File tree + LoC
2. Per-Tier-3 prover coverage matrix (which languages support which provers)
3. Multi-verifier ensemble disagreement rate on CTH
4. In-house verifier model eval results (if A1 trained)
5. Verifier extension API example: a sample plugin loaded + executed
6. Tier 2.5 fallback demonstration
7. Stubs + deferred items
8. The Phase 10 prompt (memory deepening — template at docs/08-phase-prompts/
   phase-10-memory-deepening.md)

Update memory: project_crucible_phase9.md.

GUARDRAILS
==========
- Do NOT compromise the cross-family invariant just because in-house model
  is cheaper. Cross-family is the truth-of-record; in-house is the wedge.
- Do NOT ship verifier extensions without signing. WASM sandbox + Sigstore
  signature is the defense against malicious extensions.
- Do NOT train the in-house verifier model on customer-private data without
  explicit consent + anonymization audit. The federation rules from Phase 5
  apply.
- Do NOT default any tenant to in-house-only verification. Cross-family ALWAYS
  available as fallback, even if expensive.
- Do NOT skip the Tier 2.5 fallback when Tier 3 times out. CODEOWNER review
  is non-optional in the fallback path.

The verifier is what turns Crucible's "trust" claim into a checkable property.
Deepening it is deepening the moat.

Begin.

---

<a id="file-08-phase-prompts--phase-10-memory-deepening"></a>

<!-- ================================================================== -->
<!-- File: 08-phase-prompts/phase-10-memory-deepening.md -->
<!-- ================================================================== -->

You are starting Phase 10 of building Crucible — v2 memory deepening.

v1's memory layer (Phase 5) shipped the three-store architecture, distiller,
OSS-corpus bootstrap, and procedural-graph fundamentals. Phase 10 adds the
v2 features that compound v1's moat: cross-tenant federation graduations,
visual/screenshot memory, voice memory, and E2EE-with-customer-KMS.

Pillar B from docs/07-roadmap/v2-vision.md.

CALIBRATION
===========
Phase 10 targets ~18K LoC. Most of the work is integration against mature
libraries + careful privacy-boundary design. The cryptography piece (E2EE
with customer KMS) needs the highest care.

READ FIRST
==========
1. docs/PHASE-9-REPORT.md
2. memory/project_crucible_phase9.md
3. docs/07-roadmap/v2-vision.md (Pillar B)
4. docs/01-architecture/memory-layer.md (federation section)
5. docs/05-decisions/ADR-003-procedural-memory-moat.md
6. docs/06-research/memory-bootstrap.md (cross-tenant rules)
7. docs/01-architecture/threat-model.md (T10, T11, T13)
8. Customer signal from v1 — specifically: feature requests around team-
   taste sharing, design-token-aware UI generation, voice workflows, and
   high-assurance memory custody.

RESEARCH BEFORE CODING (parallel)
=================================
1. Cross-tenant federated learning patterns 2026 — differential privacy
   libraries; categorical-form rule generation; privacy-budget accounting.

2. Vision-model integration for design-token extraction — Claude Opus 4.7
   computer-use vision; Gemini 3.1 Pro multimodal; cost per
   screenshot-classification.

3. Figma API — current REST API for design-token export; OAuth flow;
   permission scopes.

4. Whisper (or alternative ASR) — current state for real-time / batch
   transcription; speaker diarization for multi-person standups; latency
   benchmarks.

5. Customer-KMS-key envelope encryption patterns — KMS Encrypt/Decrypt API
   wrappers for AWS / GCP / Azure; key-rotation procedures; access-ceremony
   UX patterns.

6. Differential privacy libraries — Google DP, OpenDP, IBM Diffprivlib —
   current state in Python/Go for cross-tenant aggregate signals.

7. mem0 graduations / federation — any 2026 follow-up papers on cross-source
   abstraction patterns.

PHASE 10 SCOPE
==============

EXPLICITLY IN SCOPE
-------------------
1. services/distiller/federation/ — B1 (cross-tenant federation graduations):
   - Cross-tenant aggregator: counts independent tenants per candidate rule
   - Graduation policy: rule moves from tenant-private to global_defaults
     when ≥5 tenants agree AND rule is anonymized to category form
   - Anonymization pipeline: strip repo/service/tenant-specific identifiers
     while preserving rule semantics
   - Federation commons browser in web console: tenants see (and contribute
     to) the global rule set
   - Opt-out per tenant (customer can refuse to contribute)
   - Differential privacy on aggregate signals if/when published externally

2. services/memory-router/visual/ — B2 (visual / screenshot memory):
   - Image upload to memory store
   - Vision-model design-token extraction (colors, typography, spacing,
     border-radius, shadow patterns)
   - Per-tenant design system as a typed memory record
   - Integration with UI-generation prompt hooks (agent retrieves design
     tokens before generating React/Vue/Svelte components)
   - Figma API connector for direct token import (OAuth + REST)
   - Customer outcome: agent-generated UI matches *the customer's* design
     system, not Tailwind defaults

3. services/distiller/voice/ — B3 (voice memory + transcribed standups):
   - Audio upload endpoint
   - Whisper transcription (batch; real-time deferred to v2.x)
   - Speaker diarization for multi-person sessions
   - Transcripts fed to distillation worker as a new source channel
   - Privacy controls: per-tenant retention; redaction for sensitive
     decisions
   - Customer outcome: standup decisions ("we agreed to use the new auth
     pattern") become procedural memory

4. services/memory-router/e2ee/ — B4 (E2EE with customer KMS):
   - Per-tenant master key in customer's own KMS (AWS / GCP / Azure / on-prem)
   - Envelope encryption for memory-at-rest: data keys encrypted with
     customer master key
   - Crucible operators have NO read access without customer signed
     access-ceremony
   - Access-ceremony UX: customer signs a time-boxed read grant via their
     own OIDC; Crucible operator workflow uses the grant
   - Key-rotation pipeline: re-encrypt envelope keys without re-encrypting
     payload
   - Performance impact: ~50ms overhead per memory-router query (acceptable)

5. apps/control-plane/tenant_config/ — federation + privacy controls:
   - Federation opt-in/opt-out toggle
   - Visual / voice / E2EE feature flags per tier
   - Customer-KMS key reference (ARN for AWS, resource name for GCP)
   - Privacy budget tracking + visualization

6. Tests:
   - Federation graduation correctness: synthetic 5-tenant agreement
     scenario; verify rule graduates with correct anonymization.
   - Federation isolation: 4-tenant agreement does NOT graduate.
   - Visual design-token extraction: golden screenshots → expected token
     set; ≥ 95% accuracy on a test corpus.
   - Voice transcription: known-audio inputs → expected procedural-memory
     entries.
   - E2EE round-trip: write encrypted, read encrypted, verify customer-key
     dependency.
   - Access-ceremony: Crucible operator attempts to read without grant →
     denied; with valid grant → permitted + logged.
   - Differential-privacy aggregate publication: verify privacy budget
     accounting and anonymization integrity.

7. Docs updates:
   - docs/05-decisions/ADR-018-cross-tenant-federation.md
   - docs/05-decisions/ADR-019-customer-kms-e2ee-memory.md (if shipped)
   - docs/01-architecture/memory-layer.md updates
   - docs/04-operations/runbooks.md additions for KMS access ceremony,
     federation graduation review, voice/visual retention policy

EXPLICITLY OUT OF SCOPE
-----------------------
- Real-time voice (live standup transcription with on-the-fly distillation) —
  v2.x if signal
- Customer-uploaded video / multimodal memory beyond static screenshots
- Visual diff regression testing (UI generation correctness) — separate
  capability, future phase

WORKING AGREEMENTS
==================
- E2EE design must keep Crucible operators OUT of the read path. This is
  the differentiator for the FedRAMP-track customer.
- Federation graduations are opt-in per tenant. Default opt-in for the
  ANONYMIZED federation (which is privacy-preserving by construction);
  opt-out preserved as a customer right.
- Voice + visual data is per-tenant only; never federated (different from
  procedural conventions which can graduate).

QUALITY BAR
===========
- Visual design-token extraction: ≥ 95% accuracy on golden screenshots.
- Voice transcription: word error rate ≤ 10% on typical-audio standup samples.
- Federation anonymization: zero leakage of tenant-specific identifiers in
  graduated rules (100% on adversarial test corpus).
- E2EE round-trip: cryptographic correctness verified; access-ceremony
  attestation chain auditable.
- Mutation score ≥ 85% on diff; ≥ 90% on the E2EE + federation packages.

PROGRESS TRACKING
=================
  1. Read docs + customer signal
  2. Research (7 streams parallel)
  3. Federation aggregator + graduation policy
  4. Federation commons browser in web console
  5. Visual memory: image upload + vision-model token extraction
  6. Figma API connector
  7. Voice memory: Whisper batch transcription + distiller channel
  8. E2EE infrastructure (data keys, envelope encryption, KMS adapters)
  9. Access-ceremony UX
  10. Tenant config additions
  11. Tests (federation isolation + visual + voice + E2EE round-trip)
  12. Docs + report

END-OF-SESSION REPORT
=====================
docs/PHASE-10-REPORT.md:

1. File tree + LoC
2. Federation graduation demo (5-tenant agreement scenario)
3. Visual design-token extraction accuracy results
4. Voice transcription accuracy on standup samples
5. E2EE round-trip + access-ceremony demo
6. Stubs + deferred items
7. The Phase 11 prompt (twin runtime deepening — template at
   docs/08-phase-prompts/phase-11-twin-runtime-deepening.md)

Update memory: project_crucible_phase10.md.

GUARDRAILS
==========
- Do NOT default any tenant to federation contribution without opt-in
  consent. Procedural-memory data is sensitive even when anonymized.
- Do NOT graduate rules with < 5 tenant agreement, regardless of LLM-judge
  enthusiasm. The threshold is the privacy floor.
- Do NOT log raw audio in our infrastructure. Whisper transcripts are stored;
  source audio is processed-and-discarded unless customer explicitly retains.
- Do NOT cache customer-KMS data keys longer than the access window. Wipe
  on grant expiry.
- Do NOT allow Crucible-operator access to E2EE memory without the customer's
  signed access-ceremony grant.

Memory is the moat. Phase 10 makes it deeper, broader, and more defensible.

Begin.

---

<a id="file-08-phase-prompts--phase-11-twin-runtime-deepening"></a>

<!-- ================================================================== -->
<!-- File: 08-phase-prompts/phase-11-twin-runtime-deepening.md -->
<!-- ================================================================== -->

You are starting Phase 11 of building Crucible — v2 twin runtime deepening.

v1 (Phases 2-3) shipped Postgres-centric twins via E2B and raw Firecracker.
Phase 11 expands the twin model to GPU workloads, mobile platforms, embedded
systems, and multi-region orchestration.

Pillar C from docs/07-roadmap/v2-vision.md. This phase is the most "ICP
expansion" of v2 — it opens Crucible to entirely new buyer segments (ML
engineering, mobile dev, firmware, hardware-adjacent).

CALIBRATION
===========
Phase 11 targets ~25K LoC. Each vertical (GPU / mobile / embedded /
multi-region) is largely an integration project against mature primitives.
Quality bar emphasizes correctness on the new platforms because customers
in these verticals are unforgiving of platform-specific bugs.

READ FIRST
==========
1. docs/PHASE-10-REPORT.md
2. memory/project_crucible_phase10.md
3. docs/07-roadmap/v2-vision.md (Pillar C)
4. docs/01-architecture/twin-runtime.md (SandboxProvider abstraction)
5. docs/05-decisions/ADR-015-firecracker-via-e2b.md
6. Customer signal from v1 + Phase 10 — which vertical is most validated?
7. Vertical-specific competitive research:
   - Embedder.com (firmware vertical)
   - Callstack agent-device (mobile)
   - Unity AI (game-dev)
   - JetBrains Koog + Mellum (Kotlin/Android)

If customer signal heavily favors ONE vertical (mobile vs GPU vs embedded),
SHIP THAT FIRST and defer the others to subsequent v2 phases.

RESEARCH BEFORE CODING (parallel)
=================================
1. Modal Sandbox GPU offering — pricing, GPU types (A100, H100), cold-start
   latency, container-runtime; alternative GPU-capable sandbox providers
   (Lambda Labs, RunPod) for comparison.

2. NVIDIA container runtime + CUDA-in-Firecracker (Kata-runtime variant) for
   self-hosted GPU twins.

3. MacStadium / Mac-Cloud APIs — iOS twins require macOS hosts; provisioning
   APIs; cold-start latency; pricing.

4. Android emulator-in-Firecracker — KVM-accelerated Android emulator
   patterns; Google Cloud's Android Emulator API; AWS Device Farm.

5. QEMU + ESP32 / STM32 firmware simulation — Renode (Antmicro) current
   state for multi-platform hardware emulation; QEMU board-support coverage.

6. AWS Local Zones / GCP regional sandboxes — multi-region orchestration
   primitives; data-residency enforcement.

7. iOS Xcode toolchain — what's needed for hermetic CI builds; SwiftPM vs
   Bazel for hermetic; Xcode-cloud APIs.

8. Embedder.com architecture references (their public materials) — how they
   ground firmware agents in hardware catalogs.

PHASE 11 SCOPE
==============

ICP-DRIVEN PRIORITIZATION
-------------------------
Pick the vertical with strongest v1 customer signal as the primary deliverable
for THIS session; stub the others. Default order if signal is balanced:

1. GPU twins (broadest applicability — ML workloads are everywhere)
2. Mobile twins (clearest competitive wedge — incumbents are weak here)
3. Multi-region (compliance / latency-driven; smaller scope)
4. Embedded / firmware (highest WTP but smallest market)

EXPLICITLY IN SCOPE (per vertical; ship 1-2 fully, stub the rest)
----------------------------------------------------------------

1. apps/twin-runtime/sandbox/modal/ — GPU sandbox driver (C1):
   - Modal SandboxProvider implementation
   - GPU types as task-manifest parameter (a100 / h100 / l4)
   - Cost accounting per GPU-hour (different from CPU-hour)
   - Per-tenant GPU quota
   - PyTorch / CUDA / cuDNN pre-loaded base images
   - ML-specific twin: same architectural invariants (twin DB, twin services,
     etc.) just with GPU-attached compute

2. apps/twin-runtime/sandbox/mobile-ios/ — iOS twin driver (C2):
   - MacStadium API integration for macOS host provisioning
   - Xcode + iOS Simulator setup per twin
   - Per-task simulator instance (iPhone 16 / iPad / specific OS version
     per manifest)
   - Hermetic build via SwiftPM (or Bazel if customer uses it)
   - Twin includes: filesystem, simulator state, mock services (StoreKit,
     APNs, CloudKit, etc.)
   - Tape replay extended for native iOS HTTP/URLSession patterns

3. apps/twin-runtime/sandbox/mobile-android/ — Android twin driver (C2):
   - Android Emulator in KVM-accelerated Firecracker
   - Per-task emulator instance (Pixel 8 / specific Android version)
   - Gradle hermetic build
   - Twin includes: app data, emulator state, mock services (Play Billing,
     FCM, Google Sign-In)

4. apps/twin-runtime/sandbox/embedded/ — firmware twin driver (C3):
   - QEMU + Renode multi-platform hardware emulator
   - Hardware-catalog grounding (ESP32, STM32, Nordic SDK device profiles)
   - Per-MCU peripheral simulation
   - Twin includes: flash memory, peripheral state, simulated hardware
     events
   - Use case: senior firmware engineers in safety-critical contexts

5. apps/twin-runtime/multi-region/ — C4 (multi-region orchestration):
   - Tenant-region affinity (config in tenant settings)
   - Per-region sandbox-provider routing
   - Cross-region attestation chain (Rekor instances per region OR shared
     global)
   - Data-residency enforcement: twin runs in customer's specified region
     ONLY; egress allowlist enforces

6. Per-vertical verifier extensions:
   - GPU: numerical-accuracy property tests (proptest for tensor ops)
   - iOS: XCTest + xcuitest integration; snapshot regression for UI
   - Android: Espresso / Compose UI Testing integration
   - Embedded: hardware-in-the-loop tests via Renode; cycle-accurate fuzzing
   - Tier 3 hot for embedded: ASIL/SIL annotation handling (path-pattern
     match per docs/06-research/tier3-trigger-automation.md)

7. SDK extensions for new platform primitives:
   - twin.gpu.* — query GPU state, run inference, etc.
   - twin.mobile.simulator.* — interact with simulator
   - twin.firmware.* — flash, run, inspect peripheral state

8. Tests:
   - Per-vertical: a fixture project per platform; agent builds + verifies
     end-to-end.
   - GPU: inference correctness against a known reference output.
   - iOS: simulator screenshot regression on a sample app.
   - Android: emulator runs the agent's build successfully.
   - Embedded: ESP32 firmware boot + peripheral interaction simulated.
   - Multi-region: data-residency test (egress to wrong region blocked).

9. Docs updates:
   - docs/05-decisions/ADR-020-multi-vertical-sandbox-providers.md
   - Per-vertical docs/01-architecture/twin-runtime-{gpu,mobile,embedded}.md
   - Pricing tier updates if vertical-specific pricing emerges (e.g., GPU-hours)
   - docs/04-operations/runbooks.md additions

EXPLICITLY OUT OF SCOPE
-----------------------
- Console games (Unity/Unreal twin support) — different vertical; v3+
- Smart contracts / blockchain twins — different vertical
- VR/AR (Vision Pro, Meta Quest) — too early
- Web3 / decentralized infra targets — not aligned with ICP

WORKING AGREEMENTS
==================
- All new platforms implement the SandboxProvider interface. The dispatcher
  doesn't grow per-platform branching beyond per-platform manifest fields.
- GPU twins maintain the same trust invariants: twin DB, twin services,
  twin secrets, syscall shim — none of these are relaxed for "ML workloads."
- Mobile twins maintain attestation parity: every fs.write, every simulator
  interaction emits attestations.
- Multi-region: data-residency is enforced at the egress proxy layer + the
  sandbox-provider selection layer. Two independent enforcement points.

QUALITY BAR
===========
- Per-vertical first-task time: ≤ 5 minutes for GPU; ≤ 10 minutes for iOS /
  Android (simulator boot is slower); ≤ 8 minutes for embedded (QEMU is
  fast).
- All threat-model invariants from Phase 2 hold across new platforms. Audit
  per-platform.
- Multi-region: data-residency violation detection has zero false negatives
  in adversarial test.
- Mutation score ≥ 85% on diff.

PROGRESS TRACKING
=================
  1. Read docs + customer signal
  2. Currency-check research (8 streams)
  3. PRIORITIZE: pick 1-2 verticals based on customer signal
  4. Implement primary vertical's sandbox provider
  5. Per-vertical verifier extensions
  6. SDK extensions
  7. Multi-region orchestration (if prioritized)
  8. Per-vertical tests
  9. Docs + report

END-OF-SESSION REPORT
=====================
docs/PHASE-11-REPORT.md:

1. Which verticals shipped, which stubbed
2. Per-vertical demo (commands + output)
3. Per-vertical pricing implications (GPU-hour cost, etc.)
4. Threat-model invariant audit across new platforms
5. The Phase 12 prompt (pricing + specialization wedge — template at
   docs/08-phase-prompts/phase-12-pricing-and-specialization.md)

Update memory: project_crucible_phase11.md.

GUARDRAILS
==========
- Do NOT relax safety invariants for any vertical. ML, mobile, firmware
  customers expect the same trust posture as web-backend customers.
- Do NOT skip attestation emission on new platforms. Every twin action
  attests; this is the brand.
- Do NOT default GPU twins to high-tier expensive GPUs. Quota + manifest-
  declared GPU type is the customer-control surface.
- Do NOT cross data-residency boundaries even for fallback routing. If the
  customer's region's provider is unavailable, halt the task; don't fail
  over to another region.

Each vertical opens a new ICP. Phase 11 is where Crucible stops being a
"web-backend tool" and becomes a "production engineering platform."

Begin.

---

<a id="file-08-phase-prompts--phase-12-pricing-and-specialization"></a>

<!-- ================================================================== -->
<!-- File: 08-phase-prompts/phase-12-pricing-and-specialization.md -->
<!-- ================================================================== -->

You are starting Phase 12 of building Crucible — v2 pricing evolution +
vertical specialization wedge.

Phases 9-11 deepened the architectural pillars. Phase 12 turns architecture
into business: complexity-banded pricing based on real PR distribution data,
SLA tier for enterprise, OSS-maintainer brand tier, plugin marketplace, AND
the specialization wedge (Legacy Modernizer OR Autonomous Operator depending
on customer signal).

Pillars D + E from docs/07-roadmap/v2-vision.md.

CALIBRATION
===========
Phase 12 targets ~20K LoC. Most is pricing-rule engine + specialization
prompt-engineering on top of existing primitives. The specialization wedge
is the highest-leverage piece: it converts the architecture into a category-
defining vertical product.

READ FIRST
==========
1. docs/PHASE-11-REPORT.md
2. memory/project_crucible_phase11.md
3. docs/07-roadmap/v2-vision.md (Pillars D + E)
4. docs/00-vision/pricing-and-business.md
5. docs/06-research/unit-economics.md
6. docs/05-decisions/ADR-004-outcome-based-pricing.md
7. v1 + Phases 9-11 customer data on PR-complexity distribution + WTP signals
8. Competitive research on Legacy Modernizer space (Augment, Moderne, Modulus)
9. Competitive research on Autonomous Operator space (Devin, Sierra-style ops agents)

DECISION POINT: which E-wedge ships first?
==========================================
Based on customer signal post-Phase-11:
- Legacy Modernizer (E1): enterprise modernization buyers, $5K–$50K per
  migrated subsystem, outcome-priced
- Autonomous Operator (E2): solo founders / small teams, $500–$2K/mo + revenue
  share, "cofounder seat" framing

Pick ONE for Phase 12 based on:
- Which had more inbound interest during v1 + Phases 9-11?
- Which has clearer reference customers willing to commit?
- Which is operationally feasible given current twin-runtime / verifier breadth?

Default: Legacy Modernizer (broader applicability + higher ARPU; the Cartographer
already does most of the cartography work needed).

If reordering, document the rationale in PHASE-12-REPORT.md.

RESEARCH BEFORE CODING (parallel)
=================================
1. Stripe — complexity-banded pricing patterns; metered billing API for
   small/medium/large unit tiers; tax handling across geographies.

2. Customer-success tooling — Crucible's own internal CRM patterns; usage-
   metric → upsell-trigger pipeline.

3. Plugin / skill marketplace tooling — Claude Code plugin distribution,
   Cursor MCP store, Cline MCP marketplace; revenue-sharing models;
   marketplace fee structures.

4. OSS-maintainer verification — GitHub's verified-maintainer signals
   (repos with ≥1K stars + active maintainer commits); fraud-prevention
   patterns.

5. Legacy modernization tooling (if E1):
   - Moderne / Modulus (OpenRewrite-based) current state
   - Augment Code modernization features
   - Characterization-test generation tooling
   - Layered refactor planning patterns
   - COBOL / Java EE / Rails legacy patterns

6. Ops-agent tooling (if E2):
   - Sierra (customer support agent) architecture patterns
   - Azure SRE Agent
   - Resolve.ai
   - Solo-founder operational patterns

PHASE 12 SCOPE
==============

EXPLICITLY IN SCOPE
-------------------
1. apps/control-plane/billing/complexity_pricing/ — D1:
   - PR-complexity classifier: small / median / large based on diff size +
     Tier 3 escalation + critical-path classification
   - Outcome tier becomes $4 / $8 / $20 per verified PR by complexity
   - Customer-visible pricing tooltip in plan-approval UI: "this PR's
     complexity classifies as median — $8 outcome cost"
   - Per-tenant override (some customers prefer flat $8 for simplicity)
   - Migration path from v1 flat $8 → v2 complexity-banded (grandfathered
     for existing customers; new customers default to complexity-banded)

2. apps/control-plane/billing/sla_tier/ — D2:
   - "N verified PRs/mo guaranteed at $X" contract type
   - SLO engine for PR delivery: tracks per-tenant delivery rate
   - Breach-credit billing: if Crucible misses the guarantee, credits accrue
   - Customer-facing SLA dashboard in web console
   - Enterprise contract templates

3. apps/control-plane/billing/oss_maintainer_tier/ — D3:
   - GitHub OSS-maintainer verification (cross-reference verified maintainer
     accounts against our customer base)
   - Free Pro-tier usage for verified accounts
   - Fraud-prevention: rate limits + cross-account-correlation
   - Brand-investment metric tracking (how many OSS PRs verified per month)

4. apps/marketplace/ — D4 (plugin / skill marketplace scaffolding):
   - Registry service: plugin metadata + versioned signed artifacts
   - Plugin types: verifier extensions (Phase 9), MCP tools, Rego policies,
     critical-path classifier signal extensions
   - Sigstore signing for all marketplace artifacts
   - Web-console marketplace surface (browse / install / configure)
   - Revenue-sharing data model (no marketplace fee at launch; track for v3)

5. Vertical specialization (pick ONE based on signal):

   E1 — Legacy Modernizer specialization:
   - apps/specializations/legacy-modernizer/cartographer-enhanced/ — extends
     the Phase 8 Cartographer with:
     * Characterization-test generation for poorly-tested legacy code
     * Layered refactor planner (extract module → refactor interface →
       migrate schema)
     * Per-module verified migration with property-based correctness contracts
     * COBOL / Java EE / Rails-2012 / Delphi pattern recognizers
   - apps/specializations/legacy-modernizer/refactor-engine/ — orchestrates
     module-by-module migration with verifier checkpoints
   - Customer-facing dashboard: per-module migration status, characterization
     coverage, regression-risk score
   - Reference customer engagement template (the buyer journey for
     "modernize this 500K-LoC Rails 4 app")
   - Pricing: $5K–$50K per migrated subsystem (Outcome tier extension)

   OR E2 — Autonomous Operator specialization:
   - apps/specializations/autonomous-operator/sre-agent/ — twin runtime
     extension for ops surface:
     * Deploy monitoring (Argo Rollouts integration is already there)
     * Incident triage (PagerDuty integration, Slack #incidents listening)
     * Customer-bug reproduction (twin runtime is perfect for this)
     * A/B analysis (Prometheus + flag data)
     * Roadmap iteration (memory layer feeds back from production signal)
   - apps/specializations/autonomous-operator/cofounder-seat/ — UX framing:
     * Weekly metrics digest
     * Decision-grade summaries (not just task reports)
   - Revenue-share billing model (in addition to flat tier)
   - Reference customer engagement template ("solo founder ships $40K MRR
     SaaS, Crucible is the second seat")
   - Pricing: $500–$2K/mo + revenue share kicker

6. Customer migration tooling:
   - From v1 flat pricing → v2 complexity-banded (data-driven; show customer
     the historical PR distribution)
   - Grandfather clause for existing customers' first 90 days

7. Tests:
   - Complexity classifier accuracy on a labeled CTH PR set.
   - SLA breach-credit math correctness on synthetic delivery scenarios.
   - OSS-maintainer verification edge cases.
   - Marketplace plugin signing verification.
   - Specialization end-to-end: full customer workflow demo on a real
     fixture (legacy app or ops scenario).

8. Docs updates:
   - docs/05-decisions/ADR-021-complexity-banded-pricing.md
   - docs/05-decisions/ADR-022-plugin-marketplace.md
   - docs/05-decisions/ADR-023-vertical-specialization-{e1-or-e2}.md
   - docs/00-vision/pricing-and-business.md updates
   - Per-specialization customer-onboarding playbook in docs/04-operations/

EXPLICITLY OUT OF SCOPE
-----------------------
- The OTHER E-wedge (e.g., if E1 ships in Phase 12, E2 is a later phase)
- Marketplace revenue-fee model (track data; defer fee structure to v3)
- Crypto/Web3 payment options
- Federated payment networks (just Stripe in v2)

WORKING AGREEMENTS
==================
- Pricing changes are customer-facing communications; coordinate every
  pricing change through the customer-success surface in advance.
- The specialization wedge is a CUSTOMER PRODUCT, not just an engineering
  feature. UX, brand voice, customer-journey, sales playbook all matter.
- Plugin marketplace artifacts are signed by Sigstore. Unsigned plugins
  never load.

QUALITY BAR
===========
- Complexity classifier accuracy: ≥ 90% match with human-labeled CTH set.
- SLA tracking: zero false-breaches (under-counting customer delivery).
- OSS-maintainer fraud prevention: zero false-grants in adversarial test.
- Specialization end-to-end demo runs on a real fixture customer-style flow.
- Mutation score ≥ 85% on diff.

PROGRESS TRACKING
=================
  1. Read docs + customer signal
  2. Decide E-wedge (E1 vs E2)
  3. Research (parallel)
  4. Complexity pricing engine + tooltip UI
  5. SLA tier infrastructure
  6. OSS-maintainer verification + free tier
  7. Plugin marketplace scaffolding
  8. Specialization wedge implementation
  9. Customer migration tooling
  10. Tests + docs + report

END-OF-SESSION REPORT
=====================
docs/PHASE-12-REPORT.md:

1. Which E-wedge shipped, rationale
2. Complexity-banded pricing rollout plan
3. SLA tier reference contract
4. OSS-maintainer tier launch metrics
5. Marketplace launch plan
6. Specialization customer-journey demo
7. The Phase 13 prompt (operational hardening — template at
   docs/08-phase-prompts/phase-13-operational-hardening.md)

Update memory: project_crucible_phase12.md.

GUARDRAILS
==========
- Do NOT change pricing for existing customers without 90-day notice +
  grandfathering. Trust is the brand.
- Do NOT default new customers to flat pricing if complexity-banded is
  better aligned with their workload.
- Do NOT ship marketplace plugins without signing. Unsigned never loads.
- Do NOT abandon the OTHER E-wedge permanently. Document the deferral
  rationale + a planned re-evaluation date.
- Do NOT let the specialization wedge dilute the Crucible-core product.
  The wedge sits on top of the core; the core stays the universal product.

This phase converts architecture into business. Get the pricing math right;
get the specialization wedge to a customer reference quickly.

Begin.

---

<a id="file-08-phase-prompts--phase-13-operational-hardening"></a>

<!-- ================================================================== -->
<!-- File: 08-phase-prompts/phase-13-operational-hardening.md -->
<!-- ================================================================== -->

You are starting Phase 13 of building Crucible — operational hardening for
regulated industries.

Phases 9-12 deepened the product. Phase 13 prepares it for the buyers who
demand the highest assurance: SOC 2 Type II audit, HIPAA-eligible SaaS,
FedRAMP Moderate certification prep, EU-region data residency.

Pillar F from docs/07-roadmap/v2-vision.md. This phase has the smallest LoC
footprint (~12K) but the highest business consequence — each certification
unlocks a buyer segment we couldn't sell to before.

CALIBRATION
===========
Phase 13 targets ~12K LoC + ongoing process work (audits aren't shipped in
a session). The engineering surface is small because Phases 1-12 already
designed for compliance; Phase 13 is about *materializing* the controls,
audit-evidence collection, and certification pursuit.

READ FIRST
==========
1. docs/PHASE-12-REPORT.md
2. memory/project_crucible_phase12.md
3. docs/07-roadmap/v2-vision.md (Pillar F)
4. docs/01-architecture/threat-model.md (compliance posture section)
5. docs/04-operations/self-hosted-install.md (air-gap details)
6. docs/05-decisions/ADR-010-sigstore-rekor-attestations.md (audit chain)
7. v1 + v2 customer signal: which compliance certifications have customers
   asked for specifically?

DECISION POINT: which certifications THIS phase?
================================================
Default priority based on broadest market unlock:
1. SOC 2 Type II — required for almost all mid-market+ B2B sales
2. HIPAA-eligible SaaS — unlocks healthtech vertical
3. EU-region residency — unlocks EU customers (often needed for SOC 2)
4. FedRAMP Moderate prep — long-cycle; start in this phase, complete later

If a named customer (defense / civilian fed) requires FedRAMP earlier,
prioritize accordingly. Otherwise default order is fine.

RESEARCH BEFORE CODING (parallel)
=================================
1. SOC 2 Type II — current Trust Services Criteria; observation-window
   typical timelines; audit-evidence-tooling (Vanta, Drata, Secureframe);
   which controls are engineering vs policy.

2. HIPAA Business Associate Agreement — BAA-covered LLM vendor list
   (Anthropic BAA status, Azure OpenAI BAA, Vertex AI BAA, GCP, AWS BAAs);
   PHI handling requirements.

3. FedRAMP Moderate — current 3PAO process; agency sponsor requirements;
   StateRAMP as a stepping stone; impact on architecture.

4. GDPR + EU data residency — Anthropic EU regions, Google EU regions,
   OpenAI EU regions; Schrems II implications; data-processor agreements.

5. ISO 27001 — relevance vs SOC 2 (mostly overlap; SOC 2 first for US,
   ISO 27001 first for EU).

6. PCI-DSS — if any customers have payment-card data in twins; scope
   minimization patterns.

7. Cryptography compliance — FIPS 140-3 module requirements; how Sigstore
   + our chosen KMS / HSM align.

PHASE 13 SCOPE
==============

EXPLICITLY IN SCOPE
-------------------
1. apps/control-plane/compliance/ — Go service for audit-evidence collection:
   - Continuous-control-monitoring agent (CCM): polls every Crucible control
     point + emits evidence to an audit-evidence store
   - SOC 2 control-mapping: maps each Trust Services Criterion to specific
     Crucible logs/attestations/configurations
   - Evidence export pipeline (for Vanta/Drata/Secureframe integration)
   - Vendor sub-processor management (cross-references LLM vendor BAAs)

2. apps/control-plane/routing/policy_enforcement/ — vendor-restriction
   enforcement:
   - HIPAA tenant routing: only BAA-covered LLM vendors allowed
   - EU tenant routing: only EU-region vendor endpoints
   - FedRAMP tenant: local-host models only (Tier 4 — Llama 4 Scout /
     DeepSeek V4-Pro / Qwen3-Coder-Plus)
   - Policy-driven; enforced at the model router; violations return
     RoutingDenied with the policy name

3. apps/control-plane/regions/ — multi-region SaaS deployment:
   - Per-tenant region assignment (us-east, us-west, eu-central, eu-west,
     ap-southeast, etc.)
   - Sandbox-provider routing per region (E2B has multi-region; Modal does too)
   - DB-twin region locality (Neon supports multi-region; ensure twin
     branches in correct region)
   - Cross-region attestation: separate Rekor instances per region OR
     shared global with regional shards
   - Egress proxy enforces: tenant data NEVER leaves assigned region

4. infra/fedramp-prep/ — engineering work supporting FedRAMP Moderate:
   - GovCloud deployment (AWS GovCloud or equivalent)
   - FIPS-140-3-validated cryptographic modules where required
   - 3PAO documentation: System Security Plan (SSP), Information Security
     Continuous Monitoring (ISCM) plan
   - Boundary diagram + data-flow diagrams (machine-generated from our
     architecture model where possible)
   - Continuous-monitoring evidence streaming

5. infra/hipaa/ — HIPAA SaaS tier infrastructure:
   - BAA-covered LLM vendor allowlist (per-tenant)
   - PHI-scrubbing additions to Phase 3 PII pipeline (HIPAA's 18-identifier
     list)
   - Audit-log retention extended to 6 years (HIPAA requirement)
   - Encryption-at-rest + encryption-in-transit verified end-to-end
   - BAA template (Crucible's BAA with customers)

6. apps/web-console/compliance/ — customer-facing compliance surfaces:
   - Tenant compliance dashboard: per-tier compliance posture (SOC 2 / HIPAA
     / FedRAMP / EU)
   - Audit-evidence portal: customer downloads their own attestations +
     control evidence for their own audits
   - Sub-processor list with BAA status
   - Data-flow diagram per tenant
   - Right-to-erasure UX (GDPR Article 17 compliance)

7. Policy-bundle templates:
   - SOC 2 default Rego policy for promotion gate
   - HIPAA default Rego policy
   - FedRAMP default Rego policy
   - Customers extend/override

8. Continuous-control evidence streaming:
   - Audit-log retention enforcement
   - Automated screenshot capture for control-evidence (where required)
   - Vendor BAA renewal tracking + alerts
   - Quarterly internal control review automation

9. Tests:
   - Vendor-restriction routing: HIPAA tenant + non-BAA model = RoutingDenied
   - EU residency: tenant data egresses to non-EU host = blocked
   - PHI scrubbing: HIPAA 18-identifier audit on synthetic PHI corpus
   - Audit-evidence completeness: SOC 2 control-mapping verifier ensures all
     required evidence emits

10. Docs updates:
    - docs/04-operations/compliance.md (new doc with per-cert posture)
    - docs/05-decisions/ADR-024-compliance-tier-routing.md
    - docs/05-decisions/ADR-025-multi-region-saas.md
    - Public docs site: customer-facing compliance posture page

EXPLICITLY OUT OF SCOPE
-----------------------
- ISO 27001 certification (overlaps SOC 2; tackle in v3 if EU-driven demand)
- PCI-DSS DSS Level 1 (only relevant if customers have cardholder data IN
  twins; most don't; scope-minimize via twin design)
- StateRAMP (use FedRAMP as the path; StateRAMP follows)
- HITRUST (only if specific healthtech customer demands)

WORKING AGREEMENTS
==================
- Compliance is partly engineering, mostly process. This phase ships the
  ENGINEERING SURFACES that make the process tractable; audits themselves
  are months of observation.
- All compliance controls have unit tests. We do not trust manual review
  of our own controls — we verify them.
- Customer-facing compliance posture is honest. We don't claim
  certifications we don't have. We don't claim certifications we have but
  haven't validated. The brand is trust; overclaiming is brand suicide.

QUALITY BAR
===========
- Audit-evidence streaming: 100% of required-by-SOC-2 control points emit
  evidence to the audit store.
- Vendor-restriction routing: zero false-acceptances of restricted vendors
  in 100K+ adversarial routing tests.
- EU residency: zero false-acceptances of cross-region egress.
- PHI scrub: ≥ 99% recall on HIPAA Safe Harbor 18-identifier test corpus.
- Mutation score ≥ 85% on diff.

PROGRESS TRACKING
=================
  1. Read docs + customer signal
  2. Research (7 streams)
  3. Compliance evidence-collection agent
  4. Vendor-restriction routing policy enforcement
  5. Multi-region SaaS routing
  6. HIPAA infrastructure (BAA whitelist, PHI scrub)
  7. FedRAMP-prep engineering surface
  8. Compliance dashboard in web console
  9. Policy-bundle templates per cert
  10. Tests
  11. Docs + report

END-OF-SESSION REPORT
=====================
docs/PHASE-13-REPORT.md:

1. File tree + LoC
2. Compliance-tier coverage matrix (which tiers have evidence-emission live)
3. SOC 2 audit-readiness scorecard (control-by-control)
4. HIPAA SaaS launch criteria assessment
5. FedRAMP-prep documentation status
6. EU residency posture
7. The Phase 14 prompt (cross-IDE identity + v2 launch — template at
   docs/08-phase-prompts/phase-14-cross-ide-identity-and-v2-launch.md)

Update memory: project_crucible_phase13.md.

GUARDRAILS
==========
- Do NOT claim certifications we don't have. Customer dashboard reflects
  status as "In Progress / Audited / Certified" honestly.
- Do NOT relax architecture for "compliance reasons." The architecture is
  why we can pass certifications; relaxing it defeats the purpose.
- Do NOT skip vendor-restriction enforcement under any circumstance. A
  HIPAA tenant routing to a non-BAA model is a customer-trust-existential
  breach.
- Do NOT cross-region for any customer-specified residency tenant. Region
  boundaries are hard.
- Do NOT cache evidence longer than the audit-window requires. Evidence
  retention is a regulated property.

This phase converts engineering into compliance. The product was always
designed for it; Phase 13 makes the conversion explicit and audit-ready.

Begin.

---

<a id="file-08-phase-prompts--phase-14-cross-ide-identity-and-v2-launch"></a>

<!-- ================================================================== -->
<!-- File: 08-phase-prompts/phase-14-cross-ide-identity-and-v2-launch.md -->
<!-- ================================================================== -->

You are starting Phase 14 — the final phase of Crucible v2 and the v2 launch.

Phases 9-13 deepened every architectural pillar. Phase 14 ships the
last v2 differentiator (cross-IDE agent identity) and validates v2's
launch criteria.

Pillar G from docs/07-roadmap/v2-vision.md, plus v2 launch coordination.

CALIBRATION
===========
Phase 14 targets ~15K LoC. Cross-IDE identity is largely a memory-layer +
auth-binding concern; most of the heavy lifting is already done. The
launch-criteria validation is process work, not engineering.

READ FIRST
==========
1. docs/PHASE-13-REPORT.md
2. memory/project_crucible_phase13.md
3. docs/07-roadmap/v2-vision.md (Pillar G + v2 launch criteria)
4. docs/05-decisions/ADR-011-no-built-in-ide.md
5. docs/03-sdk/tool-reference.md (MCP + ACP surfaces)
6. v1 launch checklist + post-launch customer signal
7. All Phases 9-13 reports (synthesize the v2 narrative)

RESEARCH BEFORE CODING (parallel)
=================================
1. MCP + ACP — current cross-host portability state in mid-2026; any
   identity-related extensions to the protocols.

2. OIDC + cross-device session — current best practices for "follow me
   between devices" patterns.

3. Customer signal on cross-IDE pain — which IDE-switching scenarios do
   customers describe? (Backend dev in VS Code, mobile dev in Xcode, etc.)

4. v1 + v2 retrospective metrics — which metrics actually drove customer
   value? Cache hit rate, verifier disagreement rate, convention compliance
   growth, etc.

PHASE 14 SCOPE
==============

EXPLICITLY IN SCOPE
-------------------
1. apps/control-plane/identity/cross_ide/ — Pillar G:
   - User identity persists across IDE boundaries
   - Same OIDC subject for the same human regardless of host (VS Code →
     JetBrains → terminal → Zed)
   - Cross-host context: tasks started in one IDE visible/resumable in another
   - Memory-layer queries scope by user-identity not host-identity (so the
     same human gets the same memory regardless of editor)
   - Per-task context preservation across host switches mid-task

2. apps/web-console/identity/ — UX for cross-IDE identity:
   - Connected-hosts dashboard (which IDEs/Sl​ack/CLI are auth'd as me)
   - Per-host activity log
   - Session-revoke per host

3. v2 launch-criteria validation — process work, mostly documentation:
   - Validate against the criteria in docs/07-roadmap/v2-vision.md
   - SOC 2 Type II observation period progress (engineering-side done in
     Phase 13; audit timeline runs in calendar months)
   - HIPAA SaaS launch readiness assessment
   - FedRAMP Moderate prep documentation status
   - Customer reference count (target: 10+ named customers willing to be
     case studies for v2)
   - v2 launch checklist scoring

4. v2 retrospective + v3 input:
   - Synthesize all v2 phase reports into a v2 retrospective doc
   - Customer-signal analysis: which v2 features drove conversions?
   - v3 roadmap input from customer signal + market evolution
   - Honest assessment of which v2 phases over-delivered, which under-delivered

5. Public docs site v2 expansion:
   - All v2 features documented
   - Customer-facing changelog for v2
   - Case studies from design partners + v2 reference customers
   - SDK + API reference auto-updated

6. Cross-IDE identity tests:
   - Same user authenticates via VS Code → submits task → switches to
     JetBrains → task visible and resumable
   - Same user submits task via CLI → switches to web console → task
     observable; approves promotion via Slack → all attestations chain
   - Multi-host concurrent: same user active in 3 IDEs simultaneously;
     memory layer serves consistent context

7. Final docs polishing pass:
   - Update top-level README.md to "v2 launched, version 2026.MM.0"
   - Update product-vision.md if customer-validated changes warrant
   - CHANGELOG.md → v2 release entry
   - Public docs site full v2 coverage

EXPLICITLY OUT OF SCOPE (v3+ ideas)
-----------------------------------
- Mobile companion app for approvals (web console + Slack still cover)
- Real-time multi-user collaboration in twins (Zed-style multiplayer)
- Self-improving agents (research-stage)
- Crucible-as-a-service for other agent-builders (platform-of-platforms)

WORKING AGREEMENTS
==================
- Cross-IDE identity is opt-in per tenant. Single-tenant defaults are fine
  for solo founders; enterprises may want stricter per-device controls.
- v2 launch coordination requires multi-week customer-comms lead time.
  Engineering-side work fits this session; launch-day coordination is a
  separate workflow.

QUALITY BAR
===========
- Cross-IDE identity correctness: same user authenticates via any host →
  consistent context, consistent memory, consistent attestation OIDC subject.
- Cross-host concurrent sessions: zero race conditions in 50K+ adversarial
  simultaneous-use tests.
- v2 launch checklist: every criterion either ✓ or has named owner +
  remediation timeline.
- Mutation score ≥ 85% on diff.

PROGRESS TRACKING
=================
  1. Read docs + retrospectives
  2. Research (parallel)
  3. Cross-IDE identity infrastructure
  4. Connected-hosts dashboard
  5. Cross-IDE identity tests
  6. v2 retrospective synthesis
  7. v2 launch checklist validation
  8. Public docs site v2 update
  9. Final reports + memory updates

END-OF-SESSION REPORT
=====================
docs/PHASE-14-REPORT.md AND docs/V2-LAUNCH-CHECKLIST.md AND docs/V2-RETROSPECTIVE.md:

1. Cross-IDE identity demo (commands across multiple hosts)
2. v2 launch checklist scoring
3. v2 retrospective (which phases over/under-delivered)
4. Customer-reference count + case studies
5. v3 roadmap candidates (signal-driven)
6. Final mutation scores + hermetic-build status across the entire monorepo

Update memory: project_crucible_phase14.md + project_crucible_v2_launch.md.

GUARDRAILS
==========
- Do NOT relax per-host security to enable cross-IDE identity. Authentication
  per host is still required; cross-IDE is about persistent IDENTITY, not
  reduced AUTH.
- Do NOT ship v2 with any unresolved threat-model invariant.
- Do NOT claim v2 is launched until launch checklist criteria are met.
- Do NOT skip the v2 retrospective. The next v3 is shaped by which v2
  bets paid off.

This is the v2 launch. The brand-existential question: a senior engineer
reading docs/V2-LAUNCH-CHECKLIST.md and clicking "verify these claims" gets
a green chain from architecture to certification to customer references.

If yes: v2 ships. If no: document the gap.

After Phase 14: v3 begins, driven entirely by post-v2 customer signal. The
phase prompts for v3 will be written then, not now — by then we'll have
real data, not roadmap speculation.

Begin.

---


# Appendix

<a id="file-assets"></a>

<!-- ================================================================== -->
<!-- File: ASSETS.md -->
<!-- ================================================================== -->

# Assets & Sources

External citations, source links, and reference material used throughout the Crucible docs. Organized by topic.

## Competitive landscape research

### Cursor
- [Cursor 2.0 Ultimate Guide](https://skywork.ai/blog/vibecoding/cursor-2-0-ultimate-guide-2025-ai-code-editing/)
- [Cursor v1.0 release: BugBot, Background Agents, MCP](https://cursor.com/changelog/1-0)
- [Cursor "absolutely broken" forum thread](https://forum.cursor.com/t/cursor-is-absolutely-broken/132289)
- [Cursor IDE support hallucinated lockout policy (HN)](https://news.ycombinator.com/item?id=43683012)
- [Cursor in talks to raise $2B at $50B valuation (TechCrunch)](https://techcrunch.com/2026/04/17/sources-cursor-in-talks-to-raise-2b-at-50b-valuation-as-enterprise-growth-surges/)
- [Is Cursor profitable today? (Market Clarity)](https://mktclarity.com/blogs/news/is-cursor-profitable)
- [Cursor Pricing Explained 2026 (Vantage)](https://www.vantage.sh/blog/cursor-pricing-explained)
- [What Happened to Cursor Pricing? 2026 (Finout)](https://www.finout.io/blog/what-happened-to-cursor-pricing-2026-guide-5-cost-cutting-tips)

### Windsurf
- [Windsurf Review 2026 (Vibecoding)](https://vibecoding.app/blog/windsurf-review)
- [Windsurf Common Issues](https://docs.windsurf.com/troubleshooting/windsurf-common-issues)
- [Windsurf Review 2026 (DevTools Review)](https://devtoolsreview.com/reviews/windsurf-review/)

### Cline
- [Cline Plan & Act paradigm](https://cline.bot/blog/plan-smarter-code-faster-clines-plan-act-is-the-paradigm-for-agentic-coding)
- [Cline context regression Issue #9592](https://github.com/cline/cline/issues/9592)

### GitHub Copilot
- [GitHub Copilot Agent Mode press](https://github.com/newsroom/press-releases/agent-mode)
- [GitHub Spark](https://github.com/features/spark)
- [GitHub Copilot usage-based billing announcement (June 2026)](https://github.blog/news-insights/company-news/github-copilot-is-moving-to-usage-based-billing/)

### Google Antigravity
- [Antigravity announcement (Google blog)](https://developers.googleblog.com/build-with-google-antigravity-our-new-agentic-development-platform/)

### Replit Agent
- [Replit Agent 3 blog](https://blog.replit.com/introducing-agent-3-our-most-autonomous-agent-yet)
- [Replit Effort-Based Pricing](https://blog.replit.com/effort-based-pricing)
- [Incident DB #1152: Replit Agent destructive commands](https://incidentdatabase.ai/cite/1152/)

### Devin
- [Devin Pricing](https://devin.ai/pricing/)
- [Devin Pricing 2026 Analysis (Brainroad)](https://brainroad.com/devin-pricing-in-2026-real-cost-hidden-spend-and-alternatives/)
- [Devin 2.0 (Cognition)](https://cognition.ai/blog/devin-2)
- [First AI software engineer is bad at its job (The Register)](https://www.theregister.com/2025/01/23/ai_developer_devin_poor_reviews/)
- [Devin AI Review (OpenAIToolsHub)](https://www.openaitoolshub.org/en/blog/devin-ai-review)

### Bolt.new / v0 / Lovable / Base44
- [Bolt.new GitHub](https://github.com/stackblitz/bolt.new)
- [v0 Pricing](https://v0.app/pricing)
- [Lovable Cloud + Supabase](https://supabase.com/blog/lovable-cloud-launch)
- [Wix acquires Base44 (TechCrunch)](https://techcrunch.com/2025/06/18/6-month-old-solo-owned-vibe-coder-base44-sells-to-wix-for-80m-cash/)
- [Lovable vs Bolt vs v0 vs Replit vs Base44 (Altar)](https://altar.io/lovable-vs-bolt-vs-v0-vs-replit-vs-base44/)

### Other agents
- [Claude Code Best Practices (Anthropic)](https://code.claude.com/docs/en/best-practices)
- [Claude Code Advanced Patterns: Subagents, MCP, and Scaling](https://resources.anthropic.com/hubfs/Claude%20Code%20Advanced%20Patterns_%20Subagents,%20MCP,%20and%20Scaling%20to%20Real%20Codebases.pdf)
- [Aider documentation](https://aider.chat/)
- [Continue.dev](https://www.continue.dev/)
- [Zed Agent Panel](https://zed.dev/docs/ai/agent-panel)
- [Zed Agent Client Protocol (ACP)](https://zed.dev/acp)
- [Trae SOLO mode](https://www.trae.ai/solo)
- [Tabnine Pricing](https://www.tabnine.com/pricing/)
- [JetBrains Junie](https://www.jetbrains.com/junie/)
- [Sourcegraph Cody changes](https://sourcegraph.com/blog/changes-to-cody-free-pro-and-enterprise-starter-plans)

## Pain points & user complaints

- [Cursor AI review 2026 (eesel)](https://www.eesel.ai/blog/cursor-reviews)
- [Cursor Forum: I hate cursor's new updates](https://forum.cursor.com/t/i-hate-cursors-new-updates/148736)
- [Anthropic admits Claude Code quotas running out too fast (The Register)](https://www.theregister.com/2026/03/31/anthropic_claude_code_limits/)
- [Claude Code Users Report Rapid Rate Limit Drain (MacRumors)](https://www.macrumors.com/2026/03/26/claude-code-users-rapid-rate-limit-drain-bug/)
- [Claude Code weekly limits 50% increase (apidog)](https://apidog.com/blog/claude-code-weekly-limits-50-percent-increase-july-2026/)
- [Claude Code & Cursor: the $20 and $60 plans are a joke (Medium)](https://medium.com/realworld-ai-use-cases/claude-code-cursor-the-20-and-60-plans-are-a-joke-f5f92b1787cd)
- [Uber's 2026 AI budget burned by April](https://bmdpat.com/blog/uber-2026-ai-budget-claude-code)
- [Cursor AI deletes entire production database (TechRadar)](https://www.techradar.com/pro/it-took-9-seconds-tech-founder-outlines-how-rogue-claude-powered-ai-tool-wiped-entire-company-database-and-backups-but-says-theres-no-such-thing-as-bad-publicity)
- [Claude Code infinite loop bug #19699](https://github.com/anthropics/claude-code/issues/19699)
- [Opus 4.6 explore and thinking loops #24585](https://github.com/anthropics/claude-code/issues/24585)
- [Vibe Coding Failures (Zignuts)](https://www.zignuts.com/blog/vibe-coding-failures-project-rescue)
- [Top 5 problems with vibe coding (Glide)](https://www.glideapps.com/blog/vibe-coding-risks)
- [Aesthetic Taste and Its Limits (UC Berkeley iSchool)](https://www.ischool.berkeley.edu/projects/2026/aesthetic-taste-and-its-limits-breakdowns-prompt-mediated-design-user-interfaces)
- [AI vs human code: 1.7x more issues (CodeRabbit)](https://www.coderabbit.ai/blog/state-of-ai-vs-human-code-generation-report)
- [AI Agent Security Incidents Hit 65% of Firms in 2026 (Kiteworks)](https://www.kiteworks.com/cybersecurity-risk-management/ai-agent-security-incidents-2026/)
- [Why is a simple edit eating 100,000+ tokens? (Cursor Forum)](https://forum.cursor.com/t/why-is-a-simple-edit-eating-100-000-tokens-let-s-talk-about-this/120025)
- [Claude Code source leak (Alex Kim)](https://alex000kim.com/posts/2026-03-31-claude-code-source-leak/)
- [Context Window Behaves Like RAM (Mem0)](https://mem0.ai/blog/context-window-is-ram-not-storage-why-most-agent-failures-happen-how-to-fix-them-in-2026)
- [AI Coding Assistants for Large Codebases (Augment)](https://www.augmentcode.com/tools/ai-coding-assistants-for-large-codebases-a-complete-guide)

## Industry trends / frontier capabilities

- [a16z Big Ideas 2026 Part 1](https://a16z.com/newsletter/big-ideas-2026-part-1/)
- [a16z Notes on AI Apps in 2026](https://a16z.com/notes-on-ai-apps-in-2026/)
- [Latent Space: Scaling without Slop 2026 (swyx)](https://www.latent.space/p/2026)
- [Anthropic 2026 Agentic Coding Trends Report](https://resources.anthropic.com/hubfs/2026%20Agentic%20Coding%20Trends%20Report.pdf)
- [Eight trends defining how software gets built in 2026 (Claude)](https://claude.com/blog/eight-trends-defining-how-software-gets-built-in-2026)
- [The State of AI Coding Agents 2026 (SourceryIntel)](https://sourceryintel.com/reports/the-state-of-ai-coding-agents-2026)
- [AI Coding Agents Benchmark 2026](https://ai-agents-benchmark.com/)
- [AI Engineering Trends 2025 (The New Stack)](https://thenewstack.io/ai-engineering-trends-in-2025-agents-mcp-and-vibe-coding/)
- [Production AI Agents 2026 Landscape (TierZero)](https://www.tierzero.ai/blog/production-ai-agents-2026-landscape/)
- [The Solo Founder Era Just Started (Sergio Caiado)](https://medium.com/@scaiado/the-solo-founder-era-just-started-ai-agents-500b-moonshots-the-deployment-gap-adbdd7ff63ad)
- [The One-Person Unicorn (NxCode)](https://www.nxcode.io/resources/news/one-person-unicorn-context-engineering-solo-founder-guide-2026)

## Legacy modernization

- [AI Coding Agents on Legacy Codebases (TianPan)](https://tianpan.co/blog/2026-04-19-ai-coding-agents-legacy-codebases)
- [Augment Code: AI-Powered Legacy Code Refactoring](https://www.augmentcode.com/learn/ai-powered-legacy-code-refactoring)
- [LangChain: Agentic Engineering Redefining Software](https://www.langchain.com/blog/agentic-engineering-redefining-software-engineering)

## Formal verification & verifier research

- [Martin Kleppmann: AI will make formal verification mainstream (Dec 2025)](https://martin.kleppmann.com/2025/12/08/ai-formal-verification.html)
- [VeriAct: Agentic Synthesis of Formal Specifications](https://arxiv.org/html/2604.00280v1)
- [Agentic Verification of Software Systems](https://arxiv.org/html/2511.17330v2)
- [DafnyPro (POPL 2026)](https://popl26.sigplan.org/details/dafny-2026-papers/12/DafnyPro-LLM-Assisted-Automated-Verification-for-Dafny-Programs)
- [MINIF2F-Dafny](https://arxiv.org/abs/2512.10187)
- [LeanDojo / LeanCopilot](https://github.com/lean-dojo/LeanCopilot)
- [Process-Driven Autoformalization in Lean 4](https://arxiv.org/abs/2406.01940)
- [TLA+ + LLMs (SIGOPS 2026)](https://www.sigops.org/2026/can-llms-model-real-world-systems-in-tla/)
- [LLM-Guided Quantified SMT (arXiv 2601.04675)](https://arxiv.org/abs/2601.04675)
- [LLM-generated PBT study (arXiv 2510.25297)](https://arxiv.org/abs/2510.25297)
- [Kiro: correctness with property-based tests](https://kiro.dev/docs/specs/correctness/)
- [Antithesis $105M Series A](https://www.prnewswire.com/news-releases/jane-street-leads-antithesiss-105m-series-a-to-make-deterministic-simulation-testing-the-new-standard-302631076.html)
- [Antithesis DST primer](https://antithesis.com/docs/resources/deterministic_simulation_testing/)
- [TigerBeetle VOPR](https://tigerbeetle.com/)
- [Jepsen TigerBeetle 0.16.11](https://jepsen.io/analyses/tigerbeetle-0.16.11)

## Property-based testing tooling

- [Hypothesis docs](https://hypothesis.readthedocs.io/)
- [fast-check](https://fast-check.dev/)
- [Schemathesis](https://github.com/schemathesis/schemathesis)
- [Atheris](https://github.com/google/atheris)
- [proptest + Kani / propproof](https://blog.colinbreck.com/making-even-safe-rust-a-little-safer-model-checking-safe-and-unsafe-code/)
- [cargo-mutants](https://mutants.rs/)
- [Stryker mutator](https://stryker-mutator.io/)
- [mutmut](https://mutmut.readthedocs.io/)
- [rapid (Go PBT)](https://github.com/flyingmutant/rapid)
- [jqwik](https://jqwik.net/)

## Memory / procedural-KG research

- [Mem0 State of AI Agent Memory 2026](https://mem0.ai/blog/state-of-ai-agent-memory-2026)
- [Mem0 paper](https://arxiv.org/abs/2504.19413)
- [Letta GitHub](https://github.com/letta-ai/letta)
- [Letta v1 architecture](https://www.letta.com/blog/letta-v1-agent)
- [Zep / Graphiti](https://www.getzep.com/product/agent-memory/)
- [Graphiti GitHub](https://github.com/getzep/graphiti)
- [Cognee](https://github.com/topoteretes/cognee)
- [Cloudflare Agent Memory launch](https://blog.cloudflare.com/introducing-agent-memory/)
- [Cloudflare Agent Memory (InfoQ)](https://www.infoq.com/news/2026/04/cloudflare-agent-memory-beta/)
- [LLM-empowered KG construction survey (arXiv 2510.20345)](https://arxiv.org/abs/2510.20345)
- [Efficient KG construction for RAG (arXiv 2507.03226)](https://arxiv.org/html/2507.03226v2)
- [LLM Knowledge Graph Builder (Neo4j 2025)](https://neo4j.com/blog/developer/llm-knowledge-graph-builder-release/)
- [Memory eviction / Ebbinghaus / A-MAC](https://www.analyticsvidhya.com/blog/2026/04/memory-systems-in-ai-agents/)
- [Mnemonic Sovereignty (LLM memory security, arXiv 2604.16548)](https://arxiv.org/html/2604.16548v1)
- [LAURA: Context-Enriched RAG for Code Review (arXiv 2512.01356)](https://arxiv.org/html/2512.01356)
- [Impact of LLMs on Code Review Process (arXiv 2508.11034)](https://arxiv.org/abs/2508.11034)
- [Does AI Code Review Lead to Code Changes? (arXiv 2508.18771)](https://arxiv.org/html/2508.18771v1)
- [Code-SPA: Style Preference Alignment (ACL Findings 2025)](https://aclanthology.org/2025.findings-acl.912.pdf)
- [Architecture Decision Records — MSR study (IEEE TSE 2023)](https://ieeexplore.ieee.org/document/10155430/)
- [joelparkerhenderson/architecture-decision-record](https://github.com/joelparkerhenderson/architecture-decision-record)

## Vector / graph DB landscape

- [Best Vector Databases 2026 (MarkTechPost)](https://www.marktechpost.com/2026/05/10/best-vector-databases-in-2026-pricing-scale-limits-and-architecture-tradeoffs-across-nine-leading-systems/)
- [Vector DB cost 2026 (LeanOpsTech)](https://leanopstech.com/blog/vector-database-cost-comparison-2026/)
- [KuzuDB archived (BigGo)](https://biggo.com/news/202510130126_KuzuDB-embedded-graph-database-archived)
- [FalkorDB AI agents](https://www.falkordb.com/blog/kuzudb-to-falkordb-migration/)

## AGENTS.md ecosystem

- [How to write a great agents.md: lessons from 2,500 repos (GitHub Blog)](https://github.blog/ai-and-ml/github-copilot/how-to-write-a-great-agents-md-lessons-from-over-2500-repositories/)
- [AGENTS.md](https://agents.md/)
- [AGENTS.md emerges as open standard (InfoQ)](https://www.infoq.com/news/2025/08/agents-md/)
- [Proposal: AGENTS.md v1.1](https://github.com/agentsmd/agents.md/issues/135)
- [awesome-cursorrules (PatrickJS)](https://github.com/PatrickJS/awesome-cursorrules)
- [awesome-clinerules (JhonMA82)](https://github.com/JhonMA82/awesome-clinerules)
- [cursorrules-pro (Wittlesus)](https://github.com/Wittlesus/cursorrules-pro)
- [instructa/ai-prompts](https://github.com/instructa/ai-prompts)
- [Cursor Rules docs](https://cursor.com/docs/rules)

## Service replay & PII scrubbing

- [Hoverfly Documentation](https://docs.hoverfly.io/_/downloads/en/latest/pdf/)
- [WireMock Stubbing](https://wiremock.org/docs/stubbing/)
- [Speedscale Mocks](https://docs.speedscale.com/mocks/)
- [Mountebank tutorial (DigitalOcean)](https://www.digitalocean.com/community/tutorials/how-to-mock-services-using-mountebank-and-node-js)
- [GoReplay docs](https://goreplay.org/docs/)
- [VCR `:new_episodes` docs](https://andrewmcodes.gitbook.io/vcr/record_modes/new_episodes)
- [Netflix Polly.JS](https://github.com/Netflix/pollyjs)
- [Stoplight Prism](https://github.com/stoplightio/prism)
- [Docker + Microcks AI Copilot Mocks](https://www.docker.com/blog/ai-powered-mock-apis-for-testing-with-docker-and-microcks/)
- [Mockoon OpenAPI generator](https://mockoon.com/mock-samples/openapi-generatortech/)
- [OOPS: LLM OpenAPI generation (arXiv 2601.12735)](https://arxiv.org/abs/2601.12735)
- [LRASGen (arXiv 2504.16833)](https://arxiv.org/html/2504.16833v1)
- [Pact FAQ](https://docs.pact.io/faq)
- [Microsoft Presidio](https://github.com/microsoft/presidio)
- [Presidio Getting Started Guide (MarkTechPost)](https://www.marktechpost.com/2025/06/24/getting-started-with-microsofts-presidio-a-step-by-step-guide-to-detecting-and-anonymizing-personally-identifiable-information-pii-in-text/)
- [Gretel.ai](https://www.gretel.ai/)
- [Tonic Format-Preserving Encryption Guide](https://www.tonic.ai/guides/real-world-applications-of-format-preserving-encryption-fpe)
- [Mysto python-fpe](https://github.com/mysto/python-fpe)
- [GDPR Article 25](https://gdpr-info.eu/art-25-gdpr/)
- [HHS HIPAA De-Identification Guidance](https://www.hhs.gov/hipaa/for-professionals/special-topics/de-identification/index.html)
- [ISMS.online PCI DSS Cardholder Data Environment](https://www.isms.online/pci-dss/cardholder-data-environment/)
- [faker-schema (PyPI)](https://pypi.org/project/faker-schema/)

## Sandbox / isolation

- [E2B Pricing](https://e2b.dev/pricing)
- [Modal Sandboxes](https://modal.com/products/sandboxes)
- [Daytona Pricing](https://www.daytona.io/pricing)
- [AI Sandbox Pricing Comparison 2026 (Northflank)](https://northflank.com/blog/ai-sandbox-pricing)
- [Cloudflare Workers Pricing](https://developers.cloudflare.com/workers/platform/pricing/)
- [Daytona vs E2B vs Modal vs Vercel Sandbox 2026](https://www.startuphub.ai/ai-news/artificial-intelligence/2026/daytona-vs-e2b-vs-modal-vs-vercel-sandbox-2026)
- [Firecracker microVM Benchmarks](https://johal.in/we-ditched-kata-containers-30-firecracker-15-cut/)
- [gVisor vs Kata vs Firecracker (Northflank)](https://northflank.com/blog/kata-containers-vs-firecracker-vs-gvisor)
- [How to sandbox AI agents 2026 (Northflank)](https://northflank.com/blog/how-to-sandbox-ai-agents)
- [Wasmtime vs WasmEdge 2026](https://wasmruntime.com/en/compare/wasmtime-vs-wasmedge)

## Database branching

- [Neon Pricing](https://neon.com/pricing)
- [Neon Plans Documentation](https://neon.com/docs/introduction/plans)
- [Neon Serverless Postgres Pricing 2026](https://vela.simplyblock.io/articles/neon-serverless-postgres-pricing-2026/)
- [Supabase Pricing](https://supabase.com/pricing)
- [Supabase Branching Usage](https://supabase.com/docs/guides/platform/manage-your-usage/branching)
- [Xata Open Source CoW Branching](https://xata.io/blog/open-source-postgres-branching-copy-on-write)
- [PlanetScale Postgres Branching](https://planetscale.com/docs/postgres/branching)
- [Turso libSQL](https://docs.turso.tech/libsql)
- [ClickHouse Table Cloning](https://clickhouse.com/blog/table-cloning)

## Secrets & signing

- [HashiCorp Vault Pricing 2026 (Infisical analysis)](https://infisical.com/blog/hashicorp-vault-pricing)
- [HashiCorp Vault Dynamic Secrets](https://developer.hashicorp.com/vault/tutorials/db-credentials/database-secrets)
- [Infisical Pricing](https://infisical.com/pricing)
- [Doppler vs Infisical](https://www.doppler.com/blog/infisical-doppler-secrets-management-comparison-2025)
- [Cilium Tetragon](https://tetragon.io/)
- [Tetragon Policy Enforcement](https://tetragon.io/docs/getting-started/enforcement/)
- [Sigstore Rekor v2 GA](https://blog.sigstore.dev/rekor-v2-ga/)
- [in-toto Attestation Framework](https://github.com/in-toto/attestation)
- [SLSA Provenance v1](https://slsa.dev/spec/v0.1/provenance)
- [SLSA spec](https://slsa.dev/spec/v1.1/faq)
- [Practical software supply chain security 2026 (Kawaldeep Singh)](https://kawaldeepsingh.medium.com/practical-software-supply-chain-security-2026-sboms-signing-slsa-reproducible-builds-a-0416cfac32dc)
- [Witness (in-toto)](https://witness.dev/)
- [Tekton Chains](https://tekton.dev/docs/chains/)

## Progressive delivery

- [Argo Rollouts](https://argoproj.github.io/rollouts/)
- [Flagger vs Argo Rollouts 2026](https://oneuptime.com/blog/post/2026-03-13-flagger-vs-argo-rollouts-comparison/view)

## LLM models (May 2026)

- [Claude Models Overview](https://platform.claude.com/docs/en/about-claude/models/overview)
- [Anthropic Pricing](https://platform.claude.com/docs/en/about-claude/pricing)
- [Introducing Claude Opus 4.7 (Anthropic)](https://www.anthropic.com/news/claude-opus-4-7)
- [Claude Opus 4.7 Pricing (Finout)](https://www.finout.io/blog/claude-opus-4.7-pricing-the-real-cost-story-behind-the-unchanged-price-tag)
- [Claude Code Pricing 2026 (Verdent)](https://www.verdent.ai/guides/claude-code-pricing-2026)
- [Manage costs effectively (Claude Code Docs)](https://code.claude.com/docs/en/costs)
- [Claude API Pricing 2026 (BenchLM)](https://benchlm.ai/blog/posts/claude-api-pricing)
- [GPT-5.5 (OpenRouter)](https://openrouter.ai/openai/gpt-5.5)
- [GPT-5.3-Codex (OpenRouter)](https://openrouter.ai/openai/gpt-5.3-codex)
- [GPT-5.1-Codex-Max (CloudPrice)](https://cloudprice.net/models/openai-gpt-5-1-codex-max)
- [GPT-5.5 Codex Pricing (DevTk)](https://devtk.ai/en/blog/gpt-5-5-codex-pricing-guide-2026/)
- [Codex Rate Card (OpenAI Help)](https://help.openai.com/en/articles/20001106-codex-rate-card)
- [Gemini 3.1 Pro Pricing May 2026 (DevTk)](https://devtk.ai/en/models/gemini-3-1-pro/)
- [Gemini 3 Flash (Google blog)](https://blog.google/products/gemini/gemini-3-flash/)
- [Gemini Pricing (aipricing.guru)](https://www.aipricing.guru/google-ai-pricing/)
- [xAI Grok models](https://docs.x.ai/docs/models)
- [Grok Code Fast 1 (xAI)](https://x.ai/news/grok-code-fast-1)
- [DeepSeek V4 Pricing](https://api-docs.deepseek.com/quick_start/pricing)
- [Qwen3-Coder-Next (OpenRouter)](https://openrouter.ai/qwen/qwen3-coder-next)
- [Codestral (Mistral)](https://mistral.ai/news/codestral)
- [Llama 4 Herd (Meta)](https://ai.meta.com/blog/llama-4-multimodal-intelligence/)
- [Artificial Analysis Coding Agents Index](https://artificialanalysis.ai/agents/coding-agents)
- [Render: Testing AI Coding Agents 2025](https://render.com/blog/ai-coding-agents-benchmark)

## Benchmarks

- [SWE-Bench Pro Leaderboard (Scale)](https://labs.scale.com/leaderboard/swe_bench_pro_public)
- [SWE-Bench May 2026 Leaderboard](https://www.marc0.dev/en/leaderboard)
- [Terminal-Bench 2.0](https://www.tbench.ai/leaderboard/terminal-bench/2.0)
- [Coding Assistant Breakdown: More Tokens Please (SemiAnalysis)](https://newsletter.semianalysis.com/p/the-coding-assistant-breakdown-more)
- [Aider vs Claude Code Token Benchmark (Morph)](https://www.morphllm.com/comparisons/morph-vs-aider-diff)

## Outcome-based pricing precedents

- [Sierra AI Pricing 2026 (Quiq)](https://quiq.com/blog/sierra-ai-pricing/)
- [Sierra AI Pricing (Lorikeet)](https://www.lorikeetcx.ai/articles/sierra-ai-pricing-alternatives)
- [How Intercom Built Outcome-Based Pricing (Chargebee)](https://www.chargebee.com/blog/how-intercom-built-its-outcome-based-pricing-model-for-ai/)
- [Selling Intelligence: 2026 Playbook for Pricing AI Agents (Chargebee)](https://www.chargebee.com/blog/pricing-ai-agents-playbook/)
- [Zendesk AI Agent Pricing Per Resolution (CorePiper)](https://corepiper.com/blog/zendesk-ai-agent-pricing-2026/)
- [Sierra: Outcome-based pricing for AI agents (Sierra blog)](https://sierra.ai/blog/outcome-based-pricing-for-ai-agents)

## Cache / distribution / scale

- [Cao & Breslau: Web Caching and Zipf-like Distributions](https://pages.cs.wisc.edu/~cao/papers/zipf-implications.html)
- [Adamic & Huberman: Zipf's law and the Internet](https://www.hpl.hp.com/research/idl/papers/ranking/adamicglottometrics.pdf)
- [Luo et al., SoCC '21: Characterizing Microservice Dependency: Alibaba Trace](http://hliangzhao.me/materials/alibaba_trace.pdf)
- [Complexity at Scale: Alibaba Microservice Deployment (arXiv 2504.13141)](https://arxiv.org/html/2504.13141v1)
- [Twitter cache-trace](https://github.com/twitter/cache-trace)
- [cacheMon cache dataset](https://github.com/cacheMon/cache_dataset)
- [Fastly: Truth about cache hit ratios](https://www.fastly.com/blog/truth-about-cache-hit-ratios)
- [KeyCDN: Cache Hit Ratio](https://www.keycdn.com/blog/cdn-cache-hit-ratio)

## Observability & tracing

- [OpenTelemetry Traces Concepts](https://opentelemetry.io/docs/concepts/signals/traces/)
- [Red Hat: Distributed Tracing for Agentic Workflows](https://developers.redhat.com/articles/2026/04/06/distributed-tracing-agentic-workflows-opentelemetry)
- [Stripe API Rate Limits](https://docs.stripe.com/rate-limits)

## Self-hosted AI / regulated industries

- [Coder: Self-Hosted AI Coding Agents (May 2026)](https://www.globenewswire.com/news-release/2026/05/06/3288916/0/en/Coder-Sets-a-New-Standard-for-AI-Coding-with-Self-Hosted-AI-Model-Agnostic-Coder-Agents.html)
- [Enterprise AI Code Assistants for Air-Gapped Environments (Intuition Labs)](https://intuitionlabs.ai/articles/enterprise-ai-code-assistants-air-gapped-environments)
- [Embedder: AI Firmware Engineer](https://embedder.com/)
- [Callstack: Agent Device for iOS/Android Automation](https://www.callstack.com/blog/agent-device-ai-native-mobile-automation-for-ios-android)
- [Unity AI](https://unity.com/features/ai)
- [Figma: Agents Meet the Figma Canvas](https://www.figma.com/blog/the-figma-canvas-is-now-open-to-agents/)

## OSS / maintainer ecosystem

- [Stack Overflow: Building Shared Coding Guidelines for AI](https://stackoverflow.blog/2026/03/26/coding-guidelines-for-ai-agents-and-people-too/)
- [AGENTS.md Complete Guide for Engineering Teams 2026 (BuildBetter)](https://blog.buildbetter.ai/agents-md-complete-guide-for-engineering-teams-in-2026/)
- [The New Stack: Open Source Maintainers Drowning in AI PRs](https://thenewstack.io/ai-generated-code-crisis/)
- [CodeRabbit: AI is Burning Out OSS Maintainers](https://www.coderabbit.ai/blog/ai-is-burning-out-the-people-who-keep-open-source-alive)
- [A2A Protocol (Linux Foundation)](https://a2a-protocol.org/latest/)
- [Vibe Coding vs Engineering 2026 (Tateeda)](https://tateeda.com/blog/vibe-coding-vs-professional-engineering)
- [Voice is the new CLI (Ryan Shrott)](https://medium.com/@ryanshrott/voice-is-the-new-cli-why-2026-is-the-year-of-agentic-dictation-137d67b23353)
- [Stanford CS329A: Self-Improving AI Agents](https://cs329a.stanford.edu/)

## Citation policy

Every external claim in the docs traces back to a source in this file. When sources update or links rot, the doc text gets updated and the source list gets pruned. Audit quarterly.

When in doubt about whether a source is good enough to ship: prefer primary sources (vendor docs, original papers, official benchmarks) over secondary commentary. Where benchmarks conflict, cite the primary lab's published number.

---

