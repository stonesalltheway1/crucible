-- 0004_memory_semantic.sql — semantic store for "retrieved snippets"
--
-- Distinct from episodic to enable different TTL + retrieval weighting.
-- Episodic = "we did X"; semantic = "the codebase looks like Y".

BEGIN;
SET search_path = crucible_memory, public;

CREATE TABLE IF NOT EXISTS memory_semantic (
    memory_id        text PRIMARY KEY DEFAULT ('mem_' || encode(gen_random_bytes(20), 'hex')),
    tenant_id        text NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
    repo_id          text NOT NULL DEFAULT '',
    file_path        text NOT NULL DEFAULT '',
    snippet          text NOT NULL,
    snippet_hash     text NOT NULL,
    token_estimate   int  NOT NULL,
    embedding        halfvec(3072) NOT NULL,
    importance       real NOT NULL DEFAULT 0.5,
    written_at       timestamptz NOT NULL DEFAULT now(),
    last_recalled    timestamptz NOT NULL DEFAULT now(),
    recall_count     int  NOT NULL DEFAULT 0,
    expires_at       timestamptz NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_semantic_tenant_hash
    ON memory_semantic (tenant_id, repo_id, snippet_hash);

CREATE INDEX IF NOT EXISTS idx_semantic_tenant_recency
    ON memory_semantic (tenant_id, repo_id, last_recalled DESC);

CREATE INDEX IF NOT EXISTS idx_semantic_expires
    ON memory_semantic (expires_at)
    WHERE importance < 0.7;

COMMIT;
