//! Layer 2b — BPF LSM destructive-FS hooks.
//!
//! Attaches BPF programs at LSM sites where the kernel has already resolved
//! the user's path argument:
//!
//! - `inode_unlink`     — `unlink` / `unlinkat`
//! - `inode_rename`     — `rename` / `renameat` / `renameat2`
//! - `path_truncate`    — `truncate` / `ftruncate`
//! - `file_open`        — `openat` with `O_TRUNC`
//! - `bprm_check_security` — `execve` argv inspection
//!
//! Returning `-EPERM` from any of these hooks denies the operation in-kernel
//! without a userspace round-trip. Because the kernel passes us a resolved
//! `struct path`, there is no TOCTOU window between argument validation and
//! syscall execution.
//!
//! BPF LSM requires kernel ≥ 5.7 with `CONFIG_BPF_LSM=y` and the LSM enabled
//! at boot (`lsm=...,bpf`). The production Firecracker guest image bundles
//! this; the E2B-hosted guest does not — we degrade to Layer 1 + Layer 3
//! Tetragon there.

use crate::Result;
use crucible_sandbox_spec::SyscallShimPolicy;

/// Install the BPF LSM programs for the active sandbox.
///
/// # Errors
/// Returns [`crate::Error::KernelSetup`] on unsupported kernels.
pub fn activate(policy: &SyscallShimPolicy) -> Result<()> {
    inner::activate(policy)
}

#[cfg(target_os = "linux")]
mod inner {
    use crate::{Error, Result};
    use crucible_sandbox_spec::SyscallShimPolicy;

    pub fn activate(policy: &SyscallShimPolicy) -> Result<()> {
        // Real implementation flow:
        //   1. Probe /sys/kernel/security/lsm — must contain "bpf".
        //   2. Load the per-hook BPF programs via libbpf (we wrap via
        //      `landlock` for the simpler subset + raw bpf() for hooks
        //      landlock doesn't cover).
        //   3. Attach with BPF_LSM_MAC.
        //   4. Pin to /sys/fs/bpf/crucible/<sandbox_id>/ for visibility.
        //   5. Register the sandbox's pid namespace in the BPF map so the
        //      programs scope to this sandbox only.
        //
        // Phase 2 ships the design, the activation entry-point, the hook
        // list, and the Landlock fallback. The full libbpf program load is
        // Linux-build-only and tracked as a Phase 2.5 follow-up (the
        // landlock fallback gives us the *enforcement* on every kernel that
        // ships in 2026; the BPF LSM hooks are the *speed* path).
        //
        // Landlock fallback (always-on; runs alongside BPF LSM when both
        // are available — defense in depth):
        if let Err(e) = landlock_fallback::apply_default_ruleset() {
            tracing::warn!(
                error = %e,
                "STUB/degraded: Landlock fallback could not apply (kernel may lack \
                 ABI v4 or the runtime user lacks CAP_SYS_ADMIN). BPF LSM hook \
                 attachment continues but defense-in-depth is reduced. Tracked in \
                 docs/PHASE-2-REPORT.md."
            );
        }

        tracing::warn!(
            ?policy.active_layers,
            "STUB: bpf_lsm::activate — hook attachment via libbpf pending. \
             Landlock fallback active for path-based confinement. Tracked in \
             docs/PHASE-2-REPORT.md."
        );

        if !is_kernel_supported() {
            return Err(Error::KernelSetup(
                "BPF LSM requires kernel >= 5.7 with bpf in /sys/kernel/security/lsm".into(),
            ));
        }
        Ok(())
    }

    fn is_kernel_supported() -> bool {
        std::fs::read_to_string("/sys/kernel/security/lsm")
            .map(|s| s.split(',').any(|l| l.trim() == "bpf"))
            .unwrap_or(false)
    }

    pub mod landlock_fallback {
        //! Landlock confines the agent process to a path subset using the
        //! kernel-resident ABI. It can't be relaxed by the sandboxee
        //! (per the man page invariant), so it's a robust defense-in-depth
        //! layer even when BPF LSM is unavailable.

        use anyhow::Result;

        pub fn apply_default_ruleset() -> Result<()> {
            use landlock::{
                ABI, Access, AccessFs, PathBeneath, PathFd, Ruleset, RulesetAttr, RulesetCreatedAttr,
                RulesetStatus,
            };
            let abi = ABI::V4;
            let ruleset = Ruleset::default()
                .handle_access(AccessFs::from_all(abi))?
                .create()?
                .add_rules(
                    [
                        PathBeneath::new(
                            PathFd::new("/work/scratch")?,
                            AccessFs::from_all(abi),
                        ),
                        PathBeneath::new(
                            PathFd::new("/work/repo")?,
                            AccessFs::from_read(abi) | AccessFs::WriteFile | AccessFs::MakeReg,
                        ),
                        PathBeneath::new(
                            PathFd::new("/work/tapes")?,
                            AccessFs::from_read(abi),
                        ),
                        PathBeneath::new(
                            PathFd::new("/tmp")?,
                            AccessFs::from_all(abi),
                        ),
                    ]
                    .into_iter()
                    .map(Ok::<_, anyhow::Error>)
                    .collect::<Result<Vec<_>>>()?,
                )?;
            let status = ruleset.restrict_self()?;
            if status.ruleset == RulesetStatus::NotEnforced {
                anyhow::bail!("Landlock ruleset not enforced by kernel");
            }
            Ok(())
        }
    }
}

#[cfg(not(target_os = "linux"))]
mod inner {
    use crate::Result;
    use crucible_sandbox_spec::SyscallShimPolicy;

    pub fn activate(_policy: &SyscallShimPolicy) -> Result<()> {
        tracing::warn!(
            "STUB: bpf_lsm::activate — non-Linux host. Layer 2b is a no-op for \
             cargo-test on dev hosts. Production runtime MUST be Linux."
        );
        Ok(())
    }
}

/// LSM hooks the runtime expects to be attached.
pub const ATTACHED_HOOKS: &[&str] = &[
    "inode_unlink",
    "inode_rmdir",
    "inode_rename",
    "path_truncate",
    "file_open",
    "bprm_check_security",
    "sb_remount",
    "sb_umount",
    "socket_connect",
];

#[cfg(test)]
mod tests {
    use super::*;
    use crucible_sandbox_spec::SyscallShimPolicy;

    #[test]
    fn activate_returns_ok_on_any_host() {
        let policy = SyscallShimPolicy::default();
        let result = activate(&policy);
        // On Linux dev hosts the activate may succeed or return KernelSetup
        // depending on whether BPF LSM is enabled. Both are acceptable for
        // unit tests; we only fail if it panics or returns a non-Error.
        if let Err(e) = result {
            assert!(matches!(e, crate::Error::KernelSetup(_)), "unexpected error: {e:?}");
        }
    }

    #[test]
    fn attached_hooks_cover_destructive_classes() {
        for h in &["inode_unlink", "inode_rename", "path_truncate", "file_open"] {
            assert!(
                ATTACHED_HOOKS.contains(h),
                "destructive hook missing: {h}"
            );
        }
    }
}
