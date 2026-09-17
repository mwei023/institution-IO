package tests

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"institution-os/internal/academics"
	"institution-os/pkg/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// ─── Mock Academics Repository ────────────────────────────────────

type MockAcademicsRepo struct {
	mock.Mock
}

func (m *MockAcademicsRepo) ListClasses(ctx context.Context, institutionID uuid.UUID, f *academics.ClassFilter) ([]*academics.Class, error) {
	args := m.Called(ctx, institutionID, f)
	return args.Get(0).([]*academics.Class), args.Error(1)
}

func (m *MockAcademicsRepo) GetClassByID(ctx context.Context, institutionID, classID uuid.UUID) (*academics.Class, error) {
	args := m.Called(ctx, institutionID, classID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*academics.Class), args.Error(1)
}

func (m *MockAcademicsRepo) ListSubjects(ctx context.Context, institutionID uuid.UUID) ([]*academics.Subject, error) {
	args := m.Called(ctx, institutionID)
	return args.Get(0).([]*academics.Subject), args.Error(1)
}

func (m *MockAcademicsRepo) GetTimetableForClass(ctx context.Context, institutionID, classID, termID uuid.UUID) ([]*academics.TimetablePeriod, error) {
	args := m.Called(ctx, institutionID, classID, termID)
	return args.Get(0).([]*academics.TimetablePeriod), args.Error(1)
}

func (m *MockAcademicsRepo) GetTimetableForTeacher(ctx context.Context, institutionID, teacherID, termID uuid.UUID) ([]*academics.TimetablePeriod, error) {
	args := m.Called(ctx, institutionID, teacherID, termID)
	return args.Get(0).([]*academics.TimetablePeriod), args.Error(1)
}

func (m *MockAcademicsRepo) GetOrCreateSession(ctx context.Context, institutionID, classID, teacherID uuid.UUID, sessionDate time.Time, sessionType string, periodID *uuid.UUID) (*academics.AttendanceSession, error) {
	args := m.Called(ctx, institutionID, classID, teacherID, sessionDate, sessionType, periodID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*academics.AttendanceSession), args.Error(1)
}

func (m *MockAcademicsRepo) GetSessionStudents(ctx context.Context, institutionID, classID, sessionID uuid.UUID) ([]*academics.AttendanceRecord, error) {
	args := m.Called(ctx, institutionID, classID, sessionID)
	return args.Get(0).([]*academics.AttendanceRecord), args.Error(1)
}

func (m *MockAcademicsRepo) SaveAttendanceRecords(ctx context.Context, institutionID, sessionID uuid.UUID, entries []academics.AttendanceEntry) error {
	return m.Called(ctx, institutionID, sessionID, entries).Error(0)
}

func (m *MockAcademicsRepo) GetAttendanceSummary(ctx context.Context, institutionID, classID uuid.UUID, startDate, endDate time.Time) ([]*academics.AttendanceSummary, error) {
	args := m.Called(ctx, institutionID, classID, startDate, endDate)
	return args.Get(0).([]*academics.AttendanceSummary), args.Error(1)
}

func (m *MockAcademicsRepo) ListGradeComponents(ctx context.Context, institutionID, classID, subjectID, termID uuid.UUID) ([]*academics.GradeComponent, error) {
	args := m.Called(ctx, institutionID, classID, subjectID, termID)
	return args.Get(0).([]*academics.GradeComponent), args.Error(1)
}

func (m *MockAcademicsRepo) GetGradebook(ctx context.Context, institutionID, componentID uuid.UUID) ([]*academics.GradeEntry, error) {
	args := m.Called(ctx, institutionID, componentID)
	return args.Get(0).([]*academics.GradeEntry), args.Error(1)
}

func (m *MockAcademicsRepo) SaveGradeEntries(ctx context.Context, institutionID, componentID, enteredBy uuid.UUID, entries []academics.GradeInput) error {
	return m.Called(ctx, institutionID, componentID, enteredBy, entries).Error(0)
}

func (m *MockAcademicsRepo) GetCurrentTerm(ctx context.Context, institutionID uuid.UUID) (*uuid.UUID, error) {
	args := m.Called(ctx, institutionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*uuid.UUID), args.Error(1)
}

func (m *MockAcademicsRepo) CreateClass(ctx context.Context, institutionID uuid.UUID, req *academics.CreateClassRequest) (*academics.Class, error) {
	args := m.Called(ctx, institutionID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*academics.Class), args.Error(1)
}

