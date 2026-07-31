-- v1.0 CBC/CBE academic layer.
--
-- Deliberately parallel to the existing 8-4-4 tables (subjects,
-- grade_components, grade_entries, report_cards), not merged into them.
-- Rationale (agreed in design session): two genuinely different
-- assessment philosophies sharing one polymorphic table is the trap,
-- not the solution — same reasoning as not building a generic
-- responsibility engine after seeing only one real attendance case.
--
-- Explicitly OUT of scope for v1.0 (belongs to the v1.1 Curriculum
-- Engine, additive on top of learning_areas, no migration of this data
-- required when it lands): strands, sub_strands, learning_outcomes,
-- lesson_plans, schemes_of_work, coverage_tracking, competency_mapping.

-- ─── Curriculum type gate ──────────────────────────────────────────
-- Drives which UI/API surface an institution sees. '844' is the
-- default so every existing seeded institution keeps behaving exactly
-- as it does today with zero behavior change from this migration alone.
ALTER TABLE institutions
  ADD COLUMN IF NOT EXISTS curriculum_type VARCHAR(20) NOT NULL DEFAULT '844';
  -- expected values for now: '844', 'cbe'

-- ─── Learning Areas ─────────────────────────────────────────────────
-- The correct CBC academic entity, root of the future curriculum
-- hierarchy. In v1.0 it behaves as an assessable entity, same role
-- "subjects" plays for 8-4-4 — deliberately NOT reusing the subjects
-- table, since KNEC codes / is_compulsory semantics don't map cleanly.
CREATE TABLE IF NOT EXISTS learning_areas (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id UUID NOT NULL REFERENCES institutions(id),
    name           VARCHAR(255) NOT NULL,
    code           VARCHAR(50),
    is_compulsory  BOOLEAN NOT NULL DEFAULT true,
    -- 'institution' = school-created; 'kicd_pack' = seeded from an
    -- official curriculum pack (v1.1 concept, column exists now so
    -- v1.1 never has to guess which existing rows are which).
    source         VARCHAR(20) NOT NULL DEFAULT 'institution',
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at     TIMESTAMPTZ,
    UNIQUE (institution_id, name)
);

