//! Tier-2 contract testing via `schemathesis`.
//!
//! The Python runner owns the bulk of the schemathesis machinery; for
//! Rust we dispatch to the same CLI (schemathesis is language-agnostic
//! at the spec layer). If the request carries no `spec_changes`, the
//! tier is skipped. Otherwise we shell out to `schemathesis run --json
//! --checks all <spec>` per spec change and collect violations.

use std::process::Command;
use std::time::Instant;

use serde::Deserialize;
use sha2::{Digest, Sha256};

use crate::schema::{
    ContractStats, ContractViolation, Finding, TestReport, Verdict, VerificationRequest,
};
use crate::tiers::locate;

/// Entrypoint invoked by `tiers::dispatch`.
pub fn run(req: &VerificationRequest, mut report: TestReport) -> TestReport {
    report.framework = "schemathesis".to_string();
    let started = Instant::now();

    if req.spec_changes.is_empty() {
        report.verdict = Verdict::Skipped;
        report.passed = true;
        report.contract = Some(ContractStats::default());
        report.stamp_finished();
        return report;
    }

    let Some(bin) = locate("schemathesis") else {
        report.stamp_finished();
        return report.tool_unavailable("schemathesis not on PATH");
    };

    let mut stats = ContractStats::default();
    let mut all_violations = Vec::new();
    let mut all_passed = true;

    for change in &req.spec_changes {
        if change.path.is_empty() {
            continue;
        }
        stats.spec_path = change.path.clone();
        let mut hasher = Sha256::new();
        hasher.update(change.path.as_bytes());
        hasher.update(b"\0");
        hasher.update(change.current_hash.as_bytes());
        stats.spec_hash = hex::encode(hasher.finalize());

        let out = Command::new(&bin)
            .arg("run")
            .arg("--checks")
            .arg("all")
            .arg("--json")
            .arg(&change.path)
            .output();

        match out {
            Ok(o) => {
                let stdout = String::from_utf8_lossy(&o.stdout);
                let stderr = String::from_utf8_lossy(&o.stderr);
                if !o.status.success() {
                    all_passed = false;
                }
                let mut violations = parse_schemathesis_json(&stdout);
                if violations.is_empty() && !o.status.success() {
                    violations.push(ContractViolation {
                        endpoint: change.path.clone(),
                        method: String::new(),
                        check: "schemathesis_failure".to_string(),
                        detail: stderr
                            .lines()
                            .last()
                            .unwrap_or("schemathesis exited non-zero")
                            .to_string(),
                        reproducer: String::new(),
                    });
                }
                all_violations.extend(violations);
            }
            Err(e) => {
                report.stamp_finished();
                return report.tool_unavailable(format!("schemathesis spawn: {e}"));
            }
        }
    }

    stats.checks = vec!["all".to_string()];
    stats.violations = all_violations.clone();
    report.findings = all_violations
        .into_iter()
        .map(|v| Finding {
            category: "contract_violation".to_string(),
            severity: "error".to_string(),
            file: v.endpoint.clone(),
            line: 0,
            detail: format!("{} {}: {}", v.method, v.check, v.detail),
            suggested_fix: String::new(),
        })
        .collect();

    let passed = all_passed && stats.violations.is_empty();
    report.verdict = if passed {
        Verdict::Passed
    } else {
        Verdict::Failed
    };
    report.passed = passed;
    report.contract = Some(stats);
    report.duration_seconds = started.elapsed().as_secs_f64();
    report.stamp_finished();
    report
}

/// Subset of the schemathesis 4.x JSON report we care about.
#[derive(Debug, Deserialize)]
struct SchemaThesisReport {
    #[serde(default)]
    results: Vec<SchemaThesisResult>,
}

#[derive(Debug, Deserialize)]
struct SchemaThesisResult {
    #[serde(default)]
    method: String,
    #[serde(default)]
    path: String,
    #[serde(default)]
    checks: Vec<SchemaThesisCheck>,
}

#[derive(Debug, Deserialize)]
struct SchemaThesisCheck {
    #[serde(default)]
    name: String,
    #[serde(default)]
    value: String,
    #[serde(default)]
    message: String,
}

fn parse_schemathesis_json(stdout: &str) -> Vec<ContractViolation> {
    // schemathesis sometimes interleaves human-readable lines before the
    // JSON document; grab the last `{`-prefixed line.
    let json_line = stdout
        .lines()
        .filter(|l| l.trim_start().starts_with('{'))
        .next_back()
        .unwrap_or("");
    if json_line.is_empty() {
        return Vec::new();
    }
    let doc: Result<SchemaThesisReport, _> = serde_json::from_str(json_line);
    let mut violations = Vec::new();
    let Ok(doc) = doc else {
        return violations;
    };
    for r in doc.results {
        for c in r.checks {
            if c.value.eq_ignore_ascii_case("failure") || c.value.eq_ignore_ascii_case("error") {
                violations.push(ContractViolation {
                    endpoint: r.path.clone(),
                    method: r.method.clone(),
                    check: c.name,
                    detail: c.message,
                    reproducer: String::new(),
                });
            }
        }
    }
    violations
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn parses_failure_check() {
        let body = r#"{"results":[{"method":"GET","path":"/foo","checks":[{"name":"response_schema_conformance","value":"failure","message":"oh no"}]}]}"#;
        let v = parse_schemathesis_json(body);
        assert_eq!(v.len(), 1);
        assert_eq!(v[0].endpoint, "/foo");
        assert_eq!(v[0].check, "response_schema_conformance");
    }

    #[test]
    fn ignores_success_checks() {
        let body = r#"{"results":[{"method":"GET","path":"/ok","checks":[{"name":"response_schema_conformance","value":"success","message":""}]}]}"#;
        let v = parse_schemathesis_json(body);
        assert!(v.is_empty());
    }
}
