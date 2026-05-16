# Tape Coverage Strategy

Resolves open question #1 from the original architecture: what % of agent service calls hit endpoints that ARE in the tape vs not, and what's the right behavior for the misses?

## The empirical baseline

Production API endpoint hit frequency follows a Zipf-like distribution with exponent α typically 0.8–1.2:

- **Alibaba microservice trace (SoCC '21, 20K microservices):** "super microservices" with in-degree ≥ 16 appear in 90% of call graphs and handle 95% of total invocations.
- **Twitter open cache traces (54 clusters, March 2020):** popularity highly skewed.
- **CDN literature:** healthy cache hit rates are 95–99% for static, 20–60% for dynamic personalized.

For coding agents specifically (Cursor, Claude Code, etc.), per-task external call patterns:

- 5–20 distinct external endpoints per feature task.
- 40–200 total calls including retries and pagination.
- Clustered on a handful of read endpoints (e.g., `GET /customers/{id}`, `GET /charges`) and a smaller number of writes.

**Implication for Crucible:** recording the top 5–10% of endpoints by call volume per service covers 80–95% of agent task traffic on read paths. The remaining 5–20% is the long tail — and in agent tasks it correlates strongly with novel/feature-specific work, which is precisely the work where the tape must NOT silently lie.

## How existing tools handle misses

| Tool | Miss strategy |
|---|---|
| Hoverfly | Modes: capture, simulate, modify, spy, synthesize. Spy = replay-or-passthrough. |
| WireMock | Default = HTTP 404. Catch-all low-priority stub. |
| VCR | `:once` / `:new_episodes` / `:none` / `:all` record modes |
| Polly.JS | record / replay / passthrough; per-request `.passthrough()` opt-in |
| Speedscale | "Responsive mocks" with state — replicates production behavior |
| GoReplay | Capture-and-replay; no built-in stub-on-miss |

The mature pattern across these tools: **three primitives — strict replay, replay-or-passthrough, record-on-miss — per session.** Crucible adopts the same primitives but selects per request class, not globally.

## The Crucible decision tree

On every outgoing request from the twin, in priority order:

```
1. Match tape entry EXACTLY (path + method + sig)
     → REPLAY, tag X-Crucible-Tape: hit-exact

2. Match tape entry by TEMPLATE (path pattern + method,
   differing only in IDs / pagination / timestamps)
     → REPLAY with parameter substitution
     → tag X-Crucible-Tape: hit-template
     → confidence: high

3. Miss, but endpoint is in OpenAPI spec
   AND method is read-only (GET / HEAD / OPTIONS)
     → SYNTHESIZE response from schema
       (Prism / Microcks-style Faker + optional LLM augmentation)
     → persist as CANDIDATE tape entry (not auto-promoted)
     → tag X-Crucible-Tape: synth-readonly

4. Miss, endpoint in spec, MUTATING method (POST/PUT/PATCH/DELETE)
     → DETERMINISTIC STUB: spec's default success example
     → RECORD mutation to twin's in-memory state journal
     → NEVER forward to real service
     → tag X-Crucible-Tape: synth-mutation
     → surface to agent: "Mutation simulated; not live."

5. Miss, NOT in spec, task manifest declares live-call allowed for host
     → PASSTHROUGH through PII-scrubbing egress proxy
     → persist response to tape for future runs (VCR :new_episodes)
     → tag X-Crucible-Tape: live-passthrough

6. Miss, NOT in spec, live NOT allowed, request requires auth
     → FAIL CLOSED with 599 Crucible-Tape-Miss
     → structured error body describes what was missing
     → agent sees the error and adapts
     → tag X-Crucible-Tape: miss-blocked

7. Miss, NOT in spec, live NOT allowed, no auth required
     → Policy-driven; default FAIL CLOSED with 599
     → optional per-task override to synth-from-shape
```

## Policy knobs surfaced to users

```
tape.mode             = strict | hybrid | adaptive
tape.synth_engine     = none | schema | schema+llm
tape.allow_live       = [host_allowlist]
tape.mutation_policy  = journal | block
tape.miss_status      = 599
```

Defaults: `hybrid + schema+llm + [] + journal + 599`.

The `X-Crucible-Tape` response header is **the single most important design decision**: agents AND the verifier both *see* whether a response was real, replayed, or synthesized, and weight trust accordingly.

## Auth handling on replay

- **On replay:** match requests after stripping `Authorization` header.
- **On passthrough:** egress proxy injects sandbox-tenant token (not real prod creds).
- **Production tokens never leave the twin.** Twin-scoped Infisical credentials only.

## State-mutating calls

Mutations are **never** replayed as having had effect on real systems. The flow:

1. Agent calls `POST /v1/charges`.
2. Decision tree: synth-mutation (stub success response).
3. Mutation written to twin's in-memory state journal: `{charges: [{id: ch_synth_1, amount: 1234}]}`.
4. Subsequent reads consult the state journal first, then fall through to the tape.

This is the Speedscale "responsive mock" pattern done right. Speedscale's product gestures at it but doesn't ship it cleanly; we make it deterministic.

## PII scrubbing — at capture, not replay

GDPR Art. 25 (data minimization) and HIPAA Safe Harbor (18-identifier list) make it clear: prod-derived test data without de-identification is non-compliant. PCI-DSS pulls raw PAN-bearing data into CDE scope.

Capture pipeline:

```
HTTP/gRPC request/response received
    │
    ▼
Microsoft Presidio Analyzer + Anonymizer
    │ (NER for: names, SSN, credit cards, phones, addresses, emails, MRNs)
    ▼
spaCy NER (free-text fields, response bodies)
    │
    ▼
FF3-1 Format-Preserving Encryption (mysto/python-fpe or Vault transform)
    │ (BINs, phone formats, account-number checksums — structure-bearing fields)
    ▼
Deterministic pseudonymization (per-tape-set key)
    │ (preserves referential integrity: cus_abc → cus_zzz consistently)
    ▼
Synthetic augmentation (Gretel / SDV / MOSTLY AI)
    │ (preserves distributional properties; Jensen-Shannon < 0.1 typical)
    ▼
Audit log (which scrubbers fired, which fields rewritten)
    │
    ▼
Tape persisted (content-addressed by request_hash)
```

Scrubbing must happen **at capture**, before bytes hit disk. Scrubbing on replay is too late — the bytes already exist.

## Tape lifecycle

- **TTL:** 90 days default unless explicitly pinned.
- **Per-tenant storage quota.**
- **LRU eviction** when quota reached.
- **Versioning:** tapes are content-addressed; a service upgrade that changes response shape creates new tape entries; old entries remain available for historical replay but flagged stale after 30 days.
- **Re-capture:** customers configure periodic re-capture schedules; we automate the shadow-recording where they grant permission.

## Tape staleness — the irreducible problem

When upstream service ships a breaking change, tapes silently lie. Mitigations:

1. **Tape-age metrics** surfaced to agents and the verifier (per-endpoint `last_recorded` timestamp).
2. **Promotion canary catches lying tapes** because the canary hits real services, not the tape.
3. **Auto-rollback on canary regression.**
4. **Periodic re-capture cron.** Customer-configurable; default monthly for high-traffic endpoints.
5. **Tape staleness warning** in PR descriptions: "this PR was verified against a tape last refreshed 47 days ago; consider re-recording."

We cannot eliminate this risk entirely. The honest design choice is to expose it.

## Cold-start: brand-new endpoint, no recording

When the agent's task touches an endpoint we've never seen:

1. Check OpenAPI spec for the service. If present → synth-readonly or synth-mutation per decision tree.
2. If no spec and not in `allow_live`: fail closed.
3. If no spec and `allow_live`: passthrough (one-shot capture for future).

The cold-start case is irreducibly worse than the warm-tape case. Customers should populate tapes aggressively in onboarding; we surface tape-coverage metrics in the dashboard.

## Honest assessment of what's unsolved

- **Stateful replay across mutations.** State journal handles short tasks well; long sessions degrade as the journal diverges from "what real prod would have done."
- **Semantically wrong synth responses.** A Faker address is valid JSON but won't pass real address-validation. Agent may take wrong actions based on synth.
- **Free-text PII in JSON.** Presidio + spaCy miss ~5–15% of free-text PII (no public benchmark for adversarial test).
- **Long-tail coverage on novel tasks.** A genuinely novel feature is, by definition, hitting endpoints not yet popular enough to record. First-run of a new feature is where tape fails most.
- **Per-task call-count telemetry for coding agents not publicly published.** Our 5–20 distinct / 40–200 total estimate is from inference. We should instrument and publish.

## Customer-facing onboarding

The recommended setup:

1. **Day 1 of install:** point the shadow-capture agent at staging. Capture for 7 days. Scrub. Result: tape for top 80% of endpoints.
2. **Week 2:** review the scrub-audit report; confirm scrubbing matches compliance requirements.
3. **Week 3:** start running real agent tasks. The first 10% of tasks will hit fail-closed misses; agent reflects, asks for tape extension or live-allow. User accepts or denies; tape grows.
4. **Month 1:** tape coverage stabilizes at ~95%+ for the customer's workload pattern.

We bill onboarding cost (shadow capture, scrub compute) as a one-time setup fee for Team / Enterprise tiers, absorbed for Pro.

## References

- [01-architecture/twin-runtime.md#layer-4-service-twin-tapes](../01-architecture/twin-runtime.md)
- [ADR-007: Hoverfly tape replay](../05-decisions/ADR-007-hoverfly-tape-replay.md)
