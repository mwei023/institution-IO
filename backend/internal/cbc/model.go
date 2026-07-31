// Package cbc implements the v1.0 CBC/CBE academic layer — deliberately
// a separate package from `academics` (which remains the 8-4-4 engine),
// mirroring the DB-level separation. See migration
// 000009_create_cbc_academic_layer for the schema and full rationale.
//
// Explicitly out of scope here, reserved for the v1.1 Curriculum Engine:
// strands, sub-strands, learning outcomes, lesson plans, schemes of
// work, coverage tracking, competency mapping. Everything in this file
// is designed to remain valid and untouched once that engine attaches
// beneath LearningArea — additive, not destructive.
package cbc

import (
	"time"

	"github.com/google/uuid"
)

// LearningArea is the CBC root academic entity — plays the same role
// "Subject" plays for 8-4-4, but deliberately not the same table
// (KNEC codes / is_compulsory semantics don't map cleanly across the
// two curricula).
type LearningArea struct {
	ID            uuid.UUID `json:"id"`
	InstitutionID uuid.UUID `json:"institution_id"`
	Name          string    `json:"name"`
	Code          *string   `json:"code,omitempty"`
	IsCompulsory  bool      `json:"is_compulsory"`
	// Source distinguishes school-created rows from ones seeded by an
	// official curriculum pack — a v1.1 concept, tracked from day one
	// so v1.1 never has to guess which existing rows are which.
	Source    string    `json:"source"` // "institution" | "kicd_pack"
	CreatedAt time.Time `json:"created_at"`
}

// RatingBand is a curriculum concept first, an institution
// customization second. InstitutionID nil means this row is the
// official default for this CurriculumType+GradeBand; a non-nil value
// means a specific school's override. Lookup order belongs in the
// repository: prefer the institution's own rows, fall back to the
// official default rows.
type RatingBand struct {
	ID                uuid.UUID  `json:"id"`
	CurriculumType    string     `json:"curriculum_type"`      // "cbe", "844", etc.
	GradeBand         *string    `json:"grade_band,omitempty"` // "junior_school" | "senior_school" | nil
	InstitutionID     *uuid.UUID `json:"institution_id,omitempty"`
	Code              string     `json:"code"`  // "EE1", "ME2", etc.
	Label             string     `json:"label"` // "Exceeding Expectation"
	MinPct            float64    `json:"min_pct"`
	MaxPct            float64    `json:"max_pct"`
	Points            float64    `json:"points"`
	DefaultStatement  *string    `json:"default_statement,omitempty"`
	SortOrder         int        `json:"sort_order"`
	CreatedAt         time.Time  `json:"created_at"`
}

// Component is one configurable assessment component for a learning
// area (Activity, Oral, Project, End Term...) — same role
// academics.GradeComponent plays for 8-4-4, kept as a separate table
// rather than a shared one with a nullable discriminator column.
type Component struct {
	ID             uuid.UUID `json:"id"`
	InstitutionID  uuid.UUID `json:"institution_id"`
	ClassID        uuid.UUID `json:"class_id"`
	LearningAreaID uuid.UUID `json:"learning_area_id"`
	TermID         uuid.UUID `json:"term_id"`
	Name           string    `json:"name"`
	Weight         float64   `json:"weight"`
	MaxMarks       float64   `json:"max_marks"`
}

