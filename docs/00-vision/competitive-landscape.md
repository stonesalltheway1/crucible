# Competitive Landscape (May 2026)

A snapshot of every major coding-agent product as of the design date, with table-stakes / differentiators / bleeding-edge framing. The exhaustive per-product detail is in [ASSETS.md](../ASSETS.md). This doc focuses on the *gaps* that justify Crucible.

## The agents we benchmark against

**IDE-resident agents:** Cursor, Windsurf (Cognition-owned post-Dec 2025), Cline, Continue.dev, Zed AI, JetBrains Junie, Trae, Cody/Amp.

**Standalone / cloud agents:** Devin, Claude Code, Codex CLI + cloud, GitHub Copilot Workspace + Spark, Google Antigravity, Replit Agent 3.

**Full-stack chat builders:** Bolt.new, v0, Lovable, Base44.

**Enterprise / specialized:** Tabnine (air-gapped), Aider (OSS CLI), Codestral, GitHub Copilot Coding Agent.

## Table-stakes (everyone has, by early 2026)

- Multi-model routing (Claude/GPT/Gemini at minimum).
- MCP support — Anthropic donated MCP to Linux Foundation December 2025, now universal.
- Chat + inline edit + autocomplete + agent mode.
- File/folder/@-mention context.
- Terminal execution with approval gates.
- Git integration (auto-commits or PR-based).
- Codebase indexing/RAG.
- Plan-then-execute mode separation.
- Some form of rules/memory file (`.cursorrules`, `CLAUDE.md`, `AGENTS.md`).

If we ship without these, we're not in the conversation.

## Differentiators that already exist somewhere

| Feature | Owner | Why it matters |
|---|---|---|
| Bespoke Tab/edit prediction model | Cursor (Tab), Windsurf (SWE-1.5) | UX speed perception |
| Multi-agent parallelism with merge | Cursor 2.0, Antigravity Manager, Zed Parallel Agents | Throughput on hard tasks |
| In-browser dev environment | Bolt (WebContainer) | Zero-install onboarding |
| One-click full-stack provisioning | Lovable+Supabase, Base44, Spark | Non-tech founder wedge |
| Self-improving PR bot | Cursor BugBot | Compounds over time |
| Air-gapped enterprise deploy | Tabnine, Sourcegraph | Regulated buyers |
| Skills / Hooks / Subagents as primitives | Claude Code | Agent extensibility |
| Voice input | Trae SOLO, Cursor optional | Hands-free workflow |
| Auto-generated repo wiki | Devin Wiki | Onboarding ergonomics |
| CI-enforceable AI rules | Continue.dev | Team consistency |
| Native multiplayer human+AI | Zed | Pair programming |
| Agent Client Protocol | Zed/ACP | Cross-editor agent portability |

**Implication:** any of these we want, we adopt. Most are now well-paved cowpaths. The actual moat must be elsewhere.

## Bleeding edge (announced 2025–26, not yet widespread)

- **Long-horizon autonomy** — Replit Agent 3 (200-min runs), Devin parallel cloud IDEs, Cursor Background Agents.
- **Computer use** as a default loop (Antigravity, Trae SOLO, Replit self-test, Cursor BG).
- **Persistent cross-session memory** (Cursor Memories, Antigravity learning primitive, Claude Code Skills).
- **Manager/dashboard for agent fleets** (Antigravity Manager, Cursor 2.0 multi-agent view, Devin).
- **In-house orchestration models** (Cursor Composer-2, Windsurf SWE-1.5).
- **Spec-driven development** (GitHub Spark, Trae SOLO PRD→deploy).
- **Plug-in / skill marketplaces** (Cline MCP Marketplace, Claude Code plugins, Cursor MCP store).
- **Usage-based credit pricing** replacing flat seats (Replit, Windsurf, GitHub June 2026, v0, Cursor).
- **Cross-agent interop standard** — Agent Client Protocol (Zed) likely the next MCP-style standard.
- **Auto-evolving repo wiki** (Devin Wiki, Antigravity knowledge artifacts).

**Implication:** most of these are 6–18 months out from full saturation. We can ship without leading on any individual one, as long as we own the trust dimension.

## What no one has nailed (the white space)

1. **Trust by construction.** Every agent above edits real files, hits real services, uses real credentials. The PocketOS 9-second wipe is the inevitable consequence. No incumbent has decoupled "try" from "commit" at the architecture level.

2. **Verifiable completion.** Every agent above marks tasks done on its own say-so. "Tests passed" is the same agent that wrote the tests grading itself. Cross-family adversarial verification is unclaimed.

3. **Honest cost transparency.** Every agent above shows token spend *after* the fact. Plan-time previews of "$0.42, 3 minutes, 4 files, top risk: webhook signature verification" exist nowhere.

