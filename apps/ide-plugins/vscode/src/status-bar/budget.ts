import * as vscode from "vscode";
import type { CrucibleClient } from "../client";

// Status bar item that surfaces the day's spend vs cap. Updates every 30s
// and on demand when the user runs a Crucible command.
export class BudgetStatusBar implements vscode.Disposable {
  private item: vscode.StatusBarItem;
  private timer?: NodeJS.Timeout;

  constructor(private client: CrucibleClient) {
    this.item = vscode.window.createStatusBarItem(vscode.StatusBarAlignment.Right, 50);
    this.item.command = "crucible.openWebConsole";
    this.item.tooltip = "Crucible — open the web console";
    this.item.text = "$(shield) Crucible · —";
    this.item.show();
    void this.refresh();
    this.timer = setInterval(() => void this.refresh(), 30_000);
  }

  async refresh() {
    try {
      const s = await this.client.getBudgetSnapshot();
      const pct = Math.min(100, Math.round((s.spent_today_usd / Math.max(s.cap_today_usd, 0.01)) * 100));
      this.item.text = `$(shield) Crucible · $${s.spent_today_usd.toFixed(2)} / $${s.cap_today_usd.toFixed(0)} (${pct}%)`;
      this.item.backgroundColor =
        pct >= 90
          ? new vscode.ThemeColor("statusBarItem.errorBackground")
          : pct >= 75
            ? new vscode.ThemeColor("statusBarItem.warningBackground")
            : undefined;
    } catch {
      this.item.text = "$(shield) Crucible · offline";
      this.item.backgroundColor = undefined;
    }
  }

  dispose() {
    if (this.timer) clearInterval(this.timer);
    this.item.dispose();
  }
}
