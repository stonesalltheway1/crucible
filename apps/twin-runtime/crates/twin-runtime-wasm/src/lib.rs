//! WASM tool runner.
//!
//! The outer Crucible sandbox is a Firecracker microVM (Phase 2). For
//! short-running tools the agent invokes (`cargo`, `npm`, `pip`,
//! LLM-generated shell-equivalent code), spawning a full microVM per
//! invocation is wasteful — and worse, the LLM-generated code is the
//! inner-layer threat model in the NVIDIA "Sandboxing Agentic AI
//! Workflows with WebAssembly" (Dec 2024) and "Practical Security
//! Guidance" (Dec 2025) papers.
//!
//! Phase 3 ships a Wasmtime-embedded second sandbox INSIDE the microVM:
//!
//! - Capability model: WASI Preview 2 (production-stable as of 2025).
//!   No fs / net / env access unless the host explicitly preopens it.
//! - Resource limits via the `ResourceLimiter` trait + fuel + epoch
//!   interruption.
//! - Pooling allocator across invocations so cold-start is amortised.
//! - All tool input/output flows via host-provided typed component
//!   imports — there is no inherited stdio, no Wasi-net socket capability,
//!   no `inherit_env`.
//!
//! See `tests/containment.rs` for the adversarial corpus + the proptest
//! that asserts containment over 10 000 random LLM-generated module
//! attempts.

#![forbid(unsafe_code)]
#![allow(clippy::module_name_repetitions)]

pub mod capabilities;
pub mod limits;
pub mod runner;

pub use capabilities::{Capabilities, FsCapability, MemoryCapability, NetCapability};
pub use limits::{ResourceQuota, ResourceUsage};
pub use runner::{ExecutionReport, ToolRunner, ToolRunnerError, ToolSpec};

/// Crate version stamped on every [`ExecutionReport`].
pub const RUNNER_VERSION: &str = env!("CARGO_PKG_VERSION");
