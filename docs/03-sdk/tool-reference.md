# MCP Tool Reference

Crucible exposes its agent SDK as MCP tools (and Agent Client Protocol tools where applicable), so any MCP-compatible agent host can drive a Crucible task.

The MCP server is `crucible-mcp`. The tool definitions below are surfaced to the host LLM via the MCP protocol.

## Tool surface

### File operations

#### `twin_fs_read`

```json
{
  "name": "twin_fs_read",
  "description": "Read a file from the twin filesystem (overlayfs upper merged with base SHA).",
  "input_schema": {
    "type": "object",
    "properties": {
      "path": { "type": "string", "description": "File path relative to repo root" }
    },
    "required": ["path"]
  }
}
```

#### `twin_fs_write`

```json
{
  "name": "twin_fs_write",
  "description": "Write content to a file in the twin. Creates if missing. Returns a signed write attestation.",
  "input_schema": {
    "type": "object",
    "properties": {
      "path": { "type": "string" },
      "content": { "type": "string" }
    },
    "required": ["path", "content"]
  }
}
```

#### `twin_fs_delete`, `twin_fs_list`, `twin_fs_diff`

Schemas mirror the SDK methods in [agent-sdk-reference.md](agent-sdk-reference.md).

### Database operations

#### `twin_db_query`

```json
{
  "name": "twin_db_query",
  "description": "Execute SQL against the twin's Neon branch. Returns rows + column metadata. Use parametrized queries.",
  "input_schema": {
    "type": "object",
    "properties": {
      "sql": { "type": "string" },
      "params": { "type": "array", "items": {} }
    },
    "required": ["sql"]
  }
}
```

#### `twin_db_migrate`

```json
{
  "name": "twin_db_migrate",
  "description": "Apply a migration file to the twin DB. Returns a MigrationProposal (twin-scoped, auto-approved for twin) or rejection with schema-diff impact analysis.",
  "input_schema": {
    "type": "object",
    "properties": {
      "file": { "type": "string", "description": "Path to migration file in the repo" }
    },
    "required": ["file"]
  }
}
```

### Service operations

#### `twin_svc_call`

```json
{
  "name": "twin_svc_call",
  "description": "Call an external service. Returns a response carrying X-Crucible-Tape header indicating whether the response was replayed from tape, synthesized from schema, or live. Mutating calls without live-allowed go to deterministic stubs.",
  "input_schema": {
    "type": "object",
    "properties": {
      "service": { "type": "string", "description": "Service name configured in task manifest" },
      "endpoint": { "type": "string", "description": "Path + query, e.g. '/v1/charges'" },
      "method": { "type": "string", "enum": ["GET","POST","PUT","PATCH","DELETE","OPTIONS","HEAD"] },
      "payload": {}, 
      "headers": { "type": "object" }
    },
    "required": ["service", "endpoint"]
  }
}
```

### Secret access

#### `twin_secret_use`

```json
{
  "name": "twin_secret_use",
  "description": "Reference a secret by name in a service call. The value is never returned to the agent; it is injected at the egress proxy when the call fires.",
  "input_schema": {
    "type": "object",
    "properties": {
      "name": { "type": "string" }
    },
    "required": ["name"]
  }
}
```

Note: there is intentionally no `twin_secret_get` MCP tool. The SDK has `twin.secret.get` returning an opaque `SecretRef`, but the MCP surface goes one step further — agents simply reference secrets by name and the substitution happens server-side.

### Shell

#### `twin_shell_exec`

```json
{
  "name": "twin_shell_exec",
  "description": "Run a shell command inside the twin sandbox. Destructive commands are intercepted and require explicit approval. Returns ExecResult or DestructiveProposal.",
  "input_schema": {
    "type": "object",
    "properties": {
      "cmd": { "type": "string" },
      "cwd": { "type": "string" },
      "env": { "type": "object" },
      "timeout_sec": { "type": "integer" }
    },
    "required": ["cmd"]
  }
}
```

#### `twin_shell_approve_destructive`

```json
{
  "name": "twin_shell_approve_destructive",
  "description": "Approve a DestructiveProposal from a prior twin_shell_exec call. Twin-scoped destructives auto-execute; real-scoped require human approval via the Promotion Contract.",
  "input_schema": {
    "type": "object",
    "properties": {
      "proposal_id": { "type": "string" },
      "justification": { "type": "string" }
    },
    "required": ["proposal_id", "justification"]
  }
}
```

