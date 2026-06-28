package academics

import (
	"context"
	"fmt"
	"time"

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
func (s *Service) GetAttendanceSheet(ctx context.Context, institutionID, classID, teacherID uuid.UUID, dateStr string) (*AttendanceSession, error) {
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

	session, err := s.repo.GetOrCreateSession(ctx, institutionID, classID, teacherID, sessionDate, "daily")
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
// SubmitAttendance saves attendance records and locks the session.
func (s *Service) SubmitAttendance(ctx context.Context, institutionID, teacherID uuid.UUID, req *TakeAttendanceRequest) (*AttendanceSession, error) {
	sessionDate, err := time.Parse("2006-01-02", req.SessionDate)
	if err != nil {
		return nil, fmt.Errorf("invalid session_date format")
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

	session, err := s.repo.GetOrCreateSession(ctx, institutionID, req.ClassID, teacherID, sessionDate, req.SessionType)
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
func (s *Service) GetGradebook(ctx context.Context, institutionID, componentID uuid.UUID) ([]*GradeEntry, error) {
	return s.repo.GetGradebook(ctx, institutionID, componentID)
}

// SaveGrades bulk-saves grade entries for a component.
func (s *Service) SaveGrades(ctx context.Context, institutionID, teacherID uuid.UUID, req *BulkGradeEntryRequest) error {
	if len(req.Entries) == 0 {
		return fmt.Errorf("no entries provided")
	}
	return s.repo.SaveGradeEntries(ctx, institutionID, req.ComponentID, teacherID, req.Entries)
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

// GenerateReportCard computes and saves a report card for a single student,
func (s *Service) GenerateReportCard(ctx context.Context, institutionID, studentID, classID, termID uuid.UUID) (*ReportCard, error) {
	aggregate, subjectCount, err := s.repo.GetAggregateMarksForStudent(ctx, institutionID, studentID, termID)
	if err != nil {
		return nil, fmt.Errorf("get aggregate marks: %w", err)
	}
	if subjectCount == 0 {
		return nil, fmt.Errorf("no graded subjects found for this student in this term")
	}

	rankings, err := s.repo.GetClassRankings(ctx, institutionID, classID, termID)
	if err != nil {
		return nil, fmt.Errorf("get class rankings: %w", err)
	}

	var classPosition *int
	for _, r := range rankings {
		if r.StudentID == studentID {
			pos := r.Rank
			classPosition = &pos
			break
		}
	}

	// Use the term's actual real start/end dates, not an open-ended range,
	// so attendance from other terms is never counted against this report.
	termStart, termEnd, err := s.repo.GetTermDates(ctx, institutionID, termID)
	if err != nil {
		return nil, fmt.Errorf("get term dates: %w", err)
	}

	attendance, err := s.repo.GetAttendanceSummary(ctx, institutionID, classID, termStart, termEnd)
	if err != nil {
		return nil, fmt.Errorf("get attendance summary: %w", err)
	}
	var present, total int
	for _, a := range attendance {
		if a.StudentID == studentID {
			present = a.Present + a.Late
			total = a.Total
			break
		}
	}

	_, gradePoints := gradeFromPercentage(aggregate)

	rc := &ReportCard{
		StudentID:             studentID,
		AggregateMarks:        aggregate,
		GradePoints:           gradePoints,
		ClassPosition:         classPosition,
		AttendanceDaysPresent: present,
		AttendanceDaysTotal:   total,
		Status:                "draft",
	}

	if err := s.repo.UpsertReportCard(ctx, institutionID, classID, termID, rc); err != nil {
		return nil, fmt.Errorf("save report card: %w", err)
	}

	return rc, nil
}

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
