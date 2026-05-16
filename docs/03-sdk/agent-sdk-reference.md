# Agent SDK Reference

The complete `twin.*` API surface. This is the only path through which an agent interacts with Crucible — there is no "raw exec" escape hatch. Every call emits an in-toto attestation, signed via the agent's keyless OIDC identity, published to Sigstore Rekor.

SDK languages: Go (`sdk-go`), TypeScript (`sdk-ts`), Python (`sdk-py`), Rust (`sdk-rs`). All generated from the same gRPC schema in `libs/twin-spec/`. Examples below are TypeScript flavored; the API shape is identical across languages.

## Common types

```typescript
type Path = string;
type FilePath = string;
type Glob = string;
type Diff = { files: FileChange[] };
type FileChange = { path: FilePath; action: "add" | "modify" | "delete"; content?: string };

type Attestation = {
  uuid: RekorUUID;
  predicate_type: string;
  subject: string;        // OIDC subject of signer
  signed_at: timestamp;
};

type Budget = {
  spent_usd: number;
  cap_usd: number;
  steps_used: number;
  steps_cap: number;
  wall_clock_used_seconds: number;
  wall_clock_cap_seconds: number;
};

type SourceRef =
  | { kind: "pr_comment"; pr: number; comment_id: string }
  | { kind: "incident"; id: string; service: string }
  | { kind: "adr"; path: string; commit: string }
  | { kind: "agent_observation"; task_id: string; step_id: string };

type Scope = { repo?: string; file_glob?: Glob; category?: string } | "all";
```

## Filesystem (`twin.fs`)

Reads and writes against the twin's overlayfs upper layer. The lower layer (the actual repo) is read-only; agent writes go to the upper.

```typescript
twin.fs.read(path: FilePath): Promise<FileContent>
```
Returns the file's current content from the twin. If the file hasn't been written in this task, returns the version from base_sha.

```typescript
twin.fs.write(path: FilePath, content: string): Promise<WriteAttestation>
```
Writes to the overlayfs upper. Returns an attestation. Never throws on "file doesn't exist" — writes are creates.

```typescript
twin.fs.delete(path: FilePath): Promise<DestructiveProposal | DeleteAttestation>
```
File deletion is a destructive op even in the twin. Returns `DestructiveProposal` if the file is in the critical-path set (auto-approved for twin scope); returns `DeleteAttestation` if approved.

```typescript
twin.fs.list(glob: Glob): Promise<FilePath[]>
```
Lists files matching the glob, merged view of overlayfs upper + lower.

```typescript
twin.fs.diff(): Promise<Diff>
```
Returns the cumulative diff of all writes in this task vs base_sha. Used at task end to construct the PromotionBundle.

## Database (`twin.db`)

Operates against the Neon CoW branch (or per-engine equivalent) provisioned for the task.

```typescript
twin.db.query(sql: string): Promise<QueryResult>
twin.db.queryParametrized(sql: string, params: any[]): Promise<QueryResult>
```
Executes against the twin DB. Returns rows + column metadata. Idempotent in twin: re-running the same query gives the same result until other writes occur.

```typescript
twin.db.migrate(file: FilePath): Promise<MigrationProposal | MigrationAttestation>
```
Applies a migration file to the twin branch. Returns a `MigrationProposal` containing schema diff + DML impact for the verifier to evaluate. Approved migrations return `MigrationAttestation`.

```typescript
twin.db.schemaDiff(): Promise<SchemaDiff>
```
Returns schema diff vs base. Useful for verifier checks ("did this migration touch a critical table?").

## Services (`twin.svc`)

Operates against Hoverfly replay tapes or, if explicitly allowed in the task manifest, live services through a scrubbing egress proxy.

```typescript
twin.svc.call(
  service: string,
  endpoint: string,
  payload?: any,
  options?: { method?: string; headers?: Headers }
): Promise<Response>
```
Returns a response. The response carries `X-Crucible-Tape: hit-exact | hit-template | synth-readonly | synth-mutation | live-passthrough | miss-blocked` so the agent can reason about response trustworthiness.

```typescript
twin.svc.listAvailable(): Promise<ServiceManifest[]>
```
Lists services configured for this task with their tape coverage, OpenAPI spec, and allowed methods.

```typescript
twin.svc.recordOnce(service: string, endpoint: string): Promise<RecordResult>
```
For development: capture a single live call to a service and add it to the tape. Requires the task manifest to allow this service.

## Secrets (`twin.secret`)

Accesses the twin-scoped Infisical vault. Values are never returned to the agent process; they're injected at the egress proxy when a request uses the secret.

