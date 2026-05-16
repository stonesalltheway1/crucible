//! Reasoning-leak audit guard.
//!
//! Mirrors `apps/verifier/internal/verification/verification.go` —
//! `AuditNoLeakage` rejects any payload whose JSON-key namespace
//! (recursively) contains a substring on the reasoning denylist. The
//! Crucible ADR-002 invariant is that the verifier never sees the
//! executor's reasoning trace; this audit is the runtime defence in
//! depth on top of the schema design.

use serde_json::Value;

/// Lower-case substring set the audit checks against every JSON-key in
/// the request payload. Intentionally aggressive — false-positives are
/// far cheaper than a leaked reasoning trace.
pub const REASONING_DENYLIST: &[&str] = &[
    "reasoning",
    "chain_of_thought",
    "chain-of-thought",
    "cot",
    "thinking_trace",
    "thinking-trace",
    "thoughts",
    "scratchpad",
    "internal_monologue",
    "hidden_state",
    "agent_trace",
    "executor_trace",
    "trajectory",
    "plan_critique",
    "reflection",
];

/// Path patterns that indicate a reasoning artefact has been smuggled
/// into the diff's file list. Mirrors the Go `isReasoningPath` helper.
pub const REASONING_PATH_PATTERNS: &[&str] = &[
    ".reasoning.",
    "/reasoning/",
    ".cot.",
    "/cot/",
    "_thinking_",
    "_scratchpad_",
    "agent_trace",
    "executor_trace",
];

/// The error returned when the audit finds a suspicious field or path.
#[derive(Debug, Clone)]
pub struct LeakageError {
    /// Dotted path to the offending field.
    pub offending_field: String,
    /// Denylist pattern that matched.
    pub pattern: String,
}

impl std::fmt::Display for LeakageError {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(
            f,
            "executor-reasoning leak detected — field {:?} matched pattern {:?} (ADR-002 invariant)",
            self.offending_field, self.pattern
        )
    }
}

impl std::error::Error for LeakageError {}

/// Recursively audit a `serde_json::Value`. Keys are checked
/// case-insensitively against the denylist. Arrays of objects are
/// recursed into; scalar leaves are ignored (only keys carry semantics).
pub fn audit(value: &Value) -> Result<(), LeakageError> {
    audit_inner(value, "")
}

fn audit_inner(value: &Value, prefix: &str) -> Result<(), LeakageError> {
    match value {
        Value::Object(map) => {
            // Iterate in sorted order so the offending-field path is
            // deterministic — matches the Go implementation.
            let mut keys: Vec<&String> = map.keys().collect();
            keys.sort();
            for key in keys {
                let full = if prefix.is_empty() {
                    key.clone()
                } else {
                    format!("{prefix}.{key}")
                };
                let lk = key.to_lowercase();
                for deny in REASONING_DENYLIST {
                    if lk.contains(deny) {
                        return Err(LeakageError {
                            offending_field: full,
                            pattern: (*deny).to_string(),
                        });
                    }
                }
                audit_inner(&map[key], &full)?;
            }
            Ok(())
        }
        Value::Array(arr) => {
            for (i, e) in arr.iter().enumerate() {
                let full = format!("{prefix}[{i}]");
                audit_inner(e, &full)?;
            }
            Ok(())
        }
        // Scalars never carry reasoning leakage at the key-name level.
        _ => Ok(()),
    }
}

/// Reject `diff.files[].path` entries that look like reasoning artefacts.
pub fn audit_paths<I, S>(paths: I) -> Result<(), LeakageError>
where
    I: IntoIterator<Item = S>,
    S: AsRef<str>,
{
    for p in paths {
        let pl = p.as_ref().to_lowercase();
        for pat in REASONING_PATH_PATTERNS {
            if pl.contains(pat) {
                return Err(LeakageError {
                    offending_field: format!("diff.files.{}", p.as_ref()),
                    pattern: "path-pattern".to_string(),
                });
            }
        }
    }
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;
    use serde_json::json;

    #[test]
    fn clean_payload_passes() {
        let v = json!({
            "task_id": "T-1",
            "diff": { "files": [ { "path": "src/lib.rs" } ] },
        });
        audit(&v).expect("clean payload must pass audit");
    }

    #[test]
    fn reasoning_field_at_top_level_fails() {
        let v = json!({
            "task_id": "T-1",
            "executor_reasoning": "I think therefore I am.",
        });
        let err = audit(&v).expect_err("denylist hit must reject");
        assert_eq!(err.offending_field, "executor_reasoning");
    }

    #[test]
    fn nested_chain_of_thought_fails() {
        let v = json!({
            "diff": { "files": [], "chain_of_thought": "x" },
        });
        let err = audit(&v).expect_err("nested denylist hit must reject");
        assert!(err.offending_field.contains("chain_of_thought"));
    }

    #[test]
    fn array_recursion_finds_scratchpad() {
        let v = json!({
            "spec_changes": [ { "path": "spec.yaml", "scratchpad": "..." } ],
        });
        let err = audit(&v).expect_err("array element denylist hit");
        assert!(err.offending_field.starts_with("spec_changes[0]"));
    }

    #[test]
    fn reasoning_path_blocked() {
        let err = audit_paths(["src/agent_trace/log.rs"]).expect_err("path must be blocked");
        assert_eq!(err.pattern, "path-pattern");
    }
}
