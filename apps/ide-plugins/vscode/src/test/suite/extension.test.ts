import * as assert from "node:assert";
import * as vscode from "vscode";

export async function runSuite() {
  // The activation event is `onStartupFinished`; by the time tests run, the
  // extension has activated. We verify the registered commands exist.
  const all = await vscode.commands.getCommands(true);
  const expected = [
    "crucible.newTask",
    "crucible.approvePlan",
    "crucible.rejectPlan",
    "crucible.interruptTask",
    "crucible.viewAttestation",
    "crucible.openWebConsole",
  ];
  for (const c of expected) {
    assert.ok(all.includes(c), `missing command: ${c}`);
  }
}
