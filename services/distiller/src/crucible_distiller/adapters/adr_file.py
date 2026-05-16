"""Filesystem ADR adapter.

Walks an `adr/`, `docs/adr/`, or `docs/architecture/` directory and
yields each `.md` file as one extraction input. Used by:

  - The distiller for tenants who keep ADRs in-repo.
  - The cartographer at onboarding time.
  - The OSS-corpus-bootstrap pipeline.
"""

from __future__ import annotations

import os
from dataclasses import dataclass
from datetime import datetime, timezone
from typing import Iterable

from ..types import ExtractionInput, SourceChannel, SourceRef
from .base import Adapter


@dataclass
class ADRFileAdapter:
    root: str
    repo: str = ""

    def name(self) -> str:
        return f"adr:{self.root}"

    def iter_items(self, *, tenant_id: str, cursor: str = "") -> Iterable[ExtractionInput]:
        if not os.path.isdir(self.root):
            return
        for dirpath, _dirs, files in os.walk(self.root):
            for name in files:
                if not name.lower().endswith((".md", ".markdown")):
                    continue
                full = os.path.join(dirpath, name)
                try:
                    with open(full, "r", encoding="utf-8") as f:
                        body = f.read()
                except OSError:
                    continue
                if len(body) < 80:
                    continue
                rel = os.path.relpath(full, self.root).replace(os.sep, "/")
                yield ExtractionInput(
                    tenant_id=tenant_id,
                    repo=self.repo,
                    source_channel=SourceChannel.ADR_FILE,
                    source=SourceRef(kind="adr", path=rel, commit=""),
                    raw_text=body,
                    extracted_at=datetime.now(timezone.utc),
                )
