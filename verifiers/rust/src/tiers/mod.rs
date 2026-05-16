//! Per-tier runners. Each submodule exposes a `run` function that takes
//! a parsed `VerificationRequest` plus a `TestReport` skeleton and
//! returns the completed report. The CLI in `main.rs` only knows about
//! these `run` entry points; it has no per-tier logic of its own.

#![allow(missing_docs)]
#![allow(clippy::module_name_repetitions)]

pub mod tier0_mutation;
pub mod tier1_pbt;
pub mod tier2_contract;
pub mod tier3_proof;
pub mod tier4_honest_ci;

use crate::schema::{TestReport, Tier, VerificationRequest};

/// Common wall-clock budget defaults (seconds), mirroring the
/// per-tier numbers in `docs/01-architecture/verifier-pipeline.md`.
pub const TIER0_BUDGET_SECS: f64 = 120.0;
pub const TIER1_BUDGET_SECS: f64 = 300.0;
pub const TIER2_BUDGET_SECS: f64 = 900.0;
pub const TIER3_BUDGET_SECS: f64 = 600.0;
pub const TIER4_BUDGET_SECS: f64 = 1800.0;

/// Dispatch the request to the appropriate tier-specific runner.
pub fn dispatch(tier: Tier, req: &VerificationRequest) -> TestReport {
    let mut report = TestReport::new(
        tier,
        crate::schema::Language::Rust,
        &req.task_id,
        &req.base_sha,
    );
    report.wall_clock_budget_seconds = match tier {
        Tier::Mutation => TIER0_BUDGET_SECS,
        Tier::Pbt => TIER1_BUDGET_SECS,
        Tier::Contract => TIER2_BUDGET_SECS,
        Tier::Proof => TIER3_BUDGET_SECS,
        Tier::HonestCi => TIER4_BUDGET_SECS,
    };
    match tier {
        Tier::Mutation => tier0_mutation::run(req, report),
        Tier::Pbt => tier1_pbt::run(req, report),
        Tier::Contract => tier2_contract::run(req, report),
        Tier::Proof => tier3_proof::run(req, report),
        Tier::HonestCi => tier4_honest_ci::run(req, report),
    }
}

/// Convenience: locate a tool on `PATH`. Returns `None` if absent.
pub fn locate(tool: &str) -> Option<std::path::PathBuf> {
    which::which(tool).ok()
}
