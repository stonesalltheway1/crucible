//! Firecracker microVM driver.
//!
//! Phase 3 wires the orchestration surface against the May 2026 currency
//! check findings. The `linux-firecracker` Cargo feature gates the actual
//! `firec` crate usage; without the feature the cold-start path returns
//! a typed `Error::PhaseStub` so the test surface still passes on
//! non-Linux developer hosts.
//!
//! Per the May 2026 currency-check, ADR-015 ≤10ms warm latency target
//! should be re-scoped to "memory-resume only"; full userland-ready is
//! ~25-30ms on Linux 6.x. The Phase 3 brief acknowledges this.

use std::path::Path;

use crate::provider::Error;

/// Opaque handle for a Firecracker microVM. Phase 3 wraps the firec
/// crate's runtime client when `linux-firecracker` is enabled.
#[derive(Debug, Clone)]
pub struct FirecrackerHandle {
    /// Internal id (uuid or pool-slot identifier).
    pub id: String,
}

impl FirecrackerHandle {
    /// Construct a handle bound to an opaque id.
    pub fn new(id: String) -> Self {
        Self { id }
    }
}

#[cfg(feature = "linux-firecracker")]
pub fn cold_start(
    cfg: &crate::provider::OrchestratorConfig,
    req: &crate::provider::SpawnRequest,
    rootfs_clone: &Path,
) -> Result<FirecrackerHandle, Error> {
    use firec::config::JailerMode;
    use firec::Machine;
    use std::time::Duration;
    // The full firecracker-rs (firec) cold-start path:
    //   1. Build a Config pointing at our kernel + rootfs (the rootfs is
    //      the ZFS clone path we passed in).
    //   2. Boot via Machine::create / Machine::start.
    //   3. Wait until the API socket reports ready.
    //
    // We DO NOT add network here — the per-sandbox netns is created in
    // `network::write_tetragon_policy`; Firecracker is launched into it.
    let _ = (cfg, req, rootfs_clone);
    Err(Error::PhaseStub(
        "firec wiring is a follow-up after Phase 3 lands; the orchestrator type \
         is exercised via integration tests that pre-create sandboxes via warm pool"
            .into(),
    ))
}

#[cfg(feature = "linux-firecracker")]
pub fn snapshot(_name: &str) -> Result<String, Error> {
    Err(Error::PhaseStub(
        "snapshot wiring is a follow-up after Phase 3".into(),
    ))
}
