"""End-to-end distillation pipeline.

  Adapter ──▶ extractor ──▶ deterministic-judge ──▶ LLM-judge ──▶
    cross-source aggregation ──▶ Platt confidence ──▶ A-MAC admission
    score ──▶ admission HTTP call ──▶ distiller_runs audit row

The pipeline is synchronous + testable; the async queue consumer in
``queue/consumer.py`` wraps `process_one` per work unit.
"""

from __future__ import annotations

import logging
from dataclasses import dataclass, field
from datetime import datetime, timezone
from typing import Iterable

from .admission.amac import AdmissionInput, admit_score
from .admission.client import AdmissionClient, AdmissionResult
from .confidence.cross_source import AgreementInputs, agreement
from .confidence.platt import platt_scale
from .extractor.mem0_hierarchical import LLMClient, deterministic_extract, extract
from .judge.deterministic import deterministic_verdict
from .judge.llm_judge import JudgeLLM, judge
from .types import (
    AdmissionDecision,
    ConventionCandidate,
    DistillerConfig,
    ExtractionInput,
    JudgeVerdict,
    SourceChannel,
)

logger = logging.getLogger(__name__)


@dataclass
class PipelineDeps:
    """Wires every dependency. Production wires real clients; tests use
    fakes (extractor.FakeLLM, judge.FakeJudge, admission.FakeRouter)."""

    extractor_client: LLMClient | None
    judge_client: JudgeLLM | None
    admission: AdmissionClient
    config: DistillerConfig = field(default_factory=DistillerConfig)


@dataclass
class StageOutcome:
    """The per-input result of running the pipeline once."""

    input: ExtractionInput
    candidates_before_judge: int = 0
    candidates_after_judge: int = 0
    admitted: list[AdmissionResult] = field(default_factory=list)
    quarantined: list[JudgeVerdict] = field(default_factory=list)
    rejected: list[str] = field(default_factory=list)
    null_reason: str = ""


def process_one(deps: PipelineDeps, ein: ExtractionInput) -> StageOutcome:
    """Run the full pipeline against one ExtractionInput.

    The function is deliberately synchronous + side-effect-free except
    for the admission HTTP call. Tests can run it without setting up
    asyncio.
    """
    out = StageOutcome(input=ein)

    # 1. Extraction
    candidates: list[ConventionCandidate]
    if ein.source_channel == SourceChannel.LINT_CONFIG or deps.extractor_client is None:
        candidates = deterministic_extract(
            ein.raw_text, repo=ein.repo, tenant_id=ein.tenant_id, source=ein.source
        )
    else:
        res = extract(deps.extractor_client, ein)
        candidates = res.candidates
        out.null_reason = res.null_extraction_reason

    out.candidates_before_judge = len(candidates)
    if not candidates:
        return out

    # 2. Two-stage judge — deterministic, then LLM (if wired).
    surviving: list[ConventionCandidate] = []
    for c in candidates:
        v = deterministic_verdict(c)
        if v.quarantine:
            out.quarantined.append(v)
            continue
        if deps.judge_client is not None:
            v = judge(deps.judge_client, c)
            if v.quarantine:
                out.quarantined.append(v)
                continue
        c.judge_score = v.score
        c.judge_rationale = v.rationale
        surviving.append(c)

    out.candidates_after_judge = len(surviving)

    # 3. Confidence — Platt-scaled cross-source agreement.
    for c in surviving:
        # Without external aggregation, this single observation has
        # distinct_repos=1; cross-source agreement is therefore close
        # to 1/log(1+1). The cross-tenant aggregator (offline batch)
        # raises this over time.
        raw = agreement(AgreementInputs(
            distinct_repos=1,
            distinct_authors=1,
            corroborating_evidence_count=1,
            contradicting_evidence_count=0,
            sources_examined=1,
        ))
        conf = platt_scale(raw)
        c.cross_source_agreement = raw

        # 4. A-MAC + threshold
        score, label = admit_score(AdmissionInput(
            category=c.category,
            confidence=conf,
            judge_score=c.judge_score,
            source_channel=c.source_channel,
            last_evidence_at=c.extracted_at,
        ))
        if label == "rejected":
            out.rejected.append(c.id)
            continue

        # 5. Admission HTTP call (memory-router runs its own filter again).
        result = deps.admission.admit(c, confidence=conf, judge_score=c.judge_score)
        out.admitted.append(result)

    return out


def process_many(deps: PipelineDeps, items: Iterable[ExtractionInput]) -> list[StageOutcome]:
    """Convenience iterator. Production uses the async queue consumer."""
    return [process_one(deps, ein) for ein in items]
