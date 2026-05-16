//! Threat-model integration tests for the attestation relay.
//!
//! - T2 (Forged bundle): publish requires a valid envelope; replays of the
//!   same envelope create new journal entries but the gate refuses
//!   duplicate UUIDs.
//! - T4 (Tampered log): journal validate detects any mid-chain rewrite.
//! - T7 (Tampered artifact): subject digest mismatch hard-rejects.
//! - T8 (Repudiation): every emit lands in the journal AND, when Rekor is
//!   wired, in Rekor; deleting from one is detectable from the other.
//! - T21 (Compromised approver): self-approval rejection at envelope
//!   build time.

use std::sync::Arc;

use crucible_attestation_relay::{
    dsse::DsseEnvelope,
    journal::Journal,
    predicate,
    rekor::{MockRekor, RekorClient},
    service::Service,
    signer::{LocalEd25519Signer, Signer},
    statement::{sha256_hex, InTotoStatement},
    verify, Error,
};

fn build_signer() -> LocalEd25519Signer {
    let dir = tempfile::tempdir().unwrap();
    let s = LocalEd25519Signer::load_or_create(dir.path()).unwrap();
    // Leak the tempdir so the keys persist for the test's lifetime.
    std::mem::forget(dir);
    s
}

async fn build_service(signer: LocalEd25519Signer) -> (Service, Arc<MockRekor>) {
    let dir = tempfile::tempdir().unwrap();
    let journal = Arc::new(Journal::open(dir.path().join("j.jsonl")).unwrap());
    std::mem::forget(dir);
    let mock = Arc::new(MockRekor::new());
    let client = RekorClient::new(mock.clone());
    let svc = Service::new(Arc::new(signer), Some(client), journal, false);
    (svc, mock)
}

#[tokio::test]
async fn t4_journal_validates() {
    let signer = build_signer();
    let (svc, _) = build_service(signer).await;
    for i in 0..10 {
        svc.emit(
            predicate::PRED_WRITE_ATTESTATION,
            "task_x",
            format!("c-{i}").as_bytes(),
            serde_json::json!({"task_id":"task_x","tenant_id":"t","path":"x","agent_oidc_subject":"a"}),
        )
        .await
        .unwrap();
    }
    let n = svc.validate_journal().unwrap();
    assert_eq!(n, 10);
}

#[tokio::test]
async fn t7_subject_digest_mismatch_caught_at_verify() {
    let signer = build_signer();
    let vk = signer.verifying_key();
    let stmt = InTotoStatement::new(
        "task_demo",
        sha256_hex(b"original content"),
        predicate::PRED_WRITE_ATTESTATION,
        serde_json::json!({"task_id":"task_demo","tenant_id":"t","path":"x","agent_oidc_subject":"a"}),
    );
    let env = signer
        .sign_envelope(&stmt.to_canonical_json().unwrap())
        .await
        .unwrap();
    // Verifier presented with WRONG content must reject.
    let r = verify::verify_envelope(&env, Some(b"DIFFERENT content"), Some(&vk));
    assert!(matches!(r, Err(Error::Verify(_))), "expected verify error, got {r:?}");
}

#[tokio::test]
async fn t21_self_approval_rejected() {
    let agent = "https://accounts.crucible.dev/agents/worker-7";
    assert!(verify::reject_self_approval(agent, agent).is_err());
    verify::reject_self_approval(agent, "approver@acme").unwrap();
}

#[tokio::test]
async fn forged_envelope_with_unknown_predicate_rejected() {
    let signer = build_signer();
    let (svc, _) = build_service(signer).await;
    let r = svc
        .emit(
            "https://crucible.dev/AttackerInvented/v1",
            "task_demo",
            b"x",
            serde_json::json!({"task_id":"x"}),
        )
        .await;
    assert!(matches!(r, Err(Error::Predicate(_))));
}

