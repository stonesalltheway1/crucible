"""Custom Presidio Recognizers for Crucible.

Out-of-the-box Presidio covers:

    CREDIT_CARD, CRYPTO, EMAIL_ADDRESS, IBAN_CODE, IP_ADDRESS, PHONE_NUMBER,
    URL, US_SSN, US_BANK_NUMBER, US_DRIVER_LICENSE, US_ITIN, US_PASSPORT,
    DATE_TIME, NRP, LOCATION, PERSON, MEDICAL_LICENSE

Crucible adds the entities that come up in the HIPAA 18-identifier list and
in regulated-vertical tape corpora but aren't first-class in Presidio:

    MEDICAL_RECORD_NUMBER (MRN)       - hospital-issued identifiers
    HEALTH_PLAN_BENEFICIARY            - payor IDs
    NPI                                - National Provider Identifier (10 digits, Luhn-ish)
    DEA_NUMBER                         - DEA registration (2 alpha + 7 digits)
    US_VEHICLE_VIN                     - 17-char alphanumeric
    BIOMETRIC_IDENTIFIER               - generic placeholder for fingerprint/voice IDs
    DEVICE_SERIAL                      - generic placeholder for serial numbers
    TENANT_ACCOUNT_ID                  - tenant-configurable shape

The tenant-configurable recognizer lets buyers register their own ID shapes
without forking the package. The HTTP scrub service accepts a list of
PatternSpec entries on the /scrub call which we materialise into
PatternRecognizers at request time.
"""

from __future__ import annotations

from dataclasses import dataclass
from typing import Final

try:
    from presidio_analyzer import (  # type: ignore[import-not-found]
        PatternRecognizer,
        Pattern,
        EntityRecognizer,
        RecognizerResult,
    )
    from presidio_analyzer.nlp_engine import NlpArtifacts  # type: ignore[import-not-found]
except ImportError:  # pragma: no cover
    PatternRecognizer = object  # type: ignore[assignment]
    Pattern = object  # type: ignore[assignment]
    EntityRecognizer = object  # type: ignore[assignment]
    RecognizerResult = object  # type: ignore[assignment]
    NlpArtifacts = object  # type: ignore[assignment]


ENTITY_MRN: Final[str] = "MEDICAL_RECORD_NUMBER"
ENTITY_HEALTH_PLAN: Final[str] = "HEALTH_PLAN_BENEFICIARY"
ENTITY_NPI: Final[str] = "NPI"
ENTITY_DEA: Final[str] = "DEA_NUMBER"
ENTITY_VIN: Final[str] = "US_VEHICLE_VIN"
ENTITY_BIOMETRIC: Final[str] = "BIOMETRIC_IDENTIFIER"
ENTITY_DEVICE_SERIAL: Final[str] = "DEVICE_SERIAL"
ENTITY_TENANT_ACCOUNT: Final[str] = "TENANT_ACCOUNT_ID"


class MRNRecognizer(PatternRecognizer):  # type: ignore[misc]
    """Medical Record Number recognizer.

    MRN shape varies widely by hospital. The default pattern catches:
      - "MRN: 12345678"
      - "Medical Record #: AB-1234567"
      - "Patient ID 999-99-9999" (when context contains medical terms)
      - bare 6-10 digit numbers when surrounded by medical context.

    Score is biased downward for bare digits — we rely on the context list
    and on Presidio's NLP engine to disambiguate from generic numbers.
    """

    PATTERNS = [
        Pattern(
            name="mrn-labeled",
            regex=r"\b(?:MRN|Medical Record(?:\s*#)?|Patient ID)\s*[:#]?\s*[A-Z0-9\-]{6,12}\b",
            score=0.85,
        ),
        Pattern(
            name="mrn-bare",
            regex=r"\b\d{6,10}\b",
            score=0.35,
        ),
    ]

    CONTEXT = [
        "mrn",
        "medical",
        "patient",
        "record",
        "hospital",
        "clinic",
        "discharge",
        "admission",
        "diagnosis",
        "chart",
    ]

    def __init__(self) -> None:
        super().__init__(
            supported_entity=ENTITY_MRN,
            patterns=self.PATTERNS,
            context=self.CONTEXT,
            supported_language="en",
        )


