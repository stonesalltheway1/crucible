# v2 Vision (6-month horizon)

What ships after v1's PMF validation. Calibrated to agent throughput — each block here is 2–5 agent-days, not engineer-months.

## Sequencing principle

v1 is "the thesis end-to-end." v2 is "the thesis deepened where customer signal demands." We don't pre-build for hypothetical demand; we ship v1, learn from design partners, and prioritize v2 features by signal.

That said, several v2 features are predictable enough that pre-design pays off.

## Pillar A: Verifier deepening

### A1. Custom Crucible Verifier Model

Cursor's Composer-2 demonstrated that a small in-house orchestration model cuts cost ~10× vs frontier models. For Crucible specifically, a **verifier-tuned small model** could shrink verification cost while maintaining or improving cross-family error decorrelation.

**Scope:** ~3 agent-days for fine-tuning pipeline; ~$X for training compute; ~1 agent-day for routing integration.

**Trigger to build:** sustained ≥ 20% of total cost going to verifier across the customer base; or vendor token price increase >20%.

### A2. Multi-verifier ensemble

For high-stakes promotions, two cross-family verifiers ensemble. Disagreement between verifiers triggers human review.

**Scope:** ~2 agent-days. Mostly orchestration; small UX additions.

**Trigger:** enterprise customer demand for "no single point of LLM trust."

### A3. Tier 3 expansion

v1 ships Dafny as default Tier 3. v2 expands to:

- **Lean 4 + LeanCopilot** for crypto/numerical code.
- **TLA+ + Apalache** for distributed-invariants.
- **Kani** for Rust `unsafe` blocks and FFI boundaries.
- **Z3 / CVC5** as inline SMT for SMT-friendly proofs.

Each ~1.5 agent-days for the integration.

### A4. Customer-defined verifier extensions

Open the verifier extension API. Customers ship their own verifiers as Crucible plugins.

