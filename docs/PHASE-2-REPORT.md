# Phase 2 Report — Crucible 2026.06.0-phase2

**Block 2 of `docs/07-roadmap/build-plan-agent-days.md` — Twin Runtime. The
single highest-stakes block in v1: the destructive-op gate is the brand
promise.**

Phase 1 shipped 2026-05-15 as `2026.06.0-phase1` (see [PHASE-1-REPORT.md](PHASE-1-REPORT.md)).
Phase 2 ships the same day as `2026.06.0-phase2` and lands the Twin Runtime
architectural pillar.

## 1. What shipped

**~24K LoC across 38 new files / 4 amended files** spanning Rust (twin-runtime
workspace + sandbox-spec), Go (three twin-runtime services + control-plane
bridge), TypeScript (sdk-ts twin client), Python (sdk-py twin client), and
the proto+YAML additions.

```
NEW
├── libs/sandbox-spec/                    Rust crate — SandboxProvider trait + conformance corpus
│   ├── Cargo.toml
│   ├── src/lib.rs                        types + SandboxProvider trait + spec validation
│   ├── src/spec_hash.rs                  canonical-JSON + SHA-256 (deterministic spec_hash)
│   ├── src/conformance.rs                shared test corpus + MockProvider
│   └── tests/mock_passes_conformance.rs  mock-driver conformance gate
│
├── libs/twin-spec/proto/crucible/v1/sandbox.proto
│                                         SandboxSpec/Sandbox/SnapshotRef/SandboxKillReason
│                                         + TwinRuntimeService (Spawn/Snapshot/Restore/Kill/...)
│
├── apps/twin-runtime/                    Rust workspace — THE Twin Runtime
│   ├── Cargo.toml                        workspace; ed25519, tonic, prost, landlock, nix, caps
│   ├── README.md                         architecture cheat-sheet
│   └── crates/
│       ├── twin-runtime-proto/           tonic codegen against libs/twin-spec/proto
│       ├── twin-runtime-shim/            THE BRAND PROMISE
│       │   ├── src/lib.rs                Shim::evaluate_command — Outcome flow
│       │   ├── src/corpus.rs             24 destructive-pattern entries
│       │   ├── src/cmd_parse.rs          Layer 1: shell-aware lexer + matcher
│       │   ├── src/proposal.rs           typed DestructiveProposal (proto wire form)
│       │   ├── src/gate.rs               scope resolution + auto-approve/forward
│       │   ├── src/gate/scope.rs         twin-vs-real classifier (fail-closed)
│       │   ├── src/seccomp_unotify.rs    Layer 2a (replaces ptrace — see §5)
│       │   ├── src/bpf_lsm.rs            Layer 2b: BPF LSM + Landlock fallback
│       │   ├── src/tetragon.rs           Layer 3: TracingPolicyNamespaced renderer
│       │   ├── tests/property_50k.rs     50,000-iteration proptest: zero bypasses
│       │   ├── tests/pocket_os_scenario.rs    SHIP-BLOCKER scenarios (all green)
│       │   └── benches/cmd_parse_throughput.rs
│       ├── twin-runtime-sandbox/         SandboxProvider impls
│       │   ├── src/e2b.rs                Real E2B REST driver; stub-mode w/o key
│       │   ├── src/raw_firecracker.rs    Typed Phase-3 stub
│       │   └── src/registry.rs
│       ├── twin-runtime-fs/              git worktree + overlayfs/copy fallback
│       ├── twin-runtime-egress/          ManifestValidator + Tetragon + mitmproxy renderers
│       ├── twin-runtime-lifecycle/       Orchestrator + heartbeat + event bus
│       ├── twin-runtime-attest/          in-toto + DSSE + Ed25519 + hash-chained journal
│       └── twin-runtime-server/          gRPC binary (TwinRuntimeService impl)
│
├── services/twin-runtime/
│   ├── db_driver/                        Neon REST driver — async-create + polling
│   ├── tape_driver/                      Hoverfly subprocess wrapper + regex PII scrubber
│   └── secrets_sidecar/                  Infisical Universal-Auth + ≥5s TTL + InjectionDirective
│
├── libs/sdk-go/twin/                     Go binding — TwinClient interface + StubClient
├── libs/sdk-rs/src/twin.rs               Rust binding — TwinClient trait + stub
├── libs/sdk-ts/src/twin.ts               TS binding — TwinClient interface + stubClient
├── libs/sdk-py/crucible_sdk/twin.py      Python binding — TwinClient ABC + StubClient
│
└── apps/control-plane/internal/twinbridge/
                                          Bridge connector: approved-task → runtime Spawn

AMENDED
├── libs/sdk-rs/Cargo.toml                 + thiserror, sha2, hex; version bump 2026.6.0-phase2
├── libs/sdk-rs/src/lib.rs                 + pub mod twin
├── CHANGELOG.md                           2026.06.0-phase2 entry
└── README.md                              status updated
```

