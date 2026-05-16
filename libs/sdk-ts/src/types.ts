// Crucible SDK — TypeScript types
//
// Hand-rolled Phase-1 equivalents of the protobuf code that will be generated
// from libs/twin-spec/proto/crucible/v1/*.proto by `buf generate` in Phase 2.
// Names and JSON encoding match the Go types in libs/sdk-go/crucible/v1/.

export type Action = "add" | "modify" | "delete";

export type Complexity =
  | "trivial"
  | "standard"
  | "complex"
  | "critical"
  | "modernization";

export type Reversibility = "trivial" | "snapshot" | "lossy" | "irreversible";

export type TaskStatus =
  | "received"
  | "planning"
  | "awaiting_approval"
  | "approved"
  | "rejected"
  | "executing"
  | "verifying"
  | "promoting"
  | "landed"
  | "rolled_back"
  | "budget_exceeded"
  | "retry_limit_exceeded"
  | "wall_clock_exceeded"
  | "failed";

export type ConventionStatus = "active" | "drifting" | "superseded" | "rejected";

export type MemoryKind = "hot" | "episodic" | "semantic" | "procedural";

export type ErrorCode =
  | "BudgetExceeded"
  | "RetryLimitExceeded"
  | "WallClockExceeded"
  | "EgressDenied"
  | "SecretAccessDenied"
  | "DestructiveProposalRejected"
  | "TwinSetupError"
  | "TapeIntegrityError"
  | "VerifierRejection"
  | "PromotionPolicyDenied"
  | "ApprovalTimeout"
  | "CanaryRollback"
  | "TenantQuotaExceeded"
  | "ModelRoutingDenied";

export interface Glob {
  pattern: string;
}

export interface ScopeFilter {
  repo?: string;
  file_glob?: string;
  category?: string;
}

export type Scope = "all" | ScopeFilter;

export interface FileChange {
  path: string;
  action: Action;
  content?: string;
  content_sha256?: string;
  size_bytes?: number;
}

export interface Diff {
  files: FileChange[];
  base_sha?: string;
}

export interface SourceRefPrComment {
  kind: "pr_comment";
  pr: number;
  comment_id: string;
}
export interface SourceRefIncident {
  kind: "incident";
  id: string;
  service: string;
}
export interface SourceRefAdr {
  kind: "adr";
  path: string;
  commit: string;
}
export interface SourceRefAgentObservation {
  kind: "agent_observation";
  task_id: string;
  step_id: string;
}
export type SourceRef =
  | SourceRefPrComment
  | SourceRefIncident
  | SourceRefAdr
  | SourceRefAgentObservation;

export interface SecretRef {
  name: string;
  handle: string;
  expires_at?: string;
}

export interface ExecResult {
  stdout: string;
  stderr: string;
  exit_code: number;
  duration_ms: number;
  signed_attestation?: string;
}

export interface BlastRadius {
  affected_resources: string[];
  reversibility: Reversibility;
  impact_score: number;
}

export interface DestructiveProposal {
  task_id: string;
  tenant_id: string;
  command: string;
  scope: "twin" | "real";
  justification?: string;
  blast_radius: BlastRadius;
  justification_required: boolean;
  intercepted_at_layer: "syscall-shim" | "cmd-line-parse" | "ebpf";
  proposed_at: string;
  agent_oidc_subject: string;
}

export interface ExternalEffect {
  service: string;
  endpoints: string[];
  live: boolean;
}

export interface Risk {
  description: string;
  impact: "low" | "medium" | "high";
}

export interface PlanStep {
  ordinal: number;
  description: string;
  retry_budget: number;
  retries_used: number;
}

export interface Plan {
  task_id: string;
  description: string;
  steps: PlanStep[];
  estimated_cost_usd: number;
  estimated_duration_min: number;
  files_to_touch: string[];
  db_migrations: number;
  external_effects: ExternalEffect[];
  top_risks: Risk[];
  retry_budget_per_step: number;
  wall_clock_budget_min: number;
  complexity: Complexity;
  plan_hash: string;
  built_at: string;
}

export interface PlanApproval {
  task_id: string;
  plan_hash: string;
  approver_oidc_subject: string;
  approved_at: string;
  attestation_id?: string;
  cost_cap_usd: number;
  wall_clock_cap_min: number;
  retry_cap_per_subgoal: number;
}

export interface PlanRejection {
  task_id: string;
  plan_hash: string;
  reason: string;
  rejecter_oidc_subject: string;
  rejected_at: string;
}

export interface Routing {
  executor_model: string;
  executor_vendor: "anthropic" | "google" | "openai" | string;
  executor_tier: 0 | 1 | 2 | 3 | 4;
  verifier_model: string;
  verifier_vendor: string;
  verifier_tier: 0 | 1 | 2 | 3 | 4;
  critical_score: number;
  is_critical: boolean;
  decided_at: string;
  classifier_attestation_id?: string;
}

export interface Budget {
  spent_usd: number;
  cap_usd: number;
  steps_used: number;
  steps_cap: number;
  wall_clock_used_seconds: number;
  wall_clock_cap_seconds: number;
  retries_used: number;
  retry_cap: number;
}

export interface Task {
  id: string;
  tenant_id: string;
  repo: string;
  base_sha: string;
  description: string;
  status: TaskStatus;
  created_at: string;
  updated_at: string;
  submitted_by: string;
  plan?: Plan;
  routing?: Routing;
  budget?: Budget;
  related_task_ids?: string[];
}

export interface CrucibleError {
  code: ErrorCode;
  message: string;
  retryable: boolean;
  hint?: string;
  details?: unknown;
}

// Predicate-type URIs.
export const Predicates = {
  WriteAttestation: "https://crucible.dev/WriteAttestation/v1",
  MigrationAttestation: "https://crucible.dev/MigrationAttestation/v1",
  ServiceCallAttestation: "https://crucible.dev/ServiceCallAttestation/v1",
  DestructiveProposal: "https://crucible.dev/DestructiveProposal/v1",
  DestructiveApproval: "https://crucible.dev/DestructiveApproval/v1",
  TestReport: "https://crucible.dev/TestReport/v1",
  VerifierApproval: "https://crucible.dev/VerifierApproval/v1",
  VerifierRejection: "https://crucible.dev/VerifierRejection/v1",
  PlanProposal: "https://crucible.dev/PlanProposal/v1",
  PlanApproval: "https://crucible.dev/PlanApproval/v1",
  PromotionBundle: "https://crucible.dev/PromotionBundle/v1",
  PromotionApproval: "https://crucible.dev/PromotionApproval/v1",
  PromotionOutcome: "https://crucible.dev/PromotionOutcome/v1",
  MemoryWrite: "https://crucible.dev/MemoryWrite/v1",
} as const;
export type PredicateType = (typeof Predicates)[keyof typeof Predicates];
