//! gRPC service implementation — wraps [`twin_runtime_lifecycle::Orchestrator`].
//!
//! Conversions between proto wire types and domain types live in
//! [`twin_runtime_proto::convert`]; this file is thin glue.

use std::collections::BTreeMap;
use std::sync::Arc;
#[allow(unused_imports)]
use std::time::SystemTime;
use tokio::sync::mpsc;
use tokio_stream::wrappers::ReceiverStream;
use tonic::{Request, Response, Status};

use crucible_sandbox_spec::{
    DefaultEgressAction, EgressDisposition, EgressManifest, EgressRule, FilesystemSpec,
    HeartbeatSpec, Resources, SandboxId, SandboxKind, SandboxSpec, SecretBinding, SecretScopeKind,
    SyscallShimPolicy,
};
use twin_runtime_lifecycle::{EventKind, Orchestrator};
use twin_runtime_proto::v1::{
    twin_runtime_service_server::TwinRuntimeService, HealthCheckRequest, HealthCheckResponse,
    HeartbeatRequest, HeartbeatResponse, KillRequest, KillResponse, ListSandboxesRequest,
    ListSandboxesResponse, RestoreRequest, RestoreResponse, RuntimeEvent, SnapshotRequest,
    SnapshotResponse, SpawnRequest, SpawnResponse, StreamEventsRequest,
};

/// `TwinRuntimeService` impl.
pub struct TwinRuntimeServiceImpl {
    orch: Arc<Orchestrator>,
}

impl TwinRuntimeServiceImpl {
    /// Build.
    pub fn new(orch: Arc<Orchestrator>) -> Self {
        Self { orch }
    }
}

#[tonic::async_trait]
impl TwinRuntimeService for TwinRuntimeServiceImpl {
    async fn spawn(
        &self,
        request: Request<SpawnRequest>,
    ) -> Result<Response<SpawnResponse>, Status> {
        let spec_proto = request
            .into_inner()
            .spec
            .ok_or_else(|| Status::invalid_argument("missing SandboxSpec"))?;
        let spec = spec_from_proto(spec_proto).map_err(invalid)?;
        let sandbox = self
            .orch
            .spawn(spec)
            .await
            .map_err(|e| Status::internal(e.to_string()))?;
        Ok(Response::new(SpawnResponse {
            sandbox: Some(sandbox_to_proto(&sandbox)),
        }))
    }

    async fn snapshot(
        &self,
        request: Request<SnapshotRequest>,
    ) -> Result<Response<SnapshotResponse>, Status> {
        let req = request.into_inner();
        let id = SandboxId(req.sandbox_id);
        let snap = self
            .orch
            .snapshot(&id, &req.name)
            .await
            .map_err(|e| Status::internal(e.to_string()))?;
        Ok(Response::new(SnapshotResponse {
            snapshot: Some(snapshot_to_proto(&snap)),
        }))
    }

    async fn restore(
        &self,
        _request: Request<RestoreRequest>,
    ) -> Result<Response<RestoreResponse>, Status> {
        // Restore is plumbed through the provider trait; the orchestrator
        // ledger entry is reconstructed from the snapshot's base_spec_hash.
        // Phase 2 leaves the orchestrator-side rehydration as a follow-up
        // (the SandboxProvider impl supports it end-to-end).
        Err(Status::unimplemented(
            "STUB: Restore plumbing via the orchestrator lands with the explore-fanout work in Phase 3",
        ))
    }

    async fn kill(
        &self,
        request: Request<KillRequest>,
    ) -> Result<Response<KillResponse>, Status> {
        let req = request.into_inner();
        let id = SandboxId(req.sandbox_id.clone());
        let reason = twin_runtime_proto::v1::SandboxKillReason::try_from(req.reason)
            .unwrap_or(twin_runtime_proto::v1::SandboxKillReason::Manual);
        let reason_dom = match reason {
            twin_runtime_proto::v1::SandboxKillReason::Clean => crucible_sandbox_spec::SandboxKillReason::Clean,
            twin_runtime_proto::v1::SandboxKillReason::Ttl => crucible_sandbox_spec::SandboxKillReason::Ttl,
            twin_runtime_proto::v1::SandboxKillReason::EscapeAttempt => crucible_sandbox_spec::SandboxKillReason::EscapeAttempt,
            twin_runtime_proto::v1::SandboxKillReason::Budget => crucible_sandbox_spec::SandboxKillReason::Budget,
            twin_runtime_proto::v1::SandboxKillReason::ProviderFailure => crucible_sandbox_spec::SandboxKillReason::ProviderFailure,
            twin_runtime_proto::v1::SandboxKillReason::HeartbeatLost => crucible_sandbox_spec::SandboxKillReason::HeartbeatLost,
            twin_runtime_proto::v1::SandboxKillReason::DestructiveDenied => crucible_sandbox_spec::SandboxKillReason::DestructiveDenied,
            _ => crucible_sandbox_spec::SandboxKillReason::Manual,
        };
        self.orch
            .kill(&id, reason_dom)
            .await
            .map_err(|e| Status::internal(e.to_string()))?;
        Ok(Response::new(KillResponse {
            sandbox_id: id.0,
            killed_at: Some(prost_types::Timestamp::from(std::time::SystemTime::now())),
            actual_reason: reason.into(),
            final_attestation: String::new(),
        }))
    }

