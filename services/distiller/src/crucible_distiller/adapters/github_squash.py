"""GitHub squash-merge commit-message adapter.

Squash messages encode the team's "intent" sentence — a high-quality
distillation source (per docs/01-architecture/memory-layer.md
§"Inputs", squash-merges are tied for top weight with ADRs).
"""

from __future__ import annotations

from dataclasses import dataclass, field
from datetime import datetime, timezone
from typing import Iterable

from ..types import ExtractionInput, SourceChannel, SourceRef
from .base import Adapter


@dataclass
class GitHubSquashCommit:
    pr: int
    commit_sha: str
    message: str
    merged_at: datetime | None = None


@dataclass
class GitHubSquashAdapter:
    repo: str
    commits: list[GitHubSquashCommit] = field(default_factory=list)

    def name(self) -> str:
        return f"github_squash:{self.repo}"

    def iter_items(self, *, tenant_id: str, cursor: str = "") -> Iterable[ExtractionInput]:
        for c in self.commits:
            if not c.message.strip():
                continue
            yield ExtractionInput(
                tenant_id=tenant_id,
                repo=self.repo,
                source_channel=SourceChannel.GITHUB_SQUASH_MERGE,
                source=SourceRef(kind="pr_comment", pr=c.pr, comment_id=c.commit_sha),
                raw_text=c.message,
                extracted_at=c.merged_at or datetime.now(timezone.utc),
            )
