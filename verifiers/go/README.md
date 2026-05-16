# crucible-verify-go

Phase-4 per-language verifier runner for Go. Spawned by the verifier
daemon (`apps/verifier`) in an isolated sandbox; reads
`VerificationRequest` JSON from stdin, writes a `TestReport` JSON
object (prefixed by the `===CRUCIBLE-TESTREPORT===` wire delimiter) to
stdout. All logs go to stderr.

## CLI

```bash
crucible-verify-go --tier=<tier> [--work-dir=<dir>] [--timeout=<duration>] < request.json
```

Supported tiers:

| Flag value          | Backend                                | Wall-clock default |
|---------------------|----------------------------------------|---------------------|
| `tier_0_mutation`   | `avito-tech/go-mutesting` v2.3.1       | 2 min               |
| `tier_1_pbt`        | `pgregory.net/rapid` v1.3.0 + `testing.F` fuzz | 15 min      |
| `tier_2_contract`   | `schemathesis` shell-out               | 45 min              |
| `tier_3_proof`      | (n/a) emits `tool_unavailable`         | immediate           |
| `tier_4_honest_ci`  | double `go build` + SHA-256 compare    | 30 min              |

The runner exits 0 unless it itself crashed (stdin unreadable, JSON
parse error, panic). Substantive failures of the tier under test are
surfaced via `TestReport.Verdict=failed` and the dispatcher reads that
from stdout — the process still exits 0.

## Tier 0 — mutation testing

Shells out to `go-mutesting` with the diff-filtered list of `.go`
source files. The MutationStats schema requires `diff_scoped=true`
(Crucible mandate); we set this explicitly because the source list is
already filtered by `internal/diff`.

### Inverted "PASS"/"FAIL" semantics

go-mutesting reports each surviving mutant as `PASS` and each
killed mutant as `FAIL` — the **inverse** of mutmut (Python) and
stryker (JS). The runner re-inverts so the canonical TestReport's
fields stay consistent:

- `MutationStats.Killed`   = lines beginning `FAIL ` (test suite
  caught the mutation)
- `MutationStats.Survived` = lines beginning `PASS ` (test suite
  missed it)
- `MutationStats.Score`    = killed / (killed + survived)

This semantic flip is the single highest-risk parser detail in this
runner — `TestMutationParserHandlesInvertedSemantics` in `pool_test.go`
pins it.

### Threshold

The brief asks for **0.75** as the gate; we hold the line at that
value. The realistic May-2026 achievable score with go-mutesting's
narrow mutator catalogue is closer to **0.60** — we surface the
realistic target in the `mutation_score_below_threshold` finding so
the rubric LLM-judge can soften its rejection language when the score
lands in `[0.60, 0.75)`. Crucible's verifier ladder over-indexes on
Tiers 1 and 3 anyway; Tier 0 is a smoke check, not the final word.

## Tier 1 — PBT + native fuzz

Two phases, sequential:

1. **rapid PBT driver.** `go test ./<diff-pkgs>... -run=^Property -rapid.checks=10000 -count=1 -v`. We
   discover diff-scoped packages from the FileSet partition. The
   `-rapid.checks=10000` flag enforces the Crucible mandate
   (`IterationsMin=10_000`); user requests below that are clamped up.
   Properties are identified by the `func Property<Name>(t *testing.T)`
   naming convention.

2. **Native fuzz driver.** For each `func Fuzz<X>(f *testing.F)`
   declared in a diff-touched `*_test.go`, `go test -fuzz=^Fuzz<X>$
   -fuzztime=15s ./pkg`. Failing inputs persisted to
   `testdata/fuzz/Fuzz<X>/<id>` are read back and surfaced as
   `Counterexamples`.

Both phases must pass for the tier to pass. Either phase can fail
independently — findings are tagged `property_failed` vs
`fuzz_crash` so the rubric can disambiguate.

## Tier 2 — Schemathesis contract testing

Skipped (Verdict=skipped, Passed=true) when the diff carries no
`SpecChange`s. When OpenAPI/GraphQL spec changes are present, shells
out to `schemathesis run --checks all --output=json <spec>` and folds
the per-endpoint results into `ContractStats.Violations`. Pip-
installed; the sandbox image is responsible for having it on PATH.

## Tier 3 — Formal verification

Always emits `Verdict=tool_unavailable`. There is no Go-native formal
verifier in Crucible v1; the daemon's Tier 3 dispatcher routes to the
Dafny / Lean / TLA+ adapters in `apps/verifier/internal/tier3`
against the spec artefacts in the diff. The runner records the
dispatch attempt for the per-tenant calibration dataset.

## Tier 4 — Honest CI

Invokes `go build` twice with:

- `-trimpath`
- `-buildvcs=false`
- `-ldflags="-buildid= -s -w"`
- `SOURCE_DATE_EPOCH=0`
- `CGO_ENABLED=0`

then SHA-256s each binary and sets `HonestCIStats.BitIdentical =
hashA == hashB`. The daemon-side Tier 4 driver folds this into the
larger Sigstore / Rekor / Witness / Tekton Chains attestation chain;
the per-language runner only contributes the rebuild-hash pair.

Drift between the two hashes almost always means: an embedded
timestamp (look for `//go:embed` with a generated file), a host-
specific cgo path, or a non-deterministic codegen step (`go
generate` writing wall-clock data). The runner emits a
`non_reproducible_build` finding with a `SuggestedFix` pointing at the
common causes.

## Audit guard (ADR-002)

`internal/audit` enforces the executor-reasoning leak guard at the
runner ingress boundary. The check runs against the **raw decoded
JSON map** before binding to the typed struct — this catches
denylisted field names the typed struct would silently drop. A leak
detection yields `Verdict=tool_unavailable` with the offending field
path in `TestReport.Error`. Denylist mirrors
`apps/verifier/internal/verification/verification.go`.

## Wire protocol

```
===CRUCIBLE-TESTREPORT===
{ "schema_version": "1", "task_id": "...", "tier": "...", ... }
```

The daemon's `processpool.trimPrelude` strips both the delimiter and
any pre-amble noise the toolchain might emit (e.g. `go test`'s leading
`=== RUN` lines if they leak to stdout).

## Layout

```
verifiers/go/
├── go.mod
├── README.md
├── cmd/crucible-verify-go/
│   └── main.go                  # CLI entrypoint + tier dispatch
├── internal/
│   ├── schema/request.go        # runner-side VerificationRequest mirror
│   ├── audit/audit.go           # reasoning-leak denylist
│   ├── diff/diff.go             # Go-file filter for the diff
│   └── tiers/
│       ├── tier0_mutation.go        # go-mutesting wrapper, inverted-semantics parser
│       ├── tier1_pbt.go             # rapid + testing.F driver
│       ├── tier2_contract.go        # schemathesis shell-out
│       ├── tier3_proof.go           # tool_unavailable stub
│       ├── tier4_honest_ci.go       # double go build + sha256 compare
│       └── export_test_helpers.go   # *ForTest re-exports for module-root tests
├── pool_test.go                 # hermetic end-to-end test suite
└── fixtures/
    ├── good/                    # well-tested fixture (Tier 0/1 happy path)
    └── weak/                    # under-tested fixture (Tier 0 catches survivors)
```

## Testing locally

```bash
# Hermetic unit tests (no go-mutesting / rapid required on host):
go test ./...

# Smoke test against the daemon's process pool:
( cd ../../apps/verifier && go test ./internal/processpool -run TestPool )
```
