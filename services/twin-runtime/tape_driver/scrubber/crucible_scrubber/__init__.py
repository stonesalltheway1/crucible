"""Crucible PII scrub pipeline.

Phase 3 of the Crucible build. The package exposes the same high-level
contract as the Phase 2 Go regex scrubber but is materially more capable:

- Presidio Analyzer + Anonymizer for NER-driven detection
- spaCy or HuggingFace TransformersNlpEngine for free-text NER
- FF3-1 format-preserving encryption for structure-bearing fields
- Deterministic pseudonymisation for referential integrity
- Custom recognizer registration for tenant-specific PII
- Scrub audit log keyed for compliance auditors

Most callers import the high-level pipeline:

    from crucible_scrubber.pipeline import ScrubPipeline, ScrubRequest
    p = ScrubPipeline.from_env()
    out, report = p.scrub(ScrubRequest(tape_set="t", payload=raw))

The FastAPI service in `crucible_scrubber.app` is what the Go tape_driver
calls in production.
"""

from .pipeline import ScrubPipeline, ScrubRequest, ScrubResult
from .audit import AuditEntry, AuditLog
from .operators import (
    DeterministicHashOperator,
    Ff3FpeOperator,
    OPERATOR_DETERMINISTIC,
    OPERATOR_FF3,
)
from .ff3 import Ff3Cipher, Ff3DomainError
from .recognizers import (
    MRNRecognizer,
    TenantAccountRecognizer,
    register_default_custom_recognizers,
)

__version__ = "2026.6.0-phase3"

__all__ = [
    "ScrubPipeline",
    "ScrubRequest",
    "ScrubResult",
    "AuditEntry",
    "AuditLog",
    "DeterministicHashOperator",
    "Ff3FpeOperator",
    "OPERATOR_DETERMINISTIC",
    "OPERATOR_FF3",
    "Ff3Cipher",
    "Ff3DomainError",
    "MRNRecognizer",
    "TenantAccountRecognizer",
    "register_default_custom_recognizers",
    "__version__",
]
