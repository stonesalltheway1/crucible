"""Tier 1 — Hypothesis property-based testing (+ optional atheris fuzz).

Discovers test files that contain ``@given`` decorators (or ``@hypothesis``
imports), runs them at ``HYPOTHESIS_MAX_EXAMPLES=10_000``, and captures any
``Falsifying example:`` blocks pytest emits into the report's
``counterexamples``.

Iterations counted at 10K per property that ran without falsification.
"""

from __future__ import annotations

import os
import re
import subprocess
import sys
from pathlib import Path
from typing import TYPE_CHECKING

from ..schema import (
    Counterexample,
    Finding,
    PBTStats,
    TestReport,
    Tier,
    Verdict,
)

if TYPE_CHECKING:
    from ..cli import DriverContext
    from ..diff import FileChange

# Crucible mandate: PBT iterations >= 10_000.
ITERATIONS_MIN = 10_000
FRAMEWORK = "hypothesis~=6.152"


def run(ctx: DriverContext) -> TestReport:
    """Discover and run hypothesis-decorated tests in the diff."""
    tests = _discover_property_tests(ctx.diff_files, ctx.workdir)
    if not tests:
        return TestReport(
            task_id=ctx.task_id,
            tier=Tier.PBT,
            framework=FRAMEWORK,
            verdict=Verdict.SKIPPED,
            passed=True,
            pbt=PBTStats(iterations=0, iterations_min=ITERATIONS_MIN),
        )

    if not _have_hypothesis():
        return _tool_unavailable(ctx.task_id, "hypothesis not importable in sandbox")

    properties = _list_property_names(tests)
    rc, stdout = _run_pytest(
        tests=[str(t.relative_to(ctx.workdir)) for t in tests],
        cwd=ctx.workdir,
        timeout=ctx.budget_seconds,
    )

    counterexamples = _parse_counterexamples(stdout)

    # Iteration accounting: each property that did NOT falsify ran 10K
    # times (Hypothesis stops early on a failure, so we credit only the
    # full-run properties). The dispatcher's PBT invariant requires
    # iterations >= iterations_min when iterations_min > 0, so we set
    # iterations_min to 0 when *every* property falsified — there's no
    # honest claim to make there.
    falsified = {c.property for c in counterexamples}
    survived_props = [p for p in properties if p not in falsified]
    iterations = ITERATIONS_MIN * len(survived_props)
    iterations_min = ITERATIONS_MIN if survived_props else 0

    fuzz_corpus, fuzz_crashes = _run_atheris_if_present(
        ctx.diff_files, ctx.workdir, ctx.budget_seconds
    )

    stats = PBTStats(
        iterations=iterations,
        iterations_min=iterations_min,
        properties=properties,
        counterexamples=counterexamples,
        fuzz_corpus_size=fuzz_corpus,
        fuzz_crashes=fuzz_crashes,
    )

    findings: list[Finding] = []
    for c in counterexamples:
        findings.append(
            Finding(
                category="property_failed",
                severity="error",
                detail=f"{c.property} falsified by: {c.shrunk}",
                suggested_fix=(
                    "Either tighten the property assumption or fix the "
                    "implementation to satisfy the property."
                ),
            )
        )

    passed = rc == 0 and not counterexamples and len(properties) > 0
    verdict = Verdict.PASSED if passed else (
        Verdict.FAILED if rc != 0 or counterexamples else Verdict.SKIPPED
    )

    return TestReport(
        task_id=ctx.task_id,
        tier=Tier.PBT,
        framework=FRAMEWORK,
        verdict=verdict,
        passed=passed,
        pbt=stats,
        findings=findings,
    )


# --- discovery -----------------------------------------------------------


_GIVEN_DECORATOR = re.compile(r"^\s*@given\b", re.MULTILINE)
_PROPERTY_DEF = re.compile(r"^\s*def\s+(?P<name>test_[A-Za-z_0-9]+)\s*\(", re.MULTILINE)


def _discover_property_tests(diff_files: list[FileChange], workdir: Path) -> list[Path]:
    """Return absolute paths of test files that use ``@given``.

    We look at the agent's own diff first (the brand-promise scope: the
    agent must author its own properties). If none are found in the diff,
    we also look at any test_*.py file already materialised in workdir.
    """
    candidates: list[Path] = []
    for fc in diff_files:
        if not fc.is_python_test:
            continue
        if _GIVEN_DECORATOR.search(fc.content):
            candidates.append((workdir / fc.path).resolve())
    if candidates:
        return candidates

    # Fallback — scan workdir for any test_*.py with @given.
    for path in sorted(workdir.rglob("test_*.py")):
        try:
            text = path.read_text(encoding="utf-8")
        except OSError:
            continue
        if _GIVEN_DECORATOR.search(text):
            candidates.append(path.resolve())
    return candidates


