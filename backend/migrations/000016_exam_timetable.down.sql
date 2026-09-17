-- Down migration for 000016_exam_timetable (review only).
DROP TABLE IF EXISTS exam_special_arrangements;
DROP TABLE IF EXISTS exam_invigilators;
DROP TABLE IF EXISTS exam_sittings;
DROP TABLE IF EXISTS exam_paper_groups;
DROP TABLE IF EXISTS exam_papers;
DROP TABLE IF EXISTS exam_sessions;
