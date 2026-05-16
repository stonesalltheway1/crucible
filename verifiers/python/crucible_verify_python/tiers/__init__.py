"""Per-tier drivers for the Python verifier.

Each module exposes a single ``run(ctx: DriverContext) -> TestReport``
function. Drivers MUST:

- never write to stdout (logs go to stderr);
- fill in the per-tier stats union (``mutation`` / ``pbt`` / etc.);
- populate ``verdict`` and ``passed`` consistently with the tier
  threshold from ``docs/01-architecture/verifier-pipeline.md``;
- attach a :class:`~crucible_verify_python.schema.Finding` for each
  substantive failure so the rubric judge can compose
  ``VerifierRejection.RejectionReasons``.

Common report fields (task_id, diff_hash, language, timing, reporter
identity) are stamped by the CLI wrapper after the driver returns.
"""

from __future__ import annotations

__all__ = [
    "tier0_mutation",
    "tier1_pbt",
    "tier2_contract",
    "tier3_proof",
    "tier4_honest_ci",
]
