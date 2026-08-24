package academics

import (
	"context"
	"fmt"
	"time"
	"errors"

	"institution-os/pkg/events"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

// Service implements all academics domain business logic.
type Service struct {
	repo      *Repository
	publisher events.Publisher
}

// NewService creates a new academics service.
func NewService(repo *Repository, publisher events.Publisher) *Service {
	return &Service{repo: repo, publisher: publisher}
}

// ─── Classes ──────────────────────────────────────────────────────

// ListClasses returns filtered classes for an institution.
func (s *Service) ListClasses(ctx context.Context, institutionID uuid.UUID, f *ClassFilter) ([]*Class, error) {
	if f == nil {
		f = &ClassFilter{}
	}
	return s.repo.ListClasses(ctx, institutionID, f)
}

// GetClass retrieves a single class.
func (s *Service) GetClass(ctx context.Context, institutionID, classID uuid.UUID) (*Class, error) {
	return s.repo.GetClassByID(ctx, institutionID, classID)
}

// ListSubjects returns all subjects for an institution.
func (s *Service) ListSubjects(ctx context.Context, institutionID uuid.UUID) ([]*Subject, error) {
	return s.repo.ListSubjects(ctx, institutionID)
}

// ─── Timetable ────────────────────────────────────────────────────

// GetClassTimetable returns the timetable for a specific class.
func (s *Service) GetClassTimetable(ctx context.Context, institutionID, classID uuid.UUID, termID *uuid.UUID) ([]*TimetablePeriod, error) {
	if termID == nil {
		t, err := s.repo.GetCurrentTerm(ctx, institutionID)
		if err != nil {
			return nil, err
		}
		if t == nil {
			return []*TimetablePeriod{}, nil
		}
		termID = t
	}
	return s.repo.GetTimetableForClass(ctx, institutionID, classID, *termID)
}

// GetTeacherTimetable returns a teacher's timetable.
func (s *Service) GetTeacherTimetable(ctx context.Context, institutionID, teacherID uuid.UUID, termID *uuid.UUID) ([]*TimetablePeriod, error) {
	if termID == nil {
		t, err := s.repo.GetCurrentTerm(ctx, institutionID)
		if err != nil {
			return nil, err
		}
		if t == nil {
			return []*TimetablePeriod{}, nil
		}
		termID = t
	}
	return s.repo.GetTimetableForTeacher(ctx, institutionID, teacherID, *termID)
}

// ─── Attendance ───────────────────────────────────────────────────

// GetAttendanceSheet returns an open session + blank records for taking attendance.
func (s *Service) GetAttendanceSheet(ctx context.Context, institutionID, classID, teacherID uuid.UUID, dateStr, sessionType string, periodID *uuid.UUID) (*AttendanceSession, error) {
	var sessionDate time.Time
	var err error
	if dateStr == "" {
		sessionDate = time.Now().UTC().Truncate(24 * time.Hour)
	} else {
		sessionDate, err = time.Parse("2006-01-02", dateStr)
		if err != nil {
			return nil, fmt.Errorf("invalid date format, use YYYY-MM-DD")
		}
	}
	if sessionType == "" {
		sessionType = "daily"
	}

	session, err := s.repo.GetOrCreateSession(ctx, institutionID, classID, teacherID, sessionDate, sessionType, periodID)
	if err != nil {
		return nil, err
	}

	// Load real saved records if this session already has them, otherwise blank roster
	records, err := s.repo.GetSessionStudents(ctx, institutionID, classID, session.ID)
	if err != nil {
		return nil, err
	}
	session.Records = records

	return session, nil
}

// SubmitAttendance saves attendance records and locks the session.
//
// Authorization (Kenyan-school convention): daily whole-class attendance
// is a class-teacher responsibility, not something any teacher with
// attendance:write should be able to submit for a class that isn't
// theirs. canOverrideClassTeacher is computed by the caller from the
// attendance:approve permission (held by Super Admin, and anyone else a
// school chooses to grant it to — e.g. a deputy covering for an absent
// class teacher) and lets that override happen without hardcoding a role
// name here.
//
// Period-type attendance gets a lighter check for now: the teacher must
// be timetabled somewhere in this class at all. It does NOT yet verify
// they're timetabled for the specific subject/period being taken, and it
// does NOT yet restrict the roster to only students enrolled in that
// subject — both require a subject_enrollments concept that doesn't
// exist yet (see TODO.md). Until then, period attendance still marks
// every active student in the class, same as daily.
func (s *Service) SubmitAttendance(ctx context.Context, institutionID, teacherID uuid.UUID, canOverrideClassTeacher bool, req *TakeAttendanceRequest) (*AttendanceSession, error) {
	sessionDate, err := time.Parse("2006-01-02", req.SessionDate)
	if err != nil {
		return nil, fmt.Errorf("invalid session_date format")
	}
 
	class, err := s.repo.GetClassByID(ctx, institutionID, req.ClassID)
	if err != nil {
		return nil, fmt.Errorf("get class: %w", err)
	}
 
	if req.SessionType == "daily" {
    switch {
    case class.ClassTeacherID == nil:
        // No class teacher assigned yet — this is a setup gap, not a
        // green light for any teacher. Only an explicit override
        // (attendance:approve) may submit until an admin assigns one.
        if !canOverrideClassTeacher {
            return nil, ErrClassNeedsTeacherAssignment
        }
    case *class.ClassTeacherID != teacherID:
        if !canOverrideClassTeacher {
            return nil, ErrNotClassTeacher
        }
    }
} else {
    if req.PeriodID == nil {
        return nil, fmt.Errorf("period_id is required for period attendance")
    }
    period, err := s.repo.GetTimetablePeriodByID(ctx, institutionID, *req.PeriodID)
    if err != nil {
        return nil, err
    }
    if period.ClassID != req.ClassID {
        return nil, fmt.Errorf("this period does not belong to the specified class")
    }
    if period.TeacherID != teacherID && !canOverrideClassTeacher {
        return nil, ErrNotAssignedToClass
    }
}
 
	// Validate every student in the request actually belongs to this class,
	// before writing anything. Without this check, a wrong class_id in the
	// request silently creates attendance records for students who were
	// never enrolled in that class.
	validStudentIDs, err := s.repo.GetActiveStudentIDsForClass(ctx, institutionID, req.ClassID)
	if err != nil {
		return nil, fmt.Errorf("verify class roster: %w", err)
	}
	validSet := make(map[uuid.UUID]bool, len(validStudentIDs))
	for _, id := range validStudentIDs {
		validSet[id] = true
	}
	for _, entry := range req.Records {
		if !validSet[entry.StudentID] {
			return nil, fmt.Errorf("student %s is not enrolled in class %s", entry.StudentID, req.ClassID)
		}
	}
 
	session, err := s.repo.GetOrCreateSession(ctx, institutionID, req.ClassID, teacherID, sessionDate, req.SessionType, req.PeriodID)
	if err != nil {
		return nil, err
	}
 
	if session.Status == "locked" {
		return nil, ErrSessionLocked
	}
 
	if err := s.repo.SaveAttendanceRecords(ctx, institutionID, session.ID, req.Records); err != nil {
		return nil, fmt.Errorf("save attendance: %w", err)
	}
 
	// Publish absent events so notifications are sent to parents
	for _, entry := range req.Records {
		if entry.Status == "absent" {
			contact, err := s.repo.GetPrimaryContactForStudent(ctx, institutionID, entry.StudentID)
			if err != nil {
				log.Error().Err(err).Str("student_id", entry.StudentID.String()).
					Msg("could not find primary contact for absent student — skipping notification")
				continue
			}
 
			evt := events.New(events.EventStudentAbsent, institutionID.String())
			evt.EntityType = "student"
			evt.EntityID = entry.StudentID.String()
			evt.ActorID = teacherID.String()
			evt.Payload = map[string]interface{}{
				"parent_phone": contact.ParentPhone,
				"student_name": contact.StudentName,
			}
			_ = s.publisher.Publish(evt)
		}
	}
 
	session.Status = "submitted"
	now := time.Now()
	session.SubmittedAt = &now
	return session, nil
}

// GetAttendanceSummary returns attendance stats for a class over a date range.
func (s *Service) GetAttendanceSummary(ctx context.Context, institutionID, classID uuid.UUID, startDate, endDate time.Time) ([]*AttendanceSummary, error) {
	return s.repo.GetAttendanceSummary(ctx, institutionID, classID, startDate, endDate)
}

// ─── Grades ───────────────────────────────────────────────────────

// GetGradeComponents returns assessment components for a class/subject/term.
func (s *Service) GetGradeComponents(ctx context.Context, institutionID, classID, subjectID, termID uuid.UUID) ([]*GradeComponent, error) {
	return s.repo.ListGradeComponents(ctx, institutionID, classID, subjectID, termID)
}

// GetGradebook returns grade entries for a component.
var ErrNotTimetabledForSubject = errors.New("teacher is not timetabled to teach this subject in this class")

func (s *Service) SaveGrades(ctx context.Context, institutionID, teacherID uuid.UUID, isTeacherRole bool, req *BulkGradeEntryRequest) error {
	if len(req.Entries) == 0 {
		return fmt.Errorf("no entries provided")
	}

	if isTeacherRole {
		component, err := s.repo.GetGradeComponentByID(ctx, institutionID, req.ComponentID)
		if err != nil {
			return err
		}
		ok, err := s.repo.TeacherIsTimetabledForSubjectInClass(ctx, institutionID, teacherID, component.ClassID, component.SubjectID, component.TermID)
		if err != nil {
			return err
		}
		if !ok {
			return ErrNotTimetabledForSubject
		}
	}

	return s.repo.SaveGradeEntries(ctx, institutionID, req.ComponentID, teacherID, req.Entries)
}

func (s *Service) GetGradebook(ctx context.Context, institutionID, teacherID uuid.UUID, isTeacherRole bool, componentID uuid.UUID) ([]*GradeEntry, error) {
	if isTeacherRole {
		component, err := s.repo.GetGradeComponentByID(ctx, institutionID, componentID)
		if err != nil {
			return nil, err
		}
		ok, err := s.repo.TeacherIsTimetabledForSubjectInClass(ctx, institutionID, teacherID, component.ClassID, component.SubjectID, component.TermID)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, ErrNotTimetabledForSubject
		}
	}
	return s.repo.GetGradebook(ctx, institutionID, componentID)
}

