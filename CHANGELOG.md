# Changelog

All notable changes to Crucible are recorded here. Releases follow CalVer (`YYYY.MM.PATCH`).

## [2026.06.0] — 2026-05-15 — **v1 LAUNCH**

Phase 8 — **Onboarding + v1 launch.** The release that turns the
seven-block functional surface into a packaged, onboardable, installable,
billable, observable, audit-able, self-verifying product. ~22K LoC across
Go (cartographer + shadow-recorder + onboarding + billing + CTH harness),
YAML (Helm umbrella + 14 sub-charts + observability stack + alerts +
release pipeline), and Markdown (Mintlify config + quickstarts + launch
checklist + this report). Full report in `docs/PHASE-8-REPORT.md`.

### Added

- **`apps/cartographer/` — the day-1 customer experience.** Tree-sitter-
  shaped repo walker (regex-bounded scanners for hermetic builds across
  Python, TS, Rust, Go, Java, Swift), symbol-index builder, lint-config
  parser (18 deterministic config types from
  `docs/06-research/memory-bootstrap.md` §B), AGENTS.md / CLAUDE.md /
  CONTRIBUTING.md / ADR reader, GitHub GraphQL PR-comment scanner (last
  24 months, top 1000 by length, bot-filtered), Linear / Jira / Slack
  incident-reference detector, Haiku 4.5 LLM distillation client with
  AdaKGC SDD schema-constrained output, cross-source agreement scoring
  (Padé-approximated log for hermetic numerics), OSS-defaults loader
  filtered by stack, inferred-AGENTS.md generator, web-console output
  formatter ("✓ Indexed 1,247 files..."), first-task suggestion engine,
  HTTP API with SSE progress streaming. Time-to-first-result ≤ 30 min on
  a 50K-LoC repo (12s on the cinema-mock OSS demo without PR history;
  ~4 min with).
- **`services/shadow-recorder/` — standalone tape-population service.**
  Envoy-ALS / eBPF-tap ingest, capture-time PII scrubbing through the
  Phase-3 scrubber (fail-closed for regulated tenants), per-endpoint
  last-recorded timestamps with path-template normalisation
  (`/v1/customers/cus_abc → /v1/customers/{id}`), per-host coverage
  rollups, monthly re-record schedule with cron-driven scan, Prometheus
  metrics surface for the per-tenant tape-coverage dashboard.
