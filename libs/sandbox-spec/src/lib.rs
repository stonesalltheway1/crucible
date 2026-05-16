//! Crucible Sandbox Provider — trait + conformance suite.
//!
//! The Twin Runtime is provider-agnostic: E2B in SaaS, raw Firecracker in the
//! self-hosted tier, Daytona/Fly Machines in the solo-founder tier. Each
//! provider implements [`SandboxProvider`] and must pass the conformance
//! corpus shipped under `#[cfg(feature = "conformance")]`.
//!
//! This crate is intentionally free of provider-specific dependencies. It is
//! the *spec* — concrete implementations live in
//! `apps/twin-runtime/twin-runtime-sandbox/`.

#![forbid(unsafe_code)]
#![warn(missing_docs)]
#![warn(clippy::pedantic)]
#![allow(clippy::missing_errors_doc)] // Errors are typed in [`Error`] and propagated.

use async_trait::async_trait;
use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use std::collections::BTreeMap;
use std::fmt;
use std::time::Duration;
use thiserror::Error;

pub mod spec_hash;

#[cfg(feature = "conformance")]
pub mod conformance;

// ─────────────────────────────────────────────────────────────────────────────
// Identifiers and primitive value types
// ─────────────────────────────────────────────────────────────────────────────

/// Opaque sandbox identifier assigned by the runtime at spawn time.
///
/// Uniqueness is per-runtime-instance: an id may collide across runtimes,
/// but never within one runtime's lifetime.
#[derive(Clone, Debug, PartialEq, Eq, Hash, Serialize, Deserialize)]
#[serde(transparent)]
pub struct SandboxId(pub String);

impl SandboxId {
    /// Returns the underlying string slice.
    #[must_use]
    pub fn as_str(&self) -> &str {
        &self.0
    }
}

impl fmt::Display for SandboxId {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        f.write_str(&self.0)
    }
}

/// Opaque snapshot identifier from [`SandboxProvider::snapshot`].
#[derive(Clone, Debug, PartialEq, Eq, Hash, Serialize, Deserialize)]
#[serde(transparent)]
pub struct SnapshotId(pub String);

impl SnapshotId {
    /// Returns the underlying string slice.
    #[must_use]
    pub fn as_str(&self) -> &str {
        &self.0
    }
}

/// Hex-encoded SHA-256 of the canonical [`SandboxSpec`] JSON. Used as a
/// content-address so two requests with identical spec produce comparable
/// sandboxes for cache-warming and snapshot reuse.
#[derive(Clone, Debug, PartialEq, Eq, Hash, Serialize, Deserialize)]
#[serde(transparent)]
pub struct SpecHash(pub String);

// ─────────────────────────────────────────────────────────────────────────────
// SandboxKind — concrete provider implementations
// ─────────────────────────────────────────────────────────────────────────────

/// Identifies the provider family. Stable across serialisation.
#[derive(Clone, Copy, Debug, PartialEq, Eq, Hash, Serialize, Deserialize)]
#[serde(rename_all = "kebab-case")]
pub enum SandboxKind {
    /// Hosted Firecracker via E2B (SaaS-tier default per ADR-015).
    E2b,
    /// Self-orchestrated Firecracker + containerd + ZFS (enterprise self-host).
    RawFirecracker,
    /// Modal Sandbox — fallback for GPU-capable workloads.
    Modal,
    /// Daytona dev workspaces — solo-founder tier.
    Daytona,
    /// Fly Machines (scale-to-zero) — solo-founder tier.
    FlyMachines,
    /// Local Docker — dev + CI only; **never** for tenant traffic.
    LocalDocker,
}

impl SandboxKind {
    /// Returns true for kinds that provide hardware-grade isolation
    /// (microVM, not just namespace/cgroup).
    #[must_use]
    pub fn is_hardware_isolated(self) -> bool {
        matches!(self, Self::E2b | Self::RawFirecracker | Self::Modal)
    }

    /// Returns true for kinds approved for production tenant traffic.
    /// `LocalDocker` is the only currently-disallowed kind.
    #[must_use]
    pub fn allowed_for_tenant_traffic(self) -> bool {
        !matches!(self, Self::LocalDocker)
    }
}

// ─────────────────────────────────────────────────────────────────────────────
// SandboxSpec — content-addressable spawn specification
// ─────────────────────────────────────────────────────────────────────────────

