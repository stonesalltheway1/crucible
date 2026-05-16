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
