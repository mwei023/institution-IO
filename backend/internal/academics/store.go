package academics

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Store is everything academics.Service needs from a repository — same
// idea as cbc.Store. The real, database-backed Repository struct
// already has every one of these methods, so *Repository satisfies
// Store automatically. This interface exists only so tests can swap in
// a fake implementation instead of a real database connection.
type Store interface {
	// Classes
	ListClasses(ctx context.Context, institutionID uuid.UUID, f *ClassFilter) ([]*Class, error)
	GetClassByID(ctx context.Context, institutionID, classID uuid.UUID) (*Class, error)
	CreateClass(ctx context.Context, institutionID uuid.UUID, req *CreateClassRequest) (*Class, error)
	AssignClassTeacher(ctx context.Context, institutionID, classID, teacherID uuid.UUID) error
	ListTeachers(ctx context.Context, institutionID uuid.UUID) ([]*TeacherOption, error)
	// academics/store.go — add to Store interface
	CreateTeachingAssignment(ctx context.Context, institutionID uuid.UUID, req *CreateTeachingAssignmentRequest) (*TeachingAssignment, error)
	ListAcademicYears(ctx context.Context, institutionID uuid.UUID) ([]*AcademicYear, error)

	// Subjects
	ListSubjects(ctx context.Context, institutionID uuid.UUID) ([]*Subject, error)

	// Timetable
	GetCurrentTerm(ctx context.Context, institutionID uuid.UUID) (*uuid.UUID, error)
	GetTimetableForClass(ctx context.Context, institutionID, classID, termID uuid.UUID) ([]*TimetablePeriod, error)
	GetTimetableForTeacher(ctx context.Context, institutionID, teacherID, termID uuid.UUID) ([]*TimetablePeriod, error)
	GetTimetablePeriodByID(ctx context.Context, institutionID, periodID uuid.UUID) (*TimetablePeriodInfo, error)

	// Attendance
	GetOrCreateSession(ctx context.Context, institutionID, classID, teacherID uuid.UUID, sessionDate time.Time, sessionType string, periodID *uuid.UUID) (*AttendanceSession, error)
	GetSessionStudents(ctx context.Context, institutionID, classID, sessionID uuid.UUID) ([]*AttendanceRecord, error)
	SaveAttendanceRecords(ctx context.Context, institutionID, sessionID uuid.UUID, entries []AttendanceEntry) error
	GetAttendanceSummary(ctx context.Context, institutionID, classID uuid.UUID, startDate, endDate time.Time) ([]*AttendanceSummary, error)
	GetActiveStudentIDsForClass(ctx context.Context, institutionID, classID uuid.UUID) ([]uuid.UUID, error)
	GetPrimaryContactForStudent(ctx context.Context, institutionID, studentID uuid.UUID) (*PrimaryContactInfo, error)

	// Grades — the methods our tests actually exercise
	ListGradeComponents(ctx context.Context, institutionID, classID, subjectID, termID uuid.UUID) ([]*GradeComponent, error)
	GetGradeComponentByID(ctx context.Context, institutionID, componentID uuid.UUID) (*GradeComponent, error)
	TeacherIsTimetabledForSubjectInClass(ctx context.Context, institutionID, teacherID, classID, subjectID, termID uuid.UUID) (bool, error)
	GetGradebook(ctx context.Context, institutionID, componentID uuid.UUID) ([]*GradeEntry, error)
	SaveGradeEntries(ctx context.Context, institutionID, componentID, enteredBy uuid.UUID, entries []GradeInput) error
	CreateGradeComponent(ctx context.Context, institutionID uuid.UUID, req *CreateGradeComponentRequest) (*GradeComponent, error)

	// Report cards
	GetClassRankings(ctx context.Context, institutionID, classID, termID uuid.UUID) ([]*StudentRank, error)
	GetTermDates(ctx context.Context, institutionID, termID uuid.UUID) (time.Time, time.Time, error)
	GetAggregateMarksForStudent(ctx context.Context, institutionID, studentID, termID uuid.UUID) (float64, int, error)
	UpsertReportCard(ctx context.Context, institutionID, classID, termID uuid.UUID, rc *ReportCard) error
	PublishReportCard(ctx context.Context, institutionID, studentID, termID uuid.UUID) (*ReportCard, error)
	UnpublishReportCard(ctx context.Context, institutionID, studentID, termID uuid.UUID) (*ReportCard, error)
	GetPublishedReportCardForStudent(ctx context.Context, institutionID, studentID uuid.UUID) (*ReportCard, error)
	GetSubjectScoresForStudent(ctx context.Context, institutionID, studentID, termID uuid.UUID) ([]*SubjectScore, error)

	// Terms
	ListTerms(ctx context.Context, institutionID uuid.UUID) ([]*Term, error)

	// Dashboard (defined in dashboard.go, not repository.go)
	GetInstitutionAttendanceToday(ctx context.Context, institutionID uuid.UUID, date time.Time) (*InstitutionAttendanceToday, error)

	// academics/store.go — add to Store interface
	ListTeachingAssignmentsForClass(ctx context.Context, institutionID, classID, termID uuid.UUID) ([]*TeachingAssignmentOption, error)
	CreateTimetablePeriod(ctx context.Context, institutionID uuid.UUID, req *CreateTimetablePeriodRequest) (*TimetablePeriod, error)
}