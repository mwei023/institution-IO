package cbc

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"institution-os/internal/academics"
)

// MockStore is a fake Store — it doesn't touch a real database. Every
// method just records that it was called and returns whatever we told
// it to return with .On(...). This is the exact same shape as
// MockAuthRepo in auth_test.go, just implementing our new Store
// interface instead of auth's repository interface.
type MockStore struct {
	mock.Mock
}

// We only need to implement the methods our tests actually call.
// Go requires every method in the interface to exist on MockStore for
// it to compile as a Store — but for methods we never call in a given
// test, testify's mock just won't have a matching .On(...) expectation
// and will panic loudly if accidentally called. That's a feature: it
// means a test can't silently pass while depending on behavior nobody
// set up.

func (m *MockStore) GetComponentClassID(ctx context.Context, institutionID, componentID uuid.UUID) (uuid.UUID, error) {
	args := m.Called(ctx, institutionID, componentID)
	return args.Get(0).(uuid.UUID), args.Error(1)
}

func (m *MockStore) TeacherHasLearningAreaAssignment(ctx context.Context, institutionID, teacherID, classID uuid.UUID) (bool, error) {
	args := m.Called(ctx, institutionID, teacherID, classID)
	return args.Bool(0), args.Error(1)
}

func (m *MockStore) SaveEntries(ctx context.Context, institutionID, componentID, enteredBy uuid.UUID, entries []EntryInput) error {
	args := m.Called(ctx, institutionID, componentID, enteredBy, entries)
	return args.Error(0)
}

// MockAcademicsRepo stubs the service's cross-package dependency
// (the subset of *academics.Repository declared as Service.AcademicsRepo).
// Before this existed, tests built Service{repo: store} with a nil
// academicsRepo and panicked the moment SaveEntries touched it.
type MockAcademicsRepo struct {
	mock.Mock
}

func (m *MockAcademicsRepo) IsChildOfParent(ctx context.Context, institutionID, parentUserID, studentID uuid.UUID) (bool, error) {
	args := m.Called(ctx, institutionID, parentUserID, studentID)
	return args.Bool(0), args.Error(1)
}
func (m *MockAcademicsRepo) TeacherHasAssignmentInClass(ctx context.Context, institutionID, teacherID, classID uuid.UUID) (bool, error) {
	args := m.Called(ctx, institutionID, teacherID, classID)
	return args.Bool(0), args.Error(1)
}
func (m *MockAcademicsRepo) GetClassByID(ctx context.Context, institutionID, classID uuid.UUID) (*academics.Class, error) {
	args := m.Called(ctx, institutionID, classID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*academics.Class), args.Error(1)
}
func (m *MockAcademicsRepo) GetActiveStudentIDsForClass(ctx context.Context, institutionID, classID uuid.UUID) ([]uuid.UUID, error) {
	args := m.Called(ctx, institutionID, classID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]uuid.UUID), args.Error(1)
}
func (m *MockAcademicsRepo) GetTermDates(ctx context.Context, institutionID, termID uuid.UUID) (time.Time, time.Time, error) {
	args := m.Called(ctx, institutionID, termID)
	return args.Get(0).(time.Time), args.Get(1).(time.Time), args.Error(2)
}
func (m *MockAcademicsRepo) GetAttendanceSummary(ctx context.Context, institutionID, classID uuid.UUID, startDate, endDate time.Time) ([]*academics.AttendanceSummary, error) {
	args := m.Called(ctx, institutionID, classID, startDate, endDate)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*academics.AttendanceSummary), args.Error(1)
}

func (m *MockStore) GetEntries(ctx context.Context, institutionID, componentID uuid.UUID) ([]*Entry, error) {
	args := m.Called(ctx, institutionID, componentID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*Entry), args.Error(1)
}

// The rest of the Store interface's methods aren't used by the tests
// in this file, but Go still requires MockStore to have every method
// on the interface to compile as one. Each just calls m.Called(...) and
// returns zero values — if a test that doesn't set up an expectation
// for one of these accidentally triggers it, testify panics immediately
// with a clear "I don't know how to handle this" error, rather than
// silently returning nothing and passing anyway.

