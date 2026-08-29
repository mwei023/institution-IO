package academics

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockStore implements the Store interface without touching a real
// database — same shape as MockStore in the cbc package. Only the
// methods our tests actually call have real behavior; every other
// method exists just so this type satisfies the interface, and will
// panic loudly (via testify) if a test accidentally reaches one
// without setting up an expectation first.
type MockStore struct {
	mock.Mock
}

// ─── Methods our SaveGrades tests actually use ─────────────────────

func (m *MockStore) GetGradeComponentByID(ctx context.Context, institutionID, componentID uuid.UUID) (*GradeComponent, error) {
	args := m.Called(ctx, institutionID, componentID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*GradeComponent), args.Error(1)
}

func (m *MockStore) TeacherIsTimetabledForSubjectInClass(ctx context.Context, institutionID, teacherID, classID, subjectID, termID uuid.UUID) (bool, error) {
	args := m.Called(ctx, institutionID, teacherID, classID, subjectID, termID)
	return args.Bool(0), args.Error(1)
}

func (m *MockStore) SaveGradeEntries(ctx context.Context, institutionID, componentID, enteredBy uuid.UUID, entries []GradeInput) error {
	args := m.Called(ctx, institutionID, componentID, enteredBy, entries)
	return args.Error(0)
}

func (m *MockStore) GetGradebook(ctx context.Context, institutionID, componentID uuid.UUID) ([]*GradeEntry, error) {
	args := m.Called(ctx, institutionID, componentID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*GradeEntry), args.Error(1)
}

// ─── Everything else on Store — stubs only, needed to compile ──────

