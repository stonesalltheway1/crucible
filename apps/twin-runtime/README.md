# Crucible Twin Runtime

The execution surface for every agent action — Phase 2 of the build plan.

The runtime is a Rust workspace with one binary (`twin-runtime-server`) and
several supporting crates. The split exists for two reasons:

1. **Mutation testing scopes per-crate.** The syscall-shim's brand-promise
   bar of ≥95% mutation score is far easier to verify when the shim is its
   own crate with its own test corpus.
2. **The compile graph is faster.** A change to the sandbox driver doesn't
   force rebuilds of the egress allowlist code.

## Layout

```
apps/twin-runtime/
├── crates/
│   ├── twin-runtime-proto/      — tonic-generated proto types
│   ├── twin-runtime-shim/       — THE brand promise: 3-layer destructive-op gate
│   ├── twin-runtime-sandbox/    — SandboxProvider implementations (E2B, raw-Firecracker)
│   ├── twin-runtime-fs/         — git worktree + overlayfs orchestration
│   ├── twin-runtime-egress/     — manifest enforcement + Tetragon TracingPolicy + mitmproxy
│   ├── twin-runtime-lifecycle/  — spawn / snapshot / kill / GC + heartbeat loop
│   ├── twin-runtime-attest/     — attestation emission + signing
│   └── twin-runtime-server/     — gRPC TwinRuntimeService binary
├── Cargo.toml                   — workspace
└── README.md                    — this file
```

## Architecture cheat-sheet

```
┌──────────────────────────────────────────────────────────────────────┐
│ Control plane                                                        │
└──────────────────────────────────┬───────────────────────────────────┘
                                   │ TwinRuntimeService gRPC
                                   ▼
┌──────────────────────────────────────────────────────────────────────┐
│ twin-runtime-server (Rust, tokio + tonic)                            │
│                                                                      │
│  Spawn(SandboxSpec)                                                  │
│     ├── lifecycle::Orchestrator::spawn()                             │
│     │     ├── sandbox::Provider::spawn() ─→ E2B / Firecracker        │
│     │     ├── fs::mount(base_sha, overlay)                           │
│     │     ├── egress::apply(manifest)                                │
│     │     ├── shim::install_layers(policy) ── Layer 1 cmd-parse      │
│     │     │                                ── Layer 2a seccomp_unotify│
│     │     │                                ── Layer 2b BPF-LSM       │
│     │     │                                ── Layer 3  Tetragon      │
│     │     └── attest::emit(SandboxSpawned)                           │
│     └── return Sandbox{ ready, id, attestation_socket }              │
│                                                                      │
│  AgentSdkService (twin.* — runs on the same gRPC server, scoped to   │
│  the sandbox's unix/vsock endpoint)                                  │
└──────────────────────────────────────────────────────────────────────┘
```

## Build

```
nix develop .#rust-only          # hermetic toolchain
cargo build --workspace
cargo test --workspace
cargo clippy --workspace -- -D warnings
cargo mutants -p twin-runtime-shim --in-diff   # CI gate ≥ 95%
```

For the property tests:

```
cargo test -p twin-runtime-shim --release -- --include-ignored property::shim_intercepts_50k_adversarial
```

## Layer 2 — design note

Phase 2 deliberately diverges from `docs/01-architecture/twin-runtime.md` on
layer 2. The doc names `ptrace` as the runtime-interception primitive.
Currency-check research (May 2026) found:

- ptrace adds 300–1000× syscall overhead — unusable on `rg`/`find`-heavy
  agent workloads.
- ptrace+seccomp has well-known TOCTOU bypass classes (Outflank Dec-2025
  seccomp-notify-injection writeup).
- All examined production AI sandboxes (Modal Labs, GKE Agent Sandbox,
  Bottlerocket) use seccomp-bpf + BPF LSM, **not** ptrace.

We replace `ptrace` with:

- **seccomp-bpf classic** — static allow/deny list, ~70–100 ns/syscall.
- **`SECCOMP_RET_USER_NOTIF`** with mandatory `SECCOMP_IOCTL_NOTIF_ID_VALID`
  cookie gating — for syscalls needing supervisor policy (argv inspection
  on `execve`, destination on `connect`).
- **BPF LSM** at `inode_unlink` / `inode_rename` / `path_truncate` /
  `file_open` — kernel-resolved `struct path`, no TOCTOU.
- **Landlock** as defense-in-depth FS confinement.
- **Tetragon** (host-attached on self-hosted Firecracker; not in-guest on
  E2B because the E2B guest lacks `CAP_BPF`) for post-exec audit + async
  kill.

The `SyscallShimPolicy.active_layers` field accepts the legacy `"ptrace"`
identifier and the runtime maps it to `"seccomp-unotify"` with a deprecation
warning.

## Phase 2 stub markers

`grep -R "STUB:" apps/twin-runtime/` enumerates the deferred Phase 3 surface:

- Raw Firecracker self-host (E2B-only in Phase 2)
- MySQL / SQLite / Mongo DB twins
- Presidio + spaCy + FF3-1 PII pipeline (regex-only in Phase 2)
- WASM tool runner
- LLM-synth tape responses for unknown endpoints
- ZFS-based filesystem isolation
- Multi-region orchestration
- GPU-capable twins
