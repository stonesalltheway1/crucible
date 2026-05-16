//! Signers for DSSE envelopes.
//!
//! Two implementations:
//!
//! - `LocalEd25519Signer` — generates / loads a keypair from disk; used in
//!   dev + offline mode. Identical envelope shape to the keyless path, only
//!   `cert` is empty.
//! - `SigstoreKeylessSigner` — exchanges an OIDC token for a short-lived
//!   x509 cert from Fulcio, then signs with the ephemeral key. The cert
//!   chain is embedded in `DsseSignature.cert`.

use async_trait::async_trait;
use base64::Engine as _;
use ed25519_dalek::{Signer as _, SigningKey, VerifyingKey};
use std::path::{Path, PathBuf};

use crate::dsse::{DsseEnvelope, DsseSignature};
use crate::error::{Error, Result};
use crate::fulcio::FulcioClient;

/// Abstraction over DSSE signers.
#[async_trait]
pub trait Signer: Send + Sync + std::fmt::Debug {
    /// Sign an in-toto Statement (serialized canonically); wraps the result
    /// in a DSSE envelope.
    async fn sign_envelope(&self, payload: &[u8]) -> Result<DsseEnvelope>;

    /// OIDC subject baked into the signer's cert (or the synthetic local
    /// subject for `LocalEd25519Signer`).
    fn oidc_subject(&self) -> &str;

    /// Short key id.
    fn key_id(&self) -> &str;
}

// ── Local Ed25519 signer ───────────────────────────────────────────────────

/// Dev-mode signer backed by a static Ed25519 keypair on disk.
#[derive(Debug, Clone)]
pub struct LocalEd25519Signer {
    signing_key: SigningKey,
    verifying_key: VerifyingKey,
    key_id: String,
    oidc_subject: String,
}

impl LocalEd25519Signer {
    /// Loads or creates an Ed25519 keypair under `dir`. On first call we
    /// generate fresh bytes; subsequent calls re-load.
    pub fn load_or_create(dir: &Path) -> Result<Self> {
        std::fs::create_dir_all(dir)?;
        let priv_path = dir.join("relay.ed25519");
        let pub_path = dir.join("relay.ed25519.pub");
        let signing_key = if priv_path.exists() {
            let bytes = std::fs::read(&priv_path)?;
            if bytes.len() != ed25519_dalek::SECRET_KEY_LENGTH {
                return Err(Error::Signer(format!(
                    "corrupt key (size {}, want {})",
                    bytes.len(),
                    ed25519_dalek::SECRET_KEY_LENGTH
                )));
            }
            let mut secret = [0u8; ed25519_dalek::SECRET_KEY_LENGTH];
            secret.copy_from_slice(&bytes);
            SigningKey::from_bytes(&secret)
        } else {
            use rand::rngs::OsRng;
            let mut csprng = OsRng;
            let sk = SigningKey::generate(&mut csprng);
            std::fs::write(&priv_path, sk.as_bytes())?;
            std::fs::write(&pub_path, sk.verifying_key().as_bytes())?;
            sk
        };
        let verifying_key = signing_key.verifying_key();
        let key_id = key_id_from(&verifying_key);
        let oidc_subject = format!("https://accounts.crucible.dev/relay/local/{key_id}");
        Ok(Self {
            signing_key,
            verifying_key,
            key_id,
            oidc_subject,
        })
    }

    /// Returns the verifying key. The promotion-gate's bundle_validator
    /// uses this for offline-mode signature verification.
    #[must_use]
    pub fn verifying_key(&self) -> VerifyingKey {
        self.verifying_key
    }

    /// Build a signer rooted at the directory above and return its path.
    pub fn dir_for(custom: Option<PathBuf>) -> PathBuf {
        custom.unwrap_or_else(|| {
            #[cfg(unix)]
            {
                let home = std::env::var_os("HOME").map(PathBuf::from).unwrap_or_default();
                home.join(".crucible/relay-keys")
            }
            #[cfg(not(unix))]
            {
                std::env::temp_dir().join("crucible-relay-keys")
            }
        })
    }
}