func (m *MockStore) GetInstitutionCurriculumType(ctx context.Context, institutionID uuid.UUID) (string, error) {
	args := m.Called(ctx, institutionID)
	return args.String(0), args.Error(1)
}
func (m *MockStore) CreateLearningArea(ctx context.Context, institutionID uuid.UUID, req *CreateLearningAreaRequest) (*LearningArea, error) {
	args := m.Called(ctx, institutionID, req)
	return args.Get(0).(*LearningArea), args.Error(1)
}
func (m *MockStore) ListLearningAreas(ctx context.Context, institutionID uuid.UUID) ([]*LearningArea, error) {
	args := m.Called(ctx, institutionID)
	return args.Get(0).([]*LearningArea), args.Error(1)
}
func (m *MockStore) CreateComponent(ctx context.Context, institutionID uuid.UUID, req *CreateComponentRequest) (*Component, error) {
	args := m.Called(ctx, institutionID, req)
	return args.Get(0).(*Component), args.Error(1)
}
func (m *MockStore) ListComponents(ctx context.Context, institutionID, classID, learningAreaID, termID uuid.UUID) ([]*Component, error) {
	args := m.Called(ctx, institutionID, classID, learningAreaID, termID)
	return args.Get(0).([]*Component), args.Error(1)
}
func (m *MockStore) GetAllLearningAreaPercentagesForStudent(ctx context.Context, institutionID, studentID, termID uuid.UUID) (map[uuid.UUID]float64, error) {
	args := m.Called(ctx, institutionID, studentID, termID)
	return args.Get(0).(map[uuid.UUID]float64), args.Error(1)
}
func (m *MockStore) CreateTeacherAssignment(ctx context.Context, institutionID uuid.UUID, classID, learningAreaID, teacherID, termID uuid.UUID) (*TeacherAssignment, error) {
	args := m.Called(ctx, institutionID, classID, learningAreaID, teacherID, termID)
	return args.Get(0).(*TeacherAssignment), args.Error(1)
}
func (m *MockStore) GetRatingBands(ctx context.Context, institutionID uuid.UUID, curriculumType string, gradeBand *string) ([]*RatingBand, error) {
	args := m.Called(ctx, institutionID, curriculumType, gradeBand)
	return args.Get(0).([]*RatingBand), args.Error(1)
}
func (m *MockStore) GetStudentNames(ctx context.Context, institutionID uuid.UUID, studentIDs []uuid.UUID) (map[uuid.UUID]string, error) {
	args := m.Called(ctx, institutionID, studentIDs)
	return args.Get(0).(map[uuid.UUID]string), args.Error(1)
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

// ─── The actual test ──────────────────────────────────────────────
//
// This is the automated version of the exact curl test we ran by hand
// tonight: David Mumo, assigned to Grade 7 A + Mathematics, tries to
// save marks for a component belonging to Grade 8 A — a class he is
// NOT assigned to. Expect ErrNotAssignedToLearningArea, and expect
// the real database write (SaveEntries on the mock) to never happen.

func TestSaveEntries_NotAssignedToLearningArea(t *testing.T) {
	institutionID := uuid.New()
	teacherID := uuid.New()      // stands in for David Mumo
	componentID := uuid.New()    // stands in for the Grade 8 A component
	unassignedClassID := uuid.New()

	store := new(MockStore)
	acad := new(MockAcademicsRepo)

	// Step 1 of what SaveEntries does internally: look up which class
	// this component belongs to.
	store.On("GetComponentClassID", mock.Anything, institutionID, componentID).
		Return(unassignedClassID, nil)

	// Step 2: check whether the teacher is assigned to that class.
	// We're telling the mock: "when asked, say NO" — this is the
	// entire point of the test, simulating the exact real-world state
	// we seeded tonight (David assigned to Grade 7 A, not Grade 8 A).
	acad.On("TeacherHasAssignmentInClass", mock.Anything, institutionID, teacherID, unassignedClassID).
		Return(false, nil)

	// Deliberately NOT setting up an expectation for SaveEntries here.
	// If the service code has a bug and calls it anyway, testify will
	// panic with "mock: I don't know what to return" — which is
	// exactly what we want: proof the real database write never
	// happens when authorization fails.

	svc := &Service{repo: store, academicsRepo: acad}

	req := &BulkEntryRequest{
		ComponentID: componentID,
		Entries: []EntryInput{
			{StudentID: uuid.New(), Marks: floatPtr(60), IsAbsent: false},
		},
	}

	err := svc.SaveEntries(context.Background(), institutionID, teacherID, req)

	// The real assertion: did we get exactly the error the fix is
	// supposed to produce?
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNotAssignedToLearningArea)

	// Confirms every .On(...) we set up was actually called — catches
	// the case where the test would "pass" but the code never even
	// reached the checks we're testing.
	store.AssertExpectations(t)
}

// The positive case: same class, but this time the teacher IS
// assigned. Expect success, and expect the real SaveEntries write to
// actually be called with the right arguments.
func TestSaveEntries_Assigned_Succeeds(t *testing.T) {
	institutionID := uuid.New()
	teacherID := uuid.New()
	componentID := uuid.New()
	assignedClassID := uuid.New()

	store := new(MockStore)
	acad := new(MockAcademicsRepo)

	store.On("GetComponentClassID", mock.Anything, institutionID, componentID).
		Return(assignedClassID, nil)
	acad.On("TeacherHasAssignmentInClass", mock.Anything, institutionID, teacherID, assignedClassID).
		Return(true, nil) // this time: yes, assigned

	// This time we DO expect SaveEntries to actually be called — it's
	// the whole point of the positive case. mock.Anything for the
	// entries argument since we're only checking that it's reached at
	// all, not the exact slice contents.
	store.On("SaveEntries", mock.Anything, institutionID, componentID, teacherID, mock.Anything).
		Return(nil)

	svc := &Service{repo: store, academicsRepo: acad}

	req := &BulkEntryRequest{
		ComponentID: componentID,
		Entries: []EntryInput{
			{StudentID: uuid.New(), Marks: floatPtr(85), IsAbsent: false},
		},
	}

	err := svc.SaveEntries(context.Background(), institutionID, teacherID, req)

	require.NoError(t, err)
	store.AssertExpectations(t)
}

func floatPtr(f float64) *float64 { return &f } 