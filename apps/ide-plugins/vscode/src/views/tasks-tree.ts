import * as vscode from "vscode";
import type { CrucibleClient, TaskSummary } from "../client";

export class TaskTreeProvider implements vscode.TreeDataProvider<TaskNode> {
  private _onDidChangeTreeData = new vscode.EventEmitter<TaskNode | undefined | void>();
  readonly onDidChangeTreeData = this._onDidChangeTreeData.event;

  constructor(private client: CrucibleClient) {}

  refresh() {
    this._onDidChangeTreeData.fire();
  }

  getTreeItem(el: TaskNode): vscode.TreeItem {
    return el;
  }

  async getChildren(): Promise<TaskNode[]> {
    try {
      const tasks = await this.client.listTasks();
      return tasks.map((t) => new TaskNode(t));
    } catch {
      return [];
    }
  }
}

class TaskNode extends vscode.TreeItem {
  constructor(public task: TaskSummary) {
    super(task.description, vscode.TreeItemCollapsibleState.None);
    this.id = task.id;
    this.tooltip = `${task.id} · ${task.status}`;
    this.description = `${statusGlyph(task.status)} $${task.cost_usd.toFixed(2)}`;
    this.contextValue = `task:${task.status}`;
    this.command =
      task.status === "plan_pending_approval"
        ? { command: "crucible.openTask", title: "Open", arguments: [task.id] }
        : { command: "crucible.viewAttestation", title: "View", arguments: [task.id] };
    this.iconPath = new vscode.ThemeIcon(iconFor(task.status));
  }
}

function statusGlyph(s: string): string {
  if (s === "plan_pending_approval") return "● review";
  if (s === "verified" || s === "completed" || s === "promoted") return "✓";
  if (s === "failed" || s === "verification_failed") return "✗";
  if (s === "executing" || s === "verifying") return "…";
  return "·";
}
function iconFor(s: string): string {
  if (s === "plan_pending_approval") return "circle-large-outline";
  if (s === "completed" || s === "verified" || s === "promoted") return "check";
  if (s === "failed" || s === "verification_failed") return "error";
  if (s === "executing" || s === "verifying") return "sync~spin";
  return "circle-small";
}
