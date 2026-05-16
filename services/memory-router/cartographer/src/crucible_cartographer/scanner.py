"""Cartographer orchestrator.

Walks a repo, parses lint configs / AGENTS.md / CONTRIBUTING.md / ADRs,
scans recent PR review comments (when provided), and emits a single
CartographerResult that the onboarding UI consumes.
"""

from __future__ import annotations

import logging
import os
import time
from dataclasses import dataclass, field
from datetime import datetime, timezone
from typing import Iterable, Optional

from .stack_detect import StackDetection, detect as detect_stack
from .agents_md import infer_agents_md_markdown

try:  # pragma: no cover
    from crucible_distiller.adapters.lint_config import LintConfigAdapter
    from crucible_distiller.adapters.adr_file import ADRFileAdapter
    from crucible_distiller.adapters.github_pr import GitHubPRAdapter, GitHubPRComment
    from crucible_distiller.extractor.mem0_hierarchical import deterministic_extract
    from crucible_distiller.types import (
        ConventionCandidate,
        ConventionCategory,
        ExtractionInput,
        ScopeFilter,
        SourceChannel,
        SourceRef,
    )
except ImportError:  # pragma: no cover
    from .compat_distiller import (
        ADRFileAdapter, ConventionCandidate, ConventionCategory,
        ExtractionInput, GitHubPRAdapter, GitHubPRComment, LintConfigAdapter,
        ScopeFilter, SourceChannel, SourceRef, deterministic_extract,
    )

logger = logging.getLogger(__name__)


@dataclass
class CartographerJob:
    tenant_id: str
    repo: str
    repo_local_path: str
    stack_hint: str = ""
    include_pr_history: bool = True
    pr_history_months: int = 24
    pr_history_max_comments: int = 1000
    enqueued_at: datetime = field(default_factory=lambda: datetime.now(timezone.utc))
    # When provided, used in place of live GitHub fetching (the offline
    # corpus path). The distiller's GitHubPRAdapter consumes this.
    pr_comments: list = field(default_factory=list)


@dataclass
class CartographerResult:
    job_id: str
    tenant_id: str
    repo: str
    files_indexed: int
    directories: int
    stack: StackDetection
    conventions_from_configs: int = 0
    conventions_from_agents_md: int = 0
    conventions_from_contributing: int = 0
    conventions_from_adrs: int = 0
    conventions_from_pr_review: int = 0
    conventions_from_oss_defaults: int = 0
    high_confidence_count: int = 0
    medium_confidence_count: int = 0
    low_confidence_count: int = 0
    sample: list[ConventionCandidate] = field(default_factory=list)
    inferred_agents_md_markdown: str = ""
    has_customer_override: bool = False
    customer_override_path: str = ""
    started_at: datetime = field(default_factory=lambda: datetime.now(timezone.utc))
    completed_at: Optional[datetime] = None
    wall_clock_seconds: float = 0.0


_CUSTOMER_OVERRIDE_NAMES = ("AGENTS.md", "CLAUDE.md", ".cursorrules")


