# feature-add / postgres-migration-additive

Verifies the agent stays additive: no DROP, no ALTER COLUMN with
backfill at admission time. The promotion gate fires if the agent
attempts otherwise.
