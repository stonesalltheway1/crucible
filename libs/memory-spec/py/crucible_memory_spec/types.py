"""Disk-serialized memory-spec types.

Hand-rolled dataclasses in lock-step with
``libs/memory-spec/proto/crucible/v1/memory_layer.proto`` and
``distiller.proto``. JSON tags match the schemas in
``libs/memory-spec/schemas/``.
"""

from __future__ import annotations

from dataclasses import dataclass, field
from datetime import datetime, timezone
from enum import Enum
from typing import Any, Optional

from .errors import InvalidConventionError, LicenseUnsafeBundleError


# ─── Layering ───────────────────────────────────────────────────────────────


class MemoryLayer(str, Enum):
    GLOBAL_DEFAULTS = "global_defaults"
    ORG_OVERRIDES = "org_overrides"
    REPO_OVERRIDES = "repo_overrides"

    @property
    def priority(self) -> int:
        return {
            MemoryLayer.GLOBAL_DEFAULTS: 1,
            MemoryLayer.ORG_OVERRIDES: 2,
            MemoryLayer.REPO_OVERRIDES: 3,
        }[self]


# ─── Convention taxonomy ────────────────────────────────────────────────────


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


ALL_CATEGORIES: list[ConventionCategory] = list(ConventionCategory)


def valid_category(name: str) -> bool:
    """Whether ``name`` is one of the 12 taxonomy buckets.

    Admission rejects extractions with ``category="other"`` or anything
    not in this list — the brief calls this out as a hard gate.
    """
    try:
        ConventionCategory(name)
        return True
    except ValueError:
        return False


class ConventionStatus(str, Enum):
    ACTIVE = "active"
    DRIFTING = "drifting"
    SUPERSEDED = "superseded"
    REJECTED = "rejected"
    # Server-internal pre-admission buckets — not surfaced via SDK.
    CANDIDATE = "candidate"
    SUGGESTED = "suggested"


# ─── Stacks ─────────────────────────────────────────────────────────────────


class Stack(str, Enum):
    NEXTJS = "nextjs"
    DJANGO = "django"
    FASTAPI = "fastapi"
    FLASK = "flask"
    RAILS = "rails"
    SPRING_BOOT = "spring_boot"
    GO_SERVICES = "go_services"
    RUST_SERVICES = "rust_services"
    PHOENIX_ELIXIR = "phoenix_elixir"
    VUE = "vue"
    EXPRESS = "express"
    LARAVEL = "laravel"


ALL_STACKS: list[Stack] = list(Stack)


# ─── Source channels ────────────────────────────────────────────────────────


class SourceChannel(str, Enum):
    GITHUB_PR_REVIEW = "github_pr_review"
    GITHUB_SQUASH_MERGE = "github_squash_merge"
    INCIDENT_EXPORT = "incident_export"
    SLACK_INCIDENTS = "slack_incidents"
    CONFLUENCE_PAGE = "confluence_page"
    NOTION_PAGE = "notion_page"
    ADR_FILE = "adr_file"
    LINT_CONFIG = "lint_config"


# ─── Source refs ────────────────────────────────────────────────────────────


@dataclass
class SourceRef:
    kind: str  # "pr_comment" | "incident" | "adr" | "agent_observation"
    pr: int | None = None
    comment_id: str | None = None
    id: str | None = None
    service: str | None = None
    path: str | None = None
    commit: str | None = None
    task_id: str | None = None
    step_id: str | None = None

    def to_dict(self) -> dict[str, Any]:
        out: dict[str, Any] = {"kind": self.kind}
        for k in ("pr", "comment_id", "id", "service", "path", "commit",
                  "task_id", "step_id"):
            v = getattr(self, k)
            if v is not None:
                out[k] = v
        return out


@dataclass
class ScopeFilter:
    repo: str = ""
    file_glob: str = ""
    category: str = ""

    def to_dict(self) -> dict[str, Any]:
        out: dict[str, Any] = {}
        if self.repo:
            out["repo"] = self.repo
        if self.file_glob:
            out["file_glob"] = self.file_glob
        if self.category:
            out["category"] = self.category
        return out


# ─── Convention (disk shape) ───────────────────────────────────────────────