func (m *MockAcademicsRepo) AssignClassTeacher(ctx context.Context, institutionID, classID, teacherID uuid.UUID) error {
	return m.Called(ctx, institutionID, classID, teacherID).Error(0)
}

func (m *MockAcademicsRepo) ListTeachers(ctx context.Context, institutionID uuid.UUID) ([]*academics.TeacherOption, error) {
	args := m.Called(ctx, institutionID)
	return args.Get(0).([]*academics.TeacherOption), args.Error(1)
}

func (m *MockAcademicsRepo) CreateTeachingAssignment(ctx context.Context, institutionID uuid.UUID, req *academics.CreateTeachingAssignmentRequest) (*academics.TeachingAssignment, error) {
	args := m.Called(ctx, institutionID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*academics.TeachingAssignment), args.Error(1)
}

func (m *MockAcademicsRepo) ListAcademicYears(ctx context.Context, institutionID uuid.UUID) ([]*academics.AcademicYear, error) {
	args := m.Called(ctx, institutionID)
	return args.Get(0).([]*academics.AcademicYear), args.Error(1)
}

func (m *MockAcademicsRepo) GetTimetablePeriodByID(ctx context.Context, institutionID, periodID uuid.UUID) (*academics.TimetablePeriodInfo, error) {
	args := m.Called(ctx, institutionID, periodID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*academics.TimetablePeriodInfo), args.Error(1)
}

func (m *MockAcademicsRepo) GetActiveStudentIDsForClass(ctx context.Context, institutionID, classID uuid.UUID) ([]uuid.UUID, error) {
	args := m.Called(ctx, institutionID, classID)
	return args.Get(0).([]uuid.UUID), args.Error(1)
}

func (m *MockAcademicsRepo) GetPrimaryContactForStudent(ctx context.Context, institutionID, studentID uuid.UUID) (*academics.PrimaryContactInfo, error) {
	args := m.Called(ctx, institutionID, studentID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*academics.PrimaryContactInfo), args.Error(1)
}

func (m *MockAcademicsRepo) GetGradeComponentByID(ctx context.Context, institutionID, componentID uuid.UUID) (*academics.GradeComponent, error) {
	args := m.Called(ctx, institutionID, componentID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*academics.GradeComponent), args.Error(1)
}

func (m *MockAcademicsRepo) TeacherIsTimetabledForSubjectInClass(ctx context.Context, institutionID, teacherID, classID, subjectID, termID uuid.UUID) (bool, error) {
	args := m.Called(ctx, institutionID, teacherID, classID, subjectID, termID)
	return args.Bool(0), args.Error(1)
}

func (m *MockAcademicsRepo) CreateGradeComponent(ctx context.Context, institutionID uuid.UUID, req *academics.CreateGradeComponentRequest) (*academics.GradeComponent, error) {
	args := m.Called(ctx, institutionID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*academics.GradeComponent), args.Error(1)
}

func (m *MockAcademicsRepo) GetClassRankings(ctx context.Context, institutionID, classID, termID uuid.UUID) ([]*academics.StudentRank, error) {
	args := m.Called(ctx, institutionID, classID, termID)
	return args.Get(0).([]*academics.StudentRank), args.Error(1)
}

func (m *MockAcademicsRepo) GetTermDates(ctx context.Context, institutionID, termID uuid.UUID) (time.Time, time.Time, error) {
	args := m.Called(ctx, institutionID, termID)
	return args.Get(0).(time.Time), args.Get(1).(time.Time), args.Error(2)
}

func (m *MockAcademicsRepo) GetAggregateMarksForStudent(ctx context.Context, institutionID, studentID, termID uuid.UUID) (float64, int, error) {
	args := m.Called(ctx, institutionID, studentID, termID)
	return args.Get(0).(float64), args.Int(1), args.Error(2)
}

func (m *MockAcademicsRepo) UpsertReportCard(ctx context.Context, institutionID, classID, termID uuid.UUID, rc *academics.ReportCard) error {
	return m.Called(ctx, institutionID, classID, termID, rc).Error(0)
}

func (m *MockAcademicsRepo) PublishReportCard(ctx context.Context, institutionID, studentID, termID uuid.UUID) (*academics.ReportCard, error) {
	args := m.Called(ctx, institutionID, studentID, termID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*academics.ReportCard), args.Error(1)
}

func (m *MockAcademicsRepo) UnpublishReportCard(ctx context.Context, institutionID, studentID, termID uuid.UUID) (*academics.ReportCard, error) {
	args := m.Called(ctx, institutionID, studentID, termID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*academics.ReportCard), args.Error(1)
}

