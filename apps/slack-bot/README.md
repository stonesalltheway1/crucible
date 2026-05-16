# crucible-slack-bot

The approval-routing UI for Crucible's promotion gate. Minimal Phase-6
surface: receive `task.promotion_proposed` webhooks, render an
approve/reject button, route signed approvals back to the gate.

## Architecture

```
gate (POST /v1/promotions) ──webhook──▶ slack-bot ──Slack Web API──▶ #crucible-approvals
                                                            │
                                                            ▼
                                              human clicks Approve / Reject
                                                            │
                                                            ▼
                                              slack-bot verifies SSO + signs
                                              approval as DSSE via the relay
                                                            │
                                                            ▼
                                              POST /v1/promotions/{id}/approve
```

## Identity

The bot enforces **OIDC**:

- Slack OAuth establishes the user's Slack identity.
- The workspace's SAML/SSO binding maps the Slack user to a corporate
  OIDC subject (Okta / Auth0 / WorkOS / Azure AD).
- The bot's `MintApprovalCert` flow exchanges the OIDC token for a
  short-lived Sigstore-keyless cert via the relay's `/v1/attestations`
  endpoint (the cert binds the approval to the human, not the bot).
- The bot signs the approval-attestation predicate with that cert and
  publishes via the relay.

## API

| Method | Path | Purpose |
|---|---|---|
| POST | `/webhook/promotion_proposed` | Inbound webhook from the gate |
| POST | `/slack/interactive` | Slack's interactive callback |
| POST | `/slack/slash/promote` | Admin slash command |
| GET  | `/healthz` | Health |

## Env

| Variable | Purpose |
|---|---|
| `CRUCIBLE_SLACK_BOT_ADDR` | Bind |
| `CRUCIBLE_GATE_ADDR` | Where to POST `/v1/promotions/{id}/approve` |
| `CRUCIBLE_RELAY_ADDR` | Relay base URL |
| `SLACK_BOT_TOKEN` | xoxb token |
| `SLACK_SIGNING_SECRET` | for inbound signature verification |
| `CRUCIBLE_SAML_BINDING_URL` | optional |
| `CRUCIBLE_APPROVERS_CHANNEL` | default `#crucible-approvals` |

## Guardrails

- **Self-approval forbidden.** When a user clicks approve, the bot checks
  the rendered promotion's `agent_oidc_subject` against the user's bound
  OIDC subject. Mismatch on equality → reject before signing.
- **Stale approval rejection.** The interactive message carries the
  bundle's `diff_hash` as part of the action's `value`; the bot embeds it
  in the approval payload and the gate refuses the approval if the
  current bundle differs.
- **N-of-M routing.** When the gate's decision requires N approvers, the
  Slack message reflects "approved 1 / 2" until the quorum is reached.

## Dev

```bash
export CRUCIBLE_SLACK_BOT_ADDR=:9280 \
       CRUCIBLE_GATE_ADDR=http://127.0.0.1:9180 \
       CRUCIBLE_RELAY_ADDR=http://127.0.0.1:9120 \
       SLACK_BOT_TOKEN=xoxb-test \
       SLACK_SIGNING_SECRET=test \
       CRUCIBLE_APPROVERS_CHANNEL=#crucible-test
go run ./cmd/crucible-slack-bot
```

Use ngrok or Cloudflare Tunnel to expose `:9280` so Slack's interactive
callbacks can reach you in dev.
