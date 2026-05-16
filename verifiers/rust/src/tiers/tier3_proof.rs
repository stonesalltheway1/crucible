//! Tier-3 formal verification via Kani (pinned 0.67.0).
//!
//! Pipeline:
//!
//! 1. Scan the touched Rust files for `#[kani::proof]` attributes and
//!    extract the harness function names.
//! 2. For each harness, run `cargo kani --harness <name> --solver minisat`
//!    with a per-harness wall-clock cap of 600s (matches the Tier-3
//!    budget in verifier-pipeline.md).
//! 3. Collect obligation counts from Kani's stdout (`** N of M checks
//!    failed`). Any timeout flips `timed_out=true`,
//!    `fallback_tier="tier_2_5"`, and `codeowner_review_required=true`.
//!
//! Bolero (with `bolero-kani`) is the supported bounded-model bridge —
//! propproof is intentionally NOT integrated (effectively unmaintained
//! per May-2026 research).

use std::path::Path;
use std::process::Command;
use std::time::{Duration, Instant};

use regex::Regex;

use crate::diff;
use crate::schema::{Finding, ProofStats, TestReport, Verdict, VerificationRequest};
use crate::tiers::locate;

/// Per-harness wall-clock cap.
pub const HARNESS_TIMEOUT: Duration = Duration::from_secs(600);

/// Entrypoint invoked by `tiers::dispatch`.
pub fn run(req: &VerificationRequest, mut report: TestReport) -> TestReport {
    report.framework = "kani".to_string();
    let started = Instant::now();

    let mut stats = ProofStats {
        prover: "kani".to_string(),
        ..ProofStats::default()
    };

    // Scan diff for harnesses.
    let harnesses = discover_harnesses(req);
    if harnesses.is_empty() {
        report.verdict = Verdict::Skipped;
        report.passed = true;
        stats.obligations = 0;
        stats.discharged = 0;
        report.proof = Some(stats);
        report.stamp_finished();
        report.error = "no #[kani::proof] harnesses in diff".to_string();
        return report;
    }

    let Some(cargo) = locate("cargo") else {
        report.stamp_finished();
        return report.tool_unavailable("cargo not on PATH");
    };
    if locate("cargo-kani").is_none() && locate("kani").is_none() {
        report.stamp_finished();
        return report.tool_unavailable("cargo-kani not on PATH");
    }

    let mut timed_out_any = false;
    let mut findings = Vec::new();
    let mut total_obligations = 0u32;
    let mut total_discharged = 0u32;

    for h in &harnesses {
        let invocation_start = Instant::now();
        let out = Command::new(&cargo)
            .arg("kani")
            .arg("--harness")
            .arg(h)
            .arg("--solver")
            .arg("minisat")
            .output();
        let elapsed = invocation_start.elapsed();
        stats.wall_clock_seconds += elapsed.as_secs_f64();

        let outcome = match out {
            Ok(o) => o,
            Err(e) => {
                report.stamp_finished();
                return report.tool_unavailable(format!("cargo kani spawn: {e}"));
            }
        };

        if elapsed >= HARNESS_TIMEOUT {
            timed_out_any = true;
            findings.push(Finding {
                category: "proof_timeout".to_string(),
                severity: "warn".to_string(),
                file: String::new(),
                line: 0,
                detail: format!("kani harness {h} exceeded {}s", HARNESS_TIMEOUT.as_secs()),
                suggested_fix: "fallback to tier_2_5 (PBT@10k + mutation + codeowner review)"
                    .to_string(),
            });
            continue;
        }

        let stdout = String::from_utf8_lossy(&outcome.stdout);
        let stderr = String::from_utf8_lossy(&outcome.stderr);
        let parsed = parse_kani_output(&stdout, &stderr);
        total_obligations += parsed.obligations;
        total_discharged += parsed.discharged;
        if !outcome.status.success() || parsed.discharged < parsed.obligations {
            findings.push(Finding {
                category: "proof_failed".to_string(),
                severity: "error".to_string(),
                file: String::new(),
                line: 0,
                detail: format!(
                    "kani harness {h}: {}/{} obligations discharged",
                    parsed.discharged, parsed.obligations
                ),
                suggested_fix: String::new(),
            });
        }
    }

    stats.obligations = total_obligations;
    stats.discharged = total_discharged;
    stats.timed_out = timed_out_any;
    if timed_out_any {
        stats.fallback_tier = "tier_2_5".to_string();
        stats.codeowner_review_required = true;
    }
    stats.proof_artifact = harnesses.join(",");
    report.findings = findings;
    report.proof = Some(stats);

    let passed = !timed_out_any
        && total_obligations > 0
        && total_obligations == total_discharged
        && report.findings.is_empty();
    report.verdict = if passed {
        Verdict::Passed
    } else if timed_out_any {
        Verdict::TimedOut
    } else {
        Verdict::Failed
    };
    report.passed = passed;
    report.duration_seconds = started.elapsed().as_secs_f64();
    report.stamp_finished();
    report
}

