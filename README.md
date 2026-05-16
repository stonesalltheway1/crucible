# Crucible

> The AI engineer that tests every change in a digital twin before touching your real code.

Crucible is a coding agent positioned against Cursor/Windsurf/Devin/Antigravity on the **trust and verifiability** axis. Every change runs in a faithful ephemeral mirror of the user's project — twin filesystem, twin database, twin services, twin secrets — and is independently verified by a different-family model before promotion to real systems.

## Status

**v1 LAUNCH-READY — Crucible 2026.06.0 (2026-05-15).** Phase 8 closes
the v1 build. The product is onboardable, installable, billable,
observable, audit-able, and self-verifying. Every v1 launch criterion
in [`docs/V1-LAUNCH-CHECKLIST.md`](docs/V1-LAUNCH-CHECKLIST.md) is ✓
or has a documented operator gate (Stripe-live-keys flip, DNS,
Marketplace publish, Mintlify deploy — none are engineering blockers).

Phase 8 ships:

- **`apps/cartographer/`** — the day-1 customer experience (tree-sitter
  walker + symbol indexer + lint-config parser + AGENTS.md / ADR
  reader + GitHub PR-comment scanner + Haiku-4.5 distillation +
  agreement scoring + OSS-defaults loader + inferred-AGENTS.md
  generator + first-task suggestions). Time-to-first-result ≤ 30 min
  on 50K-LoC repos.
- **`services/shadow-recorder/`** — standalone tape-population service
  with fail-closed scrubbing, per-endpoint coverage, monthly
  re-record schedule.
- **`apps/control-plane/internal/{onboarding,billing}/`** — 4-stage
  onboarding (GitHub App + Slack OAuth + 4 source-data adapters +
  weekly digest + day-1/2/5/30 CS hooks) + Stripe billing across the
  five published tiers with refund-on-reject.
- **`infra/helm/crucible/`** — production Helm umbrella + 14 sub-charts
  + per-cloud variants + air-gap defaults.
- **`infra/air-gap-bundle/`** — signed FedRAMP / defense installer;
  end-to-end install ≤ 1 hour from a clean cluster.
- **`infra/observability/`** — Prometheus + Grafana + Loki + Tempo +
  the four KPI dashboards + RB-01..RB-15 alert rules + public SLO
  status page.
- **`.github/workflows/release.yml`** — six-stage release pipeline with
  reproducible-build comparison + **Crucible-self-verification gate**
  (we eat our own dogfood demonstrably).
- **`cth/`** — Crucible Test Harness, 25 cases across 6 categories
  with a Go grading harness; per-category gating wired into CI.
- **`docs/mint.json` + quickstarts + V1-LAUNCH-CHECKLIST + Phase 8
  report** — public docs site at `docs.crucible.dev`.

Earlier phases (the foundations Phase 8 ships on top of):

- **Phase 7 — Agent-Facing UX** ([`docs/PHASE-7-REPORT.md`](docs/PHASE-7-REPORT.md))
- **Phase 6 — Promotion Contract + Provenance** ([`docs/PHASE-6-REPORT.md`](docs/PHASE-6-REPORT.md))
- **Phase 5 — Memory Layer** ([`docs/PHASE-5-REPORT.md`](docs/PHASE-5-REPORT.md))
- **Phase 4 — Verifier Pipeline** ([`docs/PHASE-4-REPORT.md`](docs/PHASE-4-REPORT.md))
- **Phase 3 — Twin Runtime breadth** ([`docs/PHASE-3-REPORT.md`](docs/PHASE-3-REPORT.md))
- **Phase 2 — Twin Runtime** ([`docs/PHASE-2-REPORT.md`](docs/PHASE-2-REPORT.md))
- **Phase 1 — Agent Control Plane** ([`docs/PHASE-1-REPORT.md`](docs/PHASE-1-REPORT.md))

The full design (architecture, ADRs, SDK reference, threat model, etc.) lives under [`docs/`](docs/). Start at [`docs/README.md`](docs/README.md). For the customer-facing reproducibility contract, see [`docs/V1-LAUNCH-CHECKLIST.md`](docs/V1-LAUNCH-CHECKLIST.md).

## Quick start (developers)

```bash
# Enter a hermetic dev shell with Go, Node, Python, Rust, buf, cosign, opa
nix develop

# Or per-language
nix develop .#go-only
nix develop .#node-only

# Build
nix build .#control-plane
nix build .#cli

# Run the control plane locally (requires ANTHROPIC_API_KEY env var)
./result/bin/crucible-control-plane

# In another shell — submit a task
./result/bin/crucible task new --description "Add a Stripe webhook handler"
```

Without Nix (best-effort, non-hermetic):

```bash
cd apps/control-plane && go build ./...
cd apps/cli && go build ./...
```

See [`docs/02-engineering/local-dev.md`](docs/02-engineering/local-dev.md) for the full local-dev guide.

## Layout

```
apps/                 user-facing surfaces (control-plane, cli, twin-runtime, verifier, etc.)
services/             supporting microservices (attestation-relay, tape-scrubber, etc.)
libs/                 shared libraries (sdk-{go,ts,py,rs}, attestation, twin-spec, policy, ...)
verifiers/            per-language verifier integrations
infra/                IaC for hosted + self-hosted deployments
examples/             end-to-end demos / sample integrations
docs/                 the design — start here
scripts/              build / release / dev helpers
.github/              CI workflows + issue templates
flake.nix             Nix-flake build entry point
```

Per [`docs/02-engineering/repo-structure.md`](docs/02-engineering/repo-structure.md).

## License

Apache-2.0 for OSS components. See [`LICENSE`](LICENSE) and individual component `LICENSE` files.

## Contributing

This repo follows Conventional Commits, mutation-tested unit tests on the diff (≥85%), and hermetic Nix builds. See [`docs/02-engineering/testing-strategy.md`](docs/02-engineering/testing-strategy.md). Every PR is verified by Crucible's own Tier 0–4 ladder once Phase 2 lands — see [`docs/PHASE-1-REPORT.md`](docs/PHASE-1-REPORT.md) for what's wired today.
