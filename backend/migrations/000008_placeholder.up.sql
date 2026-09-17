-- Placeholder to close the 000007 -> 000009 gap.
-- No 000008 migration ever existed on disk; the database was migrated
-- manually past this number. This no-op keeps sequential runners
-- (golang-migrate and scripts/migrate.sh) working without implying
-- any schema change.
SELECT 1;
