//! Tier-1 property-based tests + fuzz.
//!
//! Pipeline:
//!
//! 1. Run `cargo test --tests` with `PROPTEST_CASES=10000` so proptest
//!    auto-runs at the Crucible-mandated iteration count.
//! 2. Scan stdout/stderr for `Falsifying example` lines (proptest's
//!    shrunk-counterexample marker) and `test ... panicked` lines.
//! 3. If a `fuzz/` directory exists in the working tree, locate the
//!    first fuzz target via `cargo fuzz list` and run
//!    `cargo fuzz run <target> -- -max_total_time=15`. Any crashes are
//!    recorded as `fuzz_crashes`.

use std::path::{Path, PathBuf};
use std::process::Command;
use std::time::Instant;

use regex::Regex;

use crate::schema::{Counterexample, Finding, PbtStats, TestReport, Verdict, VerificationRequest};
use crate::tiers::locate;

/// Crucible mandate: proptest must run at ≥10,000 iterations.
pub const ITERATIONS_MIN: u64 = 10_000;

/// Maximum wall-clock for `cargo fuzz run` (seconds). Kept short to stay
/// within Tier-1's 5-minute default budget; the dispatcher's own
/// wall-clock cap is the upper bound.
pub const FUZZ_MAX_SECONDS: u64 = 15;

/// Entrypoint invoked by `tiers::dispatch`.
pub fn run(_req: &VerificationRequest, mut report: TestReport) -> TestReport {
    report.framework = "proptest+cargo-fuzz".to_string();
    let started = Instant::now();

    let Some(cargo) = locate("cargo") else {
        report.stamp_finished();
        return report.tool_unavailable("cargo not on PATH");
    };

    let mut stats = PbtStats {
        iterations: 0,
        iterations_min: ITERATIONS_MIN,
        ..PbtStats::default()
    };

    // ---- proptest pass -------------------------------------------------
    let test_output = Command::new(&cargo)
        .arg("test")
        .arg("--tests")
        .arg("--no-fail-fast")
        .env("PROPTEST_CASES", ITERATIONS_MIN.to_string())
        // Force colour off for predictable parsing.
        .env("CARGO_TERM_COLOR", "never")
        .output();

    let test_result = match test_output {
        Ok(o) => o,
        Err(e) => {
            report.stamp_finished();
            return report.tool_unavailable(format!("cargo test spawn: {e}"));
        }
    };

    let stdout = String::from_utf8_lossy(&test_result.stdout);
    let stderr = String::from_utf8_lossy(&test_result.stderr);

    let (props, counters) = parse_proptest_output(&stdout, &stderr);
    stats.properties = props;
    stats.iterations = if stats.properties.is_empty() {
        0
    } else {
        ITERATIONS_MIN * stats.properties.len() as u64
    };
    stats.counterexamples = counters;

    let pbt_passed = test_result.status.success() && stats.counterexamples.is_empty();

    // ---- fuzz pass -----------------------------------------------------
    // The fuzz directory lives at the workdir's root, not in `req`.
    let fuzz_dir = std::env::current_dir()
        .ok()
        .map(|d| d.join("fuzz"))
        .filter(|p| p.exists());

    if let Some(fdir) = fuzz_dir.as_ref() {
        match run_fuzz(&cargo, fdir) {
            Ok(fuzz) => {
                stats.fuzz_corpus_size = fuzz.corpus_size;
                stats.fuzz_new_seeds = fuzz.new_seeds;
                stats.fuzz_crashes = fuzz.crashes;
            }
            Err(e) => {
                eprintln!("crucible-verify-rust: cargo fuzz skipped: {e}");
            }
        }
    }

    report.findings = stats
        .counterexamples
        .iter()
        .map(|c| Finding {
            category: "property_failed".to_string(),
            severity: "error".to_string(),
            file: String::new(),
            line: 0,
            detail: format!("{}: {}", c.property, c.shrunk),
            suggested_fix: String::new(),
        })
        .collect();

    let passed = pbt_passed && stats.fuzz_crashes == 0;
    report.verdict = if passed {
        Verdict::Passed
    } else {
        Verdict::Failed
    };
    report.passed = passed;
    report.pbt = Some(stats);
    report.duration_seconds = started.elapsed().as_secs_f64();
    report.stamp_finished();
    report
}

#[derive(Debug, Default)]
struct FuzzOutcome {
    corpus_size: u64,
    new_seeds: u64,
    crashes: u64,
}

