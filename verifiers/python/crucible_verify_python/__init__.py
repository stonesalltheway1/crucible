"""Crucible Phase-4 per-language verifier runner — Python edition.

The package exposes a single CLI entrypoint (``crucible-verify-python``)
that the Go verifier daemon spawns once per (tier, language) cell. The CLI
reads a ``VerificationRequest`` from stdin and writes a ``TestReport`` to
stdout, framed by the ``CRUCIBLE-TESTREPORT`` delimiter.

The TestReport schema is canonical in Go (``apps/verifier/pkg/testreport``);
this package keeps a field-for-field mirror in :mod:`.schema`.
"""

from __future__ import annotations

__all__ = [
    "REPORTER_ID",
    "REPORTER_VERSION",
    "REPORT_DELIMITER",
    "SCHEMA_VERSION",
    "__version__",
]

__version__ = "0.1.0"

# Wire-protocol constants — match the Go reader in apps/verifier/internal/dispatcher.
SCHEMA_VERSION = "1"
REPORT_DELIMITER = "===CRUCIBLE-TESTREPORT==="
REPORTER_ID = "crucible-verify-python"
REPORTER_VERSION = __version__
