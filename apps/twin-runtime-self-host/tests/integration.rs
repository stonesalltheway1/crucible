//! Integration tests for the self-host orchestrator.
//!
//! These tests exercise the orchestrator's coordination logic (pool +
//! ZFS + cgroup-path generation + Tetragon-policy rendering) without
//! depending on the real Firecracker runtime. The cold-start path
//! returns a typed `PhaseStub` without `linux-firecracker`; we assert
//! it returns the right typed error rather than panicking.

use std::path::PathBuf;
use std::time::Duration;
use tempfile::tempdir;

use twin_runtime_self_host::{
    cgroups::CgroupQuota,
    network::{EgressRule, NetworkPolicy},
    provider::{Error, Orchestrator, OrchestratorConfig, SpawnRequest},
};

fn cfg(td: &tempfile::TempDir) -> OrchestratorConfig {
    OrchestratorConfig {
        host_id: "ci-host".into(),
        listen_address: "127.0.0.1:0".into(),
        zfs_pool_root: td.path().join("zfs"),
        warm_pool_size: 2,
        cgroup_parent: td.path().join("cgroup"),
        tetragon_policy_dir: td.path().join("tetragon"),
        firecracker_binary: PathBuf::from("/usr/local/bin/firecracker"),
    }
}

fn req() -> SpawnRequest {
    SpawnRequest {
        spec_hash: "spec_int".into(),
        tenant_id: "tenant_a".into(),
        project_id: "proj_p".into(),
        oci_image: "registry/test:1".into(),
        restore_from_snapshot: None,
        quota: CgroupQuota::default(),
        network: NetworkPolicy {
            egress: vec![EgressRule {
                cidr: "10.0.0.0/8".into(),
                ports: vec![443],
            }],
        },
    }
}

#[tokio::test]
async fn pool_warm_then_acquire_yields_served_from_pool() {
    let td = tempdir().unwrap();
    let orch = Orchestrator::new(cfg(&td)).await.unwrap();
    let _ = orch.rewarm("spec_int").await.unwrap();

    // With a warm slot available, spawn proceeds past the cold-start path
    // and reaches the cgroup/Tetragon-policy emission steps; the spawn
    // succeeds even without `linux-firecracker`.
    let result = orch.spawn(req()).await;
    if cfg!(feature = "linux-firecracker") {
        // Linux build still needs root + KVM; this test doesn't try.
        let _ = result;
    } else {
        // The non-feature build now serves from the warm pool and reports
        // served_from_pool=true.
        let sb = result.unwrap();
        assert!(sb.served_from_pool);
        assert!(sb.restored_from.is_some());
        assert!(!sb.id.is_empty());
        assert_eq!(sb.net_namespace, "cr_tenant_a_proj_p");
    }
}

#[tokio::test]
async fn cold_start_without_feature_returns_phasestub() {
    let td = tempdir().unwrap();
    let orch = Orchestrator::new(cfg(&td)).await.unwrap();
    // No warm-up; cold path.
    let res = orch.spawn(req()).await;
    if !cfg!(feature = "linux-firecracker") {
        match res {
            Err(Error::PhaseStub(msg)) => {
                assert!(msg.contains("linux-firecracker"));
            }
            other => panic!("expected PhaseStub, got {:?}", other),
        }
    }
}

#[tokio::test]
async fn rewarm_top_ups_to_target() {
    let td = tempdir().unwrap();
    let orch = Orchestrator::new(cfg(&td)).await.unwrap();
    let added = orch.rewarm("spec_int").await.unwrap();
    assert_eq!(added, 2);
    // Second call is a no-op once at target.
    let added2 = orch.rewarm("spec_int").await.unwrap();
    assert_eq!(added2, 0);
}

#[tokio::test]
async fn kill_destroys_zfs_clone_path() {
    let td = tempdir().unwrap();
    let orch = Orchestrator::new(cfg(&td)).await.unwrap();
    let _ = orch.rewarm("spec_int").await.unwrap();
    if !cfg!(feature = "linux-firecracker") {
        let sb = orch.spawn(req()).await.unwrap();
        // Destroy is best-effort and no-ops on non-Linux; this test just
        // asserts the call does not error.
        orch.kill(&sb).await.unwrap();
    }
}

#[tokio::test]
async fn spawn_latency_under_200ms_for_warm_path() {
    let td = tempdir().unwrap();
    let orch = Orchestrator::new(cfg(&td)).await.unwrap();
    orch.rewarm("spec_int").await.unwrap();
    if !cfg!(feature = "linux-firecracker") {
        let start = std::time::Instant::now();
        let _sb = orch.spawn(req()).await.unwrap();
        // Phase 3 sentinel: warm-pool acquisition + bookkeeping must
        // stay well under 200ms even on slow CI runners.
        assert!(
            start.elapsed() < Duration::from_millis(200),
            "warm spawn took {:?}",
            start.elapsed()
        );
    }
}
