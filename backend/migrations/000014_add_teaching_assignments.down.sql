-- Down for 000014: drop curriculum-neutral bridge tables.
-- Note: also drops the teaching_assignment_id column on timetable_periods
-- if it was added by the up migration; guarded with IF EXISTS so a
-- partially-applied 000014 still rolls back cleanly.
ALTER TABLE timetable_periods DROP COLUMN IF EXISTS teaching_assignment_id;
DROP TABLE IF EXISTS teaching_assignments;
DROP TABLE IF EXISTS academic_offerings;
