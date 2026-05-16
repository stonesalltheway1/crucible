"""Field-for-field Python mirror of ``apps/verifier/pkg/testreport``.

Every JSON tag here must match the Go canonical exactly. The roundtrip
test (``tests/test_schema_roundtrip.py``) compares against a sample emitted
by the Go canonical encoder; if it drifts, both sides must move together.

We use ``dataclasses`` (not pydantic) to keep the dependency footprint
small inside the verifier sandbox and to keep ``mypy --strict`` happy.
"""

from __future__ import annotations

import json
from dataclasses import asdict, dataclass, field
from datetime import UTC, datetime
from enum import StrEnum
from typing import Any, Literal

# --- Enumerations --------------------------------------------------------


class Language(StrEnum):
    PYTHON = "python"
    TYPESCRIPT = "typescript"
    RUST = "rust"
    GO = "go"
    JAVA = "java"
    SWIFT = "swift"
    POLYGLOT = "polyglot"


class Tier(StrEnum):
    MUTATION = "tier_0_mutation"
    PBT = "tier_1_pbt"
    CONTRACT = "tier_2_contract"
    PROOF = "tier_3_proof"
    HONEST_CI = "tier_4_honest_ci"


class Verdict(StrEnum):
    PASSED = "passed"
    FAILED = "failed"
    TIMED_OUT = "timed_out"
    TOOL_UNAVAILABLE = "tool_unavailable"
    SKIPPED = "skipped"


# Helper aliases for findings — keep flexible so the Go side can evolve.
Category = Literal[
    "mutation_survived",
    "property_failed",
    "fuzz_crash",
    "contract_violation",
    "proof_obligation_unmet",
    "honest_ci_mismatch",
    "tool_error",
]
Severity = Literal["info", "warn", "error"]


# --- Per-tier stats ------------------------------------------------------


@dataclass(slots=True)
class SurvivedMutant:
    file: str
    line: int
    mutator: str
    original: str = ""
    replacement: str = ""


@dataclass(slots=True)
class MutationStats:
    killed: int = 0
    survived: int = 0
    not_covered: int = 0
    timeout: int = 0
    total: int = 0
    score: float = 0.0
    threshold: float = 0.85
    diff_scoped: bool = True
    mutated_files: list[str] = field(default_factory=list)
    survived_summary: list[SurvivedMutant] = field(default_factory=list)


@dataclass(slots=True)
class Counterexample:
    property: str
    shrunk: str
    seed: str = ""
    stack_hint: str = ""


@dataclass(slots=True)
class PBTStats:
    iterations: int = 0
    iterations_min: int = 10_000
    properties: list[str] = field(default_factory=list)
    counterexamples: list[Counterexample] = field(default_factory=list)
    fuzz_corpus_size: int = 0
    fuzz_new_seeds: int = 0
    fuzz_crashes: int = 0


@dataclass(slots=True)
class ContractViolation:
    endpoint: str
    method: str
    check: str
    detail: str
    reproducer: str = ""


@dataclass(slots=True)
class ContractStats:
    spec_path: str = ""
    spec_hash: str = ""
    stateful_workflows: int = 0
    checks: list[str] = field(default_factory=list)
    violations: list[ContractViolation] = field(default_factory=list)
    dst_iterations: int = 0
    dst_replay_id: str = ""
    dst_failing_schedule: str = ""


@dataclass(slots=True)
class ProofStats:
    prover: str = ""
    proof_artifact: str = ""
    obligations: int = 0
    discharged: int = 0
    timed_out: bool = False
    wall_clock_seconds: float = 0.0
    cached_partial: bool = False
    fallback_tier: str = ""
    codeowner_review_required: bool = False
    unsoundness_hints: list[str] = field(default_factory=list)


@dataclass(slots=True)
class HonestCIStats:
    builder_id: str = ""
    nix_flake_hash: str = ""
    nix_lock_hash: str = ""
    executor_rebuild_hash: str = ""
    verifier_rebuild_hash: str = ""
    bit_identical: bool = False
    slsa_level: int = 0
    in_toto_statement_hash: str = ""
    fulcio_cert_hash: str = ""
    rekor_uuid: str = ""
    witness_attestation: str = ""
    tekton_chains_ref: str = ""
    diffoscope_report: str = ""
    scrubber_audit_ok: bool = False
    scrubber_audit_entries: int = 0


@dataclass(slots=True)
class Finding:
    category: str
    severity: str
    detail: str
    file: str = ""
    line: int = 0
    suggested_fix: str = ""


# --- The report ----------------------------------------------------------


