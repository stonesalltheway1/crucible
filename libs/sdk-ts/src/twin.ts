// Crucible Agent SDK — TypeScript binding for the agent-side `twin.*`
// runtime API.
//
// Phase 2 ships the typed surface; the gRPC transport is wired against the
// Rust runtime-server in apps/twin-runtime. Upstream unit tests should use
// `StubClient`.

import type {
  Diff,
  ExecResult,
  SecretRef,
  SourceRef,
  Task,
  Budget,
  DestructiveProposal,
  ConventionStatus,
  MemoryKind,
  ScopeFilter,
} from "./types.js";

/** Memory is a single retrieved item from twin.memory.recall. */
export interface Memory {
  id: string;
  content: string;
  importance: number;
  written_at: string;
  last_recalled: string;
  kind: MemoryKind;
  source?: SourceRef;
}

/** Convention is the procedural-memory unit (Phase 5).
 *  Mirrors libs/twin-spec/proto/crucible/v1/memory.proto:Convention. */
export interface Convention {
  id: string;
  tenant_id: string;
  scope: ScopeFilter;
  rule_nl: string;
  category: string;
  status: ConventionStatus;
  confidence: number;
  judge_score?: number;
  source_evidence?: SourceRef[];
  valid_from: string;
  valid_to?: string;
  supersedes?: string;
  writer_oidc_subject?: string;
  written_at: string;
}

export interface ComplianceReportViolation {
  convention_id: string;
  rule_nl: string;
  offending_file: string;
  offending_line?: number;
  snippet?: string;
  severity: "info" | "warn" | "error";
}

export interface ComplianceReport {
  diff_hash: string;
  violations: ComplianceReportViolation[];
  conventions_checked: number;
  generated_at: string;
}

export interface ClientConfig {
  /** Runtime control socket (unix or vsock URI). */
  endpoint: string;
  /** Task id this client is bound to. */
  taskId: string;
  /** Heartbeat interval in ms. */
  heartbeatIntervalMs?: number;
}

export type ShellOutcome =
  | { kind: "result"; result: ExecResult }
  | { kind: "proposal"; proposal: DestructiveProposal };

export interface WriteAttestation {
  attestationId: string;
  contentSha256: string;
}

export interface SvcCallRequest {
  service: string;
  endpoint: string;
  method?: string;
  headers?: Record<string, string>;
  body?: Uint8Array;
}

export interface SvcCallResponse {
  status: number;
  headers: Record<string, string>;
  body: Uint8Array;
  /** Value of the X-Crucible-Tape header — `hit-exact` etc. */
  tapeDisposition: string;
}

/** The TwinClient surface. */
export interface TwinClient {
  fsRead(path: string): Promise<string>;
  fsWrite(path: string, content: string, stepId?: string): Promise<WriteAttestation>;
  fsDelete(path: string, stepId?: string): Promise<{ attestationId?: string; proposal?: DestructiveProposal }>;
  fsList(glob: string): Promise<string[]>;
  fsDiff(): Promise<Diff>;
  shellExec(cmd: string, opts?: { cwd?: string; env?: Record<string, string>; timeoutSec?: number }): Promise<ShellOutcome>;
  shellApproveDestructive(proposalId: string, justification: string): Promise<ExecResult>;
  memoryRecall(query: string, maxTokens?: number): Promise<Memory[]>;
  memoryNote(fact: string, source: SourceRef): Promise<string>;
  memoryConventions(scope: ScopeFilter): Promise<Convention[]>;
  memoryCheckCompliance(diff: Diff): Promise<ComplianceReport>;
  planCheckBudget(): Promise<Budget>;
  planRequestReplan(reason: string): Promise<Task>;
  secretGet(name: string): Promise<SecretRef>;
  secretList(): Promise<string[]>;
  svcCall(req: SvcCallRequest): Promise<SvcCallResponse>;
  heartbeat(): Promise<void>;
  close(): Promise<void>;
}

/** Build a client. The TypeScript transport awaits Phase 2's integration
 *  tests against the Rust runtime-server. For unit tests, use `stubClient()`. */
export function newClient(cfg: ClientConfig): TwinClient {
  if (!cfg.taskId) {
    throw new Error("ClientConfig.taskId required");
  }
  return grpcClient(cfg);
}

const STUB_MSG =
  "STUB: full gRPC TS client surface is wired in Phase 2 integration tests against the Rust runtime-server. " +
  "Use stubClient() for unit tests. See PHASE-2-REPORT.md.";

function grpcClient(cfg: ClientConfig): TwinClient {
  const todo = async (): Promise<never> => {
    throw new Error(STUB_MSG);
  };
  return {
    async fsRead() { return todo(); },
    async fsWrite() { return todo(); },
    async fsDelete() { return todo(); },
    async fsList() { return todo(); },
    async fsDiff() { return todo(); },
    async shellExec() { return todo(); },
    async shellApproveDestructive() { return todo(); },
    async memoryRecall() { return todo(); },
    async memoryNote() { return todo(); },
    async memoryConventions() { return todo(); },
    async memoryCheckCompliance() { return todo(); },
    async planCheckBudget() { return todo(); },
    async planRequestReplan() { return todo(); },
    async secretGet() { return todo(); },
    async secretList() { return todo(); },
    async svcCall() { return todo(); },
    async heartbeat() { return todo(); },
    async close() { /* no-op */ },
  };
}

/** In-memory stub for unit tests. Records calls; never reaches a real runtime. */
export function stubClient(taskId: string): TwinClient {
  const files = new Map<string, string>();
  return {
    async fsRead(path) {
      const v = files.get(path);
      if (v === undefined) throw new Error(`file not found: ${path}`);
      return v;
    },
    async fsWrite(path, content) {
      files.set(path, content);
      return { attestationId: `stub:${path}`, contentSha256: "" };
    },
    async fsDelete(path) {
      files.delete(path);
      return { attestationId: `stub-delete:${path}` };
    },
    async fsList() {
      return Array.from(files.keys());
    },
    async fsDiff() {
      return { files: Array.from(files, ([path, content]) => ({ path, action: "modify" as const, content })) } as Diff;
    },
    async shellExec(cmd) {
      return { kind: "result", result: { stdout: `[stub] ${cmd}`, stderr: "", exitCode: 0, durationMs: 0, signedAttestation: "" } as ExecResult };
    },
    async shellApproveDestructive() {
      return { stdout: "", stderr: "", exitCode: 0, durationMs: 0, signedAttestation: "" } as ExecResult;
    },
    async memoryRecall() { return []; },
    async memoryNote() { return "mem_stub"; },
    async memoryConventions() { return []; },
    async memoryCheckCompliance() {
      return {
        diff_hash: "",
        violations: [],
        conventions_checked: 0,
        generated_at: new Date().toISOString(),
      };
    },
    async planCheckBudget() { return {} as Budget; },
    async planRequestReplan() { return {} as Task; },
    async secretGet(name) {
      return { name, handle: `stub-handle:${name}`, expiresAt: new Date(Date.now() + 60_000) } as unknown as SecretRef;
    },
    async secretList() { return []; },
    async svcCall() {
      return { status: 200, headers: { "X-Crucible-Tape": "hit-exact" }, body: new Uint8Array(), tapeDisposition: "hit-exact" };
    },
    async heartbeat() {},
    async close() {},
  };
}