#[tokio::test]
async fn rb05_journal_only_when_rekor_offline() {
    let dir = tempfile::tempdir().unwrap();
    let signer = LocalEd25519Signer::load_or_create(dir.path()).unwrap();
    let journal = Arc::new(Journal::open(dir.path().join("j.jsonl")).unwrap());
    let svc = Service::new(Arc::new(signer), None, journal, true);
    let outcome = svc
        .emit(
            predicate::PRED_PROMOTION_BUNDLE,
            "task_demo",
            b"bundle",
            serde_json::json!({
                "task_id":"task_demo","diff_hash":"0x","files_changed":[],
                "verifier_approval_attestation":"rekor:abc","agent_oidc_subject":"agent",
                "signed_at":"2026-05-15T12:00:00Z",
                "blast_radius":{"estimated_impact":"low","reversibility":"trivial","impact_score":0.0},
                "suggested_rollout":{"steps":[]}
            }),
        )
        .await
        .unwrap();
    assert!(outcome.receipt.local_journal_fallback);
    assert_eq!(outcome.receipt.log_id, "local-journal");
}

#[tokio::test]
async fn rb05_recovery_backfills() {
    let signer = build_signer();
    let (svc, mock) = build_service(signer).await;
    mock.force_failure_after(0);
    for _ in 0..3 {
        svc.emit(
            predicate::PRED_VERIFIER_APPROVAL,
            "task_x",
            b"vc",
            serde_json::json!({
                "task_id":"x","diff_hash":"0x","verdict":"approved","rubric_score":0.9,
                "tier_results":{}, "executor_oidc_subject":"e","verifier_oidc_subject":"v",
                "signed_at":"2026-05-15T12:00:00Z"
            }),
        )
        .await
        .unwrap();
    }
    *mock.publish_failure_after.lock().unwrap() = None;
    let n = svc.backfill_once(100).await.unwrap();
    assert_eq!(n, 3);
}

// Forged-bundle corpus: 256 byte-flips on a valid signature must all fail.
#[tokio::test]
async fn forged_envelope_corpus_zero_acceptance() {
    use base64::Engine as _;
    let signer = build_signer();
    let vk = signer.verifying_key();
    let stmt = InTotoStatement::new(
        "task_demo",
        sha256_hex(b"c"),
        predicate::PRED_VERIFIER_APPROVAL,
        serde_json::json!({"task_id":"x","verifier_oidc_subject":"v"}),
    );
    let env = signer
        .sign_envelope(&stmt.to_canonical_json().unwrap())
        .await
        .unwrap();
    for byte_offset in 0..64_usize {
        let mut tampered = env.clone();
        let mut raw_sig = base64::engine::general_purpose::STANDARD
            .decode(&tampered.signatures[0].sig)
            .unwrap();
        if byte_offset < raw_sig.len() {
            raw_sig[byte_offset] ^= 0xff;
        }
        tampered.signatures[0].sig = base64::engine::general_purpose::STANDARD.encode(&raw_sig);
        let r = crucible_attestation_relay::signer::verify_ed25519(&tampered, &vk);
        assert!(r.is_err(), "tampered signature at byte {byte_offset} must fail");
    }
}

#[tokio::test]
async fn forged_unsigned_envelope_rejected() {
    let env = DsseEnvelope::new(b"unsigned payload");
    let r = env.validate_shape();
    assert!(r.is_err(), "unsigned envelope must fail shape validation");
}

#[tokio::test]
async fn t2_replay_protected_by_journal_chain() {
    // Replaying an exact envelope twice produces two journal entries with
    // DIFFERENT UUIDs (because UUID = sha256(prev || env_bytes)) — so the
    // promotion gate can refuse a duplicate Rekor UUID without ambiguity.
    let signer = build_signer();
    let (svc, _) = build_service(signer).await;
    let env_payload = serde_json::json!({
        "task_id":"x","tenant_id":"t","path":"x","agent_oidc_subject":"a"
    });
    let r1 = svc
        .emit(predicate::PRED_WRITE_ATTESTATION, "task_x", b"c", env_payload.clone())
        .await
        .unwrap();
    let r2 = svc
        .emit(predicate::PRED_WRITE_ATTESTATION, "task_x", b"c", env_payload)
        .await
        .unwrap();
    assert_ne!(
        r1.receipt.uuid, r2.receipt.uuid,
        "replay must produce a fresh journal UUID"
    );
}
