"""Tier 0 — mutation testing via ``mutmut`` 3.5.

mutmut 3.5 ships no native JSON output, so we run ``mutmut results`` and
parse its plain-text summary. We always set ``diff_scoped=True`` and
restrict mutation to the Python source files in the agent diff by writing
a temporary ``setup.cfg``/``pyproject.toml`` overlay with ``paths_to_mutate``.

Threshold: **0.85** killed / (killed + survived).
"""

from __future__ import annotations

import os
import re
import subprocess
import sys
from pathlib import Path
from typing import TYPE_CHECKING

from ..schema import (
    Finding,
    MutationStats,
    SurvivedMutant,
    TestReport,
    Tier,
    Verdict,
)

if TYPE_CHECKING:  # avoid circular import at runtime
    from ..cli import DriverContext
    from ..diff import FileChange

# Crucible mandate, per docs/01-architecture/verifier-pipeline.md.
MUTATION_THRESHOLD = 0.85
FRAMEWORK = "mutmut~=3.5"


def run(ctx: DriverContext) -> TestReport:
    """Run mutmut against the diff's Python source files."""
    sources = [f for f in ctx.diff_files if f.is_python_source]
    if not sources:
        return TestReport(
            task_id=ctx.task_id,
            tier=Tier.MUTATION,
            framework=FRAMEWORK,
            verdict=Verdict.SKIPPED,
            passed=True,  # Nothing Python to mutate — vacuously fine.
            mutation=MutationStats(diff_scoped=True, threshold=MUTATION_THRESHOLD),
        )

    if not _have_mutmut():
        return _tool_unavailable(ctx.task_id, "mutmut binary not on PATH")

    _write_mutmut_config(ctx.workdir, sources)

    rc_run = _spawn_mutmut(
        ["run", "--no-progress"],
        cwd=ctx.workdir,
        timeout=ctx.budget_seconds,
    )
    # Non-zero rc from `mutmut run` is normal — it returns the count of
    # surviving mutants. We only treat a crash (signal-killed, etc.) as a
    # procedural failure.
    if rc_run < 0:
        return _tool_unavailable(ctx.task_id, f"mutmut run terminated with signal {-rc_run}")

    rc_results, stdout = _capture_mutmut_results(ctx.workdir, ctx.budget_seconds)
    if rc_results not in (0, 1, 2):
        return _tool_unavailable(
            ctx.task_id, f"mutmut results exited with {rc_results}"
        )

    stats = _parse_mutmut_results(stdout, sources)
    findings: list[Finding] = []
    if stats.score < MUTATION_THRESHOLD and (stats.killed + stats.survived) > 0:
        for sm in stats.survived_summary[:20]:  # cap surface area
            findings.append(
                Finding(
                    category="mutation_survived",
                    severity="error",
                    file=sm.file,
                    line=sm.line,
                    detail=(
                        f"surviving mutant: {sm.mutator} replaced "
                        f"{sm.original!r} with {sm.replacement!r}"
                    ),
                    suggested_fix=(
                        "Add or strengthen a test that distinguishes the "
                        "original from the mutant."
                    ),
                )
            )

    passed = (
        stats.total > 0
        and stats.score >= MUTATION_THRESHOLD
        and (stats.killed + stats.survived) > 0
    )
    if stats.total == 0:
        # mutmut produced no mutants — fall back to tool_unavailable so the
        # dispatcher can engage the coverage+rubric fallback per the pipeline doc.
        verdict = Verdict.TOOL_UNAVAILABLE
        passed = False
    elif passed:
        verdict = Verdict.PASSED
    else:
        verdict = Verdict.FAILED

    return TestReport(
        task_id=ctx.task_id,
        tier=Tier.MUTATION,
        framework=FRAMEWORK,
        verdict=verdict,
        passed=passed,
        mutation=stats,
        findings=findings,
    )


# --- helpers -------------------------------------------------------------


def _have_mutmut() -> bool:
    try:
        import mutmut  # noqa: F401  # presence-check only
    except ImportError:
        return False
    return True


def _write_mutmut_config(workdir: Path, sources: list[FileChange]) -> None:
    """Overlay a ``[tool.mutmut]`` block on ``pyproject.toml``.

    mutmut 3.5 looks up ``paths_to_mutate`` in ``pyproject.toml`` under
    ``[tool.mutmut]``, falling back to ``setup.cfg``. We append a fresh
    ``[tool.mutmut]`` block (TOML allows duplicate tables only across
    files, but mutmut tolerates a single-section overlay file as well, so
    if a ``pyproject.toml`` already exists we drop our config into
    ``mutmut.cfg`` which mutmut also recognises via the legacy loader).

    We write TOML by hand rather than pulling in ``tomli_w`` so the
    runner has no extra runtime dep — the format is fixed and trivial.
    """
    paths = sorted(str(s.path).replace(os.sep, "/") for s in sources)
    paths_value = ",".join(paths)
    body = (
        "[tool.mutmut]\n"
        f'paths_to_mutate = "{paths_value}"\n'
        'runner = "python -m pytest -x -q"\n'
        'tests_dir = "tests/"\n'
        "backup = false\n"
        "use_coverage = false\n"
    )
    pyproject = workdir / "pyproject.toml"
    if pyproject.exists():
        # Append; duplicate [tool.mutmut] would be a TOML error, so first
        # strip any pre-existing block we may have written.
        existing = pyproject.read_text(encoding="utf-8")
        existing = _strip_section(existing, "tool.mutmut")
        pyproject.write_text(existing.rstrip() + "\n\n" + body, encoding="utf-8")
    else:
        pyproject.write_text(body, encoding="utf-8")


