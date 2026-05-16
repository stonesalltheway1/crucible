"""Integration tests for the Tier 1 hypothesis driver.

The pure-Python parsing tests run unconditionally; the end-to-end tests
that spawn ``pytest -m hypothesis`` are skipped when hypothesis is absent.
"""

from __future__ import annotations

from pathlib import Path

import pytest

from crucible_verify_python.cli import DriverContext
from crucible_verify_python.diff import FileChange
from crucible_verify_python.schema import Tier, Verdict
from crucible_verify_python.tiers import tier1_pbt
from crucible_verify_python.tiers.tier1_pbt import (
    ITERATIONS_MIN,
    _list_property_names,
    _parse_counterexamples,
)

# --- pure-Python parsing tests ------------------------------------------


def test_parse_counterexamples_extracts_property_and_args() -> None:
    stdout = """
============= test session starts =============
collected 1 items
tests/test_buggy.py F

================== FAILURES ===================
___________ test_buggy_abs_nonneg ____________

Falsifying example: test_buggy_abs_nonneg(
    x=-2147483648,
)
"""
    ces = _parse_counterexamples(stdout)
    assert len(ces) == 1
    assert ces[0].property == "test_buggy_abs_nonneg"
    assert "x=-2147483648" in ces[0].shrunk


def test_parse_counterexamples_handles_no_failures() -> None:
    assert _parse_counterexamples("all good, 100 passed") == []


def test_parse_counterexamples_multiple() -> None:
    stdout = """
Falsifying example: test_a(
    x=0,
)
some other output
Falsifying example: test_b(
    y=[],
)
"""
    ces = _parse_counterexamples(stdout)
    assert {c.property for c in ces} == {"test_a", "test_b"}


# --- property discovery --------------------------------------------------


def test_list_property_names_finds_given_decorated_funcs(tmp_path: Path) -> None:
    src = tmp_path / "test_a.py"
    src.write_text(
        "from hypothesis import given, strategies as st\n"
        "@given(st.integers())\n"
        "def test_one(x):\n"
        "    assert isinstance(x, int)\n"
        "@given(st.lists(st.integers()))\n"
        "def test_two(xs):\n"
        "    assert sorted(xs) == sorted(xs)\n",
        encoding="utf-8",
    )
    names = _list_property_names([src])
    assert any("test_one" in n for n in names)
    assert any("test_two" in n for n in names)


# --- end-to-end tests ----------------------------------------------------


def _have_hypothesis() -> bool:
    try:
        import hypothesis  # noqa: F401
        import pytest as _pt  # noqa: F401
    except ImportError:
        return False
    return True


@pytest.mark.skipif(not _have_hypothesis(), reason="hypothesis not installed")
@pytest.mark.slow
def test_falsifying_property_surfaces_counterexample(
    tmp_path: Path,
    hypothesis_failing_files: list[dict],
) -> None:
    """A property that intentionally falsifies must surface a counterexample."""
    diff_files = [FileChange.from_dict(f) for f in hypothesis_failing_files]
    for fc in diff_files:
        dest = tmp_path / fc.path
        dest.parent.mkdir(parents=True, exist_ok=True)
        dest.write_text(fc.content, encoding="utf-8")
    ctx = DriverContext(
        task_id="t-fail",
        diff_files=diff_files,
        spec_changes=[],
        workdir=tmp_path,
        budget_seconds=60.0,
        request={},
    )
    report = tier1_pbt.run(ctx)
    assert report.tier == Tier.PBT
    assert report.pbt is not None
    assert report.passed is False
    assert report.verdict == Verdict.FAILED
    assert len(report.pbt.counterexamples) >= 1
    assert any(
        f.category == "property_failed" and "falsified" in f.detail
        for f in report.findings
    )


@pytest.mark.skipif(not _have_hypothesis(), reason="hypothesis not installed")
@pytest.mark.slow
def test_passing_property_runs_at_least_10k_iterations(
    tmp_path: Path,
    hypothesis_passing_files: list[dict],
) -> None:
    """A passing property must be credited with >= 10K iterations."""
    diff_files = [FileChange.from_dict(f) for f in hypothesis_passing_files]
    for fc in diff_files:
        dest = tmp_path / fc.path
        dest.parent.mkdir(parents=True, exist_ok=True)
        dest.write_text(fc.content, encoding="utf-8")
    ctx = DriverContext(
        task_id="t-pass",
        diff_files=diff_files,
        spec_changes=[],
        workdir=tmp_path,
        budget_seconds=60.0,
        request={},
    )
    report = tier1_pbt.run(ctx)
    assert report.pbt is not None
    assert report.passed is True
    assert report.pbt.iterations >= ITERATIONS_MIN
    assert report.pbt.iterations_min == ITERATIONS_MIN


def test_no_property_tests_skips(tmp_path: Path) -> None:
    """An empty Python diff means there are no properties to run; verdict skipped."""
    ctx = DriverContext(
        task_id="t-empty",
        diff_files=[FileChange(path="README.md", action="modify", content="docs")],
        spec_changes=[],
        workdir=tmp_path,
        budget_seconds=30.0,
        request={},
    )
    report = tier1_pbt.run(ctx)
    assert report.verdict == Verdict.SKIPPED
    assert report.passed is True
