//! Crucible Twin Runtime — gRPC server binary.
//!
//! Entry point. Parses config from env + optional `twin-runtime.toml`,
//! builds the lifecycle orchestrator, and exposes
//! [`twin_runtime_proto::v1::twin_runtime_service_server::TwinRuntimeService`]
//! over Tonic.

#![warn(missing_docs)]

use anyhow::{anyhow, Context, Result};
use std::net::SocketAddr;
use std::sync::Arc;
use tokio::signal::ctrl_c;
use tokio::sync::oneshot;
use tracing::{error, info};
use twin_runtime_lifecycle::Orchestrator;

mod config;
mod service;

#[tokio::main(flavor = "multi_thread", worker_threads = 4)]
async fn main() -> Result<()> {
    init_tracing();
    let cfg = config::Config::from_env().context("loading config")?;
    info!(version = env!("CARGO_PKG_VERSION"), listen = %cfg.listen, "starting twin-runtime");

    let orch = Orchestrator::with_defaults(&cfg.journal_path)
        .map_err(|e| anyhow!("orchestrator: {e}"))?;
    let orch = Arc::new(orch);

    let svc = service::TwinRuntimeServiceImpl::new(orch.clone());

    let addr: SocketAddr = cfg
        .listen
        .parse()
        .with_context(|| format!("parse listen addr {}", cfg.listen))?;

    let (shutdown_tx, shutdown_rx) = oneshot::channel::<()>();

    let server = tonic::transport::Server::builder()
        .add_service(
            twin_runtime_proto::v1::twin_runtime_service_server::TwinRuntimeServiceServer::new(svc),
        )
        .serve_with_shutdown(addr, async move {
            let _ = shutdown_rx.await;
            info!("shutdown signal received");
        });

    let signal_task = tokio::spawn(async move {
        if let Err(e) = ctrl_c().await {
            error!(error = %e, "ctrl_c handler failed; will exit anyway");
        }
        let _ = shutdown_tx.send(());
    });

    server.await.map_err(|e| anyhow!("tonic serve: {e}"))?;
    let _ = signal_task.await;
    info!("twin-runtime exited cleanly");
    Ok(())
}

fn init_tracing() {
    use tracing_subscriber::{fmt, EnvFilter};
    let filter = EnvFilter::try_from_env("CRUCIBLE_LOG")
        .or_else(|_| EnvFilter::try_new("info,twin_runtime=debug"))
        .unwrap();
    fmt()
        .with_env_filter(filter)
        .with_target(true)
        .with_thread_ids(false)
        .compact()
        .init();
}
