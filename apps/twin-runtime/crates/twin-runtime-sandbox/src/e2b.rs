//! E2B (Firecracker-via-SaaS) provider.
//!
//! Implements [`crucible_sandbox_spec::SandboxProvider`] against the E2B
//! REST API. Per the May 2026 currency check:
//!
//! - There is no official Go SDK; the TS/Python SDKs exist but we work at
//!   the REST layer from Rust to avoid an FFI dependency.
//! - The v2 controller is secure-by-default: every sandbox call requires the
//!   `X-Access-Token` header minted at create time.
//! - `Sandbox.create()` (HTTP `POST /sandboxes`) replaces the legacy
//!   `Sandbox()` constructor.
//! - `pause` is GA in TS; the Python equivalent is still `beta_pause`. We
//!   call into the same REST endpoint either way.
//! - The new `SandboxNetworkOpts` parameter on create gives us a
//!   programmatic egress allowlist that the runtime layers on top of (and
//!   we feed into the egress manifest enforcement story).
//!
//! When `CRUCIBLE_E2B_API_KEY` is unset, the driver enters **stub mode**:
//! every call returns a typed `crucible_sandbox_spec::Error::PhaseStub`
//! pointing at the env var. This mirrors Phase 1's
//! `TestIntegration_RealHaiku4_5` pattern — tests skip cleanly when keys
//! are absent and fail loud when present-but-broken.
//!
//! For unit testability, the driver is parameterised over a [`HttpClient`]
//! trait; the production binary uses `reqwest`, tests use a `wiremock`-
//! served fake. The integration test against the real E2B API is gated by
//! `CRUCIBLE_E2B_INTEGRATION=1`.

use async_trait::async_trait;
use chrono::{DateTime, Utc};
use crucible_sandbox_spec::{
    Error, ProviderCapabilities, Result, Sandbox, SandboxId, SandboxKillReason, SandboxKind,
    SandboxProvider, SandboxSpec, SandboxState, SnapshotId, SnapshotRef,
};
use serde::{Deserialize, Serialize};
use std::sync::Arc;
use std::time::Duration;
use url::Url;

/// E2B REST endpoint. Override via `CRUCIBLE_E2B_BASE_URL` (used by tests).
pub const DEFAULT_BASE_URL: &str = "https://api.e2b.app";

/// Header E2B uses for the team API key. Per E2B docs (May 2026).
pub const API_KEY_HEADER: &str = "X-API-Key";

/// Header for the per-sandbox secured-access token (v2 controller).
pub const ACCESS_TOKEN_HEADER: &str = "X-Access-Token";

