# Memory Bootstrap Strategy

Resolves open question #2 from the original architecture: how does procedural memory work on day 1 for a fresh customer with no PR history to learn from?

## The cold-start problem

A new Crucible install has no `(commenter, requested_change_type, code_pattern, accepted?)` data. The procedural memory graph is empty. Without intervention, the agent's day-1 output reflects model defaults (often "Tailwind Blue gradients and rounded corners aesthetic") — exactly the "generic AI aesthetic / convention drift" complaint we exist to solve.

## The four-tier seed corpus

Bootstrap procedural memory from public OSS sources, license-filtered, before the customer's own PR history accumulates. Tiers ranked by signal-to-noise:

### Tier A — Curated style guides (~40 documents)

Deterministic, authoritative, license-clean. Direct ingestion; no LLM re-interpretation needed beyond categorization.

- **Google style guides** — C++, Java, Python, Shell, TypeScript, JavaScript, R
- **Airbnb JavaScript / React style guide**
- **Microsoft TypeScript coding guidelines**
- **PEP 8 + PEP 257 + PEP 484 + PEP 526** (Python)
- **Effective Go + google/styleguide/go + Uber Go style guide**
- **Rust API Guidelines + tokio style notes**
- **Ruby/Rails style guide (bbatsov + rubocop/rails-style-guide)**
- **HackSoft Django Styleguide + Django coding-style docs**
- **Spring framework code-quality docs + spring-petclinic reference layout**
- **Phoenix/Elixir: christopheradams/elixir_style_guide + Credo defaults**
- **Swift API design guidelines (Apple)**
- **tum-esi/common-coding-conventions**

Weighted **×1.5 confidence** because these are authoritative.

### Tier B — Top-N OSS repos per stack (~2,400 repos)

Top 200 repos per major stack (12 stacks: Next.js, Django, FastAPI, Flask, Rails, Spring Boot, Go services, Rust services, Phoenix, Vue, Express, Laravel) by signal:

```
score = log(stars) × log(commits_last_90d + 1) × test_coverage_signal
filters:
  LICENSE in {MIT, Apache-2.0, BSD-*, MPL-2.0, ISC, Unlicense}
  has_ci
  has_codeowners_or_editorconfig
  not in {GPL-*, AGPL-*, SSPL-*, BUSL-*}
```

Extract:

1. **Lint configs** parsed deterministically (zero LLM cost):
   - `.editorconfig`, `.prettierrc`, `.eslintrc`, `tsconfig.json`
   - `.rubocop.yml`, `pyproject.toml` (ruff/black/isort)
   - `rustfmt.toml`, `clippy.toml`, `.golangci.yml`
   - `phpcs.xml`, `checkstyle.xml`, `.stylelintrc`, `.markdownlint.json`
   - `CODEOWNERS`, `commitlint.config.js`, `renovate.json`, `.gitleaks.toml`

2. **AGENTS.md ecosystem** — 60K+ repos by January 2026. Section-segment, LLM-categorize. The GitHub Blog's 2,500-repo analysis is the canonical pattern.

3. **CONTRIBUTING.md** — community-facing convention statements.

4. **`docs/architecture/`, `docs/adr/`** — ADRs, design rationale.

Expected yield: ~25K convention candidates after dedup.

### Tier C — PR review comment corpus (~300K diff-comment pairs)

Mine merged PRs from Tier-B repos, last 24 months, with ≥1 non-author review comment.

Filter aggressively:

- Drop "LGTM", "approved", trivial.
- Drop bot comments (dependabot, renovate, github-actions).
- Drop typo fixes.
- Min 20-char comment length.
- Comments that resulted in change (not just discussion).

Cluster by embedding (HDBSCAN); dense clusters become candidate rules. Target ~300K diff-comment pairs in, ~8K clusters out, ~3K surviving the cross-repo agreement threshold.

The LAURA dataset (arXiv 2512.01356, 301K diff-comment-info triples from 1,807 popular GitHub projects) is the directly-usable existing corpus.

### Tier D — ADR + post-mortem corpus (~5K records)

Smaller but very high signal — ADRs are *intentional* convention statements with rationale.

