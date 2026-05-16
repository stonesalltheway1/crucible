-- diskann.sql — DiskANN index for the 10M+ vector tier
--
-- Applied per-tenant when the tenant's row count crosses 10M. Smaller
-- tenants stay on HNSW (hnsw.sql); DiskANN's setup cost is amortized
-- by lower RAM footprint and predictable latency above 10M.
--
-- pgvector 0.9+ ships DiskANN under the vector_diskann extension. The
-- index is created CONCURRENTLY so reads stay live during build.

-- Episodic + semantic share the same dimension; both get DiskANN.

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_episodic_embedding_diskann
    ON crucible_memory.memory_episodic
    USING diskann (embedding halfvec_cosine_ops)
    WITH (max_neighbors = 50, l_build = 100);

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_semantic_embedding_diskann
    ON crucible_memory.memory_semantic
    USING diskann (embedding halfvec_cosine_ops)
    WITH (max_neighbors = 50, l_build = 100);