/// Resource envelope for the sandbox.
#[derive(Clone, Debug, PartialEq, Eq, Serialize, Deserialize)]
pub struct Resources {
    /// vCPU count. Default 2.
    pub vcpus: u32,
    /// RAM in megabytes. Default 4096.
    pub memory_mb: u32,
    /// Disk in gigabytes. Default 8.
    pub disk_gb: u32,
    /// When true, routes to a GPU-capable provider (Modal). Off by default.
    pub require_gpu: bool,
    /// Informational GPU kind, e.g. "a10" / "a100" / "h100".
    pub gpu_kind: Option<String>,
}

impl Default for Resources {
    fn default() -> Self {
        Self {
            vcpus: 2,
            memory_mb: 4096,
            disk_gb: 8,
            require_gpu: false,
            gpu_kind: None,
        }
    }
}

/// One entry in the per-task egress allowlist.
#[derive(Clone, Debug, PartialEq, Eq, Serialize, Deserialize)]
pub struct EgressRule {
    /// FQDN or CIDR. FQDN is resolved by the runtime's DNS sidecar (Tetragon
    /// still does not support FQDN selectors natively per the May 2026
    /// currency check); CIDR is enforced directly.
    pub host: String,
    /// Allowed TCP ports. Empty = all ports.
    pub ports: Vec<u16>,
    /// Disposition for matching traffic.
    pub disposition: EgressDisposition,
    /// When true, requests for this host are served from tape only — even if
    /// the disposition would otherwise allow live passthrough.
    pub tape_only: bool,
    /// Free-text reason. Surfaced in attestations and the egress-violation
    /// runbook (RB-03).
    pub justification: String,
}

/// What to do when a request matches a rule.
#[derive(Clone, Copy, Debug, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "kebab-case")]
pub enum EgressDisposition {
    /// Allow the request through unchanged.
    Allow,
    /// Pass through the PII-scrubbing egress proxy.
    Scrub,
    /// Record the call in the in-memory mutation journal; never forward.
    Journal,
}

/// What to do when no rule matches.
#[derive(Clone, Copy, Debug, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "kebab-case")]
pub enum DefaultEgressAction {
    /// Drop the connection (production default).
    Deny,
    /// Forward through the scrubbing proxy with PII redaction.
    ScrubPassthrough,
}

/// The full egress manifest. Frozen for the lifetime of a sandbox; any
/// change requires a new sandbox.
#[derive(Clone, Debug, PartialEq, Eq, Serialize, Deserialize)]
pub struct EgressManifest {
    /// Ordered rule list. First match wins.
    pub rules: Vec<EgressRule>,
    /// Action when no rule matches.
    pub default_action: DefaultEgressAction,
}

impl EgressManifest {
    /// Returns the deny-everything default. Used by tests and as a safe
    /// fallback when a task manifest is malformed.
    #[must_use]
    pub fn deny_all() -> Self {
        Self {
            rules: Vec::new(),
            default_action: DefaultEgressAction::Deny,
        }
    }
}

/// Static-vs-dynamic secret-binding kind.
#[derive(Clone, Copy, Debug, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "kebab-case")]
pub enum SecretScopeKind {
    /// Long-lived value pinned at spawn time.
    Static,
    /// Dynamic Postgres credential (Infisical dynamic-secrets).
    DynamicPg,
    /// Dynamic MySQL credential.
    DynamicMysql,
    /// Dynamic MongoDB credential.
    DynamicMongo,
    /// Dynamic AWS IAM credential (sts:AssumeRole).
    DynamicAwsIam,
    /// Any other named dynamic-secret type configured in Infisical.
    Other,
}

/// One secret binding declared by the task manifest.
#[derive(Clone, Debug, PartialEq, Eq, Serialize, Deserialize)]
pub struct SecretBinding {
    /// Logical name the agent uses via `twin.secret.get(name)`.
    pub name: String,
    /// Path in Infisical (or override vault).
    pub vault_path: String,
    /// Scope kind.
    pub scope_kind: SecretScopeKind,
    /// Lease TTL. **Floor 5 seconds** per Infisical's documented minimum
    /// (May 2026 currency check) — runtime rejects shorter values.
    pub ttl: Duration,
    /// When true, the value never leaves the egress proxy: the agent gets
    /// a [`SecretBindingHandle`] only. Production default.
    pub egress_inject_only: bool,
}

