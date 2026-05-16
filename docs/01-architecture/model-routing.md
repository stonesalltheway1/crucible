# Model Routing

Five tiers, ~12 models, route by task class. Cross-family executor/verifier pairing is the architectural anti-hallucination contract.

## The tier table (May 2026 reference pricing)

| Tier | Role | Primary | Alternates | $ in / out / cache | Context |
|---|---|---|---|---|---|
| **0** | File reads, grep, planning decomposition, leaf retrieval | `claude-haiku-4-5` | `gemini-3-flash-lite` ($0.10/$0.40), `deepseek-v4-flash` ($0.14/$0.28) | $1 / $5 / $0.10 | 200K |
| **1** | Standard coding, multi-file edits, test authoring | `claude-sonnet-4-6` | `gpt-5.1-codex-max` ($1.25/$10), `gemini-3-flash` ($0.50/$3) | $3 / $15 / $0.30 | 1M |
| **2** | Hard refactors, architecture, property-test/invariant authoring | `claude-opus-4-7` | `gpt-5.5` ($5/$30), `gemini-3.1-pro` ($2-4/$12-18, 2M ctx) | $5 / $25 / $0.50 | 1M |
| **3** | **Verifier** — must be different family from executor | When primary = Opus 4.7 → `gemini-3.1-pro` | When primary = GPT-5.5 → Opus 4.7 | Varies | — |
| **4** | Local / privacy-sensitive | `Llama-4-Scout` (10M ctx) | `DeepSeek-V4-Pro` (MIT, 1M), `Qwen3-Coder-Plus` (262K) | self-hosted | — |

## Routing rules

### 1. Task class drives tier

Inferred from manifest + procedural memory.

- "Fix typo / rename" → Tier 0.
- "Add field to a struct, propagate" → Tier 1.
- "Refactor service to use new dependency" → Tier 2.
- "Authoring property tests / formal invariants" → Tier 2.
- "Touched file scores ≥ 80 in critical classifier" → Tier 2 executor + Tier 3 verifier with Dafny/Lean/TLA+.

The router's classifier is itself a small LLM call (Haiku 4.5, cacheable) over the task description + initial repo cartography.

### 2. Verifier pairing is mandatory

Every task has both an executor model and a verifier model. **They must be from different families** (different vendor lineage, different tokenizer, different RL recipe). Strong pairings:

- `claude-opus-4-7` ↔ `gemini-3.1-pro` (high thinking)
- `gpt-5.5` ↔ `claude-opus-4-7`
- `Llama-4-Maverick` (local) ↔ `DeepSeek-V4-Pro` (local)

Configured per-tenant. BYOK and self-hosted customers can override.

### 3. Cache strategy

Anthropic and Google both expose explicit prompt caching. OpenAI uses automatic caching (~5–10 min TTL). The router schedules:

- **System prompt + repo cartography** → 1h cache slot. Saves ~90% of input cost across a single task.
- **Active file context** → 5m cache slot. Refreshed on every edit.
- **Tool definitions** → 1h cache slot. Static for the task.

Cross-vendor cache transfer is impossible — verifying with Gemini incurs full input cost on its first pass even though the executor was Anthropic. This is the single biggest cost line item; engineering investment in keeping verifier prompts small is critical.

### 4. Budget enforcement

Every plan declares a dollar budget. The Bounded Budget Enforcer (Control Plane) tracks token spend per call and halts execution when the budget is exceeded. The user must re-plan to continue.

Budgets per tier (default; user-tunable):

| Plan tier | Budget cap |
|---|---|
| Trivial | $0.50 |
| Standard | $2.00 |
| Complex | $10.00 |
| Critical | $25.00 |
| Modernization (Outcome tier) | $50.00 |

## Per-vendor specifics

### Anthropic

- **Opus 4.7** (`claude-opus-4-7`): 1M context, 128K output, $5/$25/$0.50, 5m and 1h cache TTLs. Adaptive thinking (model decides depth). New tokenizer uses ~35% more tokens than older Claude — account for in budget.
- **Sonnet 4.6** (`claude-sonnet-4-6`): 1M, 64K output, $3/$15/$0.30. Extended thinking toggleable.
- **Haiku 4.5** (`claude-haiku-4-5`): 200K, 64K output, $1/$5/$0.10. Extended thinking yes.
- All support computer-use, tool calling, vision, MCP.
- **Best for:** agentic loops, tool use, computer use. Default executor.

### OpenAI

