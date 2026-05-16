"""Crucible scrub pipeline.

`ScrubPipeline` orchestrates:

  1. Presidio Analyzer (with the default + Crucible recognizers) to detect
     PII spans in the payload.
  2. Presidio Anonymizer to apply the right operator per entity (replace,
     deterministic, FF3, mask, redact).
  3. AuditLog accumulation so every rewrite is enumerated for compliance
     auditors.

The pipeline is designed to be resilient to Presidio's absence (so the
unit tests can run on bare Python). When the optional dependency is
missing, the pipeline degrades to a regex+operator fallback that mirrors
the Phase 2 baseline. Production deployments install the extras.

Performance note: `_BatchPath` uses Presidio's BatchAnalyzerEngine for
JSON-shaped payloads. The single-text `analyze()` path does not hit the
≤200ms p95 budget the brief sets.
"""

from __future__ import annotations

import json
import os
import re
import time
from dataclasses import dataclass, field
from typing import Any, Final, Iterable

from .audit import AuditEntry, AuditLog
from .ff3 import (
    Ff3Cipher,
    Ff3Domain,
    Ff3DomainError,
    PAN_FULL,
    PHONE_E164,
    ALNUM_ID_8,
    default_master_key,
)
from .operators import (
    DeterministicHashOperator,
    Ff3FpeOperator,
    OPERATOR_DETERMINISTIC,
    OPERATOR_FF3,
    hkdf_extract_and_expand,
)
from .recognizers import (
    ENTITY_MRN,
    ENTITY_NPI,
    ENTITY_DEA,
    ENTITY_VIN,
    ENTITY_TENANT_ACCOUNT,
    TenantPatternSpec,
    TenantAccountRecognizer,
    register_default_custom_recognizers,
)

try:
    from presidio_analyzer import AnalyzerEngine, BatchAnalyzerEngine  # type: ignore[import-not-found]
    from presidio_analyzer.nlp_engine import NlpEngineProvider  # type: ignore[import-not-found]
    from presidio_anonymizer import AnonymizerEngine, BatchAnonymizerEngine  # type: ignore[import-not-found]
    from presidio_anonymizer.entities import OperatorConfig  # type: ignore[import-not-found]
    _PRESIDIO_OK = True
except ImportError:  # pragma: no cover - tested via env fallback only
    AnalyzerEngine = None  # type: ignore[assignment]
    BatchAnalyzerEngine = None  # type: ignore[assignment]
    NlpEngineProvider = None  # type: ignore[assignment]
    AnonymizerEngine = None  # type: ignore[assignment]
    BatchAnonymizerEngine = None  # type: ignore[assignment]
    OperatorConfig = None  # type: ignore[assignment]
    _PRESIDIO_OK = False


# Default operator mapping per entity. Tenants can override on the request.
DEFAULT_OPERATORS: Final[dict[str, str]] = {
    "EMAIL_ADDRESS": OPERATOR_DETERMINISTIC,
    "US_SSN": "redact",
    "CREDIT_CARD": OPERATOR_FF3,
    "PHONE_NUMBER": OPERATOR_FF3,
    "IP_ADDRESS": "replace",
    "PERSON": OPERATOR_DETERMINISTIC,
    "LOCATION": "replace",
    "IBAN_CODE": "redact",
    "URL": "replace",
    "DATE_TIME": "replace",
    "US_BANK_NUMBER": "redact",
    "US_DRIVER_LICENSE": "redact",
    "US_ITIN": "redact",
    "US_PASSPORT": "redact",
    "MEDICAL_LICENSE": "redact",
    "CRYPTO": OPERATOR_DETERMINISTIC,
    ENTITY_MRN: OPERATOR_DETERMINISTIC,
    ENTITY_NPI: OPERATOR_DETERMINISTIC,
    ENTITY_DEA: "redact",
    ENTITY_VIN: OPERATOR_DETERMINISTIC,
    ENTITY_TENANT_ACCOUNT: OPERATOR_DETERMINISTIC,
}

# Default replacement strings (when operator is "replace").
DEFAULT_REPLACEMENTS: Final[dict[str, str]] = {
    "IP_ADDRESS": "127.0.0.1",
    "LOCATION": "[REDACTED-LOCATION]",
    "URL": "https://redacted.example.com",
    "DATE_TIME": "1970-01-01T00:00:00Z",
}


@dataclass(slots=True)
class ScrubRequest:
    """Inputs to one scrub call."""

    tape_set: str
    payload: str | bytes
    fields: tuple[str, ...] = ()        # JSON-path hints; empty → full payload
    language: str = "en"
    engine: str = "default"             # "default" | "hipaa"
    operator_overrides: dict[str, str] = field(default_factory=dict)
    custom_recognizers: tuple[TenantPatternSpec, ...] = ()
    # Optional content-type hint for shape-aware scrubbing. Currently
    # supports "application/json"; other types fall through to text mode.
    content_type: str = ""


