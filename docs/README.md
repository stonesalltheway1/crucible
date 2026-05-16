# Crucible

> The AI engineer that tests every change in a digital twin before touching your real code.

Crucible is a coding agent positioned against Cursor/Windsurf/Devin/Antigravity on the **trust and verifiability** axis, not the autonomy/speed axis. Every change runs in a faithful ephemeral mirror of the user's project — twin filesystem, twin database, twin services, twin secrets — and is independently verified by a *different-family* model before promotion to real systems.

## What this is

This directory is the full design + architecture + operational documentation for Crucible. It's structured so a fresh agent (or human) can pick up any layer of the system and have enough context to build, extend, or operate it without re-asking the conversation.

## How to read it

| If you want... | Start at |
|---|---|
| The product pitch and why it exists | [00-vision/product-vision.md](00-vision/product-vision.md) |
| How the system fits together | [01-architecture/system-overview.md](01-architecture/system-overview.md) |
| What to build first | [07-roadmap/v1-mvp.md](07-roadmap/v1-mvp.md) and [07-roadmap/build-plan-agent-days.md](07-roadmap/build-plan-agent-days.md) |
| How to call the agent | [03-sdk/agent-sdk-reference.md](03-sdk/agent-sdk-reference.md) |
| The reasoning behind a specific choice | [05-decisions/](05-decisions/) |
| Resolved deep-dive research questions | [06-research/](06-research/) |
| Sources and citations | [ASSETS.md](ASSETS.md) |

## The core thesis in 90 seconds

The 2025–26 generation of coding agents is fast but structurally untrustworthy. Public disasters (PocketOS's 9-second prod-DB wipe in April 2026, Uber's full-year Claude Code budget burned in four months, Replit Agent ignoring a code freeze) and the universal user complaints (memory amnesia, runaway costs, destructive shell commands, hallucinated APIs, fake-test-pass claims, infinite explore loops) all share one root cause: **agents act directly on real systems with no architectural separation between "try" and "commit."**

Crucible fixes this by making the digital twin the *primary* execution surface. The agent gets unlimited freedom to experiment because it cannot reach production. Only verified changes — verified by a separate model from a different lineage, plus a tiered ladder of property tests, fuzz, contract checks, and (for `@critical` paths) formal proofs — are promoted via a signed gate that requires HSM-backed approval for destructive operations.

The compounding moat is a per-tenant procedural-memory graph that learns from every PR review comment, incident post-mortem, and ADR — so the agent's day-90 output reflects the team's actual conventions, not Tailwind defaults.

## Status

Design-stage. No code yet. Documentation reflects the v0 design decisions that should hold through the first ~20 agent-days of build.

## Calibration

Build estimates throughout these docs are quoted in **agent-days** at a rate of ~10–20K LoC/day of working code. A v1 Crucible is roughly **19 agent-days, ~315K LoC** — about three calendar weeks of focused agent work. See [07-roadmap/build-plan-agent-days.md](07-roadmap/build-plan-agent-days.md) for the breakdown.

## Conventions used in these docs

- **Concrete > abstract.** Named technologies, version numbers, pricing per million tokens. If a doc says "we pick X," there's a reason linked.
- **Opinions are signed.** Every ADR (`05-decisions/`) makes a single decision with stated alternatives and consequences.
- **Cross-references via relative paths.** No external link rot inside the design.
- **Sources separately.** External citations live in [ASSETS.md](ASSETS.md) at the root so doc text stays clean.
