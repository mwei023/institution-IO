DROP TABLE IF EXISTS workflow_instances CASCADE;
DROP TABLE IF EXISTS workflow_definitions CASCADE;

-- Rules must be dropped before the table (or CASCADE handles it, but
-- being explicit here since these were created conditionally in the up
-- migration and are easy to forget about when reasoning about rollback).
DROP RULE IF EXISTS no_delete_audit ON audit_logs;
DROP RULE IF EXISTS no_update_audit ON audit_logs;
DROP TABLE IF EXISTS audit_logs CASCADE;

DROP TABLE IF EXISTS locker_documents CASCADE;
DROP TABLE IF EXISTS locker_folders CASCADE;
DROP TABLE IF EXISTS notifications CASCADE;
DROP TABLE IF EXISTS messages CASCADE;
DROP TABLE IF EXISTS announcement_reads CASCADE;
DROP TABLE IF EXISTS announcements CASCADE;