func (m *MockAcademicsRepo) GetPublishedReportCardForStudent(ctx context.Context, institutionID, studentID uuid.UUID) (*academics.ReportCard, error) {
	args := m.Called(ctx, institutionID, studentID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*academics.ReportCard), args.Error(1)
}

func (m *MockAcademicsRepo) GetSubjectScoresForStudent(ctx context.Context, institutionID, studentID, termID uuid.UUID) ([]*academics.SubjectScore, error) {
	args := m.Called(ctx, institutionID, studentID, termID)
	return args.Get(0).([]*academics.SubjectScore), args.Error(1)
}

func (m *MockAcademicsRepo) ListTerms(ctx context.Context, institutionID uuid.UUID) ([]*academics.Term, error) {
	args := m.Called(ctx, institutionID)
	return args.Get(0).([]*academics.Term), args.Error(1)
}

func (m *MockAcademicsRepo) GetInstitutionAttendanceToday(ctx context.Context, institutionID uuid.UUID, date time.Time) (*academics.InstitutionAttendanceToday, error) {
	args := m.Called(ctx, institutionID, date)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*academics.InstitutionAttendanceToday), args.Error(1)
}

func (m *MockAcademicsRepo) ListTeachingAssignmentsForClass(ctx context.Context, institutionID, classID, termID uuid.UUID) ([]*academics.TeachingAssignmentOption, error) {
	args := m.Called(ctx, institutionID, classID, termID)
	return args.Get(0).([]*academics.TeachingAssignmentOption), args.Error(1)
}

func (m *MockAcademicsRepo) CreateTimetablePeriod(ctx context.Context, institutionID uuid.UUID, req *academics.CreateTimetablePeriodRequest) (*academics.TimetablePeriod, error) {
	args := m.Called(ctx, institutionID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*academics.TimetablePeriod), args.Error(1)
}

func (m *MockAcademicsRepo) ListScheduleProfiles(ctx context.Context, institutionID uuid.UUID) ([]*academics.ScheduleProfile, error) {
	args := m.Called(ctx, institutionID)
	return args.Get(0).([]*academics.ScheduleProfile), args.Error(1)
}

func (m *MockAcademicsRepo) CreateScheduleProfile(ctx context.Context, institutionID uuid.UUID, req *academics.CreateScheduleProfileRequest) (*academics.ScheduleProfile, error) {
	args := m.Called(ctx, institutionID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*academics.ScheduleProfile), args.Error(1)
}

func (m *MockAcademicsRepo) ListScheduleSlots(ctx context.Context, institutionID, profileID uuid.UUID) ([]*academics.ScheduleSlot, error) {
	args := m.Called(ctx, institutionID, profileID)
	return args.Get(0).([]*academics.ScheduleSlot), args.Error(1)
}

func (m *MockAcademicsRepo) CreateScheduleSlot(ctx context.Context, institutionID, profileID uuid.UUID, req *academics.CreateScheduleSlotRequest) (*academics.ScheduleSlot, error) {
	args := m.Called(ctx, institutionID, profileID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*academics.ScheduleSlot), args.Error(1)
}

func (m *MockAcademicsRepo) UpdateScheduleSlot(ctx context.Context, institutionID, slotID uuid.UUID, req *academics.UpdateScheduleSlotRequest) (*academics.ScheduleSlot, error) {
	args := m.Called(ctx, institutionID, slotID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*academics.ScheduleSlot), args.Error(1)
}

func (m *MockAcademicsRepo) DeleteScheduleSlot(ctx context.Context, institutionID, slotID uuid.UUID) error {
	return m.Called(ctx, institutionID, slotID).Error(0)
}

func (m *MockAcademicsRepo) CreateScheduleSlotsBulk(ctx context.Context, institutionID, profileID uuid.UUID, req *academics.BulkCreateScheduleSlotsRequest) (*academics.BulkCreateScheduleSlotsResult, error) {
	args := m.Called(ctx, institutionID, profileID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*academics.BulkCreateScheduleSlotsResult), args.Error(1)
}

func (m *MockAcademicsRepo) CopyScheduleDay(ctx context.Context, institutionID, profileID uuid.UUID, req *academics.CopyScheduleDayRequest) (*academics.BulkCreateScheduleSlotsResult, error) {
	args := m.Called(ctx, institutionID, profileID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*academics.BulkCreateScheduleSlotsResult), args.Error(1)
}

