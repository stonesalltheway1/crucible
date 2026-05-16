"""Cross-tenant isolation tests for the distiller pipeline.

The distiller is upstream of the memory-router's enforcement, so the
key invariant here is that the AdmissionClient sends each candidate
with the correct tenant_id — and refuses to mix tenants in a single
admission call.
"""

from __future__ import annotations

import json
from datetime import datetime, timezone

from crucible_distiller.adapters.github_pr import GitHubPRAdapter, GitHubPRComment
from crucible_distiller.admission.client import AdmissionClient, FakeRouter
from crucible_distiller.extractor.mem0_hierarchical import FakeLLM
from crucible_distiller.judge.llm_judge import FakeJudge
from crucible_distiller.pipeline import PipelineDeps, process_one


def _pair_of_adapters() -> tuple[GitHubPRAdapter, GitHubPRAdapter]:
    a = GitHubPRAdapter(repo="acme/a", comments=[
        GitHubPRComment(pr=1, comment_id="ca", body="use slog over fmt.Printf in handlers", resulted_in_change=True),
    ])
    b = GitHubPRAdapter(repo="acme/b", comments=[
        GitHubPRComment(pr=2, comment_id="cb", body="use slog over fmt.Printf in handlers", resulted_in_change=True),
    ])
    return a, b


def _wire_pipeline() -> tuple[PipelineDeps, FakeRouter]:
    pattern_responses = {
        "slog over fmt.Printf": json.dumps([{
            "category": "Logging",
            "rule": "Use slog over fmt.Printf in service code.",
            "file_glob": "**/*.go",
            "rationale": "structured logs",
            "evidence_quote": "use slog over fmt.Printf",
        }])
    }
    deps = PipelineDeps(
        extractor_client=FakeLLM(pattern_responses=pattern_responses),
        judge_client=FakeJudge(),
        admission=AdmissionClient(FakeRouter()),
    )
    return deps, deps.admission.http  # type: ignore[return-value]


def test_distiller_tags_each_admission_with_correct_tenant() -> None:
    deps, router = _wire_pipeline()
    a, b = _pair_of_adapters()

    for ein in a.iter_items(tenant_id="ten_A"):
        process_one(deps, ein)
    for ein in b.iter_items(tenant_id="ten_B"):
        process_one(deps, ein)

    # Each tenant must have exactly one admission call carrying ITS tenant_id.
    tenants_seen = [body["tenant_id"] for path, body in router.calls]
    assert tenants_seen.count("ten_A") == 1
    assert tenants_seen.count("ten_B") == 1
    # No call should carry a wrong tenant.
    for path, body in router.calls:
        assert body["convention"]["tenant_id"] in {"ten_A", "ten_B"}
        assert body["tenant_id"] == body["convention"]["tenant_id"], \
            "tenant_id must match in outer + nested fields"


def test_distiller_admission_payload_carries_no_cross_tenant_data() -> None:
    deps, router = _wire_pipeline()
    a, b = _pair_of_adapters()
    for ein in a.iter_items(tenant_id="ten_A"):
        process_one(deps, ein)
    for ein in b.iter_items(tenant_id="ten_B"):
        process_one(deps, ein)
    for path, body in router.calls:
        body_str = json.dumps(body)
        other = "ten_B" if body["tenant_id"] == "ten_A" else "ten_A"
        assert other not in body_str, f"cross-tenant leak in payload for {body['tenant_id']}"
