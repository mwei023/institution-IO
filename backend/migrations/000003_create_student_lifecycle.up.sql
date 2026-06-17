-- Academic years
CREATE TABLE IF NOT EXISTS academic_years (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id  UUID NOT NULL REFERENCES institutions(id),
    name            VARCHAR(20) NOT NULL,
    start_date      DATE NOT NULL,
    end_date        DATE NOT NULL,
    is_current      BOOLEAN NOT NULL DEFAULT false,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (institution_id, name)
);

-- Terms
CREATE TABLE IF NOT EXISTS terms (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id      UUID NOT NULL REFERENCES institutions(id),
    academic_year_id    UUID NOT NULL REFERENCES academic_years(id),
    name                VARCHAR(20) NOT NULL,
    term_number         SMALLINT NOT NULL,
    start_date          DATE NOT NULL,
    end_date            DATE NOT NULL,
    is_current          BOOLEAN NOT NULL DEFAULT false,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (institution_id, academic_year_id, term_number)
);

-- Departments
CREATE TABLE IF NOT EXISTS departments (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id  UUID NOT NULL REFERENCES institutions(id),
    name            VARCHAR(100) NOT NULL,
    code            VARCHAR(20),
    head_id         UUID REFERENCES users(id),
    deleted_at      TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (institution_id, code)
);

-- Classes
CREATE TABLE IF NOT EXISTS classes (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id      UUID NOT NULL REFERENCES institutions(id),
    academic_year_id    UUID NOT NULL REFERENCES academic_years(id),
    name                VARCHAR(50) NOT NULL,
    stream              VARCHAR(50),
    level               SMALLINT NOT NULL DEFAULT 1,
    class_teacher_id    UUID REFERENCES users(id),
    capacity            SMALLINT NOT NULL DEFAULT 45,
    deleted_at          TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by          UUID REFERENCES users(id)
);

CREATE INDEX IF NOT EXISTS idx_classes_institution ON classes(institution_id, academic_year_id) WHERE deleted_at IS NULL;

-- Prospects
CREATE TABLE IF NOT EXISTS prospects (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id      UUID NOT NULL REFERENCES institutions(id),
    first_name          VARCHAR(100) NOT NULL,
    last_name           VARCHAR(100) NOT NULL,
    date_of_birth       DATE,
    gender              CHAR(1),
    intended_class      VARCHAR(50),
    parent_name         VARCHAR(200),
    parent_phone        VARCHAR(20),
    parent_email        VARCHAR(255),
    source              VARCHAR(50) DEFAULT 'walk_in',
    notes               TEXT,
    status              VARCHAR(30) NOT NULL DEFAULT 'new',
    follow_up_date      DATE,
    deleted_at          TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by          UUID REFERENCES users(id)
);

-- Admissions
CREATE TABLE IF NOT EXISTS admissions (
    id                          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id              UUID NOT NULL REFERENCES institutions(id),
    application_number          VARCHAR(50) NOT NULL,
    academic_year               VARCHAR(10) NOT NULL,
    intended_class              VARCHAR(50) NOT NULL,
    first_name                  VARCHAR(100) NOT NULL,
    last_name                   VARCHAR(100) NOT NULL,
    other_names                 VARCHAR(100),
    date_of_birth               DATE NOT NULL,
    gender                      CHAR(1) NOT NULL,
    birth_certificate_number    VARCHAR(100),
    nationality                 VARCHAR(50) NOT NULL DEFAULT 'Kenyan',
    previous_school             VARCHAR(255),
    previous_class              VARCHAR(50),
    leaving_cert_number         VARCHAR(100),
    blood_group                 VARCHAR(5),
    medical_conditions          TEXT,
    allergies                   TEXT,
    medications                 TEXT,
    special_needs               BOOLEAN NOT NULL DEFAULT false,
    special_needs_details       TEXT,
    boarding_required           BOOLEAN NOT NULL DEFAULT false,
    transport_required          BOOLEAN NOT NULL DEFAULT false,
    preferred_route             VARCHAR(100),
    status                      VARCHAR(50) NOT NULL DEFAULT 'submitted',
    reviewed_by                 UUID REFERENCES users(id),
    reviewed_at                 TIMESTAMPTZ,
    review_notes                TEXT,
    rejection_reason            TEXT,
    parents_json                JSONB NOT NULL DEFAULT '[]',
    student_id                  UUID,
    deleted_at                  TIMESTAMPTZ,
    created_at                  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at                  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by                  UUID REFERENCES users(id),
    updated_by                  UUID REFERENCES users(id),
    UNIQUE (institution_id, application_number)
);

CREATE INDEX IF NOT EXISTS idx_admissions_institution ON admissions(institution_id, status) WHERE deleted_at IS NULL;