func (m *MockAcademicsRepo) PropagateScheduleSlot(ctx context.Context, institutionID, slotID uuid.UUID, req *academics.PropagateScheduleSlotRequest) (*academics.PropagateScheduleSlotResult, error) {
	args := m.Called(ctx, institutionID, slotID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*academics.PropagateScheduleSlotResult), args.Error(1)
}

func (m *MockAcademicsRepo) ListLessonRequirements(ctx context.Context, institutionID, classID, termID uuid.UUID) ([]*academics.LessonRequirement, error) {
	args := m.Called(ctx, institutionID, classID, termID)
	return args.Get(0).([]*academics.LessonRequirement), args.Error(1)
}

func (m *MockAcademicsRepo) UpsertLessonRequirement(ctx context.Context, institutionID, classID uuid.UUID, req *academics.UpsertLessonRequirementRequest) (*academics.LessonRequirement, error) {
	args := m.Called(ctx, institutionID, classID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*academics.LessonRequirement), args.Error(1)
}

func (m *MockAcademicsRepo) DeleteLessonRequirement(ctx context.Context, institutionID, requirementID uuid.UUID) error {
	return m.Called(ctx, institutionID, requirementID).Error(0)
}

func (m *MockAcademicsRepo) IsChildOfParent(ctx context.Context, institutionID, parentUserID, studentID uuid.UUID) (bool, error) {
	args := m.Called(ctx, institutionID, parentUserID, studentID)
	return args.Bool(0), args.Error(1)
}

func (m *MockAcademicsRepo) GetStudentCurrentClass(ctx context.Context, institutionID, studentID uuid.UUID) (uuid.UUID, error) {
	args := m.Called(ctx, institutionID, studentID)
	return args.Get(0).(uuid.UUID), args.Error(1)
}

// ─── Academics Service Tests ──────────────────────────────────────

func TestGetAttendanceSheet_CreatesNewSession(t *testing.T) {
	institutionID := uuid.New()
	classID := uuid.New()
	teacherID := uuid.New()
	sessionID := uuid.New()

	now := time.Now().UTC().Truncate(24 * time.Hour)

	session := &academics.AttendanceSession{
		ID:            sessionID,
		InstitutionID: institutionID,
		ClassID:       classID,
		TeacherID:     teacherID,
		SessionDate:   now,
		SessionType:   "daily",
		Status:        "open",
		CreatedAt:     time.Now(),
	}

	students := []*academics.AttendanceRecord{
		{StudentID: uuid.New(), Name: "Amara Odhiambo", Status: "present"},
		{StudentID: uuid.New(), Name: "Brian Kipchoge",  Status: "present"},
	}

	repo := new(MockAcademicsRepo)
	svc := academics.NewService(repo, &events.NoopPublisher{})

	repo.On("GetOrCreateSession", mock.Anything, institutionID, classID, teacherID, now, "daily", mock.Anything).
		Return(session, nil)
	repo.On("GetSessionStudents", mock.Anything, institutionID, classID, sessionID).
		Return(students, nil)

	result, err := svc.GetAttendanceSheet(context.Background(), institutionID, classID, teacherID, "", "daily", nil)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "open", result.Status)
	assert.Len(t, result.Records, 2)
	repo.AssertExpectations(t)
}

func TestSubmitAttendance_PublishesAbsentEvents(t *testing.T) {
	institutionID := uuid.New()
	teacherID := uuid.New()
	classID := uuid.New()
	sessionID := uuid.New()
	student1ID := uuid.New()
	student2ID := uuid.New()

	session := &academics.AttendanceSession{
		ID:          sessionID,
		ClassID:     classID,
		Status:      "open",
		SessionDate: time.Now(),
		SessionType: "daily",
	}

	repo := new(MockAcademicsRepo)
	var publishedEvents []events.Event
	publisher := &capturePublisher{events: &publishedEvents}
	svc := academics.NewService(repo, publisher)

	repo.On("GetClassByID", mock.Anything, institutionID, classID).
		Return(&academics.Class{ID: classID, InstitutionID: institutionID, ClassTeacherID: &teacherID}, nil)
	repo.On("GetActiveStudentIDsForClass", mock.Anything, institutionID, classID).
		Return([]uuid.UUID{student1ID, student2ID}, nil)
	repo.On("GetOrCreateSession", mock.Anything, institutionID, classID, teacherID,
		mock.AnythingOfType("time.Time"), "daily", mock.Anything).Return(session, nil)
	repo.On("SaveAttendanceRecords", mock.Anything, institutionID, sessionID,
		mock.AnythingOfType("[]academics.AttendanceEntry")).Return(nil)
	repo.On("GetPrimaryContactForStudent", mock.Anything, institutionID, student2ID).
		Return(&academics.PrimaryContactInfo{ParentPhone: "+254711200001", StudentName: "Brian Kipchoge"}, nil)

	req := &academics.TakeAttendanceRequest{
		ClassID:     classID,
		SessionDate: time.Now().Format("2006-01-02"),
		SessionType: "daily",
		Records: []academics.AttendanceEntry{
			{StudentID: student1ID, Status: "present"},
			{StudentID: student2ID, Status: "absent"},
		},
	}

	result, err := svc.SubmitAttendance(context.Background(), institutionID, teacherID, false, req)

	require.NoError(t, err)
	assert.Equal(t, "submitted", result.Status)

	// Should have published exactly one absent event
	absentEvents := 0
	for _, e := range publishedEvents {
		if e.Type == events.EventStudentAbsent {
			absentEvents++
		}
	}
	assert.Equal(t, 1, absentEvents)
	repo.AssertExpectations(t)
}

