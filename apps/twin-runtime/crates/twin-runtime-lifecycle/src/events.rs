//! Runtime event bus — exposes per-sandbox lifecycle events to the gRPC
//! `StreamEvents` RPC and the cost-meter / observer pipelines.

use chrono::{DateTime, Utc};
use crucible_sandbox_spec::{Sandbox, SandboxId, SandboxKillReason};
use tokio::sync::broadcast;

/// One emitted event.
#[derive(Debug, Clone)]
pub struct Event {
    /// When the event was emitted.
    pub at: DateTime<Utc>,
    /// Sandbox the event is about.
    pub sandbox_id: SandboxId,
    /// Task the sandbox served.
    pub task_id: String,
    /// Tenant.
    pub tenant_id: String,
    /// Event class.
    pub kind: EventKind,
    /// Reason — populated for `Killed` events.
    pub reason: Option<SandboxKillReason>,
}

impl Event {
    /// Build a non-kill event.
    #[must_use]
    pub fn new(kind: EventKind, sandbox: &Sandbox) -> Self {
        Self {
            at: Utc::now(),
            sandbox_id: sandbox.id.clone(),
            task_id: sandbox.task_id.clone(),
            tenant_id: sandbox.tenant_id.clone(),
            kind,
            reason: None,
        }
    }

    /// Build a kill event.
    #[must_use]
    pub fn new_kill(kind: EventKind, sandbox: &Sandbox, reason: SandboxKillReason) -> Self {
        Self {
            at: Utc::now(),
            sandbox_id: sandbox.id.clone(),
            task_id: sandbox.task_id.clone(),
            tenant_id: sandbox.tenant_id.clone(),
            kind,
            reason: Some(reason),
        }
    }
}

/// Event class — mirrors the `RuntimeEvent` proto oneof.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum EventKind {
    /// Sandbox spawned.
    Spawned,
    /// Sandbox snapshotted.
    Snapshot,
    /// Sandbox killed.
    Killed,
    /// Sandbox heartbeat lost.
    HeartbeatLost,
    /// Destructive operation intercepted.
    DestructiveIntercepted,
    /// Egress policy violation.
    EgressViolation,
    /// Resource cap exceeded.
    ResourceExceeded,
}

/// The bus.
#[derive(Clone)]
pub struct EventBus {
    inner: broadcast::Sender<Event>,
}

impl EventBus {
    /// Build a new bus with a given channel capacity.
    #[must_use]
    pub fn new(capacity: usize) -> Self {
        let (tx, _rx) = broadcast::channel(capacity);
        Self { inner: tx }
    }

    /// Subscribe to events.
    pub fn subscribe(&self) -> broadcast::Receiver<Event> {
        self.inner.subscribe()
    }

    /// Publish an event. Slow subscribers may drop entries; the runtime
    /// records `lagged` events at the consumer.
    pub async fn publish(&self, event: Event) {
        // best-effort; broadcast::send returns Err if there are no receivers.
        let _ = self.inner.send(event);
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[tokio::test]
    async fn publish_reaches_subscriber() {
        let bus = EventBus::new(16);
        let mut rx = bus.subscribe();
        let sandbox = Sandbox {
            id: SandboxId("sb_1".into()),
            task_id: "t".into(),
            tenant_id: "ten".into(),
            kind: crucible_sandbox_spec::SandboxKind::LocalDocker,
            provider_handle: "h".into(),
            control_endpoint: "u".into(),
            spawned_at: Utc::now(),
            expires_at: Utc::now() + chrono::Duration::seconds(60),
            state: crucible_sandbox_spec::SandboxState::Ready,
            attestation_socket: "s".into(),
            spec_hash: crucible_sandbox_spec::SpecHash(String::new()),
        };
        bus.publish(Event::new(EventKind::Spawned, &sandbox)).await;
        let got = rx.recv().await.unwrap();
        assert_eq!(got.sandbox_id.0, "sb_1");
        assert_eq!(got.kind, EventKind::Spawned);
    }
}
