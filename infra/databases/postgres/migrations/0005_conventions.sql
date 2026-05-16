-- 0005_conventions.sql — procedural-memory canonical store (Postgres mirror)
--
-- The authoritative procedural-memory graph lives in FalkorDB. This table
-- is the Postgres mirror used for:
--   - Bulk reads when the router fast-paths a non-graph query
--   - Cross-tenant federation graduation analytics
--   - The drift-detector's 30-day rolling-ratio query
--   - The cartographer's repo_overrides write path
--
-- Writes go FalkorDB-first; the mirror row is upserted in the same
-- transaction. The router reads from Postgres for "list all active
-- conventions in this scope" (fast, no Cypher) and from FalkorDB for
-- multi-hop traversals.

BEGIN;
SET search_path = crucible_memory, public;

-- ENUMs are declared as CHECK constraints (not Postgres ENUM types) so
-- adding new taxonomy buckets is a non-locking ALTER.
CREATE TABLE IF NOT EXISTS conventions (
    convention_id       text PRIMARY KEY,
    tenant_id           text NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
    layer               text NOT NULL CHECK (layer IN ('global_defaults', 'org_overrides', 'repo_overrides')),
    scope_repo          text NOT NULL DEFAULT '',
    scope_file_glob     citext NOT NULL DEFAULT '',
    scope_category      text NOT NULL DEFAULT '',
    rule_nl             text NOT NULL CHECK (char_length(rule_nl) BETWEEN 1 AND 1024),
    rule_machine        text,
    category            text NOT NULL CHECK (category IN (
        'Naming', 'Layering', 'LibraryPreferences', 'TestPatterns',
        'ErrorHandling', 'Logging', 'MigrationPatterns', 'PrCommitHygiene',
        'SecurityDefaults', 'PerformanceDefaults', 'Concurrency', 'ApiShape'
    )),
    status              text NOT NULL CHECK (status IN (
        'active', 'drifting', 'superseded', 'rejected', 'candidate', 'suggested'
    )),
    confidence          real NOT NULL CHECK (confidence BETWEEN 0 AND 1),
    judge_score         real NOT NULL DEFAULT 0 CHECK (judge_score BETWEEN 0 AND 1),
    judge_rationale     text NOT NULL DEFAULT '',
    source_evidence     jsonb NOT NULL DEFAULT '[]'::jsonb,
    positive_examples   jsonb NOT NULL DEFAULT '[]'::jsonb,
    negative_examples   jsonb NOT NULL DEFAULT '[]'::jsonb,
    first_seen          timestamptz NOT NULL,
    last_reinforced     timestamptz NOT NULL DEFAULT now(),
    last_violated       timestamptz,
    valid_from          timestamptz NOT NULL,
    valid_to            timestamptz,
    supersedes          text[] NOT NULL DEFAULT ARRAY[]::text[],
    writer_oidc_subject text NOT NULL DEFAULT '',
    written_at          timestamptz NOT NULL DEFAULT now(),
    stack_tag           text NOT NULL DEFAULT '',
    anonymized_form     text NOT NULL DEFAULT '',
    -- A-MAC importance composite, cached for fast re-rank.
    importance          real NOT NULL DEFAULT 0.5,
    -- Counters used by the drift detector.
    positives_30d       int NOT NULL DEFAULT 0,
    negatives_30d       int NOT NULL DEFAULT 0
);

-- Scope queries are the hot path.
CREATE INDEX IF NOT EXISTS idx_conv_tenant_scope
    ON conventions (tenant_id, scope_repo, scope_category, status);

CREATE INDEX IF NOT EXISTS idx_conv_tenant_file_glob
    ON conventions (tenant_id, scope_file_glob)
    WHERE status = 'active';

-- Global-defaults is read by every tenant on every cold-start; partial
-- index on the global pseudo-tenant keeps it page-cached.
CREATE INDEX IF NOT EXISTS idx_conv_global_active
    ON conventions (stack_tag, scope_file_glob)
    WHERE layer = 'global_defaults' AND status = 'active';

-- Drift detector queries this every night.
CREATE INDEX IF NOT EXISTS idx_conv_drift_candidates
    ON conventions (last_reinforced)
    WHERE status = 'active';

-- The candidate-bucket layer queried only when the customer reviews
-- "suggestions". Partial keeps it small.
CREATE INDEX IF NOT EXISTS idx_conv_suggested
    ON conventions (tenant_id)
    WHERE status IN ('suggested', 'candidate');

COMMIT;
