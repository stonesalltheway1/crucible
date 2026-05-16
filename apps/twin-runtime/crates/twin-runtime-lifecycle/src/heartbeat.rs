//! Heartbeat tracker — kills sandboxes whose agent process has stopped
//! reporting via `/work/.crucible/heartbeat`.
//!
//! The runtime spawns one heartbeat tracker task per sandbox at spawn time.
//! Per [`crucible_sandbox_spec::HeartbeatSpec`] the tracker expects a
//! heartbeat every `interval`; after `stale_after` of silence it triggers a
//! kill with [`crucible_sandbox_spec::SandboxKillReason::HeartbeatLost`].

use crucible_sandbox_spec::{HeartbeatSpec, SandboxId, SandboxKillReason};
use std::sync::Arc;
use std::time::{Duration, Instant};
use tokio::sync::Mutex;
use tokio::task::JoinHandle;

use crate::Orchestrator;

/// One tracker instance.
pub struct Tracker {
    sandbox_id: SandboxId,
    spec: HeartbeatSpec,
    last_beat: Arc<Mutex<Instant>>,
    handle: Option<JoinHandle<()>>,
}

impl Tracker {
    /// Spawn the tracker. Sends the kill request to `orch` when the
    /// sandbox goes silent for `spec.stale_after`.
    pub fn spawn(
        orch: Arc<Orchestrator>,
        sandbox_id: SandboxId,
        spec: HeartbeatSpec,
    ) -> Self {
        let last_beat = Arc::new(Mutex::new(Instant::now()));
        let handle = tokio::spawn(monitor(orch, sandbox_id.clone(), spec, last_beat.clone()));
        Self {
            sandbox_id,
            spec,
            last_beat,
            handle: Some(handle),
        }
    }

    /// Record a heartbeat tick.
    pub async fn tick(&self) {
        *self.last_beat.lock().await = Instant::now();
    }

    /// Stop the tracker.
    pub fn stop(mut self) {
        if let Some(h) = self.handle.take() {
            h.abort();
        }
    }

    /// Returns the sandbox the tracker is bound to.
    pub fn sandbox_id(&self) -> &SandboxId {
        &self.sandbox_id
    }

    /// Returns the spec the tracker is using.
    pub fn spec(&self) -> HeartbeatSpec {
        self.spec
    }
}

async fn monitor(
    orch: Arc<Orchestrator>,
    sandbox_id: SandboxId,
    spec: HeartbeatSpec,
    last_beat: Arc<Mutex<Instant>>,
) {
    let mut ticker = tokio::time::interval(Duration::from_millis(
        (spec.interval.as_millis() as u64 / 2).max(500),
    ));
    loop {
        ticker.tick().await;
        let elapsed = {
            let lb = last_beat.lock().await;
            lb.elapsed()
        };
        if elapsed > spec.stale_after {
            tracing::error!(
                sandbox = %sandbox_id,
                ?elapsed,
                "heartbeat lost — killing sandbox"
            );
            let _ = orch
                .kill(&sandbox_id, SandboxKillReason::HeartbeatLost)
                .await;
            break;
        }
    }
}