func (m *MockStore) ListClasses(ctx context.Context, institutionID uuid.UUID, f *ClassFilter) ([]*Class, error) {
	args := m.Called(ctx, institutionID, f)
	return args.Get(0).([]*Class), args.Error(1)
}
func (m *MockStore) GetClassByID(ctx context.Context, institutionID, classID uuid.UUID) (*Class, error) {
	args := m.Called(ctx, institutionID, classID)
	return args.Get(0).(*Class), args.Error(1)
}
func (m *MockStore) CreateClass(ctx context.Context, institutionID uuid.UUID, req *CreateClassRequest) (*Class, error) {
	args := m.Called(ctx, institutionID, req)
	return args.Get(0).(*Class), args.Error(1)
}
func (m *MockStore) AssignClassTeacher(ctx context.Context, institutionID, classID, teacherID uuid.UUID) error {
	return m.Called(ctx, institutionID, classID, teacherID).Error(0)
}
func (m *MockStore) ListTeachers(ctx context.Context, institutionID uuid.UUID) ([]*TeacherOption, error) {
	args := m.Called(ctx, institutionID)
	return args.Get(0).([]*TeacherOption), args.Error(1)
}
func (m *MockStore) ListAcademicYears(ctx context.Context, institutionID uuid.UUID) ([]*AcademicYear, error) {
	args := m.Called(ctx, institutionID)
	return args.Get(0).([]*AcademicYear), args.Error(1)
}
func (m *MockStore) ListSubjects(ctx context.Context, institutionID uuid.UUID) ([]*Subject, error) {
	args := m.Called(ctx, institutionID)
	return args.Get(0).([]*Subject), args.Error(1)
}
func (m *MockStore) GetCurrentTerm(ctx context.Context, institutionID uuid.UUID) (*uuid.UUID, error) {
	args := m.Called(ctx, institutionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*uuid.UUID), args.Error(1)
}
func (m *MockStore) GetTimetableForClass(ctx context.Context, institutionID, classID, termID uuid.UUID) ([]*TimetablePeriod, error) {
	args := m.Called(ctx, institutionID, classID, termID)
	return args.Get(0).([]*TimetablePeriod), args.Error(1)
}
func (m *MockStore) GetTimetableForTeacher(ctx context.Context, institutionID, teacherID, termID uuid.UUID) ([]*TimetablePeriod, error) {
	args := m.Called(ctx, institutionID, teacherID, termID)
	return args.Get(0).([]*TimetablePeriod), args.Error(1)
}
func (m *MockStore) GetTimetablePeriodByID(ctx context.Context, institutionID, periodID uuid.UUID) (*TimetablePeriod, error) {
	args := m.Called(ctx, institutionID, periodID)
	return args.Get(0).(*TimetablePeriod), args.Error(1)
}
func (m *MockStore) GetOrCreateSession(ctx context.Context, institutionID, classID, teacherID uuid.UUID, sessionDate time.Time, sessionType string, periodID *uuid.UUID) (*AttendanceSession, error) {
	args := m.Called(ctx, institutionID, classID, teacherID, sessionDate, sessionType, periodID)
	return args.Get(0).(*AttendanceSession), args.Error(1)
}
func (m *MockStore) GetSessionStudents(ctx context.Context, institutionID, classID, sessionID uuid.UUID) ([]*AttendanceRecord, error) {
	args := m.Called(ctx, institutionID, classID, sessionID)
	return args.Get(0).([]*AttendanceRecord), args.Error(1)
}
func (m *MockStore) SaveAttendanceRecords(ctx context.Context, institutionID, sessionID uuid.UUID, entries []AttendanceEntry) error {
	return m.Called(ctx, institutionID, sessionID, entries).Error(0)
}
func (m *MockStore) GetAttendanceSummary(ctx context.Context, institutionID, classID uuid.UUID, startDate, endDate time.Time) ([]*AttendanceSummary, error) {
	args := m.Called(ctx, institutionID, classID, startDate, endDate)
	return args.Get(0).([]*AttendanceSummary), args.Error(1)
}
func (m *MockStore) GetActiveStudentIDsForClass(ctx context.Context, institutionID, classID uuid.UUID) ([]uuid.UUID, error) {
	args := m.Called(ctx, institutionID, classID)
	return args.Get(0).([]uuid.UUID), args.Error(1)
}
func (m *MockStore) GetPrimaryContactForStudent(ctx context.Context, institutionID, studentID uuid.UUID) (*PrimaryContactInfo, error) {
	args := m.Called(ctx, institutionID, studentID)
	return args.Get(0).(*PrimaryContactInfo), args.Error(1)
}
func (m *MockStore) ListGradeComponents(ctx context.Context, institutionID, classID, subjectID, termID uuid.UUID) ([]*GradeComponent, error) {
	args := m.Called(ctx, institutionID, classID, subjectID, termID)
	return args.Get(0).([]*GradeComponent), args.Error(1)
}
func (m *MockStore) CreateGradeComponent(ctx context.Context, institutionID uuid.UUID, req *CreateGradeComponentRequest) (*GradeComponent, error) {
	args := m.Called(ctx, institutionID, req)
	return args.Get(0).(*GradeComponent), args.Error(1)
}
func (m *MockStore) GetClassRankings(ctx context.Context, institutionID, classID, termID uuid.UUID) ([]*StudentRank, error) {
	args := m.Called(ctx, institutionID, classID, termID)
	return args.Get(0).([]*StudentRank), args.Error(1)
}
func (m *MockStore) GetTermDates(ctx context.Context, institutionID, termID uuid.UUID) (time.Time, time.Time, error) {
	args := m.Called(ctx, institutionID, termID)
	return args.Get(0).(time.Time), args.Get(1).(time.Time), args.Error(2)
}
func (m *MockStore) GetAggregateMarksForStudent(ctx context.Context, institutionID, studentID, termID uuid.UUID) (float64, int, error) {
	args := m.Called(ctx, institutionID, studentID, termID)
	return args.Get(0).(float64), args.Get(1).(int), args.Error(2)
}
func (m *MockStore) UpsertReportCard(ctx context.Context, institutionID, classID, termID uuid.UUID, rc *ReportCard) error {
	return m.Called(ctx, institutionID, classID, termID, rc).Error(0)
}
func (m *MockStore) PublishReportCard(ctx context.Context, institutionID, studentID, termID uuid.UUID) (*ReportCard, error) {
	args := m.Called(ctx, institutionID, studentID, termID)
	return args.Get(0).(*ReportCard), args.Error(1)
}
func (m *MockStore) UnpublishReportCard(ctx context.Context, institutionID, studentID, termID uuid.UUID) (*ReportCard, error) {
	args := m.Called(ctx, institutionID, studentID, termID)
	return args.Get(0).(*ReportCard), args.Error(1)
}
func (m *MockStore) GetPublishedReportCardForStudent(ctx context.Context, institutionID, studentID uuid.UUID) (*ReportCard, error) {
	args := m.Called(ctx, institutionID, studentID)
	return args.Get(0).(*ReportCard), args.Error(1)
}
func (m *MockStore) GetSubjectScoresForStudent(ctx context.Context, institutionID, studentID, termID uuid.UUID) ([]*SubjectScore, error) {
	args := m.Called(ctx, institutionID, studentID, termID)
	return args.Get(0).([]*SubjectScore), args.Error(1)
}
func (m *MockStore) ListTerms(ctx context.Context, institutionID uuid.UUID) ([]*Term, error) {
	args := m.Called(ctx, institutionID)
	return args.Get(0).([]*Term), args.Error(1)
}
func (m *MockStore) GetInstitutionAttendanceToday(ctx context.Context, institutionID uuid.UUID, date time.Time) (*InstitutionAttendanceToday, error) {
	args := m.Called(ctx, institutionID, date)
	return args.Get(0).(*InstitutionAttendanceToday), args.Error(1)
}

