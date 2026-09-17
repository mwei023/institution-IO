-- 000015_timetable_engine DOWN.
-- NOTE: the day_of_week restore assumes no Sat/Sun (6/7) periods exist;
-- delete or move those rows before rolling back.
ALTER TABLE timetable_periods
    DROP CONSTRAINT timetable_periods_day_of_week_check;
ALTER TABLE timetable_periods
    ADD CONSTRAINT timetable_periods_day_of_week_check
    CHECK (day_of_week >= 1 AND day_of_week <= 5);

DROP INDEX idx_timetable_periods_schedule_slot;
ALTER TABLE timetable_periods DROP COLUMN schedule_slot_id;

DROP TABLE lesson_requirements;
DROP TABLE schedule_slots;
DROP TABLE schedule_profiles;
