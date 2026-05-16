# v1 MVP

What ships in version 1 — and what explicitly does not. Calibrated to AI-agent throughput, not human-team cadence: see [build-plan-agent-days.md](build-plan-agent-days.md) for the day-by-day breakdown.

## v1 = "the thesis, working end-to-end"

A senior engineer should be able to:

1. Install Crucible (5 min).
2. Let the Cartographer run on a real codebase (≤ 30 min, automated).
3. Submit a real task — feature add, refactor, or bug fix.
4. Watch Crucible spin up a twin, execute, run cross-family verification, propose a PromotionBundle.
5. Approve the promotion.
6. See the change land via canary rollout with auto-rollback safety.
7. Verify every step cryptographically against Sigstore Rekor.

If all seven steps work end-to-end for the first three design-partner customers, v1 ships.

## In scope

### Twin Runtime (the trust foundation)

- Firecracker microVM via E2B (SaaS); raw Firecracker (self-host).
- Git worktree + overlayfs filesystem twin.
- Neon Postgres CoW branching (Postgres customers only).
- Hoverfly OSS service-tape replay with PII scrub (Presidio + spaCy + FF3-1).
- Infisical-issued ephemeral secrets.
- Cilium/Tetragon egress allowlist.
- Syscall shim with destructive-op typed proposals.

### Verifier Pipeline

- Tier 0 mutation testing (mutmut, stryker, cargo-mutants, go-mutesting, pitest, muter).
- Tier 1 property tests + fuzz (hypothesis, fast-check, proptest, rapid, jqwik).
- Tier 2 schemathesis contract testing + in-house DST harness.
- Tier 3 Dafny (others as deferred-load on first @critical hit).
- Tier 4 Nix hermetic rebuild + SLSA-L3 in-toto attestation via Sigstore Rekor v2.
- Cross-family executor/verifier routing (Opus 4.7 ↔ Gemini 3.1 Pro pairing default).

### Memory Layer

- Redis hot cache.
- pgvector episodic + semantic store.
- FalkorDB + Graphiti procedural graph.
- Background distillation worker (PR comments, post-mortems, ADRs).
- Mem0 hierarchical extraction algorithm.
- LLM-as-judge filter on writes (prompt-injection defense).
- OSS-corpus bootstrap (Tier A–D seed) — ~400 active rules on a fresh Next.js+FastAPI repo.
- Convention drift detection.

### Model Routing

- 5-tier router with Anthropic primary + Google verifier as default.
- Per-tenant model overrides.
- 5m + 1h prompt caching.
- Per-task budget enforcement.
- Cost telemetry.

### Promotion Contract

- Rego policy evaluation.
- Slack-button + web-UI human approval.
- KMS-signed credential leases (AWS KMS for SaaS; HSM for enterprise).
- Argo Rollouts canary integration.
- GrowthBook feature-flag rollback.
- In-toto attestation chain.

### Agent SDK

- `twin.*` API in Go, TS, Python, Rust.
- MCP server (`crucible-mcp`) for IDE integration.
- ACP support for Zed.
- gRPC + REST for direct integrations.
- Webhook events spec.

### UI Surfaces

- VS Code extension (plan approval, budget viewer, attestation chain explorer).
- JetBrains plugin (same affordances).
- Zed extension via ACP.
- `crucible` CLI (Go binary).
- Web console (Next.js + shadcn): task dashboard, cost dashboard, memory browser, attestation viewer, approval inbox, SLO dashboard.
- GitHub App for PR-comment invocation.
- Slack bot for approval routing.

### Pricing tiers (all five live at launch)

- Pro / Team / Outcome / BYOK / Enterprise (self-hosted).
- Stripe integration for billing.
- Usage-based metering with hard caps.

### Observability

- OpenTelemetry traces → Honeycomb (SaaS) / Tempo (self-host).
- Prometheus metrics.
- Loki logs (self-host) / Honeycomb events (SaaS).
- The four KPI dashboards (per-task economics, verifier health, safety/trust, memory/learning).
- Public SLO status page.

### Tier 4 self-verification