impl SecretBinding {
    /// Validates the TTL floor. Used by spec validators.
    ///
    /// # Errors
    /// Returns [`Error::InvalidSpec`] if `ttl` is less than 5 seconds.
    pub fn validate_ttl(&self) -> Result<()> {
        if self.ttl < Duration::from_secs(5) {
            return Err(Error::InvalidSpec(format!(
                "SecretBinding '{}' has ttl={:?}; Infisical floor is 5s",
                self.name, self.ttl
            )));
        }
        Ok(())
    }
}

/// Opaque secret handle returned to the agent. Per ADR-014, the raw value
/// **never** appears in this struct; the egress proxy substitutes it at
/// request time via the `$secret(name)$` placeholder.
#[derive(Clone, Debug, PartialEq, Eq, Serialize, Deserialize)]
pub struct SecretBindingHandle {
    /// Logical name. Echoes [`SecretBinding::name`].
    pub name: String,
    /// Opaque token consumed by the egress proxy. Not the secret value.
    pub handle: String,
    /// Wall-clock expiry of the handle.
    pub expires_at: DateTime<Utc>,
}

/// Filesystem twin description.
#[derive(Clone, Debug, PartialEq, Eq, Serialize, Deserialize)]
pub struct FilesystemSpec {
    /// Git revision the worktree is pinned to.
    pub base_sha: String,
    /// Source repo URL (https or ssh).
    pub repo_url: String,
    /// Shallow-clone depth. `1` is enough for most tasks.
    pub depth: u32,
    /// Overlay mode — `"overlayfs-linux"` in production, `"copy"` is the
    /// cross-platform fallback for dev hosts without overlayfs support.
    pub overlay_mode: String,
    /// Paths to materialise into the overlay before agent start.
    pub prewarm_paths: Vec<String>,
}

/// Database twin description.
#[derive(Clone, Debug, PartialEq, Eq, Serialize, Deserialize)]
pub struct DbBranchSpec {
    /// Driver engine. Currently supported: `"postgres-neon"`. Other engines
    /// return a typed `STUB:` error pointing at Phase 3.
    pub engine: String,
    /// Parent branch name (typically `"twin-base"`).
    pub base_branch: String,
    /// Parent LSN, optional. Pin to avoid race with parent migration —
    /// strongly recommended per Phase 2 Neon currency-check finding.
    pub parent_lsn: Option<String>,
    /// Auto-delete TTL. Defaults to the task wall-clock budget + 15 min grace.
    pub ttl: Duration,
    /// When true, the branch is treated as read-only by the twin.
    pub protected: bool,
}

/// Service twin description.
#[derive(Clone, Debug, PartialEq, Eq, Serialize, Deserialize)]
pub struct TapeSpec {
    /// Content-addressed tape-bundle id.
    pub tape_set: String,
    /// Mode per `tape-coverage-strategy.md`: `"strict"`, `"hybrid"`, `"adaptive"`.
    pub mode: String,
    /// Synth engine: `"none"`, `"schema"`, `"schema+llm"`. Phase 2 ships
    /// `"none"` and `"schema"`; the LLM augmentation is Phase 3.
    pub synth_engine: String,
    /// Mutation policy: `"journal"` (default) or `"block"`.
    pub mutation_policy: String,
    /// HTTP status returned on fail-closed miss. Default 599.
    pub miss_status: u16,
    /// Hosts on which live passthrough is permitted (subset of [`EgressManifest`]).
    pub allow_live_hosts: Vec<String>,
}

/// Destructive-op gate behaviour.
#[derive(Clone, Debug, PartialEq, Eq, Serialize, Deserialize)]
pub struct SyscallShimPolicy {
    /// Layers active in this sandbox. Production default is all three layers.
    ///
    /// Note: Phase 2 replaced the legacy `"ptrace"` layer with
    /// `"seccomp-unotify"` + `"bpf-lsm"` per the Phase 2 currency-check
    /// finding (ptrace adds 300–1000× syscall overhead and has TOCTOU
    /// bypass classes documented in Outflank Dec-2025). Existing manifests
    /// referencing `"ptrace"` are migrated to `"seccomp-unotify"` with a
    /// deprecation warning.
    pub active_layers: Vec<String>,
    /// Gate mode: `"intercept"` (production default), `"block"`,
    /// `"audit-only"` (dev only — never used in prod).
    pub gate_mode: String,
    /// When true, twin-scoped destructives auto-approve. Real-scoped
    /// destructives ALWAYS require Promotion Contract approval regardless
    /// of this flag (architectural invariant — see threat-model.md).
    pub auto_approve_twin_scope: bool,
    /// When true, every interception is recorded to the property-test corpus.
    pub adversarial_test_mode: bool,
}

