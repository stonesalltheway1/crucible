"""Platt scaling — calibrate raw agreement scores to true probabilities.

  P(true_positive | raw) = 1 / (1 + exp(A * raw + B))

The (A, B) coefficients are fitted offline against a labelled
evaluation set; Phase-5 ships sensible defaults that match the
distribution of the OSS-corpus-bootstrap output. Customers using the
distiller against their own PR history can refine the coefficients
via `crucible-distiller calibrate` (CLI).
"""

from __future__ import annotations

import math
from dataclasses import dataclass


@dataclass(frozen=True)
class PlattCoefficients:
    a: float = -8.0   # slope
    b: float = 4.0    # intercept

    @classmethod
    def default(cls) -> "PlattCoefficients":
        return cls()


def platt_scale(raw: float, coeffs: PlattCoefficients | None = None) -> float:
    """Sigmoid calibration. Returns a probability in [0, 1]."""
    if coeffs is None:
        coeffs = PlattCoefficients.default()
    # Clip raw to a sane range to avoid overflow.
    if raw < 0:
        raw = 0
    if raw > 1:
        raw = 1
    z = coeffs.a * raw + coeffs.b
    return 1.0 / (1.0 + math.exp(z))
