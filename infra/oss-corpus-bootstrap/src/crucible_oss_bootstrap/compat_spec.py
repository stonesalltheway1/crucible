"""Compatibility shim for memory-spec types used by the bootstrap.

Re-exports the editable install when available; otherwise provides a
minimal fallback for offline / lightweight dev runs.
"""

from __future__ import annotations

import sys
from pathlib import Path

# Make sure the editable memory-spec install is on sys.path even when
# the consumer didn't install the optional extra. This is the most
# common dev setup.
_THIS = Path(__file__).resolve().parent
_MEMORY_SPEC = _THIS.parent.parent.parent.parent / "libs" / "memory-spec" / "py"
if _MEMORY_SPEC.exists() and str(_MEMORY_SPEC) not in sys.path:
    sys.path.insert(0, str(_MEMORY_SPEC))

from crucible_memory_spec import (  # noqa: E402
    BundleLicense,
    BundleStats,
    Convention,
    ConventionCategory,
    ConventionStatus,
    MemoryLayer,
    PerStackBundle,
    ScopeFilter,
    Stack,
)

__all__ = [
    "BundleLicense",
    "BundleStats",
    "Convention",
    "ConventionCategory",
    "ConventionStatus",
    "MemoryLayer",
    "PerStackBundle",
    "ScopeFilter",
    "Stack",
]
