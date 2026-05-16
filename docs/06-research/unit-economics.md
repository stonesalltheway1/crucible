# Unit Economics

Resolves open question #4 from the original architecture: can per-verified-PR pricing work when the architecture uses both an executor model AND a separate-family verifier?

## TL;DR

The "2× tokens" worry is wrong in practice. Verification runs **once at the end of a task** (not in-loop), so total cost rises ~8%, not 2×. With aggressive 1h prompt caching and the right cross-family routing, median task cost lands at **~$1.69**. The Outcome tier at $8/verified-PR gives 79% GM and is the profit center; Pro/Team are breakeven-on-bundle by design and rely on overage for margin.

Two engineering KPIs determine company viability: **cache hit rate ≥ 70%** and **median task token budget ≤ 400K total tokens**.

## Real-world token consumption per agent task (May 2026 reference)

Major vendors don't publish official numbers, but leaks + forum reports cluster tightly:

| Product | Per-task profile |
|---|---|
| Cursor | ~22.5% of $20 Pro credit pool per agent run on 50K-LoC repo. Effective rate: $4.50 / ~500K–1M tokens per task. ~90% is cache write/read. |
| Claude Code | $13/active-day average (enterprise anchor), 90th percentile <$30/day. ~300K–700K billable tokens per active day = ~50K–100K per discrete task. |
| Aider | 4.2× fewer tokens than Claude Code (Morph Feb 2026 benchmark), accuracy 78→71%. Typical hour: 200K–400K tokens, $1–3 cost. |
| Devin | 1 ACU ≈ 15 min active work, $2.00–$2.25/ACU. Task average 1.5–3 ACUs = $3–$7 per task. |
| Replit Agent 3 | Simple edits ~$0.10; complex feature builds $5+. Heavy users report $1K/week. |
| Codex / GPT-5.5-Codex | API $5/$30 per M tokens, $0.50 cached. ChatGPT plan conversion: ~$1.20 entry-task. |

**Median agent task in market:** 200K–500K total tokens, $0.50–$5 marginal cost. Heavy tasks 1–2M tokens.

## Pricing landscape comparison

| Product | Entry | Mid | Top | Pricing primitive |
|---|---|---|---|---|
| Cursor | $20 Pro ($20 credit) | $60 Pro+ | $200 Ultra | Credit pool → API passthrough |
| Claude Code | $20 Pro | $100 Max5 | $200 Max20 | Weekly token ceiling |
| Codex / ChatGPT | $20 Plus | $30 Business | $200 Pro | Credit allowance (1 credit = $0.01) |
| GitHub Copilot (June 2026) | $10 Pro | $19 Business | $39 Enterprise/Pro+ | Plan-included AI credits, then usage |
| Devin | $20 entry | $500 Team | Enterprise | ACU ($2.00–$2.25) |
| Replit | $20 Core | $35 Pro+ | Enterprise | Effort-based |
| Windsurf | $15 Pro | $30 Teams | $60 Enterprise | Credits |
| v0 | $5 free | $20 Premium | $30/seat Team | Credits |
| Tabnine | $9 Dev | $39 Ent/seat | Enterprise | Seat |
| JetBrains AI | $10 | $20 | $30 | Seat, tiered quota |

**Market patterns:** seat-only (Tabnine, JetBrains) is collapsing; the dominant model is seat + credit pool + on-demand burst (Cursor, GitHub June 2026, Codex). Outcome pricing exists in adjacent markets (Sierra, Intercom Fin, Zendesk AI) but no coding-agent vendor has shipped it.

## Crucible per-task cost model

### Routing assumption (May 2026)

| Phase | Model | Justification |
|---|---|---|
| Planning | Sonnet 4.6 or Gemini 3.1 Pro | Tier 2 decisions; quality matters |
| Execution loop | Opus 4.7 | Agentic tool use leader |
| Verification | Gemini 3.1 Pro | Cross-family from Opus |
| Memory recall | Haiku 4.5 | Cheap, cached |

