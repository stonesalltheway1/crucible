# Crucible Helm chart

Production-deploy umbrella for the full Crucible stack. This is the
unit you `helm install` to ship Crucible into a Kubernetes cluster.
Per the Phase-8 brief, this is the production-deploy unit; local dev
keeps using docker-compose.

## Sub-charts

Every Crucible service has its own sub-chart under `charts/`. They are
deployed in dependency order; the umbrella handles the wiring.

| Sub-chart | Default port | Purpose |
|---|---|---|
| control-plane | :8080 | Task router + plan builder + budget enforcer |
| twin-runtime | :7444 | Sandbox + filesystem twin + DB twin |
| verifier | :9080 | Tier 0–4 verification daemon |
| memory-router | :8090 | Hot-path memory retrieval |
| distiller | :8091 | LLM-driven Convention extraction |
| cartographer | :9420 | Day-1 customer experience |
| shadow-recorder | :9520 | Tape population from staging traffic |
| tape-scrubber | :9100 | Presidio + spaCy + FF3-1 PII scrub |
| promotion-gate | :9180 | Rego + KMS + Argo Rollouts |
| attestation-relay | :9120 | Sigstore Rekor v2 publisher |
| cost-meter | :9220 | Per-task USD + token telemetry |
| web-console | :3000 | Next.js 15 dashboard |
| github-app | :9320 | `/crucible <description>` PR-comment surface |
| slack-bot | :9280 | Slash command + DM notifier |

## Install

### Online (VPC / hybrid)

```bash
helm repo add crucible https://charts.crucible.dev
helm repo update
helm install crucible crucible/crucible \
    --namespace crucible-system \
    --create-namespace \
    --values values.yaml \
    --values values-aws.yaml          # or -gcp.yaml / -azure.yaml
```

### Air-gap

```bash
# Load images into your local OCI registry first via
# infra/air-gap-bundle/scripts/load-images.sh.
helm install crucible ./infra/helm/crucible \
    --namespace crucible-system \
    --create-namespace \
    --values ./infra/helm/crucible/values.yaml \
    --values ./infra/helm/crucible/values-airgap-default.yaml \
    --set global.imageRegistry=registry.internal
```

## Cosign signing

The packaged chart is signed via Sigstore keyless OIDC at release
time. To verify:

```bash
cosign verify-blob \
    --certificate-identity-regexp 'https://github.com/crucible/.*' \
    --certificate-oidc-issuer https://token.actions.githubusercontent.com \
    --signature crucible-2026.6.0.tgz.sig \
    --bundle crucible-2026.6.0.tgz.cosign.bundle \
    crucible-2026.6.0.tgz
```

## Upgrade + rollback

```bash
helm upgrade crucible crucible/crucible --values values.yaml
crucible verify-release 2026.06.0   # confirm release attestations
helm rollback crucible <revision>   # always tested before each release
```

Database migrations are forward-only; the previous version can read
the new schema (additive only). See
docs/04-operations/self-hosted-install.md.

## Verifying the install

```bash
crucible verify-install
  ✓ Control plane reachable
  ✓ Twin runtime provisioning a test sandbox in 187ms
  ✓ DB connectivity verified
  ✓ KMS signing test passed
  ✓ Object storage write/read passed
  ✓ Verifier daemon healthy
  ✓ Web console reachable at https://crucible.acme.internal
```