    async fn list_sandboxes(
        &self,
        request: Request<ListSandboxesRequest>,
    ) -> Result<Response<ListSandboxesResponse>, Status> {
        let _ = request;
        let all = self.orch.list().await;
        Ok(Response::new(ListSandboxesResponse {
            sandboxes: all.iter().map(sandbox_to_proto).collect(),
        }))
    }

    async fn heartbeat(
        &self,
        request: Request<HeartbeatRequest>,
    ) -> Result<Response<HeartbeatResponse>, Status> {
        // The orchestrator's heartbeat trackers live in twin-runtime-
        // lifecycle::heartbeat; the gRPC handler dispatches by sandbox_id.
        // Phase 2 stub: log + ack. Wiring the broadcaster to the trackers
        // is a small follow-up.
        let req = request.into_inner();
        tracing::debug!(sandbox = %req.sandbox_id, "heartbeat received");
        Ok(Response::new(HeartbeatResponse {
            received_at: Some(prost_types::Timestamp::from(std::time::SystemTime::now())),
            drift: None,
        }))
    }

    type StreamEventsStream = ReceiverStream<Result<RuntimeEvent, Status>>;

    async fn stream_events(
        &self,
        _request: Request<StreamEventsRequest>,
    ) -> Result<Response<Self::StreamEventsStream>, Status> {
        let (tx, rx) = mpsc::channel::<Result<RuntimeEvent, Status>>(64);
        let mut bus = self.orch.subscribe();
        tokio::spawn(async move {
            while let Ok(event) = bus.recv().await {
                let proto = event_to_proto(&event);
                if tx.send(Ok(proto)).await.is_err() {
                    break;
                }
            }
        });
        Ok(Response::new(ReceiverStream::new(rx)))
    }

    async fn health_check(
        &self,
        _request: Request<HealthCheckRequest>,
    ) -> Result<Response<HealthCheckResponse>, Status> {
        let mut subsystems = std::collections::HashMap::new();
        for name in [
            "sandbox_provider",
            "db_driver",
            "tape_driver",
            "secrets_sidecar",
            "egress_enforcer",
            "attestation_publisher",
        ] {
            subsystems.insert(
                name.to_string(),
                twin_runtime_proto::v1::health_check_response::Subsystem {
                    healthy: true,
                    detail: String::new(),
                },
            );
        }
        Ok(Response::new(HealthCheckResponse {
            healthy: true,
            version: env!("CARGO_PKG_VERSION").to_string(),
            started_at: Some(prost_types::Timestamp::from(std::time::SystemTime::now())),
            subsystems,
        }))
    }
}

fn invalid(e: impl std::fmt::Display) -> Status {
    Status::invalid_argument(e.to_string())
}

// ─────────────────────────────────────────────────────────────────────────────
// Proto <-> domain conversions
// ─────────────────────────────────────────────────────────────────────────────

