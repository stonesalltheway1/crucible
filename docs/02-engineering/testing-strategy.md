# Testing Strategy

How Crucible tests Crucible. We use our own verifier ladder to grade our own changes — eating our dogfood is non-negotiable, and the test surface reflects that.

## The five-tier internal test pyramid

Per-component CI runs each tier in order; failures at any tier block merge.

### Tier 0: Unit tests, mutation-tested

Every public function has unit tests. Every PR's changed lines are mutation-tested on diff (not the whole repo).

- **Threshold:** ≥85% mutation score on diff for Go/Rust/TS/Python; ≥75% for Go (mutation tooling weaker).
- **Frameworks:** `mutmut` (Py), `stryker-js` (TS), `cargo-mutants` (Rust), `go-mutesting` (Go), Pitest (Java).
- **Budget:** 30s default, 2 min max. Diff-scoped, parallel.
- **Failure mode:** PR comment with the surviving mutants. The author writes more tests or explains why a mutant is acceptable.

### Tier 1: Property tests + fuzz harness

Non-trivial functions get property tests covering invariants. Fuzz harnesses are required for any function that parses external input.

- **Frameworks:** `hypothesis` (Py), `fast-check` (TS), `proptest` + `cargo fuzz` (Rust), `rapid` + native fuzz (Go).
- **Iteration count:** ≥10,000 cases for property tests on PR. Continuous fuzzing in nightly with corpora retained.
- **Pairing rule:** every property test ships alongside example-based tests. Property tests alone catch ~68% of bugs; combined with examples 81%.
- **Budget:** 5 min PR, 30 min nightly.

### Tier 2: Integration + DST

Cross-service tests against ephemeral infrastructure: Neon branch, fresh Redis, fresh FalkorDB, simulated Hoverfly tapes.

- **Per-PR:** integration tests for the components changed and their direct dependents.
- **Nightly:** full integration suite — every cross-service call exercised.
- **DST:** the twin-runtime and promotion-gate are run inside a deterministic-simulation harness (in-house, TigerBeetle-style) with virtualized clock+disk+net, simulating partitions, restarts, message drops.
- **Antithesis** (when budget allows): full-system DST for the cross-service flows.

### Tier 3: Formal verification for `@critical` paths in our own code

Our own auth, secrets, attestation-signing, KMS-leasing, and policy-evaluation code is annotated `@critical` and verified.

- **promotion-gate Rego evaluation:** formally specified in Dafny; every Rego policy admission has a corresponding Dafny proof obligation.
- **attestation chain validation:** TLA+ spec for the OIDC subject-chaining invariants; Apalache model-checked.
- **KMS credential leasing:** Dafny proof that no lease can be reused.
- **Egress allowlist:** Coq spec (the one Tier-3 tool we use for this specific component; small footprint, well-validated).

When proofs time out: Tier 2.5 fallback — exhaustive PBT + mutation + mandatory CODEOWNER human review.

### Tier 4: Hermetic rebuild verification

Every release artifact (binaries, container images, Helm charts, air-gap bundle) is rebuilt independently from the same source by a second CI runner, and bit-identical hashes are required.

- **Build system:** Nix flakes (default), Bazel (alternative).
- **Provenance:** in-toto attestation signed via Sigstore keyless OIDC; published to Rekor.
- **SLSA level:** Level 3 (hardened GitHub-hosted runners + dual-platform rebuild).
- **Customer-visible:** every customer can verify our releases against the published attestations.

## The Crucible Test Harness (CTH)

A curated suite of test repositories used to validate the agent's behavior end-to-end.

### CTH composition

```
cth/
├── greenfield/             # Brand new repos; agent builds from scratch
│   ├── nextjs-todo/
│   ├── go-grpc-service/
│   ├── django-blog/
│   └── rust-cli/
│
├── feature-add/            # Existing repos; agent adds a feature
│   ├── stripe-webhook-handler/
│   ├── auth-rate-limit/
│   ├── postgres-migration-additive/
│   └── react-form-validation/
│
├── refactor/               # Existing repos; agent refactors
│   ├── extract-service-from-monolith/
│   ├── upgrade-react-17-to-19/
│   ├── replace-moment-with-date-fns/
│   └── consolidate-error-handling/
│
├── critical-path/          # Agent must trigger Tier 3
│   ├── auth-oauth-implementation/
│   ├── billing-refund-engine/
│   ├── distributed-consensus-bug-fix/
│   └── crypto-key-rotation/
│
├── adversarial/            # Designed to trick the agent
│   ├── tape-poisoned-stripe/        # Tape has malicious response
│   ├── prompt-injected-pr-comment/  # Memory poisoning attempt
│   ├── destructive-shell-disguised/ # rm -rf hidden in benign script
│   ├── hallucinated-api-trap/       # Tests pass only with fake API
│   └── sandbox-escape-attempt/      # Red-team sandbox probe
│
└── regression/             # Bugs we've fixed; must stay fixed
    ├── opus-46-loop-bug/
    ├── pocketos-style-wipe-attempt/
    ├── verifier-tier3-timeout-recovery/
    └── memory-cross-tenant-leak-check/
```

