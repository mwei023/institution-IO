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

func (m *MockAcademicsRepo) GetOrCreateSession(ctx context.Context, institutionID, classID, teacherID uuid.UUID, sessionDate time.Time, sessionType string) (*academics.AttendanceSession, error) {
	args := m.Called(ctx, institutionID, classID, teacherID, sessionDate, sessionType)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*academics.AttendanceSession), args.Error(1)
}

func (m *MockAcademicsRepo) GetSessionStudents(ctx context.Context, institutionID, classID uuid.UUID) ([]*academics.AttendanceRecord, error) {
	args := m.Called(ctx, institutionID, classID)
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

func (m *MockAcademicsRepo) SaveGradeEntries(ctx context.Context, institutionID, enteredBy uuid.UUID, entries []academics.GradeInput) error {
	return m.Called(ctx, institutionID, enteredBy, entries).Error(0)
}

func (m *MockAcademicsRepo) GetCurrentTerm(ctx context.Context, institutionID uuid.UUID) (*uuid.UUID, error) {
	args := m.Called(ctx, institutionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*uuid.UUID), args.Error(1)
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

	repo.On("GetOrCreateSession", mock.Anything, institutionID, classID, teacherID, now, "daily").
		Return(session, nil)
	repo.On("GetSessionStudents", mock.Anything, institutionID, classID).
		Return(students, nil)

	result, err := svc.GetAttendanceSheet(context.Background(), institutionID, classID, teacherID, "")

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

	repo.On("GetOrCreateSession", mock.Anything, institutionID, classID, teacherID,
		mock.AnythingOfType("time.Time"), "daily").Return(session, nil)
	repo.On("SaveAttendanceRecords", mock.Anything, institutionID, sessionID,
		mock.AnythingOfType("[]academics.AttendanceEntry")).Return(nil)

	req := &academics.TakeAttendanceRequest{
		ClassID:     classID,
		SessionDate: time.Now().Format("2006-01-02"),
		SessionType: "daily",
		Records: []academics.AttendanceEntry{
			{StudentID: student1ID, Status: "present"},
			{StudentID: student2ID, Status: "absent"},
		},
	}

	result, err := svc.SubmitAttendance(context.Background(), institutionID, teacherID, req)

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

	repo.On("GetOrCreateSession", mock.Anything, institutionID, classID, teacherID,
		mock.AnythingOfType("time.Time"), "daily").Return(lockedSession, nil)

	req := &academics.TakeAttendanceRequest{
		ClassID:     classID,
		SessionDate: time.Now().Format("2006-01-02"),
		SessionType: "daily",
		Records:     []academics.AttendanceEntry{{StudentID: uuid.New(), Status: "present"}},
	}

	_, err := svc.SubmitAttendance(context.Background(), institutionID, teacherID, req)

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

	err := svc.SaveGrades(context.Background(), institutionID, teacherID, req)
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
