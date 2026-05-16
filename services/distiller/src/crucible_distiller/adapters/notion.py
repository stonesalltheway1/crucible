"""Notion runbook + ADR-page adapter."""

from __future__ import annotations

from dataclasses import dataclass, field
from datetime import datetime, timezone
from typing import Iterable

from ..types import ExtractionInput, SourceChannel, SourceRef
from .base import Adapter


@dataclass
class NotionPage:
    page_id: str
    title: str
    body_markdown: str
    updated_at: datetime | None = None


@dataclass
class NotionAdapter:
    pages: list[NotionPage] = field(default_factory=list)
    workspace: str = ""

    def name(self) -> str:
        return f"notion:{self.workspace}"

    def iter_items(self, *, tenant_id: str, cursor: str = "") -> Iterable[ExtractionInput]:
        for p in self.pages:
            text = (p.title + "\n\n" + p.body_markdown).strip()
            if len(text) < 80:
                continue
            yield ExtractionInput(
                tenant_id=tenant_id,
                repo="",
                source_channel=SourceChannel.NOTION_PAGE,
                source=SourceRef(kind="adr", path=f"notion/{p.page_id}", commit=""),
                raw_text=text,
                extracted_at=p.updated_at or datetime.now(timezone.utc),
            )
