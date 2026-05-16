"""Deterministic pre-filter.

Mirrors the keyword + structural scan in the memory-router's
`cmd/memory-router/main.go::DeterministicVerdict`. Producing identical
reasons here means an offline distiller audit reproduces the gateway's
decision exactly.
"""

from __future__ import annotations

from ..types import ConventionCandidate, JudgeVerdict


_INJECTION_PATTERNS = [
    "eval(", " eval ", "eval input",
    "exec(", "execfile(", "spawnshell",
    "rm -rf",
]
_SQL_PATTERNS = ["select * from", "drop table", "delete from "]
_CREDENTIAL_PATTERNS = ["secret=", "password=", "api_key=", "bearer ey"]
_LOW_SPECIFICITY = [
    "ignore the rules", "ignore all rules", "ignore previous",
    "do whatever you want", "no rules apply", "anything goes",
    "override", "bypass", "skip verification",
]
_MALFORMED_MARKERS = ["{{", "}}", "<script", "<%"]


def deterministic_verdict(c: ConventionCandidate) -> JudgeVerdict:
    """Return a JudgeVerdict. quarantine=True means refuse to admit."""
    r = c.rule_nl.lower()

    for pat in _INJECTION_PATTERNS:
        if pat in r:
            return JudgeVerdict(
                candidate_id=c.id,
                quarantine=True,
                score=0.0,
                rationale=f"prompt-injection: {pat!r} pattern",
                injection_category="prompt_injection",
                judge_confidence=1.0,
            )
    for pat in _SQL_PATTERNS:
        if pat in r:
            return JudgeVerdict(
                candidate_id=c.id,
                quarantine=True,
                score=0.0,
                rationale="prompt-injection: SQL-construction pattern",
                injection_category="prompt_injection",
                judge_confidence=1.0,
            )
    for pat in _CREDENTIAL_PATTERNS:
        if pat in r:
            return JudgeVerdict(
                candidate_id=c.id,
                quarantine=True,
                score=0.0,
                rationale="prompt-injection: credential-leak pattern",
                injection_category="prompt_injection",
                judge_confidence=1.0,
            )
    for pat in _LOW_SPECIFICITY:
        if pat in r:
            return JudgeVerdict(
                candidate_id=c.id,
                quarantine=True,
                score=0.0,
                rationale="prompt-injection: low-specificity directive",
                injection_category="prompt_injection",
                judge_confidence=1.0,
            )
    for pat in _MALFORMED_MARKERS:
        if pat in r:
            return JudgeVerdict(
                candidate_id=c.id,
                quarantine=True,
                score=0.0,
                rationale="malformed: template-injection markers in rule",
                injection_category="malformed",
                judge_confidence=1.0,
            )
    if len(c.rule_nl) > 1024:
        return JudgeVerdict(
            candidate_id=c.id,
            quarantine=True,
            score=0.0,
            rationale="malformed: rule exceeds 1024 chars",
            injection_category="malformed",
            judge_confidence=1.0,
        )

    return JudgeVerdict(
        candidate_id=c.id,
        quarantine=False,
        score=0.85,  # deterministic-pass baseline; the LLM judge refines
        rationale="deterministic filter pass",
        injection_category="",
        judge_confidence=0.85,
    )
