//! in-toto Statement v1.
//!
//! Statement shape:
//!
//! ```json
//! {
//!   "_type":         "https://in-toto.io/Statement/v1",
//!   "subject":       [{ "name": "...", "digest": {"sha256": "..."} }],
//!   "predicateType": "https://crucible.dev/<type>/v1",
//!   "predicate":     { ... }
//! }
//! ```

use serde::{Deserialize, Serialize};
use sha2::{Digest, Sha256};
use std::collections::BTreeMap;

use crate::error::{Error, Result};

/// in-toto Statement v1 type URI.
pub const IN_TOTO_STATEMENT_V1: &str = "https://in-toto.io/Statement/v1";

/// One subject in an in-toto Statement.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct StatementSubject {
    /// Subject name — typically a file path, hash, or task identifier.
    pub name: String,
    /// `{"sha256": "<hex>"}` (other algorithms permitted by the spec but
    /// unused by Crucible).
    pub digest: BTreeMap<String, String>,
}

/// The full in-toto Statement v1.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct InTotoStatement {
    /// MUST be `https://in-toto.io/Statement/v1`.
    #[serde(rename = "_type")]
    pub stmt_type: String,
    /// Non-empty.
    pub subject: Vec<StatementSubject>,
    /// Predicate-type URI.
    #[serde(rename = "predicateType")]
    pub predicate_type: String,
    /// Typed payload as `serde_json::Value`.
    pub predicate: serde_json::Value,
}

impl InTotoStatement {
    /// Builds a Statement around a single subject + predicate.
    pub fn new(
        subject_name: impl Into<String>,
        subject_digest_hex: impl Into<String>,
        predicate_type: impl Into<String>,
        predicate: serde_json::Value,
    ) -> Self {
        let mut digest = BTreeMap::new();
        digest.insert("sha256".into(), subject_digest_hex.into());
        Self {
            stmt_type: IN_TOTO_STATEMENT_V1.into(),
            subject: vec![StatementSubject {
                name: subject_name.into(),
                digest,
            }],
            predicate_type: predicate_type.into(),
            predicate,
        }
    }

    /// Asserts subject + predicate type are non-empty + the digest is a
    /// 64-hex sha256.
    pub fn validate(&self) -> Result<()> {
        if self.stmt_type != IN_TOTO_STATEMENT_V1 {
            return Err(Error::Predicate(format!(
                "_type {} not {}",
                self.stmt_type, IN_TOTO_STATEMENT_V1
            )));
        }
        if self.subject.is_empty() {
            return Err(Error::Predicate("statement has no subject".into()));
        }
        for s in &self.subject {
            if s.name.is_empty() {
                return Err(Error::Predicate("subject name empty".into()));
            }
            let h = s
                .digest
                .get("sha256")
                .ok_or_else(|| Error::Predicate("subject digest missing sha256".into()))?;
            if h.len() != 64 || !h.chars().all(|c| c.is_ascii_hexdigit()) {
                return Err(Error::Predicate(format!("bad sha256 digest: {h:?}")));
            }
        }
        if self.predicate_type.is_empty() {
            return Err(Error::Predicate("predicateType empty".into()));
        }
        Ok(())
    }

    /// Serializes to canonical JSON (key-ordered).
    pub fn to_canonical_json(&self) -> Result<Vec<u8>> {
        Ok(serde_json::to_vec(self)?)
    }
}

/// SHA-256 of arbitrary content as hex.
#[must_use]
pub fn sha256_hex(content: &[u8]) -> String {
    let mut h = Sha256::new();
    h.update(content);
    hex::encode(h.finalize())
}
