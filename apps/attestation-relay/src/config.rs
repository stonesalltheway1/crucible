//! Relay configuration.

use std::path::PathBuf;

use crate::error::{Error, Result};

/// Relay configuration. All fields are populated from environment variables
/// by `Config::from_env`. Override programmatically for tests.
#[derive(Debug, Clone)]
pub struct Config {
    /// Bind address (e.g., `0.0.0.0:9120`).
    pub listen_addr: String,
    /// Rekor v2 base URL.
    pub rekor_url: String,
    /// Self-hosted Rekor mode? When `true` the relay does not fall through
    /// to public Sigstore on Rekor failures; it journals locally and waits
    /// for the self-hosted Rekor to recover.
    pub rekor_self_hosted: bool,
    /// Optional path to a PEM bundle for self-hosted Rekor.
    pub rekor_root_ca: Option<PathBuf>,
    /// Fulcio v2 base URL.
    pub fulcio_url: String,
    /// OIDC issuer URI that signs OIDC tokens for Fulcio.
    pub oidc_issuer: String,
    /// Pre-issued OIDC token (dev / test only).
    pub oidc_token_dev: Option<String>,
    /// Local hash-chained journal path.
    pub journal_path: PathBuf,
    /// Dev signer keypair directory.
    pub dev_keys_dir: PathBuf,
    /// If `true`, the relay never reaches Rekor — journal-only.
    pub offline: bool,
}

impl Config {
    /// Builds a `Config` from environment variables. Falls back to local-dev
    /// defaults — every default is "safe": offline-equivalent until env vars
    /// flip the relay into a real Rekor publish path.
    pub fn from_env() -> Result<Self> {
        let listen_addr = std::env::var("CRUCIBLE_RELAY_ADDR").unwrap_or_else(|_| "0.0.0.0:9120".into());
        let rekor_url = std::env::var("CRUCIBLE_REKOR_URL").unwrap_or_else(|_| "https://rekor.sigstore.dev".into());
        let rekor_self_hosted = matches!(std::env::var("CRUCIBLE_REKOR_SELF_HOSTED").as_deref(), Ok("1"));
        let rekor_root_ca = std::env::var("CRUCIBLE_REKOR_ROOT_CA").ok().map(PathBuf::from);
        let fulcio_url = std::env::var("CRUCIBLE_FULCIO_URL").unwrap_or_else(|_| "https://fulcio.sigstore.dev".into());
        let oidc_issuer = std::env::var("CRUCIBLE_OIDC_ISSUER").unwrap_or_else(|_| "https://accounts.crucible.dev".into());
        let oidc_token_dev = std::env::var("CRUCIBLE_OIDC_TOKEN").ok();
        let home = home_dir().ok_or_else(|| Error::Config("locate $HOME".into()))?;
        let journal_path = std::env::var("CRUCIBLE_JOURNAL_PATH")
            .map(PathBuf::from)
            .unwrap_or_else(|_| home.join(".crucible/attestations/relay-journal.jsonl"));
        let dev_keys_dir = std::env::var("CRUCIBLE_RELAY_DEV_KEYS")
            .map(PathBuf::from)
            .unwrap_or_else(|_| home.join(".crucible/relay-keys"));
        let offline = matches!(std::env::var("CRUCIBLE_RELAY_OFFLINE").as_deref(), Ok("1"));
        Ok(Self {
            listen_addr,
            rekor_url,
            rekor_self_hosted,
            rekor_root_ca,
            fulcio_url,
            oidc_issuer,
            oidc_token_dev,
            journal_path,
            dev_keys_dir,
            offline,
        })
    }
}

fn home_dir() -> Option<PathBuf> {
    #[cfg(unix)]
    {
        std::env::var_os("HOME").map(PathBuf::from)
    }
    #[cfg(windows)]
    {
        std::env::var_os("USERPROFILE").map(PathBuf::from)
    }
    #[cfg(not(any(unix, windows)))]
    {
        None
    }
}