#[async_trait]
impl Signer for LocalEd25519Signer {
    async fn sign_envelope(&self, payload: &[u8]) -> Result<DsseEnvelope> {
        let pae = crate::dsse::pae(crate::dsse::DSSE_PAYLOAD_TYPE_IN_TOTO_V1, payload);
        let sig = self.signing_key.sign(&pae);
        let mut env = DsseEnvelope::new(payload);
        env.signatures.push(DsseSignature {
            keyid: self.key_id.clone(),
            sig: base64::engine::general_purpose::STANDARD.encode(sig.to_bytes()),
            cert: None,
        });
        Ok(env)
    }

    fn oidc_subject(&self) -> &str {
        &self.oidc_subject
    }

    fn key_id(&self) -> &str {
        &self.key_id
    }
}

/// Compute a short key id from an Ed25519 verifying key — base64(sha256[..8]).
#[must_use]
pub fn key_id_from(vk: &VerifyingKey) -> String {
    use sha2::{Digest, Sha256};
    let mut h = Sha256::new();
    h.update(vk.as_bytes());
    let digest = h.finalize();
    base64::engine::general_purpose::URL_SAFE_NO_PAD.encode(&digest[..8])
}

/// Verify a DSSE envelope against a known Ed25519 verifying key. Used by
/// readers (gate, verifier, slack-bot) when they have an explicit trust
/// anchor rather than a Fulcio cert chain.
pub fn verify_ed25519(env: &DsseEnvelope, vk: &VerifyingKey) -> Result<()> {
    env.validate_shape()?;
    let pae = env.pae()?;
    for sig_entry in &env.signatures {
        let bytes = base64::engine::general_purpose::STANDARD.decode(&sig_entry.sig)?;
        if bytes.len() != ed25519_dalek::SIGNATURE_LENGTH {
            continue;
        }
        let mut buf = [0u8; ed25519_dalek::SIGNATURE_LENGTH];
        buf.copy_from_slice(&bytes);
        let sig = ed25519_dalek::Signature::from_bytes(&buf);
        if vk.verify_strict(&pae, &sig).is_ok() {
            return Ok(());
        }
    }
    Err(Error::Verify("no valid Ed25519 signature in envelope".into()))
}

// ── Sigstore keyless signer ────────────────────────────────────────────────

/// Sigstore keyless signer: an ephemeral keypair + Fulcio-issued cert chain.
///
/// The flow:
///
/// 1. Generate ephemeral Ed25519 keypair.
/// 2. Build a signed proof of possession (PoP) — DSSE PAE over the OIDC
///    subject — and sign with the ephemeral key.
/// 3. POST to Fulcio with `(oidc_token, public_key, pop_signature)`.
/// 4. Receive an x509 cert chain bound to the OIDC subject + ephemeral key.
/// 5. Sign DSSE envelopes with the ephemeral key; embed the chain in
///    `DsseSignature.cert`.
///
/// In Phase 6 we hold the ephemeral key for the duration of a single
/// envelope to keep the operational complexity inside the relay's bounds.
/// Persistent caching (cert reuse for ≤10min as Sigstore recommends) is a
/// v2 optimisation behind a feature flag.
#[derive(Debug)]
pub struct SigstoreKeylessSigner {
    oidc_token: String,
    oidc_issuer: String,
    fulcio: FulcioClient,
}

impl SigstoreKeylessSigner {
    /// Construct a keyless signer.
    #[must_use]
    pub fn new(oidc_token: String, oidc_issuer: String, fulcio: FulcioClient) -> Self {
        Self {
            oidc_token,
            oidc_issuer,
            fulcio,
        }
    }

    /// Visible for tests.
    #[must_use]
    pub fn oidc_issuer(&self) -> &str {
        &self.oidc_issuer
    }
}

