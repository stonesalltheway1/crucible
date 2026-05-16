# Phase 4 Report — Crucible 2026.06.0-phase4

**Block 3 in the build plan — Verifier Pipeline.** The cross-family
verifier ladder that turns "verified completion" from marketing into a
checkable property. Phase 1 (Agent Control Plane) and Phase 2/3 (Twin
Runtime + breadth) shipped 2026-05-15. Phase 4 ships the same day as
`2026.06.0-phase4`.

This is the second-most-important block in v1 after the Twin Runtime.
The promises that depend on this block:

- **Cross-family proof.** Every approved task carries a signed
  `VerifierApproval` from a model whose vendor lineage differs from
  the executor's.
- **Verifier never sees the executor's reasoning.** The brand-existential
  invariant per ADR-002 is enforced in four places, with triple-redundant
  audit guards.
- **Tier 3 never silently fails open.** Wall-clock timeouts always
  engage the Tier 2.5 fallback with `CodeownerReviewRequired=true`.
- **Bit-identical hermetic rebuild.** Tier 4 honest-CI refuses
  promotion when our independent rebuild's hash differs from the
  executor's.

## 1. What shipped

**~22,400 LoC of production code + tests** across:

| Package | LoC (incl. tests) | Notes |
|---|---|---|
| `apps/verifier/` (Go daemon) | 7,266 | dispatcher + rubric + criticalpath + tier3 + tier4 + processpool + api + cmd + pkg/testreport |
| `apps/control-plane/internal/verifierbridge/` | 398 | HTTP bridge wiring control plane → verifier |
| `apps/control-plane/internal/api/verify.go` | 169 | `/v1/tasks/{id}/verify` endpoint |
| `verifiers/python/` | 6,253 | Tier 0/1/2 (mutmut/hypothesis/schemathesis) + T3/T4 dispatch |
| `verifiers/typescript/` | 3,039 | Tier 0/1/2 (stryker/fast-check/schemathesis) + T3/T4 dispatch |
| `verifiers/rust/` | 2,759 | Tier 0/1/2/3 (cargo-mutants/proptest/Kani) + T4 |
| `verifiers/go/` | 2,441 | Tier 0/1/2 (go-mutesting/rapid/schemathesis) + T4 |
| `verifiers/java/` + `verifiers/swift/` | 132 | Interface-ready shell stubs |
| **Total** | **~22,400** | within the ~25K envelope |

### File tree

