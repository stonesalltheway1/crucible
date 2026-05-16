//! Conformance corpus every [`crate::SandboxProvider`] implementation must
//! pass. Concrete implementations (E2B, raw-Firecracker, Modal, ...) include
//! this crate with `features = ["conformance"]` and call
//! [`run_conformance`] from their integration tests.
//!
//! The corpus deliberately uses small, fast specs that don't actually need
//! to do real work — providers should mock external dependencies in their
//! test harness (e.g., the E2B driver tests run against a stubbed HTTP
//! server). Real-cloud integration tests are separate and env-gated.

use crate::{
    DefaultEgressAction, EgressManifest, FilesystemSpec, HeartbeatSpec, Resources, SandboxId,
    SandboxKillReason, SandboxKind, SandboxProvider, SandboxSpec, SandboxState, SyscallShimPolicy,
};
use std::collections::BTreeMap;
use std::time::Duration;

/// Runs the full conformance corpus against `provider`.
///
/// Each check is independent; failures are reported via `panic!` so any
/// failure surfaces as a test failure with backtrace.
///
/// # Errors
///
/// Returns the first error from a provider call that isn't expected to
/// fail. Expected failures (e.g. `kill` after `kill`) are asserted in-place.
pub async fn run_conformance<P: SandboxProvider>(provider: &P) -> crate::Result<()> {
    check_kind_consistent(provider);
    check_capabilities_cheap(provider);
    check_invalid_spec_rejected(provider).await?;
    check_spawn_and_kill_roundtrip(provider).await?;
    check_kill_is_idempotent(provider).await?;
    check_state_after_kill(provider).await?;
    check_list_returns_recent_spawn(provider).await?;
    if provider.capabilities().supports_snapshot {
        check_snapshot_roundtrip(provider).await?;
    }
    Ok(())
}

fn check_kind_consistent<P: SandboxProvider>(provider: &P) {
    let k = provider.kind();
    // Calling kind twice must give the same answer — trait method is read-only.
    assert_eq!(provider.kind(), k, "provider.kind() must be deterministic");
}

fn check_capabilities_cheap<P: SandboxProvider>(provider: &P) {
    // We can't directly assert "cheap" but we can assert successive calls
    // return identical structs.
    let c1 = provider.capabilities();
    let c2 = provider.capabilities();
    assert_eq!(c1, c2, "provider.capabilities() must be stable");
}

async fn check_invalid_spec_rejected<P: SandboxProvider>(provider: &P) -> crate::Result<()> {
    // Empty task id is invalid at the spec layer; provider should not be
    // contacted, but if it is, it should still refuse.
    let mut spec = minimal_spec(provider.kind());
    spec.task_id.clear();
    let result = provider.spawn(&spec).await;
    assert!(
        result.is_err(),
        "spawn must reject empty task_id, got {:?}",
        result.ok().map(|s| s.id)
    );
    Ok(())
}

async fn check_spawn_and_kill_roundtrip<P: SandboxProvider>(provider: &P) -> crate::Result<()> {
    let spec = minimal_spec(provider.kind());
    let sandbox = provider.spawn(&spec).await?;
    assert!(
        !sandbox.state.is_terminal(),
        "fresh sandbox must not start terminal"
    );
    assert_eq!(
        sandbox.spec_hash, spec.canonical_hash(),
        "sandbox must echo the spec hash it was spawned from"
    );
    provider.kill(&sandbox, SandboxKillReason::Clean).await?;
    Ok(())
}

async fn check_kill_is_idempotent<P: SandboxProvider>(provider: &P) -> crate::Result<()> {
    let spec = minimal_spec(provider.kind());
    let sandbox = provider.spawn(&spec).await?;
    provider.kill(&sandbox, SandboxKillReason::Clean).await?;
    provider.kill(&sandbox, SandboxKillReason::Clean).await?;
    Ok(())
}

async fn check_state_after_kill<P: SandboxProvider>(provider: &P) -> crate::Result<()> {
    let spec = minimal_spec(provider.kind());
    let sandbox = provider.spawn(&spec).await?;
    provider.kill(&sandbox, SandboxKillReason::Clean).await?;
    let state = provider.state(&sandbox.id).await?;
    assert!(
        state.is_terminal(),
        "state after kill must be terminal, got {state:?}"
    );
    Ok(())
}

