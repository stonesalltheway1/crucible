# Self-Hosted Install

The enterprise tier ships Crucible as a Helm chart + air-gap bundle for fully on-prem or VPC deployment. SLSA-L3 attested, Sigstore-signed, FedRAMP-compatible architecture.

## Deployment topologies

### A. Single-tenant cloud (customer's VPC)

- Customer runs Crucible inside their own AWS/GCP/Azure VPC.
- Outbound to frontier LLM APIs allowed (Anthropic, Google, etc.) over their existing egress.
- Their own KMS for production-promotion signing.
- Their own object storage for tape archives.
- Their own Postgres for memory + attestations.

### B. Air-gapped

- Crucible runs entirely on-prem.
- No outbound connectivity to public LLM APIs.
- Local model inference via vLLM / sglang (Llama 4, DeepSeek V4-Pro, Qwen3-Coder-Plus).
- Local Sigstore Rekor + Fulcio CA.
- Local HSM (Thales / YubiHSM / AWS CloudHSM standalone).
- Air-gap installer bundle (~12 GB) loaded from media.

### C. Hybrid

- Crucible runs on-prem.
- Outbound only to BAA-covered LLM APIs (Anthropic w/ BAA, Azure OpenAI w/ BAA, Vertex AI w/ BAA).
- All other components on-prem.
- Suitable for HIPAA / SOC-2 / financial-services compliance contexts.

## System requirements

| Component | Minimum | Recommended |
|---|---|---|
| Kubernetes cluster | 1.28+, 3 nodes | 1.30+, 6 nodes for HA |
| Node sizing (control plane) | 8 vCPU / 32 GB RAM | 16 vCPU / 64 GB RAM |
| Node sizing (twin runtime pool) | 4 vCPU / 16 GB RAM (per concurrent twin) | scale to peak twin demand |
| Postgres | 15+, 100 GB | 16+, 500 GB+ for memory layer |
| Redis | 7+ | 7+ cluster mode for HA |
| Object storage | S3-compatible, 1 TB | 10 TB+ for tape archive |
| FalkorDB | 1 instance | clustered for HA |
| KMS / HSM | required for promotion-gate signing | hardware HSM for FedRAMP |

GPU optional. Required only for the air-gapped tier where local LLM inference is the only option.

## Components shipped

```
crucible-enterprise/
├── helm/
│   └── crucible/                     # Umbrella chart
│       ├── charts/
│       │   ├── control-plane/
│       │   ├── twin-runtime/         # SaaS-tier: E2B-backed Rust workspace
│       │   ├── twin-runtime-self-host/  # Phase 3: raw Firecracker + ZFS orchestrator
│       │   ├── tape-scrubber/        # Phase 3: Presidio + spaCy + FF3-1
│       │   ├── shadow-recorder/      # Phase 3: capture-time scrub + audit log
│       │   ├── verifier/
│       │   ├── distiller/
│       │   ├── promotion-gate/
│       │   ├── attestation-relay/
│       │   ├── memory-router/
│       │   ├── cost-meter/
│       │   └── web-console/
│       └── values-airgap-default.yaml
│
├── images/                            # OCI image bundle (air-gap)
│   └── *.oci                          # All Crucible images, SLSA-attested
│
├── sigstore/
│   ├── rekor-bundle/                  # Self-hosted Rekor
│   ├── fulcio-bundle/                 # Self-hosted Fulcio CA
│   └── trusted-root.json
│
├── policies/
│   ├── default-rego/                  # Promotion-gate policies
│   └── default-egress/                # Cilium / Tetragon templates
│
├── verifiers/                         # Per-language verifier images
│
├── models/                            # (air-gap only) local model weights
│   ├── llama-4-scout/
│   ├── deepseek-v4-pro/
│   └── qwen3-coder-plus/
│
├── docs/                              # Local copy of these docs
├── slsa-provenance.json               # Provenance for this bundle
└── INSTALL.md
```

## Install steps

### Online install (topologies A + C)

