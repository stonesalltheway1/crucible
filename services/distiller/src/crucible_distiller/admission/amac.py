"""A-MAC (Adaptive Multi-dimensional Admission Control) scoring.

Composite = utility × confidence × novelty × recency × content_prior.

  utility       — domain prior on category (security/migration > naming)
  confidence    — Platt-scaled cross-source agreement
  novelty       — penalty for near-duplicates already in the tenant's graph
  recency       — boost for recently-seen evidence (Ebbinghaus shape)
  content_prior — per-source-channel prior (ADR > squash-merge > PR comment)

Threshold labels (from docs/06-research/memory-bootstrap.md):
  ≥ 0.70  active
  ≥ 0.40  suggested
  ≥ 0.25  candidate (invisible bucket; not surfaced)
  <  0.25 rejected
"""

from __future__ import annotations

import math
from dataclasses import dataclass
from datetime import datetime, timezone

from ..types import ConventionCategory, SourceChannel


@dataclass
class AdmissionInput:
    """Inputs to the A-MAC composite."""

    category: ConventionCategory
    confidence: float
    judge_score: float
    novelty: float = 1.0
    last_evidence_at: datetime | None = None
    source_channel: SourceChannel | None = None


# Category-level utility prior — security / migration / API rules are
# higher-value than naming nits because customers care more about
# preventing regressions there.
_CATEGORY_UTILITY: dict[ConventionCategory, float] = {
    ConventionCategory.SECURITY_DEFAULTS: 1.0,
    ConventionCategory.MIGRATION_PATTERNS: 1.0,
    ConventionCategory.API_SHAPE: 0.95,
    ConventionCategory.ERROR_HANDLING: 0.90,
    ConventionCategory.CONCURRENCY: 0.90,
    ConventionCategory.PERFORMANCE_DEFAULTS: 0.85,
    ConventionCategory.LIBRARY_PREFERENCES: 0.80,
    ConventionCategory.LOGGING: 0.78,
    ConventionCategory.TEST_PATTERNS: 0.78,
    ConventionCategory.LAYERING: 0.80,
    ConventionCategory.PR_COMMIT_HYGIENE: 0.65,
    ConventionCategory.NAMING: 0.60,
}


# Per-source-channel prior — ADR > squash > PR comment > slack > runbook.
_CHANNEL_PRIOR: dict[SourceChannel, float] = {
    SourceChannel.ADR_FILE: 1.0,
    SourceChannel.GITHUB_SQUASH_MERGE: 0.85,
    SourceChannel.INCIDENT_EXPORT: 0.85,
    SourceChannel.GITHUB_PR_REVIEW: 0.80,
    SourceChannel.LINT_CONFIG: 0.95,        # deterministic; high prior
    SourceChannel.CONFLUENCE_PAGE: 0.70,
    SourceChannel.NOTION_PAGE: 0.70,
    SourceChannel.SLACK_INCIDENTS: 0.60,
}


def admit_score(inp: AdmissionInput) -> tuple[float, str]:
    """Return (composite_score, threshold_label).

    label ∈ {"active", "suggested", "candidate", "rejected"}.
    """
    utility = _CATEGORY_UTILITY.get(inp.category, 0.60)
    confidence = _clamp(inp.confidence)
    novelty = _clamp(inp.novelty)
    recency = _recency_factor(inp.last_evidence_at)
    prior = _CHANNEL_PRIOR.get(inp.source_channel, 0.70) if inp.source_channel else 0.70

    # judge_score gates: a low judge_score forces a low composite even
    # if every other factor is high. (multiplicative).
    judge = _clamp(inp.judge_score)
    composite = utility * confidence * novelty * recency * prior * judge
    composite = _clamp(composite)

    if composite >= 0.70:
        return composite, "active"
    if composite >= 0.40:
        return composite, "suggested"
    if composite >= 0.25:
        return composite, "candidate"
    return composite, "rejected"


def _recency_factor(last: datetime | None) -> float:
    if last is None:
        return 0.6
    age_days = (datetime.now(timezone.utc) - last).total_seconds() / 86400.0
    if age_days < 0:
        age_days = 0
    # τ = 30 days for distiller intake; freshly-corroborated rules win.
    return math.exp(-age_days / 30.0)


def _clamp(v: float) -> float:
    if math.isnan(v):
        return 0.0
    if v < 0:
        return 0.0
    if v > 1:
        return 1.0
    return v