### Median task token math

12 tool calls, 6 reads, 3 writes, 2 test runs, 1 plan, 1 verify:

| Phase | Model | Raw input | Cached input | Output | Cost |
|---|---|---|---|---|---|
| Plan | Sonnet 4.6 | 50K (45K cached, 5K fresh) | 45K @ $0.30 = $0.0135 | 3K @ $15 = $0.045 + 5K @ $3 = $0.015 | **$0.074** |
| Exec × 6 steps | Opus 4.7 | 30K each (24K cached, 6K fresh) | 6×24K @ $0.50 = $0.072 | 6×8K @ $25 = $1.20 + 6×6K @ $5 = $0.18 | **$1.452** |
| Verify | Gemini 3.1 Pro | 40K (no cross-vendor cache) | n/a | 5K @ $12 = $0.06 + 40K @ $2 = $0.08 | **$0.14** |
| Memory recall | Haiku 4.5 | 4×20K (90% cached) | 72K @ $0.10 = $0.0072 | 4×500 @ $5 = $0.01 + 8K @ $1 = $0.008 | **$0.025** |
| **Total** | | | | | **~$1.69** |

### Three scenarios

| Scenario | Repo | Context/step | Cache hit | Marginal $/task |
|---|---|---|---|---|
| Small | 5K LoC | 15K avg | 85% | $0.55 |
| Median | 50K LoC | 30K avg | 75% | $1.69 |
| Large | 500K LoC | 80K avg, 10 steps | 60% | $6.80 |

### The "2× tokens" insight

Verification is end-of-task and uses ~8% of total cost — not 2×. The architecture's apparent cost penalty is closer to **1.08×**. This is a key narrative anchor: cross-family verification is essentially free compared to single-model execution.

## Pricing tier table (decision)

| Tier | Price | Included | Overage | Target |
|---|---|---|---|---|
| Pro | $40/mo | 25 verified PRs (median) | $2.50/PR | Individual dev, weekend builder |
| Team | $120/dev/mo | 80 verified PRs/dev pooled | $2.00/PR (volume) | 5–50 dev teams |
| Outcome | $8/PR + $500/mo min | No subscription, true PAYG | n/a | Legacy modernization, agencies, indie founders |
| BYOK | $25/dev/mo flat | Unlimited, customer brings keys | $0 token markup | Privacy-conscious, large enterprise |
| Enterprise (self-host) | $50K/yr + $400/node/mo | Unlimited, on-prem inference | Custom SLA | Regulated (FedRAMP, defense, healthcare) |

### Rationale

- **Pro $40 / 25 PRs = $1.60/PR effective.** Deliberately breakeven on the bundle; profitable on overage. Mirrors Cursor's $20-includes-$20-credit psychology but with a verified-PR unit.
- **Team $120/dev / 80 PRs = $1.50/PR effective.** Pooling lets heavy users average out with light users (typical team: 3–4 heavy committers per 10 devs).
- **Outcome $8/PR.** Mental anchor: 1 hour of contractor = $80–120; this is 10% of that. Legacy-modernization buyers compare to hourly consulting.
- **BYOK $25/dev flat.** Captures the "orchestrator without markup" segment (Cline/Aider archetype).
- **Enterprise $50K base.** Competes with self-hosted Sourcegraph Cody ($120K–300K typical).

## Margin analysis

GM per tier (median task assumption, $1.69 cost, 75% cache):

| Tier | Revenue/PR | Cost/PR | GM | GM @ 30% cache |
|---|---|---|---|---|
| Pro (included) | $1.60 | $1.69 | -5.6% | -45% |
| Pro (overage) | $2.50 | $1.69 | 32% | 6% |
| Team (pooled) | $1.50 | $1.69 | -13% | -52% |
| Team (overage) | $2.00 | $1.69 | 16% | -16% |
| **Outcome** | **$8.00** | **$1.69** | **79%** | **71%** |
| BYOK | $25/dev flat | ~$0 | ~100% | 100% |

