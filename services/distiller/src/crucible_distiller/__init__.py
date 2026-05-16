"""Crucible procedural-memory distiller — Phase 5 background worker.

Turns upstream signals (PR review comments, ADRs, incident
post-mortems, Slack channels, runbooks, merge commits) into
schema-validated Convention candidates, runs them through the
LLM-as-judge filter, computes confidence + admission scores, and
calls the memory-router's admission API.

The distiller is **not** on the agent hot path. Latency target:
"PR merged → rule lands in graph" ≤ 5 min p95.
"""

from .types import (
    AdmissionDecision,
    AggregatedSignal,
    DistillerConfig,
    ExtractionInput,
    ExtractionResult,
)

__all__ = [
    "AdmissionDecision",
    "AggregatedSignal",
    "DistillerConfig",
    "ExtractionInput",
    "ExtractionResult",
]
