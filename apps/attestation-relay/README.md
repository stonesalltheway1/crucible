# crucible-attestation-relay

The Rust service that turns Crucible action records into signed, publicly
verifiable attestations. Replaces the Phase-1 local-journal-only publisher.

## Responsibilities

| Stage | Implementation |
|---|---|
| Build the in-toto Statement | `predicate::*` — generators for all 13 Crucible predicate types |
| Sign via DSSE | `dsse` — DSSEv1 PAE + Ed25519 or Sigstore-issued cert |
| Obtain Fulcio cert | `fulcio` — OIDC token → short-lived x509 cert |
| Publish to Rekor v2 | `rekor` — POST `/api/v2/log/entries`; verify inclusion proof |
| Fallback to local journal | `journal` — hash-chained JSONL, back-fills on recovery |
| Self-hosted Rekor | `rekor::Endpoint::SelfHosted{url, root_ca}` |
| HTTP surface | `server` — REST + webhook on `:9120` |

## API

| Method | Path | Purpose |
|---|---|---|
| POST | `/v1/attestations` | Build → sign → publish; returns RekorEntry |
| POST | `/v1/attestations/raw` | Caller supplies a pre-built Statement |
| GET  | `/v1/attestations/{uuid}` | Fetch the DSSE envelope by UUID |
| GET  | `/v1/attestations/{uuid}/inclusion` | Inclusion proof against Rekor root |
| GET  | `/v1/journal/tail` | Last N entries of the local journal (admin) |
| POST | `/v1/journal/backfill` | Trigger manual backfill |
| GET  | `/healthz` | Health + Rekor reachability + journal size |
| GET  | `/v1/predicates` | List the 13 supported predicate types |

## Environment

| Variable | Purpose | Default |
|---|---|---|
| `CRUCIBLE_RELAY_ADDR` | Bind address | `:9120` |
| `CRUCIBLE_REKOR_URL` | Rekor v2 endpoint | `https://rekor.sigstore.dev` |
| `CRUCIBLE_REKOR_SELF_HOSTED` | `1` to skip public Sigstore | unset |
| `CRUCIBLE_REKOR_ROOT_CA` | Path to self-hosted Rekor CA bundle | unset |
| `CRUCIBLE_FULCIO_URL` | Fulcio endpoint | `https://fulcio.sigstore.dev` |
| `CRUCIBLE_OIDC_ISSUER` | OIDC issuer URI | `https://accounts.crucible.dev` |
| `CRUCIBLE_OIDC_TOKEN` | Pre-issued OIDC token (dev) | unset |
| `CRUCIBLE_JOURNAL_PATH` | Local hash-chained journal | `~/.crucible/attestations/relay-journal.jsonl` |
| `CRUCIBLE_RELAY_DEV_KEYS` | Dev-key directory (Ed25519 local fallback) | `~/.crucible/relay-keys/` |
| `CRUCIBLE_RELAY_OFFLINE` | `1` forces journal-only (no Rekor) | unset |

## Production vs dev

- **Dev:** `CRUCIBLE_RELAY_OFFLINE=1` writes to the local journal only. The
  envelopes are signed with a locally-generated Ed25519 key. Same shape as
  production; the only delta is the Cert field.
- **SaaS:** Fulcio keyless OIDC + public Sigstore Rekor.
- **Self-hosted:** point `CRUCIBLE_FULCIO_URL` and `CRUCIBLE_REKOR_URL` at
  the customer's own Sigstore instances; provide `CRUCIBLE_REKOR_ROOT_CA`.

## Self-hosted Rekor

The relay verifies inclusion proofs against a configured root. The check is
identical regardless of whether the entry came from public Sigstore or a
customer's on-prem Rekor — only the trust root differs.

## Threat-model alignment

- T2 (forged bundle): the relay refuses to publish a Statement whose subject
  digest doesn't match the supplied content.
- T7 (tampered artifact): the relay's `verify_chain` recomputes every
  subject digest off the envelope and refuses on any mismatch.
- T8 (action repudiation): every publish lands in the journal AND Rekor
  (when available); the local journal is hash-chained so deletion is
  detectable.
- T20 (egress in promotion path): the relay only reaches Rekor + Fulcio;
  the OS-level allowlist on the relay's host is asserted by the deploy
  manifest (`infra/argo-rollouts/relay.yaml`).
- T21 (compromised approver): the relay enforces `agent_oidc_subject !=
  approver_oidc_subject` at envelope construction time when building the
  PromotionApproval/v1 predicate.

## Build

```
cargo build --release -p crucible-attestation-relay
```

The binary is fully static against musl when built with the Nix profile.
