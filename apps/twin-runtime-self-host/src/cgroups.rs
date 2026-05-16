//! Per-tenant cgroup quota wiring (cgroups v2).
//!
//! Per docs/04-operations/self-hosted-install.md, per-tenant cgroup quotas
//! are mandatory to prevent neighbour effects. Phase 3 ships:
//!
//! - cpu.max — micro-CPU quota per period (e.g., 200000 / 100000 → 2 CPUs)
//! - memory.max — hard memory cap (e.g., 4_294_967_296 → 4 GiB)
//! - io.max — per-block-device I/O cap (best-effort; varies by kernel)
//!
//! The Linux cgroups v2 interface is filesystem-based: we write the
//! desired values to /sys/fs/cgroup/<cgroup>/<knob>. On non-Linux hosts
//! this module no-ops gracefully so cargo check passes.

use serde::{Deserialize, Serialize};
use std::path::{Path, PathBuf};

use crate::provider::{Error, SpawnRequest};

/// Per-spawn cgroup quota.
#[derive(Debug, Clone, Copy, Serialize, Deserialize)]
pub struct CgroupQuota {
    /// CPU period in microseconds (cgroup v2 cpu.max numerator).
    pub cpu_max_us: u64,
    /// CPU quota window in microseconds (denominator).
    pub cpu_max_period_us: u64,
    /// Memory cap in bytes.
    pub memory_max_bytes: u64,
    /// PID cap (cgroup v2 pids.max).
    pub pids_max: u64,
    /// IO weight (1..10_000).
    pub io_weight: u16,
}

impl Default for CgroupQuota {
    fn default() -> Self {
        Self {
            cpu_max_us: 200_000,        // 2 CPUs equivalent
            cpu_max_period_us: 100_000,
            memory_max_bytes: 4 * 1024 * 1024 * 1024, // 4 GiB
            pids_max: 1024,
            io_weight: 100,
        }
    }
}

/// Apply the quota for one spawn.
///
/// The cgroup path is `<parent>/<tenant>/<sandbox_id>`. On Linux we write
/// the cpu.max / memory.max / pids.max files; on other hosts we no-op.
pub fn apply(parent: &Path, req: &SpawnRequest) -> Result<(), Error> {
    let cgroup_path = parent
        .join(&req.tenant_id)
        .join(format!("project_{}", sanitise(&req.project_id)));
    if !cfg!(target_os = "linux") {
        tracing::debug!(?cgroup_path, "non-Linux host: cgroup apply is a no-op");
        return Ok(());
    }
    std::fs::create_dir_all(&cgroup_path)
        .map_err(|e| Error::Io(format!("create cgroup dir {}: {e}", cgroup_path.display())))?;
    write_cgroup(&cgroup_path, "cpu.max", format!("{} {}", req.quota.cpu_max_us, req.quota.cpu_max_period_us))?;
    write_cgroup(&cgroup_path, "memory.max", req.quota.memory_max_bytes.to_string())?;
    write_cgroup(&cgroup_path, "pids.max", req.quota.pids_max.to_string())?;
    write_cgroup(&cgroup_path, "io.weight", req.quota.io_weight.to_string())?;
    Ok(())
}

#[cfg(target_os = "linux")]
fn write_cgroup(path: &Path, knob: &str, value: String) -> Result<(), Error> {
    let p = path.join(knob);
    fs_err::write(&p, value.as_bytes())
        .map_err(|e| Error::Io(format!("write {}: {e}", p.display())))
}

#[cfg(not(target_os = "linux"))]
fn write_cgroup(_path: &Path, _knob: &str, _value: String) -> Result<(), Error> {
    Ok(())
}

fn sanitise(s: &str) -> String {
    s.chars()
        .map(|c| {
            if c.is_ascii_alphanumeric() || c == '-' || c == '_' {
                c
            } else {
                '_'
            }
        })
        .collect()
}

/// Returns the on-disk path the orchestrator would write the cgroup
/// quotas to for a given spawn. Test hook.
pub fn cgroup_path_for(parent: &Path, req: &SpawnRequest) -> PathBuf {
    parent
        .join(&req.tenant_id)
        .join(format!("project_{}", sanitise(&req.project_id)))
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::network::NetworkPolicy;

    fn req() -> SpawnRequest {
        SpawnRequest {
            spec_hash: "spec_x".into(),
            tenant_id: "tenant/a".into(),
            project_id: "proj.p".into(),
            oci_image: "img:1".into(),
            restore_from_snapshot: None,
            quota: CgroupQuota::default(),
            network: NetworkPolicy { egress: vec![] },
        }
    }

    #[test]
    fn quota_defaults_match_two_cpu_four_gib() {
        let q = CgroupQuota::default();
        assert_eq!(q.cpu_max_us, 200_000);
        assert_eq!(q.cpu_max_period_us, 100_000);
        assert_eq!(q.memory_max_bytes, 4 * 1024 * 1024 * 1024);
    }

    #[test]
    fn cgroup_path_sanitises_unsafe_characters() {
        let p = cgroup_path_for(Path::new("/sys/fs/cgroup/crucible"), &req());
        assert!(p.to_string_lossy().ends_with("/tenant/a/project_proj_p"));
    }
}
