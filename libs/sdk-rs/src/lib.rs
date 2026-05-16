//! Crucible Agent SDK — Rust.
//!
//! Phase 2 adds the agent-side `twin.*` runtime client. Phase 1 ships types
//! only; the runtime `twin.*` surface lands here in Phase 2. Types mirror
//! the protobuf source-of-truth in `libs/twin-spec/proto/crucible/v1/`.

pub mod types;
pub mod twin;

pub const SDK_VERSION: &str = "2026.6.0-phase2";

pub use types::*;