-- ─── Curriculum rating bands ────────────────────────────────────────
-- A curriculum concept first, an institution customization second.
-- institution_id NULL = the official default for this curriculum_type
-- (+ grade_band, since Junior School and Senior School CBE use a
-- different number of bands — 4 vs 8 — confirmed from real report
-- cards, not assumed). institution_id set = that school's override.
-- Lookup order at read time: institution's own rows for this
-- curriculum_type+grade_band if any exist, else fall back to the
-- official (institution_id IS NULL) default rows.
CREATE TABLE IF NOT EXISTS curriculum_rating_bands (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    curriculum_type    VARCHAR(20) NOT NULL,
    grade_band         VARCHAR(20),  -- 'junior_school' | 'senior_school' | NULL
    institution_id     UUID REFERENCES institutions(id),  -- NULL = official default
    code               VARCHAR(10) NOT NULL,   -- 'EE1', 'ME2', 'A', etc.
    label              VARCHAR(100) NOT NULL,  -- 'Exceeding Expectation', etc.
    min_pct            NUMERIC(5,2) NOT NULL,
    max_pct            NUMERIC(5,2) NOT NULL,
    points             NUMERIC(4,2) NOT NULL,
    default_statement  TEXT,  -- teacher-editable suggestion, not an auto-comment
    sort_order         INT NOT NULL,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ─── Learning area assessment components (parallel to grade_components) ──
-- Institution-configurable small set per learning area/class/term
-- (e.g. Activity, Oral, Project, End Term). Teachers enter marks per
-- component; the system computes the final percentage — never a
-- single hand-calculated final number entered by the teacher.
CREATE TABLE IF NOT EXISTS learning_area_components (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id   UUID NOT NULL REFERENCES institutions(id),
    class_id         UUID NOT NULL REFERENCES classes(id),
    learning_area_id UUID NOT NULL REFERENCES learning_areas(id),
    term_id          UUID NOT NULL REFERENCES terms(id),
    name             VARCHAR(100) NOT NULL,
    weight           NUMERIC(5,2) NOT NULL,
    max_marks        NUMERIC(6,2) NOT NULL,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ─── Learning area grade entries (parallel to grade_entries) ──────────
CREATE TABLE IF NOT EXISTS learning_area_entries (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id UUID NOT NULL REFERENCES institutions(id),
    component_id   UUID NOT NULL REFERENCES learning_area_components(id),
    student_id     UUID NOT NULL REFERENCES students(id),
    marks          NUMERIC(6,2),
    is_absent      BOOLEAN NOT NULL DEFAULT false,
    entered_by     UUID REFERENCES users(id),
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (institution_id, component_id, student_id)
);

-- ─── Teacher assignment to learning areas ──────────────────────────
-- Mirrors the class_subjects concept (already in the schema, never
-- given CRUD endpoints per the earlier academics audit) but scoped to
-- learning areas so period-attendance authorization and gradebook
-- access have something real to check against for CBC classes.
CREATE TABLE IF NOT EXISTS learning_area_teacher_assignments (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id   UUID NOT NULL REFERENCES institutions(id),
    class_id         UUID NOT NULL REFERENCES classes(id),
    learning_area_id UUID NOT NULL REFERENCES learning_areas(id),
    teacher_id       UUID NOT NULL REFERENCES users(id),
    term_id          UUID NOT NULL REFERENCES terms(id),
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (institution_id, class_id, learning_area_id, term_id)
);

-- ─── CBC report cards — a genuinely different document, not a variant ──
-- report_cards (8-4-4) stays completely untouched: no new columns, no
-- shared rows. aggregate_marks/grade_points/class_position there and
-- total_points/mean_points here happen to be structurally similar
-- (both are "sum and average something per student"), but the source
-- data underneath is different enough that forcing one schema to serve
-- both was exactly what we agreed not to do.
CREATE TABLE IF NOT EXISTS learning_area_report_cards (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id          UUID NOT NULL REFERENCES institutions(id),
    student_id              UUID NOT NULL REFERENCES students(id),
    class_id                UUID NOT NULL REFERENCES classes(id),
    term_id                 UUID NOT NULL REFERENCES terms(id),
    total_points            NUMERIC(6,2),
    mean_points             NUMERIC(4,2),
    class_position          INT,
    stream_position         INT,
    attendance_days_present INT,
    attendance_days_total   INT,
    class_teacher_remark    TEXT,
    principal_remark        TEXT,
    status                  VARCHAR(20) NOT NULL DEFAULT 'draft',  -- draft | published
    published_at            TIMESTAMPTZ,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (institution_id, student_id, term_id)
);

-- ─── Per-learning-area lines on a CBC report card ───────────────────
-- rating_band_id is kept for traceability only. label/points/statement
-- are snapshotted onto the row itself at generation time so an edited
-- curriculum_rating_bands row never silently reinterprets an
-- already-published report card — same principle as report_cards
-- storing grade_points as a value, not deriving it live from a join.
CREATE TABLE IF NOT EXISTS learning_area_report_card_lines (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    report_card_id    UUID NOT NULL REFERENCES learning_area_report_cards(id) ON DELETE CASCADE,
    learning_area_id  UUID NOT NULL REFERENCES learning_areas(id),
    percentage        NUMERIC(5,2) NOT NULL,
    rating_band_id    UUID REFERENCES curriculum_rating_bands(id),
    rating_code       VARCHAR(10) NOT NULL,   -- snapshot, e.g. 'ME1'
    rating_label      VARCHAR(100) NOT NULL,  -- snapshot, e.g. 'Meets basic standards'
    rating_points     NUMERIC(4,2) NOT NULL,  -- snapshot
    statement         TEXT,                    -- teacher-editable, defaults from rating_band's default_statement
    teacher_id        UUID REFERENCES users(id),
    UNIQUE (report_card_id, learning_area_id)
);

-- ─── Seed the official CBE default bands ───────────────────────────
-- Sourced from KNEC's published KJSEA grading structure (the
-- 8-level/8-point scale is real and confirmed across multiple
-- independent reports). CAUTION: exact percentage cutoffs vary
-- slightly even across sources reporting the same KNEC document —
-- these are the most-corroborated numbers found, NOT verified against
-- KNEC's primary document. Treat as a reasonable starting default,
-- not an authoritative citation, until checked against the source.
INSERT INTO curriculum_rating_bands
  (curriculum_type, grade_band, institution_id, code, label, min_pct, max_pct, points, default_statement, sort_order)
VALUES
  ('cbe', 'senior_school', NULL, 'EE1', 'Exceeding Expectation', 90, 100, 8, 'Outstanding performance.', 1),
  ('cbe', 'senior_school', NULL, 'EE2', 'Exceeding Expectation', 75, 89,  7, 'Very good performance.', 2),
  ('cbe', 'senior_school', NULL, 'ME1', 'Meeting Expectation',   58, 74,  6, 'Meets the expected competency.', 3),
  ('cbe', 'senior_school', NULL, 'ME2', 'Meeting Expectation',   41, 57,  5, 'Meets the expected competency, fairly.', 4),
  ('cbe', 'senior_school', NULL, 'AE1', 'Approaching Expectation', 31, 40, 4, 'Approaching expectation; continued practice recommended.', 5),
  ('cbe', 'senior_school', NULL, 'AE2', 'Approaching Expectation', 21, 30, 3, 'Approaching expectation; needs more practice.', 6),
  ('cbe', 'senior_school', NULL, 'BE1', 'Below Expectation', 11, 20, 2, 'Requires additional support and improvement.', 7),
  ('cbe', 'senior_school', NULL, 'BE2', 'Below Expectation', 0,  10, 1, 'Requires significant support and improvement.', 8),
  -- Junior School default: 4 bands, no sub-levels, per New Bridgeport's
  -- real Grade 7 report card. Percentage cutoffs are that school's own
  -- convention, not independently verified against a KICD primary
  -- source — same caveat as above, arguably stronger here since this
  -- is a single data point, not a cross-corroborated national scale.
  ('cbe', 'junior_school', NULL, 'EE', 'Exceeding Expectation',   91, 100, 4, 'Outstanding performance.', 1),
  ('cbe', 'junior_school', NULL, 'ME', 'Meeting Expectation',    61, 90,  3, 'Meets the expected competency.', 2),
  ('cbe', 'junior_school', NULL, 'AE', 'Approaching Expectation', 31, 60,  2, 'Approaching expectation; continued practice recommended.', 3),
  ('cbe', 'junior_school', NULL, 'BE', 'Below Expectation',       0,  30,  1, 'Requires additional support and improvement.', 4);