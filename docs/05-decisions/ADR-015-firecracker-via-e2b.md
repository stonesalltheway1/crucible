# ADR-015: E2B (Firecracker) as default sandbox in SaaS tier

**Status:** Accepted  
**Date:** 2026-05-15

## Context

The twin runtime requires a sandbox per task that:

- Spawns fast (< 300ms perceived latency).
- Provides hardware-grade isolation (syscall-level, not just namespace-level).
- Supports overlayfs for COW filesystem.
- Supports network namespace + eBPF egress policy.
- Discards cleanly at task end.
- Scales horizontally with no operational pain.

May 2026 mature options:

| Tech | Cold start | Isolation | Operational complexity | Notes |
|---|---|---|---|---|
| Docker container | 100–500ms | Namespace + cgroup | Low | Insufficient isolation for hostile code |
| gVisor | 200ms | Syscall filter | Medium | 18% syscall overhead; good middle ground |
| Kata Containers | 130–200ms | Hardware (VM) | Medium-high | 130–200MB per pod; heavier |
| Firecracker | 110ms | Hardware (VM) | High (self-orchestrate) | Best isolation/perf ratio |
| E2B | 150ms | Hardware (via Firecracker) | Low (managed) | Vendor-managed, $0.0504/vCPU-hr |
| Daytona | 90ms | Container-based | Low | Docker under the hood |
| Modal Sandbox | 250ms | Hardware (Firecracker) | Low | GPU-capable; heavier |
| Cloudflare Workers | 5ms (V8 isolate) | V8 isolate | Low | Too restrictive for arbitrary code |

## Decision

- **SaaS tier:** E2B as default sandbox provider. Firecracker isolation, managed orchestration, sub-200ms cold start, $0.0504/vCPU-hr, mature SDK in Python and TypeScript.
- **Self-hosted enterprise:** Raw Firecracker + containerd + ZFS, orchestrated by our own scheduler. Marginal cost ≈ $0 per twin.
- **Solo-founder tier:** Daytona or Fly Machines as cheap entry points; degraded isolation guarantees but acceptable for a single-tenant indie-founder context.

## Consequences

### Positive

- **Hardware-grade isolation.** Firecracker boots a real microVM per task; syscall surface is constrained at the hypervisor boundary, not just by seccomp profiles.
- **Fast enough.** ~150ms cold start (E2B managed) doesn't dominate task latency budget.
- **Operationally simple in SaaS.** E2B handles orchestration; we focus on the agent layer.
- **Self-host path exists.** Firecracker is OSS (Apache-2.0); the air-gap installer can run raw Firecracker without E2B dependency.
- **Snapshot-restore is cheap.** Once warm, snapshot-restore is 3–10ms — enables checkpoint-and-fork for fan-out exploration.

### Negative

- **Vendor dependency on E2B for SaaS.** Mitigation: Modal Sandbox is the backup; our orchestration layer abstracts the specific provider via a `SandboxProvider` interface.
- **GPU support requires different provider.** E2B doesn't currently target GPU workloads; if customers need GPU-twins, route to Modal. Not a v1 ICP need.
- **Self-host operational burden.** Running raw Firecracker requires expertise. Mitigation: ship a curated Helm chart for the typical case; offer professional services for non-typical deployments.
- **24-hour max session.** E2B caps at 24h; we don't need longer (median task is minutes).

### Trade-offs we accept

We pay E2B for SaaS-tier convenience and absorb the operational cost of raw Firecracker for self-hosted. The split is correct — the SaaS tier should be operationally lean; the enterprise tier accepts complexity for compliance / data-residency.

## Alternatives considered

### Alternative 1: Docker containers (no microVM)

**Rejected**:

- Namespace isolation is insufficient for hostile-code threat model.
- Container escape vulnerabilities are a regular CVE class.
- The architectural commitment to "treat agent as hostile" demands hardware isolation.

### Alternative 2: gVisor

**Rejected as default** (kept as a possible budget option):

- 18% syscall overhead is meaningful for shell-heavy workloads.
- Isolation is better than plain Docker but not equivalent to Firecracker.
- Worth considering for the solo-founder tier where cost matters more than maximum isolation.

### Alternative 3: Kata Containers

**Rejected**:

- 130–200MB per pod is heavy.
- VM isolation similar to Firecracker but operational story heavier.

### Alternative 4: Self-orchestrate Firecracker from day one

**Rejected for SaaS**:

- ~2 agent-days of orchestration work just to match E2B's offering.
- E2B at $0.0504/vCPU-hr is cheap enough that the build vs buy doesn't justify build.
- Self-orchestration becomes necessary for the self-hosted tier anyway; we do it once for that, not twice.

### Alternative 5: Modal Sandbox

**Considered**; kept as fallback:

- $0.25/vCPU-hr is more expensive than E2B.
- GPU-capable, which is a v2 differentiator but not v1 need.

### Alternative 6: Cloudflare Workers

**Rejected**:

- V8 isolates are too restrictive — can't run arbitrary languages or binary tools.
- Wrong abstraction level for "agent runs a shell command."

## Provider abstraction

The `SandboxProvider` interface lets us swap providers:

```go
type SandboxProvider interface {
    Spawn(ctx context.Context, spec SandboxSpec) (*Sandbox, error)
    Snapshot(sandbox *Sandbox, name string) (*SnapshotRef, error)
    Restore(snapshot *SnapshotRef) (*Sandbox, error)
    Kill(sandbox *Sandbox) error
}
```

Implementations: `e2b`, `modal`, `daytona`, `fly-machines`, `raw-firecracker`. Per-tenant config selects the provider.

## Operational notes

### E2B SaaS

- API key per tenant for usage attribution.
- Crucible itself maintains a default "platform" account for shared infrastructure.
- Sandbox lifetime tied to task lifetime; orphan-detection sweeps every 5 min.

### Raw Firecracker (self-host)

- ZFS pool per node for snapshot speed.
- Pre-warmed sandbox pool (default 20 per node) to absorb burst.
- Per-tenant cgroup quotas to prevent neighbor effects.
- Network namespaces with Cilium / Tetragon eBPF policy.

### Backup providers in routing

If E2B is unreachable, fallback to Modal automatically. Customer-visible: "fallback provider in use" banner.

## Open issues

- **GPU workloads.** v1 ICP doesn't need them; v2 if ML-engineering customers materialize.
- **Windows containers.** Some legacy enterprise stacks require Windows VMs; not supported in v1.
- **macOS targets (iOS/macOS dev).** Mac-Cloud or MacStadium integration possible; not v1 scope.

## References

- [01-architecture/twin-runtime.md#layer-1-sandbox](../01-architecture/twin-runtime.md)
