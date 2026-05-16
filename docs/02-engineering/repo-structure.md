# Repo Structure

The Crucible codebase is a monorepo. One repo, one build graph, one CI pipeline. Components communicate via gRPC over a service mesh in production, but the source-of-truth is co-located.

## Top-level layout

```
crucible/
├── apps/                          # User-facing surfaces
│   ├── control-plane/             # Orchestrator API + Plan Builder + Budget Enforcer
│   ├── twin-runtime/              # Twin sandbox manager + lifecycle
│   ├── verifier/                  # Verifier daemon (Tier 0–4 ladder)
│   ├── distiller/                 # Background memory distillation worker
│   ├── promotion-gate/            # Rego policy engine + KMS signing pipeline
│   ├── web-console/               # Team dashboard (Next.js / shadcn)
│   ├── cli/                       # `crucible` CLI
│   └── ide-plugins/               # VS Code, JetBrains, Zed (ACP) bridges
│
├── services/                      # Supporting microservices
│   ├── attestation-relay/         # In-toto attestation publisher → Sigstore Rekor
│   ├── tape-scrubber/             # PII scrub + record/replay pipeline (Hoverfly wrapper)
│   ├── memory-router/             # Multi-signal retrieval router
│   └── cost-meter/                # Per-task cost telemetry + cap enforcement
│
├── libs/                          # Shared internal libraries
│   ├── sdk-go/                    # `twin.*` SDK in Go (agent process side)
│   ├── sdk-ts/                    # SDK in TypeScript
│   ├── sdk-py/                    # SDK in Python
│   ├── sdk-rs/                    # SDK in Rust
│   ├── attestation/               # in-toto / Sigstore signing helpers
│   ├── twin-spec/                 # Type definitions for Plan, Bundle, Verdict, etc.
│   ├── memory-spec/               # Convention data model, retrieval query types
│   ├── tape-format/               # Hoverfly tape format + scrub manifest
│   ├── policy/                    # Rego policy bundles + helpers
│   └── model-routing/             # Multi-vendor LLM router (Anthropic/Google/OAI/etc.)
│
├── verifiers/                     # Per-language verifier integrations
│   ├── python/                    # hypothesis, schemathesis, mutmut, atheris, dafnypro
│   ├── typescript/                # fast-check, stryker, jsfuzz
│   ├── rust/                      # proptest, cargo-mutants, kani, cargo-fuzz
│   ├── go/                        # rapid, go-mutesting, native fuzz
│   ├── java/                      # jqwik, pitest, jqf
│   ├── swift/                     # swift-testing, muter
│   ├── tier3-dafny/               # DafnyPro adapter
│   ├── tier3-lean/                # LeanCopilot adapter
│   ├── tier3-tla/                 # Apalache adapter
│   └── tier4-honest-ci/           # Nix/Bazel hermetic rebuild + SLSA-L3 attestation
│
├── infra/                         # IaC for hosted + self-hosted deployments
│   ├── terraform/                 # AWS/GCP/Azure base infra
│   ├── helm/                      # Kubernetes charts (control plane + workers)
│   ├── argo-rollouts/             # AnalysisTemplate library
│   ├── air-gap-bundle/            # Offline installer for enterprise tier
│   └── observability/             # Honeycomb/Tempo/Prometheus configs
│
├── examples/                      # End-to-end demos / sample integrations
│   ├── nextjs-stripe-demo/        # Reference customer workload
│   ├── django-payments/
│   ├── rust-axum-api/
│   └── go-grpc-service/
│
├── docs/                          # ← you are here
│
├── scripts/                       # Build / release / dev helpers
├── .github/                       # CI workflows + issue templates
└── BUILD                          # Bazel root (or `flake.nix` for Nix-based builds)
```

## Why monorepo

- **One build graph.** The verifier integrations depend on the SDK types depend on the twin-spec depend on attestation. Cross-cutting refactors need a single PR.
- **One CI pipeline.** Reproducible builds for the entire system. Tier 4 honest CI applies to *our own* releases.
- **One test surface.** Integration tests across components are first-class.
- **Versioning is monorepo-wide.** Components don't have independent semver; the system has one version.

## Build system

**Default: Nix flakes.** Hermetic, reproducible, multi-language. Required for Tier 4 self-verification (we eat our own dogfood).

**Alternative: Bazel.** If the team prefers, Bazel works — but Nix is the default because (a) it's the easier reproducibility story for SLSA-L3, (b) it integrates cleanly with the air-gapped enterprise installer, (c) Nix flakes are familiar to the senior-engineer ICP.

## Language-per-component decisions

Each app/service picks the right language for its job. No "one language for everything" mandate. Specific choices:

| Component | Language | Rationale |
|---|---|---|
| control-plane | Go | Single-binary deploy, strong gRPC story, predictable GC |
| twin-runtime | Rust | Firecracker integration, syscall shim performance, safety |
| verifier daemon | Go | Orchestrates per-language verifier processes; gRPC fan-out |
| distiller worker | Python | Best LLM SDK ecosystem; not perf-critical |
| promotion-gate | Go | OPA/Rego embedding via go-rego; KMS clients in Go |
| web-console | TypeScript (Next.js + shadcn) | Standard 2026 React stack |
| cli | Go | Cross-platform single binary |
| IDE plugins | TypeScript | VS Code / Zed ACP / JetBrains all support TS |
| attestation-relay | Rust | Sigstore client mature; perf-critical at scale |
| tape-scrubber | Python | Presidio is Python-native; Hoverfly wrapping via subprocess |
| memory-router | Go | Hot-path retrieval; latency-sensitive |
| cost-meter | Go | Hot-path telemetry; latency-sensitive |

SDKs: one per supported agent host language (Go, TS, Python, Rust). All generated from the same gRPC/protobuf schema in `libs/twin-spec/`.

## Inter-service communication

- **gRPC** for internal service-to-service. Schemas in `libs/twin-spec/` and `libs/memory-spec/`.
- **HTTP/JSON** for the public REST API (control-plane → external) and the IDE plugins.
- **MCP** for IDE integration (the IDE plugin acts as an MCP host; control-plane is the MCP server).
- **ACP** (Agent Client Protocol) for cross-IDE portability.
- **Webhooks** for outbound events (Slack approvals, GitHub PR comments, etc.).

## Dependency policy

- **Frontier libraries only.** We use the actively-maintained, latest-version, popular libraries. No vendored legacy code.
- **Open-source dependencies must be license-clean** for our redistribution context: MIT, Apache-2.0, BSD-3-Clause, MPL-2.0, or LGPL with dynamic linking. **No GPL, AGPL, SSPL, BUSL in core libs.** Self-hosted enterprise installer can include GPL components if they're sandboxed in user-runtime.
- **Vendored dependencies tracked in `THIRD_PARTY.md`** with version, license, source URL.
- **Renovate auto-PR** keeps deps fresh; security updates auto-merge after Tier 0 verification (yes, we use Crucible to maintain Crucible).

## Versioning

- **Calendar versioning** for releases: `YYYY.MM.PATCH` (e.g., `2026.06.0`). Plays well with quarterly release cadence and customer procurement.
- **Semver internal** for SDKs and protocol schemas. Breaking changes to the protocol bump major; we maintain backward compatibility for one major version.
- **API stability tier per component:**
  - **Stable** — public REST API, SDK types, MCP tool definitions. Breaking changes require major version + 90-day deprecation window.
  - **Beta** — promotion-gate Rego policy schema, memory-spec types. Breaking changes documented in CHANGELOG; minor-version cadence.
  - **Internal** — everything else; refactor at will.

## Code style

- **One linter per language**, configured at repo root and enforced in CI.
  - Python: `ruff` + `mypy --strict`
  - TS: `biome` (replacing prettier+eslint; fewer configs)
  - Go: `gofmt` + `golangci-lint` with our preset
  - Rust: `rustfmt` + `clippy -W clippy::pedantic` (selective opts)
- **One import order** per language; auto-enforced.
- **No comments unless they explain WHY.** Per [02-engineering/testing-strategy.md](testing-strategy.md), tests + types document WHAT.
- **One naming convention** per language (idiomatic). No team-wide enforcement of e.g. snake_case across all languages.

## Documentation in-repo

- `README.md` at every component root explains what it does in 3 paragraphs.
- `CHANGELOG.md` per component, auto-generated from conventional commits.
- `THREAT_MODEL.md` per security-critical component (twin-runtime, promotion-gate, attestation-relay).
- API docs auto-generated from protobuf schemas and exported types; published to `docs.crucible.dev`.

## Testing layout

Mirrors the source tree:

```
apps/control-plane/
  src/...
  test/
    unit/       # Per-file tests, mutation-tested
    integration/# Cross-service, against ephemeral Neon branch
    e2e/        # Full task lifecycle, against test tenant
```

Verifier integrations are tested against the Crucible Test Harness (CTH) — a curated set of test repos with known-good and known-bad PRs. See [testing-strategy.md](testing-strategy.md).

## What lives outside this repo

- **Customer test harnesses / fixtures** — those live in customer repos.
- **Public website + marketing content** — separate repo.
- **The OSS-released verifier harness** — split out to its own repo (`crucible/verifier`) under Apache-2.0 once stable; here it lives as the source of truth.
- **The OSS-released tape-scrub pipeline** — same pattern: developed here, mirrored to its own OSS repo.
- **Crucible Skills marketplace** — separate repo + registry (v2).

## CI pipeline

- **PR checks:** lint, type-check, unit tests (mutation-tested), integration tests against ephemeral infra, Tier 4 hermetic build verification.
- **Main-branch merges:** all of the above, plus full e2e on the Crucible Test Harness, plus chaos tests on the twin-runtime, plus a Crucible-self-verification run (our own agent verifies the PR we're about to merge).
- **Release:** Nix-bundled artifacts published to the public registry; SLSA-L3 attestations published to Rekor; Helm charts and air-gap bundle built and signed.

See [04-operations/self-hosted-install.md](../04-operations/self-hosted-install.md) for what gets shipped to customers.