impl Default for SyscallShimPolicy {
    fn default() -> Self {
        Self {
            active_layers: vec![
                "cmd-line-parse".into(),
                "seccomp-unotify".into(),
                "bpf-lsm".into(),
                "tetragon".into(),
            ],
            gate_mode: "intercept".into(),
            auto_approve_twin_scope: true,
            adversarial_test_mode: false,
        }
    }
}

/// Heartbeat configuration.
#[derive(Clone, Copy, Debug, PartialEq, Eq, Serialize, Deserialize)]
pub struct HeartbeatSpec {
    /// Agent sends a heartbeat every `interval`.
    pub interval: Duration,
    /// Runtime kills the sandbox if no heartbeat seen for `stale_after`.
    pub stale_after: Duration,
}

impl Default for HeartbeatSpec {
    fn default() -> Self {
        Self {
            interval: Duration::from_secs(5),
            stale_after: Duration::from_secs(30),
        }
    }
}

/// Full spawn specification. Content-addressed via [`Self::canonical_hash`].
#[derive(Clone, Debug, PartialEq, Eq, Serialize, Deserialize)]
pub struct SandboxSpec {
    /// Task this sandbox serves.
    pub task_id: String,
    /// Tenant the task belongs to.
    pub tenant_id: String,
    /// Concrete provider.
    pub kind: SandboxKind,
    /// Provider region, e.g. `"aws-us-east-1"`.
    pub provider_region: String,
    /// Resource envelope.
    pub resources: Resources,
    /// Egress allowlist.
    pub egress: EgressManifest,
    /// Secret bindings.
    pub secrets: Vec<SecretBinding>,
    /// Database twin, optional.
    pub db: Option<DbBranchSpec>,
    /// Filesystem twin.
    pub filesystem: FilesystemSpec,
    /// Service twin, optional.
    pub tape: Option<TapeSpec>,
    /// Syscall-shim policy.
    pub shim: SyscallShimPolicy,
    /// Heartbeat config.
    pub heartbeat: HeartbeatSpec,
    /// Absolute TTL (default 1h per ADR-015).
    pub absolute_ttl: Duration,
    /// Informational labels (not security-bearing).
    pub labels: BTreeMap<String, String>,
}

impl SandboxSpec {
    /// Computes the content-address of the spec. The hash is stable across
    /// serialisations because [`BTreeMap`] gives deterministic ordering and
    /// [`spec_hash::canonical_json`] sorts keys recursively.
    #[must_use]
    pub fn canonical_hash(&self) -> SpecHash {
        SpecHash(spec_hash::sha256_canonical(self))
    }

    /// Validates the spec at the boundary. Concrete providers may add
    /// further restrictions — `validate` is the minimum every spec must
    /// pass before any provider sees it.
    ///
    /// # Errors
    /// Returns [`Error::InvalidSpec`] with a human-readable message on the
    /// first violation found. Validations are not exhaustively checked — a
    /// successful return guarantees the listed invariants, not the absence
    /// of all bugs.
    pub fn validate(&self) -> Result<()> {
        if self.task_id.is_empty() {
            return Err(Error::InvalidSpec("task_id empty".into()));
        }
        if self.tenant_id.is_empty() {
            return Err(Error::InvalidSpec("tenant_id empty".into()));
        }
        if !self.kind.allowed_for_tenant_traffic()
            && !self
                .labels
                .get("crucible.io/local-dev")
                .is_some_and(|v| v == "true")
        {
            return Err(Error::InvalidSpec(format!(
                "kind {:?} not allowed for tenant traffic (label crucible.io/local-dev=true required)",
                self.kind
            )));
        }
        if self.resources.vcpus == 0 || self.resources.memory_mb < 256 {
            return Err(Error::InvalidSpec(format!(
                "resources too small: vcpus={}, memory_mb={}",
                self.resources.vcpus, self.resources.memory_mb
            )));
        }
        if self.absolute_ttl < Duration::from_secs(60) {
            return Err(Error::InvalidSpec(format!(
                "absolute_ttl={:?} is below the 60s floor",
                self.absolute_ttl
            )));
        }
        if self.absolute_ttl > Duration::from_secs(24 * 3600) {
            return Err(Error::InvalidSpec(format!(
                "absolute_ttl={:?} exceeds E2B 24h ceiling (ADR-015)",
                self.absolute_ttl
            )));
        }
        for binding in &self.secrets {
            binding.validate_ttl()?;
        }
        if self.heartbeat.interval >= self.heartbeat.stale_after {
            return Err(Error::InvalidSpec(format!(
                "heartbeat interval {:?} must be < stale_after {:?}",
                self.heartbeat.interval, self.heartbeat.stale_after
            )));
        }
        if self.filesystem.base_sha.is_empty() {
            return Err(Error::InvalidSpec("filesystem.base_sha empty".into()));
        }
        if self.shim.gate_mode == "audit-only"
            && self.labels.get("crucible.io/dev-mode").map(String::as_str) != Some("true")
        {
            return Err(Error::InvalidSpec(
                "shim gate_mode=audit-only requires crucible.io/dev-mode=true label"
                    .into(),
            ));
        }
        Ok(())
    }
}

