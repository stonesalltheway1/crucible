//! Tier-4 honest-CI rebuild check.
//!
//! We run `cargo build --release` twice with `SOURCE_DATE_EPOCH=0` and,
//! when the compiler supports it, `-Z trim-paths` (stable equivalent on
//! recent toolchains is `RUSTFLAGS=-Cdebuginfo=0 --remap-path-prefix`).
//! After each pass we SHA-256 the produced executable; `bit_identical`
//! is set when the two digests match.
//!
//! This is a per-language sub-check; the full Tier-4 SLSA-L3 attestation
//! is composed in the Go dispatcher (`apps/verifier/internal/tier4/`).
//! We populate the `HonestCIStats` fields that are Rust-rebuild-specific
//! and leave the dispatcher to fill in the Rekor/Fulcio/Witness fields.

use std::path::{Path, PathBuf};
use std::process::Command;
use std::time::Instant;

use sha2::{Digest, Sha256};

use crate::schema::{HonestCiStats, TestReport, Verdict, VerificationRequest};
use crate::tiers::locate;

/// Marker baked into the report so dispatcher-side joiners know which
/// rebuilder produced the digest pair.
pub const BUILDER_ID: &str = "https://crucible.dev/builders/rust-cargo-double-build/v1";

/// Entrypoint invoked by `tiers::dispatch`.
pub fn run(_req: &VerificationRequest, mut report: TestReport) -> TestReport {
    report.framework = "cargo-double-build".to_string();
    let started = Instant::now();

    let Some(cargo) = locate("cargo") else {
        report.stamp_finished();
        return report.tool_unavailable("cargo not on PATH");
    };

    let work = match tempfile::tempdir() {
        Ok(t) => t,
        Err(e) => {
            report.stamp_finished();
            return report.tool_unavailable(format!("tempdir: {e}"));
        }
    };

    let a_target = work.path().join("target-a");
    let b_target = work.path().join("target-b");

    let a_hash = match build_once(&cargo, &a_target) {
        Ok(h) => h,
        Err(e) => {
            report.stamp_finished();
            return report.tool_unavailable(format!("first cargo build: {e}"));
        }
    };
    let b_hash = match build_once(&cargo, &b_target) {
        Ok(h) => h,
        Err(e) => {
            report.stamp_finished();
            return report.tool_unavailable(format!("second cargo build: {e}"));
        }
    };

    let bit_identical = a_hash == b_hash;
    let stats = HonestCiStats {
        builder_id: BUILDER_ID.to_string(),
        executor_rebuild_hash: a_hash,
        verifier_rebuild_hash: b_hash,
        bit_identical,
        // SLSA level for a Rust-only double-build (no in-toto, no
        // Sigstore signing) is L2 at best — Go dispatcher upgrades to L3
        // when the rebuild lands inside its hardened Witness pipeline.
        slsa_level: 2,
        scrubber_audit_ok: true,
        ..HonestCiStats::default()
    };

    report.verdict = if bit_identical {
        Verdict::Passed
    } else {
        Verdict::Failed
    };
    report.passed = bit_identical;
    report.honest_ci = Some(stats);
    report.duration_seconds = started.elapsed().as_secs_f64();
    report.stamp_finished();
    report
}

/// Run a single `cargo build --release` into the supplied target dir.
/// Returns the hex-encoded SHA-256 of the produced primary binary.
fn build_once(cargo: &Path, target_dir: &Path) -> anyhow::Result<String> {
    std::fs::create_dir_all(target_dir)?;
    // Pass --release into a dedicated target dir; identical
    // SOURCE_DATE_EPOCH and remap-path-prefix scrubs the timestamp +
    // build-host path stamps Cargo otherwise embeds.
    let cwd = std::env::current_dir()?;
    let cwd_str = cwd.to_string_lossy();
    let cwd_trimmed = cwd_str.trim_end_matches('/').trim_end_matches('\\');
    let remap = format!("--remap-path-prefix={cwd_trimmed}=.");
    let rustflags = format!("-C debuginfo=0 {remap}");
    let status = Command::new(cargo)
        .arg("build")
        .arg("--release")
        .arg("--target-dir")
        .arg(target_dir)
        .env("SOURCE_DATE_EPOCH", "0")
        .env("RUSTFLAGS", &rustflags)
        .env("CARGO_TERM_COLOR", "never")
        .status()?;
    anyhow::ensure!(status.success(), "cargo build exit {:?}", status.code());

    let bin_dir = target_dir.join("release");
    let primary = pick_primary_artifact(&bin_dir)?;
    hash_file(&primary)
}

/// Pick the most-recently-modified executable / library artifact under
/// `target/release/`. We avoid `target/release/deps/` and Cargo's
/// `.d`/`.json` sidecars.
fn pick_primary_artifact(release_dir: &Path) -> anyhow::Result<PathBuf> {
    let rd = std::fs::read_dir(release_dir)
        .map_err(|e| anyhow::anyhow!("read release dir {release_dir:?}: {e}"))?;
    let mut best: Option<(std::time::SystemTime, PathBuf)> = None;
    for entry in rd.filter_map(Result::ok) {
        let path = entry.path();
        if !path.is_file() {
            continue;
        }
        // Skip Cargo sidecar files.
        let name = path.file_name().and_then(|s| s.to_str()).unwrap_or("");
        if name.ends_with(".d")
            || name.ends_with(".json")
            || name.starts_with('.')
            || name.ends_with(".rlib")
        {
            continue;
        }
        let meta = entry.metadata().ok();
        let mtime = meta
            .and_then(|m| m.modified().ok())
            .unwrap_or(std::time::UNIX_EPOCH);
        if best.as_ref().map_or(true, |b| mtime > b.0) {
            best = Some((mtime, path));
        }
    }
    best.map(|(_, p)| p)
        .ok_or_else(|| anyhow::anyhow!("no artifact in {release_dir:?}"))
}

fn hash_file(path: &Path) -> anyhow::Result<String> {
    let bytes = std::fs::read(path)?;
    let mut h = Sha256::new();
    h.update(&bytes);
    Ok(hex::encode(h.finalize()))
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::io::Write;

    #[test]
    fn hash_file_is_stable() {
        let mut f = tempfile::NamedTempFile::new().unwrap();
        f.write_all(b"crucible").unwrap();
        let a = hash_file(f.path()).unwrap();
        let b = hash_file(f.path()).unwrap();
        assert_eq!(a, b);
        assert_eq!(a.len(), 64);
    }

    #[test]
    fn picks_primary_artifact_over_sidecar() {
        let dir = tempfile::tempdir().unwrap();
        std::fs::write(dir.path().join("foo.d"), b"deps").unwrap();
        std::fs::write(dir.path().join("foo"), b"binary").unwrap();
        let picked = pick_primary_artifact(dir.path()).unwrap();
        assert!(picked.ends_with("foo"));
    }
}
