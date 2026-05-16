# Event Spec

Webhook event payloads emitted by Crucible to customer-configured endpoints. Subscribe via the web console or REST API at `/v1/webhooks/subscriptions`.

## Delivery semantics

- **At-least-once delivery.** Idempotency keys are included; receivers should dedupe.
- **Signed payloads.** Every webhook carries `X-Crucible-Signature` (HMAC-SHA256 of payload using the subscription's signing secret) + `X-Crucible-Sigstore-Bundle` (Sigstore keyless attestation for high-stakes events).
- **JSON content-type, UTF-8.**
- **Retry policy:** 5 attempts with exponential backoff (1s, 4s, 16s, 64s, 256s). After exhaustion, the event lands in a dead-letter queue visible in the web console.

## Event types

### `task.submitted`

```json
{
  "event_id": "evt_01HZ...",
  "event_type": "task.submitted",
  "occurred_at": "2026-05-15T18:24:31Z",
  "tenant_id": "ten_...",
  "task": {
    "id": "task_01HZ...",
    "description": "Add Stripe webhook handler for refund events",
    "submitted_by": "user_...",
    "submitted_from": "cursor-mcp",
    "repo": "github.com/acme/payments",
    "base_sha": "abcd1234..."
  }
}
```

### `task.plan_proposed`

```json
{
  "event_id": "evt_01HZ...",
  "event_type": "task.plan_proposed",
  "occurred_at": "...",
  "tenant_id": "ten_...",
  "task_id": "task_01HZ...",
  "plan": {
    "description": "...",
    "estimated_cost_usd": 1.20,
    "estimated_duration_min": 8,
    "files_to_touch": ["api/webhooks.ts", "..."],
    "db_migrations": 1,
    "external_effects": [{"service":"stripe","endpoints":["/webhooks/refund"],"live":false}],
    "top_risks": [{"description":"...","impact":"high"}],
    "retry_budget_per_step": 3,
    "wall_clock_budget_min": 15
  },
  "approval_url": "https://app.crucible.dev/tasks/.../plan"
}
```

### `task.plan_approved` / `task.plan_rejected`

```json
{
  "event_type": "task.plan_approved",
  "task_id": "task_01HZ...",
  "approved_by": "user_...",
  "approved_at": "..."
}
```

### `task.step_started` / `task.step_completed`

Granular per-step. Useful for streaming progress UIs.

```json
{
  "event_type": "task.step_completed",
  "task_id": "task_01HZ...",
  "step_id": "step_03",
  "step_name": "Author handler + idempotency key check",
  "duration_seconds": 47.3,
  "cost_usd": 0.31,
  "files_changed": ["api/webhooks.ts"]
}
```

### `task.budget_warning` / `task.budget_exceeded`

```json
{
  "event_type": "task.budget_exceeded",
  "task_id": "task_01HZ...",
  "spent_usd": 2.04,
  "cap_usd": 2.00,
  "halted": true,
  "next_action": "user_replan_required"
}
```

### `task.destructive_proposal`

Fires whenever the syscall shim intercepts a destructive command. For twin-scoped proposals, this is informational. For real-scoped, the customer's approval flow is triggered.

```json
{
  "event_type": "task.destructive_proposal",
  "task_id": "task_01HZ...",
  "proposal": {
    "id": "prop_...",
    "command": "DROP TABLE users_archived",
    "scope": "twin",
    "justification": "agent: cleaning up unused archive table",
    "blast_radius": {
      "affected_resources": ["table:users_archived"],
      "reversibility": "snapshot",
      "impact_score": 0.4
    }
  },
  "approval_required": false
}
```

### `task.verification_started` / `task.verification_completed`

```json
{
  "event_type": "task.verification_completed",
  "task_id": "task_01HZ...",
  "verdict": "approved",
  "rubric_score": 0.92,
  "tier_results": {
    "tier_0": {"passed": true, "mutation_score": 0.91},
    "tier_1": {"passed": true, "pbt_iterations": 10000, "counterexamples": []},
    "tier_4": {"passed": true, "rebuild_hash": "...", "rekor_uuid": "..."}
  },
  "rejection_reasons": [],
  "attestations": ["rekor:..."],
  "signed_by_oidc": "https://accounts.crucible.dev/agents/...",
  "signed_at": "..."
}
```

### `task.promotion_proposed` / `.approved` / `.rejected` / `.deploying` / `.canary_dwell` / `.landed` / `.rolled_back`

The full lifecycle of a promotion. The Slack approval bot uses `.promotion_proposed` to render the approve/reject button.

```json
{
  "event_type": "task.promotion_landed",
  "task_id": "task_01HZ...",
  "promotion_id": "prom_...",
  "rollout_strategy": "canary",
  "canary_steps": [
    {"weight": 1, "dwell_seconds": 300, "slo_check": "passed"},
    {"weight": 5, "dwell_seconds": 600, "slo_check": "passed"},
    {"weight": 25, "dwell_seconds": 1800, "slo_check": "passed"},
    {"weight": 100, "dwell_seconds": 0, "slo_check": "passed"}
  ],
  "final_attestation": "rekor:..."
}
```

### `task.completed` / `task.failed` / `task.cancelled`

Final event for any task. `task.completed` indicates both verification and (if applicable) promotion succeeded.

```json
{
  "event_type": "task.completed",
  "task_id": "task_01HZ...",
  "outcome": "verified_and_promoted",
  "total_cost_usd": 1.69,
  "total_duration_min": 12.4,
  "files_changed": [...],
  "pr_url": "https://github.com/acme/payments/pull/...",
  "rekor_attestations": ["rekor:...", "..."]
}
```

### `memory.convention_drift_detected`

Fired by the distillation worker when an active convention's positive/negative ratio drops below threshold.

```json
{
  "event_type": "memory.convention_drift_detected",
  "tenant_id": "ten_...",
  "convention_id": "conv_...",
  "rule_nl": "API errors return { error: { code, message } } envelope",
  "scope": {"file_glob":"api/**/*.ts"},
  "positive_ratio_30d": 1.2,
  "threshold": 1.5,
  "suggested_action": "user_confirm_or_supersede",
  "console_url": "https://app.crucible.dev/memory/conventions/conv_..."
}
```

### `memory.convention_learned`

Fired when a candidate convention graduates to active.

```json
{
  "event_type": "memory.convention_learned",
  "tenant_id": "ten_...",
  "convention_id": "conv_...",
  "rule_nl": "PRs that touch billing/ require @payments-leads approval",
  "category": "PR/commit hygiene",
  "confidence": 0.82,
  "source_evidence": [{"kind":"pr_comment","pr":1234,"comment_id":"..."},"..."],
  "now_active": true
}
```

### `security.egress_violation` (P0)

```json
{
  "event_type": "security.egress_violation",
  "tenant_id": "ten_...",
  "task_id": "task_01HZ...",
  "attempted_host": "evil.example.com:443",
  "blocked_at_layer": "tetragon",
  "agent_process": "...",
  "killed": true,
  "incident_id": "inc_..."
}
```

### `security.cross_tenant_access_attempt` (P0)

The most severe internal event. Always alerts on-call.

### `system.sigstore_unreachable` (P1)

Attestation publish failed. Local journaling continues; back-fill is automatic.

### `system.kms_signing_failure` (P1)

Promotion blocked.

## Subscription configuration

Webhook subscriptions are managed at the tenant level:

```bash
crucible webhook create \
  --url https://hooks.acme.com/crucible \
  --events 'task.*,memory.convention_drift_detected,security.*' \
  --signing-secret-from $CRUCIBLE_HOOK_SECRET \
  --description "Production webhooks"
```

Or via REST:

```
POST /v1/webhooks/subscriptions
{
  "url": "https://hooks.acme.com/crucible",
  "events": ["task.*", "memory.convention_drift_detected", "security.*"],
  "signing_secret": "<from-vault>",
  "active": true
}
```

Event-name globs are supported. `*` matches one segment; `**` matches all (`task.**` = every task event).

## Signature verification

```python
import hmac, hashlib

def verify(payload_bytes, signing_secret, header_signature):
    expected = hmac.new(
      signing_secret.encode(),
      payload_bytes,
      hashlib.sha256
    ).hexdigest()
    return hmac.compare_digest(expected, header_signature)
```

For high-stakes events (`promotion.*`, `security.*`), additionally verify the `X-Crucible-Sigstore-Bundle` against Sigstore's trust root.

## Rate limits

- Per-subscription: 100 events/sec sustained; 1000 events/sec burst.
- Beyond burst: events queue; if queue exceeds 10K events, oldest dropped + `system.webhook_queue_overflow` event fired.

## Replay

The web console exposes a "redeliver event" button per event. Bulk replay via API:

```
POST /v1/webhooks/subscriptions/{id}/redeliver
{ "event_ids": ["evt_...", "evt_..."] }
```

Useful for catch-up after a receiver outage.
