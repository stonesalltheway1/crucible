// tier3Proof.ts — placeholder dispatcher.
//
// There is no mainstream formal verifier targeting TypeScript in v1 (May
// 2026). Options like Stainless-for-TS or Liquid-TS are research-grade.
// The verifier daemon expects a TestReport regardless, so we emit
// `tool_unavailable` with a structured `proof` block that names the
// (absent) prover. The daemon's Tier-3 logic handles fallback to
// Tier 2.5 (exhaustive PBT + CODEOWNER review) per
// docs/01-architecture/verifier-pipeline.md.
import { finalizeTestReport, newTestReport, } from "../schema.js";
export async function runTier3Proof(req, opts = {}) {
    const startedAt = new Date();
    const wallClockSeconds = opts.wallClockSeconds ??
        Math.max(0, req.budget.wall_clock_cap_seconds || 0);
    const baseReport = newTestReport({
        task_id: req.task_id,
        diff_hash: req.diff.base_sha ?? req.base_sha,
        tier: "tier_3_proof",
        framework: "tier3-typescript-placeholder",
        reporter_id: "crucible-verify-typescript",
        reporter_version: "2026.06.0-phase4",
        wall_clock_budget_seconds: wallClockSeconds,
        started_at: startedAt,
    });
    const finding = {
        category: "tier3_tool_unavailable",
        severity: "info",
        detail: "No mainstream TypeScript formal verifier wired in v1. Dispatcher should fall back to Tier 2.5 (exhaustive PBT + CODEOWNER review) per docs/01-architecture/verifier-pipeline.md.",
    };
    return finalizeTestReport({
        ...baseReport,
        verdict: "tool_unavailable",
        passed: false,
        proof: {
            prover: "none",
            timed_out: false,
            fallback_tier: "tier_2_5",
            codeowner_review_required: true,
            unsoundness_hints: [
                "TypeScript has no production-grade formal verifier as of 2026-05; consider Lean/Dafny extraction or rewrite the critical path in Rust+Kani.",
            ],
        },
        findings: [finding],
        // Daemon expects "verifier crashed" vs "tool not wired" to be
        // distinguishable. Leaving Error empty signals the latter.
    }, new Date());
}
//# sourceMappingURL=tier3Proof.js.map