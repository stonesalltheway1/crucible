"""Confluence runbook adapter."""

from __future__ import annotations

from dataclasses import dataclass, field
from datetime import datetime, timezone
from typing import Iterable

from ..types import ExtractionInput, SourceChannel, SourceRef
from .base import Adapter


@dataclass
class ConfluencePage:
    page_id: str
    space: str
    title: str
    body_markdown: str
    updated_at: datetime | None = None


@dataclass
class ConfluenceAdapter:
    pages: list[ConfluencePage] = field(default_factory=list)
    base_url: str = ""

    def name(self) -> str:
        return f"confluence:{self.base_url}"

    def iter_items(self, *, tenant_id: str, cursor: str = "") -> Iterable[ExtractionInput]:
        for p in self.pages:
            text = (p.title + "\n\n" + p.body_markdown).strip()
            if len(text) < 80:
                continue
            yield ExtractionInput(
                tenant_id=tenant_id,
                repo="",
                source_channel=SourceChannel.CONFLUENCE_PAGE,
                source=SourceRef(kind="adr", path=f"confluence/{p.space}/{p.page_id}", commit=""),
                raw_text=text,
                extracted_at=p.updated_at or datetime.now(timezone.utc),
            )
