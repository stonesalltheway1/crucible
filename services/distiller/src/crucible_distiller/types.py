"""Distiller-internal types.

The Convention / ConventionCandidate / JudgeVerdict / etc.
dataclasses live in ``libs/memory-spec/py``; this module imports
them and adds a few orchestration-only types.
"""

from __future__ import annotations

from dataclasses import dataclass, field
from datetime import datetime
from typing import Optional

# memory-spec is installed in production via the `spec` extra; if the
# import fails (lightweight dev/test mode without the editable install)
# we provide a minimal local fallback so the distiller's tests still
# exercise its logic.
try:  # pragma: no cover
    from crucible_memory_spec import (
        ConventionCandidate,
        ConventionCategory,
        JudgeVerdict,
        SourceChannel,
        SourceRef,
        ScopeFilter,
    )
except ImportError:  # pragma: no cover
    from .compat_types import (  # type: ignore[no-redef]
        ConventionCandidate,
        ConventionCategory,
        JudgeVerdict,
        SourceChannel,
        SourceRef,
        ScopeFilter,
    )


@dataclass
class DistillerConfig:
    """Top-level distiller daemon config."""

    memory_router_addr: str = "http://127.0.0.1:8090"
    extractor_model: str = "claude-haiku-4-5-20251001"
    judge_model: str = "claude-haiku-4-5-20251001"
    # Catch-rate gate: any deployment whose adversarial-corpus score
    # drops below this number fails the distiller's `crucible-distiller
    # selfcheck` boot probe.
    judge_min_catch_rate: float = 0.99
    # Per-tenant rate limits (per 5-minute window, per 24h).
    rate_limit_per_5m: int = 200
    rate_limit_per_24h: int = 5_000
    # Confidence threshold below which candidates land in the
    # invisible CANDIDATE bucket; ≥ 0.4 surfaces as SUGGESTED; ≥ 0.7
    # is ACTIVE.
    surface_threshold: float = 0.40
    active_threshold: float = 0.70
    # Drift detector ratio threshold.
    drift_ratio_threshold: float = 1.5


@dataclass
class ExtractionInput:
    """One upstream-channel item ready for the extractor."""

    tenant_id: str
    repo: str
    source_channel: SourceChannel
    source: SourceRef
    raw_text: str
    extracted_at: datetime = field(default_factory=datetime.utcnow)


@dataclass
class ExtractionResult:
    """The extractor's output for one input."""

    candidates: list[ConventionCandidate] = field(default_factory=list)
    extractor_model: str = ""
    input_tokens: int = 0
    output_tokens: int = 0
    cost_usd: float = 0.0
    null_extraction_reason: str = ""


@dataclass
class AdmissionDecision:
    """The composite decision the admission pipeline makes."""

    admitted: bool
    convention_id: Optional[str]
    quarantine_reason: str = ""
    injection_category: str = ""
    judge_score: float = 0.0
    confidence: float = 0.0


@dataclass
class AggregatedSignal:
    """A multi-source aggregate for the cross-source-agreement calculator."""

    canonical_form: str
    sources_seen: int = 0
    distinct_repos: int = 0
    distinct_authors: int = 0
    positives: int = 0
    negatives: int = 0
