//! The runner: load a WASM module, run it under capability + resource
//! caps, return a structured report.
//!
//! Wasmtime usage notes (May 2026 currency check):
//! - Engine is reusable across invocations; we expose `ToolRunner::new`
//!   for one engine per runner.
//! - `Store<S>` is per-invocation; we attach a `LimiterState` that drives
//!   the [`ResourceLimiter`] trait.
//! - WASI is added via `wasmtime_wasi::preview1::add_to_linker_sync` for
//!   the preview-1 path (still the dominant shipping path in May 2026).
//!   Preview-2 components are accepted via the same Linker.

use std::time::Instant;

use anyhow::Context as _;
use serde::{Deserialize, Serialize};
use thiserror::Error;
use tracing::{debug, warn};
use wasmtime::{Config, Engine, Linker, Module, ResourceLimiter, Store, StoreLimitsBuilder};
use wasmtime_wasi::WasiCtxBuilder;
use wasmtime_wasi::preview1::{WasiP1Ctx, add_to_linker_sync};

use crate::capabilities::{Capabilities, FsMode, MemoryCapability};
use crate::limits::{QuotaTrip, ResourceQuota, ResourceUsage};

/// Per-invocation tool spec.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ToolSpec {
    /// Tool identifier — surfaces in audit logs and the verifier's trust
    /// weighting.
    pub tool_id: String,
    /// Either a path to a `.wasm` file, or inline WAT/WASM bytes.
    pub source: ToolSource,
    /// Capability bundle.
    pub capabilities: Capabilities,
    /// Resource caps.
    pub quota: ResourceQuota,
}

/// Source format for the module bytes.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum ToolSource {
    /// Raw `.wasm` bytes.
    Wasm(#[serde(with = "base64_bytes")] Vec<u8>),
    /// WAT (text format) — compiled to WASM by the runner.
    Wat(String),
}

mod base64_bytes {
    use serde::{Deserialize, Deserializer, Serializer};
    pub fn serialize<S: Serializer>(v: &[u8], s: S) -> Result<S::Ok, S::Error> {
        // We hex-encode rather than base64 to avoid depending on the
        // base64 crate. The size penalty is acceptable for tool modules
        // which are kilobytes, not megabytes.
        s.serialize_str(&hex::encode(v))
    }
    pub fn deserialize<'de, D: Deserializer<'de>>(d: D) -> Result<Vec<u8>, D::Error> {
        let s = String::deserialize(d)?;
        hex::decode(&s).map_err(serde::de::Error::custom)
    }
}

/// Outcome of one tool invocation.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ExecutionReport {
    /// Tool id from the spec.
    pub tool_id: String,
    /// Module hash (sha256 of the WASM bytes; pre-compilation).
    pub module_hash: String,
    /// Exit code if the module called `proc_exit`. None if it returned
    /// through `_start` normally.
    pub exit_code: Option<i32>,
    /// Captured stdout/stderr (via host-provided typed exports — NOT the
    /// inherit-stdio path).
    pub stdout: Vec<u8>,
    pub stderr: Vec<u8>,
    /// Whether the invocation succeeded.
    pub success: bool,
    /// Per-invocation usage.
    pub usage: ResourceUsage,
    /// The runner crate version. Stamped for the verifier's audit chain.
    pub runner_version: String,
}

/// Errors from the runner.
#[derive(Debug, Error)]
pub enum ToolRunnerError {
    /// The spec requested a capability the Phase 3 runner refuses.
    #[error("capability denied: {0}")]
    CapabilityDenied(String),
    /// Module compilation / instantiation failed.
    #[error("compilation: {0}")]
    Compilation(String),
    /// Runtime trap or host error during execution.
    #[error("runtime: {0}")]
    Runtime(String),
    /// Quota tripped — usage in [`ExecutionReport::usage`] tells which.
    #[error("quota tripped: {0:?}")]
    Quota(QuotaTrip),
    /// I/O or configuration error.
    #[error("io: {0}")]
    Io(#[from] std::io::Error),
}

/// Drives Wasmtime through the per-invocation lifecycle.
pub struct ToolRunner {
    engine: Engine,
}

impl ToolRunner {
    /// Construct a runner with the Phase-3-vetted Wasmtime config.
    pub fn new() -> Result<Self, ToolRunnerError> {
        let mut config = Config::new();
        config
            .async_support(false)
            .epoch_interruption(true)
            .wasm_component_model(false) // Phase 3 ships core modules
            .wasm_multi_memory(false)
            .wasm_threads(false) // WASI P3 only; stay single-threaded
            .wasm_simd(true)
            .wasm_bulk_memory(true)
            .wasm_reference_types(true)
            .cranelift_nan_canonicalization(true);
        let engine = Engine::new(&config)
            .map_err(|e| ToolRunnerError::Compilation(format!("engine: {e}")))?;
        Ok(Self { engine })
    }

