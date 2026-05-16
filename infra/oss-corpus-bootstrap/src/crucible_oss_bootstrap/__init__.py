"""Crucible OSS-corpus bootstrap — Phase 5 cold-start pipeline.

Generates per-stack default convention bundles by mining license-clean
OSS sources (style guides, lint configs, AGENTS.md, ADRs, PR comments)
and running them through the same extractor + judge pipeline the
distiller uses at runtime.
"""

from .license_filter import license_safe, ALLOWED_LICENSES, BLOCKED_LICENSES
from .pipeline import BootstrapPipeline, build_bundle
from .seeds import TIER_A_STYLE_GUIDES, stacks_with_seeds

__all__ = [
    "license_safe",
    "ALLOWED_LICENSES",
    "BLOCKED_LICENSES",
    "BootstrapPipeline",
    "build_bundle",
    "TIER_A_STYLE_GUIDES",
    "stacks_with_seeds",
]
