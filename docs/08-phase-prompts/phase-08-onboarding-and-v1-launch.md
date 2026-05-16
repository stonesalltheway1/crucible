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
