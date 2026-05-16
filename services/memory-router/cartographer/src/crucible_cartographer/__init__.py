"""Crucible cartographer — Phase 5 onboarding per-repo mining tool.

Runs ONCE per repo at customer-install time. Walks the filesystem,
parses lint configs deterministically, reads AGENTS.md / CONTRIBUTING.md /
ADRs, and emits a per-repo seed convention bundle that lands in the
memory-router's repo_overrides layer.

Sibling to the distiller; reuses the extractor + judge pipeline.
"""

from .scanner import CartographerJob, CartographerResult, scan, StackDetection

__all__ = ["CartographerJob", "CartographerResult", "scan", "StackDetection"]
