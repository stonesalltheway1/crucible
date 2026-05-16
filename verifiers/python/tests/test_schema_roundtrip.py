"""Round-trip + invariant tests for the TestReport schema mirror.

The schema is canonical on the Go side. These tests assert:

1. Every field documented in ``apps/verifier/pkg/testreport/testreport.go``
   is present in the Python dataclasses (by JSON tag).
2. Zero-valued fields are elided in the same places Go's ``,omitempty`` does.
3. ``TestReport.validate()`` enforces the same invariants as Go's
   ``TestReport.Validate``.
4. The reasoning-leak audit refuses requests with ``reasoning`` or
   ``agent_trace`` keys.
"""

from __future__ import annotations

import io
import json
import sys
from datetime import UTC, datetime

import pytest

from crucible_verify_python import REPORT_DELIMITER, REPORTER_ID
from crucible_verify_python.audit import LeakageError, audit_payload
from crucible_verify_python.cli import run
from crucible_verify_python.schema import (
    ContractStats,
    ContractViolation,
    Counterexample,
    Finding,
    HonestCIStats,
    Language,
    MutationStats,
    PBTStats,
    ProofStats,
    SurvivedMutant,
    TestReport,
    Tier,
    Verdict,
)

# --- presence ------------------------------------------------------------


def test_testreport_has_all_documented_fields() -> None:
    """Every Go JSON tag is present in our schema dict."""
    r = TestReport(
        task_id="t",
        tier=Tier.MUTATION,
        language=Language.PYTHON,
        framework="mutmut",
        verdict=Verdict.PASSED,
        passed=True,
        mutation=MutationStats(killed=1, survived=0, total=1, score=1.0),
    )
    d = r.to_json_dict()
    required = {
        "schema_version",
        "task_id",
        "diff_hash",
        "tier",
        "language",
        "framework",
        "verdict",
        "passed",
        "started_at",
        "finished_at",
        "duration_seconds",
        "wall_clock_budget_seconds",
        "reporter_id",
    }
    missing = required - set(d.keys())
    assert not missing, f"required Go JSON fields missing: {missing}"


def test_omitempty_elides_zero_valued_optionals() -> None:
    """``error``, ``tool_digest``, etc. must not appear when empty."""
    r = TestReport(
        task_id="t",
        tier=Tier.MUTATION,
        language=Language.PYTHON,
        verdict=Verdict.PASSED,
        passed=True,
    )
    d = r.to_json_dict()
    # These should be absent when zero-valued — Go's omitempty.
    assert "error" not in d
    assert "tool_digest" not in d
    assert "reporter_oidc_subject" not in d
    assert "findings" not in d
    assert "mutation" not in d  # not set on this report
    assert "pbt" not in d


def test_omitempty_present_when_populated() -> None:
    r = TestReport(
        task_id="t",
        tier=Tier.MUTATION,
        verdict=Verdict.FAILED,
        mutation=MutationStats(
            killed=2,
            survived=3,
            total=5,
            score=0.4,
            mutated_files=["app.py"],
            survived_summary=[SurvivedMutant(file="app.py", line=10, mutator="BinOp")],
        ),
        findings=[Finding(category="mutation_survived", severity="error", detail="x")],
        error="boom",
    )
    d = r.to_json_dict()
    assert d["mutation"]["mutated_files"] == ["app.py"]
    assert d["mutation"]["survived_summary"][0]["mutator"] == "BinOp"
    assert d["error"] == "boom"
    assert d["findings"][0]["category"] == "mutation_survived"


# --- invariants ----------------------------------------------------------


def test_validate_rejects_non_diff_scoped_mutation() -> None:
    r = TestReport(
        task_id="t",
        tier=Tier.MUTATION,
        verdict=Verdict.PASSED,
        passed=True,
        mutation=MutationStats(killed=1, survived=0, total=1, score=1.0, diff_scoped=False),
    )
    with pytest.raises(ValueError, match="diff-scoped"):
        r.validate()


def test_validate_rejects_killed_plus_survived_over_total() -> None:
    r = TestReport(
        task_id="t",
        tier=Tier.MUTATION,
        verdict=Verdict.FAILED,
        mutation=MutationStats(killed=5, survived=5, total=2, score=0.5),
    )
    with pytest.raises(ValueError, match="killed\\+survived"):
        r.validate()


def test_validate_rejects_pbt_below_iterations_min() -> None:
    r = TestReport(
        task_id="t",
        tier=Tier.PBT,
        verdict=Verdict.PASSED,
        pbt=PBTStats(iterations=100, iterations_min=10_000),
    )
    with pytest.raises(ValueError, match="iterations"):
        r.validate()


def test_validate_requires_task_id() -> None:
    r = TestReport(task_id="", tier=Tier.MUTATION, verdict=Verdict.PASSED)
    with pytest.raises(ValueError, match="task_id"):
        r.validate()


def test_validate_rejects_wrong_schema_version() -> None:
    r = TestReport(
        task_id="t", tier=Tier.MUTATION, verdict=Verdict.PASSED, schema_version="2"
    )
    with pytest.raises(ValueError, match="schema_version"):
        r.validate()


# --- canonical JSON ------------------------------------------------------


def test_canonical_json_is_sorted_and_stable() -> None:
    now = datetime(2026, 5, 15, 12, 0, 0, tzinfo=UTC)
    r = TestReport(
        task_id="t",
        tier=Tier.MUTATION,
        framework="mutmut",
        verdict=Verdict.PASSED,
        passed=True,
        started_at=now,
        finished_at=now,
        mutation=MutationStats(killed=1, survived=0, total=1, score=1.0),
    )
    canon = r.canonical_json()
    # Sorted keys -> "diff_hash" precedes "duration_seconds".
    a = canon.index('"diff_hash"')
    b = canon.index('"duration_seconds"')
    assert a < b
    # Round-trips through json.loads cleanly.
    parsed = json.loads(canon)
    assert parsed["task_id"] == "t"
    assert parsed["mutation"]["score"] == 1.0


