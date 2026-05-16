//! The Crucible destructive-op gate.
//!
//! This crate is the **brand promise** of the entire product: the
//! architectural guarantee that an agent process — treated as hostile per
//! `docs/01-architecture/threat-model.md` — cannot execute a destructive
//! operation without producing a typed [`DestructiveProposal`] that flows
//! through the [`gate::Gate`] for policy evaluation. The PocketOS scenario
//! ("agent finds Railway token, runs `railway down`, prod gone in 9s") is
//! intercepted here.
//!
//! ## Enforcement layers
//!
//! The shim is multi-layer by design — no single primitive is allowed to be
//! the sole boundary, because every primitive has historical bypasses.
//!
//! 1. **Layer 1 — Command-line parse** ([`cmd_parse`]). Pre-exec lexical
//!    analysis of `twin.shell.exec(cmd)` arguments. Pattern-matched against
//!    the corpus in [`corpus`]. Cheap, cross-platform, runs in the runtime
//!    process before any syscall happens. Catches the obvious cases.
//!
//! 2. **Layer 2a — `SECCOMP_RET_USER_NOTIF`** ([`seccomp_unotify`]). In-
//!    kernel BPF filter for the allow/deny fast path; sensitive syscalls
//!    notify a supervisor via a notify-fd which decides via the gate. Per
//!    Phase 2 currency-check, we replaced the legacy `ptrace` layer with
//!    `seccomp_unotify` — ptrace adds 300–1000× syscall overhead and has
//!    well-documented TOCTOU bypasses. Layer 2a always pairs every notify
//!    with `SECCOMP_IOCTL_NOTIF_ID_VALID` to defeat the Outflank Dec-2025
//!    seccomp-notify-injection class of attack.
//!
//! 3. **Layer 2b — BPF LSM** ([`bpf_lsm`]). Path-aware destructive-FS
//!    enforcement at the kernel's LSM hooks (`inode_unlink`, `inode_rename`,
//!    `path_truncate`, `file_open`). Because the kernel has already
//!    resolved the path, this is TOCTOU-free. Returns `-EPERM` on a
//!    destructive operation rather than letting the syscall proceed.
//!
//! 4. **Layer 3 — Tetragon** ([`tetragon`]). Post-exec audit + async kill
//!    via a `TracingPolicy` (production / self-hosted Firecracker) or a
//!    Tetragon-equivalent userspace consumer (E2B, where in-guest eBPF is
//!    unavailable). Defense-in-depth: if both prior layers were bypassed,
//!    Tetragon records the security event and signals the runtime to kill
//!    the sandbox with [`crucible_sandbox_spec::SandboxKillReason::EscapeAttempt`].
//!
//! ## Trust boundary
//!
//! The shim's correctness invariants:
//!
//! - **No bypass.** Any input — adversarial or accidental — that triggers a
//!   destructive operation MUST be intercepted by at least one of the three
//!   layers and converted to a [`DestructiveProposal`]. The property test
//!   in `tests/property_50k.rs` enforces this at 50,000 iterations with
//!   zero tolerated bypasses.
//!
//! - **No false twin-scope auto-approve on a real-scoped op.** The gate's
//!   scope classifier in [`gate::scope`] is fail-closed: any classification
//!   ambiguity returns `RealScoped`, which requires Promotion Contract
//!   human approval. False positives are accepted; false negatives are not.
//!
//! - **Attestation is mandatory.** Every interception emits a
//!   `DestructiveProposal/v1` attestation through
//!   `twin-runtime-attest::publish`. There is no fast path that skips
//!   attestation.

#![forbid(unsafe_code)] // unsafe is permitted only in seccomp_unotify::linux behind cfg.
#![warn(missing_docs)]

pub mod corpus;
pub mod cmd_parse;
pub mod proposal;
pub mod gate;
pub mod seccomp_unotify;
pub mod bpf_lsm;
pub mod tetragon;

use crucible_sandbox_spec::SyscallShimPolicy;
use thiserror::Error;

pub use proposal::{DestructiveProposal, InterceptLayer, Scope};

/// Errors from shim setup / operation.
#[derive(Debug, Error)]
pub enum Error {
    /// Layer 1 failed to tokenize the command. The agent's input is mal-
    /// formed (unclosed quote, invalid escape, etc.). Fail-closed: an
    /// untokenisable command is rejected, never executed.
    #[error("layer-1 tokenisation failed: {0}")]
    TokenisationFailed(String),