**Scope:** ~3 agent-days for plugin API + marketplace scaffolding. Skill-marketplace primitives (Claude Code's pattern is the reference).

**Trigger:** customer demand for stack-specific verification (e.g., compliance-rule-checker for a specific regulated domain).

## Pillar B: Memory deepening

### B1. Cross-tenant federation graduations

Once we have ≥ 5 tenants in each major stack, cross-tenant abstract-rule graduations become non-trivial. v1 has the data model; v2 ships the policy engine and surfaces the federated commons to customers.

**Scope:** ~2 agent-days for the graduation pipeline + tenant-visible commons browser.

**Trigger:** ~10 tenants on Team-tier-or-above in the same stack.

### B2. Visual / screenshot memory

Customers paste a screenshot of a UI mockup; Crucible's design subagent extracts design tokens (colors, typography, spacing) and applies them in generated UI code.

**Scope:** ~3 agent-days. Vision-model integration; design-token storage; UI-generation prompt hooks.

**Trigger:** "generic AI aesthetic" complaints from frontend-heavy customers.

### B3. Voice-input memory + transcribed stand-ups

Customer records team stand-ups, code reviews, or pair-programming sessions. Transcripts feed the distillation worker as a new source.

**Scope:** ~2 agent-days. Whisper-class transcription + distiller adapter.

**Trigger:** "we have lots of context in our standups that the agent doesn't see" feedback.

### B4. E2E-encrypted memory with customer key

For the highest-assurance enterprise tier. Memory at rest is encrypted with the customer's own KMS key; Crucible operators have read access only via signed access ceremony.

**Scope:** ~4 agent-days. Per-store crypto wrapper; access-ceremony UX; key rotation pipeline.

**Trigger:** FedRAMP / defense customer procurement requirement.

## Pillar C: Twin runtime deepening

### C1. GPU sandbox / ML-workload twins

For customers running ML services, the twin must include GPU access. Route to Modal Sandbox (Firecracker + GPU) or self-hosted with NVIDIA's container runtime.

**Scope:** ~3 agent-days. SandboxProvider interface already supports it; the work is GPU-specific orchestration + cost accounting.

**Trigger:** ML-engineering ICP customers materialize.

### C2. Mobile/iOS/Android twins

Xcode-in-cloud (MacStadium / Mac-Cloud) for iOS twins; Android emulator-in-Firecracker for Android. The simulator-first verification loop ([01-architecture/verifier-pipeline.md](../01-architecture/verifier-pipeline.md)) becomes especially valuable for native mobile because the simulator IS the feedback loop.

**Scope:** ~4 agent-days iOS, ~3 agent-days Android.

**Trigger:** the native-mobile vertical-specialist concept from the competitive research surfaces enough demand.

### C3. Embedded / firmware twins

ESP32 / STM32 / Nordic SDK twins running in QEMU + hardware-catalog-grounded mocks (Embedder.com's pattern). Pair with formal verification for safety-of-life code.

**Scope:** ~5 agent-days.

**Trigger:** embedded vertical customer wants to license.

### C4. Multi-region twin orchestration

For latency-sensitive workloads, twins co-located with customer's primary region.

**Scope:** ~2 agent-days. Region selection + provider routing.

**Trigger:** customer with global presence requests it.

## Pillar D: Pricing & business model evolution

### D1. Complexity-banded Outcome tier

$4 small / $8 median / $20 large per verified PR, based on diff size + Tier 3 escalation + critical-path classification.

**Scope:** ~1 agent-day. Pricing-rule engine + customer-facing pricing tooltips.

**Trigger:** 30 days of closed-beta PR-distribution data showing Pareto-tail customers under-paying.

### D2. SLA tier

"N verified PRs/mo guaranteed at $X" for enterprise customers.

**Scope:** ~2 agent-days. SLO engine for PR delivery + breach-credit billing.

**Trigger:** enterprise procurement asks for the SLA framing.

### D3. Open-source maintainer tier (free)

Verified-maintainer accounts get free Pro-tier usage. Brand investment.

**Scope:** ~1 agent-day. Verification flow (GitHub OSS-maintainer signal) + free-tier gating.

**Trigger:** brand-investment narrative timing (typically post-PMF, before scaled marketing).

### D4. Plugin / skill marketplace

Customers publish Crucible-compatible verifier extensions, MCP tools, custom Rego policies. Marketplace fee model (later).

**Scope:** ~4 agent-days. Marketplace registry + signing + discovery + payments.

**Trigger:** post-Pillar A4 (verifier extension API). Doesn't pencil until plugin ecosystem has scale.

## Pillar E: Specialization toward the vertical wedge

The competitive research identified five white-space concepts that emerged in v1. v2 picks the strongest signal:

### E1. Legacy Modernizer specialization

The Cartographer-as-product. Aggressive enhancement of:

- Characterization-test generation for poorly-tested legacy code.
- Layered refactor planner (extract this module → refactor that interface → migrate this DB schema).
- Per-module verified migration with property-based correctness contracts.

**Scope:** ~5 agent-days. Largely a customer-facing UX layer on existing Crucible primitives + specialized prompts for the LLM driver.

**Trigger:** legacy-modernization buyers convert at higher rate than other Outcome-tier customers, AND we have 2–3 reference modernization wins.

### E2. Autonomous Operator (cofounder seat)

Crucible owns the deployed product: deploys, on-call, incident triage, A/B analysis. Solo-founder-shaped buyer.

**Scope:** ~6 agent-days. Twin runtime extension to ops surface; SRE-agent specialization; observability deeper integration; revenue-share billing model.

**Trigger:** if Outcome-tier customers organically pull us into ops work, we productize it.

### E3, E4, E5 (Verifiable / Mobile / Convention-Learning)

Already covered in Pillars A, C, B respectively.

## Pillar F: Operational hardening

### F1. SOC 2 Type II certification

Required for the regulated tier. Year-long observation window; controls already designed.

**Scope:** ~0 agent-days for engineering (controls already in place); ~ongoing for audit support.

**Trigger:** target completion: ~12 months post-launch.

### F2. HIPAA-eligible SaaS tier

BAA-covered LLM vendors only. Per-tenant configuration enforced.

**Scope:** ~2 agent-days for routing-policy enforcement + BAA-vendor whitelist.

**Trigger:** healthtech customers convert.

### F3. FedRAMP Moderate certification

For defense / civilian-fed buyers.

**Scope:** engineering minimal; certification ~6 months of process.

**Trigger:** named defense customer commits to deployment.

### F4. EU-region data-residency tier

Anthropic EU + Vertex EU routing only. Pre-warmed cache in EU regions.

**Scope:** ~1 agent-day.

**Trigger:** EU customer demand.

## Pillar G: Cross-IDE agent identity

The "agent that follows you from VS Code → JetBrains → Terminal with shared memory" concept. With ACP as the standard, this is mostly a memory-layer + auth bind concern.

**Scope:** ~2 agent-days. Already feasible architecturally; just needs the cross-IDE auth state binding to be polished.

**Trigger:** customer feedback about IDE-fragmentation pain.

## How v2 sequences

Roughly:

1. **Month 4–5 (post-launch):** address top-3 customer-pain signals from design-partner + open-beta data. Likely: cache-hit improvement, Cartographer scaling, Tier 3 expansion.
2. **Month 6–7:** pricing iteration (D1 complexity-banded Outcome, possibly D3 OSS-maintainer tier).
3. **Month 8–9:** specialization (whichever vertical wins on signal — likely E1 or E2).
4. **Month 10–12:** compliance certifications (F1 SOC 2, F2 HIPAA SaaS).

This is rough. Real v2 is signal-driven.

## What we explicitly don't roadmap

- **Our own IDE.** Decided. ADR-011.
- **A new LLM.** We route. Composer-style in-house model is a verifier-cost optimization, not a product line.
- **Chat-with-LLM surface.** The IDE owns that; we own the verified deliverable.
- **Vibe-coding "build an app from prompt" surface.** Wrong ICP.

## How v2 is funded

By Outcome tier revenue + Team tier expansion + Enterprise contracts. The business model from v1 is intact through v2; v2 is depth, not pivot.

## Customer signal we watch

- **Top NPS detractors:** what specifically frustrates them?
- **Outcome tier churn:** are PR-bills predictable enough?
- **Enterprise customers' compliance requests:** which certifications do they actually ask for?
- **Memory growth rate per tenant:** are conventions accumulating, or stalling?
- **Cross-family verifier disagreement rate:** is the architecture working?
- **Self-hosted install time:** is the air-gap path realistic?

Each of these data points triggers v2 prioritization decisions.

## References

- [v1-mvp.md](v1-mvp.md)
- [build-plan-agent-days.md](build-plan-agent-days.md)
- [00-vision/competitive-landscape.md](../00-vision/competitive-landscape.md)