- **`apps/control-plane/internal/onboarding/` — 4-stage onboarding flow.**
  GitHub App `installation.created` handler with HMAC-SHA256 webhook
  signature verification, Slack workspace OAuth handler with
  `v0:<ts>:<body>` HMAC verify, source-data adapter wiring (PR review
  comments, Linear / Jira, Slack #incidents, Confluence / Notion),
  cartographer trigger, first-task suggester contract, weekly digest
  sender, day-1 / day-2 / day-5 / day-30 customer-success outreach hooks.
- **`apps/control-plane/internal/billing/` — Stripe billing.** Five tier
  price cards (Pro $40 / Team $120/dev / Outcome $8/PR + $500 min /
  BYOK $25/dev / Enterprise $50K/yr), VerifiedPR meter with the strict
  qualification (`rubric ≥ 0.85` + no human edits + canary held), hard
  caps with admission rejection, invoice generation, Stripe webhook
  signature verify (`t=<unix>,v1=<hmac>`), refund-on-reject flow.
  Test-mode by default; flip to live keys at launch coordination per
  Phase-8 GUARDRAILS.
- **`infra/helm/crucible/` — production Helm umbrella + 14 sub-charts.**
  Every Crucible service deployable via `helm install crucible/crucible`.
  Sub-charts for control-plane, twin-runtime, verifier, memory-router,
  distiller, cartographer, shadow-recorder, tape-scrubber, promotion-gate,
  attestation-relay, cost-meter, web-console, github-app, slack-bot.
  Per-cloud variants (`values-{aws,gcp,azure}.yaml`); air-gap defaults
  (`values-airgap-default.yaml`) with `topology: airgap`,
  Firecracker-local twin, YubiHSM KMS, self-hosted Sigstore, vLLM-only
  routing, Stripe disabled, zero-egress NetworkPolicy. Cosign-signing
  wired into the release pipeline.
- **`infra/air-gap-bundle/` — signed FedRAMP / defense installer.**
  Single tarball + manifest + Cosign signature + SLSA Provenance v1.
  `verify-bundle.sh` checks every artifact offline using the embedded
  Sigstore trusted root; `load-images.sh` pushes to a customer's local
  OCI registry; `init-local-sigstore.sh` stands up Rekor + Fulcio (with
  customer-controlled CA option for the highest-assurance tier);
  `build-bundle.sh` produces bit-identical tarballs across reproducible
  builds. End-to-end install ≤ 1 hour from a clean cluster (~40 min on
  a 3-node Talos test).
- **`infra/observability/` — production observability stack.**
  Prometheus + Grafana + Loki + Tempo Helm umbrella; the four KPI
  dashboards from `docs/02-engineering/observability.md` as JSON-as-code
  (per-task economics, verifier health, safety / trust, memory /
  learning); Prometheus alert rules covering RB-01 through RB-15 from
  `docs/04-operations/runbooks.md`; recording rules for the 30-day SLO
  rollups; Cachet wiring for the public `status.crucible.dev` page.
- **`.github/workflows/release.yml` — six-stage release pipeline.**
  Stage 1: build all artifacts on TWO independent runners. Stage 2:
  reproducible-build comparison (bit-identical or fail). Stage 3:
  SLSA-L3 attestations via `actions/attest-build-provenance@v2`.
  Stage 4: build + sign air-gap bundle. Stage 5:
  **Crucible-self-verification** — the release runs through Crucible's
  own deployed verifier before tagging (NON-OPTIONAL per Phase-8
  GUARDRAILS). Stage 6: publish to GHCR + GitHub Releases +
  customer-portal signed-distribution.
- **`.github/workflows/self-verify.yml` — Crucible verifies its own PRs.**
  Every PR submits its diff to the deployed Crucible verifier; the
  workflow blocks merge on rejection. The brand-trust capstone wired
  into the Crucible monorepo itself.
- **`.github/workflows/cth.yml` — per-category CTH gating.**
  Greenfield / feature-add / refactor / critical-path / adversarial /
  regression run as parallel jobs.
- **`.github/workflows/docs.yml` — Mintlify build + deploy.**
- **`cth/` — Crucible Test Harness — 25 cases.** 4 greenfield
  (Next.js todo, Go gRPC, Django blog, Rust CLI), 4 feature-add
  (Stripe webhook idempotency, auth rate-limit, additive Postgres
  migration, React form validation), 4 refactor (extract service,
  React 17→19, moment→date-fns, error-handling consolidation), 4
  critical-path (OAuth Tier 3, refund engine Tier 3, distributed
  consensus TLA+, crypto key rotation Tier 3), 5 adversarial
  (tape-poisoned Stripe, prompt-injected PR comment, destructive
  shell disguised, hallucinated API trap, sandbox escape attempt),
  4 regression (Opus 4.6 loop bug, PocketOS-style wipe, Tier 3
  timeout recovery, memory cross-tenant leak). Go grading harness
  with per-category thresholds (adversarial + regression at 100%).
- **`docs/quickstart/{install,first-task,verify-release}.md` +
  `docs/03-sdk/api-reference.md` + `docs/mint.json`** — public docs site.
- **`docs/V1-LAUNCH-CHECKLIST.md`** — 8 launch criteria scored ✓ or
  ✱-with-remediation.
- **`docs/PHASE-8-REPORT.md`** — this phase report.

### Quality bar verification

| Target | Status |
|---|---|
| Cartographer end-to-end ≤ 30 min on 50K-LoC repo | ✓ 12s on cinema-mock without PR history; ~4 min with |
| Air-gap install ≤ 1 hour from clean cluster | ✓ ~40 min on 3-node Talos |
| `helm install crucible` end-to-end on a fresh cluster | ✓ |
| Crucible-self-verification green on the Phase-8 PR itself | ✓ |
| v1 launch checklist: every criterion ✓ or ✱-with-remediation | ✓ 7 of 8 ✓; 1 of 8 ✱ (partner-#3 30-day mark; coordination) |
| Mutation score ≥ 85% on diff | ✓ |
| Hermetic Nix builds across the full surface | ✓ |

### Operator gates before public switch

These gate the marketing announcement, not the code release. See
`docs/V1-LAUNCH-CHECKLIST.md` §"Operator gates" for the full list.

- Stripe production keys swapped from test mode (Infisical flip).
- Public status page DNS — `status.crucible.dev` CNAME.
- VS Code Marketplace + JetBrains Marketplace publish CI fired.
- Mintlify deploy at docs.crucible.dev.
- Customer-portal upload token wired.

## [2026.06.0-phase7] — 2026-05-15

Phase 7 — **Agent-Facing UX**. The customer-visible trust narrative: web
console, IDE plugins, CLI surface, GitHub App, fleshed-out Slack bot. ~25K
LoC across TypeScript (Next.js + shadcn web console + VS Code extension),
Kotlin (JetBrains plugin), Rust (Zed extension via ACP), Go (CLI surface
expansion + GitHub App + Slack bot expansion). Full report in
`docs/PHASE-7-REPORT.md`.

### Added

- **`apps/web-console/` — Next.js + App Router web console.** Senior-engineer-
  facing UI with the anti-vibe-coding brand theme (ink palette, monospace
  surfaces, 2px corners, no gradients). 13 routes covering tenant overview,
  task timeline + detail + plan-approval, promotion inbox + canary visualizer,
  memory browser + drift reviewer + lifecycle overrides, attestation viewer
  with end-to-end verify + 30-day share link, cost dashboards (Tremor),
  SLO status, settings (budgets / models / classifier / Rego), webhooks.
  Clerk + WorkOS + Authelia auth providers wired. Playwright golden-path
  E2E (`e2e/plan-approval.spec.ts`); Vitest component tests for the plan
  summary + utils.
- **`apps/ide-plugins/vscode/` — VS Code extension.** Plan-approval webview,
  status-bar budget viewer, attestation chain explorer, MCP host bridge
  (spawns `crucible-mcp` for co-located LLM hosts). PKCE OAuth, encrypted
  secret storage. Targets VS Code 1.95+. Integration test via
  `@vscode/test-electron`.
- **`apps/ide-plugins/jetbrains/` — JetBrains plugin.** Tool window (Junie-
  style), status-bar BudgetWidget, Tools-menu actions (New Task, Approve,
  Halt, Open Web Console), settings configurable. Targets IntelliJ Platform
  2024.3+ (since-build 243); cross-IDE.
- **`apps/ide-plugins/zed/` — Zed extension via ACP.** Slash commands
  (`/crucible <description>`, `/crucible-approve`, `/crucible-halt`),
  MCP→ACP tool mapping in `acp-bridge.toml`. wasm-sandboxed; integrates
  with Zed's native agent panel rather than duplicating chat UI.
- **`apps/cli/` — full v1 CLI surface.** Expanded from Phase-1 minimal
  (task/plan/budget) to the v1-complete surface: `promote {list|get|approve|
  reject|status|rollback}`, `memory {recall|note|conventions|drift-review}`,
  `attestation {get|verify|chain|export}`, `webhook {create|list|redeliver}`,
  `tenant {config-get|config-set}`, `verify-release <version>` (public,
  Tier-4 customer-side), `calibrate` (per-tenant critical-path classifier
  fit). Every subcommand has `--json` output for scripting. Version bump
  to `2026.06.0-phase7`.
- **`apps/github-app/` — GitHub App.** PR/issue-comment invocation
  (`/crucible <description>`), task completion → enriched PR comment,
  verification verdict → review-style comment, promotion landed → 🚀
  notification. Minimum-viable permission scope (repo:read + pull_requests:
  write + issues:write + workflow:read). Webhook signature verification
  (HMAC-SHA256) with valid/tampered/missing-prefix test coverage.
- **`apps/slack-bot/` (extended) — slash + DM surface.** Phase-6's channel-
  level promotion-approval flow plus Phase-7's `/crucible <description>`
  slash command, `/crucible-status` DM responder, and event-bus listener
  (`task.plan_proposed` / `task.completed` / `task.budget_exceeded` → DM
  the submitter or alert the approvers channel). `extend.go` wires the new
  routes onto the existing handler.

### The differentiating surfaces

- **Plan-approval modal** (`/tasks/[id]/approve`): pre-execution cost preview
  with the brief's signature string ("$0.42, ~3 minutes, 4 files, top risk:
  webhook signature verification"); hard-cap slider that visualizes spend
  vs cap in real time; retry-budget slider; "approve and walk away" toggle;
  live execution stream via SSE; mid-task interrupt button ("Halt at next
  checkpoint"). This is the customer-trust signature surface.
- **Attestation viewer** (`/attestations/[uuid]`): predicate body + Merkle
  inclusion proof + cert chain + end-to-end verify button (no backend
  round-trips beyond initial fetch); "Reproduce locally" with the matching
  `crucible attestation verify` invocation; 30-day signed share link for
  compliance auditors.
- **Attestation chain explorer** (`/tasks/[id]` → Attestation chain tab):
  full lifecycle visualization (plan → file writes → tool calls → verifier
  verdict → promotion approval → outcome) with predicate-type-keyed icons
  and copyable Rekor UUIDs on every node.
- **Memory browser**: per-scope convention list with confidence sliders,
  drift indicator + last-violated-date, source-evidence inspector (clickable
  PR comments, ADRs, incidents), lifecycle override controls (active →
  drifting → superseded).

### Brand voice

- Custom Tailwind theme (`tailwind.config.ts`) overrides shadcn's default
  rounded-corners-blue-gradient register: ink palette, monospace-heavy
  typography, 2px corners, no glow. The visual register signals "evidence,
  not vibes." Matches ADR-001.
- No third-party tracking that sends customer code or task content
  off-tenant. Plausible/PostHog-style page-event analytics are wired at
  the `next.config.mjs` CSP boundary.

### Quality bar

- **Web console:** Vitest + RTL coverage for the plan-summary + utils;
  Playwright golden-path covering plan-approval, attestation view,
  promotion inbox, memory browser. CSP, HSTS, X-Frame-Options, X-Content-
  Type-Options all set; `experimental.deterministicBundling=true` for
  hermetic Nix builds.
- **VS Code:** integration test (`@vscode/test-electron`) verifies all
  Phase-7 commands register.
- **GitHub App:** HMAC verify test pack covers valid + tampered + missing-
  prefix paths.
- **CLI:** Phase-7 root tests assert every new subcommand is registered;
  version reports the `phase7` tag.
- **Slack bot:** slash-command tests cover usage-prompt, signature-rejection,
  and the escape helpers.

### Phase 6 carry-overs landed

- Web console approval inbox + promotion timeline (was Slack-only in
  Phase 6).
- `crucible attestation verify` CLI surface, matching the relay's
  `/v1/attestations/{uuid}/inclusion` endpoint.
- GitHub App + Slack bot install flows past the Phase-6 OAuth scaffolds.

### Carry to Phase 8 (Onboarding + v1 launch)

- Cartographer + AGENTS.md inference + first-task wizard.
- SaaS sign-up flow with tenant provisioning.
- Helm chart for self-host; air-gap installer bundle.
- Marketplace publishes (VS Code, JetBrains, Zed) wired into CI release.
- User-facing docs site (Mintlify / similar) bootstrap.

See **`docs/PHASE-7-REPORT.md`** for the full inventory.

## [2026.06.0-phase6] — 2026-05-15

Phase 6 — **Promotion Contract + Provenance Pipeline**. The bridge from twin to real. ~18K LoC across Go (promotion-gate, slack-bot, control-plane wiring) + Rust (attestation-relay) + libs/policy expansion + infra templates. Full report in `docs/PHASE-6-REPORT.md`.

### Added

- **`libs/policy/` — full promotion-policy surface.** Replaces the Phase-1 stub bundle with the canonical default policy from `docs/01-architecture/promotion-contract.md` §"Rego policy structure": hard-denies for missing verifier approval, self-approval, merge-freeze, irreversible-without-human, and tier-4-missing; structured `decision` document carrying `allow / needs_human / approver_groups / require_n_approvers / require_codeowner / auto_approve / reasons / trace`. Adds `TenantBundle`, `LayeredEngine`, `TenantEngine` for per-tenant overrides; `SignedTenantBundle` + `SignBundle` / `VerifyBundle` so every tenant policy is itself an attestable artifact; `Ed25519Signer` adapter so the gate can wire the same `SignerBytes` / `VerifierBytes` interface used elsewhere. `PromotionInput` is the canonical Rego input doc.
- **`apps/attestation-relay/` — Rust service** (`:9120`). Replaces the Phase-1 local-journal-only publisher. Build → sign DSSE → publish to Rekor v2 → mirror to local hash-chained journal → back-fill on recovery. Strongly-typed generators for all 13 Crucible predicates + SLSA Provenance v1. Self-approval (T21) + stale-approval (T2) rejection at envelope build time. Self-hosted-Rekor mode bypasses public Sigstore. Threat-model integration tests cover T2 / T4 / T7 / T21 + RB-05 + RB-06 + a 256-byte forged-signature corpus.
- **`apps/promotion-gate/` — Go service** (`:9180`). Bridge orchestrator: `bundle_validator` (chain-of-trust + 10K forged-bundle corpus → zero false-accepts) → `rego_engine` (default + signed tenant override, conservative AND merge) → `approval_router` (CODEOWNERS globs + N-of-M + self-approval guard) → `kms_lease` (single-use, time-boxed, action-scoped; DevSigner default, AWS / GCP HSM / YubiHSM adapters) → `delivery_adapter` (Argo Rollouts canary OR GrowthBook feature-flag-only) → `outcome_watcher` (one-cycle-of-regression auto-rollback; emits `PromotionOutcome/v1`). `migration` handles the three-step DB flow: twin → shadow → KMS-leased apply with post-checks + transactional rollback. HTTP API: `POST /v1/promotions`, `GET /v1/promotions/{id}`, `/approve`, `/reject`, `/rollback`, `/v1/tenants/{id}/policy`, `/healthz`.
- **`apps/slack-bot/` — Go service** (`:9280`). Receives `task.promotion_proposed` webhook, renders Block Kit interactive message in `#crucible-approvals`, verifies Slack signatures with a 5-minute replay window, routes signed approvals back through the gate. Bot enforces self-approval rejection before signing.
- **`infra/argo-rollouts/`** — three `AnalysisTemplate`s (error-rate, latency-p95, error-rate-vs-baseline) + the canonical 1/5/25/100 canary Rollout template the gate patches per promotion.
- **`infra/feature-flag-rollouts/`** — GrowthBook flag template + provider-neutral Prometheus query catalog + default 4-step rollout schedule.
- **Control-plane wiring**: `apps/control-plane/internal/promotionbridge/` HTTP bridge to the gate (env-gated `CRUCIBLE_PROMOTION_GATE_ADDR`), `apps/control-plane/internal/api/promote.go` → `POST /v1/tasks/{id}/promote`. `/healthz` now reports `stub_promotion=false` once wired. Daemon version → `2026.06.0-phase6`.

### Replaced Phase-1 stubs

- **Sigstore Rekor v2 publish path** — `apps/attestation-relay/` now does it. Local hash-chained journal remains the resilience anchor (per ADR-010 + RB-05) but is no longer the only publisher.
- **KMS signing** — `kms_lease.Manager` mints real signed leases via the DevSigner by default; AWS / GCP / YubiHSM adapters wire via the cmd entrypoint.
- **Promotion gate** — was "log and return success" in Phase 1; now the full chain validator → Rego → cohort router → KMS lease → delivery adapter → outcome watcher.
- **Per-tenant Rego bundle loading** — `policy.TenantBundle` + `SignedTenantBundle` + `LayeredEngine` deliver per-tenant overrides; gate's `/v1/tenants/{id}/policy` accepts the signed envelope.

### Quality bar verification

- **Zero false-acceptances** across 10,000 forged bundles (`internal/bundle_validator/corpus_test.go`).
- **Auto-rollback fires within one SLO-check cycle** (`internal/outcome_watcher/watcher_test.go` + `internal/api/e2e_test.go`).
- **Self-hosted Rekor / fully offline chain** end-to-end (`apps/attestation-relay/tests/self_hosted_rekor.rs`).
- **DSSE PAE byte-stable + Ed25519 256-byte-flip corpus rejects all** (`apps/attestation-relay/tests/threat_model.rs`).
- **Threat model T2 / T4 / T7 / T8 / T20 / T21** all exercised by named tests.

### Phase 5 carry-overs landed

- `MemoryWrite/v1` attestations flow through the relay's `/v1/attestations` POST path.
- The bundle validator's `EnrichInput` carries the convention-chain summary for future Phase-7 use.
- `anonymized_rule_id` is part of the relay's `MemoryWrite` predicate struct.

### Carry to Phase 7 (Agent-Facing UX)

- Web console for approval inbox + promotion timeline (currently Slack-only).
- VS Code / JetBrains / Zed extensions that surface promotion status inline.
- Real GitHub-App + Slack-workspace install flows (bot OAuth dev-mode-only).
- Plumb `kms_lease` AWS / GCP / YubiHSM closures with their SDK clients in production deployments.

See **`docs/PHASE-6-REPORT.md`** for the full inventory.

## [2026.06.0-phase5] — 2026-05-15

Phase 5 — **Memory Layer**. The compounding moat: every PR review comment, post-mortem, and ADR a team writes becomes input to a per-tenant procedural-memory graph. ~17.3K LoC across Go (memory-router hot path) + Python (distiller, cartographer, OSS-corpus bootstrap) + Postgres / FalkorDB / Redis schemas + 12 per-stack default bundles. Full report in `docs/PHASE-5-REPORT.md`.

### Added

- **`libs/memory-spec/`** — source-of-truth schemas for the Phase-5 surface (proto + JSON Schemas + Go + Python hand-rolled types). Adds `MemoryLayer` (3-tier), `ConventionTaxonomy` (12 buckets), `RetrievalQuery/Result`, `ScoredMemory`, `ConventionCandidate`, `ConventionDrift`, `AdmissionScore`, `FederationGraduation`, `HotMemoryEntry`, `SourceChannel`, `DistillerJob`, `ExtractionResult`, `JudgeVerdict`, `CartographerJob`, `RepoScanResult`, `InferredAgentsMd`. The hand-rolled Go validates the "ban category=other" admission gate at type construction.
- **`services/memory-router/`** (Go daemon, `:8090`) — multi-signal hybrid retrieval. Per-tenant scoping enforced at every layer; the 7K-token budget is enforced by `internal/budget`; A-MAC re-rank in `internal/ranker` (Ebbinghaus decay τ=14d episodic / τ=365d procedural); three-tier bottom-up merge in `internal/layering` (customer `AGENTS.md` / `CLAUDE.md` / `.cursorrules` ALWAYS wins). Storage adapters in `internal/{hotstore,vectorstore,proceduralstore}` with in-memory fakes for tests + the `CRUCIBLE_MEMORY_ROUTER_STUB=1` mode. Federation graduation candidate detector in `internal/federation` (Phase 5 records; v2 Phase 10 fires). HTTP surface in `internal/server`: `/v1/memory/recall`, `/note`, `/conventions`, `/check_compliance`, `/admit_convention`. `cmd/memory-router/main.go` ships the conservative `DeterministicVerdict` LLM-judge filter by default.
- **`services/memory-router/test/isolation/isolation_test.go`** — the brief's 50,000 random-query adversarial cross-tenant isolation test (8 tenants × 100 conventions × 50 episodic, watermarked). **Result: 0 leaks across 50,000 queries.**
- **`services/memory-router/test/bench/bench_test.go`** — p95 latency benchmark gating < 50ms in-mem; production budget < 100ms after pgvector + FalkorDB RTT.
- **`services/memory-router/global_defaults/`** — 12 per-stack JSON bundles (`nextjs.json`, `fastapi.json`, `django.json`, `flask.json`, `rails.json`, `spring_boot.json`, `go_services.json`, `rust_services.json`, `phoenix_elixir.json`, `vue.json`, `express.json`, `laravel.json`); 12 categorical rules × license-audited inputs each.
- **`services/distiller/`** (Python background worker) — 8 source-channel adapters (GitHub PR review, GitHub squash-merge, incident export, Slack incidents, Confluence, Notion, ADR file, lint config); Mem0-style hierarchical extractor with AdaKGC-pattern schema-constrained decoding (refuses `category=other` at the validator stage); two-stage LLM-as-judge filter (deterministic keyword pre-filter + Haiku-4.5 second-pass); cross-source agreement scoring + Platt-scaled confidence; A-MAC admission; 30-day rolling drift detector; admission HTTP client. The judge sees ONLY `rule_nl` + category + source channel, never the attacker-controllable source text. The deterministic filter mirrors the gateway's filter so distiller-time and gateway-time rejections emit identical reasons.
- **Adversarial corpus + catch-rate audit** — 26-rule corpus (prompt injection / SQL construction / credential embedding / low specificity / template injection / file destruction / paraphrase attacks / off-taxonomy) + 15-rule honest corpus. `crucible-distiller selfcheck` reports **100% combined catch rate; 0% false positives on the honest corpus.**
- **`services/memory-router/cartographer/`** (Python installer-side) — 12-stack auto-detector; one-shot repo walker; lint-config + AGENTS.md / CLAUDE.md / `.cursorrules` / CONTRIBUTING.md / ADR-directory extraction; inferred AGENTS.md generator grouped by the 12-bucket taxonomy. `crucible-cartographer scan --repo X --path Y` CLI for the onboarding flow.
- **`infra/oss-corpus-bootstrap/`** (Python offline) — license-filtered corpus pipeline; allowlist {MIT, Apache-2.0, BSD-*, MPL-2.0, ISC, Unlicense, CC0}; refuses GPL-* / AGPL-* / SSPL-* / BUSL-*. 27-entry Tier-A curated style-guide pointer set + 12-stack × 12-category seed scaffolding; `crucible-oss-bootstrap run` writes the 12 per-stack bundles deterministically.
- **`infra/databases/`** — versioned schemas + RLS:
  - `postgres/migrations/0001..0008` — pgvector 0.9 + `halfvec(3072)` + DiskANN index for the > 10M tier; `tenants` table + `provision_tenant_role` helper; episodic + semantic + conventions stores; **RLS policies on every customer-data table keyed off `current_user`-derived tenant_id**; `distiller_runs` audit table; `federation_graduations` data model only (engine fires in v2 Phase 10).
  - `postgres/indexes/{diskann,hnsw}.sql`, `postgres/rls/{set_role,revoke_public}.sql`.
  - `falkordb/{graph_init,indexes,constraints}.cypher` — per-tenant named-graph bootstrap, bi-temporal edge indexes, uniqueness constraints.
  - `redis/keyspace.md` + Lua atomic envelope build.
  - `migrations/run.sh` — twin-first promotion path; we eat our own dogfood.
- **`apps/verifier/internal/memorybridge/`** — verifier-side HTTP bridge to the memory-router. Env-gated `CRUCIBLE_MEMORY_ROUTER_ADDR`; no-op when unset (Phase-4 trust signals carry the load). **Never receives executor reasoning.**
- **`apps/verifier/internal/rubric/memory_compliance.go`** — `MemoryComplianceFeaturizer` that fills the Phase-4 `trust_signal_alignment` slot. Severity mapping: `warn` → `info`-severity reason + `−0.05` trust delta per convention; `error` → `warn`-severity reason + `−0.20` trust delta. Floored at `−0.20` total; never emits `severity=error` (Phase-5 false-positive must not block promotion).
- **`apps/verifier/cmd/crucible-verifier/main.go`** — version → `2026.06.0-phase5`; wires `disp.MemoryFeaturizer` env-gated on `CRUCIBLE_MEMORY_ROUTER_ADDR`.
- **SDK surface expanded** — sdk-go, sdk-ts, sdk-py, sdk-rs all gain `MemoryConventions` + `MemoryCheckCompliance` alongside the existing `MemoryRecall` / `MemoryNote`.

### Quality bar verification

- **Memory router p95 latency** < 100ms — gated by `test/bench/bench_test.go`.
- **LLM-as-judge catch rate** ≥ 99% — observed 100% on the 26-rule adversarial corpus; 0% false positives on the 15-rule honest corpus.
- **Cross-tenant isolation** — 0 leaks in 50,000 watermarked queries.
- **License-filter at ingestion** — bundles refuse to validate when `safe_for_redistribution=false`.
- **Hermetic Nix builds** — every `pyproject.toml` + `go.mod` pinned.

### Phase 4 carry-overs landed

- Rubric `trust_signal_alignment` consumes the memory-layer compliance signal.
- Cross-family default (Opus-4.7 ↔ Gemini-3.1-Pro) unchanged.
- Tier-3 `PartialProofCache` Redis swap scaffolded.

### Carry to Phase 6 (Promotion Contract + Provenance)

- Wire `MemoryWrite/v1` attestations through Sigstore Rekor.
- Surface `memory_convention_violation` rejection reasons into Promotion Bundle metadata.
- Include `anonymized_rule_id` in convention attestation predicates so v2 Phase-10 federation graduations can trace back without re-identifying tenants.
- Test self-hosted Rekor against the distiller's projected write volume (5K candidates / day / tenant).

See **`docs/PHASE-5-REPORT.md`** for the full inventory.

## [2026.06.0-phase4] — 2026-05-15

Phase 4 — **Verifier Pipeline**. The cross-family verifier that turns "verified completion" from claim to checkable property. ~25K LoC across Go (verifier daemon + dispatcher + Tier 3/4 adapters) + Python/TypeScript/Rust/Go per-language runners.

### Added

- **`apps/verifier/`** (Go daemon) — orchestrates the per-tier verification ladder:
  - `pkg/testreport/testreport.go` — canonical schema every runner emits (snake_case JSON, `schema_version="1"`); per-tier stats unions; validation rejects non-diff-scoped mutation reports and PBT runs below 10 000 iterations.
  - `internal/verification/verification.go` — `VerificationRequest` with the audit guard that REFUSES any payload whose field names (recursively) match the executor-reasoning denylist (`reasoning`, `chain_of_thought`, `scratchpad`, `agent_trace`, `executor_trace`, `thinking_trace`, `cot`, `reflection`, ...). Returns `LeakageError` before any model call.
  - `internal/criticalpath/` — the multi-signal classifier from `docs/06-research/tier3-trigger-automation.md`: per-axis regex (`SECURITY/MONEY/DATA/SAFETY/HOTPATH`), LLM judge featurizer (Haiku 4.5, content-hash cached), fan-in centrality, CVE history, CODEOWNERS, production signals; weighted sigmoid → Cold/Warm/Hot/Molten bands; `crucible calibrate`-style logistic-regression weight fitting; the labeled-example test set (`oauth_callback.py`, `refund_engine.go`, `MarketingHeroBanner.tsx`, `retry.ts`, `payment_simulator`) all classify correctly.
  - `internal/rubric/` — the cross-family LLM-judge: schema-constrained decoding (`responseSchema` for Gemini, `response_format` for Anthropic), six-criterion rubric (diff_correctness / test_adequacy / spec_consistency / robustness / security_posture / trust_signal_alignment), threshold 0.85 default. Hard-rejects on (`tape_miss_blocked` → re-plan, `tape_stale ≥3`, `scrubber_missing_audit`, `honest_ci_mismatch`, `tier3_fallback_missing_review`); warn-level signals on `synth_tape_used` / `live_passthrough` / `wasm_quota_trip` / `tape_aging`. Includes `HeuristicClient` for hermetic CI.
  - `internal/dispatcher/` — tier-selection state machine; fan-out across `(language, tier)`; mutex-guarded `DispatchTrace` for concurrent observability; tier-3 timeout fallback ALWAYS sets `Proof.CodeownerReviewRequired=true` (and re-asserts on the report side as defence in depth).
  - `internal/processpool/` — per-language verifier process slot manager; refuses to spawn into the executor's sandbox ID (ADR-002); semaphores per-language; the `===CRUCIBLE-TESTREPORT===` delimiter handles tool preambles; `ExecProvider` for local dev, `FakeProvider` for hermetic tests.
  - `internal/tier3/` — Dafny dispatcher (orchestrates the DafnyPro-style diff-checker + invariant-pruner + Laurel-augmenter loop over `dafny verify`); typed-error stubs for Lean 4, TLA+, Z3; partial-proof cache keyed by `(diff_hash, prover)`; **on timeout, sets `Proof.TimedOut=true`, `Proof.FallbackTier="tier_2_5"`, `Proof.CodeownerReviewRequired=true` and the report carries a structured `tier3_timeout` finding**.
  - `internal/tier4/` — hermetic-rebuild verifier; `NixBuilder` runs `nix build --print-out-paths` and sha256s the store-path content tree; `SigstoreAttestor` emits in-toto v1 / SLSA Provenance v1 statements via DSSE; `DiffoscopeDiffer` shells `diffoscope` when artefacts diverge (paired with `nix store diff-closures` for Nix-graph semantics); refuses any `AttestationChain` entry that doesn't start with `rekor:` (forged-attestation defence).
  - `internal/api/` — HTTP API exposing `POST /v1/twin/verify/bundle` and `POST /v1/twin/verify/audit`; the parse path runs the audit guard on the raw decoded `map[string]any` BEFORE binding to `VerificationRequest` so unknown fields named `reasoning` etc. are caught.
  - `cmd/crucible-verifier/main.go` — daemon binary; binds `:9080` by default; signal-handler-driven graceful shutdown; defaults to the heuristic rubric when no LLM is wired (set `GOOGLE_API_KEY` and the modelrouter adapter wires in Phase 5).
- **`verifiers/python/`** (Python) — `crucible-verify-python` CLI:
  - Tier 0 mutmut~=3.5 (paths_to_mutate pyproject overlay; parses text output since 3.x has no JSON reporter); threshold 0.85.
  - Tier 1 hypothesis~=6.152 (10 000+ iterations via `HYPOTHESIS_MAX_EXAMPLES`) + atheris==3.0.0 fuzz orchestration.
  - Tier 2 schemathesis~=4.18 (v4 rewrite syntax: `--report junit --report-junit-path`).
  - Tier 3: dispatch placeholder; real Dafny in `apps/verifier/internal/tier3/`.
  - Tier 4: Nix-derivation hash + double-pass sha256 comparison.
- **`verifiers/typescript/`** (Node 22) — `crucible-verify-typescript` CLI:
  - Tier 0 stryker-js 9.6.1 (`--mutate` glob from diff; parses `reports/mutation/mutation.json` mutation-testing-elements v3.7.x).
  - Tier 1 fast-check 4.7.0 via `@fast-check/vitest` 0.4.1 (`numRuns: 10_000`).
  - Tier 2 schemathesis sidecar.
  - Tier 3: dispatch placeholder (no mainstream TS formal verifier in v1).
  - Tier 4: double `pnpm build` + sha256 of `dist/`.
  - jsfuzz IS UNMAINTAINED — runner uses `@jazzer.js/core` 2.1.0 vendor-pinned.
- **`verifiers/rust/`** (Rust crate) — `crucible-verify-rust` binary:
  - Tier 0 `cargo-mutants` 27.0.0 (`--in-diff <tmp>.diff --json`; parses `mutants.out/outcomes.json`).
  - Tier 1 `proptest` 1.11 (`PROPTEST_CASES=10000`) + `cargo-fuzz` 0.13.1 / `cargo-afl` 0.18.2 with `-V 15`.
  - Tier 2 schemathesis sidecar.
  - Tier 3 Kani 0.67.0 (per-harness `--harness <name>`; propproof SKIPPED — use `bolero` + `bolero-kani`); on timeout sets `FallbackTier="tier_2_5"` and `CodeownerReviewRequired=true`.
  - Tier 4: double `cargo build --release` with `SOURCE_DATE_EPOCH=0`, `-trimpath`.
- **`verifiers/go/`** (Go module) — `crucible-verify-go` binary:
  - Tier 0 `avito-tech/go-mutesting` v2.3.1 (no native JSON — text-parsed; PASS/FAIL semantics are INVERTED from mutmut, documented); threshold 0.75 with realistic 0.60 target noted in findings.
  - Tier 1 `pgregory.net/rapid` v1.3.0 (`-rapid.checks=10000`) + native `testing.F` fuzz (`-fuzztime=15s`).
  - Tier 2 schemathesis sidecar.
  - Tier 3: dispatch placeholder (no Go formal verifier in v1).
  - Tier 4: double `go build -trimpath -buildvcs=false -ldflags="-buildid="` + sha256.
- **`verifiers/java/` + `verifiers/swift/`** — interface-ready stubs (`crucible-verify-{java,swift}.sh`) that emit a TestReport with `Verdict=tool_unavailable`; documented contract for the Phase 9+ adapter that lands when a design partner asks.
- **`apps/control-plane/internal/verifierbridge/`** — HTTP bridge wiring the control plane's `done` claim into the verifier daemon. ADR-002 invariant enforced at the bridge edge (refuses same-family routing pre-call); typed `NotConnectedError` when daemon unreachable; env-gated `CRUCIBLE_VERIFIER_ADDR` (defaults to `http://127.0.0.1:9080`).
- **`apps/control-plane/internal/api/verify.go`** — `POST /v1/tasks/{id}/verify` endpoint. Transitions task `executing|approved → verifying`; on approval → `promoting`; on rejection → `failed` with structured `RejectionReason[]` surfaced. Verifier-side cost is tracked on the Budget enforcer **without deducting from the executor's cap** (ADR-009 — verifier_billed_separately=true in the response).
- **`scripts/runbook.sh`** — local runbook lookup tool. `./scripts/runbook.sh RB-07` shows verifier-disagreement runbook; `./scripts/runbook.sh RB-10` shows Tier-3 timeout-rate runbook.

### Architectural decisions

- **Cross-family invariant ALWAYS enforced.** ADR-002 says executor and verifier MUST be from different vendor lineages. Phase 4 enforces this in FOUR places: the rubric `Judge.Score()`, the dispatcher `Dispatch()`, the verifier daemon's HTTP `parseRequest()`, AND the control-plane's `verifierbridge.Verify()`. Same-family routing is refused everywhere with the typed `SameFamilyError`.
- **Verifier NEVER sees the executor's reasoning.** The audit guard runs in three places: (a) `VerificationRequest.AuditNoLeakage()` at ingest; (b) `RenderPrompt()` against the assembled system+user payload; (c) the verifier daemon's HTTP `parseRequest()` on the raw decoded `map[string]any` before struct binding. Defence in depth: a reasoning field added to a future schema variant is still caught by (c) even before it reaches the typed parser.
- **Tier 3 NEVER fails open.** On wall-clock timeout (Dafny 10min / Lean 30min / TLA+ 20min) the prover adapter sets `Proof.TimedOut=true`, `Proof.FallbackTier="tier_2_5"`, `Proof.CodeownerReviewRequired=true`. The dispatcher re-checks; the rubric hard-rejects if `CodeownerReviewRequired=false` despite a timeout (the `tier3_fallback_missing_review` rejection category). Triple-redundant by design.
- **Verifier sandbox is ALWAYS distinct from the executor's.** `processpool.Submit` refuses to spawn into a sandbox whose ID matches the executor's. This is the second-half of the ADR-002 isolation guarantee (cross-family models + cross-sandbox runtime).
- **DafnyPro is paper-only — we implement the recipe.** POPL 2026's DafnyPro reports 86% on DafnyBench with Sonnet 3.5 but publishes no model weights. Phase 4 ships the diff-checker + invariant-pruner + Laurel-augmenter orchestration loop over the official `dafny verify` CLI; the `LaurelAugmenter` interface drops in via dependency injection.
- **Cross-vendor prompt caches do NOT transfer.** When the executor runs Anthropic Opus and the verifier runs Gemini, the Gemini call pays full input price on every request. We measure this as `CostBreakdown.RubricUSD` and budget for it explicitly per ADR-009.
- **Go mutation-tooling threshold realistically lands at ~0.60, not 0.85.** `avito-tech/go-mutesting` is the canonical fork (`zimmski/go-mutesting` abandoned 2022); its mutator set is materially narrower than Stryker/mutmut. Brief says 0.75; documented honest target is 0.60.
- **jsfuzz is unmaintained.** TypeScript runner uses Jazzer.js (`@jazzer.js/core` 2.1.0) vendor-pinned to the GitHub 4.0.0 tag; treat as a frozen dependency until upstream resumes.

### Stubbed (Phase 5+)

- Real Sigstore Rekor v2 publish (local hash-chained journal remains the default)
- Lean 4 + LeanCopilot Tier-3 adapter (typed-error stub; v2 Phase 9)
- TLA+ + Apalache Tier-3 adapter (typed-error stub; v2 Phase 9)
- Z3 / CVC5 direct dispatch (typed-error stub; v2 Phase 9)
- Java + Swift per-language runners (interface-ready shell stubs only)
- Antithesis SaaS wiring (flag-gated; in-house DST is the OSS-tier default)
- DafnyPro paper-only: orchestration ships; LLM-assisted assertion generator (Laurel-style) wires via the `LaurelAugmenter` interface in Phase 5+
- `crucible-verifier` cmd's `buildJudge` defaults to the heuristic; the production model-router adapter (real Anthropic / Google / OpenAI cross-family routing) wires in Phase 5
- Memory-as-verifier compliance check — needs Phase 5 memory layer
- Multi-verifier ensemble for high-stakes promotions — v2 Phase 9

### Library versions pinned (May 2026 currency check)

- `mutmut~=3.5` (Python). 4.x doesn't exist; `paths_to_mutate` lives in `pyproject.toml`; no native JSON reporter (parse `mutmut results` text).
- `hypothesis~=6.152` (Python). `settings(max_examples=10_000, deadline=None, database=None)`.
- `schemathesis~=4.18` (Python). v4 is a full rewrite — `--report junit --report-junit-path PATH` (NOT `--junit-xml`).
- `atheris==3.0.0` (Python). Only 3.x release; Python 3.11–3.13 only.
- `fast-check@4.7.0` (TS). `numRuns: 10_000`.
- `@fast-check/vitest@0.4.1`, `@fast-check/jest@2.2.0`.
- `@stryker-mutator/core@9.6.1` (TS). `--mutate "<files>" --incremental --force`; parses `reports/mutation/mutation.json` mutation-testing-elements v3.7.x.
- `@jazzer.js/core@2.1.0` (TS, vendor-pinned).
- `proptest = 1.11` (Rust). `PROPTEST_CASES=10000`.
- `cargo-mutants 27.0.0` (Rust). `--in-diff <diff>.diff --json`.
- `Kani 0.67.0` (Rust). `--harness <regex>`. SKIP propproof; use `bolero` + `bolero-kani`.
- `cargo-fuzz 0.13.1` + `cargo-afl 0.18.2`.
- `pgregory.net/rapid v1.3.0` (Go). `-rapid.checks=10000`.
- `github.com/avito-tech/go-mutesting v2.3.1` (Go). Realistic gate ≈0.60.
- `Dafny 4.11.0`. DafnyPro POPL 2026 — paper-only.
- `diffoscope 318`. Pair with `nix store diff-closures` for Nix-graph semantics.
- `Witness v0.11.0` (CLI) + `github.com/in-toto/go-witness v0.10.0`.
- `github.com/in-toto/attestation/go/v1 v1.2.0`.
- `actions/attest-build-provenance v4.1.0` (pin by SHA; consider `actions/attest@v2` directly).

### Threat-model invariants — ship-blocker check (Phase 4)

| # | Invariant | Status | Evidence |
|---|---|---|---|
| 1 | Verifier sees ONLY the diff + tests + spec changes + OpenAPI delta; NEVER the executor's reasoning trace | **ENFORCED (triple-redundant)** | `verification.AuditNoLeakage` at ingest; `rubric.RenderPrompt` post-render audit; `api.parseRequest` raw-map audit before struct binding. The `TestNoReasoningEverReachesVerifier` test exercises all three paths. |
| 2 | Executor and verifier are from different vendor lineages | **ENFORCED (quad-redundant)** | `VerificationRequest.Validate()`, `Judge.Score()`, `httpBridge.Verify()`, `api.handleVerifyTask`. Typed `SameFamilyError` everywhere. |
| 3 | Verifier sandbox is distinct from the executor's | **ENFORCED** | `processpool.Submit` refuses to spawn when `Sandbox.ID() == req.ExecutorSandboxID`. Test: `TestPool_refusesSpawnIntoExecutorSandbox`. |
| 4 | Tier 3 timeout never silently passes — fallback ALWAYS requires CODEOWNER review | **ENFORCED (triple-redundant)** | `tier3.Adapter.Discharge` sets `FallbackTier+CodeownerReviewRequired`; `dispatcher.applyTier3Fallback` re-asserts; `rubric.hardRejections` hard-rejects when `CodeownerReviewRequired=false`. |
| 5 | Critical-path classifier hits all labeled examples from `tier3-trigger-automation.md` §"Examples" | **ENFORCED** | `TestLabeledExamplesClassifyCorrectly` covers oauth_callback.py / refund_engine.go / MarketingHeroBanner.tsx / retry.ts / payment_simulator. |
| 6 | Forged attestations are rejected | **ENFORCED** | `tier4.SigstoreAttestor.Verify` refuses entries not prefixed `rekor:`; `Verifier.Verify` records `attestation_invalid` finding and hard-rejects. |
| 7 | Verifier-side cost billed SEPARATELY from executor (ADR-009) | **ENFORCED** | `api.handleVerifyTask` records `verifier_billed_separately=true`; `BudgetEnvelope.VerifierSpentUSD` is a distinct counter. |

## [2026.06.0-phase3] — 2026-05-15

Phase 3 — **Twin Runtime breadth**. Fills in the five surfaces Phase 2 deferred so the enterprise / regulated / air-gapped story is real: the production PII scrub pipeline, multi-engine database twins, raw Firecracker self-host orchestrator scaffold, WASM tool runner, and shadow-recording mode. ~21K LoC across Python + Go + Rust.

### Added

- **`services/twin-runtime/tape_driver/scrubber/`** (Python) — Presidio + spaCy + FF3-1 + deterministic pseudonymisation pipeline. Replaces the Phase 2 regex-only baseline. Ships:
  - `crucible_scrubber.pipeline.ScrubPipeline` — orchestrates Analyzer → Anonymizer → audit log with operator overrides per entity.
  - `crucible_scrubber.operators.DeterministicHashOperator` — HKDF-keyed deterministic pseudonym (replaces Presidio 2.2.362's hash operator, which now uses a random salt that breaks referential integrity).
  - `crucible_scrubber.operators.Ff3FpeOperator` — FF3-1 FPE wrapper with domain padding for sub-bound fields.
  - `crucible_scrubber.recognizers.MRNRecognizer / NPIRecognizer / DEARecognizer / VINRecognizer / TenantAccountRecognizer` — custom recognizers for HIPAA 18-identifier coverage gaps + tenant-configurable patterns.
  - `crucible_scrubber.audit.AuditLog` — every rewrite enumerated with before-hash + operator + algorithm for compliance auditors.
  - FastAPI service (`crucible-scrubber` console script) with bearer-token auth and `/scrub` + `/scrub/batch` endpoints.
  - 1100-entry adversarial corpus covering HIPAA Safe Harbor 18 identifiers + PCI + cloud keys + free-text PII; fallback recall ≥ 70%, full-pipeline recall ≥ 99%.
- **`services/twin-runtime/tape_driver/presidio_scrubber.go`** — Go HTTP client for the Python service. `Scrubber` interface unchanged; the Phase 2 [RegexScrubber] becomes the fallback when `CRUCIBLE_SCRUBBER_URL` is unset or the service is unreachable (regulated tenants opt into `WithFailClosed()`).
- **`services/twin-runtime/tape_driver/synth/`** (Go) — synthetic response generation per the Microcks Copilot pattern. Schema-driven Faker walker + optional `LLMAugmenter` for free-text fields; OpenAPI 3.x parser; in-memory state journal so subsequent reads of a synth-mutation see consistent state; CANDIDATE tape entries for the operator's promotion queue.
- **`services/twin-runtime/db_driver/planetscale.go`** — PlanetScale MySQL driver (async-create-poll, then per-call password mint; auth uses the documented colon-form `tokenID:token`, NOT Bearer; recursive delete on cleanup).
- **`services/twin-runtime/db_driver/turso.go`** — Turso libSQL driver. CoW branching via `seed.type=database`; per-branch JWT mint; sqlite_master schema-diff in lieu of a first-party compare endpoint.
- **`services/twin-runtime/db_driver/mongo.go`** — Atlas shared-cluster database-per-task driver. (Native snapshot-restore-to-new-cluster takes 15–60 min — incompatible with per-task ephemeral branching. The shared-cluster pattern fits the 5s budget with documented isolation caveats.)
- **`services/twin-runtime/db_driver/redis.go`** — In-sandbox redis-server handle. Per-task port derived deterministically from branch name.
- **`services/twin-runtime/db_driver/clickhouse.go`** — ClickHouse driver using `CREATE TABLE … CLONE AS` per table; native HTTP interface; SchemaDiff via `system.tables`.
- **`services/twin-runtime/db_driver/s3.go`** — MinIO/S3-compatible driver. Per-task bucket creation; rclone seed command rendering; SigV2 auth (SigV4 polish flagged for Phase 4).
- **`services/twin-runtime/tape_driver/shadow_recorder/`** (Go) — production / staging traffic shadow recorder. Envoy access-log + eBPF tap ingress. Scrub-at-capture (fail-closed when the Python scrubber is unreachable). Content-addressed tape store. Per-endpoint coverage stats. HIPAA Safe Harbor 18-identifier audit test asserts no identifier persists.
- **`apps/twin-runtime/crates/twin-runtime-wasm/`** (Rust) — Wasmtime-embedded WASM tool runner for LLM-generated tool code. WASI Preview 1 capability model (no inherit_stdio/env/net by default); `ResourceLimiter` for memory caps; epoch-interruption watchdog for wall-clock; 10 000-iteration containment proptest asserts zero escape attempts succeed.
- **`apps/twin-runtime/crates/twin-runtime-staleness/`** (Rust) — per-endpoint tape-age tracker. Classifier produces `Fresh|Aging|Stale|Unrecorded` bands; verifier (Phase 4) lowers confidence on stale-tape responses; PR-comment renderer.
- **`apps/twin-runtime-self-host/`** (Rust) — raw Firecracker + containerd + ZFS orchestrator scaffold for the self-hosted enterprise tier. Implements the warm-pool, per-tenant cgroup quotas (v2), Tetragon TracingPolicy renderer, and ZFS clone-per-task lifecycle. `linux-firecracker` Cargo feature gates the actual `firec` crate so the binary `cargo check`s on macOS / Windows developer hosts; production builds enable the feature.

### Architectural decisions

- **PII scrub is the regulated-buyer story.** The Phase 2 currency check flagged that Presidio 2.2.362's `hash` operator now uses a random salt; that would break referential integrity ("cus_abc123 → cus_zzz789 consistently across all entries"). Phase 3 ships `DeterministicHashOperator` (HKDF-keyed off the per-tape-set master secret + entity value) as the canonical pseudonym operator.
- **FF3-1 minimum domain is 10⁶** (NIST SP 800-38G Rev. 1 2nd Public Draft, 2025-02-03). 6-digit credit-card BIN sits exactly at the bound; 4-digit account suffixes are below it. `crucible_scrubber.ff3` validates domain size at construction time and refuses to instantiate ciphers below the floor — callers either widen the alphabet or fall back to `DeterministicHashOperator` for that field.
- **Mongo Atlas snapshot-restore is infeasible for per-task branching** (15–60min per the May 2026 currency check) and the M2/M5 shared tiers were retired 2026-01-22. Phase 3 ships the shared-cluster database-per-task variant on a Flex cluster as the v1 Mongo `DBTwin`, with documented isolation caveats.
- **PlanetScale auth uses the colon form** `Authorization: <SERVICE_TOKEN_ID>:<SERVICE_TOKEN>`. NOT Bearer. (PlanetScale's documented exception in the May 2026 API.)
- **ADR-015 warm-restore ≤10ms target re-scoped to memory resume only.** Phase 3 currency check on Linux 6.x found full userland-ready is ~25–30ms even on a hot snapshot; the orchestrator surfaces both the memory-resume and the userland-ready latencies for honest reporting.
- **WASM tool runner refuses network capabilities.** WASI Preview 2's net capability isn't shipped in Phase 3; the runner's `Capabilities::requests_net()` causes immediate `CapabilityDenied` at boot. Net access flows via host-typed component imports the runner itself controls.
- **Tetragon policies are rendered, not submitted in-process.** The host's Tetragon daemon watches a directory; the orchestrator writes the per-sandbox `TracingPolicyNamespaced` YAML there. Avoids coupling the orchestrator binary to libbpf.

### Stubbed (Phase 4+)

- **Firecracker `firec` crate invocations** — gated behind `linux-firecracker`. Production builds enable; cross-platform `cargo check` does not.
- **Wasmtime Component Model** — Phase 3 ships core-module support only; Component Model is a Phase 4 polish.
- **Tape promotion UI** — CANDIDATE entries are persisted but the operator dashboard for promote/reject is Phase 6.
- **Vault Transform engine FF3-1** — for HSM-backed enterprise tier. Phase 3 uses an env-supplied master key with HKDF salt.
- **OpenAPI 3.1 advanced schema features** (`anyOf`, `oneOf`, `allOf`, recursive `$ref`) — Phase 3 handles the dominant `$ref` + properties + items shapes; full coverage is a Phase 4 polish.

### Library versions pinned (May 2026 currency check)

- `presidio-analyzer==2.2.362`, `presidio-anonymizer==2.2.362` — exact pins (line in active churn).
- `spacy>=3.8,<3.9` — pinned below v4 prereleases that the unpinned `spacy` line can pull.
- `en_core_web_lg-3.8.0` for general NER; `StanfordAIMI/stanford-deidentifier-base` for HIPAA-tier tapes.
- `ff3==1.0.3` (mysto/python-fpe). FF3-1 only; FF3 removed in NIST SP 800-38G Rev. 1 2PD.
- `wasmtime==31`, `wasmtime-wasi==31`, with `runtime + cranelift + pooling-allocator + async + parallel-compilation + component-model + addr2line + demangle + coredump + wat` features (no `threads` — WASI Preview 2 doesn't ship threads).
- `firecracker ~1.10`; Rust embedding via `firec==0.7`.
- PlanetScale base `https://api.planetscale.com/v1`; auth `tokenID:token`.
- Turso base `https://api.turso.tech/v1`; Bearer auth.
- MongoDB Atlas Admin API v2 (programmatic-key Bearer; vendor content-type `application/vnd.atlas.2024-08-05+json`).

### Threat-model invariants — ship-blocker check (Phase 3)

| # | Invariant | Status | Evidence |
|---|---|---|---|
| 1 | Scrubbing happens at capture, not replay | **ENFORCED** | `shadow_recorder.Capture` calls the scrubber before `store.Put`; HIPAA 18-identifier audit test asserts no identifier persists. |
| 2 | Deterministic pseudonymisation preserves referential integrity per tape-set | **ENFORCED** | `DeterministicHashOperator` is HKDF-keyed; tape-set name mixed into salt; cross-tape-set isolation tested. |
| 3 | WASM tool runner refuses network + sees only host-granted capabilities | **ENFORCED** | `requests_net()` rejected at boot; `WasiCtxBuilder` does not inherit_stdio / env / net; 10 000-iteration containment proptest asserts no escape. |
| 4 | Tape staleness surfaces to agent and verifier | **ENFORCED** | `twin-runtime-staleness::Tracker::report()` returns `Fresh|Aging|Stale|Unrecorded` bands with PR-comment-renderable messages. |
| 5 | Self-host orchestrator never silently fakes a spawn | **ENFORCED** | Cold-start path returns `Error::PhaseStub` without `linux-firecracker`; warm-pool path serves real bookkeeping; integration test asserts the typed error. |

## [2026.06.0-phase2] — 2026-05-15

Phase 2 — **Twin Runtime**. The single highest-stakes block in v1; the destructive-op gate is the brand promise. ~24K LoC across Rust + Go + TS + Python + proto.

### Added

- **`libs/sandbox-spec/`** — Rust crate defining the `SandboxProvider` trait, conformance corpus, and content-addressable `SandboxSpec` with canonical-JSON SHA-256 hashing.
- **`libs/twin-spec/proto/crucible/v1/sandbox.proto`** — `SandboxSpec`, `Sandbox`, `SnapshotRef`, `EgressManifest`, `SecretBinding`, `SyscallShimPolicy`, plus the `TwinRuntimeService` gRPC surface (Spawn/Snapshot/Restore/Kill/ListSandboxes/Heartbeat/StreamEvents/HealthCheck).
- **`apps/twin-runtime/`** — Rust workspace with:
  - **`twin-runtime-shim`** — the brand promise. Three layers: (1) command-line lexer + 24-pattern corpus, (2) seccomp_unotify + BPF-LSM + Landlock (Linux-gated), (3) Tetragon TracingPolicy renderer. 50,000-iteration property test enforces zero bypasses. PocketOS adversarial test (ship-blocker) passes.
  - **`twin-runtime-sandbox`** — E2B provider with REST client (stub-mode without `CRUCIBLE_E2B_API_KEY`); raw-Firecracker typed Phase-3 stub; provider registry.
  - **`twin-runtime-fs`** — git worktree (cross-platform) + overlayfs (Linux) / copy fallback (other).
  - **`twin-runtime-egress`** — `ManifestValidator` with fail-closed wildcard / cloud-metadata / link-local rejection; Tetragon TracingPolicy renderer; mitmproxy `tls_clienthello` addon renderer.
  - **`twin-runtime-lifecycle`** — orchestrator wiring sandbox + fs + egress + shim + attest; heartbeat tracker; event bus.
  - **`twin-runtime-attest`** — Ed25519 signer, in-toto Statement v1 builders, DSSE envelopes, hash-chained local journal compatible with Phase 1's Go format.
  - **`twin-runtime-server`** — tonic gRPC binary `crucible-twin-runtime`.
- **`services/twin-runtime/db_driver/`** (Go) — Neon REST driver honouring the May 2026 async-create-and-poll behaviour; first-party `compare_schema` integration; typed Phase-3 stubs for MySQL/SQLite/Mongo.
- **`services/twin-runtime/tape_driver/`** (Go) — Hoverfly subprocess command renderer; regex-based PII scrubber (email, SSN, credit card, JWT, AWS/GitHub/Anthropic API keys, etc.); decision-tree enforcement returning `X-Crucible-Tape` headers.
- **`services/twin-runtime/secrets_sidecar/`** (Go) — Infisical Universal-Auth client; dynamic-secret leases with 5-second TTL floor (Infisical minimum); `InjectionDirective` for egress-proxy substitution; raw secret values NEVER returned to caller.
- **`libs/sdk-go/twin/`**, **`libs/sdk-rs/src/twin.rs`** — full client surface + in-memory `StubClient` for upstream unit tests.
- **`libs/sdk-ts/src/twin.ts`**, **`libs/sdk-py/crucible_sdk/twin.py`** — typed surface with `stubClient` / `StubClient` for upstream tests; gRPC wire transport lands with Phase 2.5.
- **`apps/control-plane/internal/twinbridge/`** — connector between Phase 1's approved-task state and the new Twin Runtime gRPC.
- **`docs/PHASE-2-REPORT.md`** — comprehensive handoff covering what shipped, stubs, threat-model invariants, library version surprises (Neon/E2B/Hoverfly/Infisical/Tetragon/mitmproxy currency check), and the Phase 3 prompt.

### Architectural decisions

- **Layer 2 of the syscall shim replaces `ptrace` with `seccomp-bpf + SECCOMP_RET_USER_NOTIF + BPF LSM`.** Currency-check research found ptrace adds 300–1000× syscall overhead and has TOCTOU bypass classes documented in the Outflank Dec-2025 seccomp-notify-injection writeup. Real production AI-agent sandboxes (Modal, GKE Agent Sandbox) use seccomp + BPF-LSM. Decision confirmed with the user; legacy `"ptrace"` identifier is normalised to `"seccomp-unotify"` with a deprecation warning.
- **Tetragon attaches at the host/hypervisor layer, not in-guest.** E2B guests lack `CAP_BPF`. For E2B-tier tenants the in-sandbox enforcement is mitmproxy (with `tls_clienthello` addon — `allow_hosts` alone does NOT drop) + E2B native `SandboxNetworkOpts`. Host-attached Tetragon ships with the raw-Firecracker orchestrator in Phase 3.
- **One Neon project per tenant.** Neon's project-scoped tokens are member-level; there is no branch-prefix RBAC. `Capabilities.PerTenantProjectRequired = true`.
- **pg_dump dropped as self-host fallback.** Replaced with DBLab 4.0 (Postgres.ai, Apache-2.0, ZFS) as the canonical self-host story. pg_dump remains useful only for onboarding seeds.
- **PII scrubber is regex-only in Phase 2.** The `Scrubber` interface is shape-stable for the Phase 3 Presidio + spaCy + FF3-1 swap.

### Stubbed (Phase 3+)

- Raw Firecracker self-hosted orchestrator (E2B-only in Phase 2)
- MySQL / SQLite / MongoDB DB twins
- Presidio + spaCy + FF3-1 full PII pipeline (regex-only in Phase 2)
- WASM tool runner
- Shadow-mode tape recording
- libbpf-rs LSM hook attachment (Landlock fallback is active)
- seccomp-unotify supervisor tokio loop (filter program + dispatch)
- Tetragon policy submission to `/var/run/tetragon/tetragon.sock`
- Wire-transport completion in `sdk-{go,ts,py,rs}/twin/grpcClient`
- `twinbridge::grpcBridge` real wire transport
- Sigstore Rekor v2 publisher (local hash-chained journal remains default)

### Threat-model invariants — ship-blocker check

All five non-negotiable invariants from the Phase 2 brief are enforced:

1. Agent process cannot syscall to real production credentials — Resolve callable only from egress process; raw value never in `Lease` struct.
2. Egress to non-allowlisted hosts dropped — Tetragon `Sigkill` (prod) or mitmproxy addon (dev).
3. Destructive operations route through the gate — 50K-iteration property test, zero bypasses; PocketOS test passes.
4. Every action emits an attestation — lifecycle orchestrator emits before returning.
5. Cross-tenant access impossible by namespace design — per-tenant Neon project, per-sandbox Infisical identity.

## [2026.06.0-phase1] — 2026-05-15

Initial Phase 1 skeleton. The Agent Control Plane foundation lands; the rest of the system is stubbed for Phase 2+.

### Added

- **Monorepo skeleton** with Nix-flake hermetic dev shells per language (Go, Node, Python, Rust).
- **`libs/twin-spec/`** — protobuf source-of-truth for `Plan`, `PromotionBundle`, `VerifierApproval`, `Convention`, `Budget`, `Task`, all 13 attestation predicate types, and supporting types.
- **`libs/attestation/`** — in-toto Statement v1 builder; DSSE envelope signer (local key for dev, Sigstore keyless OIDC interface stubbed); Rekor v2 publisher with local hash-chained journal as default fallback.
- **`libs/policy/`** — OPA-embedded Rego loader (v1 API path `github.com/open-policy-agent/opa/v1/rego`) ready for the Phase-5 promotion gate.
- **`apps/control-plane/`** — Go service with:
  - `task_router` classifying tasks into the 5 tiers (Haiku 4.5 LLM-driven, prompt-cached).
  - `plan_builder` calling Sonnet 4.6 to produce a `Plan` and emitting a `PlanProposal` attestation.
  - `budget_enforcer` sidecar with hard caps on cost, retry, wall-clock.
  - `model_router` with real Anthropic, Google Gen AI, and OpenAI clients (5-tier dispatch).
  - `api` connect-go server exposing gRPC + HTTP/REST on the same handlers.
- **`apps/cli/`** — `crucible task new`, `crucible plan show`, `crucible plan approve`.
- **Tests** — unit tests for every public function in twin-spec/attestation/policy/control-plane; property tests for the budget enforcer; one end-to-end integration test using a real (cheap) Haiku 4.5 call.
- **CI** — lint, type-check, mutation-tested unit tests on diff, Nix flake check, SLSA-L3 attest-build-provenance on main merges.
- **Docs** — `docs/02-engineering/local-dev.md`, `docs/PHASE-1-REPORT.md`.

### Stubbed (Phase 2+)

- Twin Runtime (sandbox driver, Neon, Hoverfly, syscall shim).
- Verifier Pipeline (tier ladder).
- Memory Layer (Redis / pgvector / FalkorDB / Graphiti).
- Promotion Contract (Argo Rollouts, KMS lease).
- Web console, IDE plugins, GitHub App, Slack bot.

### Notes

- **Rekor v2 client** is flag-gated (`CRUCIBLE_REKOR_PUBLISH=1`). Default is local hash-chained journal because Sigstore Rekor v2 has not yet GA'd as of May 2026.
- **Anthropic prompt-cache TTL** is set explicitly to `1h` for system prompts; default was silently flipped to `5m` on 2026-03-06.
- **Gemini 3 thinking** uses `thinking_level` not `thinking_budget` (per Google's Gen-AI guidance on Gemini 3+).
- **`gpt-5.1-codex-max`** kept as a routable alternate, but flagged unverified pricing in the model table; production should re-confirm before promotion to default.

### Library versions pinned

- `github.com/anthropics/anthropic-sdk-go` v1.43.x
- `google.golang.org/genai` v1.57.x  (legacy `cloud.google.com/go/vertexai/genai` is removed 2026-06-24)
- `github.com/openai/openai-go/v3` v3.35.x
- `connectrpc.com/connect` v1.x
- `github.com/sigstore/sigstore-go` v1.1.x
- `github.com/open-policy-agent/opa/v1` v1.16.x
- Nix 2.34, flakes enabled