fn run_fuzz(cargo: &Path, fuzz_dir: &Path) -> anyhow::Result<FuzzOutcome> {
    // List fuzz targets first.
    let list = Command::new(cargo)
        .arg("fuzz")
        .arg("list")
        .current_dir(fuzz_dir.parent().unwrap_or(fuzz_dir))
        .output()?;
    let listed = String::from_utf8_lossy(&list.stdout);
    let target = listed
        .lines()
        .map(str::trim)
        .find(|l| !l.is_empty())
        .ok_or_else(|| anyhow::anyhow!("cargo fuzz list returned no targets"))?
        .to_string();

    let run = Command::new(cargo)
        .arg("fuzz")
        .arg("run")
        .arg(&target)
        .arg("--")
        .arg(format!("-max_total_time={FUZZ_MAX_SECONDS}"))
        .current_dir(fuzz_dir.parent().unwrap_or(fuzz_dir))
        .output()?;
    let stderr = String::from_utf8_lossy(&run.stderr);
    let crashes = u64::from(stderr.contains("crash-") || stderr.contains("ERROR:"));

    let corpus_dir: PathBuf = fuzz_dir.join("corpus").join(&target);
    let mut corpus_size = 0u64;
    if let Ok(rd) = std::fs::read_dir(&corpus_dir) {
        corpus_size = rd.filter_map(Result::ok).count() as u64;
    }
    Ok(FuzzOutcome {
        corpus_size,
        new_seeds: 0,
        crashes,
    })
}

/// Parse proptest output for the two markers that matter:
///   - `proptest: Modifying configuration: …` (kept for completeness)
///   - `Falsifying example: …` (shrunk counter-example)
///   - `thread 'test_name' panicked` (catch-all)
///
/// Returns (property names seen, counterexamples).
fn parse_proptest_output(stdout: &str, stderr: &str) -> (Vec<String>, Vec<Counterexample>) {
    // Pattern: "test foo::bar ..." that proptest annotates as a property.
    let test_name_re = Regex::new(r"(?m)^test\s+([A-Za-z0-9_:]+)\s+\.\.\.").expect("static regex");
    let falsifying_re = Regex::new(r"(?ms)Falsifying example for `([^`]+)`:\s*(.+?)(?:\n\n|\z)")
        .expect("static regex");
    let panic_re = Regex::new(r"thread '([^']+)' panicked at ([^\n]+)").expect("static regex");

    let mut properties = Vec::new();
    for cap in test_name_re.captures_iter(stdout) {
        let name = cap[1].to_string();
        if name.contains("prop_") || name.contains("property_") {
            properties.push(name);
        }
    }
    properties.sort();
    properties.dedup();

    let mut counters = Vec::new();
    for cap in falsifying_re.captures_iter(stdout) {
        counters.push(Counterexample {
            property: cap[1].to_string(),
            shrunk: cap[2].trim().to_string(),
            seed: String::new(),
            stack_hint: String::new(),
        });
    }
    for cap in falsifying_re.captures_iter(stderr) {
        counters.push(Counterexample {
            property: cap[1].to_string(),
            shrunk: cap[2].trim().to_string(),
            seed: String::new(),
            stack_hint: String::new(),
        });
    }
    for cap in panic_re.captures_iter(stderr) {
        // A panic without a matching `Falsifying example` line. Still
        // record it so the dispatcher sees the failure.
        if !counters.iter().any(|c| c.property == cap[1]) {
            counters.push(Counterexample {
                property: cap[1].to_string(),
                shrunk: cap[2].trim().to_string(),
                seed: String::new(),
                stack_hint: "panic".to_string(),
            });
        }
    }
    (properties, counters)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn captures_falsifying_example() {
        let stdout = "
test prop_addition_is_commutative ... FAILED
Falsifying example for `prop_addition_is_commutative`:
    a = 1
    b = 2

test result: FAILED. 0 passed; 1 failed
";
        let (props, counters) = parse_proptest_output(stdout, "");
        assert!(props.iter().any(|p| p.contains("prop_addition")));
        assert_eq!(counters.len(), 1);
        assert_eq!(counters[0].property, "prop_addition_is_commutative");
        assert!(counters[0].shrunk.contains("a = 1"));
    }

    #[test]
    fn captures_panic_without_falsifying_line() {
        let stderr = "thread 'tests::prop_sample' panicked at src/lib.rs:10:5\n";
        let (_, counters) = parse_proptest_output("", stderr);
        assert_eq!(counters.len(), 1);
        assert_eq!(counters[0].stack_hint, "panic");
    }
}
