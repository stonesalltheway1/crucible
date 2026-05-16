"""Fallback memory-spec types used when the editable install of
``crucible_memory_spec`` isn't on sys.path. Keeps the distiller's
test suite runnable in lightweight dev environments without forcing
the full memory-spec install. Production deployments always install
the editable package; this file is purely a fallback.
"""

from __future__ import annotations

from dataclasses import dataclass, field
from datetime import datetime, timezone
from enum import Enum


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
    GITHUB_SQUASH_MERGE = "github_squash_merge"
    INCIDENT_EXPORT = "incident_export"
    SLACK_INCIDENTS = "slack_incidents"
    CONFLUENCE_PAGE = "confluence_page"
    NOTION_PAGE = "notion_page"
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
    judge_rationale: str = ""
    judge_quarantined: bool = False
    judge_quarantine_reason: str = ""
    cross_source_agreement: float = 0.0
    cross_source_count: int = 0
    extracted_at: datetime = field(default_factory=lambda: datetime.now(timezone.utc))
    extractor_model: str = ""
    source_channel: SourceChannel | None = None
    would_supersede: str = ""


@dataclass
class JudgeVerdict:
    candidate_id: str
    quarantine: bool
    score: float
    rationale: str = ""
    judge_model: str = ""
    injection_category: str = ""
    judge_confidence: float = 0.0
