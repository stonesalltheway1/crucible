// schema.ts — wire-format mirrors of the Go canonical types in
// apps/verifier/pkg/testreport/testreport.go and
// apps/verifier/internal/verification/verification.go.
//
// IMPORTANT: keys are snake_case to match the Go JSON tags exactly.
// This file is the authoritative TS contract — the daemon parses
// whatever we emit into the Go struct, so any drift here breaks
// the pipeline. SchemaVersion is pinned to "1".
export const SCHEMA_VERSION = "1";
export const PREDICATE_TYPE = "https://crucible.dev/TestReport/v1";
export const TIER_VALUES = [
    "tier_0_mutation",
    "tier_1_pbt",
    "tier_2_contract",
    "tier_3_proof",
    "tier_4_honest_ci",
];
// ---- Helpers ------------------------------------------------------------
/**
 * Construct a TestReport with the schema-version-locked fields filled in.
 * Callers populate the tier-specific stats and findings.
 */
export function newTestReport(args) {
    return {
        schema_version: SCHEMA_VERSION,
        task_id: args.task_id,
        diff_hash: args.diff_hash,
        tier: args.tier,
        language: "typescript",
        framework: args.framework,
        verdict: "tool_unavailable",
        passed: false,
        started_at: args.started_at.toISOString(),
        finished_at: args.started_at.toISOString(),
        duration_seconds: 0,
        wall_clock_budget_seconds: args.wall_clock_budget_seconds,
        reporter_id: args.reporter_id,
        ...(args.reporter_version !== undefined
            ? { reporter_version: args.reporter_version }
            : {}),
    };
}
export function finalizeTestReport(report, finished_at) {
    const started = new Date(report.started_at).getTime();
    const ended = finished_at.getTime();
    const duration = Math.max(0, (ended - started) / 1000);
    return {
        ...report,
        finished_at: finished_at.toISOString(),
        duration_seconds: duration,
    };
}
//# sourceMappingURL=schema.js.map