# crucible-verify-typescript

Per-language verifier runner for TypeScript / JavaScript diffs, dispatched by the
Crucible verifier daemon (`apps/verifier/`). One Node CLI, one tier per invocation,
TestReport JSON on stdout — the daemon does the rest.

## Wire protocol

```
stdin:  VerificationRequest (JSON)            -- see apps/verifier/internal/verification
args:   --tier=<tier>
stdout: "===CRUCIBLE-TESTREPORT===\n" + TestReport JSON
stderr: human-readable logs
exit:   0 substantive (verdict carries pass/fail)
        1 procedural crash (no report)
        2 executor-reasoning leak (ADR-002 invariant)
```

`TestReport` keys are snake_case to match the canonical Go schema in
`apps/verifier/pkg/testreport/testreport.go`. `schema_version` is pinned to `"1"`.

## Tiers

| Tier | Tool | Threshold / Mandate |
|---|---|---|
| `tier_0_mutation` | Stryker 9.6.1 + vitest-runner 9.6.1 | 0.85 mutants killed, diff-scoped |
| `tier_1_pbt` | fast-check 4.7.0 + @fast-check/vitest 0.4.1 | >=10,000 iterations per property |
| `tier_2_contract` | schemathesis sidecar (pipx) | zero contract violations |
| `tier_3_proof` | placeholder (no mainstream TS prover) | always `tool_unavailable` |
| `tier_4_honest_ci` | `pnpm install --frozen-lockfile && pnpm build` x 2 | bit-identical sha256(dist/) |

Notes:
- `jsfuzz` is unmaintained (last release 2020). The PBT tier uses `@jazzer.js/core 2.1.0`
  when fuzz coverage is required; it's pinned as an optional dependency.
- Tier 3 has no production-grade TS formal verifier as of May 2026. The runner
  emits `tool_unavailable` with `fallback_tier="tier_2_5"` so the daemon
  degrades to exhaustive PBT + CODEOWNER review (verifier-pipeline.md).

## fast-check iteration count

Tests in this repo can pin `numRuns` per property:

```ts
import { it, fc } from "@fast-check/vitest";

it.prop([fc.string()], { numRuns: 10000 })("idempotent", (s) => { /* ... */ });
```

The verifier sets `VITEST_FAST_CHECK_NUM_RUNS=10000` and `FAST_CHECK_NUM_RUNS=10000`
in the subprocess environment, so existing tests at the default 100 iterations
are automatically scaled to the Crucible minimum without source edits.

## Building / testing locally

```
pnpm install
pnpm run typecheck   # strict-mode tsc --noEmit
pnpm test            # vitest run
pnpm run build       # emits dist/
```

## ADR-002 leak guard

Every request is walked for keys matching the reasoning denylist
(`reasoning`, `chain_of_thought`, `scratchpad`, `agent_trace`,
`executor_trace`, `thinking_trace`, `cot`, `reflection`, plus the
broader Go-side list). Diff file paths are scanned for fragments like
`/reasoning/`. Any hit → exit 2, before any tool invocation.
