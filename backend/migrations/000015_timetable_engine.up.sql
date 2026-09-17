-- 000015_timetable_engine: configurable bell schedule + lesson requirements.
--
-- Curriculum-neutral: profiles/slots describe WHEN the institution teaches
-- (days 1-7 Mon-Sun, bell times, breaks); lesson_requirements describe HOW
-- MUCH each class needs per academic_offering. No curriculum is embedded:
-- 8-4-4 subjects and CBC learning areas both hang off academic_offerings.
--
-- timetable_periods keeps day_of_week/period_number for now and gains an
-- optional schedule_slot_id link; the day range widens 1-5 -> 1-7 so
-- Saturday/Sunday can be configured.

-- ── Schedule profiles: one bell-schedule configuration ─────────────
CREATE TABLE schedule_profiles (
    id             UUID         NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    institution_id UUID         NOT NULL REFERENCES institutions(id),
    term_id        UUID         REFERENCES terms(id),
    name           VARCHAR(100) NOT NULL,
    is_default     BOOLEAN      NOT NULL DEFAULT false,
    created_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    deleted_at     TIMESTAMPTZ
);
-- term_id NULL = profile applies to every term unless overridden.
-- At most one default profile per institution (among live rows).
CREATE UNIQUE INDEX uq_schedule_profiles_default
    ON schedule_profiles(institution_id)
    WHERE is_default AND deleted_at IS NULL;
CREATE INDEX idx_schedule_profiles_institution
    ON schedule_profiles(institution_id) WHERE deleted_at IS NULL;

-- ── Schedule slots: reusable time blocks (lessons AND activities) ──
CREATE TABLE schedule_slots (
    id               UUID         NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    institution_id   UUID         NOT NULL REFERENCES institutions(id),
    profile_id       UUID         NOT NULL REFERENCES schedule_profiles(id) ON DELETE CASCADE,
    day_of_week      SMALLINT     NOT NULL CHECK (day_of_week BETWEEN 1 AND 7),
    sequence         SMALLINT     NOT NULL CHECK (sequence >= 1),
    start_time       TIME         NOT NULL,
    end_time         TIME         NOT NULL,
    slot_type        VARCHAR(20)  NOT NULL DEFAULT 'lesson'
        CHECK (slot_type IN ('lesson', 'break', 'lunch', 'assembly', 'activity')),
    is_instructional BOOLEAN      NOT NULL DEFAULT true,
    is_active        BOOLEAN      NOT NULL DEFAULT true,
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    deleted_at       TIMESTAMPTZ,
    CHECK (end_time > start_time)
);
-- Sequence is unique per day within a profile: it is a display position,
-- NOT a global identity (Mon P1 and Sat P1 are different slots).
CREATE UNIQUE INDEX uq_schedule_slots_day_sequence
    ON schedule_slots(profile_id, day_of_week, sequence)
    WHERE deleted_at IS NULL;
CREATE INDEX idx_schedule_slots_profile
    ON schedule_slots(profile_id, day_of_week, sequence)
    WHERE deleted_at IS NULL;

-- ── Lesson requirements: weekly demand per class + offering ────────
CREATE TABLE lesson_requirements (
    id                   UUID         NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    institution_id       UUID         NOT NULL REFERENCES institutions(id),
    class_id             UUID         NOT NULL REFERENCES classes(id),
    academic_offering_id UUID         NOT NULL REFERENCES academic_offerings(id),
    term_id              UUID         NOT NULL REFERENCES terms(id),
    sessions_per_week    INTEGER      NOT NULL CHECK (sessions_per_week > 0),
    allow_single         BOOLEAN      NOT NULL DEFAULT true,
    allow_double         BOOLEAN      NOT NULL DEFAULT false,
    allow_triple         BOOLEAN      NOT NULL DEFAULT false,
    spread_across_days   BOOLEAN      NOT NULL DEFAULT true,
    max_per_day          INTEGER      CHECK (max_per_day IS NULL OR max_per_day > 0),
    created_at           TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    deleted_at           TIMESTAMPTZ,
    CHECK (allow_single OR allow_double OR allow_triple)
);
CREATE UNIQUE INDEX uq_lesson_requirements_scope
    ON lesson_requirements(institution_id, class_id, academic_offering_id, term_id)
    WHERE deleted_at IS NULL;
CREATE INDEX idx_lesson_requirements_class
    ON lesson_requirements(institution_id, class_id, term_id)
    WHERE deleted_at IS NULL;

-- ── Link scheduled periods to slots; open days 1-7 ────────────────
ALTER TABLE timetable_periods
    ADD COLUMN schedule_slot_id UUID REFERENCES schedule_slots(id);
CREATE INDEX idx_timetable_periods_schedule_slot
    ON timetable_periods(schedule_slot_id) WHERE deleted_at IS NULL;

ALTER TABLE timetable_periods
    DROP CONSTRAINT timetable_periods_day_of_week_check;
ALTER TABLE timetable_periods
    ADD CONSTRAINT timetable_periods_day_of_week_check
    CHECK (day_of_week BETWEEN 1 AND 7);
