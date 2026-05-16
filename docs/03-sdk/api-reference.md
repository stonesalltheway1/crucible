---
title: API reference
description: Auto-generated reference for the Crucible REST + gRPC + MCP surfaces.
---

# API reference

The Crucible API is generated from `libs/twin-spec/proto/*.proto`. The
canonical surfaces are gRPC, REST (gRPC-gateway), and MCP. All three
share the same wire types.

## Stability

The protobuf and OpenAPI files are version-pinned in releases. The
v1 API is **frozen at GA** for 12 months. Breaking changes go to v2;
we document the migration path 90 days in advance.

## Surfaces

| Surface | Purpose |
|---|---|
| REST `/v1/...` | Web console + GitHub App + Slack bot + integrations |
| gRPC `crucible.v1.*` | SDK clients (Go / TS / Python / Rust) |
| MCP `crucible-mcp` | IDE plugin transport (VS Code / JetBrains / Zed via ACP) |
| Webhooks | event push to customer endpoints |

The Mintlify build runs `buf generate --template buf.gen.openapi.yaml`
on the protos and renders the OpenAPI under
[api-reference](/api-reference) here. See
[`libs/twin-spec/proto/`](https://github.com/crucible/crucible/tree/main/libs/twin-spec/proto)
for source.

## Quick examples

```bash
# Submit a task
curl -X POST https://api.crucible.dev/v1/tasks \
    -H "Authorization: Bearer $CRUCIBLE_TOKEN" \
    -H "Content-Type: application/json" \
    -d '{"repo": "acme/payments", "description": "Add idempotency key…"}'

# Stream task events
curl -N https://api.crucible.dev/v1/tasks/$ID/events \
    -H "Authorization: Bearer $CRUCIBLE_TOKEN"

# Verify an attestation
curl https://api.crucible.dev/v1/attestations/$REKOR_UUID/verify
```
