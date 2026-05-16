// Crucible API client for the web console.
//
// All reads go through this module. Auth is injected by Clerk / WorkOS via
// the `Authorization: Bearer <jwt>` header at the request-time boundary;
// server-side calls inject from the tenant session cookie.
//
// Tenant scoping: every endpoint takes a tenantId. The control plane refuses
// any call whose JWT tenant claim mismatches the path tenant — defence in
// depth on top of the JWT validation.

import { z } from "zod";

export const TaskStatus = z.enum([
  "submitted",
  "planning",
  "plan_pending_approval",
  "plan_approved",
  "executing",
  "verifying",
  "verified",
  "verification_failed",
  "promotion_pending",
  "promoted",
  "completed",
  "failed",
  "cancelled",
]);
export type TaskStatusT = z.infer<typeof TaskStatus>;

export const PlanRiskSchema = z.object({
  description: z.string(),
  impact: z.enum(["low", "med", "high"]),
  mitigation: z.string().optional(),
});

export const PlanSchema = z.object({
  description: z.string(),
  estimated_cost_usd: z.number(),
  estimated_duration_min: z.number(),
  files_to_touch: z.array(z.string()),
  db_migrations: z.number().int().default(0),
  external_effects: z.array(
    z.object({
      service: z.string(),
      endpoints: z.array(z.string()),
      live: z.boolean(),
    }),
  ),
  top_risks: z.array(PlanRiskSchema),
  retry_budget_per_step: z.number().int(),
  wall_clock_budget_min: z.number(),
  hard_cap_usd: z.number(),
});
export type Plan = z.infer<typeof PlanSchema>;

export const StepSchema = z.object({
  step_id: z.string(),
  name: z.string(),
  status: z.enum(["pending", "running", "completed", "failed", "skipped"]),
  cost_usd: z.number().default(0),
  duration_seconds: z.number().default(0),
  files_changed: z.array(z.string()).default([]),
  started_at: z.string().optional(),
  completed_at: z.string().optional(),
  error: z.string().optional(),
});
export type Step = z.infer<typeof StepSchema>;

export const VerifierReportSchema = z.object({
  verdict: z.enum(["approved", "rejected", "pending"]),
  rubric_score: z.number(),
  tier_results: z.record(
    z.object({
      passed: z.boolean(),
      mutation_score: z.number().optional(),
      pbt_iterations: z.number().optional(),
      counterexamples: z.array(z.unknown()).optional(),
      rebuild_hash: z.string().optional(),
      rekor_uuid: z.string().optional(),
    }),
  ),
  rejection_reasons: z
    .array(
      z.object({
        severity: z.enum(["info", "warn", "error"]),
        code: z.string(),
        message: z.string(),
      }),
    )
    .default([]),
  attestations: z.array(z.string()).default([]),
  signed_by_oidc: z.string().optional(),
  signed_at: z.string().optional(),
});

export const TaskSchema = z.object({
  id: z.string(),
  tenant_id: z.string(),
  description: z.string(),
  submitted_by: z.string(),
  submitted_from: z.string().optional(),
  repo: z.string(),
  base_sha: z.string().optional(),
  status: TaskStatus,
  submitted_at: z.string(),
  completed_at: z.string().nullable().optional(),
  cost_usd: z.number().default(0),
  duration_seconds: z.number().default(0),
  pr_url: z.string().nullable().optional(),
  plan: PlanSchema.optional(),
  steps: z.array(StepSchema).optional(),
  verifier: VerifierReportSchema.optional(),
  attestations: z.array(z.string()).default([]),
});
export type Task = z.infer<typeof TaskSchema>;

export const PromotionStatus = z.enum([
  "pending_approval",
  "approved",
  "rejected",
  "deploying",
  "canary_dwell",
  "landed",
  "rolled_back",
  "cancelled",
]);

export const PromotionSchema = z.object({
  id: z.string(),
  task_id: z.string(),
  tenant_id: z.string(),
  status: PromotionStatus,
  submitted_at: z.string(),
  decision: z
    .object({
      allow: z.boolean(),
      needs_human: z.boolean(),
      approver_groups: z.array(z.string()).optional(),
      require_n_approvers: z.number().int().optional(),
      auto_approve: z.boolean().optional(),
      reasons: z.array(z.string()).optional(),
      trace: z.object({ path: z.string(), policy_hash: z.string() }).optional(),
    })
    .optional(),
  bundle: z
    .object({
      diff_hash: z.string(),
      files_changed: z.array(
        z.object({ path: z.string(), action: z.string() }),
      ),
      blast_radius: z.object({
        reversibility: z.string(),
        impact_score: z.number(),
      }),
      agent_oidc_subject: z.string(),
    })
    .optional(),
  canary: z
    .object({
      adapter: z.string(),
      steps: z.array(
        z.object({
          weight: z.number(),
          dwell_seconds: z.number(),
          slo_check: z.enum(["pending", "passed", "failed"]),
          started_at: z.string().optional(),
        }),
      ),
      current_step: z.number().int().default(0),
    })
    .optional(),
  approvals: z
    .array(
      z.object({
        approver_oidc_subject: z.string(),
        group: z.string(),
        approved_at: z.string(),
        attestation: z.string(),
      }),
    )
    .default([]),
});
export type Promotion = z.infer<typeof PromotionSchema>;

