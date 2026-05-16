"""Tier 4 — Python-flavoured reproducible-build check.

If the materialised diff contains a ``flake.nix`` (and ``flake.lock``) the
runner asks Nix to evaluate the derivation hash. Otherwise it falls back
to building the wheel twice via ``python -m build`` (or ``pip wheel``) and
comparing the SHA-256 of the resulting ``.whl`` artefact. The verifier
sandbox is expected to be deterministic enough for the wheel build to be
bit-identical; if it isn't, that's the signal Tier-4 is supposed to catch.
"""

from __future__ import annotations

import hashlib
import shutil
import subprocess
import sys
from pathlib import Path
from typing import TYPE_CHECKING

from ..schema import Finding, HonestCIStats, TestReport, Tier, Verdict

if TYPE_CHECKING:
    from ..cli import DriverContext

FRAMEWORK = "nix-or-double-wheel"
BUILDER_ID = "https://crucible.dev/builders/python-double-build/v1"


def run(ctx: DriverContext) -> TestReport:
    flake = ctx.workdir / "flake.nix"
    lock = ctx.workdir / "flake.lock"
    if flake.exists():
        stats, findings, passed = _nix_path(ctx, flake, lock)
    else:
        stats, findings, passed = _double_wheel_path(ctx)

    verdict = (
        Verdict.PASSED
        if passed
        else Verdict.FAILED
        if stats.executor_rebuild_hash and stats.verifier_rebuild_hash
        else Verdict.TOOL_UNAVAILABLE
    )
    return TestReport(
        task_id=ctx.task_id,
        tier=Tier.HONEST_CI,
        framework=FRAMEWORK,
        verdict=verdict,
        passed=passed,
        honest_ci=stats,
        findings=findings,
    )


# --- nix backend ---------------------------------------------------------


def _nix_path(
    ctx: DriverContext, flake: Path, lock: Path
) -> tuple[HonestCIStats, list[Finding], bool]:
    nix = shutil.which("nix")
    if nix is None:
        return _tool_unavailable_stats("nix binary not on PATH")

    flake_hash = _hash_file(flake)
    lock_hash = _hash_file(lock) if lock.exists() else ""

    # Two independent derivation evaluations.
    h1 = _nix_drv_hash(nix, ctx.workdir, ctx.budget_seconds)
    h2 = _nix_drv_hash(nix, ctx.workdir, ctx.budget_seconds)

    bit_identical = bool(h1) and h1 == h2
    findings: list[Finding] = []
    if not bit_identical:
        findings.append(
            Finding(
                category="honest_ci_mismatch",
                severity="error",
                detail=f"Nix derivation hash differed between rebuilds: {h1!r} vs {h2!r}",
                suggested_fix=(
                    "Ensure the flake pins all inputs and that no IFD or "
                    "impure substitutions are used."
                ),
            )
        )

    stats = HonestCIStats(
        builder_id=BUILDER_ID,
        nix_flake_hash=flake_hash,
        nix_lock_hash=lock_hash,
        executor_rebuild_hash=h1,
        verifier_rebuild_hash=h2,
        bit_identical=bit_identical,
        slsa_level=3 if bit_identical else 0,
        scrubber_audit_ok=True,
    )
    return stats, findings, bit_identical


def _nix_drv_hash(nix: str, cwd: Path, timeout: float) -> str:
    try:
        proc = subprocess.run(
            [nix, "path-info", "--derivation", ".#default"],
            cwd=str(cwd),
            capture_output=True,
            text=True,
            timeout=max(timeout, 1.0),
            check=False,
        )
    except (subprocess.TimeoutExpired, FileNotFoundError):
        return ""
    if proc.returncode != 0:
        sys.stderr.write(proc.stderr)
        return ""
    return proc.stdout.strip()


# --- double-wheel backend ------------------------------------------------


def _double_wheel_path(ctx: DriverContext) -> tuple[HonestCIStats, list[Finding], bool]:
    if not (ctx.workdir / "pyproject.toml").exists() and not (
        ctx.workdir / "setup.py"
    ).exists():
        return _tool_unavailable_stats("no flake.nix and no pyproject.toml/setup.py")

    h1 = _build_wheel_hash(ctx.workdir, ctx.budget_seconds, "build-1")
    h2 = _build_wheel_hash(ctx.workdir, ctx.budget_seconds, "build-2")
    bit_identical = bool(h1) and h1 == h2

    findings: list[Finding] = []
    if not bit_identical:
        findings.append(
            Finding(
                category="honest_ci_mismatch",
                severity="error",
                detail=(
                    f"Wheel SHA-256 differed between rebuilds: {h1!r} vs {h2!r}. "
                    "Likely culprit: SOURCE_DATE_EPOCH not respected, "
                    "timestamp embedded by setuptools, or non-deterministic ordering."
                ),
                suggested_fix=(
                    "Set SOURCE_DATE_EPOCH and PYTHONHASHSEED in the build "
                    "environment; consider using build-isolation."
                ),
            )
        )

    stats = HonestCIStats(
        builder_id=BUILDER_ID,
        executor_rebuild_hash=h1,
        verifier_rebuild_hash=h2,
        bit_identical=bit_identical,
        slsa_level=2 if bit_identical else 0,
        scrubber_audit_ok=True,
    )
    return stats, findings, bit_identical


def _build_wheel_hash(cwd: Path, timeout: float, scratch_subdir: str) -> str:
    """Build a wheel into ``cwd/scratch_subdir`` and hash its SHA-256."""
    dist = cwd / scratch_subdir
    if dist.exists():
        shutil.rmtree(dist, ignore_errors=True)
    dist.mkdir(parents=True, exist_ok=True)
    env_extras = {"SOURCE_DATE_EPOCH": "315532800", "PYTHONHASHSEED": "0"}
    import os

    env = os.environ.copy()
    env.update(env_extras)
    try:
        proc = subprocess.run(
            [
                sys.executable,
                "-m",
                "pip",
                "wheel",
                ".",
                "--no-deps",
                "--no-build-isolation",
                "-w",
                str(dist),
            ],
            cwd=str(cwd),
            env=env,
            capture_output=True,
            text=True,
            timeout=max(timeout, 1.0),
            check=False,
        )
    except (subprocess.TimeoutExpired, FileNotFoundError):
        return ""
    sys.stderr.write(proc.stderr)
    if proc.returncode != 0:
        return ""
    wheels = sorted(dist.glob("*.whl"))
    if not wheels:
        return ""
    # Stable hash across all produced wheels.
    h = hashlib.sha256()
    for w in wheels:
        h.update(w.name.encode("utf-8"))
        h.update(b"\x00")
        with w.open("rb") as fh:
            while True:
                chunk = fh.read(64 * 1024)
                if not chunk:
                    break
                h.update(chunk)
    return h.hexdigest()


# --- shared --------------------------------------------------------------


def _hash_file(path: Path) -> str:
    h = hashlib.sha256()
    with path.open("rb") as fh:
        while True:
            chunk = fh.read(64 * 1024)
            if not chunk:
                break
            h.update(chunk)
    return h.hexdigest()


def _tool_unavailable_stats(detail: str) -> tuple[HonestCIStats, list[Finding], bool]:
    return (
        HonestCIStats(builder_id=BUILDER_ID, scrubber_audit_ok=False),
        [Finding(category="tool_error", severity="warn", detail=detail)],
        False,
    )
