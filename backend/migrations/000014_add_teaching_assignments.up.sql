-- Step 1 of 2 toward a curriculum-neutral timetable. Deliberately
-- non-destructive: adds new tables and one new nullable column, backfills
-- everything, but changes nothing that existing code currently reads from.
-- Migration 015 (run only after verifying this one) does the destructive
-- part — NOT NULL + dropping the old columns.

-- Bridge table: exactly one of subject_id / learning_area_id is set.
-- Neither timetable_periods nor teaching_assignments (below) ever need to
-- know which — they only ever reference academic_offerings.id. Unique
-- constraints on the nullable FK columns work correctly in Postgres since
-- NULLs are never considered equal to each other, so many NULL
-- learning_area_id rows can coexist alongside the real subject_id values.
CREATE TABLE academic_offerings (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id    UUID NOT NULL REFERENCES institutions(id),
    subject_id        UUID REFERENCES subjects(id),
    learning_area_id  UUID REFERENCES learning_areas(id),
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT academic_offerings_exactly_one_source CHECK (
        (subject_id IS NOT NULL AND learning_area_id IS NULL) OR
        (subject_id IS NULL AND learning_area_id IS NOT NULL)
    ),
    CONSTRAINT academic_offerings_subject_unique UNIQUE (subject_id),
    CONSTRAINT academic_offerings_learning_area_unique UNIQUE (learning_area_id)
);
CREATE INDEX idx_academic_offerings_institution ON academic_offerings(institution_id);

-- One row per existing subject and one per existing learning area.
INSERT INTO academic_offerings (institution_id, subject_id)
SELECT institution_id, id FROM subjects WHERE deleted_at IS NULL;

INSERT INTO academic_offerings (institution_id, learning_area_id)
SELECT institution_id, id FROM learning_areas WHERE deleted_at IS NULL;

-- Curriculum-neutral teaching assignment. Named teaching_assignments,
-- deliberately not just "assignments" — that name is already taken by the
-- unrelated student-homework module (assignments / assignment_submissions
-- tables), and reusing it would be confusing the moment that module gets
-- built out.
CREATE TABLE teaching_assignments (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id        UUID NOT NULL REFERENCES institutions(id),
    class_id              UUID NOT NULL REFERENCES classes(id),
    academic_offering_id  UUID NOT NULL REFERENCES academic_offerings(id),
    teacher_id            UUID NOT NULL REFERENCES users(id),
    term_id               UUID NOT NULL REFERENCES terms(id),
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at            TIMESTAMPTZ,
    UNIQUE (institution_id, class_id, academic_offering_id, term_id)
);
CREATE INDEX idx_teaching_assignments_class   ON teaching_assignments(institution_id, class_id, term_id)   WHERE deleted_at IS NULL;
CREATE INDEX idx_teaching_assignments_teacher ON teaching_assignments(institution_id, teacher_id, term_id) WHERE deleted_at IS NULL;

-- Backfill from CBC's existing assignment table — this is the pattern
-- being generalized, not replaced from scratch.
INSERT INTO teaching_assignments (institution_id, class_id, academic_offering_id, teacher_id, term_id)
SELECT lata.institution_id, lata.class_id, ao.id, lata.teacher_id, lata.term_id
FROM learning_area_teacher_assignments lata
JOIN academic_offerings ao
  ON ao.learning_area_id = lata.learning_area_id
 AND ao.institution_id = lata.institution_id
ON CONFLICT (institution_id, class_id, academic_offering_id, term_id) DO NOTHING;

-- Backfill from timetable_periods, NOT from class_subjects — class_subjects
-- has a nullable teacher_id, no term_id, and nothing manages it, so it
-- can't be trusted as complete or current. timetable_periods is the only
-- place 8-4-4 assignment actually lives today.
INSERT INTO teaching_assignments (institution_id, class_id, academic_offering_id, teacher_id, term_id)
SELECT DISTINCT tp.institution_id, tp.class_id, ao.id, tp.teacher_id, tp.term_id
FROM timetable_periods tp
JOIN academic_offerings ao
  ON ao.subject_id = tp.subject_id
 AND ao.institution_id = tp.institution_id
WHERE tp.deleted_at IS NULL
ON CONFLICT (institution_id, class_id, academic_offering_id, term_id) DO NOTHING;

-- New column on timetable_periods — nullable for now. Migration 015 makes
-- it NOT NULL and drops subject_id/teacher_id, but only after this
-- backfill is verified.
ALTER TABLE timetable_periods ADD COLUMN teaching_assignment_id UUID REFERENCES teaching_assignments(id);

UPDATE timetable_periods tp
SET teaching_assignment_id = ta.id
FROM teaching_assignments ta
JOIN academic_offerings ao ON ao.id = ta.academic_offering_id
WHERE ao.subject_id = tp.subject_id
  AND ta.class_id = tp.class_id
  AND ta.teacher_id = tp.teacher_id
  AND ta.term_id = tp.term_id
  AND ta.institution_id = tp.institution_id
  AND tp.deleted_at IS NULL;