```typescript
twin.secret.get(name: string): Promise<SecretRef>
```
Returns a typed handle, not the value. The handle is consumed by `twin.svc.call` via the `Authorization: $Bearer $secret(name)$` placeholder syntax.

```typescript
twin.secret.list(): Promise<string[]>
```
Lists names available in the twin's vault scope.

The agent **cannot** retrieve a secret's raw value. Attempting to does throw `SecretAccessDenied`. This is the architectural enforcement of secrets isolation.

## Shell (`twin.shell`)

Wrapped via the syscall shim. Destructive commands are intercepted and converted to typed `DestructiveProposal`.

```typescript
twin.shell.exec(cmd: string, options?: { cwd?: Path; env?: Record<string,string>; timeoutSec?: number }):
  Promise<ExecResult | DestructiveProposal>
```
Runs a command in the sandbox. Returns either:

- `ExecResult { stdout, stderr, exitCode, durationMs, signed_attestation }`, or
- `DestructiveProposal { command, blast_radius, justification_required: true }` if the command matches destructive patterns.

The agent must explicitly approve a `DestructiveProposal` to proceed:

```typescript
twin.shell.approveDestructive(proposal: DestructiveProposal, justification: string):
  Promise<ExecResult>
```

For twin-scoped destructives (operating on twin DB, twin FS, twin tapes), this auto-approves on the gate's side after the agent provides justification. Real-scoped destructives require human approval through the Promotion Contract.

## Tests (`twin.test`)

```typescript
twin.test.run(suite?: string, options?: { pattern?: string; timeout?: number }):
  Promise<TestReport>
```
Runs the project's test suite (or a subset). Auto-detects the test framework from the repo. Returns structured pass/fail per test + coverage if available.

```typescript
twin.test.runMutation(diff: Diff): Promise<MutationReport>
```
Runs mutation testing on the diff. Returns mutation score, killed mutants, survived mutants.

```typescript
twin.test.runProperty(spec: PBTSpec): Promise<PBTReport>
```
Runs property-based tests. `spec` declares the function under test, the input generators, and the invariants.

```typescript
twin.test.runFuzz(target: string, options?: { iterations?: number; corpus?: Path }):
  Promise<FuzzReport>
```
Runs the project's fuzz harness for `iterations` iterations.

## Verifier (`twin.verify`)

Invoked at the end of a task to compute the verifier's verdict for the Promotion Bundle. Each method delegates to the separate verifier process (different model family).

```typescript
twin.verify.tier0(diff: Diff): Promise<MutationReport>
twin.verify.tier1(spec: PBTSpec): Promise<PBTReport>
twin.verify.tier2(spec: ContractSpec): Promise<ContractReport>
twin.verify.tier3(spec: FormalSpec): Promise<ProofReport>
twin.verify.tier4(): Promise<HonestCIReport>
twin.verify.bundle(): Promise<VerifierApproval | VerifierRejection>
```

`twin.verify.bundle()` is the orchestration call — it runs the appropriate tiers based on the task's critical-path classification, and returns the final verdict for use in `twin.promote(...)`.

## Memory (`twin.memory`)

```typescript
twin.memory.recall(query: string, scope?: Scope): Promise<Memory[]>
```
Multi-signal retrieval against hot + episodic + procedural stores. Returns up to 7K tokens of relevant memory, importance-ranked.

```typescript
twin.memory.note(fact: string, source: SourceRef): Promise<MemoryId>
```
Explicit save — used when the agent learns something the background distiller would miss (e.g., a user correction in the current task).

```typescript
twin.memory.conventions(scope: Scope): Promise<Convention[]>
```
Returns active conventions for the given scope. Used at plan time and during verifier compliance check.

```typescript
twin.memory.checkCompliance(diff: Diff): Promise<ComplianceReport>
```
Compares a diff against the active conventions and returns violations. Called by the verifier during Tier 1+.

## Plan + budget (`twin.plan`)

```typescript
twin.plan.propose(plan: Plan): Promise<PlanApproval | PlanRejection>
```
Submits a plan for user approval. Blocks until user approves, rejects, or edits. Plan structure:

```typescript
type Plan = {
  description: string;
  steps: Step[];
  estimated_cost_usd: number;
  estimated_duration_min: number;
  files_to_touch: FilePath[];
  db_migrations: number;
  external_effects: ExternalEffect[];
  top_risks: Risk[];
  retry_budget_per_step: number;       // default 3
  wall_clock_budget_min: number;
};
```

```typescript
twin.plan.checkBudget(): Promise<Budget>
```
Returns current consumption vs cap. Agent should call this between steps; the Bounded Budget Enforcer also halts execution automatically if exceeded.

```typescript
twin.plan.checkpoint(name: string): Promise<Snapshot>
```
Saves a checkpoint of the twin state. The user can fork from any checkpoint via the web console.

