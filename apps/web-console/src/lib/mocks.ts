// Demo data for the web-console pages.
//
// In production, every route hydrates from the control-plane SDK; the mocks
// here keep the surface legible during local development when the backend
// stubs are off. The shapes match libs/sdk-go and the zod schemas in api.ts.

import type { Attestation, Convention, Plan, Promotion, Task } from "./api";

export function mockTask(id: string): Task {
  const plan: Plan = {
    description:
      "Add an idempotency-key check to the Stripe refund webhook so retried deliveries do not double-refund. Use the existing `idempotency_keys` table; do not introduce a new feature flag.",
    estimated_cost_usd: 0.42,
    estimated_duration_min: 3,
    files_to_touch: ["api/webhooks/stripe.ts", "api/webhooks/stripe.test.ts", "db/idempotency_keys_repo.ts"],
    db_migrations: 0,
    external_effects: [
      { service: "stripe", endpoints: ["/v1/webhook_endpoints/te_*"], live: false },
      { service: "internal-events-bus", endpoints: ["events:refund_processed"], live: false },
    ],
    top_risks: [
      {
        description: "Webhook signature verification path must remain on the same code path",
        impact: "high",
        mitigation: "Property-based test asserts pre-check signature verify still runs first",
      },
      {
        description: "Idempotency key collision with the existing payment_intents path",
        impact: "med",
        mitigation: "Use namespaced key `refund:<event_id>` not the raw event id",
      },
      { description: "Existing tests assume happy-path; need to assert replay branch", impact: "low" },
    ],
    retry_budget_per_step: 3,
    wall_clock_budget_min: 15,
    hard_cap_usd: 2.0,
  };
  return {
    id,
    tenant_id: "ten_demo",
    description: "Add idempotency key to /webhooks/stripe/refund",
    submitted_by: "sarah@acme.dev",
    submitted_from: "cursor-mcp",
    repo: "github.com/acme/payments",
    base_sha: "abcd1234ef56...",
    status: "plan_pending_approval",
    submitted_at: new Date(Date.now() - 4 * 60_000).toISOString(),
    cost_usd: 0,
    duration_seconds: 0,
    pr_url: null,
    plan,
    steps: [],
    attestations: [],
  };
}

export function mockTaskList(): Task[] {
  const now = Date.now();
  return [
    {
      ...mockTask("task_01HZAB_x4"),
      status: "plan_pending_approval",
      submitted_at: new Date(now - 4 * 60_000).toISOString(),
    },
    {
      ...mockTask("task_01HZAB_m2"),
      description: "Bump fast-check to 4.x across packages/*",
      status: "verifying",
      submitted_at: new Date(now - 21 * 60_000).toISOString(),
      submitted_by: "marcus@acme.dev",
      cost_usd: 0.91,
      duration_seconds: 540,
    },
    {
      ...mockTask("task_01HZAB_d9"),
      description: "Replace direct DB writes in /lib/orders with the repo pattern",
      status: "promotion_pending",
      submitted_at: new Date(now - 2.5 * 3600_000).toISOString(),
      cost_usd: 1.84,
      duration_seconds: 1320,
      submitted_by: "priya@acme.dev",
    },
    {
      ...mockTask("task_01HYZ_99"),
      description: "Remove dead module `lib/legacy_invoice` and its tests",
      status: "completed",
      submitted_at: new Date(now - 6 * 3600_000).toISOString(),
      completed_at: new Date(now - 5.6 * 3600_000).toISOString(),
      pr_url: "https://github.com/acme/payments/pull/2841",
      cost_usd: 0.34,
      duration_seconds: 420,
    },
    {
      ...mockTask("task_01HYZ_71"),
      description: "Convert callbacks to async/await in /jobs/dunning_worker.ts",
      status: "failed",
      submitted_at: new Date(now - 1.2 * 24 * 3600_000).toISOString(),
      cost_usd: 1.10,
      duration_seconds: 850,
    },
  ];
}

