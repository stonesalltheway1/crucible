//! Crate-wide error type.

use thiserror::Error;

/// Crucible relay error. Implements `IntoResponse` via `server::error_response`.
#[derive(Debug, Error)]
pub enum Error {
    /// Configuration error (missing env var, bad URL, etc.).
    #[error("config: {0}")]
    Config(String),

    /// Predicate validation failure (e.g., missing required field, schema mismatch).
    #[error("predicate: {0}")]
    Predicate(String),

    /// DSSE / PAE / envelope construction error.
    #[error("dsse: {0}")]
    Dsse(String),

    /// Local signer (Ed25519) error.
    #[error("signer: {0}")]
    Signer(String),

    /// Fulcio OIDC + cert issuance error.
    #[error("fulcio: {0}")]
    Fulcio(String),

    /// Rekor publish or inclusion-proof error.
    #[error("rekor: {0}")]
    Rekor(String),

    /// Local journal IO / hash-chain error.
    #[error("journal: {0}")]
    Journal(String),

    /// Verification — signature, chain, or subject-digest mismatch.
    #[error("verify: {0}")]
    Verify(String),

    /// Self-approval rejected at envelope build time (T21).
    #[error("self-approval: agent_oidc_subject {agent} matches approver_oidc_subject {approver}")]
    SelfApproval { agent: String, approver: String },

    /// Stale approval: bundle hash differs from the hash the approval was signed against (T2).
    #[error("stale approval: bundle hash {bundle} differs from approval-bound {approval}")]
    StaleApproval { bundle: String, approval: String },

    /// HTTP / network IO.
    #[error("http: {0}")]
    Http(#[from] reqwest::Error),

    /// JSON serde.
    #[error("json: {0}")]
    Json(#[from] serde_json::Error),

    /// Lower-level IO.
    #[error("io: {0}")]
    Io(#[from] std::io::Error),

    /// Catch-all.
    #[error("other: {0}")]
    Other(String),
}

impl From<base64::DecodeError> for Error {
    fn from(e: base64::DecodeError) -> Self {
        Error::Dsse(format!("base64 decode: {e}"))
    }
}

impl From<hex::FromHexError> for Error {
    fn from(e: hex::FromHexError) -> Self {
        Error::Verify(format!("hex decode: {e}"))
    }
}

impl From<ed25519_dalek::SignatureError> for Error {
    fn from(e: ed25519_dalek::SignatureError) -> Self {
        Error::Signer(format!("ed25519: {e}"))
    }
}

/// Convenience alias used throughout the crate.
pub type Result<T> = std::result::Result<T, Error>;
