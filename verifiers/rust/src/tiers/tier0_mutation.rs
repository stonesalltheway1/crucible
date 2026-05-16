//! Tier-0 mutation testing via `cargo-mutants` (pinned 27.0.0).
//!
//! Pipeline:
//!
//! 1. Materialise the request's unified diff to a tempfile.
//! 2. Run `cargo mutants --in-diff <tmp>.diff --json --output mutants.out/`.
//! 3. Parse `mutants.out/outcomes.json` (cargo-mutants 27.x schema).
//! 4. Fold the parsed outcomes into `MutationStats`.
//!
//! The Crucible threshold for Rust is 0.85 (85% mutants killed) per the
//! verifier-pipeline doc. `diff_scoped` is **always** `true` (we only
//! ever pass `--in-diff`).

use std::path::Path;
use std::process::Command;
use std::time::Instant;

use serde::Deserialize;

use crate::diff;
use crate::schema::{
    Finding, MutationStats, SurvivedMutant, TestReport, Verdict, VerificationRequest,
};
use crate::tiers::locate;

/// Crucible-wide threshold for Rust mutation score (diff-scoped).
pub const RUST_MUTATION_THRESHOLD: f64 = 0.85;

/// Entrypoint invoked by `tiers::dispatch`.
pub fn run(req: &VerificationRequest, mut report: TestReport) -> TestReport {
    report.framework = "cargo-mutants".to_string();
    let started = Instant::now();

    // Always emit a diff-scoped report skeleton so the dispatcher's
    // validator never trips on `diff_scoped=false`, even when we end
    // up bailing on tool_unavailable below.
    let mut stats = MutationStats {
        threshold: RUST_MUTATION_THRESHOLD,
        diff_scoped: true,
        mutated_files: diff::rust_paths(&req.diff),
        ..MutationStats::default()
    };

    let Some(cargo) = locate("cargo") else {
        report.mutation = Some(stats);
        report.stamp_finished();
        return report.tool_unavailable("cargo not on PATH");
    };

    if locate("cargo-mutants").is_none() {
        // cargo-mutants might still be reachable via `cargo mutants`
        // if a shim exists; fall through and let the cargo invocation
        // surface the real error.
        eprintln!("crucible-verify-rust: warning — cargo-mutants binary not found on PATH; will attempt `cargo mutants` anyway");
    }

    // Materialise the unified diff.
    let diff_body = diff::build_unified_diff(&req.diff);
    if diff_body.trim().is_empty() {
        report.verdict = Verdict::Skipped;
        report.passed = true;
        report.mutation = Some(stats);
        report.stamp_finished();
        report.error = "no rust files in diff — mutation testing skipped".to_string();
        return report;
    }

    let tmp = match tempfile::tempdir() {
        Ok(t) => t,
        Err(e) => {
            report.mutation = Some(stats);
            report.stamp_finished();
            return report.tool_unavailable(format!("tempdir failed: {e}"));
        }
    };
    let diff_path = tmp.path().join("crucible.diff");
    if let Err(e) = std::fs::write(&diff_path, &diff_body) {
        report.mutation = Some(stats);
        report.stamp_finished();
        return report.tool_unavailable(format!("write diff: {e}"));
    }
    let out_dir = tmp.path().join("mutants.out");

    // cargo-mutants runs against the working tree of the executor's
    // repo, which the dispatcher has already mounted into the verifier
    // sandbox at cwd. We DON'T set `current_dir(...)` — that would
    // break cargo's manifest discovery.
    let status = Command::new(&cargo)
        .arg("mutants")
        .arg("--in-diff")
        .arg(&diff_path)
        .arg("--json")
        .arg("--output")
        .arg(&out_dir)
        .stderr(std::process::Stdio::inherit())
        .stdout(std::process::Stdio::inherit())
        .status();

    let exit_ok = match status {
        Ok(s) => {
            // cargo-mutants returns:
            //   0 → all mutants killed
            //   1 → some mutants survived (still a valid run)
            //   2+ → tool failure
            s.code().is_some_and(|c| c < 2)
        }
        Err(e) => {
            report.mutation = Some(stats);
            report.stamp_finished();
            return report.tool_unavailable(format!("cargo mutants spawn: {e}"));
        }
    };

    let outcomes_path = out_dir.join("outcomes.json");
    if !outcomes_path.exists() || !exit_ok {
        report.stamp_finished();
        report.mutation = Some(stats);
        return report.tool_unavailable("cargo-mutants produced no outcomes.json (tool failure)");
    }

    match parse_outcomes(&outcomes_path) {
        Ok(parsed) => {
            stats.killed = parsed.killed;
            stats.survived = parsed.survived;
            stats.not_covered = parsed.not_covered;
            stats.timeout = parsed.timeout;
            stats.total = parsed.total;
            stats.survived_summary = parsed.survived_summary.clone();
            stats.score = if stats.killed + stats.survived == 0 {
                0.0
            } else {
                f64::from(stats.killed) / f64::from(stats.killed + stats.survived)
            };
            let passed = stats.score >= stats.threshold && stats.survived == 0;
            report.verdict = if passed {
                Verdict::Passed
            } else {
                Verdict::Failed
            };
            report.passed = passed;
            report.findings = parsed
                .survived_summary
                .into_iter()
                .map(|m| Finding {
                    category: "mutation_survived".to_string(),
                    severity: "error".to_string(),
                    file: m.file,
                    line: m.line,
                    detail: format!(
                        "{} survived (replaced {:?} with {:?})",
                        m.mutator, m.original, m.replacement
                    ),
                    suggested_fix: String::new(),
                })
                .collect();
        }
        Err(e) => {
            report.error = format!("parse outcomes.json: {e}");
            report.verdict = Verdict::Failed;
            report.passed = false;
        }
    }

    report.mutation = Some(stats);
    report.duration_seconds = started.elapsed().as_secs_f64();
    report.stamp_finished();
    report
}

