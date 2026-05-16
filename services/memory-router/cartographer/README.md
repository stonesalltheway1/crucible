# Cartographer (Phase 5 onboarding mining)

One-time-per-repo run at customer onboarding (the "Stage 2: Cartography"
flow in `docs/04-operations/onboarding.md`). Walks the repo, parses
lint configs deterministically, reads AGENTS.md / CONTRIBUTING.md / ADRs,
scans recent PR review comments, and writes the result into the
memory-router's `repo_overrides` layer.

Lives in Python because it shares the distiller's extractor + judge
pipeline. Runs as a Crucible task inside the customer's twin (the
onboarding flow spawns it like any other agent task) so it respects
the same cost cap + budget envelope.

## Quality bar

A 50K-LoC repo must complete in ≤ 30 minutes wall-clock — the
onboarding UX promise. Phase 5 measures this against a synthetic
Next.js + FastAPI fixture in `tests/test_cold_start.py`.

## Output

```
{
  "tenant_id": "ten_acme",
  "repo": "acme/payments",
  "stack": {"primary": "nextjs", "secondary": ["fastapi"], ...},
  "conventions_from_configs":     27,
  "conventions_from_agents_md":    14,
  "conventions_from_contributing":  3,
  "conventions_from_adrs":         11,
  "conventions_from_pr_review":     8,
  "conventions_from_oss_defaults":312,
  "high_confidence_count":         42,
  "medium_confidence_count":       23,
  "low_confidence_count":          12,
  "inferred_agents_md": { ... },
  "sample": [ ... up to 10 conventions for UI preview ... ]
}
```

The result lands in the memory-router via per-convention admission
calls (same path as the distiller) so the LLM-as-judge filter runs
on every cartographer write too.
