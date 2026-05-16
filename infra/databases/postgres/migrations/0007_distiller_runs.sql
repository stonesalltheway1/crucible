-- 0007_distiller_runs.sql — distiller job audit trail + quarantine log
--
-- The distiller writes one row per processed item with its judge verdict.
-- This is the audit trail the brief calls out for the LLM-as-judge
-- filter; auditors can verify the catch rate against the adversarial
-- corpus offline.

BEGIN;
SET search_path = crucible_memory, public;

CREATE TABLE IF NOT EXISTS distiller_runs (
    run_id           text PRIMARY KEY DEFAULT ('drun_' || encode(gen_random_bytes(20), 'hex')),
    tenant_id        text NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
    repo             text NOT NULL DEFAULT '',
    job_id           text NOT NULL,
    source_channel   text NOT NULL,
    extractor_model  text NOT NULL,
    judge_model      text NOT NULL,
    candidates_total int NOT NULL DEFAULT 0,
    quarantined      int NOT NULL DEFAULT 0,
    admitted         int NOT NULL DEFAULT 0,
    cost_usd         numeric(10,4) NOT NULL DEFAULT 0,
    latency_ms       int NOT NULL DEFAULT 0,
    processed_at     timestamptz NOT NULL DEFAULT now(),
    -- Each candidate's verdict, indexed by candidate_id within the
    -- aggregated JSON. Replays the run for offline audit.
    judge_verdicts   jsonb NOT NULL DEFAULT '[]'::jsonb
);

CREATE INDEX IF NOT EXISTS idx_distiller_tenant_at
    ON distiller_runs (tenant_id, processed_at DESC);

CREATE INDEX IF NOT EXISTS idx_distiller_quarantined
    ON distiller_runs (processed_at DESC)
    WHERE quarantined > 0;

-- RLS: tenants see their own runs; the bootstrap role can audit across
-- all tenants for the catch-rate metrics.
ALTER TABLE distiller_runs ENABLE ROW LEVEL SECURITY;
ALTER TABLE distiller_runs FORCE ROW LEVEL SECURITY;

CREATE POLICY drun_read_own ON distiller_runs
    FOR SELECT
    USING (tenant_id = crucible_memory.current_tenant_id());

CREATE POLICY drun_router_write ON distiller_runs
    FOR INSERT
    TO crucible_router_service
    WITH CHECK (true);

COMMIT;
