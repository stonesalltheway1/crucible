"""Crucible Agent SDK — Python binding for the agent-side ``twin.*`` runtime API.

Phase 2 ships the typed surface (Pydantic models) plus an in-memory
``StubClient`` for upstream unit tests. The gRPC transport against the
Rust runtime-server lands in Phase 2's integration tests; full wire-up
is tracked in PHASE-2-REPORT.md.

All raw secret values are kept out of the Python process by design — the
egress proxy substitutes ``$secret(name)$`` placeholders at request time.
``twin.secret.get`` returns only a typed handle.
"""
from __future__ import annotations

import abc
import dataclasses
import hashlib
import time
from datetime import datetime, timezone
from typing import Any, Iterable, Literal, Optional


@dataclasses.dataclass(frozen=True)
class WriteAttestation:
    """Returned from ``twin.fs.write``."""

    attestation_id: str
    content_sha256: str


@dataclasses.dataclass(frozen=True)
class SecretRef:
    """Opaque handle returned by ``twin.secret.get``.

    The value is NEVER carried here — it's substituted at the egress proxy.
    """

    name: str
    handle: str
    expires_at_unix: float


@dataclasses.dataclass(frozen=True)
class ShellResult:
    """Returned from ``twin.shell.exec`` for benign commands."""

    stdout: str
    stderr: str
    exit_code: int
    duration_ms: int
    signed_attestation: str


@dataclasses.dataclass(frozen=True)
class DestructiveProposal:
    """Returned from ``twin.shell.exec`` when the shim intercepts."""

    proposal_id: str
    command: str
    reason: str
    scope: str  # "twin" | "real"


@dataclasses.dataclass(frozen=True)
class SourceRef:
    """Where a memory write came from. Subset of the proto SourceRef shape."""

    kind: Literal["pr_comment", "incident", "adr", "agent_observation"]
    pr: int = 0
    comment_id: str = ""
    incident_id: str = ""
    service: str = ""
    path: str = ""
    commit: str = ""
    task_id: str = ""
    step_id: str = ""


@dataclasses.dataclass(frozen=True)
class ScopeFilter:
    repo: str = ""
    file_glob: str = ""
    category: str = ""


@dataclasses.dataclass(frozen=True)
class Memory:
    """Returned from twin.memory.recall — a single retrieved item."""

    id: str
    content: str
    importance: float
    kind: Literal["hot", "episodic", "semantic", "procedural"]
    written_at: datetime
    last_recalled: datetime


@dataclasses.dataclass(frozen=True)
class Convention:
    """Returned from twin.memory.conventions — a procedural rule."""

    id: str
    tenant_id: str
    scope: ScopeFilter
    rule_nl: str
    category: str
    status: Literal["active", "drifting", "superseded", "rejected"]
    confidence: float
    valid_from: datetime
    judge_score: float = 0.0


@dataclasses.dataclass(frozen=True)
class ComplianceReportViolation:
    convention_id: str
    rule_nl: str
    offending_file: str
    severity: Literal["info", "warn", "error"]
    offending_line: int = 0
    snippet: str = ""


@dataclasses.dataclass(frozen=True)
class ComplianceReport:
    """Returned from twin.memory.check_compliance."""

    diff_hash: str
    violations: list[ComplianceReportViolation]
    conventions_checked: int
    generated_at: datetime


@dataclasses.dataclass(frozen=True)
class SvcCallResponse:
    """Returned from ``twin.svc.call``."""

    status: int
    headers: dict[str, str]
    body: bytes
    tape_disposition: str  # value of X-Crucible-Tape


class TwinClient(abc.ABC):
    """The agent-side runtime client surface."""

    @abc.abstractmethod
    def fs_read(self, path: str) -> str:
        ...

    @abc.abstractmethod
    def fs_write(self, path: str, content: str, step_id: str = "") -> WriteAttestation:
        ...

    @abc.abstractmethod
    def shell_exec(self, cmd: str) -> ShellResult | DestructiveProposal:
        ...

    @abc.abstractmethod
    def secret_get(self, name: str) -> SecretRef:
        ...

    @abc.abstractmethod
    def svc_call(self, service: str, endpoint: str, body: bytes = b"") -> SvcCallResponse:
        ...

    @abc.abstractmethod
    def heartbeat(self) -> None:
        ...

    # ── twin.memory ──────────────────────────────────────────────────────
    @abc.abstractmethod
    def memory_recall(self, query: str, *, max_tokens: int = 7000) -> list[Memory]:
        ...

    @abc.abstractmethod
    def memory_note(self, fact: str, source: SourceRef) -> str:
        ...

    @abc.abstractmethod
    def memory_conventions(self, scope: ScopeFilter) -> list[Convention]:
        ...

    @abc.abstractmethod
    def memory_check_compliance(self, diff: dict[str, Any]) -> ComplianceReport:
        ...


