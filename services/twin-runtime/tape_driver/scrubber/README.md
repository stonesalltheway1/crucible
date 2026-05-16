# Crucible Scrubber

Production PII scrub pipeline for Crucible Twin Runtime tapes. Phase 3 of the
overall build (`2026.06.0-phase3`); replaces the Phase 2 regex-only baseline.

This is the **regulated-buyer story**. The pipeline runs at **capture time**
(before bytes hit disk), produces a structured **scrub audit log**, and is
designed against the HIPAA Safe Harbor 18-identifier list, GDPR Art. 25
data-minimization, and PCI-DSS PAN containment.

## Pipeline

```
HTTP/gRPC request/response received (raw)
    │
    ▼
┌─────────────────────────────────────────────────────────────────┐
│ 1. PresidioAnalyzer                                             │
│    • default recognizers (email, SSN, CC, phone, IP, IBAN, ...) │
│    • custom recognizers (MRN, account-IDs, tenant-specific)    │
│    • optional TransformersNlpEngine (Stanford clinical NER)    │
│    • spaCy NER backbone for free-text fields                   │
└─────────────────────────────────────────────────────────────────┘
    │
    ▼
┌─────────────────────────────────────────────────────────────────┐
│ 2. PresidioAnonymizer + custom Operators                        │
│    • DETERMINISTIC operator → HKDF(tape_set_key, entity_value) │
│      preserves referential integrity (cus_abc → cus_zzz)       │
│    • FPE operator → FF3-1 for structure-bearing fields         │
│    • REDACT / REPLACE / MASK / HASH still available            │
└─────────────────────────────────────────────────────────────────┘
    │
    ▼
┌─────────────────────────────────────────────────────────────────┐
│ 3. AuditLog                                                     │
│    • every rewrite recorded with (scrubber, field, before_hash, │
│      after, operator, algorithm, ff3_domain_size)               │
│    • signed by the scrubber identity (the in-toto Statement is  │
│      emitted by the Go side after the response returns)        │
└─────────────────────────────────────────────────────────────────┘
    │
    ▼
Tape entry persisted (content-addressed; never returns to client unscrubbed).
```

## Critical design notes (Phase 3 currency check, May 2026)

- **Presidio 2.2.362's `hash` operator now uses a random salt by default.**
  That breaks referential integrity. We ship `DeterministicHashOperator` in
  `crucible_scrubber.operators`, keyed via HKDF off the per-tape-set master
  secret. Operators in user configs SHOULD use `operator_name="DETERMINISTIC"`
  instead of `"hash"`.

- **FF3-1's minimum domain is 10⁶** (NIST SP 800-38G Rev. 1 2PD, 2025-02-03).
  A 6-digit credit-card BIN sits exactly at the bound; a 4-digit account
  suffix is below it. `crucible_scrubber.ff3` pads any sub-bound field into
  a wider alphabet before calling the cipher and validates at config time.

- **Presidio has no built-in auth.** The Go tape_driver fronts this service;
  do not expose `/scrub` directly to the agent.

- **`BatchAnalyzerEngine`** is the path for tape-payload scrubbing — single-
  text `analyze()` will not hit our latency budget. `pipeline.scrub_batch`
  uses it.

## SLA and quality

- ≥99% recall on the adversarial corpus in `tests/test_recall_corpus.py`
  (1000 synthetic PII strings spanning 22 categories).
- ≤ 200ms p95 per HTTP scrub call on a 4 KB payload (Anthropic Haiku class
  hosts).
- False-negative rate surfaced via the audit log so customers can audit.

## HTTP API

```
POST /scrub
Authorization: Bearer <SHARED_TOKEN>   # enforced by the Go fronting service
Content-Type: application/json

{
  "tape_set": "acme/staging/2026-05",
  "payload": "...",
  "fields": ["body", "headers.X-User-Email"],  // optional
  "language": "en",
  "engine": "default" | "hipaa"
}

→ 200
{
  "scrubbed": "...",
  "report": {
    "rewrites": [{
      "scrubber": "us-ssn",
      "field": "body",
      "before_hash": "sha256:...",
      "after": "XXX-XX-XXXX",
      "operator": "REDACT"
    }, ...],
    "elapsed_ms": 87
  }
}
```

## Out of scope (Phase 3 explicit)

- Tape-replay scrubbing — too late by definition. Scrub happens at capture.
- Custom-tokenizer training. The HIPAA tier uses an off-the-shelf clinical
  de-identifier; bespoke training is a Phase 5+ Memory Layer concern.
- HSM-backed FF3-1 keys for self-host. Wired through Vault Transform later;
  Phase 3 uses an environment-supplied key with an HKDF salt.
