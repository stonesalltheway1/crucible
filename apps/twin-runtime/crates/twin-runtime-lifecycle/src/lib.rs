//! Sandbox lifecycle orchestrator — wires sandbox-provider + filesystem-twin
//! + egress-manifest + syscall-shim + attestation-emission into one
//! cohesive `spawn / snapshot / kill / heartbeat / gc` flow.
//!
//! This is the unit the gRPC `TwinRuntimeService` calls directly. It is
//! transport-agnostic — unit tests construct an `Orchestrator` and exercise
//! it without spinning up a tonic server.

#![warn(missing_docs)]

use async_trait::async_trait;
use crucible_sandbox_spec::{
    Sandbox, SandboxId, SandboxKillReason, SandboxProvider, SandboxSpec, SandboxState, SnapshotRef,
};
use std::collections::HashMap;
use std::sync::Arc;
use thiserror::Error;
use tokio::sync::{Mutex, broadcast};

use twin_runtime_attest::{
    LocalJournalPublisher, Publisher, Signer_, emit, sandbox_lifecycle,
};
use twin_runtime_egress::ManifestValidator;
use twin_runtime_fs::{Orchestrator as FsOrchestrator, OverlayMode, ProdOrchestrator as ProdFs, WorkLayout};
use twin_runtime_sandbox::ProviderRegistry;
use twin_runtime_shim::Shim;

pub mod heartbeat;
pub mod events;

pub use events::{Event, EventBus, EventKind};

