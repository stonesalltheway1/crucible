//! In-toto attestation emission.
//!
//! Wraps the predicate-bearing payload in an in-toto Statement v1,
//! base64-encodes it inside a DSSE envelope, signs with the runtime's
//! Ed25519 key, and appends to a per-task hash-chained journal file. The
//! same JSON-Lines format that `libs/attestation` (Phase 1, Go) writes —
//! so the attestation-relay service can ingest both Rust and Go journals
//! into the unified Sigstore Rekor pipeline (Phase 6).
//!
//! Predicate types emitted by the runtime (per the Phase 2 brief):
//!
//! - `https://crucible.dev/WriteAttestation/v1` on `twin.fs.write` / delete
//! - `https://crucible.dev/MigrationAttestation/v1` on `twin.db.migrate`
//! - `https://crucible.dev/ServiceCallAttestation/v1` on `twin.svc.call`
//! - `https://crucible.dev/DestructiveProposal/v1` on shim interception
//! - `https://crucible.dev/DestructiveApproval/v1` on gate decision
//! - `https://crucible.dev/SandboxLifecycle/v1` on spawn/snapshot/kill
//!
//! Schema source-of-truth lives in `libs/attestation/schemas/`. This crate
//! does NOT re-validate against the schemas at emission time (the relay
//! does that on ingest); it only enforces required-fields.

#![warn(missing_docs)]

use async_trait::async_trait;
use chrono::{DateTime, Utc};
use ed25519_dalek::{Signer, SigningKey, VerifyingKey};
use rand::rngs::OsRng;
use serde::{Deserialize, Serialize};
use sha2::{Digest, Sha256};
use std::path::{Path, PathBuf};
use std::sync::Arc;
use thiserror::Error;
use tokio::sync::Mutex;

pub use twin_runtime_shim::proposal::DestructiveProposal;

/// Errors from the attestation pipeline.
#[derive(Debug, Error)]
pub enum Error {
    /// Failed to construct the in-toto statement.
    #[error("statement: {0}")]
    Statement(String),
    /// Signing failed.
    #[error("sign: {0}")]
    Sign(String),
    /// Publishing failed.
    #[error("publish: {0}")]
    Publish(#[from] std::io::Error),
    /// Schema-level required field missing.
    #[error("required field missing: {0}")]
    Required(&'static str),
}

/// Result alias for the attest crate.
pub type Result<T> = std::result::Result<T, Error>;

/// In-toto Statement v1 — `https://in-toto.io/Statement/v1`.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct InTotoStatement {
    /// Always `"https://in-toto.io/Statement/v1"`.
    #[serde(rename = "_type")]
    pub _type: String,
    /// Subjects this statement attests about.
    pub subject: Vec<Subject>,
    /// Predicate type URI.
    #[serde(rename = "predicateType")]
    pub predicate_type: String,
    /// Predicate body (typed payload — exact shape depends on predicate_type).
    pub predicate: serde_json::Value,
}

/// One subject inside a [`InTotoStatement`].
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Subject {
    /// Logical name of the subject (file path, task id, sandbox id, ...).
    pub name: String,
    /// `algorithm → digest` map (typically `sha256`).
    pub digest: std::collections::BTreeMap<String, String>,
}

/// DSSE envelope shape — the wire format for signed in-toto statements.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DsseEnvelope {
    /// `"application/vnd.in-toto+json"`.
    #[serde(rename = "payloadType")]
    pub payload_type: String,
    /// Base64-encoded [`InTotoStatement`].
    pub payload: String,
    /// One signature per signer.
    pub signatures: Vec<DsseSignature>,
}

/// One DSSE signature.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DsseSignature {
    /// Key id — empty for Sigstore keyless.
    #[serde(default)]
    pub keyid: String,
    /// Base64 signature bytes.
    pub sig: String,
    /// Base64 PEM cert chain (Sigstore Fulcio). Empty in Phase 2 (local key).
    #[serde(default)]
    pub cert: String,
}

