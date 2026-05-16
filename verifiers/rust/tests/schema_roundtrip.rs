//! Schema-roundtrip tests. Asserts the emitted JSON shape matches the Go
//! `testreport.TestReport` contract: snake_case keys, `schema_version`
//! pinned to `"1"`, and the union-of-tiers stats blocks omitted when
//! unused.
//!
//! The Go side is at `apps/verifier/pkg/testreport/testreport.go`. Any
//! drift here will be caught at the dispatcher
//! (`processpool.parseRunnerOutput`) before it reaches the rubric.

use crucible_verify_rust::schema::{
    HonestCiStats, Language, MutationStats, PbtStats, ProofStats, SurvivedMutant, TestReport, Tier,
    Verdict, REPORTER_ID, SCHEMA_VERSION,
};
use serde_json::Value;

fn to_value(r: &TestReport) -> Value {
    serde_json::to_value(r).expect("serialise TestReport")
}

#[test]
fn schema_version_is_pinned_to_one() {
    assert_eq!(SCHEMA_VERSION, "1");
    let r = TestReport::new(Tier::Mutation, Language::Rust, "T-1", "sha");
    let v = to_value(&r);
    assert_eq!(v["schema_version"], "1");
}

#[test]
fn tier_serialises_as_underscore_constant() {
    for (t, wire) in [
        (Tier::Mutation, "tier_0_mutation"),
        (Tier::Pbt, "tier_1_pbt"),
        (Tier::Contract, "tier_2_contract"),
        (Tier::Proof, "tier_3_proof"),
        (Tier::HonestCi, "tier_4_honest_ci"),
    ] {
        let r = TestReport::new(t, Language::Rust, "T", "sha");
        let v = to_value(&r);
        assert_eq!(v["tier"], wire, "tier {} should wire as {wire}", t.as_str());
    }
}

#[test]
fn language_serialises_as_lowercase() {
    let r = TestReport::new(Tier::Mutation, Language::Rust, "T", "sha");
    let v = to_value(&r);
    assert_eq!(v["language"], "rust");
}

#[test]
fn verdict_serialises_as_snake_case() {
    for (v, wire) in [
        (Verdict::Passed, "passed"),
        (Verdict::Failed, "failed"),
        (Verdict::TimedOut, "timed_out"),
        (Verdict::ToolUnavailable, "tool_unavailable"),
        (Verdict::Skipped, "skipped"),
    ] {
        let mut r = TestReport::new(Tier::Mutation, Language::Rust, "T", "sha");
        r.verdict = v;
        let out = to_value(&r);
        assert_eq!(out["verdict"], wire);
    }
}

#[test]
fn unused_stats_blocks_are_omitted() {
    let r = TestReport::new(Tier::Mutation, Language::Rust, "T", "sha");
    let v = to_value(&r);
    let map = v.as_object().expect("object");
    assert!(!map.contains_key("mutation"));
    assert!(!map.contains_key("pbt"));
    assert!(!map.contains_key("contract"));
    assert!(!map.contains_key("proof"));
    assert!(!map.contains_key("honest_ci"));
}

#[test]
fn mutation_block_round_trips() {
    let mut r = TestReport::new(Tier::Mutation, Language::Rust, "T", "sha");
    r.mutation = Some(MutationStats {
        killed: 9,
        survived: 1,
        total: 10,
        score: 0.9,
        threshold: 0.85,
        diff_scoped: true,
        mutated_files: vec!["src/a.rs".to_string()],
        survived_summary: vec![SurvivedMutant {
            file: "src/a.rs".into(),
            line: 12,
            mutator: "BinaryOperator".into(),
            original: "+".into(),
            replacement: "-".into(),
        }],
        ..Default::default()
    });
    let v = to_value(&r);
    let m = &v["mutation"];
    assert_eq!(m["killed"], 9);
    assert_eq!(m["survived"], 1);
    assert_eq!(m["total"], 10);
    assert_eq!(m["score"], 0.9);
    assert_eq!(m["threshold"], 0.85);
    assert_eq!(m["diff_scoped"], true);
    assert_eq!(m["mutated_files"][0], "src/a.rs");
    assert_eq!(m["survived_summary"][0]["mutator"], "BinaryOperator");
    // not_covered and timeout default to 0 → omitted.
    assert!(m.get("not_covered").is_none());
    assert!(m.get("timeout").is_none());
}

#[test]
fn pbt_block_carries_iterations_min() {
    let mut r = TestReport::new(Tier::Pbt, Language::Rust, "T", "sha");
    r.pbt = Some(PbtStats {
        iterations: 10_000,
        iterations_min: 10_000,
        ..Default::default()
    });
    let v = to_value(&r);
    assert_eq!(v["pbt"]["iterations"], 10_000);
    assert_eq!(v["pbt"]["iterations_min"], 10_000);
}

#[test]
fn proof_block_serialises_kani_shape() {
    let mut r = TestReport::new(Tier::Proof, Language::Rust, "T", "sha");
    r.proof = Some(ProofStats {
        prover: "kani".into(),
        obligations: 5,
        discharged: 5,
        timed_out: false,
        ..Default::default()
    });
    let v = to_value(&r);
    let p = &v["proof"];
    assert_eq!(p["prover"], "kani");
    assert_eq!(p["obligations"], 5);
    assert_eq!(p["discharged"], 5);
    assert_eq!(p["timed_out"], false);
    // Defaults omitted.
    assert!(p.get("fallback_tier").is_none());
    assert!(p.get("codeowner_review_required").is_none());
}

#[test]
fn honest_ci_block_shape() {
    let mut r = TestReport::new(Tier::HonestCi, Language::Rust, "T", "sha");
    r.honest_ci = Some(HonestCiStats {
        builder_id: "https://crucible.dev/builders/rust-cargo-double-build/v1".into(),
        executor_rebuild_hash: "aaa".into(),
        verifier_rebuild_hash: "aaa".into(),
        bit_identical: true,
        slsa_level: 2,
        scrubber_audit_ok: true,
        ..Default::default()
    });
    let v = to_value(&r);
    let h = &v["honest_ci"];
    assert_eq!(h["bit_identical"], true);
    assert_eq!(h["slsa_level"], 2);
    assert_eq!(h["scrubber_audit_ok"], true);
}

#[test]
fn reporter_identity_baked_in() {
    let r = TestReport::new(Tier::Mutation, Language::Rust, "T", "sha");
    let v = to_value(&r);
    assert_eq!(v["reporter_id"], REPORTER_ID);
    // reporter_version follows Cargo's CARGO_PKG_VERSION; non-empty.
    let ver = v["reporter_version"].as_str().unwrap_or("");
    assert!(!ver.is_empty(), "reporter_version should be set");
}

#[test]
fn timestamps_are_rfc3339() {
    let r = TestReport::new(Tier::Mutation, Language::Rust, "T", "sha");
    let v = to_value(&r);
    let s = v["started_at"].as_str().expect("started_at string");
    // RFC3339 has a 'T' separator and ends with 'Z' or timezone offset.
    assert!(s.contains('T'));
}