// Entry is one student's marks for one Component. The system computes
// the learning area's final percentage from these — a teacher never
// hand-calculates and enters a single final number.
type Entry struct {
	ID          uuid.UUID  `json:"id"`
	ComponentID uuid.UUID  `json:"component_id"`
	StudentID   uuid.UUID  `json:"student_id"`
	StudentName string     `json:"student_name"` // joined
	Marks       *float64   `json:"marks"`
	IsAbsent    bool       `json:"is_absent"`
	EnteredBy   *uuid.UUID `json:"entered_by,omitempty"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// TeacherAssignment links a teacher to a learning area they teach in a
// specific class/term — gives period-attendance authorization and
// gradebook access something real to check against for CBC classes,
// mirroring the (still CRUD-less) class_subjects concept for 8-4-4.
type TeacherAssignment struct {
	ID             uuid.UUID `json:"id"`
	InstitutionID  uuid.UUID `json:"institution_id"`
	ClassID        uuid.UUID `json:"class_id"`
	LearningAreaID uuid.UUID `json:"learning_area_id"`
	TeacherID      uuid.UUID `json:"teacher_id"`
	TermID         uuid.UUID `json:"term_id"`
	CreatedAt      time.Time `json:"created_at"`
}

// ReportCard is the CBC report — a genuinely different document from
// academics.ReportCard, not a variant of it. Structurally similar
// concepts (TotalPoints/MeanPoints here vs AggregateMarks/GradePoints
// there) but different enough underlying data that forcing one schema
// to serve both was explicitly rejected during design.
type ReportCard struct {
	ID                    uuid.UUID  `json:"id"`
	InstitutionID         uuid.UUID  `json:"institution_id"`
	StudentID             uuid.UUID  `json:"student_id"`
	StudentName           string     `json:"student_name"` // joined
	ClassID               uuid.UUID  `json:"class_id"`
	ClassName             string     `json:"class_name"` // joined
	TermID                uuid.UUID  `json:"term_id"`
	Term                  string     `json:"term"` // joined
	TotalPoints           *float64   `json:"total_points,omitempty"`
	MeanPoints            *float64   `json:"mean_points,omitempty"`
	ClassPosition         *int       `json:"class_position,omitempty"`
	StreamPosition        *int       `json:"stream_position,omitempty"`
	AttendanceDaysPresent int        `json:"attendance_days_present"`
	AttendanceDaysTotal   int        `json:"attendance_days_total"`
	ClassTeacherRemark    *string    `json:"class_teacher_remark,omitempty"`
	PrincipalRemark       *string    `json:"principal_remark,omitempty"`
	Status                string     `json:"status"` // draft | published
	PublishedAt           *time.Time `json:"published_at,omitempty"`
	Lines                 []*ReportCardLine `json:"lines,omitempty"`
}

// ReportCardLine is one learning area's row on a CBC report card.
// Label/Points/Statement are snapshotted at generation time — NOT
// derived live from RatingBandID — so editing curriculum_rating_bands
// later never silently reinterprets an already-published report card.
// RatingBandID is kept only for traceability.
type ReportCardLine struct {
	ID               uuid.UUID  `json:"id"`
	ReportCardID     uuid.UUID  `json:"report_card_id"`
	LearningAreaID   uuid.UUID  `json:"learning_area_id"`
	LearningAreaName string     `json:"learning_area_name"` // joined
	Percentage       float64    `json:"percentage"`
	RatingBandID     *uuid.UUID `json:"rating_band_id,omitempty"`
	RatingCode       string     `json:"rating_code"`   // snapshot, e.g. "ME1"
	RatingLabel      string     `json:"rating_label"`  // snapshot
	RatingPoints     float64    `json:"rating_points"` // snapshot
	Statement        *string    `json:"statement,omitempty"`
	TeacherID        *uuid.UUID `json:"teacher_id,omitempty"`
}

// ─── Request/response shapes ────────────────────────────────────────

type CreateLearningAreaRequest struct {
	Name         string  `json:"name" validate:"required"`
	Code         *string `json:"code,omitempty"`
	IsCompulsory bool    `json:"is_compulsory"`
}

type CreateComponentRequest struct {
	ClassID        uuid.UUID `json:"class_id" validate:"required"`
	LearningAreaID uuid.UUID `json:"learning_area_id" validate:"required"`
	TermID         uuid.UUID `json:"term_id" validate:"required"`
	Name           string    `json:"name" validate:"required"`
	Weight         float64   `json:"weight" validate:"required,gt=0,lte=100"`
	MaxMarks       float64   `json:"max_marks" validate:"required,gt=0"`
}

type BulkEntryRequest struct {
	ComponentID uuid.UUID     `json:"component_id" validate:"required"`
	Entries     []EntryInput  `json:"entries" validate:"required"`
}

type EntryInput struct {
	StudentID uuid.UUID `json:"student_id" validate:"required"`
	Marks     *float64  `json:"marks"`
	IsAbsent  bool      `json:"is_absent"`
}

type GenerateReportCardsForClassRequest struct {
	ClassID uuid.UUID `json:"class_id" validate:"required"`
	TermID  uuid.UUID `json:"term_id" validate:"required"`
}

type PublishReportCardRequest struct {
	StudentID uuid.UUID `json:"student_id" validate:"required"`
	TermID    uuid.UUID `json:"term_id" validate:"required"`
}

// BulkReportCardResult mirrors academics.BulkReportCardResult's
// partial-success semantics deliberately — same principle: process
// every student, record failures, never abort the whole batch on one.
type BulkReportCardResult struct {
	ClassID        uuid.UUID          `json:"class_id"`
	TermID         uuid.UUID          `json:"term_id"`
	TotalStudents  int                `json:"total_students"`
	SucceededCount int                `json:"succeeded_count"`
	FailedCount    int                `json:"failed_count"`
	Succeeded      []*ReportCard      `json:"succeeded"`
	Failed         []BulkFailure      `json:"failed"`
}

type BulkFailure struct {
	StudentID uuid.UUID `json:"student_id"`
	Reason    string    `json:"reason"`
}