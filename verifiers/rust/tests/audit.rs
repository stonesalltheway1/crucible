//! Integration tests for the executor-reasoning leak guard.
//!
//! The guard is the runtime defence on top of the schema design — see
//! `apps/verifier/internal/verification/verification.go`. These tests
//! exercise every denylist pattern and confirm scrub-clean payloads pass.

use crucible_verify_rust::audit;
use serde_json::json;

#[test]
fn passes_a_clean_verification_request() {
    let req = json!({
        "task_id": "T-1",
        "tenant_id": "tenant-A",
        "base_sha": "0123456789abcdef",
        "executor_sandbox_id": "sb-executor-1",
        "diff": {
            "files": [
                { "path": "src/lib.rs", "status": "modified",
                  "unified_diff": "@@ -1 +1 @@\n-old\n+new\n" }
            ]
        }
    });
    audit::audit(&req).expect("clean request must pass leak guard");
}

#[test]
fn every_denylist_pattern_is_rejected() {
    for pat in audit::REASONING_DENYLIST {
        let mut map = serde_json::Map::new();
        map.insert("task_id".to_string(), json!("T-1"));
        map.insert((*pat).to_string(), json!("anything"));
        let body = serde_json::Value::Object(map);
        let err = audit::audit(&body)
            .err()
            .unwrap_or_else(|| panic!("denylist pattern {pat:?} should trip the guard"));
        // The audit's `pattern` is the substring on the denylist; for a
        // key that IS the pattern, `offending_field` equals `pattern`.
        assert!(
            err.offending_field.contains(*pat) || err.offending_field == *pat,
            "expected offending_field to contain {pat:?}, got {:?}",
            err.offending_field
        );
        assert!(
            audit::REASONING_DENYLIST.contains(&err.pattern.as_str()),
            "{:?} not in denylist (matched against key {pat:?})",
            err.pattern,
        );
    }
}

#[test]
fn rejects_case_variations() {
    let body = json!({
        "task_id": "T-1",
        "Executor_Reasoning_Trace": "x",
    });
    let err = audit::audit(&body).expect_err("case-insensitive match");
    assert!(err.offending_field.contains("Executor_Reasoning_Trace"));
}

#[test]
fn rejects_nested_chain_of_thought_in_attestation_chain() {
    let body = json!({
        "task_id": "T-1",
        "attestation_chain": [
            { "rekor_uuid": "abc", "chain_of_thought": "leaky" }
        ],
    });
    let err = audit::audit(&body).expect_err("nested CoT must trip");
    assert!(err.offending_field.contains("chain_of_thought"));
}

#[test]
fn rejects_reasoning_path_in_diff() {
    let paths = vec!["src/agent_trace/leak.rs"];
    let err = audit::audit_paths(paths).expect_err("path pattern must trip");
    assert_eq!(err.pattern, "path-pattern");
}

#[test]
fn aggressive_substring_match_is_intentional() {
    // The denylist is a SUBSTRING match. We pin the aggressive
    // behaviour: a key like `"trajectory_id"` (literal "trajectory" on
    // the denylist) is rejected even though it is plausibly innocuous
    // telemetry. Future relaxation must be a conscious choice.
    let body = json!({ "trajectory_id": "abc" });
    let err = audit::audit(&body).expect_err("aggressive substring match");
    assert_eq!(err.pattern, "trajectory");
}

#[test]
fn report_paths_recursively() {
    let body = json!({
        "outer": {
            "middle": {
                "inner": {
                    "scratchpad": "secret"
                }
            }
        }
    });
    let err = audit::audit(&body).expect_err("nested scratchpad must trip");
    assert_eq!(err.offending_field, "outer.middle.inner.scratchpad");
    assert_eq!(err.pattern, "scratchpad");
}
