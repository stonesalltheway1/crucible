# ADR-007: Hoverfly OSS for service replay

**Status:** Accepted  
**Date:** 2026-05-15

## Context

The twin runtime needs to handle the agent's outbound HTTP/gRPC service calls without touching real services. Three primitives are required:

1. **Recording** — capture production (or staging) traffic for replay.
2. **Replay** — serve recorded responses to the agent during twin tasks.
3. **Modes** — strict / hybrid / passthrough behavior per request class.

The service-virtualization market has several mature options (WireMock, Mountebank, Speedscale, Hoverfly, GoReplay, Mockoon, Prism).

## Decision

**Hoverfly OSS** is the default service-replay engine.

- Capture-replay is first-class (not bolt-on like WireMock).
- Five modes (capture, simulate, modify, spy, synthesize) cover the decision tree in [01-architecture/twin-runtime.md#layer-4-service-twin-tapes](../01-architecture/twin-runtime.md).
- Apache-2.0 license; redistributable.
- Active development, mature operational story.

Crucible wraps Hoverfly with:

1. **PII scrubber at capture** — Presidio + spaCy + FF3-1 + deterministic pseudonymization, applied before bytes hit disk.
2. **Content-addressed tape storage** — keyed by `(service, endpoint, request_hash)`.
3. **Tape decision tree** — exact / template / synthesize / passthrough / fail-closed per [twin-runtime.md](../01-architecture/twin-runtime.md).
4. **`X-Crucible-Tape` response header** — agents see whether a response was real / replayed / synthesized.

For specific cases:

- **gRPC / JVM-heavy stacks:** add WireMock as a complement (stronger gRPC story).
- **Cold-start (no recorded traffic):** synthesize from OpenAPI via Microcks-style LLM-augmented Faker.
- **Pact-defined contracts:** ingest Pact files as a tape source.

## Consequences

### Positive

- **Industry-mature replay primitives.** Hoverfly's modes are battle-tested; we don't reinvent.
- **Apache-2.0 license.** No redistribution issues.
- **Capture+replay in one tool.** Many alternatives separate these concerns.
- **Modes map cleanly to our decision tree.** The spy mode (replay if matched, else forward) is exactly the hybrid behavior we want.

### Negative

- **Hoverfly's stateful-mutation handling is limited.** A POST followed by a GET to the same resource doesn't natively reconcile. Mitigation: Crucible's own state journal sits between Hoverfly and the agent, reconciling write-side mutations.
- **gRPC support is thinner than WireMock's.** Mitigation: pair Hoverfly + WireMock for gRPC-heavy customers.
- **Open-source vendor risk (smaller team).** Mitigation: SpectoLabs maintains the project well; if it ever falters, the OSS code is forkable and our PII-scrub layer is independent.

### Trade-offs we accept

We're betting on a single OSS project for a load-bearing layer. The Apache-2.0 license + active community means we can fork if needed; that's the safety valve.

## Alternatives considered

### Alternative 1: WireMock as primary

Mature, large community, strong JVM ecosystem. **Rejected as primary**:

- Mock-first model; capture-replay is bolt-on (Wiremock-Recorder).
- Java-centric; our team is Go/Rust/Python first.
- Used as a complement for gRPC, not the primary.

### Alternative 2: Speedscale (commercial)

SaaS service-virtualization with auto-detected dependencies and "responsive mocks." **Rejected as primary**:

- Commercial-only; our self-hosted enterprise tier can't bundle it without ongoing license complications.
- Vendor lock-in.
- Useful complement for customers who already use it, but not our default.

### Alternative 3: Mountebank

Multi-protocol (HTTP, TCP, SMTP), Node.js-based. **Rejected**:

- HTTP-only is fine for us; Mountebank's multi-protocol is overkill.
- Hoverfly's capture-replay is stronger.

### Alternative 4: GoReplay

Production traffic shadowing tool. **Rejected as primary**:

- Designed for shadow-testing production, not for offline replay against an agent.
- No native stub-on-miss.
- Useful for the initial-recording phase (capture from production); not for runtime replay.

### Alternative 5: Roll our own

Build a service-replay engine ourselves. **Rejected**:

- ~5+ agent-days that adds zero unique value.
- Hoverfly's behavioral surface is exactly what we need.

### Alternative 6: Mockoon

Lightweight, offline mock server. **Rejected**:

- Designed for solo developer mocking, not production-grade replay at scale.
- No record mode at the depth we need.

## The PII scrubber is the load-bearing addition

Hoverfly itself doesn't scrub PII. Our wrapper does:

```
Capture pipeline (at record time):
  HTTP/gRPC request/response
    ↓
  Presidio Analyzer (NER for PII)
    ↓
  spaCy NER (free-text PII catch)
    ↓
  FF3-1 format-preserving encryption (structure-bearing fields)
    ↓
  Deterministic pseudonymization (referential integrity)
    ↓
  Scrub audit log
    ↓
  Tape persisted
```

This is the regulated-buyer story. GDPR Art. 25 and HIPAA Safe Harbor both demand de-identification before prod-derived test data lands at rest. Hoverfly alone doesn't satisfy; the scrubber does.

## Open issues

- **Tape staleness detection.** When upstream services change their response shape, tapes silently lie. We need a tape-age metric and periodic re-capture pipeline; currently scoped for v2.
- **gRPC streaming support.** Hoverfly's gRPC streaming story is incomplete. WireMock is better for this; we use WireMock as a fallback in gRPC-streaming-heavy stacks.
- **LLM-synthesized response correctness.** When we synthesize for a miss, the response may be syntactically valid but semantically wrong. Currently mitigated via the `X-Crucible-Tape: synth-*` header so the verifier can weight it lower; long-term, fingerprint-and-improve from observed misses.

## References

- [01-architecture/twin-runtime.md#layer-4-service-twin-tapes](../01-architecture/twin-runtime.md)
- [06-research/tape-coverage-strategy.md](../06-research/tape-coverage-strategy.md)
