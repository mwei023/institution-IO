-- Reverse dependency order: lines before their parent report card,
-- everything before the learning_areas/curriculum_rating_bands they
-- reference.
DROP TABLE IF EXISTS learning_area_report_card_lines CASCADE;
DROP TABLE IF EXISTS learning_area_report_cards CASCADE;
DROP TABLE IF EXISTS learning_area_teacher_assignments CASCADE;
DROP TABLE IF EXISTS learning_area_entries CASCADE;
DROP TABLE IF EXISTS learning_area_components CASCADE;
DROP TABLE IF EXISTS curriculum_rating_bands CASCADE;
DROP TABLE IF EXISTS learning_areas CASCADE;

-- This migration (009) is what actually created curriculum_type via
-- ADD COLUMN IF NOT EXISTS — safe to drop it here. The corrected 010
-- only adds a constraint on top of this column, so 010's own down
-- migration just needs to drop that constraint, not the column.
ALTER TABLE institutions DROP COLUMN IF EXISTS curriculum_type;
