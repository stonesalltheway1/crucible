// OAuth + PKCE flow for the Crucible API.
//
// We use VS Code's built-in authentication providers when available, falling
// back to a device-code-style sign-in URL. Tokens are stored in the secret
// storage (encrypted by VS Code), never in workspace settings or files.

import * as vscode from "vscode";
import * as crypto from "node:crypto";

export class Auth {
  private static readonly SECRET_TOKEN = "crucible.token";
  private static readonly SECRET_REFRESH = "crucible.refresh";
  private static readonly STATE_TENANT = "crucible.tenant_id";

  constructor(private context: vscode.ExtensionContext) {}

  async signIn(): Promise<void> {
    const endpoint =
      vscode.workspace.getConfiguration("crucible").get<string>("apiEndpoint") ?? "https://api.crucible.dev";
    const codeVerifier = base64url(crypto.randomBytes(32));
    const codeChallenge = base64url(crypto.createHash("sha256").update(codeVerifier).digest());
    const state = base64url(crypto.randomBytes(16));
    const url = new URL(`${endpoint}/oauth/authorize`);
    url.searchParams.set("response_type", "code");
    url.searchParams.set("client_id", "crucible-vscode");
    url.searchParams.set("scope", "tenant:read tenant:write tasks:*");
    url.searchParams.set("code_challenge", codeChallenge);
    url.searchParams.set("code_challenge_method", "S256");
    url.searchParams.set("state", state);
    url.searchParams.set("redirect_uri", "vscode://crucible.crucible-vscode/auth-callback");

    await this.context.secrets.store("crucible.pkce_verifier", codeVerifier);
    await this.context.secrets.store("crucible.pkce_state", state);
    await vscode.env.openExternal(vscode.Uri.parse(url.toString()));
    vscode.window.showInformationMessage("Crucible sign-in opened in your browser. Approve and return here.");
  }

  async signOut(): Promise<void> {
    await this.context.secrets.delete(Auth.SECRET_TOKEN);
    await this.context.secrets.delete(Auth.SECRET_REFRESH);
    vscode.window.showInformationMessage("Signed out of Crucible.");
  }

  async token(): Promise<string | undefined> {
    return this.context.secrets.get(Auth.SECRET_TOKEN);
  }

  tenantId(): string | undefined {
    return this.context.globalState.get<string>(Auth.STATE_TENANT);
  }
}

function base64url(buf: Buffer): string {
  return buf.toString("base64").replace(/=+$/, "").replace(/\+/g, "-").replace(/\//g, "_");
}
