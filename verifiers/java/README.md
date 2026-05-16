# crucible-verify-java (stub)

Phase 4 stub. No design partner requested Java/Kotlin v1; the interface
ships so Phase 9+ can fill in.

## Planned tier coverage

| Tier | Tool | Status |
|---|---|---|
| 0 — Mutation | Pitest 1.16+ | stub; binary returns `tool_unavailable` |
| 1 — PBT | jqwik 1.9 + JQF | stub |
| 2 — Contract | schemathesis (Polyglot) | stub |
| 3 — Proof | KeY / Dafny (translated) | stub |
| 4 — Honest-CI | Maven `mvn -B verify -Dreproducible` + `diffoscope` | stub |

## CLI contract

When v1 ships:

```
crucible-verify-java --tier=tier_0_mutation < request.json > report.json
```

Reads VerificationRequest JSON from stdin, writes `===CRUCIBLE-TESTREPORT===\n` + TestReport JSON to stdout.

## What ships in Phase 4

- `crucible-verify-java.sh` — a shim that emits a fully-formed
  TestReport with `Verdict=tool_unavailable`. The verifier daemon's
  process pool surfaces this as a `tool_unavailable` finding rather than
  silently skipping.
- This README documenting the contract.