### CTH grading

For each test case, the harness records:

- **Correctness:** did the agent produce a verified-passing PR?
- **Cost:** total token spend.
- **Wall-clock:** total task duration.
- **Cache hit rate:** % of input tokens served from cache.
- **Verifier strictness:** did verifier catch a bad change that should be caught?
- **Safety:** did any destructive-op gate fire? Did the agent attempt anything that should be flagged?

Aggregate scores published per-release. Regression in any dimension blocks release.

### Adversarial subset

The adversarial cases are the most important. They're our equivalent of red-team continuous evaluation. Every fix to a real incident or red-team finding becomes a new adversarial case.

## Continuous evaluation against the public benchmarks

We run against the public benchmarks weekly and publish results:

- **SWE-Bench Verified** (`princeton-nlp/SWE-bench`)
- **SWE-Bench Pro** (Scale's harder set)
- **Terminal-Bench 2.0**
- **Aider Polyglot benchmark**
- **LiveCodeBench**
- **BigCodeBench**

These are not our primary KPI (the CTH is), but they let us position credibly against incumbents and detect regressions in the upstream models we route to.

## Property tests for our own SDK contracts

Every typed SDK call has property tests on its contract:

```
property "twin.fs.write always emits a WriteAttestation":
  forall (path, content) where path is valid:
    result = twin.fs.write(path, content)
    assert result is WriteAttestation
    assert result.signed_by_oidc is valid
    assert result.path == path
    assert hash(twin.fs.read(path)) == hash(content)
```

These tests run against the real twin-runtime in CI, not a mock. They catch contract drift that unit tests miss.

## Chaos / fault injection

The twin-runtime is the highest-stakes component. We chaos-test it weekly:

- **Network partition during task:** kill egress proxy mid-step; verify the agent receives a clean error.
- **Sandbox OOM mid-task:** force-OOM the sandbox; verify graceful failure + clean state.
- **Neon branch creation flake:** simulate 10s+ timeout; verify fallback to lite-twin works.
- **Hoverfly tape corruption:** flip random bytes; verify mount-time checksum rejects.
- **Sigstore Rekor unreachable:** verify local journaling continues and back-fills when Rekor returns.
- **KMS slow / unavailable:** verify promotion queues retry-with-backoff cleanly.

## Self-verification

We use Crucible to verify Crucible's own PRs. Every PR to the Crucible monorepo runs through:

1. Our control-plane spawns a twin from our own repo.
2. Our verifier (with cross-family pairing) runs Tier 0–4 on the diff.
3. Our promotion gate evaluates a Rego policy specifically for our internal release process.
4. The PR is merged only if all of the above pass + human approval.

This is the dogfooding test. If our own engineers find Crucible too slow / annoying / wrong to use on our own code, we ship that pain to customers. Don't.

## Customer-side test harnesses

For paying customers, we install a "shadow Crucible" mode where the agent runs against their PRs in shadow (no merge, no promotion), and we compare the agent's verifier verdict against the human reviewer's verdict. Disagreements are the most valuable signal we have for improving the verifier.

Opt-in. Anonymized. The agreement-rate is published in our public eval as a fairness signal ("Crucible agrees with human reviewers ≥X% of the time on N customer repos").

## What we explicitly do NOT test

- **The frontier LLM vendors' models.** They publish their own benchmarks; we treat them as black boxes.
- **Customer-side integrations** beyond the Crucible boundary. We test the contract; the customer tests their use of it.
- **Performance microbenchmarks of every helper function.** We optimize the hot path (twin spawn, memory router, cost meter) and tolerate the rest.
- **UI pixel-level regressions.** Tremor + shadcn are stable; we don't snapshot-test every component.

## Release cadence

- **Weekly releases** of the SaaS control plane.
- **Monthly releases** of the SDK, CLI, IDE plugins.
- **Quarterly releases** of the self-hosted helm chart and air-gap bundle.
- **Continuous releases** of the OSS verifier and tape-scrubber.

Every release has a CHANGELOG entry, Tier 4 attestation, and a public verification command (`crucible verify-release 2026.06.0`).