export function mockPromotion(id: string): Promotion {
  return {
    id,
    task_id: "task_01HZAB_d9",
    tenant_id: "ten_demo",
    status: "canary_dwell",
    submitted_at: new Date(Date.now() - 8 * 60_000).toISOString(),
    decision: {
      allow: true,
      needs_human: true,
      approver_groups: ["@payments-leads", "@platform-team"],
      require_n_approvers: 2,
      auto_approve: false,
      reasons: ["touches billing/orders/*", "blast_radius.reversibility = drift"],
      trace: { path: "allow.canary_with_approvers", policy_hash: "9d3c0f8a92f7...4e" },
    },
    bundle: {
      diff_hash: "0x8c7eaa9b6d4f...",
      files_changed: [
        { path: "lib/orders/repo.ts", action: "create" },
        { path: "lib/orders/index.ts", action: "modify" },
        { path: "lib/orders/repo.test.ts", action: "create" },
      ],
      blast_radius: { reversibility: "drift", impact_score: 0.36 },
      agent_oidc_subject: "https://accounts.crucible.dev/agents/worker-7",
    },
    canary: {
      adapter: "argo-rollouts",
      current_step: 1,
      steps: [
        { weight: 1, dwell_seconds: 300, slo_check: "passed", started_at: new Date(Date.now() - 600_000).toISOString() },
        { weight: 25, dwell_seconds: 1800, slo_check: "pending", started_at: new Date(Date.now() - 60_000).toISOString() },
        { weight: 100, dwell_seconds: 0, slo_check: "pending" },
      ],
    },
    approvals: [
      {
        approver_oidc_subject: "sarah@acme.dev",
        group: "@payments-leads",
        approved_at: new Date(Date.now() - 7 * 60_000).toISOString(),
        attestation: "rekor:appr-a1b2c3d4",
      },
    ],
  };
}

export function mockPromotions(): Promotion[] {
  const a = mockPromotion("prom_01HZAB_p7");
  const b = mockPromotion("prom_01HZAB_p3");
  return [
    a,
    { ...b, status: "pending_approval", canary: undefined, approvals: [] },
    { ...mockPromotion("prom_01HYZ_91"), status: "landed", canary: undefined },
    { ...mockPromotion("prom_01HYZ_72"), status: "rolled_back" },
  ];
}

export function mockConventions(): Convention[] {
  return [
    {
      id: "conv_01HZA_envelope",
      tenant_id: "ten_demo",
      rule_nl: "API errors return { error: { code, message } } envelope",
      category: "error-handling",
      status: "active",
      scope: { repo: "github.com/acme/payments", file_glob: "api/**/*.ts" },
      confidence: 0.93,
      positive_examples_30d: 47,
      negative_examples_30d: 1,
      last_violated_at: new Date(Date.now() - 12 * 86_400_000).toISOString(),
      source_evidence: [
        { kind: "adr", url: "https://github.com/acme/payments/blob/main/docs/adr/0012-error-envelope.md", ref: "ADR-0012" },
        { kind: "pr_comment", ref: "PR #2123", excerpt: "use the envelope, don't return bare strings" },
      ],
      supersedes: [],
      superseded_by: null,
      source_layer: "tenant_distilled",
    },
    {
      id: "conv_01HZA_idem",
      tenant_id: "ten_demo",
      rule_nl: "Webhook handlers must implement idempotency via the idempotency_keys table",
      category: "PR/commit hygiene",
      status: "drifting",
      scope: { repo: "github.com/acme/payments", file_glob: "api/webhooks/**/*.ts" },
      confidence: 0.74,
      positive_examples_30d: 4,
      negative_examples_30d: 3,
      last_violated_at: new Date(Date.now() - 36 * 3600_000).toISOString(),
      source_evidence: [
        { kind: "incident", ref: "INC-2231", excerpt: "double-refund regression Apr 2026" },
        { kind: "pr_comment", ref: "PR #2740", excerpt: "we always use the idempotency table for /webhooks" },
      ],
      supersedes: [],
      superseded_by: null,
      source_layer: "tenant_distilled",
    },
    {
      id: "conv_01HZA_repo",
      tenant_id: "ten_demo",
      rule_nl: "DB access goes through repository modules in lib/<entity>/repo.ts; no direct query builders in handlers",
      category: "architecture",
      status: "active",
      scope: { repo: "github.com/acme/payments" },
      confidence: 0.88,
      positive_examples_30d: 23,
      negative_examples_30d: 0,
      last_violated_at: null,
      source_evidence: [
        { kind: "agents_md", url: "AGENTS.md", excerpt: "DB queries belong in repo modules" },
      ],
      supersedes: [],
      superseded_by: null,
      source_layer: "customer",
    },
    {
      id: "conv_01HZA_axios_old",
      tenant_id: "ten_demo",
      rule_nl: "Use `axios` for HTTP calls",
      category: "tooling",
      status: "superseded",
      scope: { repo: "github.com/acme/payments" },
      confidence: 0.31,
      positive_examples_30d: 0,
      negative_examples_30d: 0,
      last_violated_at: null,
      source_evidence: [],
      supersedes: [],
      superseded_by: "conv_01HZA_fetch",
      source_layer: "tenant_distilled",
    },
    {
      id: "conv_01HZA_pii_log",
      tenant_id: "ten_demo",
      rule_nl: "Never log PII (email, name, full card numbers) — use the scrubber",
      category: "security",
      status: "active",
      scope: { repo: "github.com/acme/payments" },
      confidence: 0.97,
      positive_examples_30d: 89,
      negative_examples_30d: 0,
      last_violated_at: null,
      source_evidence: [
        { kind: "adr", ref: "ADR-0008", url: "https://github.com/acme/payments/blob/main/docs/adr/0008-pii-logging.md" },
      ],
      supersedes: [],
      superseded_by: null,
      source_layer: "global_default",
    },
  ];
}