_STUB_MSG = (
    "STUB: full gRPC Python client surface is wired in Phase 2 integration tests "
    "against the Rust runtime-server. Use StubClient for unit tests. See PHASE-2-REPORT.md."
)


def new_client(*, endpoint: str, task_id: str) -> TwinClient:
    """Factory. Real wire client is gRPC against ``endpoint``; for now this
    raises ``NotImplementedError`` from every method except ``heartbeat``,
    pointing the user at ``StubClient``."""
    if not task_id:
        raise ValueError("task_id required")
    return _StubError(endpoint=endpoint, task_id=task_id)


class _StubError(TwinClient):
    def __init__(self, *, endpoint: str, task_id: str):
        self.endpoint = endpoint
        self.task_id = task_id

    def fs_read(self, path: str) -> str:
        raise NotImplementedError(_STUB_MSG)

    def fs_write(self, path: str, content: str, step_id: str = "") -> WriteAttestation:
        raise NotImplementedError(_STUB_MSG)

    def shell_exec(self, cmd: str) -> ShellResult | DestructiveProposal:
        raise NotImplementedError(_STUB_MSG)

    def secret_get(self, name: str) -> SecretRef:
        raise NotImplementedError(_STUB_MSG)

    def svc_call(self, service: str, endpoint: str, body: bytes = b"") -> SvcCallResponse:
        raise NotImplementedError(_STUB_MSG)

    def heartbeat(self) -> None:
        # Heartbeat must always succeed even in stub mode so caller's
        # keepalive loop doesn't crash.
        pass

    def memory_recall(self, query: str, *, max_tokens: int = 7000) -> list[Memory]:
        raise NotImplementedError(_STUB_MSG)

    def memory_note(self, fact: str, source: SourceRef) -> str:
        raise NotImplementedError(_STUB_MSG)

    def memory_conventions(self, scope: ScopeFilter) -> list[Convention]:
        raise NotImplementedError(_STUB_MSG)

    def memory_check_compliance(self, diff: dict[str, Any]) -> ComplianceReport:
        raise NotImplementedError(_STUB_MSG)


class StubClient(TwinClient):
    """In-memory client. Records writes; deterministic for tests."""

    def __init__(self, task_id: str = ""):
        self.task_id = task_id
        self._files: dict[str, str] = {}

    def fs_read(self, path: str) -> str:
        if path not in self._files:
            raise FileNotFoundError(path)
        return self._files[path]

    def fs_write(self, path: str, content: str, step_id: str = "") -> WriteAttestation:
        self._files[path] = content
        return WriteAttestation(
            attestation_id=f"stub:{path}",
            content_sha256=hashlib.sha256(content.encode("utf-8")).hexdigest(),
        )

    def shell_exec(self, cmd: str) -> ShellResult | DestructiveProposal:
        return ShellResult(stdout=f"[stub] {cmd}", stderr="", exit_code=0, duration_ms=0, signed_attestation="")

    def secret_get(self, name: str) -> SecretRef:
        return SecretRef(name=name, handle=f"stub-handle:{name}", expires_at_unix=time.time() + 60)

    def svc_call(self, service: str, endpoint: str, body: bytes = b"") -> SvcCallResponse:
        return SvcCallResponse(
            status=200,
            headers={"X-Crucible-Tape": "hit-exact"},
            body=b"",
            tape_disposition="hit-exact",
        )

    def heartbeat(self) -> None:
        pass

    def memory_recall(self, query: str, *, max_tokens: int = 7000) -> list[Memory]:
        return []

    def memory_note(self, fact: str, source: SourceRef) -> str:
        return "mem_stub"

    def memory_conventions(self, scope: ScopeFilter) -> list[Convention]:
        return []

    def memory_check_compliance(self, diff: dict[str, Any]) -> ComplianceReport:
        return ComplianceReport(
            diff_hash="",
            violations=[],
            conventions_checked=0,
            generated_at=datetime.now(timezone.utc),
        )
