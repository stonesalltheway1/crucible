"""Cross-source agreement scoring.

The raw confidence formula from docs/06-research/memory-bootstrap.md:

  confidence = log(distinct_repos_agreeing + 1) / log(repos_examined_in_stack + 1)

Adjusted for the distiller (per-tenant, no fixed "repos_examined" — we
use a rolling window of recent PRs as the denominator).
"""

from __future__ import annotations

import math
from dataclasses import dataclass


@dataclass(frozen=True)
class AgreementInputs:
    """Aggregated cross-source signals for one anonymized rule key."""

    distinct_repos: int = 0
    distinct_authors: int = 0
    corroborating_evidence_count: int = 0
    contradicting_evidence_count: int = 0
    sources_examined: int = 1


def agreement(inputs: AgreementInputs) -> float:
    """Return the raw cross-source agreement score (0..1, monotone).

    The formula intentionally penalises tiny corpora (a rule seen in
    only one PR gets a low score) and disagreement (contradicting
    evidence pulls the score down toward 0.5).
    """
    distinct = max(inputs.distinct_repos, 1)
    denom = max(inputs.sources_examined, distinct)
    raw = math.log(distinct + 1) / math.log(denom + 1)

    # Authorship diversity bumps confidence — three different reviewers
    # citing a rule is stronger than three PRs from the same author.
    if inputs.distinct_authors >= 2:
        raw *= 1 + min(0.2, 0.05 * (inputs.distinct_authors - 1))

    # Contradiction penalty.
    if inputs.contradicting_evidence_count > 0:
        ratio = inputs.corroborating_evidence_count / max(inputs.contradicting_evidence_count, 1)
        raw *= min(1.0, ratio / (ratio + 1))

    if raw < 0:
        raw = 0
    if raw > 1:
        raw = 1
    return raw
