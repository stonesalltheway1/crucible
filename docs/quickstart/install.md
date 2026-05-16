---
title: Install
description: Five-minute install for SaaS or self-hosted Crucible.
---

# Install

There are three install paths. Pick the one that matches your
environment.

## SaaS (recommended for most teams)

1. Sign up at [app.crucible.dev/signup](https://app.crucible.dev/signup)
   — your tenant is provisioned within seconds.
2. Install the **Crucible GitHub App** on the repos you want covered.
3. (Optional) Install the **Crucible Slack App** for approval routing.
4. (Optional) Install the IDE plugin:
   - VS Code Marketplace: `crucible.crucible-vscode`
   - JetBrains Marketplace: `Crucible`
   - Zed: `extensions install crucible`

Time to first verified PR target: **≤ 30 minutes**.

## Self-hosted (single-tenant cloud / VPC)

```bash
helm repo add crucible https://charts.crucible.dev
helm install crucible crucible/crucible \
    --namespace crucible-system --create-namespace \
    --values values.yaml \
    --values values-aws.yaml          # or -gcp / -azure
crucible-cli verify-install
```

See [04-operations/self-hosted-install](/04-operations/self-hosted-install)
for the full configuration surface.

## Air-gap (FedRAMP / regulated)

Download the signed bundle from the customer portal, verify, load
images, install. End-to-end target: **≤ 1 hour from a clean cluster**.

```bash
tar xzf crucible-airgap-bundle-2026.06.0.tar.gz
cd crucible-airgap-bundle-2026.06.0
./scripts/verify-bundle.sh
./scripts/load-images.sh --registry registry.internal.acme.com
./scripts/init-local-sigstore.sh
helm install crucible ./helm/crucible-2026.6.0.tgz \
    --namespace crucible-system --create-namespace \
    --values values.yaml --values ./helm/values-airgap-default.yaml
```

The bundle includes Sigstore Rekor + Fulcio CA + local LLM weights.
Zero outbound network access required.
