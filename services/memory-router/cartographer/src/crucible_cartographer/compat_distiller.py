"""Fallback distiller types used when the editable install of
``crucible_distiller`` isn't on sys.path. Re-exports the symbols
the scanner needs.

Production deployments always install the editable distiller package;
this file is the test-environment fallback.
"""

from __future__ import annotations

import os
import re
import sys
from dataclasses import dataclass, field
from datetime import datetime, timezone
from enum import Enum
from typing import Iterable
from uuid import uuid4


class ConventionCategory(str, Enum):
    NAMING = "Naming"
    LAYERING = "Layering"
    LIBRARY_PREFERENCES = "LibraryPreferences"
    TEST_PATTERNS = "TestPatterns"
    ERROR_HANDLING = "ErrorHandling"
    LOGGING = "Logging"
    MIGRATION_PATTERNS = "MigrationPatterns"
    PR_COMMIT_HYGIENE = "PrCommitHygiene"
    SECURITY_DEFAULTS = "SecurityDefaults"
    PERFORMANCE_DEFAULTS = "PerformanceDefaults"
    CONCURRENCY = "Concurrency"
    API_SHAPE = "ApiShape"


class SourceChannel(str, Enum):
    GITHUB_PR_REVIEW = "github_pr_review"
    ADR_FILE = "adr_file"
    LINT_CONFIG = "lint_config"


@dataclass
class SourceRef:
    kind: str
    pr: int | None = None
    comment_id: str | None = None
    id: str | None = None
    service: str | None = None
    path: str | None = None
    commit: str | None = None
    task_id: str | None = None
    step_id: str | None = None


@dataclass
class ScopeFilter:
    repo: str = ""
    file_glob: str = ""
    category: str = ""


@dataclass
class ConventionCandidate:
    id: str
    tenant_id: str
    scope: ScopeFilter
    rule_nl: str
    category: ConventionCategory
    rationale: str = ""
    evidence_quote: str = ""
    source_evidence: list[SourceRef] = field(default_factory=list)
    judge_score: float = 0.0
    cross_source_agreement: float = 0.0
    extracted_at: datetime = field(default_factory=lambda: datetime.now(timezone.utc))
    source_channel: SourceChannel | None = None


@dataclass
class ExtractionInput:
    tenant_id: str
    repo: str
    source_channel: SourceChannel
    source: SourceRef
    raw_text: str
    extracted_at: datetime = field(default_factory=lambda: datetime.now(timezone.utc))


_RULE_PATTERNS = [
    (re.compile(r"\buse\s+(slog|zap|pino|winston|logback)\b", re.I), ConventionCategory.LOGGING),
    (re.compile(r"\b(context\.context|pass context)\b", re.I), ConventionCategory.CONCURRENCY),
    (re.compile(r"\b(cursor pagination|no offset pagination)\b", re.I), ConventionCategory.PERFORMANCE_DEFAULTS),
    (re.compile(r"\b(conventional commits|semantic-release)\b", re.I), ConventionCategory.PR_COMMIT_HYGIENE),
    (re.compile(r"\b(date-fns|day\.js)\b.*\b(over|instead of)\b\s+moment", re.I), ConventionCategory.LIBRARY_PREFERENCES),
    (re.compile(r"\bauth middleware\b.*\bbefore\b", re.I), ConventionCategory.SECURITY_DEFAULTS),
    (re.compile(r"\b(additive[- ]only|never drop column)\b", re.I), ConventionCategory.MIGRATION_PATTERNS),
    (re.compile(r"\b(structured (logs|logging))\b", re.I), ConventionCategory.LOGGING),
    (re.compile(r"\b(Result<|exceptions for control flow)\b", re.I), ConventionCategory.ERROR_HANDLING),
    (re.compile(r"\btest files (end in|ending in)\b", re.I), ConventionCategory.NAMING),
    (re.compile(r"\bidempotency key\b", re.I), ConventionCategory.API_SHAPE),
    (re.compile(r"\bcolocate tests\b|\b__tests__\b", re.I), ConventionCategory.TEST_PATTERNS),
]


def deterministic_extract(text: str, repo: str = "", tenant_id: str = "", source=None):
    out = []
    for pat, cat in _RULE_PATTERNS:
        m = pat.search(text)
        if not m:
            continue
        out.append(ConventionCandidate(
            id=f"cand_{uuid4().hex}",
            tenant_id=tenant_id,
            scope=ScopeFilter(repo=repo),
            rule_nl=m.group(0).strip()[:240],
            category=cat,
            rationale="deterministic-extracted (no LLM)",
            evidence_quote=m.group(0).strip()[:240],
            source_evidence=[source] if source else [],
            source_channel=SourceChannel.LINT_CONFIG,
        ))
    return out


_LINT_CONFIG_FILES = {
    ".editorconfig", ".prettierrc", ".eslintrc.json", "tsconfig.json",
    ".rubocop.yml", "rustfmt.toml", ".golangci.yml", "pyproject.toml",
}


@dataclass
class LintConfigAdapter:
    repo_root: str
    repo: str = ""

    def name(self) -> str:
        return f"lint_config:{self.repo}"

    def iter_items(self, *, tenant_id: str, cursor: str = "") -> Iterable[ExtractionInput]:
        if not os.path.isdir(self.repo_root):
            return
        for entry in os.listdir(self.repo_root):
            if entry not in _LINT_CONFIG_FILES:
                continue
            full = os.path.join(self.repo_root, entry)
            try:
                with open(full, "r", encoding="utf-8", errors="replace") as f:
                    body = f.read()
            except OSError:
                continue
            yield ExtractionInput(
                tenant_id=tenant_id,
                repo=self.repo,
                source_channel=SourceChannel.LINT_CONFIG,
                source=SourceRef(kind="adr", path=entry),
                raw_text=body,
            )


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
                rel = os.path.relpath(full, self.root).replace(os.sep, "/")
                yield ExtractionInput(
                    tenant_id=tenant_id,
                    repo=self.repo,
                    source_channel=SourceChannel.ADR_FILE,
                    source=SourceRef(kind="adr", path=rel),
                    raw_text=body,
                )


@dataclass
class GitHubPRComment:
    pr: int
    comment_id: str
    body: str
    author: str = ""
    created_at: datetime | None = None
    resulted_in_change: bool = False
    diff_hunk: str = ""


@dataclass
class GitHubPRAdapter:
    repo: str
    comments: list = field(default_factory=list)
    min_length: int = 20

    def name(self) -> str:
        return f"github_pr:{self.repo}"

    def iter_items(self, *, tenant_id: str, cursor: str = "") -> Iterable[ExtractionInput]:
        for c in self.comments:
            if not c.resulted_in_change:
                continue
            if len(c.body) < self.min_length:
                continue
            body = c.body.strip()
            if body.lower() in {"lgtm", "approved", "+1", "ship it"}:
                continue
            yield ExtractionInput(
                tenant_id=tenant_id,
                repo=self.repo,
                source_channel=SourceChannel.GITHUB_PR_REVIEW,
                source=SourceRef(kind="pr_comment", pr=c.pr, comment_id=c.comment_id),
                raw_text=body,
            )
