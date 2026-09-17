-- 000016_exam_timetable: examination timetable subsystem.
--
-- Curriculum-neutral: papers reference academic_offerings (subjects or
-- learning areas both hang off that table) plus an optional free-text
-- paper name/code. No curriculum, subject count, session count, or
-- duration is hardcoded.
--
-- Reused (NOT duplicated): institutions, academic_years, terms, classes,
-- academic_offerings, users (staff/invigilators), students (via classes).
--
-- New tables: exam_sessions, exam_papers, exam_paper_groups,
-- exam_sittings, exam_invigilators, exam_special_arrangements.
--
-- REVIEW ONLY — do not apply without DBA review. No migration commands
-- were executed as part of this change.

-- ── Examination sessions (one canonical schedule per session) ──
CREATE TABLE IF NOT EXISTS exam_sessions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id  UUID NOT NULL REFERENCES institutions(id),
    academic_year_id UUID REFERENCES academic_years(id),
    term_id         UUID REFERENCES terms(id),
    name            VARCHAR(200) NOT NULL,
    exam_type       VARCHAR(50) NOT NULL DEFAULT 'internal'
        CHECK (exam_type IN ('internal','mock','end_of_term','mid_term','external','other')),
    start_date      DATE NOT NULL,
    end_date        DATE NOT NULL,
    status          VARCHAR(20) NOT NULL DEFAULT 'draft'
        CHECK (status IN ('draft','validated','published','archived')),
    version         INTEGER NOT NULL DEFAULT 1 CHECK (version >= 1),
    created_by      UUID REFERENCES users(id),
    published_at    TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ,
    CHECK (end_date >= start_date)
);

CREATE INDEX IF NOT EXISTS idx_exam_sessions_institution
    ON exam_sessions(institution_id, status) WHERE deleted_at IS NULL;

-- ── Examination papers (components; many papers per offering allowed) ──
CREATE TABLE IF NOT EXISTS exam_papers (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id          UUID NOT NULL REFERENCES institutions(id),
    exam_session_id         UUID NOT NULL REFERENCES exam_sessions(id) ON DELETE CASCADE,
    academic_offering_id    UUID REFERENCES academic_offerings(id),
    paper_name              VARCHAR(200) NOT NULL,
    paper_code              VARCHAR(50),
    paper_number            VARCHAR(20),
    duration_minutes        INTEGER NOT NULL CHECK (duration_minutes > 0),
    max_marks               NUMERIC(7,2),
    instructions            TEXT,
    is_shared               BOOLEAN NOT NULL DEFAULT false,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at              TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_exam_papers_session
    ON exam_papers(institution_id, exam_session_id) WHERE deleted_at IS NULL;

-- ── Candidate groups: which classes/cohorts sit each paper ──
-- class_id reuses the existing classes table (streams/groups/cohorts are
-- classes in this schema). label supports programme/pathway/level/year
-- groups that have no dedicated class row.
CREATE TABLE IF NOT EXISTS exam_paper_groups (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id  UUID NOT NULL REFERENCES institutions(id),
    exam_paper_id   UUID NOT NULL REFERENCES exam_papers(id) ON DELETE CASCADE,
    class_id        UUID REFERENCES classes(id),
    label           VARCHAR(200),
    candidate_count INTEGER CHECK (candidate_count IS NULL OR candidate_count >= 0),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (exam_paper_id, class_id, label),
    CHECK (class_id IS NOT NULL OR label IS NOT NULL)
);

CREATE INDEX IF NOT EXISTS idx_exam_paper_groups_paper
    ON exam_paper_groups(exam_paper_id);
CREATE INDEX IF NOT EXISTS idx_exam_paper_groups_class
    ON exam_paper_groups(institution_id, class_id);

-- ── Sittings/placements: a paper scheduled at a date/time + room ──
CREATE TABLE IF NOT EXISTS exam_sittings (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id      UUID NOT NULL REFERENCES institutions(id),
    exam_session_id     UUID NOT NULL REFERENCES exam_sessions(id) ON DELETE CASCADE,
    exam_paper_id       UUID NOT NULL REFERENCES exam_papers(id) ON DELETE CASCADE,
    exam_date           DATE NOT NULL,
    start_time          TIME NOT NULL,
    end_time            TIME NOT NULL,
    room_name           VARCHAR(100),
    room_capacity       INTEGER CHECK (room_capacity IS NULL OR room_capacity > 0),
    candidate_count     INTEGER CHECK (candidate_count IS NULL OR candidate_count >= 0),
    seating_note        TEXT,
    status              VARCHAR(20) NOT NULL DEFAULT 'draft'
        CHECK (status IN ('draft','published','cancelled')),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at          TIMESTAMPTZ,
    CHECK (end_time > start_time)
);

CREATE INDEX IF NOT EXISTS idx_exam_sittings_session
    ON exam_sittings(institution_id, exam_session_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_exam_sittings_paper
    ON exam_sittings(exam_paper_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_exam_sittings_date
    ON exam_sittings(institution_id, exam_date) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_exam_sittings_room
    ON exam_sittings(institution_id, room_name, exam_date) WHERE deleted_at IS NULL AND room_name IS NOT NULL;

-- ── Invigilator assignments (staff supervising a sitting) ──
CREATE TABLE IF NOT EXISTS exam_invigilators (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id  UUID NOT NULL REFERENCES institutions(id),
    exam_sitting_id UUID NOT NULL REFERENCES exam_sittings(id) ON DELETE CASCADE,
    staff_id        UUID NOT NULL REFERENCES users(id),
    role            VARCHAR(50) NOT NULL DEFAULT 'invigilator'
        CHECK (role IN ('chief','invigilator','assistant','rover')),
    notes           TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (exam_sitting_id, staff_id)
);

CREATE INDEX IF NOT EXISTS idx_exam_invigilators_staff
    ON exam_invigilators(institution_id, staff_id);
CREATE INDEX IF NOT EXISTS idx_exam_invigilators_sitting
    ON exam_invigilators(exam_sitting_id);

-- ── Special arrangements (extensible scheduling accommodations) ──
CREATE TABLE IF NOT EXISTS exam_special_arrangements (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id  UUID NOT NULL REFERENCES institutions(id),
    exam_sitting_id UUID NOT NULL REFERENCES exam_sittings(id) ON DELETE CASCADE,
    student_id      UUID REFERENCES students(id),
    class_id        UUID REFERENCES classes(id),
    arrangement_type VARCHAR(50) NOT NULL
        CHECK (arrangement_type IN ('extra_time','separate_room','accessibility','reader','scribe','other')),
    details         TEXT,
    extra_minutes   INTEGER CHECK (extra_minutes IS NULL OR extra_minutes > 0),
    room_name       VARCHAR(100),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_exam_arrangements_sitting
    ON exam_special_arrangements(exam_sitting_id);
