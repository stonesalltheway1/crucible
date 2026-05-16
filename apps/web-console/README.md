# Crucible Web Console

The senior-engineer-facing UI for Crucible. Built with Next.js (App Router) +
shadcn/ui + Tremor.

**Brand voice:** evidence-driven engineering. Anti-vibe-coding aesthetic —
ink palette, monospace surfaces, sharp 2-px corners, no gradients. Per
ADR-001.

## Dev

```bash
pnpm install
pnpm dev
# → http://localhost:3000
```

The dev server hydrates from the control plane on `NEXT_PUBLIC_CRUCIBLE_API`
(default `http://localhost:8080`). If the control plane is offline, the pages
fall back to deterministic demo payloads from `src/lib/mocks.ts` so the
surface remains legible.

## Tests

```bash
pnpm test          # vitest component tests
pnpm e2e           # playwright golden-path flows
pnpm lint          # biome check
pnpm typecheck     # tsc --noEmit
```

## Routes

| Route | Purpose |
|---|---|
| `/` | Tenant overview — counters, in-flight, latest attestation |
| `/tasks` | Task timeline with cost / duration / verdict |
| `/tasks/[id]` | Task detail — plan, steps, verifier, attestation chain |
| `/tasks/[id]/approve` | **Plan-approval modal** — trust-narrative surface |
| `/promotions` | Approval inbox + recent history |
| `/promotions/[id]` | Canary status, Rego decision, approval flow |
| `/memory` | Convention browser + drift reviewer |
| `/memory/conventions/[id]` | Convention detail with source evidence |
| `/attestations` | Rekor UUID search + recent entries |
| `/attestations/[uuid]` | Predicate body + inclusion proof + verify |
| `/cost` | Per-task / per-repo / per-dev cost rollups |
| `/slo` | Public-style SLO status page |
| `/settings` | Budgets / models / classifier / Rego policy |
| `/webhooks` | Subscription management |

## Auth

- **SaaS:** Clerk (org-scoped JWT; `org_id` claim maps to tenant)
- **Enterprise:** WorkOS (SAML + OIDC; organization → tenant)
- **Self-host:** Authelia / Dex

## Hermetic build

`next.config.mjs` sets `experimental.deterministicBundling=true`. Combined
with the workspace's Nix flake, the production bundle is bit-identical
across builds; this is mandatory for the Tier-4 honest-CI verifier.

## What we deliberately do NOT do

- No tracking / analytics that send customer code or task content off-tenant.
- No customer prod credentials anywhere in the UI; we surface task events and
  attestation references only.
- No IDE chat panel replication. The IDE has chat; we don't compete.