4. **Procedural memory from PR review comments.** Cursor Memories and Claude Code Skills are user-written. Mining PR review comments and post-mortems to build a *learned* team-conventions graph is unbuilt.

5. **Signed, replayable provenance.** No agent emits in-toto SLSA-L3 attestations by default. Compliance buyers cannot procure today's tools without bolting on their own audit layer.

6. **Native legacy-codebase modernization.** Every leader is greenfield-optimized. The 500K-line Rails 4 monolith / COBOL payments / Java EE estate market is open.

7. **Truly cross-IDE agent identity.** An agent that follows you from VS Code → JetBrains → terminal with shared memory is unbuilt; ACP gestures at it.

8. **Verifiable correctness on critical paths.** Formal-methods integration (Dafny/Lean/TLA+) as a default for `@critical` code is unclaimed — DafnyPro POPL 2026 made this technically tractable but no product has shipped it.

Crucible targets #1, #2, #3, #4, #5, and #8 directly. #6 falls out as a natural specialization. #7 is solved indirectly by integrating via ACP rather than building our own IDE.

## Specific incumbents to position against

### vs Cursor

Cursor is the volume leader and the speed-perception leader. Direct head-to-head on Tab autocomplete or Composer edits is a losing game. **Crucible's positioning vs Cursor:**

- "Cursor is your sprint pace. Crucible is your release manager."
- Cursor is greenfield-and-prototype-optimized; we're production-and-correctness-optimized.
- The demo is side-by-side on a deliberately-destructive scenario: ask both to "clean up unused database tables." Cursor will `DROP`. Crucible will route to a typed `DestructiveProposal` with a blast-radius preview.

### vs Devin

Devin and Crucible share the "autonomous, long-running, verified" framing — Devin is the closest philosophical competitor. **Where Crucible wins:**

- Devin's ACU pricing is opaque; "verified PR" is auditable.
- Devin verifies internally (same model lineage); Crucible verifies cross-family.
- Devin owns its own IDE; Crucible plugs into the user's existing one via MCP/ACP.
- Devin has no formal-methods integration; Crucible escalates Tier 3 on `@critical` paths.
- Devin is cloud-only; Crucible has day-one self-hosted/air-gapped.

### vs Claude Code

Claude Code is the power-user CLI standard. **Crucible is what Claude Code becomes when you wire skills + subagents + hooks into a coherent product, plus the twin runtime.** We are not anti-Claude-Code — Crucible can *include* Claude Code as one of its primary executor models. The positioning is "Claude Code with guardrails and verification, productized for teams."

### vs Antigravity

Google's bet. Strong on the manager-view UI and the agent-fleet dashboard. Brand-new in November 2025, sparse third-party tooling. **Crucible is more conservative on UX, more aggressive on trust.** The Antigravity manager paradigm is worth borrowing for our team console; the trust gap is wide open.

### vs Tabnine

The closest match on the air-gap + enterprise + privacy axis. Tabnine has weak agent loop, weak verification, weaker brand among engineers. **Crucible is Tabnine's enterprise positioning with a 2026-grade agent loop and cross-family verifier.**

### vs Aider / Continue.dev / Cline

These are the OSS-aligned, BYO-key, power-user tools. **Crucible is what they become when productized for teams** — same philosophical alignment (transparency, BYO-key option, plugin marketplaces), with the twin runtime, verifier, and procedural memory as the value-add. We should be friendly to this community, not competitive: open-source our verifier harness, our cartographer, and our PII-scrub pipeline as evangelism tools.

## Pricing landscape (May 2026)

Detailed per-agent pricing comparison is in [pricing-and-business.md](pricing-and-business.md). Top-line takeaways:

- The market has bifurcated: seat-only (Tabnine, JetBrains) is collapsing; the dominant model is seat + included credit pool + on-demand burst (Cursor, GitHub June 2026, Codex).
- Devin's ACU = 15 min compute at $2.00–$2.25 is the closest precedent to a verified-PR outcome unit.
- Sierra ($0.99–$1.50/resolved conversation), Intercom Fin ($0.99/resolution), Zendesk ($1.50–$2.00) prove outcome pricing works in adjacent markets.
- No coding-agent vendor has shipped outcome pricing yet — first-mover opportunity.

## Strategic implication

Compete where senior engineers already hate the alternatives. Don't compete on Tab autocomplete speed or marketing-page rounded-corner aesthetic. Compete on the dimensions every incumbent has structurally ceded:

1. Trust by architectural construction (twin runtime).
2. Verified completion (cross-family verifier, four-tier ladder).
3. Compounding team memory (PR-mined procedural graph).
4. Cryptographic provenance (Sigstore Rekor by default).
5. Air-gap from day one.
6. Outcome-priced.

Every one of these is unsolved in the incumbents. Owning all six simultaneously is the wedge.