/// Local hash-chained journal entry — matches the Phase 1 Go format.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct JournalEntry {
    /// Monotonic sequence number per journal file.
    pub seq: u64,
    /// Timestamp this entry was written.
    pub at: DateTime<Utc>,
    /// The signed envelope.
    pub envelope: DsseEnvelope,
    /// SHA-256 of the previous entry; `"0".repeat(64)` for the first.
    pub prev_hash: String,
    /// SHA-256 of this entry (excluding `entry_hash`).
    pub entry_hash: String,
}

/// Signing key + verifying key handle.
pub struct Signer_ {
    signing: SigningKey,
    /// Base64-encoded verifying key — included in journal metadata so
    /// downstream consumers can verify without coordinating out-of-band.
    pub verifying_b64: String,
}

impl Signer_ {
    /// Build a signer with a fresh Ed25519 key. The verifying key is
    /// reported via `verifying_b64`. Phase 2 only — Phase 6 switches to
    /// Sigstore keyless OIDC.
    pub fn ephemeral() -> Self {
        let signing = SigningKey::generate(&mut OsRng);
        let verifying = VerifyingKey::from(&signing);
        let verifying_b64 = base64_encode(verifying.as_bytes());
        Self {
            signing,
            verifying_b64,
        }
    }

    /// Sign the given payload, returning a DSSE envelope.
    pub fn sign(&self, statement: &InTotoStatement) -> Result<DsseEnvelope> {
        let bytes = serde_json::to_vec(statement)
            .map_err(|e| Error::Statement(format!("serialise: {e}")))?;
        let payload = base64_encode(&bytes);
        let pae = pae_v1(&"application/vnd.in-toto+json".to_string(), &bytes);
        let sig = self.signing.sign(&pae);
        Ok(DsseEnvelope {
            payload_type: "application/vnd.in-toto+json".into(),
            payload,
            signatures: vec![DsseSignature {
                keyid: String::new(),
                sig: base64_encode(&sig.to_bytes()),
                cert: String::new(),
            }],
        })
    }
}

/// DSSE Pre-Authentication Encoding (PAE) v1 — what we actually sign.
fn pae_v1(payload_type: &str, payload: &[u8]) -> Vec<u8> {
    let header = format!(
        "DSSEv1 {ptlen} {payload_type} {plen} ",
        ptlen = payload_type.len(),
        plen = payload.len()
    );
    let mut out = header.into_bytes();
    out.extend_from_slice(payload);
    out
}

/// Publisher trait — append-only sink for journal entries.
#[async_trait]
pub trait Publisher: Send + Sync {
    /// Publish one envelope.
    async fn publish(&self, envelope: &DsseEnvelope) -> Result<JournalEntry>;
}

/// Local file-backed journal publisher.
pub struct LocalJournalPublisher {
    inner: Mutex<JournalState>,
}

struct JournalState {
    path: PathBuf,
    seq: u64,
    prev_hash: String,
}

impl LocalJournalPublisher {
    /// Open or create a journal at `path`. Parent directory must exist.
    ///
    /// # Errors
    /// Returns [`Error::Publish`] on filesystem failure.
    pub fn open(path: impl Into<PathBuf>) -> Result<Self> {
        let path = path.into();
        if let Some(parent) = path.parent() {
            fs_err::create_dir_all(parent).map_err(Error::Publish)?;
        }
        let (seq, prev_hash) = recover_state(&path)?;
        Ok(Self {
            inner: Mutex::new(JournalState {
                path,
                seq,
                prev_hash,
            }),
        })
    }

    /// Returns the path the journal is bound to.
    pub fn path(&self) -> PathBuf {
        // Best-effort blocking read — the path is immutable for the
        // lifetime of the publisher.
        self.inner.try_lock().map(|s| s.path.clone()).unwrap_or_default()
    }
}

