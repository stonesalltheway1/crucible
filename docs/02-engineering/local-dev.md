# Local Dev

How to run the Crucible stack locally. For background, read [system-overview.md](../01-architecture/system-overview.md) first.

> **Phase 2** added the Twin Runtime (Rust workspace at `apps/twin-runtime/`) plus three Go services under `services/twin-runtime/`. The Phase 1 control plane is unchanged. Phase 2 setup notes are in §"Twin Runtime (Phase 2)" below.

## Prerequisites

- **Nix** (2.34+). Install via `curl -L https://nixos.org/nix/install | sh`. Flakes must be enabled:
  ```bash
  mkdir -p ~/.config/nix && echo 'experimental-features = nix-command flakes' >> ~/.config/nix/nix.conf
  ```
- An **`ANTHROPIC_API_KEY`**. Without it the control plane runs in heuristic + fallback-plan mode (useful for offline dev, but not the full path).
- Optional: `GOOGLE_API_KEY` (or `GEMINI_API_KEY`) to wire the verifier vendor; `OPENAI_API_KEY` for alternate Tier 1/2 routing.

## Dev shell

```bash
cd "E:\AI Coding Agent"     # Windows path; on Linux/macOS just clone the repo
nix develop                  # all-language shell (Go + Node + Python + Rust + buf + cosign + opa)

# Or per-language:
nix develop .#go-only
nix develop .#node-only
nix develop .#python-only
nix develop .#rust-only
```

On Windows, use **WSL2** for the Nix shell — Nix on native Windows is nascent (see [ADR-013](../05-decisions/ADR-013-nix-for-tier4-builds.md) §Open issues).

## Running the control plane

```bash
export ANTHROPIC_API_KEY=sk-ant-...
nix build .#control-plane
./result/bin/crucible-control-plane
```

You should see:

```
{"level":"INFO","msg":"attestation wired","signer":"...","journal":"~/.crucible/attestations/journal.jsonl"}
{"level":"INFO","msg":"LLM vendors wired","vendors":["anthropic"]}
{"level":"INFO","msg":"control-plane listening","addr":":8080","version":"2026.06.0-phase1"}
```

Smoke-test:

```bash
nix build .#cli
./result/bin/crucible health
./result/bin/crucible task new --description "Add a Stripe refund webhook handler" --repo github.com/acme/payments
./result/bin/crucible task list
./result/bin/crucible plan show <task_id>
./result/bin/crucible plan approve <task_id>
./result/bin/crucible budget show <task_id>
```

## Without Nix (best-effort, non-hermetic)

```bash
# Go 1.23, Node 22, Python 3.12, Rust stable required.
cd apps/control-plane && go build ./...
cd ../cli && go build ./...
./apps/control-plane/control-plane &
./apps/cli/crucible health
```

You'll fail the SLSA-L3 hermetic-rebuild check on this path; use Nix for any artifact you intend to publish.

## Tests

```bash
# Per-module
cd apps/control-plane && go test -short ./...
cd libs/attestation && go test ./...
cd libs/policy && go test ./...
cd libs/sdk-go && go test ./...

# Python SDK
cd libs/sdk-py && pip install -e .[dev] && pytest

# TS SDK
cd libs/sdk-ts && pnpm install && pnpm test

# Rust SDK
cd libs/sdk-rs && cargo test --all-targets

# The real-Haiku-4.5 integration test only runs with the env var set:
ANTHROPIC_API_KEY=sk-ant-... go test -run TestIntegration_RealHaiku4_5 -v ./apps/control-plane/internal/api
```

The budget-enforcer property test (`TestProperty_NeverExceedsCap`) runs 50 seeds × 8 goroutines × 500 ops and is the strongest correctness guarantee in Phase 1. It asserts the ADR-009 invariant: once a cap is breached, the enforcer is frozen and no further mutation succeeds.

## Environment variables the control plane reads

| Var                          | Default                                          | Purpose                                              |
|------------------------------|--------------------------------------------------|------------------------------------------------------|
| `ANTHROPIC_API_KEY`          | unset                                            | Wires the Anthropic vendor (Tier 0/1/2 default)      |
| `GOOGLE_API_KEY`             | unset                                            | Wires Gemini (verifier-default vendor)               |
| `OPENAI_API_KEY`             | unset                                            | Wires OpenAI (alternate Tier 1/2)                    |
| `CRUCIBLE_LISTEN_ADDR`       | `:8080`                                          | HTTP bind address                                    |
| `CRUCIBLE_DEFAULT_TENANT`    | `single-tenant`                                  | Tenant ID when callers omit `tenant_id`              |
| `CRUCIBLE_KEY_DIR`           | `~/.crucible/dev-keys/`                          | Local Ed25519 keypair for attestation signing        |
| `CRUCIBLE_JOURNAL_PATH`      | `~/.crucible/attestations/journal.jsonl`         | Hash-chained attestation journal                     |
| `CRUCIBLE_COSTLOG_DIR`       | `~/.crucible/costlog/`                           | Per-task cost JSONL                                  |
| `CRUCIBLE_WEBHOOK_URL`       | unset                                            | If set, every event POSTs here                       |
| `CRUCIBLE_REKOR_PUBLISH`     | `0`                                              | Gates the Phase-2 real Rekor v2 publisher            |

