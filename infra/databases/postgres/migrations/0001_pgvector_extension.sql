-- 0001_pgvector_extension.sql — enable pgvector
--
-- Requires Postgres 16+. The `vector` and `halfvec` types ship with
-- pgvector 0.9+. Crucible self-host install asserts the version at boot.
--
-- Twin-first promotion path: this migration is applied via
-- twin.db.migrate against the Crucible-owned Postgres branch before
-- promotion to real.

BEGIN;

CREATE EXTENSION IF NOT EXISTS vector;

-- vector_diskann is gated behind pgvector 0.9 + the diskann extension.
-- The migration succeeds on 0.8 (drops to HNSW); failure here would
-- block the entire memory layer at fresh install.
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_extension WHERE extname = 'vector'
    ) THEN
        RAISE EXCEPTION 'pgvector required';
    END IF;
END$$;

-- Required for ULID / convention_id generation when callers omit it.
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- citext for case-insensitive scope.file_glob comparisons.
CREATE EXTENSION IF NOT EXISTS citext;

COMMIT;
