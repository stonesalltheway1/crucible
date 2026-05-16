"""GitHub PR review-comment adapter.

Production: uses the per-tenant GitHub App token to page through
review comments via GraphQL (paginated by `repository.pullRequest.reviews`)
with secondary-rate-limit-aware backoff. Phase 5 ships the offline
"corpus replay" mode (used by tests + the bootstrap corpus pipeline);
the live HTTP page-fetcher is a thin wrapper added in Phase 7
(real-time GitHub-App webhooks).
"""

from __future__ import annotations

from dataclasses import dataclass, field
from datetime import datetime, timezone
from typing import Iterable

from ..types import ExtractionInput, SourceChannel, SourceRef
from .base import Adapter


@dataclass
class GitHubPRComment:
    """A single PR review comment in the offline-corpus shape."""

    pr: int
    comment_id: str
    body: str
    author: str = ""
    created_at: datetime | None = None
    resulted_in_change: bool = False
    diff_hunk: str = ""


@dataclass
class GitHubPRAdapter:
    """Adapter consuming the offline-corpus shape.

    For tests + bootstrap pipelines; the live HTTP fetcher wraps this
    so the parsing + yielding logic is identical.
    """

    repo: str
    comments: list[GitHubPRComment] = field(default_factory=list)
    min_length: int = 20

    def name(self) -> str:
        return f"github_pr:{self.repo}"

    def iter_items(self, *, tenant_id: str, cursor: str = "") -> Iterable[ExtractionInput]:
        for c in self.comments:
            if not c.resulted_in_change:
                continue
            if len(c.body) < self.min_length:
                continue
            # The agreement filter rejects "LGTM" / bot comments; keep
            # the adapter cheap and let downstream do the heavier
            # filtering.
            body = c.body.strip()
            lower = body.lower()
            if lower in {"lgtm", "approved", "+1", "ship it"}:
                continue
            yield ExtractionInput(
                tenant_id=tenant_id,
                repo=self.repo,
                source_channel=SourceChannel.GITHUB_PR_REVIEW,
                source=SourceRef(kind="pr_comment", pr=c.pr, comment_id=c.comment_id),
                raw_text=body,
                extracted_at=c.created_at or datetime.now(timezone.utc),
            )
