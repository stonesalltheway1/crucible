# Pricing and Business Model

Detailed unit economics live in [06-research/unit-economics.md](../06-research/unit-economics.md). This doc states the *decisions* — the published pricing surface and the business assumptions behind it.

## Pricing tiers (v1, public)

| Tier | Price | Included | Overage | Target buyer |
|---|---|---|---|---|
| **Crucible Pro** | $40 / mo | 25 verified PRs (median complexity), pooled within plan | $2.50 / PR | Individual dev, weekend builder, indie maker |
| **Crucible Team** | $120 / dev / mo | 80 verified PRs / dev, pooled team-wide | $2.00 / PR (volume) | 5–50 dev teams |
| **Crucible Outcome** | $8 / verified PR | $500 / mo minimum spend | Pure PAYG above minimum | Legacy modernization, agencies, contractors, indie founders |
| **Crucible BYOK** | $25 / dev / mo flat | Unlimited verified PRs; customer brings model API keys | $0 token markup | Privacy-conscious teams, large enterprises hedging model-spend volatility |
| **Crucible Enterprise (self-hosted)** | $50K / yr base + $400 / node / mo | Unlimited use, on-prem inference allowed, air-gap support | Custom SLA | Regulated industries (FedRAMP, defense, healthcare, banking) |

## The unit: "verified PR"

A PR counts as **verified** when:

1. All existing tests pass on the real codebase post-promotion (not just the twin).
2. The verifier model (different family from executor) rates the diff ≥0.85 on its scoring rubric.
3. No human edits the PR before merge — i.e., the agent's output stood on its own.
4. The promotion canary holds clean for the configured dwell window.

This bar is deliberately strict so the metering isn't gameable and the unit means something to a buyer ("a senior engineer would have merged this without changes").

PRs that fail to meet the bar are *not* billed. This both protects margin (we don't bill for trash) and reinforces the brand promise ("verified" actually means verified).

## Why this shape (and not Cursor-style credit pools)

Three considerations:

**Outcome-based unit aligns with buyer mental model.** A 50-dev engineering org procures "engineering hours" or "story points." "Verified PR" maps cleanly to both. Tokens and ACUs are vendor-internal units that don't map to anything a procurement committee recognizes.

**Hard ceiling kills bill-shock.** Cursor and Replit users have publicly reported $200–$1000/day blow-ups from runaway agent sessions. Our Pro/Team tiers cap exposure at the overage rate; Outcome is PAYG by design, with a clearly stated per-unit price. No one ever opens a Crucible invoice and sees a 10× surprise.

**Outcome tier is the GTM wedge.** Legacy modernization buyers (the highest-WTP segment) have no internal frame for token cost and compare to consultant hourly rates ($80–$200/hr). At $8/PR with a 2-hour-equivalent of senior-engineer work per PR, we're 5–10% of what they'd pay a contractor.

## Margin model

The verifier-doubles-token-cost concern turns out to be wrong in practice. Verification runs *once at the end* of a task, not in-loop, so the additional spend is ~8% of total token cost, not 2×. With aggressive 1h prompt caching and the cross-family verification routed cheaper (e.g., Gemini 3.1 Pro verifying Opus 4.7 output), the median task lands at **~$1.69 marginal cost**.

Gross margin by tier (median-task assumption, 75% cache hit rate):

| Tier | Revenue / PR | Cost / PR | GM | GM if cache drops to 30% |
|---|---|---|---|---|
| Pro (included) | $1.60 | $1.69 | -5.6% | -45% |
| Pro (overage) | $2.50 | $1.69 | 32% | 6% |
| Team (pooled) | $1.50 | $1.69 | -13% | -52% |
| Team (overage) | $2.00 | $1.69 | 16% | -16% |
| **Outcome** | **$8.00** | **$1.69** | **79%** | **71%** |
| BYOK | $25/dev flat | ~$0 | ~100% | 100% |

The Outcome tier is the profit center; Pro/Team are breakeven-on-bundle by design and rely on overage for margin. The two **engineering KPIs that determine company viability** are therefore:

1. **Cache hit rate ≥ 70%.** Engineering investment in cartography caching (5-min and 1h TTLs) is non-negotiable.
2. **Median-task token budget ≤ 400K total tokens.** Aggressive context-window discipline; never dump entire repos into prompts.

Below these thresholds the included-bundle tiers go deeply negative. Above them we have a real business.

## Risk-mitigated revenue forecast

We do not publish forecast numbers in design docs. Pricing assumptions live and die by closed-beta unit-economics data; see [06-research/unit-economics.md](../06-research/unit-economics.md) for the sensitivities table.

Top three risks:

1. **Cache-hit assumption fails at scale.** Multi-developer team usage may reduce locality and drag cache effectiveness below 50%. Mitigation: per-repo dedicated cache keyspace, persistent context pre-warming on a recurring schedule.
2. **PR-complexity distribution is heavy-tailed.** 20% of PRs likely consume 60% of cost. Mitigation: smart-throttle on heavy users (free up to 2× included, then $2.50/PR); complexity-banded pricing in v2 if needed.
3. **Token-price war from Anthropic/Google.** If prices fall 30% by Q4 2026 (likely), our GM expands ~20pp. If they hold and tighten rate limits instead, included-bundle tiers may need a price bump.

## Business model assumptions

- **GTM motion is bottom-up via senior engineers**, not top-down enterprise sales (until we've earned the air-gap tier customers). The Outcome tier is the wedge.
- **Open-source the verifier harness, the Hoverfly scrub pipeline, and the cartographer.** They are evangelism assets — give engineering taste-makers something to play with that earns the brand before the paid product lands. The orchestrator, memory graph, team console, and promotion contract stay proprietary.
- **BYOK is a deliberate concession to the Aider/Cline-aligned segment** who will never accept a token markup. It's a high-margin tier because we pay no model COGS.
- **Self-hosted enterprise is a non-trivial product surface** (air-gap installer, on-prem Rekor, on-prem KMS, etc.). Don't ship it until we have 2–3 named design partners willing to pay the full sticker price.

## Pricing changes we explicitly are NOT making

- **No annual commit discount on Pro tier.** Monthly is fine; we're not big enough yet to optimize for cash collection over churn protection.
- **No "free tier" beyond verifier OSS.** The OSS verifier is the free-tier substitute. A free hosted tier would attract vibe-coders, which is a wrong-customer problem.
- **No per-seat pricing on Team without a verified-PR cap.** Unlimited-seat plans bleed margin to whales; the cap is the lever.
- **No marketplace fee on plugins/skills (yet).** Premature for v1. Revisit when we have a plugin ecosystem worth taxing.

## Pricing roadmap

- **v1 (launch):** the five tiers above as published.
- **v2 (Q+1 after PMF):** add complexity-banded pricing on Outcome ($4 small / $8 median / $20 large) once we have empirical PR distribution data.
- **v3:** add a Crucible-for-Open-Source tier (free for verified-maintainer accounts) as a brand investment.
- **v4:** outcome SLAs ("we guarantee N verified PRs per month at this price") if customer demand surfaces.

## Key competitive context

The market has bifurcated:

- **Seat-only is collapsing** as agent costs scale with use, not seats (Tabnine, JetBrains squeezed).
- **Pure usage-based credit pools** are GM-positive in theory but produce bill-shock that kills adoption (Cursor 2025 trauma).
- **Outcome-based** works in adjacent markets (Sierra $1-$2/resolution, Intercom Fin $0.99, Zendesk $1.50-$2.00) but no coding-agent vendor has shipped it.

We are the first. The Outcome tier is the moat; everything else is positioning.
