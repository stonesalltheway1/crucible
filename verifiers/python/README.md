# crucible-verify-python

Per-language verifier runner for the Crucible Phase-4 verification pipeline.

The verifier daemon (Go, at `apps/verifier/`) spawns this CLI inside an
isolated sandbox. The daemon writes a `VerificationRequest` JSON document to
stdin; this CLI runs the requested tier and writes a `TestReport` JSON
document to stdout.

The TestReport schema is **canonical and lives in Go** at
`apps/verifier/pkg/testreport/testreport.go`. This Python package mirrors it
field-for-field via the dataclasses in `crucible_verify_python/schema.py`.
The mirror is round-trip tested against the Go canonical encoding by
`tests/test_schema_roundtrip.py`.

## Tiers

| Flag | Driver | Backing tool |
|---|---|---|
| `--tier=tier_0_mutation` | `tier0_mutation.py` | `mutmut` 3.5 |
| `--tier=tier_1_pbt` | `tier1_pbt.py` | `hypothesis` 6.152 (+ `atheris` if installed) |
| `--tier=tier_2_contract` | `tier2_contract.py` | `schemathesis` 4.18 |
| `--tier=tier_3_proof` | `tier3_proof.py` | stub — real Dafny/Lean adapter lives in `apps/verifier/internal/tier3` |
| `--tier=tier_4_honest_ci` | `tier4_honest_ci.py` | Nix derivation hash + double-build sha256 compare |

## Wire protocol

1. The CLI emits its `TestReport` body **after** a stable delimiter, so
   tooling that mixes preamble with stdout (pytest, mutmut) is safe to
   embed:

   ```
   ===CRUCIBLE-TESTREPORT===
   { ... json ... }
   ```

2. All logs go to **stderr** — stdout is reserved for the delimiter + JSON.

3. If the parsed `VerificationRequest` contains any key matching the
   executor-reasoning denylist (`reasoning`, `chain_of_thought`,
   `scratchpad`, `agent_trace`, ...) the CLI exits **code 2** with a
   stderr message and writes no report. This is defence-in-depth — the Go
   side already audits the same denylist at ingest.

## Usage

```
$ crucible-verify-python --tier=tier_0_mutation < request.json > report.json
```

## Development

```
pip install -e ".[dev,fuzz]"
mypy --strict crucible_verify_python
ruff check
pytest -q
```

## Quality bar

- `mypy --strict` clean on the package source.
- `ruff check` clean.
- Tier 0 against the strong-tests fixture achieves >=85% mutation score.
- Tier 0 against the weak-tests fixture is correctly **rejected**.
- Tier 1 reports a counterexample when one exists.
- The audit guard refuses any request body containing `reasoning` or
  `agent_trace` keys (anywhere in the JSON tree).
