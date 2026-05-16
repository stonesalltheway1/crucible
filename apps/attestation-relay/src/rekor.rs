//! Sigstore Rekor v2 client.
//!
//! Rekor is the append-only transparency log. The Phase-6 relay:
//!
//! 1. Builds an in-toto Statement (`crate::statement`).
//! 2. Signs it as DSSE (`crate::signer`).
//! 3. POSTs the envelope to `/api/v2/log/entries`.
//! 4. Persists the returned UUID + integrated time + verification material.
//! 5. On read, verifies the inclusion proof against Rekor's signed tree
//!    head (STH).
//!
//! When Rekor is unreachable, the journal-backed publisher (RB-05) absorbs
//! the write and a background back-fill task re-submits when the log
//! recovers.

use async_trait::async_trait;
use serde::{Deserialize, Serialize};

use crate::dsse::DsseEnvelope;
use crate::error::{Error, Result};

/// Rekor inner trait. Implementations: `RekorHttpClient`, `MockRekor`.
#[async_trait]
pub trait RekorInner: Send + Sync + std::fmt::Debug {
    /// Publish a DSSE envelope.
    async fn publish(&self, env: &DsseEnvelope) -> Result<RekorEntry>;
    /// Fetch the envelope at the given UUID.
    async fn fetch(&self, uuid: &str) -> Result<DsseEnvelope>;
    /// Fetch the inclusion proof for the given UUID.
    async fn inclusion_proof(&self, uuid: &str) -> Result<InclusionProof>;
}

/// A handle to a Rekor implementation.
#[derive(Debug, Clone)]
pub struct RekorClient {
    inner: std::sync::Arc<dyn RekorInner>,
}

impl RekorClient {
    /// Construct.
    #[must_use]
    pub fn new(inner: std::sync::Arc<dyn RekorInner>) -> Self {
        Self { inner }
    }

    /// Publish.
    pub async fn publish(&self, env: &DsseEnvelope) -> Result<RekorEntry> {
        self.inner.publish(env).await
    }

    /// Fetch.
    pub async fn fetch(&self, uuid: &str) -> Result<DsseEnvelope> {
        self.inner.fetch(uuid).await
    }

    /// Inclusion proof.
    pub async fn inclusion_proof(&self, uuid: &str) -> Result<InclusionProof> {
        self.inner.inclusion_proof(uuid).await
    }
}

/// Receipt returned by `publish`. Mirrors `cruciblev1.RekorEntry` on the Go
/// side via the `/v1/attestations` HTTP response.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RekorEntry {
    /// Rekor UUID (or local-journal UUID).
    pub uuid: String,
    /// Monotonic log index.
    pub log_index: String,
    /// Log ID (Rekor tree id, or `local-journal`).
    pub log_id: String,
    /// Integrated-at RFC 3339.
    pub integrated_time: String,
    /// Verifiable URL.
    pub url: String,
    /// `true` if this entry only exists in the local journal.
    #[serde(default)]
    pub local_journal_fallback: bool,
    /// Self-hosted Rekor signals.
    #[serde(default)]
    pub self_hosted: bool,
}

/// Inclusion proof returned by Rekor.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct InclusionProof {
    /// Leaf index inside the Merkle tree.
    pub log_index: u64,
    /// Tree size at the time the proof was issued.
    pub tree_size: u64,
    /// Tree root hash (hex).
    pub root_hash: String,
    /// Audit path — siblings from leaf to root.
    #[serde(default)]
    pub hashes: Vec<String>,
    /// Signed Tree Head.
    pub signed_tree_head: String,
}

// ── HTTP client ────────────────────────────────────────────────────────────

/// Rekor v2 HTTP client.
#[derive(Debug, Clone)]
pub struct RekorHttpClient {
    base_url: String,
    http: reqwest::Client,
    self_hosted: bool,
}

impl RekorHttpClient {
    /// Build a client.
    pub fn new(base_url: impl Into<String>, self_hosted: bool) -> Self {
        Self {
            base_url: base_url.into(),
            http: reqwest::Client::new(),
            self_hosted,
        }
    }
}

#[derive(Debug, Serialize)]
struct CreateEntryRequest<'a> {
    api_version: &'a str,
    spec: serde_json::Value,
}

#[derive(Debug, Deserialize)]
struct CreateEntryResponse {
    #[serde(default)]
    uuid: String,
    #[serde(default)]
    log_index: u64,
    #[serde(default)]
    log_id: String,
    #[serde(default)]
    integrated_time: i64,
    #[serde(default)]
    body: String,
}

