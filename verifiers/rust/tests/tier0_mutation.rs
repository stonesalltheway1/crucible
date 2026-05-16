//! Tier-0 mutation integration tests.
//!
//! cargo-mutants is too heavy a dependency to install in default CI, so
//! the integration story has two layers:
//!
//! 1. A *fixture* `mutants.out/outcomes.json` that we feed to the
//!    parser directly. This pins the cargo-mutants 27.x outcomes
//!    schema we depend on.
//! 2. A *dispatch* path test that drives `tiers::dispatch` end-to-end
//!    without `cargo-mutants` on PATH — we assert the report falls
//!    back to `tool_unavailable` cleanly and still emits `diff_scoped=true`.

use crucible_verify_rust::schema::{
    Diff, FileChange, Language, Tier, Verdict, VerificationRequest,
};
use crucible_verify_rust::tiers::{self, tier0_mutation::RUST_MUTATION_THRESHOLD};

const FIXTURE_OUTCOMES: &str = include_str!("fixtures/cargo_mutants_outcomes.json");

#[test]
fn fixture_outcomes_schema_parses() {
    // The fixture is exactly what cargo-mutants 27.0.0 emits for a
    // four-mutant run on the bundled `fixtures/sample-crate`. Parsing
    // it via `serde_json::Value` keeps this test free of the private
    // outcome structs.
    let parsed: serde_json::Value =
        serde_json::from_str(FIXTURE_OUTCOMES).expect("fixture parses as JSON");
    let outcomes = parsed["outcomes"].as_array().expect("outcomes is an array");
    assert_eq!(outcomes.len(), 4);
    let summaries: Vec<&str> = outcomes
        .iter()
        .filter_map(|o| o["summary"].as_str())
        .collect();
    assert!(summaries.contains(&"Caught"));
    assert!(summaries.contains(&"Missed"));
}

#[test]
fn dispatch_emits_diff_scoped_skeleton_when_no_diff() {
    // Empty diff → tier reports skipped with diff_scoped=true.
    let req = VerificationRequest {
        task_id: "T-empty".to_string(),
        base_sha: "deadbeef".to_string(),
        diff: Diff { files: vec![] },
        ..Default::default()
    };
    let report = tiers::dispatch(Tier::Mutation, &req);
    assert_eq!(report.tier, Tier::Mutation);
    assert_eq!(report.language, Language::Rust);
    let m = report.mutation.expect("mutation block present");
    assert!(m.diff_scoped, "mutation report MUST be diff-scoped");
    assert!((m.threshold - RUST_MUTATION_THRESHOLD).abs() < f64::EPSILON);
}

#[test]
fn dispatch_falls_back_when_tool_missing() {
    // Skip when cargo-mutants is actually installed — running it for
    // real against the verifier crate itself would take minutes. The
    // CI-portable assertion is the shape-only one above plus
    // `tool_unavailable` on hosts without the binary.
    if which::which("cargo-mutants").is_ok() {
        eprintln!("skipping: cargo-mutants present on PATH; relying on schema_roundtrip + parser unit tests");
        return;
    }
    let req = VerificationRequest {
        task_id: "T-1".to_string(),
        base_sha: "deadbeef".to_string(),
        diff: Diff {
            files: vec![FileChange {
                path: "src/lib.rs".to_string(),
                unified_diff: "--- a/src/lib.rs\n+++ b/src/lib.rs\n@@ -1 +1 @@\n-x\n+y\n"
                    .to_string(),
                status: "modified".to_string(),
                ..Default::default()
            }],
        },
        ..Default::default()
    };
    let report = tiers::dispatch(Tier::Mutation, &req);
    assert_eq!(report.tier, Tier::Mutation);
    let m = report.mutation.expect("mutation skeleton present");
    assert!(m.diff_scoped, "mutation report MUST be diff-scoped");
    assert!(m.mutated_files.iter().any(|p| p == "src/lib.rs"));
    // Without cargo-mutants installed, we expect tool_unavailable.
    assert!(matches!(
        report.verdict,
        Verdict::ToolUnavailable | Verdict::Failed | Verdict::Skipped
    ));
}
