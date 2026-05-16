-- 0003_memory_episodic.sql — episodic memory (cross-task agent observations)
--
-- 30..90 day TTL, importance-weighted. The router queries this table
-- alongside semantic recall + procedural-graph lookup on every recall().
--
-- Embedding dimension: 3072 (matches Anthropic text-embedding-3-large
-- and OpenAI text-embedding-3-large). Stored as halfvec to halve disk
-- footprint; pgvector 0.9 supports halfvec(3072) in DiskANN.

BEGIN;
SET search_path = crucible_memory, public;

CREATE TABLE IF NOT EXISTS memory_episodic (
    memory_id        text PRIMARY KEY DEFAULT ('mem_' || encode(gen_random_bytes(20), 'hex')),
    tenant_id        text NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
    repo_id          text NOT NULL DEFAULT '',  -- '' = org-scoped
    task_id          text,
    content          text NOT NULL,
    -- 7000-token-budget logic re-counts on the fly via tiktoken; stored
    -- approximation here is len(content)/4 for fast cheap filtering.
    token_estimate   int  NOT NULL,
    embedding        halfvec(3072) NOT NULL,
    importance       real NOT NULL DEFAULT 0.5,  -- A-MAC composite
    kind             text NOT NULL CHECK (kind IN ('episodic', 'semantic')),
    source_kind      text,                       -- 'pr_comment' | 'incident' | 'adr' | 'agent_observation'
    source_payload   jsonb,                      -- typed SourceRef payload
    written_at       timestamptz NOT NULL DEFAULT now(),
    last_recalled    timestamptz NOT NULL DEFAULT now(),
    recall_count     int  NOT NULL DEFAULT 0,
    -- The expiry field is set at admission time to written_at + TTL.
    -- A nightly job drops rows past expiry; importance > 0.7 stays
    -- forever via the reinforce-on-access rule.
    expires_at       timestamptz NOT NULL
);

-- Per-tenant partial index keeps the per-tenant working set hot in cache.
CREATE INDEX IF NOT EXISTS idx_episodic_tenant_recency
    ON memory_episodic (tenant_id, repo_id, last_recalled DESC);

CREATE INDEX IF NOT EXISTS idx_episodic_expires
    ON memory_episodic (expires_at)
    WHERE importance < 0.7;

-- Source-payload GIN for "find all rows referencing pr=1234" lookups.
CREATE INDEX IF NOT EXISTS idx_episodic_source_payload
    ON memory_episodic USING gin (source_payload jsonb_path_ops);

COMMIT;
