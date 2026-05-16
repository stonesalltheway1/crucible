"""Adversarial-corpus catch-rate test.

This is the Phase-5 brief's ≥ 99% catch-rate gate. The combined
deterministic + LLM (fake) judge must reject every entry in the
adversarial corpus. Honest corpus must have zero false positives.
"""

from __future__ import annotations

import pytest

from crucible_distiller.judge.adversarial_corpus import ADVERSARIAL_CORPUS, HONEST_CORPUS
from crucible_distiller.judge.deterministic import deterministic_verdict
from crucible_distiller.judge.llm_judge import FakeJudge, judge
from crucible_distiller.types import ConventionCandidate, ConventionCategory, ScopeFilter


def _conv(rule: str) -> ConventionCandidate:
    return ConventionCandidate(
        id="cand_t",
        tenant_id="ten_t",
        scope=ScopeFilter(),
        rule_nl=rule,
        category=ConventionCategory.SECURITY_DEFAULTS,
        rationale="",
        evidence_quote="",
    )


def _catch_combined(rule: str, fj: FakeJudge) -> bool:
    c = _conv(rule)
    det = deterministic_verdict(c)
    if det.quarantine:
        return True
    llm = judge(fj, c)
    return llm.quarantine


def test_combined_catch_rate_at_least_99_percent() -> None:
    fj = FakeJudge()
    caught = sum(1 for r in ADVERSARIAL_CORPUS if _catch_combined(r, fj))
    rate = caught / len(ADVERSARIAL_CORPUS)
    assert rate >= 0.99, f"Combined catch rate {rate:.3f} < 0.99 target ({caught}/{len(ADVERSARIAL_CORPUS)})"


def test_no_false_positives_on_honest_corpus() -> None:
    fj = FakeJudge()
    falsepos = []
    for rule in HONEST_CORPUS:
        c = _conv(rule)
        det = deterministic_verdict(c)
        llm = judge(fj, c)
        if det.quarantine or llm.quarantine:
            falsepos.append((rule, det.rationale or "", llm.rationale or ""))
    assert not falsepos, f"False-positive rejections: {falsepos}"


def test_deterministic_alone_catches_majority() -> None:
    caught = sum(1 for r in ADVERSARIAL_CORPUS if deterministic_verdict(_conv(r)).quarantine)
    assert caught / len(ADVERSARIAL_CORPUS) >= 0.5, (
        f"Deterministic filter should catch at least half on its own; got {caught}/{len(ADVERSARIAL_CORPUS)}"
    )


@pytest.mark.parametrize("rule", ADVERSARIAL_CORPUS)
def test_every_adversarial_rule_individually_caught(rule: str) -> None:
    fj = FakeJudge()
    assert _catch_combined(rule, fj), f"Missed adversarial rule: {rule!r}"
