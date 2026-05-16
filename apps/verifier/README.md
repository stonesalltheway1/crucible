# crucible-verifier

The Phase-4 verifier daemon. Receives `VerificationRequest`s from the
control plane after the executor agent claims `done`, fans out tier
runners against a per-language verifier process, runs the
cross-family LLM-judge rubric, and emits a signed
`VerifierApproval` / `VerifierRejection` attestation.

## Architectural invariants (ADR-002 / ADR-008 / Phase 4 brief)

1. **The verifier model MUST be from a different vendor lineage than
   the executor.** Same-family pairings are refused at the routing
   layer — `apps/verifier/internal/rubric` returns an error rather
   than fall through.
2. **The verifier never sees the executor's reasoning trace.** Each
   `VerificationRequest` is audited at ingest: any field whose name
   matches the deny-list (`reasoning`, `chain_of_thought`,
   `thinking_trace`, `cot`, `scratchpad`, ...) causes a
   `LeakageError` before the model call. See
   `internal/rubric/audit.go`.
3. **Verifier-side cost is billed separately.** The control plane's
   Bounded Budget Enforcer tracks `verifier_spent_usd` independently
   of the executor budget (ADR-009).
4. **Tier 3 timeout never fails open.** On Dafny / Lean / TLA+
   timeout the dispatcher falls back to Tier 2.5 with mandatory
   CODEOWNER review; the `VerifierApproval` records
   `tier_2_5_fallback_engaged=true` and is otherwise refused.
5. **The Tier 4 honest-CI verifier bit-identically re-derives the
   build.** Any drift between the executor's rebuild_hash and the
   verifier's is a hard rejection.

## Subpackages

| Path | Responsibility |
|---|---|
| `internal/api`            | gRPC + HTTP surface exposing `twin.verify.*` |
| `internal/dispatcher`     | tier-selection state machine; fans out to runners |
| `internal/processpool`    | per-language verifier process slot manager |
| `internal/rubric`         | cross-family LLM-judge rubric scoring |
| `internal/criticalpath`   | the multi-signal classifier (regex/centrality/CVE/marker/LLM) |
| `internal/runners`        | per-language runner client (talks to the per-language CLI) |
| `internal/tier3`          | Dafny + Lean/TLA+/Z3 dispatchers |
| `internal/tier4`          | honest-CI rebuild + SLSA-L3 attestation |
| `internal/calibrate`      | `crucible calibrate` per-tenant weight fitting |
| `pkg/testreport`          | canonical TestReport type (used by all runners) |
| `cmd/crucible-verifier`   | the daemon entrypoint |

## Running locally

```
export CRUCIBLE_VERIFIER_LISTEN_ADDR=:9080
export CRUCIBLE_VERIFIER_KEY_DIR=$HOME/.crucible/verifier-keys
export CRUCIBLE_VERIFIER_JOURNAL_PATH=$HOME/.crucible/verifier-journal.log
export ANTHROPIC_API_KEY=...   # executor side (audit-only here)
export GOOGLE_API_KEY=...      # default verifier model (Gemini 3.1 Pro)
crucible-verifier
```

Without `GOOGLE_API_KEY` the rubric falls back to a heuristic scorer
that still enforces the cross-family invariant — useful for
hermetic CI runs.

See `docs/PHASE-4-REPORT.md` for the end-of-session report and
`docs/01-architecture/verifier-pipeline.md` for the design.
