"""Memory-spec invariants — admission gates, layer ordering, license safety."""

from __future__ import annotations

from datetime import datetime, timezone

import pytest

from crucible_memory_spec import (
    ALL_CATEGORIES,
    ALL_STACKS,
    BundleLicense,
    Convention,
    ConventionCategory,
    ConventionStatus,
    InvalidConventionError,
    LicenseUnsafeBundleError,
    MemoryLayer,
    PerStackBundle,
    ScopeFilter,
    Stack,
    valid_category,
)


def _valid_convention() -> Convention:
    now = datetime.now(timezone.utc)
    return Convention(
        id="conv_01HQABCDEFGHJKLMNPQRSTVWX",
        tenant_id="ten_test",
        layer=MemoryLayer.ORG_OVERRIDES,
        scope=ScopeFilter(file_glob="api/**/*.py"),
        rule_nl="Use structured slog calls; no fmt.Printf in non-test code.",
        category=ConventionCategory.LOGGING,
        status=ConventionStatus.ACTIVE,
        confidence=0.74,
        judge_score=0.91,
        valid_from=now,
        written_at=now,
    )


def test_validate_ok() -> None:
    _valid_convention().validate()


def test_taxonomy_has_twelve_buckets() -> None:
    assert len(ALL_CATEGORIES) == 12


def test_stacks_has_twelve() -> None:
    assert len(ALL_STACKS) == 12


def test_validate_rejects_invalid_category() -> None:
    c = _valid_convention()
    c.category = "Other"  # type: ignore[assignment]
    with pytest.raises(InvalidConventionError, match="invalid category"):
        c.validate()


def test_validate_rejects_out_of_range_confidence() -> None:
    c = _valid_convention()
    c.confidence = 1.5
    with pytest.raises(InvalidConventionError, match="confidence"):
        c.validate()


def test_validate_rejects_oversized_rule() -> None:
    c = _valid_convention()
    c.rule_nl = "x" * 1025
    with pytest.raises(InvalidConventionError):
        c.validate()


def test_layer_priority_bottom_up() -> None:
    assert MemoryLayer.GLOBAL_DEFAULTS.priority < MemoryLayer.ORG_OVERRIDES.priority
    assert MemoryLayer.ORG_OVERRIDES.priority < MemoryLayer.REPO_OVERRIDES.priority


def test_valid_category_helper() -> None:
    for c in ALL_CATEGORIES:
        assert valid_category(c.value)
    assert not valid_category("Other")
    assert not valid_category("")


def test_bundle_refuses_unsafe_license() -> None:
    b = PerStackBundle(
        bundle_version="1",
        stack=Stack.NEXTJS,
        generated_at=datetime.now(timezone.utc),
        license=BundleLicense(safe_for_redistribution=False, excluded_licenses=["GPL-3.0"]),
    )
    with pytest.raises(LicenseUnsafeBundleError):
        b.validate()


def test_bundle_requires_global_layer() -> None:
    c = _valid_convention()
    c.layer = MemoryLayer.ORG_OVERRIDES
    b = PerStackBundle(
        bundle_version="1",
        stack=Stack.NEXTJS,
        generated_at=datetime.now(timezone.utc),
        license=BundleLicense(safe_for_redistribution=True),
        conventions=[c],
    )
    with pytest.raises(InvalidConventionError):
        b.validate()


def test_bundle_happy_path() -> None:
    c = _valid_convention()
    c.layer = MemoryLayer.GLOBAL_DEFAULTS
    c.tenant_id = "global"
    b = PerStackBundle(
        bundle_version="1",
        stack=Stack.FASTAPI,
        generated_at=datetime.now(timezone.utc),
        license=BundleLicense(safe_for_redistribution=True),
        conventions=[c],
    )
    b.validate()
    d = b.to_dict()
    assert d["stack"] == "fastapi"
    assert d["conventions"][0]["layer"] == "global_defaults"
