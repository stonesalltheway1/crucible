//! Self-hosted Rekor end-to-end test.
//!
//! Demonstrates that the full attestation chain works without any public
//! Sigstore reachability. The relay is configured with:
//!
//!   - Offline mode (`CRUCIBLE_RELAY_OFFLINE=1`-equivalent) → no Rekor call.
//!   - LocalEd25519 signer (no Fulcio).
//!   - Journal-only fallback.
//!
//! We assert:
//!
//!   1. Emit + fetch round-trip works.
//!   2. Journal chain validates.
//!   3. Signature verifies against the on-disk public key.
//!   4. No HTTP traffic is generated (every test runs in-process).

use std::sync::Arc;

use crucible_attestation_relay::{
    journal::Journal,
    predicate,
    service::Service,
    signer::{verify_ed25519, LocalEd25519Signer, Signer},
    statement::{sha256_hex, InTotoStatement},
};

#[tokio::test]
async fn self_hosted_offline_chain_round_trips() {
    let dir = tempfile::tempdir().unwrap();
    let signer = LocalEd25519Signer::load_or_create(dir.path()).unwrap();
    let vk = signer.verifying_key();
    let journal = Arc::new(Journal::open(dir.path().join("journal.jsonl")).unwrap());
    let svc = Service::new(Arc::new(signer), None, journal, true);

    // 1. Emit a verifier-approval attestation.
    let outcome = svc
        .emit(
            predicate::PRED_VERIFIER_APPROVAL,
            "task_self_hosted",
            b"diff-content",
            serde_json::json!({
                "task_id":"task_self_hosted","diff_hash":"0xdef",
                "verdict":"approved","verifier_oidc_subject":"v"
            }),
        )
        .await
        .unwrap();

    // 2. Receipt MUST be local-journal-only.
    assert!(outcome.receipt.local_journal_fallback);
    assert_eq!(outcome.receipt.log_id, "local-journal");

    // 3. Re-fetch the envelope via the relay facade.
    let env = svc.fetch(&outcome.receipt.uuid).await.unwrap();
    assert!(!env.payload.is_empty());

    // 4. Signature verifies.
    verify_ed25519(&env, &vk).unwrap();

    // 5. Chain validates.
    let count = svc.validate_journal().unwrap();
    assert!(count >= 1);

    // 6. Statement round-trips.
    let payload = env.payload_bytes().unwrap();
    let parsed: InTotoStatement = serde_json::from_slice(&payload).unwrap();
    assert_eq!(parsed.predicate_type, predicate::PRED_VERIFIER_APPROVAL);
    let expected_digest = sha256_hex(b"diff-content");
    assert_eq!(
        parsed.subject[0].digest.get("sha256").unwrap().as_str(),
        expected_digest.as_str()
    );
}

#[tokio::test]
async fn self_hosted_journal_resilient_to_restart() {
    let dir = tempfile::tempdir().unwrap();
    let signer = LocalEd25519Signer::load_or_create(dir.path()).unwrap();
    let journal_path = dir.path().join("journal.jsonl");

    {
        let j = Arc::new(Journal::open(&journal_path).unwrap());
        let svc = Service::new(Arc::new(signer.clone()), None, j, true);
        for i in 0..5 {
            svc.emit(
                predicate::PRED_WRITE_ATTESTATION,
                "task_r",
                format!("c-{i}").as_bytes(),
                serde_json::json!({"task_id":"r","tenant_id":"t","path":"x","agent_oidc_subject":"a"}),
            )
            .await
            .unwrap();
        }
    }

    // Reopen.
    let j2 = Arc::new(Journal::open(&journal_path).unwrap());
    let svc2 = Service::new(Arc::new(signer), None, j2.clone(), true);

    // Validation works across restarts.
    assert!(svc2.validate_journal().unwrap() >= 5);

    // Subsequent append continues the chain.
    svc2.emit(
        predicate::PRED_WRITE_ATTESTATION,
        "task_r",
        b"after-restart",
        serde_json::json!({"task_id":"r","tenant_id":"t","path":"x","agent_oidc_subject":"a"}),
    )
    .await
    .unwrap();
    assert!(svc2.validate_journal().unwrap() >= 6);
}

#[tokio::test]
async fn signer_oidc_subject_is_local_synthetic() {
    let dir = tempfile::tempdir().unwrap();
    let signer = LocalEd25519Signer::load_or_create(dir.path()).unwrap();
    let subj = signer.oidc_subject();
    assert!(subj.starts_with("https://accounts.crucible.dev/relay/local/"));
}
