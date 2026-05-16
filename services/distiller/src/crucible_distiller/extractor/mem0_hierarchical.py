"""Mem0 hierarchical extraction algorithm — Phase 5 reference impl.

Single-pass extraction (the April-2026 Mem0 update went ADD-only; we
follow that pattern for now and rely on the supersession mechanism in
the procedural store to retire stale rules).

The extractor accepts an `LLMClient` so production wires the real
Anthropic Haiku-4.5 client while tests use the deterministic fake.
"""

from __future__ import annotations

import json
import re
from dataclasses import dataclass
from datetime import datetime, timezone
from typing import Any, Callable, Iterable, Optional, Protocol
from uuid import uuid4

from ..types import (
    ConventionCandidate,
    ConventionCategory,
    ExtractionInput,
    ExtractionResult,
    ScopeFilter,
    SourceRef,
)
from .prompts import SYSTEM_PROMPT, render_user
from .schema import validate_extraction


class LLMClient(Protocol):
    """Minimal extractor LLM surface. The production Anthropic client
    implements this; the deterministic fake below is wired in tests."""

    def generate_json(self, *, system: str, user: str, schema: dict[str, Any], max_tokens: int) -> tuple[str, int, int, float]:
        """Returns (raw_json, input_tokens, output_tokens, cost_usd)."""
        ...


def extract(client: LLMClient, ein: ExtractionInput) -> ExtractionResult:
    """Run the extractor against one source item.

    Returns an ExtractionResult. On JSON / schema-validation failure,
    the result is empty and ``null_extraction_reason`` is set — the
    distiller persists the rejection to ``distiller_runs`` for audit.
    """
    user_prompt = render_user(
        source_channel=getattr(ein.source_channel, "value", str(ein.source_channel)),
        repo=ein.repo,
        source_summary=_source_summary(ein.source),
        raw_text=ein.raw_text,
    )
    raw, in_tok, out_tok, cost = client.generate_json(
        system=SYSTEM_PROMPT,
        user=user_prompt,
        schema={"type": "array"},
        max_tokens=1024,
    )

    try:
        items = validate_extraction(raw)
    except ValueError as e:
        return ExtractionResult(
            candidates=[],
            extractor_model="",
            input_tokens=in_tok,
            output_tokens=out_tok,
            cost_usd=cost,
            null_extraction_reason=f"schema-validation failed: {e}",
        )

    candidates: list[ConventionCandidate] = []
    for item in items:
        try:
            category = ConventionCategory(item["category"])
        except ValueError:
            # Schema validation should have caught this, but defence in depth.
            continue
        candidates.append(
            ConventionCandidate(
                id=f"cand_{uuid4().hex}",
                tenant_id=ein.tenant_id,
                scope=ScopeFilter(repo=ein.repo, file_glob=item.get("file_glob", "")),
                rule_nl=item["rule"],
                category=category,
                rationale=item.get("rationale", ""),
                evidence_quote=item.get("evidence_quote", ""),
                source_evidence=[ein.source],
                extracted_at=ein.extracted_at if ein.extracted_at else datetime.now(timezone.utc),
                source_channel=ein.source_channel,
            )
        )

    return ExtractionResult(
        candidates=candidates,
        extractor_model="",  # Filled by the caller from the client metadata.
        input_tokens=in_tok,
        output_tokens=out_tok,
        cost_usd=cost,
    )


def _source_summary(src: SourceRef) -> str:
    if src.kind == "pr_comment":
        return f"PR #{src.pr} comment {src.comment_id}"
    if src.kind == "incident":
        return f"incident {src.id} ({src.service})"
    if src.kind == "adr":
        return f"ADR {src.path}@{src.commit}"
    if src.kind == "agent_observation":
        return f"agent task {src.task_id} step {src.step_id}"
    return src.kind


# ───────────────────────────────────────────────────────────────────────────
# Deterministic regex-based extractor used when LLM access isn't desired
# (CI hermetic mode, dev runs). Covers the common ADR + lint-config patterns.
# ───────────────────────────────────────────────────────────────────────────

