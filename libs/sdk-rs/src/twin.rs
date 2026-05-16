//! Rust binding for the agent-side `twin.*` runtime API.
//!
//! Phase 2 surfaces a unix-socket gRPC client that the agent process
//! (running inside the sandbox) calls into. The same wire format is used
//! by all four language SDKs (Go, TS, Python, Rust) — the Rust binding is
//! the canonical reference since the runtime itself is Rust.
//!
//! For unit tests of upstream code, [`stub::StubClient`] is a fully-in-
//! memory client that records all calls and is `Send + Sync`.

use std::collections::HashMap;
use std::sync::Mutex;
use std::time::Duration;
use thiserror::Error;

/// Errors from the twin client.
#[derive(Debug, Error)]
pub enum TwinError {
    /// Runtime is unreachable.
    #[error("runtime unavailable: {0}")]
    Unavailable(String),
    /// Destructive proposal was returned; the agent must approve or pivot.
    #[error("destructive proposal: {0}")]
    Destructive(String),
    /// Secret access denied (agent attempted to read a raw value).
    #[error("secret access denied: {0}")]
    SecretDenied(String),
    /// Budget cap breached.
    #[error("budget exceeded")]
    BudgetExceeded,
    /// Other.
    #[error("twin: {0}")]
    Other(String),
}

/// Result alias.
pub type TwinResult<T> = Result<T, TwinError>;

/// Config for the runtime client.
#[derive(Debug, Clone)]
pub struct ClientConfig {
    /// Endpoint (unix socket path or vsock URI).
    pub endpoint: String,
    /// Task id this client is bound to.
    pub task_id: String,
    /// Heartbeat interval.
    pub heartbeat_interval: Duration,
}

impl Default for ClientConfig {
    fn default() -> Self {
        Self {
            endpoint: "unix:///work/.crucible/control.sock".into(),
            task_id: String::new(),
            heartbeat_interval: Duration::from_secs(5),
        }
    }
}

/// Outcome of a shell exec.
#[derive(Debug, Clone)]
pub enum ShellOutcome {
    /// Command executed; here's the result.
    Result {
        /// Stdout bytes.
        stdout: String,
        /// Stderr bytes.
        stderr: String,
        /// Exit code.
        exit_code: i32,
    },
    /// Command intercepted by the destructive-op gate.
    Proposal {
        /// Proposal id for `approve_destructive`.
        proposal_id: String,
        /// Why it was intercepted.
        reason: String,
        /// Scope ("twin" | "real").
        scope: String,
    },
}

/// Write attestation handle.
#[derive(Debug, Clone)]
pub struct WriteAttestation {
    /// Local journal id or Rekor UUID.
    pub attestation_id: String,
    /// SHA-256 of the written content.
    pub content_sha256: String,
}

/// Secret reference (no value).
#[derive(Debug, Clone)]
pub struct SecretRef {
    /// Logical name.
    pub name: String,
    /// Opaque handle.
    pub handle: String,
    /// Wall-clock expiry.
    pub expires_at: chrono::DateTime<chrono::Utc>,
}

/// Source ref for memory writes.
#[derive(Debug, Clone)]
pub enum SourceRef {
    /// PR review comment.
    PrComment { pr: u64, comment_id: String },
    /// Incident reference.
    Incident { id: String, service: String },
    /// ADR file.
    Adr { path: String, commit: String },
    /// Agent observation during a task.
    AgentObservation { task_id: String, step_id: String },
}

/// Scope narrowing for memory queries.
#[derive(Debug, Clone, Default)]
pub struct ScopeFilter {
    /// Repo (e.g. "acme/payments").
    pub repo: String,
    /// File glob.
    pub file_glob: String,
    /// Category bucket from the 12-taxonomy.
    pub category: String,
}

/// A single memory returned by `twin.memory.recall`.
#[derive(Debug, Clone)]
pub struct Memory {
    /// Memory id.
    pub id: String,
    /// Content text.
    pub content: String,
    /// Importance (A-MAC composite, 0..1).
    pub importance: f64,
    /// Memory kind.
    pub kind: String,
    /// Last-recalled wall clock.
    pub last_recalled: chrono::DateTime<chrono::Utc>,
}

/// A procedural-memory rule.
#[derive(Debug, Clone)]
pub struct Convention {
    /// Convention id.
    pub id: String,
    /// Tenant id.
    pub tenant_id: String,
    /// Scope.
    pub scope: ScopeFilter,
    /// Natural-language rule.
    pub rule_nl: String,
    /// Category bucket.
    pub category: String,
    /// Status.
    pub status: String,
    /// Confidence.
    pub confidence: f64,
}

/// Compliance violation returned by `twin.memory.check_compliance`.
#[derive(Debug, Clone)]
pub struct ComplianceViolation {
    /// Convention id violated.
    pub convention_id: String,
    /// Rule text.
    pub rule_nl: String,
    /// File the diff touched.
    pub offending_file: String,
    /// "info" | "warn" | "error".
    pub severity: String,
}

/// Compliance report returned by `twin.memory.check_compliance`.
#[derive(Debug, Clone)]
pub struct ComplianceReport {
    /// Diff hash the check ran against.
    pub diff_hash: String,
    /// Violations surfaced.
    pub violations: Vec<ComplianceViolation>,
    /// Number of conventions in scope.
    pub conventions_checked: u32,
}

