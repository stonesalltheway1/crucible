//! Tetragon TracingPolicy renderer — production-tier enforcement.
//!
//! Reuses [`twin_runtime_shim::tetragon::render_egress_tracing_policy`] as
//! the canonical renderer (defined in the shim crate to keep its policy
//! emission paths next to its consumer). This module is a thin re-export
//! so consumers don't reach across crate boundaries.

pub use twin_runtime_shim::tetragon::render_egress_tracing_policy;
