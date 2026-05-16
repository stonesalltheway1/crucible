You are starting Phase 7 of building Crucible. The backend is functionally
complete: agent submits tasks (P1), executes in isolated twins (P2-3),
ships verified work (P4), learns conventions (P5), promotes via signed
gates (P6). Phase 7 builds the SURFACES users actually touch.

This is Block 7 from the build plan (3 agent-days, ~50K LoC originally).
We compress to a ~25K LoC session by leveraging shadcn/Tremor for the web
console and the MCP/ACP standards for IDE integration (no custom forks).

CALIBRATION
===========
Phase 7 targets ~25K LoC. Most of this is web UI + thin IDE plugins. The work
is well-paved (Next.js + shadcn for the console; MCP/ACP for IDEs). Quality
bar emphasizes UX polish on the plan-approval / cost-preview surfaces because
those are the customer-visible trust narrative.

READ FIRST
==========
1. docs/PHASE-6-REPORT.md
2. memory/project_crucible_phase6.md
3. docs/05-decisions/ADR-011-no-built-in-ide.md         — we plug into existing IDEs
4. docs/03-sdk/tool-reference.md                        — MCP tool surface
5. docs/04-operations/onboarding.md                     — what the UI must support
6. docs/03-sdk/event-spec.md                            — webhook events the UI listens to
7. docs/02-engineering/observability.md                 — the four KPI dashboards
8. docs/02-engineering/tech-stack.md (frontend section)
9. docs/00-vision/target-users.md                       — what each persona does in the UI
10. docs/07-roadmap/build-plan-agent-days.md (Block 7)

RESEARCH BEFORE CODING (parallel)
=================================
1. Next.js — current major version + App Router conventions; React Server
   Components default patterns; `use client` boundary best practices in 2026.

2. shadcn/ui — current component library + Radix primitives; theming system;
   the install pattern (copy-paste, no npm dep).

3. Tremor — current dashboarding components; integration with Recharts.

4. zod + react-hook-form — current form-validation idioms.

5. Clerk (SaaS auth) — current SDK + tenant scoping.

6. WorkOS (enterprise SSO) — SAML + OIDC flow + tenant assignment.

7. MCP — Linux Foundation hosting status; latest protocol version; reference
   implementations across hosts (Claude Desktop, Cursor, Zed, etc.).

8. ACP (Agent Client Protocol) — Zed's spec; tool surface mapping from MCP.

9. VS Code Extension API — current contribution points for chat panels +
   webview + status bar; required for the plan-approval UX.

10. JetBrains Platform — current Junie / AI Assistant plugin patterns; Project
    Open Telemetry (POT) integration.

11. GitHub App framework — current best practices; webhooks for PR-comment-
    based invocation (`/crucible <description>`).

12. Slack Bolt SDK — current Go/TS SDK; block-kit for the approval button UI.

PHASE 7 SCOPE
=============

EXPLICITLY IN SCOPE
-------------------
1. apps/web-console/ — Next.js + App Router + shadcn:
   Pages:
   - / (landing/tenant-overview)
   - /tasks — task timeline with cost, duration, verifier verdict per task
   - /tasks/[id] — task detail with plan, steps, attestation chain explorer
   - /tasks/[id]/approve — plan-approval modal with cost preview, risk
     callouts, retry-budget config (the customer-trust signature surface)
   - /promotions — pending approval inbox + recent promotion history
   - /promotions/[id] — promotion detail with Rego decision + canary status
   - /memory — convention browser, drift reviewer, supersession history
   - /memory/conventions/[id] — convention detail with positive/negative
     examples + source references
   - /attestations — Rekor UUID search + verification UI
   - /attestations/[uuid] — full attestation content viewer
   - /cost — per-task / per-dev / per-repo cost dashboard (Tremor charts)
   - /slo — public-style status page per docs/02-engineering/observability.md
   - /settings — tenant config (model overrides, retry caps, dollar budgets,
     critical-path classifier weights, promotion policy editor)
   - /webhooks — webhook subscription management
   Components:
   - shadcn primitives + Tremor for charts
   - Server-Sent Events for streaming plan/verifier progress
   - WebSocket only where bi-directional needed
   Auth:
   - Clerk for SaaS tenants
   - WorkOS for enterprise SSO + SAML
   - Authelia/Dex as the self-host option