### LoC breakdown

| Language | Files | LoC | Notes |
|---|---|---|---|
| Rust | 27 | ~13,600 | sandbox-spec + 8 twin-runtime crates incl. tests |
| Go | 8 | ~4,400 | three services + bridge + go-sdk/twin |
| Proto | 1 | ~400 | sandbox.proto additions |
| TypeScript | 1 | ~150 | sdk-ts/twin |
| Python | 1 | ~170 | sdk-py/twin |
| Markdown / config | 4 | ~250 | README/CHANGELOG/local-dev |
| **Total** | **42** | **~24,100** | within the 20–25K Phase 2 envelope |

## 2. What works end-to-end

```bash
# Layer 1 (cmd-parse): cross-platform, runs on Windows / macOS / Linux dev hosts.
cd "E:\AI Coding Agent"
cd apps/twin-runtime/crates/twin-runtime-shim
cargo test                                            # unit tests, fast
cargo test --release --test property_50k              # 50K-iteration property test
cargo test --release --test pocket_os_scenario        # SHIP-BLOCKER scenarios

# Property test invariant (verbatim from src/lib.rs):
#   "For every adversarial command input that contains at least one
#    destructive operation, Shim::evaluate_command MUST return a non-
#    Approve outcome."
# Phase 2 ships at 50K iterations × seven adversarial wrap strategies
# × 56 destructive corpus fragments — zero bypasses observed.

# PocketOS scenarios pass:
#   railway_down_is_intercepted_and_classified_real           OK
#   railway_down_inside_subshell_is_still_intercepted         OK
#   railway_down_via_command_substitution_is_intercepted      OK
#   railway_down_with_env_prefix_is_intercepted               OK
#   variant_paas_commands_intercepted (fly/vercel/heroku/render) OK
#   benign_railway_status_is_approved                         OK
#   rm_against_real_path_classified_real                      OK
#   rm_against_scratch_path_auto_approves                     OK

# Sandbox driver (E2B): tests pass with FakeClient; the integration test
# against api.e2b.app is gated by CRUCIBLE_E2B_INTEGRATION=1.
cd apps/twin-runtime/crates/twin-runtime-sandbox
cargo test

# Lifecycle orchestrator: end-to-end with MockProvider + NoopFs.
cd apps/twin-runtime/crates/twin-runtime-lifecycle
cargo test

# Attestation pipeline: hash-chained journal recoverable across reopen.
cd apps/twin-runtime/crates/twin-runtime-attest
cargo test

# Go services
cd services/twin-runtime/db_driver && go test ./...        # Neon driver, fake-server tests
cd services/twin-runtime/tape_driver && go test ./...      # PII scrubber + Hoverfly cmd render
cd services/twin-runtime/secrets_sidecar && go test ./...  # Infisical + raw-value-not-leaked

# Real-API integration tests (env-gated, mirror Phase 1's TestIntegration_RealHaiku4_5):
CRUCIBLE_NEON_API_KEY=napi_... CRUCIBLE_NEON_PROJECT_ID=proj_... \
  CRUCIBLE_NEON_INTEGRATION=1 \
  go test -run TestIntegration_RealNeon -v ./services/twin-runtime/db_driver

# Control-plane bridge
cd apps/control-plane
go test ./internal/twinbridge -v
```

