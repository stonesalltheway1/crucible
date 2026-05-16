//! Pre-warmed sandbox pool.
//!
//! Per ADR-015 operational notes, the self-host tier maintains a per-host
//! pre-warmed sandbox pool (default 20) to absorb burst. The pool's job is
//! "have a snapshot ready that another spawn can restore from in 3–10ms
//! warm".
//!
//! Phase 3 ships the bookkeeping; production deployments enable the
//! `linux-firecracker` feature to actually drive Firecracker snapshot
//! creation.

use std::collections::HashMap;
use std::path::PathBuf;
use std::sync::atomic::{AtomicU64, Ordering};

use crate::firecracker::FirecrackerHandle;

/// One pre-warmed slot.
pub struct WarmSlot {
    /// Firecracker handle (opaque to the orchestrator).
    pub handle: FirecrackerHandle,
    /// The snapshot id this slot was warmed from.
    pub snapshot_id: String,
    /// On-disk path of the snapshot.
    pub snapshot_path: PathBuf,
}

/// Per-host warm pool.
pub struct WarmPool {
    target_size: usize,
    slots: HashMap<String, Vec<WarmSlot>>, // spec_hash → slots
    seq: AtomicU64,
}

impl WarmPool {
    /// Construct with a per-spec target depth.
    pub fn new(target_size: usize) -> Self {
        Self {
            target_size,
            slots: HashMap::new(),
            seq: AtomicU64::new(0),
        }
    }

    /// Acquire a slot for a spec hash. Returns None if the pool is dry.
    pub fn try_acquire(&mut self, spec_hash: &str) -> Option<WarmSlot> {
        self.slots.get_mut(spec_hash).and_then(|v| v.pop())
    }

    /// Top up the pool to `target_size` for the given spec hash. Returns
    /// the number of slots ADDED (zero if already at target).
    pub fn ensure_warm(&mut self, spec_hash: &str, target: usize) -> usize {
        let entry = self.slots.entry(spec_hash.to_string()).or_default();
        let need = target.saturating_sub(entry.len());
        for _ in 0..need {
            let n = self.seq.fetch_add(1, Ordering::Relaxed);
            entry.push(WarmSlot {
                handle: FirecrackerHandle::new(format!("warm-{n}")),
                snapshot_id: format!("snap_{spec_hash}_{n}"),
                snapshot_path: PathBuf::from(format!("/var/lib/crucible/snaps/{spec_hash}_{n}")),
            });
        }
        need
    }

    /// Returns the current depth for a spec hash.
    pub fn depth(&self, spec_hash: &str) -> usize {
        self.slots.get(spec_hash).map(|v| v.len()).unwrap_or(0)
    }

    /// Returns the target depth.
    pub fn target_size(&self) -> usize {
        self.target_size
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn ensure_warm_tops_up_to_target() {
        let mut p = WarmPool::new(5);
        assert_eq!(p.ensure_warm("s", 3), 3);
        assert_eq!(p.depth("s"), 3);
        assert_eq!(p.ensure_warm("s", 3), 0);
        assert_eq!(p.ensure_warm("s", 5), 2);
        assert_eq!(p.depth("s"), 5);
    }

    #[test]
    fn try_acquire_decrements_depth() {
        let mut p = WarmPool::new(2);
        p.ensure_warm("s", 2);
        let _ = p.try_acquire("s").unwrap();
        assert_eq!(p.depth("s"), 1);
        let _ = p.try_acquire("s").unwrap();
        assert_eq!(p.depth("s"), 0);
        assert!(p.try_acquire("s").is_none());
    }
}
