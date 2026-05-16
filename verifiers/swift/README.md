# crucible-verify-swift (stub)

Phase 4 stub. No design partner requested Swift/Apple v1.

## Planned tier coverage

| Tier | Tool | Status |
|---|---|---|
| 0 — Mutation | muter | stub |
| 1 — PBT | swift-testing + Sourcery | stub |
| 2 — Contract | schemathesis (Polyglot) | stub |
| 3 — Proof | (no mainstream Swift formal verifier) | stub |
| 4 — Honest-CI | `swift build --static-swift-stdlib` w/ SOURCE_DATE_EPOCH | stub |

## CLI contract

```
crucible-verify-swift --tier=tier_0_mutation < request.json > report.json
```

## What ships in Phase 4

`crucible-verify-swift.sh` — emits `Verdict=tool_unavailable` so the
verifier daemon surfaces the gap. Real adapter lands in Phase 9 if a
design partner asks.