_RULE_PATTERNS: list[tuple[re.Pattern[str], ConventionCategory]] = [
    (re.compile(r"\buse\s+(slog|zap|pino|winston|logback)\b", re.I), ConventionCategory.LOGGING),
    (re.compile(r"\b(context\.context|pass context)\b", re.I), ConventionCategory.CONCURRENCY),
    (re.compile(r"\b(cursor pagination|no offset pagination)\b", re.I), ConventionCategory.PERFORMANCE_DEFAULTS),
    (re.compile(r"\b(conventional commits|semantic-release)\b", re.I), ConventionCategory.PR_COMMIT_HYGIENE),
    (re.compile(r"\b(date-fns|day\.js|luxon)\b.*\b(over|instead of)\b\s+moment", re.I), ConventionCategory.LIBRARY_PREFERENCES),
    (re.compile(r"\bauth middleware\b.*\bbefore\b", re.I), ConventionCategory.SECURITY_DEFAULTS),
    (re.compile(r"\b(additive[- ]only|never drop column|deprecation period)\b", re.I), ConventionCategory.MIGRATION_PATTERNS),
    (re.compile(r"\b(structured (logs|logging))\b", re.I), ConventionCategory.LOGGING),
    (re.compile(r"\b(Result<|Either<|exceptions for control flow)\b", re.I), ConventionCategory.ERROR_HANDLING),
    (re.compile(r"\btest files (end in|ending in)\b", re.I), ConventionCategory.NAMING),
    (re.compile(r"\bidempotency key\b", re.I), ConventionCategory.API_SHAPE),
    (re.compile(r"\bcolocate tests\b|\b__tests__\b", re.I), ConventionCategory.TEST_PATTERNS),
]


def deterministic_extract(text: str, repo: str = "", tenant_id: str = "", source: Optional[SourceRef] = None) -> list[ConventionCandidate]:
    """Regex-based deterministic extractor.

    Used by the lint-config adapter (which produces "free" rules per
    docs/06-research/memory-bootstrap.md) and by CI tests that don't
    want to hit the LLM. The patterns catch the common rule shapes
    that show up in AGENTS.md / ADRs.
    """
    out: list[ConventionCandidate] = []
    for pat, cat in _RULE_PATTERNS:
        m = pat.search(text)
        if not m:
            continue
        out.append(
            ConventionCandidate(
                id=f"cand_{uuid4().hex}",
                tenant_id=tenant_id,
                scope=ScopeFilter(repo=repo),
                rule_nl=m.group(0).strip()[:240],
                category=cat,
                rationale="deterministic-extracted (no LLM)",
                evidence_quote=m.group(0).strip()[:240],
                source_evidence=[source] if source else [],
                extracted_at=datetime.now(timezone.utc),
            )
        )
    return out


# ───────────────────────────────────────────────────────────────────────────
# Deterministic-LLM fake for unit tests
# ───────────────────────────────────────────────────────────────────────────


@dataclass
class FakeLLM:
    """Deterministic extractor for tests.

    Programmable: callers either provide `responses` (a list of JSON
    bodies the extractor will replay in order) or `pattern_responses`
    (a dict from substring → JSON body). Tests use this to assert the
    extractor handles malformed JSON, prompt-injection, and good
    extraction equally well.
    """

    responses: list[str] | None = None
    pattern_responses: dict[str, str] | None = None
    cost_per_call: float = 0.0001

    def generate_json(self, *, system: str, user: str, schema: dict[str, Any], max_tokens: int) -> tuple[str, int, int, float]:
        if self.pattern_responses:
            for needle, response in self.pattern_responses.items():
                if needle in user:
                    return response, len(user) // 4, len(response) // 4, self.cost_per_call
        if self.responses:
            r = self.responses.pop(0)
            return r, len(user) // 4, len(r) // 4, self.cost_per_call
        return "[]", len(user) // 4, 2, self.cost_per_call
