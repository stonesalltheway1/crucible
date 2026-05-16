"""Tier 2 — schemathesis contract tests against OpenAPI specs.

Looks at ``SpecChange`` entries (from the request) and at the diff for any
OpenAPI/Swagger YAML/JSON file. For each spec we can locate, we invoke
``schemathesis run --checks all --report junit ...`` and parse the JUnit
XML output into ``ContractViolation`` rows.

The runner does NOT spin up a service. Schemathesis can run in
"stateful workflow" mode against a base URL the dispatcher arranges; if no
base URL is provided we run in "schema-only" mode (linting). The base URL
is read from the request's ``contract_base_url`` field — absent fields are
treated as schema-only.
"""

from __future__ import annotations

import hashlib
import subprocess
import sys
import xml.etree.ElementTree as ET
from pathlib import Path
from typing import TYPE_CHECKING

import yaml  # type: ignore[import-untyped]

from ..schema import (
    ContractStats,
    ContractViolation,
    Finding,
    TestReport,
    Tier,
    Verdict,
)

if TYPE_CHECKING:
    from ..cli import DriverContext
    from ..diff import FileChange

FRAMEWORK = "schemathesis~=4.18"


def run(ctx: DriverContext) -> TestReport:
    specs = _discover_specs(ctx.diff_files, ctx.spec_changes, ctx.workdir)
    if not specs:
        return TestReport(
            task_id=ctx.task_id,
            tier=Tier.CONTRACT,
            framework=FRAMEWORK,
            verdict=Verdict.SKIPPED,
            passed=True,
            contract=ContractStats(),
        )

    if not _have_schemathesis():
        return _tool_unavailable(ctx.task_id, "schemathesis not importable in sandbox")

    base_url = str(ctx.request.get("contract_base_url") or "").strip()
    violations: list[ContractViolation] = []
    checks: set[str] = {
        "not_a_server_error",
        "status_code_conformance",
        "content_type_conformance",
        "response_schema_conformance",
    }
    spec_hash = ""
    spec_path = ""
    workflows_run = 0
    aggregate_rc = 0

    for spec in specs:
        spec_path = str(spec.relative_to(ctx.workdir).as_posix())
        spec_hash = _hash_file(spec)
        junit_path = ctx.workdir / f"schemathesis-{spec.stem}.junit.xml"
        rc = _run_schemathesis(
            spec=spec,
            base_url=base_url,
            junit_path=junit_path,
            cwd=ctx.workdir,
            timeout=ctx.budget_seconds,
        )
        aggregate_rc = aggregate_rc or rc
        if junit_path.exists():
            workflows_run += 1
            violations.extend(_parse_junit(junit_path))

    stats = ContractStats(
        spec_path=spec_path,
        spec_hash=spec_hash,
        stateful_workflows=workflows_run,
        checks=sorted(checks),
        violations=violations,
    )

    findings: list[Finding] = []
    for v in violations[:50]:  # cap surface
        findings.append(
            Finding(
                category="contract_violation",
                severity="error",
                detail=f"{v.method} {v.endpoint}: {v.check} — {v.detail}",
                suggested_fix=(
                    "Update the OpenAPI spec to match the implementation "
                    "(or the implementation to match the spec)."
                ),
            )
        )

    passed = aggregate_rc == 0 and not violations
    verdict = (
        Verdict.PASSED if passed
        else Verdict.FAILED if violations
        else Verdict.TOOL_UNAVAILABLE
    )

    return TestReport(
        task_id=ctx.task_id,
        tier=Tier.CONTRACT,
        framework=FRAMEWORK,
        verdict=verdict,
        passed=passed,
        contract=stats,
        findings=findings,
    )


# --- discovery -----------------------------------------------------------


def _discover_specs(
    diff_files: list[FileChange],
    spec_changes: list[dict[str, object]],
    workdir: Path,
) -> list[Path]:
    specs: list[Path] = []
    seen: set[Path] = set()

    def _add(path: Path) -> None:
        rp = path.resolve()
        if rp not in seen and rp.exists():
            specs.append(rp)
            seen.add(rp)

    # 1) Anything the dispatcher flagged as a spec change.
    for sc in spec_changes:
        kind = str(sc.get("kind", "")).lower()
        path = str(sc.get("path", ""))
        if not path:
            continue
        if kind not in {"openapi", "graphql", ""}:
            continue  # schemathesis handles openapi + (with extension) graphql
        _add(workdir / path)

    # 2) Pattern-match anything that looks like an openapi/swagger doc.
    for fc in diff_files:
        if not fc.is_openapi:
            continue
        candidate = workdir / fc.path
        if not candidate.exists():
            continue
        if _looks_like_openapi(candidate):
            _add(candidate)

    return specs


def _looks_like_openapi(path: Path) -> bool:
    try:
        text = path.read_text(encoding="utf-8")
    except OSError:
        return False
    try:
        doc = yaml.safe_load(text)
    except yaml.YAMLError:
        return False
    if not isinstance(doc, dict):
        return False
    return any(k in doc for k in ("openapi", "swagger"))


def _hash_file(path: Path) -> str:
    h = hashlib.sha256()
    with path.open("rb") as fh:
        while True:
            chunk = fh.read(64 * 1024)
            if not chunk:
                break
            h.update(chunk)
    return h.hexdigest()


# --- schemathesis invocation --------------------------------------------


def _have_schemathesis() -> bool:
    try:
        import schemathesis  # noqa: F401
    except ImportError:
        return False
    return True


def _run_schemathesis(
    *,
    spec: Path,
    base_url: str,
    junit_path: Path,
    cwd: Path,
    timeout: float,
) -> int:
    args: list[str] = [
        sys.executable,
        "-m",
        "schemathesis.cli",
        "run",
        "--checks=all",
        "--report=junit",
        f"--report-junit-path={junit_path}",
        str(spec),
    ]
    if base_url:
        args.extend(["--base-url", base_url])
    try:
        proc = subprocess.run(
            args,
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


# --- JUnit parsing -------------------------------------------------------


def _parse_junit(junit_path: Path) -> list[ContractViolation]:
    try:
        tree = ET.parse(junit_path)
    except (OSError, ET.ParseError):
        return []
    out: list[ContractViolation] = []
    for case in tree.iter("testcase"):
        for failure in case.findall("failure"):
            classname = case.attrib.get("classname", "")
            name = case.attrib.get("name", "")
            # schemathesis encodes "METHOD /path" in `name`.
            method = ""
            endpoint = name
            if " " in name:
                method, endpoint = name.split(" ", 1)
            out.append(
                ContractViolation(
                    endpoint=endpoint,
                    method=method,
                    check=classname or failure.attrib.get("type", "unknown"),
                    detail=(failure.text or failure.attrib.get("message", "")).strip()[:512],
                )
            )
    return out


def _tool_unavailable(task_id: str, detail: str) -> TestReport:
    return TestReport(
        task_id=task_id,
        tier=Tier.CONTRACT,
        framework=FRAMEWORK,
        verdict=Verdict.TOOL_UNAVAILABLE,
        passed=False,
        contract=ContractStats(),
        error=detail,
        findings=[Finding(category="tool_error", severity="warn", detail=detail)],
    )