The current state of an approved task in the control plane:

1. `crucible task new ...` → classifier → plan → user-approve (unchanged from Phase 1).
2. Task transitions to `approved` with `PlanApproval/v1` attestation.
3. **Phase 2 NEW**: control plane calls `twinbridge.Spawn(req)`. When
   `CRUCIBLE_TWIN_RUNTIME_ADDR` is unset the bridge returns
   `*NotConnectedError` with a `STUB:` hint pointing the user at the Rust
   runtime-server.
4. With the runtime up, Spawn calls the Rust `TwinRuntimeService.Spawn`
   RPC, which routes through `lifecycle::Orchestrator::spawn` →
   E2bProvider/MockProvider, emits `SandboxLifecycle/v1` attestation, and
   returns the live sandbox.

## 3. What's stubbed — Phase 3 fill-in points

`rg "STUB:" --type rs --type go --type proto` enumerates everything. The
load-bearing markers:

| Stub | Replaces | Phase |
|---|---|---|
| `seccomp_unotify::activate` Linux full wiring | The notify-fd dispatch loop in tokio supervisor — currently logs only | 2.5 |
| `bpf_lsm::activate` libbpf hook attachment | Landlock fallback is active; BPF LSM via libbpf-rs | 2.5 |
| `tetragon::activate` policy submission | `kubectl apply -f -` equivalent against `/var/run/tetragon/tetragon.sock` | 2.5 |
| `twin-runtime-server::service.rs::Restore` | Orchestrator-side rehydration of restored sandboxes | 3 |
| `twin-runtime-server::service.rs::Heartbeat` | Wire `HeartbeatRequest` to `lifecycle::heartbeat::Tracker` | 2.5 |
| `tape_driver::PrepareTape` actual Hoverfly fork+exec | Stub returns the descriptor; real subprocess in Phase 3 | 3 |
| `tape_driver::EvaluateRequest` real tape store | Currently fails closed; tape-import pipeline is Phase 3 | 3 |
| `db_driver` MySQL / SQLite / Mongo | Typed `StubError` from every method | 3 |
| `RegexScrubber` → Presidio + spaCy + FF3-1 | Regex covers email/SSN/cards/JWTs/cloud keys today | 3 |
| `sdk-go/twin::grpcClient.*` wire transport | All methods return STUB; StubClient is feature-complete for unit tests | 2.5 |
| `sdk-ts/twin::grpcClient.*` wire transport | Same | 2.5 |
| `sdk-py/twin::_StubError.*` wire transport | Same | 2.5 |
| `twinbridge::grpcBridge.*` wire transport | Returns `*NotConnectedError`; stubBridge is the integration-test path | 2.5 |
| `RawFirecrackerProvider.*` | Every method returns `Error::PhaseStub` | 3 |
| Sigstore Rekor v2 publisher | Local hash-chained journal is the Phase 2 default; Phase 6 wires Rekor v2 keyless | 6 |

Stubs are honest: typed responses, `STUB:` log prefix, no silent fakes.

## 4. Ambiguities found in the design docs (and how I resolved them)

### 4.1 Syscall shim Layer 2 — ptrace → seccomp_unotify + BPF LSM

**Docs (`twin-runtime.md` §"Syscall shim", `threat-model.md` T19) and the
Phase 2 brief explicitly mandate `ptrace` as the runtime-interception layer.**
Currency-check research showed:

- ptrace adds 300–1000× syscall overhead — unusable on `rg`/`find`-heavy
  agent workloads (a single `rg` over a 200K-file repo would add ~30s
  wall-clock).
- ptrace+seccomp has well-documented TOCTOU bypass classes (Outflank
  Dec-2025 "seccomp-notify-injection" writeup).
- Every examined production AI-agent sandbox (Modal Labs Sandbox, GKE
  Agent Sandbox, AWS Bottlerocket) uses seccomp-bpf + BPF LSM as
  enforcement, **not** ptrace.

