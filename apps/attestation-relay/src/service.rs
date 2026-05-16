//! High-level relay facade.
//!
//! `Service` composes a Signer, a Rekor client, and a Journal. It is the
//! single place where the "build → sign → publish → mirror to journal →
//! return receipt" choreography lives. The HTTP layer in `crate::server`
//! is a thin wrapper.

use std::sync::Arc;

use serde_json::Value;
use tracing::{info, warn};

use crate::dsse::DsseEnvelope;
use crate::error::{Error, Result};
use crate::journal::Journal;
use crate::predicate;
use crate::rekor::{RekorClient, RekorEntry};
use crate::signer::Signer;
use crate::statement::{sha256_hex, InTotoStatement};

/// The composed relay service.
#[derive(Debug, Clone)]
pub struct Service {
    signer: Arc<dyn Signer>,
    rekor: Option<RekorClient>,
    journal: Arc<Journal>,
    offline: bool,
}

impl Service {
    /// Build.
    #[must_use]
    pub fn new(signer: Arc<dyn Signer>, rekor: Option<RekorClient>, journal: Arc<Journal>, offline: bool) -> Self {
        Self {
            signer,
            rekor,
            journal,
            offline,
        }
    }

    /// Build → sign → publish in one call.
    ///
    /// Parameters:
    /// - `predicate_type`: one of `crate::predicate::ALL_PREDICATES`.
    /// - `subject_name`: in-toto Statement subject (file path / task id).
    /// - `subject_content`: raw bytes whose sha256 is the subject digest.
    /// - `predicate_payload`: typed JSON for the predicate.
    pub async fn emit(
        &self,
        predicate_type: &str,
        subject_name: &str,
        subject_content: &[u8],
        predicate_payload: Value,
    ) -> Result<EmitOutcome> {
        predicate::validate_loose(predicate_type, &predicate_payload)?;
        let stmt = InTotoStatement::new(
            subject_name.to_string(),
            sha256_hex(subject_content),
            predicate_type.to_string(),
            predicate_payload,
        );
        stmt.validate()?;
        let payload = stmt.to_canonical_json()?;
        let envelope = self.signer.sign_envelope(&payload).await?;
        let receipt = self.publish(&envelope).await?;
        Ok(EmitOutcome {
            envelope,
            receipt,
            statement: stmt,
        })
    }

    /// Publish a pre-built envelope. Used by the runtime shim, which builds
    /// its own statements (it owns the subject digest).
    pub async fn publish(&self, env: &DsseEnvelope) -> Result<RekorEntry> {
        env.validate_shape()?;

        // Always journal first — that's the recovery anchor.
        let journal_receipt = self.journal.append_envelope(env).await?;

        if self.offline || self.rekor.is_none() {
            return Ok(journal_receipt);
        }

        match self.rekor.as_ref().unwrap().publish(env).await {
            Ok(mut r) => {
                // Mark the journal entry as back-filled.
                if let Err(e) = self
                    .journal
                    .mark_backfilled(&journal_receipt.uuid, &r.uuid)
                    .await
                {
                    warn!(error = %e, "mark_backfilled failed; continuing");
                }
                r.local_journal_fallback = false;
                Ok(r)
            }
            Err(e) => {
                // RB-05: Rekor unreachable — return the journal receipt and
                // let the back-fill task try again later.
                warn!(error = %e, "rekor publish failed; falling back to journal");
                Ok(journal_receipt)
            }
        }
    }

    /// Run one back-fill pass: take up to `max` journal entries with no
    /// rekor_uuid and try to publish them. Returns the number successfully
    /// back-filled.
    pub async fn backfill_once(&self, max: usize) -> Result<usize> {
        let Some(rekor) = self.rekor.as_ref() else {
            return Ok(0);
        };
        if self.offline {
            return Ok(0);
        }
        let pending = self.journal.pending_backfill(max)?;
        let mut ok = 0_usize;
        for e in pending {
            match rekor.publish(&e.envelope).await {
                Ok(r) => {
                    self.journal.mark_backfilled(&e.uuid, &r.uuid).await?;
                    ok += 1;
                    info!(rekor_uuid=%r.uuid, journal_uuid=%e.uuid, "back-filled");
                }
                Err(err) => {
                    warn!(error=%err, "back-fill: rekor still unreachable; will retry");
                    return Ok(ok);
                }
            }
        }
        Ok(ok)
    }

    /// Fetch an envelope by uuid. Tries the journal first; falls through to
    /// Rekor when the journal has no record.
    pub async fn fetch(&self, uuid: &str) -> Result<DsseEnvelope> {
        if let Ok(env) = self.journal.fetch(uuid) {
            return Ok(env);
        }
        if let Some(r) = self.rekor.as_ref() {
            return r.fetch(uuid).await;
        }
        Err(Error::Other(format!("uuid {uuid} not found")))
    }

