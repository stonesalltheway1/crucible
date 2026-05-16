//! Sigstore Fulcio v2 client.
//!
//! Fulcio is the OIDC-bound certificate-authority for Sigstore keyless
//! signing. We POST `(oidc_token, public_key, proof_of_possession)`; Fulcio
//! returns an x509 cert chain bound to the OIDC subject.
//!
//! The relay supports two modes:
//!
//! - **Public Sigstore** — `https://fulcio.sigstore.dev`.
//! - **Self-hosted** — customer's own Fulcio instance, configured via
//!   `CRUCIBLE_FULCIO_URL`. The CA root used by downstream verifiers is
//!   bundled with the air-gap installer (see `infra/air-gap-bundle/`).
//!
//! Phase-6 ships a thin client + a `MockFulcio` that the relay's tests use
//! to exercise the keyless flow without network.

use async_trait::async_trait;
use base64::Engine as _;
use ed25519_dalek::VerifyingKey;
use serde::{Deserialize, Serialize};

use crate::error::{Error, Result};

/// Fulcio client trait — implementations: `FulcioHttpClient` (real),
/// `MockFulcio` (tests), `NullFulcio` (offline / dev).
#[async_trait]
pub trait FulcioInner: Send + Sync + std::fmt::Debug {
    /// Exchange (OIDC token, public key, PoP) for an x509 cert chain.
    async fn sign_cert_chain(
        &self,
        oidc_token: &str,
        verifying_key: &VerifyingKey,
        proof_of_possession: &[u8],
    ) -> Result<Vec<u8>>;
}

/// A handle to a Fulcio implementation.
#[derive(Debug, Clone)]
pub struct FulcioClient {
    inner: std::sync::Arc<dyn FulcioInner>,
}

impl FulcioClient {
    /// Construct a `FulcioClient` from any inner.
    #[must_use]
    pub fn new(inner: std::sync::Arc<dyn FulcioInner>) -> Self {
        Self { inner }
    }

    /// Calls through.
    pub async fn sign_cert_chain(
        &self,
        oidc_token: &str,
        verifying_key: &VerifyingKey,
        proof_of_possession: &[u8],
    ) -> Result<Vec<u8>> {
        self.inner
            .sign_cert_chain(oidc_token, verifying_key, proof_of_possession)
            .await
    }
}

// ── HTTP client ────────────────────────────────────────────────────────────

/// Real HTTP client against a Fulcio v2 instance.
#[derive(Debug, Clone)]
pub struct FulcioHttpClient {
    base_url: String,
    http: reqwest::Client,
}

impl FulcioHttpClient {
    /// Build a client.
    pub fn new(base_url: impl Into<String>) -> Self {
        Self {
            base_url: base_url.into(),
            http: reqwest::Client::new(),
        }
    }
}

/// Fulcio request body.
#[derive(Debug, Serialize)]
struct FulcioSigningRequest {
    #[serde(rename = "publicKey")]
    public_key: PublicKey,
    #[serde(rename = "signedEmailAddress")]
    signed_email_address: String,
}

#[derive(Debug, Serialize)]
struct PublicKey {
    algorithm: String,
    content: String,
}

/// Fulcio response shape (subset).
#[derive(Debug, Deserialize)]
struct FulcioSigningResponse {
    #[serde(rename = "signedCertificateEmbeddedSct", default)]
    signed_cert_embedded_sct: Option<CertChain>,
    #[serde(rename = "signedCertificateDetachedSct", default)]
    signed_cert_detached_sct: Option<CertChain>,
}

#[derive(Debug, Deserialize)]
struct CertChain {
    chain: ChainPem,
}

#[derive(Debug, Deserialize)]
struct ChainPem {
    certificates: Vec<String>,
}

