# Customer Onboarding

The "first week" experience that turns a sign-up into a verified, value-producing customer. We optimize for time-to-first-verified-PR.

## The four-stage journey

```
1. Install               (≤ 5 min)
2. Cartography           (≤ 30 min, automated; 1 agent-day of one-time work)
3. First verified PR     (same day)
4. Convention bootstrap  (rolling, daily)
```

## Stage 1 — Install

### SaaS install

1. Sign up at app.crucible.dev → tenant created.
2. Install the GitHub App on target repos.
3. (Optional) Connect Slack workspace for approval routing.
4. (Optional) Install IDE plugin (VS Code Marketplace, Zed via ACP, JetBrains, etc.).
5. Connect data sources for procedural memory bootstrap:
   - GitHub (required) — PR review comments, ADRs, merged code.
   - Linear / Jira (optional) — incident references.
   - Slack #incidents channel (optional) — post-mortem text.
   - Confluence / Notion (optional) — runbooks, ADR pages.

The GitHub App requests scoped permissions: repo read, PR write (for the agent's PRs), workflow read (for verifier results). No org-admin permissions needed.

### Self-hosted install

See [self-hosted-install.md](self-hosted-install.md) — Helm chart, KMS/HSM, observability stack. Hours of setup, but a one-time event. The product surface is identical to SaaS after install.

## Stage 2 — Cartography

The first task Crucible runs on every new repo is the **Cartographer**. This is a Crucible-internal task that:

1. Walks the repo, builds a tree-sitter symbol index.
2. Parses lint configs (`.editorconfig`, `.prettierrc`, `.eslintrc`, `.rubocop.yml`, `pyproject.toml`, `rustfmt.toml`, etc.) into deterministic Convention rules.
3. Reads `AGENTS.md`, `CLAUDE.md`, `CONTRIBUTING.md`, ADR directories.
4. Scans recent PR review comments (last 24 months, capped at top 1,000 by length).
5. Scans incident/post-mortem references found in PR descriptions.
6. Generates an inferred `AGENTS.md` (if one doesn't exist) and presents it for user review.
7. Bootstraps the per-tenant memory with the above + OSS-derived defaults (Tier A–D corpus per [06-research/memory-bootstrap.md](../06-research/memory-bootstrap.md)).

The Cartographer runs once per repo. It costs ~$3–$10 on a 50K-LoC repo, ~$30–$100 on a 500K-LoC repo. We absorb this cost — it's not billed against the customer's PR quota.

Output presented to the customer:

```
✓ Indexed 1,247 files across 38 directories.
✓ Detected stack: Next.js 14, FastAPI 0.110, PostgreSQL 16.
✓ Extracted 184 conventions from your existing config + AGENTS.md.
✓ Loaded 312 OSS-derived defaults for your stack.
✓ Inferred 47 additional conventions from your PR review history.
  - 12 high-confidence (recommended; surfaced active)
  - 23 medium-confidence (surfaced as suggestions)
  - 12 low-confidence (stored as candidates; not surfaced yet)

Review at: https://app.crucible.dev/memory
```

The customer can edit, accept, or reject any convention. Edits feed back into procedural memory as authoritative overrides.

## Stage 3 — First verified PR

The fast-path "wow moment" we optimize for.

1. Customer's senior engineer picks a small, real, in-progress task they've been putting off — usually:
   - A focused refactor ("replace `fetch` calls with our new API client").
   - A small feature ("add idempotency key to the webhook endpoint").
   - A bug fix from the team's bug tracker.

2. They invoke Crucible via their preferred surface:
   - IDE: open command palette, `Crucible: New Task`.
   - CLI: `crucible task new --repo acme/payments --description "..."`.
   - Web: paste the description into the dashboard.
   - GitHub: comment `/crucible <description>` on an issue.

3. Crucible spins up a twin, generates a plan, presents it. **Time-to-plan target: < 90 seconds.**

4. Customer approves the plan. Crucible executes. **Median execution: 5–15 minutes.**

5. Verifier runs cross-family. Returns approval or structured rejection.

6. If approved, Crucible opens a PR on the customer's repo with:
   - The diff.
   - The verifier's report (mutation score, PBT counterexamples (none), conventions applied, attestation chain).
   - A signed `PromotionBundle` linked.
   - Suggested rollout strategy.

7. Customer reviews and merges. Promotion contract executes (canary, dwell, land).

**The full first-task experience targets under 30 minutes from "first task submitted" to "verified PR merged."** Every minute longer materially reduces the conversion-to-paid rate.

## Stage 4 — Convention bootstrap (rolling)

After the first task, the distillation worker is consuming the customer's PR review activity in real time. Every PR that gets merged feeds:

- Positive examples for active conventions.
- Candidate conventions from comments that were addressed.
- Negative examples when reviewers correct violations.
- Anti-patterns from any post-mortems that reference the changed files.

After ~30 days of typical activity (assuming 5–10 merged PRs/week), the per-tenant memory has compounded enough to be **noticeably** better at applying team conventions than day-1.

The customer sees this in the web console:

```
Memory growth (past 30 days)
  +18 active conventions (from candidates)
  +5 conventions superseded
  +2 conventions flagged drifting (review required)
  
Convention compliance on agent PRs:
  Week 1: 91%
  Week 4: 97%
```

This compounding is the **stickiness mechanism**. Switching to another agent loses 30 days of learned taste.

## Onboarding touchpoints

### Week 1 — Setup

- Day 1: install + Cartographer + first verified PR.
- Day 2: customer success outreach to the senior-engineer champion. "Anything unexpected?"
- Day 3: dashboard walk-through (live or recorded).
- Day 5: weekly digest email starts (sent every Friday).

### Week 2 — Expand

- Customer adds 2–3 more repos.
- Customer invites the rest of the team (Team tier).
- We propose three "good first tasks" specific to their codebase based on the Cartographer's output.

### Week 4 — Review

- 30-day check-in. Show the compounding metrics.
- Convention drift review (one-touch acknowledgment of any drifting conventions).
- Pricing-tier conversation if usage indicates upgrade.

### Quarter 1 — Embedded

- Customer's team has Crucible PRs as a normal part of the workflow.
- We're an ambient capability, not a "tool we're trying."
- Renewal conversation initiated 60 days before renewal date.

## Self-serve vs assisted

### Self-serve (default for Pro tier)

The product onboards itself. Customer success monitors but only reaches out when:
- Time-to-first-verified-PR exceeds 24 hours.
- Customer drops out mid-Cartography.
- Customer hits a destructive-op gate (good opportunity to highlight the safety story).

### Assisted (default for Team / Outcome / Enterprise tiers)

A named contact at Crucible:
- Joins the install call.
- Reviews the Cartographer output with the customer.
- Selects the first task together.
- Reviews the first 5 verified PRs.

The assisted-onboarding budget is funded by the price differential between Pro and Team.

## Anti-patterns we explicitly avoid

- **Bigger demos before smaller value.** Don't do a 90-minute architecture walkthrough before they've seen one verified PR.
- **Synthetic demo tasks.** The first task is theirs, on their real repo, against their real conventions. Demo tasks teach customers that we only work on toy problems.
- **Quota-anxiety messaging.** Don't make new customers worry about their 25 PRs/mo before they've seen their first.
- **Convention prescription.** We surface defaults; we never override their explicit rules without user confirmation.

## Failure modes and recovery

| Failure | Detection | Recovery |
|---|---|---|
| Cartographer crashes on huge repo | Job watchdog | Auto-restart at last checkpoint; chunk by directory if >1M LoC |
| First task verifier rejects | Verifier verdict | Surface rejection with structured reasons; offer one auto-retry or human assist |
| Customer never installs GitHub App | Tracker missing GitHub events | Drip email + Slack-friendly nudge from CS |
| Customer installs but never submits task | Empty task log | Day-3 outreach with three suggested first-task options based on the Cartographer's output |
| Customer hits unexpected destructive-op gate | Gate event | Surface as a *good* thing in the dashboard — that's the safety story in action |
| Customer's tape coverage is too thin | Live-call denials | Walk through shadow-recording setup; help them populate tapes |

## What the customer never has to do

- Write `.cursorrules`, `AGENTS.md`, or any manual prompt. The Cartographer infers from their codebase.
- Configure model routing. The router picks per task.
- Manage budgets manually. Plans show projected cost.
- Manage memory eviction. Background distiller and importance scorer handle it.
- Sign individual attestations. Sigstore keyless OIDC is automatic.

The right experience is **the agent shows up, learns, ships verified PRs, and gets better monthly.**