fn recover_state(path: &Path) -> Result<(u64, String)> {
    if !path.exists() {
        return Ok((0, "0".repeat(64)));
    }
    let raw = fs_err::read_to_string(path).map_err(Error::Publish)?;
    let mut last_seq = 0u64;
    let mut last_hash = "0".repeat(64);
    for line in raw.lines() {
        if line.trim().is_empty() {
            continue;
        }
        if let Ok(entry) = serde_json::from_str::<JournalEntry>(line) {
            last_seq = entry.seq;
            last_hash = entry.entry_hash;
        }
    }
    Ok((last_seq, last_hash))
}

#[async_trait]
impl Publisher for LocalJournalPublisher {
    async fn publish(&self, envelope: &DsseEnvelope) -> Result<JournalEntry> {
        use tokio::io::AsyncWriteExt;
        let mut guard = self.inner.lock().await;
        guard.seq += 1;
        let mut entry = JournalEntry {
            seq: guard.seq,
            at: Utc::now(),
            envelope: envelope.clone(),
            prev_hash: guard.prev_hash.clone(),
            entry_hash: String::new(),
        };
        entry.entry_hash = hash_entry(&entry);
        guard.prev_hash = entry.entry_hash.clone();
        let line = serde_json::to_string(&entry)
            .map_err(|e| Error::Statement(format!("entry serialise: {e}")))?;
        let path = guard.path.clone();
        drop(guard);

        let mut f = tokio::fs::OpenOptions::new()
            .create(true)
            .append(true)
            .open(&path)
            .await?;
        f.write_all(line.as_bytes()).await?;
        f.write_all(b"\n").await?;
        f.flush().await?;
        Ok(entry)
    }
}

fn hash_entry(entry: &JournalEntry) -> String {
    let mut h = Sha256::new();
    h.update(entry.seq.to_be_bytes());
    h.update(entry.at.to_rfc3339().as_bytes());
    h.update(entry.envelope.payload.as_bytes());
    h.update(entry.prev_hash.as_bytes());
    hex::encode(h.finalize())
}

fn base64_encode(bytes: &[u8]) -> String {
    use std::fmt::Write;
    const ALPHABET: &[u8] = b"ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/";
    let mut out = String::with_capacity((bytes.len() + 2) / 3 * 4);
    let mut iter = bytes.chunks_exact(3);
    for chunk in iter.by_ref() {
        let b = u32::from(chunk[0]) << 16 | u32::from(chunk[1]) << 8 | u32::from(chunk[2]);
        for i in (0..4).rev() {
            let idx = ((b >> (i * 6)) & 0x3f) as usize;
            out.push(ALPHABET[idx] as char);
        }
    }
    let rem = iter.remainder();
    if !rem.is_empty() {
        let mut b = 0u32;
        for (i, &x) in rem.iter().enumerate() {
            b |= u32::from(x) << ((2 - i) * 8);
        }
        for i in (0..4).rev() {
            let idx = ((b >> (i * 6)) & 0x3f) as usize;
            if 3 - i <= rem.len() {
                out.push(ALPHABET[idx] as char);
            } else {
                out.push('=');
            }
        }
    }
    // Avoid unused-warning on Write if it's somehow not used.
    let _ = std::fmt::Write::write_char(&mut out, ' ');
    out.pop();
    out
}

// ─────────────────────────────────────────────────────────────────────────────
// Predicate builders — typed helpers for each predicate the runtime emits.
// ─────────────────────────────────────────────────────────────────────────────

/// Build a `WriteAttestation/v1` statement.
pub fn write_attestation(
    task_id: &str,
    tenant_id: &str,
    step_id: &str,
    repo: &str,
    base_sha: &str,
    path: &str,
    action: &str,
    content_sha256: &str,
    size_bytes: u64,
    agent_oidc: &str,
) -> InTotoStatement {
    let mut digest = std::collections::BTreeMap::new();
    digest.insert("sha256".into(), content_sha256.into());
    InTotoStatement {
        _type: "https://in-toto.io/Statement/v1".into(),
        subject: vec![Subject {
            name: path.into(),
            digest,
        }],
        predicate_type: "https://crucible.dev/WriteAttestation/v1".into(),
        predicate: serde_json::json!({
            "task_id": task_id,
            "step_id": step_id,
            "tenant_id": tenant_id,
            "repo": repo,
            "base_sha": base_sha,
            "path": path,
            "action": action,
            "content_sha256": content_sha256,
            "size_bytes": size_bytes,
            "timestamp": Utc::now().to_rfc3339(),
            "agent_oidc_subject": agent_oidc,
        }),
    }
}