## Twin Runtime (Phase 2)

The Twin Runtime is a Rust workspace at `apps/twin-runtime/`. On Linux it links against `libbpf` / `libcap` / `landlock`; on macOS / Windows it compiles with the Linux-only features cfg-gated to no-op stubs (you can run the cross-platform layer-1 shim tests, the E2B driver tests against the fake HTTP server, and the lifecycle orchestrator tests — but not the actual seccomp / BPF-LSM / Tetragon enforcement).

### Build + test (any host)

```bash
cd apps/twin-runtime
cargo build --workspace
cargo test --workspace

# The brand-promise tests (50K-iteration property test, PocketOS scenarios):
cargo test --release -p twin-runtime-shim --test property_50k
cargo test --release -p twin-runtime-shim --test pocket_os_scenario

# Mutation gate (CI runs on diff):
cargo mutants -p twin-runtime-shim --in-diff
```

### Build + test (Linux host with full enforcement layers)

Use a recent kernel (≥ 5.7 for BPF LSM; ≥ 5.11 for `SECCOMP_RET_USER_NOTIF`) and these capabilities for full enforcement:

```bash
# Inside WSL2 / Linux VM:
sudo apt install libbpf-dev libcap-dev
cargo build --workspace --release
```

### Run the runtime locally

```bash
# Minimal: no E2B / Neon / Infisical wired — every operation returns a
# typed `STUB:` error pointing at the env var.
cargo run -p twin-runtime-server

# Full: with real E2B + Neon + Infisical creds (NEVER reuse other-project
# keys per the isolation rule).
export CRUCIBLE_E2B_API_KEY=e2b_...
export CRUCIBLE_NEON_API_KEY=napi_...
export CRUCIBLE_NEON_PROJECT_ID=proj_crucible_dedicated
export CRUCIBLE_INFISICAL_API_URL=https://app.infisical.com/api
export CRUCIBLE_INFISICAL_CLIENT_ID=...
export CRUCIBLE_INFISICAL_CLIENT_SECRET=...
export CRUCIBLE_TWIN_RUNTIME_ADDR=127.0.0.1:7444    # control-plane bridge target

cargo run --release -p twin-runtime-server
```

### Twin Runtime env vars

| Variable | Default | Purpose |
|---|---|---|
| `CRUCIBLE_TWIN_LISTEN` | `127.0.0.1:7444` | gRPC bind address |
| `CRUCIBLE_TWIN_JOURNAL` | `~/.crucible/twin-runtime-journal.jsonl` | Hash-chained attestation journal |
| `CRUCIBLE_TWIN_RUNTIME_ADDR` | unset | Control-plane bridge target |
| `CRUCIBLE_E2B_API_KEY` | unset | E2B SaaS sandbox provider |
| `CRUCIBLE_E2B_BASE_URL` | `https://api.e2b.app` | E2B endpoint (used by tests) |
| `CRUCIBLE_E2B_INTEGRATION` | `0` | Gate the real-E2B integration test |
| `CRUCIBLE_NEON_API_KEY` | unset | Neon REST token (project-scoped) |
| `CRUCIBLE_NEON_BASE_URL` | `https://console.neon.tech/api/v2` | Override (used by tests) |
| `CRUCIBLE_NEON_PROJECT_ID` | unset | Crucible-dedicated Neon project id |
| `CRUCIBLE_NEON_INTEGRATION` | `0` | Gate the real-Neon integration test |
| `CRUCIBLE_INFISICAL_API_URL` | `https://app.infisical.com/api` | Infisical base URL |
| `CRUCIBLE_INFISICAL_CLIENT_ID` | unset | Universal Auth client id |
| `CRUCIBLE_INFISICAL_CLIENT_SECRET` | unset | Universal Auth client secret |
| `CRUCIBLE_INFISICAL_PROJECT_ID` | unset | Crucible-dedicated Infisical project |
| `CRUCIBLE_LOG` | `info,twin_runtime=debug` | tracing filter |

**Isolation note**: never reuse the user's existing-project tokens (e.g. EpsteinExposed Neon, personal Vercel/Cloudflare/GitHub) to provision Crucible resources. Always create a Crucible-dedicated project per service. The bridge fails-closed when project ids are mis-namespaced.

