//! Runtime config — read from env + optional TOML.

use serde::Deserialize;
use std::path::PathBuf;

/// Runtime configuration.
#[derive(Debug, Clone, Deserialize)]
pub struct Config {
    /// `host:port` to listen on for the gRPC server.
    #[serde(default = "default_listen")]
    pub listen: String,
    /// Path to the local attestation journal.
    #[serde(default = "default_journal_path")]
    pub journal_path: PathBuf,
}

fn default_listen() -> String {
    std::env::var("CRUCIBLE_TWIN_LISTEN").unwrap_or_else(|_| "127.0.0.1:7444".into())
}

fn default_journal_path() -> PathBuf {
    std::env::var("CRUCIBLE_TWIN_JOURNAL")
        .ok()
        .map(PathBuf::from)
        .or_else(|| dirs())
        .unwrap_or_else(|| PathBuf::from("./.crucible/twin-runtime-journal.jsonl"))
}

fn dirs() -> Option<PathBuf> {
    std::env::var("HOME")
        .ok()
        .map(|h| PathBuf::from(h).join(".crucible/twin-runtime-journal.jsonl"))
}

impl Config {
    /// Load config from env (and optional TOML pointed at by `CRUCIBLE_TWIN_CONFIG`).
    pub fn from_env() -> anyhow::Result<Self> {
        if let Ok(path) = std::env::var("CRUCIBLE_TWIN_CONFIG") {
            let raw = std::fs::read_to_string(&path)?;
            return Ok(toml::from_str::<Config>(&raw)?);
        }
        Ok(Self {
            listen: default_listen(),
            journal_path: default_journal_path(),
        })
    }
}