@dataclass
class Convention:
    id: str
    tenant_id: str
    scope: ScopeFilter
    rule_nl: str
    category: ConventionCategory
    status: ConventionStatus
    confidence: float
    valid_from: datetime
    written_at: datetime
    layer: Optional[MemoryLayer] = None
    rule_machine: Optional[str] = None
    judge_score: float = 0.0
    judge_rationale: str = ""
    positive_examples: list[SourceRef] = field(default_factory=list)
    negative_examples: list[SourceRef] = field(default_factory=list)
    source_evidence: list[SourceRef] = field(default_factory=list)
    first_seen: Optional[datetime] = None
    last_reinforced: Optional[datetime] = None
    last_violated: Optional[datetime] = None
    valid_to: Optional[datetime] = None
    supersedes: list[str] = field(default_factory=list)
    writer_oidc_subject: str = ""
    stack_tag: str = ""
    anonymized_form: str = ""

    def validate(self) -> None:
        if not self.id:
            raise InvalidConventionError("convention.id required")
        if not self.tenant_id:
            raise InvalidConventionError(f"convention {self.id}: tenant_id required")
        if not self.rule_nl or len(self.rule_nl) > 1024:
            raise InvalidConventionError(
                f"convention {self.id}: rule_nl must be 1..1024 chars"
            )
        if not valid_category(self.category.value if isinstance(self.category, ConventionCategory) else str(self.category)):
            raise InvalidConventionError(
                f"convention {self.id}: invalid category {self.category!r} "
                f"(must be one of the 12 taxonomy buckets)"
            )
        if not (0.0 <= self.confidence <= 1.0):
            raise InvalidConventionError(
                f"convention {self.id}: confidence {self.confidence} out of [0,1]"
            )
        if not (0.0 <= self.judge_score <= 1.0):
            raise InvalidConventionError(
                f"convention {self.id}: judge_score {self.judge_score} out of [0,1]"
            )

    def to_dict(self) -> dict[str, Any]:
        out: dict[str, Any] = {
            "id": self.id,
            "tenant_id": self.tenant_id,
            "scope": self.scope.to_dict(),
            "rule_nl": self.rule_nl,
            "category": self.category.value if isinstance(self.category, ConventionCategory) else self.category,
            "status": self.status.value if isinstance(self.status, ConventionStatus) else self.status,
            "confidence": self.confidence,
            "valid_from": _iso(self.valid_from),
            "written_at": _iso(self.written_at),
        }
        if self.layer:
            out["layer"] = self.layer.value
        if self.judge_score:
            out["judge_score"] = self.judge_score
        if self.judge_rationale:
            out["judge_rationale"] = self.judge_rationale
        if self.source_evidence:
            out["source_evidence"] = [s.to_dict() for s in self.source_evidence]
        if self.positive_examples:
            out["positive_examples"] = [s.to_dict() for s in self.positive_examples]
        if self.negative_examples:
            out["negative_examples"] = [s.to_dict() for s in self.negative_examples]
        if self.first_seen:
            out["first_seen"] = _iso(self.first_seen)
        if self.last_reinforced:
            out["last_reinforced"] = _iso(self.last_reinforced)
        if self.last_violated:
            out["last_violated"] = _iso(self.last_violated)
        if self.valid_to:
            out["valid_to"] = _iso(self.valid_to)
        if self.supersedes:
            out["supersedes"] = list(self.supersedes)
        if self.writer_oidc_subject:
            out["writer_oidc_subject"] = self.writer_oidc_subject
        if self.stack_tag:
            out["stack_tag"] = self.stack_tag
        if self.anonymized_form:
            out["anonymized_form"] = self.anonymized_form
        if self.rule_machine:
            out["rule_machine"] = self.rule_machine
        return out


# ─── Candidate + judge verdict (distiller-internal) ────────────────────────


@dataclass
class JudgeVerdict:
    candidate_id: str
    quarantine: bool
    score: float
    rationale: str = ""
    judge_model: str = ""
    injection_category: str = ""
    judge_confidence: float = 0.0


@dataclass
class ConventionCandidate:
    id: str
    tenant_id: str
    scope: ScopeFilter
    rule_nl: str
    category: ConventionCategory
    rationale: str
    evidence_quote: str
    source_evidence: list[SourceRef] = field(default_factory=list)
    judge_score: float = 0.0
    judge_rationale: str = ""
    judge_quarantined: bool = False
    judge_quarantine_reason: str = ""
    cross_source_agreement: float = 0.0
    cross_source_count: int = 0
    extracted_at: datetime = field(default_factory=lambda: datetime.now(timezone.utc))
    extractor_model: str = ""
    source_channel: Optional[SourceChannel] = None
    would_supersede: str = ""