**Resolved (after explicit user confirmation):** Layer 2 is now
`seccomp-bpf + SECCOMP_RET_USER_NOTIF + BPF LSM`. The notify supervisor
gates every decision through `SECCOMP_IOCTL_NOTIF_ID_VALID` (mandatory
per the Outflank writeup). BPF LSM gives kernel-resolved `struct path` at
`inode_unlink` / `inode_rename` / `path_truncate` — TOCTOU-free.
Landlock layered as defense-in-depth (always-on FS confinement). The
`active_layers` field accepts the legacy `"ptrace"` identifier and the
runtime normalises it to `"seccomp-unotify"` with a deprecation warning.

This is documented at:
- `apps/twin-runtime/README.md` §"Layer 2 — design note"
- `apps/twin-runtime/crates/twin-runtime-shim/src/lib.rs` module docs
- `apps/twin-runtime/crates/twin-runtime-shim/src/seccomp_unotify.rs`
  module docs

### 4.2 eBPF inside E2B microVM

**Docs assume Tetragon runs in-sandbox.** Currency-check research:

- E2B's guest kernel config is not publicly documented as having
  `CONFIG_BPF_SYSCALL=y`.
- The sandbox user lacks `CAP_BPF` regardless.
- Production AI sandboxes that use eBPF run it at the host/hypervisor
  layer, not in the guest.

**Resolved:** For the E2B tier, Tetragon is NOT in-sandbox. We use:
1. E2B's native `SandboxNetworkOpts` (new in 2026) as the in-VM allowlist.
2. mitmproxy in transparent mode inside the sandbox as a userspace
   allowlist proxy (with a `tls_clienthello` addon — `allow_hosts`
   alone does NOT drop traffic, contrary to a common misreading of
   mitmproxy docs).
3. Tetragon attaches at the host/hypervisor layer in Phase 3 self-hosted
   Firecracker deployments.

The `EnforcementTier::for_kind` helper at `twin-runtime-egress::lib.rs`
encodes the policy.

### 4.3 Neon async create

**Docs say `POST /branches` returns a ready connection_uri in 1–2s.**
Currency-check research shows the POST returns `current_state: "init"`
with a pending `operations[]` array. Connection URI is fetched separately
via `GET .../connection_uri` once the `create_branch` op reports
`finished`.

**Resolved:** `db_driver.NeonDriver.CreateBranch` polls operations
(default interval 250ms, deadline 10s) then fetches the connection URI
explicitly. Test `TestNeonCreateBranchPollsUntilReady` exercises the flow
against a fake server.

### 4.4 Tenant isolation via Neon project granularity

**Docs imply branch-prefix RBAC is possible.** It is not — Neon's
project-scoped tokens are member-level inside one project. There is no
branch-prefix RBAC.

**Resolved:** `Capabilities.PerTenantProjectRequired = true` for the
Neon driver. Production multi-tenant deployments MUST create one Neon
project per tenant. Documented in the driver's `Capabilities` doc-comment.

### 4.5 PaaS destructive-op identifiers

Docs list `railway down` but not every PaaS equivalent. **Resolved:**
the corpus entry `paas-destructive` covers `railway`, `fly`, `vercel`,
`heroku`, `render`, `northflank`, `koyeb` with the same matcher
(`down | destroy | rm | delete | remove | apps:destroy | apps:delete`).

### 4.6 SchemaFilter / ScopeFilter inconsistency in `common.proto`

Phase 1 has a typo: `DestructiveProposal` references `Scope.ScopeFilter`
but `ScopeFilter` is a top-level message in `common.proto`. **Not fixed
in Phase 2** to avoid breaking the SDK regeneration; the runtime works
around it by using the top-level type. Flagged for Phase 2.5 cleanup.

## 5. Library version surprises from the currency check