def test_to_json_compact_no_whitespace() -> None:
    r = TestReport(task_id="t", tier=Tier.MUTATION, verdict=Verdict.PASSED)
    s = r.to_json()
    assert ", " not in s
    assert ": " not in s


# --- all stats unions ----------------------------------------------------


def test_all_per_tier_stats_serialise() -> None:
    """Sanity-check that each of the five stats blocks serialise."""
    cases: list[TestReport] = [
        TestReport(
            task_id="t",
            tier=Tier.PBT,
            verdict=Verdict.FAILED,
            pbt=PBTStats(
                iterations=10_000,
                iterations_min=10_000,
                counterexamples=[Counterexample(property="p", shrunk="x=0")],
            ),
        ),
        TestReport(
            task_id="t",
            tier=Tier.CONTRACT,
            verdict=Verdict.FAILED,
            contract=ContractStats(
                spec_path="openapi.yaml",
                violations=[
                    ContractViolation(
                        endpoint="/users", method="GET", check="schema", detail="bad"
                    )
                ],
            ),
        ),
        TestReport(
            task_id="t",
            tier=Tier.PROOF,
            verdict=Verdict.TOOL_UNAVAILABLE,
            proof=ProofStats(prover="dafny", timed_out=False),
        ),
        TestReport(
            task_id="t",
            tier=Tier.HONEST_CI,
            verdict=Verdict.PASSED,
            passed=True,
            honest_ci=HonestCIStats(
                builder_id="b",
                executor_rebuild_hash="abc",
                verifier_rebuild_hash="abc",
                bit_identical=True,
                slsa_level=3,
                scrubber_audit_ok=True,
            ),
        ),
    ]
    for r in cases:
        d = r.to_json_dict()
        # No KeyError, round-trips to JSON.
        json.dumps(d)


# --- audit guard ---------------------------------------------------------


def test_audit_rejects_reasoning_key() -> None:
    with pytest.raises(LeakageError) as ei:
        audit_payload({"task_id": "t", "reasoning": "boom"})
    assert "reasoning" in str(ei.value)


def test_audit_rejects_nested_agent_trace() -> None:
    payload = {
        "task_id": "t",
        "diff": {"files": [{"path": "a.py", "agent_trace": "leak"}]},
    }
    with pytest.raises(LeakageError) as ei:
        audit_payload(payload)
    assert "agent_trace" in str(ei.value)


def test_audit_rejects_chain_of_thought_underscore_or_dash() -> None:
    with pytest.raises(LeakageError):
        audit_payload({"chain_of_thought": ["x"]})
    with pytest.raises(LeakageError):
        audit_payload({"chain-of-thought": ["x"]})


def test_audit_allows_clean_payload() -> None:
    audit_payload(
        {
            "task_id": "t",
            "diff": {"files": [{"path": "a.py", "action": "modify", "content": "x"}]},
            "routing": {"executor_vendor": "anthropic", "verifier_vendor": "google"},
        }
    )


# --- CLI exit-code contract ---------------------------------------------


def _invoke_cli(payload: dict, *, tier: str) -> tuple[int, str, str]:
    """Run the CLI in-process via ``cli.run`` with stdin/stdout redirected."""
    import io as _io

    old_stdin, old_stdout, old_stderr = sys.stdin, sys.stdout, sys.stderr
    sys.stdin = _io.StringIO(json.dumps(payload))
    sys.stdout = out = _io.StringIO()
    sys.stderr = err = _io.StringIO()
    try:
        code = run(["--tier", tier])
    finally:
        sys.stdin = old_stdin
        sys.stdout = old_stdout
        sys.stderr = old_stderr
    return code, out.getvalue(), err.getvalue()


def test_cli_refuses_request_with_reasoning_key() -> None:
    code, out, err = _invoke_cli(
        {"task_id": "t", "reasoning": "I planned to..."},
        tier="tier_0_mutation",
    )
    assert code == 2, f"expected exit 2, got {code}; stderr={err}"
    assert "REFUSING" in err
    assert out == ""  # No report when refused.


def test_cli_refuses_request_with_agent_trace_key() -> None:
    code, out, err = _invoke_cli(
        {"task_id": "t", "agent_trace": ["step1", "step2"]},
        tier="tier_0_mutation",
    )
    assert code == 2
    assert "agent_trace" in err
    assert out == ""


def test_cli_emits_delimiter_and_valid_json_on_empty_diff() -> None:
    code, out, _err = _invoke_cli(
        {"task_id": "t-empty", "diff": {"files": []}},
        tier="tier_0_mutation",
    )
    assert code == 0
    assert REPORT_DELIMITER in out
    body = out.split(REPORT_DELIMITER, 1)[1].strip()
    parsed = json.loads(body)
    assert parsed["task_id"] == "t-empty"
    assert parsed["tier"] == "tier_0_mutation"
    assert parsed["language"] == "python"
    assert parsed["reporter_id"] == REPORTER_ID
    assert parsed["schema_version"] == "1"


def test_cli_empty_stdin_is_procedural_failure() -> None:
    old_stdin, old_stdout, old_stderr = sys.stdin, sys.stdout, sys.stderr
    sys.stdin = io.StringIO("")
    sys.stdout = io.StringIO()
    sys.stderr = io.StringIO()
    try:
        code = run(["--tier", "tier_0_mutation"])
    finally:
        sys.stdin = old_stdin
        sys.stdout = old_stdout
        sys.stderr = old_stderr
    assert code == 1
