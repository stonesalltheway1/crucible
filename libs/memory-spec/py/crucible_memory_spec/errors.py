"""Domain errors raised by memory-spec validators and the distiller."""

from __future__ import annotations


class InvalidConventionError(ValueError):
    """A Convention failed structural / taxonomy validation.

    Raised at admission time when an extracted candidate doesn't conform
    to the 12-category taxonomy or the 0..1 confidence range.
    """


class LicenseUnsafeBundleError(RuntimeError):
    """A PerStackBundle would have included GPL/AGPL/SSPL/BUSL inputs.

    The bootstrap pipeline raises this to refuse persisting a bundle
    whose ``license.safe_for_redistribution`` is False.
    """


class JudgeQuarantineError(RuntimeError):
    """The LLM-as-judge filter quarantined a memory write.

    Carries the quarantine reason so the distiller can persist it to the
    rejected-writes log without re-running the judge.
    """

    def __init__(self, reason: str, *, injection_category: str = "") -> None:
        super().__init__(reason)
        self.reason = reason
        self.injection_category = injection_category