def _strip_section(toml: str, header: str) -> str:
    """Remove a ``[header]`` section (and its key/value lines) from ``toml``."""
    out: list[str] = []
    needle = f"[{header}]"
    in_section = False
    for line in toml.splitlines():
        stripped = line.strip()
        if stripped == needle:
            in_section = True
            continue
        if in_section and stripped.startswith("[") and stripped.endswith("]"):
            in_section = False
        if not in_section:
            out.append(line)
    return "\n".join(out)


def _spawn_mutmut(args: list[str], *, cwd: Path, timeout: float) -> int:
    """Spawn ``mutmut`` with the given args. Stdout/stderr go to OUR stderr."""
    try:
        proc = subprocess.run(
            [sys.executable, "-m", "mutmut", *args],
            cwd=str(cwd),
            stdout=sys.stderr,
            stderr=sys.stderr,
            timeout=max(timeout, 1.0),
            check=False,
        )
    except subprocess.TimeoutExpired:
        return 124
    except FileNotFoundError:
        return -1
    return proc.returncode


def _capture_mutmut_results(cwd: Path, timeout: float) -> tuple[int, str]:
    """Run ``mutmut results`` and capture stdout for parsing.

    Unlike ``mutmut run``, ``mutmut results`` is meant to be parsed, so
    stdout-capture is correct.
    """
    try:
        proc = subprocess.run(
            [sys.executable, "-m", "mutmut", "results"],
            cwd=str(cwd),
            capture_output=True,
            text=True,
            timeout=max(timeout, 1.0),
            check=False,
        )
    except subprocess.TimeoutExpired:
        return 124, ""
    except FileNotFoundError:
        return -1, ""
    # Echo to stderr too so the dispatcher sees the trail.
    sys.stderr.write(proc.stderr)
    return proc.returncode, proc.stdout


# mutmut 3.5 results format (excerpted):
#
#     mutmut cache version 4
#     Killed 17 mutants
#     Survived (S) 2 mutants
#     Timeout (T) 0 mutants
#     Suspicious (?) 0 mutants
#     Skipped (s) 0 mutants
#     Not checked 0 mutants
#
#     Surviving mutants:
#     -- mutmut.MutantInfo id=src.foo.bar.x_1
#        src/foo/bar.py:12 :: BinOpMutator '+' -> '-'
#
# We tolerate moderate format drift by matching on labels rather than
# absolute line positions.

_RESULT_LABELS: dict[str, str] = {
    "killed": r"Killed\s+(\d+)",
    "survived": r"Survived(?:\s+\(S\))?\s+(\d+)",
    "timeout": r"Timeout(?:\s+\(T\))?\s+(\d+)",
    "not_covered": r"Not\s+checked\s+(\d+)",
}

_SURVIVOR_LINE = re.compile(
    r"^\s*(?P<file>[^:\s]+):(?P<line>\d+)\s*::\s*(?P<mutator>[^\s]+)\s*"
    r"(?:'(?P<orig>[^']*)'\s*->\s*'(?P<repl>[^']*)')?",
)


def _parse_mutmut_results(stdout: str, sources: list[FileChange]) -> MutationStats:
    counts: dict[str, int] = {}
    for label, pattern in _RESULT_LABELS.items():
        m = re.search(pattern, stdout)
        counts[label] = int(m.group(1)) if m else 0

    survivors: list[SurvivedMutant] = []
    in_surv_block = False
    for line in stdout.splitlines():
        if line.lower().startswith("surviving mutants"):
            in_surv_block = True
            continue
        if not in_surv_block:
            continue
        m = _SURVIVOR_LINE.match(line)
        if m is None:
            continue
        survivors.append(
            SurvivedMutant(
                file=m.group("file"),
                line=int(m.group("line")),
                mutator=m.group("mutator"),
                original=m.group("orig") or "",
                replacement=m.group("repl") or "",
            )
        )

    killed = counts.get("killed", 0)
    survived = counts.get("survived", 0)
    timeout = counts.get("timeout", 0)
    not_covered = counts.get("not_covered", 0)
    total = killed + survived + timeout + not_covered

    denom = killed + survived
    score = (killed / denom) if denom > 0 else 0.0

    return MutationStats(
        killed=killed,
        survived=survived,
        not_covered=not_covered,
        timeout=timeout,
        total=total,
        score=score,
        threshold=MUTATION_THRESHOLD,
        diff_scoped=True,
        mutated_files=sorted({s.path for s in sources}),
        survived_summary=survivors,
    )


def _tool_unavailable(task_id: str, detail: str) -> TestReport:
    return TestReport(
        task_id=task_id,
        tier=Tier.MUTATION,
        framework=FRAMEWORK,
        verdict=Verdict.TOOL_UNAVAILABLE,
        passed=False,
        mutation=MutationStats(diff_scoped=True, threshold=MUTATION_THRESHOLD),
        error=detail,
        findings=[Finding(category="tool_error", severity="warn", detail=detail)],
    )