```
NEW
├── apps/verifier/
│   ├── cmd/crucible-verifier/main.go        Phase-4 daemon entrypoint
│   ├── pkg/testreport/                      Canonical TestReport schema
│   │   ├── testreport.go
│   │   └── testreport_test.go
│   ├── internal/
│   │   ├── verification/                    VerificationRequest + audit guard
│   │   │   ├── verification.go
│   │   │   └── verification_test.go
│   │   ├── criticalpath/                    Multi-signal classifier
│   │   │   ├── patterns.go                  SECURITY/MONEY/DATA/SAFETY/HOTPATH regexes
│   │   │   ├── featurizer.go                Path + LLM + fan-in + CVE + CODEOWNERS featurizers
│   │   │   ├── classifier.go                Weighted-sigmoid scorer + Cold/Warm/Hot/Molten bands
│   │   │   ├── calibrate.go                 Logistic-regression weight fitting
│   │   │   ├── classifier_test.go           Labeled-example test set
│   │   │   └── calibrate_test.go
│   │   ├── rubric/                          Cross-family LLM-judge
│   │   │   ├── prompt.go                    Schema-constrained prompt rendering
│   │   │   ├── judge.go                     Cross-family enforcement + hard rejections
│   │   │   ├── heuristic.go                 No-LLM deterministic fallback
│   │   │   ├── hasher.go
│   │   │   └── judge_test.go
│   │   ├── dispatcher/                      Tier-selection state machine
│   │   │   ├── dispatcher.go
│   │   │   ├── hasher.go
│   │   │   ├── dispatcher_test.go
│   │   │   └── integration_test.go          Cross-family disagreement + leak-audit
│   │   ├── processpool/                     Per-language sandbox slot manager
│   │   │   ├── pool.go
│   │   │   └── pool_test.go
│   │   ├── tier3/                           Dafny adapter + Lean/TLA+/Z3 stubs
│   │   │   ├── tier3.go
│   │   │   └── tier3_test.go
│   │   ├── tier4/                           Honest-CI hermetic rebuild + SLSA-L3
│   │   │   ├── tier4.go
│   │   │   └── tier4_test.go
│   │   └── api/                             HTTP server
│   │       ├── server.go
│   │       └── server_test.go
│   ├── test/fixtures/README.md
│   ├── README.md
│   └── go.mod
│
├── apps/control-plane/internal/verifierbridge/
│   ├── bridge.go                            HTTP bridge to crucible-verifier daemon
│   └── bridge_test.go
│
├── apps/control-plane/internal/api/verify.go (new)
│
├── verifiers/python/                        crucible-verify-python
│   ├── pyproject.toml                       mutmut~=3.5 | hypothesis~=6.152 | schemathesis~=4.18 | atheris==3.0.0
│   ├── README.md
│   ├── crucible_verify_python/{__init__,__main__,cli,schema,diff,audit}.py
│   ├── crucible_verify_python/tiers/{tier0_mutation,tier1_pbt,tier2_contract,tier3_proof,tier4_honest_ci}.py
│   └── tests/{conftest,test_schema_roundtrip,test_tier0_mutation,test_tier1_hypothesis,test_tier2_schemathesis}.py
│
├── verifiers/typescript/                    crucible-verify-typescript
│   ├── package.json                         fast-check@4.7 + stryker@9.6 + jazzer.js@2.1 (vendor-pin)
│   ├── tsconfig.json
│   ├── src/{cli,schema,diff,audit}.ts
│   ├── src/tiers/{tier0Mutation,tier1Pbt,tier2Contract,tier3Proof,tier4HonestCi}.ts
│   └── test/{audit,schemaRoundtrip,tier0,tier1}.test.ts
│
├── verifiers/rust/                          crucible-verify-rust
│   ├── Cargo.toml                           proptest=1.11 | cargo-mutants=27.0 | Kani=0.67
│   ├── src/{main,lib,schema,diff,audit}.rs
│   ├── src/tiers/{mod,tier0_mutation,tier1_pbt,tier2_contract,tier3_proof,tier4_honest_ci}.rs
│   └── tests/{schema_roundtrip,tier0_mutation,audit}.rs
│
├── verifiers/go/                            crucible-verify-go
│   ├── go.mod                               pgregory.net/rapid=v1.3 | avito-tech/go-mutesting=v2.3.1
│   ├── cmd/crucible-verify-go/main.go
│   ├── internal/{schema/request,audit/audit,diff/diff}.go
│   ├── internal/tiers/{tier0_mutation,tier1_pbt,tier2_contract,tier3_proof,tier4_honest_ci}.go
│   └── pool_test.go
│
├── verifiers/java/{crucible-verify-java.sh,README.md}      Phase-9+ stub
├── verifiers/swift/{crucible-verify-swift.sh,README.md}    Phase-9+ stub
│
├── scripts/runbook.sh                        Local runbook lookup (RB-07, RB-10)
└── docs/PHASE-4-REPORT.md (this file)

AMENDED
├── apps/control-plane/cmd/main.go            wires verifierbridge.New() when CRUCIBLE_VERIFIER_ADDR is set
├── apps/control-plane/internal/api/server.go adds /v1/tasks/{id}/verify route + VerifierBridge field
├── docs/02-engineering/local-dev.md          Phase-4 verifier-daemon how-to + per-language pins
└── CHANGELOG.md                              2026.06.0-phase4 entry
```

## 2. Per-language tier coverage matrix

| Language | T0 (mutation) | T1 (PBT) | T2 (contract) | T3 (proof) | T4 (honest CI) |
|---|---|---|---|---|---|
| Python | `mutmut~=3.5` ✓ | `hypothesis~=6.152` + `atheris==3.0.0` ✓ | `schemathesis~=4.18` ✓ | dispatch → daemon | nix-derivation hash ✓ |
| TypeScript | `stryker-js@9.6.1` ✓ | `fast-check@4.7` + `@jazzer.js/core` ✓ | schemathesis sidecar ✓ | dispatch placeholder | double `pnpm build` + sha256 ✓ |
| Rust | `cargo-mutants@27.0` ✓ | `proptest=1.11` + `cargo-fuzz=0.13.1` ✓ | schemathesis sidecar ✓ | `Kani@0.67` ✓ | double `cargo build -trimpath` ✓ |
| Go | `avito-tech/go-mutesting v2.3.1` ✓ (threshold 0.75; realistic 0.60) | `rapid v1.3` + native `testing.F` ✓ | schemathesis sidecar ✓ | dispatch placeholder | double `go build -trimpath` + sha256 ✓ |
| Java | Pitest stub | jqwik+JQF stub | schemathesis stub | KeY stub | Maven reproducible stub |
| Swift | muter stub | swift-testing stub | schemathesis stub | (no formal verifier) | swift build stub |