func TestSubmitAttendance_LockedSession(t *testing.T) {
	institutionID := uuid.New()
	teacherID := uuid.New()
	classID := uuid.New()

	lockedSession := &academics.AttendanceSession{
		ID:          uuid.New(),
		ClassID:     classID,
		Status:      "locked",
		SessionDate: time.Now(),
		SessionType: "daily",
	}

	repo := new(MockAcademicsRepo)
	svc := academics.NewService(repo, &events.NoopPublisher{})

	lockedStudentID := uuid.New()
	repo.On("GetClassByID", mock.Anything, institutionID, classID).
		Return(&academics.Class{ID: classID, InstitutionID: institutionID, ClassTeacherID: &teacherID}, nil)
	repo.On("GetActiveStudentIDsForClass", mock.Anything, institutionID, classID).
		Return([]uuid.UUID{lockedStudentID}, nil)
	repo.On("GetOrCreateSession", mock.Anything, institutionID, classID, teacherID,
		mock.AnythingOfType("time.Time"), "daily", mock.Anything).Return(lockedSession, nil)

	req := &academics.TakeAttendanceRequest{
		ClassID:     classID,
		SessionDate: time.Now().Format("2006-01-02"),
		SessionType: "daily",
		Records:     []academics.AttendanceEntry{{StudentID: lockedStudentID, Status: "present"}},
	}

	_, err := svc.SubmitAttendance(context.Background(), institutionID, teacherID, false, req)

	assert.ErrorIs(t, err, academics.ErrSessionLocked)
}

func TestSaveGrades_EmptyEntries(t *testing.T) {
	institutionID := uuid.New()
	teacherID := uuid.New()

	repo := new(MockAcademicsRepo)
	svc := academics.NewService(repo, &events.NoopPublisher{})

	req := &academics.BulkGradeEntryRequest{
		ComponentID: uuid.New(),
		Entries:     []academics.GradeInput{}, // empty
	}

	err := svc.SaveGrades(context.Background(), institutionID, teacherID, true, req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no entries")
}

func TestGetClassTimetable_UsesCurrentTermWhenNoneProvided(t *testing.T) {
	institutionID := uuid.New()
	classID := uuid.New()
	termID := uuid.New()

	repo := new(MockAcademicsRepo)
	svc := academics.NewService(repo, &events.NoopPublisher{})

	repo.On("GetCurrentTerm", mock.Anything, institutionID).Return(&termID, nil)
	repo.On("GetTimetableForClass", mock.Anything, institutionID, classID, termID).
		Return([]*academics.TimetablePeriod{}, nil)

	periods, err := svc.GetClassTimetable(context.Background(), institutionID, classID, nil)

	require.NoError(t, err)
	assert.NotNil(t, periods)
	repo.AssertExpectations(t)
}

func TestGetClassTimetable_ReturnsEmptyWhenNoCurrentTerm(t *testing.T) {
	institutionID := uuid.New()
	classID := uuid.New()

	repo := new(MockAcademicsRepo)
	svc := academics.NewService(repo, &events.NoopPublisher{})

	repo.On("GetCurrentTerm", mock.Anything, institutionID).Return(nil, nil)

	periods, err := svc.GetClassTimetable(context.Background(), institutionID, classID, nil)

	require.NoError(t, err)
	assert.Empty(t, periods)
}

// ─── Helper: event capture publisher ─────────────────────────────

type capturePublisher struct {
	events *[]events.Event
}

func (c *capturePublisher) Publish(evt events.Event) error {
	*c.events = append(*c.events, evt)
	return nil
}
