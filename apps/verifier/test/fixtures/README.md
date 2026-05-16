# Test fixtures for the verifier daemon

Per-language fixture projects used by the end-to-end test suite at
`apps/verifier/test/`. Each language ships TWO variants:

- `*-correct/` — a fully-implemented project whose tests prove the
  invariants. Phase 4 expects every runner to APPROVE this fixture
  (≥98% acceptance per the brief).
- `*-buggy/`   — the same project with a deliberately-buggy diff
  applied. Phase 4 expects every runner to REJECT this fixture (≥95%
  rejection per the brief).

The verifier daemon's integration tests spawn the per-language CLIs
against these fixtures via the `ExecProvider` and assert verdict
parity.

## Fixture index

| Language | Correct fixture | Buggy fixture | Tier coverage |
|---|---|---|---|
| Python | `python-correct/` | `python-buggy/` | T0, T1, T2 |
| TypeScript | `ts-correct/` | `ts-buggy/` | T0, T1, T2 |
| Rust | `rust-correct/` | `rust-buggy/` | T0, T1, T2 |
| Go | `go-correct/` | `go-buggy/` | T0, T1, T2 |

T3 fixtures live in `tier3-dafny/` and exercise the Dafny adapter
directly (the per-language runners dispatch to it).

T4 fixtures live in `tier4-rebuild/` and exercise the honest-CI
hermetic-build comparison.

## How to add a fixture

```
test/fixtures/<lang>-<variant>/
├── manifest.yaml         # describes the fixture: tier set, expected verdict
├── diff.unified          # the agent-authored diff under test
├── src/                  # source tree the runner sees as "current"
├── tests/                # tests the runner uses to kill mutants / prove
└── openapi.yaml?         # optional spec for Tier 2
```

The runner's `manifest.yaml` declares the expected verdict; the
end-to-end test asserts that the runner reaches it.
