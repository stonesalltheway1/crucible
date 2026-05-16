"""Drift detector — runs nightly against every active convention.

The detector reads (positives_30d, negatives_30d) from the procedural
store and emits a structured event when the ratio falls below the
configured threshold.

The action suggestion is heuristic:
  ratio < 0.5  → "archive"   (rule is actively contradicted)
  ratio < 1.0  → "supersede" (newer rule beats it)
  ratio < 1.5  → "demote"    (down to suggested, then candidate)
"""

from __future__ import annotations

from dataclasses import dataclass
from datetime import datetime, timezone


@dataclass(frozen=True)
class DriftInputs:
    convention_id: str
    tenant_id: str
    positives_30d: int
    negatives_30d: int
    threshold: float = 1.5


@dataclass(frozen=True)
class DriftEvent:
    convention_id: str
    tenant_id: str
    positives_30d: int
    negatives_30d: int
    ratio: float
    threshold: float
    detected_at: datetime
    suggested_action: str


def detect(inp: DriftInputs) -> DriftEvent | None:
    """Return a DriftEvent if the ratio is below threshold; else None.

    Insufficient data (fewer than 5 reinforcements + 0 violations) yields
    None — we don't want to flag a young rule as drifting just because
    we haven't seen it recently.
    """
    total = inp.positives_30d + inp.negatives_30d
    if total < 5:
        return None
    ratio = inp.positives_30d / max(inp.negatives_30d, 1)
    if ratio >= inp.threshold:
        return None
    action = (
        "archive" if ratio < 0.5 else
        "supersede" if ratio < 1.0 else
        "demote"
    )
    return DriftEvent(
        convention_id=inp.convention_id,
        tenant_id=inp.tenant_id,
        positives_30d=inp.positives_30d,
        negatives_30d=inp.negatives_30d,
        ratio=ratio,
        threshold=inp.threshold,
        detected_at=datetime.now(timezone.utc),
        suggested_action=action,
    )