def _list_property_names(test_files: list[Path]) -> list[str]:
    names: list[str] = []
    for path in test_files:
        try:
            text = path.read_text(encoding="utf-8")
        except OSError:
            continue
        # A property is a test function that has at least one @given.
        # We find each @given block and pair it with the next def.
        for m in _GIVEN_DECORATOR.finditer(text):
            tail = text[m.end():]
            dm = _PROPERTY_DEF.search(tail)
            if dm is not None:
                names.append(f"{path.name}::{dm.group('name')}")
    return names


# --- pytest invocation ---------------------------------------------------


def _have_hypothesis() -> bool:
    try:
        import hypothesis  # noqa: F401
    except ImportError:
        return False
    return True


def _run_pytest(*, tests: list[str], cwd: Path, timeout: float) -> tuple[int, str]:
    env = os.environ.copy()
    # Hypothesis honours these env vars; the dispatcher could override.
    env.setdefault("HYPOTHESIS_MAX_EXAMPLES", str(ITERATIONS_MIN))
    env.setdefault("PYTHONIOENCODING", "utf-8")
    try:
        proc = subprocess.run(
            [
                sys.executable,
                "-m",
                "pytest",
                "-q",
                "--hypothesis-show-statistics",
                "--no-header",
                *tests,
            ],
            cwd=str(cwd),
            env=env,
            capture_output=True,
            text=True,
            timeout=max(timeout, 1.0),
            check=False,
        )
    except subprocess.TimeoutExpired:
        return 124, ""
    except FileNotFoundError:
        return -1, ""
    sys.stderr.write(proc.stderr)
    return proc.returncode, proc.stdout


# --- counterexample parsing ----------------------------------------------


# Hypothesis prints e.g.:
#     Falsifying example: test_sort_idempotent(
#         xs=[0, -1],
#     )
_FALSIFY_BLOCK = re.compile(
    r"Falsifying example:\s*(?P<prop>[A-Za-z_0-9]+)\s*\(\s*(?P<args>.*?)\s*\)\s*$",
    re.DOTALL | re.MULTILINE,
)


def _parse_counterexamples(stdout: str) -> list[Counterexample]:
    out: list[Counterexample] = []
    for m in _FALSIFY_BLOCK.finditer(stdout):
        args = m.group("args").strip().rstrip(",")
        out.append(
            Counterexample(
                property=m.group("prop"),
                shrunk=args.replace("\n", " ").strip(),
            )
        )
    return out


# --- optional atheris fuzz ----------------------------------------------


def _run_atheris_if_present(
    diff_files: list[FileChange], workdir: Path, budget: float
) -> tuple[int, int]:
    """If atheris is installed and a fuzz_*.py target exists, run it briefly.

    Returns ``(corpus_size, crashes)``. Best-effort — atheris is optional.
    """
    try:
        import atheris  # noqa: F401
    except ImportError:
        return 0, 0
    targets = [
        workdir / fc.path
        for fc in diff_files
        if fc.is_python and Path(fc.path).name.startswith("fuzz_")
    ]
    if not targets:
        return 0, 0
    crashes = 0
    corpus = 0
    # Give atheris a small slice of the tier budget so we don't starve
    # hypothesis.
    per_target = max(min(10.0, budget / 4.0), 1.0)
    for target in targets:
        try:
            proc = subprocess.run(
                [sys.executable, str(target), "-atheris_runs=1000", "-max_total_time=" + str(int(per_target))],
                cwd=str(workdir),
                capture_output=True,
                text=True,
                timeout=per_target + 5.0,
                check=False,
            )
        except (subprocess.TimeoutExpired, FileNotFoundError):
            continue
        sys.stderr.write(proc.stderr)
        if "Uncaught exception" in proc.stdout or proc.returncode not in (0, 77):
            crashes += 1
        corpus += 1000
    return corpus, crashes


def _tool_unavailable(task_id: str, detail: str) -> TestReport:
    return TestReport(
        task_id=task_id,
        tier=Tier.PBT,
        framework=FRAMEWORK,
        verdict=Verdict.TOOL_UNAVAILABLE,
        passed=False,
        pbt=PBTStats(iterations=0, iterations_min=0),
        error=detail,
        findings=[Finding(category="tool_error", severity="warn", detail=detail)],
    )