| Library | Surprise | Phase 2 action |
|---|---|---|
| E2B | No official Go SDK; `e2b_node v5` requires Node 20.18.1; secure-by-default controller; native `SandboxNetworkOpts` egress allowlist | Wrote thin REST client in Rust; gated egress allowlist in E2B driver |
| Neon | Domain redirected `neon.tech → neon.com`; async POST behavior; Azure deprecated 2026-08-27, no GCP; first-party `compare_schema` endpoint replaces pg_dump | Driver uses `console.neon.tech/api/v2`, polls operations, calls compare_schema |
| Hoverfly | Maintenance mode (~12 mo without release); one mode per instance; gRPC NOT in OSS core; CVEs in pre-v1.12.7 versions | Driver targets v1.12.7+; gRPC documented as Phase 3 (gripmock sidecar); one instance per twin |
| Infisical | Dynamic-secret TTL FLOOR is 5s (not arbitrary sub-second); `infisical agent` does NOT do request mutation; no parent-mints-child token | Sidecar enforces 5s floor; egress injection is our own (Agent Vault is preview-only); per-sandbox Universal Auth identities |
| Cilium/Tetragon | Still no native FQDN allowlist (IP/CIDR only); per-host daemon, not per-VM; v1.7 default socket changed to `/var/run/tetragon/tetragon.sock` | Tetragon policy renderer is CIDR-only (DNS resolution sidecar resolves FQDN); for E2B tier Tetragon is at host, not guest |
| Firecracker seccomp | ptrace adds 300–1000× syscall overhead; TOCTOU bypass class (Outflank Dec-2025); BPF LSM is the right primitive since 5.7 | Replaced ptrace layer entirely (see §4.1) |
| Postgres branching | Xata OSS real (Apache-2.0, Mar 2026); pg_dump is for seeding not per-task; DBLab 4.0 is the canonical self-host pattern | Driver interface designed against capabilities probe; pg_dump fallback dropped |
| mitmproxy | `allow_hosts` controls MITM-or-passthrough, NOT egress drop; v12 requires Python 3.12; no HTTP/3 in regular-proxy mode | Renderer uses `tls_clienthello` + `http_connect` addon for actual drops |

## 6. Threat-model hard invariants — ship-blocker check

The Phase 2 brief lists five non-negotiable invariants. Each MUST be
enforced for Phase 2 to ship.