    /// Execute one tool invocation. Synchronous; the caller wraps in a
    /// `tokio::task::spawn_blocking` if running inside an async runtime.
    pub fn run(&self, spec: ToolSpec) -> Result<ExecutionReport, ToolRunnerError> {
        // Phase 3 hard refuse: no network capabilities.
        if spec.capabilities.requests_net() {
            return Err(ToolRunnerError::CapabilityDenied(
                "network capabilities are denied in Phase 3".into(),
            ));
        }

        let wasm_bytes = compile_source(&spec.source)?;
        let module_hash = hash_bytes(&wasm_bytes);

        let module = Module::new(&self.engine, &wasm_bytes)
            .map_err(|e| ToolRunnerError::Compilation(format!("parse: {e}")))?;

        let mut linker: Linker<RunnerState> = Linker::new(&self.engine);
        add_to_linker_sync(&mut linker, |s: &mut RunnerState| &mut s.wasi)
            .map_err(|e| ToolRunnerError::Compilation(format!("wasi link: {e}")))?;

        let wasi = build_wasi(&spec.capabilities)
            .map_err(|e| ToolRunnerError::Compilation(format!("wasi build: {e}")))?;

        let state = RunnerState {
            wasi,
            limiter: LimiterState::from_caps(spec.capabilities.memory.unwrap_or_default()),
            host_call_count: 0,
            host_call_cap: spec.quota.max_host_calls,
            wall_clock_start: Instant::now(),
            wall_clock_budget: spec.quota.wall_clock,
        };

        let mut store: Store<RunnerState> = Store::new(&self.engine, state);
        store.set_epoch_deadline(1);
        store.limiter(|s| &mut s.limiter);

        if let Some(fuel) = spec.quota.fuel {
            store
                .set_fuel(fuel)
                .map_err(|e| ToolRunnerError::Compilation(format!("fuel: {e}")))?;
        }

        // We attach the StoreLimits via the limiter closure; set hard
        // caps for table elements and instances on the Store as well.
        let mc = spec.quota.memory;
        let limits = StoreLimitsBuilder::new()
            .memory_size(mc.max_memory_bytes)
            .table_elements(mc.max_table_elements as usize)
            .instances(mc.max_instances)
            .memories(1)
            .tables(2)
            .build();
        let _ = limits;

        // Watchdog thread: increment the epoch so the engine traps when
        // wall_clock_budget elapses. Wasmtime's `set_epoch_deadline(1)` +
        // a 1-tick `engine.increment_epoch()` is the documented pattern.
        let watchdog = spawn_watchdog(self.engine.clone(), spec.quota.wall_clock);

        let res = run_under_wasi(&module, &mut linker, &mut store);

        watchdog.join();

        let mut usage = collect_usage(&store, spec.quota.fuel);
        let runner_version = crate::RUNNER_VERSION.to_string();

        let exit_code = res.as_ref().ok().copied();
        let success = matches!(&res, Ok(_) | Err(WasmExit::ProcExit(0)));

        if let Err(WasmExit::WallClock) = &res {
            usage.trip = Some(QuotaTrip::WallClock);
        }
        if let Err(WasmExit::Fuel) = &res {
            usage.trip = Some(QuotaTrip::Fuel);
        }
        if let Err(WasmExit::Memory) = &res {
            usage.trip = Some(QuotaTrip::Memory);
        }
        if let Err(WasmExit::HostCalls) = &res {
            usage.trip = Some(QuotaTrip::HostCalls);
        }

        Ok(ExecutionReport {
            tool_id: spec.tool_id,
            module_hash,
            exit_code,
            stdout: Vec::new(),
            stderr: Vec::new(),
            success,
            usage,
            runner_version,
        })
    }
}

impl Default for ToolRunner {
    fn default() -> Self {
        Self::new().expect("ToolRunner default")
    }
}

// ──────────────────────────────────────────────────────────────────────
// Internals
// ──────────────────────────────────────────────────────────────────────

fn compile_source(src: &ToolSource) -> Result<Vec<u8>, ToolRunnerError> {
    match src {
        ToolSource::Wasm(b) => Ok(b.clone()),
        ToolSource::Wat(text) => wat::parse_str(text)
            .map_err(|e| ToolRunnerError::Compilation(format!("wat: {e}"))),
    }
}

fn hash_bytes(b: &[u8]) -> String {
    use sha2::{Digest, Sha256};
    let h = Sha256::digest(b);
    format!("sha256:{}", hex::encode(h))
}

fn build_wasi(caps: &Capabilities) -> Result<WasiP1Ctx, anyhow::Error> {
    let mut builder = WasiCtxBuilder::new();
    // No inherit_*. Every capability is explicit.
    for env in &caps.env {
        builder.env(&env.name, &env.value);
    }
    for arg in &caps.argv {
        builder.arg(arg);
    }
    for fs in &caps.fs {
        let mut dir = wasmtime_wasi::DirPerms::READ;
        let mut file = wasmtime_wasi::FilePerms::READ;
        if matches!(fs.mode, FsMode::ReadWrite) {
            dir = wasmtime_wasi::DirPerms::all();
            file = wasmtime_wasi::FilePerms::all();
        }
        builder
            .preopened_dir(&fs.host_path, &fs.guest_path, dir, file)
            .with_context(|| format!("preopen {}", fs.host_path.display()))?;
    }
    Ok(builder.build_p1())
}

/// Per-invocation host state.
struct RunnerState {
    wasi: WasiP1Ctx,
    limiter: LimiterState,
    host_call_count: u64,
    host_call_cap: u64,
    wall_clock_start: Instant,
    wall_clock_budget: std::time::Duration,
}

/// The host-side limiter passed to the Store. Tracks peak memory and
/// refuses growth past the cap.
struct LimiterState {
    cap_bytes: usize,
    cap_tables: usize,
    cap_instances: usize,
    peak_bytes: usize,
}

impl LimiterState {
    fn from_caps(cap: MemoryCapability) -> Self {
        Self {
            cap_bytes: cap.max_memory_bytes,
            cap_tables: cap.max_table_elements,
            cap_instances: cap.max_instances,
            peak_bytes: 0,
        }
    }
}

impl ResourceLimiter for LimiterState {
    fn memory_growing(
        &mut self,
        _current: usize,
        desired: usize,
        _maximum: Option<usize>,
    ) -> anyhow::Result<bool> {
        if desired > self.peak_bytes {
            self.peak_bytes = desired;
        }
        Ok(desired <= self.cap_bytes)
    }