## Twin Runtime breadth (Phase 3)

### PII scrubber (Python)

```bash
cd services/twin-runtime/tape_driver/scrubber

# Dev install (regex fallback path only — no Presidio models)
python -m venv .venv && source .venv/bin/activate
pip install -e .

# Full Presidio + spaCy install
pip install -e ".[hipaa]"
python -m spacy download en_core_web_lg

# Run the unit tests
pytest tests/test_pipeline.py -v

# Run the recall corpus (≥ 70% fallback, ≥ 99% Presidio)
pytest tests/test_recall_corpus.py -v

# Start the HTTP service
export CRUCIBLE_SCRUBBER_TOKEN=$(openssl rand -hex 16)
export CRUCIBLE_SCRUBBER_FF3_KEY=hex:$(openssl rand -hex 32)
crucible-scrubber
# → POST http://127.0.0.1:9100/scrub
```

The Go tape driver wires the scrubber via `CRUCIBLE_SCRUBBER_URL`;
absent that env var, the Phase 2 regex baseline is used.

### Per-engine DB drivers (Go)

Each engine has its own `CRUCIBLE_<ENGINE>_*` env-var namespace; all
drivers fall back to a typed stub when env is unset (matches Phase 2
Neon behaviour).

| Engine | Env vars |
|---|---|
| MySQL (PlanetScale) | `CRUCIBLE_PLANETSCALE_TOKEN_ID`, `CRUCIBLE_PLANETSCALE_TOKEN`, `CRUCIBLE_PLANETSCALE_ORG`, `CRUCIBLE_PLANETSCALE_DB` |
| libSQL (Turso) | `CRUCIBLE_TURSO_TOKEN`, `CRUCIBLE_TURSO_ORG`, `CRUCIBLE_TURSO_GROUP` |
| Mongo (Atlas) | `CRUCIBLE_MONGO_ATLAS_PUBLIC_KEY`, `CRUCIBLE_MONGO_ATLAS_PRIVATE_KEY`, `CRUCIBLE_MONGO_ATLAS_GROUP_ID`, `CRUCIBLE_MONGO_ATLAS_CLUSTER` |
| ClickHouse | `CRUCIBLE_CLICKHOUSE_URL`, `CRUCIBLE_CLICKHOUSE_USER`, `CRUCIBLE_CLICKHOUSE_PASSWORD`, `CRUCIBLE_CLICKHOUSE_SOURCE_DB` |
| S3 / MinIO | `CRUCIBLE_S3_ENDPOINT`, `CRUCIBLE_S3_ACCESS_KEY`, `CRUCIBLE_S3_SECRET_KEY`, `CRUCIBLE_S3_SOURCE_BUCKET`, `CRUCIBLE_S3_MIRROR_PREFIX` |
| Redis | `CRUCIBLE_REDIS_BIND_HOST`, `CRUCIBLE_REDIS_BASE_PORT` |

### Raw Firecracker self-host (Linux only, opt-in)

```bash
# Cross-platform check (no real spawns — typed PhaseStub when called)
cd apps/twin-runtime-self-host
cargo test

# Production-class Linux build
cargo build --release --features linux-firecracker
# Requires: Linux 6.x, KVM, ZFS pool mounted at /var/lib/crucible/zfs,
# firecracker binary at /usr/local/bin/firecracker, cgroups v2,
# CAP_NET_ADMIN, CAP_SYS_ADMIN for ZFS clone.
```

### WASM tool runner

Cross-platform (Wasmtime works on Linux / macOS / Windows):

```bash
cd apps/twin-runtime
cargo test -p twin-runtime-wasm
# 10 000-iteration containment proptest (release-only to match epoch granularity):
cargo test --release -p twin-runtime-wasm --test containment
```

### Shadow recorder

```bash
cd services/twin-runtime/tape_driver/shadow_recorder
go test ./...
```

The recorder accepts `POST /ingest/envoy` from an Envoy access-log
sink. Production deployments wire it behind a sidecar that taps
production / staging traffic through the customer's existing edge.

## What's stubbed in Phase 3

- Real Firecracker invocations (gated behind `linux-firecracker` Cargo feature)
- Wasmtime Component Model (Phase 3 ships core modules only)
- OpenAPI 3.1 advanced schemas (`anyOf` / `oneOf` / `allOf` / recursive `$ref`)
- Tape promotion UI (CANDIDATE entries persist; operator dashboard is Phase 6)
- HSM-backed FF3-1 keys via Vault Transform (env-supplied master key in Phase 3)
- SigV4 for the S3 driver (SigV2 ships; MinIO accepts both)
- Real-API integration tests against PlanetScale / Turso / Mongo / Atlas / ClickHouse / S3 (env-gated)