### Critical insights

1. **Included-bundle pricing is negative-GM in worst caching cases.** This is the Cursor 2025 trap. Defense: cache hit rate must stay > 70%.
2. **Outcome tier is the profit center.** Sales motion should weight toward it.
3. **BYOK is high-margin** because we pay no model COGS. Don't undersell it.

### Break-even per seat

- Pro: $40 / $1.69 = **23.7 PRs cost floor** vs 25 included = **1.3 PRs slack**. Thin.
- Team: $120 / $1.69 = **71 PRs cost floor** vs 80 included = **9 PRs slack**. Healthier.

## Sensitivities

| Scenario | Impact |
|---|---|
| **Token prices -30% (likely by Q4 2026)** | GM +~20pp across bundled tiers. Pro -5.6% → +14% |
| **10× median volume per seat** | Pro at 250 PRs/mo costs $422 vs $40 revenue = catastrophic. **Hard usage caps mandatory.** |
| **Cache hit at 30%** | Median task → $3.10. All bundled tiers deeply negative. **Caching is THE engineering priority.** |
| **Verifier cost inflates to 25%** | Median → $1.94. Bundled tiers slightly more negative; Outcome still ~75% GM. |

## What's still uncertain

1. **Cross-family cache transfer.** Assumed verifier (Gemini) pays full fresh input cost. If verifier stays in Anthropic family (Sonnet verifying Opus), cost drops ~60% — but loses cross-family error decorrelation. Tradeoff TBD with eval data.
2. **Opus 4.7 tokenizer inflation.** New tokenizer consumes ~35% more tokens for same text. Actual median may be $2.10, not $1.69.
3. **Cache TTL at team scale.** 1h cache assumes user-session locality. Across a 10-dev team, locality drops. Team-pooled tier might effectively run at 50–60% cache hit and need revenue bump to $130/dev.
4. **PR-complexity distribution.** Probably Pareto: 20% of PRs consume 60% of cost. Need closed-beta data; complexity-banded pricing is v2.
5. **Verifier-rubric strictness.** Too strict → low pass rate → not enough verified PRs to count → user perceives non-value. Too lenient → bad PRs counted → reputation damage.
6. **Anthropic/Google price war probability.** Opus 4.8 at $4/$20 in Q3 2026 (likely) → GM +15pp. Status quo with tighter rate limits → opposite.

## Comparison to incumbents

| Incumbent | Their GM (rough) | Crucible position |
|---|---|---|
| Cursor | "slightly GM-positive" April 2026; achieved via Composer-2 in-house model | ~8% cost premium but ~25–50% price premium plausible |
| Anthropic / Claude Code | 60–70% (pays COGS, not retail) | Structural disadvantage; need BYOK + self-host |
| Devin | 65–75% (~$0.40–0.80 actual cost / $2.00–2.25 ACU price) | Aligned philosophy, cheaper per outcome, more transparent |

## GTM consequence

**Lead with Outcome tier and Team plans.** Pro is top-of-funnel, not profit. Build everything on the assumption that:

- Cache hit rate stays >70%.
- Median task ≤ 400K total tokens.

These are the KPIs that determine viability. They are observable from day 1 of beta. We do not commit to pricing until we have 30 days of real-customer telemetry validating both.

## Pricing-change roadmap

- **v1 (launch):** the five tiers above.
- **v2 (post-PMF):** complexity-banded Outcome ($4 small / $8 median / $20 large).
- **v3:** free OSS-maintainer tier (brand investment).
- **v4:** outcome SLAs ("N verified PRs/mo guaranteed at $X") if customer demand surfaces.

## References

- [00-vision/pricing-and-business.md](../00-vision/pricing-and-business.md)
- [ADR-004: Outcome-based pricing](../05-decisions/ADR-004-outcome-based-pricing.md)
- [01-architecture/model-routing.md](../01-architecture/model-routing.md)
