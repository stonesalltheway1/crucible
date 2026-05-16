//! Raw-Firecracker provider — Phase 3 surface.
//!
//! Self-hosted enterprise tier per ADR-015. Phase 2 ships the typed stub so
//! downstream code can wire against the SandboxProvider trait; the actual
//! orchestrator (containerd + ZFS + Cilium/Tetragon on the host) lands in
//! Phase 3 alongside the multi-region scheduling work.

use async_trait::async_trait;
use crucible_sandbox_spec::{
    Error, ProviderCapabilities, Result, Sandbox, SandboxId, SandboxKillReason, SandboxKind,
    SandboxProvider, SandboxSpec, SandboxState, SnapshotRef,
};

/// Raw-Firecracker provider. All operations return [`Error::PhaseStub`].
pub struct RawFirecrackerProvider;

impl RawFirecrackerProvider {
    /// Construct.
    #[must_use]
    pub fn new() -> Self {
        Self
    }
}

impl Default for RawFirecrackerProvider {
    fn default() -> Self {
        Self::new()
    }
}

const STUB_MSG: &str = "raw-Firecracker provider is Phase 3. \
    See docs/08-phase-prompts/phase-03-twin-runtime-breadth.md.";

#[async_trait]
impl SandboxProvider for RawFirecrackerProvider {
    fn kind(&self) -> SandboxKind {
        SandboxKind::RawFirecracker
    }

    fn capabilities(&self) -> ProviderCapabilities {
        // Per ADR-015, raw-Firecracker will have full feature parity plus
        // guest eBPF (host attaches Tetragon).
        ProviderCapabilities {
            supports_snapshot: true,
            supports_restore: true,
            supports_pause: true,
            supports_native_egress: true,
            supports_guest_ebpf: true,
            max_concurrent: None,
        }
    }

    async fn spawn(&self, _spec: &SandboxSpec) -> Result<Sandbox> {
        Err(Error::PhaseStub(STUB_MSG.into()))
    }

    async fn snapshot(&self, _sandbox: &Sandbox, _name: &str) -> Result<SnapshotRef> {
        Err(Error::PhaseStub(STUB_MSG.into()))
    }

    async fn restore(
        &self,
        _snapshot: &SnapshotRef,
        _new_task_id: Option<&str>,
    ) -> Result<Sandbox> {
        Err(Error::PhaseStub(STUB_MSG.into()))
    }

    async fn kill(&self, _sandbox: &Sandbox, _reason: SandboxKillReason) -> Result<()> {
        Err(Error::PhaseStub(STUB_MSG.into()))
    }

    async fn state(&self, _id: &SandboxId) -> Result<SandboxState> {
        Err(Error::PhaseStub(STUB_MSG.into()))
    }

    async fn list(&self, _tenant_id: &str) -> Result<Vec<Sandbox>> {
        Err(Error::PhaseStub(STUB_MSG.into()))
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[tokio::test]
    async fn all_calls_return_phase_stub() {
        let p = RawFirecrackerProvider::new();
        assert_eq!(p.kind(), SandboxKind::RawFirecracker);
        assert!(p.capabilities().supports_guest_ebpf);
        let err = p.list("ten").await.unwrap_err();
        assert!(matches!(err, Error::PhaseStub(_)));
    }
}
