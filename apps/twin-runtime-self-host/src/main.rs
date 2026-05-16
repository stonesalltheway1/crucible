//! Crucible self-hosted twin-runtime orchestrator.
//!
//! Binary entry point. Wires `tracing-subscriber`, parses the
//! configuration file, builds the orchestrator, and runs the gRPC
//! sandbox-provider server.

#![forbid(unsafe_code)]

mod cgroups;
mod firecracker;
mod network;
mod pool;
mod provider;
mod zfs;

use std::env;
use std::path::PathBuf;

use anyhow::{Context, Result};
use tracing::{info, warn};
use tracing_subscriber::{prelude::*, EnvFilter};

use crate::provider::{Orchestrator, OrchestratorConfig};

#[tokio::main]
async fn main() -> Result<()> {
    init_tracing();

    let cfg_path = env::var_os("CRUCIBLE_SELF_HOST_CONFIG")
        .map(PathBuf::from)
        .unwrap_or_else(|| PathBuf::from("/etc/crucible/self-host.yaml"));
    let cfg = OrchestratorConfig::load(&cfg_path)
        .with_context(|| format!("load config from {}", cfg_path.display()))?;
    info!(?cfg.host_id, "starting crucible-twin-self-host");

    #[cfg(not(feature = "linux-firecracker"))]
    warn!(
        "compiled without `linux-firecracker` feature — spawn paths will \
         return Error::PhaseStub. Enable the feature for production builds."
    );

    let orch = Orchestrator::new(cfg).await?;
    orch.run().await?;
    Ok(())
}

fn init_tracing() {
    let filter =
        EnvFilter::try_from_default_env().unwrap_or_else(|_| EnvFilter::new("info,twin_runtime=debug"));
    tracing_subscriber::registry()
        .with(filter)
        .with(tracing_subscriber::fmt::layer().with_target(true).json())
        .init();
}
