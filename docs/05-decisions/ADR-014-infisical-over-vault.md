# ADR-014: Infisical as default secrets vault

**Status:** Accepted  
**Date:** 2026-05-15

## Context

The twin runtime's secrets layer requires:

- **Dynamic, short-lived credentials.** Sub-minute TTL.
- **OSS self-host option.** Air-gapped enterprise tier must run without vendor connectivity.
- **Modern developer experience.** SDKs in our four languages, clean CLI.
- **Per-tenant scoping.** Each tenant's secrets isolated.
- **Reasonable cost at scale.** Per-tenant secrets management can multiply quickly.

The mainstream options in May 2026:

| Vault | OSS self-host | Dynamic secrets | Pricing | Notes |
|---|---|---|---|---|
| HashiCorp Vault Community | Yes | Yes (mature) | Free OSS / $1,150+/mo HCP Dedicated | HCP Vault Secrets EOL July 2026 |
| Infisical | Yes (OSS) | Yes (PG/MySQL/Mongo/etc.) | $8/user/mo Pro; free OSS | Modern DX |
| Doppler | No self-host | Limited dynamic | $7/user/mo Team | SaaS-only |
| AWS Secrets Manager + STS | No (AWS-only) | IAM session tokens | $0.40/secret/mo + API | Best in all-AWS |
| 1Password Connect | Yes | Limited | Per-user | Dev-friendly but limited dynamic |

## Decision

**Infisical** is the default secrets vault for Crucible.

- **SaaS tier:** Infisical Cloud (or our managed Infisical deployment).
- **Self-hosted enterprise:** Infisical OSS self-host.
- **Customer override:** customers with existing Vault investment can swap via values.yaml (`vault.provider: hashicorp-vault`).

For the production-promotion signing key (separate concern from twin secrets), we use **AWS KMS / GCP Cloud HSM / YubiHSM** per deployment — these handle HSM-backed signing for the unseal ceremony.

## Consequences

### Positive

- **Modern dev experience.** The SDK and CLI are pleasant; engineers adopt without complaint.
- **OSS self-host is real.** Air-gap installation works without licensing dramas.
- **Dynamic secrets across the engines we care about.** Postgres, MySQL, Mongo, Redis, custom — all supported.
- **Lower operational footprint than Vault.** Infisical is "vault for small/medium teams"; Vault is "vault for large enterprises with dedicated team."
- **Pricing scales sanely.** $8/user/mo on Cloud is reasonable; OSS is free.

### Negative

- **Younger company than HashiCorp.** More single-vendor risk; smaller community.
- **Some advanced Vault features missing.** Vault's auth-method ecosystem is broader (Vault has Kubernetes, AWS IAM, AppRole, LDAP, OIDC, JWT, GitHub, GCP, Azure, AliCloud, Kerberos auth methods out of the box). Infisical is catching up but not at parity.
- **Customer existing-Vault investment.** Some customers already run Vault; we support them via the override but our default is Infisical.

### Trade-offs we accept

We bet on the modern-DX project over the incumbent. The HCP Vault Secrets EOL July 2026 announcement created uncertainty about HashiCorp's roadmap for the "secrets-as-a-service" use case; Infisical's roadmap is cleaner for our needs.

## Alternatives considered

### Alternative 1: HashiCorp Vault Community as default

**Rejected as default** (kept as override option):

- Operational footprint heavy for what we need.
- HCP-EOL drama creates strategic uncertainty.
- Vault's dynamic secrets are mature, but Infisical is sufficient.

### Alternative 2: AWS Secrets Manager + STS

**Rejected as default**:

- AWS-locked. Multi-cloud and air-gap customers can't adopt.
- Reasonable for AWS-native customers; supported via override.

### Alternative 3: Doppler

**Rejected**:

- SaaS-only; no self-host story.
- Dynamic secrets less mature than Infisical's.

### Alternative 4: Roll our own

Build a minimal secrets engine. **Rejected** — pointless reinvention; commodity layer.

### Alternative 5: Cloud-native KMS only

Use AWS KMS / GCP Secret Manager + workload identity directly, skip a vault layer. **Rejected**:

- Couples the architecture to a specific cloud.
- Doesn't handle the twin-scoped ephemeral-credential pattern cleanly.
- Vault-like abstraction is the right level for our use.

## Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│  TWIN SANDBOX                                                       │
│  ┌─────────────────────┐         ┌──────────────────────────────┐  │
│  │  Agent process      │         │  Infisical sidecar           │  │
│  │  - calls twin.secret.get(name) │  - holds long-lived token to │  │
│  │  - receives SecretRef          │    Infisical (sidecar-only)  │  │
│  │  - never sees raw value        │  - issues dynamic, twin-     │  │
│  └──────────┬──────────┘         │    scoped, sub-min TTL token │  │
│             │                     └──────────────┬───────────────┘  │
│             ▼                                    │                  │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │  Egress proxy                                                │   │
│  │  - intercepts outgoing service calls                         │   │
│  │  - resolves $secret(name)$ placeholder to injected token     │   │
│  │  - logs which secrets used in which calls                    │   │
│  └─────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────┘
                              │
                              ▼  (real services)
```

The agent process **never sees secret values.** It holds opaque `SecretRef`s. At egress, the proxy substitutes the actual token. This is the architectural enforcement of secrets isolation.

For production promotion (entirely separate path), the **KMS-signed credential lease** is issued by AWS KMS / GCP Cloud HSM / YubiHSM directly to the deploy pipeline, never to the agent.

## Backup / disaster recovery

- Infisical Cloud: vendor-managed.
- Self-host: standard Postgres backup (Infisical's data lives in Postgres); customer responsibility.
- KMS keys: customer responsibility (typical AWS KMS / GCP / HSM rotation policies apply).

## References

- [01-architecture/twin-runtime.md#layer-5-secrets-twin](../01-architecture/twin-runtime.md)
- [01-architecture/promotion-contract.md](../01-architecture/promotion-contract.md)
- [01-architecture/threat-model.md](../01-architecture/threat-model.md)
