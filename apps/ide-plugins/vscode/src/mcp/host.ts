// MCP host bridge.
//
// The VS Code extension's interaction with the Crucible MCP server is mainly
// supervisory — we don't drive tool calls from inside the extension. Instead
// we ensure the MCP server process is running so that whatever LLM host the
// user runs (Claude Desktop, Cursor, etc.) can reach it.

import * as vscode from "vscode";
import { spawn, type ChildProcess } from "node:child_process";

export class McpHost {
  private proc?: ChildProcess;

  constructor(private context: vscode.ExtensionContext) {}

  async start() {
    const cmd =
      vscode.workspace.getConfiguration("crucible").get<string>("mcpServerCmd") ?? "crucible-mcp";
    try {
      this.proc = spawn(cmd, [], {
        env: { ...process.env, CRUCIBLE_FROM: "vscode" },
        stdio: ["ignore", "pipe", "pipe"],
      });
      this.proc.on("exit", (code, signal) => {
        if (code !== 0 && code !== null) {
          vscode.window.showWarningMessage(
            `Crucible MCP server exited (${code}/${signal}). Restart with "Developer: Reload Window".`,
          );
        }
      });
      this.proc.stdout?.on("data", (b) => this.context.workspaceState.update("crucible.mcp.last", b.toString()));
    } catch (e) {
      vscode.window.showErrorMessage(`Failed to start MCP server: ${(e as Error).message}`);
    }
  }

  stop() {
    if (this.proc && !this.proc.killed) {
      this.proc.kill("SIGTERM");
    }
  }
}
