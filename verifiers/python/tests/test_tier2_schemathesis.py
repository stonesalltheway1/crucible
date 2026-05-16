"""Tests for the Tier 2 schemathesis driver.

End-to-end execution requires a running HTTP target plus schemathesis, so
we keep those gated and focus the always-run tests on discovery + JUnit
parsing.
"""

from __future__ import annotations

import textwrap
import xml.etree.ElementTree as ET
from pathlib import Path

import pytest

from crucible_verify_python.cli import DriverContext
from crucible_verify_python.diff import FileChange
from crucible_verify_python.schema import Tier, Verdict
from crucible_verify_python.tiers import tier2_contract
from crucible_verify_python.tiers.tier2_contract import (
    _discover_specs,
    _looks_like_openapi,
    _parse_junit,
)


def _write_openapi(tmp_path: Path, name: str = "openapi.yaml") -> Path:
    spec = textwrap.dedent(
        """\
        openapi: 3.0.3
        info:
          title: Toy API
          version: 1.0.0
        paths:
          /ping:
            get:
              responses:
                '200':
                  description: OK
        """
    )
    p = tmp_path / name
    p.parent.mkdir(parents=True, exist_ok=True)
    p.write_text(spec, encoding="utf-8")
    return p


def _write_junit(tmp_path: Path, *, with_failure: bool) -> Path:
    if with_failure:
        body = """<?xml version="1.0"?>
<testsuites>
  <testsuite name="schemathesis">
    <testcase classname="response_schema_conformance" name="GET /ping">
      <failure type="schema">expected integer, got string</failure>
    </testcase>
  </testsuite>
</testsuites>
"""
    else:
        body = """<?xml version="1.0"?>
<testsuites>
  <testsuite name="schemathesis">
    <testcase classname="response_schema_conformance" name="GET /ping"/>
  </testsuite>
</testsuites>
"""
    p = tmp_path / "junit.xml"
    p.write_text(body, encoding="utf-8")
    return p


def test_looks_like_openapi_true(tmp_path: Path) -> None:
    p = _write_openapi(tmp_path)
    assert _looks_like_openapi(p) is True


def test_looks_like_openapi_false_for_random_yaml(tmp_path: Path) -> None:
    p = tmp_path / "data.yaml"
    p.write_text("foo: 1\nbar: 2\n", encoding="utf-8")
    assert _looks_like_openapi(p) is False


def test_discover_specs_picks_up_openapi_from_diff(tmp_path: Path) -> None:
    p = _write_openapi(tmp_path, "api/openapi.yaml")
    # Move it to align with diff path.
    target = tmp_path / "api" / "openapi.yaml"
    target.parent.mkdir(parents=True, exist_ok=True)
    if p != target:
        target.write_text(p.read_text(encoding="utf-8"), encoding="utf-8")
    diff = [
        FileChange(
            path="api/openapi.yaml",
            action="modify",
            content=target.read_text(encoding="utf-8"),
        )
    ]
    specs = _discover_specs(diff, [], tmp_path)
    assert len(specs) == 1
    assert specs[0].name == "openapi.yaml"


def test_discover_specs_honours_explicit_spec_changes(tmp_path: Path) -> None:
    target = tmp_path / "spec.yaml"
    target.write_text(_write_openapi(tmp_path, "tmp_openapi.yaml").read_text(), encoding="utf-8")
    specs = _discover_specs(
        diff_files=[],
        spec_changes=[{"path": "spec.yaml", "kind": "openapi"}],
        workdir=tmp_path,
    )
    assert any(s.name == "spec.yaml" for s in specs)


def test_parse_junit_extracts_violation(tmp_path: Path) -> None:
    p = _write_junit(tmp_path, with_failure=True)
    violations = _parse_junit(p)
    assert len(violations) == 1
    v = violations[0]
    assert v.endpoint == "/ping"
    assert v.method == "GET"
    assert v.check == "response_schema_conformance"
    assert "expected integer" in v.detail


def test_parse_junit_returns_empty_on_success(tmp_path: Path) -> None:
    p = _write_junit(tmp_path, with_failure=False)
    assert _parse_junit(p) == []


def test_parse_junit_returns_empty_on_malformed_xml(tmp_path: Path) -> None:
    p = tmp_path / "bad.xml"
    p.write_text("<<<not xml>>>", encoding="utf-8")
    assert _parse_junit(p) == []


def test_no_specs_skips(tmp_path: Path) -> None:
    ctx = DriverContext(
        task_id="t-no-spec",
        diff_files=[FileChange(path="README.md", action="modify", content="x")],
        spec_changes=[],
        workdir=tmp_path,
        budget_seconds=30.0,
        request={},
    )
    report = tier2_contract.run(ctx)
    assert report.tier == Tier.CONTRACT
    assert report.verdict == Verdict.SKIPPED
    assert report.passed is True


def _have_schemathesis() -> bool:
    try:
        import schemathesis  # noqa: F401
    except ImportError:
        return False
    return True


@pytest.mark.skipif(_have_schemathesis(), reason="negative-path test runs when schemathesis is absent")
def test_tool_unavailable_when_schemathesis_missing(tmp_path: Path) -> None:
    p = _write_openapi(tmp_path)
    diff = [
        FileChange(
            path="openapi.yaml",
            action="modify",
            content=p.read_text(encoding="utf-8"),
        )
    ]
    ctx = DriverContext(
        task_id="t-no-st",
        diff_files=diff,
        spec_changes=[],
        workdir=tmp_path,
        budget_seconds=30.0,
        request={},
    )
    report = tier2_contract.run(ctx)
    assert report.verdict == Verdict.TOOL_UNAVAILABLE
    assert report.passed is False


def test_invalid_junit_file_does_not_crash_parser(tmp_path: Path) -> None:
    # XML that's well-formed but lacks the expected structure.
    p = tmp_path / "empty.xml"
    p.write_text("<root/>", encoding="utf-8")
    # Should not throw.
    assert _parse_junit(p) == []
    # Sanity: ensure ElementTree confirms our test file parses.
    assert ET.parse(p).getroot().tag == "root"