# ─── Admission + drift ─────────────────────────────────────────────────────


@dataclass
class AdmissionScore:
    utility: float
    confidence: float
    novelty: float
    recency: float
    content_prior: float
    composite: float
    admitted: bool
    admission_threshold_label: str


@dataclass
class ConventionDrift:
    convention_id: str
    tenant_id: str
    positives_30d: int
    negatives_30d: int
    ratio: float
    threshold: float
    detected_at: datetime
    suggested_action: str  # "demote"|"supersede"|"archive"


MIN_TENANTS_FOR_GRADUATION = 5


@dataclass
class FederationGraduation:
    anonymized_rule_id: str
    category: ConventionCategory
    canonical_form_nl: str
    distinct_tenant_count: int
    contributing_convention_ids: list[str] = field(default_factory=list)
    eligible_at: datetime = field(default_factory=lambda: datetime.now(timezone.utc))
    fired: bool = False
    promoted_to_layer: str = ""


# ─── Cartographer ──────────────────────────────────────────────────────────


@dataclass
class InferredAgentsMd:
    tenant_id: str
    repo: str
    content_markdown: str
    rule_count: int
    generated_at: datetime
    job_id: str = ""
    source_summary: str = ""
    review_status: str = "draft"


# ─── Per-stack bundle ──────────────────────────────────────────────────────


@dataclass
class BundleLicense:
    safe_for_redistribution: bool
    input_licenses_seen: list[str] = field(default_factory=list)
    excluded_licenses: list[str] = field(default_factory=list)
    attribution_file: str = ""


@dataclass
class BundleStats:
    repos_examined: int = 0
    configs_parsed: int = 0
    agents_md_parsed: int = 0
    pr_comments_mined: int = 0
    adrs_parsed: int = 0
    raw_candidates: int = 0
    post_judge: int = 0
    post_agreement: int = 0
    active_rules: int = 0
    suggested_rules: int = 0
    candidate_rules: int = 0


@dataclass
class PerStackBundle:
    bundle_version: str
    stack: Stack
    generated_at: datetime
    license: BundleLicense
    conventions: list[Convention] = field(default_factory=list)
    stats: BundleStats = field(default_factory=BundleStats)
    generator_commit: str = ""

    def validate(self) -> None:
        if self.bundle_version != "1":
            raise InvalidConventionError(
                f"bundle: only bundle_version=1 supported, got {self.bundle_version!r}"
            )
        if not self.license.safe_for_redistribution:
            raise LicenseUnsafeBundleError(
                f"bundle {self.stack.value}: refused to ship — "
                f"license inputs included {self.license.excluded_licenses!r}"
            )
        for i, c in enumerate(self.conventions):
            c.validate()
            if c.layer is not None and c.layer != MemoryLayer.GLOBAL_DEFAULTS:
                raise InvalidConventionError(
                    f"bundle {self.stack.value} convention[{i}]: must be "
                    f"layer=global_defaults, got {c.layer!r}"
                )

    def to_dict(self) -> dict[str, Any]:
        return {
            "bundle_version": self.bundle_version,
            "stack": self.stack.value,
            "generated_at": _iso(self.generated_at),
            "generator_commit": self.generator_commit,
            "license": {
                "safe_for_redistribution": self.license.safe_for_redistribution,
                "input_licenses_seen": list(self.license.input_licenses_seen),
                "excluded_licenses": list(self.license.excluded_licenses),
                "attribution_file": self.license.attribution_file,
            },
            "stats": {
                "repos_examined": self.stats.repos_examined,
                "configs_parsed": self.stats.configs_parsed,
                "agents_md_parsed": self.stats.agents_md_parsed,
                "pr_comments_mined": self.stats.pr_comments_mined,
                "adrs_parsed": self.stats.adrs_parsed,
                "raw_candidates": self.stats.raw_candidates,
                "post_judge": self.stats.post_judge,
                "post_agreement": self.stats.post_agreement,
                "active_rules": self.stats.active_rules,
                "suggested_rules": self.stats.suggested_rules,
                "candidate_rules": self.stats.candidate_rules,
            },
            "conventions": [c.to_dict() for c in self.conventions],
        }


def _iso(t: datetime) -> str:
    """ISO-8601 with Z suffix matching the JSON-Schema format=date-time."""
    if t.tzinfo is None:
        t = t.replace(tzinfo=timezone.utc)
    return t.astimezone(timezone.utc).strftime("%Y-%m-%dT%H:%M:%S.%fZ")