// ─── The actual tests ───────────────────────────────────────────────
//
// Mirrors tonight's real curl test: James Odhiambo, timetabled for
// Mathematics in Form 1 East, tries to save marks for an English
// component in the same class — a subject he does NOT teach there.

func TestSaveGrades_NotTimetabled_Rejected(t *testing.T) {
	institutionID := uuid.New()
	teacherID := uuid.New() // stands in for James
	componentID := uuid.New() // stands in for the English/Form1East component
	classID := uuid.New()
	englishSubjectID := uuid.New()
	termID := uuid.New()

	store := new(MockStore)

	store.On("GetGradeComponentByID", mock.Anything, institutionID, componentID).
		Return(&GradeComponent{
			ID: componentID, ClassID: classID, SubjectID: englishSubjectID, TermID: termID,
		}, nil)

	// The real question: is James timetabled for English in this class?
	// We tell the mock: no.
	store.On("TeacherIsTimetabledForSubjectInClass", mock.Anything,
		institutionID, teacherID, classID, englishSubjectID, termID).
		Return(false, nil)

	// Deliberately no .On("SaveGradeEntries", ...) — if the real code
	// reaches the database write anyway, testify panics.

	svc := &Service{repo: store}

	marks := 60.0
	req := &BulkGradeEntryRequest{
		ComponentID: componentID,
		Entries:     []GradeInput{{StudentID: uuid.New(), Marks: &marks}},
	}

	err := svc.SaveGrades(context.Background(), institutionID, teacherID, true /* isTeacherRole */, req)

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNotTimetabledForSubject)
	store.AssertExpectations(t)
}

func TestSaveGrades_Timetabled_Succeeds(t *testing.T) {
	institutionID := uuid.New()
	teacherID := uuid.New() // James
	componentID := uuid.New() // Mathematics/Form1East component
	classID := uuid.New()
	mathSubjectID := uuid.New()
	termID := uuid.New()

	store := new(MockStore)

	store.On("GetGradeComponentByID", mock.Anything, institutionID, componentID).
		Return(&GradeComponent{
			ID: componentID, ClassID: classID, SubjectID: mathSubjectID, TermID: termID,
		}, nil)
	store.On("TeacherIsTimetabledForSubjectInClass", mock.Anything,
		institutionID, teacherID, classID, mathSubjectID, termID).
		Return(true, nil) // James IS timetabled for Math here

	store.On("SaveGradeEntries", mock.Anything, institutionID, componentID, teacherID, mock.Anything).
		Return(nil)

	svc := &Service{repo: store}

	marks := 78.0
	req := &BulkGradeEntryRequest{
		ComponentID: componentID,
		Entries:     []GradeInput{{StudentID: uuid.New(), Marks: &marks}},
	}

	err := svc.SaveGrades(context.Background(), institutionID, teacherID, true, req)

	require.NoError(t, err)
	store.AssertExpectations(t)
}

// Non-teacher roles (e.g. Margaret, super_admin) bypass the check
// entirely — mirrors the role-gate we verified live with curl earlier
// tonight. Deliberately does NOT set up GetGradeComponentByID or
// TeacherIsTimetabledForSubjectInClass expectations: if the service
// code calls either of them when isTeacherRole is false, this test
// will panic, proving the bypass actually skips those checks rather
// than just happening to return the right answer.
func TestSaveGrades_NonTeacherRole_BypassesCheck(t *testing.T) {
	institutionID := uuid.New()
	adminID := uuid.New()
	componentID := uuid.New()

	store := new(MockStore)
	store.On("SaveGradeEntries", mock.Anything, institutionID, componentID, adminID, mock.Anything).
		Return(nil)

	svc := &Service{repo: store}

	marks := 90.0
	req := &BulkGradeEntryRequest{
		ComponentID: componentID,
		Entries:     []GradeInput{{StudentID: uuid.New(), Marks: &marks}},
	}

	err := svc.SaveGrades(context.Background(), institutionID, adminID, false /* isTeacherRole */, req)

	require.NoError(t, err)
	store.AssertExpectations(t)
}