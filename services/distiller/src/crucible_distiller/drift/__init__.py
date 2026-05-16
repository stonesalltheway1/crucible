"""Convention drift detector — 30-day rolling pos/neg ratio.

When a convention's 30-day positive-to-negative reinforcement ratio
drops below threshold (default 1.5), the detector emits a
ConventionDrift event. The web console surfaces this to the customer
as "your rule X is aging; suggested action: ..." per
docs/01-architecture/memory-layer.md §"Convention drift detection".
"""

from .detector import detect, DriftInputs

__all__ = ["detect", "DriftInputs"]
