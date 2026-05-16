"""Admission stage — A-MAC scoring + memory-router HTTP client."""

from .amac import admit_score, AdmissionInput
from .client import AdmissionClient

__all__ = ["admit_score", "AdmissionInput", "AdmissionClient"]