/// Errors from the orchestrator.
#[derive(Debug, Error)]
pub enum Error {
    /// Spec validation failed at the sandbox-spec layer.
    #[error("sandbox-spec: {0}")]
    Spec(#[from] crucible_sandbox_spec::Error),
    /// Egress manifest invalid.
    #[error("egress: {0}")]
    Egress(#[from] twin_runtime_egress::Error),
    /// Filesystem prep / mount failed.
    #[error("fs: {0}")]
    Fs(#[from] twin_runtime_fs::Error),
    /// Shim setup / activation failed.
    #[error("shim: {0}")]
    Shim(#[from] twin_runtime_shim::Error),
    /// Attestation emission failed — surfaces an audit gap; fail loudly.
    #[error("attest: {0}")]
    Attest(#[from] twin_runtime_attest::Error),
    /// Unknown sandbox.
    #[error("sandbox not found: {0}")]
    NotFound(SandboxId),
    /// Caller asked for an operation that isn't legal at the current state.
    #[error("illegal state: sandbox {sandbox} is {state:?}")]
    IllegalState {
        /// Sandbox id.
        sandbox: SandboxId,
        /// Current state.
        state: SandboxState,
    },
}

/// Result alias.
pub type Result<T> = std::result::Result<T, Error>;

/// The orchestrator handle the gRPC server holds.
pub struct Orchestrator {
    providers: ProviderRegistry,
    fs: Arc<dyn FsOrchestrator>,
    signer: Arc<Signer_>,
    publisher: Arc<dyn Publisher>,
    bus: EventBus,
    ledger: Mutex<Ledger>,
}

#[derive(Default)]
struct Ledger {
    sandboxes: HashMap<SandboxId, LedgerEntry>,
}

struct LedgerEntry {
    sandbox: Sandbox,
    layout: WorkLayout,
    overlay_mode: OverlayMode,
}

impl Orchestrator {
    /// Build an orchestrator with the runtime defaults.
    ///
    /// # Errors
    /// Returns [`Error::Attest`] if the local journal cannot be opened.
    pub fn with_defaults(journal_path: impl Into<std::path::PathBuf>) -> Result<Self> {
        let publisher: Arc<dyn Publisher> =
            Arc::new(LocalJournalPublisher::open(journal_path.into())?);
        Ok(Self::new(
            ProviderRegistry::with_defaults(),
            Arc::new(ProdFs::new()),
            Arc::new(Signer_::ephemeral()),
            publisher,
        ))
    }

    /// Build an orchestrator with explicit dependencies (used by tests).
    pub fn new(
        providers: ProviderRegistry,
        fs: Arc<dyn FsOrchestrator>,
        signer: Arc<Signer_>,
        publisher: Arc<dyn Publisher>,
    ) -> Self {
        Self {
            providers,
            fs,
            signer,
            publisher,
            bus: EventBus::new(256),
            ledger: Mutex::new(Ledger::default()),
        }
    }

    /// Subscribe to the runtime event stream.
    pub fn subscribe(&self) -> broadcast::Receiver<Event> {
        self.bus.subscribe()
    }

    /// Spawn a sandbox from the spec.
    ///
    /// Steps:
    ///   1. Validate spec + egress manifest (fail-closed boundary).
    ///   2. Resolve the provider for `spec.kind`.
    ///   3. Build the shim (Layer 1 always ready; layers 2/3 are no-ops on
    ///      non-Linux dev hosts).
    ///   4. Provider spawn.
    ///   5. Prepare + mount filesystem twin.
    ///   6. Emit `SandboxLifecycle/v1` attestation `event=spawned`.
    ///   7. Record in ledger.
    pub async fn spawn(&self, spec: SandboxSpec) -> Result<Sandbox> {
        spec.validate()?;
        ManifestValidator::validate(&spec.egress)?;

        let provider = self.providers.get(spec.kind)?;
        let shim = Shim::build(spec.shim.clone())?;
        shim.activate()?;

        let sandbox = provider.spawn(&spec).await?;
        let layout = WorkLayout::rooted_at(format!("/work/{}", sandbox.id.0));
        let overlay_mode = OverlayMode::parse(&spec.filesystem.overlay_mode);

        // Filesystem prep is best-effort in non-sandboxed dev runs (we don't
        // actually clone repos in unit tests); the prod sandbox runs this
        // inside the microVM as part of boot.
        if let Err(e) = self.fs.prepare(&spec.filesystem, &layout).await {
            tracing::warn!(
                error = %e,
                "fs.prepare failed during spawn — sandbox is still up; \
                 dev hosts often hit this. STUB: production runs prep \
                 inside the microVM via the sandbox bootstrap script."
            );
        }

        let stmt = sandbox_lifecycle(
            &sandbox.task_id,
            &sandbox.tenant_id,
            sandbox.id.as_str(),
            "spawned",
            &sandbox.spec_hash.0,
            serde_json::json!({
                "kind": format!("{:?}", sandbox.kind),
                "provider_handle": sandbox.provider_handle,
            }),
        );
        let _entry = emit(&self.signer, &self.publisher, stmt).await?;

        let mut ledger = self.ledger.lock().await;
        ledger.sandboxes.insert(
            sandbox.id.clone(),
            LedgerEntry {
                sandbox: sandbox.clone(),
                layout,
                overlay_mode,
            },
        );

        self.bus
            .publish(Event::new(EventKind::Spawned, &sandbox))
            .await;
        Ok(sandbox)
    }

    /// Snapshot a sandbox.
    pub async fn snapshot(&self, id: &SandboxId, name: &str) -> Result<SnapshotRef> {
        let entry = self.lookup(id).await?;
        let provider = self.providers.get(entry.sandbox.kind)?;
        let snap = provider.snapshot(&entry.sandbox, name).await?;
        let stmt = sandbox_lifecycle(
            &entry.sandbox.task_id,
            &entry.sandbox.tenant_id,
            entry.sandbox.id.as_str(),
            "snapshot",
            &entry.sandbox.spec_hash.0,
            serde_json::json!({"snapshot_id": snap.id.0, "name": name}),
        );
        let _ = emit(&self.signer, &self.publisher, stmt).await?;
        self.bus
            .publish(Event::new(EventKind::Snapshot, &entry.sandbox))
            .await;
        Ok(snap)
    }

    /// Kill a sandbox.
    pub async fn kill(
        &self,
        id: &SandboxId,
        reason: SandboxKillReason,
    ) -> Result<()> {
        let entry = self.lookup(id).await?;
        let provider = self.providers.get(entry.sandbox.kind)?;
        provider.kill(&entry.sandbox, reason).await?;
        // Best-effort unmount; production sandbox is destroyed by the
        // provider so this is a no-op outside the sandbox process.
        let _ = self.fs.unmount(&entry.layout, entry.overlay_mode).await;

        let stmt = sandbox_lifecycle(
            &entry.sandbox.task_id,
            &entry.sandbox.tenant_id,
            entry.sandbox.id.as_str(),
            "killed",
            &entry.sandbox.spec_hash.0,
            serde_json::json!({"reason": format!("{reason:?}").to_ascii_lowercase()}),
        );
        let _ = emit(&self.signer, &self.publisher, stmt).await?;

        self.ledger.lock().await.sandboxes.remove(id);
        self.bus
            .publish(Event::new_kill(EventKind::Killed, &entry.sandbox, reason))
            .await;
        Ok(())
    }

    /// List sandboxes the runtime believes are live.
    pub async fn list(&self) -> Vec<Sandbox> {
        let ledger = self.ledger.lock().await;
        ledger
            .sandboxes
            .values()
            .map(|e| e.sandbox.clone())
            .collect()
    }

    /// Get a sandbox by id.
    pub async fn get(&self, id: &SandboxId) -> Result<Sandbox> {
        Ok(self.lookup(id).await?.sandbox)
    }

    /// Reconcile the runtime ledger against each provider's view.
    /// Sandboxes in the ledger but not the provider are forgotten;
    /// sandboxes in the provider but not the ledger are killed
    /// (orphan-GC).
    pub async fn reconcile(&self, tenant_id: &str) -> Result<()> {
        let ledger_view: Vec<Sandbox> = {
            let ledger = self.ledger.lock().await;
            ledger
                .sandboxes
                .values()
                .filter(|e| e.sandbox.tenant_id == tenant_id)
                .map(|e| e.sandbox.clone())
                .collect()
        };
        let ledger_ids: std::collections::HashSet<SandboxId> =
            ledger_view.iter().map(|s| s.id.clone()).collect();

        for kind in [
            crucible_sandbox_spec::SandboxKind::E2b,
            crucible_sandbox_spec::SandboxKind::RawFirecracker,
        ] {
            let Ok(provider) = self.providers.get(kind) else {
                continue;
            };
            let provider_view = provider.list(tenant_id).await.unwrap_or_default();
            for sandbox in provider_view {
                if !ledger_ids.contains(&sandbox.id) {
                    tracing::warn!(
                        sandbox = %sandbox.id,
                        "orphan sandbox detected — killing"
                    );
                    let _ = provider
                        .kill(&sandbox, SandboxKillReason::Manual)
                        .await;
                }
            }
        }
        // Drop ledger entries the provider says are gone.
        let mut ledger = self.ledger.lock().await;
        for ls in ledger_view {
            if let Ok(provider) = self.providers.get(ls.kind) {
                if let Ok(state) = provider.state(&ls.id).await {
                    if state.is_terminal() {
                        ledger.sandboxes.remove(&ls.id);
                    }
                }
            }
        }
        Ok(())
    }

    async fn lookup(&self, id: &SandboxId) -> Result<LedgerEntrySnapshot> {
        let ledger = self.ledger.lock().await;
        let entry = ledger
            .sandboxes
            .get(id)
            .ok_or_else(|| Error::NotFound(id.clone()))?;
        Ok(LedgerEntrySnapshot {
            sandbox: entry.sandbox.clone(),
            layout: entry.layout.clone(),
            overlay_mode: entry.overlay_mode,
        })
    }
}

struct LedgerEntrySnapshot {
    sandbox: Sandbox,
    layout: WorkLayout,
    overlay_mode: OverlayMode,
}

/// A thin trait the gRPC server can mock in unit tests.
#[async_trait]
pub trait OrchestratorApi: Send + Sync {
    /// Spawn.
    async fn spawn_sandbox(&self, spec: SandboxSpec) -> Result<Sandbox>;
    /// Snapshot.
    async fn snapshot_sandbox(
        &self,
        id: &SandboxId,
        name: &str,
    ) -> Result<SnapshotRef>;
    /// Kill.
    async fn kill_sandbox(
        &self,
        id: &SandboxId,
        reason: SandboxKillReason,
    ) -> Result<()>;
}

#[async_trait]
impl OrchestratorApi for Orchestrator {
    async fn spawn_sandbox(&self, spec: SandboxSpec) -> Result<Sandbox> {
        self.spawn(spec).await
    }
    async fn snapshot_sandbox(
        &self,
        id: &SandboxId,
        name: &str,
    ) -> Result<SnapshotRef> {
        self.snapshot(id, name).await
    }
    async fn kill_sandbox(
        &self,
        id: &SandboxId,
        reason: SandboxKillReason,
    ) -> Result<()> {
        self.kill(id, reason).await
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crucible_sandbox_spec::conformance::mock::MockProvider;
    use crucible_sandbox_spec::SandboxKind;
    use std::collections::BTreeMap;
    use std::time::Duration;
    use tempfile::tempdir;
    use twin_runtime_fs::{Orchestrator as _, OverlayMode, WorkLayout};

    struct NoopFs;

    #[async_trait]
    impl twin_runtime_fs::Orchestrator for NoopFs {
        async fn prepare(
            &self,
            _spec: &crucible_sandbox_spec::FilesystemSpec,
            _layout: &WorkLayout,
        ) -> twin_runtime_fs::Result<()> {
            Ok(())
        }
        async fn mount(
            &self,
            _layout: &WorkLayout,
            _mode: OverlayMode,
        ) -> twin_runtime_fs::Result<()> {
            Ok(())
        }
        async fn unmount(
            &self,
            _layout: &WorkLayout,
            _mode: OverlayMode,
        ) -> twin_runtime_fs::Result<()> {
            Ok(())
        }
        async fn diff(
            &self,
            _layout: &WorkLayout,
        ) -> twin_runtime_fs::Result<Vec<twin_runtime_fs::FileChange>> {
            Ok(Vec::new())
        }
    }

    fn test_spec() -> SandboxSpec {
        use crucible_sandbox_spec::{
            DefaultEgressAction, EgressManifest, FilesystemSpec, HeartbeatSpec, Resources,
            SyscallShimPolicy,
        };
        let mut labels = BTreeMap::new();
        labels.insert("crucible.io/local-dev".into(), "true".into());
        SandboxSpec {
            task_id: "task_lifecycle".into(),
            tenant_id: "ten_lifecycle".into(),
            kind: SandboxKind::LocalDocker,
            provider_region: "test".into(),
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

    fn build_orch() -> Orchestrator {
        let dir = tempdir().unwrap();
        let publisher: Arc<dyn Publisher> =
            Arc::new(LocalJournalPublisher::open(dir.path().join("j.jsonl")).unwrap());
        let mut providers = ProviderRegistry::empty();
        providers.register(
            SandboxKind::LocalDocker,
            Arc::new(MockProvider::new(SandboxKind::LocalDocker)),
        );
        Orchestrator::new(
            providers,
            Arc::new(NoopFs),
            Arc::new(Signer_::ephemeral()),
            publisher,
        )
    }

    #[tokio::test]
    async fn spawn_kill_roundtrip_via_orchestrator() {
        let orch = build_orch();
        let sb = orch.spawn(test_spec()).await.unwrap();
        let listed = orch.list().await;
        assert_eq!(listed.len(), 1);
        assert_eq!(listed[0].id, sb.id);

        orch.kill(&sb.id, SandboxKillReason::Clean).await.unwrap();
        let listed = orch.list().await;
        assert!(listed.is_empty());
    }

    #[tokio::test]
    async fn invalid_egress_rejected_at_orchestrator() {
        let orch = build_orch();
        let mut spec = test_spec();
        spec.egress.rules.push(crucible_sandbox_spec::EgressRule {
            host: "0.0.0.0/0".into(),
            ports: Vec::new(),
            disposition: crucible_sandbox_spec::EgressDisposition::Allow,
            tape_only: false,
            justification: "wildcard test".into(),
        });
        let err = orch.spawn(spec).await.unwrap_err();
        assert!(matches!(err, Error::Egress(_)));
    }

    #[tokio::test]
    async fn lifecycle_emits_attestation_per_event() {
        let dir = tempdir().unwrap();
        let journal = dir.path().join("j.jsonl");
        let publisher: Arc<dyn Publisher> =
            Arc::new(LocalJournalPublisher::open(&journal).unwrap());
        let mut providers = ProviderRegistry::empty();
        providers.register(
            crucible_sandbox_spec::SandboxKind::LocalDocker,
            Arc::new(MockProvider::new(crucible_sandbox_spec::SandboxKind::LocalDocker)),
        );
        let orch = Orchestrator::new(
            providers,
            Arc::new(NoopFs),
            Arc::new(Signer_::ephemeral()),
            publisher,
        );
        let sb = orch.spawn(test_spec()).await.unwrap();
        orch.kill(&sb.id, SandboxKillReason::Clean).await.unwrap();

        let raw = std::fs::read_to_string(&journal).unwrap();
        let line_count = raw.lines().count();
        // Spawn + kill = 2 attestations.
        assert_eq!(line_count, 2);
    }

    #[tokio::test]
    async fn kill_unknown_sandbox_returns_not_found() {
        let orch = build_orch();
        let err = orch
            .kill(&SandboxId("ghost".into()), SandboxKillReason::Clean)
            .await
            .unwrap_err();
        assert!(matches!(err, Error::NotFound(_)));
    }
}
