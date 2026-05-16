"""Diff parsing — extract the Python source/test files from a Diff payload.

The verifier daemon hands us the entire agent-authored diff as
``VerificationRequest.diff.files[*]``. Each entry is
``{"path": ..., "action": ..., "content": ...}``. We materialise the
Python-relevant subset onto disk so the tier drivers (mutmut, hypothesis,
schemathesis) can operate on a real project tree, then return the paths.
"""

from __future__ import annotations

import hashlib
import os
from collections.abc import Iterable, Sequence
from dataclasses import dataclass
from pathlib import Path


@dataclass(frozen=True, slots=True)
class FileChange:
    """One file-level entry from the agent diff.

    Matches the JSON shape of :go:`cruciblev1.FileChange`.
    """

    path: str
    action: str  # "create" | "modify" | "delete" | "rename" — Go enum string
    content: str = ""
    content_sha256: str = ""

    @classmethod
    def from_dict(cls, raw: dict[str, object]) -> FileChange:
        return cls(
            path=str(raw.get("path", "")),
            action=str(raw.get("action", "")),
            content=str(raw.get("content", "")),
            content_sha256=str(raw.get("content_sha256", "")),
        )

    @property
    def is_python(self) -> bool:
        return self.path.endswith(".py")

    @property
    def is_python_test(self) -> bool:
        base = os.path.basename(self.path)
        return self.is_python and (base.startswith("test_") or base.endswith("_test.py"))

    @property
    def is_python_source(self) -> bool:
        return self.is_python and not self.is_python_test

    @property
    def is_openapi(self) -> bool:
        lower = self.path.lower()
        return lower.endswith((".yaml", ".yml", ".json")) and (
            "openapi" in lower or "swagger" in lower
        )


def parse_diff_files(raw_files: Iterable[dict[str, object]]) -> list[FileChange]:
    """Parse the ``diff.files[]`` array out of the request payload."""
    return [FileChange.from_dict(f) for f in raw_files if isinstance(f, dict)]


def materialise(files: Sequence[FileChange], root: Path) -> list[Path]:
    """Write each create/modify file into ``root`` and return its absolute path.

    Deleted files are *not* materialised. We assume the dispatcher has
    given us a fresh empty directory.
    """
    out: list[Path] = []
    for fc in files:
        if fc.action == "delete":
            continue
        dest = root / fc.path
        dest.parent.mkdir(parents=True, exist_ok=True)
        dest.write_text(fc.content, encoding="utf-8")
        out.append(dest)
    return out


def hash_diff(files: Sequence[FileChange]) -> str:
    """Stable, order-independent SHA-256 of the agent diff.

    Hashes ``path + "\\0" + content`` for each file in sorted-path order
    and joins them with "\\0". The Go side hashes the protobuf-encoded
    Diff, so this value is informational (it goes into TestReport.diff_hash
    so logs can correlate) — it does not need to equal Go's hash.
    """
    h = hashlib.sha256()
    for fc in sorted(files, key=lambda f: f.path):
        h.update(fc.path.encode("utf-8"))
        h.update(b"\x00")
        h.update(fc.content.encode("utf-8"))
        h.update(b"\x00")
    return h.hexdigest()