// CreateGradeComponent defines a new assessment component for a class/subject/term.
func (s *Service) CreateGradeComponent(ctx context.Context, institutionID uuid.UUID, req *CreateGradeComponentRequest) (*GradeComponent, error) {
	if req.Weight <= 0 || req.Weight > 100 {
		return nil, fmt.Errorf("weight must be between 1 and 100")
	}
	if req.MaxMarks <= 0 {
		return nil, fmt.Errorf("max_marks must be greater than 0")
	}
	return s.repo.CreateGradeComponent(ctx, institutionID, req)
}

//* GenerateReportCard computes and saves a report card for a single student.


// generateReportCardForStudent computes one student's report card using
// class-wide data (rankings, attendance) the caller already fetched.
// GenerateReportCard fetches that data for a single call; bulk generation
// fetches it ONCE per class and calls this directly for every student,
// instead of re-running the rankings/attendance queries per student.


// GenerateReportCardsForClass generates (or regenerates) a report card for
// every active student in a class for the given term. Each student is
// processed independently — one student's failure is recorded and skipped;
// it never aborts the rest of the batch. Rankings and attendance are
// computed once for the whole class and reused for every student.


// PublishReportCard transitions a single student's report card from
// draft to published. See Repository.PublishReportCard for idempotency
// semantics.



