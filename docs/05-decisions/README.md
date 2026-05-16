# Architectural Decision Records

A record of the load-bearing choices in Crucible's design, why each was picked, what alternatives were considered, and what consequences follow.

Format: lightweight ADR — Context, Decision, Status, Consequences, Alternatives.

## Index

| ID | Title | Status |
|---|---|---|
| [ADR-001](ADR-001-digital-twin-first.md) | Digital-twin-first execution as the primary trust mechanism | Accepted |
| [ADR-002](ADR-002-cross-family-verifier.md) | Mandatory cross-family verifier for task completion | Accepted |
| [ADR-003](ADR-003-procedural-memory-moat.md) | Per-tenant procedural memory as the primary moat | Accepted |
| [ADR-004](ADR-004-outcome-based-pricing.md) | "Verified PR" as the pricing unit | Accepted |
| [ADR-005](ADR-005-neon-db-branching.md) | Neon for Postgres copy-on-write branching | Accepted |
| [ADR-006](ADR-006-falkordb-over-alternatives.md) | FalkorDB for procedural memory graph backend | Accepted |
| [ADR-007](ADR-007-hoverfly-tape-replay.md) | Hoverfly OSS for service replay | Accepted |
| [ADR-008](ADR-008-tier3-annotation-default-off.md) | Tier 3 formal verification is auto-classified, not default-on | Accepted |
| [ADR-009](ADR-009-anti-loop-protocol.md) | Hard retry cap and bounded-budget enforcer | Accepted |
| [ADR-010](ADR-010-sigstore-rekor-attestations.md) | Sigstore Rekor v2 for transparency log | Accepted |
| [ADR-011](ADR-011-no-built-in-ide.md) | Crucible integrates with existing IDEs via MCP/ACP; no proprietary IDE | Accepted |
| [ADR-012](ADR-012-monorepo-structure.md) | Monorepo with per-component language choices | Accepted |
| [ADR-013](ADR-013-nix-for-tier4-builds.md) | Nix flakes as default for hermetic builds | Accepted |
| [ADR-014](ADR-014-infisical-over-vault.md) | Infisical as default secrets vault | Accepted |
| [ADR-015](ADR-015-firecracker-via-e2b.md) | E2B (Firecracker) as default sandbox in SaaS tier | Accepted |
