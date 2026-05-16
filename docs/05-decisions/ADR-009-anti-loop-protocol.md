# ADR-009: Hard retry cap and bounded-budget enforcer

**Status:** Accepted  
**Date:** 2026-05-15

## Context

The single most expensive failure mode of 2025-era agents is infinite explore/thinking loops. Opus 4.6 specifically called out for this in published GitHub issues (#19699, #24585). Uber reportedly burned its full-year 2026 Claude Code budget in four months due in part to such loops. Individuals report $200/day burns from single stuck sessions.

The pattern: agent attempts subtask → fails → "reflects" → attempts the same flawed approach → fails → repeat. The agent has no self-imposed limit; the user has no visibility until the bill arrives.

## Decision

Crucible enforces three hard bounds, in-process, surfaced visibly to the user throughout:

### 1. Retry budget per subgoal: 3 attempts maximum

After 3 failed attempts at the same subgoal:
- The agent **cannot** retry the same subgoal.
- It must either (a) re-plan with a different approach (which the Plan Builder signs off on), or (b) halt and call `twin.plan.requestReplan(reason)` for human input.
- The retry counter is per-subgoal-identity (the goal description, not the raw tool call); same-approach retries count.

### 2. Dollar budget per plan: hard cap

- Every plan declares an `estimated_cost_usd`. Plan approval sets the budget cap (default = estimate × 1.5, or user-specified).
- Bounded Budget Enforcer tracks token spend in real-time.
- At 80% of budget: agent receives a warning; can request a budget extension via `twin.plan.requestReplan`.
- At 100% of budget: execution **halts**. Task moved to `budget_exceeded` state. User must approve continuation.

### 3. Wall-clock budget per task: hard cap

- Default: 60 minutes per task.
- Customer-configurable, max 4 hours.
- At cap: same behavior as dollar-budget exceeded.

All three bounds are visible in the task UI throughout execution: `[$0.31 / $1.00 budget — retry 1/3 — 4:31 / 30:00 elapsed]`.

## Consequences

### Positive

- **The Opus-4.6 loop class is architecturally eliminated.** Three retries; if no progress, halt and ask. No more $200 stuck sessions.
- **Customer trust in costs.** The plan shows an estimate; the bound caps the deviation. Cost is predictable to within 50%.
- **Forces agent to reflect strategically.** With only 3 retries, the agent must change approach on each retry, not iterate the same code.
- **Surface for user intervention.** When the bound fires, the user sees a structured "the agent is stuck on X" — opportunity to redirect, not silently fail.

### Negative

- **Some genuinely complex tasks need more than 3 retries to converge.** Mitigation: re-planning is the escape valve; the user explicitly approves a new plan with reset budget.
- **Budget estimates are noisy.** A bad estimate means a hard cap fires on a legitimate task. Mitigation: 1.5× headroom; learning loop refines estimates per-tenant over time.
- **Wall-clock cap may fire on Tier 3 proof-heavy tasks.** Mitigation: critical-path-flagged tasks get a higher default wall-clock cap (4 hours), and Tier 3 proofs have their own per-proof timeout that doesn't count against task wall-clock.

### Trade-offs we accept

- A small fraction of tasks (estimated < 5%) will hit a cap that a longer-running agent would have completed. We accept this; the alternative (no cap) is the documented disaster.
- Estimate-vs-actual divergence is a tunable; we err on the side of overshooting estimates so the cap doesn't fire spuriously.

## Implementation

The Bounded Budget Enforcer is a sidecar to the agent process, not a library the agent calls. It:

1. Subscribes to the cost-meter's per-call telemetry.
2. Tracks against the plan's caps.
3. Returns `BudgetExceeded` from `twin.*` SDK calls when caps reach.
4. Cannot be bypassed by the agent — no SDK method skips enforcement.

The retry counter is enforced at the task router level — when the agent restarts a step, the router checks the per-subgoal retry counter and refuses to re-dispatch.

## Alternatives considered

### Alternative 1: Soft warnings only, no hard cap

Show warnings; trust the agent to stop. **Rejected** — that's exactly what 2025-era agents do, and it produces the documented $200/day disasters.

### Alternative 2: Per-tool-call cap, not per-task

Cap each LLM call at $X. **Rejected**:

- Doesn't solve the loop problem (1,000 cheap calls = same end cost).
- Surface is the wrong level of abstraction; users plan in tasks, not calls.

### Alternative 3: Auto-extend budget on user opt-in

Configure tenant-level "budget extends automatically up to $Y." **Considered**; deferred to v2:

- Useful for heavy enterprise users who hate manual approvals.
- Not v1 because it's a foot-gun: customers will set $Y too high and hit bill-shock anyway.

### Alternative 4: Variable retry budget per subgoal complexity

Easier subgoals get fewer retries; harder ones get more. **Rejected for v1**:

- Complex to estimate; complexity is itself uncertain.
- Three retries is empirically right for the vast majority of subgoals.

### Alternative 5: Retry budget is a hint, not a hard cap

Allow agent to override with strong justification. **Rejected**:

- Agents always have a "strong justification" in their own reasoning trace.
- The whole point is the cap is non-negotiable.

## Customer-tunable parameters

Tenant config (in the web console / `crucible-cli tenant config set`):

```
retry_cap_per_subgoal: 3              # default; min 1, max 5
dollar_budget_multiplier: 1.5         # default; min 1.0, max 3.0
wall_clock_cap_min: 60                # default; min 5, max 240
auto_extend_on_progress: false        # opt-in; allows budget extension if progress detected
critical_path_wall_clock_cap_min: 240 # for tier-3-heavy tasks
```

## Observability

Every cap-firing event becomes:
- A task-state transition (`budget_exceeded`, `retry_cap_exceeded`, `wall_clock_exceeded`).
- A webhook event (`task.budget_exceeded`).
- An OTel span attribute (for cost-cap pattern detection).
- Visible in the customer's cost dashboard ("you hit the cap 14 times this week — consider re-planning approach").

A tenant repeatedly hitting caps is a customer-success signal: their workload pattern doesn't fit defaults, and we should reach out.

## References

- [01-architecture/system-overview.md](../01-architecture/system-overview.md)
- [03-sdk/agent-sdk-reference.md](../03-sdk/agent-sdk-reference.md)
