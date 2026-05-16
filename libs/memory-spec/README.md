# memory-spec

Source-of-truth schemas for Crucible's procedural memory layer (Phase 5).

This package consolidates types that span:

- `services/memory-router/` — Go hot-path retrieval daemon
- `services/distiller/` — Python background extraction worker
- `apps/control-plane/` — wires `twin.memory.*` through to the router
- `apps/verifier/` — runs `twin.memory.checkCompliance` during T1+ rubric scoring
- `infra/databases/` — pgvector + FalkorDB schema migrations
- `infra/oss-corpus-bootstrap/` — generates the global-defaults JSON bundles

The procedural-memory `Convention` + `Memory` + `ComplianceReport` proto
messages already live in `libs/twin-spec/proto/crucible/v1/memory.proto`
because every SDK language must generate them. This package adds the
memory-layer-only types that DON'T need to ride the agent-facing SDK
surface — taxonomy enums, layer enums, ingest payloads, retrieval scores,
drift events, federation graduation records, RetrievalRouter scoring
weights.

## Files

```
proto/crucible/v1/
  memory_layer.proto       MemoryLayer, ConventionTaxonomy, RuleStatus,
                           RetrievalQuery, RetrievalResult, ConventionCandidate,
                           ConventionDrift, FederationGraduation, AdmissionScore
  distiller.proto          DistillerJob, ExtractionResult, JudgeVerdict,
                           SourceChannel, AgreementScore
  cartographer.proto       CartographerJob, RepoScanResult, InferredAgentsMd

schemas/
  convention_v1.json       JSON-Schema for the disk-serialized Convention
                           (per-stack default bundles + override files)
  bundle_v1.json           Per-stack default bundle (Tier A-D corpus output)
  agents_md_inferred_v1.json   Cartographer output shape

go/                        Hand-rolled Go types in lock-step with the proto;
                           consumed by services/memory-router and the verifier
                           bridge. JSON tags snake_case to match disk format.

py/                        Hand-rolled Python dataclasses for the distiller;
                           pydantic v2 models gated behind a try-import.
```

## Why not put everything in twin-spec?

twin-spec is the agent-visible surface. Adding `ConventionDrift` or
`AdmissionScore` to it would make those types ride every SDK language.
Memory-layer internals (admission control, federation graduation, drift
state machines) are server-side; bloat costs us SDK size + version
discipline. memory-spec absorbs the server-internal types and re-exports
the agent-visible ones from twin-spec.

## Versioning

Memory-spec types live at `crucible.v1.*` alongside twin-spec. Bumping a
disk schema requires a `convention_v2.json` shadow file; the
RetrievalRouter loads v1 + v2 transparently and migrates on next write.
