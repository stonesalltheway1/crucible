//! Crucible attestation relay — the production-grade replacement for the
//! Phase-1 local-journal publisher.
//!
//! Architecture, per `docs/05-decisions/ADR-010-sigstore-rekor-attestations.md`:
//!
//! ```text
//!   Predicate (1 of 13) ──┐
//!                         ├─→ in-toto Statement v1 ─→ DSSE envelope ─→ Rekor v2
//!   subject content ──────┘                            │            (verified inclusion proof)
//!                                                      │
//!                                                      └─→ local hash-chained
//!                                                          journal (fallback)
//! ```
//!
//! The crate is structured so each layer is independently swappable:
//!
//! - `predicate` — strongly-typed payloads for all 13 Crucible predicate types,
//!   plus the SLSA Provenance v1 wrapper Tier 4 emits.
//! - `dsse` — DSSEv1 envelope construction + Pre-Authentication Encoding.
//! - `signer` — Ed25519 (dev) and Sigstore keyless (Fulcio-issued cert).
//! - `fulcio` — OIDC token → x509 cert.
//! - `rekor` — Rekor v2 publish / fetch / inclusion-proof verification.
//! - `journal` — append-only hash-chained JSONL; survives Rekor outages.
//! - `service` — the high-level facade: `relay.emit(predicate)` does
//!   build → sign → publish → mirror.
//! - `server` — the axum HTTP surface used by the Go control plane,
//!   promotion gate, verifier, twin-runtime shim, distiller, slack-bot.

#![warn(missing_docs)]
#![allow(clippy::module_name_repetitions)]
#![allow(clippy::missing_errors_doc)]

pub mod config;
pub mod dsse;
pub mod error;
pub mod fulcio;
pub mod journal;
pub mod predicate;
pub mod rekor;
pub mod server;
pub mod service;
pub mod signer;
pub mod statement;
pub mod verify;

pub use config::Config;
pub use error::Error;
pub use predicate::{Predicate, PredicateType, ALL_PREDICATES};
pub use rekor::RekorEntry;
pub use service::Service;
pub use statement::{InTotoStatement, StatementSubject};

/// Crate version exposed via /healthz.
pub const VERSION: &str = env!("CARGO_PKG_VERSION");
