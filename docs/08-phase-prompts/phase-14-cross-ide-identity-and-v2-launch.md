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
