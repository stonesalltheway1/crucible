# OSS-corpus bootstrap (Phase 5 cold-start)

Generates the per-stack default convention bundles that ship at
`services/memory-router/global_defaults/<stack>.json`. Runs offline
(it's an ingestion pipeline, not a customer-facing daemon).

```
license filter ──▶ Tier A (40 curated style guides, ~×1.5 weight)
                 ──▶ Tier B (top 200 repos × 12 stacks, lint+AGENTS.md)
                 ──▶ Tier C (PR review comment corpus, top-clustered)
                 ──▶ Tier D (ADR + post-mortem corpus, ~×1.5 weight)
                          │
                          ▼
              ┌──────────────────────────────┐
              │ deterministic extractor      │
              │  (configs, AGENTS.md sections) │
              └─────────────┬────────────────┘
                            ▼
              ┌──────────────────────────────┐
              │ LLM extractor (Haiku 4.5)    │
              │  Mem0 hierarchical pattern   │
              └─────────────┬────────────────┘
                            ▼
              ┌──────────────────────────────┐
              │ judge filter (det + LLM)     │
              └─────────────┬────────────────┘
                            ▼
              ┌──────────────────────────────┐
              │ cross-source agreement +     │
              │   Platt-scaled confidence    │
              └─────────────┬────────────────┘
                            ▼
              ┌──────────────────────────────┐
              │ counter-example pass         │
              └─────────────┬────────────────┘
                            ▼
              per-stack JSON bundle (active ≥ 0.4 surface,
              0.25..0.4 candidate, < 0.25 dropped)
```

## License filter

Allowlist: MIT, Apache-2.0, BSD-*, MPL-2.0, ISC, Unlicense.
Refused at ingestion: GPL-*, AGPL-*, SSPL-*, BUSL-*. Refusal is
recorded in the bundle's `license.excluded_licenses` for audit.

## Run

```bash
crucible-oss-bootstrap run --output services/memory-router/global_defaults/
```

For air-gapped customers who want to verify a bundle: load the JSON,
inspect each convention's `source_evidence`, run
`crucible-distiller selfcheck` against the judge corpus to confirm the
extraction pipeline is still calibrated.
