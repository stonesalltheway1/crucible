"""Defence-in-depth audit guard against executor-reasoning leakage.

The Go side already audits the same denylist at request ingest
(see ``apps/verifier/internal/verification/verification.go``::``auditMap``).
This module re-applies the same rules inside the Python runner so a
mis-routed payload cannot reach a tool subprocess that might serialise it
back into the report.

Mirror the Go list exactly — drift between the two is a load-bearing bug.
"""

from __future__ import annotations

from collections.abc import Iterable, Mapping
from dataclasses import dataclass
from typing import Any

# Intentionally aggressive — false positives are cheaper than the
# brand-existential cost of a leaked reasoning trace (ADR-002).
REASONING_DENYLIST: tuple[str, ...] = (
    "reasoning",
    "chain_of_thought",
    "chain-of-thought",
    "cot",
    "thinking_trace",
    "thinking-trace",
    "thoughts",
    "scratchpad",
    "internal_monologue",
    "hidden_state",
    "agent_trace",
    "executor_trace",
    "trajectory",
    "plan_critique",
    "reflection",
)

# Path patterns banned inside Diff.Files[*].Path.
REASONING_PATH_PATTERNS: tuple[str, ...] = (
    ".reasoning.",
    "/reasoning/",
    ".cot.",
    "/cot/",
    "_thinking_",
    "_scratchpad_",
    "agent_trace",
    "executor_trace",
)


@dataclass(frozen=True, slots=True)
class LeakageError(Exception):
    """Raised when a payload contains a reasoning-tagged field."""

    field: str
    pattern: str

    def __str__(self) -> str:  # pragma: no cover — trivial
        return (
            f"executor-reasoning leak detected — field {self.field!r} "
            f"matched pattern {self.pattern!r} (ADR-002 invariant)"
        )


def _match_key(key: str) -> str | None:
    lowered = key.lower()
    for deny in REASONING_DENYLIST:
        if deny in lowered:
            return deny
    return None


def audit_payload(payload: Any, prefix: str = "") -> None:
    """Recursively scan ``payload`` for reasoning-tagged keys.

    Accepts arbitrary JSON-decoded values (dicts, lists, scalars). Raises
    :class:`LeakageError` on the first match — we fail closed.
    """
    if isinstance(payload, Mapping):
        # Sort to make audit errors deterministic across Python dict orders.
        for key in sorted(payload.keys(), key=str):
            full = f"{prefix}.{key}" if prefix else str(key)
            match = _match_key(str(key))
            if match is not None:
                raise LeakageError(field=full, pattern=match)
            audit_payload(payload[key], full)
        return
    if isinstance(payload, (list, tuple)):
        # str/bytes don't satisfy this branch (they're not list/tuple
        # subclasses), so we don't need to filter them out explicitly.
        for i, item in enumerate(payload):
            audit_payload(item, f"{prefix}[{i}]")
        return
    # Scalars (str/int/float/bool/None) are not audited by value — that
    # would produce too many false positives on natural English. We only
    # gate on field names, matching the Go side.


def audit_diff_paths(paths: Iterable[str]) -> None:
    """Refuse paths that obviously came from a reasoning sidecar.

    Mirrors ``isReasoningPath`` in the Go verification package.
    """
    for path in paths:
        lowered = path.lower()
        for pattern in REASONING_PATH_PATTERNS:
            if pattern in lowered:
                raise LeakageError(field=f"diff.files.{path}", pattern="path-pattern")
