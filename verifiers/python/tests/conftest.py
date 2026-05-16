"""Shared pytest fixtures for the crucible-verify-python test suite.

Test layout:

- ``fixtures/strong_tests`` — a fixture project with thorough tests that
  should achieve >=85% mutation score against its source files.
- ``fixtures/weak_tests`` — same source files but with intentionally
  shallow tests; mutation should be rejected.
- ``fixtures/hypothesis_breaking`` — a property test that intentionally
  falsifies, so we can assert counterexamples surface.

The fixtures are *generated in code* rather than committed to disk to keep
the package tree minimal and to make it trivial to vary them per test.
"""

from __future__ import annotations

import json
import sys
from collections.abc import Iterator
from dataclasses import dataclass, field
from pathlib import Path
from typing import Any

import pytest

# Make ``crucible_verify_python`` importable when running from a checkout.
ROOT = Path(__file__).resolve().parent.parent
if str(ROOT) not in sys.path:
    sys.path.insert(0, str(ROOT))


@dataclass(slots=True)
class RequestBuilder:
    """Helper to construct VerificationRequest payloads for tests."""

    task_id: str = "task_fixture_001"
    tenant_id: str = "tenant_test"
    base_sha: str = "0" * 40
    executor_sandbox_id: str = "sb_executor_test"
    files: list[dict[str, Any]] = field(default_factory=list)
    spec_changes: list[dict[str, Any]] = field(default_factory=list)
    extra: dict[str, Any] = field(default_factory=dict)
    routing: dict[str, Any] = field(
        default_factory=lambda: {
            "executor_model": "claude-opus-4-7",
            "executor_vendor": "anthropic",
            "executor_tier": "tier_a",
            "verifier_model": "gemini-3-1-pro",
            "verifier_vendor": "google",
            "verifier_tier": "tier_a",
        }
    )

    def add_file(self, path: str, content: str, action: str = "modify") -> RequestBuilder:
        self.files.append({"path": path, "action": action, "content": content})
        return self

    def build(self) -> dict[str, Any]:
        body: dict[str, Any] = {
            "task_id": self.task_id,
            "tenant_id": self.tenant_id,
            "base_sha": self.base_sha,
            "executor_sandbox_id": self.executor_sandbox_id,
            "diff": {"files": list(self.files), "base_sha": self.base_sha},
            "routing": dict(self.routing),
            "languages": ["python"],
            "spec_changes": list(self.spec_changes),
        }
        body.update(self.extra)
        return body

    def to_json(self) -> str:
        return json.dumps(self.build())


@pytest.fixture
def request_builder() -> RequestBuilder:
    return RequestBuilder()


# --- on-disk fixture projects -------------------------------------------


_STRONG_SOURCE = '''\
"""A tiny library with strong, mutation-resistant tests."""


def is_palindrome(s: str) -> bool:
    """Return True iff ``s`` reads the same forwards and backwards."""
    return s == s[::-1]


def clamp(x: int, lo: int, hi: int) -> int:
    """Clamp ``x`` into ``[lo, hi]``. Assumes ``lo <= hi``."""
    if x < lo:
        return lo
    if x > hi:
        return hi
    return x
'''

_STRONG_TESTS = '''\
from app import clamp, is_palindrome


def test_is_palindrome_true_cases():
    assert is_palindrome("")
    assert is_palindrome("a")
    assert is_palindrome("aba")
    assert is_palindrome("abba")


def test_is_palindrome_false_cases():
    assert not is_palindrome("ab")
    assert not is_palindrome("abc")
    assert not is_palindrome("Abba")  # case-sensitive


def test_clamp_within_range():
    assert clamp(5, 0, 10) == 5
    assert clamp(0, 0, 10) == 0
    assert clamp(10, 0, 10) == 10


def test_clamp_below():
    assert clamp(-1, 0, 10) == 0
    assert clamp(-100, 0, 10) == 0


def test_clamp_above():
    assert clamp(11, 0, 10) == 10
    assert clamp(1000, 0, 10) == 10


def test_clamp_negative_range():
    assert clamp(-5, -10, -1) == -5
    assert clamp(0, -10, -1) == -1
'''

_WEAK_TESTS = '''\
from app import clamp, is_palindrome


def test_smoke():
    # Only one trivial assertion per function — survives almost any mutant.
    assert is_palindrome("aba") is True
    assert isinstance(clamp(0, 0, 1), int)
'''

_HYPOTHESIS_FAILING = '''\
from hypothesis import given, strategies as st


def buggy_abs(x: int) -> int:
    # Off-by-one: returns -1 for the smallest int. Hypothesis will find it.
    if x == -2**31:
        return -1
    return abs(x)


@given(st.integers())
def test_buggy_abs_nonneg(x):
    assert buggy_abs(x) >= 0
'''

_HYPOTHESIS_PASSING = '''\
from hypothesis import given, strategies as st


def my_sorted(xs):
    return sorted(xs)


@given(st.lists(st.integers()))
def test_sorted_is_idempotent(xs):
    assert my_sorted(my_sorted(xs)) == my_sorted(xs)
'''


@pytest.fixture
def strong_fixture_files() -> list[dict[str, str]]:
    return [
        {"path": "app.py", "action": "modify", "content": _STRONG_SOURCE},
        {"path": "tests/test_app.py", "action": "modify", "content": _STRONG_TESTS},
    ]


@pytest.fixture
def weak_fixture_files() -> list[dict[str, str]]:
    return [
        {"path": "app.py", "action": "modify", "content": _STRONG_SOURCE},
        {"path": "tests/test_app.py", "action": "modify", "content": _WEAK_TESTS},
    ]


@pytest.fixture
def hypothesis_failing_files() -> list[dict[str, str]]:
    return [
        {"path": "tests/test_buggy.py", "action": "create", "content": _HYPOTHESIS_FAILING},
    ]


@pytest.fixture
def hypothesis_passing_files() -> list[dict[str, str]]:
    return [
        {"path": "tests/test_sorted.py", "action": "create", "content": _HYPOTHESIS_PASSING},
    ]


@pytest.fixture
def temp_workdir(tmp_path: Path) -> Iterator[Path]:
    """Yield a fresh, isolated workdir for a tier driver."""
    yield tmp_path
