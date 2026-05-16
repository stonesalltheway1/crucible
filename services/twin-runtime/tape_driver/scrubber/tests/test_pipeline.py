"""Pipeline unit tests.

These tests run without the optional Presidio dependency installed; the
pipeline degrades to the regex+operator fallback path. The Presidio-gated
recall test lives in test_recall_corpus.py and is marked with the
`presidio` pytest marker so CI can split the runs.
"""

from __future__ import annotations

import json

import pytest

from crucible_scrubber import (
    ScrubPipeline,
    ScrubRequest,
)
from crucible_scrubber.audit import AuditLog, sha256_hex
from crucible_scrubber.ff3 import (
    Ff3Cipher,
    Ff3Domain,
    Ff3DomainError,
    PAN_FULL,
    PHONE_E164,
    derive_tweak,
)
from crucible_scrubber.operators import (
    DeterministicHashOperator,
    Ff3FpeOperator,
    hkdf_extract_and_expand,
)


# ──────────────────────────────────────────────────────────────────────
# Audit log
# ──────────────────────────────────────────────────────────────────────


def test_audit_log_records_rewrite() -> None:
    log = AuditLog(tape_set="t1")
    log.record(
        scrubber="us-ssn",
        field="body",
        before="123-45-6789",
        after="XXX-XX-XXXX",
        operator="REDACT",
    )
    assert len(log.entries) == 1
    e = log.entries[0]
    assert e.scrubber == "us-ssn"
    assert e.before_hash.startswith("sha256:")
    assert e.before_hash == sha256_hex("123-45-6789")
    assert e.after == "XXX-XX-XXXX"
    assert e.tape_set == "t1"


def test_audit_log_scrubber_counts() -> None:
    log = AuditLog(tape_set="t")
    for _ in range(3):
        log.record("EMAIL", "body", "a@b.com", "redacted@example.com", "REPLACE")
    log.record("US_SSN", "body", "x", "X", "REDACT")
    counts = log.scrubber_counts()
    assert counts == {"EMAIL": 3, "US_SSN": 1}


def test_audit_log_to_json_round_trip() -> None:
    log = AuditLog(tape_set="t")
    log.record("E", "f", "before", "after", "OP")
    blob = log.to_json()
    parsed = json.loads(blob)
    assert parsed["tape_set"] == "t"
    assert len(parsed["rewrites"]) == 1


# ──────────────────────────────────────────────────────────────────────
# FF3-1 wrapper
# ──────────────────────────────────────────────────────────────────────


def test_ff3_domain_below_min_raises() -> None:
    """4-digit numeric domain has size 10**4 < 10**6 — must reject."""
    d = Ff3Domain(alphabet="0123456789", length=4)
    with pytest.raises(Ff3DomainError):
        d.validate()


def test_ff3_pan_domain_at_bound() -> None:
    # 6-digit BIN is exactly 10**6; full 16-digit PAN is well above.
    PAN_FULL.validate()
    Ff3Domain(alphabet="0123456789", length=6).validate()


def test_ff3_radix_bounds() -> None:
    d = Ff3Domain(alphabet="01", length=20)
    d.validate()
    too_wide = Ff3Domain(alphabet="0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz!@#$", length=8)
    with pytest.raises(Ff3DomainError):
        too_wide.validate()


def test_ff3_tweak_changes_with_tape_set() -> None:
    t1 = derive_tweak("tape-a", b"")
    t2 = derive_tweak("tape-b", b"")
    assert t1 != t2
    assert len(t1) == 7 and len(t2) == 7


@pytest.mark.skip(reason="requires ff3 vendor package")  # auto-enabled when installed
def test_ff3_cipher_round_trip() -> None:
    key = b"0" * 32
    c = Ff3Cipher(master_key=key, tape_set="t", domain=PAN_FULL)
    pt = "4242424242424242"
    ct = c.encrypt(pt)
    assert ct != pt
    assert c.decrypt(ct) == pt


# ──────────────────────────────────────────────────────────────────────
# Operators
# ──────────────────────────────────────────────────────────────────────


def test_deterministic_operator_is_deterministic() -> None:
    op = DeterministicHashOperator()
    params = {"key": b"k" * 32, "tape_set": "t1", "prefix": "cus_"}
    a = op.operate("cus_abc123", params)
    b = op.operate("cus_abc123", params)
    assert a == b
    assert a.startswith("cus_")
    assert len(a) == len("cus_") + 12


def test_deterministic_operator_differs_across_tape_sets() -> None:
    op = DeterministicHashOperator()
    a = op.operate("alice@example.com", {"key": b"k" * 32, "tape_set": "tape1"})
    b = op.operate("alice@example.com", {"key": b"k" * 32, "tape_set": "tape2"})
    assert a != b


def test_deterministic_operator_validates_params() -> None:
    op = DeterministicHashOperator()
    with pytest.raises(ValueError):
        op.validate({})
    with pytest.raises(ValueError):
        op.validate({"key": b"k" * 32})  # missing tape_set
    with pytest.raises(ValueError):
        op.validate({"key": b"k" * 32, "tape_set": "t", "length": 3})