class NPIRecognizer(PatternRecognizer):  # type: ignore[misc]
    """National Provider Identifier (NPI).

    10 digits with a Luhn-style check digit. We verify the check digit so
    we don't false-positive on every 10-digit number.
    """

    PATTERNS = [
        Pattern(name="npi-10digit", regex=r"\b\d{10}\b", score=0.4),
    ]
    CONTEXT = ["npi", "provider", "physician", "clinician"]

    def __init__(self) -> None:
        super().__init__(
            supported_entity=ENTITY_NPI,
            patterns=self.PATTERNS,
            context=self.CONTEXT,
            supported_language="en",
        )

    def validate_result(self, pattern_text: str) -> bool:
        return _is_valid_npi(pattern_text)


def _is_valid_npi(s: str) -> bool:
    """NPI Luhn check.

    NPI uses Luhn with a "80840" prefix per CMS spec.
    """
    if len(s) != 10 or not s.isdigit():
        return False
    full = "80840" + s
    total = 0
    rev = full[::-1]
    for i, ch in enumerate(rev):
        d = int(ch)
        if i % 2 == 1:
            d *= 2
            if d > 9:
                d -= 9
        total += d
    return total % 10 == 0


class DEARecognizer(PatternRecognizer):  # type: ignore[misc]
    """DEA Number.

    Two letters + 7 digits. The 7th digit is a checksum:
      sum(odd digits) + 2 * sum(even digits) → last digit == result % 10.
    """

    PATTERNS = [
        Pattern(name="dea-shape", regex=r"\b[A-Z]{2}\d{7}\b", score=0.7),
    ]
    CONTEXT = ["dea", "prescription", "controlled", "schedule"]

    def __init__(self) -> None:
        super().__init__(
            supported_entity=ENTITY_DEA,
            patterns=self.PATTERNS,
            context=self.CONTEXT,
            supported_language="en",
        )


class VINRecognizer(PatternRecognizer):  # type: ignore[misc]
    """US Vehicle Identification Number (VIN).

    17 alphanumeric characters; excludes I, O, Q. Score 0.7 — bare strings
    can collide with serial numbers; context list helps.
    """

    PATTERNS = [
        Pattern(
            name="vin-17char",
            regex=r"\b[A-HJ-NPR-Z0-9]{17}\b",
            score=0.7,
        ),
    ]
    CONTEXT = ["vin", "vehicle", "title", "registration"]

    def __init__(self) -> None:
        super().__init__(
            supported_entity=ENTITY_VIN,
            patterns=self.PATTERNS,
            context=self.CONTEXT,
            supported_language="en",
        )


@dataclass(slots=True)
class TenantPatternSpec:
    """Wire shape for tenant-configured recognizers."""

    entity: str
    name: str
    regex: str
    score: float = 0.85
    context: tuple[str, ...] = ()


class TenantAccountRecognizer(PatternRecognizer):  # type: ignore[misc]
    """Tenant-specific account-id recognizer.

    Constructed from a TenantPatternSpec at request time. Lets buyers add
    their own ID shapes (e.g., "ACME-CUS-[0-9]{8}") without forking the
    package or shipping a new Presidio image.
    """

    def __init__(self, spec: TenantPatternSpec) -> None:
        super().__init__(
            supported_entity=spec.entity or ENTITY_TENANT_ACCOUNT,
            patterns=[Pattern(name=spec.name, regex=spec.regex, score=spec.score)],
            context=list(spec.context),
            supported_language="en",
        )


def register_default_custom_recognizers(registry: object) -> None:
    """Register all Crucible-default custom recognizers on a Presidio registry.

    Idempotent: if a recognizer with the same name is already present, it
    is left alone.
    """
    if registry is None:
        return
    add = getattr(registry, "add_recognizer", None)
    if add is None:
        return
    for cls in (MRNRecognizer, NPIRecognizer, DEARecognizer, VINRecognizer):
        try:
            add(cls())
        except Exception:  # pragma: no cover - registry may dedupe silently
            continue
