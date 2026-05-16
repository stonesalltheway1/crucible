"""Mem0-style hierarchical extraction (Phase-5 reference implementation).

Single-pass extraction over the source text → JSON array of typed
Convention candidates. Schema-constrained decoding via outlines
(AdaKGC pattern) — the model is constrained to emit valid taxonomy
buckets, so admission's "ban category=other" gate is enforced at
generation time, not just at admission.
"""

from .mem0_hierarchical import extract, deterministic_extract
from .schema import EXTRACTION_SCHEMA, validate_extraction

__all__ = ["extract", "deterministic_extract", "EXTRACTION_SCHEMA", "validate_extraction"]