def scan(job: CartographerJob, *, max_files: int = 50_000) -> CartographerResult:
    """Run the cartographer over a local repo path. Returns a structured
    result that the onboarding UI displays + persists to the
    memory-router."""
    start = time.monotonic()
    root = job.repo_local_path

    files_indexed = 0
    directories = 0
    has_override = False
    override_path = ""
    if os.path.isdir(root):
        for dirpath, dirs, files in os.walk(root):
            # Skip vendored / build dirs.
            dirs[:] = [d for d in dirs if d not in {"node_modules", ".git", ".venv", "target", "dist", "build", "vendor"}]
            directories += 1
            for f in files:
                files_indexed += 1
                if files_indexed >= max_files:
                    break
                if dirpath == root and f in _CUSTOMER_OVERRIDE_NAMES:
                    has_override = True
                    override_path = f
            if files_indexed >= max_files:
                break

    stack = detect_stack(root)
    if job.stack_hint:
        stack.primary = job.stack_hint

    candidates: list[ConventionCandidate] = []
    src_counts = {"configs": 0, "agents_md": 0, "contributing": 0, "adrs": 0, "pr": 0}

    # 1. Lint configs — deterministic.
    if os.path.isdir(root):
        for ein in LintConfigAdapter(repo_root=root, repo=job.repo).iter_items(tenant_id=job.tenant_id):
            extracted = deterministic_extract(ein.raw_text, repo=ein.repo, tenant_id=ein.tenant_id, source=ein.source)
            candidates.extend(extracted)
            src_counts["configs"] += len(extracted)

    # 2. AGENTS.md / CONTRIBUTING.md
    for name in _CUSTOMER_OVERRIDE_NAMES:
        path = os.path.join(root, name)
        if os.path.isfile(path):
            text = _read(path)
            extracted = deterministic_extract(text, repo=job.repo, tenant_id=job.tenant_id,
                                              source=SourceRef(kind="adr", path=name, commit=""))
            candidates.extend(extracted)
            src_counts["agents_md"] += len(extracted)
    contrib = os.path.join(root, "CONTRIBUTING.md")
    if os.path.isfile(contrib):
        text = _read(contrib)
        extracted = deterministic_extract(text, repo=job.repo, tenant_id=job.tenant_id,
                                          source=SourceRef(kind="adr", path="CONTRIBUTING.md", commit=""))
        candidates.extend(extracted)
        src_counts["contributing"] += len(extracted)

    # 3. ADR directories.
    for adr_dir in ("docs/adr", "docs/architecture", "adr"):
        p = os.path.join(root, adr_dir)
        if os.path.isdir(p):
            for ein in ADRFileAdapter(root=p, repo=job.repo).iter_items(tenant_id=job.tenant_id):
                extracted = deterministic_extract(ein.raw_text, repo=ein.repo, tenant_id=ein.tenant_id, source=ein.source)
                candidates.extend(extracted)
                src_counts["adrs"] += len(extracted)

    # 4. PR review comments (offline corpus only — live GitHub fetch is
    #    a Phase-7 wiring).
    if job.include_pr_history and job.pr_comments:
        adapter = GitHubPRAdapter(repo=job.repo, comments=job.pr_comments)
        for ein in adapter.iter_items(tenant_id=job.tenant_id):
            extracted = deterministic_extract(ein.raw_text, repo=ein.repo, tenant_id=ein.tenant_id, source=ein.source)
            candidates.extend(extracted)
            src_counts["pr"] += len(extracted)

    # Bucket confidence: deterministic_extract produces moderate-
    # confidence candidates; the surface threshold uses a heuristic
    # based on source channel.
    hi, md, lo = _bucket_counts(candidates)

    res = CartographerResult(
        job_id=f"carto_{int(time.time() * 1000)}",
        tenant_id=job.tenant_id,
        repo=job.repo,
        files_indexed=files_indexed,
        directories=directories,
        stack=stack,
        conventions_from_configs=src_counts["configs"],
        conventions_from_agents_md=src_counts["agents_md"],
        conventions_from_contributing=src_counts["contributing"],
        conventions_from_adrs=src_counts["adrs"],
        conventions_from_pr_review=src_counts["pr"],
        conventions_from_oss_defaults=0,  # filled by the loader at install time
        high_confidence_count=hi,
        medium_confidence_count=md,
        low_confidence_count=lo,
        sample=candidates[:10],
        inferred_agents_md_markdown=infer_agents_md_markdown(stack, candidates),
        has_customer_override=has_override,
        customer_override_path=override_path,
        started_at=datetime.now(timezone.utc),
    )
    res.completed_at = datetime.now(timezone.utc)
    res.wall_clock_seconds = time.monotonic() - start
    return res


def _read(path: str) -> str:
    try:
        with open(path, "r", encoding="utf-8", errors="replace") as f:
            return f.read()
    except OSError:
        return ""


def _bucket_counts(candidates: Iterable) -> tuple[int, int, int]:
    """Heuristic bucketing for the onboarding UI. The real confidence
    score comes from the distiller pipeline; the cartographer pre-bins
    so the customer sees counts before admission completes."""
    hi = md = lo = 0
    for c in candidates:
        # Source-channel-derived prior — ADRs land high, lint configs
        # medium, raw PR comments low until corroborated.
        ch = getattr(c, "source_channel", None)
        if ch and getattr(ch, "value", "") == "adr_file":
            hi += 1
        elif ch and getattr(ch, "value", "") == "lint_config":
            md += 1
        else:
            lo += 1
    return hi, md, lo