// ─────────────────────────────────────────────────────────────────────────────
// Sandbox state and snapshot types
// ─────────────────────────────────────────────────────────────────────────────

/// Lifecycle state. Advances forward only; terminal states are
/// `Terminated` and `Failed`.
#[derive(Clone, Copy, Debug, PartialEq, Eq, Hash, Serialize, Deserialize)]
#[serde(rename_all = "kebab-case")]
pub enum SandboxState {
    /// Resources reserved, sandbox not yet booted.
    Provisioning,
    /// VM booting / language runtime warming.
    Booting,
    /// Agent SDK handle is live.
    Ready,
    /// Mid-snapshot; transient.
    Paused,
    /// Kill initiated, not yet acked.
    Terminating,
    /// Resources released; only metadata remains.
    Terminated,
    /// Boot failed; never reached Ready.
    Failed,
}

impl SandboxState {
    /// Returns true for terminal states.
    #[must_use]
    pub fn is_terminal(self) -> bool {
        matches!(self, Self::Terminated | Self::Failed)
    }

    /// Returns true if a transition `self -> next` is permitted by the
    /// lifecycle invariants. Used by the runtime's state ledger to reject
    /// out-of-order updates from misbehaving providers.
    #[must_use]
    pub fn can_transition_to(self, next: Self) -> bool {
        use SandboxState::{Booting, Failed, Paused, Provisioning, Ready, Terminated, Terminating};
        match (self, next) {
            (Provisioning, Booting | Failed | Terminating)
            | (Booting, Ready | Failed | Terminating)
            | (Ready, Paused | Terminating | Failed)
            | (Paused, Ready | Terminating | Failed)
            | (Terminating, Terminated) => true,
            _ => false,
        }
    }
}

/// Reasons a sandbox died — recorded in attestations.
#[derive(Clone, Copy, Debug, PartialEq, Eq, Hash, Serialize, Deserialize)]
#[serde(rename_all = "kebab-case")]
pub enum SandboxKillReason {
    /// Task finished normally.
    Clean,
    /// Absolute TTL elapsed.
    Ttl,
    /// Sandbox escape attempted (T18/T19/T20 in threat-model.md).
    EscapeAttempt,
    /// ADR-009 hard cap breached.
    Budget,
    /// Operator or user requested.
    Manual,
    /// Provider host died.
    ProviderFailure,
    /// Heartbeat missed for `stale_after`.
    HeartbeatLost,
    /// Real-scoped destructive proposal was rejected.
    DestructiveDenied,
}

/// Live handle to a running sandbox.
#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct Sandbox {
    /// Runtime-assigned id.
    pub id: SandboxId,
    /// Task and tenant the sandbox serves.
    pub task_id: String,
    /// Tenant.
    pub tenant_id: String,
    /// Concrete provider.
    pub kind: SandboxKind,
    /// Provider-side handle (e.g., E2B sandbox id, Firecracker socket path).
    pub provider_handle: String,
    /// Control endpoint URI (unix or vsock).
    pub control_endpoint: String,
    /// Spawned timestamp.
    pub spawned_at: DateTime<Utc>,
    /// Wall-clock expiry.
    pub expires_at: DateTime<Utc>,
    /// Current state.
    pub state: SandboxState,
    /// Path inside the sandbox the agent SDK writes attestations to.
    pub attestation_socket: String,
    /// Spec hash that spawned this sandbox.
    pub spec_hash: SpecHash,
}

