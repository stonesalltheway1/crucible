"""Adapter contract.

Each adapter yields ExtractionInput items the distiller pool consumes.
The interface is intentionally synchronous-iterable so backpressure
flows naturally; the async wrapper in the queue/consumer wraps the
iterator with asyncio.to_thread for fan-out.
"""

from __future__ import annotations

from typing import Iterable, Protocol

from ..types import ExtractionInput


class AdapterError(RuntimeError):
    """Adapter-level error (auth, parsing, etc.). Distillers catch and
    retry with exponential backoff."""


class RateLimitedError(AdapterError):
    """Upstream rate limit hit. Distillers back off according to the
    Retry-After header surfaced via ``retry_after_seconds``."""

    def __init__(self, msg: str, retry_after_seconds: float = 60.0) -> None:
        super().__init__(msg)
        self.retry_after_seconds = retry_after_seconds


class Adapter(Protocol):
    """An adapter produces a stream of ExtractionInput items for one
    tenant + channel + cursor position."""

    def iter_items(self, *, tenant_id: str, cursor: str = "") -> Iterable[ExtractionInput]:
        ...

    def name(self) -> str:
        ...