Sources:
- `joelparkerhenderson/architecture-decision-record` (largest curated set)
- Lullabot architecture decisions log
- `opendatahub-io/architecture-decision-records`
- Tier-B repos with `docs/adr/` directories
- Public post-mortem corpus (Increment, Honeycomb's "What I Learned" series, etc.)

Weighted **×1.5 confidence** because authoritative.

## The 12-category taxonomy

Conventions are categorized into 12 buckets matching AGENTS.md section heading conventions used by the top 2,500 repos:

| Category | Example rule |
|---|---|
| Naming | "Test files end in `_test.go` (Go) or `.test.ts` (TS)" |
| Layering | "Code in `db/` cannot import from `web/`" |
| Library preferences | "Use date-fns; don't introduce moment.js" |
| Test patterns | "Tests colocate with source in `__tests__/`" |
| Error handling | "Use Result<T,E> for fallible ops; no exceptions for control flow" |
| Logging | "Structured slog calls; no fmt.Printf in non-test code" |
| Migration patterns | "Migrations are additive-only; deprecation period >= 30 days" |
| PR/commit hygiene | "Conventional Commits; max 250-line diff" |
| Security defaults | "Auth middleware before any route handler" |
| Performance defaults | "Use cursor pagination, not offset" |
| Concurrency | "Pass context.Context through every async chain" |
| API shape | "Errors return { error: { code, message } } envelope" |

Each convention carries:

```typescript
Convention {
  id, scope (file_glob), confidence (0..1),
  rule_nl, category,
  positive_examples: SourceRef[], 
  negative_examples: SourceRef[],
  source: SourceRef[],
  first_seen, last_reinforced, last_violated,
  status: active | drifting | superseded,
  supersedes: ConventionId[]
}
```

## Stack-specific defaults

Per stack, a "day-1 ship-ready" bundle:

- **Rails** — `rubocop/rails-style-guide` + Rails Guides + `bbatsov/ruby-style-guide`. Highest signal density of any stack (Convention over Configuration).
- **Django** — HackSoft Django Styleguide + Django coding-style docs + Django contrib guide. Pair with ruff + black + isort defaults.
- **FastAPI** — `zhanymkanov/fastapi-best-practices` + tiangolo's docs + Pydantic v2 idioms.
- **Flask** — Pallets project docs + `cookiecutter-flask` patterns.
- **Next.js/React** — Vercel's `vercel/commerce` reference + `shadcn/ui` + Airbnb JS + react.dev hooks rules + Pages-vs-App-Router conventions.
- **Go** — Effective Go + `google/styleguide/go` + Uber Go Style Guide + golangci-lint defaults.
- **Rust** — Rust API Guidelines + clippy pedantic-lints (selective) + tokio style.
- **Spring Boot** — `spring-projects/spring-petclinic` reference + Google Java + Spring contributing.
- **Phoenix/Elixir** — `christopheradams/elixir_style_guide` + Credo + Phoenix Guides.

## Extraction pipeline

```
Public OSS Corpora ──▶ License filter (MIT / Apache-2.0 / BSD / MPL only)
                                │
                                ▼
                    ┌───────────────────────────┐
                    │ Deterministic config pass │
                    │ (lint configs → rules,     │
                    │  ~30% of conventions free) │
                    └─────────────┬─────────────┘
                                │
                                ▼
                    ┌───────────────────────────┐
                    │ LLM distillation pass     │
                    │ (Haiku 4.5, schema-fixed) │
                    │  textual corpus → typed   │
                    │  Convention candidates    │
                    └─────────────┬─────────────┘
                                │
                                ▼
                    ┌───────────────────────────┐
                    │ Cross-source agreement    │
                    │ embed → cluster → confidence│
                    │ confidence = log(distinct_repos_agreeing) /
                    │              log(repos_examined_in_stack) │
                    └─────────────┬─────────────┘
                                │
                                ▼
                    ┌───────────────────────────┐
                    │ Counter-example pass      │
                    │ find contradictions in    │
                    │ corpus; attach to rules   │
                    └─────────────┬─────────────┘
                                │
                                ▼
                    ┌───────────────────────────┐
                    │ Surface at install        │
                    │ confidence >= 0.4 → ACTIVE│
                    │ 0.25–0.4 → SUGGESTED      │
                    │ < 0.25 → CANDIDATE        │
                    └───────────────────────────┘
```

**Extraction model + prompt:**

```
Given this excerpt from {source_type: AGENTS.md | CONTRIBUTING.md | ADR | PR comment | style guide},
extract zero or more enforceable rules. Output JSON array of:
  { category, rule, file_glob, rationale, evidence_quote }
Emit nothing if no enforceable convention is stated.
```

Validated against schema (AdaKGC SDD pattern); retry once on validation failure, then drop.

## License / IP considerations

**Facts are not copyrightable; expression is.** We're extracting facts (which library, naming pattern, file structure) — analogous to a style guide author summarizing prior art.

Strict rules:

- **MIT / Apache-2.0 / BSD / MPL inputs:** safe to derive defaults; preserve attribution in `THIRD_PARTY_SOURCES.md`.
- **GPL / AGPL / SSPL / BUSL inputs:** *exclude entirely* from seed corpus. Even if extraction is arguably fair use, the downstream-redistribution exposure isn't worth it.
- **Code snippets in examples:** never ship verbatim OSS code as a positive example unless it's MIT/Apache, <10 lines, and attributed. Prefer LLM-paraphrased synthetic examples.

## Cross-tenant leakage prevention

Customer A's procedural memory must never leak to Customer B's agent.

- **Three-tier memory:**
  - `global_defaults` (from OSS seed corpus, shippable to all tenants)
  - `org_overrides` (customer-private, tenant-scoped)
  - `repo_overrides` (per-repo, lowest layer)
  - Agent reads bottom-up; only bottom two are tenant-scoped.

- **Generalization-upward rule:** customer-derived rules can graduate into `global_defaults` *only* when:
  - They appear in ≥ K independent customer tenants (K = 5 minimum)
  - The rule is anonymized to its category form (e.g., "prefer webhooks over polling for payment-provider integrations" — never "use Stripe webhooks")

- **Embedding-space isolation:** never share embeddings of customer-private rules across tenants. Per-tenant namespaces in the vector store (pgvector RLS, Qdrant per-tenant collections).

- **Differential privacy** on cross-tenant aggregate signals if/when published.

## First-week ingestion plan (concrete)

A specific runnable schedule for bootstrapping a fresh deployment:

| Day | Task | Yield |
|---|---|---|
| 1 | License gate + deterministic configs (top 200 repos per 12 stacks) | ~6K rules |
| 2 | AGENTS.md / CONTRIBUTING.md from Tier-B + 60K AGENTS.md universe, rate-limited | ~25K candidates |
| 3 | Style guides (~40) + ADR corpus (joelparkerhenderson + Lullabot + opendatahub) | ~4K rules + ~2K decisions |
| 4 | PR review comment mining (Tier-B repos, 24 months, GraphQL API) | ~3K rules surviving agreement |
| 5 | Cross-source agreement + confidence assignment | merged catalog |
| 6 | Stack-defaults packaging — emit per-stack JSON bundles | per-stack ship-ready bundles |
| 7 | Override mechanism + drift detection wiring | full system operational |

Day-1 customer experience on a fresh Next.js + FastAPI monorepo:

```
✓ ~400 active rules
✓ Correctly scoped by file glob
✓ Carrying rationale + source URLs
✓ Agent visibly cites "OSS consensus" vs "your team's rule"
```

## Override mechanism

Customer-supplied `AGENTS.md` / `CLAUDE.md` / `.cursorrules` at repo root **always** wins over defaults:

- Matched by rule-id where overlap exists.
- New customer rules added on top.
- Default rules contradicted by customer rules are demoted to `superseded` with reference to the customer override.

The Cartographer ([04-operations/onboarding.md](../04-operations/onboarding.md)) generates an inferred AGENTS.md from the customer's repo, presents it for review, and uses the result as the seed customer-override layer.

## Drift detection on defaults

Defaults age. Strategy:

- Every 30 days, re-extract from the seed repos.
- If a rule's cross-repo agreement drops > 20%, demote confidence.
- If a new contradictory rule passes threshold, mark the old as `drifting`.
- Customer-facing: "Your default rule X is aging; suggested update: Y."
- Maintain `last_validated` timestamp per rule; auto-archive rules unvalidated for 180 days.

## What we honestly don't solve at v1

- **Multi-language convention conflicts.** Rails conventions don't apply to FastAPI but our extraction may bleed. Mitigation: stack-tagging at extraction.
- **Anti-pattern of OSS defaults dictating taste.** Customers in unusual contexts (game dev, embedded, ML) may find OSS web-app conventions wrong. Mitigation: per-stack bundles; opt-out via empty seed-rule flag.
- **Stale corpus.** OSS practice evolves. The quarterly re-extraction handles slow drift; rapid shifts (e.g., new framework release) may need manual curation triggered.

## References

- [01-architecture/memory-layer.md](../01-architecture/memory-layer.md)
- [04-operations/onboarding.md](../04-operations/onboarding.md)
- [ADR-003: Procedural memory moat](../05-decisions/ADR-003-procedural-memory-moat.md)