/// Snapshot of a sandbox at a checkpoint boundary.
#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct SnapshotRef {
    /// Snapshot id.
    pub id: SnapshotId,
    /// Source sandbox.
    pub sandbox_id: SandboxId,
    /// Task the source sandbox served.
    pub task_id: String,
    /// Semantic checkpoint name (e.g., `"post-plan"`).
    pub name: String,
    /// Snapshot timestamp.
    pub taken_at: DateTime<Utc>,
    /// Provider-side handle.
    pub provider_handle: String,
    /// Best-effort size estimate in bytes.
    pub size_bytes: u64,
    /// Spec hash of the source sandbox — Restore requires a compatible spec.
    pub base_spec_hash: SpecHash,
    /// Rekor UUID of the last attestation emitted before the snapshot;
    /// restored chains continue from this head.
    pub attestation_chain_head: Option<String>,
}

// ─────────────────────────────────────────────────────────────────────────────
// Error taxonomy
// ─────────────────────────────────────────────────────────────────────────────

/// Errors from any [`SandboxProvider`] method. Mapped 1-to-1 to the
/// `CrucibleError` proto wire type by the runtime gRPC layer.
#[derive(Debug, Error)]
pub enum Error {
    /// Spec failed validation before any provider was contacted.
    #[error("invalid spec: {0}")]
    InvalidSpec(String),

    /// Provider rejected the spec (capability mismatch, unknown region, ...).
    #[error("provider rejected spec: {0}")]
    ProviderRejected(String),

    /// Resource limits exhausted at the provider side.
    #[error("provider quota exhausted: {0}")]
    QuotaExhausted(String),

    /// Auth failure (bad API key, expired token, etc.). Not retryable.
    #[error("authentication failed: {0}")]
    AuthFailed(String),

    /// Sandbox could not be located (id doesn't exist or has been GC'd).
    #[error("sandbox not found: {0}")]
    NotFound(SandboxId),

    /// Snapshot could not be located.
    #[error("snapshot not found: {0:?}")]
    SnapshotNotFound(SnapshotId),

    /// Lifecycle violation — caller attempted an illegal transition.
    #[error("illegal state transition: {0:?} -> {1:?}")]
    IllegalTransition(SandboxState, SandboxState),

    /// Provider-side timeout. Retryable.
    #[error("provider timed out after {0:?}")]
    Timeout(Duration),

    /// Network-layer failure. Retryable.
    #[error("network error: {0}")]
    Network(String),

    /// Sandbox killed for a security violation. Not retryable; surface
    /// to security on-call (RB-03 / RB-04).
    #[error("sandbox killed for security: {0:?}")]
    SecurityKill(SandboxKillReason),

    /// Phase 2 stub: feature was deliberately deferred to Phase 3.
    /// The error message includes the Phase 3 ticket path.
    #[error("STUB: {0}")]
    PhaseStub(String),

    /// Other / unclassified.
    #[error("provider error: {0}")]
    Other(String),
}

impl Error {
    /// Returns true if the caller may retry the operation. Mirrors the
    /// `retryable` field of `CrucibleError` in the proto.
    #[must_use]
    pub fn is_retryable(&self) -> bool {
        matches!(self, Self::Timeout(_) | Self::Network(_))
    }
}

/// Result alias used across the spec.
pub type Result<T> = std::result::Result<T, Error>;

// ─────────────────────────────────────────────────────────────────────────────
// The trait
// ─────────────────────────────────────────────────────────────────────────────

