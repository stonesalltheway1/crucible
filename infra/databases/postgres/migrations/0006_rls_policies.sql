-- 0006_rls_policies.sql — Row-Level Security for tenant_id + repo_id
--
-- Hard requirement from ADR-003: Customer A's memory never leaks to
-- Customer B's agent. Defense in depth — even if a memory-router bug
-- forgets to scope a query, the database refuses to return rows the
-- caller's role isn't allowed to see.
--
-- Pattern: every per-tenant connection sets `SET ROLE
-- crucible_tenant_<tenant_id>` at acquire time. RLS USING expressions
-- read current_user via a stable function. global_defaults rows are
-- readable by every tenant role but writable only by the bootstrap role.

BEGIN;
SET search_path = crucible_memory, public;

-- Helper: extracts the tenant_id portion of the current role.
-- 'crucible_tenant_ten_abc' -> 'ten_abc'. STABLE so the planner caches.
CREATE OR REPLACE FUNCTION crucible_memory.current_tenant_id()
RETURNS text
LANGUAGE sql
STABLE
AS $$
    SELECT CASE
        WHEN current_user LIKE 'crucible_tenant_%'
        THEN substring(current_user FROM length('crucible_tenant_') + 1)
        ELSE NULL
    END
$$;

-- The bootstrap role is the only role allowed to write global_defaults.
-- All tenant roles can read global_defaults rows; never write.
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'crucible_bootstrap') THEN
        CREATE ROLE crucible_bootstrap NOLOGIN;
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'crucible_router_service') THEN
        CREATE ROLE crucible_router_service NOLOGIN;
    END IF;
END$$;

GRANT USAGE ON SCHEMA crucible_memory TO crucible_bootstrap, crucible_router_service;

-- ─────────────────────────────────────────────────────────────────────
-- conventions
-- ─────────────────────────────────────────────────────────────────────

ALTER TABLE conventions ENABLE ROW LEVEL SECURITY;
ALTER TABLE conventions FORCE ROW LEVEL SECURITY;

CREATE POLICY conv_read_own ON conventions
    FOR SELECT
    USING (
        tenant_id = crucible_memory.current_tenant_id()
        OR (layer = 'global_defaults' AND tenant_id = 'global')
    );

-- Tenants can write rows for themselves only — never to global_defaults.
CREATE POLICY conv_write_own ON conventions
    FOR INSERT
    WITH CHECK (
        tenant_id = crucible_memory.current_tenant_id()
        AND layer IN ('org_overrides', 'repo_overrides')
    );

CREATE POLICY conv_update_own ON conventions
    FOR UPDATE
    USING (tenant_id = crucible_memory.current_tenant_id())
    WITH CHECK (
        tenant_id = crucible_memory.current_tenant_id()
        AND layer IN ('org_overrides', 'repo_overrides')
    );

-- Bootstrap role can write to global_defaults.
CREATE POLICY conv_bootstrap_write ON conventions
    FOR ALL
    TO crucible_bootstrap
    USING (true)
    WITH CHECK (tenant_id = 'global' AND layer = 'global_defaults');

-- ─────────────────────────────────────────────────────────────────────
-- memory_episodic
-- ─────────────────────────────────────────────────────────────────────

ALTER TABLE memory_episodic ENABLE ROW LEVEL SECURITY;
ALTER TABLE memory_episodic FORCE ROW LEVEL SECURITY;

CREATE POLICY ep_read_own ON memory_episodic
    FOR SELECT
    USING (tenant_id = crucible_memory.current_tenant_id());

CREATE POLICY ep_write_own ON memory_episodic
    FOR INSERT
    WITH CHECK (tenant_id = crucible_memory.current_tenant_id());

CREATE POLICY ep_update_own ON memory_episodic
    FOR UPDATE
    USING (tenant_id = crucible_memory.current_tenant_id())
    WITH CHECK (tenant_id = crucible_memory.current_tenant_id());

CREATE POLICY ep_delete_own ON memory_episodic
    FOR DELETE
    USING (tenant_id = crucible_memory.current_tenant_id());

-- ─────────────────────────────────────────────────────────────────────
-- memory_semantic
-- ─────────────────────────────────────────────────────────────────────

ALTER TABLE memory_semantic ENABLE ROW LEVEL SECURITY;
ALTER TABLE memory_semantic FORCE ROW LEVEL SECURITY;

CREATE POLICY sem_read_own ON memory_semantic
    FOR SELECT
    USING (tenant_id = crucible_memory.current_tenant_id());

CREATE POLICY sem_write_own ON memory_semantic
    FOR INSERT
    WITH CHECK (tenant_id = crucible_memory.current_tenant_id());

CREATE POLICY sem_update_own ON memory_semantic
    FOR UPDATE
    USING (tenant_id = crucible_memory.current_tenant_id())
    WITH CHECK (tenant_id = crucible_memory.current_tenant_id());

-- ─────────────────────────────────────────────────────────────────────
-- The router-service role bypasses RLS at the gateway layer ONLY for
-- the maintenance GC + drift jobs. It cannot serve customer reads;
-- customer-facing connections always re-SET ROLE to crucible_tenant_*.
-- ─────────────────────────────────────────────────────────────────────

GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA crucible_memory TO crucible_router_service;

-- The router_service role is BYPASS RLS — but production deployment
-- pins it to only the GC + drift jobs via PgBouncer config. Customer
-- requests never get this role.
ALTER ROLE crucible_router_service BYPASSRLS;

COMMIT;
