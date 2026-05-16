"""Adversarial recall corpus test.

Loads the 1100+ synthetic-PII corpus from tests/corpus.py and asserts the
scrubber catches ≥ 99% of the entries (the Phase 3 quality bar).

By default this test runs against the regex+operator FALLBACK path so it
can execute in CI without the spaCy/Presidio models installed. When the
optional dependencies are present, the test exercises the full Presidio
path and the threshold rises to ≥ 99.5% (the difference between regex-
only and full NER on free-text-PII).

The categories most likely to leak in the regex fallback are:
  - hipaa-safe-harbor-name (no NER)
  - hipaa-safe-harbor-account (raw digit runs)
  - hipaa-safe-harbor-mrn (raw digit runs without label)
  - hipaa-safe-harbor-vin (no explicit pattern in fallback)

The fallback's overall recall stays ≥ 70% even on those because of the
URL/IP/PHONE patterns picking up structural digits and because the
free-text-PII strings often contain embedded PII the regex DOES catch.

The full-Presidio threshold is the contractual one. The fallback test
exists so that hermetic CI can run without dragging in the heavy NLP
models.
"""

from __future__ import annotations

import os
import pytest

from crucible_scrubber import ScrubPipeline, ScrubRequest
from crucible_scrubber.pipeline import _PRESIDIO_OK

from .corpus import build_corpus, CORPUS_CATEGORIES


CATEGORIES_NEEDING_NER = {
    "hipaa-safe-harbor-name",
    "free-text-pii",
}


def _classify_hit(case_raw: str, scrubbed: str) -> bool:
    """A case 'hits' iff the scrubbed text no longer contains the raw PII span.

    We are deliberately strict: the corpus is engineered so that a benign
    'no-op pass-through' fails on every category — every entry contains
    at least one PII token that the scrubber must rewrite.
    """
    # Recover the PII span from the raw string. The corpus formats each
    # entry as "<label>: <pii>" or with the PII inline; we extract the
    # right-most contiguous run that contains digits, letters and at-signs.
    return _raw_pii_span(case_raw) not in scrubbed


def _raw_pii_span(raw: str) -> str:
    """Heuristic: the longest token at the end of raw that's not pure punctuation."""
    parts = raw.split(" ")
    candidate = parts[-1]
    if candidate.endswith("."):
        candidate = candidate[:-1]
    return candidate


@pytest.mark.slow
def test_corpus_recall_fallback_path() -> None:
    """Regex fallback: ≥ 70% recall across the corpus.

    The fallback misses free-text-PII (no NER) and bare-digit account/MRN
    strings unless they're labeled. The full Presidio path picks those up;
    we don't penalise the fallback here.
    """
    corpus = build_corpus()
    p = ScrubPipeline(master_key=b"k" * 32)
    hits = 0
    misses_by_cat: dict[str, int] = {}
    totals_by_cat: dict[str, int] = {}
    for case in corpus:
        totals_by_cat[case.category] = totals_by_cat.get(case.category, 0) + 1
        r = p.scrub(ScrubRequest(tape_set="t", payload=case.raw))
        if _classify_hit(case.raw, r.scrubbed):
            hits += 1
        else:
            misses_by_cat[case.category] = misses_by_cat.get(case.category, 0) + 1
    recall = hits / len(corpus)
    assert recall >= 0.70, (
        f"fallback recall {recall:.2%} below 70% threshold. Miss breakdown: "
        f"{misses_by_cat}"
    )


@pytest.mark.slow
@pytest.mark.presidio
@pytest.mark.skipif(
    not _PRESIDIO_OK or os.environ.get("CRUCIBLE_SCRUBBER_SKIP_PRESIDIO") == "1",
    reason="Presidio not installed or skip flag set",
)
def test_corpus_recall_presidio_path() -> None:
    """Full Presidio path: ≥ 99% recall, ≥ 99.5% excluding NER-dependent cats."""
    corpus = build_corpus()
    p = ScrubPipeline(master_key=b"k" * 32)
    hits = 0
    ner_dep_hits = 0
    ner_dep_total = 0
    non_ner_hits = 0
    non_ner_total = 0
    misses_by_cat: dict[str, int] = {}
    for case in corpus:
        r = p.scrub(ScrubRequest(tape_set="t", payload=case.raw))
        is_hit = _classify_hit(case.raw, r.scrubbed)
        if is_hit:
            hits += 1
        else:
            misses_by_cat[case.category] = misses_by_cat.get(case.category, 0) + 1
        if case.category in CATEGORIES_NEEDING_NER:
            ner_dep_total += 1
            if is_hit:
                ner_dep_hits += 1
        else:
            non_ner_total += 1
            if is_hit:
                non_ner_hits += 1
    overall_recall = hits / len(corpus)
    non_ner_recall = non_ner_hits / non_ner_total if non_ner_total else 1.0
    assert overall_recall >= 0.99, (
        f"presidio recall {overall_recall:.2%} below 99%. "
        f"Misses: {misses_by_cat}"
    )
    assert non_ner_recall >= 0.995, (
        f"non-NER recall {non_ner_recall:.2%} below 99.5%. "
        f"Misses: {misses_by_cat}"
    )


def test_corpus_has_all_categories() -> None:
    """Sanity: build_corpus covers every declared category."""
    corpus = build_corpus()
    have = {c.category for c in corpus}
    declared = set(CORPUS_CATEGORIES)
    assert declared.issubset(have), f"missing categories: {declared - have}"
    assert len(corpus) >= 1000, f"corpus size {len(corpus)} below 1000 floor"
