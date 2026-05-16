"""HTTP service.

The Go tape_driver calls POST /scrub at capture time. Presidio is no-auth
by design; this service expects a shared bearer token (`CRUCIBLE_SCRUBBER_TOKEN`)
that the Go fronting service injects. **Never** expose this directly to the
agent or to the public internet.

Endpoints:
  GET  /healthz   - liveness probe.
  GET  /readyz    - readiness (analyzer + anonymizer + ff3 cipher all initialised).
  POST /scrub     - run the pipeline on a single payload.
  POST /scrub/batch - run the pipeline on a list of payloads (one shared tape-set).

The service is launched via the `crucible-scrubber` console script.
"""

from __future__ import annotations

import os
from typing import Any

try:
    from fastapi import FastAPI, Header, HTTPException, Request
    from pydantic import BaseModel, Field
except ImportError:  # pragma: no cover - the dev shell installs fastapi
    FastAPI = None  # type: ignore[assignment]
    BaseModel = object  # type: ignore[assignment]
    Field = lambda *args, **kwargs: None  # type: ignore[assignment]
    Header = lambda *args, **kwargs: None  # type: ignore[assignment]
    HTTPException = Exception  # type: ignore[assignment]
    Request = object  # type: ignore[assignment]

from .pipeline import ScrubPipeline, ScrubRequest
from .recognizers import TenantPatternSpec


ENV_TOKEN = "CRUCIBLE_SCRUBBER_TOKEN"


class ScrubBody(BaseModel):  # type: ignore[misc]
    tape_set: str = Field(..., description="Per-tape-set namespace.")
    payload: str = Field(..., description="Raw payload to scrub.")
    fields: list[str] = Field(default_factory=list)
    language: str = "en"
    engine: str = "default"
    content_type: str = ""
    operator_overrides: dict[str, str] = Field(default_factory=dict)
    custom_recognizers: list[dict[str, Any]] = Field(default_factory=list)


class BatchScrubBody(BaseModel):  # type: ignore[misc]
    tape_set: str
    items: list[ScrubBody]


def _custom_recognizers_from_payload(
    payload: list[dict[str, Any]]
) -> tuple[TenantPatternSpec, ...]:
    out: list[TenantPatternSpec] = []
    for raw in payload:
        out.append(
            TenantPatternSpec(
                entity=str(raw.get("entity", "TENANT_ACCOUNT_ID")),
                name=str(raw.get("name", "tenant-pattern")),
                regex=str(raw.get("regex", "")),
                score=float(raw.get("score", 0.85)),
                context=tuple(raw.get("context", [])),
            )
        )
    return tuple(out)


def _require_token(authorization: str | None) -> None:
    expected = os.environ.get(ENV_TOKEN, "")
    if not expected:
        # No token configured → service refuses all requests in prod mode.
        # The Go fronting service sets one at deploy time.
        raise HTTPException(  # type: ignore[misc]
            status_code=503,
            detail=f"{ENV_TOKEN} not set; scrubber is dev-mode-only",
        )
    if not authorization or not authorization.startswith("Bearer "):
        raise HTTPException(status_code=401, detail="missing bearer token")  # type: ignore[misc]
    presented = authorization[len("Bearer ") :].strip()
    if presented != expected:
        raise HTTPException(status_code=403, detail="bad token")  # type: ignore[misc]


def build_app(pipeline: ScrubPipeline | None = None) -> Any:
    if FastAPI is None:  # pragma: no cover
        raise RuntimeError("fastapi is not installed")

    app = FastAPI(
        title="Crucible Scrubber",
        version="2026.6.0-phase3",
        docs_url=None,
        redoc_url=None,
        openapi_url=None,
    )
    p = pipeline or ScrubPipeline.from_env()

    @app.get("/healthz")
    def healthz() -> dict[str, str]:
        return {"status": "ok"}

    @app.get("/readyz")
    def readyz() -> dict[str, Any]:
        return {
            "status": "ready",
            "version": "2026.6.0-phase3",
            "fallback_mode": p._fallback_only,
        }

    @app.post("/scrub")
    def scrub(body: ScrubBody, authorization: str | None = Header(default=None)) -> dict[str, Any]:
        _require_token(authorization)
        result = p.scrub(
            ScrubRequest(
                tape_set=body.tape_set,
                payload=body.payload,
                fields=tuple(body.fields),
                language=body.language,
                engine=body.engine,
                content_type=body.content_type,
                operator_overrides=body.operator_overrides,
                custom_recognizers=_custom_recognizers_from_payload(body.custom_recognizers),
            )
        )
        return {"scrubbed": result.scrubbed, "report": result.report}

    @app.post("/scrub/batch")
    def scrub_batch(
        body: BatchScrubBody, authorization: str | None = Header(default=None)
    ) -> dict[str, Any]:
        _require_token(authorization)
        out: list[dict[str, Any]] = []
        for item in body.items:
            r = p.scrub(
                ScrubRequest(
                    tape_set=body.tape_set,
                    payload=item.payload,
                    fields=tuple(item.fields),
                    language=item.language,
                    engine=item.engine,
                    content_type=item.content_type,
                    operator_overrides=item.operator_overrides,
                    custom_recognizers=_custom_recognizers_from_payload(item.custom_recognizers),
                )
            )
            out.append({"scrubbed": r.scrubbed, "report": r.report})
        return {"items": out}

    return app


def run() -> None:
    """Entrypoint for the console script."""
    import uvicorn  # type: ignore[import-not-found]
    host = os.environ.get("CRUCIBLE_SCRUBBER_HOST", "127.0.0.1")
    port = int(os.environ.get("CRUCIBLE_SCRUBBER_PORT", "9100"))
    uvicorn.run(build_app(), host=host, port=port, log_level="info")
