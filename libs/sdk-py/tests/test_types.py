"""Crucible Python SDK type tests."""

from __future__ import annotations

import json
from datetime import datetime, timezone

import pytest
from hypothesis import given, strategies as st

from crucible_sdk import (
    Budget,
    Complexity,
    DestructiveProposal,
    Plan,
    PlanStep,
    Predicates,
    Task,
    TaskStatus,
)


def test_predicates_complete_set() -> None:
    uris = [p.value for p in Predicates]
    assert len(uris) == 14, "expected 14 predicate types"
    for uri in uris:
        assert uri.startswith("https://crucible.dev/")
        assert uri.endswith("/v1")


def test_plan_roundtrip_json() -> None:
    p = Plan(
        task_id="task_1",
        description="d",
        steps=[
            PlanStep(ordinal=1, description="s1"),
            PlanStep(ordinal=2, description="s2"),
        ],
        estimated_cost_usd=1.25,
        estimated_duration_min=10,
        files_to_touch=["a.py"],
        db_migrations=1,
        external_effects=[],
        top_risks=[],
        retry_budget_per_step=3,
        wall_clock_budget_min=30,
        complexity=Complexity.STANDARD,
        plan_hash="0" * 64,
        built_at=datetime(2026, 5, 15, 12, 0, tzinfo=timezone.utc),
    )
    js = p.model_dump_json()
    parsed = json.loads(js)
    assert parsed["task_id"] == "task_1"
    assert parsed["complexity"] == "standard"
    p2 = Plan.model_validate_json(js)
    assert p2.estimated_cost_usd == 1.25
    assert p2.steps[0].retry_budget == 3  # default applied


def test_budget_defaults() -> None:
    b = Budget(cap_usd=1.0)
    assert b.spent_usd == 0.0
    assert b.retries_used == 0


def test_task_status_round_trip() -> None:
    t = Task(
        id="task_1",
        tenant_id="t",
        repo="r",
        base_sha="abc",
        description="d",
        status=TaskStatus.AWAITING_APPROVAL,
        created_at=datetime.now(timezone.utc),
        updated_at=datetime.now(timezone.utc),
        submitted_by="u",
    )
    js = t.model_dump_json()
    parsed = json.loads(js)
    assert parsed["status"] == "awaiting_approval"


def test_destructive_proposal_requires_scope_literal() -> None:
    with pytest.raises(Exception):
        DestructiveProposal.model_validate(
            {
                "task_id": "t",
                "tenant_id": "te",
                "command": "DROP TABLE x",
                "scope": "not-a-scope",  # invalid literal
                "blast_radius": {
                    "affected_resources": [],
                    "reversibility": "snapshot",
                    "impact_score": 0.5,
                },
                "intercepted_at_layer": "syscall-shim",
                "proposed_at": datetime.now(timezone.utc).isoformat(),
                "agent_oidc_subject": "https://x",
            }
        )


def test_plan_rejects_extra_fields() -> None:
    with pytest.raises(Exception):
        Plan.model_validate(
            {
                "task_id": "t",
                "description": "d",
                "estimated_cost_usd": 1,
                "estimated_duration_min": 10,
                "complexity": "standard",
                "plan_hash": "0" * 64,
                "built_at": datetime.now(timezone.utc).isoformat(),
                "ghost_field": "no",  # extra
            }
        )


# Property test: complexity enum has stable string values.
@given(st.sampled_from(list(Complexity)))
def test_complexity_strings_roundtrip(c: Complexity) -> None:
    assert Complexity(c.value) == c
    assert isinstance(c.value, str)