export const ConventionSchema = z.object({
  id: z.string(),
  tenant_id: z.string(),
  rule_nl: z.string(),
  category: z.string(),
  status: z.enum(["candidate", "active", "drifting", "superseded"]),
  scope: z.object({
    repo: z.string().optional(),
    file_glob: z.string().optional(),
  }),
  confidence: z.number(),
  positive_examples_30d: z.number().int(),
  negative_examples_30d: z.number().int(),
  last_violated_at: z.string().nullable().optional(),
  source_evidence: z
    .array(
      z.object({
        kind: z.string(),
        url: z.string().optional(),
        ref: z.string().optional(),
        excerpt: z.string().optional(),
      }),
    )
    .default([]),
  supersedes: z.array(z.string()).default([]),
  superseded_by: z.string().nullable().optional(),
  source_layer: z.enum(["customer", "tenant_distilled", "global_default"]),
});
export type Convention = z.infer<typeof ConventionSchema>;

export const AttestationSchema = z.object({
  rekor_uuid: z.string(),
  predicate_type: z.string(),
  subject: z.object({ name: z.string(), digest: z.record(z.string()) }),
  signed_at: z.string(),
  signed_by_oidc: z.string(),
  validation: z.enum(["valid", "invalid", "pending"]),
  rekor_inclusion_proof: z.unknown().optional(),
  cert_chain: z.array(z.string()).optional(),
  predicate: z.record(z.unknown()),
  self_hosted: z.boolean().default(false),
});
export type Attestation = z.infer<typeof AttestationSchema>;

export const AttestationChainSchema = z.object({
  task_id: z.string(),
  nodes: z.array(
    z.object({
      rekor_uuid: z.string(),
      predicate_type: z.string(),
      signed_at: z.string(),
      label: z.string(),
    }),
  ),
  edges: z.array(z.object({ from: z.string(), to: z.string() })),
});

const BASE = process.env.NEXT_PUBLIC_CRUCIBLE_API ?? "http://localhost:8080";

async function get<T>(path: string, schema: z.ZodSchema<T>, init?: RequestInit): Promise<T> {
  const res = await fetch(`${BASE}${path}`, {
    headers: { Accept: "application/json", ...(init?.headers || {}) },
    cache: "no-store",
    ...init,
  });
  if (!res.ok) {
    throw new Error(`GET ${path}: ${res.status} ${await res.text()}`);
  }
  return schema.parse(await res.json());
}

async function post<T>(
  path: string,
  body: unknown,
  schema: z.ZodSchema<T>,
  init?: RequestInit,
): Promise<T> {
  const res = await fetch(`${BASE}${path}`, {
    method: "POST",
    headers: { "Content-Type": "application/json", Accept: "application/json", ...(init?.headers || {}) },
    body: JSON.stringify(body),
    ...init,
  });
  if (!res.ok) {
    throw new Error(`POST ${path}: ${res.status} ${await res.text()}`);
  }
  return schema.parse(await res.json());
}

