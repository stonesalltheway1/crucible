-- 0008_federation_graduation.sql — federation graduation data model
--
-- Phase 5 wires the data model only; the graduation engine fires in
-- v2 Phase 10. We persist eligibility records as they're computed so
-- the rolling tenant-count is materialized; the engine then becomes a
-- straightforward "promote rows where fired=false AND tenant_count >= 5
-- AND tenant of every contributor consents".

BEGIN;
SET search_path = crucible_memory, public;

CREATE TABLE IF NOT EXISTS federation_graduations (
    anonymized_rule_id    text PRIMARY KEY,
    category              text NOT NULL,
    canonical_form_nl     text NOT NULL,
    distinct_tenant_count int  NOT NULL,
    contributing_convention_ids text[] NOT NULL DEFAULT ARRAY[]::text[],
    eligible_at           timestamptz NOT NULL DEFAULT now(),
    fired                 boolean NOT NULL DEFAULT FALSE,
    promoted_to_layer     text,
    -- Tenant consent — each contributing tenant's federation_optout
    -- value at eligibility time; we re-check on graduation in v2.
    consent_snapshot      jsonb NOT NULL DEFAULT '{}'::jsonb
);

CREATE INDEX IF NOT EXISTS idx_fedgrad_eligible
    ON federation_graduations (eligible_at DESC)
    WHERE fired = FALSE;

-- Read-only to tenants; full access to bootstrap (the engine).
ALTER TABLE federation_graduations ENABLE ROW LEVEL SECURITY;
ALTER TABLE federation_graduations FORCE ROW LEVEL SECURITY;

-- No per-tenant policy: federation_graduations is anonymized aggregate
-- data and only the bootstrap role (running the v2 engine) reads it.
GRANT SELECT, INSERT, UPDATE ON federation_graduations TO crucible_bootstrap;
GRANT SELECT ON federation_graduations TO crucible_router_service;

COMMIT;
