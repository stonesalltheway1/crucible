---
title: Verify a Crucible release
description: How to verify Crucible's own releases against published Sigstore attestations.
---

# Verify a Crucible release

Crucible eats its own dogfood: every release passes through Crucible's
own Tier 0–4 verifier ladder before tagging. Customers can verify
each release independently.

```bash
crucible verify-release 2026.06.0
  Verifying 47 image attestations against Sigstore Rekor...
  ✓ control-plane@sha256:abc... → rekor:7d8a...
  ✓ twin-runtime@sha256:def... → rekor:9e2b...
  ...
  ✓ All 47 artifacts attested with OIDC subject https://accounts.crucible.dev/builders/...
  ✓ Reproducible-build comparison passed (2 of 2 independent builds bit-identical)
  ✓ Bundle signature verified against Crucible trust root
```

Every component in every release ships with:

- A **SLSA Provenance v1** in-toto attestation
- A **Sigstore keyless OIDC signature** (no long-lived keys held by
  Crucible)
- A **Rekor v2 transparency log entry**

For air-gap deployments the Rekor instance is local; the verification
command uses the bundled trusted root.

## Customer-controlled signing

The highest-assurance enterprise tier (FedRAMP / defense) signs every
release artifact with the customer's own Fulcio CA. Crucible operators
do not have signing authority. See
[04-operations/self-hosted-install#customer-controlled-signing-key](/04-operations/self-hosted-install).