#[async_trait]
impl FulcioInner for FulcioHttpClient {
    async fn sign_cert_chain(
        &self,
        oidc_token: &str,
        verifying_key: &VerifyingKey,
        proof_of_possession: &[u8],
    ) -> Result<Vec<u8>> {
        let body = FulcioSigningRequest {
            public_key: PublicKey {
                algorithm: "ed25519".into(),
                content: base64::engine::general_purpose::STANDARD.encode(verifying_key.as_bytes()),
            },
            signed_email_address: base64::engine::general_purpose::STANDARD.encode(proof_of_possession),
        };
        let url = format!("{}/api/v2/signingCert", self.base_url.trim_end_matches('/'));
        let resp = self
            .http
            .post(&url)
            .bearer_auth(oidc_token)
            .header("Content-Type", "application/json")
            .json(&body)
            .send()
            .await?;
        if !resp.status().is_success() {
            let status = resp.status();
            let txt = resp.text().await.unwrap_or_default();
            return Err(Error::Fulcio(format!("status={status} body={txt}")));
        }
        let parsed: FulcioSigningResponse = resp.json().await?;
        let chain = parsed
            .signed_cert_embedded_sct
            .or(parsed.signed_cert_detached_sct)
            .ok_or_else(|| Error::Fulcio("Fulcio response missing cert chain".into()))?;
        let mut concatenated = Vec::new();
        for pem in &chain.chain.certificates {
            concatenated.extend_from_slice(pem.as_bytes());
            if !pem.ends_with('\n') {
                concatenated.push(b'\n');
            }
        }
        Ok(concatenated)
    }
}

// ── Mock + Null ────────────────────────────────────────────────────────────

/// In-memory mock Fulcio for tests.
#[derive(Debug, Default)]
pub struct MockFulcio {
    /// Captured invocations.
    pub captures: std::sync::Mutex<Vec<MockFulcioCall>>,
    /// Optional canned cert chain (defaults to a deterministic placeholder).
    pub canned_cert: Vec<u8>,
}

/// A captured request.
#[derive(Debug, Clone)]
pub struct MockFulcioCall {
    /// OIDC token presented.
    pub oidc_token: String,
    /// Verifying-key bytes.
    pub verifying_key: Vec<u8>,
    /// Proof-of-possession bytes.
    pub pop: Vec<u8>,
}

impl MockFulcio {
    /// Build a default mock that returns a synthetic cert chain.
    #[must_use]
    pub fn new() -> Self {
        Self {
            captures: std::sync::Mutex::new(vec![]),
            canned_cert: b"-----BEGIN CERTIFICATE-----\nMOCK-FULCIO\n-----END CERTIFICATE-----\n".to_vec(),
        }
    }
}

#[async_trait]
impl FulcioInner for MockFulcio {
    async fn sign_cert_chain(
        &self,
        oidc_token: &str,
        verifying_key: &VerifyingKey,
        proof_of_possession: &[u8],
    ) -> Result<Vec<u8>> {
        let mut g = self.captures.lock().unwrap();
        g.push(MockFulcioCall {
            oidc_token: oidc_token.into(),
            verifying_key: verifying_key.as_bytes().to_vec(),
            pop: proof_of_possession.to_vec(),
        });
        Ok(self.canned_cert.clone())
    }
}

/// Null Fulcio — refuses to issue. Used when the relay is in offline mode.
#[derive(Debug, Default)]
pub struct NullFulcio;

#[async_trait]
impl FulcioInner for NullFulcio {
    async fn sign_cert_chain(
        &self,
        _oidc_token: &str,
        _verifying_key: &VerifyingKey,
        _proof_of_possession: &[u8],
    ) -> Result<Vec<u8>> {
        Err(Error::Fulcio("relay in offline mode — Fulcio unavailable".into()))
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use ed25519_dalek::SigningKey;

    #[tokio::test]
    async fn mock_fulcio_captures() {
        let mock = std::sync::Arc::new(MockFulcio::new());
        let client = FulcioClient::new(mock.clone());
        let sk = SigningKey::generate(&mut rand::rngs::OsRng);
        let chain = client
            .sign_cert_chain("dev-token", &sk.verifying_key(), b"pop")
            .await
            .unwrap();
        assert!(!chain.is_empty());
        assert_eq!(mock.captures.lock().unwrap().len(), 1);
    }

    #[tokio::test]
    async fn null_fulcio_refuses() {
        let null = std::sync::Arc::new(NullFulcio);
        let client = FulcioClient::new(null);
        let sk = SigningKey::generate(&mut rand::rngs::OsRng);
        let err = client.sign_cert_chain("t", &sk.verifying_key(), b"p").await.unwrap_err();
        assert!(matches!(err, Error::Fulcio(_)));
    }
}
