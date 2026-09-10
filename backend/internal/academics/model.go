package academics

import (
	"time"

	"github.com/google/uuid"
)

// Class represents a form/stream combination for an academic year.
type Class struct {
	ID              uuid.UUID  `json:"id"`
	InstitutionID   uuid.UUID  `json:"institution_id"`
	AcademicYearID  uuid.UUID  `json:"academic_year_id"`
	Name            string     `json:"name"`
	Stream          *string    `json:"stream,omitempty"`
	Level           int        `json:"level"`
	ClassTeacherID  *uuid.UUID `json:"class_teacher_id,omitempty"`
	ClassTeacher    *string    `json:"class_teacher,omitempty"`
	Capacity        int        `json:"capacity"`
	Enrollment      int        `json:"enrollment"`
	CreatedAt       time.Time  `json:"created_at"`
}

type Subject struct {
	ID             uuid.UUID `json:"id"`
	InstitutionID  uuid.UUID `json:"institution_id"`
	DepartmentID   *uuid.UUID `json:"department_id,omitempty"`
	Code           string    `json:"code"`
	Name           string    `json:"name"`
	KNECCode       *string   `json:"knec_code,omitempty"`
	IsCompulsory   bool      `json:"is_compulsory"`
	CreditHoursPW  int       `json:"credit_hours_pw"`
}

type TimetablePeriod struct {
	ID                   uuid.UUID  `json:"id"`
	InstitutionID        uuid.UUID  `json:"institution_id"`
	TermID               uuid.UUID  `json:"term_id"`
	ClassID              uuid.UUID  `json:"class_id"`
	TeachingAssignmentID *uuid.UUID `json:"teaching_assignment_id,omitempty"`
	TeacherID            *uuid.UUID `json:"teacher_id,omitempty"`
	TeacherName          *string    `json:"teacher_name,omitempty"`
	OfferingName         *string    `json:"offering_name,omitempty"`
	OfferingType         *string    `json:"offering_type,omitempty"` // "subject" | "learning_area"
	DayOfWeek            int        `json:"day_of_week"`
	PeriodNumber         int        `json:"period_number"`
	StartTime            string     `json:"start_time"`
	EndTime              string     `json:"end_time"`
	Room                 *string    `json:"room,omitempty"`
	IsPublished          bool       `json:"is_published"`
}

type AttendanceSession struct {
	ID            uuid.UUID  `json:"id"`
	InstitutionID uuid.UUID  `json:"institution_id"`
	ClassID       uuid.UUID  `json:"class_id"`
	ClassName     string     `json:"class_name"`
	TeacherID     uuid.UUID  `json:"teacher_id"`
	PeriodID      *uuid.UUID `json:"period_id,omitempty"`
	SessionDate   time.Time  `json:"session_date"`
	SessionType   string     `json:"session_type"`
	Status        string     `json:"status"`
	SubmittedAt   *time.Time `json:"submitted_at,omitempty"`
	Records       []*AttendanceRecord `json:"records"`

	CreatedAt     time.Time  `json:"created_at"`
}

type AttendanceRecord struct {
	ID        uuid.UUID `json:"id"`
	StudentID uuid.UUID `json:"student_id"`
	Name      string    `json:"name"`
	Status    string    `json:"status"`
	Remarks   *string   `json:"remarks,omitempty"`
}

type TakeAttendanceRequest struct {
	ClassID     uuid.UUID           `json:"class_id" validate:"required"`
	SessionDate string              `json:"session_date" validate:"required"`
	SessionType string              `json:"session_type" validate:"required,oneof=daily period"`
	PeriodID    *uuid.UUID        `json:"period_id,omitempty"`
	Records     []AttendanceEntry   `json:"records" validate:"required,min=1"`
}

type AttendanceEntry struct {
	StudentID uuid.UUID `json:"student_id" validate:"required"`
	Status    string    `json:"status" validate:"required,oneof=present absent late excused"`
	Remarks   string    `json:"remarks"`
}

type AttendanceSummary struct {
	StudentID   uuid.UUID `json:"student_id"`
	StudentName string    `json:"student_name"`
	Present     int       `json:"present"`
	Absent      int       `json:"absent"`
	Late        int       `json:"late"`
	Excused     int       `json:"excused"`
	Total       int       `json:"total"`
	Percentage  float64   `json:"percentage"`
}

