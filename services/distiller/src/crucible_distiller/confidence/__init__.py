"""Cross-source agreement + Platt-scaled confidence."""

from .cross_source import agreement, AgreementInputs
from .platt import platt_scale

__all__ = ["agreement", "AgreementInputs", "platt_scale"]