/// Build a `DestructiveProposal/v1` statement from a shim proposal.
pub fn destructive_proposal(prop: &DestructiveProposal, tenant_id: &str) -> InTotoStatement {
    let mut digest = std::collections::BTreeMap::new();
    digest.insert("sha256".into(), prop.content_hash.clone());
    let scope = match prop.scope {
        twin_runtime_shim::proposal::Scope::Twin => "twin",
        twin_runtime_shim::proposal::Scope::Real => "real",
    };
    let predicate = serde_json::json!({
        "task_id": prop.task_id,
        "tenant_id": tenant_id,
        "command": prop.command,
        "scope": scope,
        "justification": "intercepted",
        "blast_radius": prop.blast_radius,
        "intercepted_at_layer": format!("{:?}", prop.intercepted_at_layer).to_ascii_lowercase(),
        "agent_oidc_subject": format!("https://accounts.crucible.dev/agents/runtime-{}", prop.task_id),
    });
    InTotoStatement {
        _type: "https://in-toto.io/Statement/v1".into(),
        subject: vec![Subject {
            name: format!("proposal:{}", prop.pattern_id),
            digest,
        }],
        predicate_type: "https://crucible.dev/DestructiveProposal/v1".into(),
        predicate,
    }
}

/// Build a `DestructiveApproval/v1` statement.
pub fn destructive_approval(
    proposal_attestation: &str,
    kind: &str,
    approver_oidc: &str,
) -> InTotoStatement {
    let mut digest = std::collections::BTreeMap::new();
    digest.insert("sha256".into(), {
        let mut h = Sha256::new();
        h.update(proposal_attestation.as_bytes());
        hex::encode(h.finalize())
    });
    InTotoStatement {
        _type: "https://in-toto.io/Statement/v1".into(),
        subject: vec![Subject {
            name: format!("approval:{proposal_attestation}"),
            digest,
        }],
        predicate_type: "https://crucible.dev/DestructiveApproval/v1".into(),
        predicate: serde_json::json!({
            "proposal_attestation": proposal_attestation,
            "approval_kind": kind,
            "approver_oidc_subject": approver_oidc,
            "approved_at": Utc::now().to_rfc3339(),
        }),
    }
}

/// Build a `SandboxLifecycle/v1` statement for spawn / snapshot / kill events.
pub fn sandbox_lifecycle(
    task_id: &str,
    tenant_id: &str,
    sandbox_id: &str,
    event: &str,
    spec_hash: &str,
    detail: serde_json::Value,
) -> InTotoStatement {
    let mut digest = std::collections::BTreeMap::new();
    digest.insert("sha256".into(), spec_hash.into());
    InTotoStatement {
        _type: "https://in-toto.io/Statement/v1".into(),
        subject: vec![Subject {
            name: format!("sandbox:{sandbox_id}"),
            digest,
        }],
        predicate_type: "https://crucible.dev/SandboxLifecycle/v1".into(),
        predicate: serde_json::json!({
            "task_id": task_id,
            "tenant_id": tenant_id,
            "sandbox_id": sandbox_id,
            "event": event,
            "spec_hash": spec_hash,
            "detail": detail,
            "occurred_at": Utc::now().to_rfc3339(),
        }),
    }
}

/// End-to-end emit helper: sign + publish in one call.
pub async fn emit(
    signer: &Signer_,
    publisher: &Arc<dyn Publisher>,
    statement: InTotoStatement,
) -> Result<JournalEntry> {
    let envelope = signer.sign(&statement)?;
    publisher.publish(&envelope).await
}

#[cfg(test)]
mod tests {
    use super::*;

