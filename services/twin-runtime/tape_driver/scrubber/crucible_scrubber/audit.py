"""Scrub audit log.

Every scrub-rewrite produces one [AuditEntry]. The audit log is the artifact
compliance auditors look at: it must enumerate every field that was rewritten,
which scrubber fired, which operator was applied, and (for FF3-1) the domain
size and tweak length.

The audit log is content-addressed so that two scrubs of the same payload
produce identical hashes for the rewrite set (modulo deterministic
pseudonymisation keying).
"""

from __future__ import annotations

import hashlib
import json
import time
from dataclasses import dataclass, field, asdict
from typing import Any


def sha256_hex(value: str | bytes) -> str:
    """SHA-256 hex digest with a sha256: prefix.

    The audit log stores the *hash* of the original sensitive bytes, never the
    raw bytes — a compliance auditor must be able to verify that the scrubber
    fired without leaking the original PII through the audit channel itself.
    """
    if isinstance(value, str):
        value = value.encode("utf-8")
    return "sha256:" + hashlib.sha256(value).hexdigest()


@dataclass(slots=True)
class AuditEntry:
    """One rewrite event.

    Fields:
        scrubber: the recognizer/entity name that matched (e.g., "us-ssn").
        field: the JSON-path or "[inline]" for raw payloads.
        before_hash: sha256: prefixed hash of the original bytes.
        after: the post-scrub replacement bytes (safe to log).
        operator: the operator applied (REDACT, REPLACE, MASK, DETERMINISTIC, FF3).
        algorithm: for FF3 / DETERMINISTIC, the cipher/digest name. Else "".
        ff3_domain_size: the domain size used (only populated for FF3).
        tape_set: which per-tape-set key namespace was in effect.
        timestamp_ms: epoch ms when the rewrite was recorded.
    """

    scrubber: str
    field: str
    before_hash: str
    after: str
    operator: str
    algorithm: str = ""
    ff3_domain_size: int = 0
    tape_set: str = ""
    timestamp_ms: int = field(default_factory=lambda: int(time.time() * 1000))

    def to_dict(self) -> dict[str, Any]:
        return asdict(self)


@dataclass(slots=True)
class AuditLog:
    """Ordered list of audit entries for one scrub call.

    The log is append-only within a request; multiple requests get distinct
    logs. Caller is expected to persist the JSON form alongside the scrubbed
    tape entry.
    """

    entries: list[AuditEntry] = field(default_factory=list)
    tape_set: str = ""
    elapsed_ms: int = 0

    def record(
        self,
        scrubber: str,
        field: str,
        before: str,
        after: str,
        operator: str,
        algorithm: str = "",
        ff3_domain_size: int = 0,
    ) -> AuditEntry:
        entry = AuditEntry(
            scrubber=scrubber,
            field=field,
            before_hash=sha256_hex(before),
            after=after,
            operator=operator,
            algorithm=algorithm,
            ff3_domain_size=ff3_domain_size,
            tape_set=self.tape_set,
        )
        self.entries.append(entry)
        return entry

    def to_dict(self) -> dict[str, Any]:
        return {
            "tape_set": self.tape_set,
            "elapsed_ms": self.elapsed_ms,
            "rewrites": [e.to_dict() for e in self.entries],
        }

    def to_json(self) -> str:
        return json.dumps(self.to_dict(), separators=(",", ":"), sort_keys=True)

    def scrubber_counts(self) -> dict[str, int]:
        """Per-scrubber rewrite counts. Used for SLO reporting."""
        out: dict[str, int] = {}
        for e in self.entries:
            out[e.scrubber] = out.get(e.scrubber, 0) + 1
        return out
