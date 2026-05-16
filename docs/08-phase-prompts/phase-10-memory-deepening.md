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