### Tests

#### `twin_test_run`, `twin_test_run_mutation`, `twin_test_run_property`, `twin_test_run_fuzz`

Same shape as SDK methods.

### Verifier

#### `twin_verify_bundle`

```json
{
  "name": "twin_verify_bundle",
  "description": "Run the appropriate verifier tier ladder on the current task state. Returns VerifierApproval or VerifierRejection with structured failure reasons.",
  "input_schema": { "type": "object", "properties": {} }
}
```

The host LLM typically calls this once near the end of a task. Tier selection is automatic based on the critical-path classifier; agents may explicitly invoke `twin_verify_tier3` to escalate.

### Memory

#### `twin_memory_recall`

```json
{
  "name": "twin_memory_recall",
  "description": "Retrieve relevant memory (conventions, prior decisions, code snippets) for a query. Returns up to 7K tokens, importance-ranked.",
  "input_schema": {
    "type": "object",
    "properties": {
      "query": { "type": "string" },
      "scope": {
        "type": "object",
        "properties": {
          "repo": { "type": "string" },
          "file_glob": { "type": "string" },
          "category": { "type": "string" }
        }
      }
    },
    "required": ["query"]
  }
}
```

#### `twin_memory_note`, `twin_memory_conventions`, `twin_memory_check_compliance`

Schemas mirror the SDK.

### Plan + promotion

#### `twin_plan_propose`

```json
{
  "name": "twin_plan_propose",
  "description": "Submit a plan for user approval. Plan must include cost estimate, file impact, top risks, and retry budget. Blocks until user approves, edits, or rejects.",
  "input_schema": { "$ref": "schemas/Plan.json" }
}
```

#### `twin_plan_check_budget`

Returns current spend vs cap.

#### `twin_promote`

```json
{
  "name": "twin_promote",
  "description": "Submit a verified PromotionBundle to the Promotion Contract. Returns PromotionId. Status is queryable via twin_promote_status.",
  "input_schema": { "$ref": "schemas/PromotionBundle.json" }
}
```

## ACP (Agent Client Protocol) compatibility

For Zed and any other ACP-compatible editor, Crucible exposes the same tool set via the ACP `tools/list` and `tools/call` methods. The schemas are identical to MCP. This lets a Zed user run Crucible as their primary agent backend without writing a custom adapter.

## Authentication

MCP host authenticates to Crucible via either:

- **OAuth 2.0 + PKCE** (default for IDE plugins and CLI)
- **API token** (for CI / scripts; tenant-scoped, revocable)
- **Sigstore keyless OIDC** (for GitHub Actions, etc., where the runner has an OIDC token)

Tokens are bound to a specific tenant + workspace. Agents cannot access other tenants' state.

## Permissions model

Each tool call goes through a per-tenant authorization check. Defaults:

| Tool group | Default authorization |
|---|---|
| `twin_fs_*`, `twin_memory_recall`, `twin_plan_check_budget` | Always allowed |
| `twin_db_*`, `twin_svc_call`, `twin_test_*`, `twin_verify_*` | Allowed if task is in `active` state |
| `twin_shell_exec` | Allowed; destructive ops require approve flow |
| `twin_shell_approve_destructive` (real-scoped) | Requires human signature via Promotion Contract |
| `twin_memory_note` | Allowed; subject to LLM-judge filter |
| `twin_plan_propose` | Allowed; blocks for approval |
| `twin_promote` | Allowed; Promotion Contract evaluates policy |

Tenants can lock down further (e.g., disable `twin_shell_exec` entirely for repos that don't need it; cap `twin_db_migrate` to specific paths).

## Versioning

Tool schemas are versioned via the MCP `meta.version` field. Currently `2026.06`. Breaking changes bump the schema version + 90-day deprecation window. Hosts that don't advertise the new version receive the old schema.

## Discovery

Hosts call `tools/list` to discover available tools per their authorization. The list is filtered by tenant capabilities (e.g., `twin_verify_tier3` only appears if the tenant's tier includes formal verification).

## Examples

See the `examples/` directory in the repo:

- `examples/cursor-mcp/` — Cursor as the MCP host driving Crucible
- `examples/claude-desktop/` — Claude Desktop as the host
- `examples/zed-acp/` — Zed via ACP
- `examples/github-actions/` — CI-driven agent flow
- `examples/cli-direct/` — `crucible` CLI as the host