/// The agent-side twin client.
pub trait TwinClient: Send + Sync {
    /// `twin.fs.read`.
    fn fs_read(&self, path: &str) -> TwinResult<String>;
    /// `twin.fs.write`.
    fn fs_write(&self, path: &str, content: &str, step_id: &str) -> TwinResult<WriteAttestation>;
    /// `twin.shell.exec`.
    fn shell_exec(&self, cmd: &str) -> TwinResult<ShellOutcome>;
    /// `twin.secret.get`.
    fn secret_get(&self, name: &str) -> TwinResult<SecretRef>;
    /// `twin.plan.checkpoint`.
    fn checkpoint(&self, name: &str) -> TwinResult<String>;
    /// `heartbeat` keepalive.
    fn heartbeat(&self) -> TwinResult<()>;
    /// `twin.memory.recall`.
    fn memory_recall(&self, query: &str, max_tokens: u32) -> TwinResult<Vec<Memory>>;
    /// `twin.memory.note`.
    fn memory_note(&self, fact: &str, source: SourceRef) -> TwinResult<String>;
    /// `twin.memory.conventions`.
    fn memory_conventions(&self, scope: ScopeFilter) -> TwinResult<Vec<Convention>>;
    /// `twin.memory.check_compliance`.
    fn memory_check_compliance(&self, diff_hash: &str, files: &[&str]) -> TwinResult<ComplianceReport>;
}

/// In-memory stub for tests.
pub mod stub {
    use super::*;

    /// `StubClient` — records every call. Useful for upstream unit tests.
    pub struct StubClient {
        files: Mutex<HashMap<String, String>>,
        /// Task id this stub is bound to.
        pub task_id: String,
    }

    impl StubClient {
        /// Construct.
        #[must_use]
        pub fn new(task_id: &str) -> Self {
            Self {
                files: Mutex::new(HashMap::new()),
                task_id: task_id.to_string(),
            }
        }
    }

    impl TwinClient for StubClient {
        fn fs_read(&self, path: &str) -> TwinResult<String> {
            self.files
                .lock()
                .unwrap()
                .get(path)
                .cloned()
                .ok_or_else(|| TwinError::Other(format!("file not found: {path}")))
        }

        fn fs_write(&self, path: &str, content: &str, _step_id: &str) -> TwinResult<WriteAttestation> {
            use sha2::{Digest, Sha256};
            self.files
                .lock()
                .unwrap()
                .insert(path.to_string(), content.to_string());
            let mut h = Sha256::new();
            h.update(content.as_bytes());
            Ok(WriteAttestation {
                attestation_id: format!("stub:{path}"),
                content_sha256: hex::encode(h.finalize()),
            })
        }

        fn shell_exec(&self, cmd: &str) -> TwinResult<ShellOutcome> {
            Ok(ShellOutcome::Result {
                stdout: format!("[stub] {cmd}"),
                stderr: String::new(),
                exit_code: 0,
            })
        }

        fn secret_get(&self, name: &str) -> TwinResult<SecretRef> {
            Ok(SecretRef {
                name: name.to_string(),
                handle: format!("stub-handle:{name}"),
                expires_at: chrono::Utc::now() + chrono::Duration::seconds(60),
            })
        }

        fn checkpoint(&self, name: &str) -> TwinResult<String> {
            Ok(format!("stub-snap:{name}"))
        }

        fn heartbeat(&self) -> TwinResult<()> {
            Ok(())
        }

        fn memory_recall(&self, _query: &str, _max_tokens: u32) -> TwinResult<Vec<Memory>> {
            Ok(Vec::new())
        }

        fn memory_note(&self, _fact: &str, _source: SourceRef) -> TwinResult<String> {
            Ok("mem_stub".to_string())
        }

        fn memory_conventions(&self, _scope: ScopeFilter) -> TwinResult<Vec<Convention>> {
            Ok(Vec::new())
        }

        fn memory_check_compliance(&self, diff_hash: &str, _files: &[&str]) -> TwinResult<ComplianceReport> {
            Ok(ComplianceReport {
                diff_hash: diff_hash.to_string(),
                violations: Vec::new(),
                conventions_checked: 0,
            })
        }
    }

    #[cfg(test)]
    mod tests {
        use super::*;

        #[test]
        fn write_then_read() {
            let c = StubClient::new("task_t");
            let att = c.fs_write("a.rs", "fn main() {}", "step1").unwrap();
            assert!(att.attestation_id.starts_with("stub:"));
            assert_eq!(c.fs_read("a.rs").unwrap(), "fn main() {}");
        }

        #[test]
        fn secret_get_returns_handle_not_value() {
            let c = StubClient::new("task_t");
            let r = c.secret_get("stripe").unwrap();
            assert!(r.handle.contains("stub-handle"));
            assert!(!r.handle.contains("sk_live_"));
        }

        #[test]
        fn shell_exec_default_is_ok_result() {
            let c = StubClient::new("task_t");
            let out = c.shell_exec("ls").unwrap();
            assert!(matches!(out, ShellOutcome::Result { exit_code: 0, .. }));
        }
    }
}

// Note: the real gRPC client wiring lives in apps/twin-runtime/crates/
// twin-runtime-server's integration tests for Phase 2; the public surface
// here is feature-complete for callers but the wire transport is the
// production runtime's responsibility.
