// Crucible VS Code extension entry point.
//
// We are NOT trying to be a chat panel, an autocomplete provider, or a
// multi-file rewrite UI. The IDE already has those. We surface:
//
//   1. The plan-approval modal (webview).
//   2. The budget viewer (status bar).
//   3. The attestation chain explorer (custom view).
//   4. Commands that drive the Crucible MCP server.
//
// Per ADR-011: we plug into the IDE, we do not fork it.

import * as vscode from "vscode";
import { CrucibleClient } from "./client";
import { TaskTreeProvider } from "./views/tasks-tree";
import { AttestationTreeProvider } from "./views/attestation-tree";
import { PlanApprovalPanel } from "./webview/plan-approval-panel";
import { BudgetStatusBar } from "./status-bar/budget";
import { McpHost } from "./mcp/host";
import { Auth } from "./auth";

export async function activate(context: vscode.ExtensionContext) {
  const auth = new Auth(context);
  const client = new CrucibleClient(auth);
  const mcp = new McpHost(context);

  const tasksProvider = new TaskTreeProvider(client);
  const attestationProvider = new AttestationTreeProvider(client);
  vscode.window.registerTreeDataProvider("crucible.tasks", tasksProvider);
  vscode.window.registerTreeDataProvider("crucible.attestations", attestationProvider);

  const budget = new BudgetStatusBar(client);
  context.subscriptions.push(budget);

  context.subscriptions.push(
    vscode.commands.registerCommand("crucible.signIn", () => auth.signIn()),
    vscode.commands.registerCommand("crucible.signOut", () => auth.signOut()),

    vscode.commands.registerCommand("crucible.newTask", async () => {
      const description = await vscode.window.showInputBox({
        title: "Crucible — describe the task",
        placeHolder: "e.g. add idempotency-key support to /webhooks/stripe/refund",
        ignoreFocusOut: true,
      });
      if (!description) return;
      const repo = await client.detectRepoFromWorkspace();
      const task = await client.submitTask({ description, repo });
      tasksProvider.refresh();
      PlanApprovalPanel.show(context, client, task.id);
    }),

    vscode.commands.registerCommand("crucible.openTask", async (taskId?: string) => {
      const id =
        taskId ??
        (await vscode.window.showInputBox({ title: "Crucible — task ID", prompt: "task_..." }));
      if (id) PlanApprovalPanel.show(context, client, id);
    }),

    vscode.commands.registerCommand("crucible.approvePlan", async (taskId?: string) => {
      const id = taskId ?? (await pickPendingTask(client));
      if (!id) return;
      const ok = await vscode.window.showWarningMessage(
        "Approve plan? The agent will sign your approval and begin execution.",
        { modal: true },
        "Approve",
      );
      if (ok === "Approve") {
        await client.approvePlan(id);
        vscode.window.showInformationMessage("Plan approved — Crucible is executing.");
        tasksProvider.refresh();
      }
    }),

    vscode.commands.registerCommand("crucible.rejectPlan", async (taskId?: string) => {
      const id = taskId ?? (await pickPendingTask(client));
      if (!id) return;
      const reason = await vscode.window.showInputBox({ title: "Reason", placeHolder: "why reject?" });
      if (reason) {
        await client.rejectPlan(id, reason);
        tasksProvider.refresh();
      }
    }),

    vscode.commands.registerCommand("crucible.interruptTask", async (taskId?: string) => {
      const id = taskId ?? (await pickPendingTask(client));
      if (!id) return;
      const ok = await vscode.window.showWarningMessage(
        "Halt the agent at the next checkpoint? The current step finishes; nothing partial is merged.",
        { modal: true },
        "Halt",
      );
      if (ok === "Halt") {
        await client.interrupt(id, "user halt from VS Code");
        vscode.window.showInformationMessage("Halt requested.");
      }
    }),

    vscode.commands.registerCommand("crucible.viewAttestation", async (uuid?: string) => {
      const u =
        uuid ??
        (await vscode.window.showInputBox({ title: "Rekor UUID", placeHolder: "rekor:..." }));
      if (!u) return;
      const a = await client.getAttestation(u);
      const doc = await vscode.workspace.openTextDocument({
        content: JSON.stringify(a, null, 2),
        language: "json",
      });
      vscode.window.showTextDocument(doc, { preview: true });
    }),

    vscode.commands.registerCommand("crucible.openWebConsole", () => {
      const endpoint = vscode.workspace.getConfiguration("crucible").get<string>("apiEndpoint") ?? "";
      const url = endpoint.replace("api.", "app.");
      vscode.env.openExternal(vscode.Uri.parse(url));
    }),
  );

  // Spawn the MCP server as a child process for hosts (Claude Desktop, Cursor's
  // MCP integration, etc.) running alongside VS Code. The host-side process is
  // long-lived; the extension owns its lifecycle.
  await mcp.start();
  context.subscriptions.push({ dispose: () => mcp.stop() });
}

async function pickPendingTask(client: CrucibleClient): Promise<string | undefined> {
  const tasks = await client.listPendingTasks();
  if (tasks.length === 0) {
    vscode.window.showInformationMessage("No pending tasks.");
    return undefined;
  }
  const pick = await vscode.window.showQuickPick(
    tasks.map((t) => ({ label: t.description, description: t.id })),
    { title: "Select a task" },
  );
  return pick?.description;
}

export function deactivate() {
  // VS Code disposes registered subscriptions automatically.
}