#[async_trait]
impl RekorInner for RekorHttpClient {
    async fn publish(&self, env: &DsseEnvelope) -> Result<RekorEntry> {
        let url = format!("{}/api/v2/log/entries", self.base_url.trim_end_matches('/'));
        // Rekor v2 spec wraps the envelope inside `spec`. Crucible only emits
        // DSSE-wrapped in-toto statements, so `intoto/v0.0.2` is the
        // canonical type.
        let req = CreateEntryRequest {
            api_version: "0.0.2",
            spec: serde_json::json!({
                "envelope": env,
                "publicKey": "" // Fulcio cert is embedded in DSSE
            }),
        };
        let resp = self
            .http
            .post(&url)
            .header("Content-Type", "application/json")
            .json(&req)
            .send()
            .await?;
        if !resp.status().is_success() {
            let status = resp.status();
            let txt = resp.text().await.unwrap_or_default();
            return Err(Error::Rekor(format!("publish status={status} body={txt}")));
        }
        let parsed: CreateEntryResponse = resp.json().await?;
        let integrated = chrono::DateTime::<chrono::Utc>::from_timestamp(parsed.integrated_time, 0)
            .unwrap_or_else(chrono::Utc::now);
        Ok(RekorEntry {
            uuid: parsed.uuid.clone(),
            log_index: parsed.log_index.to_string(),
            log_id: parsed.log_id,
            integrated_time: integrated.to_rfc3339(),
            url: format!("{}/api/v2/log/entries/{}", self.base_url, parsed.uuid),
            local_journal_fallback: false,
            self_hosted: self.self_hosted,
        })
    }

    async fn fetch(&self, uuid: &str) -> Result<DsseEnvelope> {
        let url = format!("{}/api/v2/log/entries/{}", self.base_url.trim_end_matches('/'), uuid);
        let resp = self.http.get(&url).send().await?;
        if !resp.status().is_success() {
            return Err(Error::Rekor(format!("fetch status={}", resp.status())));
        }
        let parsed: CreateEntryResponse = resp.json().await?;
        let body = base64::Engine::decode(&base64::engine::general_purpose::STANDARD, parsed.body)
            .map_err(|e| Error::Rekor(format!("decode body: {e}")))?;
        let env: DsseEnvelope = serde_json::from_slice(&body)?;
        Ok(env)
    }

    async fn inclusion_proof(&self, uuid: &str) -> Result<InclusionProof> {
        let url = format!(
            "{}/api/v2/log/entries/{}/proof",
            self.base_url.trim_end_matches('/'),
            uuid
        );
        let resp = self.http.get(&url).send().await?;
        if !resp.status().is_success() {
            return Err(Error::Rekor(format!("inclusion status={}", resp.status())));
        }
        let parsed: InclusionProof = resp.json().await?;
        Ok(parsed)
    }
}

// ── Mock ───────────────────────────────────────────────────────────────────

/// In-memory mock Rekor for tests. Maintains a monotonically increasing
/// log index and a synthetic Merkle root.
#[derive(Debug, Default)]
pub struct MockRekor {
    log: std::sync::Mutex<Vec<(String, DsseEnvelope)>>,
    /// Visible to other modules (the service uses it for failure-injection
    /// tests via `publish_failure_after_unlocked`).
    pub publish_failure_after: std::sync::Mutex<Option<usize>>,
}

impl MockRekor {
    /// Build.
    #[must_use]
    pub fn new() -> Self {
        Self {
            log: std::sync::Mutex::new(vec![]),
            publish_failure_after: std::sync::Mutex::new(None),
        }
    }

    /// Force the next N publishes to fail. Used to drive RB-05 / journal
    /// fallback tests.
    pub fn force_failure_after(&self, count: usize) {
        *self.publish_failure_after.lock().unwrap() = Some(count);
    }

    /// Number of entries.
    pub fn len(&self) -> usize {
        self.log.lock().unwrap().len()
    }

    /// Empty?
    pub fn is_empty(&self) -> bool {
        self.log.lock().unwrap().is_empty()
    }
}

#[async_trait]
impl RekorInner for MockRekor {
    async fn publish(&self, env: &DsseEnvelope) -> Result<RekorEntry> {
        {
            let mut g = self.publish_failure_after.lock().unwrap();
            if let Some(n) = g.as_mut() {
                if *n == 0 {
                    return Err(Error::Rekor("mock: forced failure".into()));
                }
                *n -= 1;
            }
        }
        let uuid = uuid::Uuid::new_v4().to_string();
        let now = chrono::Utc::now().to_rfc3339();
        let mut g = self.log.lock().unwrap();
        let idx = g.len() as u64;
        g.push((uuid.clone(), env.clone()));
        Ok(RekorEntry {
            uuid: uuid.clone(),
            log_index: idx.to_string(),
            log_id: "mock-rekor".into(),
            integrated_time: now,
            url: format!("mock://rekor/{uuid}"),
            local_journal_fallback: false,
            self_hosted: false,
        })
    }

    async fn fetch(&self, uuid: &str) -> Result<DsseEnvelope> {
        let g = self.log.lock().unwrap();
        g.iter()
            .find(|(u, _)| u == uuid)
            .map(|(_, env)| env.clone())
            .ok_or_else(|| Error::Rekor(format!("mock: uuid {uuid} not found")))
    }

    async fn inclusion_proof(&self, uuid: &str) -> Result<InclusionProof> {
        let g = self.log.lock().unwrap();
        let idx = g
            .iter()
            .position(|(u, _)| u == uuid)
            .ok_or_else(|| Error::Rekor("mock: uuid not found".into()))? as u64;
        Ok(InclusionProof {
            log_index: idx,
            tree_size: g.len() as u64,
            root_hash: hex::encode(sha2::Sha256::digest(format!("mock-root-{}", g.len()).as_bytes())),
            hashes: vec![],
            signed_tree_head: "mock-sth".into(),
        })
    }
}

// Pull in sha2::Digest for the mock root construction.
use sha2::Digest;
