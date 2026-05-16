# ADR-004: "Verified PR" as the pricing unit

**Status:** Accepted  
**Date:** 2026-05-15

## Context

The coding-agent market has bifurcated:

- **Seat-only pricing** (Tabnine, JetBrains) is collapsing because agent costs scale with use, not seats.
- **Pure usage-based credit pools** (Cursor, GitHub June 2026, Replit) produce bill-shock that kills adoption — $200/day blow-ups have been documented across multiple products.
- **Outcome-based pricing** works in adjacent markets (Sierra $1–2/resolution, Intercom Fin $0.99, Zendesk $1.50–$2.00) but no coding-agent vendor has shipped it.

Crucible's architectural commitment to *verification* gives us a clean, defensible unit that no incumbent can claim: the **verified PR**.

## Decision

Crucible prices primarily on "verified PRs delivered" — a PR counts as verified when:

1. All existing tests pass on the real codebase post-promotion.
2. The verifier model rates the diff ≥ 0.85 on its rubric.
3. No human edits the PR before merge — the agent's output stood on its own.
4. The promotion canary holds clean for the configured dwell window.

PRs that fail to meet the bar are not billed. The metering is therefore non-gameable from the customer's side, and the unit means something concrete to a buyer.

Five tiers:

| Tier | Price | Mechanism |
|---|---|---|
| Pro | $40/mo | 25 verified PRs included; $2.50/PR overage |
| Team | $120/dev/mo | 80 verified PRs/dev pooled; $2.00/PR overage |
| Outcome | $8/PR + $500/mo minimum | Pure PAYG above minimum |
| BYOK | $25/dev/mo flat | Unlimited; customer brings model API keys |
| Enterprise (self-hosted) | $50K/yr base + $400/node/mo | Unlimited use, on-prem inference |

## Consequences

### Positive

- **Aligns with buyer mental model.** A 50-dev engineering org procures engineering-hours or story-points. "Verified PR" maps cleanly to both. Tokens / ACUs don't map to anything procurement recognizes.
- **Hard ceiling kills bill-shock.** Pro/Team have included caps + capped overage. Outcome is PAYG with clear per-unit price. No one ever opens a Crucible invoice and sees a 10× surprise.
- **Outcome tier is the profit center.** At $8/PR and ~$1.69 median cost, GM is 79%. Legacy-modernization buyers (highest WTP) compare to consultant hourly rates ($80–$200/hr); $8/PR is ~5–10% of that.
- **First-mover advantage.** No coding-agent vendor has shipped outcome pricing. The narrative differentiation is durable until copied.

### Negative

- **Included-bundle tiers are GM-thin.** Pro at $1.60/PR effective revenue vs $1.69 median cost is structurally negative-GM on the bundle, breakeven-to-profitable on overage. Requires cache hit rate ≥ 70% to be sustainable.
- **Metering complexity.** "Verified" is a 4-condition AND; building the metering correctly is non-trivial. Customer disputes are expensive.
- **PR-complexity distribution risk.** A 1-line config fix and a 2,000-line migration both count as 1 PR. The pricing math assumes "median complexity"; heavy tails distort it. Mitigation: complexity-banded pricing in v2 once we have data.
- **Verifier-rejection edge cases.** If our verifier rejects a PR a human would have merged, the customer feels we're being precious. Mitigation: shadow-mode tracking; rubric tuning per RB-07.

### Trade-offs we accept

- Pro/Team bundles are deliberately a customer-acquisition cost, not a profit center. Outcome tier and Enterprise tier pay the rent.
- We will lose price-sensitive customers who can self-tune their LLM keys cheaper with Aider/Cline. The BYOK tier is our concession to that segment.

## Alternatives considered

### Alternative 1: Cursor-style credit pools

$X/mo includes $Y model-spend. **Rejected**:

- Bill-shock UX is bad. Cursor's 2025 trauma demonstrates this.
- "Credit" isn't a recognizable procurement unit.
- Doesn't reward our verification investment.

### Alternative 2: Pure seat-only ($30/dev/mo flat)

Tabnine model. **Rejected**:

- Whales subsidize light users; team plans bleed margin on heavy users.
- Doesn't align with COGS, which scales with use.

### Alternative 3: Per-token markup over BYOK

Customer pays for tokens + a flat platform fee. **Rejected for primary tier**:

- Same bill-shock problem as Cursor.
- Doesn't capture the verification value (we'd be billing for executor tokens but the verifier is free? Or doubled?).

(BYOK tier exists as a deliberate concession to the Aider/Cline-aligned segment.)

### Alternative 4: Devin-style ACU (Agent Compute Units, 15-min intervals)

Vendor-defined opaque time unit. **Rejected**:

- ACU is internal-bookkeeping made customer-visible. Buyer can't predict cost.
- "Verified PR" is auditable; ACU isn't.

### Alternative 5: Per-story-point or per-Jira-ticket

Charge by the customer's own ticket size. **Rejected**:

- Requires integration with the customer's PM tool to bill — operational dependency.
- Customer disputes over story-point sizing become invoice disputes.

### Alternative 6: Per-test-passing

Charge per test the agent makes pass. **Rejected**:

- Easy to game; agents would write trivial tests.
- Doesn't capture refactor work (no test changes but real value).

## Open issues

- **Complexity banding (v2):** introduce small/median/large PR tiers ($4/$8/$20) once empirical PR distribution data justifies.
- **SLA tier (v3):** offer "N verified PRs guaranteed per month at $X" for enterprise.
- **Open-source maintainer tier (v3):** free for verified OSS maintainer accounts as a brand investment.

## References

- [00-vision/pricing-and-business.md](../00-vision/pricing-and-business.md)
- [06-research/unit-economics.md](../06-research/unit-economics.md)