See [PHASE-3-REPORT.md](../PHASE-3-REPORT.md) for the full stub
inventory and Phase 4 hand-off.

## Verifier Pipeline (Phase 4)

Phase 4 ships the verifier daemon at `apps/verifier/` and the
per-language runners at `verifiers/{python,typescript,rust,go,java,swift}/`.

### Running the verifier daemon

```bash
# Start the verifier on its own port
export CRUCIBLE_VERIFIER_LISTEN_ADDR=:9080
export CRUCIBLE_VERIFIER_KEY_DIR=$HOME/.crucible/verifier-keys
export CRUCIBLE_VERIFIER_JOURNAL_PATH=$HOME/.crucible/verifier-journal.log
export GOOGLE_API_KEY=...        # default verifier model (Gemini 3.1 Pro)
# Optional: CRUCIBLE_VERIFIER_HEURISTIC=1 forces the no-LLM heuristic
#           (useful for hermetic CI runs)

nix build .#crucible-verifier
./result/bin/crucible-verifier
```

You should see:

```
{"level":"INFO","msg":"crucible-verifier listening","addr":":9080","version":"2026.06.0-phase4",
 "judge_vendor":"crucible-heuristic","judge_model":"rubric-heuristic-v1"}
```

### Wiring the control plane

```bash
export CRUCIBLE_VERIFIER_ADDR=http://127.0.0.1:9080
./result/bin/crucible-control-plane
# → logs: "verifier bridge wired"
# → /v1/tasks/{id}/verify now dispatches to the verifier daemon
```

### Per-language verifier runners

The daemon spawns one runner per (language, tier) tuple. Each runner
is a separate binary in its native language so it can drive the
per-language tooling natively. CLIs:

| Language | Binary | Install |
|---|---|---|
| Python | `crucible-verify-python` | `pip install -e verifiers/python` |
| TypeScript | `crucible-verify-typescript` | `pnpm -C verifiers/typescript install && pnpm -C verifiers/typescript build` |
| Rust | `crucible-verify-rust` | `cargo install --path verifiers/rust` |
| Go | `crucible-verify-go` | `go install ./verifiers/go/cmd/crucible-verify-go` |
| Java | `crucible-verify-java.sh` | `chmod +x verifiers/java/crucible-verify-java.sh` (stub — Phase 9+) |
| Swift | `crucible-verify-swift.sh` | `chmod +x verifiers/swift/crucible-verify-swift.sh` (stub — Phase 9+) |

Each binary reads a VerificationRequest JSON from stdin and writes
`===CRUCIBLE-TESTREPORT===\n` + TestReport JSON to stdout. Logs go to
stderr.

### Pinning the per-language tools

| Tool | Pin | Why |
|---|---|---|
| `mutmut` | `~=3.5` | 3.x is current (no 4.x exists in May 2026); config-only paths_to_mutate |
| `hypothesis` | `~=6.152` | settings(max_examples=10_000); ~weekly cadence stable |
| `schemathesis` | `~=4.18` | v4 is a full rewrite — pin minor, block 5.x |
| `atheris` | `==3.0.0` | single 3.x release; Python 3.11–3.13 only |
| `fast-check` | `4.7.0` | numRuns: 10_000; avoid 3.x EOL |
| `@fast-check/vitest` | `0.4.1` | Vitest 4 support |
| `@stryker-mutator/core` | `9.6.1` | mutation-testing-elements schema v3.7.x |
| `@jazzer.js/core` | `2.1.0` (vendor-pin) | jsfuzz is unmaintained — use Jazzer.js |
| `proptest` | `=1.11` | PROPTEST_CASES=10000 |
| `cargo-mutants` | `27.0.0` | `--in-diff <path>.diff --json` |
| `Kani` | `0.67.0` | `--harness <regex>`; skip propproof — use bolero |
| `cargo-fuzz` | `0.13.1` | nightly-only |
| `pgregory.net/rapid` | `v1.3.0` | `-rapid.checks=10000` |
| `avito-tech/go-mutesting` | `v2.3.1` | no native JSON — parse text; realistic gate ~60% |
| `Dafny` | `4.11.0` | DafnyPro POPL 2026 paper-only — orchestration recipe in `internal/tier3/` |
| `diffoscope` | `318` | content-level diff; pair with `nix store diff-closures` |
| `actions/attest-build-provenance` | `v4.1.0` (by SHA) | or prefer `actions/attest@v2` |
| `in-toto/attestation/go/v1` | `v1.2.0` | canonical Go bindings |

### Cross-family pairing

The daemon enforces ADR-002 at the rubric edge AND at the dispatcher
edge AND at the HTTP API edge. The default pairing is:

