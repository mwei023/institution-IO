-- Subjects
CREATE TABLE IF NOT EXISTS subjects (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id  UUID NOT NULL REFERENCES institutions(id),
    department_id   UUID REFERENCES departments(id),
    code            VARCHAR(20) NOT NULL,
    name            VARCHAR(100) NOT NULL,
    knec_code       VARCHAR(20),
    is_compulsory   BOOLEAN NOT NULL DEFAULT true,
    credit_hours_pw SMALLINT NOT NULL DEFAULT 5,
    deleted_at      TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (institution_id, code)
);

-- Class ↔ Subject ↔ Teacher assignments
CREATE TABLE IF NOT EXISTS class_subjects (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id  UUID NOT NULL REFERENCES institutions(id),
    class_id        UUID NOT NULL REFERENCES classes(id),
    subject_id      UUID NOT NULL REFERENCES subjects(id),
    teacher_id      UUID REFERENCES users(id),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (institution_id, class_id, subject_id)
);

CREATE INDEX IF NOT EXISTS idx_class_subjects_class ON class_subjects(institution_id, class_id);
CREATE INDEX IF NOT EXISTS idx_class_subjects_teacher ON class_subjects(institution_id, teacher_id);

-- Timetable periods
CREATE TABLE IF NOT EXISTS timetable_periods (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id  UUID NOT NULL REFERENCES institutions(id),
    term_id         UUID NOT NULL REFERENCES terms(id),
    class_id        UUID NOT NULL REFERENCES classes(id),
    subject_id      UUID NOT NULL REFERENCES subjects(id),
    teacher_id      UUID NOT NULL REFERENCES users(id),
    day_of_week     SMALLINT NOT NULL CHECK (day_of_week BETWEEN 1 AND 5),
    period_number   SMALLINT NOT NULL CHECK (period_number BETWEEN 1 AND 10),
    start_time      TIME NOT NULL,
    end_time        TIME NOT NULL,
    room            VARCHAR(50),
    is_published    BOOLEAN NOT NULL DEFAULT false,
    deleted_at      TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_timetable_class ON timetable_periods(institution_id, class_id, term_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_timetable_teacher ON timetable_periods(institution_id, teacher_id, term_id) WHERE deleted_at IS NULL;

-- Attendance sessions
CREATE TABLE IF NOT EXISTS attendance_sessions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id  UUID NOT NULL REFERENCES institutions(id),
    class_id        UUID NOT NULL REFERENCES classes(id),
    teacher_id      UUID NOT NULL REFERENCES users(id),
    session_date    DATE NOT NULL,
    session_type    VARCHAR(20) NOT NULL DEFAULT 'daily',
    period_id       UUID REFERENCES timetable_periods(id),
    status          VARCHAR(20) NOT NULL DEFAULT 'open',
    submitted_at    TIMESTAMPTZ,
    locked_at       TIMESTAMPTZ,
    locked_by       UUID REFERENCES users(id),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (institution_id, class_id, session_date, session_type, period_id)
);

CREATE INDEX IF NOT EXISTS idx_att_session_class ON attendance_sessions(institution_id, class_id, session_date DESC);

-- Attendance records (per student per session)
CREATE TABLE IF NOT EXISTS attendance_records (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id  UUID NOT NULL REFERENCES institutions(id),
    session_id      UUID NOT NULL REFERENCES attendance_sessions(id),
    student_id      UUID NOT NULL REFERENCES students(id),
    status          VARCHAR(20) NOT NULL CHECK (status IN ('present','absent','late','excused')),
    remarks         TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (institution_id, session_id, student_id)
);

CREATE INDEX IF NOT EXISTS idx_att_records_student ON attendance_records(institution_id, student_id);
CREATE INDEX IF NOT EXISTS idx_att_records_session ON attendance_records(institution_id, session_id);

-- Assignments
CREATE TABLE IF NOT EXISTS assignments (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id      UUID NOT NULL REFERENCES institutions(id),
    class_id            UUID NOT NULL REFERENCES classes(id),
    subject_id          UUID NOT NULL REFERENCES subjects(id),
    teacher_id          UUID NOT NULL REFERENCES users(id),
    term_id             UUID NOT NULL REFERENCES terms(id),
    title               VARCHAR(255) NOT NULL,
    description         TEXT,
    due_date            TIMESTAMPTZ NOT NULL,
    max_marks           NUMERIC(5,2) NOT NULL DEFAULT 100,
    submission_type     VARCHAR(20) NOT NULL DEFAULT 'physical',
    status              VARCHAR(20) NOT NULL DEFAULT 'draft',
    published_at        TIMESTAMPTZ,
    deleted_at          TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_assignments_class ON assignments(institution_id, class_id, term_id) WHERE deleted_at IS NULL;

-- Assignment submissions
CREATE TABLE IF NOT EXISTS assignment_submissions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id  UUID NOT NULL REFERENCES institutions(id),
    assignment_id   UUID NOT NULL REFERENCES assignments(id),
    student_id      UUID NOT NULL REFERENCES students(id),
    status          VARCHAR(20) NOT NULL DEFAULT 'not_submitted',
    submitted_at    TIMESTAMPTZ,
    marks           NUMERIC(5,2),
    feedback        TEXT,
    marked_at       TIMESTAMPTZ,
    marked_by       UUID REFERENCES users(id),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (institution_id, assignment_id, student_id)
);

-- Grade components (CAT 1, CAT 2, Midterm, End-term)
CREATE TABLE IF NOT EXISTS grade_components (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id  UUID NOT NULL REFERENCES institutions(id),
    class_id        UUID NOT NULL REFERENCES classes(id),
    subject_id      UUID NOT NULL REFERENCES subjects(id),
    term_id         UUID NOT NULL REFERENCES terms(id),
    name            VARCHAR(100) NOT NULL,
    weight          NUMERIC(5,2) NOT NULL,
    max_marks       NUMERIC(5,2) NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Grade entries
CREATE TABLE IF NOT EXISTS grade_entries (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id  UUID NOT NULL REFERENCES institutions(id),
    component_id    UUID NOT NULL REFERENCES grade_components(id),
    student_id      UUID NOT NULL REFERENCES students(id),
    marks           NUMERIC(5,2),
    is_absent       BOOLEAN NOT NULL DEFAULT false,
    entered_by      UUID REFERENCES users(id),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (institution_id, component_id, student_id)
);

CREATE INDEX IF NOT EXISTS idx_grade_entries_student ON grade_entries(institution_id, student_id);

-- Report cards
CREATE TABLE IF NOT EXISTS report_cards (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id          UUID NOT NULL REFERENCES institutions(id),
    student_id              UUID NOT NULL REFERENCES students(id),
    class_id                UUID NOT NULL REFERENCES classes(id),
    term_id                 UUID NOT NULL REFERENCES terms(id),
    aggregate_marks         NUMERIC(6,2),
    grade_points            NUMERIC(5,2),
    class_position          SMALLINT,
    stream_position         SMALLINT,
    attendance_days_present SMALLINT,
    attendance_days_total   SMALLINT,
    teacher_comment         TEXT,
    principal_comment       TEXT,
    status                  VARCHAR(20) NOT NULL DEFAULT 'draft',
    published_at            TIMESTAMPTZ,
    pdf_url                 TEXT,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (institution_id, student_id, term_id)
);

-- Lesson plans
CREATE TABLE IF NOT EXISTS lesson_plans (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id      UUID NOT NULL REFERENCES institutions(id),
    teacher_id          UUID NOT NULL REFERENCES users(id),
    class_id            UUID NOT NULL REFERENCES classes(id),
    subject_id          UUID NOT NULL REFERENCES subjects(id),
    term_id             UUID NOT NULL REFERENCES terms(id),
    week_number         SMALLINT NOT NULL,
    topic               VARCHAR(255) NOT NULL,
    subtopic            VARCHAR(255),
    learning_objectives TEXT,
    teaching_aids       TEXT,
    activities          TEXT,
    assessment_method   TEXT,
    notes               TEXT,
    status              VARCHAR(20) NOT NULL DEFAULT 'draft',
    reviewed_by         UUID REFERENCES users(id),
    reviewed_at         TIMESTAMPTZ,
    review_notes        TEXT,
    deleted_at          TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