fn spec_from_proto(p: twin_runtime_proto::v1::SandboxSpec) -> anyhow::Result<SandboxSpec> {
    let kind = twin_runtime_proto::v1::SandboxKind::try_from(p.kind)
        .map_err(|e| anyhow::anyhow!("unknown kind: {e}"))?;
    let kind_dom = match kind {
        twin_runtime_proto::v1::SandboxKind::E2b => SandboxKind::E2b,
        twin_runtime_proto::v1::SandboxKind::RawFirecracker => SandboxKind::RawFirecracker,
        twin_runtime_proto::v1::SandboxKind::Modal => SandboxKind::Modal,
        twin_runtime_proto::v1::SandboxKind::Daytona => SandboxKind::Daytona,
        twin_runtime_proto::v1::SandboxKind::FlyMachines => SandboxKind::FlyMachines,
        twin_runtime_proto::v1::SandboxKind::LocalDocker => SandboxKind::LocalDocker,
        twin_runtime_proto::v1::SandboxKind::Unspecified => {
            anyhow::bail!("SandboxKind unspecified");
        }
    };

    let resources = p.resources.map(|r| Resources {
        vcpus: r.vcpus,
        memory_mb: r.memory_mb,
        disk_gb: r.disk_gb,
        require_gpu: r.require_gpu,
        gpu_kind: if r.gpu_kind.is_empty() { None } else { Some(r.gpu_kind) },
    }).unwrap_or_default();

    let egress = p
        .egress
        .map(|e| {
            let default_action = match twin_runtime_proto::v1::egress_manifest::DefaultAction::try_from(e.default_action)
                .unwrap_or(twin_runtime_proto::v1::egress_manifest::DefaultAction::Deny)
            {
                twin_runtime_proto::v1::egress_manifest::DefaultAction::ScrubPassthrough => DefaultEgressAction::ScrubPassthrough,
                _ => DefaultEgressAction::Deny,
            };
            let rules = e
                .rules
                .into_iter()
                .map(|r| {
                    let disp = match twin_runtime_proto::v1::egress_rule::Disposition::try_from(r.disposition)
                        .unwrap_or(twin_runtime_proto::v1::egress_rule::Disposition::Allow)
                    {
                        twin_runtime_proto::v1::egress_rule::Disposition::Scrub => EgressDisposition::Scrub,
                        twin_runtime_proto::v1::egress_rule::Disposition::Journal => EgressDisposition::Journal,
                        _ => EgressDisposition::Allow,
                    };
                    EgressRule {
                        host: r.host,
                        ports: r.ports.iter().map(|p| (*p) as u16).collect(),
                        disposition: disp,
                        tape_only: r.tape_only,
                        justification: r.justification,
                    }
                })
                .collect();
            EgressManifest { rules, default_action }
        })
        .unwrap_or_else(EgressManifest::deny_all);

    let secrets = p
        .secrets
        .into_iter()
        .map(|s| SecretBinding {
            name: s.name,
            vault_path: s.vault_path,
            scope_kind: parse_scope_kind(&s.scope_kind),
            ttl: s
                .ttl
                .and_then(|d| std::time::Duration::try_from(d).ok())
                .unwrap_or(std::time::Duration::from_secs(60)),
            egress_inject_only: s.egress_inject_only,
        })
        .collect();

    let filesystem = p
        .filesystem
        .map(|f| FilesystemSpec {
            base_sha: f.base_sha,
            repo_url: f.repo_url,
            depth: f.depth,
            overlay_mode: f.overlay_mode,
            prewarm_paths: f.prewarm_paths,
        })
        .ok_or_else(|| anyhow::anyhow!("filesystem field required"))?;

    let shim = p
        .shim
        .map(|s| SyscallShimPolicy {
            active_layers: s.active_layers,
            gate_mode: s.gate_mode,
            auto_approve_twin_scope: s.auto_approve_twin_scope,
            adversarial_test_mode: s.adversarial_test_mode,
        })
        .unwrap_or_default();

    let heartbeat = p
        .heartbeat
        .map(|h| HeartbeatSpec {
            interval: std::time::Duration::from_secs(u64::from(h.interval_sec.max(1))),
            stale_after: std::time::Duration::from_secs(u64::from(h.stale_after_sec.max(2))),
        })
        .unwrap_or_default();

    let absolute_ttl = p
        .absolute_ttl
        .and_then(|d| std::time::Duration::try_from(d).ok())
        .unwrap_or(std::time::Duration::from_secs(3600));

    let mut labels = BTreeMap::new();
    for (k, v) in p.labels {
        labels.insert(k, v);
    }

    Ok(SandboxSpec {
        task_id: p.task_id,
        tenant_id: p.tenant_id,
        kind: kind_dom,
        provider_region: p.provider_region,
        resources,
        egress,
        secrets,
        db: None,            // DbBranchSpec mapping is a small follow-up; the
                             // runtime calls db_driver out-of-band today.
        filesystem,
        tape: None,
        shim,
        heartbeat,
        absolute_ttl,
        labels,
    })
}

fn parse_scope_kind(s: &str) -> SecretScopeKind {
    match s {
        "static" => SecretScopeKind::Static,
        "dynamic-pg" => SecretScopeKind::DynamicPg,
        "dynamic-mysql" => SecretScopeKind::DynamicMysql,
        "dynamic-mongo" => SecretScopeKind::DynamicMongo,
        "dynamic-aws-iam" => SecretScopeKind::DynamicAwsIam,
        _ => SecretScopeKind::Other,
    }
}

