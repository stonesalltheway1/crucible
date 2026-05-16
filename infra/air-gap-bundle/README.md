# Air-gap installer bundle

The single signed tarball that ships Crucible to FedRAMP / defense /
regulated customers. Per docs/04-operations/self-hosted-install.md
§"Air-gap install (topology B)" and the Phase-8 brief.

## Contents

```
crucible-airgap-bundle-2026.06.0.tar.gz
├── images/                    # OCI images of every Crucible service
│   └── *.oci                  # SLSA-L3 attested
├── helm/
│   └── crucible-2026.6.0.tgz  # Cosign-signed helm chart
├── sigstore/
│   ├── rekor-bundle/          # Self-hosted Sigstore Rekor v2
│   ├── fulcio-bundle/         # Self-hosted Fulcio CA
│   └── trusted-root.json
├── policies/
│   ├── default-rego/          # Promotion-gate baseline policies
│   └── default-egress/        # Cilium / Tetragon templates
├── verifiers/                 # Per-language verifier images
├── models/                    # Local LLM weights for the air-gap fallback
│   ├── llama-4-scout/
│   ├── deepseek-v4-pro/
│   └── qwen3-coder-plus/
├── docs/                      # Local copy of these docs
├── slsa-provenance.json       # Provenance for THIS bundle
├── bundle.manifest.json       # SHA-256 manifest of every file
├── bundle.cosign.bundle       # Signature
└── INSTALL.md                 # Walkthrough
```

## Build the bundle

```bash
infra/air-gap-bundle/scripts/build-bundle.sh \
    --version 2026.06.0 \
    --out dist/crucible-airgap-bundle-2026.06.0.tar.gz
```

The build is hermetic via Nix: `nix build .#airgap-bundle` reproduces
the same byte-identical tarball on any host.

## Verify the bundle (customer side)

```bash
tar xzf crucible-airgap-bundle-2026.06.0.tar.gz
cd crucible-airgap-bundle-2026.06.0
./scripts/verify-bundle.sh
  ✓ All OCI images verified against in-toto attestations
  ✓ Helm chart signature verified
  ✓ Model weights checksums match published manifest
  ✓ Sigstore trusted root authenticated
```

## Load the images

```bash
./scripts/load-images.sh --registry registry.internal.acme.com
```

## Initialize the local Sigstore

```bash
./scripts/init-local-sigstore.sh
```

## Install

```bash
helm install crucible ./helm/crucible-2026.6.0.tgz \
    --namespace crucible-system \
    --create-namespace \
    --values ./helm/values-airgap-default.yaml
```

## SLSA L3 Provenance v1

`slsa-provenance.json` is in-toto attestation of the bundle build.
Verify with `cosign verify-blob`:

```bash
cosign verify-blob \
    --certificate-identity-regexp 'https://github.com/crucible/.*' \
    --certificate-oidc-issuer https://token.actions.githubusercontent.com \
    --signature bundle.cosign.bundle \
    crucible-airgap-bundle-2026.06.0.tar.gz
```

## Offline verification

The bundle ships with a self-contained `verify-bundle.sh` that uses the
embedded Sigstore trusted root and local cosign binary. No network
access required.