/// Capability flags any provider must expose via [`SandboxProvider::capabilities`].
///
/// Driven by the Postgres-branching research finding (April 2026): the
/// abstract interface needs a probe so the runtime can route correctly
/// across providers with different feature shapes.
#[derive(Clone, Copy, Debug, PartialEq, Eq, Serialize, Deserialize)]
pub struct ProviderCapabilities {
    /// Provider can snapshot a running sandbox without destroying it.
    pub supports_snapshot: bool,
    /// Provider can restore from a snapshot to a new sandbox.
    pub supports_restore: bool,
    /// Provider can pause + resume a sandbox in place.
    pub supports_pause: bool,
    /// Provider supports a per-sandbox egress allowlist *natively* (e.g., E2B
    /// `SandboxNetworkOpts`). If false, the runtime layers its own egress
    /// proxy inside the sandbox.
    pub supports_native_egress: bool,
    /// Provider supports eBPF inside the guest (almost never true on hosted
    /// Firecracker; true on self-hosted with host-attached Tetragon).
    pub supports_guest_ebpf: bool,
    /// Max concurrent sandboxes the provider exposes to one tenant API key.
    /// `None` if not advertised.
    pub max_concurrent: Option<u32>,
}

impl ProviderCapabilities {
    /// E2B's capabilities as of the May 2026 currency check.
    #[must_use]
    pub fn e2b_default() -> Self {
        Self {
            supports_snapshot: true,
            supports_restore: true,
            supports_pause: true,           // GA in TS SDK; Python is beta_pause.
            supports_native_egress: true,    // SandboxNetworkOpts is new in 2026.
            supports_guest_ebpf: false,      // No CAP_BPF inside guest.
            max_concurrent: Some(100),       // Pro plan default.
        }
    }
}

/// The Sandbox Provider trait.
///
/// Implementations are async and must be `Send + Sync` so the runtime can
/// share them across tokio tasks.
#[async_trait]
pub trait SandboxProvider: Send + Sync {
    /// The provider's [`SandboxKind`].
    fn kind(&self) -> SandboxKind;

    /// Capability probe. Should be cheap (no network call) — providers
    /// may cache the response.
    fn capabilities(&self) -> ProviderCapabilities;

    /// Spawn a new sandbox. The returned [`Sandbox`] is in state
    /// [`SandboxState::Provisioning`] or later — never `Failed` (failure
    /// is conveyed via [`Error`] instead).
    async fn spawn(&self, spec: &SandboxSpec) -> Result<Sandbox>;

    /// Snapshot a running sandbox. The sandbox returns to its prior state
    /// after the snapshot completes; if it was `Ready`, it's `Ready` again.
    async fn snapshot(&self, sandbox: &Sandbox, name: &str) -> Result<SnapshotRef>;

    /// Restore a snapshot, optionally rebinding to a new task id. The
    /// resulting sandbox is in `Booting` or later state.
    async fn restore(
        &self,
        snapshot: &SnapshotRef,
        new_task_id: Option<&str>,
    ) -> Result<Sandbox>;

    /// Kill the sandbox. Idempotent: killing an already-terminated sandbox
    /// returns `Ok(())`.
    async fn kill(&self, sandbox: &Sandbox, reason: SandboxKillReason) -> Result<()>;

    /// Report the current state of a sandbox by id. Returns `NotFound` if
    /// the id is unknown to the provider.
    async fn state(&self, id: &SandboxId) -> Result<SandboxState>;

    /// List the provider's view of live sandboxes for a tenant. Used by
    /// the GC daemon to reconcile against the runtime's ledger.
    async fn list(&self, tenant_id: &str) -> Result<Vec<Sandbox>>;
}

// ─────────────────────────────────────────────────────────────────────────────
// Re-exports for downstream crates
// ─────────────────────────────────────────────────────────────────────────────

#[cfg(test)]
mod tests {
    use super::*;
    use std::time::Duration;

    fn minimal_spec() -> SandboxSpec {
        SandboxSpec {
            task_id: "task_test".into(),
            tenant_id: "ten_test".into(),
            kind: SandboxKind::E2b,
            provider_region: "aws-us-east-1".into(),
            resources: Resources::default(),
            egress: EgressManifest::deny_all(),
            secrets: Vec::new(),
            db: None,
            filesystem: FilesystemSpec {
                base_sha: "abcdef0123456789".into(),
                repo_url: "https://github.com/example/repo".into(),
                depth: 1,
                overlay_mode: "overlayfs-linux".into(),
                prewarm_paths: Vec::new(),
            },
            tape: None,
            shim: SyscallShimPolicy::default(),
            heartbeat: HeartbeatSpec::default(),
            absolute_ttl: Duration::from_secs(3600),
            labels: BTreeMap::new(),
        }
    }

    #[test]
    fn minimal_spec_validates() {
        minimal_spec().validate().expect("minimal spec should pass");
    }

