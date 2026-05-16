"""JSON Schema for the extractor's output.

This is the AdaKGC schema-constrained-decoding target. Outlines (or any
equivalent JSON-schema constrained-decoding wrapper) generates only
outputs that validate against this schema. Admission's "ban category=other"
gate is enforced at generation time as a result.
"""

from __future__ import annotations

import json
from typing import Any

from ..types import ConventionCategory

# Build the enum list once from the canonical Python source.
_CATEGORY_ENUM = [c.value for c in ConventionCategory]


EXTRACTION_SCHEMA: dict[str, Any] = {
    "$schema": "https://json-schema.org/draft/2020-12/schema",
    "type": "array",
    "items": {
        "type": "object",
        "required": ["category", "rule", "file_glob", "rationale", "evidence_quote"],
        "additionalProperties": False,
        "properties": {
            "category":        {"type": "string", "enum": _CATEGORY_ENUM},
            "rule":            {"type": "string", "minLength": 8, "maxLength": 1024},
            "file_glob":       {"type": "string", "maxLength": 512},
            "rationale":       {"type": "string", "minLength": 1, "maxLength": 512},
            "evidence_quote":  {"type": "string", "minLength": 1, "maxLength": 1024},
        },
    },
}


def validate_extraction(raw: str) -> list[dict[str, Any]]:
    """Parse + validate a raw model response against EXTRACTION_SCHEMA.

    Returns the parsed list of candidate dicts. Raises ``ValueError`` if
    the payload doesn't validate — admission rejects malformed
    extractions outright (the Phase-5 brief explicitly bans
    ``category=other`` and friends).

    We use a hand-rolled validator rather than importing jsonschema so
    the extractor stays dependency-free in the hot path.
    """
    try:
        parsed = json.loads(raw)
    except json.JSONDecodeError as e:
        raise ValueError(f"extractor: invalid JSON: {e}") from e
    if not isinstance(parsed, list):
        raise ValueError(f"extractor: top-level must be array, got {type(parsed).__name__}")
    out: list[dict[str, Any]] = []
    for i, item in enumerate(parsed):
        if not isinstance(item, dict):
            raise ValueError(f"extractor: item[{i}] must be object")
        for required in ("category", "rule", "file_glob", "rationale", "evidence_quote"):
            if required not in item:
                raise ValueError(f"extractor: item[{i}] missing field {required}")
        if item["category"] not in _CATEGORY_ENUM:
            raise ValueError(
                f"extractor: item[{i}] category={item['category']!r} "
                f"not in taxonomy (admission bans this)"
            )
        if not isinstance(item["rule"], str) or not (8 <= len(item["rule"]) <= 1024):
            raise ValueError(f"extractor: item[{i}] rule out of length [8,1024]")
        out.append(item)
    return out
