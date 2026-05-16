//! Host-granted capability set for a WASM tool invocation.
//!
//! Default is the **empty** capability set: no fs, no net, no env, no
//! command-line args. Callers grant capabilities one at a time. Per the
//! NVIDIA practical-security-guidance paper, the threat model assumes
//! the WASM module is hostile; every capability must be a positive
//! decision.

use serde::{Deserialize, Serialize};
use std::path::PathBuf;

/// Top-level capability bundle passed to [`crate::ToolRunner::run`].
#[derive(Debug, Clone, Default, Serialize, Deserialize)]
pub struct Capabilities {
    /// Filesystem preopens. Empty = no fs access.
    pub fs: Vec<FsCapability>,

    /// Network egress allow-list. Empty = no net access.
    ///
    /// Phase 3 ships network capabilities as a hard NO — the host doesn't
    /// grant socket access to WASM modules under any circumstance. The
    /// field exists so the API can evolve forward without a breaking
    /// change, but `ToolRunner` returns an error if any [`NetCapability`]
    /// is non-`None`.
    pub net: Vec<NetCapability>,

    /// Memory cap. None = use [`MemoryCapability::default()`].
    pub memory: Option<MemoryCapability>,

    /// Environment variable allow-list. Empty = NO env. Inheritance is
    /// never automatic; explicit names only.
    pub env: Vec<EnvAllow>,

    /// Command-line argv passed to the WASM module's WASI start
    /// function. Empty = `[]`.
    pub argv: Vec<String>,
}

/// One filesystem preopen.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct FsCapability {
    /// Path inside the WASM module.
    pub guest_path: String,
    /// Path on the host (inside the microVM).
    pub host_path: PathBuf,
    /// Read-only or read-write.
    pub mode: FsMode,
}

/// FS access mode.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
pub enum FsMode {
    /// Read-only preopen. Default for tape-fixture mounts.
    ReadOnly,
    /// Read-write preopen. Should only be granted for /work/scratch-style
    /// dirs that get nuked at task end.
    ReadWrite,
}

/// One network capability. **Phase 3 does not honour any non-`None`
/// variant** — declared so the type can grow without a breaking change.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum NetCapability {
    /// Reserved. Phase 3 always denies.
    OutboundHTTP { host: String, port: u16 },
}

/// One env var allow.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct EnvAllow {
    /// Variable name (e.g., `"LANG"`).
    pub name: String,
    /// Value to set. The host does NOT read its own environment.
    pub value: String,
}

/// Memory + table caps.
#[derive(Debug, Clone, Copy, Serialize, Deserialize)]
pub struct MemoryCapability {
    /// Max linear memory in bytes. Default 256 MiB.
    pub max_memory_bytes: usize,
    /// Max table elements. Default 10 000.
    pub max_table_elements: usize,
    /// Max simultaneous instances per Store. Default 1.
    pub max_instances: usize,
}

impl Default for MemoryCapability {
    fn default() -> Self {
        Self {
            max_memory_bytes: 256 * 1024 * 1024,
            max_table_elements: 10_000,
            max_instances: 1,
        }
    }
}

impl Capabilities {
    /// Empty capability set — the baseline (deny by default).
    #[must_use]
    pub const fn empty() -> Self {
        Self {
            fs: Vec::new(),
            net: Vec::new(),
            memory: None,
            env: Vec::new(),
            argv: Vec::new(),
        }
    }

    /// Returns true iff any non-empty net capability is requested. Phase 3
    /// rejects any such request at runner boot.
    #[must_use]
    pub fn requests_net(&self) -> bool {
        !self.net.is_empty()
    }
}
