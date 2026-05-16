# Crucible — VS Code extension

The Crucible UX surfaces inside VS Code: plan approval, budget viewer,
attestation chain explorer, MCP server wiring.

Per **ADR-011**, this extension does not replicate the IDE's chat panel or
completion UX. It surfaces the differentiated Crucible affordances and gets
out of the way.

## What ships

| Surface | Where | What it does |
|---|---|---|
| Activity-bar view | left rail | Live list of tasks and recent attestations |
| Status bar | bottom-right | "Crucible · $0.42 / $150 (0.3%)" — day's spend vs cap |
| Plan-approval webview | side panel | Cost preview, files, risks, retry budget, approve / reject / halt |
| Commands | command palette | `Crucible: New Task`, `Approve Plan`, `View Attestation`, `Open Web Console`, sign-in / sign-out |
| MCP server bridge | background | Spawns `crucible-mcp` so co-located LLM hosts (Claude Desktop, Cursor) can reach Crucible's tool surface |

## Auth

PKCE-based OAuth against the Crucible API. Tokens stored in VS Code's
encrypted secret storage. No tokens in `settings.json` or workspace files.

## Permissions

The extension's `package.json` declares only what it needs:
- Access to the workspace folder name (for repo detection).
- Network access to `crucible.apiEndpoint` (default `https://api.crucible.dev`).
- Secret storage (token cache).

No file watcher, no telemetry, no third-party service connections.

## Install (dev)

```bash
pnpm install
pnpm compile
# Press F5 in VS Code to launch a dev extension host
```

## Publish

CI runs `vsce package` and uploads the `.vsix` to the VS Code Marketplace
on every tagged release.