export const api = {
  base: BASE,

  listTasks: (tenantId: string, opts?: { limit?: number; status?: string }) =>
    get(
      `/v1/tenants/${tenantId}/tasks?` +
        new URLSearchParams({
          limit: String(opts?.limit ?? 50),
          ...(opts?.status ? { status: opts.status } : {}),
        }),
      z.object({ tasks: z.array(TaskSchema), next_cursor: z.string().nullable() }),
    ),

  getTask: (tenantId: string, taskId: string) =>
    get(`/v1/tenants/${tenantId}/tasks/${taskId}`, TaskSchema),

  approvePlan: (tenantId: string, taskId: string, edits: Partial<Plan> | null) =>
    post(
      `/v1/tenants/${tenantId}/tasks/${taskId}/plan/approve`,
      { edits },
      z.object({ ok: z.literal(true), task: TaskSchema }),
    ),

  rejectPlan: (tenantId: string, taskId: string, reason: string) =>
    post(
      `/v1/tenants/${tenantId}/tasks/${taskId}/plan/reject`,
      { reason },
      z.object({ ok: z.literal(true) }),
    ),

  interruptTask: (tenantId: string, taskId: string, reason: string) =>
    post(
      `/v1/tenants/${tenantId}/tasks/${taskId}/interrupt`,
      { reason },
      z.object({ ok: z.literal(true) }),
    ),

  listPromotions: (tenantId: string, opts?: { status?: string }) =>
    get(
      `/v1/tenants/${tenantId}/promotions?` +
        new URLSearchParams({ ...(opts?.status ? { status: opts.status } : {}) }),
      z.object({ promotions: z.array(PromotionSchema) }),
    ),

  getPromotion: (tenantId: string, promotionId: string) =>
    get(`/v1/tenants/${tenantId}/promotions/${promotionId}`, PromotionSchema),

  approvePromotion: (
    tenantId: string,
    promotionId: string,
    body: { group: string; bundle_hash_bound: string },
  ) =>
    post(
      `/v1/tenants/${tenantId}/promotions/${promotionId}/approve`,
      body,
      z.object({ ok: z.literal(true), promotion: PromotionSchema }),
    ),

  rejectPromotion: (tenantId: string, promotionId: string, reason: string) =>
    post(
      `/v1/tenants/${tenantId}/promotions/${promotionId}/reject`,
      { reason },
      z.object({ ok: z.literal(true) }),
    ),

  rollbackPromotion: (tenantId: string, promotionId: string, reason: string) =>
    post(
      `/v1/tenants/${tenantId}/promotions/${promotionId}/rollback`,
      { reason },
      z.object({ ok: z.literal(true) }),
    ),

  listConventions: (tenantId: string, opts?: { scope_repo?: string; status?: string }) =>
    get(
      `/v1/tenants/${tenantId}/memory/conventions?` +
        new URLSearchParams({
          ...(opts?.scope_repo ? { scope_repo: opts.scope_repo } : {}),
          ...(opts?.status ? { status: opts.status } : {}),
        }),
      z.object({ conventions: z.array(ConventionSchema) }),
    ),

  getConvention: (tenantId: string, conventionId: string) =>
    get(`/v1/tenants/${tenantId}/memory/conventions/${conventionId}`, ConventionSchema),

  setConventionStatus: (
    tenantId: string,
    conventionId: string,
    status: Convention["status"],
  ) =>
    post(
      `/v1/tenants/${tenantId}/memory/conventions/${conventionId}/status`,
      { status },
      z.object({ ok: z.literal(true), convention: ConventionSchema }),
    ),

  getAttestation: (rekorUuid: string) =>
    get(`/v1/attestations/${encodeURIComponent(rekorUuid)}`, AttestationSchema),

  getAttestationChain: (tenantId: string, taskId: string) =>
    get(
      `/v1/tenants/${tenantId}/tasks/${taskId}/attestation-chain`,
      AttestationChainSchema,
    ),

  verifyAttestation: (rekorUuid: string) =>
    post(
      `/v1/attestations/${encodeURIComponent(rekorUuid)}/verify`,
      {},
      z.object({
        verified: z.boolean(),
        details: z.object({
          inclusion_proof_valid: z.boolean(),
          cert_chain_valid: z.boolean(),
          subject_digest_matches: z.boolean(),
          self_hosted: z.boolean(),
        }),
      }),
    ),

  listWebhooks: (tenantId: string) =>
    get(
      `/v1/tenants/${tenantId}/webhooks/subscriptions`,
      z.object({
        subscriptions: z.array(
          z.object({
            id: z.string(),
            url: z.string(),
            events: z.array(z.string()),
            active: z.boolean(),
            created_at: z.string(),
            last_delivery_at: z.string().nullable().optional(),
            last_delivery_status: z.string().nullable().optional(),
          }),
        ),
      }),
    ),

  createWebhook: (
    tenantId: string,
    body: { url: string; events: string[]; description?: string },
  ) =>
    post(
      `/v1/tenants/${tenantId}/webhooks/subscriptions`,
      body,
      z.object({ id: z.string(), signing_secret: z.string() }),
    ),

  getTenantConfig: (tenantId: string) =>
    get(
      `/v1/tenants/${tenantId}/config`,
      z.object({
        model_overrides: z.record(z.string()),
        retry_caps: z.object({ per_step: z.number(), per_task: z.number() }),
        dollar_budget_caps: z.object({ per_task: z.number(), per_day: z.number() }),
        critical_path_weights: z.record(z.number()),
        promotion_policy_overrides: z.string().optional(),
      }),
    ),

  setTenantConfig: (tenantId: string, body: unknown) =>
    post(`/v1/tenants/${tenantId}/config`, body, z.object({ ok: z.literal(true) })),

  getCostRollup: (tenantId: string, opts: { groupBy: "day" | "repo" | "dev"; days: number }) =>
    get(
      `/v1/tenants/${tenantId}/cost/rollup?` +
        new URLSearchParams({ group_by: opts.groupBy, days: String(opts.days) }),
      z.object({
        series: z.array(
          z.object({
            key: z.string(),
            label: z.string(),
            cost_usd: z.number(),
            tasks: z.number().int(),
            tokens_input: z.number().int(),
            tokens_output: z.number().int(),
            cache_hit_rate: z.number(),
          }),
        ),
      }),
    ),

  getSloStatus: (tenantId: string) =>
    get(
      `/v1/tenants/${tenantId}/slo`,
      z.object({
        slos: z.array(
          z.object({
            name: z.string(),
            description: z.string(),
            objective: z.number(),
            actual_30d: z.number(),
            window: z.string(),
            status: z.enum(["healthy", "burning", "violated"]),
            burn_rate: z.number(),
          }),
        ),
      }),
    ),

  redeliverWebhook: (tenantId: string, subscriptionId: string, eventId: string) =>
    post(
      `/v1/tenants/${tenantId}/webhooks/subscriptions/${subscriptionId}/redeliver`,
      { event_ids: [eventId] },
      z.object({ ok: z.literal(true) }),
    ),
};
