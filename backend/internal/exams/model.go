package exams

import (
	"time"

	"github.com/google/uuid"
)

const (
	SessionDraft     = "draft"
	SessionValidated = "validated"
	SessionPublished = "published"
	SessionArchived  = "archived"

	SittingDraft     = "draft"
	SittingPublished = "published"
	SittingCancelled = "cancelled"
)

// ExamSession is one scheduled examination event (mid-term, end-of-term,
// mock, internal assessment, programme-specific period, or an external
// exam the institution plans locally). One canonical schedule per session.
type ExamSession struct {
	ID             uuid.UUID  `json:"id"`
	InstitutionID  uuid.UUID  `json:"institution_id"`
	AcademicYearID *uuid.UUID `json:"academic_year_id,omitempty"`
	TermID         *uuid.UUID `json:"term_id,omitempty"`
	Name           string     `json:"name"`
	ExamType       string     `json:"exam_type"`
	StartDate      string     `json:"start_date"`
	EndDate        string     `json:"end_date"`
	Status         string     `json:"status"`
	Version        int        `json:"version"`
	PublishedAt    *time.Time `json:"published_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// ExamPaper is one paper/component. Many papers may reference the same
// academic offering; offering is curriculum-neutral (subject or learning
// area behind academic_offerings, or NULL for a free-text paper).
type ExamPaper struct {
	ID               uuid.UUID  `json:"id"`
	InstitutionID    uuid.UUID  `json:"institution_id"`
	ExamSessionID    uuid.UUID  `json:"exam_session_id"`
	AcademicOfferingID *uuid.UUID `json:"academic_offering_id,omitempty"`
	OfferingName     *string    `json:"offering_name,omitempty"`
	OfferingType     *string    `json:"offering_type,omitempty"`
	PaperName        string     `json:"paper_name"`
	PaperCode        *string    `json:"paper_code,omitempty"`
	PaperNumber      *string    `json:"paper_number,omitempty"`
	DurationMinutes  int        `json:"duration_minutes"`
	MaxMarks         *float64   `json:"max_marks,omitempty"`
	Instructions     *string    `json:"instructions,omitempty"`
	IsShared         bool       `json:"is_shared"`
}

// PaperGroup is one candidate group sitting a paper. class_id reuses the
// existing classes table; label covers programme/pathway/level/year/stage
// groups with no dedicated class row. A paper may have many groups.
type PaperGroup struct {
	ID             uuid.UUID  `json:"id"`
	InstitutionID  uuid.UUID  `json:"institution_id"`
	ExamPaperID    uuid.UUID  `json:"exam_paper_id"`
	ClassID        *uuid.UUID `json:"class_id,omitempty"`
	ClassName      *string    `json:"class_name,omitempty"`
	Label          *string    `json:"label,omitempty"`
	CandidateCount *int       `json:"candidate_count,omitempty"`
}

// ExamSitting places a paper at a date/time + room. The same paper may
// have several sittings (different rooms/groups).
type ExamSitting struct {
	ID             uuid.UUID  `json:"id"`
	InstitutionID  uuid.UUID  `json:"institution_id"`
	ExamSessionID  uuid.UUID  `json:"exam_session_id"`
	ExamPaperID    uuid.UUID  `json:"exam_paper_id"`
	ExamDate       string     `json:"exam_date"`
	StartTime      string     `json:"start_time"`
	EndTime        string     `json:"end_time"`
	RoomName       *string    `json:"room_name,omitempty"`
	RoomCapacity   *int       `json:"room_capacity,omitempty"`
	CandidateCount *int       `json:"candidate_count,omitempty"`
	SeatingNote    *string    `json:"seating_note,omitempty"`
	Status         string     `json:"status"`
}

// InvigilatorAssignment links a staff user to a sitting.
type InvigilatorAssignment struct {
	ID            uuid.UUID `json:"id"`
	InstitutionID uuid.UUID `json:"institution_id"`
	ExamSittingID uuid.UUID `json:"exam_sitting_id"`
	StaffID       uuid.UUID `json:"staff_id"`
	StaffName     *string   `json:"staff_name,omitempty"`
	Role          string    `json:"role"`
	Notes         *string   `json:"notes,omitempty"`
}

// SpecialArrangement is an extensible scheduling accommodation. Only
// scheduling-relevant fields are stored — no medical detail.
type SpecialArrangement struct {
	ID              uuid.UUID  `json:"id"`
	InstitutionID   uuid.UUID  `json:"institution_id"`
	ExamSittingID   uuid.UUID  `json:"exam_sitting_id"`
	StudentID       *uuid.UUID `json:"student_id,omitempty"`
	ClassID         *uuid.UUID `json:"class_id,omitempty"`
	ArrangementType string     `json:"arrangement_type"`
	Details         *string    `json:"details,omitempty"`
	ExtraMinutes    *int       `json:"extra_minutes,omitempty"`
	RoomName        *string    `json:"room_name,omitempty"`
}

// SittingView is the canonical enriched row every view derives from.
type SittingView struct {
	Sitting          *ExamSitting             `json:"sitting"`
	Paper            *ExamPaper               `json:"paper"`
	Groups           []*PaperGroup            `json:"groups"`
	Invigilators     []*InvigilatorAssignment `json:"invigilators"`
	Arrangements     []*SpecialArrangement    `json:"arrangements,omitempty"`
	CapacityStatus   string                   `json:"capacity_status"`
}

// ValidationIssue is one blocking error or warning.
type ValidationIssue struct {
	Code      string `json:"code"`
	Severity  string `json:"severity"` // "blocking" | "warning"
	Message   string `json:"message"`
	SittingID *string `json:"sitting_id,omitempty"`
}

// ValidationResult is the outcome of validating a draft schedule.
type ValidationResult struct {
	SessionID     string            `json:"session_id"`
	Valid         bool              `json:"valid"`
	Blocking      []*ValidationIssue `json:"blocking"`
	Warnings      []*ValidationIssue `json:"warnings"`
	CheckedAt     time.Time         `json:"checked_at"`
	SittingsCount int               `json:"sittings_count"`
}

// ── Requests ──

type CreateSessionRequest struct {
	AcademicYearID *uuid.UUID `json:"academic_year_id,omitempty"`
	TermID         *uuid.UUID `json:"term_id,omitempty"`
	Name           string     `json:"name"`
	ExamType       string     `json:"exam_type"`
	StartDate      string     `json:"start_date"`
	EndDate        string     `json:"end_date"`
}

type CreatePaperRequest struct {
	AcademicOfferingID *uuid.UUID `json:"academic_offering_id,omitempty"`
	PaperName          string     `json:"paper_name"`
	PaperCode          *string    `json:"paper_code,omitempty"`
	PaperNumber        *string    `json:"paper_number,omitempty"`
	DurationMinutes    int        `json:"duration_minutes"`
	MaxMarks           *float64   `json:"max_marks,omitempty"`
	Instructions       *string    `json:"instructions,omitempty"`
	IsShared           bool       `json:"is_shared,omitempty"`
}

type AddPaperGroupRequest struct {
	ClassID        *uuid.UUID `json:"class_id,omitempty"`
	Label          *string    `json:"label,omitempty"`
	CandidateCount *int       `json:"candidate_count,omitempty"`
}

type CreateSittingRequest struct {
	ExamPaperID    uuid.UUID `json:"exam_paper_id"`
	ExamDate       string    `json:"exam_date"`
	StartTime      string    `json:"start_time"`
	EndTime        string    `json:"end_time"`
	RoomName       *string   `json:"room_name,omitempty"`
	RoomCapacity   *int      `json:"room_capacity,omitempty"`
	CandidateCount *int      `json:"candidate_count,omitempty"`
	SeatingNote    *string   `json:"seating_note,omitempty"`
}

type UpdateSittingRequest struct {
	ExamDate       *string `json:"exam_date,omitempty"`
	StartTime      *string `json:"start_time,omitempty"`
	EndTime        *string `json:"end_time,omitempty"`
	RoomName       *string `json:"room_name,omitempty"`
	RoomCapacity   *int    `json:"room_capacity,omitempty"`
	CandidateCount *int    `json:"candidate_count,omitempty"`
	SeatingNote    *string `json:"seating_note,omitempty"`
}

type AssignInvigilatorRequest struct {
	StaffID uuid.UUID `json:"staff_id"`
	Role    string    `json:"role"`
	Notes   *string   `json:"notes,omitempty"`
}

type AddArrangementRequest struct {
	StudentID       *uuid.UUID `json:"student_id,omitempty"`
	ClassID         *uuid.UUID `json:"class_id,omitempty"`
	ArrangementType string     `json:"arrangement_type"`
	Details         *string    `json:"details,omitempty"`
	ExtraMinutes    *int       `json:"extra_minutes,omitempty"`
	RoomName        *string    `json:"room_name,omitempty"`
}

type GenerateDraftRequest struct {
	StartDate        string `json:"start_date"`
	EndDate          string `json:"end_date"`
	DailyStartTime   string `json:"daily_start_time"`
	DailyEndTime     string `json:"daily_end_time"`
	GapMinutes       int    `json:"gap_minutes"`
	MaxSittingsPerDay int   `json:"max_sittings_per_day,omitempty"`
}

type GenerateDraftResult struct {
	Scheduled   []*ExamSitting `json:"scheduled"`
	Unscheduled []UnscheduledItem `json:"unscheduled"`
}

type UnscheduledItem struct {
	PaperID uuid.UUID `json:"paper_id"`
	Reason  string    `json:"reason"`
}