- **GPT-5.5** (`gpt-5.5`): ~920K input, 128K output, $5/$30, automatic caching. `reasoning_effort` parameter.
- **GPT-5.3-Codex** (`gpt-5.3-codex`): 400K context, $1.75/$14. Code-specialized; #1 Terminal-Bench 2.0.
- **GPT-5.1-Codex-Max** (`gpt-5.1-codex-max`): 400K, $1.25/$10. Cheapest OpenAI agentic option.
- First-class JSON Schema strict mode + function calling.
- **Best for:** terminal-bound verification, JSON-schema-strict outputs.

### Google

- **Gemini 3.1 Pro** (`gemini-3.1-pro-preview`): 2M context, $2/$12 (<200K) or $4/$18 (>200K). Configurable thinking levels. #1 LiveCodeBench Elo 2887.
- **Gemini 3 Flash**: ~1.05M, $0.50/$3.
- **Gemini 3 Flash-Lite**: 1M, ~$0.10/$0.40.
- All support explicit + implicit caching, JSON-Schema responseSchema, native multimodal.
- **Best for:** Tier 3 verifier on Opus-executed tasks; algorithmic invariant authoring; 2M-context whole-repo passes.

### xAI

- **Grok 4.3** (`grok-4.3`): 1M, $1.25/$2.50. Code-ready successor to Grok-Code-Fast-1.
- Useful as a third-family fallback for sensitive teams who want non-Big-Three vendor mix.

### DeepSeek

- **DeepSeek V4-Pro**: 1M, $1.74/$3.48 standard (75% off through May 31 2026: $0.435/$0.87). MIT-licensed open weights. Native and `/anthropic` endpoints.
- **Best for:** self-hosted privacy tier; cheap verifier when paired with Claude/GPT executor.

### Open-weights (Llama, Qwen)

- **Llama 4 Scout** (10M context, 73.4% SWE-Bench Verified): primary local-host pick for privacy-sensitive customers.
- **Qwen3-Coder-Plus** (80B MoE, 262K context, strong agent tool calling): alternative; open weights on HuggingFace.

## Routing decision algorithm

```python
def route(task: Task, tenant: Tenant) -> Routing:
    # 1. Classify task complexity
    complexity = classify_complexity(task)  # Haiku 4.5 call, cached

    # 2. Determine if critical-path scoring applies
    critical_score = critical_classifier(task.touched_files, tenant)
    is_critical = critical_score >= 80

    # 3. Pick executor tier
    if complexity == "trivial":
        executor_tier = 0
    elif complexity == "standard":
        executor_tier = 1
    elif complexity == "complex" or is_critical:
        executor_tier = 2

    # 4. Pick executor model from tenant config or default
    executor = tenant.model_overrides.get(executor_tier, DEFAULTS[executor_tier])

    # 5. Pick verifier from DIFFERENT family
    verifier = pick_cross_family_verifier(executor)
    if is_critical:
        verifier = upgrade_to_tier3(verifier, prover_choice(task))

    # 6. Budget allocation
    budget = budget_for(complexity, critical=is_critical, tenant=tenant)

    return Routing(executor, verifier, budget, complexity, critical_score)
```

## Privacy / data-residency rules

Per-tenant policy controls which routes are allowed:

- **Standard tenant:** any frontier model.
- **EU-data-residency tenant:** Anthropic EU region, Gemini EU region, no US-only models.
- **Healthcare HIPAA tenant:** BAA-covered models only (Anthropic w/ BAA, Azure OpenAI w/ BAA, Vertex AI w/ BAA).
- **Air-gap / FedRAMP tenant:** Tier 4 models only, local-host. No external API calls.

Policy enforced at the router; violations return `RoutingDenied` with the policy name.

## Cost telemetry

Every model call emits an OTel span with:

- `model.vendor`, `model.id`, `model.tier`
- `tokens.input.fresh`, `tokens.input.cached`, `tokens.output`
- `cost.usd` (computed via current price table)
- `task_id`, `step_id`, `tenant_id`

Dashboards in [02-engineering/observability.md](../02-engineering/observability.md) aggregate these to:

- Per-task cost (median, p95)
- Cache hit rate (the critical KPI; must stay ≥ 70%)
- Verifier cost as % of total (sanity check that we're not hitting 2× regression)
- Per-tenant routing distribution (informs upsell)

## What changes in v2

- **Custom Composer-2-style in-house model** for Tier 1 cost-cutting (Cursor's strategic move). Tabled until v1 PMF clear.
- **Speculative-decoding pairings** (cheap proposer + frontier verifier in the same call) when vendor APIs support it broadly. Currently emerging in Anthropic Sonnet/Opus pairings.
- **Model price oracle** — auto-rebalance routing as vendor prices shift quarterly. v1 hardcodes May 2026 pricing.

See [05-decisions/ADR-006-cross-family-verifier.md](../05-decisions/ADR-006-cross-family-verifier.md) for the rationale on mandatory cross-family pairing.