// gradeScale maps the standard KNEC-style percentage scale to letter
// grades and points, used for termly internal assessment reporting.
// NOTE: the official November KCSE exam uses examiner-set boundaries
// that vary by subject and year — this fixed scale is the conventional
// approximation used for internal school report cards, and is the
// Phase 1 default. Should become a per-institution configurable table
// in a future phase, the same way M-Pesa allocation policy was deferred.
type gradeBand struct {
	min    float64
	letter string
	points float64
}

var gradeScale = []gradeBand{
	{80, "A", 12},
	{75, "A-", 11},
	{70, "B+", 10},
	{65, "B", 9},
	{60, "B-", 8},
	{55, "C+", 7},
	{50, "C", 6},
	{45, "C-", 5},
	{40, "D+", 4},
	{35, "D", 3},
	{30, "D-", 2},
	{0, "E", 1},
}

// gradeFromPercentage converts a percentage score into its letter grade
// and point value, using the Phase 1 default scale.
func gradeFromPercentage(pct float64) (letter string, points float64) {
	for _, band := range gradeScale {
		if pct >= band.min {
			return band.letter, band.points
		}
	}
	return "E", 1
}

// UnpublishReportCard reverts a published report card to draft so it
// can be corrected and republished. See Repository.UnpublishReportCard
// for idempotency semantics.
func (s *Service) UnpublishReportCard(ctx context.Context, institutionID, studentID, termID uuid.UUID) (*ReportCard, error) {
	return s.repo.UnpublishReportCard(ctx, institutionID, studentID, termID)
}

// GetChildReportCard returns the most recently published report card
// for a student, intended for parent-portal consumption.
func (s *Service) GetChildReportCard(ctx context.Context, institutionID, studentID uuid.UUID) (*ReportCard, error) {
	return s.repo.GetPublishedReportCardForStudent(ctx, institutionID, studentID)
}

// GetChildSubjectScores returns per-subject scores for the current term,
// for display on the parent-facing report card detail page.
func (s *Service) GetChildSubjectScores(ctx context.Context, institutionID, studentID uuid.UUID) ([]*SubjectScore, error) {
	termID, err := s.repo.GetCurrentTerm(ctx, institutionID)
	if err != nil {
		return nil, err
	}
	if termID == nil {
		return []*SubjectScore{}, nil
	}
	return s.repo.GetSubjectScoresForStudent(ctx, institutionID, studentID, *termID)
}

// ListTerms returns all terms for an institution.
func (s *Service) ListTerms(ctx context.Context, institutionID uuid.UUID) ([]*Term, error) {
	return s.repo.ListTerms(ctx, institutionID)
}

func (s *Service) CreateClass(ctx context.Context, institutionID uuid.UUID, req *CreateClassRequest) (*Class, error) {
	if req.Capacity <= 0 {
		return nil, fmt.Errorf("capacity must be greater than 0")
	}
	return s.repo.CreateClass(ctx, institutionID, req)
}

func (s *Service) AssignClassTeacher(ctx context.Context, institutionID, classID, teacherID uuid.UUID) error {
	return s.repo.AssignClassTeacher(ctx, institutionID, classID, teacherID)
}

func (s *Service) ListTeachers(ctx context.Context, institutionID uuid.UUID) ([]*TeacherOption, error) {
	return s.repo.ListTeachers(ctx, institutionID)
}

func (s *Service) ListAcademicYears(ctx context.Context, institutionID uuid.UUID) ([]*AcademicYear, error) {
	return s.repo.ListAcademicYears(ctx, institutionID)
}