"""LLM-as-judge filter — defense against prompt injection in memory writes.

Two-stage defense, run sequentially on every candidate:

  1. Deterministic pre-filter — cheap, keyword + structural scan.
     Mirrors the gateway's filter (so distiller-time rejection and
     gateway-time rejection produce identical reasons).
  2. LLM judge — Haiku-4.5 second pass. Catches the long-tail that
     the deterministic filter misses.

Defense in depth: every write goes through the deterministic filter
in this package AND through the gateway's deterministic filter when
the admission API is called. Two layers of the same logic; the model-
routed judge adds a third.
"""

from .deterministic import deterministic_verdict
from .llm_judge import judge as llm_judge
from .adversarial_corpus import ADVERSARIAL_CORPUS, HONEST_CORPUS

__all__ = [
    "deterministic_verdict",
    "llm_judge",
    "ADVERSARIAL_CORPUS",
    "HONEST_CORPUS",
]
