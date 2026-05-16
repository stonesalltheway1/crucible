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
