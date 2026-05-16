-- set_role.sql — per-connection RLS role pattern
--
-- The memory-router calls this at connection-acquire time. The role name
-- is computed from the request's tenant_id, sanitised, and pinned for
-- the lifetime of the connection — no DROP ROLE / RESET ROLE inside a
-- transaction.
--
-- This file is reference only; the router runs the equivalent via a
-- prepared statement to avoid SQL-injection on the tenant_id input.

-- Example flow:
--
--   conn = pool.acquire()
--   conn.execute("SET ROLE 'crucible_tenant_' || $1", [tenant_id])
--   ... query ...
--   conn.execute("RESET ROLE")
--   pool.release(conn)

-- The router-service role's connection that runs the GC + drift jobs
-- skips SET ROLE since it has BYPASSRLS for those maintenance paths.
