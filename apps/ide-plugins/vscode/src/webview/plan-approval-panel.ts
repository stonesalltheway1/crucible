import * as vscode from "vscode";
import type { CrucibleClient } from "../client";

// The plan-approval webview. Mirrors the web console surface but lives inside
// VS Code. We render plain HTML + a CSP that disallows remote scripts; the
// webview talks back to the extension through `postMessage`.
export class PlanApprovalPanel {
  private static panels = new Map<string, PlanApprovalPanel>();

  static show(context: vscode.ExtensionContext, client: CrucibleClient, taskId: string) {
    const existing = PlanApprovalPanel.panels.get(taskId);
    if (existing) {
      existing.panel.reveal();
      return;
    }
    const panel = vscode.window.createWebviewPanel(
      "cruciblePlanApproval",
      "Crucible — review plan",
      vscode.ViewColumn.Beside,
      { enableScripts: true, retainContextWhenHidden: true },
    );
    PlanApprovalPanel.panels.set(taskId, new PlanApprovalPanel(panel, client, taskId, context));
  }

  private streamAbort?: AbortController;

  constructor(
    private panel: vscode.WebviewPanel,
    private client: CrucibleClient,
    private taskId: string,
    _context: vscode.ExtensionContext,
  ) {
    panel.webview.html = this.loading();
    void this.load();

    panel.webview.onDidReceiveMessage(async (msg) => {
      if (msg.type === "approve") {
        await client.approvePlan(taskId, msg.edits);
        await this.streamEvents();
      } else if (msg.type === "reject") {
        await client.rejectPlan(taskId, msg.reason ?? "rejected");
        panel.dispose();
      } else if (msg.type === "interrupt") {
        await client.interrupt(taskId, msg.reason ?? "user halt");
      }
    });

    panel.onDidDispose(() => {
      this.streamAbort?.abort();
      PlanApprovalPanel.panels.delete(taskId);
    });
  }

  private async load() {
    try {
      const task = await this.client.getTask(this.taskId);
      this.panel.webview.html = this.render(task);
    } catch (e) {
      this.panel.webview.html = this.error((e as Error).message);
    }
  }

  private async streamEvents() {
    this.streamAbort?.abort();
    const ctrl = new AbortController();
    this.streamAbort = ctrl;
    void this.client.streamTaskEvents(
      this.taskId,
      (e) => {
        this.panel.webview.postMessage({ type: "event", payload: e });
      },
      ctrl.signal,
    );
  }

  private loading() {
    return `<!doctype html><html><body><p style="font-family:ui-monospace;padding:1rem">loading plan…</p></body></html>`;
  }
  private error(msg: string) {
    return `<!doctype html><html><body><p style="font-family:ui-monospace;color:#a3231f;padding:1rem">${escape(msg)}</p></body></html>`;
  }

  private render(task: Awaited<ReturnType<CrucibleClient["getTask"]>>) {
    const plan = task.plan;
    if (!plan) return this.error("No plan available yet — try again in a moment.");
    const risks = plan.top_risks
      .map(
        (r) =>
          `<li><span class="pill pill-${r.impact}">${r.impact}</span> ${escape(r.description)}</li>`,
      )
      .join("");
    const files = plan.files_to_touch.map((f) => `<code>${escape(f)}</code>`).join(" ");
    return /* html */ `
<!doctype html><html><head><meta charset="utf-8">
<style>
  body { font-family: ui-sans-serif, system-ui; padding: 1rem; color: var(--vscode-foreground); background: var(--vscode-editor-background); }
  h1 { font-size: 1.1rem; margin: 0 0 0.5rem; }
  .meta { font-family: ui-monospace; font-size: 11px; color: var(--vscode-descriptionForeground); }
  .grid { display: grid; grid-template-columns: repeat(4, 1fr); gap: 0.5rem; margin: 1rem 0; }
  .stat { border: 1px solid var(--vscode-panel-border); padding: 0.5rem; }
  .stat .l { font-family: ui-monospace; font-size: 10px; text-transform: uppercase; color: var(--vscode-descriptionForeground); }
  .stat .v { font-size: 1.1rem; font-weight: 600; }
  .pill { display: inline-block; padding: 0 0.3rem; border: 1px solid currentColor; font-size: 10px; font-weight: 600; }
  .pill-high { color: #a3231f; } .pill-med { color: #b76b00; } .pill-low { color: #2f548a; }
  code { font-family: ui-monospace; background: var(--vscode-textBlockQuote-background); padding: 0 0.2rem; }
  ul { padding-left: 1rem; }
  .actions { display: flex; gap: 0.5rem; margin-top: 1rem; }
  button { font-family: inherit; padding: 0.4rem 0.8rem; border: 1px solid var(--vscode-button-border); cursor: pointer; }
  .approve { background: var(--vscode-button-background); color: var(--vscode-button-foreground); }
  .reject { background: transparent; color: #a3231f; border-color: #a3231f; }
  #events { margin-top: 1rem; font-family: ui-monospace; font-size: 11px; max-height: 200px; overflow: auto; }
  #events div { padding: 2px 0; border-bottom: 1px solid var(--vscode-panel-border); }
</style>
</head>
<body>
  <h1>${escape(task.description)}</h1>
  <div class="meta">${escape(task.repo)} · task <code>${escape(task.id)}</code></div>
  <p>${escape(plan.description)}</p>
  <div class="grid">
    <div class="stat"><div class="l">cost est.</div><div class="v">$${plan.estimated_cost_usd.toFixed(2)}</div></div>
    <div class="stat"><div class="l">duration est.</div><div class="v">~${plan.estimated_duration_min}m</div></div>
    <div class="stat"><div class="l">files</div><div class="v">${plan.files_to_touch.length}</div></div>
    <div class="stat"><div class="l">migrations</div><div class="v">${plan.db_migrations}</div></div>
  </div>
  <div><strong>Files:</strong> ${files}</div>
  <p><strong>Top risks</strong></p>
  <ul>${risks}</ul>
  <div class="actions">
    <button class="approve" onclick="approve()">Approve plan</button>
    <button class="reject" onclick="reject()">Reject</button>
    <button onclick="interrupt()">Halt</button>
  </div>
  <div id="events"></div>
  <script>
    const vscode = acquireVsCodeApi();
    function approve(){ vscode.postMessage({type:'approve'}); }
    function reject(){
      const reason = prompt('reason?');
      if (reason) vscode.postMessage({type:'reject', reason});
    }
    function interrupt(){ vscode.postMessage({type:'interrupt'}); }
    window.addEventListener('message', e => {
      if (e.data.type === 'event') {
        const el = document.getElementById('events');
        const d = document.createElement('div');
        d.textContent = e.data.payload.event + ' · ' + JSON.stringify(e.data.payload.data).slice(0,180);
        el.appendChild(d);
        el.scrollTop = el.scrollHeight;
      }
    });
  </script>
</body></html>`;
  }
}

function escape(s: string): string {
  return s.replace(/[&<>"']/g, (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" }[c]!));
}