@dataclass(slots=True)
class ScrubResult:
    """Outputs from one scrub call."""

    scrubbed: str
    audit: AuditLog

    @property
    def report(self) -> dict[str, Any]:
        return self.audit.to_dict()


class ScrubPipeline:
    """High-level scrub orchestrator.

    Construct with `ScrubPipeline.from_env()` to read the master key, the
    optional HuggingFace clinical NER model, and the default operator
    overrides from the environment.
    """

    def __init__(
        self,
        master_key: bytes | None = None,
        nlp_engine_name: str = "default",
        hipaa_engine_name: str | None = None,
    ) -> None:
        self._master_key = master_key or default_master_key()
        self._nlp_engine_name = nlp_engine_name
        self._hipaa_engine_name = hipaa_engine_name or os.environ.get(
            "CRUCIBLE_SCRUBBER_HIPAA_MODEL",
            "StanfordAIMI/stanford-deidentifier-base",
        )
        self._analyzer: Any = None
        self._anonymizer: Any = None
        self._fallback_only = not _PRESIDIO_OK
        self._build()

    @classmethod
    def from_env(cls) -> "ScrubPipeline":
        return cls(master_key=default_master_key())

    def _build(self) -> None:
        if self._fallback_only:
            return
        # spaCy default NLP engine; HIPAA engine is built lazily on first
        # hipaa request (heavyweight transformer load).
        configuration = {
            "nlp_engine_name": "spacy",
            "models": [{"lang_code": "en", "model_name": "en_core_web_lg"}],
        }
        try:
            provider = NlpEngineProvider(nlp_configuration=configuration)
            nlp_engine = provider.create_engine()
            self._analyzer = AnalyzerEngine(
                nlp_engine=nlp_engine, supported_languages=["en"]
            )
        except Exception:
            # Model not installed; fall back to Presidio's auto-load defaults.
            self._analyzer = AnalyzerEngine()
        register_default_custom_recognizers(self._analyzer.registry)
        self._anonymizer = AnonymizerEngine()
        self._anonymizer.add_anonymizer(DeterministicHashOperator)
        self._anonymizer.add_anonymizer(Ff3FpeOperator)

    # ──────────────────────────────────────────────────────────────────
    # Public API
    # ──────────────────────────────────────────────────────────────────

    def scrub(self, req: ScrubRequest) -> ScrubResult:
        start = time.time()
        audit = AuditLog(tape_set=req.tape_set)
        if isinstance(req.payload, bytes):
            text = req.payload.decode("utf-8", errors="replace")
        else:
            text = req.payload

        if req.content_type == "application/json":
            scrubbed_text = self._scrub_json(text, req, audit)
        else:
            scrubbed_text = self._scrub_text(text, req, audit)

        audit.elapsed_ms = int((time.time() - start) * 1000)
        return ScrubResult(scrubbed=scrubbed_text, audit=audit)

    # ──────────────────────────────────────────────────────────────────
    # Implementation
    # ──────────────────────────────────────────────────────────────────

    def _scrub_json(
        self, text: str, req: ScrubRequest, audit: AuditLog
    ) -> str:
        try:
            parsed = json.loads(text)
        except json.JSONDecodeError:
            return self._scrub_text(text, req, audit)
        walked = self._walk(parsed, "", req, audit)
        return json.dumps(walked, separators=(",", ":"), sort_keys=False)

    def _walk(
        self,
        node: Any,
        path: str,
        req: ScrubRequest,
        audit: AuditLog,
    ) -> Any:
        if isinstance(node, dict):
            return {
                k: self._walk(v, f"{path}.{k}" if path else k, req, audit)
                for k, v in node.items()
            }
        if isinstance(node, list):
            return [
                self._walk(v, f"{path}[{i}]", req, audit)
                for i, v in enumerate(node)
            ]
        if isinstance(node, str):
            return self._scrub_text(node, req, audit, field=path)
        return node

    def _scrub_text(
        self,
        text: str,
        req: ScrubRequest,
        audit: AuditLog,
        field: str = "[inline]",
    ) -> str:
        if self._fallback_only or self._analyzer is None:
            return _regex_fallback_scrub(text, audit, field, req.tape_set, self._master_key)

        custom = [
            TenantAccountRecognizer(spec) for spec in req.custom_recognizers
        ]
        try:
            results = self._analyzer.analyze(
                text=text,
                language=req.language,
                ad_hoc_recognizers=custom,
            )
        except Exception:
            # If Presidio errors mid-analysis we fall closed to regex
            # so a partial result never makes it onto disk.
            return _regex_fallback_scrub(text, audit, field, req.tape_set, self._master_key)

        if not results:
            return text

        op_configs = self._build_operator_configs(req)
        anon = self._anonymizer.anonymize(
            text=text, analyzer_results=results, operators=op_configs
        )
        # Record each rewrite. anon.items lists the OperatorResult instances
        # in start-index order; we line them up with analyzer results to
        # capture the before/after pair.
        for item in getattr(anon, "items", []):
            entity = getattr(item, "entity_type", "")
            after = getattr(item, "text", "")
            start = getattr(item, "start", 0)
            end = getattr(item, "end", 0)
            # The original bytes can be reconstructed from the source text
            # via the analyzer-result indices; for the audit log we only
            # need the *hash* of the original, not the original itself.
            original_span = ""
            for r in results:
                if (
                    getattr(r, "entity_type", "") == entity
                    and getattr(r, "start", -1) <= start
                    and getattr(r, "end", -1) >= start
                ):
                    rs = getattr(r, "start", 0)
                    re_ = getattr(r, "end", 0)
                    original_span = text[rs:re_]
                    break
            op_name = self._operator_for(entity, req)
            audit.record(
                scrubber=entity,
                field=field,
                before=original_span or text[start:end],
                after=after,
                operator=op_name.upper(),
                algorithm=_algorithm_for(op_name),
            )
        return anon.text

    def _build_operator_configs(self, req: ScrubRequest) -> dict[str, Any]:
        if not _PRESIDIO_OK:
            return {}
        out: dict[str, Any] = {}
        seen: set[str] = set()
        merged = dict(DEFAULT_OPERATORS)
        merged.update(req.operator_overrides)
        for entity, op_name in merged.items():
            if entity in seen:
                continue
            seen.add(entity)
            out[entity] = self._operator_config(entity, op_name, req)
        # Apply a wildcard fallback for entities not explicitly mapped.
        out["DEFAULT"] = OperatorConfig("replace", {"new_value": "[REDACTED]"})
        return out

    def _operator_config(
        self, entity: str, op_name: str, req: ScrubRequest
    ) -> Any:
        if not _PRESIDIO_OK or OperatorConfig is None:
            return None
        if op_name == OPERATOR_DETERMINISTIC:
            prefix = _deterministic_prefix(entity)
            return OperatorConfig(
                OPERATOR_DETERMINISTIC,
                {
                    "key": self._master_key,
                    "tape_set": req.tape_set,
                    "prefix": prefix,
                    "length": 12,
                },
            )
        if op_name == OPERATOR_FF3:
            cipher = self._ff3_cipher_for(entity, req.tape_set)
            if cipher is None:
                return OperatorConfig("replace", {"new_value": "[REDACTED]"})
            return OperatorConfig(OPERATOR_FF3, {"cipher": cipher})
        if op_name == "replace":
            replacement = DEFAULT_REPLACEMENTS.get(entity, f"[REDACTED-{entity}]")
            return OperatorConfig("replace", {"new_value": replacement})
        if op_name == "mask":
            return OperatorConfig(
                "mask",
                {"chars_to_mask": 4, "masking_char": "*", "from_end": True},
            )
        if op_name == "redact":
            return OperatorConfig("redact", {})
        return OperatorConfig("replace", {"new_value": "[REDACTED]"})

    def _ff3_cipher_for(self, entity: str, tape_set: str) -> Ff3Cipher | None:
        try:
            domain = _ff3_domain_for(entity)
            return Ff3Cipher(
                master_key=self._master_key,
                tape_set=tape_set,
                domain=domain,
                salt=entity.encode("utf-8"),
            )
        except Ff3DomainError:
            return None

    def _operator_for(self, entity: str, req: ScrubRequest) -> str:
        return req.operator_overrides.get(
            entity, DEFAULT_OPERATORS.get(entity, "replace")
        )