```bash
# 1. Add the Crucible Helm repo
helm repo add crucible https://charts.crucible.dev
helm repo update

# 2. Generate a values.yaml for your environment
crucible-cli generate-values \
  --topology vpc \
  --kms aws-kms \
  --kms-key-arn arn:aws:kms:us-east-1:...:key/... \
  --db-host postgres.internal \
  --db-credentials secretsmanager:crucible-pg \
  --object-storage-bucket s3://acme-crucible-tapes \
  --llm-provider anthropic \
  --llm-api-key-secret secretsmanager:anthropic-key \
  > values.yaml

# 3. Install
helm install crucible crucible/crucible \
  --namespace crucible-system \
  --create-namespace \
  --values values.yaml

# 4. Verify the install
crucible-cli verify-install
  ✓ Control plane reachable
  ✓ Twin runtime provisioning a test sandbox in 187ms
  ✓ DB connectivity verified
  ✓ KMS signing test passed
  ✓ Object storage write/read passed
  ✓ Verifier daemon healthy
  ✓ Web console reachable at https://crucible.acme.internal
```

### Air-gap install (topology B)

```bash
# 1. Mount or download the bundle
mkdir /opt/crucible && cd /opt/crucible
tar xzf crucible-enterprise-2026.06.0.tar.gz

# 2. Verify the bundle integrity (SLSA-L3 attestations + Sigstore root)
./scripts/verify-bundle.sh
  ✓ All OCI images verified against in-toto attestations
  ✓ Helm chart signature verified
  ✓ Model weights checksums match published manifest
  ✓ Sigstore trusted root authenticated

# 3. Load images to the local registry
./scripts/load-images.sh --registry registry.internal.acme.com

# 4. Configure for air-gap (no outbound LLM APIs)
crucible-cli generate-values \
  --topology airgap \
  --kms hsm \
  --hsm-pkcs11-lib /usr/lib/pkcs11/libsofthsm2.so \
  --hsm-slot 0 \
  --rekor-mode self-hosted \
  --llm-provider local-vllm \
  --llm-models llama-4-scout,deepseek-v4-pro,qwen3-coder-plus \
  --gpu-pool-namespace gpu-workloads \
  > values.yaml

# 5. Install
helm install crucible ./helm/crucible \
  --namespace crucible-system \
  --create-namespace \
  --values values.yaml

# 6. Initialize the local Sigstore Rekor and Fulcio
./scripts/init-local-sigstore.sh

# 7. Verify
crucible-cli verify-install --topology airgap
```

## Configuration

### values.yaml (key sections)

```yaml
crucible:
  topology: vpc | airgap | hybrid
  domain: crucible.acme.internal
  
  # Storage
  postgres:
    host: postgres.internal
    credentialsSecret: crucible-pg
    sslMode: require
  redis:
    host: redis.internal
    cluster: true
  falkordb:
    host: falkordb.internal
    cluster: true
  objectStorage:
    type: s3
    endpoint: s3.amazonaws.com
    bucket: acme-crucible-tapes
    credentialsSecret: crucible-s3
  
  # Twin runtime
  twinRuntime:
    sandboxProvider: e2b | firecracker-local
    e2bApiKeySecret: e2b-api-key       # if hosted
    firecrackerPoolSize: 100           # if local
    # Phase 3 self-host orchestrator. The `linux-firecracker` Cargo
    # feature must be enabled in the build; the air-gap installer's
    # twin-runtime-self-host image is compiled with it on.
    selfHost:
      zfsPoolRoot: /var/lib/crucible/zfs
      cgroupParent: /sys/fs/cgroup/crucible
      tetragonPolicyDir: /var/run/tetragon/policies.d
      firecrackerBinary: /usr/local/bin/firecracker

  # Phase 3: PII scrub pipeline. Production deployments REQUIRE the
  # Presidio service; the Go tape_driver's regex fallback is dev-only
  # and is not HIPAA-Safe-Harbor compliant on free-text PII.
  tapeScrubber:
    url: http://crucible-scrubber.crucible-system.svc.cluster.local:9100
    tokenSecret: crucible-scrubber-token
    ff3MasterKeySecret: crucible-scrubber-ff3-key
    engine: default | hipaa            # `hipaa` swaps in the clinical de-identifier
    failClosed: true                   # regulated tenants MUST be true

  # Phase 3: shadow recorder for tape population. Wires the customer's
  # Envoy or eBPF tap to the recorder service.
  shadowRecorder:
    ingress: envoy | ebpf
    tapeBucket: s3://acme-crucible-tapes
    recordSchedule: monthly            # default per ADR-007
    
  # KMS / signing
  kms:
    provider: aws-kms | gcp-cloud-hsm | yubihsm | softhsm | azure-keyvault
    keyRef: arn:aws:kms:...:key/...    # or PKCS11 path for hardware
  
  # Sigstore
  sigstore:
    mode: public | self-hosted
    rekorUrl: https://rekor.internal.acme.com   # if self-hosted
    fulcioUrl: https://fulcio.internal.acme.com # if self-hosted
    trustedRoot: /etc/crucible/sigstore-trusted-root.json
  
  # LLM routing
  llmRouting:
    tier0Provider: anthropic
    tier1Provider: anthropic
    tier2Provider: anthropic
    verifierProvider: google              # cross-family
    airgapLocalProvider: vllm             # for air-gap
    apiKeysSecret: crucible-llm-keys      # contains all configured providers
  
  # Auth
  auth:
    provider: workos | clerk | authelia | dex
    samlMetadataUrl: ...
    oidcDiscoveryUrl: ...
  
  # Observability
  observability:
    tracing:
      exporter: otlp
      endpoint: tempo.internal:4317
    metrics:
      prometheusEndpoint: prom.internal:9090
    logs:
      lokiEndpoint: loki.internal:3100
  
  # Tenant defaults
  defaults:
    promotionPolicy:                  # bundle of default Rego rules
    cartographer:
      maxRepoSize: 5_000_000          # LoC
      modelTier: 1
    
  # Air-gap specifics
  airgap:
    models:
      llamaScoutImage: registry.internal/llama-4-scout:1.0
      deepseekV4ProImage: registry.internal/deepseek-v4-pro:1.0
      qwen3CoderImage: registry.internal/qwen3-coder-plus:1.0
    gpuPool:
      namespace: gpu-workloads
      nodeSelector:
        nvidia.com/gpu.product: A100
```

