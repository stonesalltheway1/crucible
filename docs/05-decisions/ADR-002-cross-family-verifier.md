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
