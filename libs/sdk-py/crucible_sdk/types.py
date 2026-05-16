"""Crucible SDK types — hand-rolled Phase-1 equivalent of the proto schema.

These mirror the protobuf definitions in libs/twin-spec/proto/crucible/v1/.
JSON encoding (via Pydantic's model_dump_json) matches the wire format the
control plane and signed in-toto predicates use.
"""

from __future__ import annotations

from datetime import datetime
from enum import Enum
from typing import Annotated, Literal, Union

from pydantic import BaseModel, ConfigDict, Field

# ── enums ─────────────────────────────────────────────────────────────────


class Action(str, Enum):
    ADD = "add"
    MODIFY = "modify"
    DELETE = "delete"


class Complexity(str, Enum):
    TRIVIAL = "trivial"
    STANDARD = "standard"
    COMPLEX = "complex"
    CRITICAL = "critical"
    MODERNIZATION = "modernization"


class Reversibility(str, Enum):
    TRIVIAL = "trivial"
    SNAPSHOT = "snapshot"
    LOSSY = "lossy"
    IRREVERSIBLE = "irreversible"


class TaskStatus(str, Enum):
    RECEIVED = "received"
    PLANNING = "planning"
    AWAITING_APPROVAL = "awaiting_approval"
    APPROVED = "approved"
    REJECTED = "rejected"
    EXECUTING = "executing"
    VERIFYING = "verifying"
    PROMOTING = "promoting"
    LANDED = "landed"
    ROLLED_BACK = "rolled_back"
    BUDGET_EXCEEDED = "budget_exceeded"
    RETRY_LIMIT_EXCEEDED = "retry_limit_exceeded"
    WALL_CLOCK_EXCEEDED = "wall_clock_exceeded"
    FAILED = "failed"


class ConventionStatus(str, Enum):
    ACTIVE = "active"
    DRIFTING = "drifting"
    SUPERSEDED = "superseded"
    REJECTED = "rejected"


class MemoryKind(str, Enum):
    HOT = "hot"
    EPISODIC = "episodic"
    SEMANTIC = "semantic"
    PROCEDURAL = "procedural"


class ErrorCode(str, Enum):
    BUDGET_EXCEEDED = "BudgetExceeded"
    RETRY_LIMIT_EXCEEDED = "RetryLimitExceeded"
    WALL_CLOCK_EXCEEDED = "WallClockExceeded"
    EGRESS_DENIED = "EgressDenied"
    SECRET_ACCESS_DENIED = "SecretAccessDenied"
    DESTRUCTIVE_PROPOSAL_REJECTED = "DestructiveProposalRejected"
    TWIN_SETUP_ERROR = "TwinSetupError"
    TAPE_INTEGRITY_ERROR = "TapeIntegrityError"
    VERIFIER_REJECTION = "VerifierRejection"
    PROMOTION_POLICY_DENIED = "PromotionPolicyDenied"
    APPROVAL_TIMEOUT = "ApprovalTimeout"
    CANARY_ROLLBACK = "CanaryRollback"
    TENANT_QUOTA_EXCEEDED = "TenantQuotaExceeded"
    MODEL_ROUTING_DENIED = "ModelRoutingDenied"


# ── base ──────────────────────────────────────────────────────────────────


class _Base(BaseModel):
    model_config = ConfigDict(
        extra="forbid",
        populate_by_name=True,
        ser_json_timedelta="iso8601",
    )


class Glob(_Base):
    pattern: str


class ScopeFilter(_Base):
    repo: str | None = None
    file_glob: str | None = None
    category: str | None = None


# Scope = "all" | ScopeFilter
Scope = Union[Literal["all"], ScopeFilter]


class FileChange(_Base):
    path: str
    action: Action
    content: str | None = None
    content_sha256: str | None = None
    size_bytes: int | None = None


class Diff(_Base):
    files: list[FileChange] = Field(default_factory=list)
    base_sha: str | None = None


class _SourceRefPrComment(_Base):
    kind: Literal["pr_comment"]
    pr: int
    comment_id: str


class _SourceRefIncident(_Base):
    kind: Literal["incident"]
    id: str
    service: str


class _SourceRefAdr(_Base):
    kind: Literal["adr"]
    path: str
    commit: str


class _SourceRefAgentObservation(_Base):
    kind: Literal["agent_observation"]
    task_id: str
    step_id: str


SourceRef = Annotated[
    Union[
        _SourceRefPrComment,
        _SourceRefIncident,
        _SourceRefAdr,
        _SourceRefAgentObservation,
    ],
    Field(discriminator="kind"),
]


class SecretRef(_Base):
    name: str
    handle: str
    expires_at: datetime | None = None


