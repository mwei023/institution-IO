# Database Schema
## Institution OS
**Version**: 1.0.0

---

## Design Principles

1. **Every table has `institution_id UUID NOT NULL`** — multi-tenancy from day one
2. **UUID primary keys** — globally unique, no sequential information leakage
3. **Composite indexes** — `(institution_id, id)` and `(institution_id, fk_column)` on every table
4. **Soft deletes** — `deleted_at TIMESTAMPTZ NULL` — application never hard-deletes
5. **Audit columns** — `created_at`, `updated_at`, `created_by`, `updated_by` on every table
6. **JSONB for flexible data** — medical history, custom attributes, workflow state
7. **Enum types** — use PostgreSQL enum or check constraints for status fields
8. **No stored procedures** — business logic lives in Go

---

## Table of Contents

1. [Platform Tables](#1-platform-tables)
2. [Identity & Access](#2-identity--access)
3. [Student Lifecycle](#3-student-lifecycle)
4. [Academics](#4-academics)
5. [Finance & Accounting](#5-finance--accounting)
6. [Human Resources](#6-human-resources)
7. [Payroll](#7-payroll)
8. [Procurement & Inventory](#8-procurement--inventory)
9. [Asset Management](#9-asset-management)
10. [Boarding](#10-boarding)
11. [Clinic & Health](#11-clinic--health)
12. [Transport](#12-transport)
13. [Library](#13-library)
14. [Security & Gate](#14-security--gate)
15. [Communication](#15-communication)
16. [Governance](#16-governance)
17. [Digital Locker](#17-digital-locker)
18. [Audit & Workflow](#18-audit--workflow)

---

## 1. Platform Tables

```sql
-- ============================================================
-- institutions
-- ============================================================
CREATE TABLE institutions (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name                VARCHAR(255) NOT NULL,
    short_name          VARCHAR(50),
    type                VARCHAR(50) NOT NULL DEFAULT 'secondary',
                        -- primary | secondary | tertiary
    registration_number VARCHAR(100),
    logo_url            TEXT,
    address             TEXT,
    county              VARCHAR(100),
    sub_county          VARCHAR(100),
    phone               VARCHAR(20),
    email               VARCHAR(255),
    website_url         TEXT,
    school_slug         VARCHAR(100) UNIQUE, -- for public portal URL
    settings            JSONB NOT NULL DEFAULT '{}',
                        -- term_dates, fee_settings, sms_provider, etc.
    subscription_plan   VARCHAR(50) NOT NULL DEFAULT 'starter',
    subscription_expires_at TIMESTAMPTZ,
    is_active           BOOLEAN NOT NULL DEFAULT true,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

---

## 2. Identity & Access

```sql
-- ============================================================
-- roles
-- ============================================================
CREATE TABLE roles (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id  UUID NOT NULL REFERENCES institutions(id),
    name            VARCHAR(100) NOT NULL,
    slug            VARCHAR(100) NOT NULL,
                    -- sysadmin|super_admin|dept_admin|teacher|student|parent|
                    -- nurse|accountant|librarian|storekeeper|driver|cook|
                    -- janitor|security_officer|board_member
    description     TEXT,
    is_system       BOOLEAN NOT NULL DEFAULT false, -- system roles cannot be deleted
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (institution_id, slug)
);

CREATE INDEX idx_roles_institution_id ON roles(institution_id);

-- ============================================================
-- permissions
-- ============================================================
CREATE TABLE permissions (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    resource    VARCHAR(100) NOT NULL, -- students|fees|ledger|etc.
    action      VARCHAR(50) NOT NULL,  -- read|write|delete|approve|export
    description TEXT,
    UNIQUE (resource, action)
);

-- ============================================================
-- role_permissions
-- ============================================================
CREATE TABLE role_permissions (
    role_id       UUID NOT NULL REFERENCES roles(id),
    permission_id UUID NOT NULL REFERENCES permissions(id),
    PRIMARY KEY (role_id, permission_id)
);

-- ============================================================
-- users
-- ============================================================
CREATE TABLE users (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id          UUID NOT NULL REFERENCES institutions(id),
    role_id                 UUID NOT NULL REFERENCES roles(id),
    email                   VARCHAR(255),
    phone                   VARCHAR(20),
    first_name              VARCHAR(100) NOT NULL,
    last_name               VARCHAR(100) NOT NULL,
    other_names             VARCHAR(100),
    gender                  CHAR(1),                -- M|F|O
    date_of_birth           DATE,
    national_id             TEXT,                   -- AES-256-GCM encrypted
    profile_photo_url       TEXT,
    password_hash           TEXT NOT NULL,
    is_active               BOOLEAN NOT NULL DEFAULT true,
    mfa_enabled             BOOLEAN NOT NULL DEFAULT false,
    mfa_secret              TEXT,                   -- encrypted TOTP secret
    login_attempts          SMALLINT NOT NULL DEFAULT 0,
    locked_until            TIMESTAMPTZ,
    last_login_at           TIMESTAMPTZ,
    password_changed_at     TIMESTAMPTZ,
    must_change_password    BOOLEAN NOT NULL DEFAULT true,
    deleted_at              TIMESTAMPTZ,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by              UUID REFERENCES users(id),
    updated_by              UUID REFERENCES users(id),
    CONSTRAINT uq_user_email UNIQUE (institution_id, email),
    CONSTRAINT uq_user_phone UNIQUE (institution_id, phone)
);

CREATE INDEX idx_users_institution_id ON users(institution_id, id);
CREATE INDEX idx_users_email ON users(institution_id, email) WHERE deleted_at IS NULL;
CREATE INDEX idx_users_role ON users(institution_id, role_id) WHERE deleted_at IS NULL;

-- ============================================================
-- refresh_tokens (managed in Redis for Phase 1, table for audit)
-- ============================================================
CREATE TABLE refresh_tokens (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id  UUID NOT NULL REFERENCES institutions(id),
    user_id         UUID NOT NULL REFERENCES users(id),
    token_hash      TEXT NOT NULL UNIQUE, -- bcrypt hash of the token
    expires_at      TIMESTAMPTZ NOT NULL,
    revoked_at      TIMESTAMPTZ,
    user_agent      TEXT,
    ip_address      VARCHAR(45),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_refresh_tokens_user ON refresh_tokens(institution_id, user_id);

-- ============================================================
-- password_history
-- ============================================================
CREATE TABLE password_history (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id  UUID NOT NULL REFERENCES institutions(id),
    user_id         UUID NOT NULL REFERENCES users(id),
    password_hash   TEXT NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_password_history_user ON password_history(institution_id, user_id);
```

---

## 3. Student Lifecycle

```sql
-- ============================================================
-- prospects (pre-admission leads)
-- ============================================================
CREATE TABLE prospects (
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
    source              VARCHAR(50),
                        -- walk_in|referral|website|open_day|social_media
    notes               TEXT,
    status              VARCHAR(30) NOT NULL DEFAULT 'new',
                        -- new|contacted|applied|lost
    follow_up_date      DATE,
    deleted_at          TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by          UUID REFERENCES users(id),
    updated_by          UUID REFERENCES users(id)
);

CREATE INDEX idx_prospects_institution ON prospects(institution_id) WHERE deleted_at IS NULL;

-- ============================================================
-- admissions
-- ============================================================
CREATE TABLE admissions (
    id                          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id              UUID NOT NULL REFERENCES institutions(id),
    application_number          VARCHAR(50) NOT NULL,
    academic_year               VARCHAR(10) NOT NULL, -- e.g. 2024
    intended_class              VARCHAR(50) NOT NULL,

    -- Student personal details
    first_name                  VARCHAR(100) NOT NULL,
    last_name                   VARCHAR(100) NOT NULL,
    other_names                 VARCHAR(100),
    date_of_birth               DATE NOT NULL,
    gender                      CHAR(1) NOT NULL,
    birth_certificate_number    VARCHAR(100),
    nationality                 VARCHAR(50) NOT NULL DEFAULT 'Kenyan',

    -- Previous school
    previous_school             VARCHAR(255),
    previous_class              VARCHAR(50),
    leaving_cert_number         VARCHAR(100),

    -- Medical & special needs
    blood_group                 VARCHAR(5),
    medical_conditions          TEXT,
    allergies                   TEXT,
    medications                 TEXT,
    special_needs               BOOLEAN NOT NULL DEFAULT false,
    special_needs_details       TEXT,

    -- Boarding & transport
    boarding_required           BOOLEAN NOT NULL DEFAULT false,
    transport_required          BOOLEAN NOT NULL DEFAULT false,
    preferred_route             VARCHAR(100),

    -- Status & workflow
    status                      VARCHAR(50) NOT NULL DEFAULT 'submitted',
                                -- submitted|under_review|approved|rejected|
                                -- awaiting_payment|awaiting_enrollment|enrolled|withdrawn
    reviewed_by                 UUID REFERENCES users(id),
    reviewed_at                 TIMESTAMPTZ,
    review_notes                TEXT,
    rejection_reason            TEXT,

    -- Parent info stored in JSONB (will be split to parent_guardians table on enrollment)
    parents_json                JSONB NOT NULL DEFAULT '[]',

    -- Converted to student
    student_id                  UUID,   -- set after enrollment

    deleted_at                  TIMESTAMPTZ,
    created_at                  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at                  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by                  UUID REFERENCES users(id),
    updated_by                  UUID REFERENCES users(id),

    UNIQUE (institution_id, application_number)
);

CREATE INDEX idx_admissions_institution ON admissions(institution_id, status) WHERE deleted_at IS NULL;

-- ============================================================
-- students
-- ============================================================
CREATE TABLE students (
    id                          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id              UUID NOT NULL REFERENCES institutions(id),
    admission_id                UUID REFERENCES admissions(id),
    admission_number            VARCHAR(50) NOT NULL,
    user_id                     UUID REFERENCES users(id), -- for student portal login

    -- Personal
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

    -- Academic
    current_class_id            UUID,  -- FK to classes
    admission_date              DATE NOT NULL,
    kcpe_index_number           VARCHAR(50),
    kcpe_marks                  SMALLINT,
    knec_registration_number    VARCHAR(50),
    status                      VARCHAR(30) NOT NULL DEFAULT 'active',
                                -- active|inactive|graduated|transferred|expelled|deceased

    -- Medical (summary — full history in clinic_visits)
    medical_conditions          TEXT,
    allergies                   TEXT,
    medications                 TEXT,
    special_needs               BOOLEAN NOT NULL DEFAULT false,
    special_needs_details       TEXT,

    -- Boarding
    is_boarder                  BOOLEAN NOT NULL DEFAULT false,
    boarding_bed_id             UUID,  -- FK to boarding_beds

    -- Transport
    uses_transport              BOOLEAN NOT NULL DEFAULT false,
    transport_route_id          UUID,  -- FK to transport_routes

    -- Metadata
    exit_date                   DATE,
    exit_type                   VARCHAR(30),
                                -- transfer_out|withdrawal|expulsion|graduation|deceased
    exit_reason                 TEXT,

    deleted_at                  TIMESTAMPTZ,
    created_at                  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at                  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by                  UUID REFERENCES users(id),
    updated_by                  UUID REFERENCES users(id),

    UNIQUE (institution_id, admission_number)
);

CREATE INDEX idx_students_institution ON students(institution_id, id) WHERE deleted_at IS NULL;
CREATE INDEX idx_students_class ON students(institution_id, current_class_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_students_admission_number ON students(institution_id, admission_number);
CREATE INDEX idx_students_status ON students(institution_id, status) WHERE deleted_at IS NULL;

-- ============================================================
-- parent_guardians
-- ============================================================
CREATE TABLE parent_guardians (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id      UUID NOT NULL REFERENCES institutions(id),
    user_id             UUID REFERENCES users(id),
    first_name          VARCHAR(100) NOT NULL,
    last_name           VARCHAR(100) NOT NULL,
    relationship        VARCHAR(50) NOT NULL,
                        -- father|mother|guardian|grandparent|sibling|uncle|aunt|other
    national_id         TEXT,           -- encrypted
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

CREATE INDEX idx_parent_guardians_institution ON parent_guardians(institution_id, id);

-- ============================================================
-- student_parent_links
-- ============================================================
CREATE TABLE student_parent_links (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id      UUID NOT NULL REFERENCES institutions(id),
    student_id          UUID NOT NULL REFERENCES students(id),
    parent_id           UUID NOT NULL REFERENCES parent_guardians(id),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (institution_id, student_id, parent_id)
);

CREATE INDEX idx_spl_student ON student_parent_links(institution_id, student_id);
CREATE INDEX idx_spl_parent ON student_parent_links(institution_id, parent_id);

-- ============================================================
-- student_timeline_events (Student Journey Timeline)
-- ============================================================
CREATE TABLE student_timeline_events (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id      UUID NOT NULL REFERENCES institutions(id),
    student_id          UUID NOT NULL REFERENCES students(id),
    event_type          VARCHAR(100) NOT NULL,
                        -- enrolled|class_promoted|fee_paid|absent|
                        -- discipline_incident|clinic_visit|library_borrowed|
                        -- assignment_submitted|exam_registered|graduated|etc.
    title               VARCHAR(255) NOT NULL,
    description         TEXT,
    metadata            JSONB,          -- event-specific extra data
    event_date          DATE NOT NULL,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by          UUID REFERENCES users(id)
);

CREATE INDEX idx_timeline_student ON student_timeline_events(institution_id, student_id, event_date DESC);

-- ============================================================
-- consent_records
-- ============================================================
CREATE TABLE consent_records (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id      UUID NOT NULL REFERENCES institutions(id),
    student_id          UUID NOT NULL REFERENCES students(id),
    parent_id           UUID REFERENCES parent_guardians(id),
    consent_type        VARCHAR(50) NOT NULL,
                        -- medical|photography|trip|data_processing
    reference_id        UUID,   -- trip_id, policy_version_id, etc.
    consent_text        TEXT NOT NULL,
    is_granted          BOOLEAN NOT NULL,
    granted_at          TIMESTAMPTZ,
    revoked_at          TIMESTAMPTZ,
    ip_address          VARCHAR(45),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_consent_student ON consent_records(institution_id, student_id, consent_type);
```

---

## 4. Academics

```sql
-- ============================================================
-- academic_years
-- ============================================================
CREATE TABLE academic_years (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id  UUID NOT NULL REFERENCES institutions(id),
    name            VARCHAR(20) NOT NULL,   -- e.g. "2024"
    start_date      DATE NOT NULL,
    end_date        DATE NOT NULL,
    is_current      BOOLEAN NOT NULL DEFAULT false,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (institution_id, name)
);

-- ============================================================
-- terms
-- ============================================================
CREATE TABLE terms (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id      UUID NOT NULL REFERENCES institutions(id),
    academic_year_id    UUID NOT NULL REFERENCES academic_years(id),
    name                VARCHAR(20) NOT NULL,   -- Term 1|Term 2|Term 3
    term_number         SMALLINT NOT NULL,       -- 1|2|3
    start_date          DATE NOT NULL,
    end_date            DATE NOT NULL,
    is_current          BOOLEAN NOT NULL DEFAULT false,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (institution_id, academic_year_id, term_number)
);

-- ============================================================
-- departments
-- ============================================================
CREATE TABLE departments (
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

-- ============================================================
-- classes (Form 1 East, Form 2 West, etc.)
-- ============================================================
CREATE TABLE classes (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id      UUID NOT NULL REFERENCES institutions(id),
    academic_year_id    UUID NOT NULL REFERENCES academic_years(id),
    name                VARCHAR(50) NOT NULL,   -- Form 1
    stream              VARCHAR(50),            -- East
    level               SMALLINT NOT NULL,      -- 1|2|3|4 (or grade number)
    class_teacher_id    UUID REFERENCES users(id),
    capacity            SMALLINT NOT NULL DEFAULT 45,
    deleted_at          TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by          UUID REFERENCES users(id)
);

CREATE INDEX idx_classes_institution ON classes(institution_id, academic_year_id) WHERE deleted_at IS NULL;

-- ============================================================
-- subjects
-- ============================================================
CREATE TABLE subjects (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id  UUID NOT NULL REFERENCES institutions(id),
    department_id   UUID REFERENCES departments(id),
    code            VARCHAR(20) NOT NULL,
    name            VARCHAR(100) NOT NULL,
    knec_code       VARCHAR(20),
    is_compulsory   BOOLEAN NOT NULL DEFAULT true,
    credit_hours_pw SMALLINT NOT NULL DEFAULT 5, -- periods per week
    deleted_at      TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (institution_id, code)
);

-- ============================================================
-- class_subjects (which teacher teaches which subject in which class)
-- ============================================================
CREATE TABLE class_subjects (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id  UUID NOT NULL REFERENCES institutions(id),
    class_id        UUID NOT NULL REFERENCES classes(id),
    subject_id      UUID NOT NULL REFERENCES subjects(id),
    teacher_id      UUID REFERENCES users(id),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (institution_id, class_id, subject_id)
);

CREATE INDEX idx_class_subjects_class ON class_subjects(institution_id, class_id);
CREATE INDEX idx_class_subjects_teacher ON class_subjects(institution_id, teacher_id);

-- ============================================================
-- timetable_periods
-- ============================================================
CREATE TABLE timetable_periods (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id  UUID NOT NULL REFERENCES institutions(id),
    term_id         UUID NOT NULL REFERENCES terms(id),
    class_id        UUID NOT NULL REFERENCES classes(id),
    subject_id      UUID NOT NULL REFERENCES subjects(id),
    teacher_id      UUID NOT NULL REFERENCES users(id),
    day_of_week     SMALLINT NOT NULL,  -- 1=Mon ... 5=Fri
    period_number   SMALLINT NOT NULL,  -- 1-8
    start_time      TIME NOT NULL,
    end_time        TIME NOT NULL,
    room            VARCHAR(50),
    is_published    BOOLEAN NOT NULL DEFAULT false,
    deleted_at      TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_timetable_class ON timetable_periods(institution_id, class_id, term_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_timetable_teacher ON timetable_periods(institution_id, teacher_id, term_id) WHERE deleted_at IS NULL;

-- ============================================================
-- attendance_sessions
-- ============================================================
CREATE TABLE attendance_sessions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id  UUID NOT NULL REFERENCES institutions(id),
    class_id        UUID NOT NULL REFERENCES classes(id),
    teacher_id      UUID NOT NULL REFERENCES users(id),
    session_date    DATE NOT NULL,
    session_type    VARCHAR(20) NOT NULL DEFAULT 'daily',
                    -- daily|period (per-period attendance)
    period_id       UUID REFERENCES timetable_periods(id),
    status          VARCHAR(20) NOT NULL DEFAULT 'open',
                    -- open|submitted|locked
    submitted_at    TIMESTAMPTZ,
    locked_at       TIMESTAMPTZ,
    locked_by       UUID REFERENCES users(id),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (institution_id, class_id, session_date, session_type, period_id)
);

CREATE INDEX idx_att_session_class ON attendance_sessions(institution_id, class_id, session_date DESC);

-- ============================================================
-- attendance_records
-- ============================================================
CREATE TABLE attendance_records (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id  UUID NOT NULL REFERENCES institutions(id),
    session_id      UUID NOT NULL REFERENCES attendance_sessions(id),
    student_id      UUID NOT NULL REFERENCES students(id),
    status          VARCHAR(20) NOT NULL,
                    -- present|absent|late|excused
    remarks         TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (institution_id, session_id, student_id)
);

CREATE INDEX idx_att_records_student ON attendance_records(institution_id, student_id);
CREATE INDEX idx_att_records_session ON attendance_records(institution_id, session_id);

-- ============================================================
-- assignments
-- ============================================================
CREATE TABLE assignments (
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
                        -- physical|digital|both
    status              VARCHAR(20) NOT NULL DEFAULT 'draft',
                        -- draft|published|closed
    published_at        TIMESTAMPTZ,
    deleted_at          TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_assignments_class ON assignments(institution_id, class_id, term_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_assignments_teacher ON assignments(institution_id, teacher_id) WHERE deleted_at IS NULL;

-- ============================================================
-- assignment_submissions
-- ============================================================
CREATE TABLE assignment_submissions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id  UUID NOT NULL REFERENCES institutions(id),
    assignment_id   UUID NOT NULL REFERENCES assignments(id),
    student_id      UUID NOT NULL REFERENCES students(id),
    status          VARCHAR(20) NOT NULL DEFAULT 'not_submitted',
                    -- not_submitted|submitted|late|excused
    submitted_at    TIMESTAMPTZ,
    marks           NUMERIC(5,2),
    feedback        TEXT,
    marked_at       TIMESTAMPTZ,
    marked_by       UUID REFERENCES users(id),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (institution_id, assignment_id, student_id)
);

-- ============================================================
-- grade_components (CAT1, CAT2, Midterm, End-term)
-- ============================================================
CREATE TABLE grade_components (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id  UUID NOT NULL REFERENCES institutions(id),
    class_id        UUID NOT NULL REFERENCES classes(id),
    subject_id      UUID NOT NULL REFERENCES subjects(id),
    term_id         UUID NOT NULL REFERENCES terms(id),
    name            VARCHAR(100) NOT NULL,  -- CAT 1, Midterm, etc.
    weight          NUMERIC(5,2) NOT NULL,  -- percentage (sum must = 100)
    max_marks       NUMERIC(5,2) NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ============================================================
-- grade_entries
-- ============================================================
CREATE TABLE grade_entries (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id      UUID NOT NULL REFERENCES institutions(id),
    component_id        UUID NOT NULL REFERENCES grade_components(id),
    student_id          UUID NOT NULL REFERENCES students(id),
    marks               NUMERIC(5,2),
    is_absent           BOOLEAN NOT NULL DEFAULT false,
    entered_by          UUID REFERENCES users(id),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (institution_id, component_id, student_id)
);

CREATE INDEX idx_grade_entries_student ON grade_entries(institution_id, student_id);

-- ============================================================
-- report_cards
-- ============================================================
CREATE TABLE report_cards (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id  UUID NOT NULL REFERENCES institutions(id),
    student_id      UUID NOT NULL REFERENCES students(id),
    class_id        UUID NOT NULL REFERENCES classes(id),
    term_id         UUID NOT NULL REFERENCES terms(id),
    aggregate_marks NUMERIC(6,2),
    grade_points    NUMERIC(5,2),
    class_position  SMALLINT,
    stream_position SMALLINT,
    attendance_days_present SMALLINT,
    attendance_days_total   SMALLINT,
    teacher_comment TEXT,
    principal_comment TEXT,
    status          VARCHAR(20) NOT NULL DEFAULT 'draft',
                    -- draft|published
    published_at    TIMESTAMPTZ,
    pdf_url         TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (institution_id, student_id, term_id)
);

-- ============================================================
-- lesson_plans
-- ============================================================
CREATE TABLE lesson_plans (
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
                        -- draft|submitted|approved|returned
    reviewed_by         UUID REFERENCES users(id),
    reviewed_at         TIMESTAMPTZ,
    review_notes        TEXT,
    deleted_at          TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

---

## 5. Finance & Accounting

```sql
-- ============================================================
-- chart_of_accounts
-- ============================================================
CREATE TABLE chart_of_accounts (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id  UUID NOT NULL REFERENCES institutions(id),
    parent_id       UUID REFERENCES chart_of_accounts(id),
    code            VARCHAR(20) NOT NULL,
    name            VARCHAR(255) NOT NULL,
    type            VARCHAR(20) NOT NULL,
                    -- asset|liability|equity|income|expense
    sub_type        VARCHAR(50),
    is_active       BOOLEAN NOT NULL DEFAULT true,
    deleted_at      TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (institution_id, code)
);

-- ============================================================
-- fee_structures
-- ============================================================
CREATE TABLE fee_structures (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id  UUID NOT NULL REFERENCES institutions(id),
    academic_year_id UUID NOT NULL REFERENCES academic_years(id),
    class_id        UUID REFERENCES classes(id),   -- NULL = all classes
    name            VARCHAR(255) NOT NULL,
    is_active       BOOLEAN NOT NULL DEFAULT true,
    deleted_at      TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ============================================================
-- fee_components
-- ============================================================
CREATE TABLE fee_components (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id      UUID NOT NULL REFERENCES institutions(id),
    fee_structure_id    UUID NOT NULL REFERENCES fee_structures(id),
    name                VARCHAR(100) NOT NULL,   -- Tuition|Boarding|Transport|etc.
    amount              NUMERIC(12,2) NOT NULL,
    term_number         SMALLINT,               -- NULL = same each term; or 1|2|3
    is_mandatory        BOOLEAN NOT NULL DEFAULT true,
    due_date            DATE,
    account_id          UUID REFERENCES chart_of_accounts(id),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ============================================================
-- fee_invoices
-- ============================================================
CREATE TABLE fee_invoices (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id      UUID NOT NULL REFERENCES institutions(id),
    student_id          UUID NOT NULL REFERENCES students(id),
    term_id             UUID NOT NULL REFERENCES terms(id),
    invoice_number      VARCHAR(50) NOT NULL,
    total_amount        NUMERIC(12,2) NOT NULL,
    status              VARCHAR(20) NOT NULL DEFAULT 'unpaid',
                        -- unpaid|partial|paid|waived|cancelled
    due_date            DATE,
    paid_amount         NUMERIC(12,2) NOT NULL DEFAULT 0,
    balance             NUMERIC(12,2) GENERATED ALWAYS AS (total_amount - paid_amount) STORED,
    deleted_at          TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by          UUID REFERENCES users(id),
    UNIQUE (institution_id, invoice_number)
);

CREATE INDEX idx_invoices_student ON fee_invoices(institution_id, student_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_invoices_status ON fee_invoices(institution_id, status) WHERE deleted_at IS NULL;

-- ============================================================
-- fee_invoice_lines
-- ============================================================
CREATE TABLE fee_invoice_lines (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id  UUID NOT NULL REFERENCES institutions(id),
    invoice_id      UUID NOT NULL REFERENCES fee_invoices(id),
    component_id    UUID REFERENCES fee_components(id),
    description     VARCHAR(255) NOT NULL,
    amount          NUMERIC(12,2) NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ============================================================
-- fee_payments (receipts)
-- ============================================================
CREATE TABLE fee_payments (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id      UUID NOT NULL REFERENCES institutions(id),
    student_id          UUID NOT NULL REFERENCES students(id),
    invoice_id          UUID REFERENCES fee_invoices(id),
    receipt_number      VARCHAR(50) NOT NULL,
    amount              NUMERIC(12,2) NOT NULL,
    payment_method      VARCHAR(30) NOT NULL,
                        -- mpesa|cash|bank_transfer|cheque|waiver
    mpesa_reference     VARCHAR(50),
    bank_reference      VARCHAR(100),
    cheque_number       VARCHAR(50),
    received_by         UUID REFERENCES users(id),
    received_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    pdf_url             TEXT,
    is_reversed         BOOLEAN NOT NULL DEFAULT false,
    reversal_reason     TEXT,
    reversed_at         TIMESTAMPTZ,
    reversed_by         UUID REFERENCES users(id),
    deleted_at          TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (institution_id, receipt_number)
);

CREATE INDEX idx_payments_student ON fee_payments(institution_id, student_id) WHERE deleted_at IS NULL;

-- ============================================================
-- mpesa_transactions (raw Daraja logs)
-- ============================================================
CREATE TABLE mpesa_transactions (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id          UUID NOT NULL REFERENCES institutions(id),
    checkout_request_id     VARCHAR(100),
    merchant_request_id     VARCHAR(100),
    mpesa_receipt_number    VARCHAR(50),
    amount                  NUMERIC(12,2),
    phone_number            VARCHAR(20),
    account_reference       VARCHAR(100),  -- admission number
    result_code             INTEGER,
    result_description      TEXT,
    raw_payload             JSONB NOT NULL,
    processed               BOOLEAN NOT NULL DEFAULT false,
    payment_id              UUID REFERENCES fee_payments(id),
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (institution_id, mpesa_receipt_number)
);

-- ============================================================
-- journal_entries
-- ============================================================
CREATE TABLE journal_entries (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id      UUID NOT NULL REFERENCES institutions(id),
    entry_number        VARCHAR(50) NOT NULL,
    entry_date          DATE NOT NULL,
    description         TEXT NOT NULL,
    reference           VARCHAR(100),   -- payment_id, invoice_id, etc.
    source              VARCHAR(50),    -- fee_payment|payroll|manual|procurement
    is_posted           BOOLEAN NOT NULL DEFAULT false,
    posted_at           TIMESTAMPTZ,
    posted_by           UUID REFERENCES users(id),
    deleted_at          TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by          UUID REFERENCES users(id),
    UNIQUE (institution_id, entry_number)
);

CREATE INDEX idx_journal_entries_date ON journal_entries(institution_id, entry_date DESC) WHERE deleted_at IS NULL;

-- ============================================================
-- journal_entry_lines
-- ============================================================
CREATE TABLE journal_entry_lines (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id  UUID NOT NULL REFERENCES institutions(id),
    entry_id        UUID NOT NULL REFERENCES journal_entries(id),
    account_id      UUID NOT NULL REFERENCES chart_of_accounts(id),
    debit           NUMERIC(15,2) NOT NULL DEFAULT 0,
    credit          NUMERIC(15,2) NOT NULL DEFAULT 0,
    narration       TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_debit_credit CHECK (
        (debit > 0 AND credit = 0) OR (credit > 0 AND debit = 0)
    )
);

CREATE INDEX idx_jel_account ON journal_entry_lines(institution_id, account_id);

-- ============================================================
-- budgets
-- ============================================================
CREATE TABLE budgets (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id      UUID NOT NULL REFERENCES institutions(id),
    academic_year_id    UUID NOT NULL REFERENCES academic_years(id),
    account_id          UUID NOT NULL REFERENCES chart_of_accounts(id),
    department_id       UUID REFERENCES departments(id),
    budgeted_amount     NUMERIC(15,2) NOT NULL,
    status              VARCHAR(20) NOT NULL DEFAULT 'draft',
                        -- draft|submitted|approved|rejected
    approved_by         UUID REFERENCES users(id),
    approved_at         TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by          UUID REFERENCES users(id)
);

-- ============================================================
-- bank_accounts
-- ============================================================
CREATE TABLE bank_accounts (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id      UUID NOT NULL REFERENCES institutions(id),
    account_id          UUID NOT NULL REFERENCES chart_of_accounts(id),
    bank_name           VARCHAR(100) NOT NULL,
    branch              VARCHAR(100),
    account_number      TEXT NOT NULL,  -- encrypted
    account_name        VARCHAR(200) NOT NULL,
    is_mpesa_wallet     BOOLEAN NOT NULL DEFAULT false,
    is_active           BOOLEAN NOT NULL DEFAULT true,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

---

## 6. Human Resources

```sql
-- ============================================================
-- staff (extends users)
-- ============================================================
CREATE TABLE staff (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id      UUID NOT NULL REFERENCES institutions(id),
    user_id             UUID NOT NULL REFERENCES users(id),
    staff_number        VARCHAR(50) NOT NULL,
    tsc_number          VARCHAR(50),
    department_id       UUID REFERENCES departments(id),
    job_title           VARCHAR(100) NOT NULL,
    employment_type     VARCHAR(30) NOT NULL DEFAULT 'permanent',
                        -- permanent|contract|part_time|volunteer
    employment_start    DATE NOT NULL,
    probation_end       DATE,
    supervisor_id       UUID REFERENCES staff(id),
    is_teaching_staff   BOOLEAN NOT NULL DEFAULT false,

    -- Bank details (encrypted)
    bank_name           VARCHAR(100),
    bank_account        TEXT,           -- encrypted
    bank_branch         VARCHAR(100),

    -- Statutory numbers
    nssf_number         VARCHAR(50),
    nhif_number         VARCHAR(50),
    kra_pin             VARCHAR(20),

    -- Exit
    exit_date           DATE,
    exit_type           VARCHAR(30),
                        -- resignation|retirement|dismissal|redundancy|deceased
    exit_reason         TEXT,

    deleted_at          TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by          UUID REFERENCES users(id),
    UNIQUE (institution_id, staff_number)
);

CREATE INDEX idx_staff_institution ON staff(institution_id, id) WHERE deleted_at IS NULL;
CREATE INDEX idx_staff_department ON staff(institution_id, department_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_staff_user ON staff(institution_id, user_id);

-- ============================================================
-- staff_qualifications
-- ============================================================
CREATE TABLE staff_qualifications (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id  UUID NOT NULL REFERENCES institutions(id),
    staff_id        UUID NOT NULL REFERENCES staff(id),
    qualification   VARCHAR(100) NOT NULL,  -- B.Ed, PGDE, etc.
    institution_name VARCHAR(200),
    year_obtained   SMALLINT,
    certificate_url TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ============================================================
-- leave_entitlements
-- ============================================================
CREATE TABLE leave_entitlements (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id  UUID NOT NULL REFERENCES institutions(id),
    staff_id        UUID NOT NULL REFERENCES staff(id),
    leave_type      VARCHAR(30) NOT NULL,
                    -- annual|sick|maternity|paternity|compassionate|study|unpaid
    year            SMALLINT NOT NULL,
    entitled_days   NUMERIC(5,1) NOT NULL,
    used_days       NUMERIC(5,1) NOT NULL DEFAULT 0,
    carried_over    NUMERIC(5,1) NOT NULL DEFAULT 0,
    available_days  NUMERIC(5,1) GENERATED ALWAYS AS (entitled_days + carried_over - used_days) STORED,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (institution_id, staff_id, leave_type, year)
);

-- ============================================================
-- leave_requests
-- ============================================================
CREATE TABLE leave_requests (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id  UUID NOT NULL REFERENCES institutions(id),
    staff_id        UUID NOT NULL REFERENCES staff(id),
    leave_type      VARCHAR(30) NOT NULL,
    start_date      DATE NOT NULL,
    end_date        DATE NOT NULL,
    days_requested  NUMERIC(5,1) NOT NULL,
    reason          TEXT,
    status          VARCHAR(20) NOT NULL DEFAULT 'pending',
                    -- pending|approved|rejected|cancelled
    reviewed_by     UUID REFERENCES users(id),
    reviewed_at     TIMESTAMPTZ,
    review_notes    TEXT,
    cover_staff_id  UUID REFERENCES staff(id),
    deleted_at      TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ============================================================
-- performance_reviews
-- ============================================================
CREATE TABLE performance_reviews (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id      UUID NOT NULL REFERENCES institutions(id),
    staff_id            UUID NOT NULL REFERENCES staff(id),
    reviewer_id         UUID NOT NULL REFERENCES users(id),
    review_period       VARCHAR(20) NOT NULL,  -- 2024-H1, 2024-annual
    review_type         VARCHAR(20) NOT NULL,  -- mid_year|annual
    self_score          NUMERIC(3,1),
    reviewer_score      NUMERIC(3,1),
    final_score         NUMERIC(3,1),
    comments            TEXT,
    targets_next_period JSONB,
    status              VARCHAR(20) NOT NULL DEFAULT 'draft',
    deleted_at          TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

---

## 7. Payroll

```sql
-- ============================================================
-- salary_structures
-- ============================================================
CREATE TABLE salary_structures (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id  UUID NOT NULL REFERENCES institutions(id),
    staff_id        UUID NOT NULL REFERENCES staff(id),
    effective_date  DATE NOT NULL,
    basic_salary    NUMERIC(12,2) NOT NULL,
    allowances      JSONB NOT NULL DEFAULT '[]',
                    -- [{name, amount, is_taxable}]
    deductions      JSONB NOT NULL DEFAULT '[]',
                    -- [{name, amount, is_statutory}]
    is_active       BOOLEAN NOT NULL DEFAULT true,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by      UUID REFERENCES users(id)
);

-- ============================================================
-- payroll_runs
-- ============================================================
CREATE TABLE payroll_runs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id  UUID NOT NULL REFERENCES institutions(id),
    pay_period      VARCHAR(7) NOT NULL,    -- YYYY-MM
    status          VARCHAR(20) NOT NULL DEFAULT 'draft',
                    -- draft|processing|approved|paid
    total_gross     NUMERIC(15,2),
    total_deductions NUMERIC(15,2),
    total_net       NUMERIC(15,2),
    approved_by     UUID REFERENCES users(id),
    approved_at     TIMESTAMPTZ,
    processed_at    TIMESTAMPTZ,
    deleted_at      TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by      UUID REFERENCES users(id),
    UNIQUE (institution_id, pay_period)
);

-- ============================================================
-- payroll_items (one per employee per run)
-- ============================================================
CREATE TABLE payroll_items (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id      UUID NOT NULL REFERENCES institutions(id),
    payroll_run_id      UUID NOT NULL REFERENCES payroll_runs(id),
    staff_id            UUID NOT NULL REFERENCES staff(id),
    basic_salary        NUMERIC(12,2) NOT NULL,
    gross_salary        NUMERIC(12,2) NOT NULL,
    paye                NUMERIC(12,2) NOT NULL DEFAULT 0,
    nssf_employee       NUMERIC(12,2) NOT NULL DEFAULT 0,
    nhif_employee       NUMERIC(12,2) NOT NULL DEFAULT 0,
    housing_levy        NUMERIC(12,2) NOT NULL DEFAULT 0,
    other_deductions    NUMERIC(12,2) NOT NULL DEFAULT 0,
    total_deductions    NUMERIC(12,2) NOT NULL,
    net_pay             NUMERIC(12,2) NOT NULL,
    allowances_json     JSONB NOT NULL DEFAULT '[]',
    deductions_json     JSONB NOT NULL DEFAULT '[]',
    payslip_pdf_url     TEXT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (institution_id, payroll_run_id, staff_id)
);
```

---

## 8. Procurement & Inventory

```sql
-- ============================================================
-- suppliers
-- ============================================================
CREATE TABLE suppliers (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id  UUID NOT NULL REFERENCES institutions(id),
    name            VARCHAR(255) NOT NULL,
    registration_number VARCHAR(50),
    kra_pin         VARCHAR(20),
    contact_person  VARCHAR(200),
    phone           VARCHAR(20),
    email           VARCHAR(255),
    address         TEXT,
    bank_details    JSONB,
    category        VARCHAR(50),
    is_active       BOOLEAN NOT NULL DEFAULT true,
    deleted_at      TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ============================================================
-- purchase_requests
-- ============================================================
CREATE TABLE purchase_requests (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id      UUID NOT NULL REFERENCES institutions(id),
    pr_number           VARCHAR(50) NOT NULL,
    department_id       UUID REFERENCES departments(id),
    requested_by        UUID NOT NULL REFERENCES users(id),
    items               JSONB NOT NULL,  -- [{name, qty, unit, estimated_cost, justification}]
    total_estimated     NUMERIC(12,2),
    status              VARCHAR(30) NOT NULL DEFAULT 'pending',
                        -- pending|dept_approved|finance_approved|po_raised|rejected
    dept_approved_by    UUID REFERENCES users(id),
    dept_approved_at    TIMESTAMPTZ,
    finance_approved_by UUID REFERENCES users(id),
    finance_approved_at TIMESTAMPTZ,
    rejection_reason    TEXT,
    deleted_at          TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (institution_id, pr_number)
);

-- ============================================================
-- purchase_orders (LPO)
-- ============================================================
CREATE TABLE purchase_orders (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id  UUID NOT NULL REFERENCES institutions(id),
    lpo_number      VARCHAR(50) NOT NULL,
    pr_id           UUID REFERENCES purchase_requests(id),
    supplier_id     UUID NOT NULL REFERENCES suppliers(id),
    items           JSONB NOT NULL,
    total_amount    NUMERIC(12,2) NOT NULL,
    status          VARCHAR(30) NOT NULL DEFAULT 'issued',
                    -- issued|partially_received|received|invoiced|paid|cancelled
    issued_date     DATE NOT NULL,
    expected_date   DATE,
    deleted_at      TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by      UUID REFERENCES users(id),
    UNIQUE (institution_id, lpo_number)
);

-- ============================================================
-- goods_received_notes (GRN)
-- ============================================================
CREATE TABLE goods_received_notes (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id  UUID NOT NULL REFERENCES institutions(id),
    grn_number      VARCHAR(50) NOT NULL,
    po_id           UUID REFERENCES purchase_orders(id),
    received_by     UUID NOT NULL REFERENCES users(id),
    received_date   DATE NOT NULL,
    items           JSONB NOT NULL,  -- [{item, ordered_qty, received_qty, condition}]
    notes           TEXT,
    deleted_at      TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (institution_id, grn_number)
);

-- ============================================================
-- inventory_items
-- ============================================================
CREATE TABLE inventory_items (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id      UUID NOT NULL REFERENCES institutions(id),
    code                VARCHAR(50) NOT NULL,
    name                VARCHAR(255) NOT NULL,
    category            VARCHAR(50),
    unit_of_measure     VARCHAR(20) NOT NULL,
    reorder_level       NUMERIC(10,2) NOT NULL DEFAULT 0,
    reorder_quantity    NUMERIC(10,2) NOT NULL DEFAULT 0,
    current_stock       NUMERIC(10,2) NOT NULL DEFAULT 0,
    unit_cost           NUMERIC(12,2) NOT NULL DEFAULT 0,
    storage_location    VARCHAR(100),
    is_active           BOOLEAN NOT NULL DEFAULT true,
    deleted_at          TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (institution_id, code)
);

-- ============================================================
-- inventory_movements
-- ============================================================
CREATE TABLE inventory_movements (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id  UUID NOT NULL REFERENCES institutions(id),
    item_id         UUID NOT NULL REFERENCES inventory_items(id),
    movement_type   VARCHAR(20) NOT NULL,
                    -- grn|issue|adjustment|transfer|stocktake
    quantity        NUMERIC(10,2) NOT NULL,  -- positive=in, negative=out
    balance_after   NUMERIC(10,2) NOT NULL,
    reference_type  VARCHAR(30),  -- grn|requisition|adjustment
    reference_id    UUID,
    department_id   UUID REFERENCES departments(id),
    notes           TEXT,
    created_by      UUID NOT NULL REFERENCES users(id),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_inv_movements_item ON inventory_movements(institution_id, item_id, created_at DESC);
```

---

## 9. Asset Management

```sql
CREATE TABLE assets (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id      UUID NOT NULL REFERENCES institutions(id),
    asset_tag           VARCHAR(50) NOT NULL,
    name                VARCHAR(255) NOT NULL,
    description         TEXT,
    category            VARCHAR(50),
    serial_number       VARCHAR(100),
    model               VARCHAR(100),
    manufacturer        VARCHAR(100),
    supplier_id         UUID REFERENCES suppliers(id),
    purchase_date       DATE,
    purchase_cost       NUMERIC(12,2),
    useful_life_years   SMALLINT,
    depreciation_method VARCHAR(20) DEFAULT 'straight_line',
    current_book_value  NUMERIC(12,2),
    condition           VARCHAR(20) NOT NULL DEFAULT 'good',
                        -- excellent|good|fair|poor|condemned
    assigned_to         UUID REFERENCES users(id),
    department_id       UUID REFERENCES departments(id),
    location            VARCHAR(100),
    warranty_expiry     DATE,
    disposal_date       DATE,
    disposal_method     VARCHAR(50),
    disposal_proceeds   NUMERIC(12,2),
    account_id          UUID REFERENCES chart_of_accounts(id),
    deleted_at          TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by          UUID REFERENCES users(id),
    UNIQUE (institution_id, asset_tag)
);
```

---

## 10. Boarding

```sql
CREATE TABLE boarding_blocks (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id  UUID NOT NULL REFERENCES institutions(id),
    name            VARCHAR(100) NOT NULL,
    gender          CHAR(1),   -- M|F
    matron_id       UUID REFERENCES users(id),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE boarding_dormitories (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id  UUID NOT NULL REFERENCES institutions(id),
    block_id        UUID NOT NULL REFERENCES boarding_blocks(id),
    name            VARCHAR(100) NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE boarding_rooms (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id  UUID NOT NULL REFERENCES institutions(id),
    dormitory_id    UUID NOT NULL REFERENCES boarding_dormitories(id),
    room_number     VARCHAR(20) NOT NULL,
    capacity        SMALLINT NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE boarding_beds (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id  UUID NOT NULL REFERENCES institutions(id),
    room_id         UUID NOT NULL REFERENCES boarding_rooms(id),
    bed_label       VARCHAR(10) NOT NULL,  -- A|B|C or 1|2|3
    is_occupied     BOOLEAN NOT NULL DEFAULT false,
    student_id      UUID REFERENCES students(id),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE boarding_night_attendance (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id  UUID NOT NULL REFERENCES institutions(id),
    dormitory_id    UUID NOT NULL REFERENCES boarding_dormitories(id),
    attendance_date DATE NOT NULL,
    taken_by        UUID NOT NULL REFERENCES users(id),
    records         JSONB NOT NULL,  -- [{student_id, status: present|absent_awl|absent_unauth}]
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (institution_id, dormitory_id, attendance_date)
);
```

---

## 11. Clinic & Health

```sql
CREATE TABLE clinic_visits (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id  UUID NOT NULL REFERENCES institutions(id),
    student_id      UUID NOT NULL REFERENCES students(id),
    nurse_id        UUID NOT NULL REFERENCES users(id),
    visit_date      DATE NOT NULL,
    visit_time      TIME NOT NULL,
    complaint       TEXT NOT NULL,
    vital_signs     JSONB,  -- {temp, bp, pulse, weight}
    diagnosis       TEXT,
    treatment_given TEXT,
    medications     JSONB,  -- [{name, dosage, frequency, duration}]
    follow_up       BOOLEAN NOT NULL DEFAULT false,
    follow_up_date  DATE,
    referred_out    BOOLEAN NOT NULL DEFAULT false,
    referral_facility VARCHAR(200),
    is_emergency    BOOLEAN NOT NULL DEFAULT false,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_clinic_visits_student ON clinic_visits(institution_id, student_id, visit_date DESC);

CREATE TABLE clinic_isolation (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id  UUID NOT NULL REFERENCES institutions(id),
    student_id      UUID NOT NULL REFERENCES students(id),
    bed_id          UUID,
    admission_date  TIMESTAMPTZ NOT NULL,
    admission_reason TEXT NOT NULL,
    expected_discharge DATE,
    discharge_date  TIMESTAMPTZ,
    discharge_notes TEXT,
    admitted_by     UUID NOT NULL REFERENCES users(id),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

---

## 12. Transport

```sql
CREATE TABLE transport_routes (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id  UUID NOT NULL REFERENCES institutions(id),
    name            VARCHAR(100) NOT NULL,
    description     TEXT,
    estimated_duration_min SMALLINT,
    is_active       BOOLEAN NOT NULL DEFAULT true,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE transport_stops (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id  UUID NOT NULL REFERENCES institutions(id),
    route_id        UUID NOT NULL REFERENCES transport_routes(id),
    name            VARCHAR(100) NOT NULL,
    latitude        DECIMAL(10,8),
    longitude       DECIMAL(11,8),
    order_number    SMALLINT NOT NULL,
    pickup_time     TIME,
    dropoff_time    TIME,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE vehicles (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id      UUID NOT NULL REFERENCES institutions(id),
    registration_plate  VARCHAR(20) NOT NULL,
    make                VARCHAR(50),
    model               VARCHAR(50),
    year                SMALLINT,
    capacity            SMALLINT NOT NULL,
    insurance_expiry    DATE,
    inspection_expiry   DATE,
    is_active           BOOLEAN NOT NULL DEFAULT true,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (institution_id, registration_plate)
);

CREATE TABLE trips (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id  UUID NOT NULL REFERENCES institutions(id),
    route_id        UUID NOT NULL REFERENCES transport_routes(id),
    vehicle_id      UUID NOT NULL REFERENCES vehicles(id),
    driver_id       UUID NOT NULL REFERENCES users(id),
    trip_date       DATE NOT NULL,
    trip_type       CHAR(2) NOT NULL,   -- AM|PM
    departure_time  TIMESTAMPTZ,
    arrival_time    TIMESTAMPTZ,
    mileage_start   INTEGER,
    mileage_end     INTEGER,
    incidents       TEXT,
    manifest        JSONB,  -- [{student_id, stop_id, boarded: bool}]
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

---

## 13. Library

```sql
CREATE TABLE library_books (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id  UUID NOT NULL REFERENCES institutions(id),
    isbn            VARCHAR(20),
    title           VARCHAR(500) NOT NULL,
    authors         TEXT,
    publisher       VARCHAR(200),
    publish_year    SMALLINT,
    edition         VARCHAR(20),
    subject_id      UUID REFERENCES subjects(id),
    call_number     VARCHAR(50),
    total_copies    SMALLINT NOT NULL DEFAULT 1,
    available_copies SMALLINT NOT NULL DEFAULT 1,
    is_digital      BOOLEAN NOT NULL DEFAULT false,
    digital_url     TEXT,
    deleted_at      TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE library_copies (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id  UUID NOT NULL REFERENCES institutions(id),
    book_id         UUID NOT NULL REFERENCES library_books(id),
    copy_number     VARCHAR(20) NOT NULL,
    barcode         VARCHAR(50),
    condition       VARCHAR(20) NOT NULL DEFAULT 'good',
    status          VARCHAR(20) NOT NULL DEFAULT 'available',
                    -- available|borrowed|lost|retired
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE library_borrowings (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id  UUID NOT NULL REFERENCES institutions(id),
    copy_id         UUID NOT NULL REFERENCES library_copies(id),
    borrower_id     UUID NOT NULL REFERENCES users(id),
    issued_by       UUID NOT NULL REFERENCES users(id),
    issued_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    due_date        DATE NOT NULL,
    returned_at     TIMESTAMPTZ,
    return_condition VARCHAR(20),
    fine_amount     NUMERIC(8,2) NOT NULL DEFAULT 0,
    fine_paid       BOOLEAN NOT NULL DEFAULT false,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_borrowings_borrower ON library_borrowings(institution_id, borrower_id);
```

---

## 14. Security & Gate

```sql
CREATE TABLE gate_logs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id  UUID NOT NULL REFERENCES institutions(id),
    student_id      UUID REFERENCES students(id),
    logged_by       UUID NOT NULL REFERENCES users(id),
    direction       VARCHAR(5) NOT NULL,  -- IN|OUT
    log_time        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    is_authorized   BOOLEAN NOT NULL DEFAULT true,
    permission_ref  VARCHAR(100),
    notes           TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE visitors (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id  UUID NOT NULL REFERENCES institutions(id),
    name            VARCHAR(200) NOT NULL,
    national_id     TEXT,
    phone           VARCHAR(20),
    purpose         TEXT NOT NULL,
    host_id         UUID NOT NULL REFERENCES users(id),
    badge_number    VARCHAR(20),
    check_in_time   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    check_out_time  TIMESTAMPTZ,
    status          VARCHAR(20) NOT NULL DEFAULT 'checked_in',
                    -- pre_registered|checked_in|checked_out|denied
    id_photo_url    TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE security_incidents (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id  UUID NOT NULL REFERENCES institutions(id),
    incident_type   VARCHAR(50) NOT NULL,
    description     TEXT NOT NULL,
    location        VARCHAR(100),
    incident_time   TIMESTAMPTZ NOT NULL,
    parties_involved JSONB,
    evidence_urls   JSONB,
    status          VARCHAR(20) NOT NULL DEFAULT 'open',
    reported_by     UUID NOT NULL REFERENCES users(id),
    reviewed_by     UUID REFERENCES users(id),
    resolution      TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

---

## 15. Communication

```sql
CREATE TABLE announcements (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id  UUID NOT NULL REFERENCES institutions(id),
    title           VARCHAR(255) NOT NULL,
    body            TEXT NOT NULL,
    type            VARCHAR(30) NOT NULL DEFAULT 'general',
                    -- general|academic|event|emergency|fee_notice
    audience        VARCHAR(50) NOT NULL DEFAULT 'all',
                    -- all|parents|teachers|students|staff|class:{id}
    send_sms        BOOLEAN NOT NULL DEFAULT false,
    send_push       BOOLEAN NOT NULL DEFAULT true,
    is_published    BOOLEAN NOT NULL DEFAULT false,
    published_at    TIMESTAMPTZ,
    expires_at      TIMESTAMPTZ,
    attachments     JSONB,
    read_count      INTEGER NOT NULL DEFAULT 0,
    deleted_at      TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by      UUID NOT NULL REFERENCES users(id)
);

CREATE INDEX idx_announcements_published ON announcements(institution_id, published_at DESC)
    WHERE deleted_at IS NULL AND is_published = true;

CREATE TABLE announcement_reads (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id  UUID NOT NULL REFERENCES institutions(id),
    announcement_id UUID NOT NULL REFERENCES announcements(id),
    user_id         UUID NOT NULL REFERENCES users(id),
    read_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (institution_id, announcement_id, user_id)
);

CREATE TABLE messages (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id  UUID NOT NULL REFERENCES institutions(id),
    sender_id       UUID NOT NULL REFERENCES users(id),
    recipient_id    UUID NOT NULL REFERENCES users(id),
    body            TEXT NOT NULL,
    is_read         BOOLEAN NOT NULL DEFAULT false,
    read_at         TIMESTAMPTZ,
    deleted_at      TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_messages_recipient ON messages(institution_id, recipient_id, created_at DESC) WHERE deleted_at IS NULL;

CREATE TABLE notifications (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id  UUID NOT NULL REFERENCES institutions(id),
    user_id         UUID NOT NULL REFERENCES users(id),
    type            VARCHAR(50) NOT NULL,
    title           VARCHAR(255) NOT NULL,
    body            TEXT NOT NULL,
    channel         VARCHAR(20) NOT NULL,
                    -- in_app|push|sms|email|whatsapp
    is_read         BOOLEAN NOT NULL DEFAULT false,
    read_at         TIMESTAMPTZ,
    sent_at         TIMESTAMPTZ,
    delivered_at    TIMESTAMPTZ,
    failed_at       TIMESTAMPTZ,
    failure_reason  TEXT,
    metadata        JSONB,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_notifications_user ON notifications(institution_id, user_id, created_at DESC);
```

---

## 16. Governance

```sql
CREATE TABLE board_members (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id  UUID NOT NULL REFERENCES institutions(id),
    user_id         UUID NOT NULL REFERENCES users(id),
    board_role      VARCHAR(100) NOT NULL,  -- Chairperson|Secretary|Treasurer|Member
    term_start      DATE NOT NULL,
    term_end        DATE,
    is_active       BOOLEAN NOT NULL DEFAULT true,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE board_meetings (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id  UUID NOT NULL REFERENCES institutions(id),
    title           VARCHAR(255) NOT NULL,
    meeting_type    VARCHAR(30) NOT NULL,
                    -- ordinary|special|agm
    meeting_date    TIMESTAMPTZ NOT NULL,
    venue           VARCHAR(200),
    agenda          TEXT,
    minutes         TEXT,
    status          VARCHAR(20) NOT NULL DEFAULT 'scheduled',
                    -- scheduled|completed|cancelled
    minutes_approved_at TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE policies (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id  UUID NOT NULL REFERENCES institutions(id),
    title           VARCHAR(255) NOT NULL,
    category        VARCHAR(50),
    content         TEXT NOT NULL,
    version         VARCHAR(10) NOT NULL DEFAULT '1.0',
    status          VARCHAR(20) NOT NULL DEFAULT 'draft',
                    -- draft|under_review|approved|superseded
    effective_date  DATE,
    review_date     DATE,
    approved_by     UUID REFERENCES users(id),
    approved_at     TIMESTAMPTZ,
    previous_version_id UUID REFERENCES policies(id),
    deleted_at      TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

---

## 17. Digital Locker

```sql
CREATE TABLE locker_folders (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id  UUID NOT NULL REFERENCES institutions(id),
    entity_type     VARCHAR(30) NOT NULL,  -- student|staff|institution
    entity_id       UUID NOT NULL,
    name            VARCHAR(100) NOT NULL,
    parent_folder_id UUID REFERENCES locker_folders(id),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_locker_folders_entity ON locker_folders(institution_id, entity_type, entity_id);

CREATE TABLE locker_documents (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id  UUID NOT NULL REFERENCES institutions(id),
    folder_id       UUID NOT NULL REFERENCES locker_folders(id),
    entity_type     VARCHAR(30) NOT NULL,
    entity_id       UUID NOT NULL,
    document_type   VARCHAR(50),
                    -- birth_certificate|id_copy|certificate|contract|receipt|report_card|consent|other
    title           VARCHAR(255) NOT NULL,
    file_name       VARCHAR(255) NOT NULL,
    file_size       INTEGER,
    mime_type       VARCHAR(100),
    storage_key     TEXT NOT NULL,  -- path in object storage
    version         SMALLINT NOT NULL DEFAULT 1,
    previous_version_id UUID REFERENCES locker_documents(id),
    access_level    VARCHAR(20) NOT NULL DEFAULT 'institution',
                    -- public|institution|role_based|private
    deleted_at      TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by      UUID NOT NULL REFERENCES users(id)
);

CREATE INDEX idx_locker_docs_entity ON locker_documents(institution_id, entity_type, entity_id) WHERE deleted_at IS NULL;
```

---

## 18. Audit & Workflow

```sql
-- ============================================================
-- audit_logs (APPEND ONLY — no UPDATE or DELETE ever)
-- ============================================================
CREATE TABLE audit_logs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id  UUID,   -- NULL for platform-level actions
    user_id         UUID,
    action          VARCHAR(20) NOT NULL,
                    -- INSERT|UPDATE|DELETE|VIEW|LOGIN|LOGOUT|EXPORT
    table_name      VARCHAR(100),
    record_id       UUID,
    old_values      JSONB,
    new_values      JSONB,
    ip_address      VARCHAR(45),
    user_agent      TEXT,
    request_id      VARCHAR(50),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_audit_logs_institution ON audit_logs(institution_id, created_at DESC);
CREATE INDEX idx_audit_logs_user ON audit_logs(user_id, created_at DESC);
CREATE INDEX idx_audit_logs_record ON audit_logs(table_name, record_id, created_at DESC);

-- Prevent modifications to audit log
CREATE RULE no_update_audit AS ON UPDATE TO audit_logs DO INSTEAD NOTHING;
CREATE RULE no_delete_audit AS ON DELETE TO audit_logs DO INSTEAD NOTHING;

-- ============================================================
-- workflow_definitions
-- ============================================================
CREATE TABLE workflow_definitions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id  UUID NOT NULL REFERENCES institutions(id),
    name            VARCHAR(100) NOT NULL,
    entity_type     VARCHAR(50) NOT NULL,
                    -- admission|leave_request|purchase_request|policy|student_exit
    steps           JSONB NOT NULL,
                    -- [{step_order, role, action_label, timeout_days, escalate_to_role}]
    is_active       BOOLEAN NOT NULL DEFAULT true,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ============================================================
-- workflow_instances
-- ============================================================
CREATE TABLE workflow_instances (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id          UUID NOT NULL REFERENCES institutions(id),
    definition_id           UUID NOT NULL REFERENCES workflow_definitions(id),
    entity_type             VARCHAR(50) NOT NULL,
    entity_id               UUID NOT NULL,
    current_step            SMALLINT NOT NULL DEFAULT 1,
    status                  VARCHAR(20) NOT NULL DEFAULT 'pending',
                            -- pending|in_progress|approved|rejected|cancelled
    initiated_by            UUID NOT NULL REFERENCES users(id),
    completed_at            TIMESTAMPTZ,
    history                 JSONB NOT NULL DEFAULT '[]',
                            -- [{step, actor_id, action, notes, actioned_at}]
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_workflow_instances_entity ON workflow_instances(institution_id, entity_type, entity_id);
```

---

## Migration Files

All SQL above is split into versioned migration files:

```
backend/migrations/
000001_create_institutions.up.sql
000002_create_iam.up.sql
000003_create_student_lifecycle.up.sql
000004_create_academics.up.sql
000005_create_finance.up.sql
000006_create_hr_payroll.up.sql
000007_create_procurement_inventory.up.sql
000008_create_assets.up.sql
000009_create_boarding.up.sql
000010_create_clinic.up.sql
000011_create_transport.up.sql
000012_create_library.up.sql
000013_create_security.up.sql
000014_create_communication.up.sql
000015_create_governance.up.sql
000016_create_digital_locker.up.sql
000017_create_audit_workflow.up.sql
000018_seed_permissions.up.sql
000019_seed_phase1_institution.up.sql
```
