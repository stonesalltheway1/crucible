---
title: Your first verified PR
description: From sign-up to merged, verified PR in under 30 minutes.
---

# Your first verified PR

The "wow moment" we optimise the first hour around.

## Step 1 — Pick a real, in-progress task

Don't pick a synthetic demo. Pick something off your team's bug
tracker or backlog. Good first tasks:

- **A focused refactor** ("replace fetch calls with our new API client").
- **A small feature** ("add idempotency key to the webhook endpoint").
- **A bug fix** from your tracker.

Crucible's Cartographer surfaces three "good first task" suggestions
based on what it learned from your codebase — see them at
[app.crucible.dev/onboarding](https://app.crucible.dev/onboarding).

## Step 2 — Submit it

Pick your favourite surface:

```bash
# CLI
crucible task new --repo acme/payments \
    --description "Add idempotency key to /webhooks/stripe/refund"

# GitHub
# In any GitHub PR or issue comment:
/crucible add idempotency key to /webhooks/stripe/refund

# Slack
/crucible add idempotency key to /webhooks/stripe/refund

# IDE (VS Code)
Ctrl/Cmd-Shift-P → Crucible: New Task
```

## Step 3 — Approve the plan

Within ~90 seconds Crucible will show you the plan:

- Cost preview ($)
- Wall-clock estimate
- Files to touch
- Top risks (auto-classified)
- Hard cap (slider)
- Retry budget

You approve, amend, or reject. Approval is the trust signature.

## Step 4 — Watch it execute

Median execution: **5–15 minutes**. The web console streams every step
as it happens.

## Step 5 — Review the PR

Crucible opens a PR with:

- The diff
- The verifier's report (mutation score, conventions applied,
  attestation chain)
- A signed PromotionBundle linked
- Suggested rollout strategy

Your reviewer treats it as any other PR. The brand promise: **the
agent's output stands on its own; no human edits before merge.**

## Step 6 — Merge

Promotion contract executes (canary, dwell, land). The Sigstore
attestation chain is permanent and verifiable.

## What if it goes wrong?

- **Verifier rejected the diff:** the rejection is structured —
  category, line reference, suggestion. Crucible can auto-retry once
  with the rejection in context, or you can amend the plan.
- **Plan estimate exceeded:** the budget enforcer halted the agent
  before billing escalated. The plan you approved is the cap.
- **Canary regression:** auto-rollback fired within one SLO check
  cycle. The promotion stays unlanded.

Every step has a Sigstore Rekor UUID; every rejection has a runbook
in [04-operations/runbooks](/04-operations/runbooks).
