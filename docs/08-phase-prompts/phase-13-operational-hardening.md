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