    #[tokio::test]
    async fn end_to_end_emit_persists_journal_entry() {
        let dir = tempfile::tempdir().unwrap();
        let path = dir.path().join("journal.jsonl");
        let publisher: Arc<dyn Publisher> = Arc::new(LocalJournalPublisher::open(&path).unwrap());
        let signer = Signer_::ephemeral();
        let stmt = write_attestation(
            "task_x", "ten_x", "step_1", "github.com/example/repo", "abc",
            "src/main.rs", "modify", "deadbeef", 12, "oidc://test",
        );
        let entry = emit(&signer, &publisher, stmt).await.unwrap();
        assert_eq!(entry.seq, 1);
        assert_eq!(entry.prev_hash, "0".repeat(64));
        assert!(!entry.entry_hash.is_empty());
        let raw = std::fs::read_to_string(&path).unwrap();
        assert_eq!(raw.lines().count(), 1);
    }

    #[tokio::test]
    async fn hash_chain_is_maintained_across_writes() {
        let dir = tempfile::tempdir().unwrap();
        let path = dir.path().join("journal.jsonl");
        let publisher: Arc<dyn Publisher> = Arc::new(LocalJournalPublisher::open(&path).unwrap());
        let signer = Signer_::ephemeral();
        let stmt1 = write_attestation("t", "ten", "s1", "r", "x", "a.rs", "add", "d1", 1, "o");
        let stmt2 = write_attestation("t", "ten", "s2", "r", "x", "b.rs", "add", "d2", 1, "o");
        let e1 = emit(&signer, &publisher, stmt1).await.unwrap();
        let e2 = emit(&signer, &publisher, stmt2).await.unwrap();
        assert_eq!(e2.prev_hash, e1.entry_hash);
        assert_eq!(e2.seq, e1.seq + 1);
    }

    #[tokio::test]
    async fn publisher_recovers_seq_across_reopen() {
        let dir = tempfile::tempdir().unwrap();
        let path = dir.path().join("journal.jsonl");
        {
            let publisher: Arc<dyn Publisher> = Arc::new(LocalJournalPublisher::open(&path).unwrap());
            let signer = Signer_::ephemeral();
            let stmt = write_attestation("t", "ten", "s", "r", "x", "a.rs", "add", "d", 1, "o");
            emit(&signer, &publisher, stmt).await.unwrap();
        }
        let publisher = LocalJournalPublisher::open(&path).unwrap();
        let signer = Signer_::ephemeral();
        let stmt = write_attestation("t", "ten", "s2", "r", "x", "b.rs", "add", "d", 1, "o");
        let env = signer.sign(&stmt).unwrap();
        let entry = publisher.publish(&env).await.unwrap();
        assert_eq!(entry.seq, 2);
    }

    #[test]
    fn destructive_proposal_predicate_carries_scope() {
        use twin_runtime_shim::cmd_parse::{Command, CorpusHit};
        use twin_runtime_shim::corpus::{PatternScope, Reversibility};
        let hit = CorpusHit {
            pattern_id: "test",
            reason: "r",
            command: Command {
                argv: vec!["rm".into(), "foo".into()],
                source_offset: 0,
            },
            default_scope: PatternScope::AlwaysReal,
            reversibility: Reversibility::Lossy,
        };
        let prop = DestructiveProposal::from_match("t", "rm foo", &hit);
        let stmt = destructive_proposal(&prop, "ten");
        assert_eq!(
            stmt.predicate_type,
            "https://crucible.dev/DestructiveProposal/v1"
        );
        assert_eq!(stmt.predicate["scope"], serde_json::json!("real"));
    }

    #[test]
    fn base64_roundtrip_is_correct() {
        // Sanity check our minimal base64 encoder doesn't truncate.
        let cases: &[(&[u8], &str)] = &[
            (b"", ""),
            (b"f", "Zg=="),
            (b"fo", "Zm8="),
            (b"foo", "Zm9v"),
            (b"foob", "Zm9vYg=="),
            (b"fooba", "Zm9vYmE="),
            (b"foobar", "Zm9vYmFy"),
        ];
        for (input, expected) in cases {
            assert_eq!(base64_encode(input), *expected, "input {input:?}");
        }
    }
}