-- Students
CREATE TABLE IF NOT EXISTS students (
    id                          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id              UUID NOT NULL REFERENCES institutions(id),
    admission_id                UUID REFERENCES admissions(id),
    admission_number            VARCHAR(50) NOT NULL,
    user_id                     UUID REFERENCES users(id),
    first_name                  VARCHAR(100) NOT NULL,
    last_name                   VARCHAR(100) NOT NULL,
    other_names                 VARCHAR(100),
    date_of_birth               DATE NOT NULL,
    gender                      CHAR(1) NOT NULL,
    birth_certificate_number    VARCHAR(100),
    nationality                 VARCHAR(50) NOT NULL DEFAULT 'Kenyan',
    blood_group                 VARCHAR(5),
    home_address                TEXT,
    photo_url                   TEXT,
    current_class_id            UUID REFERENCES classes(id),
    admission_date              DATE NOT NULL,
    kcpe_index_number           VARCHAR(50),
    kcpe_marks                  SMALLINT,
    knec_registration_number    VARCHAR(50),
    status                      VARCHAR(30) NOT NULL DEFAULT 'active',
    medical_conditions          TEXT,
    allergies                   TEXT,
    medications                 TEXT,
    special_needs               BOOLEAN NOT NULL DEFAULT false,
    special_needs_details       TEXT,
    is_boarder                  BOOLEAN NOT NULL DEFAULT false,
    boarding_bed_id             UUID,
    uses_transport              BOOLEAN NOT NULL DEFAULT false,
    transport_route_id          UUID,
    exit_date                   DATE,
    exit_type                   VARCHAR(30),
    exit_reason                 TEXT,
    deleted_at                  TIMESTAMPTZ,
    created_at                  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at                  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by                  UUID REFERENCES users(id),
    updated_by                  UUID REFERENCES users(id),
    UNIQUE (institution_id, admission_number)
);

CREATE INDEX IF NOT EXISTS idx_students_institution ON students(institution_id, id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_students_class ON students(institution_id, current_class_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_students_admission ON students(institution_id, admission_number);
CREATE INDEX IF NOT EXISTS idx_students_status ON students(institution_id, status) WHERE deleted_at IS NULL;

-- Parent/guardians
CREATE TABLE IF NOT EXISTS parent_guardians (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id      UUID NOT NULL REFERENCES institutions(id),
    user_id             UUID REFERENCES users(id),
    first_name          VARCHAR(100) NOT NULL,
    last_name           VARCHAR(100) NOT NULL,
    relationship        VARCHAR(50) NOT NULL,
    national_id         TEXT,
    phone               VARCHAR(20) NOT NULL,
    email               VARCHAR(255),
    occupation          VARCHAR(100),
    employer            VARCHAR(200),
    is_primary_contact  BOOLEAN NOT NULL DEFAULT false,
    is_fee_payer        BOOLEAN NOT NULL DEFAULT true,
    deleted_at          TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by          UUID REFERENCES users(id)
);

CREATE INDEX IF NOT EXISTS idx_parent_guardians_institution ON parent_guardians(institution_id, id);

-- Student ↔ Parent links
CREATE TABLE IF NOT EXISTS student_parent_links (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id      UUID NOT NULL REFERENCES institutions(id),
    student_id          UUID NOT NULL REFERENCES students(id),
    parent_id           UUID NOT NULL REFERENCES parent_guardians(id),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (institution_id, student_id, parent_id)
);

CREATE INDEX IF NOT EXISTS idx_spl_student ON student_parent_links(institution_id, student_id);
CREATE INDEX IF NOT EXISTS idx_spl_parent ON student_parent_links(institution_id, parent_id);

-- Student journey timeline
CREATE TABLE IF NOT EXISTS student_timeline_events (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id      UUID NOT NULL REFERENCES institutions(id),
    student_id          UUID NOT NULL REFERENCES students(id),
    event_type          VARCHAR(100) NOT NULL,
    title               VARCHAR(255) NOT NULL,
    description         TEXT,
    metadata            JSONB,
    event_date          DATE NOT NULL,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by          UUID REFERENCES users(id)
);

CREATE INDEX IF NOT EXISTS idx_timeline_student ON student_timeline_events(institution_id, student_id, event_date DESC);

-- Consent records
CREATE TABLE IF NOT EXISTS consent_records (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id      UUID NOT NULL REFERENCES institutions(id),
    student_id          UUID NOT NULL REFERENCES students(id),
    parent_id           UUID REFERENCES parent_guardians(id),
    consent_type        VARCHAR(50) NOT NULL,
    reference_id        UUID,
    consent_text        TEXT NOT NULL,
    is_granted          BOOLEAN NOT NULL,
    granted_at          TIMESTAMPTZ,
    revoked_at          TIMESTAMPTZ,
    ip_address          VARCHAR(45),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_consent_student ON consent_records(institution_id, student_id, consent_type);