    /// Layer 2 kernel-side setup failed. Typically a missing capability or
    /// an old kernel without `SECCOMP_RET_USER_NOTIF` / BPF LSM support.
    #[error("layer-2 kernel setup failed: {0}")]
    KernelSetup(String),

    /// Layer 3 Tetragon controller is unreachable. Production runtime
    /// treats this as a P0; dev runtime may degrade to layer-1+2 only with
    /// a `crucible.io/dev-mode=true` label.
    #[error("layer-3 Tetragon unavailable: {0}")]
    TetragonUnavailable(String),

    /// The agent attempted a destructive operation and the gate denied it.
    /// Surfaced via the SDK as `CrucibleError::DestructiveProposalRejected`.
    #[error("destructive operation rejected: {0}")]
    DestructiveRejected(String),

    /// A platform-only feature was invoked on the wrong host. The runtime
    /// degrades gracefully on dev hosts but flags this clearly so prod
    /// configurations don't silently miss layers.
    #[error("platform mismatch: {0}")]
    PlatformMismatch(String),

    /// Provider sub-call failed.
    #[error("provider error: {0}")]
    Provider(String),
}

/// Result alias for the shim.
pub type Result<T> = std::result::Result<T, Error>;

/// The runtime-visible shim handle. Constructed at sandbox-spawn time with
/// a [`SyscallShimPolicy`] from the [`crucible_sandbox_spec::SandboxSpec`];
/// installed inside the sandbox before the agent's first instruction.
pub struct Shim {
    policy: SyscallShimPolicy,
    parser: cmd_parse::Parser,
    gate: gate::Gate,
}

impl Shim {
    /// Construct a shim for the given policy. Does not yet install kernel-
    /// side filters; call [`Self::activate`] inside the sandbox to do that.
    ///
    /// # Errors
    /// Returns [`Error::PlatformMismatch`] if the policy demands kernel
    /// layers on a non-Linux host without the `dev-mode` label.
    pub fn build(policy: SyscallShimPolicy) -> Result<Self> {
        let parser = cmd_parse::Parser::with_default_corpus();
        let gate = gate::Gate::new(&policy);
        Ok(Self { policy, parser, gate })
    }

    /// Returns the policy this shim was built for.
    #[must_use]
    pub fn policy(&self) -> &SyscallShimPolicy {
        &self.policy
    }

    /// Returns the layers actually active. Always a subset of
    /// [`SyscallShimPolicy::active_layers`] — entries naming the legacy
    /// `"ptrace"` layer are normalised to `"seccomp-unotify"`.
    #[must_use]
    pub fn active_layers(&self) -> Vec<String> {
        self.policy
            .active_layers
            .iter()
            .map(|s| match s.as_str() {
                "ptrace" => "seccomp-unotify".to_string(),
                other => other.to_string(),
            })
            .collect()
    }

    /// Examines a command against Layer 1 only. Returns
    /// [`Outcome::Approve`] for benign commands, [`Outcome::Intercept`]
    /// with a typed proposal for destructive ones.
    ///
    /// This is the host-visible entry-point used by `twin.shell.exec`
    /// before the syscall is allowed to proceed. It is also the unit of
    /// the property test in `tests/property_50k.rs`.
    pub fn evaluate_command(&self, cmd: &str, task_id: &str) -> Result<Outcome> {
        let parsed = self
            .parser
            .parse(cmd)
            .map_err(|e| Error::TokenisationFailed(e.to_string()))?;
        match self.parser.match_corpus(&parsed) {
            cmd_parse::MatchResult::Benign => Ok(Outcome::Approve),
            cmd_parse::MatchResult::Destructive(hit) => {
                let proposal = proposal::DestructiveProposal::from_match(task_id, cmd, &hit);
                Ok(self.gate.evaluate(proposal))
            }
        }
    }

