# distiller (Phase 5 background worker)

Python service that turns upstream signals (PR review comments, ADRs,
incident post-mortems, Slack #incidents, merge commits) into validated
`Convention` candidates and feeds them through the LLM-as-judge filter
into the memory-router's admission API.

```
Source channels ──▶ Queue (Kafka / SQS) ──▶ Distiller pool (Haiku 4.5)
                                                │
                                                ▼
                              ┌──────────────────────────────────┐
                              │ extractor (Mem0 hierarchical,    │
                              │  schema-constrained via outlines)│
                              └─────────────────┬────────────────┘
                                                ▼
                              ┌──────────────────────────────────┐
                              │ judge — LLM-as-judge + det.      │
                              │  pre-filter (prompt injection)   │
                              └─────────────────┬────────────────┘
                                                ▼
                              ┌──────────────────────────────────┐
                              │ confidence (cross-source         │
                              │  agreement + Platt scaling)      │
                              └─────────────────┬────────────────┘
                                                ▼
                              ┌──────────────────────────────────┐
                              │ admission (A-MAC scoring) ──▶    │
                              │  POST /v1/memory/admit_convention│
                              └─────────────────┬────────────────┘
                                                ▼
                                       Procedural memory graph
```

## Packages

```
src/crucible_distiller/
  __init__.py
  cli.py                  uvicorn-driven daemon + one-off CLI
  adapters/
    __init__.py
    github_pr.py          PR review-comment adapter (GraphQL + REST)
    github_squash.py      Merge-commit adapter
    incident.py           Rootly / FireHydrant / Jeli / Incident.io
    slack_incidents.py    Slack #incidents channel scraper
    confluence.py         Confluence runbook adapter
    notion.py             Notion runbook adapter
    adr_file.py           Filesystem ADR adapter (also used by cartographer)
  extractor/
    __init__.py
    mem0_hierarchical.py  Single-pass extraction algorithm
    schema.py             AdaKGC-pattern JSON-schema for outputs
    prompts.py
  judge/
    __init__.py
    deterministic.py      Keyword pre-filter (mirrors memory-router's)
    llm_judge.py          Haiku-4.5 second-pass judge
    adversarial_corpus.py Test corpus for catch-rate audit
  confidence/
    cross_source.py       Distinct-repos / distinct-authors aggregation
    platt.py              Calibrated confidence
  drift/
    detector.py           30-day rolling pos/neg ratio
  admission/
    amac.py               A-MAC scoring + threshold labels
    client.py             memory-router admission HTTP client
  queue/
    consumer.py           Kafka + SQS abstraction
  runs/
    audit.py              distiller_runs row writer

tests/
  test_extractor.py
  test_judge_corpus.py        ≥ 99% catch rate gate
  test_drift_detector.py
  test_cross_tenant_isolation.py
  test_e2e_pr_corpus.py
```

## Wire format

Every admission call hits `POST /v1/memory/admit_convention` on the
memory-router with the tenant_id, the constructed `Convention` body,
and the originating `SourceRef`. The router runs ITS OWN deterministic
judge AGAIN as defence in depth — never trust a single layer of judge.

## Operational

The distiller is **not on the agent hot path**. Latency target is "PR
merged → rule landed in graph" ≤ 5 minutes p95 — sufficient for the
next agent task to benefit from the new rule.

Per-tenant rate-limit: 200 candidates / 5 min, 5K candidates / 24 h.
Burst handling defers to the queue (Kafka retention, SQS DLQ).
