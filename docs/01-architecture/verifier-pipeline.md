# Verifier Pipeline

The verifier is a separate process that runs a **different model family from the executor** and validates the agent's claimed completion before any promotion to real systems is allowed.

The architectural anti-hallucination move: **the model that wrote the code never grades itself.**

## Why cross-family

LLM error modes correlate within model lineage. Two Claude calls disagree on different inputs than a Claude call and a Gemini call. Real adversarial verification requires the verifier to be a different tokenizer, different training data, different RL recipe, ideally a different post-training pipeline.

Strong pairings (validated by published benchmark divergence ~5–10% on SWE-Bench-style tasks):

- Executor `claude-opus-4-7` ↔ Verifier `gemini-3.1-pro` (high thinking)
- Executor `gpt-5.5` ↔ Verifier `claude-opus-4-7`
- Local/privacy: Executor `Llama-4-Maverick` ↔ Verifier `DeepSeek-V4-Pro`

Verifier model is configured per-tenant (BYOK and self-hosted can pick their own pairing).

## The four-tier ladder

Verification escalates by criticality. Each tier is *additive*: Tier 1 verification implies Tier 0 also ran; Tier 3 implies Tier 0, 1, 2 ran. Wall-clock budget is the constraint.

### Tier 0: Mutation-tested unit (default for ALL changes)

The verifier runs mutation testing on the diff. If the agent's tests (existing or newly written) cannot kill mutants on the changed code, the test suite is treated as too weak to certify completion.

| Lang | Lib | Threshold (diff-scoped) |
|---|---|---|
| Python | `mutmut` 4.x | 85% mutants killed |
| JS/TS | `stryker-js` | 85% |
| Rust | `cargo-mutants` | 85% |
| Java/Kotlin | Pitest | 85% |
| Go | `go-mutesting` + `testing.F` fuzz | 75% (Go mutation tooling weaker) |
| Swift | `swift-testing` + `muter` | 80% |

Diff-scoped — only mutate touched lines plus direct call sites, not the whole repo. Otherwise wall-clock explodes on large changes.

**Wall-clock budget:** 30s default, 2 min max.

**Fallback if mutation tool unavailable for the language:** mandatory line + branch coverage on the diff at ≥90%, plus an LLM-judge pass that rates the test suite's adversarial robustness.

### Tier 1: Property tests + fuzz (default for non-trivial feature work)

Verifier requires authored property tests covering the changed function's invariants. Runs them at ≥10,000 iterations (CI default is typically 100).

| Lang | PBT | Fuzz |
|---|---|---|
| Python | `hypothesis` 6.152+ + `schemathesis` for APIs | `atheris` |
| JS/TS | `fast-check` + `@fast-check/vitest` | `jsfuzz` |
| Rust | `proptest` + `quickcheck` | `cargo fuzz` + `cargo-afl` |
| Go | `rapid` (auto-shrinking, doubles as fuzz target) | native `testing.F` |
| Java/Kotlin | `jqwik` + JQF | JQF |
| C/C++ | `theft` | `libFuzzer` + AFL++ |
| Swift | `swift-testing` + Sourcery | `swift-testing` fuzz |

**Critical pairing rule:** the verifier requires *both* example-based and property-based tests. LLM-authored PBT alone catches 68.75% of HumanEval bugs (arXiv 2510.25297); combined with EBT it catches 81.25%. Crucible enforces the combination.

**Wall-clock budget:** 5 min default, 15 min max.

### Tier 2: Schemathesis contract + DST (default for service/API code, multi-component state)

#### Contract testing

For API/service changes, verifier runs `schemathesis` workflows derived directly from the OpenAPI/GraphQL spec. The agent is required to keep the spec in sync — diffs that break spec without updating it fail Tier 2 immediately.

#### Deterministic Simulation Testing

For concurrency-sensitive code (multiple goroutines, async actors, distributed transactions), the verifier runs DST.

**Enterprise tier:** Antithesis SaaS. ~$20K–$100K/yr enterprise contracts. Runs the system in a hypervisor-level deterministic simulator, branches timeline on failures, perfectly reproduces.

**OSS / self-hosted tier:** in-house DST harness patterned on TigerBeetle's VOPR / FoundationDB's Flow simulator. Single-threaded, virtualized clock+disk+net. Feasible because Crucible's primary stack is Postgres+Go/TS, which fits the deterministic-simulation pattern well.

**Wall-clock budget:** 15 min default, 45 min max. DST iteration count scales to fill the budget.

### Tier 3: Formal verification (only for `@critical` paths)

Auto-triggered when the multi-signal classifier in [06-research/tier3-trigger-automation.md](../06-research/tier3-trigger-automation.md) scores the touched file ≥ 80 (Molten band). Otherwise the developer must annotate explicitly: `@crucible:critical(spec="auth", level="dafny")`.

| Tool | Sweet spot | Notes |
|---|---|---|
| **Dafny + DafnyPro** | General business logic, auth state machines, money paths | POPL 2026: 86% on DafnyBench with Sonnet 3.5; pair with Laurel for auto-assertions. Most pragmatic Tier-3 target. |
| **Lean 4 + mathlib + LeanCopilot** | Crypto primitives, numerical kernels, math-heavy code | Premise retrieval over mathlib's 100k+ lemmas; FormL4 dataset; TheoremLlama |
| **TLA+ + Apalache** | Distributed invariants, consensus, replication | Human writes spec, LLM proposes inductive invariants, Apalache validates |
| **Kani** | Rust `unsafe` blocks, FFI boundaries | Pairs with proptest via propproof |
| **Z3 / CVC5** | Inline SMT queries, quantifier instantiation | LLM-guided hints (+80% Z3, +183% CVC5 solve rate per arXiv 2601.04675) |