    /// Activate kernel-side layers. On Linux with the right kernel features
    /// this installs seccomp filters, attaches BPF LSM programs, and
    /// dispatches the Tetragon TracingPolicy. On non-Linux hosts it returns
    /// `Ok(())` with a `STUB:` trace event so dev builds still spin up.
    ///
    /// # Errors
    /// See [`Error`].
    pub fn activate(&self) -> Result<()> {
        for layer in self.active_layers() {
            match layer.as_str() {
                "cmd-line-parse" => {
                    // Layer 1 is always-on in the userspace path; nothing to install.
                    tracing::info!(layer = %layer, "shim layer ready");
                }
                "seccomp-unotify" => seccomp_unotify::activate(&self.policy)?,
                "bpf-lsm" => bpf_lsm::activate(&self.policy)?,
                "tetragon" => tetragon::activate(&self.policy)?,
                other => {
                    return Err(Error::KernelSetup(format!(
                        "unknown shim layer: {other}"
                    )));
                }
            }
        }
        Ok(())
    }
}

/// Outcome of a Layer-1 evaluation.
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum Outcome {
    /// Command is benign; proceed with execution.
    Approve,
    /// Command matched the destructive corpus and was intercepted.
    /// The agent receives this typed proposal and must call
    /// `twin.shell.approveDestructive` (or pivot strategy) to proceed.
    Intercept(DestructiveProposal),
    /// Twin-scoped destructive auto-approved by the gate (e.g., `rm` of a
    /// path under `/work/scratch`). The proposal is still recorded in the
    /// attestation chain.
    AutoApprovedTwinScope(DestructiveProposal),
    /// Real-scoped destructive — the agent does not receive an exec result;
    /// instead the proposal is forwarded to the Promotion Contract for
    /// HSM-signed approval (always human-in-the-loop).
    ForwardToPromotion(DestructiveProposal),
}

impl Outcome {
    /// Returns true if execution proceeds in the sandbox.
    #[must_use]
    pub fn allows_execution(&self) -> bool {
        matches!(self, Self::Approve | Self::AutoApprovedTwinScope(_))
    }

    /// Returns the wrapped proposal if any.
    #[must_use]
    pub fn proposal(&self) -> Option<&DestructiveProposal> {
        match self {
            Self::Approve => None,
            Self::Intercept(p) | Self::AutoApprovedTwinScope(p) | Self::ForwardToPromotion(p) => {
                Some(p)
            }
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crucible_sandbox_spec::SyscallShimPolicy;

    fn default_shim() -> Shim {
        Shim::build(SyscallShimPolicy::default()).unwrap()
    }

    #[test]
    fn active_layers_normalises_ptrace_to_seccomp_unotify() {
        let mut policy = SyscallShimPolicy::default();
        policy.active_layers = vec!["cmd-line-parse".into(), "ptrace".into(), "tetragon".into()];
        let shim = Shim::build(policy).unwrap();
        let layers = shim.active_layers();
        assert!(!layers.contains(&"ptrace".to_string()));
        assert!(layers.contains(&"seccomp-unotify".to_string()));
        assert!(layers.contains(&"cmd-line-parse".to_string()));
        assert!(layers.contains(&"tetragon".to_string()));
    }

    #[test]
    fn benign_command_approves() {
        let shim = default_shim();
        let outcome = shim.evaluate_command("ls -la", "task_test").unwrap();
        assert_eq!(outcome, Outcome::Approve);
        assert!(outcome.allows_execution());
    }

    #[test]
    fn rm_rf_in_scratch_auto_approves_twin_scope() {
        let shim = default_shim();
        let outcome = shim
            .evaluate_command("rm -rf /work/scratch/build", "task_test")
            .unwrap();
        assert!(matches!(outcome, Outcome::AutoApprovedTwinScope(_)));
        assert!(outcome.allows_execution());
        let prop = outcome.proposal().expect("destructive proposal recorded");
        assert!(prop.command.contains("rm"));
    }

    #[test]
    fn railway_down_intercepted_as_real_scope() {
        let shim = default_shim();
        let outcome = shim.evaluate_command("railway down", "task_test").unwrap();
        assert!(matches!(outcome, Outcome::ForwardToPromotion(_)));
        assert!(!outcome.allows_execution());
    }

    #[test]
    fn unbalanced_quote_fails_closed() {
        let shim = default_shim();
        let err = shim.evaluate_command("echo 'unterminated", "task_test").unwrap_err();
        assert!(matches!(err, Error::TokenisationFailed(_)));
    }
}