// InstitutionAttendanceToday summarizes today's daily attendance across the
// whole institution, for the principal dashboard. "Present" here strictly
// means a record exists for today with that status — a student with no
// session recorded yet is not counted as absent, and RecordsTaken lets the
// UI distinguish "no one's marked attendance yet this morning" from "a bad
// attendance day" instead of collapsing both into 0%.
type InstitutionAttendanceToday struct {
	TotalActiveStudents int     `json:"total_active_students"`
	RecordsTaken        int     `json:"records_taken"`
	Present             int     `json:"present"`
	Absent              int     `json:"absent"`
	Late                int     `json:"late"`
	Excused             int     `json:"excused"`
	AttendanceRate      float64 `json:"attendance_rate"` // 0 if RecordsTaken is 0
}

type GradeComponent struct {
	ID            uuid.UUID `json:"id"`
	InstitutionID uuid.UUID `json:"institution_id"`
	ClassID       uuid.UUID `json:"class_id"`
	SubjectID     uuid.UUID `json:"subject_id"`
	TermID        uuid.UUID `json:"term_id"`
	Name          string    `json:"name"`
	Weight        float64   `json:"weight"`
	MaxMarks      float64   `json:"max_marks"`
}

type GradeEntry struct {
	ID          uuid.UUID  `json:"id"`
	ComponentID uuid.UUID  `json:"component_id"`
	StudentID   uuid.UUID  `json:"student_id"`
	StudentName string     `json:"student_name"`
	Marks       *float64   `json:"marks"`
	IsAbsent    bool       `json:"is_absent"`
	EnteredBy   *uuid.UUID `json:"entered_by,omitempty"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type GradebookRow struct {
	StudentID   uuid.UUID          `json:"student_id"`
	StudentName string             `json:"student_name"`
	Components  map[string]float64 `json:"components"`
	Total       float64            `json:"total"`
	Grade       string             `json:"grade"`
}

type BulkGradeEntryRequest struct {
	ComponentID uuid.UUID    `json:"component_id" validate:"required"`
	Entries     []GradeInput `json:"entries" validate:"required"`
}

type GradeInput struct {
	StudentID uuid.UUID `json:"student_id" validate:"required"`
	Marks     *float64  `json:"marks"`
	IsAbsent  bool      `json:"is_absent"`
}

type ReportCard struct {
	ID                    uuid.UUID `json:"id"`
	StudentID             uuid.UUID `json:"student_id"`
	StudentName           string    `json:"student_name"`
	ClassName             string    `json:"class_name"`
	Term                  string    `json:"term"`
	AggregateMarks        float64   `json:"aggregate_marks"`
	GradePoints           float64   `json:"grade_points"`
	ClassPosition         *int      `json:"class_position,omitempty"`
	StreamPosition        *int      `json:"stream_position,omitempty"`
	AttendanceDaysPresent int       `json:"attendance_days_present"`
	AttendanceDaysTotal   int       `json:"attendance_days_total"`
	TeacherComment        *string   `json:"teacher_comment,omitempty"`
	PrincipalComment      *string   `json:"principal_comment,omitempty"`
	Status                string    `json:"status"`
	PublishedAt           *time.Time `json:"published_at,omitempty"`
}

type ClassFilter struct {
	AcademicYearID *uuid.UUID
	Level          *int
	Page           int
	PageSize       int
}

type CreateGradeComponentRequest struct {
	ClassID   uuid.UUID `json:"class_id" validate:"required"`
	SubjectID uuid.UUID `json:"subject_id" validate:"required"`
	TermID    uuid.UUID `json:"term_id" validate:"required"`
	Name      string    `json:"name" validate:"required"`
	Weight    float64   `json:"weight" validate:"required,gt=0,lte=100"`
	MaxMarks  float64   `json:"max_marks" validate:"required,gt=0"`
}

type GenerateReportCardRequest struct {
	StudentID uuid.UUID `json:"student_id" validate:"required"`
	ClassID   uuid.UUID `json:"class_id" validate:"required"`
	TermID    uuid.UUID `json:"term_id" validate:"required"`
}

type GenerateReportCardsForClassRequest struct {
	ClassID uuid.UUID `json:"class_id" validate:"required"`
	TermID  uuid.UUID `json:"term_id" validate:"required"`
}

type BulkReportCardResult struct {
	ClassID        uuid.UUID                `json:"class_id"`
	TermID         uuid.UUID                `json:"term_id"`
	TotalStudents  int                      `json:"total_students"`
	SucceededCount int                      `json:"succeeded_count"`
	FailedCount    int                      `json:"failed_count"`
	Succeeded      []*ReportCard            `json:"succeeded"`
	Failed         []BulkReportCardFailure  `json:"failed"`
}

type BulkReportCardFailure struct {
	StudentID uuid.UUID `json:"student_id"`
	Reason    string    `json:"reason"`
}

type PublishReportCardRequest struct {
	StudentID uuid.UUID `json:"student_id" validate:"required"`
	TermID    uuid.UUID `json:"term_id" validate:"required"`
}

type CreateClassRequest struct {
	AcademicYearID uuid.UUID  `json:"academic_year_id" validate:"required"`
	Name           string     `json:"name" validate:"required"`
	Stream         *string    `json:"stream,omitempty"`
	Level          int        `json:"level" validate:"required"`
	Capacity       int        `json:"capacity" validate:"required,gt=0"`
	ClassTeacherID *uuid.UUID `json:"class_teacher_id,omitempty"` // optional at creation
}

type AssignClassTeacherRequest struct {
	ClassTeacherID uuid.UUID `json:"class_teacher_id" validate:"required"`
}

type TeacherOption struct {
	ID       uuid.UUID `json:"id"`
	FullName string    `json:"full_name"`
}

type AcademicYear struct {
	ID            uuid.UUID `json:"id"`
	InstitutionID uuid.UUID `json:"institution_id"`
	Name          string    `json:"name"`
	StartDate     time.Time `json:"start_date"`
	EndDate       time.Time `json:"end_date"`
	IsCurrent     bool      `json:"is_current"`
}

// academics/model.go — new request/response shapes
// academics/model.go — new request/response shapes
//
// CreateTeachingAssignmentRequest accepts exactly one of SubjectID or
// LearningAreaID — mirrors academic_offerings' own check constraint.
// Validated in Service.CreateTeachingAssignment, not via struct tags,
// since "exactly one of two optional fields" isn't expressible with
// validator's built-in rules alone.
type CreateTeachingAssignmentRequest struct {
	ClassID        uuid.UUID  `json:"class_id" validate:"required"`
	SubjectID      *uuid.UUID `json:"subject_id,omitempty"`
	LearningAreaID *uuid.UUID `json:"learning_area_id,omitempty"`
	TeacherID      uuid.UUID  `json:"teacher_id" validate:"required"`
	TermID         uuid.UUID  `json:"term_id" validate:"required"`
}

type TeachingAssignment struct {
	ID                 uuid.UUID `json:"id"`
	InstitutionID      uuid.UUID `json:"institution_id"`
	ClassID            uuid.UUID `json:"class_id"`
	AcademicOfferingID uuid.UUID `json:"academic_offering_id"`
	TeacherID          uuid.UUID `json:"teacher_id"`
	TermID             uuid.UUID `json:"term_id"`
	CreatedAt          time.Time `json:"created_at"`
}

// TeachingAssignmentOption is the display-ready shape for populating a
// "which lesson is this period?" dropdown when creating a timetable
// period — resolves the offering (subject or learning area) and
// teacher name so the frontend never has to make three separate calls
// to build one <select>.
type TeachingAssignmentOption struct {
	ID           uuid.UUID `json:"id"`
	OfferingName string    `json:"offering_name"`
	OfferingType string    `json:"offering_type"` // "subject" | "learning_area"
	TeacherID    uuid.UUID `json:"teacher_id"`
	TeacherName  string    `json:"teacher_name"`
}

type CreateTimetablePeriodRequest struct {
	ClassID              uuid.UUID `json:"class_id" validate:"required"`
	TermID               uuid.UUID `json:"term_id" validate:"required"`
	TeachingAssignmentID uuid.UUID `json:"teaching_assignment_id" validate:"required"`
	DayOfWeek            int       `json:"day_of_week" validate:"required,min=1,max=5"`
	PeriodNumber         int       `json:"period_number" validate:"required,min=1,max=10"`
	StartTime            string    `json:"start_time" validate:"required"` // "HH:MM", 24h
	EndTime              string    `json:"end_time" validate:"required"`
	Room                 *string   `json:"room,omitempty"`
}