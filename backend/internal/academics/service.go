package academics

import (
	"context"
	"fmt"
	"strings"
	"time"
	"errors"

	"institution-os/pkg/events"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

// Service implements all academics domain business logic.
type Service struct {
	repo      Store
	publisher events.Publisher
}

// NewService creates a new academics service.
func NewService(repo Store, publisher events.Publisher) *Service {
	return &Service{repo: repo, publisher: publisher}
}

// parentGuard is the narrow ownership-check contract. *Repository has it;
// test mocks don't need it unless they exercise parent paths.
type parentGuard interface {
	IsChildOfParent(ctx context.Context, institutionID, parentUserID, studentID uuid.UUID) (bool, error)
}

// VerifyParentChild enforces parent→child ownership (pilot P0-4).
func (s *Service) VerifyParentChild(ctx context.Context, institutionID, parentUserID, studentID uuid.UUID) error {
	g, ok := s.repo.(parentGuard)
	if !ok {
		return fmt.Errorf("parent guard unavailable")
	}
	linked, err := g.IsChildOfParent(ctx, institutionID, parentUserID, studentID)
	if err != nil {
		return err
	}
	if !linked {
		return fmt.Errorf("not linked to student")
	}
	return nil
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
    if period.TeacherID == nil {
		if !canOverrideClassTeacher {
			return nil, ErrNotAssignedToClass
		}
	} else if *period.TeacherID != teacherID && !canOverrideClassTeacher {
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

// GetChildAttendanceSummary is the parent-portal attendance path (P0-4):
// ownership is verified, the child's class is resolved server-side, and
// any caller-supplied class_id is ignored so a parent can't enumerate
// other classes.
func (s *Service) GetChildAttendanceSummary(ctx context.Context, institutionID, parentUserID, studentID uuid.UUID, startDate, endDate time.Time) ([]*AttendanceSummary, error) {
	if err := s.VerifyParentChild(ctx, institutionID, parentUserID, studentID); err != nil {
		return nil, err
	}
	classID, err := s.repo.GetStudentCurrentClass(ctx, institutionID, studentID)
	if err != nil {
		return nil, err
	}
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


var ErrOfferingMustBeExactlyOne = errors.New("exactly one of subject_id or learning_area_id is required")

func (s *Service) CreateTeachingAssignment(ctx context.Context, institutionID uuid.UUID, req *CreateTeachingAssignmentRequest) (*TeachingAssignment, error) {
	hasSubject := req.SubjectID != nil
	hasLearningArea := req.LearningAreaID != nil
	if hasSubject == hasLearningArea {
		return nil, ErrOfferingMustBeExactlyOne
	}
	return s.repo.CreateTeachingAssignment(ctx, institutionID, req)
}

func (s *Service) ListTeachingAssignmentsForClass(ctx context.Context, institutionID, classID, termID uuid.UUID) ([]*TeachingAssignmentOption, error) {
	return s.repo.ListTeachingAssignmentsForClass(ctx, institutionID, classID, termID)
}

func (s *Service) CreateTimetablePeriod(ctx context.Context, institutionID uuid.UUID, req *CreateTimetablePeriodRequest) (*TimetablePeriod, error) {
	return s.repo.CreateTimetablePeriod(ctx, institutionID, req)
}

// ── Timetable engine ─────────────────────────────────────────────

var (
	ErrProfileNameRequired   = errors.New("profile name is required")
	ErrInvalidDayOfWeek      = errors.New("day_of_week must be between 1 (Monday) and 7 (Sunday)")
	ErrInvalidSequence       = errors.New("sequence must be at least 1")
	ErrInvalidSlotTimes      = errors.New("start_time and end_time are required as HH:MM with end after start")
	ErrInvalidSlotType       = errors.New("slot_type must be one of lesson, break, lunch, assembly, activity")
	ErrInvalidSessionsPerWeek = errors.New("sessions_per_week must be at least 1")
	ErrNoSessionShape        = errors.New("at least one of allow_single, allow_double, allow_triple is required")
	ErrInvalidMaxPerDay      = errors.New("max_per_day must be positive when set")
)

func validSlotType(t string) bool {
	switch t {
	case "lesson", "break", "lunch", "assembly", "activity":
		return true
	}
	return false
}

func parseSlotTime(v string) (time.Time, error) {
	return time.Parse("15:04", v)
}

func (s *Service) ListScheduleProfiles(ctx context.Context, institutionID uuid.UUID) ([]*ScheduleProfile, error) {
	return s.repo.ListScheduleProfiles(ctx, institutionID)
}

func (s *Service) CreateScheduleProfile(ctx context.Context, institutionID uuid.UUID, req *CreateScheduleProfileRequest) (*ScheduleProfile, error) {
	if strings.TrimSpace(req.Name) == "" {
		return nil, ErrProfileNameRequired
	}
	return s.repo.CreateScheduleProfile(ctx, institutionID, req)
}

func (s *Service) ListScheduleSlots(ctx context.Context, institutionID, profileID uuid.UUID) ([]*ScheduleSlot, error) {
	return s.repo.ListScheduleSlots(ctx, institutionID, profileID)
}

func validateSlotShape(day, sequence int, start, end, slotType string) error {
	if day < 1 || day > 7 {
		return ErrInvalidDayOfWeek
	}
	if sequence < 1 {
		return ErrInvalidSequence
	}
	if !validSlotType(slotType) {
		return ErrInvalidSlotType
	}
	startT, err := parseSlotTime(start)
	if err != nil {
		return ErrInvalidSlotTimes
	}
	endT, err := parseSlotTime(end)
	if err != nil {
		return ErrInvalidSlotTimes
	}
	if !endT.After(startT) {
		return ErrInvalidSlotTimes
	}
	return nil
}

func (s *Service) CreateScheduleSlot(ctx context.Context, institutionID, profileID uuid.UUID, req *CreateScheduleSlotRequest) (*ScheduleSlot, error) {
	if err := validateSlotShape(req.DayOfWeek, req.Sequence, req.StartTime, req.EndTime, req.SlotType); err != nil {
		return nil, err
	}
	return s.repo.CreateScheduleSlot(ctx, institutionID, profileID, req)
}

func (s *Service) UpdateScheduleSlot(ctx context.Context, institutionID, slotID uuid.UUID, req *UpdateScheduleSlotRequest) (*ScheduleSlot, error) {
	if req.SlotType != nil && !validSlotType(*req.SlotType) {
		return nil, ErrInvalidSlotType
	}
	if req.StartTime != nil {
		if _, err := parseSlotTime(*req.StartTime); err != nil {
			return nil, ErrInvalidSlotTimes
		}
	}
	if req.EndTime != nil {
		if _, err := parseSlotTime(*req.EndTime); err != nil {
			return nil, ErrInvalidSlotTimes
		}
	}
	return s.repo.UpdateScheduleSlot(ctx, institutionID, slotID, req)
}

func (s *Service) DeleteScheduleSlot(ctx context.Context, institutionID, slotID uuid.UUID) error {
	return s.repo.DeleteScheduleSlot(ctx, institutionID, slotID)
}

// normalizeSlotDays dedupes a day list and rejects anything outside 1-7.
func normalizeSlotDays(days []int) ([]int, error) {
	if len(days) == 0 {
		return nil, ErrInvalidDayOfWeek
	}
	seen := map[int]bool{}
	var out []int
	for _, d := range days {
		if d < 1 || d > 7 {
			return nil, ErrInvalidDayOfWeek
		}
		if !seen[d] {
			seen[d] = true
			out = append(out, d)
		}
	}
	return out, nil
}

func (s *Service) CreateScheduleSlotsBulk(ctx context.Context, institutionID, profileID uuid.UUID, req *BulkCreateScheduleSlotsRequest) (*BulkCreateScheduleSlotsResult, error) {
	days, err := normalizeSlotDays(req.Days)
	if err != nil {
		return nil, err
	}
	req.Days = days
	if req.Sequence < 1 {
		return nil, ErrInvalidSequence
	}
	if err := validateSlotShape(days[0], req.Sequence, req.StartTime, req.EndTime, req.SlotType); err != nil {
		return nil, err
	}
	return s.repo.CreateScheduleSlotsBulk(ctx, institutionID, profileID, req)
}

func (s *Service) CopyScheduleDay(ctx context.Context, institutionID, profileID uuid.UUID, req *CopyScheduleDayRequest) (*BulkCreateScheduleSlotsResult, error) {
	if req.FromDay < 1 || req.FromDay > 7 {
		return nil, ErrInvalidDayOfWeek
	}
	toDays, err := normalizeSlotDays(req.ToDays)
	if err != nil {
		return nil, err
	}
	// Drop the source day if the caller included it.
	filtered := toDays[:0]
	for _, d := range toDays {
		if d != req.FromDay {
			filtered = append(filtered, d)
		}
	}
	if len(filtered) == 0 {
		return nil, ErrInvalidDayOfWeek
	}
	req.ToDays = filtered
	return s.repo.CopyScheduleDay(ctx, institutionID, profileID, req)
}

func (s *Service) PropagateScheduleSlot(ctx context.Context, institutionID, slotID uuid.UUID, req *PropagateScheduleSlotRequest) (*PropagateScheduleSlotResult, error) {
	if req.SlotType != nil && !validSlotType(*req.SlotType) {
		return nil, ErrInvalidSlotType
	}
	if req.StartTime != nil {
		if _, err := parseSlotTime(*req.StartTime); err != nil {
			return nil, ErrInvalidSlotTimes
		}
	}
	if req.EndTime != nil {
		if _, err := parseSlotTime(*req.EndTime); err != nil {
			return nil, ErrInvalidSlotTimes
		}
	}
	targetDays, err := normalizeSlotDays(req.TargetDays)
	if err != nil {
		return nil, err
	}
	req.TargetDays = targetDays
	return s.repo.PropagateScheduleSlot(ctx, institutionID, slotID, req)
}

func (s *Service) ListLessonRequirements(ctx context.Context, institutionID, classID, termID uuid.UUID) ([]*LessonRequirement, error) {
	return s.repo.ListLessonRequirements(ctx, institutionID, classID, termID)
}

func (s *Service) UpsertLessonRequirement(ctx context.Context, institutionID, classID uuid.UUID, req *UpsertLessonRequirementRequest) (*LessonRequirement, error) {
	if req.SessionsPerWeek < 1 {
		return nil, ErrInvalidSessionsPerWeek
	}
	if !req.AllowSingle && !req.AllowDouble && !req.AllowTriple {
		return nil, ErrNoSessionShape
	}
	if req.MaxPerDay != nil && *req.MaxPerDay < 1 {
		return nil, ErrInvalidMaxPerDay
	}
	return s.repo.UpsertLessonRequirement(ctx, institutionID, classID, req)
}

func (s *Service) DeleteLessonRequirement(ctx context.Context, institutionID, requirementID uuid.UUID) error {
	return s.repo.DeleteLessonRequirement(ctx, institutionID, requirementID)
}