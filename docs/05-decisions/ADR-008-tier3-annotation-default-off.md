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
