-- revoke_public.sql — deny-by-default scaffold
--
-- Applied last during provisioning. Removes the implicit PUBLIC grants
-- on the crucible_memory schema so a freshly-CREATEd role gets no
-- accidental access.

REVOKE ALL ON SCHEMA crucible_memory FROM PUBLIC;
REVOKE ALL ON ALL TABLES IN SCHEMA crucible_memory FROM PUBLIC;
REVOKE ALL ON ALL SEQUENCES IN SCHEMA crucible_memory FROM PUBLIC;
REVOKE ALL ON ALL FUNCTIONS IN SCHEMA crucible_memory FROM PUBLIC;

ALTER DEFAULT PRIVILEGES IN SCHEMA crucible_memory
    REVOKE ALL ON TABLES FROM PUBLIC;
ALTER DEFAULT PRIVILEGES IN SCHEMA crucible_memory
    REVOKE ALL ON SEQUENCES FROM PUBLIC;
ALTER DEFAULT PRIVILEGES IN SCHEMA crucible_memory
    REVOKE ALL ON FUNCTIONS FROM PUBLIC;