# Fields that are zero-valued (Go's omitempty equivalent) on the Go side
# and so MUST be elided from our JSON output to keep canonical hashes
# stable. These mirror the `,omitempty` JSON tags in testreport.go.
_OMITEMPTY_KEYS: frozenset[str] = frozenset(
    {
        "tool_digest",
        "reporter_version",
        "reporter_oidc_subject",
        "error",
        "findings",
        "mutation",
        "pbt",
        "contract",
        "proof",
        "honest_ci",
    }
)
_MUTATION_OMITEMPTY: frozenset[str] = frozenset(
    {"not_covered", "timeout", "mutated_files", "survived_summary"}
)
_PBT_OMITEMPTY: frozenset[str] = frozenset(
    {"properties", "counterexamples", "fuzz_corpus_size", "fuzz_new_seeds", "fuzz_crashes"}
)
_CONTRACT_OMITEMPTY: frozenset[str] = frozenset(
    {
        "spec_path",
        "spec_hash",
        "stateful_workflows",
        "checks",
        "violations",
        "dst_iterations",
        "dst_replay_id",
        "dst_failing_schedule",
    }
)
_PROOF_OMITEMPTY: frozenset[str] = frozenset(
    {
        "proof_artifact",
        "obligations",
        "discharged",
        "wall_clock_seconds",
        "cached_partial",
        "fallback_tier",
        "codeowner_review_required",
        "unsoundness_hints",
    }
)
_HONEST_OMITEMPTY: frozenset[str] = frozenset(
    {
        "nix_flake_hash",
        "nix_lock_hash",
        "in_toto_statement_hash",
        "fulcio_cert_hash",
        "rekor_uuid",
        "witness_attestation",
        "tekton_chains_ref",
        "diffoscope_report",
        "scrubber_audit_entries",
    }
)
_FINDING_OMITEMPTY: frozenset[str] = frozenset({"file", "line", "suggested_fix"})
_SURVIVED_OMITEMPTY: frozenset[str] = frozenset({"original", "replacement"})
_CONTRACT_VIOLATION_OMITEMPTY: frozenset[str] = frozenset({"reproducer"})
_COUNTEREXAMPLE_OMITEMPTY: frozenset[str] = frozenset({"seed", "stack_hint"})


def _is_zero(value: Any) -> bool:
    """Return True if value should be elided under Go's omitempty rule."""
    if value is None:
        return True
    if isinstance(value, str | list | dict) and len(value) == 0:
        return True
    if isinstance(value, bool):
        return value is False
    if isinstance(value, int | float):
        return value == 0
    return False


def _strip_omitempty(d: dict[str, Any], omit: frozenset[str]) -> dict[str, Any]:
    return {k: v for k, v in d.items() if not (k in omit and _is_zero(v))}


@dataclass(slots=True)
class TestReport:
    """Mirrors :go:`testreport.TestReport`.

    Required fields are at the top; optional/per-tier substructs follow.
    The schema version is pinned — bumps require a 90-day deprecation.
    """

    # Tell pytest not to collect this class as a test container.
    __test__ = False

    task_id: str
    tier: Tier
    language: Language = Language.PYTHON
    framework: str = ""
    verdict: Verdict = Verdict.SKIPPED
    passed: bool = False
    schema_version: str = "1"
    diff_hash: str = ""
    started_at: datetime = field(default_factory=lambda: datetime.now(tz=UTC))
    finished_at: datetime = field(default_factory=lambda: datetime.now(tz=UTC))
    duration_seconds: float = 0.0
    wall_clock_budget_seconds: float = 0.0
    mutation: MutationStats | None = None
    pbt: PBTStats | None = None
    contract: ContractStats | None = None
    proof: ProofStats | None = None
    honest_ci: HonestCIStats | None = None
    findings: list[Finding] = field(default_factory=list)
    tool_digest: str = ""
    reporter_id: str = "crucible-verify-python"
    reporter_version: str = ""
    reporter_oidc_subject: str = ""
    error: str = ""

    # --- serialisation ----------------------------------------------------

    def to_json_dict(self) -> dict[str, Any]:
        """Render the report as a plain dict whose shape matches Go's encoder.

        Field order mirrors the Go struct field order so that human-eyeball
        diffs against a Go-emitted sample line up cleanly. Hashing
        consumers must canonicalise themselves (sorted keys + no
        whitespace) — :func:`canonical_json` does so.
        """
        # Pull values explicitly to preserve Go field order.
        out: dict[str, Any] = {
            "schema_version": self.schema_version,
            "task_id": self.task_id,
            "diff_hash": self.diff_hash,
            "tier": str(self.tier),
            "language": str(self.language),
            "framework": self.framework,
            "verdict": str(self.verdict),
            "passed": self.passed,
            "started_at": _format_time(self.started_at),
            "finished_at": _format_time(self.finished_at),
            "duration_seconds": self.duration_seconds,
            "wall_clock_budget_seconds": self.wall_clock_budget_seconds,
        }
        if self.mutation is not None:
            out["mutation"] = _stats_to_dict_mutation(self.mutation)
        if self.pbt is not None:
            out["pbt"] = _stats_to_dict_pbt(self.pbt)
        if self.contract is not None:
            out["contract"] = _stats_to_dict_contract(self.contract)
        if self.proof is not None:
            out["proof"] = _stats_to_dict_proof(self.proof)
        if self.honest_ci is not None:
            out["honest_ci"] = _stats_to_dict_honest(self.honest_ci)
        if self.findings:
            out["findings"] = [_finding_to_dict(f) for f in self.findings]
        if self.tool_digest:
            out["tool_digest"] = self.tool_digest
        out["reporter_id"] = self.reporter_id
        if self.reporter_version:
            out["reporter_version"] = self.reporter_version
        if self.reporter_oidc_subject:
            out["reporter_oidc_subject"] = self.reporter_oidc_subject
        if self.error:
            out["error"] = self.error
        return out

    def to_json(self) -> str:
        """Render the report as a UTF-8 JSON string (compact)."""
        return json.dumps(self.to_json_dict(), separators=(",", ":"), ensure_ascii=False)

    def canonical_json(self) -> str:
        """Sorted-keys, compact JSON suitable for hashing/attestation."""
        return json.dumps(
            self.to_json_dict(), separators=(",", ":"), sort_keys=True, ensure_ascii=False
        )

    # --- invariants -------------------------------------------------------

    def validate(self) -> None:
        """Enforce the same invariants as :go:`TestReport.Validate`."""
        if self.schema_version != "1":
            raise ValueError(
                f"testreport: schema_version {self.schema_version!r} != '1'"
            )
        if not self.task_id:
            raise ValueError("testreport: task_id required")
        if not str(self.tier):
            raise ValueError("testreport: tier required")
        if not str(self.language):
            raise ValueError("testreport: language required")
        if self.duration_seconds < 0:
            raise ValueError("testreport: negative duration")
        if self.tier == Tier.MUTATION and self.mutation is not None:
            if not self.mutation.diff_scoped:
                raise ValueError(
                    "testreport: mutation report not diff-scoped (Crucible mandate)"
                )
            if (
                self.mutation.total > 0
                and self.mutation.killed + self.mutation.survived > self.mutation.total
            ):
                raise ValueError("testreport: mutation killed+survived > total")
        if (
            self.tier == Tier.PBT
            and self.pbt is not None
            and self.pbt.iterations_min > 0
            and self.pbt.iterations < self.pbt.iterations_min
        ):
            raise ValueError(
                f"testreport: PBT iterations {self.pbt.iterations} "
                f"< required {self.pbt.iterations_min}"
            )


