# `crucible-twin-self-host`

Raw Firecracker + containerd + ZFS orchestrator for the Crucible
self-hosted enterprise tier. Parallel to the E2B-backed
`apps/twin-runtime` for SaaS; the two implement the same
`SandboxProvider` trait so all sandbox-using code is orchestrator-
agnostic.

Phase 3 lands the scaffold + the ZFS clone-per-task path + the
pre-warmed pool + the per-tenant cgroup quotas. Production deployments
enable the `linux-firecracker` Cargo feature and ship the binary in the
air-gap installer's OCI image set.

## What it does

```
Per-host daemon (one per Crucible node)
    │
    ├─ Pre-warmed pool: keeps N sandbox snapshots ready (default 20)
    │  - Snapshot-restore latency: ~3ms (memory resume) per OpenZFS+Firecracker bench
    │  - First-userland-ready: ~25-30ms (the full restore picture)
    │
    ├─ ZFS dataset per project (lower layer)
    │  - `zfs clone` per task → COW upper layer
    │  - `zfs destroy` on cleanup
    │
    ├─ firecracker-containerd runtime: launches a Firecracker microVM
    │  per spawn with the requested rootfs + kernel
    │
    ├─ Per-tenant cgroup quotas (v2): cpu.max, memory.max, io.max
    │
    ├─ Network namespace per sandbox with Cilium / Tetragon eBPF policy
    │  attached at the HOST (not in-guest — per the Phase 2 finding that
    │  E2B-tier guests lack CAP_BPF)
    │
    └─ gRPC server: implements `SandboxProvider` over the same proto as
       `apps/twin-runtime`. Control plane routes per-tenant config.
```

## Cargo features

- `default` — compiles the binary on every platform but the spawn path
  returns a typed `Error::PhaseStub` indicating the missing host
  primitives (Linux + Firecracker + ZFS).
- `linux-firecracker` — wires the real `firec` crate. Requires Linux
  6.x kernel, KVM, ZFS, and CAP_NET_ADMIN.

## Production checklist

1. Linux 6.6+ kernel with KVM, vhost-user-blk, BPF, cgroups v2 enabled.
2. ZFS pool created and mounted at `/var/lib/crucible/zfs`.
3. `firecracker` binary available at `/usr/local/bin/firecracker`.
4. `firecracker-containerd` shim installed.
5. Cilium running on the host with Tetragon policies pre-deployed.
6. Per-tenant cgroups configured under `/sys/fs/cgroup/crucible/`.
7. `linux-firecracker` Cargo feature enabled in the build.

## Phase 3 stubs

- Real `firec` invocations: gated behind `linux-firecracker`.
- ZFS clone via `libzfs_core`: shell-out via `zfs` CLI for Phase 3
  (matches the Brave / Modal patterns); native FFI is a Phase 4 polish.
- Tetragon policy submission: rendered + persisted to disk; the
  per-host Tetragon daemon picks it up via its watch directory.
