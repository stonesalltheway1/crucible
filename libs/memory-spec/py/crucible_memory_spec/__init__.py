"""Crucible memory-spec: server-internal types for the Phase-5 memory layer.

This package consolidates the server-internal Python types that the
distiller (services/distiller), cartographer (services/memory-router/
cartographer), and OSS-corpus bootstrap (infra/oss-corpus-bootstrap)
all need.

Agent-visible types (Convention as seen by the SDK) live in
``libs/sdk-py/crucible_sdk``. The disk-serialization shape (used by the
per-stack bundles and override files) lives here.

Hand-rolled dataclasses are in lock-step with
``libs/memory-spec/proto/crucible/v1/*.proto``. JSON tags are
snake_case to match the on-disk JSON Schema files in
``libs/memory-spec/schemas/``.
"""

from .types import (
    AdmissionScore,
    BundleLicense,
    BundleStats,
    Convention,
    ConventionCandidate,
    ConventionCategory,
    ConventionDrift,
    ConventionStatus,
    FederationGraduation,
    InferredAgentsMd,
    JudgeVerdict,
    MIN_TENANTS_FOR_GRADUATION,
    MemoryLayer,
    PerStackBundle,
    Stack,
    SourceChannel,
    SourceRef,
    ScopeFilter,
    ALL_CATEGORIES,
    ALL_STACKS,
    valid_category,
)
from .errors import (
    InvalidConventionError,
    LicenseUnsafeBundleError,
    JudgeQuarantineError,
)

__all__ = [
    "AdmissionScore",
    "BundleLicense",
    "BundleStats",
    "Convention",
    "ConventionCandidate",
    "ConventionCategory",
    "ConventionDrift",
    "ConventionStatus",
    "FederationGraduation",
    "InferredAgentsMd",
    "JudgeVerdict",
    "MIN_TENANTS_FOR_GRADUATION",
    "MemoryLayer",
    "PerStackBundle",
    "Stack",
    "SourceChannel",
    "SourceRef",
    "ScopeFilter",
    "ALL_CATEGORIES",
    "ALL_STACKS",
    "valid_category",
    "InvalidConventionError",
    "LicenseUnsafeBundleError",
    "JudgeQuarantineError",
]
