"""Argument parsing and tier dispatch.

The CLI contract is intentionally narrow: read one JSON document from
stdin, write one TestReport JSON document to stdout (after the
``CRUCIBLE-TESTREPORT`` delimiter), exit 0 on success, exit 1 on a
procedural failure, exit 2 on an audit-guard reject. Anything else on
stdout — pytest preamble, mutmut chatter — is escaped to stderr by the
tier drivers, and the delimiter ensures the dispatcher can locate the
report deterministically.
"""

from __future__ import annotations

import argparse
import contextlib
import json
import os
import sys
import tempfile
import time
import traceback
from collections.abc import Callable
from datetime import UTC, datetime
from pathlib import Path
from typing import Any

from . import REPORT_DELIMITER, REPORTER_ID, REPORTER_VERSION
from .audit import LeakageError, audit_diff_paths, audit_payload
from .diff import FileChange, hash_diff, materialise, parse_diff_files
from .schema import Finding, Language, TestReport, Tier, Verdict
from .tiers import tier0_mutation, tier1_pbt, tier2_contract, tier3_proof, tier4_honest_ci

# Tier -> driver callable. Each driver receives a fully-prepared
# DriverContext and returns the populated TestReport. Drivers MUST NOT
# write to stdout themselves — logs go to stderr.
TierDriver = Callable[["DriverContext"], TestReport]

_DRIVERS: dict[Tier, TierDriver] = {
    Tier.MUTATION: tier0_mutation.run,
    Tier.PBT: tier1_pbt.run,
    Tier.CONTRACT: tier2_contract.run,
    Tier.PROOF: tier3_proof.run,
    Tier.HONEST_CI: tier4_honest_ci.run,
}


class DriverContext:
    """Bundle of inputs every tier driver consumes.

    Kept as a plain class (not a frozen dataclass) so the drivers can
    stash tool-specific scratch state on the instance during a run if
    they need to.
    """

    __slots__ = (
        "budget_seconds",
        "diff_files",
        "request",
        "spec_changes",
        "task_id",
        "workdir",
    )

    def __init__(
        self,
        *,
        task_id: str,
        diff_files: list[FileChange],
        spec_changes: list[dict[str, Any]],
        workdir: Path,
        budget_seconds: float,
        request: dict[str, Any],
    ) -> None:
        self.task_id = task_id
        self.diff_files = diff_files
        self.spec_changes = spec_changes
        self.workdir = workdir
        self.budget_seconds = budget_seconds
        self.request = request


# --- arg parsing ---------------------------------------------------------


def _build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(
        prog="crucible-verify-python",
        description=(
            "Per-language verifier runner. Reads a VerificationRequest "
            "JSON on stdin and writes a TestReport JSON on stdout."
        ),
        allow_abbrev=False,
    )
    parser.add_argument(
        "--tier",
        required=True,
        choices=[t.value for t in Tier],
        help="Tier to run.",
    )
    parser.add_argument(
        "--workdir",
        default="",
        help=(
            "Directory to materialise the diff into. If empty, a fresh "
            "TemporaryDirectory is used."
        ),
    )
    parser.add_argument(
        "--budget-seconds",
        type=float,
        default=0.0,
        help=(
            "Tier wall-clock budget. 0 means use the per-tier default "
            "from docs/01-architecture/verifier-pipeline.md."
        ),
    )
    return parser


# --- entrypoint ----------------------------------------------------------


def run(argv: list[str]) -> int:
    """Run one CLI invocation. Returns the process exit code."""
    args = _build_parser().parse_args(argv)
    tier = Tier(args.tier)

    # 1) Slurp stdin. Empty stdin is a procedural error.
    try:
        raw = sys.stdin.read()
    except OSError as exc:
        print(f"crucible-verify-python: stdin read failed: {exc}", file=sys.stderr)
        return 1
    if not raw.strip():
        print("crucible-verify-python: empty stdin (expected VerificationRequest JSON)",
              file=sys.stderr)
        return 1

    # 2) Parse JSON. Malformed JSON is a procedural error.
    try:
        parsed = json.loads(raw)
    except json.JSONDecodeError as exc:
        print(f"crucible-verify-python: stdin JSON decode failed: {exc}", file=sys.stderr)
        return 1
    if not isinstance(parsed, dict):
        print("crucible-verify-python: stdin payload was not a JSON object", file=sys.stderr)
        return 1
    request: dict[str, Any] = parsed

    # 3) Audit guard. Refuse anything containing reasoning-tagged fields.
    try:
        audit_payload(request)
        # Diff paths get a second pass against the path-pattern denylist.
        diff_paths = [
            str(f.get("path", ""))
            for f in (request.get("diff", {}) or {}).get("files", []) or []
            if isinstance(f, dict)
        ]
        audit_diff_paths(diff_paths)
    except LeakageError as leak:
        print(
            f"crucible-verify-python: REFUSING — {leak}",
            file=sys.stderr,
        )
        return 2

    # 4) Materialise the diff into a temp workdir (or the user-supplied one).
    task_id = str(request.get("task_id") or "")
    if not task_id:
        # Synthesise a deterministic task_id so the dispatcher still gets a
        # well-formed report even on malformed requests — but emit an error
        # finding so the failure surfaces.
        task_id = "unknown-task"

    diff_blob = (request.get("diff") or {}).get("files") or []
    diff_files = parse_diff_files(diff_blob)
    spec_changes_raw = request.get("spec_changes") or []
    spec_changes = [s for s in spec_changes_raw if isinstance(s, dict)]
    diff_hash = hash_diff(diff_files)
    budget = float(args.budget_seconds or _default_budget(tier))

    started = _now()
    start_clock = time.monotonic()

    # Use a context-managed temp dir so we always clean up.
    user_workdir = args.workdir.strip()
    if user_workdir:
        workdir_path = Path(user_workdir)
        workdir_path.mkdir(parents=True, exist_ok=True)
        report = _dispatch(
            tier=tier,
            task_id=task_id,
            diff_files=diff_files,
            spec_changes=spec_changes,
            workdir=workdir_path,
            budget=budget,
            request=request,
            started=started,
            start_clock=start_clock,
            diff_hash=diff_hash,
        )
    else:
        with tempfile.TemporaryDirectory(prefix="crucible-verify-py-") as td:
            workdir_path = Path(td)
            report = _dispatch(
                tier=tier,
                task_id=task_id,
                diff_files=diff_files,
                spec_changes=spec_changes,
                workdir=workdir_path,
                budget=budget,
                request=request,
                started=started,
                start_clock=start_clock,
                diff_hash=diff_hash,
            )

    # 5) Emit the framed report on stdout, logs on stderr.
    #
    # Exit 0 always once a well-formed report is emitted — even on a
    # failed verdict. The verdict is carried in the report body; exit
    # codes are reserved for procedural failures (so the dispatcher can
    # distinguish "I ran and rejected the diff" from "my process
    # crashed").
    _emit(report)
    return 0


