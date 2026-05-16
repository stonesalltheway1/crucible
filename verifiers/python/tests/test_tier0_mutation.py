"""Integration tests for the Tier 0 mutmut driver.

These tests are gated on ``mutmut`` being importable. When it isn't (e.g.
on a CI image that hasn't installed the verifier extras), they skip — the
unit-level parsing tests below still run so the parsing logic is covered.
"""

from __future__ import annotations

from pathlib import Path

import pytest

from crucible_verify_python.cli import DriverContext
from crucible_verify_python.diff import FileChange
from crucible_verify_python.schema import Tier, Verdict
from crucible_verify_python.tiers import tier0_mutation
from crucible_verify_python.tiers.tier0_mutation import (
    MUTATION_THRESHOLD,
    _parse_mutmut_results,
)

# --- pure-Python parsing tests (always run) -----------------------------


def test_parse_results_with_strong_score() -> None:
    out = """
mutmut cache version 4
Killed 17 mutants
Survived (S) 2 mutants
Timeout (T) 0 mutants
Suspicious (?) 0 mutants
Skipped (s) 0 mutants
Not checked 0 mutants
"""
    stats = _parse_mutmut_results(out, [FileChange(path="app.py", action="modify")])
    assert stats.killed == 17
    assert stats.survived == 2
    assert stats.total == 19
    assert stats.score == pytest.approx(17 / 19, rel=1e-9)
    assert stats.score >= MUTATION_THRESHOLD
    assert stats.diff_scoped is True
    assert stats.threshold == MUTATION_THRESHOLD


def test_parse_results_with_weak_score() -> None:
    out = """
Killed 3 mutants
Survived (S) 12 mutants
Timeout 0 mutants
Not checked 0 mutants
"""
    stats = _parse_mutmut_results(out, [FileChange(path="app.py", action="modify")])
    assert stats.killed == 3
    assert stats.survived == 12
    assert stats.score == pytest.approx(3 / 15, rel=1e-9)
    assert stats.score < MUTATION_THRESHOLD


def test_parse_results_empty() -> None:
    stats = _parse_mutmut_results("", [])
    assert stats.total == 0
    assert stats.score == 0.0
    assert stats.diff_scoped is True


def test_parse_results_extracts_surviving_mutants() -> None:
    out = """
Killed 1 mutants
Survived (S) 2 mutants
Surviving mutants:
   src/app.py:42 :: BinOpMutator '+' -> '-'
   src/app.py:55 :: BooleanMutator 'True' -> 'False'
"""
    stats = _parse_mutmut_results(out, [FileChange(path="src/app.py", action="modify")])
    assert stats.survived == 2
    assert len(stats.survived_summary) == 2
    first, second = stats.survived_summary
    assert first.file == "src/app.py"
    assert first.line == 42
    assert first.mutator == "BinOpMutator"
    assert first.original == "+"
    assert first.replacement == "-"
    assert second.line == 55


# --- end-to-end tests (require mutmut) ----------------------------------


pytestmark_e2e = pytest.mark.needs_mutmut


def _have_mutmut() -> bool:
    try:
        import mutmut  # noqa: F401
    except ImportError:
        return False
    return True


@pytest.mark.skipif(not _have_mutmut(), reason="mutmut not installed")
@pytest.mark.slow
def test_strong_tests_pass_mutation_threshold(
    tmp_path: Path,
    strong_fixture_files: list[dict],
) -> None:
    """The strong fixture should achieve >= 85% mutation score."""
    diff_files = [FileChange.from_dict(f) for f in strong_fixture_files]
    # Materialise into tmp_path so mutmut can run.
    for fc in diff_files:
        dest = tmp_path / fc.path
        dest.parent.mkdir(parents=True, exist_ok=True)
        dest.write_text(fc.content, encoding="utf-8")
    ctx = DriverContext(
        task_id="t-strong",
        diff_files=diff_files,
        spec_changes=[],
        workdir=tmp_path,
        budget_seconds=120.0,
        request={},
    )
    report = tier0_mutation.run(ctx)
    assert report.tier == Tier.MUTATION
    assert report.mutation is not None
    assert report.mutation.diff_scoped is True
    # Strong fixture should kill enough mutants.
    assert report.mutation.score >= MUTATION_THRESHOLD, (
        f"score={report.mutation.score}, "
        f"killed={report.mutation.killed}, survived={report.mutation.survived}"
    )
    assert report.passed is True
    assert report.verdict == Verdict.PASSED


@pytest.mark.skipif(not _have_mutmut(), reason="mutmut not installed")
@pytest.mark.slow
def test_weak_tests_are_rejected(
    tmp_path: Path,
    weak_fixture_files: list[dict],
) -> None:
    """The weak fixture must NOT pass — verdict failed, mutation < threshold."""
    diff_files = [FileChange.from_dict(f) for f in weak_fixture_files]
    for fc in diff_files:
        dest = tmp_path / fc.path
        dest.parent.mkdir(parents=True, exist_ok=True)
        dest.write_text(fc.content, encoding="utf-8")
    ctx = DriverContext(
        task_id="t-weak",
        diff_files=diff_files,
        spec_changes=[],
        workdir=tmp_path,
        budget_seconds=120.0,
        request={},
    )
    report = tier0_mutation.run(ctx)
    assert report.mutation is not None
    assert report.passed is False
    # Either we landed a failed verdict (score < threshold) or
    # tool_unavailable (mutmut produced no mutants on the tiny diff) —
    # in NEITHER case may we mark this as passed.
    assert report.verdict in (Verdict.FAILED, Verdict.TOOL_UNAVAILABLE)
    if report.verdict == Verdict.FAILED:
        assert report.mutation.score < MUTATION_THRESHOLD


@pytest.mark.skipif(_have_mutmut(), reason="negative-path test runs when mutmut is absent")
def test_tool_unavailable_when_mutmut_missing(tmp_path: Path) -> None:
    """If mutmut isn't installed we emit a tool_unavailable report, not a crash."""
    ctx = DriverContext(
        task_id="t-no-mutmut",
        diff_files=[FileChange(path="app.py", action="modify", content="def f(): return 1\n")],
        spec_changes=[],
        workdir=tmp_path,
        budget_seconds=30.0,
        request={},
    )
    report = tier0_mutation.run(ctx)
    assert report.verdict == Verdict.TOOL_UNAVAILABLE
    assert report.passed is False
    assert report.mutation is not None
    assert report.mutation.diff_scoped is True
