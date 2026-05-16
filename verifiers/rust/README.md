# crucible-verify-rust

Crucible Phase-4 per-language verifier runner for Rust.

This crate produces a single binary (`crucible-verify-rust`) that is spawned
by the verifier daemon (`apps/verifier/`) inside an isolated sandbox. The
runner reads a `VerificationRequest` JSON document from `stdin`, runs the
tier-specific Rust tooling, and writes a `TestReport` JSON document to
`stdout` (delimited by `===CRUCIBLE-TESTREPORT===\n`). All logs go to
`stderr`. The schema is the one defined in
`apps/verifier/pkg/testreport/testreport.go`; see `src/schema.rs` for the
serde mirror.

## Pinned tooling (May 2026 research)

| Tier | Tool | Pin |
|------|------|-----|
| 0 Mutation | `cargo-mutants` | `27.0.0` (`--in-diff` + `--json`) |
| 1 PBT | `proptest` | `1.11` (`PROPTEST_CASES=10000`) |
| 1 Fuzz | `cargo-fuzz` / `cargo-afl` | `0.13.1` / `0.18.2` |
| 2 Contract | `schemathesis` | shell-out to the Python CLI |
| 3 Proof | `cargo kani` (Kani) | `0.67.0` (`--harness` regex) |
| 3 Bounded | `bolero` + `bolero-kani` | use this; **skip propproof** |
| 4 Honest CI | `cargo build` (twice, hashed) | `SOURCE_DATE_EPOCH=0`, `-Z trim-paths` when available |

## Invocation

```
crucible-verify-rust --tier=tier_0_mutation < request.json > report.txt
```

`--tier` accepts `tier_0_mutation`, `tier_1_pbt`, `tier_2_contract`,
`tier_3_proof`, `tier_4_honest_ci`. The runner is intentionally
single-shot per tier — the dispatcher in `apps/verifier/internal/dispatcher/`
fans out one process per (language, tier) pair.

## Safety

The audit guard in `src/audit.rs` rejects any request whose JSON-key
namespace contains an executor-reasoning denylist hit. The runner refuses
to proceed in that case, exiting non-zero (the dispatcher surfaces the
error). This mirrors `apps/verifier/internal/verification/verification.go`.
