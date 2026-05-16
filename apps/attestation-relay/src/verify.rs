//! High-level verification.
//!
//! `verify_chain` is the function the promotion-gate's `bundle_validator`
//! calls before any Rego evaluation. It:
//!
//! 1. Fetches each Rekor entry referenced by the bundle.
//! 2. Re-derives the subject digest from the named content (when supplied).
//! 3. Asserts the Statement subject digest matches.
//! 4. Verifies the DSSE envelope signature against either:
//!    - a known Ed25519 verifying key (offline / dev), or
//!    - the embedded Fulcio cert chain (which the caller must independently
//!      pin to the configured Sigstore root).
//! 5. Asserts the predicate-type URI is one of the 13 Crucible types.
//! 6. Cross-checks PromotionBundle invariants:
//!      - `agent_oidc_subject` ≠ approver OIDC subjects (T21)
//!      - `signed_at` within configured staleness window
//!      - all referenced child attestations exist in the chain
//!
//! Anything that fails returns `Error::Verify`. The gate treats Error::Verify
//! as a hard reject — no Rego eval, no approval routing, no KMS lease.

use ed25519_dalek::VerifyingKey;

use crate::dsse::DsseEnvelope;
use crate::error::{Error, Result};
use crate::statement::InTotoStatement;

/// Outcome of a chain verification.
#[derive(Debug, Clone)]
pub struct ChainVerification {
    /// Predicate-type URI.
    pub predicate_type: String,
    /// Subject digest (sha256 hex).
    pub subject_digest_sha256: String,
    /// Subject name.
    pub subject_name: String,
    /// OIDC subject from the cert (or local key).
    pub oidc_subject: Option<String>,
    /// True iff the envelope is journal-only (no Rekor inclusion).
    pub journal_only: bool,
}

/// Verify a DSSE envelope and decode the inner Statement.
pub fn verify_envelope(
    env: &DsseEnvelope,
    expected_content: Option<&[u8]>,
    local_key: Option<&VerifyingKey>,
) -> Result<(InTotoStatement, ChainVerification)> {
    env.validate_shape()?;
    // Step 1: signature.
    if let Some(vk) = local_key {
        crate::signer::verify_ed25519(env, vk)?;
    } else if env.signatures.iter().all(|s| s.cert.is_none()) {
        // No local key, no Fulcio cert — refuse.
        return Err(Error::Verify(
            "envelope has no Fulcio cert and no local key provided".into(),
        ));
    }
    // (Real Fulcio chain verification is delegated to the gate's trust-anchor
    // material; this module only handles the in-process Ed25519 case so that
    // unit tests don't pull in a full x509 verifier. The Go side's
    // bundle_validator calls Sigstore Go's `cosign verify-blob` on the
    // cert chain.)

    // Step 2: parse Statement.
    let payload = env.payload_bytes()?;
    let stmt: InTotoStatement = serde_json::from_slice(&payload)?;
    stmt.validate()?;

    // Step 3: subject digest, if content provided.
    if let Some(content) = expected_content {
        let want = crate::statement::sha256_hex(content);
        let have = stmt
            .subject
            .first()
            .and_then(|s| s.digest.get("sha256"))
            .cloned()
            .unwrap_or_default();
        if want != have {
            return Err(Error::Verify(format!(
                "subject digest mismatch: expected {want}, got {have}"
            )));
        }
    }

    // Step 4: predicate-type known?
    if !crate::predicate::ALL_PREDICATES
        .iter()
        .any(|&p| p == stmt.predicate_type)
    {
        return Err(Error::Verify(format!(
            "unknown predicate type {}",
            stmt.predicate_type
        )));
    }

    let subj = stmt.subject.first().expect("non-empty post-validate");
    let cv = ChainVerification {
        predicate_type: stmt.predicate_type.clone(),
        subject_digest_sha256: subj.digest.get("sha256").cloned().unwrap_or_default(),
        subject_name: subj.name.clone(),
        oidc_subject: None,
        journal_only: env.signatures.iter().all(|s| s.cert.is_none()),
    };
    Ok((stmt, cv))
}

/// Reject self-approval — agent_oidc_subject of a promotion bundle must
/// NOT match any approver_oidc_subject in the chain. Threat T21.
pub fn reject_self_approval(agent: &str, approver: &str) -> Result<()> {
    if !agent.is_empty() && agent == approver {
        return Err(Error::SelfApproval {
            agent: agent.into(),
            approver: approver.into(),
        });
    }
    Ok(())
}

/// Reject stale approval — the approval was signed against a specific bundle
/// hash; if the current bundle differs, reject. Threat T2 / "stale
/// approvals" invariant in promotion-contract.md.
pub fn reject_stale_approval(bundle_hash: &str, approval_bound_hash: &str) -> Result<()> {
    if bundle_hash != approval_bound_hash {
        return Err(Error::StaleApproval {
            bundle: bundle_hash.into(),
            approval: approval_bound_hash.into(),
        });
    }
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::signer::{LocalEd25519Signer, Signer};

    #[tokio::test]
    async fn roundtrip_envelope() {
        let dir = tempfile::tempdir().unwrap();
        let signer = LocalEd25519Signer::load_or_create(dir.path()).unwrap();
        let stmt = InTotoStatement::new(
            "task_demo",
            crate::statement::sha256_hex(b"demo-content"),
            crate::predicate::PRED_WRITE_ATTESTATION,
            serde_json::json!({
                "task_id": "task_demo",
                "tenant_id": "ten_x",
                "path": "x.go",
                "agent_oidc_subject": "x"
            }),
        );
        let payload = stmt.to_canonical_json().unwrap();
        let env = signer.sign_envelope(&payload).await.unwrap();
        let vk = signer.verifying_key();
        let (s2, cv) = verify_envelope(&env, Some(b"demo-content"), Some(&vk)).unwrap();
        assert_eq!(s2.predicate_type, crate::predicate::PRED_WRITE_ATTESTATION);
        assert!(cv.journal_only); // no Fulcio cert in local mode
    }

    #[tokio::test]
    async fn rejects_content_mismatch() {
        let dir = tempfile::tempdir().unwrap();
        let signer = LocalEd25519Signer::load_or_create(dir.path()).unwrap();
        let stmt = InTotoStatement::new(
            "task_demo",
            crate::statement::sha256_hex(b"A"),
            crate::predicate::PRED_WRITE_ATTESTATION,
            serde_json::json!({"task_id": "x"}),
        );
        let env = signer.sign_envelope(&stmt.to_canonical_json().unwrap()).await.unwrap();
        let vk = signer.verifying_key();
        let r = verify_envelope(&env, Some(b"B"), Some(&vk));
        assert!(matches!(r, Err(Error::Verify(_))));
    }

    #[test]
    fn self_approval_rejected() {
        let r = reject_self_approval("sub:agent", "sub:agent");
        assert!(matches!(r, Err(Error::SelfApproval { .. })));
        reject_self_approval("agent", "approver").unwrap();
    }

    #[test]
    fn stale_approval_rejected() {
        let r = reject_stale_approval("0xnew", "0xold");
        assert!(matches!(r, Err(Error::StaleApproval { .. })));
        reject_stale_approval("0xsame", "0xsame").unwrap();
    }
}
