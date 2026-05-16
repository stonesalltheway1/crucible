-- 0002_tenant_table.sql — tenant registry + per-tenant DB roles
--
-- Every memory-router connection holds one of these roles via SET ROLE
-- at connection-acquire time. RLS policies key off current_user against
-- the tenant_id column, so role names are load-bearing.

BEGIN;

CREATE SCHEMA IF NOT EXISTS crucible_memory;
SET search_path = crucible_memory, public;

-- tenants is read-mostly. The memory-router caches the active set in
-- process; provisioning a new tenant fires a NOTIFY.
CREATE TABLE IF NOT EXISTS tenants (
    tenant_id     text PRIMARY KEY,
    display_name  text NOT NULL,
    plan_tier     text NOT NULL CHECK (plan_tier IN ('pro', 'team', 'outcome', 'enterprise')),
    provisioned_at timestamptz NOT NULL DEFAULT now(),
    -- Federation graduation eligibility: an org can opt out of cross-tenant
    -- generalization-upward entirely. Default is opted-in (consent comes
    -- from the terms of service); opt-out at any time.
    federation_optout boolean NOT NULL DEFAULT FALSE
);

-- The fixed "global" pseudo-tenant owns global_defaults rows. RLS
-- treats it specially: every tenant can read; only the bootstrap loader
-- can write.
INSERT INTO tenants (tenant_id, display_name, plan_tier)
VALUES ('global', '__global_defaults__', 'enterprise')
ON CONFLICT (tenant_id) DO NOTHING;

-- Per-tenant role creation lives in a helper function. The provisioning
-- pipeline calls this once per tenant; idempotent.
CREATE OR REPLACE FUNCTION crucible_memory.provision_tenant_role(p_tenant_id text)
RETURNS void
LANGUAGE plpgsql
AS $$
DECLARE
    role_name text := 'crucible_tenant_' || regexp_replace(p_tenant_id, '[^a-z0-9_]', '_', 'g');
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = role_name) THEN
        EXECUTE format('CREATE ROLE %I NOLOGIN', role_name);
        EXECUTE format('GRANT USAGE ON SCHEMA crucible_memory TO %I', role_name);
        EXECUTE format('GRANT SELECT, INSERT, UPDATE ON ALL TABLES IN SCHEMA crucible_memory TO %I', role_name);
        EXECUTE format('GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA crucible_memory TO %I', role_name);
    END IF;
END;
$$;

COMMIT;
