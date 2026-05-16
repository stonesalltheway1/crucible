# crucible CLI

The Phase-1 Crucible command-line interface. Single static Go binary; cross-platform via `nix build .#cli`.

## Commands

```
crucible version
crucible health
crucible task new      --description "..." [--repo ...] [--base-sha ...] [--cost-cap-usd N] [--wall-clock-cap-min N]
crucible task list
crucible task get      <task_id>
crucible plan show     <task_id>
crucible plan approve  <task_id> [--cost-cap-usd N] [--wall-clock-cap-min N] [--retry-cap N]
crucible plan reject   <task_id> --reason "..."
crucible budget show   <task_id>
```

## Global flags

```
--endpoint    Control-plane URL  (env: CRUCIBLE_ENDPOINT, default http://localhost:8080)
--tenant      Tenant ID          (env: CRUCIBLE_TENANT, default single-tenant)
--json        Emit JSON instead of human-readable text
```

## End-to-end example

```bash
$ crucible health
status:  ok
version: 2026.06.0-phase1
now:     2026-05-15T...
stubs:   twin=true verifier=true promotion=true

$ crucible task new --description "Add a Stripe refund webhook handler" --repo github.com/acme/payments
Task task_01HZX... submitted.
  Status:      awaiting_approval
  Executor:    claude-sonnet-4-6 (anthropic, tier 2)
  Verifier:    gemini-3.1-pro (google)
  Plan:        5 steps, est $0.84 over 12 min
  Plan hash:   c0ffee...

Review plan:    crucible plan show task_01HZX...
Approve plan:   crucible plan approve task_01HZX...

$ crucible plan show task_01HZX...
Plan for task_01HZX...
────────────────────────────────────────────────────────────────
Description:  Add a Stripe refund webhook handler
Complexity:   standard
Estimate:     $0.84 over 12 min
Caps:         retry 3/step, wall-clock 15 min
Plan hash:    c0ffeec0ffee...
Files:        api/webhooks.ts, db/migrations/..., test/webhooks.test.ts
Migrations:   1
Steps:
   1. Read existing webhook handler structure   (retry 0/3)
   2. Author handler + idempotency key check    (retry 0/3)
   ...

$ crucible plan approve task_01HZX...
Approved task_01HZX... (plan c0ffeec0ffee).
Attestation: 8a91c3f4b2e6...
Status: approved
```

## Phase 2 hand-off

- The CLI talks JSON-over-HTTP to the control plane. Phase 2 swaps the transport to connect-go's typed client without changing the surface.
- `crucible task watch <id>` for live event streams ships with Phase 2 (the InMemory event publisher + SSE bridge are already in place server-side).
- `crucible attestation chain <task_id>` and `crucible attestation verify <uuid>` ship with Phase 6 (Provenance Plumbing).