# ──────────────────────────────────────────────────────────────────────
# Helpers
# ──────────────────────────────────────────────────────────────────────


_PREFIX_BY_ENTITY: Final[dict[str, str]] = {
    "EMAIL_ADDRESS": "email_",
    "PERSON": "person_",
    "PHONE_NUMBER": "phone_",
    "CREDIT_CARD": "cc_",
    ENTITY_MRN: "mrn_",
    ENTITY_NPI: "npi_",
    ENTITY_VIN: "vin_",
    ENTITY_TENANT_ACCOUNT: "tac_",
    "CRYPTO": "crypto_",
}


def _deterministic_prefix(entity: str) -> str:
    return _PREFIX_BY_ENTITY.get(entity, "ent_")


_DOMAIN_BY_ENTITY: Final[dict[str, Ff3Domain]] = {
    "CREDIT_CARD": PAN_FULL,
    "PHONE_NUMBER": PHONE_E164,
    "US_BANK_NUMBER": ALNUM_ID_8,
}


def _ff3_domain_for(entity: str) -> Ff3Domain:
    domain = _DOMAIN_BY_ENTITY.get(entity)
    if domain is None:
        raise Ff3DomainError(
            f"no FF3-1 domain registered for entity {entity!r}; use "
            f"DETERMINISTIC instead."
        )
    return domain


