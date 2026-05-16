// graph_init.cypher — per-tenant procedural-memory graph bootstrap
//
// FalkorDB runs as a Redis module. Each tenant gets a NAMED GRAPH whose
// name is `tenant_<tenant_id>` (sanitized). The memory-router queries
// `GRAPH.QUERY tenant_<id> "...."` — one graph per tenant gives us
// per-tenant isolation at the storage layer in addition to RLS in the
// Postgres mirror.
//
// global_defaults rules are mirrored into a special `tenant_global` graph
// that the router reads alongside the active tenant's graph when an
// AGENTS.md / repo override doesn't already cover a category.
//
// Bi-temporal edges follow the Graphiti pattern:
//   valid_from / valid_to  = when the fact was true in the world
//   recorded_at            = when we observed it (write time)

// ────────────────────────────────────────────────────────────────────────
// Node label conventions
//   :Convention        — the procedural rule (Phase-5 unit)
//   :SourceRef         — pr_comment | incident | adr | agent_observation
//   :Incident         — first-class for the trigger → action → outcome chain
//   :ADR              — first-class for decision-record traversal
//   :File              — referenced from positive/negative examples
// ────────────────────────────────────────────────────────────────────────

// Run once on graph creation. memoryrouter.proceduralstore.Init invokes
// this against the tenant's graph as part of provisioning.

// Marker node so the router can detect a freshly-initialised graph and
// trigger a global_defaults pull from the seed bundles.
MERGE (g:GraphMeta {key: 'init'})
  ON CREATE SET g.schema_version = 1,
                g.initialized_at = timestamp(),
                g.bi_temporal = true
  ON MATCH  SET g.last_seen = timestamp();

// A tombstone node referenced when a convention is superseded but its
// edges need a terminal target. Cypher doesn't allow null endpoints;
// this stand-in keeps SUPERSEDED_BY traversals well-formed.
MERGE (t:Tombstone {id: 'tombstone'})
  ON CREATE SET t.created_at = timestamp();

RETURN 'ok' AS status;
