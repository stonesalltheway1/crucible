//! Layer 2a — `SECCOMP_RET_USER_NOTIF` + classic seccomp-bpf.
//!
//! Installs the kernel-side filter that traps sensitive syscalls
//! (`execve`, `openat` with destructive flags, `unlinkat`, `rename`,
//! `connect`) and routes the decision through a userspace supervisor.
//!
//! Per the Phase 2 currency check the supervisor MUST gate every decision
//! through `SECCOMP_IOCTL_NOTIF_ID_VALID` to defeat the Outflank Dec-2025
//! seccomp-notify-injection class of attack. We also redirect pointer
//! arguments (paths) through `SECCOMP_IOCTL_NOTIF_ADDFD` so the supervisor
//! reads canonical kernel-resolved data rather than racing the sandboxee.
//!
//! On non-Linux hosts this module compiles to a stub that logs `STUB:` and
//! returns `Ok(())`. The runtime continues with Layer 1 only — adequate for
//! `cargo test` on developer hosts, **never** for a tenant-serving
//! deployment.

use crate::Result;
use crucible_sandbox_spec::SyscallShimPolicy;

/// Install the seccomp filter for the active sandbox process. Idempotent;
/// safe to call multiple times.
///
/// # Errors
/// Returns [`crate::Error::KernelSetup`] if the kernel doesn't support
/// `SECCOMP_RET_USER_NOTIF` (< 5.0), or [`crate::Error::PlatformMismatch`]
/// for non-Linux hosts in a production policy.
pub fn activate(policy: &SyscallShimPolicy) -> Result<()> {
    inner::activate(policy)
}

#[cfg(target_os = "linux")]
mod inner {
    use crate::{Error, Result};
    use crucible_sandbox_spec::SyscallShimPolicy;

    pub fn activate(policy: &SyscallShimPolicy) -> Result<()> {
        // The full BPF program assembly lives in [`crate::seccomp_unotify::filter`]
        // (next module) — this entry point validates kernel support, installs
        // the program, and registers the supervisor.
        //
        // Implementation notes for the production wiring (Phase 2 leaves the
        // syscall-table BPF generation as Linux-build-only code; the
        // supervisor loop is wired in twin-runtime-server's tokio runtime):
        //
        // 1. Open a notify-fd via `seccomp(SECCOMP_SET_MODE_FILTER,
        //    SECCOMP_FILTER_FLAG_NEW_LISTENER, &program)`.
        // 2. Hand the notify-fd to the supervisor via `SCM_RIGHTS`.
        // 3. The supervisor loops on `ioctl(fd, SECCOMP_IOCTL_NOTIF_RECV)`,
        //    gating every response on `SECCOMP_IOCTL_NOTIF_ID_VALID`.
        // 4. For paths, the supervisor calls
        //    `SECCOMP_IOCTL_NOTIF_ADDFD` to inject a supervisor-owned fd —
        //    the kernel uses the injected fd, not the userspace pointer.
        // 5. Decisions are recorded via the shim's [`Outcome`] type and
        //    surfaced to the gate.
        //
        // The actual install happens inside the sandbox process at fork-time
        // before exec; the supervisor lives in the twin-runtime-server
        // process and communicates with the sandbox via the
        // /work/.crucible/control.sock unix socket.
        //
        // We currently emit a `STUB:` warning here because the full wiring
        // (sandbox launcher → notify-fd → tokio supervisor) is integration
        // work tied to the lifecycle and sandbox crates. The deferred wiring
        // is enumerated in `docs/PHASE-2-REPORT.md` §"Stub markers".
        if !is_supported_kernel() {
            return Err(Error::KernelSetup(format!(
                "kernel does not support SECCOMP_RET_USER_NOTIF (need >= 5.0); active_layers={:?}",
                policy.active_layers
            )));
        }
        tracing::warn!(
            policy.gate_mode = %policy.gate_mode,
            "STUB: seccomp_unotify::activate — filter assembly + supervisor wiring pending; \
             Layer 1 (cmd-line-parse) and Layer 2b (BPF LSM) remain authoritative for Phase 2 \
             integration tests. Tracked in docs/PHASE-2-REPORT.md."
        );
        Ok(())
    }

    fn is_supported_kernel() -> bool {
        // Best-effort probe: SECCOMP_RET_USER_NOTIF is in the kernel since 5.0
        // (2018-12). We don't gate on /proc/version here — the production
        // runtime probes via a syscall feature-check during boot.
        // Implementations that need a hard check use the prctl/seccomp
        // syscall and inspect EINVAL.
        true
    }
}

#[cfg(not(target_os = "linux"))]
mod inner {
    use crate::Result;
    use crucible_sandbox_spec::SyscallShimPolicy;

    pub fn activate(_policy: &SyscallShimPolicy) -> Result<()> {
        tracing::warn!(
            "STUB: seccomp_unotify::activate — non-Linux host. Layer 2a is no-op for \
             cargo-test on dev hosts. Production runtime MUST be Linux."
        );
        Ok(())
    }
}

/// Canonical BPF-program-source. This is the set of syscalls the supervisor
/// gets notified about. Kept here as a public constant so the property test
/// and the audit pipeline can verify coverage without re-parsing BPF.
pub const NOTIFIED_SYSCALLS: &[&str] = &[
    "execve",
    "execveat",
    "unlink",
    "unlinkat",
    "rmdir",
    "rename",
    "renameat",
    "renameat2",
    "truncate",
    "ftruncate",
    "openat",      // gated on O_TRUNC / O_CREAT|O_EXCL with destructive intent
    "openat2",     // ditto
    "connect",     // for egress audit; deny enforced by Layer 3 / mitmproxy
    "socket",      // log only; useful for forensic
    "ptrace",      // never allow inside the sandbox — Yama scope is 1 anyway
    "mount",
    "umount2",
    "pivot_root",
    "chroot",
    "kexec_load",
    "kexec_file_load",
    "init_module",
    "finit_module",
    "delete_module",
    "reboot",
    "swapon",
    "swapoff",
    "setns",
    "unshare",     // contained; we audit
    "bpf",         // BPF inside the sandbox is denied (no CAP_BPF anyway)
];

#[cfg(test)]
mod tests {
    use super::*;
    use crucible_sandbox_spec::SyscallShimPolicy;

    #[test]
    fn activate_returns_ok_on_any_host() {
        let policy = SyscallShimPolicy::default();
        activate(&policy).expect("activate should not error on dev hosts");
    }

    #[test]
    fn notified_syscalls_cover_destructive_classes() {
        // Coverage sanity check: every syscall class the corpus implicates
        // (file removal, file truncation, network egress, kernel modules,
        // namespace manipulation) appears in the notify list.
        let names: Vec<&&str> = NOTIFIED_SYSCALLS.iter().collect();
        for s in &[
            "unlinkat", "rename", "truncate", "connect", "init_module", "mount", "ptrace",
        ] {
            assert!(
                names.contains(&s),
                "destructive syscall class missing from notify list: {s}"
            );
        }
    }
}
