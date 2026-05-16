# ADR-013: Nix flakes as default for hermetic builds

**Status:** Accepted  
**Date:** 2026-05-15

## Context

Tier 4 of the verifier ladder requires hermetic builds and SLSA-L3 attestations — bit-identical artifacts independently rebuildable. Three mainstream options for reproducible polyglot builds:

- **Nix (flakes)** — pure-functional package definitions, content-addressed store, hermetic by construction.
- **Bazel** — Google-pedigree build graph, hermetic when configured strictly, language-rules ecosystem.
- **Custom Docker + lock files** — pragmatic, fragile, not actually reproducible.

## Decision

**Nix flakes** is the default for Crucible's own builds and for customer Tier 4 verification.

Bazel is supported as an alternative for customers whose internal build system is Bazel-native (so they don't have to convert).

Custom Docker is explicitly not Tier-4-compliant.

## Consequences

### Positive

- **Hermetic by construction.** Nix's content-addressing means the build inputs uniquely determine the output. SLSA-L3 attestations are clean.
- **Polyglot-friendly.** Single config covers Go, Rust, Python, TypeScript, system deps. Bazel requires per-language rulesets that lag behind.
- **Air-gap-friendly.** Nix store is offline-friendly; the air-gap installer bundles needed paths.
- **Reproducibility verification is built-in.** `nix flake check` + `nix store verify` give bit-identical guarantees.

### Negative

- **Learning curve.** Nix is famously esoteric. Mitigation: thin wrapper scripts; the typical engineer only touches Nix when adding a new dependency.
- **Tooling rough edges.** Nix flakes is the "current best practice" but still has rough corners. Mitigation: pin to specific Nix versions; track release notes.
- **Build times can be slow without caching.** Mitigation: nix-cache shared across CI runners; per-PR diffs hit cache > 95% of the time.

### Trade-offs we accept

We trade Nix's onboarding pain for build hermeticity. This is the right trade for a SLSA-L3-default product; the senior-engineer ICP is mostly Nix-friendly, and the rest can rely on the wrapper scripts.

## Alternatives considered

### Alternative 1: Bazel as default

**Considered**, but:

- Polyglot Bazel rulesets (rules_python, rules_rust, rules_go, rules_nodejs) are uneven in quality.
- Bazel's hermeticity requires careful configuration; easy to get subtle non-hermetic builds.
- Larger learning curve than Nix for our mix.

Kept as a supported alternative for Bazel-native customers.

### Alternative 2: Custom Docker + Renovate-pinned base images

**Rejected for Tier 4**:

- Docker is not hermetic by default (entrypoint differences, base image patches, timestamp noise).
- SLSA-L3 requires bit-identical rebuilds; Docker fails this casually.

(Used for our *deployment* containers, which are built from Nix outputs. Hermetic at the source; OCI-packaged at the edge.)

### Alternative 3: Buck2 (Meta's Bazel-alternative)

**Rejected**:

- Smaller ecosystem than Bazel; less mature polyglot story.
- Adopting Buck2 doesn't give us reproducibility Nix doesn't already give.

### Alternative 4: Pants

**Rejected** for similar reasons — niche, smaller ecosystem.

## Practical setup

```
flake.nix (root)
  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable"
  inputs.rust-overlay.url = ...
  outputs = ...
  
  # Per-component derivations
  packages = {
    control-plane = buildGo {...};
    twin-runtime = buildRust {...};
    distiller = buildPython {...};
    web-console = buildTypeScript {...};
    ...
  };
  
  # Development shells
  devShells = {
    default = mkShell {...};  # all languages
    go-only = mkShell {...};
    rust-only = mkShell {...};
    python-only = mkShell {...};
  };
```

Engineers run `nix develop` to enter a hermetic shell with the right toolchain. CI runs `nix build .#release-bundle`.

## Reproducibility verification

Crucible's own CI uses **two independent build platforms** (GitHub-hosted runner + a self-hosted runner) to produce bit-identical artifacts. Comparison is automated:

```bash
nix build .#release-bundle.x86_64-linux --out-link platform-a
# (on other platform)
nix build .#release-bundle.x86_64-linux --out-link platform-b

diff <(nix hash file platform-a) <(nix hash file platform-b)
# Must match for SLSA-L3 attestation
```

Any divergence is a blocker. We've found and fixed divergences in:

- Timestamp embedding in Go binaries (`-trimpath` mandatory).
- Python `.pyc` timestamp embedding (`PYTHONDONTWRITEBYTECODE=1`).
- TypeScript build output ordering (`prefer-deterministic-bundling` in webpack/biome).

## Customer-side Tier 4 verification

Customers verify our releases:

```bash
crucible verify-release 2026.06.0
```

The command:

1. Pulls the published SLSA Provenance v1 attestation from Rekor.
2. Locally rebuilds the release from the source SHA pinned in the attestation.
3. Compares hashes.
4. Verifies Sigstore signatures chain to the published trust root.

This works *because* we use Nix. Without it, "reproducible" is a marketing claim.

## Open issues

- **Nix flakes "experimental" status.** Officially still flagged experimental in Nix 2.x. Mitigation: pin Nix version; track release notes; participate in the stabilization upstream.
- **Windows support is weak.** Nix on Windows is nascent. Mitigation: the Windows CLI builds via WSL2 + Nix; engineers on Windows use WSL2 dev shells.
- **Rust crates with non-Nix-friendly build.rs scripts.** Occasional issue; fixed case by case.

## References

- [02-engineering/repo-structure.md](../02-engineering/repo-structure.md)
- [02-engineering/testing-strategy.md](../02-engineering/testing-strategy.md)
- [03-sdk/attestation-formats.md](../03-sdk/attestation-formats.md)