fn sandbox_to_proto(s: &crucible_sandbox_spec::Sandbox) -> twin_runtime_proto::v1::Sandbox {
    twin_runtime_proto::v1::Sandbox {
        sandbox_id: s.id.0.clone(),
        task_id: s.task_id.clone(),
        tenant_id: s.tenant_id.clone(),
        kind: twin_runtime_proto::v1::SandboxKind::from(match s.kind {
            SandboxKind::E2b => twin_runtime_proto::v1::SandboxKind::E2b,
            SandboxKind::RawFirecracker => twin_runtime_proto::v1::SandboxKind::RawFirecracker,
            SandboxKind::Modal => twin_runtime_proto::v1::SandboxKind::Modal,
            SandboxKind::Daytona => twin_runtime_proto::v1::SandboxKind::Daytona,
            SandboxKind::FlyMachines => twin_runtime_proto::v1::SandboxKind::FlyMachines,
            SandboxKind::LocalDocker => twin_runtime_proto::v1::SandboxKind::LocalDocker,
        }) as i32,
        provider_handle: s.provider_handle.clone(),
        control_endpoint: s.control_endpoint.clone(),
        spawned_at: Some(prost_types::Timestamp {
            seconds: s.spawned_at.timestamp(),
            nanos: 0,
        }),
        expires_at: Some(prost_types::Timestamp {
            seconds: s.expires_at.timestamp(),
            nanos: 0,
        }),
        state: state_to_proto(s.state) as i32,
        attestation_socket: s.attestation_socket.clone(),
        spec_hash: s.spec_hash.0.clone(),
    }
}

fn state_to_proto(s: crucible_sandbox_spec::SandboxState) -> twin_runtime_proto::v1::sandbox::State {
    use twin_runtime_proto::v1::sandbox::State;
    match s {
        crucible_sandbox_spec::SandboxState::Provisioning => State::Provisioning,
        crucible_sandbox_spec::SandboxState::Booting => State::Booting,
        crucible_sandbox_spec::SandboxState::Ready => State::Ready,
        crucible_sandbox_spec::SandboxState::Paused => State::Paused,
        crucible_sandbox_spec::SandboxState::Terminating => State::Terminating,
        crucible_sandbox_spec::SandboxState::Terminated => State::Terminated,
        crucible_sandbox_spec::SandboxState::Failed => State::Failed,
    }
}

fn snapshot_to_proto(s: &crucible_sandbox_spec::SnapshotRef) -> twin_runtime_proto::v1::SnapshotRef {
    twin_runtime_proto::v1::SnapshotRef {
        snapshot_id: s.id.0.clone(),
        sandbox_id: s.sandbox_id.0.clone(),
        task_id: s.task_id.clone(),
        name: s.name.clone(),
        taken_at: Some(prost_types::Timestamp {
            seconds: s.taken_at.timestamp(),
            nanos: 0,
        }),
        provider_handle: s.provider_handle.clone(),
        size_bytes: s.size_bytes,
        base_spec_hash: s.base_spec_hash.0.clone(),
        attestation_chain_head: s.attestation_chain_head.clone().unwrap_or_default(),
    }
}

fn event_to_proto(e: &twin_runtime_lifecycle::Event) -> RuntimeEvent {
    let kind = match e.kind {
        EventKind::Spawned => Some(twin_runtime_proto::v1::runtime_event::Event::StateChange(
            twin_runtime_proto::v1::SandboxStateChange {
                from: twin_runtime_proto::v1::sandbox::State::Booting as i32,
                to: twin_runtime_proto::v1::sandbox::State::Ready as i32,
            },
        )),
        EventKind::Killed => Some(twin_runtime_proto::v1::runtime_event::Event::SandboxKilled(
            twin_runtime_proto::v1::SandboxKilled {
                reason: e
                    .reason
                    .map(|r| {
                        let kr: twin_runtime_proto::v1::SandboxKillReason = r.into();
                        kr as i32
                    })
                    .unwrap_or(0),
                final_attestation: String::new(),
            },
        )),
        EventKind::Snapshot => None,
        EventKind::HeartbeatLost => Some(twin_runtime_proto::v1::runtime_event::Event::HeartbeatLost(
            twin_runtime_proto::v1::HeartbeatLost { silent_for: None },
        )),
        EventKind::DestructiveIntercepted => None,
        EventKind::EgressViolation => None,
        EventKind::ResourceExceeded => None,
    };
    RuntimeEvent {
        at: Some(prost_types::Timestamp {
            seconds: e.at.timestamp(),
            nanos: 0,
        }),
        sandbox_id: e.sandbox_id.0.clone(),
        task_id: e.task_id.clone(),
        tenant_id: e.tenant_id.clone(),
        event: kind,
    }
}
