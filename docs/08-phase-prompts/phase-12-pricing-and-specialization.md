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
