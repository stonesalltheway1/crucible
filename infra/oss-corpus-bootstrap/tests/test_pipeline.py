"""OSS-corpus bootstrap pipeline tests."""

from __future__ import annotations

import json
from pathlib import Path

import pytest

from crucible_oss_bootstrap.license_filter import license_safe
from crucible_oss_bootstrap.pipeline import build_all, build_bundle, write_all


def test_license_filter_allowlist() -> None:
    for ok in ("MIT", "Apache-2.0", "BSD-3-Clause", "MPL-2.0", "ISC"):
        assert license_safe(ok), ok


def test_license_filter_blocks_copyleft() -> None:
    for bad in ("GPL-3.0", "AGPL-3.0", "SSPL-1.0", "BUSL-1.1"):
        assert not license_safe(bad), bad


def test_license_filter_refuses_unknown() -> None:
    assert not license_safe(None)
    assert not license_safe("")
    assert not license_safe("Proprietary")


def test_build_bundle_emits_validated_bundle() -> None:
    b = build_bundle("nextjs")
    b.validate()
    assert b.bundle_version == "1"
    assert len(b.conventions) == 12, "12 taxonomy buckets per stack"
    # Every convention must be layer=global_defaults.
    for c in b.conventions:
        assert c.layer.value == "global_defaults"
        assert c.tenant_id == "global"
        assert c.confidence >= 0.4, c
        assert c.judge_score >= 0.5, c


def test_build_bundle_unknown_stack_raises() -> None:
    with pytest.raises(ValueError, match="unknown stack"):
        build_bundle("not-a-stack")


def test_build_all_covers_twelve_stacks() -> None:
    bs = build_all()
    assert len(bs) == 12


def test_build_all_total_active_rules() -> None:
    bs = build_all()
    total = sum(b.stats.active_rules for b in bs)
    # 12 stacks × 12 categories = 144 base rules in the seed
    # scaffolding. The full ~400-rule fresh-customer experience adds
    # Tier-B / Tier-C corpus rules in a follow-up; Phase 5 ships the
    # scaffold here.
    assert total >= 144, f"expected ≥144 base rules; got {total}"


def test_write_all_round_trips(tmp_path: Path) -> None:
    paths = write_all(str(tmp_path))
    assert len(paths) == 12
    # Spot-check: every file is valid JSON matching the bundle schema.
    for p in paths:
        with open(p, "r", encoding="utf-8") as f:
            data = json.load(f)
        assert data["bundle_version"] == "1"
        assert data["license"]["safe_for_redistribution"] is True
        assert len(data["conventions"]) == 12
        for c in data["conventions"]:
            assert c["layer"] == "global_defaults"
            assert c["category"] in {
                "Naming", "Layering", "LibraryPreferences", "TestPatterns",
                "ErrorHandling", "Logging", "MigrationPatterns", "PrCommitHygiene",
                "SecurityDefaults", "PerformanceDefaults", "Concurrency", "ApiShape",
            }


def test_bundle_excluded_licenses_blocks_redistribution() -> None:
    """The bundle.validate() guard refuses to ship when an unsafe
    license slipped through. This is the safety-net test."""
    from crucible_memory_spec import BundleLicense, PerStackBundle, Stack
    from crucible_memory_spec.errors import LicenseUnsafeBundleError
    from datetime import datetime, timezone

    b = PerStackBundle(
        bundle_version="1",
        stack=Stack.NEXTJS,
        generated_at=datetime.now(timezone.utc),
        license=BundleLicense(safe_for_redistribution=False, excluded_licenses=["GPL-3.0"]),
    )
    with pytest.raises(LicenseUnsafeBundleError):
        b.validate()
