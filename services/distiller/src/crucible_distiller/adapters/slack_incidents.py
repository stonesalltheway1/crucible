"""Slack #incidents channel adapter.

Per-tenant Slack OAuth. Production: Slack Web API
``conversations.history`` with thread expansion. Offline test mode
takes a pre-collected JSON dump.
"""

from __future__ import annotations

from dataclasses import dataclass, field
from datetime import datetime, timezone
from typing import Iterable

from ..types import ExtractionInput, SourceChannel, SourceRef
from .base import Adapter


@dataclass
class SlackThread:
    thread_ts: str
    channel: str
    text: str
    posted_at: datetime | None = None


@dataclass
class SlackIncidentsAdapter:
    threads: list[SlackThread] = field(default_factory=list)
    workspace: str = "default"

    def name(self) -> str:
        return f"slack_incidents:{self.workspace}"

    def iter_items(self, *, tenant_id: str, cursor: str = "") -> Iterable[ExtractionInput]:
        for th in self.threads:
            text = th.text.strip()
            if len(text) < 40:
                continue
            yield ExtractionInput(
                tenant_id=tenant_id,
                repo="",
                source_channel=SourceChannel.SLACK_INCIDENTS,
                source=SourceRef(kind="incident", id=th.thread_ts, service=th.channel),
                raw_text=text,
                extracted_at=th.posted_at or datetime.now(timezone.utc),
            )