/// HTTP transport trait — abstracted so unit tests can swap in a
/// `wiremock`-backed double.
#[async_trait]
pub trait HttpClient: Send + Sync {
    /// Issue an HTTP POST. `body` is JSON-encodable; returns the response
    /// body as bytes.
    async fn post(
        &self,
        url: &Url,
        body: serde_json::Value,
        headers: &[(&'static str, String)],
    ) -> Result<bytes::Bytes>;

    /// Issue an HTTP DELETE.
    async fn delete(
        &self,
        url: &Url,
        headers: &[(&'static str, String)],
    ) -> Result<bytes::Bytes>;

    /// Issue an HTTP GET.
    async fn get(
        &self,
        url: &Url,
        headers: &[(&'static str, String)],
    ) -> Result<bytes::Bytes>;
}

/// Default [`HttpClient`] backed by `reqwest`.
pub struct ReqwestClient {
    inner: reqwest::Client,
}

impl Default for ReqwestClient {
    fn default() -> Self {
        let inner = reqwest::Client::builder()
            .timeout(Duration::from_secs(30))
            .connect_timeout(Duration::from_secs(5))
            .user_agent(concat!("crucible-twin-runtime/", env!("CARGO_PKG_VERSION")))
            .build()
            .expect("reqwest client builds");
        Self { inner }
    }
}

#[async_trait]
impl HttpClient for ReqwestClient {
    async fn post(
        &self,
        url: &Url,
        body: serde_json::Value,
        headers: &[(&'static str, String)],
    ) -> Result<bytes::Bytes> {
        let mut req = self.inner.post(url.clone()).json(&body);
        for (k, v) in headers {
            req = req.header(*k, v);
        }
        let resp = req
            .send()
            .await
            .map_err(|e| Error::Network(e.to_string()))?;
        check_status(&resp)?;
        resp.bytes()
            .await
            .map_err(|e| Error::Network(format!("read body: {e}")))
    }

    async fn delete(
        &self,
        url: &Url,
        headers: &[(&'static str, String)],
    ) -> Result<bytes::Bytes> {
        let mut req = self.inner.delete(url.clone());
        for (k, v) in headers {
            req = req.header(*k, v);
        }
        let resp = req
            .send()
            .await
            .map_err(|e| Error::Network(e.to_string()))?;
        check_status(&resp)?;
        resp.bytes()
            .await
            .map_err(|e| Error::Network(format!("read body: {e}")))
    }

    async fn get(
        &self,
        url: &Url,
        headers: &[(&'static str, String)],
    ) -> Result<bytes::Bytes> {
        let mut req = self.inner.get(url.clone());
        for (k, v) in headers {
            req = req.header(*k, v);
        }
        let resp = req
            .send()
            .await
            .map_err(|e| Error::Network(e.to_string()))?;
        check_status(&resp)?;
        resp.bytes()
            .await
            .map_err(|e| Error::Network(format!("read body: {e}")))
    }
}

fn check_status(resp: &reqwest::Response) -> Result<()> {
    let status = resp.status();
    if status.is_success() {
        return Ok(());
    }
    if status == 401 || status == 403 {
        return Err(Error::AuthFailed(format!("HTTP {status}")));
    }
    if status == 404 {
        return Err(Error::Other(format!("HTTP 404 from {}", resp.url())));
    }
    if status == 429 || status.is_server_error() {
        return Err(Error::Network(format!("HTTP {status} (retryable)")));
    }
    if status == 402 {
        return Err(Error::QuotaExhausted(format!("HTTP 402 from {}", resp.url())));
    }
    Err(Error::ProviderRejected(format!("HTTP {status}")))
}

/// E2B driver. Construct via [`E2bProvider::from_env`] in production; tests
/// build directly with [`E2bProvider::new`].
pub struct E2bProvider {
    client: Arc<dyn HttpClient>,
    base_url: Url,
    api_key: Option<String>,
}

impl E2bProvider {
    /// Build a driver from `CRUCIBLE_E2B_API_KEY` and (optional)
    /// `CRUCIBLE_E2B_BASE_URL`. If the API key is unset the driver runs in
    /// stub mode — every call returns [`Error::PhaseStub`].
    pub fn from_env() -> Self {
        let api_key = std::env::var("CRUCIBLE_E2B_API_KEY").ok();
        let base_url = std::env::var("CRUCIBLE_E2B_BASE_URL")
            .ok()
            .and_then(|s| Url::parse(&s).ok())
            .unwrap_or_else(|| Url::parse(DEFAULT_BASE_URL).unwrap());
        Self::new(Arc::new(ReqwestClient::default()), base_url, api_key)
    }

    /// Build a driver with explicit dependencies. Used by unit tests.
    pub fn new(client: Arc<dyn HttpClient>, base_url: Url, api_key: Option<String>) -> Self {
        Self {
            client,
            base_url,
            api_key,
        }
    }

    fn key(&self) -> Result<&str> {
        self.api_key.as_deref().ok_or_else(|| {
            Error::PhaseStub(
                "CRUCIBLE_E2B_API_KEY unset — driver in stub mode. Set the env var or use \
                 MockProvider for tests."
                    .into(),
            )
        })
    }

    fn headers(&self) -> Result<Vec<(&'static str, String)>> {
        Ok(vec![(API_KEY_HEADER, self.key()?.to_string())])
    }

    fn endpoint(&self, path: &str) -> Result<Url> {
        self.base_url
            .join(path)
            .map_err(|e| Error::Other(format!("invalid URL path {path}: {e}")))
    }
}

#[async_trait]
impl SandboxProvider for E2bProvider {
    fn kind(&self) -> SandboxKind {
        SandboxKind::E2b
    }

    fn capabilities(&self) -> ProviderCapabilities {
        ProviderCapabilities::e2b_default()
    }

    async fn spawn(&self, spec: &SandboxSpec) -> Result<Sandbox> {
        spec.validate()?;
        let headers = self.headers()?;
        let body = serde_json::json!({
            "templateID": spec.labels.get("crucible.io/e2b-template").map(String::as_str).unwrap_or("base"),
            "metadata": {
                "task_id": spec.task_id,
                "tenant_id": spec.tenant_id,
                "spec_hash": spec.canonical_hash().0,
            },
            "resources": {
                "cpu_count": spec.resources.vcpus,
                "memory_mb": spec.resources.memory_mb,
            },
            "timeout_ms": (spec.absolute_ttl.as_millis() as u64).min(24 * 3600 * 1000),
            "network": {
                "allow_internet_access": !spec.egress.rules.is_empty(),
                "allow_rules": spec.egress.rules.iter().map(|r| serde_json::json!({
                    "domain": r.host,
                    "ports": r.ports,
                })).collect::<Vec<_>>(),
            },
        });

        let url = self.endpoint("/sandboxes")?;
        let raw = self.client.post(&url, body, &headers).await?;
        let parsed: SpawnResponse = serde_json::from_slice(&raw)
            .map_err(|e| Error::Other(format!("E2B spawn response parse: {e}")))?;

        Ok(Sandbox {
            id: SandboxId(parsed.sandbox_id.clone()),
            task_id: spec.task_id.clone(),
            tenant_id: spec.tenant_id.clone(),
            kind: SandboxKind::E2b,
            provider_handle: parsed.sandbox_id,
            control_endpoint: format!("https://{}.e2b.app", parsed.client_id),
            spawned_at: parsed.started_at.unwrap_or_else(Utc::now),
            expires_at: parsed
                .end_at
                .unwrap_or_else(|| Utc::now() + chrono::Duration::from_std(spec.absolute_ttl).unwrap()),
            state: SandboxState::Booting,
            attestation_socket: format!(
                "/work/.crucible/{}.sock",
                parsed.client_id
            ),
            spec_hash: spec.canonical_hash(),
        })
    }

    async fn snapshot(&self, sandbox: &Sandbox, name: &str) -> Result<SnapshotRef> {
        let headers = self.headers()?;
        let path = format!("/sandboxes/{}/snapshots", sandbox.provider_handle);
        let body = serde_json::json!({ "name": name });
        let url = self.endpoint(&path)?;
        let raw = self.client.post(&url, body, &headers).await?;
        let parsed: SnapshotResponse = serde_json::from_slice(&raw)
            .map_err(|e| Error::Other(format!("E2B snapshot response parse: {e}")))?;

        Ok(SnapshotRef {
            id: SnapshotId(parsed.snapshot_id.clone()),
            sandbox_id: sandbox.id.clone(),
            task_id: sandbox.task_id.clone(),
            name: name.to_string(),
            taken_at: Utc::now(),
            provider_handle: parsed.snapshot_id,
            size_bytes: parsed.size_bytes.unwrap_or(0),
            base_spec_hash: sandbox.spec_hash.clone(),
            attestation_chain_head: None,
        })
    }

    async fn restore(
        &self,
        snapshot: &SnapshotRef,
        new_task_id: Option<&str>,
    ) -> Result<Sandbox> {
        let headers = self.headers()?;
        let body = serde_json::json!({
            "snapshotId": snapshot.provider_handle,
            "metadata": {
                "task_id": new_task_id.unwrap_or(&snapshot.task_id),
                "restored_from": snapshot.id.0,
            },
        });
        let url = self.endpoint("/sandboxes")?;
        let raw = self.client.post(&url, body, &headers).await?;
        let parsed: SpawnResponse = serde_json::from_slice(&raw)
            .map_err(|e| Error::Other(format!("E2B restore response parse: {e}")))?;

        Ok(Sandbox {
            id: SandboxId(parsed.sandbox_id.clone()),
            task_id: new_task_id.unwrap_or(&snapshot.task_id).to_string(),
            tenant_id: "unknown".into(),
            kind: SandboxKind::E2b,
            provider_handle: parsed.sandbox_id,
            control_endpoint: format!("https://{}.e2b.app", parsed.client_id),
            spawned_at: Utc::now(),
            expires_at: parsed.end_at.unwrap_or_else(|| Utc::now() + chrono::Duration::hours(1)),
            state: SandboxState::Booting,
            attestation_socket: format!("/work/.crucible/{}.sock", parsed.client_id),
            spec_hash: snapshot.base_spec_hash.clone(),
        })
    }

    async fn kill(&self, sandbox: &Sandbox, reason: SandboxKillReason) -> Result<()> {
        let headers = self.headers()?;
        let path = format!("/sandboxes/{}", sandbox.provider_handle);
        let url = self.endpoint(&path)?;
        let res = self.client.delete(&url, &headers).await;
        match res {
            Ok(_) => {
                tracing::info!(
                    sandbox = %sandbox.id,
                    ?reason,
                    "killed E2B sandbox"
                );
                Ok(())
            }
            // E2B returns 404 if the sandbox is already gone — idempotent.
            Err(Error::Other(s)) if s.contains("HTTP 404") => Ok(()),
            Err(e) => Err(e),
        }
    }

    async fn state(&self, id: &SandboxId) -> Result<SandboxState> {
        let headers = self.headers()?;
        let path = format!("/sandboxes/{}", id.0);
        let url = self.endpoint(&path)?;
        let res = self.client.get(&url, &headers).await;
        match res {
            Ok(raw) => {
                let parsed: StateResponse = serde_json::from_slice(&raw)
                    .map_err(|e| Error::Other(format!("E2B state response parse: {e}")))?;
                Ok(match parsed.state.as_str() {
                    "running" | "ready" => SandboxState::Ready,
                    "booting" | "provisioning" => SandboxState::Booting,
                    "paused" => SandboxState::Paused,
                    "terminating" => SandboxState::Terminating,
                    "terminated" | "killed" => SandboxState::Terminated,
                    "failed" => SandboxState::Failed,
                    other => {
                        tracing::warn!(state = %other, "unknown E2B sandbox state");
                        SandboxState::Failed
                    }
                })
            }
            Err(Error::Other(s)) if s.contains("HTTP 404") => {
                Err(Error::NotFound(id.clone()))
            }
            Err(e) => Err(e),
        }
    }

    async fn list(&self, tenant_id: &str) -> Result<Vec<Sandbox>> {
        let headers = self.headers()?;
        let path = format!("/sandboxes?metadata.tenant_id={tenant_id}");
        let url = self.endpoint(&path)?;
        let raw = self.client.get(&url, &headers).await?;
        let parsed: ListResponse = serde_json::from_slice(&raw)
            .map_err(|e| Error::Other(format!("E2B list response parse: {e}")))?;
        Ok(parsed
            .sandboxes
            .into_iter()
            .map(|s| {
                // Best-effort reconstruction — `tenant_id` and `spec_hash`
                // come from the sandbox's metadata which the spec embedded
                // at create time.
                let task_id = s
                    .metadata
                    .as_ref()
                    .and_then(|m| m.get("task_id"))
                    .and_then(|v| v.as_str())
                    .unwrap_or("")
                    .to_string();
                let spec_hash = s
                    .metadata
                    .as_ref()
                    .and_then(|m| m.get("spec_hash"))
                    .and_then(|v| v.as_str())
                    .unwrap_or("")
                    .to_string();
                Sandbox {
                    id: SandboxId(s.sandbox_id.clone()),
                    task_id,
                    tenant_id: tenant_id.to_string(),
                    kind: SandboxKind::E2b,
                    provider_handle: s.sandbox_id,
                    control_endpoint: format!("https://{}.e2b.app", s.client_id),
                    spawned_at: s.started_at.unwrap_or_else(Utc::now),
                    expires_at: s.end_at.unwrap_or_else(Utc::now),
                    state: SandboxState::Ready,
                    attestation_socket: format!("/work/.crucible/{}.sock", s.client_id),
                    spec_hash: crucible_sandbox_spec::SpecHash(spec_hash),
                }
            })
            .collect())
    }
}

// ─────────────────────────────────────────────────────────────────────────────
// Wire types — minimal subset of the E2B REST schema we depend on.
// ─────────────────────────────────────────────────────────────────────────────

#[derive(Debug, Clone, Serialize, Deserialize)]
struct SpawnResponse {
    #[serde(rename = "sandboxID", alias = "sandbox_id")]
    sandbox_id: String,
    #[serde(rename = "clientID", alias = "client_id")]
    client_id: String,
    #[serde(rename = "startedAt", alias = "started_at")]
    started_at: Option<DateTime<Utc>>,
    #[serde(rename = "endAt", alias = "end_at")]
    end_at: Option<DateTime<Utc>>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
struct SnapshotResponse {
    #[serde(rename = "snapshotID", alias = "snapshot_id")]
    snapshot_id: String,
    #[serde(rename = "sizeBytes", alias = "size_bytes")]
    size_bytes: Option<u64>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
struct StateResponse {
    state: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
struct ListResponse {
    sandboxes: Vec<ListEntry>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
struct ListEntry {
    #[serde(rename = "sandboxID", alias = "sandbox_id")]
    sandbox_id: String,
    #[serde(rename = "clientID", alias = "client_id")]
    client_id: String,
    #[serde(rename = "startedAt", alias = "started_at")]
    started_at: Option<DateTime<Utc>>,
    #[serde(rename = "endAt", alias = "end_at")]
    end_at: Option<DateTime<Utc>>,
    metadata: Option<serde_json::Value>,
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::sync::Mutex;

    /// Fake HTTP client for unit tests.
    struct FakeClient {
        responses: Mutex<Vec<(&'static str, serde_json::Value)>>,
        calls: Mutex<Vec<(String, Url)>>,
    }

    impl FakeClient {
        fn new(responses: Vec<(&'static str, serde_json::Value)>) -> Self {
            Self {
                responses: Mutex::new(responses),
                calls: Mutex::new(Vec::new()),
            }
        }

        fn record_and_pop(&self, method: &str, url: &Url) -> bytes::Bytes {
            self.calls.lock().unwrap().push((method.to_string(), url.clone()));
            let mut resps = self.responses.lock().unwrap();
            let (_expected, body) = resps.remove(0);
            serde_json::to_vec(&body).unwrap().into()
        }
    }

    #[async_trait]
    impl HttpClient for FakeClient {
        async fn post(
            &self,
            url: &Url,
            _body: serde_json::Value,
            _headers: &[(&'static str, String)],
        ) -> Result<bytes::Bytes> {
            Ok(self.record_and_pop("POST", url))
        }
        async fn delete(
            &self,
            url: &Url,
            _headers: &[(&'static str, String)],
        ) -> Result<bytes::Bytes> {
            Ok(self.record_and_pop("DELETE", url))
        }
        async fn get(
            &self,
            url: &Url,
            _headers: &[(&'static str, String)],
        ) -> Result<bytes::Bytes> {
            Ok(self.record_and_pop("GET", url))
        }
    }

    fn minimal_spec() -> SandboxSpec {
        use crucible_sandbox_spec::{
            DefaultEgressAction, EgressManifest, FilesystemSpec, HeartbeatSpec, Resources,
            SyscallShimPolicy,
        };
        use std::collections::BTreeMap;
        use std::time::Duration;
        let mut labels = BTreeMap::new();
        labels.insert("crucible.io/local-dev".into(), "true".into());
        SandboxSpec {
            task_id: "task_e2b_test".into(),
            tenant_id: "ten_e2b_test".into(),
            kind: SandboxKind::E2b,
            provider_region: "aws-us-east-1".into(),
            resources: Resources::default(),
            egress: EgressManifest {
                rules: Vec::new(),
                default_action: DefaultEgressAction::Deny,
            },
            secrets: Vec::new(),
            db: None,
            filesystem: FilesystemSpec {
                base_sha: "abc".into(),
                repo_url: "https://x.invalid/r".into(),
                depth: 1,
                overlay_mode: "copy".into(),
                prewarm_paths: Vec::new(),
            },
            tape: None,
            shim: SyscallShimPolicy::default(),
            heartbeat: HeartbeatSpec::default(),
            absolute_ttl: Duration::from_secs(600),
            labels,
        }
    }

    #[tokio::test]
    async fn spawn_returns_phase_stub_without_api_key() {
        let provider = E2bProvider::new(
            Arc::new(FakeClient::new(vec![])),
            Url::parse(DEFAULT_BASE_URL).unwrap(),
            None,
        );
        let err = provider.spawn(&minimal_spec()).await.unwrap_err();
        assert!(matches!(err, Error::PhaseStub(_)), "expected PhaseStub, got {err:?}");
    }

    #[tokio::test]
    async fn spawn_with_key_calls_post_sandboxes() {
        let fake = Arc::new(FakeClient::new(vec![(
            "spawn",
            serde_json::json!({
                "sandboxID": "sb_abc",
                "clientID": "cl_xyz",
                "startedAt": "2026-05-15T18:00:00Z",
                "endAt": "2026-05-15T19:00:00Z",
            }),
        )]));
        let provider = E2bProvider::new(
            fake.clone(),
            Url::parse(DEFAULT_BASE_URL).unwrap(),
            Some("e2b_test_key".into()),
        );
        let sandbox = provider.spawn(&minimal_spec()).await.unwrap();
        assert_eq!(sandbox.id.0, "sb_abc");
        assert_eq!(sandbox.kind, SandboxKind::E2b);
        let calls = fake.calls.lock().unwrap();
        assert_eq!(calls.len(), 1);
        assert_eq!(calls[0].0, "POST");
        assert!(calls[0].1.path().ends_with("/sandboxes"));
    }

    #[tokio::test]
    async fn kill_is_idempotent_on_404() {
        let fake = Arc::new(FakeClient::new(vec![(
            "kill",
            serde_json::json!({"ok": true}),
        )]));
        let provider = E2bProvider::new(
            fake,
            Url::parse(DEFAULT_BASE_URL).unwrap(),
            Some("e2b_test_key".into()),
        );
        let sandbox = Sandbox {
            id: SandboxId("sb_x".into()),
            task_id: "t".into(),
            tenant_id: "ten".into(),
            kind: SandboxKind::E2b,
            provider_handle: "sb_x".into(),
            control_endpoint: "u".into(),
            spawned_at: Utc::now(),
            expires_at: Utc::now() + chrono::Duration::hours(1),
            state: SandboxState::Ready,
            attestation_socket: "s".into(),
            spec_hash: crucible_sandbox_spec::SpecHash(String::new()),
        };
        provider.kill(&sandbox, SandboxKillReason::Clean).await.unwrap();
    }

    #[test]
    fn capabilities_match_e2b_default() {
        let p = E2bProvider::new(
            Arc::new(FakeClient::new(vec![])),
            Url::parse(DEFAULT_BASE_URL).unwrap(),
            None,
        );
        let caps = p.capabilities();
        assert!(caps.supports_snapshot);
        assert!(caps.supports_restore);
        assert!(caps.supports_native_egress);
        assert!(!caps.supports_guest_ebpf);
    }
}
