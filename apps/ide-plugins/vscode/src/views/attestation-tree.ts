import * as vscode from "vscode";
import type { CrucibleClient } from "../client";

export class AttestationTreeProvider implements vscode.TreeDataProvider<AttNode> {
  private _onDidChangeTreeData = new vscode.EventEmitter<AttNode | undefined | void>();
  readonly onDidChangeTreeData = this._onDidChangeTreeData.event;

  constructor(private client: CrucibleClient) {}

  refresh() {
    this._onDidChangeTreeData.fire();
  }

  getTreeItem(el: AttNode): vscode.TreeItem {
    return el;
  }

  async getChildren(): Promise<AttNode[]> {
    try {
      const atts = await this.client.listRecentAttestations();
      return atts.map((a) => new AttNode(a.rekor_uuid, a.predicate_type, a.signed_at));
    } catch {
      return [];
    }
  }
}

class AttNode extends vscode.TreeItem {
  constructor(uuid: string, predicateType: string, signedAt: string) {
    super(predicateType, vscode.TreeItemCollapsibleState.None);
    this.id = uuid;
    this.description = uuid.length > 22 ? `${uuid.slice(0, 16)}…${uuid.slice(-6)}` : uuid;
    this.tooltip = `${predicateType}\n${uuid}\nsigned ${signedAt}`;
    this.command = { command: "crucible.viewAttestation", title: "View", arguments: [uuid] };
    this.iconPath = new vscode.ThemeIcon("shield");
  }
}