async fn check_list_returns_recent_spawn<P: SandboxProvider>(provider: &P) -> crate::Result<()> {
    let spec = minimal_spec(provider.kind());
    let sandbox = provider.spawn(&spec).await?;
    let listed = provider.list(&spec.tenant_id).await?;
    let found = listed.iter().any(|s| s.id == sandbox.id);
    assert!(
        found,
        "newly-spawned sandbox {:?} must appear in list({}), got {} entries",
        sandbox.id,
        spec.tenant_id,
        listed.len()
    );
    provider.kill(&sandbox, SandboxKillReason::Clean).await?;
    Ok(())
}

async fn check_snapshot_roundtrip<P: SandboxProvider>(provider: &P) -> crate::Result<()> {
    let spec = minimal_spec(provider.kind());
    let sandbox = provider.spawn(&spec).await?;
    let snap = provider.snapshot(&sandbox, "conformance-checkpoint").await?;
    assert_eq!(snap.sandbox_id, sandbox.id);
    assert_eq!(snap.base_spec_hash, sandbox.spec_hash);
    if provider.capabilities().supports_restore {
        let restored = provider.restore(&snap, None).await?;
        assert!(!restored.state.is_terminal());
        provider.kill(&restored, SandboxKillReason::Clean).await?;
    }
    provider.kill(&sandbox, SandboxKillReason::Clean).await?;
    Ok(())
}

/// Builds a minimal valid spec for conformance tests. Always uses the
/// `crucible.io/local-dev=true` label so `LocalDocker` is accepted as the
/// kind in tests without breaking the tenant-traffic invariant.
fn minimal_spec(kind: SandboxKind) -> SandboxSpec {
    let mut labels = BTreeMap::new();
    labels.insert("crucible.io/local-dev".into(), "true".into());
    SandboxSpec {
        task_id: "task_conformance".into(),
        tenant_id: "ten_conformance".into(),
        kind,
        provider_region: "test".into(),
        resources: Resources::default(),
        egress: EgressManifest {
            rules: Vec::new(),
            default_action: DefaultEgressAction::Deny,
        },
        secrets: Vec::new(),
        db: None,
        filesystem: FilesystemSpec {
            base_sha: "0000000000000000".into(),
            repo_url: "https://example.invalid/repo".into(),
            depth: 1,
            overlay_mode: "copy".into(),
            prewarm_paths: Vec::new(),
        },
        tape: None,
        shim: SyscallShimPolicy::default(),
        heartbeat: HeartbeatSpec::default(),
        absolute_ttl: Duration::from_secs(300),
        labels,
    }
}

/// A trivial in-memory provider for unit-testing higher layers without
/// touching any cloud. Behaviour is deterministic and passes
/// [`run_conformance`].
pub mod mock {
    use super::*;
    use crate::{
        ProviderCapabilities, Result, Sandbox, SandboxState, SnapshotId, SnapshotRef, SpecHash,
    };
    use async_trait::async_trait;
    use chrono::Utc;
    use std::sync::Mutex;

    /// In-memory provider for unit tests.
    pub struct MockProvider {
        state: Mutex<MockState>,
        kind: SandboxKind,
        caps: ProviderCapabilities,
    }

    struct MockState {
        next_id: u64,
        sandboxes: Vec<(Sandbox, SandboxState)>,
        snapshots: Vec<SnapshotRef>,
    }

    impl MockProvider {
        /// Construct a `MockProvider` for the given kind.
        #[must_use]
        pub fn new(kind: SandboxKind) -> Self {
            Self {
                kind,
                caps: ProviderCapabilities {
                    supports_snapshot: true,
                    supports_restore: true,
                    supports_pause: true,
                    supports_native_egress: false,
                    supports_guest_ebpf: false,
                    max_concurrent: None,
                },
                state: Mutex::new(MockState {
                    next_id: 0,
                    sandboxes: Vec::new(),
                    snapshots: Vec::new(),
                }),
            }
        }
    }

    #[async_trait]
    impl SandboxProvider for MockProvider {
        fn kind(&self) -> SandboxKind {
            self.kind
        }

        fn capabilities(&self) -> ProviderCapabilities {
            self.caps
        }

