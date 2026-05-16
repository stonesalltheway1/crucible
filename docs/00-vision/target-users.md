# Target Users

Crucible's positioning is "trust and verifiability" rather than "speed and autonomy." That positioning attracts a specific kind of buyer and repels another. Both clarifications matter — the wrong user is worse than no user because they generate noise that drowns out the signal we need.

## Primary ICP: Production-engineering teams of 5–200

The core buyer is an engineering team whose code touches real money, real customers, or real compliance obligations. Concretely:

- **Fintech / payments** — startups and scale-ups building on Stripe/Adyen/Plaid, neobanks, lending platforms, B2B finance APIs.
- **Healthtech** — companies handling PHI under HIPAA, working with FHIR data, building clinical workflow software.
- **Infrastructure & devtools** — observability, security, data infra, database-shaped products. They are themselves senior-engineer-heavy and the most allergic to AI-tooling sloppiness.
- **B2B SaaS at scale** — 50+ devs, multi-tenant, customer-data-bearing, SLA-bound. Past the prototype stage.
- **Regulated industries** — gov-tech, defense contractors, energy, anything where SLSA-L3 attestations are procurement requirements rather than nice-to-haves.

### The persona inside that team

The decision-maker we sell to is a **Principal Engineer or Staff+ IC** who:

- Has been burned by Cursor breaking working code or by Claude Code blowing through a budget.
- Reads ADRs for fun. Cares about reproducibility, hermeticity, formal verification when applicable.
- Is the person on the team who *blocks* AI-tool rollouts because the output is sloppy. They are the gate.
- Has authority (or strong influence) over tooling procurement decisions.
- Cites Kleppmann, Hillel Wayne, Antithesis, TigerBeetle, the Jepsen reports.

Their corporate context:
- A VP Eng or CTO above them who has been asked "what's our AI strategy?" and is looking for an answer that doesn't end in a public incident.
- A Security/Compliance lead who needs audit trails, attestations, and air-gap options.
- A Director of Eng who tracks PR throughput but also cares about defect rate.

### Why they buy

Crucible is the first AI coding agent they can *recommend* internally without their reputation hanging on whether the tool behaves itself. Specifically:

1. **They can let it run overnight.** Cross-family verification + destructive-op gate + bounded budgets make this safe in a way no incumbent enables.
2. **Compliance falls out for free.** SLSA-L3 attestations + Sigstore Rekor + replayable history checks the regulated-buyer procurement boxes without integration work.
3. **It learns their team's taste.** The procedural-memory graph means PR review comments don't have to be repeated; the agent absorbs them.
4. **Cost is bounded by contract.** Plan-time budget previews + hard caps + verified-PR pricing matches how their org procures engineering hours, not how token vendors bill.

## Secondary ICP: Solo founders shipping real revenue businesses

Not vibe-coders. Not weekend-app builders. People building real, customer-bearing, post-MVP businesses solo or with one collaborator:

- The Base44-style founder ($80M Wix exit, solo) — past the demo, now needs operational rigor.
- The HeadshotPro archetype ($3.6M ARR solo, Danny Postma) — generating revenue and needs to keep it generating.
- Indie SaaS at $10K–$100K MRR with one or two devs.

### Why they buy

They have no SRE team. They are the SRE team. They cannot afford a PocketOS-style 9-second disaster. They want an agent that owns operations, not just code generation. Crucible's twin runtime + signed promotion gate + per-tenant memory graph gives them an autonomous engineer they can actually trust because the architecture removes the failure modes they fear most.

The pricing tier they buy is **Crucible Outcome** — pay per verified PR, no seat commitment, easy to expense.

## Tertiary ICP: OSS maintainers drowning in AI-generated PRs

Stenberg (curl), Verschelde (Godot), and the broader "AI is burning out the people who keep OSS alive" cohort. Not a direct revenue line but a **brand-building** segment:

- Free Crucible tier for verified-OSS-maintainer accounts.
- The verifier helps them auto-reject low-quality AI-generated contributions.
- They write blog posts citing us.

## Explicitly *not* our user

- **Vibe-coders** building toy apps from a prompt. Crucible's deliberate friction (planning preview, verifier, attestations) is wrong for them. They are Bolt/Lovable/Replit-Agent buyers and should stay there.
- **Greenfield prototype-or-bust shops.** Cursor and Codex are faster on these tasks; our cross-family verification adds latency that doesn't pay off when the goal is "ship the first version this afternoon."
- **Junior-only teams.** The senior engineer who reads the verifier's output is load-bearing. Without that reader, the value collapses.
- **Pure consultancies billing hours to clients.** Their incentives reward more code, not better code. Our verifier slows down the bill clock.

## Sales motion implications

- **Bottom-up adoption via the senior engineer.** They install Crucible, run it on a small task, see the verifier's report, ship a verified PR, and bring it to their VP Eng.
- **The wedge product is the verifier itself.** Open-source it. Senior engineers will adopt the verifier standalone (point it at an existing agent's output and grade the agent). Once they trust the verifier, they upgrade to the full twin runtime.
- **Compliance-led top-down for regulated buyers.** A different motion: their procurement team asks "does this generate SLSA-L3 attestations?" and we say yes by default. They schedule the demo with the senior engineer who validates the trust story.
- **Founder Slack groups and Twitter for the secondary tier.** Indie founders read each other's recommendations.

## User journey sketches

### Sarah, Principal Engineer at a 40-dev fintech

- Sees Hacker News thread about PocketOS incident; one of the comments mentions Crucible's open-source verifier.
- Installs verifier as a GitHub Action. Points it at the team's existing Cursor-generated PRs. Sees ~12% of "passing" PRs fail the cross-family check. Posts the result in #eng-leadership.
- Schedules a Crucible demo. Runs the twin runtime on a sandboxed copy of the payments service for a week.
- Procures Team tier ($120/dev/mo) for the 8-person payments squad after seeing zero destructive incidents and a 30% reduction in revert PRs.
- Writes the case study six months later.

### Marcus, solo founder of a $40K MRR SaaS

- Bolt-and-Lovable-built MVP, then graduated to a real codebase he maintains himself.
- Burned $400 on Cursor in one stuck session. Posted a frustrated tweet. Someone linked Crucible.
- Tries the Outcome tier: $8/verified-PR, no commitment.
- Runs Crucible on Friday evening on a refactor that's been blocking him. Wakes up Saturday to a merged PR + a Slack message asking confirmation on one ambiguous decision.
- Stays on Outcome tier indefinitely.

### Priya, Director of Eng at a defense-contractor subsidiary

- VP told her "find an AI tool that procurement won't kill."
- Existing options: Tabnine (works but feels stuck in 2024), Cursor (procurement won't sign), Claude Code (no air-gap).
- Crucible's self-hosted enterprise tier ($50K/yr base + $400/node/mo) passes the procurement checklist: air-gap, SLSA-L3, in-toto attestations, no data leaving the perimeter.
- Pilots on a 200K-LoC legacy Java EE modernization. Twelve weeks later, 40% of the module migrations have been agent-generated and human-merged.

These three journeys cover the three pricing tiers and the three buying motivations. Build for them. Reject lookalikes.
