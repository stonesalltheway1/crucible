"""memory-router admission API client.

POSTs validated, judge-passed candidates to the memory-router's
/v1/memory/admit_convention endpoint. The router runs its own
deterministic judge as a second line of defence; we send candidates
that ALSO pass our local judge so the request is rejected only when
both layers disagree.
"""

from __future__ import annotations

import json
from dataclasses import dataclass
from datetime import datetime, timezone
from typing import Any, Optional, Protocol

from ..types import ConventionCandidate, SourceRef


@dataclass
class AdmissionResult:
    admitted: bool
    convention_id: str = ""
    quarantined: bool = False
    quarantine_reason: str = ""
    injection_category: str = ""
    judge_score: float = 0.0


class HTTPClient(Protocol):
    """Minimal HTTP surface; production uses httpx, tests use the fake."""

    def post_json(self, path: str, body: dict[str, Any]) -> tuple[int, dict[str, Any]]:
        ...


class AdmissionClient:
    """Per-distiller admission HTTP client."""

    def __init__(self, http: HTTPClient, base_path: str = "/v1/memory/admit_convention") -> None:
        self.http = http
        self.base_path = base_path

    def admit(self, candidate: ConventionCandidate, *, confidence: float, judge_score: float, layer: str = "org_overrides") -> AdmissionResult:
        body = self._build_body(candidate, confidence=confidence, judge_score=judge_score, layer=layer)
        status, resp = self.http.post_json(self.base_path, body)
        if status // 100 != 2:
            return AdmissionResult(
                admitted=False,
                quarantined=True,
                quarantine_reason=f"router HTTP {status}: {resp.get('error', {}).get('message', '')}",
            )
        return AdmissionResult(
            admitted=bool(resp.get("admitted", False)),
            convention_id=str(resp.get("convention_id", "")),
            quarantined=bool(resp.get("quarantined", False)),
            quarantine_reason=str(resp.get("quarantine_reason", "")),
            injection_category=str(resp.get("injection_category", "")),
            judge_score=float(resp.get("judge_score", judge_score)),
        )

    @staticmethod
    def _build_body(c: ConventionCandidate, *, confidence: float, judge_score: float, layer: str) -> dict[str, Any]:
        now = datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%S.%fZ")
        return {
            "tenant_id": c.tenant_id,
            "force_layer": layer,
            "source": _source_ref_to_dict(c.source_evidence[0]) if c.source_evidence else None,
            "convention": {
                "id": c.id.replace("cand_", "conv_", 1),
                "tenant_id": c.tenant_id,
                "scope": {
                    "repo": c.scope.repo,
                    "file_glob": c.scope.file_glob,
                    "category": c.scope.category,
                },
                "rule_nl": c.rule_nl,
                "category": c.category.value if hasattr(c.category, "value") else c.category,
                "status": "active",
                "confidence": float(confidence),
                "judge_score": float(judge_score),
                "source_evidence": [_source_ref_to_dict(s) for s in c.source_evidence if s],
                "valid_from": now,
                "written_at": now,
            },
        }


def _source_ref_to_dict(s: Optional[SourceRef]) -> dict[str, Any]:
    if s is None:
        return {}
    out: dict[str, Any] = {"kind": s.kind}
    for k in ("pr", "comment_id", "id", "service", "path", "commit", "task_id", "step_id"):
        v = getattr(s, k, None)
        if v not in (None, "", 0):
            out[k] = v
    return out


# ─── In-memory fake for unit tests ─────────────────────────────────────────


class FakeRouter:
    """Records calls and lets the test program admission responses."""

    def __init__(self) -> None:
        self.calls: list[tuple[str, dict[str, Any]]] = []
        self.response: dict[str, Any] = {"admitted": True, "convention_id": "conv_fake", "judge_score": 0.9}
        self.status: int = 200

    def post_json(self, path: str, body: dict[str, Any]) -> tuple[int, dict[str, Any]]:
        self.calls.append((path, json.loads(json.dumps(body))))
        return self.status, self.response
