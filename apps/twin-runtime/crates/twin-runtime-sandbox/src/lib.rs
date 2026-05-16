//! SandboxProvider implementations.
//!
//! Phase 2 ships the **E2B driver** as a first-class provider and a
//! **raw-Firecracker stub** that returns typed `STUB:` errors pointing at
//! Phase 3. Future drivers (Modal, Daytona, Fly Machines) plug in through
//! the same trait surface.
//!
//! Both drivers honour the threat-model invariant that real production
//! credentials are unreachable from the agent process — providers never
//! receive Crucible's master tenant API keys; only short-lived,
//! Infisical-issued tokens scoped to one sandbox.

#![warn(missing_docs)]

pub mod e2b;
pub mod raw_firecracker;
pub mod registry;

pub use registry::ProviderRegistry;