**Wall-clock budget:** Dafny 10 min, Lean 30 min, TLA+ model-check 20 min.

**Timeout policy:** do **not** fail open. Degrade to "Tier 2.5":
- Exhaustive PBT (≥10,000 cases)
- Mutation testing on the diff
- Mandatory CODEOWNER human review

Cache partial proofs — incremental verification on the next PR resumes where it left off. Chronic Tier-3 timeouts surface to the team dashboard so proof-engineering effort can be prioritized.

### Tier 4: Honest CI (reproducible build + SLSA-L3 attestation)

The verifier refuses to mark `done` unless:

1. **Hermetic rebuild.** Nix or Bazel rebuilds the artifact independently. Verifier compares hashes — bit-identical or task fails.
2. **In-toto attestation.** Build provenance materialized as an in-toto attestation, signed by Sigstore keyless OIDC. OIDC subject must match the agent worker's identity.
3. **SLSA Level 3.** GitHub's `actions/attest-build-provenance` gets to SLSA-L2 trivially; hardened runners + dual-build platform get to L3. Witness for non-GitHub pipelines. Tekton Chains for K8s.
4. **Rego policy admission.** The build's provenance bundle must satisfy the customer's Rego policy (which includes things like "artifact must be signed by an identity in the allowed set", "Dockerfile must be reproducible-build-marked", etc.).

**Wall-clock budget:** rebuild varies hugely by project; 5 min default, 30 min max. Cached intermediate artifacts (Nix store, Bazel disk cache) reduce typical rebuild to <1 min after first run.

## Verifier process architecture

Runs as a separate sandbox, separate model, separate egress policy:

```
┌──────────────────────────────────────────────────────┐
│  Executor Sandbox                                    │
│  ├ Agent (e.g. Opus 4.7)                             │
│  ├ Tools, FS, DB, services                           │
│  └ Emits: PromotionBundle candidate                  │
└────────────────────────┬─────────────────────────────┘
                         │
                         ▼
┌──────────────────────────────────────────────────────┐
│  Verifier Sandbox  (FRESH, DIFFERENT MODEL FAMILY)   │
│  ├ Verifier (e.g. Gemini 3.1 Pro)                    │
│  ├ Read-only access to executor's diff + tests       │
│  ├ Own twin DB (independent Neon branch)             │
│  ├ Own egress policy                                 │
│  ├ Runs Tier 0/1/2/3 + Tier 4 pipeline               │
│  └ Emits: VerifierApproval | VerifierRejection       │
└──────────────────────────────────────────────────────┘
```

Verifier has **no access to the executor's reasoning trace** — it sees only the diff, the tests, the spec changes, and the OpenAPI delta. This prevents the verifier from inheriting the executor's hallucinations through the reasoning channel.

## Verifier output: `VerifierApproval`

```typescript
{
  task_id: string,
  diff_hash: string,
  verdict: "approved" | "rejected",
  tier_results: {
    tier_0?: { passed: boolean, mutation_score: number, killed: number, survived: number },
    tier_1?: { passed: boolean, pbt_iterations: number, counterexamples: any[] },
    tier_2?: { passed: boolean, contract_violations: any[], dst_replay_id?: string },
    tier_3?: { passed: boolean, prover: string, proof_artifact: string, timed_out: boolean },
    tier_4?: { passed: boolean, rebuild_hash: string, rekor_uuid: string }
  },
  rubric_score: number,    // 0..1, only on approval
  rejection_reasons: string[],
  attestations: RekorUUID[],
  signed_by_oidc: string,
  signed_at: timestamp,
}
```

Rejection reasons are structured (e.g., `"tier_1.pbt_counterexample: input [1,2,3] → output []; expected non-empty"`) so the executor can reflect and retry. Up to 3 retry rounds per the Bounded Budget Enforcer; after that, halt and ask the human.

## Performance & cost

- **Median task:** Tier 0 + Tier 1 only. Verification adds 30–60s wall-clock and ~$0.14 cost (Gemini 3.1 Pro at $2/$12, ~40K input + ~5K output, no cross-vendor cache so full input price).
- **Service/API task:** + Tier 2. Adds 5–15 min wall-clock, ~$0.40.
- **Critical-path task:** + Tier 3. Adds 10–30 min wall-clock; cost varies by prover (~$0.50–$2.00 for the LLM-driven proof search).
- **Every task:** + Tier 4. ~$0.05 for the attestation publish; rebuild time is project-dependent.

**Cost engineering:** verifier runs **once at the end** of the task, not in-loop. This is why "2× tokens" is actually closer to 1.08× in practice — verification is a small fraction of total task cost. See [00-vision/pricing-and-business.md](../00-vision/pricing-and-business.md) and [06-research/unit-economics.md](../06-research/unit-economics.md).

## What the verifier cannot catch

Honest limits:

- **Spec drift the verifier shares with the executor.** If both models hallucinate the same incorrect Stripe API, neither catches it. Mitigation: Tier 2 schemathesis pulls from the *actual published spec*, not an LLM-derived one.
- **Tape-staleness bugs.** A verified-twin success can fail in real prod if the service changed since the tape was recorded. Mitigation: tape-age metrics; promotion canary catches it; auto-rollback.
- **Semantic correctness without testable invariants.** "Make this UI look good" has no verifier signal. Mitigation: design-token-based UI generation + visual regression; out of scope for v1 verifier.
- **Performance regressions invisible to the test suite.** Mitigation: tier-2 includes a perf-regression check via benchmark replay; tier-3 hot-path classification triggers explicit perf invariants.

These limits are documented in customer-facing materials — calibrated trust beats overclaimed trust.