## Upgrade flow

1. Customer receives release notification (email + status page) 30 days before each release.
2. Customer-facing changelog with breaking changes called out.
3. Air-gap customers receive the new bundle via their preferred channel (signed download + verification command).
4. Helm upgrade:
   ```
   helm upgrade crucible crucible/crucible --values values.yaml
   ```
5. Verify:
   ```
   crucible-cli verify-install
   ```
6. Rollback path (always tested before release):
   ```
   helm rollback crucible <revision>
   ```

Database migrations are forward-only; we don't auto-rollback schema. Migrations are written so that the previous version can still read the new schema (additive only, deprecation windows on column drops).

## SLSA-L3 verification

Every artifact in the bundle is SLSA-L3 attested. Customers can verify:

```bash
crucible-cli verify-release 2026.06.0
  Verifying 47 image attestations against Sigstore Rekor...
  ✓ control-plane@sha256:abc... → rekor:7d8a...
  ✓ twin-runtime@sha256:def... → rekor:9e2b...
  ...
  ✓ All 47 artifacts attested with OIDC subject https://accounts.crucible.dev/builders/...
  ✓ Reproducible-build comparison passed (2 of 2 independent builds bit-identical)
  ✓ Bundle signature verified against Crucible trust root
```

## Customer-controlled signing key

For the highest-assurance tier (FedRAMP / defense), the customer owns the Sigstore Fulcio CA root. Crucible signs nothing on the customer's behalf; the customer's CI signs attestations using their own identity. Crucible's operators do not have signing authority.

## Operational responsibilities

| Responsibility | Crucible (SaaS) | Crucible (VPC) | Customer (self-host) |
|---|---|---|---|
| Helm chart releases | yes | yes | yes |
| Upgrade execution | yes | optional (consulting) | yes |
| Backup of memory + attestations | yes | yes | customer |
| KMS key rotation | yes | yes | customer |
| Incident response | 24/7 | business hours | customer + Crucible advisory |
| Security patches | auto | notify within 24h | customer applies |
| SLO monitoring | yes | yes | customer-side; we provide dashboards |

## Air-gap-specific notes

- **Model updates:** new model weights distributed via signed manifest + media (not network).
- **Local Sigstore root:** the Fulcio CA is bound to the customer's identity provider; rotation procedures documented separately.
- **Local Rekor backup:** Rekor's transparency log must be backed up; loss = audit-trail loss for the period.
- **Telemetry phone-home:** none. We provide tooling for the customer to export anonymized usage stats if they choose.

## Pricing reminder

The self-hosted enterprise tier is $50K/yr base + $400/node/mo. Includes:

- Unlimited use, on-prem inference.
- Quarterly Helm chart releases + air-gap bundle.
- Customer Success contact + business-hours support.
- Annual security review.
- Renewal-time architectural review.

Additional services (24/7 support, named SRE, custom verifier integrations) are scoped separately.
