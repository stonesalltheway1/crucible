# Crucible — Zed extension (ACP)

Zed-native integration via the **Agent Client Protocol** (Zed's open
extension spec for agent hosts). Lightweight because Zed is itself an
AI-native editor — we integrate, we do not duplicate.

Per **ADR-011**: integrate, don't compete.

## What ships

- **Slash commands**: `/crucible <description>`, `/crucible-approve`, `/crucible-halt`
- **ACP bridge** to the Crucible MCP server (`crucible-mcp`) so Zed's agent
  panel can call `twin_*` tools natively
- **Tool surface**: file / db / svc / shell / test / verifier / memory / plan / promote — see `acp-bridge.toml` for the MCP→ACP name mapping

## Install

```bash
cargo build --release --target wasm32-wasi
# Zed → Settings → Extensions → "Install from local directory"
```

## Auth

OAuth + PKCE against `https://api.crucible.dev`. Token kept by Zed's
credential store; the wasm sandbox never sees it. Network access is
mediated by Zed's extension permission grants.

## Plan approval

The plan-approval UX lives in the Crucible web console (which the
extension opens via `/crucible <description>` returning a link). Inside
Zed, the user reviews progress through the standard agent panel; the
attestation chain is available via the `view-attestation` ACP tool.
