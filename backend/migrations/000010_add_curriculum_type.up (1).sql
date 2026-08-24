-- CORRECTED (see session notes): the original version of this migration
-- tried to ADD COLUMN curriculum_type, but migration 009 already adds
-- that column (with IF NOT EXISTS). Running both in sequence on a fresh
-- database fails with "column already exists" at this migration.
--
-- This corrected version only adds the CHECK constraint 009 didn't
-- include, and does so idempotently so it's safe to run regardless of
-- whether the constraint is already present on a given database.
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'chk_institutions_curriculum_type'
          AND conrelid = 'institutions'::regclass
    ) THEN
        ALTER TABLE institutions
          ADD CONSTRAINT chk_institutions_curriculum_type
          CHECK (curriculum_type IN ('844', 'cbc'));
    END IF;
END$$;