/// Internal aggregate returned by [`parse_outcomes`].
struct ParsedOutcomes {
    killed: u32,
    survived: u32,
    not_covered: u32,
    timeout: u32,
    total: u32,
    survived_summary: Vec<SurvivedMutant>,
}

/// cargo-mutants 27.x `outcomes.json` shape (relevant subset).
#[derive(Debug, Deserialize)]
struct OutcomesDoc {
    #[serde(default)]
    outcomes: Vec<Outcome>,
}

#[derive(Debug, Deserialize)]
struct Outcome {
    #[serde(default)]
    summary: Option<String>,
    #[serde(default)]
    scenario: Option<Scenario>,
}

#[derive(Debug, Deserialize)]
struct Scenario {
    #[serde(default)]
    mutant: Option<Mutant>,
}

#[derive(Debug, Deserialize)]
struct Mutant {
    #[serde(default)]
    file: String,
    #[serde(default)]
    line: u32,
    #[serde(default)]
    function: String,
    #[serde(default)]
    genre: String,
    #[serde(default)]
    replacement: String,
    #[serde(default)]
    original: String,
}

fn parse_outcomes(path: &Path) -> anyhow::Result<ParsedOutcomes> {
    let raw = std::fs::read_to_string(path)?;
    let doc: OutcomesDoc = serde_json::from_str(&raw)?;
    let mut killed = 0u32;
    let mut survived = 0u32;
    let mut not_covered = 0u32;
    let mut timeout = 0u32;
    let mut survived_summary = Vec::new();
    for o in &doc.outcomes {
        let summary = o.summary.as_deref().unwrap_or("");
        match summary {
            "CaughtMutant" | "Caught" => killed += 1,
            "MissedMutant" | "Missed" => {
                survived += 1;
                if let Some(m) = o.scenario.as_ref().and_then(|s| s.mutant.as_ref()) {
                    survived_summary.push(SurvivedMutant {
                        file: m.file.clone(),
                        line: m.line,
                        mutator: if m.genre.is_empty() {
                            m.function.clone()
                        } else {
                            m.genre.clone()
                        },
                        original: m.original.clone(),
                        replacement: m.replacement.clone(),
                    });
                }
            }
            "Unviable" => not_covered += 1,
            "Timeout" => timeout += 1,
            _ => {}
        }
    }
    let total = killed + survived + not_covered + timeout;
    Ok(ParsedOutcomes {
        killed,
        survived,
        not_covered,
        timeout,
        total,
        survived_summary,
    })
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::io::Write;

    #[test]
    fn parses_minimal_outcomes() {
        let body = serde_json::json!({
            "outcomes": [
                { "summary": "Caught", "scenario": { "mutant": { "file": "src/a.rs", "line": 12, "function": "f", "genre": "BinaryOperator", "original": "+", "replacement": "-" } } },
                { "summary": "Missed", "scenario": { "mutant": { "file": "src/b.rs", "line": 7, "function": "g", "genre": "BooleanLiteral", "original": "true", "replacement": "false" } } },
                { "summary": "Unviable" },
                { "summary": "Timeout" }
            ]
        });
        let mut f = tempfile::NamedTempFile::new().unwrap();
        write!(f, "{}", body).unwrap();
        let parsed = parse_outcomes(f.path()).unwrap();
        assert_eq!(parsed.killed, 1);
        assert_eq!(parsed.survived, 1);
        assert_eq!(parsed.not_covered, 1);
        assert_eq!(parsed.timeout, 1);
        assert_eq!(parsed.total, 4);
        assert_eq!(parsed.survived_summary.len(), 1);
        assert_eq!(parsed.survived_summary[0].file, "src/b.rs");
        assert_eq!(parsed.survived_summary[0].mutator, "BooleanLiteral");
    }
}