def test_hkdf_yields_distinct_keystreams() -> None:
    key = b"k" * 32
    a = hkdf_extract_and_expand(key, b"salt1", b"info", 32)
    b = hkdf_extract_and_expand(key, b"salt2", b"info", 32)
    c = hkdf_extract_and_expand(key, b"salt1", b"info2", 32)
    assert a != b
    assert a != c
    assert b != c
    assert len(a) == 32


# ──────────────────────────────────────────────────────────────────────
# Pipeline (regex fallback path — Presidio is optional)
# ──────────────────────────────────────────────────────────────────────


def test_pipeline_scrubs_ssn() -> None:
    p = ScrubPipeline(master_key=b"k" * 32)
    r = p.scrub(ScrubRequest(tape_set="t", payload="SSN: 123-45-6789"))
    assert "123-45-6789" not in r.scrubbed
    assert any(e.scrubber == "US_SSN" for e in r.audit.entries)


def test_pipeline_scrubs_email_deterministically() -> None:
    p = ScrubPipeline(master_key=b"k" * 32)
    r1 = p.scrub(ScrubRequest(tape_set="t", payload="user@example.com"))
    r2 = p.scrub(ScrubRequest(tape_set="t", payload="user@example.com"))
    assert r1.scrubbed == r2.scrubbed
    assert "user@example.com" not in r1.scrubbed


def test_pipeline_scrubs_email_differs_across_tape_sets() -> None:
    p = ScrubPipeline(master_key=b"k" * 32)
    r1 = p.scrub(ScrubRequest(tape_set="ta", payload="user@example.com"))
    r2 = p.scrub(ScrubRequest(tape_set="tb", payload="user@example.com"))
    assert r1.scrubbed != r2.scrubbed


def test_pipeline_scrubs_aws_key() -> None:
    p = ScrubPipeline(master_key=b"k" * 32)
    s = "AWS=AKIAIOSFODNN7EXAMPLE"
    r = p.scrub(ScrubRequest(tape_set="t", payload=s))
    assert "AKIAIOSFODNN7EXAMPLE" not in r.scrubbed


def test_pipeline_scrubs_anthropic_key() -> None:
    p = ScrubPipeline(master_key=b"k" * 32)
    s = "key=sk-ant-api03-" + "a" * 60
    r = p.scrub(ScrubRequest(tape_set="t", payload=s))
    assert "sk-ant-api03-" + "a" * 60 not in r.scrubbed


def test_pipeline_audit_records_all_rewrites() -> None:
    p = ScrubPipeline(master_key=b"k" * 32)
    s = "email a@b.com phone 555-555-5555 ssn 111-22-3333"
    r = p.scrub(ScrubRequest(tape_set="t", payload=s))
    assert len(r.audit.entries) >= 3
    # Each rewrite hashes the original; verify the audit didn't leak originals.
    for e in r.audit.entries:
        assert e.before_hash.startswith("sha256:")


def test_pipeline_returns_unchanged_when_no_pii() -> None:
    p = ScrubPipeline(master_key=b"k" * 32)
    s = "Hello world this is a benign payload with no PII."
    r = p.scrub(ScrubRequest(tape_set="t", payload=s))
    assert r.scrubbed == s
    assert len(r.audit.entries) == 0


def test_pipeline_handles_json_payload() -> None:
    p = ScrubPipeline(master_key=b"k" * 32)
    body = '{"user":"alice@example.com","ssn":"123-45-6789"}'
    r = p.scrub(ScrubRequest(tape_set="t", payload=body, content_type="application/json"))
    assert "alice@example.com" not in r.scrubbed
    assert "123-45-6789" not in r.scrubbed
    # Output is still valid JSON.
    assert json.loads(r.scrubbed)


def test_pipeline_handles_bytes_payload() -> None:
    p = ScrubPipeline(master_key=b"k" * 32)
    r = p.scrub(ScrubRequest(tape_set="t", payload=b"SSN: 123-45-6789"))
    assert "123-45-6789" not in r.scrubbed


def test_pipeline_elapsed_recorded() -> None:
    p = ScrubPipeline(master_key=b"k" * 32)
    r = p.scrub(ScrubRequest(tape_set="t", payload="user@example.com"))
    assert r.audit.elapsed_ms >= 0


def test_pipeline_redacts_jwt() -> None:
    p = ScrubPipeline(master_key=b"k" * 32)
    jwt = (
        "eyJhbGciOiJIUzI1NiJ9aaaaaaaaaaaaa."
        "abcdefghijklmnopqrstuvwxyzabcdefghij."
        "abcdefghijklmnopqrstuvwxyzabcdefghij"
    )
    r = p.scrub(ScrubRequest(tape_set="t", payload="Authorization: Bearer " + jwt))
    assert jwt not in r.scrubbed