# --- internal helpers ----------------------------------------------------


def _format_time(t: datetime) -> str:
    """Match Go's ``time.Time`` JSON encoder (RFC 3339 with ns precision).

    Go renders trailing-zero-trimmed nanos. Python only carries microsecond
    precision via ``datetime``, so the encoder emits at most 6 fractional
    digits. The dispatcher is tolerant of either format.
    """
    if t.tzinfo is None:
        t = t.replace(tzinfo=UTC)
    # isoformat() yields e.g. "2026-05-15T12:34:56.789012+00:00".
    # Go's default uses "Z" suffix for UTC; normalise for round-trip parity.
    s = t.astimezone(UTC).isoformat()
    if s.endswith("+00:00"):
        s = s[: -len("+00:00")] + "Z"
    return s


def _stats_to_dict_mutation(m: MutationStats) -> dict[str, Any]:
    d = asdict(m)
    d["survived_summary"] = [
        _strip_omitempty(asdict(s), _SURVIVED_OMITEMPTY) for s in m.survived_summary
    ]
    return _strip_omitempty(d, _MUTATION_OMITEMPTY)


def _stats_to_dict_pbt(p: PBTStats) -> dict[str, Any]:
    d = asdict(p)
    d["counterexamples"] = [
        _strip_omitempty(asdict(c), _COUNTEREXAMPLE_OMITEMPTY) for c in p.counterexamples
    ]
    return _strip_omitempty(d, _PBT_OMITEMPTY)


def _stats_to_dict_contract(c: ContractStats) -> dict[str, Any]:
    d = asdict(c)
    d["violations"] = [
        _strip_omitempty(asdict(v), _CONTRACT_VIOLATION_OMITEMPTY) for v in c.violations
    ]
    return _strip_omitempty(d, _CONTRACT_OMITEMPTY)


def _stats_to_dict_proof(p: ProofStats) -> dict[str, Any]:
    return _strip_omitempty(asdict(p), _PROOF_OMITEMPTY)


def _stats_to_dict_honest(h: HonestCIStats) -> dict[str, Any]:
    return _strip_omitempty(asdict(h), _HONEST_OMITEMPTY)


def _finding_to_dict(f: Finding) -> dict[str, Any]:
    return _strip_omitempty(asdict(f), _FINDING_OMITEMPTY)


# Re-exports for callers that want to introspect the omitempty contracts
# without touching the underscored helpers.
__all__ = [
    "ContractStats",
    "ContractViolation",
    "Counterexample",
    "Finding",
    "HonestCIStats",
    "Language",
    "MutationStats",
    "PBTStats",
    "ProofStats",
    "SurvivedMutant",
    "TestReport",
    "Tier",
    "Verdict",
]