        async fn spawn(&self, spec: &SandboxSpec) -> Result<Sandbox> {
            spec.validate()?;
            let mut state = self.state.lock().expect("mock state poisoned");
            state.next_id += 1;
            let id = SandboxId(format!("mock-{}", state.next_id));
            let now = Utc::now();
            let sandbox = Sandbox {
                id: id.clone(),
                task_id: spec.task_id.clone(),
                tenant_id: spec.tenant_id.clone(),
                kind: spec.kind,
                provider_handle: format!("mock://{id}"),
                control_endpoint: format!("unix:/tmp/mock-{id}.sock"),
                spawned_at: now,
                expires_at: now + chrono::Duration::from_std(spec.absolute_ttl).unwrap(),
                state: SandboxState::Ready,
                attestation_socket: format!("/tmp/mock-{id}/attest.sock"),
                spec_hash: spec.canonical_hash(),
            };
            state.sandboxes.push((sandbox.clone(), SandboxState::Ready));
            Ok(sandbox)
        }

        async fn snapshot(&self, sandbox: &Sandbox, name: &str) -> Result<SnapshotRef> {
            let mut state = self.state.lock().expect("mock state poisoned");
            if !state.sandboxes.iter().any(|(s, _)| s.id == sandbox.id) {
                return Err(crate::Error::NotFound(sandbox.id.clone()));
            }
            state.next_id += 1;
            let snap = SnapshotRef {
                id: SnapshotId(format!("mock-snap-{}", state.next_id)),
                sandbox_id: sandbox.id.clone(),
                task_id: sandbox.task_id.clone(),
                name: name.to_string(),
                taken_at: Utc::now(),
                provider_handle: format!("mock://snap-{}", state.next_id),
                size_bytes: 4096,
                base_spec_hash: sandbox.spec_hash.clone(),
                attestation_chain_head: None,
            };
            state.snapshots.push(snap.clone());
            Ok(snap)
        }

        async fn restore(
            &self,
            snapshot: &SnapshotRef,
            new_task_id: Option<&str>,
        ) -> Result<Sandbox> {
            let mut state = self.state.lock().expect("mock state poisoned");
            if !state.snapshots.iter().any(|s| s.id == snapshot.id) {
                return Err(crate::Error::SnapshotNotFound(snapshot.id.clone()));
            }
            state.next_id += 1;
            let id = SandboxId(format!("mock-{}", state.next_id));
            let now = Utc::now();
            let sandbox = Sandbox {
                id: id.clone(),
                task_id: new_task_id.unwrap_or(&snapshot.task_id).to_string(),
                tenant_id: "ten_conformance".into(),
                kind: self.kind,
                provider_handle: format!("mock://restore-{id}"),
                control_endpoint: format!("unix:/tmp/mock-{id}.sock"),
                spawned_at: now,
                expires_at: now + chrono::Duration::hours(1),
                state: SandboxState::Booting,
                attestation_socket: format!("/tmp/mock-{id}/attest.sock"),
                spec_hash: snapshot.base_spec_hash.clone(),
            };
            state.sandboxes.push((sandbox.clone(), SandboxState::Booting));
            Ok(sandbox)
        }

        async fn kill(&self, sandbox: &Sandbox, _reason: SandboxKillReason) -> Result<()> {
            let mut state = self.state.lock().expect("mock state poisoned");
            for (s, st) in &mut state.sandboxes {
                if s.id == sandbox.id {
                    *st = SandboxState::Terminated;
                }
            }
            Ok(())
        }

        async fn state(&self, id: &SandboxId) -> Result<SandboxState> {
            let state = self.state.lock().expect("mock state poisoned");
            state
                .sandboxes
                .iter()
                .find(|(s, _)| s.id == *id)
                .map(|(_, st)| *st)
                .ok_or_else(|| crate::Error::NotFound(id.clone()))
        }

        async fn list(&self, tenant_id: &str) -> Result<Vec<Sandbox>> {
            let state = self.state.lock().expect("mock state poisoned");
            Ok(state
                .sandboxes
                .iter()
                .filter(|(s, _)| s.tenant_id == tenant_id)
                .map(|(s, _)| s.clone())
                .collect())
        }
    }
}

/// Convenience: indirectly export `SpecHash` so providers can avoid importing
/// it from the crate root.
pub use crate::SpecHash as ProviderSpecHash;