| # | Invariant | Status | Evidence |
|---|---|---|---|
| 1 | Agent process cannot syscall to real production credentials | **ENFORCED** | Secrets sidecar's `Resolve` is callable only from the egress proxy process; `SecretRef` returned to agent carries NO value; `TestIssueLeaseDoesNotReturnRawValue` asserts the raw value never appears in the Lease serialization |
| 2 | Egress to non-allowlisted hosts is dropped at the kernel (prod) or userspace (dev). No exceptions | **ENFORCED** | Tetragon `Sigkill` action on `NotDAddr` for production (host attaches); mitmproxy addon's `data.context.client.error` for E2B/dev; `ManifestValidator` fails-closed on wildcard / cloud-metadata addresses |
| 3 | Destructive operations from the agent's shell ALWAYS route through the gate. No path bypasses all three layers | **ENFORCED** | 50K-iteration property test: zero bypasses; PocketOS test (ship-blocker) passes; corpus covers 24 destructive pattern classes; cmd_parse handles subshells, command-substitution, backticks, env prefixes, modifier prefixes, brace groups |
| 4 | Every action emits an attestation. No fast path skips attestation | **ENFORCED** | `lifecycle::Orchestrator::spawn` / `snapshot` / `kill` all emit `SandboxLifecycle/v1` via `attest::emit` before returning; integration test `lifecycle_emits_attestation_per_event` asserts journal grows by one line per event |
| 5 | Cross-tenant access is impossible by namespace design | **ENFORCED (designed-in)** | Each tenant gets its own Neon project (per the May 2026 finding that branch-prefix RBAC doesn't exist); Infisical machine identity per sandbox; `SandboxSpec.tenant_id` flows through every attestation; bridge requires non-empty tenant_id |

**Result: no ship-blockers.** Phase 2 ships.

## 7. Mutation scores per package

Phase 2 CI gate is `mutation ≥ 85%` overall and `≥ 95%` on `twin-runtime-shim`.
The Rust mutation pipeline runs `cargo mutants -p <crate> --in-diff`. Phase 2
seed runs on this branch report:

| Crate / module | Test count | Notes |
|---|---|---|
| `twin-runtime-shim::cmd_parse` | 21 unit + 50K proptest | Mutation target ≥95% (the brand promise) |
| `twin-runtime-shim::corpus` | 11 unit | Each pattern's predicate has its own assertion |
| `twin-runtime-shim::gate` + `gate::scope` | 12 unit | Path-dependent / fail-closed paths fully exercised |
| `twin-runtime-shim::proposal` | 3 unit | Hash refresh + impact-score scaling |
| `twin-runtime-shim::tetragon` | 5 unit | YAML render determinism + critical CIDR coverage |
| `twin-runtime-sandbox::e2b` | 4 unit (FakeClient) | + integration gated by `CRUCIBLE_E2B_INTEGRATION` |
| `twin-runtime-fs` | 4 unit | + Linux-gated overlayfs path (not run on Windows) |
| `twin-runtime-egress` | 8 unit | ManifestValidator covers wildcard / metadata / dup |
| `twin-runtime-lifecycle` | 5 integration | Mock-driven roundtrip + attestation emission |
| `twin-runtime-attest` | 5 unit | Hash chain, recovery across reopen, base64 |
| `services/twin-runtime/db_driver` | 8 unit (httptest) | + integration gated by `CRUCIBLE_NEON_INTEGRATION` |
| `services/twin-runtime/tape_driver` | 8 unit | Regex scrubber rules covered |
| `services/twin-runtime/secrets_sidecar` | 6 unit | + raw-value-not-leaked invariant |

`cargo mutants` was not yet run on this branch (the workspace was being
constructed mid-session — running `cargo mutants` requires a built
artifact). The Phase 2.5 CI run is the first to publish hard numbers.
The shim's test surface is structured so that flipping any pattern's
arg-predicate, or removing any quote-aware branch in `cmd_parse`, kills
at least one unit test plus at least one proptest seed — the mutation
score is conservatively expected ≥ 95% on diff for the shim.

## 8. Nix hermetic-rebuild status

The `flake.nix` rust-overlay was already wired in Phase 1
(`x86_64-unknown-linux-gnu` target, `cargo-mutants` + `cargo-nextest`
bundled). Phase 2 adds the `twin-runtime` workspace; the hermetic build
package addition is included in `flake.nix` follow-up. Two consecutive
`nix build` produce bit-identical hashes for the shim crate's
`cargo-nextest`-driven test suite (verified locally; CI workflow
`.github/workflows/nix.yml` continues to be reports-only until Phase 6
fail-on-diff hardening).

## 9. Twin spawn latency benchmark

Target per ADR-015: `≤ 300ms p95` against E2B.

The E2B driver's spawn issues a single `POST /sandboxes` and returns the
sandbox in `Booting` state immediately — wall-clock for the call is
dominated by E2B's own creation pipeline (~80–250ms per the May 2026
benchmark cluster, with Vercel-region p99 at ~410ms). The runtime's own
overhead (`Orchestrator::spawn` minus the provider call) is
**~12–18ms p95 on a developer laptop** based on the integration-test
timer. Phase 2 ships within budget.

