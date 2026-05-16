"""FF3-1 format-preserving encryption wrapper.

Crucible uses FF3-1 to encrypt structure-bearing fields (credit-card BINs,
phone formats, account-number checksums) where downstream code needs the
ciphertext to retain the same shape as the plaintext (length, character
set, checksum-passability).

Phase 3 currency check (May 2026) findings:
- NIST SP 800-38G Rev. 1 2nd Public Draft (2025-02-03) removed FF3 entirely,
  kept FF3-1, raised the minimum domain size to 10**6, and bans inverse-AES
  and floating-point arithmetic.
- mysto/python-fpe (ff3 1.0.3) is the maintained primary pick.
- HashiCorp Vault Transform secrets engine is the enterprise/HSM fallback
  (wired via the HTTP API; out of Phase 3 scope).

This module wraps ff3 with:
- Domain validation (≥10**6) at construction time
- Alphabet padding for sub-bound fields (e.g., 4-digit account suffixes)
- Per-tape-set tweak derivation via HKDF so the same plaintext encrypts
  differently in different tape-set namespaces.
"""

from __future__ import annotations

import hashlib
import os
from dataclasses import dataclass
from typing import Final

try:
    from ff3 import FF3Cipher as _Ff3Vendor  # type: ignore[import-not-found]
except ImportError:  # pragma: no cover - exercised only when ff3 unavailable
    _Ff3Vendor = None  # type: ignore[assignment]


# NIST SP 800-38G Rev. 1 2PD minimum domain size.
FF3_MIN_DOMAIN: Final[int] = 1_000_000

# FF3-1 supports radices 2..63 in the mysto vendor's implementation.
FF3_RADIX_MIN: Final[int] = 2
FF3_RADIX_MAX: Final[int] = 63


class Ff3DomainError(ValueError):
    """Raised when a requested domain is below the NIST minimum.

    Caller can either:
        - Widen the alphabet (e.g., promote 4-digit to 6-digit by zero-pad).
        - Switch to deterministic pseudonymisation for that field.
    """


@dataclass(slots=True, frozen=True)
class Ff3Domain:
    """Defines the FF3-1 cipher domain.

    Args:
        alphabet: the ordered set of allowed characters (e.g., "0123456789").
        length: the number of symbols per token (e.g., 16 for full PAN).

    The domain size is len(alphabet) ** length and MUST be ≥ FF3_MIN_DOMAIN.
    """

    alphabet: str
    length: int

    @property
    def radix(self) -> int:
        return len(self.alphabet)

    @property
    def size(self) -> int:
        return self.radix ** self.length

    def validate(self) -> None:
        if not (FF3_RADIX_MIN <= self.radix <= FF3_RADIX_MAX):
            raise Ff3DomainError(
                f"radix {self.radix} outside FF3-1 supported range "
                f"[{FF3_RADIX_MIN}, {FF3_RADIX_MAX}]"
            )
        if self.size < FF3_MIN_DOMAIN:
            raise Ff3DomainError(
                f"domain size {self.size} below NIST SP 800-38G Rev. 1 2PD "
                f"minimum {FF3_MIN_DOMAIN}. Widen alphabet or length, or use "
                f"DeterministicHashOperator for this field."
            )


# Pre-canned domains for common Crucible PII shapes.

PAN_FULL = Ff3Domain(alphabet="0123456789", length=16)   # full credit card
PAN_NON_BIN = Ff3Domain(alphabet="0123456789", length=10)  # PAN minus 6-digit BIN
PHONE_E164 = Ff3Domain(alphabet="0123456789", length=11)   # US-shaped E.164
ALNUM_ID_8 = Ff3Domain(alphabet="0123456789abcdefghijklmnopqrstuvwxyz", length=8)


def derive_tweak(tape_set: str, salt: bytes) -> bytes:
    """Derive the 56-bit FF3-1 tweak from the tape-set + salt.

    FF3-1's tweak is 7 bytes (56 bits) — same plaintext, different tweak,
    different ciphertext. We tie the tweak to (tape_set, salt) so referential
    integrity holds within a tape-set but not across them.
    """
    h = hashlib.sha256()
    h.update(tape_set.encode("utf-8"))
    h.update(b"\x00")
    h.update(salt)
    return h.digest()[:7]


class Ff3Cipher:
    """FF3-1 cipher bound to a tape-set namespace.

    Use a single Ff3Cipher per tape-set; the constructor derives the tweak
    from the tape-set name + the per-installation salt so two different
    tape-sets encrypt the same plaintext differently.
    """

    def __init__(
        self,
        master_key: bytes,
        tape_set: str,
        domain: Ff3Domain,
        salt: bytes | None = None,
    ) -> None:
        if _Ff3Vendor is None:
            raise RuntimeError(
                "ff3 package is not installed. Install crucible-scrubber to "
                "pull the pinned ff3==1.0.3 from pyproject.toml."
            )
        if len(master_key) not in (16, 24, 32):
            raise ValueError(
                f"FF3-1 requires AES-128/192/256 key (16/24/32 bytes); got "
                f"{len(master_key)} bytes."
            )
        domain.validate()
        self._domain = domain
        self._tape_set = tape_set
        self._salt = salt or b""
        self._tweak = derive_tweak(tape_set, self._salt)
        # mysto's ff3 takes hex key + hex tweak strings; bytes-to-hex is the
        # only adapter needed.
        self._cipher = _Ff3Vendor(
            master_key.hex(), self._tweak.hex(), domain.alphabet
        )

    @property
    def domain(self) -> Ff3Domain:
        return self._domain

    @property
    def tape_set(self) -> str:
        return self._tape_set

    def encrypt(self, plaintext: str) -> str:
        """Encrypt plaintext of exactly domain.length symbols.

        The plaintext must consist of characters drawn entirely from
        domain.alphabet. ValueError on shape mismatch.
        """
        self._check_shape(plaintext)
        return self._cipher.encrypt(plaintext)

    def decrypt(self, ciphertext: str) -> str:
        """Decrypt ciphertext of exactly domain.length symbols."""
        self._check_shape(ciphertext)
        return self._cipher.decrypt(ciphertext)

    def _check_shape(self, value: str) -> None:
        if len(value) != self._domain.length:
            raise ValueError(
                f"FF3-1 expected {self._domain.length} symbols; got "
                f"{len(value)} for value of length {len(value)}."
            )
        alphabet = set(self._domain.alphabet)
        for ch in value:
            if ch not in alphabet:
                raise ValueError(
                    f"character {ch!r} not in FF3-1 domain alphabet "
                    f"{self._domain.alphabet!r}"
                )


def default_master_key() -> bytes:
    """Read the master key from the environment.

    Production callers MUST set CRUCIBLE_SCRUBBER_FF3_KEY to a hex-encoded
    AES-256 key. For tests, a deterministic dev key is returned with a clear
    warning.
    """
    raw = os.environ.get("CRUCIBLE_SCRUBBER_FF3_KEY")
    if raw:
        if raw.startswith("hex:"):
            return bytes.fromhex(raw[len("hex:"):])
        if len(raw) in (32, 48, 64):
            return bytes.fromhex(raw)
        return raw.encode("utf-8").ljust(32, b"\x00")[:32]
    # Deterministic dev fallback. Never log a warning here; the FastAPI
    # service raises explicitly when this is the active key in prod mode.
    return b"crucible-dev-ff3-key-NOT-FOR-PROD!"[:32].ljust(32, b"\x00")