| Executor | Verifier |
|---|---|
| `claude-opus-4-7` (Anthropic) | `gemini-3.1-pro` (Google) |
| `gpt-5.5` (OpenAI) | `claude-opus-4-7` (Anthropic) |
| `Llama-4-Maverick` (local) | `DeepSeek-V4-Pro` (local) |

The verifier sandbox is ALWAYS a different sandbox from the executor's;
the process pool refuses to spawn into the executor's sandbox ID even
if requested.

### Dafny install (Tier 3)

```bash
# Linux
curl -L https://github.com/dafny-lang/dafny/releases/download/v4.11.0/dafny-4.11.0-x64-ubuntu-20.04.zip -o dafny.zip
unzip dafny.zip && sudo mv dafny /opt/ && sudo ln -s /opt/dafny/dafny /usr/local/bin/dafny

# macOS
brew install dafny  # may install 4.11.x
```

Without Dafny on PATH, the Tier 3 adapter returns
`Verdict=tool_unavailable` rather than fail-open.

### Antithesis credentials (optional)

If your tenant licenses Antithesis SaaS, set:

```bash
export ANTITHESIS_TENANT_ID=...
export ANTITHESIS_API_KEY=...
```

Without these, the daemon uses the in-house DST harness (TigerBeetle
VOPR / FoundationDB Flow-style; Polar Signals' `frostdb/dst` is the
canonical Go OSS reference for the pattern).

## Memory Layer (Phase 5)

Phase 5 ships:

- `services/memory-router/` — Go daemon (HTTP `:8090`), the hot-path retrieval layer.
- `services/distiller/` — Python background worker that mines PR comments / ADRs / incidents / Slack / runbooks into Conventions.
- `services/memory-router/cartographer/` — installer-side Python tool that runs once per repo at onboarding.
- `infra/oss-corpus-bootstrap/` — offline pipeline that emits `services/memory-router/global_defaults/*.json` per-stack bundles.
- `infra/databases/` — Postgres + pgvector + FalkorDB + Redis schemas with per-tenant RLS.

### Backends (per ADR-006 / ADR-005)

```bash
# Postgres 16 + pgvector 0.9 (DiskANN for the > 10M tier)
docker run -d --name crucible-pg -p 5432:5432 \
  -e POSTGRES_PASSWORD=local \
  pgvector/pgvector:pg16

# FalkorDB (Redis module)
docker run -d --name crucible-falkordb -p 6379:6379 \
  falkordb/falkordb:latest

# Apply Phase 5 migrations (direct mode, local dev)
export CRUCIBLE_MEMORY_PG_DSN='postgres://postgres:local@localhost/postgres'
./infra/databases/migrations/run.sh --direct
```

### Memory-router daemon

```bash
cd services/memory-router

# CI / hermetic mode — uses the in-memory fakes for every backend.
# Required when no Postgres / FalkorDB / Redis is running.
CRUCIBLE_MEMORY_ROUTER_STUB=1 go run ./cmd/memory-router \
    --global-defaults global_defaults \
    --addr :8090

# Tests
go test ./...

# 50K cross-tenant adversarial isolation gate (slow; skip with -short)
go test ./test/isolation/...

# p95 latency benchmark
go test -run TestP95Latency ./test/bench/...
```

### Distiller

```bash
cd services/distiller

# Editable install (one-time)
pip install -e ../../libs/memory-spec/py
pip install -e .

# The catch-rate CI gate — runs the adversarial corpus, exits 1 on miss.
crucible-distiller selfcheck
# {
#   "adversarial_caught_combined": 26,
#   "adversarial_catch_rate_combined": 1.0,
#   "honest_falsepos_det": 0,
#   "honest_falsepos_llm": 0
# }

# Tests
pytest -q
```

### Cartographer

```bash
cd services/memory-router/cartographer
pip install -e ../../../libs/memory-spec/py
pip install -e ../../../services/distiller   # for adapters + extractor
pip install -e .

# Run against any local repo
crucible-cartographer scan \
    --repo acme/payments \
    --path /path/to/local/checkout \
    --tenant-id ten_local \
    --out result.json
```

### Per-stack OSS-corpus bundles

```bash
cd infra/oss-corpus-bootstrap
pip install -e ../../libs/memory-spec/py
pip install -e .

# Build all 12 bundles
crucible-oss-bootstrap run --output ../../services/memory-router/global_defaults/

# Per-stack rule counts
crucible-oss-bootstrap stats
```

### Verifier wiring

When `CRUCIBLE_MEMORY_ROUTER_ADDR` is set, the Phase-4 verifier daemon
(`apps/verifier/`) auto-wires the `MemoryComplianceFeaturizer`:

```bash
CRUCIBLE_MEMORY_ROUTER_ADDR=http://127.0.0.1:8090 \
CRUCIBLE_VERIFIER_HEURISTIC=1 \
go run ./apps/verifier/cmd/crucible-verifier
```

