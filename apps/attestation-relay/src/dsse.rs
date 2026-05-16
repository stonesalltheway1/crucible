//! DSSE (Dead Simple Signing Envelope) v1.
//!
//! Per <https://github.com/secure-systems-lab/dsse/blob/master/protocol.md>:
//!
//! ```text
//!   PAE = "DSSEv1 " || PAYLOAD_TYPE_LENGTH || " " || PAYLOAD_TYPE ||
//!         " " || PAYLOAD_LENGTH || " " || PAYLOAD
//! ```
//!
//! `Signer.sign(PAE)` → `Envelope.signatures[i].sig`. Verifiers recompute the
//! PAE from the envelope's `payload` + `payloadType` and check the signature.

use base64::Engine as _;
use serde::{Deserialize, Serialize};

use crate::error::{Error, Result};

/// DSSE payload type for in-toto v1.
pub const DSSE_PAYLOAD_TYPE_IN_TOTO_V1: &str = "application/vnd.in-toto+json";

/// Detached signature inside a DSSE envelope.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DsseSignature {
    /// Key ID (Ed25519 fingerprint, Fulcio cert SKI, …).
    #[serde(default)]
    pub keyid: String,
    /// Base64-encoded signature over the PAE.
    pub sig: String,
    /// Base64-encoded x509 cert (Fulcio-issued); empty for keyed local mode.
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub cert: Option<String>,
}

/// The DSSE envelope.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DsseEnvelope {
    /// MUST be `application/vnd.in-toto+json`.
    #[serde(rename = "payloadType")]
    pub payload_type: String,
    /// Base64-encoded payload bytes.
    pub payload: String,
    /// Signature set (multi-sig supported).
    pub signatures: Vec<DsseSignature>,
}

impl DsseEnvelope {
    /// Build an envelope around the given payload bytes; signatures are
    /// added by the signer in `crate::signer`.
    pub fn new(payload: &[u8]) -> Self {
        Self {
            payload_type: DSSE_PAYLOAD_TYPE_IN_TOTO_V1.into(),
            payload: base64::engine::general_purpose::STANDARD.encode(payload),
            signatures: vec![],
        }
    }

    /// Decode the base64 payload.
    pub fn payload_bytes(&self) -> Result<Vec<u8>> {
        Ok(base64::engine::general_purpose::STANDARD.decode(&self.payload)?)
    }

    /// Compute the DSSE PAE for verification.
    pub fn pae(&self) -> Result<Vec<u8>> {
        let payload = self.payload_bytes()?;
        Ok(pae(&self.payload_type, &payload))
    }

    /// Validate basic shape: non-empty `payloadType`, non-empty payload,
    /// at least one signature.
    pub fn validate_shape(&self) -> Result<()> {
        if self.payload_type.is_empty() {
            return Err(Error::Dsse("payloadType empty".into()));
        }
        if self.payload.is_empty() {
            return Err(Error::Dsse("payload empty".into()));
        }
        if self.signatures.is_empty() {
            return Err(Error::Dsse("envelope has no signatures".into()));
        }
        Ok(())
    }
}

/// DSSE Pre-Authentication Encoding.
///
/// Returns `DSSEv1 <len> <type> <len> <payload>` as bytes.
#[must_use]
pub fn pae(payload_type: &str, payload: &[u8]) -> Vec<u8> {
    let mut out = Vec::with_capacity(64 + payload.len());
    out.extend_from_slice(b"DSSEv1 ");
    out.extend_from_slice(payload_type.len().to_string().as_bytes());
    out.push(b' ');
    out.extend_from_slice(payload_type.as_bytes());
    out.push(b' ');
    out.extend_from_slice(payload.len().to_string().as_bytes());
    out.push(b' ');
    out.extend_from_slice(payload);
    out
}