```typescript
twin.plan.requestReplan(reason: string): Promise<PlanApproval>
```
Used when the agent realizes its plan is wrong. Halts current execution, surfaces to user, awaits new plan approval.

## Promotion (`twin.promote`)

```typescript
twin.promote(bundle: PromotionBundle): Promise<PromotionId>
```
Submits a verified `PromotionBundle` to the Promotion Contract. Blocks until policy evaluation completes; if human approval is required, the agent's call returns and the human is notified out-of-band. The agent can poll:

```typescript
twin.promote.status(id: PromotionId): Promise<PromotionStatus>
```
Status progresses: `pending_policy` → `pending_approval` → `approved` → `deploying` → `canary_dwell` → `landed` | `rolled_back` | `rejected`.

## Attestation (`twin.attest`)

```typescript
twin.attest(action: string, metadata?: any): Promise<RekorEntry>
```
Explicit attestation for actions the SDK doesn't auto-attest. Most uses are covered by auto-attestation in other methods; this is the escape hatch.

```typescript
twin.attest.verify(uuid: RekorUUID): Promise<AttestationContent>
```
Verifies and fetches an attestation. Used by verifier and promotion-gate.

## Error contract

Every method can throw structured errors:

```typescript
class CrucibleError extends Error {
  code: ErrorCode;
  retryable: boolean;
  hint?: string;
}

enum ErrorCode {
  BudgetExceeded,
  RetryLimitExceeded,
  WallClockExceeded,
  EgressDenied,
  SecretAccessDenied,
  DestructiveProposalRejected,
  TwinSetupError,
  TapeIntegrityError,
  VerifierRejection,
  PromotionPolicyDenied,
  ApprovalTimeout,
  CanaryRollback,
  TenantQuotaExceeded,
  ModelRoutingDenied,
  // ...
}
```

Agents should handle errors by class:

- **Retryable** (network blips, transient model errors): retry with backoff.
- **Budget / quota**: halt and surface to user via `requestReplan`.
- **Denial** (security policy, destructive proposals): pivot strategy; do not retry the same.

## Lifecycle example

```typescript
const plan = await twin.plan.propose({
  description: "Add Stripe webhook handler for refund events",
  steps: [
    "Read existing webhook handler structure",
    "Author handler + idempotency key check",
    "Author migration for refunds table",
    "Author tests + property tests",
    "Run verifier",
  ],
  estimated_cost_usd: 1.20,
  estimated_duration_min: 8,
  files_to_touch: ["api/webhooks.ts", "db/migrations/20260515_refunds.sql", "test/webhooks.test.ts"],
  db_migrations: 1,
  external_effects: [{ service: "stripe", endpoints: ["/webhooks/refund"], live: false }],
  top_risks: [
    { description: "Webhook signature verification", impact: "high" },
    { description: "Idempotency key collision", impact: "medium" },
  ],
  retry_budget_per_step: 3,
  wall_clock_budget_min: 15,
});

if (plan.status !== "approved") return;

// Execute step 1...
const existing = await twin.fs.read("api/webhooks.ts");
const conventions = await twin.memory.conventions({ file_glob: "api/**/*.ts" });

// Execute step 2...
const newCode = generate(existing, conventions);
await twin.fs.write("api/webhooks.ts", newCode);

// Execute step 3...
await twin.db.migrate("db/migrations/20260515_refunds.sql");

// Execute step 4...
await twin.fs.write("test/webhooks.test.ts", testCode);
const testReport = await twin.test.run();
if (!testReport.passed) {
  // reflect, fix, retry (Bounded Budget Enforcer caps at 3 retries)
}

// Verify...
const verdict = await twin.verify.bundle();
if (verdict.kind !== "approved") {
  // surface rejection_reasons to the agent's reasoning, retry once
}

// Promote...
const bundle = await constructBundle(verdict);
const promotionId = await twin.promote(bundle);

// Wait or return...
const status = await twin.promote.status(promotionId);
```

## What's deliberately not in the SDK

- **A "raw exec" method** that bypasses the syscall shim. There is no such thing.
- **A way to access real production credentials.** The agent process is architecturally unable to.
- **A way to disable the verifier.** Tasks complete only with a verifier verdict.
- **A way to bypass attestation.** Every method auto-emits.
- **Long-lived state (beyond the task).** Persistence is the Memory Layer's job, accessed through `twin.memory.*`.

See [03-sdk/tool-reference.md](tool-reference.md) for the MCP tool definitions exposed to LLM-tool-calling agents, [03-sdk/event-spec.md](event-spec.md) for webhook payloads, and [03-sdk/attestation-formats.md](attestation-formats.md) for the in-toto schemas.