Without the env var the bridge is a no-op and the rubric's
`trust_signal_alignment` criterion runs unchanged (Phase 4 behaviour).

### Env vars summary

| Var | Default | Purpose |
|---|---|---|
| `CRUCIBLE_MEMORY_ROUTER_ADDR` | unset | Wires the verifier MemoryComplianceFeaturizer + downstream agent SDK |
| `CRUCIBLE_MEMORY_ROUTER_STUB` | `0` | Forces in-memory fakes for every backend (CI hermetic mode) |
| `CRUCIBLE_MEMORY_PG_DSN` | unset | Postgres DSN for direct migration mode |
| `CRUCIBLE_FALKOR_ADDR` | `127.0.0.1:6379` | FalkorDB host:port |
| `CRUCIBLE_REDIS_ADDR` | `127.0.0.1:6379` | Redis host:port (separate instance from FalkorDB in production) |

## What's stubbed in Phase 5

- Production model-routed LLM judge (the `FakeJudge` + cheap deterministic pre-filter ship by default; Haiku 4.5 wires into the same `LLMClient` interface)
- Live GitHub-App PR-comment webhook ingestion (offline corpus path ships; live webhook lands in Phase 7)
- Kafka + SQS queue consumer for the distiller daemon (sync `process_one` path ships)
- Federation graduation engine (data model + `Detector.Scan` candidate emission ship; the actual graduation write fires in v2 Phase 10)
- pgvector HNSW → DiskANN flip job (DiskANN index file ships; auto-flip when a tenant crosses 10M vectors is Phase 7)
- Tree-sitter symbol-density classifier in the cartographer (filesystem walk + lint-config detection ships)
- Tier-B (top-200-repos × 12 stacks) corpus mining (per-stack seed scaffolding ships at 12 rules × 12 stacks = 144 active defaults; the ~280-rule-per-stack expansion is Phase 7)

See [PHASE-5-REPORT.md](../PHASE-5-REPORT.md) for the full inventory.

## What's stubbed in Phase 4

- Real Sigstore Rekor v2 publish (local hash-chained journal remains the default)
- Lean 4 + LeanCopilot Tier-3 adapter (typed-error stub; v2 Phase 9)
- TLA+ + Apalache Tier-3 adapter (typed-error stub; v2 Phase 9)
- Z3 / CVC5 direct dispatch (typed-error stub; v2 Phase 9)
- Java + Swift per-language runners (interface-ready stubs; no design partner in v1)
- Antithesis SaaS wiring (flagged; in-house DST is the OSS-tier default)
- DafnyPro paper-only — orchestration loop ships; LLM-assisted assertion
  generator (Laurel-style) wires via the `LaurelAugmenter` interface in
  Phase 5+
- The `crucible-verifier` cmd binary's `buildJudge` returns the heuristic
  by default; the production model-router adapter wires in Phase 5

See [PHASE-4-REPORT.md](../PHASE-4-REPORT.md) for the full stub
inventory.

## What's stubbed in Phase 2

- libbpf-rs LSM hook attachment (Landlock fallback is active)
- seccomp-unotify supervisor tokio loop
- Tetragon policy submission to `/var/run/tetragon/tetragon.sock`
- gRPC wire transport in the four SDK twin/ clients (StubClient is feature-complete for unit tests)
- `twinbridge::grpcBridge` real wire transport
- Sigstore Rekor v2 publisher (local hash-chained journal remains default)

See [PHASE-2-REPORT.md](../PHASE-2-REPORT.md) for the full stub inventory.

## What's stubbed in Phase 1

