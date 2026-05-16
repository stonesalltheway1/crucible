# Crucible Air-Gap Install — full walkthrough

This walkthrough takes a clean Kubernetes cluster from zero to a fully
functional air-gapped Crucible installation in **≤ 1 hour**.

## Prerequisites

| Component | Minimum |
|---|---|
| Kubernetes | 1.28+ |
| Nodes | 3 (HA), 1 (single-tenant) |
| OCI registry | reachable from cluster, with anonymous push enabled OR a credential mount |
| HSM | one of: YubiHSM 2, Thales, AWS CloudHSM standalone, SoftHSM (lab only) |
| Postgres | 15+ (CloudSQL / RDS / on-prem) |
| Object storage | S3-compatible (MinIO / Ceph / EMC ECS / etc.) |
| GPUs | required for local LLM inference; see `models/` for the per-pool sizing |

## Step-by-step

### 1. Mount or copy the bundle

```bash
mkdir /opt/crucible && cd /opt/crucible
tar xzf crucible-airgap-bundle-2026.06.0.tar.gz
cd crucible-airgap-bundle-2026.06.0
```

### 2. Verify the bundle

```bash
./scripts/verify-bundle.sh
```

Expected output:
```
✓ Manifest hash matches all 47 files
✓ All 47 OCI images verified against in-toto attestations
✓ Helm chart signature verified (cosign keyless OIDC)
✓ Model-weights checksums match the published manifest
✓ Sigstore trusted root authenticated against the bundled chain
✓ SLSA Provenance v1 verified
```

### 3. Load images into your local registry

```bash
./scripts/load-images.sh --registry registry.internal.acme.com
```

This pushes ~47 images to your registry. The script preserves SLSA
attestations as OCI referrers, so customers can `cosign verify
--certificate-identity-regexp ...` on the loaded images.

### 4. Initialize the local Sigstore (Rekor + Fulcio)

```bash
./scripts/init-local-sigstore.sh \
    --rekor-url https://rekor.internal.acme.com \
    --fulcio-url https://fulcio.internal.acme.com \
    --oidc-issuer https://accounts.acme.internal
```

This deploys Rekor v2 + Fulcio CA into your cluster (or stands them
up via systemd — the script handles both).

### 5. Configure for air-gap

```bash
./scripts/generate-values.sh \
    --kms yubihsm \
    --hsm-pkcs11-lib /usr/lib/pkcs11/libsofthsm2.so \
    --hsm-slot 0 \
    --rekor-mode self-hosted \
    --llm-provider local-vllm \
    --llm-models llama-4-scout,deepseek-v4-pro,qwen3-coder-plus \
    --gpu-pool-namespace gpu-workloads \
    > values.yaml
```

### 6. Install

```bash
helm install crucible ./helm/crucible-2026.6.0.tgz \
    --namespace crucible-system \
    --create-namespace \
    --values values.yaml \
    --values ./helm/values-airgap-default.yaml
```

### 7. Verify the install

```bash
crucible-cli verify-install --topology airgap
```

Expected:
```
✓ Control plane reachable
✓ Twin runtime provisioning a test sandbox in 187ms
✓ DB connectivity verified
✓ KMS signing test passed
✓ Object storage write/read passed
✓ Verifier daemon healthy
✓ Web console reachable at https://crucible.acme.internal
```

## Wall-clock measurement (target ≤ 1h)

On a 3-node cluster with cached dependencies:

| Step | Typical |
|---|---|
| 1. Mount + tar xzf | 2 min |
| 2. verify-bundle.sh | 2 min |
| 3. load-images.sh | 18 min |
| 4. init-local-sigstore.sh | 8 min |
| 5. generate-values.sh | 1 min |
| 6. helm install | 6 min |
| 7. verify-install | 3 min |
| **Total** | **~40 min** |

## Disconnected-host operation

The bundle is fully self-contained. The above commands require ZERO
outbound network access from the install host to anywhere on the
public internet.

The signature-verify step uses the embedded Sigstore trusted root
under `sigstore/trusted-root.json`; no calls to `rekor.sigstore.dev`
are made.

## Troubleshooting

If `verify-bundle.sh` fails:

- **Manifest hash mismatch** → bundle was corrupted in transit;
  re-download.
- **OCI image attestation missing** → verify the bundle came from a
  signed-distribution channel; do NOT attempt to bypass.
- **Helm chart signature** → `cosign` binary version too old; the
  bundle ships a known-good cosign at `bin/cosign`.

If `load-images.sh` fails:

- **Registry unreachable** → ensure you can `crane catalog
  registry.internal.acme.com` first.
- **Push refused** → registry requires auth; export `REGISTRY_AUTH`
  before re-running.

## Customer-controlled signing key (FedRAMP tier)

For the highest-assurance tier, the customer's CI signs every release
artifact with the customer's own Fulcio CA. Crucible operators do not
have signing authority. Per
docs/04-operations/self-hosted-install.md §"Customer-controlled signing
key".

```bash
./scripts/init-local-sigstore.sh \
    --customer-fulcio-ca /etc/customer-fulcio/root.pem \
    --customer-fulcio-key-store yubihsm:slot=1
```
