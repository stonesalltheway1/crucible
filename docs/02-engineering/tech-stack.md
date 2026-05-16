# Tech Stack

The full inventory of technologies Crucible composes. Decisions are pre-made; the ADRs in [05-decisions/](../05-decisions/) document the reasoning.

## Compute & isolation

| Layer | Default (hosted) | Self-hosted | Solo-founder tier |
|---|---|---|---|
| Microvm sandbox | E2B (Firecracker) | Firecracker + containerd | Daytona / Fly Machines |
| Filesystem isolation | overlayfs + git worktrees | overlayfs + ZFS + git worktrees | overlayfs alone |
| Egress enforcement | Cilium + Tetragon | Cilium + Tetragon | mitmproxy allowlist |
| Container runtime | crun (Firecracker-friendly) | crun | Docker / podman |
| Orchestration | Kubernetes (EKS / GKE / AKS) | Kubernetes | docker-compose |

## Data layer

| Component | Pick | Notes |
|---|---|---|
| Twin Postgres | Neon (CoW branching) | $0.002/branch-hr; instant branch via `POST /branches` |
| Twin MySQL | PlanetScale | Branching mature for MySQL |
| Twin SQLite/libSQL | Turso | Instant per-DB branch |
| Twin MongoDB | Atlas snapshot-restore-to-new-cluster | Slower (minutes), acceptable |
| Twin Redis/KV | Fresh `redis-server` inside sandbox | Stateless |
| Twin S3 | MinIO inside sandbox + rclone mirror | |
| Production-side DB (Crucible's own) | Postgres 16 (managed: Neon for SaaS, RDS for self-host) | Hot path for memory + attestations |
| Vector store | pgvector (default), Qdrant (greenfield), Turbopuffer (scale) | Per-tenant isolation |
| Graph store | FalkorDB (default), Neo4j (alt) | Avoid KuzuDB — archived Oct 2025 |
| Hot cache | Redis 7+ | Per-tenant namespaces |
| Queue (distiller / async tasks) | Kafka (high-volume) or AWS SQS (small-team) | |
| Object storage | S3-compatible (AWS S3, R2, MinIO) | |

## Service replay & mocking

| Function | Pick |
|---|---|
| HTTP/gRPC capture-replay | Hoverfly OSS |
| Mock-only (when no recording) | WireMock (JVM stacks), Mockoon (broad) |
| LLM-generated stubs | Microcks AI Copilot pattern |
| Contract testing | Pact (for explicit contracts) |
| OpenAPI mock generation | Stoplight Prism |
| Schema-grounded fakes | json-schema-faker, Faker.js |

## PII scrubbing

| Function | Pick |
|---|---|
| Named-entity recognition | Microsoft Presidio Analyzer + Anonymizer |
| NLP backbone | spaCy 3.x (Presidio default) |
| Format-preserving encryption | FF3-1 via mysto/python-fpe or Vault transform |
| Synthetic data augmentation | Gretel, MOSTLY AI, or SDV (open source) |
| Audit | append-only per-tape scrub log |

## Secrets & signing

| Function | Pick |
|---|---|
| Secrets vault (hosted) | Infisical Cloud |
| Secrets vault (self-host) | Infisical OSS / HashiCorp Vault Community |
| Production-promotion signing | AWS KMS / GCP Cloud HSM / YubiHSM (per deployment) |
| Code signing (sigstore) | Cosign + Sigstore keyless OIDC |
| Transparency log | Sigstore Rekor v2 (public for SaaS; self-hosted for enterprise) |
| Build provenance | in-toto attestations + SLSA-L3 |
| Build attestation tools | GitHub `actions/attest-build-provenance`, Witness, Tekton Chains |

## Verifier toolchain (per language)

### Python
- `hypothesis` 6.152+, `schemathesis` (APIs), `mutmut` 4.x, `atheris`
- `ruff` (lint), `mypy --strict` (types)

### JS / TS
- `fast-check` + `@fast-check/vitest` / `@fast-check/jest`
- `stryker-js` (mutation), `jsfuzz` (fuzz)
- `biome` (lint + format)

### Rust
- `proptest`, `quickcheck`, `cargo-mutants`, `kani`, `cargo-fuzz`, `cargo-afl`
- `rustfmt`, `clippy`

### Go
- `pgregory.net/rapid` (PBT), native `testing.F` (fuzz), `go-mutesting`
- `gofmt`, `golangci-lint`

### Java / Kotlin
- `jqwik` (PBT), `pitest` (mutation), JQF (coverage-guided PBT)

### Swift
- `swift-testing` (Xcode 16+), `muter` (mutation)

### C / C++
- `theft` (PBT), `libFuzzer`, AFL++

### Tier 3 (formal verification)
- **Dafny + DafnyPro** (POPL 2026): general business logic, money paths
- **Lean 4 + mathlib + LeanCopilot**: crypto, math-heavy
- **TLA+ + Apalache**: distributed invariants
- **Kani**: Rust `unsafe` + FFI
- **Z3 v4.15+ / CVC5 v1.2+**: SMT direct queries

### Tier 4 (honest CI)
- **Nix flakes** (default reproducible builds)
- **Bazel** (alternative for Java/Kotlin/C++ shops)
- **Sigstore Cosign** (signing)
- **in-toto** (attestation format)
- **SLSA provenance generator** (`slsa-framework/slsa-github-generator`)

## Memory layer

| Function | Pick |
|---|---|
| Hot cache | Redis 7+ |
| Episodic + semantic store | pgvector (default) / Qdrant / Turbopuffer |
| Procedural graph backend | FalkorDB |
| Procedural graph abstraction | Graphiti (Zep's OSS) atop FalkorDB |
| Extraction algorithm | Mem0's hierarchical extraction (Apache-2.0) |
| Schema-constrained decoding | AdaKGC SDD pattern |
| Embedding model | OpenAI `text-embedding-3-large` (default) / Cohere v3 (EU) / open-weights option |

## LLM routing

| Tier | Model | API |
|---|---|---|
| 0 | `claude-haiku-4-5` | Anthropic Messages API |
| 1 | `claude-sonnet-4-6` | Anthropic Messages API |
| 2 | `claude-opus-4-7` | Anthropic Messages API |
| 2 (alt, terminal-heavy) | `gpt-5.5`, `gpt-5.3-codex` | OpenAI Responses API |
| 2 (alt, algorithmic) | `gemini-3.1-pro` | Gen AI Direct (Vertex) |
| 3 (verifier, default pairing) | cross-family of executor | — |
| 4 (local) | Llama 4 Scout / DeepSeek V4-Pro / Qwen3-Coder-Plus | vLLM / sglang / Ollama |

## Observability

| Function | Pick |
|---|---|
| Tracing | OpenTelemetry → Honeycomb (SaaS) / Tempo (self-host) |
| Metrics | Prometheus + Grafana |
| Logs | Loki (self-host) / Honeycomb structured events (SaaS) |
| Errors | Sentry |
| Cost telemetry | Custom (OTel spans → ClickHouse) |
| Uptime | Crucible's own SLO dashboards backed by Prometheus AnalysisTemplate (eating our dogfood) |

## Progressive delivery

| Function | Pick |
|---|---|
| Canary controller (K8s) | Argo Rollouts |
| Canary controller (service mesh) | Flagger (Linkerd / Istio) |
| Feature flags | GrowthBook (OSS, self-host friendly) |
| Shadow traffic | Hoverfly tape replay against new version |
| Traffic mirroring | Argo Rollouts + service mesh |
| Rollback | GrowthBook flag flip (millisecond) |

## Front-end stack

- **Framework:** Next.js (App Router, RSC default, `use client` boundaries explicit)
- **Component lib:** shadcn/ui + Radix primitives
- **Styling:** Tailwind CSS
- **Form validation:** zod + react-hook-form
- **Charts:** Tremor (dashboards) / Recharts (custom)
- **Realtime:** Server-Sent Events for plan + verifier progress; WebSocket only when bi-directional needed
- **Auth:** Clerk (SaaS) / WorkOS (enterprise SSO + SAML) / Authelia (self-host)
- **Hosting:** Vercel (SaaS) / customer-supplied (self-host)

## Backend services

- **API framework (Go):** `connect-go` (gRPC + HTTP from same handler)
- **API framework (Python, distiller):** FastAPI
- **DB access (Go):** sqlc + pgx
- **DB access (Python):** SQLAlchemy 2.x (typed) + Alembic
- **Migrations:** sqlc generate + Atlas (declarative migrations)
- **Background jobs:** asynq (Go), Celery (Python)

## CLI

- **Language:** Go
- **Framework:** Cobra + Viper
- **TUI:** Bubble Tea (when interactive flows needed)
- **Distribution:** GitHub Releases + Homebrew + Scoop + apt/yum repos

## Why these picks (one-line each)

- **Nix > Docker for reproducible builds.** Bit-identical artifacts are mandatory for Tier 4.
- **Neon > self-hosted Postgres branching.** CoW at storage layer in 1–2s is irreplaceable.
- **FalkorDB > Neo4j.** Lower latency, simpler ops, source-available. Neo4j ecosystem advantage doesn't justify the cost premium for our use.
- **Hoverfly > WireMock.** Hoverfly's capture-replay is first-class; WireMock is mock-first with bolt-on capture.
- **Infisical > Vault.** Modern DX, OSS self-host, real dynamic secrets without enterprise-tier upcharges.
- **Sigstore > custom signing.** Public transparency log, ecosystem momentum, OIDC keyless approach. Customer's compliance team already knows the name.
- **Argo Rollouts > Flagger.** Argo's analysis ecosystem is richer; Flagger is fine if you're already in Linkerd.
- **GrowthBook > LaunchDarkly.** OSS, self-host friendly, no per-MAU tax.
- **Anthropic + Google as primary vendors.** Cross-family verifier requires both; OpenAI is in the routing table but not load-bearing.

## What we explicitly do NOT use

- **Pinecone** (vector DB lock-in, pricey at scale)
- **Milvus** (operational complexity not worth it under 100M vectors)
- **KuzuDB** (archived October 2025)
- **HashiCorp Vault HCP Dedicated for v1** (EOL plans for HCP Vault Secrets July 2026 created uncertainty; Infisical is the safer modern choice)
- **AWS QLDB** (sunset; no clean replacement narrative)
- **LocalStack OSS** (archived March 2026; Pro-only is auth-required)
- **jsverify, gopter** (superseded by fast-check and rapid respectively)
- **Java EE / Spring on Crucible's own backend** (Go and Rust are better fits for our service shape)

## Upgrade path

We track frontier libs weekly. Major version bumps (e.g., `hypothesis 7.0`, `fast-check 4.0`) go through the standard PR + Tier 0–4 verification flow. Customer-impacting changes get a 30-day deprecation window communicated via changelog and console banner.

Model routing tracks vendor pricing and capability changes monthly. The May 2026 reference pricing in [01-architecture/model-routing.md](../01-architecture/model-routing.md) is hardcoded for v1; v2 introduces a model-price oracle.
