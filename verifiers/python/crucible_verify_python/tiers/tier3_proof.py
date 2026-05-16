"""Tier 3 — formal-proof dispatcher (Dafny / Lean / TLA+).

The actual prover adapters live in ``apps/verifier/internal/tier3`` on the
Go side. This Python runner exists only so the dispatcher can fan tier-3
out by language uniformly; we always emit ``tool_unavailable`` and let the
Go-side adapter take over. Emitting a well-formed report (rather than
crashing) keeps the dispatcher's tier-result folding logic uniform.
"""

from __future__ import annotations

from typing import TYPE_CHECKING

from ..schema import Finding, ProofStats, TestReport, Tier, Verdict

if TYPE_CHECKING:
    from ..cli import DriverContext

FRAMEWORK = "dispatcher-stub"


def run(ctx: DriverContext) -> TestReport:
    # The dispatcher (Go) is responsible for invoking the real prover. We
    # return a structured placeholder so the report parser doesn't choke.
    detail = (
        "tier-3 proof dispatch handled by apps/verifier/internal/tier3 "
        "Go adapter; Python runner emits placeholder."
    )
    return TestReport(
        task_id=ctx.task_id,
        tier=Tier.PROOF,
        framework=FRAMEWORK,
        verdict=Verdict.TOOL_UNAVAILABLE,
        passed=False,
        proof=ProofStats(prover="dafny", obligations=0, discharged=0, timed_out=False),
        findings=[Finding(category="tool_error", severity="info", detail=detail)],
    )
