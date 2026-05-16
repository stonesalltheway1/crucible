"""Drift detector tests — sliding-window contradictory examples."""

from __future__ import annotations

from crucible_distiller.drift.detector import DriftInputs, detect


def test_no_drift_when_ratio_above_threshold() -> None:
    out = detect(DriftInputs("conv_1", "ten_a", positives_30d=10, negatives_30d=2))
    assert out is None


def test_drift_fires_below_threshold() -> None:
    out = detect(DriftInputs("conv_1", "ten_a", positives_30d=5, negatives_30d=5))
    assert out is not None
    assert out.suggested_action in {"demote", "supersede", "archive"}


def test_archive_when_ratio_below_half() -> None:
    out = detect(DriftInputs("conv_1", "ten_a", positives_30d=1, negatives_30d=10))
    assert out is not None
    assert out.suggested_action == "archive"


def test_supersede_in_middle_band() -> None:
    out = detect(DriftInputs("conv_1", "ten_a", positives_30d=4, negatives_30d=6))
    assert out is not None
    assert out.suggested_action == "supersede"


def test_demote_just_below_threshold() -> None:
    out = detect(DriftInputs("conv_1", "ten_a", positives_30d=9, negatives_30d=8))
    assert out is not None
    assert out.suggested_action == "demote"


def test_insufficient_data_returns_none() -> None:
    out = detect(DriftInputs("conv_1", "ten_a", positives_30d=1, negatives_30d=2))
    assert out is None  # below the 5-total minimum