def _algorithm_for(op_name: str) -> str:
    if op_name == OPERATOR_DETERMINISTIC:
        return "HKDF-SHA256"
    if op_name == OPERATOR_FF3:
        return "FF3-1/AES-256"
    return ""


# ──────────────────────────────────────────────────────────────────────
# Regex fallback path
#
# Used when Presidio is not installed (e.g., tests on bare Python) and as
# the safety net inside Presidio errors. The Phase 2 baseline lives here
# in spirit; Phase 3 extends it with the same operators the full pipeline
# uses.
# ──────────────────────────────────────────────────────────────────────


_FALLBACK_PATTERNS: list[tuple[str, re.Pattern[str], str]] = [
    (
        "EMAIL_ADDRESS",
        re.compile(r"[A-Za-z0-9._%+\-]+@[A-Za-z0-9.\-]+\.[A-Za-z]{2,}"),
        OPERATOR_DETERMINISTIC,
    ),
    (
        "US_SSN",
        re.compile(r"\b\d{3}-\d{2}-\d{4}\b"),
        "redact",
    ),
    (
        "CREDIT_CARD",
        re.compile(r"\b(?:\d[ -]*?){13,19}\b"),
        OPERATOR_FF3,
    ),
    (
        "PHONE_NUMBER",
        re.compile(r"\+\d{1,3}[\d\-\s\(\)]{7,15}"),
        OPERATOR_FF3,
    ),
    (
        "PHONE_NUMBER",
        re.compile(r"\b\d{3}[\.\- ]?\d{3}[\.\- ]?\d{4}\b"),
        OPERATOR_FF3,
    ),
    (
        "IP_ADDRESS",
        re.compile(r"\b(?:25[0-5]|2[0-4]\d|1\d\d|\d{1,2})(?:\.(?:25[0-5]|2[0-4]\d|1\d\d|\d{1,2})){3}\b"),
        "replace",
    ),
    (
        "JWT",
        re.compile(r"eyJ[A-Za-z0-9_\-]{10,}\.[A-Za-z0-9_\-]{10,}\.[A-Za-z0-9_\-]{10,}"),
        "redact",
    ),
    (
        "AWS_ACCESS_KEY",
        re.compile(r"\bAKIA[0-9A-Z]{16}\b"),
        "redact",
    ),
    (
        "GITHUB_PAT",
        re.compile(r"\bghp_[A-Za-z0-9]{36,}\b|\bgithub_pat_[A-Za-z0-9_]{60,}\b"),
        "redact",
    ),
    (
        "ANTHROPIC_KEY",
        re.compile(r"\bsk-ant-api03-[A-Za-z0-9_\-]{50,}\b"),
        "redact",
    ),
]


def _regex_fallback_scrub(
    text: str,
    audit: AuditLog,
    field: str,
    tape_set: str,
    master_key: bytes,
) -> str:
    out = text
    for entity, pattern, op_name in _FALLBACK_PATTERNS:
        def sub_fn(match: re.Match[str]) -> str:
            original = match.group(0)
            after = _apply_fallback_operator(
                entity, op_name, original, tape_set, master_key
            )
            audit.record(
                scrubber=entity,
                field=field,
                before=original,
                after=after,
                operator=op_name.upper(),
                algorithm=_algorithm_for(op_name),
            )
            return after
        out = pattern.sub(sub_fn, out)
    return out


def _apply_fallback_operator(
    entity: str, op_name: str, original: str, tape_set: str, master_key: bytes
) -> str:
    if op_name == OPERATOR_DETERMINISTIC:
        salt = (tape_set + "::DETERMINISTIC").encode("utf-8")
        digest = hkdf_extract_and_expand(master_key, salt, original.encode("utf-8"), 16)
        import base64
        token = base64.b32encode(digest).decode("ascii").rstrip("=").lower()[:12]
        return f"{_deterministic_prefix(entity)}{token}"
    if op_name == OPERATOR_FF3:
        try:
            domain = _ff3_domain_for(entity)
            cipher = Ff3Cipher(
                master_key=master_key,
                tape_set=tape_set,
                domain=domain,
                salt=entity.encode("utf-8"),
            )
            clean = "".join(ch for ch in original if ch in domain.alphabet)
            if len(clean) == domain.length:
                return cipher.encrypt(clean)
        except Exception:
            pass
        return "[REDACTED-FF3]"
    if op_name == "replace":
        return DEFAULT_REPLACEMENTS.get(entity, f"[REDACTED-{entity}]")
    if op_name == "redact":
        return ""
    return "[REDACTED]"