def _dispatch(
    *,
    tier: Tier,
    task_id: str,
    diff_files: list[FileChange],
    spec_changes: list[dict[str, Any]],
    workdir: Path,
    budget: float,
    request: dict[str, Any],
    started: datetime,
    start_clock: float,
    diff_hash: str,
) -> TestReport:
    """Materialise diff, run the tier, and stamp common report fields."""
    try:
        materialise(diff_files, workdir)
    except OSError as exc:
        return _procedural_failure(
            task_id=task_id,
            tier=tier,
            diff_hash=diff_hash,
            started=started,
            start_clock=start_clock,
            budget=budget,
            detail=f"workdir materialise failed: {exc}",
        )

    ctx = DriverContext(
        task_id=task_id,
        diff_files=diff_files,
        spec_changes=spec_changes,
        workdir=workdir,
        budget_seconds=budget,
        request=request,
    )

    driver = _DRIVERS[tier]
    try:
        report = driver(ctx)
    except Exception as exc:
        traceback.print_exc(file=sys.stderr)
        return _procedural_failure(
            task_id=task_id,
            tier=tier,
            diff_hash=diff_hash,
            started=started,
            start_clock=start_clock,
            budget=budget,
            detail=f"driver crashed: {type(exc).__name__}: {exc}",
        )

    # Stamp common fields the drivers shouldn't have to fill in.
    report.task_id = report.task_id or task_id
    report.tier = tier
    report.language = Language.PYTHON
    report.diff_hash = report.diff_hash or diff_hash
    report.reporter_id = REPORTER_ID
    report.reporter_version = REPORTER_VERSION
    report.started_at = started
    report.finished_at = _now()
    report.duration_seconds = time.monotonic() - start_clock
    report.wall_clock_budget_seconds = budget
    try:
        report.validate()
    except ValueError as exc:
        # A driver produced an invariant violation — convert to procedural
        # failure rather than crash the dispatcher's JSON parser.
        report.verdict = Verdict.FAILED
        report.passed = False
        report.error = f"invariant violated: {exc}"
        report.findings.append(
            Finding(
                category="tool_error",
                severity="error",
                detail=str(exc),
            )
        )
    return report


def _procedural_failure(
    *,
    task_id: str,
    tier: Tier,
    diff_hash: str,
    started: datetime,
    start_clock: float,
    budget: float,
    detail: str,
) -> TestReport:
    return TestReport(
        task_id=task_id,
        tier=tier,
        language=Language.PYTHON,
        framework="",
        verdict=Verdict.FAILED,
        passed=False,
        diff_hash=diff_hash,
        started_at=started,
        finished_at=_now(),
        duration_seconds=time.monotonic() - start_clock,
        wall_clock_budget_seconds=budget,
        reporter_id=REPORTER_ID,
        reporter_version=REPORTER_VERSION,
        error=detail,
        findings=[Finding(category="tool_error", severity="error", detail=detail)],
    )


def _default_budget(tier: Tier) -> float:
    # Per docs/01-architecture/verifier-pipeline.md.
    return {
        Tier.MUTATION: 30.0,  # 30s default, 2 min max
        Tier.PBT: 300.0,  # 5 min default, 15 min max
        Tier.CONTRACT: 900.0,  # 15 min default, 45 min max
        Tier.PROOF: 600.0,  # Dafny 10 min default
        Tier.HONEST_CI: 300.0,  # 5 min default, 30 min max
    }[tier]


def _emit(report: TestReport) -> None:
    """Write the framed report to stdout (and flush)."""
    # Use the os.write path so the framing is unbuffered — important if a
    # tool subprocess we forked is still draining stderr.
    body = report.to_json()
    framed = f"{REPORT_DELIMITER}\n{body}\n"
    sys.stdout.write(framed)
    sys.stdout.flush()
    # Belt-and-braces: if stdout is a real fd, fsync to flush kernel buffers.
    # ValueError fires when stdout has no fileno (e.g. captured to StringIO);
    # OSError fires on platforms that disallow fsync on pipes — both ignorable.
    with contextlib.suppress(OSError, ValueError):
        os.fsync(sys.stdout.fileno())


def _now() -> datetime:
    return datetime.now(tz=UTC)