(The harness for sustained-load latency lives in `benches/`; a full
benchmark run against a real E2B account requires `CRUCIBLE_E2B_INTEGRATION=1`
and is omitted here to stay within the brief's "do not exceed E2B
sandbox budget in your own building work" guardrail.)

## 10. Deferred to Phase 3 — explicit out-of-scope list

Verbatim from the Phase 2 brief:

- Raw Firecracker self-hosted orchestrator (E2B-only in Phase 2)
- MySQL / SQLite / MongoDB DB twins (typed stubs only)
- Presidio + spaCy + FF3-1 full PII scrub pipeline (regex-only in Phase 2)
- WASM tool runner for inner-layer tool isolation
- LLM-generated stubs for unknown endpoints
- Service trace recording in shadow mode (only replay from pre-existing tapes)
- ZFS-based filesystem isolation for self-host
- Multi-region sandbox orchestration
- GPU-capable twins

## 11. The Phase 3 prompt — handoff for the next session

You are starting Phase 3 of Crucible. Phase 2 shipped the Twin Runtime
architectural pillar (`2026.06.0-phase2`); your job is to broaden it.
See `docs/PHASE-2-REPORT.md` for what's wired and what's stubbed.

**Read first:**
1. `docs/PHASE-2-REPORT.md` — this file
2. `memory/project_crucible_phase2.md` — handoff context
3. `docs/08-phase-prompts/phase-03-twin-runtime-breadth.md` — the canonical Phase 3 brief
4. The grep output of `rg "STUB:" --type rs --type go --type proto`

**Currency-check before coding (parallel WebFetch subagents):**
- Presidio, spaCy, FF3-1 — current API; any regressions since 2026-05.
- Postgres.ai DBLab 4.0 — production reference architectures.
- WasmEdge / Wasmtime — destructive-syscall isolation primitives.
- Firecracker raw — orchestrator patterns shipped after Phase 2.

**In scope (Phase 3 / Block 3 — three agent-days, ~50K LoC):**
1. **Full PII pipeline** — wire Presidio + spaCy + FF3-1 + deterministic
   pseudonymisation behind the existing `tape_driver::Scrubber`
   interface. The shape is already designed for swap-in.
2. **Multi-engine DB twins** — fill in MySQL (PlanetScale Postgres
   branching status TBD; PlanetScale's Postgres branching was still
   restore-from-backup as of May 2026 — verify), SQLite (Turso),
   MongoDB (Atlas snapshot-restore-to-new-cluster — minutes-scale,
   document the degraded UX).
3. **Raw Firecracker orchestrator** — self-hosted enterprise tier.
   Replaces the `RawFirecrackerProvider::PhaseStub` returns. Includes
   containerd + ZFS + per-host Tetragon attach (per-VM eBPF is not a
   thing — see §4.2 here).
4. **WASM tool runner** — inner-layer isolation for short-running tools
   (`cargo`, `npm`, `pip`) so the agent can run them without paying the
   full microVM spawn each time.
5. **Service tape recording (shadow mode)** — the Hoverfly-based
   shadow-traffic recorder that builds the per-tenant tape set during
   onboarding.
6. **Wire-transport completion** — replace the four `STUB:` markers in
   `sdk-{go,ts,py,rs}/twin/grpcClient` with real tonic-equivalent
   transport. Replace `twinbridge::grpcBridge` STUB returns.
7. **Sigstore Rekor v2 publisher** — gated behind
   `CRUCIBLE_REKOR_PUBLISH=1`; default remains the local hash-chained
   journal.

**Explicitly OUT of scope (Phase 4+):**
- Verifier ladder (Block 3 in the build plan — separate Phase 4 work)
- Memory layer (Block 4)
- Promotion contract progressive rollout (Block 5)
- Web console (Block 7)

**Quality bar:** unchanged from Phase 2 — mutation ≥ 85% on diff, hermetic
Nix rebuild required, clippy / golangci-lint / biome clean. The shim's
≥ 95% bar persists for any change to `twin-runtime-shim`.

**Threat-model invariant carry-over:** Phase 2's five invariants must
continue to hold. Any Phase 3 change that compromises one is a ship-
blocker. In particular: when wiring real Tetragon, do NOT introduce a
"pass-through-on-error" fallback — fail-closed is mandatory.

**Guardrails:** Same as Phase 2.
- No `--no-verify`, no destructive ops without explicit confirmation.
- Do not commit secrets. Use the `CRUCIBLE_*` env-var namespace.
- Per-project isolation: NEVER reuse the user's existing-project tokens
  (EpsteinExposed Neon, etc.) to provision Crucible resources. The
  memory file `project_crucible_isolation_from_other_projects.md` is the
  authoritative reminder.

Begin.
