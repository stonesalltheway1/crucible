"""Incident-export adapter (Rootly / FireHydrant / Jeli / Incident.io).

All four vendors expose a JSON export shape that we normalise to the
``Incident`` dataclass below. The distiller treats incident
post-mortems as a high-weight signal because they encode "never do X
again" patterns that ADRs don't always capture.
"""

from __future__ import annotations

from dataclasses import dataclass, field
from datetime import datetime, timezone
from typing import Iterable

from ..types import ExtractionInput, SourceChannel, SourceRef
from .base import Adapter


@dataclass
class Incident:
    id: str
    service: str
    summary: str
    post_mortem: str
    resolved_at: datetime | None = None


@dataclass
class IncidentAdapter:
    """Streams normalised incidents into the distiller."""

    incidents: list[Incident] = field(default_factory=list)
    vendor: str = "generic"

    def name(self) -> str:
        return f"incident:{self.vendor}"

    def iter_items(self, *, tenant_id: str, cursor: str = "") -> Iterable[ExtractionInput]:
        for inc in self.incidents:
            text = (inc.summary + "\n\n" + inc.post_mortem).strip()
            if not text:
                continue
            yield ExtractionInput(
                tenant_id=tenant_id,
                repo="",  # incidents are org-scoped, not repo-scoped
                source_channel=SourceChannel.INCIDENT_EXPORT,
                source=SourceRef(kind="incident", id=inc.id, service=inc.service),
                raw_text=text,
                extracted_at=inc.resolved_at or datetime.now(timezone.utc),
            )
