package academics

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"institution-os/pkg/events"
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

	// Load existing records or blank roster
	records, err := s.repo.GetSessionStudents(ctx, institutionID, classID)
	if err != nil {
		return nil, err
	}
	session.Records = records

	return session, nil
}

// SubmitAttendance saves attendance records and locks the session.
func (s *Service) SubmitAttendance(ctx context.Context, institutionID, teacherID uuid.UUID, req *TakeAttendanceRequest) (*AttendanceSession, error) {
	sessionDate, err := time.Parse("2006-01-02", req.SessionDate)
	if err != nil {
		return nil, fmt.Errorf("invalid session_date format")
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
	return s.repo.SaveGradeEntries(ctx, institutionID, teacherID, req.Entries)
}
