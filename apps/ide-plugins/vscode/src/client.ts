// HTTP client for the Crucible control plane.
//
// Thin: every method maps 1:1 to a control-plane route. Auth handled by the
// `Auth` module — bearer JWT injected into every request.

import * as vscode from "vscode";
import type { Auth } from "./auth";
import { EventSourceParserStream } from "eventsource-parser/stream";

export type TaskSummary = {
  id: string;
  description: string;
  status: string;
  repo: string;
  cost_usd: number;
  submitted_at: string;
};

export type TaskDetail = TaskSummary & {
  plan?: {
    description: string;
    estimated_cost_usd: number;
    estimated_duration_min: number;
    files_to_touch: string[];
    db_migrations: number;
    external_effects: { service: string; endpoints: string[]; live: boolean }[];
    top_risks: { description: string; impact: "low" | "med" | "high" }[];
    retry_budget_per_step: number;
    wall_clock_budget_min: number;
    hard_cap_usd: number;
  };
};

export type AttestationDetail = {
  rekor_uuid: string;
  predicate_type: string;
  subject: { name: string; digest: Record<string, string> };
  signed_at: string;
  signed_by_oidc: string;
  validation: "valid" | "invalid" | "pending";
  predicate: unknown;
  self_hosted: boolean;
};

export class CrucibleClient {
  constructor(private auth: Auth) {}

  private endpoint(): string {
    return vscode.workspace.getConfiguration("crucible").get<string>("apiEndpoint") ?? "http://localhost:8080";
  }

  private tenantId(): string {
    return (
      vscode.workspace.getConfiguration("crucible").get<string>("tenantId") ?? this.auth.tenantId() ?? "ten_demo"
    );
  }

  private async req<T>(method: string, path: string, body?: unknown): Promise<T> {
    const token = await this.auth.token();
    const res = await fetch(`${this.endpoint()}${path}`, {
      method,
      headers: {
        "Content-Type": "application/json",
        Accept: "application/json",
        ...(token ? { Authorization: `Bearer ${token}` } : {}),
      },
      body: body ? JSON.stringify(body) : undefined,
    });
    if (!res.ok) throw new Error(`Crucible ${method} ${path}: ${res.status} ${await res.text()}`);
    return (await res.json()) as T;
  }

  async detectRepoFromWorkspace(): Promise<string> {
    const folders = vscode.workspace.workspaceFolders;
    if (!folders || folders.length === 0) return "unknown/unknown";
    const name = folders[0].uri.path.split("/").slice(-2).join("/");
    return `github.com/${name}`;
  }

  async submitTask(input: { description: string; repo: string; baseSha?: string }): Promise<TaskSummary> {
    return this.req("POST", `/v1/tenants/${this.tenantId()}/tasks`, {
      description: input.description,
      repo: input.repo,
      base_sha: input.baseSha,
      submitted_from: "vscode",
    });
  }

  async listPendingTasks(): Promise<TaskSummary[]> {
    const out = await this.req<{ tasks: TaskSummary[] }>("GET", `/v1/tenants/${this.tenantId()}/tasks?status=plan_pending_approval`);
    return out.tasks;
  }

  async listTasks(): Promise<TaskSummary[]> {
    const out = await this.req<{ tasks: TaskSummary[] }>("GET", `/v1/tenants/${this.tenantId()}/tasks?limit=50`);
    return out.tasks;
  }

  async getTask(id: string): Promise<TaskDetail> {
    return this.req("GET", `/v1/tenants/${this.tenantId()}/tasks/${id}`);
  }

  async approvePlan(id: string, edits?: { hard_cap_usd?: number; retry_budget_per_step?: number }) {
    return this.req("POST", `/v1/tenants/${this.tenantId()}/tasks/${id}/plan/approve`, { edits: edits ?? null });
  }

  async rejectPlan(id: string, reason: string) {
    return this.req("POST", `/v1/tenants/${this.tenantId()}/tasks/${id}/plan/reject`, { reason });
  }

  async interrupt(id: string, reason: string) {
    return this.req("POST", `/v1/tenants/${this.tenantId()}/tasks/${id}/interrupt`, { reason });
  }

  async getAttestation(uuid: string): Promise<AttestationDetail> {
    return this.req("GET", `/v1/attestations/${encodeURIComponent(uuid)}`);
  }

  async listRecentAttestations(): Promise<{ rekor_uuid: string; predicate_type: string; signed_at: string }[]> {
    const out = await this.req<{ attestations: { rekor_uuid: string; predicate_type: string; signed_at: string }[] }>(
      "GET",
      `/v1/tenants/${this.tenantId()}/attestations?limit=20`,
    );
    return out.attestations;
  }

  async getBudgetSnapshot(): Promise<{ spent_today_usd: number; cap_today_usd: number; tasks_today: number }> {
    return this.req("GET", `/v1/tenants/${this.tenantId()}/budget/snapshot`);
  }

  async streamTaskEvents(
    id: string,
    onEvent: (e: { event: string; data: unknown }) => void,
    signal: AbortSignal,
  ) {
    const token = await this.auth.token();
    const res = await fetch(`${this.endpoint()}/v1/tenants/${this.tenantId()}/tasks/${id}/events`, {
      headers: {
        Accept: "text/event-stream",
        ...(token ? { Authorization: `Bearer ${token}` } : {}),
      },
      signal,
    });
    if (!res.body) throw new Error("SSE: no body");
    const stream = res.body.pipeThrough(new TextDecoderStream()).pipeThrough(new EventSourceParserStream());
    const reader = stream.getReader();
    while (true) {
      const { done, value } = await reader.read();
      if (done) break;
      try {
        const data = JSON.parse(value.data);
        onEvent({ event: value.event ?? "message", data });
      } catch {
        // bad frame; the parser already isolated us at the boundary
      }
    }
  }
}