- Crucible's own monorepo is built via Nix flakes; releases are SLSA-L3 attested; customers can verify our releases.
- Tier 0–4 verification gates every Crucible PR. We eat our own dogfood.

### Documentation

- All docs in `docs/` (this directory) shipped with v1.
- Quickstart on docs site.
- Reference API docs auto-generated from protobuf.

## Explicitly out of scope for v1

- **Tab autocomplete.** That's Cursor's turf; we don't compete on it.
- **Vibe-coding chat-only builder.** Not our ICP.
- **Custom in-house Crucible model (Composer-2-style).** v2 if PMF + cost engineering demand it.
- **GPU sandbox / ML workload twins.** v2 if ICP shifts.
- **Multi-region twin orchestration.** Single-region per task.
- **End-to-end encrypted memory with customer key.** v2 enterprise feature.
- **Mobile app for approvals.** Web console + Slack cover it.
- **Plugin / skill marketplace.** v2.
- **Visual / Figma-aware UI generation.** v3.
- **Voice input.** Not v1.
- **Automatic Tier 3 across all languages.** Dafny is default; Lean / TLA+ / Kani / Z3 require manual annotation in v1.
- **HIPAA-eligible SaaS tier.** Self-hosted only for HIPAA in v1; SaaS BAA in v2.
- **FedRAMP Moderate certification.** Self-hosted air-gap supports the architecture; formal cert in v2.
- **Cassandra / DynamoDB / non-mainstream DB twin support.** Postgres + MySQL + SQLite + MongoDB only.
- **Multi-tenant procedural memory federation graduations.** Single-tenant only; cross-tenant federation requires ≥5 tenants and is unlikely to fire pre-PMF.
- **Customer-built verifier integrations.** Verifier extension API is internal-only; v2 opens it.

## v1 customer experience floor

If any of the following are *missing* at v1, the product fails its thesis:

| Capability | Required | Why |
|---|---|---|
| Twin runtime spawn < 300ms | yes | UX latency floor |
| Destructive-op gate fires on `rm -rf` / `DROP TABLE` / etc. | yes | The core safety story |
| Verifier rejects fake test-pass | yes | The core verification story |
| Plan UI shows $ + time estimate before execution | yes | The cost-transparency story |
| Sigstore Rekor attestation for every action | yes | The audit-trail story |
| Cross-family verifier pairing | yes | The ADR-002 architectural commitment |
| Procedural memory active rules visible in dashboard | yes | The compounding-moat story |
| Air-gap install works end-to-end | yes | The enterprise wedge |

If any of these slip, we don't ship v1. They are non-negotiable.

## v1 launch criteria

- 3 design-partner customers running Crucible on real production codebases for ≥ 30 days each.
- 100+ verified PRs landed across design partners.
- Zero security incidents at the boundaries documented in [threat-model.md](../01-architecture/threat-model.md).
- Cache hit rate ≥ 70% sustained.
- Median task cost ≤ $2.00 sustained.
- SLOs in [observability.md](../02-engineering/observability.md) met for the prior 30 days.
- All 15 ADRs accepted by the build team without unresolved objections.
- Tier 4 self-verification clean on every release in the prior 30 days.

## Open beta → GA

After the design-partner phase:

- **Open beta:** invite-only Pro tier. Cap at 200 users. Monitor cache + cost KPIs against the financial model.
- **GA:** open Pro + Team + Outcome tiers. Enterprise tier remains direct-sales.

## Risk register at v1

| Risk | Mitigation |
|---|---|
| Tape coverage falls below 80% in real customer workloads | Aggressive shadow-recording in onboarding; tier 2 fallback to live-passthrough |
| Verifier disagreement rate > 25% with humans | Shadow-mode tuning; rubric_score threshold adjustment |
| Median task cost exceeds $2.50 sustained | Routing-tier re-classification; cache investment |
| Customer's repo too large for Cartographer in v1 | Chunked processing; explicit "large-repo" mode |
| Frontier model API outage during launch | Multi-vendor routing; status banner |
| Customer-reported false promotion approval | RB-11 runbook; immediate incident response |
| Self-hosted install too complex for design partners | Concierge install with our SRE team |

See [v2-vision.md](v2-vision.md) for what comes after.
