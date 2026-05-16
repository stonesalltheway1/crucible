# crucible-control-plane

The Phase-1 implementation of the Crucible Agent Control Plane (Block 1 in `docs/07-roadmap/build-plan-agent-days.md`).

## What it does

Accepts a task description, classifies the task (Tier 0–4), builds a Plan via an LLM, enforces hard budget/retry/wall-clock caps, and emits signed in-toto attestations for every interesting transition. The Plan is presented to a user for approval; once approved, Phase 2 will hand off to the Twin Runtime.

```
SubmitTask → taskrouter.Classify (Haiku 4.5, cached 1h)
           → taskrouter.Route    (executor + cross-family verifier)
           → planbuilder.Build   (Sonnet 4.6, JSON output)
           → emit PlanProposal attestation
           → register Enforcer (cost / retry / wall-clock caps)
           → user reviews via GET /v1/tasks/{id}
           → ApprovePlan         (emit PlanApproval attestation)
           → STUB: hand off to twin runtime (Phase 2)
```

## Layout

```
cmd/
  main.go                          process entry; wires every dependency
internal/
  api/server.go                    HTTP handlers, connect-go-compatible shape
  taskrouter/router.go             LLM-driven complexity classification + verifier pairing
  planbuilder/builder.go           Plan construction + plan_hash + PlanProposal attestation
  budgetenforcer/enforcer.go       ADR-009 hard caps (cost, retry, wall-clock)
  modelrouter/tiers.go             5-tier model table (May 2026 pricing)
  modelrouter/client.go            vendor-neutral Request / Response
  modelrouter/anthropic.go         anthropic-sdk-go v1.43 client
  modelrouter/google.go            google.golang.org/genai v1.57 client
  modelrouter/openai.go            openai-go/v3 Responses API client
  costmeter/meter.go               per-task cost JSONL + enforcer debit
  events/publisher.go              in-memory + webhook fan-out
  tenantpolicy/loader.go           per-tenant residency / vendor allow-list
  store/store.go                   in-memory task store (Phase 2 → Postgres)
```

## Build & run

```bash
nix develop
cd apps/control-plane
go build ./...
./cmd/main &

# Env that matters
export ANTHROPIC_API_KEY=sk-ant-...
export GOOGLE_API_KEY=AIza...          # optional, for verifier
export OPENAI_API_KEY=sk-...           # optional, alternate Tier 1/2
export CRUCIBLE_LISTEN_ADDR=:8080      # default
export CRUCIBLE_DEFAULT_TENANT=acme    # default "single-tenant"

# Smoke test
curl -s http://localhost:8080/healthz | jq
curl -s -X POST http://localhost:8080/v1/tasks \
  -H 'content-type: application/json' \
  -d '{"description":"Add a Stripe refund webhook handler","repo":"github.com/acme/payments","base_sha":"abc123"}' | jq
```

## REST endpoints

| Method | Path                              | Purpose                               |
|--------|-----------------------------------|---------------------------------------|
| GET    | `/healthz`                        | liveness + version + Phase-1 stubs    |
| POST   | `/v1/tasks`                       | submit a task                         |
| GET    | `/v1/tasks`                       | list tasks                            |
| GET    | `/v1/tasks/{id}`                  | fetch task with Plan + Budget         |
| POST   | `/v1/tasks/{id}/approve`          | approve the Plan                      |
| POST   | `/v1/tasks/{id}/reject`           | reject the Plan                       |
| POST   | `/v1/tasks/{id}/replan`           | rebuild the Plan with new caps        |
| GET    | `/v1/tasks/{id}/budget`           | live budget snapshot                  |

Shapes match `libs/twin-spec/proto/crucible/v1/control_plane.proto` 1:1.

## Currency notes (May 2026 — flagged in `docs/PHASE-1-REPORT.md`)

- Anthropic cache TTL default flipped from 1h to 5m on 2026-03-06; we set `ttl: "1h"` explicitly on the system + tool slots.
- Gemini 3+ requires `thinking_level` (`LOW`/`MEDIUM`/`HIGH`) over the legacy `thinking_budget`.
- `gpt-5.1-codex-max` is in the model table but flagged unverified on official pricing as of May 2026.
- Vertex AI Go SDK (`cloud.google.com/go/vertexai/genai`) is being removed 2026-06-24; we use `google.golang.org/genai` (the GA unified SDK).
- `openai-go` v3.28+ changed the `voice` param shape; we only use the Responses API so we're unaffected.

## Phase 2 hand-off

- The `api.Server` HTTP handlers are 1:1 with the proto `ControlPlaneService`; wire them into a `connect-go` service definition once `buf generate` runs.
- The `costmeter` + `events.MultiPublisher` are constructed in `main.go` but the Phase-1 API surface does not drive them yet (they activate once the Twin Runtime starts charging the Enforcer per LLM call). Hooks are in place; the runtime is just stubbed.
- The Twin Runtime, Verifier, Promotion Gate, and Memory Layer are all stubbed via the absence of their RPCs. Phase 2 adds them as sibling apps under `apps/`.
