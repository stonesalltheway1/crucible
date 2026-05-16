-- hnsw.sql — HNSW indexes for the < 10M vector tier
--
-- Default at fresh tenant provision. Switches to DiskANN automatically
-- when the maintenance job detects > 10M rows per (tenant_id, kind).

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_episodic_embedding_hnsw
    ON crucible_memory.memory_episodic
    USING hnsw (embedding halfvec_cosine_ops)
    WITH (m = 16, ef_construction = 64);

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_semantic_embedding_hnsw
    ON crucible_memory.memory_semantic
    USING hnsw (embedding halfvec_cosine_ops)
    WITH (m = 16, ef_construction = 64);
