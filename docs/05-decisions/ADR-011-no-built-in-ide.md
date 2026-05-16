# ADR-011: Crucible integrates with existing IDEs via MCP/ACP; no proprietary IDE

**Status:** Accepted  
**Date:** 2026-05-15

## Context

Cursor, Windsurf, Antigravity, and Trae all built their own VS Code forks as the primary interface. The advantages: tight UX integration, brand surface, direct user telemetry. The costs: ongoing fork maintenance, upstream divergence, user-side switching friction.

MCP (Model Context Protocol) standardized in late 2024 and was donated to Linux Foundation in December 2025 — now universally adopted. ACP (Agent Client Protocol, from Zed) is gaining traction as the cross-editor agent-portability standard.

## Decision

Crucible **does not ship its own IDE.** Integration is via:

1. **MCP** — Crucible exposes its `twin.*` SDK as MCP tools. Any MCP-compatible host (Cursor, Claude Desktop, Zed via ACP, etc.) can drive a Crucible task.
2. **ACP** — for Zed and future ACP-compatible editors, same tool surface natively.
3. **CLI** — `crucible` standalone binary for non-IDE workflows (CI, scripts, Slack, GitHub Actions).
4. **Web console** — for plan approval, task monitoring, memory browsing, attestation viewing.
5. **GitHub App** — for PR-comment-driven invocation (`/crucible <description>`).

We ship thin plugins for the major IDEs that surface Crucible's specific UX needs (plan-approval modal, budget viewer, attestation chain explorer):

- VS Code extension
- JetBrains plugin
- Zed extension (uses ACP)

These plugins are ~3K LoC each — they wrap the MCP/ACP integration and add Crucible-specific UI affordances. They do **not** fork the IDE.

## Consequences

### Positive

- **No fork-maintenance tax.** We don't track VS Code upstream; we're a plugin.
- **Customer adoption friction near zero.** "Install this extension" beats "switch your editor."
- **IDE-agnostic value.** Customer's preference between VS Code, JetBrains, Zed, or terminal-only is preserved.
- **MCP momentum carries us.** Every new MCP-compatible host inherits Crucible support for free.
- **Smaller engineering surface.** Three plugins × 3K LoC = ~9K LoC vs. an entire IDE fork's ~150K LoC.

### Negative

- **No tight UX integration.** Cursor's Tab autocomplete is a Cursor-specific moat; we can't compete on that surface.
- **Less brand surface.** Users associate the work with their IDE, not with Crucible. Mitigation: plan-approval modal + cost preview prominently branded; attestation chain explorer is a Crucible-specific affordance.
- **MCP/ACP feature lag.** When MCP gains a new feature, we wait for hosts to adopt before we can use it.
- **Less direct user telemetry.** We see what the agent does, not what the user did before invoking the agent. Mitigation: track agent-side context (task description, files in context manifest, approval timing).

### Trade-offs we accept

We give up the Tab-autocomplete-style "spend hours in our IDE" mindshare to Cursor. We win by being the agent-of-record for verified deliverables — the thing that makes the merged PR matter, not the thing that helps you type faster.

## Alternatives considered

### Alternative 1: Fork VS Code

**Rejected**:

- ~150K LoC of ongoing fork maintenance.
- Customer switching cost.
- Cursor / Windsurf / Antigravity already crowd this space.
- Our brand is "verified output," not "fastest editing."

### Alternative 2: Build a web-only IDE

Bolt.new / v0 / Lovable model — entire dev environment in browser. **Rejected**:

- Different ICP (vibe-coders, not senior engineers).
- Browser dev envs hit JavaScript-only limits (WebContainer).
- Our ICP wants to keep their existing setup.

### Alternative 3: Terminal-only (CLI as primary interface)

Claude Code / Aider model. **Considered**; CLI is *one* of our surfaces but not the only one:

- Senior engineers love CLI; junior teammates and approvers don't.
- The plan-approval flow benefits from rich UI; CLI alone is awkward.
- Slack / GitHub / web console serve approval roles CLI can't.

### Alternative 4: Build a chat-only web interface

Devin / Replit Agent model — chat with the agent in browser. **Rejected as primary**:

- Replicates the existing chat-with-LLM surface; no differentiation.
- Pulls users out of their existing tools — switching cost.
- Web console exists as a complement, not the primary surface.

## What ships at v1

- VS Code extension (~3K LoC) — plan approval, budget viewer, attestation chain explorer, MCP wiring.
- Zed extension via ACP (~2K LoC) — same affordances.
- JetBrains plugin (~3K LoC) — same affordances.
- `crucible` CLI (~15K LoC across Go binary) — task submit/monitor, attestation verify, memory browse, runbook helpers.
- Web console (Next.js, ~50K LoC) — plan approval, task timeline, cost dashboard, memory browser, attestation viewer, approval inbox.
- GitHub App + Slack bot (~10K LoC) — PR-comment invocation, approval routing.

## What does NOT ship at v1

- Tab-autocomplete-quality inline completion.
- Cursor-Composer-style multi-file rewrite UI.
- Built-in chat panel (the IDE's chat panel is the chat panel).

These are not in scope. Our differentiation is verification + memory + provenance, not editing speed.

## References

- [02-engineering/repo-structure.md](../02-engineering/repo-structure.md)
- [03-sdk/tool-reference.md](../03-sdk/tool-reference.md)