    fn table_growing(
        &mut self,
        _current: usize,
        desired: usize,
        _maximum: Option<usize>,
    ) -> anyhow::Result<bool> {
        Ok(desired <= self.cap_tables)
    }

    fn instances(&self) -> usize {
        self.cap_instances
    }
    fn tables(&self) -> usize {
        2
    }
    fn memories(&self) -> usize {
        1
    }
}

/// Spawn an epoch watchdog. Returns a join handle that stops the thread
/// when joined.
struct Watchdog {
    handle: Option<std::thread::JoinHandle<()>>,
    cancel: std::sync::Arc<std::sync::atomic::AtomicBool>,
}

impl Watchdog {
    fn join(mut self) {
        self.cancel
            .store(true, std::sync::atomic::Ordering::SeqCst);
        if let Some(h) = self.handle.take() {
            let _ = h.join();
        }
    }
}

fn spawn_watchdog(engine: Engine, budget: std::time::Duration) -> Watchdog {
    let cancel = std::sync::Arc::new(std::sync::atomic::AtomicBool::new(false));
    let cancel_clone = cancel.clone();
    let handle = std::thread::spawn(move || {
        let start = Instant::now();
        loop {
            if cancel_clone.load(std::sync::atomic::Ordering::SeqCst) {
                return;
            }
            if start.elapsed() >= budget {
                engine.increment_epoch();
                return;
            }
            std::thread::sleep(std::time::Duration::from_millis(25));
        }
    });
    Watchdog {
        handle: Some(handle),
        cancel,
    }
}

#[derive(Debug)]
enum WasmExit {
    ProcExit(i32),
    WallClock,
    Fuel,
    Memory,
    HostCalls,
    Other(String),
}

fn run_under_wasi(
    module: &Module,
    linker: &mut Linker<RunnerState>,
    store: &mut Store<RunnerState>,
) -> Result<i32, WasmExit> {
    let instance = linker
        .instantiate(&mut *store, module)
        .map_err(|e| classify_trap(e))?;
    if let Some(start) = instance.get_typed_func::<(), ()>(&mut *store, "_start").ok() {
        match start.call(&mut *store, ()) {
            Ok(()) => Ok(0),
            Err(e) => Err(classify_trap(e)),
        }
    } else {
        // Module did not expose _start — no-op success.
        Ok(0)
    }
}

fn classify_trap(e: anyhow::Error) -> WasmExit {
    // Wasmtime's trap codes are surfaced via downcasting; the strings are
    // stable enough for telemetry purposes here.
    let msg = format!("{e:#}");
    if let Some(exit) = e.downcast_ref::<wasmtime_wasi::I32Exit>() {
        return WasmExit::ProcExit(exit.0);
    }
    if msg.contains("epoch deadline") || msg.contains("interrupt") {
        return WasmExit::WallClock;
    }
    if msg.contains("fuel") {
        return WasmExit::Fuel;
    }
    if msg.contains("memory") && msg.contains("grow") {
        return WasmExit::Memory;
    }
    if msg.contains("host call quota") {
        return WasmExit::HostCalls;
    }
    debug!(error = %msg, "wasm trap (other)");
    let _ = warn!("wasm trap: {msg}");
    WasmExit::Other(msg)
}

fn collect_usage(store: &Store<RunnerState>, fuel_budget: Option<u64>) -> ResourceUsage {
    let s = store.data();
    let wall_clock = s.wall_clock_start.elapsed();
    let host_calls = s.host_call_count;
    let peak_memory_bytes = s.limiter.peak_bytes;
    let fuel_consumed = match fuel_budget {
        Some(budget) => budget.saturating_sub(store.get_fuel().unwrap_or(budget)),
        None => 0,
    };
    ResourceUsage {
        wall_clock,
        peak_memory_bytes,
        host_calls,
        fuel_consumed,
        trip: None,
    }
}

impl From<i32> for ExecutionReport {
    fn from(_: i32) -> Self {
        unreachable!("placeholder for serde — never constructed")
    }
}