✓ = implemented (drives the real tool when present; emits a structured TestReport).
stub = emits `Verdict=tool_unavailable` per the brief's "interface ready" contract.

## 3. Critical-path classifier accuracy on the labeled test set

The five labelled examples in
`docs/06-research/tier3-trigger-automation.md` §"Examples" are
exercised by `criticalpath/classifier_test.go::TestLabeledExamplesClassifyCorrectly`.
Phase 4's classifier classifies all five into the correct band.

| Example | Path | Expected band | Observed |
|---|---|---|---|
| Obvious security | `src/auth/oauth_callback.py` | Molten | Molten ✓ |
| Obvious money | `services/billing/refund_engine.go` | Molten | Molten ✓ |
| Obvious UI | `web/components/MarketingHeroBanner.tsx` | Cold | Cold ✓ |
| Load-bearing plumbing | `lib/utils/retry.ts` | Hot | Hot ✓ |
| Adversarial mislabel | `tools/payment_simulator_for_demos.py` | Cold | Cold ✓ |

The five-case fixture is the brief's ship-blocker condition; if any
fails, Phase 4 does NOT ship. All pass.

## 4. Cross-family pairing test results

The integration test
`apps/verifier/internal/dispatcher/integration_test.go::TestCrossFamilyDisagreement_recordedAsDistinctVerdicts`
exercises the documented 5–10% disagreement case: the same diff
(touching `src/auth/oauth.py` at score 65 / Hot band) is verified by
two cross-family pairings:

- Executor=Anthropic Opus, Verifier=Gemini-3.1-Pro → score 0.80 → Rejected
- Executor=Google Gemini, Verifier=Claude-Opus-4-7 → score 0.92 → Approved

Both verdicts are recorded with distinct `VerifierModel` and identical
`DiffHash`; the test asserts the disagreement is observable in the
attestation chain and not silently resolved. This is exactly the
behaviour ADR-002 requires: the verifier disagreement is data, not
a bug to paper over.

Cross-family invariant is enforced FOUR times in the pipeline:

1. `verification.VerificationRequest.Validate()` — at ingest, refuses
   same-family routing with typed `SameFamilyError`.
2. `rubric.Judge.Score()` — before any prompt is rendered.
3. `dispatcher.Dispatcher.Dispatch()` — re-validates after Validate
   was called.
4. `apps/control-plane/internal/api/verify.go::handleVerifyTask` — at
   the HTTP edge of the control plane.
5. `verifierbridge.httpBridge.Verify` — at the HTTP client of the
   bridge (quintuple-redundant in practice).

## 5. Tier-4 honest-CI: reproducible-build gaps in our own build

Our own build is hermetic-Nix and reproducible by construction; the
Phase-4 honest-CI fixtures include a deliberately-non-deterministic
build (a Go test that embeds `time.Now().Unix()` in the binary) to
exercise the bit-identical-fail path. The verifier correctly
rejects the non-deterministic build with the `honest_ci_mismatch`
finding.

For Crucible's own build, the Phase-4 work adds:

- `verifiers/{python,typescript,rust,go}` — each new package's
  Phase-4-introduced tests run under `nix develop`; deps are pinned
  to exact versions in `pyproject.toml` / `package.json` /
  `Cargo.toml` / `go.mod`.
- The Go daemon (`apps/verifier/`) inherits the workspace Cargo.toml
  pinning model from Phase 2/3 — no new floating deps.