    #[test]
    fn empty_task_id_rejected() {
        let mut s = minimal_spec();
        s.task_id.clear();
        assert!(matches!(s.validate(), Err(Error::InvalidSpec(_))));
    }

    #[test]
    fn ttl_floor_enforced() {
        let mut s = minimal_spec();
        s.secrets.push(SecretBinding {
            name: "db".into(),
            vault_path: "/db".into(),
            scope_kind: SecretScopeKind::DynamicPg,
            ttl: Duration::from_secs(3), // below the 5s Infisical floor
            egress_inject_only: true,
        });
        assert!(matches!(s.validate(), Err(Error::InvalidSpec(_))));
    }

    #[test]
    fn ttl_at_floor_accepted() {
        let mut s = minimal_spec();
        s.secrets.push(SecretBinding {
            name: "db".into(),
            vault_path: "/db".into(),
            scope_kind: SecretScopeKind::DynamicPg,
            ttl: Duration::from_secs(5),
            egress_inject_only: true,
        });
        s.validate().expect("5s ttl should pass");
    }

    #[test]
    fn absolute_ttl_ceiling_enforced() {
        let mut s = minimal_spec();
        s.absolute_ttl = Duration::from_secs(24 * 3600 + 1);
        assert!(matches!(s.validate(), Err(Error::InvalidSpec(_))));
    }

    #[test]
    fn heartbeat_must_be_strictly_less_than_stale() {
        let mut s = minimal_spec();
        s.heartbeat = HeartbeatSpec {
            interval: Duration::from_secs(30),
            stale_after: Duration::from_secs(30),
        };
        assert!(matches!(s.validate(), Err(Error::InvalidSpec(_))));
    }

    #[test]
    fn local_docker_rejected_without_label() {
        let mut s = minimal_spec();
        s.kind = SandboxKind::LocalDocker;
        assert!(matches!(s.validate(), Err(Error::InvalidSpec(_))));

        s.labels.insert("crucible.io/local-dev".into(), "true".into());
        s.validate().expect("local-dev label should re-enable");
    }

    #[test]
    fn state_transitions_respect_invariants() {
        assert!(SandboxState::Provisioning.can_transition_to(SandboxState::Booting));
        assert!(SandboxState::Booting.can_transition_to(SandboxState::Ready));
        assert!(SandboxState::Ready.can_transition_to(SandboxState::Paused));
        assert!(SandboxState::Paused.can_transition_to(SandboxState::Ready));
        assert!(SandboxState::Terminating.can_transition_to(SandboxState::Terminated));

        // Forbidden transitions
        assert!(!SandboxState::Terminated.can_transition_to(SandboxState::Ready));
        assert!(!SandboxState::Failed.can_transition_to(SandboxState::Ready));
        assert!(!SandboxState::Ready.can_transition_to(SandboxState::Booting));
    }

    #[test]
    fn canonical_hash_is_deterministic() {
        let s = minimal_spec();
        let h1 = s.canonical_hash();
        let h2 = s.canonical_hash();
        assert_eq!(h1, h2);

        // Permuting labels (insertion order) yields the same hash because
        // BTreeMap normalises ordering.
        let mut s2 = minimal_spec();
        s2.labels.insert("z-last".into(), "v".into());
        s2.labels.insert("a-first".into(), "v".into());
        let mut s3 = minimal_spec();
        s3.labels.insert("a-first".into(), "v".into());
        s3.labels.insert("z-last".into(), "v".into());
        assert_eq!(s2.canonical_hash(), s3.canonical_hash());
    }

    #[test]
    fn canonical_hash_changes_when_security_fields_change() {
        let s1 = minimal_spec();
        let mut s2 = minimal_spec();
        s2.shim.gate_mode = "audit-only".into();
        assert_ne!(s1.canonical_hash(), s2.canonical_hash());
    }

    #[test]
    fn error_retryability_classification() {
        assert!(Error::Timeout(Duration::from_secs(1)).is_retryable());
        assert!(Error::Network("eof".into()).is_retryable());
        assert!(!Error::AuthFailed("bad token".into()).is_retryable());
        assert!(!Error::InvalidSpec("nope".into()).is_retryable());
        assert!(!Error::SecurityKill(SandboxKillReason::EscapeAttempt).is_retryable());
    }
}