#[async_trait]
impl Signer for SigstoreKeylessSigner {
    async fn sign_envelope(&self, payload: &[u8]) -> Result<DsseEnvelope> {
        // 1. Ephemeral key.
        use rand::rngs::OsRng;
        let mut csprng = OsRng;
        let signing_key = SigningKey::generate(&mut csprng);
        let verifying_key = signing_key.verifying_key();
        let key_id = key_id_from(&verifying_key);

        // 2. Mint a PoP — Fulcio expects the signing identity to prove
        // possession over its OIDC subject.
        let oidc_subject = derive_oidc_subject(&self.oidc_token).unwrap_or_else(|| self.oidc_issuer.clone());
        let pop_pae = crate::dsse::pae("application/vnd.sigstore.fulcio+pop", oidc_subject.as_bytes());
        let pop_sig = signing_key.sign(&pop_pae);

        // 3. Exchange for a cert chain.
        let cert_chain = self
            .fulcio
            .sign_cert_chain(
                &self.oidc_token,
                &verifying_key,
                &pop_sig.to_bytes(),
            )
            .await?;

        // 4. Sign envelope.
        let pae = crate::dsse::pae(crate::dsse::DSSE_PAYLOAD_TYPE_IN_TOTO_V1, payload);
        let sig = signing_key.sign(&pae);
        let mut env = DsseEnvelope::new(payload);
        env.signatures.push(DsseSignature {
            keyid: key_id,
            sig: base64::engine::general_purpose::STANDARD.encode(sig.to_bytes()),
            cert: Some(base64::engine::general_purpose::STANDARD.encode(&cert_chain)),
        });
        Ok(env)
    }

    fn oidc_subject(&self) -> &str {
        // The cert binds the subject; we expose the issuer here for log
        // messages and let callers parse the actual sub claim out of the
        // cert when they need it.
        &self.oidc_issuer
    }

    fn key_id(&self) -> &str {
        "sigstore-keyless"
    }
}

/// Cheap parse of an OIDC token to pull the `sub` claim out. The relay does
/// NOT verify the JWT signature here — that's Fulcio's job — but logging the
/// claim lets us surface "who's about to sign" in audit lines.
fn derive_oidc_subject(token: &str) -> Option<String> {
    let parts: Vec<&str> = token.split('.').collect();
    if parts.len() != 3 {
        return None;
    }
    let payload = base64::engine::general_purpose::URL_SAFE_NO_PAD.decode(parts[1].as_bytes()).ok()?;
    let v: serde_json::Value = serde_json::from_slice(&payload).ok()?;
    v.get("sub").and_then(|s| s.as_str()).map(|s| s.to_string())
}

#[cfg(test)]
mod tests {
    use super::*;

    #[tokio::test]
    async fn local_signer_round_trip() {
        let dir = tempfile::tempdir().unwrap();
        let signer = LocalEd25519Signer::load_or_create(dir.path()).unwrap();
        let env = signer.sign_envelope(b"hello").await.unwrap();
        verify_ed25519(&env, &signer.verifying_key()).unwrap();
    }

    #[tokio::test]
    async fn local_signer_rejects_tamper() {
        let dir = tempfile::tempdir().unwrap();
        let signer = LocalEd25519Signer::load_or_create(dir.path()).unwrap();
        let mut env = signer.sign_envelope(b"hello").await.unwrap();
        // Tamper with the payload after signing.
        env.payload = base64::engine::general_purpose::STANDARD.encode(b"tampered");
        let res = verify_ed25519(&env, &signer.verifying_key());
        assert!(res.is_err(), "tampered envelope must fail verification");
    }

    #[test]
    fn pae_is_stable() {
        // Spec-mandated stable representation.
        let p1 = crate::dsse::pae("application/vnd.in-toto+json", b"x");
        let p2 = crate::dsse::pae("application/vnd.in-toto+json", b"x");
        assert_eq!(p1, p2);
        // The string MUST start with DSSEv1.
        assert!(p1.starts_with(b"DSSEv1 "));
    }
}
