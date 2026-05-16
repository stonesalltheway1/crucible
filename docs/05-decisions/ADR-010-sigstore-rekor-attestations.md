# ADR-010: Sigstore Rekor v2 for transparency log

**Status:** Accepted  
**Date:** 2026-05-15

## Context

Crucible's promise of "verified, auditable, reproducible" requires a cryptographic audit trail. Every agent action — file write, tool call, shell command, plan approval, verifier verdict, promotion decision — must be:

1. Signed by an authenticated identity.
2. Published to a write-once, tamper-evident log.
3. Independently verifiable by any party.
4. Resilient to operator (Crucible) compromise.

## Decision

**Sigstore Rekor v2** is the default transparency log. Attestations follow the **in-toto attestation framework** with **DSSE** envelopes signed by **Sigstore keyless OIDC** (Fulcio-issued short-lived certs).

For each deployment:

- **SaaS tier:** public Sigstore Rekor v2.
- **Self-hosted enterprise:** self-hosted Rekor v2 + self-hosted Fulcio CA, bound to customer's identity provider.
- **Air-gapped:** same as enterprise; transparency log lives entirely on-prem.

Tier 4 (Honest CI) emits SLSA Provenance v1 in addition to Crucible-specific predicate types.

## Consequences

### Positive

- **Industry standard.** Sigstore is the de-facto signing infrastructure for open-source supply chains; customers' security teams already know it.
- **Keyless OIDC eliminates long-lived signing key management.** Each signing event mints a short-lived cert; no key rotation operationally.
- **Public-by-default.** SaaS tier attestations go to the public Rekor log — anyone can verify our customers' agent actions without our cooperation. Strong trust signal.
- **Replayability.** Every task's attestation chain reconstructs the full action sequence; debugging and audit become the same workflow.
- **SLSA-L3 by default.** Tier 4 emits the SLSA Provenance v1 predicate, which is exactly what regulated buyers want.

### Negative

- **Public log = public metadata.** SaaS-tier attestations expose the customer's action timestamps and OIDC subjects (not the code itself, but the existence of activity). Mitigation: customers can opt for self-hosted Rekor.
- **Sigstore dependency.** Public Rekor outages block attestation publishing. Mitigation: local journaling continues during outages; back-fill on recovery (RB-05).
- **Storage growth.** Rekor entries are append-only. Customer-side mirroring grows ~1MB/day for typical usage. Negligible; documented in retention policy.
- **OIDC issuer dependency.** Sigstore Fulcio depends on an OIDC issuer (GitHub, Google, custom). For self-hosted, the customer must run their own.

## Alternatives considered

### Alternative 1: Custom append-only Postgres table with hash chain

Implement a simple hash-chained ledger in Postgres. **Rejected as primary**:

- Reinvents Rekor poorly.
- No external verification; depends on Crucible operators not lying.
- Acceptable as solo-founder tier fallback when Rekor is overkill.

### Alternative 2: AWS QLDB

Append-only ledger with cryptographic verification. **Rejected**:

- AWS QLDB EOL'd 2025; no clean replacement narrative.
- AWS-locked.

### Alternative 3: Hyperledger Fabric / blockchain-style ledger

Use a permissioned blockchain. **Rejected**:

- Operational complexity wildly disproportionate to need.
- Customers' security teams have heard the word "blockchain" enough times that it's a sales-cycle slowing word, not accelerating.

### Alternative 4: Custom signed manifests + S3 Object Lock

Sign manifests with our own key, write to S3 with immutability. **Rejected**:

- Depends on long-lived signing keys (key-management overhead).
- No external transparency-log verification.
- Customer's compliance team would need to vet our key custody.

### Alternative 5: Signing only at promotion-time

Sign the final promotion bundle but not every intermediate action. **Rejected**:

- Doesn't enable replay / fork-from-step / blame.
- Doesn't catch attestation chain breaks mid-task.
- Misses the cost-accountability narrative ("every token spend traceable").

## In-toto predicate types Crucible defines

See [03-sdk/attestation-formats.md](../03-sdk/attestation-formats.md) for full schemas. Summary:

- `https://crucible.dev/WriteAttestation/v1` — file writes
- `https://crucible.dev/MigrationAttestation/v1` — DB migrations
- `https://crucible.dev/ServiceCallAttestation/v1` — service calls
- `https://crucible.dev/DestructiveProposal/v1` — intercepted destructive ops
- `https://crucible.dev/DestructiveApproval/v1` — approved destructive ops
- `https://crucible.dev/TestReport/v1` — verifier test runs
- `https://crucible.dev/VerifierApproval/v1` / `VerifierRejection/v1` — final verdicts
- `https://crucible.dev/PlanApproval/v1` — user plan approval
- `https://crucible.dev/PromotionBundle/v1` — promotion submissions
- `https://crucible.dev/PromotionApproval/v1` — gate decisions
- `https://crucible.dev/PromotionOutcome/v1` — final outcome (landed/rolled-back)
- `https://crucible.dev/MemoryWrite/v1` — procedural memory writes

## OIDC issuer chain

- **SaaS tier:** `accounts.crucible.dev` (our own issuer; runs on Dex).
- **Enterprise tier:** customer's existing IdP (Okta, Auth0, Azure AD, WorkOS, custom).
- **Air-gapped tier:** customer's on-prem IdP (Authelia, Keycloak, custom).

Crucible's own employee actions (deploy attestations, etc.) use Sigstore's standard `accounts.google.com` / `github.com` OIDC paths.

## Trust root rotation

Sigstore root rotation happens out-of-band per Sigstore's published schedule. We track and consume root updates within 30 days. For customer-controlled deployments (enterprise), the customer manages their own root and rotation schedule.

## Open issues

- **Rekor v2 GA stability** — v2 GA'd recently; one or two rough edges expected. Mitigation: pin to specific versions; backport bug fixes if needed.
- **Inclusion proof verification at scale** — Rekor witness verification has some latency at p99; not user-blocking but worth monitoring.
- **Post-quantum migration** — Sigstore uses standard ECDSA; PQC transition follows industry timeline (no v1 action).

## References

- [03-sdk/attestation-formats.md](../03-sdk/attestation-formats.md)
- [01-architecture/threat-model.md](../01-architecture/threat-model.md)