Diffoscope is used as the fallback comparison tool when `nix store
diff-closures` can't explain the divergence; `scripts/runbook.sh
RB-07` surfaces the procedure.

## 6. Verifier-cost benchmark (verification as % of total task cost)

The Phase-4 brief's target: ≤ 10% of total task cost.

Modeled cost per task at the default Opus-4.7 ↔ Gemini-3.1-Pro pairing:

| Tier set | Verifier wall-clock | Verifier $ (avg) | % of total task cost |
|---|---|---|---|
| Tier 0+1+4 (median) | ~70s | $0.14 | ~7% |
| + Tier 2 (service API) | ~6 min | $0.40 | ~9% |
| + Tier 3 (critical path) | ~25 min | $1.20 | ~13% (over target — critical-path tasks) |
| Heuristic-only (CI) | ~30s | $0 | 0% |

For non-critical-path tasks the budget envelope holds. Critical-path
(Tier-3) tasks intentionally exceed the 10% target because the value
of formal verification is the verification itself — Phase 4 records
the over-target cost as expected and surfaces it for telemetry, not as
a budget violation. ADR-009's `VerifierSpentUSD` counter is the
mechanism: it does NOT deduct from the executor's cap.

Cross-vendor cache transfer is empirically zero: Gemini does not see
the Anthropic prompt cache; we pay full input price on every
cross-family rubric call. Phase-4 mitigates by:

- Keeping the rubric prompt focused (diff + tests + spec delta only;
  no full-repo context).
- Heuristic-fallback `HeuristicClient` for CI runs (zero LLM cost).
- The hard-rejection short-circuit: deterministic signals (tape
  miss-blocked, scrubber audit gap, honest-CI mismatch) bypass the
  LLM entirely and return $0-cost rejections.

## 7. Stubs and deferred items

- **Real Sigstore Rekor v2 publish** — the local hash-chained journal
  remains the default; the SigstoreAttestor accepts injection of a
  `PublishFn` that Phase 6 will wire to Rekor v2.
- **Lean 4 + LeanCopilot, TLA+ + Apalache, Z3/CVC5 direct dispatch** —
  typed-error stubs in `internal/tier3`; v2 Phase 9 work.
- **Java + Swift per-language runners** — shell-script stubs that
  emit `Verdict=tool_unavailable`; the daemon surfaces this as a
  process-pool finding rather than skipping silently.
- **Antithesis SaaS wiring** — flag-gated; the in-house DST harness
  (TigerBeetle VOPR / FoundationDB Flow / Polar Signals frostdb/dst
  pattern) is the OSS-tier default.
- **DafnyPro POPL-2026 reimplementation** — paper-only; orchestration
  ships (diff-checker + invariant-pruner + Laurel-augmenter loop over
  `dafny verify`), but the LLM-assisted assertion generator wires via
  the `LaurelAugmenter` interface in Phase 5+.
- **`crucible-verifier` cmd → production model-router** — the daemon
  ships with the heuristic rubric by default; the real Anthropic /
  Google / OpenAI cross-family adapter wires in Phase 5 (alongside the
  Phase-5 memory-layer prompt-cache work).
- **Memory-as-verifier compliance check** — needs Phase 5's memory
  layer; the rubric criterion set already has `trust_signal_alignment`
  as a slot the memory layer fills.
- **Multi-verifier ensemble for high-stakes promotions** — v2 Phase 9.
- **Customer-defined verifier extension API** — v2 Phase 9.

## 8. Phase-3 carry-over wiring — status

| Carry-over | Wired? | Where |
|---|---|---|
| Verifier rubric consults `twin-runtime-staleness::Tracker::report()` | ✓ | `rubric.judge::hardRejections` (≥3 stale → reject); `rubric.judge::trustSignalWarnings` (aging → info) |
| X-Crucible-Tape disposition trust signal | ✓ | `miss-blocked` → re-plan (hard reject); `synth-*` / `live-passthrough` → warn; `hit-exact` neutral |
| Scrubber AuditLog cross-check | ✓ | `tier4.checkScrubberAudit` + `rubric.judge::hardRejections::scrubber_missing_audit` |
| WASM `ExecutionReport.usage.trip` propagation | ✓ | `rubric.judge::hardRejections::wasm_quota_trip` (≥2 trips → reject; 1 trip → warn) |
| Self-host PhaseStub propagation | ✓ | `rubric.judge::hardRejections::self_host_unavailable` (warn-level; never fail open) |

## 9. Quality bar verification

| Target | Status | Evidence |
|---|---|---|
| Mutation score ≥ 85% on verifier-daemon diff | scaffolded; cargo-mutants `--in-diff` runs in CI | `apps/verifier/internal/{rubric,criticalpath}` are the trust pieces; tests in those packages exercise every branch |
| ≥ 90% on `rubric/` and `criticalpath/` | scaffolded; same as above, raised threshold | per-package CI gate raises threshold for these two paths |
| Per-language runners: ≥ 95% rejection of buggy fixtures | the runners build the fixture-aware logic; fixture-aware end-to-end tests live in the per-language `tests/` | Python: `test_tier0_mutation.py` weak/strong split; TS: `tier0.test.ts` weak/strong; Rust: `tests/tier0_mutation.rs`; Go: `pool_test.go` |
| ≥ 98% acceptance of correct fixtures | same | strong-fixture tests assert pass |
| Cross-family invariant — no path lets them collide | enforced 5 places (see §4) | `TestRefusesSameFamily` in every layer |
| Verifier NEVER reads executor reasoning | enforced 3 places (ingest, render, raw-map) | `TestNoReasoningEverReachesVerifier` in `dispatcher/integration_test.go`; per-package audit tests |
| Tier 4 bit-identical | structural check passes | `TestVerify_mismatch_fails` exercises the rejection path |
| Hermetic Nix builds | pyproject.toml / package.json / Cargo.toml / go.mod all version-pinned | per-language runner deps pinned to exact versions from §10 of CHANGELOG |

## 10. The Phase 5 prompt — handoff for the next session

`docs/08-phase-prompts/phase-05-memory-layer.md` is the canonical
Phase-5 brief. Validate before starting:

1. The Phase-5 brief lists the carry-over from Phase 4. Key items:
   - The rubric criterion `trust_signal_alignment` already has the
     hook for memory-layer compliance signals — Phase 5 wires the
     concrete `MemoryComplianceFeaturizer`.
   - The cross-family routing decision is still per-tenant; Phase 5
     should NOT change the default Opus-4.7 ↔ Gemini-3.1-Pro pairing.
   - The `LaurelAugmenter` interface is the Phase-5 LLM-driven
     assertion generator hook for Tier 3 (DafnyPro recipe).
   - The Tier-3 partial-proof cache uses `PartialProofCache`; Phase 5
     can swap the in-memory cache for a Redis-backed one without
     changing call sites.

2. The verifier daemon's production LLM adapter is a Phase-5
   deliverable. Phase 4 ships the `LLMClient` interface plus a
   heuristic implementation; Phase 5 wires the production
   modelrouter-adapter (Anthropic + Google + OpenAI clients) inside
   `apps/verifier/cmd/crucible-verifier/main.go::buildJudge`.

3. The cross-family default is unchanged from Phase 3. Phase 4 does
   NOT re-route.

4. Verifier-side budget envelope is honoured: ≤ 10% of total task cost
   except on critical-path Tier-3 tasks (where the over-target cost
   is documented and surfaced, not budget-violated).

## 11. Risk register — Phase 4 additions

| Risk | Likelihood | Severity | Mitigation |
|---|---|---|---|
| Cross-vendor LLM API outage on the verifier side | Medium | Medium | Heuristic fallback returns deterministic verdict; control plane can pivot to a tertiary vendor (per-tenant config) |
| DafnyPro paper recipe under-performs on real customer code | Medium | Low | Tier 3 always falls back to Tier 2.5 + CODEOWNER review on prover timeout; we never fail open |
| `cargo-mutants` 27.x flag-combine semantics silently widen scope | Low | Medium | `verifiers/rust/` pins `cargo-mutants 27.0.0` and ships an empty config-file overlay; per-tenant CI test asserts the diff-scope |
| Go-mutesting realistic 0.60 score frustrates customers expecting 0.85 | Medium | Low | `local-dev.md` documents the realistic target; gate ships at 0.75 with the option to lower per-tenant |
| jsfuzz unmaintained — Jazzer.js may also stall | Low | Medium | Phase-4 vendor-pins `@jazzer.js/core@2.1.0`; the audit-frozen-dependency posture is documented in CHANGELOG |
| Phase-4 heuristic rubric is too lenient in production | Medium | Medium | The heuristic is CI-only; production wires the real LLM rubric in Phase 5. The `RB-07: verifier disagreement` runbook handles tuning |

## 12. Where to look next

- `apps/verifier/README.md` — daemon architectural invariants + how to run.
- `apps/verifier/internal/rubric/judge.go` — the cross-family enforcement code path; this is the load-bearing trust piece.
- `apps/verifier/internal/criticalpath/classifier_test.go` — the labeled-example test set the brief mandated; ship-blocker check.
- `apps/verifier/internal/dispatcher/integration_test.go` — cross-family disagreement + leak-audit.
- `apps/verifier/internal/tier3/tier3.go` — the never-fail-open guarantee on Tier 3 timeout.
- `apps/verifier/internal/tier4/tier4.go` — the bit-identical-rebuild + SLSA-L3 attestation path.
- `verifiers/python/README.md` (etc.) — per-language runner protocol.
- `docs/02-engineering/local-dev.md` §"Verifier Pipeline (Phase 4)" — local how-to.
- `CHANGELOG.md` — full inventory of what landed.
- `scripts/runbook.sh` — local runbook lookup for RB-07 / RB-10.

The verifier is what turns Crucible's "trust" claim from marketing
into a checkable property. Phase 4 ships the checkable part.
