//! Binary entry-point for `crucible-attestation-relay`.

use std::sync::Arc;

use crucible_attestation_relay::{
    config::Config,
    fulcio::{FulcioClient, FulcioHttpClient, NullFulcio},
    journal::Journal,
    rekor::{RekorClient, RekorHttpClient},
    server,
    service::Service,
    signer::{LocalEd25519Signer, Signer, SigstoreKeylessSigner},
};
use tracing::info;

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error + Send + Sync>> {
    tracing_subscriber::fmt()
        .with_env_filter(tracing_subscriber::EnvFilter::try_from_default_env().unwrap_or_else(|_| "info".into()))
        .json()
        .init();

    let cfg = Config::from_env()?;
    info!(?cfg.listen_addr, ?cfg.rekor_url, offline = cfg.offline, "starting relay");

    // Journal — always wired.
    let journal = Arc::new(Journal::open(cfg.journal_path.clone())?);

    // Signer — keyless when an OIDC token is available + we're not offline;
    // otherwise local Ed25519.
    let signer: Arc<dyn Signer> = if !cfg.offline && cfg.oidc_token_dev.is_some() {
        let fulcio_inner = Arc::new(FulcioHttpClient::new(cfg.fulcio_url.clone()));
        let fulcio = FulcioClient::new(fulcio_inner);
        Arc::new(SigstoreKeylessSigner::new(
            cfg.oidc_token_dev.clone().unwrap(),
            cfg.oidc_issuer.clone(),
            fulcio,
        ))
    } else {
        Arc::new(LocalEd25519Signer::load_or_create(&cfg.dev_keys_dir)?)
    };
    info!(oidc_subject = signer.oidc_subject(), key_id = signer.key_id(), "signer wired");

    // Rekor — None when offline.
    let rekor = if cfg.offline {
        None
    } else {
        let inner = Arc::new(RekorHttpClient::new(cfg.rekor_url.clone(), cfg.rekor_self_hosted));
        Some(RekorClient::new(inner))
    };

    // Suppress unused-warning when offline.
    let _ = NullFulcio;

    let service = Arc::new(Service::new(signer, rekor, journal, cfg.offline));

    // Background back-fill task. Runs every 60s.
    {
        let s = service.clone();
        tokio::spawn(async move {
            let mut tick = tokio::time::interval(std::time::Duration::from_secs(60));
            tick.tick().await; // first immediate tick
            loop {
                tick.tick().await;
                if let Err(e) = s.backfill_once(500).await {
                    tracing::warn!(error=%e, "backfill task failed");
                }
            }
        });
    }

    let app = server::router(service);
    let listener = tokio::net::TcpListener::bind(&cfg.listen_addr).await?;
    info!(addr = %cfg.listen_addr, "relay listening");
    axum::serve(listener, app).await?;
    Ok(())
}