export function mockAttestation(uuid: string): Attestation {
  return {
    rekor_uuid: uuid,
    predicate_type: "VerifierApproval/v1",
    subject: { name: "task_01HZAB_d9", digest: { sha256: "8c7eaa9b6d4f..." } },
    signed_at: new Date(Date.now() - 9 * 60_000).toISOString(),
    signed_by_oidc: "https://accounts.crucible.dev/agents/verifier-3",
    validation: "valid",
    rekor_inclusion_proof: { tree_size: 184_722, log_index: 184_711, root_hash: "be1cf9d3..." },
    cert_chain: [],
    predicate: {
      verdict: "approved",
      rubric_score: 0.92,
      tier_results: {
        tier_0: { passed: true, mutation_score: 0.91 },
        tier_1: { passed: true, pbt_iterations: 10000, counterexamples: [] },
        tier_4: { passed: true, rebuild_hash: "be1cf9d3..." },
      },
      memory_compliance: { conventions_checked: 12, violations: 0 },
    },
    self_hosted: false,
  };
}

export function mockAttestationChain(taskId: string) {
  const ids = [
    { rekor_uuid: "rekor:plan-a1b2", predicate_type: "Plan/v1", label: "Plan proposed", signed_at: new Date(Date.now() - 20 * 60_000).toISOString() },
    { rekor_uuid: "rekor:approval-c3d4", predicate_type: "PlanApproval/v1", label: "Plan approved by sarah@", signed_at: new Date(Date.now() - 19 * 60_000).toISOString() },
    { rekor_uuid: "rekor:write-e5f6", predicate_type: "TwinFsWrite/v1", label: "Write lib/orders/repo.ts", signed_at: new Date(Date.now() - 15 * 60_000).toISOString() },
    { rekor_uuid: "rekor:test-g7h8", predicate_type: "TestRun/v1", label: "Run tier-0 mutation", signed_at: new Date(Date.now() - 12 * 60_000).toISOString() },
    { rekor_uuid: "rekor:verifier-i9j0", predicate_type: "VerifierApproval/v1", label: "Verifier verdict: approved", signed_at: new Date(Date.now() - 9 * 60_000).toISOString() },
    { rekor_uuid: "rekor:bundle-k1l2", predicate_type: "PromotionBundle/v1", label: "Promotion bundle assembled", signed_at: new Date(Date.now() - 8 * 60_000).toISOString() },
    { rekor_uuid: "rekor:appr-a1b2c3d4", predicate_type: "PromotionApproval/v1", label: "Promotion approved by @payments-leads", signed_at: new Date(Date.now() - 7 * 60_000).toISOString() },
    { rekor_uuid: "rekor:lease-m3n4", predicate_type: "KmsLease/v1", label: "Lease minted (action=deploy_artifact)", signed_at: new Date(Date.now() - 6 * 60_000).toISOString() },
    { rekor_uuid: "rekor:outcome-o5p6", predicate_type: "PromotionOutcome/v1", label: "Canary step 1: passed", signed_at: new Date(Date.now() - 4 * 60_000).toISOString() },
  ];
  const edges = ids.slice(1).map((n, i) => ({ from: ids[i].rekor_uuid, to: n.rekor_uuid }));
  return { task_id: taskId, nodes: ids, edges };
}