2. apps/ide-plugins/vscode/ — VS Code extension (~3K LoC):
   - Plan approval modal in webview
   - Budget viewer in status bar
   - Attestation chain explorer panel
   - MCP host client wiring to the Crucible MCP server
   - Commands: Crucible: New Task, Crucible: Approve Plan, etc.

3. apps/ide-plugins/jetbrains/ — JetBrains plugin (~3K LoC):
   - Same affordances as VS Code, idiomatic to JetBrains UI
   - Junie-style toolwindow integration

4. apps/ide-plugins/zed/ — Zed extension via ACP (~2K LoC):
   - Same affordances; ACP routing
   - Lighter-weight (Zed is itself an AI-native editor; we integrate, not duplicate)

5. apps/cli/ — flesh out from Phase 1 minimal:
   - task new / show / approve / monitor / cancel
   - plan show / approve / reject / amend
   - promote / status / rollback
   - memory recall / note / conventions / drift-review
   - attestation get / verify / chain / export
   - webhook create / list / redeliver
   - tenant config get / set
   - verify-release <version> (the public Tier 4 customer-side command)
   - calibrate (the per-tenant critical-path classifier weight fit)
   - All subcommands have --output json for scripting

6. apps/github-app/ — Go + GitHub App framework:
   - PR-comment-driven invocation: `/crucible <description>`
   - Auto-open PR for agent-authored verified bundles
   - Verifier report rendered as PR comment
   - Attestation chain linked in PR description
   - Approval buttons (defers to Phase 6's promotion gate)

7. apps/slack-bot/ — flesh out from Phase 6 minimal:
   - Channel-level promotion approval requests with block-kit
   - Approver button → Sigstore keyless OIDC signing
   - Slash-command task submission `/crucible <description>`
   - DM-based task status notifications
   - Approval routing matches tenant config

8. The plan-approval UX (the differentiating surface):
   - Pre-execution cost preview: "$0.42, ~3 minutes, 4 files, top risk:
     webhook signature verification"
   - Hard-cap visualization: "budget cap: $2.00 (current spend: $0)"
   - Retry-budget visualization: "3 retries per subgoal, halt-and-ask after"
   - Approve / edit / reject buttons
   - One-click "approve and walk away" for trusted task types
   - Live progress as the task executes (SSE)
   - Mid-task interrupt button (halt cleanly at next checkpoint)

9. The attestation viewer (the trust narrative surface):
   - Rekor UUID-based search
   - Full chain visualization for any task: plan → file writes → tool calls →
     verifier verdict → promotion approval → outcome
   - Per-attestation: predicate type, OIDC subject, signed-at, validation
     status, content
   - "Reproduce this attestation" button — local re-verification
   - Public-share link (signed URL) for compliance auditors

10. The memory browser:
    - Per-scope convention list with confidence sliders
    - Drift indicator + last-violated date
    - Source-evidence inspector (clickable PR comments, ADRs, incidents)
    - Override editor with diff vs default
    - Status lifecycle controls (active → drifting → superseded)

11. Branding + design system:
    - Custom Tailwind theme override (NOT default rounded-corners-blue-gradients
      per ADR-001 brand voice — "anti-vibe-coding aesthetic")
    - Monospace-heavy typography for code surfaces
    - Subtle / professional color palette
    - The brand voice is "evidence-driven engineering," not "AI magic"

12. Tests:
    - E2E via Playwright on the web console (key user flows: submit task →
      approve plan → view verification → approve promotion → view attestation)
    - Component tests via Vitest + React Testing Library
    - VS Code extension integration test (vscode-test framework)
    - GitHub App webhook signature verification tests
    - Slack bot block-kit rendering tests

13. Docs updates:
    - docs/02-engineering/local-dev.md — Phase 7 additions (web-console dev
      server, IDE plugin sideload, GitHub App ngrok setup)
    - CHANGELOG.md → 2026.06.0-phase7
    - User-facing docs site bootstrap (Mintlify or similar)

EXPLICITLY OUT OF SCOPE (defer to v2 or Phase 8)
------------------------------------------------
- Mobile app for approvals (web console + Slack cover it; v2 if signal)
- Visual diff editor for agent's proposed code (the IDE's diff is the diff)
- Tab autocomplete (ADR-011: not our competition surface)
- Composer-style multi-file rewrite UI (not our surface)
- Voice input (v2 if signal)
- IDE chat panel (the IDE has chat; we don't duplicate)
- Public marketing website (separate repo per repo-structure.md)
- Premium themes / multi-theme support (v2)

WORKING AGREEMENTS
==================
- TypeScript everywhere on the frontend.
- Next.js App Router default; RSC for data, `use client` only at boundaries
  that need interactivity.
- shadcn/ui + Radix primitives; do NOT write custom-from-scratch components
  when shadcn ships one.
- Tremor for analytics dashboards.
- zod for form validation and API contract enforcement at runtime.
- biome for lint + format.
- pnpm for the monorepo (root workspace; per-app package).
- IDE plugins published to their respective marketplaces from CI.
- Per ADR-011: we do NOT fork an IDE. We plug into existing ones.

QUALITY BAR
===========
- Web console Lighthouse score ≥ 95 on key pages (performance + accessibility).
- Plan-approval modal renders in < 200ms from event receipt.
- Attestation viewer can verify a chain end-to-end without backend round-trips
  beyond initial fetch.
- Mutation score on web-console business logic ≥ 80% (UI tooling is weaker
  than backend; 80% is the practical bar).
- IDE plugins: each one passes its own integration test.
- E2E Playwright suite green across all key flows.
- Hermetic Nix builds — verify the Next.js build is deterministic
  (`prefer-deterministic-bundling` etc. set).

PROGRESS TRACKING
=================
  1. Read docs + PHASE-6-REPORT
  2. Currency-check research (12 streams parallel)
  3. apps/web-console scaffolding (Next.js + shadcn + Clerk/WorkOS)
  4. Plan-approval modal + budget viewer (the differentiating surface)
  5. Task dashboard + task detail
  6. Promotion approval inbox + canary visualization
  7. Memory browser + drift reviewer
  8. Attestation viewer + chain explorer
  9. Cost dashboard + SLO status
  10. Settings page + tenant config editor
  11. apps/ide-plugins/vscode
  12. apps/ide-plugins/jetbrains
  13. apps/ide-plugins/zed (ACP)
  14. apps/cli full surface
  15. apps/github-app (PR-comment invocation)
  16. apps/slack-bot full surface
  17. E2E + integration tests
  18. Docs + report

END-OF-SESSION REPORT
=====================
docs/PHASE-7-REPORT.md:

1. File tree + LoC
2. Demo URLs (dev deployment links if applicable)
3. Lighthouse scores per key page
4. IDE plugin install instructions
5. GitHub App / Slack bot install instructions
6. Stubs + deferred items
7. The Phase 8 prompt (onboarding + v1 launch — template at
   docs/08-phase-prompts/phase-08-onboarding-and-v1-launch.md)

Update memory: project_crucible_phase7.md.

GUARDRAILS
==========
- Do NOT default to Tailwind blue + rounded corners. Brand voice is anti-vibe;
  the UI reflects that.
- Do NOT add tracking/analytics that send customer code or task content to
  third parties. Plausible/PostHog-style page-event analytics OK; agent-task
  content stays in-tenant.
- Do NOT embed customer prod credentials into the UI. The UI surfaces task
  events and attestation references; never secret values.
- Do NOT replicate the IDE's chat panel or completion UX. We integrate, we
  don't compete.
- Do NOT skip accessibility. shadcn ships Radix which is a11y-correct;
  preserve that.
- Do NOT ship the IDE plugins or GitHub App with permissions broader than
  needed. Scope to repo:read + PR:write + workflow:read.
- Do NOT enable webhook subscriptions without signature verification.

The UX is where the customer perceives the trust narrative. Build for the
senior engineer who is going to scrutinize "what did this agent actually do
to my codebase, and how do I verify it?" Every surface should answer that.

Begin.
