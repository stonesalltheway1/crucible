//! `crucible-verify-rust` — Rust per-language verifier runner.
//!
//! Wire protocol:
//!
//! - Read a `VerificationRequest` JSON document from stdin.
//! - Audit it against the executor-reasoning denylist.
//! - Run the requested tier (`--tier=tier_X_…`).
//! - Emit `===CRUCIBLE-TESTREPORT===\n` + a `TestReport` JSON document
//!   on stdout. All logs go to stderr.
//!
//! Exit codes:
//!
//! | Code | Meaning                                          |
//! |------|--------------------------------------------------|
//! | 0    | report written; verdict may be passed OR failed  |
//! | 64   | malformed CLI / unparsable stdin                 |
//! | 65   | leak-guard rejected the request                  |
//! | 70   | internal error before a report could be written  |

use std::io::{self, Read, Write};
use std::process::ExitCode;

use clap::Parser;

use crucible_verify_rust::audit;
use crucible_verify_rust::schema::{Tier, VerificationRequest};
use crucible_verify_rust::tiers;

/// Delimiter the process pool greps for in stdout. Matches
/// `apps/verifier/internal/processpool/pool.go::trimPrelude`.
pub const REPORT_DELIMITER: &str = "===CRUCIBLE-TESTREPORT===\n";

#[derive(Parser, Debug)]
#[command(
    name = "crucible-verify-rust",
    version,
    about = "Crucible Phase-4 Rust verifier runner"
)]
struct Cli {
    /// Tier to execute (e.g. `tier_0_mutation`).
    #[arg(long)]
    tier: String,
}

fn main() -> ExitCode {
    let cli = Cli::parse();
    match real_main(&cli) {
        Ok(()) => ExitCode::from(0),
        Err(MainError::Cli(msg)) => {
            eprintln!("crucible-verify-rust: cli: {msg}");
            ExitCode::from(64)
        }
        Err(MainError::Leak(msg)) => {
            eprintln!("crucible-verify-rust: leak-guard refused request: {msg}");
            ExitCode::from(65)
        }
        Err(MainError::Internal(msg)) => {
            eprintln!("crucible-verify-rust: internal: {msg}");
            ExitCode::from(70)
        }
    }
}

#[derive(Debug)]
enum MainError {
    Cli(String),
    Leak(String),
    Internal(String),
}

fn real_main(cli: &Cli) -> Result<(), MainError> {
    let tier = Tier::parse(&cli.tier).map_err(|e| MainError::Cli(e.to_string()))?;

    // Read the whole request from stdin.
    let mut raw = Vec::new();
    io::stdin()
        .read_to_end(&mut raw)
        .map_err(|e| MainError::Cli(format!("read stdin: {e}")))?;

    // Audit the raw JSON before any deserialisation so an attacker's
    // extra fields are caught even if our struct rejects them.
    let raw_value: serde_json::Value = serde_json::from_slice(&raw)
        .map_err(|e| MainError::Cli(format!("parse request json: {e}")))?;
    audit::audit(&raw_value).map_err(|e| MainError::Leak(e.to_string()))?;

    let req: VerificationRequest = serde_json::from_value(raw_value)
        .map_err(|e| MainError::Cli(format!("decode VerificationRequest: {e}")))?;

    // Path-level audit on the diff.
    let paths = req
        .diff
        .files
        .iter()
        .map(|f| f.path.clone())
        .collect::<Vec<_>>();
    audit::audit_paths(paths.iter().map(String::as_str))
        .map_err(|e| MainError::Leak(e.to_string()))?;

    eprintln!(
        "crucible-verify-rust: tier={tier} task={tid} files={n}",
        tier = tier.as_str(),
        tid = req.task_id,
        n = req.diff.files.len()
    );

    let report = tiers::dispatch(tier, &req);

    let stdout = io::stdout();
    let mut handle = stdout.lock();
    handle
        .write_all(REPORT_DELIMITER.as_bytes())
        .map_err(|e| MainError::Internal(format!("write delimiter: {e}")))?;
    serde_json::to_writer(&mut handle, &report)
        .map_err(|e| MainError::Internal(format!("encode report: {e}")))?;
    handle
        .write_all(b"\n")
        .map_err(|e| MainError::Internal(format!("trailing newline: {e}")))?;
    handle
        .flush()
        .map_err(|e| MainError::Internal(format!("flush stdout: {e}")))?;
    Ok(())
}