class ExecResult(_Base):
    stdout: str = ""
    stderr: str = ""
    exit_code: int = 0
    duration_ms: int = 0
    signed_attestation: str | None = None


class BlastRadius(_Base):
    affected_resources: list[str] = Field(default_factory=list)
    reversibility: Reversibility
    impact_score: float


class DestructiveProposal(_Base):
    task_id: str
    tenant_id: str
    command: str
    scope: Literal["twin", "real"]
    justification: str | None = None
    blast_radius: BlastRadius
    justification_required: bool = True
    intercepted_at_layer: Literal["syscall-shim", "cmd-line-parse", "ebpf"]
    proposed_at: datetime
    agent_oidc_subject: str


class ExternalEffect(_Base):
    service: str
    endpoints: list[str]
    live: bool = False


class Risk(_Base):
    description: str
    impact: Literal["low", "medium", "high"]


class PlanStep(_Base):
    ordinal: int
    description: str
    retry_budget: int = 3
    retries_used: int = 0


class Plan(_Base):
    task_id: str
    description: str
    steps: list[PlanStep] = Field(default_factory=list)
    estimated_cost_usd: float
    estimated_duration_min: int
    files_to_touch: list[str] = Field(default_factory=list)
    db_migrations: int = 0
    external_effects: list[ExternalEffect] = Field(default_factory=list)
    top_risks: list[Risk] = Field(default_factory=list)
    retry_budget_per_step: int = 3
    wall_clock_budget_min: int = 60
    complexity: Complexity
    plan_hash: str
    built_at: datetime


class PlanApproval(_Base):
    task_id: str
    plan_hash: str
    approver_oidc_subject: str
    approved_at: datetime
    attestation_id: str | None = None
    cost_cap_usd: float
    wall_clock_cap_min: int
    retry_cap_per_subgoal: int


class PlanRejection(_Base):
    task_id: str
    plan_hash: str
    reason: str
    rejecter_oidc_subject: str
    rejected_at: datetime


class Routing(_Base):
    executor_model: str
    executor_vendor: str
    executor_tier: int
    verifier_model: str
    verifier_vendor: str
    verifier_tier: int
    critical_score: float = 0.0
    is_critical: bool = False
    decided_at: datetime
    classifier_attestation_id: str | None = None


class Budget(_Base):
    spent_usd: float = 0.0
    cap_usd: float
    steps_used: int = 0
    steps_cap: int = 0
    wall_clock_used_seconds: int = 0
    wall_clock_cap_seconds: int = 0
    retries_used: int = 0
    retry_cap: int = 0


class Task(_Base):
    id: str
    tenant_id: str
    repo: str
    base_sha: str
    description: str
    status: TaskStatus
    created_at: datetime
    updated_at: datetime
    submitted_by: str
    plan: Plan | None = None
    routing: Routing | None = None
    budget: Budget | None = None
    related_task_ids: list[str] = Field(default_factory=list)


class Convention(_Base):
    id: str
    tenant_id: str
    scope: ScopeFilter
    rule_nl: str
    category: str
    status: ConventionStatus
    confidence: float = 0.0
    judge_score: float = 0.0
    source_evidence: list[SourceRef] = Field(default_factory=list)
    valid_from: datetime
    valid_to: datetime | None = None
    supersedes: str | None = None
    writer_oidc_subject: str
    written_at: datetime


# ── predicate-type URIs ───────────────────────────────────────────────────


class Predicates(str, Enum):
    WRITE = "https://crucible.dev/WriteAttestation/v1"
    MIGRATION = "https://crucible.dev/MigrationAttestation/v1"
    SERVICE_CALL = "https://crucible.dev/ServiceCallAttestation/v1"
    DESTRUCTIVE_PROPOSAL = "https://crucible.dev/DestructiveProposal/v1"
    DESTRUCTIVE_APPROVAL = "https://crucible.dev/DestructiveApproval/v1"
    TEST_REPORT = "https://crucible.dev/TestReport/v1"
    VERIFIER_APPROVAL = "https://crucible.dev/VerifierApproval/v1"
    VERIFIER_REJECTION = "https://crucible.dev/VerifierRejection/v1"
    PLAN_PROPOSAL = "https://crucible.dev/PlanProposal/v1"
    PLAN_APPROVAL = "https://crucible.dev/PlanApproval/v1"
    PROMOTION_BUNDLE = "https://crucible.dev/PromotionBundle/v1"
    PROMOTION_APPROVAL = "https://crucible.dev/PromotionApproval/v1"
    PROMOTION_OUTCOME = "https://crucible.dev/PromotionOutcome/v1"
    MEMORY_WRITE = "https://crucible.dev/MemoryWrite/v1"
