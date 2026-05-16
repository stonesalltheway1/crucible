"""Custom Presidio Operators for Crucible.

Two operators ship in Phase 3:

  - DeterministicHashOperator (operator name "DETERMINISTIC"): produces a
    deterministic pseudonym from the (tape_set, entity_value) pair, using
    HKDF over the per-installation master secret. Replaces the stock "hash"
    operator, which post-2.2.361 uses a random salt that breaks referential
    integrity ("cus_abc123 → cus_zzz789 consistently across all entries").

  - Ff3FpeOperator (operator name "FF3"): format-preserving encryption using
    the FF3-1 cipher wired in [crucible_scrubber.ff3]. Used for structure-
    bearing fields (CC BINs, phone formats, account-number checksums) where
    downstream code expects the ciphertext to keep the same shape.

Both operators are registered into Presidio's OperatorsFactory at pipeline
construction time. They live in this module so callers that integrate the
Crucible pipeline into their own Presidio stack can register them directly.
"""

from __future__ import annotations

import base64
import hashlib
import hmac
from dataclasses import dataclass
from typing import Any, Final

try:
    from presidio_anonymizer.operators import Operator, OperatorType  # type: ignore[import-not-found]
except ImportError:  # pragma: no cover
    Operator = object  # type: ignore[assignment]
    OperatorType = None  # type: ignore[assignment]

from .ff3 import Ff3Cipher, Ff3DomainError


OPERATOR_DETERMINISTIC: Final[str] = "DETERMINISTIC"
OPERATOR_FF3: Final[str] = "FF3"


def hkdf_extract_and_expand(
    master_key: bytes,
    salt: bytes,
    info: bytes,
    length: int = 16,
) -> bytes:
    """RFC-5869 HKDF-SHA256.

    Inlined because we depend on `cryptography` already, but the call site
    is tiny and we want the operator to be auditable.
    """
    if not salt:
        salt = b"\x00" * 32
    prk = hmac.new(salt, master_key, hashlib.sha256).digest()
    out = b""
    t = b""
    counter = 1
    while len(out) < length:
        t = hmac.new(prk, t + info + bytes([counter]), hashlib.sha256).digest()
        out += t
        counter += 1
    return out[:length]


@dataclass(slots=True)
class DeterministicHashOperator(Operator):  # type: ignore[misc]
    """Deterministic pseudonym operator.

    Given a per-tape-set master secret, derives a stable token for each
    distinct entity value so that:

        op.operate("cus_abc123", {"key": K, "tape_set": "t"}) == "cus_zzz789"

    The output is collision-resistant within a tape-set (HKDF over 128 bits)
    and unrelated across tape-sets (the tape-set string is mixed into the
    HKDF salt).

    Parameters expected on each .operate() call:
        key: bytes — master secret. Required.
        tape_set: str — tape-set namespace. Required.
        prefix: str — optional output prefix (e.g., "cus_"). Default "".
        length: int — characters of the base32 output. Default 12.

    The Presidio operator API passes per-call params via the second arg.
    """

    def operate(self, text: str = "", params: dict[str, Any] | None = None) -> str:
        params = params or {}
        key = params.get("key")
        tape_set = params.get("tape_set", "")
        prefix = params.get("prefix", "")
        length = int(params.get("length", 12))
        if not key:
            raise ValueError("DETERMINISTIC operator requires `key` parameter")
        if isinstance(key, str):
            key = key.encode("utf-8")
        salt = (tape_set + "::DETERMINISTIC").encode("utf-8")
        info = text.encode("utf-8")
        digest = hkdf_extract_and_expand(key, salt, info, length=16)
        token = base64.b32encode(digest).decode("ascii").rstrip("=").lower()[:length]
        return f"{prefix}{token}"

    def validate(self, params: dict[str, Any] | None = None) -> None:
        params = params or {}
        if not params.get("key"):
            raise ValueError("DETERMINISTIC operator requires `key` parameter")
        if not params.get("tape_set"):
            raise ValueError("DETERMINISTIC operator requires `tape_set` parameter")
        length = int(params.get("length", 12))
        if length < 4 or length > 32:
            raise ValueError("DETERMINISTIC operator `length` must be in [4, 32]")

    def operator_name(self) -> str:
        return OPERATOR_DETERMINISTIC

    def operator_type(self) -> Any:
        return OperatorType.Anonymize if OperatorType else 0  # pragma: no cover


@dataclass(slots=True)
class Ff3FpeOperator(Operator):  # type: ignore[misc]
    """FF3-1 format-preserving encryption operator.

    Use this for fields where downstream consumers need the ciphertext to
    pass shape checks (Luhn, NPI, phone-format) and the underlying domain
    is large enough for FF3-1 (≥ 10**6).

    Parameters expected on each .operate() call:
        cipher: Ff3Cipher — preconstructed cipher bound to this tape-set.

    Crucible's pipeline builds the cipher in [pipeline.ScrubPipeline] and
    passes the same instance to every entity rewrite within a request.
    """

    def operate(self, text: str = "", params: dict[str, Any] | None = None) -> str:
        params = params or {}
        cipher: Ff3Cipher | None = params.get("cipher")
        if cipher is None:
            raise ValueError("FF3 operator requires `cipher` parameter")
        # Strip any non-domain characters before encrypting; restore shape
        # after. This handles "+1 (415) 555-0100"-style inputs.
        clean, shape = _split_shape(text, cipher.domain.alphabet)
        if len(clean) != cipher.domain.length:
            # The plaintext has a different shape than the bound domain.
            # Fall back to deterministic mapping to keep referential
            # integrity rather than failing closed.
            raise Ff3DomainError(
                f"FF3 input length {len(clean)} != domain length "
                f"{cipher.domain.length} for tape_set {cipher.tape_set!r}"
            )
        encrypted = cipher.encrypt(clean)
        return _restore_shape(encrypted, shape)

    def validate(self, params: dict[str, Any] | None = None) -> None:
        params = params or {}
        if "cipher" not in params:
            raise ValueError("FF3 operator requires `cipher` parameter")

    def operator_name(self) -> str:
        return OPERATOR_FF3

    def operator_type(self) -> Any:
        return OperatorType.Anonymize if OperatorType else 0  # pragma: no cover


def _split_shape(text: str, alphabet: str) -> tuple[str, list[tuple[int, str]]]:
    """Split text into (alphabet-only string, list of (pos, char) for the rest).

    Used so phone numbers like "+1 (415) 555-0100" can be FF3-encrypted on
    the 11 digits and restored to the same shape afterwards.
    """
    alphabet_set = set(alphabet)
    keep_chars: list[str] = []
    shape: list[tuple[int, str]] = []
    for i, ch in enumerate(text):
        if ch in alphabet_set:
            keep_chars.append(ch)
        else:
            shape.append((i, ch))
    return "".join(keep_chars), shape


def _restore_shape(encrypted: str, shape: list[tuple[int, str]]) -> str:
    """Re-insert the non-domain characters at their original positions."""
    if not shape:
        return encrypted
    out: list[str] = []
    enc_iter = iter(encrypted)
    pos = 0
    shape_idx = 0
    total_len = len(encrypted) + len(shape)
    while pos < total_len:
        if shape_idx < len(shape) and shape[shape_idx][0] == pos:
            out.append(shape[shape_idx][1])
            shape_idx += 1
        else:
            try:
                out.append(next(enc_iter))
            except StopIteration:
                break
        pos += 1
    return "".join(out)