    /// Convenience for /healthz.
    pub fn health(&self) -> Health {
        Health {
            version: crate::VERSION.to_string(),
            journal_entries: self.journal.len(),
            journal_path: self.journal.path().display().to_string(),
            rekor_wired: self.rekor.is_some(),
            offline: self.offline,
            signer_oidc_subject: self.signer.oidc_subject().to_string(),
            signer_key_id: self.signer.key_id().to_string(),
        }
    }

    /// Validate the journal hash chain. Used by /v1/journal/validate.
    pub fn validate_journal(&self) -> Result<usize> {
        self.journal.validate_chain()
    }

    /// Borrow the inner journal — used by the server's /v1/journal/tail.
    #[must_use]
    pub fn journal(&self) -> &Journal {
        &self.journal
    }

    /// Borrow the signer.
    #[must_use]
    pub fn signer(&self) -> &Arc<dyn Signer> {
        &self.signer
    }
}

/// One-shot output of an emit.
#[derive(Debug)]
pub struct EmitOutcome {
    /// Built envelope.
    pub envelope: DsseEnvelope,
    /// Rekor or local-journal receipt.
    pub receipt: RekorEntry,
    /// Inner Statement.
    pub statement: InTotoStatement,
}

/// /healthz payload.
#[derive(Debug, Clone, serde::Serialize)]
pub struct Health {
    /// Crate version.
    pub version: String,
    /// Journal entry count.
    pub journal_entries: usize,
    /// Path to the on-disk journal.
    pub journal_path: String,
    /// True iff a Rekor client is configured.
    pub rekor_wired: bool,
    /// True iff `CRUCIBLE_RELAY_OFFLINE=1`.
    pub offline: bool,
    /// OIDC subject the signer will mint.
    pub signer_oidc_subject: String,
    /// Key id.
    pub signer_key_id: String,
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::rekor::{MockRekor, RekorClient};
    use crate::signer::LocalEd25519Signer;

    async fn build_service(offline: bool) -> (Service, Arc<MockRekor>) {
        let dir = tempfile::tempdir().unwrap();
        let signer = LocalEd25519Signer::load_or_create(dir.path()).unwrap();
        let journal = Arc::new(Journal::open(dir.path().join("j.jsonl")).unwrap());
        let mock = Arc::new(MockRekor::new());
        let client = RekorClient::new(mock.clone());
        let svc = Service::new(Arc::new(signer), Some(client), journal, offline);
        (svc, mock)
    }

    #[tokio::test]
    async fn emit_round_trip() {
        let (svc, mock) = build_service(false).await;
        let r = svc
            .emit(
                predicate::PRED_WRITE_ATTESTATION,
                "task_x",
                b"some-content",
                serde_json::json!({"task_id":"task_x","tenant_id":"ten_x","path":"x.go","agent_oidc_subject":"a"}),
            )
            .await
            .unwrap();
        assert_eq!(mock.len(), 1);
        assert!(!r.receipt.local_journal_fallback);
        assert!(svc.journal().validate_chain().unwrap() == 1);
    }

    #[tokio::test]
    async fn emit_falls_back_on_rekor_failure() {
        let (svc, mock) = build_service(false).await;
        mock.force_failure_after(0);
        let r = svc
            .emit(
                predicate::PRED_WRITE_ATTESTATION,
                "task_x",
                b"some-content",
                serde_json::json!({"task_id":"task_x","tenant_id":"ten_x","path":"x.go","agent_oidc_subject":"a"}),
            )
            .await
            .unwrap();
        assert!(r.receipt.local_journal_fallback);
        // The journal must hold the entry even though Rekor refused.
        assert_eq!(svc.journal().len(), 1);
    }

    #[tokio::test]
    async fn backfill_after_recovery() {
        let (svc, mock) = build_service(false).await;
        mock.force_failure_after(0);
        for _ in 0..3 {
            svc.emit(
                predicate::PRED_WRITE_ATTESTATION,
                "task_x",
                b"c",
                serde_json::json!({"task_id":"task_x","tenant_id":"t","path":"x","agent_oidc_subject":"a"}),
            )
            .await
            .unwrap();
        }
        // Rekor recovers.
        *mock.publish_failure_after_unlocked() = None;
        let n = svc.backfill_once(100).await.unwrap();
        assert_eq!(n, 3);
    }

    #[tokio::test]
    async fn offline_mode_never_calls_rekor() {
        let (svc, mock) = build_service(true).await;
        svc.emit(
            predicate::PRED_WRITE_ATTESTATION,
            "t",
            b"c",
            serde_json::json!({"task_id":"t","tenant_id":"x","path":"y","agent_oidc_subject":"a"}),
        )
        .await
        .unwrap();
        assert!(mock.is_empty());
    }
}

// Helper for tests: expose the mock-rekor's failure switch.
impl crate::rekor::MockRekor {
    /// Lock the failure counter for tests.
    pub fn publish_failure_after_unlocked(&self) -> std::sync::MutexGuard<'_, Option<usize>> {
        self.publish_failure_after.lock().unwrap()
    }
}
