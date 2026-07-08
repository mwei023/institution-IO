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
	ClassTeacher    *string    `json:"class_teacher,omitempty"` // computed: teacher name
	Capacity        int        `json:"capacity"`
	Enrollment      int        `json:"enrollment"` // computed
	CreatedAt       time.Time  `json:"created_at"`
}

// Subject represents a school subject.
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

// TimetablePeriod is a single slot in the weekly timetable.
type TimetablePeriod struct {
	ID            uuid.UUID `json:"id"`
	InstitutionID uuid.UUID `json:"institution_id"`
	TermID        uuid.UUID `json:"term_id"`
	ClassID       uuid.UUID `json:"class_id"`
	SubjectID     uuid.UUID `json:"subject_id"`
	TeacherID     uuid.UUID `json:"teacher_id"`
	SubjectName   string    `json:"subject_name"`  // joined
	TeacherName   string    `json:"teacher_name"`  // joined
	DayOfWeek     int       `json:"day_of_week"`  // 1=Mon…5=Fri
	PeriodNumber  int       `json:"period_number"`
	StartTime     string    `json:"start_time"`   // HH:MM
	EndTime       string    `json:"end_time"`
	Room          *string   `json:"room,omitempty"`
	IsPublished   bool      `json:"is_published"`
}

// AttendanceSession is one attendance-taking event (daily or per period).
type AttendanceSession struct {
	ID            uuid.UUID  `json:"id"`
	InstitutionID uuid.UUID  `json:"institution_id"`
	ClassID       uuid.UUID  `json:"class_id"`
	ClassName     string     `json:"class_name"` // joined
	TeacherID     uuid.UUID  `json:"teacher_id"`
	SessionDate   time.Time  `json:"session_date"`
	SessionType   string     `json:"session_type"` // daily | period
	Status        string     `json:"status"`       // open | submitted | locked
	SubmittedAt   *time.Time `json:"submitted_at,omitempty"`
	Records       []*AttendanceRecord `json:"records,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
}

// AttendanceRecord is one student's status in a session.
type AttendanceRecord struct {
	ID        uuid.UUID `json:"id"`
	StudentID uuid.UUID `json:"student_id"`
	Name      string    `json:"name"`     // joined: student full name
	Status    string    `json:"status"`   // present | absent | late | excused
	Remarks   *string   `json:"remarks,omitempty"`
}

// TakeAttendanceRequest is the body to submit a full session's attendance.
type TakeAttendanceRequest struct {
	ClassID     uuid.UUID           `json:"class_id" validate:"required"`
	SessionDate string              `json:"session_date" validate:"required"` // YYYY-MM-DD
	SessionType string              `json:"session_type" validate:"required,oneof=daily period"`
	Records     []AttendanceEntry   `json:"records" validate:"required,min=1"`
}

// AttendanceEntry is one student's status in a TakeAttendanceRequest.
type AttendanceEntry struct {
	StudentID uuid.UUID `json:"student_id" validate:"required"`
	Status    string    `json:"status" validate:"required,oneof=present absent late excused"`
	Remarks   string    `json:"remarks"`
}

// AttendanceSummary is the aggregate attendance for a student over a period.
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

// GradeComponent is a weighted assessment (CAT 1, Midterm, etc.).
type GradeComponent struct {
	ID            uuid.UUID `json:"id"`
	InstitutionID uuid.UUID `json:"institution_id"`
	ClassID       uuid.UUID `json:"class_id"`
	SubjectID     uuid.UUID `json:"subject_id"`
	TermID        uuid.UUID `json:"term_id"`
	Name          string    `json:"name"`
	Weight        float64   `json:"weight"`   // percentage
	MaxMarks      float64   `json:"max_marks"`
}

// GradeEntry is a single student's mark for one component.
type GradeEntry struct {
	ID          uuid.UUID  `json:"id"`
	ComponentID uuid.UUID  `json:"component_id"`
	StudentID   uuid.UUID  `json:"student_id"`
	StudentName string     `json:"student_name"` // joined
	Marks       *float64   `json:"marks"`
	IsAbsent    bool       `json:"is_absent"`
	EnteredBy   *uuid.UUID `json:"entered_by,omitempty"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// GradebookRow is one student's grades across all components in a subject/term.
type GradebookRow struct {
	StudentID   uuid.UUID          `json:"student_id"`
	StudentName string             `json:"student_name"`
	Components  map[string]float64 `json:"components"` // component name → marks
	Total       float64            `json:"total"`
	Grade       string             `json:"grade"`
}

// BulkGradeEntryRequest is the body to save marks for a whole class.
type BulkGradeEntryRequest struct {
	ComponentID uuid.UUID    `json:"component_id" validate:"required"`
	Entries     []GradeInput `json:"entries" validate:"required"`
}

// GradeInput is one row in a bulk grade entry.
type GradeInput struct {
	StudentID uuid.UUID `json:"student_id" validate:"required"`
	Marks     *float64  `json:"marks"`
	IsAbsent  bool      `json:"is_absent"`
}

// ReportCard is a compiled end-of-term report for a student.
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
	Status                string    `json:"status"` // draft | published
	PublishedAt           *time.Time `json:"published_at,omitempty"`
}

// ClassFilter holds query parameters for filtering class lists.
type ClassFilter struct {
	AcademicYearID *uuid.UUID
	Level          *int
	Page           int
	PageSize       int
}

// CreateGradeComponentRequest is the body to define a new assessment component.
type CreateGradeComponentRequest struct {
	ClassID   uuid.UUID `json:"class_id" validate:"required"`
	SubjectID uuid.UUID `json:"subject_id" validate:"required"`
	TermID    uuid.UUID `json:"term_id" validate:"required"`
	Name      string    `json:"name" validate:"required"`
	Weight    float64   `json:"weight" validate:"required,gt=0,lte=100"`
	MaxMarks  float64   `json:"max_marks" validate:"required,gt=0"`
}

// GenerateReportCardRequest is the body to trigger report card generation.
type GenerateReportCardRequest struct {
	StudentID uuid.UUID `json:"student_id" validate:"required"`
	ClassID   uuid.UUID `json:"class_id" validate:"required"`
	TermID    uuid.UUID `json:"term_id" validate:"required"`
}

// GenerateReportCardsForClassRequest is the body to bulk-generate report
// cards for every active student in a class.
type GenerateReportCardsForClassRequest struct {
	ClassID uuid.UUID `json:"class_id" validate:"required"`
	TermID  uuid.UUID `json:"term_id" validate:"required"`
}

// BulkReportCardResult summarizes the outcome of generating report cards
// for a whole class. Processing never aborts on the first failure — every
// student that can succeed does, and every student that can't is recorded
// here with a reason.
type BulkReportCardResult struct {
	ClassID        uuid.UUID                `json:"class_id"`
	TermID         uuid.UUID                `json:"term_id"`
	TotalStudents  int                      `json:"total_students"`
	SucceededCount int                      `json:"succeeded_count"`
	FailedCount    int                      `json:"failed_count"`
	Succeeded      []*ReportCard            `json:"succeeded"`
	Failed         []BulkReportCardFailure  `json:"failed"`
}

// BulkReportCardFailure records why one student's report card could not
// be generated during a bulk run.
type BulkReportCardFailure struct {
	StudentID uuid.UUID `json:"student_id"`
	Reason    string    `json:"reason"`
}

// PublishReportCardRequest is the body to publish a single student's
// report card for a term.
type PublishReportCardRequest struct {
	StudentID uuid.UUID `json:"student_id" validate:"required"`
	TermID    uuid.UUID `json:"term_id" validate:"required"`
}