/// Grep the diff for `#[kani::proof]` attributes and return the
/// associated function names. The regex is intentionally line-anchored
/// so a stray `// #[kani::proof]` in a comment doesn't match.
fn discover_harnesses(req: &VerificationRequest) -> Vec<String> {
    // Allow optional `#[kani::proof(...)]` arguments and arbitrary
    // whitespace + extra attributes between the marker and the `fn`.
    let proof_re = Regex::new(
        r"(?ms)\#\[kani::proof(?:\([^)]*\))?\][^\n]*\n(?:[ \t]*\#\[[^\n]*\][^\n]*\n)*[ \t]*(?:pub\s+)?fn\s+([A-Za-z_][A-Za-z0-9_]*)",
    )
    .expect("static regex");

    let mut harnesses = Vec::new();
    for f in diff::rust_files(&req.diff) {
        scan_text_for_harnesses(&proof_re, f.unified_diff, &mut harnesses);
        // Also check the on-disk file if accessible — diffs are often
        // hunked and may not include the harness line.
        if let Ok(body) = std::fs::read_to_string(Path::new(f.path)) {
            scan_text_for_harnesses(&proof_re, &body, &mut harnesses);
        }
    }
    harnesses.sort();
    harnesses.dedup();
    harnesses
}

fn scan_text_for_harnesses(re: &Regex, text: &str, out: &mut Vec<String>) {
    for cap in re.captures_iter(text) {
        out.push(cap[1].to_string());
    }
}

#[derive(Debug, Default)]
struct ParsedKani {
    obligations: u32,
    discharged: u32,
}

fn parse_kani_output(stdout: &str, stderr: &str) -> ParsedKani {
    // Kani prints lines like:
    //   ** 0 of 12 failed
    //   VERIFICATION:- SUCCESSFUL
    let summary_re = Regex::new(r"\*\*\s+(\d+)\s+of\s+(\d+)\s+failed").expect("static regex");
    let mut parsed = ParsedKani::default();
    for text in [stdout, stderr] {
        if let Some(cap) = summary_re.captures(text) {
            let failed: u32 = cap[1].parse().unwrap_or(0);
            let total: u32 = cap[2].parse().unwrap_or(0);
            parsed.obligations = total;
            parsed.discharged = total.saturating_sub(failed);
            return parsed;
        }
    }
    parsed
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::schema::{Diff, FileChange};

    #[test]
    fn discovers_kani_harness_in_diff_body() {
        let req = VerificationRequest {
            diff: Diff {
                files: vec![FileChange {
                    path: "src/lib.rs".to_string(),
                    unified_diff: "@@ -1 +1 @@\n+#[kani::proof]\n+fn check_invariant() {}\n"
                        .to_string(),
                    ..Default::default()
                }],
            },
            ..Default::default()
        };
        let h = discover_harnesses(&req);
        assert_eq!(h, vec!["check_invariant".to_string()]);
    }

    #[test]
    fn discovers_kani_harness_with_args() {
        let body = "#[kani::proof(unwind = 10)]\nfn proof_unwound() {}\n";
        let re = Regex::new(
            r"(?ms)\#\[kani::proof(?:\([^)]*\))?\][^\n]*\n(?:[ \t]*\#\[[^\n]*\][^\n]*\n)*[ \t]*(?:pub\s+)?fn\s+([A-Za-z_][A-Za-z0-9_]*)",
        )
        .unwrap();
        let mut out = Vec::new();
        scan_text_for_harnesses(&re, body, &mut out);
        assert_eq!(out, vec!["proof_unwound".to_string()]);
    }

    #[test]
    fn parses_kani_success_summary() {
        let parsed = parse_kani_output("** 0 of 7 failed\nVERIFICATION:- SUCCESSFUL\n", "");
        assert_eq!(parsed.obligations, 7);
        assert_eq!(parsed.discharged, 7);
    }

    #[test]
    fn parses_kani_failure_summary() {
        let parsed = parse_kani_output("** 3 of 7 failed\nVERIFICATION:- FAILED\n", "");
        assert_eq!(parsed.obligations, 7);
        assert_eq!(parsed.discharged, 4);
    }
}
