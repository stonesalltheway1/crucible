# apps/cartographer

The day-1 customer experience: every new repo's first Crucible task is a
Cartographer run. We walk the repo, build a tree-sitter symbol index,
parse every lint config, scan recent PR review comments, look for
incident references, then distill the result into a typed
ConventionCandidate set plus an inferred `AGENTS.md` the customer
reviews before activation.

This Go service is the production orchestrator. It composes:

- `internal/walker/` — tree-sitter file walker for Python, TypeScript,
  Rust, Go, Java, Swift (the six stacks the brief targets).
- `internal/symbols/` — symbol-index builder with per-language adapters
  (pyan / jdeps / go-callvis / ts-morph patterns).
- `internal/lintconfig/` — Tier-A deterministic config extraction (the
  ~30 config types from `docs/06-research/memory-bootstrap.md` §B).
- `internal/agentsmd/` — AGENTS.md / CLAUDE.md / .cursorrules /
  CONTRIBUTING.md / ADR reader.
- `internal/prcomments/` — PR review-comment scanner (last 24 months,
  top 1000 by length, GitHub GraphQL API).
- `internal/incidents/` — incident-reference detector (Linear / Jira /
  Slack #incidents URLs in PR descriptions + body text).
- `internal/distill/` — Haiku 4.5 LLM distillation client with
  schema-constrained output (AdaKGC SDD pattern).
- `internal/agreement/` — cross-source agreement + confidence scoring
  per `docs/06-research/memory-bootstrap.md` §3.
- `internal/oss/` — OSS-derived defaults loader, filtered by stack.
- `internal/inferred/` — inferred AGENTS.md generator (used when the
  customer's repo doesn't already have one).
- `internal/console/` — web-console output formatter ("✓ Indexed 1,247
  files...").
- `internal/api/` — HTTP + gRPC surface the control plane drives.

## Time-to-first-result target

≤ 30 minutes wall-clock on a 50K-LoC repo. The walker is parallelised
by directory; the distillation pass is parallelised by source-document;
the agreement pass is single-threaded but bounded by O(N log N).

## Wiring

Drive from the control plane:

```bash
crucible-control-plane &       # already running
crucible-cartographer &        # this service, default :9420

# Submit a cartography job
curl -X POST http://localhost:9420/v1/cartography \
  -H 'Content-Type: application/json' \
  -d '{
    "tenant_id": "ten_acme",
    "repo": "acme/payments",
    "repo_local_path": "/var/twin/checkout",
    "include_pr_history": true,
    "github_token_secret_ref": "infisical://crucible/acme/github-readonly"
  }'
```

Response is a CartographyResult with the field counts and the inferred
AGENTS.md markdown.

## Streaming progress

The web console subscribes to `/v1/cartography/{job_id}/events` (SSE) to
render the live progress markers ("✓ Indexed 1,247 files. ✓ Extracted
184 conventions...") as each pipeline stage completes.

## Build

```bash
cd apps/cartographer
go build ./...
go test ./...
```

## Reuse of Phase 5

The Python cartographer at `services/memory-router/cartographer/`
remains the deterministic-extraction reference. We do NOT call it as a
subprocess from this service — every primitive is reimplemented in Go
for production performance. The two are kept consistent via shared
fixtures under `tests/fixtures/`.
