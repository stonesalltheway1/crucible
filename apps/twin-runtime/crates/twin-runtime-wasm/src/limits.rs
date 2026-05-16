//! Resource quotas (memory / fuel / wall-clock) wrapping Wasmtime's
//! `ResourceLimiter` + fuel + epoch-interruption primitives.
//!
//! Phase 3 picks:
//! - `epoch_interruption` over `fuel` for tool runs — fuel ticks add
//!   2–3× overhead; epoch interruption is the lighter cost-effective
//!   path for cancellation.
//! - Per-Store `ResourceLimiter` for memory/table growth caps.
//! - Wall-clock budget enforced at the runner level via tokio.

use serde::{Deserialize, Serialize};
use std::time::Duration;

use crate::capabilities::MemoryCapability;

/// Quotas applied per-invocation.
#[derive(Debug, Clone, Copy, Serialize, Deserialize)]
pub struct ResourceQuota {
    /// Wall-clock budget. Default 30s.
    pub wall_clock: Duration,
    /// Fuel budget (None = use epoch interruption only). When set, the
    /// runner installs fuel and aborts when exhausted; useful for
    /// deterministic execution at higher cost.
    pub fuel: Option<u64>,
    /// Memory caps. Default is [`MemoryCapability::default()`].
    pub memory: MemoryCapability,
    /// Hard cap on the number of host-call invocations the WASM module
    /// can make. Default 10 000. Prevents pathological loops calling
    /// host functions.
    pub max_host_calls: u64,
}

impl Default for ResourceQuota {
    fn default() -> Self {
        Self {
            wall_clock: Duration::from_secs(30),
            fuel: None,
            memory: MemoryCapability::default(),
            max_host_calls: 10_000,
        }
    }
}

/// Per-invocation usage telemetry. Populated by the runner.
#[derive(Debug, Clone, Default, Serialize, Deserialize)]
pub struct ResourceUsage {
    /// Wall-clock spent executing.
    pub wall_clock: Duration,
    /// Peak linear-memory bytes the module ever held.
    pub peak_memory_bytes: usize,
    /// Host-call invocations counted at the trampoline.
    pub host_calls: u64,
    /// Fuel consumed (only populated when `fuel` is set).
    pub fuel_consumed: u64,
    /// Did the module hit any quota?
    pub trip: Option<QuotaTrip>,
}

/// Which quota tripped if the invocation was aborted.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
pub enum QuotaTrip {
    /// Wall-clock budget exceeded.
    WallClock,
    /// Fuel budget exceeded.
    Fuel,
    /// Memory growth refused by `ResourceLimiter`.
    Memory,
    /// Host-call count exceeded `max_host_calls`.
    HostCalls,
}