- Verifier Pipeline (Tier 0–4 ladder)
- Memory Layer (Redis / pgvector / FalkorDB / Graphiti / distiller)
- Promotion Contract (Argo Rollouts, KMS lease, Slack approvals)
- Real Sigstore Rekor v2 publish (the local journal is the default — Rekor v2 had not GA'd as of May 2026)
- Web console, IDE plugins, GitHub App, Slack bot

See [PHASE-1-REPORT.md](../PHASE-1-REPORT.md) for the full stub inventory.

## Promotion Contract + Provenance (Phase 6)

Phase 6 ships the bridge from twin to real: the **promotion gate**
(`apps/promotion-gate/`), the **attestation relay**
(`apps/attestation-relay/`, Rust), the **slack-bot**
(`apps/slack-bot/`), and the Argo Rollouts + GrowthBook templates under
`infra/argo-rollouts/` and `infra/feature-flag-rollouts/`. The
control-plane gains a `promotionbridge` package and a
`POST /v1/tasks/{id}/promote` endpoint.

### Running the full Phase-6 stack in dev

Three daemons run in parallel:

```bash
# Terminal 1 — attestation relay (Rust). Offline mode uses the
# local hash-chained journal and a local Ed25519 signer; perfect for
# fully-air-gapped dev.
export CRUCIBLE_RELAY_OFFLINE=1
export CRUCIBLE_JOURNAL_PATH=~/.crucible/relay/journal.jsonl
cargo run --release -p crucible-attestation-relay
# Listens on :9120 by default.

# Terminal 2 — promotion gate (Go). Wired to the relay above.
export CRUCIBLE_RELAY_ADDR=http://127.0.0.1:9120
export CRUCIBLE_KMS_PROVIDER=dev
export CRUCIBLE_KMS_DEV_DIR=~/.crucible/kms-dev
nix build .#promotion-gate
./result/bin/crucible-promotion-gate
# Listens on :9180 by default.

# Terminal 3 — slack-bot (Go).
export CRUCIBLE_GATE_ADDR=http://127.0.0.1:9180
export CRUCIBLE_RELAY_ADDR=http://127.0.0.1:9120
export SLACK_BOT_TOKEN=xoxb-test
export SLACK_SIGNING_SECRET=test
nix build .#slack-bot
./result/bin/crucible-slack-bot
# Listens on :9280 by default.

# Terminal 4 — control plane with promotion bridge.
export CRUCIBLE_PROMOTION_GATE_ADDR=http://127.0.0.1:9180
nix build .#control-plane
./result/bin/crucible-control-plane
# /healthz now reports stub_promotion=false.
```

### Production KMS providers

```bash
# AWS KMS (asymmetric SignatureAlgorithm = ECDSA_SHA_256)
export CRUCIBLE_KMS_PROVIDER=aws
export CRUCIBLE_KMS_KEY_ARN=arn:aws:kms:us-east-1:123:key/abcd
# AWS credentials discovered via the standard chain.

# GCP Cloud HSM
export CRUCIBLE_KMS_PROVIDER=gcp
export CRUCIBLE_KMS_KEY_ARN=projects/.../cryptoKeys/.../cryptoKeyVersions/...

# YubiHSM
export CRUCIBLE_KMS_PROVIDER=yubi
export CRUCIBLE_KMS_KEY_ARN=arn:yubihsm:crucible-prod
```

Phase 6 ships the closure-based scaffolds for AWS / GCP / YubiHSM; the
SDK wiring lives in the cmd entrypoint where the closures are constructed
from `aws-sdk-go-v2 / cloud-kms / pkcs11` clients.

### Self-hosted Sigstore Rekor

```bash
export CRUCIBLE_REKOR_URL=https://rekor.acme.internal
export CRUCIBLE_REKOR_SELF_HOSTED=1
export CRUCIBLE_REKOR_ROOT_CA=/etc/crucible/rekor-root.pem
export CRUCIBLE_FULCIO_URL=https://fulcio.acme.internal
export CRUCIBLE_OIDC_ISSUER=https://accounts.acme.internal
```

The relay's `/v1/predicates` endpoint lists the 13 Crucible types + SLSA
Provenance v1 it can ingest.

### Slack bot (ngrok)

In dev, expose `:9280` so Slack's interactive callbacks can reach you:

```bash
ngrok http 9280
# Set the Slack app's Interactive Components callback URL to
#   https://<your-ngrok>.ngrok-free.app/slack/interactive
```

Without Slack credentials (the default `SLACK_BOT_TOKEN=xoxb-test`), the
bot stubs out chat.postMessage with a deterministic `dev.<promotion-id>`
message ts — the gate's webhook flow still exercises end-to-end.

### Argo Rollouts

```bash
kubectl apply -f infra/argo-rollouts/templates/analysis/
kubectl apply -f infra/argo-rollouts/templates/rollout/
```

For non-K8s customers, the feature-flag-only path uses the templates in
`infra/feature-flag-rollouts/` against a GrowthBook instance.

## What's stubbed in Phase 6

- **Public Sigstore Rekor v2 production wiring**: the HTTP client and
  publish path are implemented; the real Sigstore Rekor v2 GA still
  involves vendor-side rough edges (per ADR-010 §Open issues). Phase 6
  ships offline + self-hosted Rekor as first-class; SaaS-tier customers
  using public Sigstore retain the local-journal fallback (RB-05).
- **AWS / GCP / YubiHSM SDK plumbing**: the lease shape + signer
  interface are production-ready; the actual aws-sdk-go-v2 / cloud-kms
  / pkcs11 client construction lives in the cmd entrypoint and is
  feature-flagged behind `CRUCIBLE_KMS_PROVIDER` — production deployments
  swap the dev closures for SDK-backed ones.
- **Multi-region KMS replication**: explicitly deferred to v2 hardening.
- **Customer-controlled signing keys for the highest FedRAMP tier**:
  YubiHSM scaffold is present; full FedRAMP-track key attestation chain
  is v2 hardening.
- **Plugin marketplace for Rego policies**: deferred to v2 Phase 12.

See [PHASE-6-REPORT.md](../PHASE-6-REPORT.md) for the full inventory.

## Phase 7 — Agent-Facing UX

The surfaces customers actually touch. Web console, IDE plugins (VS Code +
JetBrains + Zed via ACP), expanded CLI, GitHub App, fleshed-out Slack bot.

### Web console dev server

```bash
cd apps/web-console
pnpm install                              # one-time
pnpm dev                                  # → http://localhost:3000
```

The server hydrates from the control plane on
`NEXT_PUBLIC_CRUCIBLE_API` (default `http://localhost:8080`). If the
control plane is offline, the pages fall back to deterministic demo
payloads in `src/lib/mocks.ts` so the surface stays legible.

Brand-voice note: the Tailwind theme in `tailwind.config.ts` overrides
shadcn's default rounded-corners-blue-gradient register. Ink palette,
monospace surfaces, 2px corners, no glow. Do not soften.

### VS Code extension sideload

```bash
cd apps/ide-plugins/vscode
pnpm install
pnpm compile
# F5 in VS Code with the extension folder open → launches an Extension
# Development Host with the local build loaded.
```

To package and install:

```bash
pnpm package                              # → crucible-vscode-2026.06.0-phase7.vsix
code --install-extension crucible-vscode-2026.06.0-phase7.vsix
```

### JetBrains plugin sideload

```bash
cd apps/ide-plugins/jetbrains
./gradlew buildPlugin
# Plugins → ⚙ → Install Plugin from Disk → build/distributions/*.zip
```

For dev iteration, `./gradlew runIde` launches a sandbox IntelliJ with
the plugin loaded.

### Zed extension sideload

```bash
cd apps/ide-plugins/zed
cargo build --release --target wasm32-wasi
# Zed → ⌘+, → Extensions → "Install from local directory"
```

### GitHub App via ngrok

```bash
ngrok http 9320
# Paste the ngrok URL into the GitHub App settings' webhook URL.

export GITHUB_WEBHOOK_SECRET=$(openssl rand -hex 32)
export CRUCIBLE_API_ADDR=http://localhost:8080
go run apps/github-app/cmd/crucible-github-app
# {"version":"2026.06.0-phase7","msg":"github-app listening","addr":":9320"}
```

In any GitHub PR or issue comment thread, post
`/crucible add idempotency key to /webhooks/stripe/refund`. The app
posts an acknowledgement with a link to the plan-approval UI.

### Slack bot — slash command in dev mode

```bash
ngrok http 9280
# Slack App settings:
#   Slash commands: /crucible, /crucible-status → <ngrok>/slack/slash
#   Interactivity: <ngrok>/slack/interactive

export CRUCIBLE_SLACK_BOT_TOKEN=xoxb-...
export CRUCIBLE_SLACK_SIGNING_SECRET=...
crucible-slack-bot
# Slack: /crucible add idempotency key to /webhooks/stripe/refund
# → ephemeral ack; DM follows with the plan-approval link.
```

### CLI smoke test

```bash
cd apps/cli
go build -o crucible ./cmd
./crucible version                        # 2026.06.0-phase7
./crucible promote list
./crucible memory drift-review
./crucible attestation chain task_01HZAB_d9
./crucible verify-release 2026.06.0-phase7
./crucible calibrate --dry-run --samples 500
```

### Web console tests

```bash
pnpm test                                 # vitest component tests
pnpm e2e                                  # playwright golden-path
pnpm lint && pnpm typecheck
```

### Hermetic Nix builds

The Next.js config sets `experimental.deterministicBundling=true`. With
the workspace's `flake.nix`, `nix build .#web-console` produces a
bit-identical bundle across builds. This is mandatory for the Tier-4
honest-CI verifier.

## What's stubbed in Phase 7

- **Marketplace publish CI**: VS Code Marketplace + JetBrains Marketplace
  publish targets are scaffolded; Phase 8 hooks them into the SaaS
  release pipeline.
- **Lighthouse score check in CI**: wired in Phase 8 alongside the SaaS
  deploy URL.
- **GitHub App JWT-minting in `postIssueComment`**: omitted from the Phase
  7 commit; production deploys plumb the PEM via `--private-key`. The
  public-facing API contract is what carries the Phase-7 narrative.
- **Mintlify docs site bootstrap**: Phase 8.
- **Public-marketing website**: separate repo per repo-structure.md.

See [PHASE-7-REPORT.md](../PHASE-7-REPORT.md) for the full inventory.
