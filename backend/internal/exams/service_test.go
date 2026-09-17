package exams

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ── In-memory fake Store (no database) ──

type fakeStore struct {
	sessions map[uuid.UUID]*ExamSession
	papers   map[uuid.UUID]*ExamPaper
	groups   map[uuid.UUID][]*PaperGroup
	sittings map[uuid.UUID]*ExamSitting
	invig    map[uuid.UUID][]*InvigilatorAssignment
	arr      map[uuid.UUID][]*SpecialArrangement
	classes  map[uuid.UUID]string
	staff    map[uuid.UUID]bool
}

func newFake() *fakeStore {
	return &fakeStore{
		sessions: map[uuid.UUID]*ExamSession{},
		papers:   map[uuid.UUID]*ExamPaper{},
		groups:   map[uuid.UUID][]*PaperGroup{},
		sittings: map[uuid.UUID]*ExamSitting{},
		invig:    map[uuid.UUID][]*InvigilatorAssignment{},
		arr:      map[uuid.UUID][]*SpecialArrangement{},
		classes:  map[uuid.UUID]string{},
		staff:    map[uuid.UUID]bool{},
	}
}

func (f *fakeStore) Now() time.Time { return time.Now() }

func (f *fakeStore) CreateSession(_ context.Context, inst uuid.UUID, _ *uuid.UUID, req *CreateSessionRequest) (*ExamSession, error) {
	s := &ExamSession{ID: uuid.New(), InstitutionID: inst, Name: req.Name, ExamType: req.ExamType,
		StartDate: req.StartDate, EndDate: req.EndDate, Status: SessionDraft, Version: 1,
		AcademicYearID: req.AcademicYearID, TermID: req.TermID}
	f.sessions[s.ID] = s
	return s, nil
}

func (f *fakeStore) GetSession(_ context.Context, _ uuid.UUID, id uuid.UUID) (*ExamSession, error) {
	s, ok := f.sessions[id]
	if !ok {
		return nil, ErrSessionNotFound
	}
	return s, nil
}

func (f *fakeStore) ListSessions(_ context.Context, _ uuid.UUID) ([]*ExamSession, error) {
	out := []*ExamSession{}
	for _, s := range f.sessions {
		out = append(out, s)
	}
	return out, nil
}

func (f *fakeStore) SetSessionStatus(_ context.Context, _ uuid.UUID, id uuid.UUID, status string, v *int) (*ExamSession, error) {
	s, ok := f.sessions[id]
	if !ok {
		return nil, ErrSessionNotFound
	}
	s.Status = status
	if v != nil {
		s.Version = *v
	}
	return s, nil
}

func (f *fakeStore) CreatePaper(_ context.Context, inst, sid uuid.UUID, req *CreatePaperRequest) (*ExamPaper, error) {
	p := &ExamPaper{ID: uuid.New(), InstitutionID: inst, ExamSessionID: sid,
		AcademicOfferingID: req.AcademicOfferingID, PaperName: req.PaperName, PaperCode: req.PaperCode,
		PaperNumber: req.PaperNumber, DurationMinutes: req.DurationMinutes, MaxMarks: req.MaxMarks,
		Instructions: req.Instructions, IsShared: req.IsShared}
	f.papers[p.ID] = p
	return p, nil
}

func (f *fakeStore) ListPapers(_ context.Context, _ uuid.UUID, sid uuid.UUID) ([]*ExamPaper, error) {
	out := []*ExamPaper{}
	for _, p := range f.papers {
		if p.ExamSessionID == sid {
			out = append(out, p)
		}
	}
	return out, nil
}

func (f *fakeStore) GetPaper(_ context.Context, _ uuid.UUID, id uuid.UUID) (*ExamPaper, error) {
	p, ok := f.papers[id]
	if !ok {
		return nil, ErrPaperNotFound
	}
	return p, nil
}

func (f *fakeStore) AddPaperGroup(_ context.Context, inst, pid uuid.UUID, req *AddPaperGroupRequest) (*PaperGroup, error) {
	g := &PaperGroup{ID: uuid.New(), InstitutionID: inst, ExamPaperID: pid,
		ClassID: req.ClassID, Label: req.Label, CandidateCount: req.CandidateCount}
	if req.ClassID != nil {
		if n, ok := f.classes[*req.ClassID]; ok {
			g.ClassName = &n
		}
	}
	f.groups[pid] = append(f.groups[pid], g)
	return g, nil
}

func (f *fakeStore) ListGroupsForPapers(_ context.Context, _ uuid.UUID, ids []uuid.UUID) (map[uuid.UUID][]*PaperGroup, error) {
	out := map[uuid.UUID][]*PaperGroup{}
	for _, id := range ids {
		out[id] = f.groups[id]
	}
	return out, nil
}

func (f *fakeStore) CreateSitting(_ context.Context, inst, sid uuid.UUID, req *CreateSittingRequest) (*ExamSitting, error) {
	s := &ExamSitting{ID: uuid.New(), InstitutionID: inst, ExamSessionID: sid,
		ExamPaperID: req.ExamPaperID, ExamDate: req.ExamDate, StartTime: req.StartTime, EndTime: req.EndTime,
		RoomName: req.RoomName, RoomCapacity: req.RoomCapacity, CandidateCount: req.CandidateCount,
		SeatingNote: req.SeatingNote, Status: SittingDraft}
	f.sittings[s.ID] = s
	return s, nil
}

func (f *fakeStore) UpdateSitting(_ context.Context, _ uuid.UUID, id uuid.UUID, req *UpdateSittingRequest) (*ExamSitting, error) {
	s, ok := f.sittings[id]
	if !ok {
		return nil, ErrSittingNotFound
	}
	if req.ExamDate != nil {
		s.ExamDate = *req.ExamDate
	}
	if req.StartTime != nil {
		s.StartTime = *req.StartTime
	}
	if req.EndTime != nil {
		s.EndTime = *req.EndTime
	}
	if req.RoomName != nil {
		s.RoomName = req.RoomName
	}
	if req.RoomCapacity != nil {
		s.RoomCapacity = req.RoomCapacity
	}
	if req.CandidateCount != nil {
		s.CandidateCount = req.CandidateCount
	}
	if req.SeatingNote != nil {
		s.SeatingNote = req.SeatingNote
	}
	return s, nil
}

func (f *fakeStore) DeleteSitting(_ context.Context, _ uuid.UUID, id uuid.UUID) error {
	if _, ok := f.sittings[id]; !ok {
		return ErrSittingNotFound
	}
	delete(f.sittings, id)
	return nil
}

func (f *fakeStore) ListSittings(_ context.Context, _ uuid.UUID, sid uuid.UUID) ([]*ExamSitting, error) {
	out := []*ExamSitting{}
	for _, s := range f.sittings {
		if s.ExamSessionID == sid {
			out = append(out, s)
		}
	}
	return out, nil
}

func (f *fakeStore) GetSitting(_ context.Context, _ uuid.UUID, id uuid.UUID) (*ExamSitting, error) {
	s, ok := f.sittings[id]
	if !ok {
		return nil, ErrSittingNotFound
	}
	return s, nil
}

func (f *fakeStore) AssignInvigilator(_ context.Context, inst, sid uuid.UUID, req *AssignInvigilatorRequest) (*InvigilatorAssignment, error) {
	a := &InvigilatorAssignment{ID: uuid.New(), InstitutionID: inst, ExamSittingID: sid,
		StaffID: req.StaffID, Role: req.Role, Notes: req.Notes}
	f.invig[sid] = append(f.invig[sid], a)
	return a, nil
}

func (f *fakeStore) RemoveInvigilator(_ context.Context, _ uuid.UUID, sid, staff uuid.UUID) error {
	kept := []*InvigilatorAssignment{}
	for _, a := range f.invig[sid] {
		if a.StaffID != staff {
			kept = append(kept, a)
		}
	}
	f.invig[sid] = kept
	return nil
}

func (f *fakeStore) ListInvigilators(_ context.Context, _ uuid.UUID, ids []uuid.UUID) (map[uuid.UUID][]*InvigilatorAssignment, error) {
	out := map[uuid.UUID][]*InvigilatorAssignment{}
	for _, id := range ids {
		out[id] = f.invig[id]
	}
	return out, nil
}

func (f *fakeStore) AddArrangement(_ context.Context, inst, sid uuid.UUID, req *AddArrangementRequest) (*SpecialArrangement, error) {
	a := &SpecialArrangement{ID: uuid.New(), InstitutionID: inst, ExamSittingID: sid,
		StudentID: req.StudentID, ClassID: req.ClassID, ArrangementType: req.ArrangementType,
		Details: req.Details, ExtraMinutes: req.ExtraMinutes, RoomName: req.RoomName}
	f.arr[sid] = append(f.arr[sid], a)
	return a, nil
}

func (f *fakeStore) ListArrangements(_ context.Context, _ uuid.UUID, ids []uuid.UUID) (map[uuid.UUID][]*SpecialArrangement, error) {
	out := map[uuid.UUID][]*SpecialArrangement{}
	for _, id := range ids {
		out[id] = f.arr[id]
	}
	return out, nil
}

func (f *fakeStore) ClassExists(_ context.Context, _ uuid.UUID, id uuid.UUID) (bool, error) {
	_, ok := f.classes[id]
	return ok, nil
}

func (f *fakeStore) OfferingExists(_ context.Context, _ uuid.UUID, _ uuid.UUID) (bool, error) {
	return true, nil
}

func (f *fakeStore) OfferingName(_ context.Context, _ uuid.UUID, _ uuid.UUID) (string, string, error) {
	return "Mathematics", "subject", nil
}

func (f *fakeStore) StaffExists(_ context.Context, _ uuid.UUID, id uuid.UUID) (bool, error) {
	return f.staff[id], nil
}

func (f *fakeStore) ClassName(_ context.Context, _ uuid.UUID, id uuid.UUID) (string, error) {
	return f.classes[id], nil
}

func (f *fakeStore) StudentCountForClass(_ context.Context, _ uuid.UUID, _ uuid.UUID) (int, error) {
	return 40, nil
}

// ── Helpers ──

func testSetup() (*fakeStore, *Service, uuid.UUID, *ExamSession) {
	ctx := context.Background()
	inst := uuid.New()
	fs := newFake()
	svc := NewService(fs)
	sess, _ := svc.CreateSession(ctx, inst, nil, &CreateSessionRequest{
		Name: "End of Term 2", ExamType: "end_of_term",
		StartDate: "2026-07-01", EndDate: "2026-07-10",
	})
	return fs, svc, inst, sess
}

func mustPaper(t *testing.T, svc *Service, inst uuid.UUID, sess *ExamSession, name string, mins int) *ExamPaper {
	t.Helper()
	p, err := svc.CreatePaper(context.Background(), inst, sess.ID, &CreatePaperRequest{
		PaperName: name, DurationMinutes: mins,
	})
	require.NoError(t, err)
	return p
}

func mustGroup(t *testing.T, svc *Service, inst uuid.UUID, paper *ExamPaper, classID *uuid.UUID, label *string) *PaperGroup {
	t.Helper()
	g, err := svc.AddPaperGroup(context.Background(), inst, paper.ID, &AddPaperGroupRequest{
		ClassID: classID, Label: label,
	})
	require.NoError(t, err)
	return g
}

func mustSitting(t *testing.T, svc *Service, inst uuid.UUID, sess *ExamSession, paper *ExamPaper,
	date, start, end string, room *string) *ExamSitting {
	t.Helper()
	s, err := svc.CreateSitting(context.Background(), inst, sess.ID, &CreateSittingRequest{
		ExamPaperID: paper.ID, ExamDate: date, StartTime: start, EndTime: end, RoomName: room,
	})
	require.NoError(t, err)
	return s
}

// ── Tests ──

func TestCreateSession_Valid(t *testing.T) {
	_, _, inst, sess := testSetup()
	require.NotNil(t, sess)
	assert.Equal(t, "draft", sess.Status)
	assert.Equal(t, 1, sess.Version)
	assert.Equal(t, inst, sess.InstitutionID)
}

func TestCreateSession_Invalid(t *testing.T) {
	_, svc, inst, _ := testSetup()
	_, err := svc.CreateSession(context.Background(), inst, nil, &CreateSessionRequest{
		Name: "", ExamType: "end_of_term", StartDate: "2026-07-01", EndDate: "2026-07-10",
	})
	assert.ErrorIs(t, err, ErrInvalidSession)

	_, err = svc.CreateSession(context.Background(), inst, nil, &CreateSessionRequest{
		Name: "X", ExamType: "end_of_term", StartDate: "2026-07-10", EndDate: "2026-07-01",
	})
	assert.Error(t, err)
}

func TestCreatePaper_InvalidDuration(t *testing.T) {
	_, svc, inst, sess := testSetup()
	_, err := svc.CreatePaper(context.Background(), inst, sess.ID, &CreatePaperRequest{
		PaperName: "Maths P1", DurationMinutes: 0,
	})
	assert.ErrorIs(t, err, ErrInvalidPaper)
}

func TestAddPaperGroup_RequiresClassOrLabel(t *testing.T) {
	fs, svc, inst, sess := testSetup()
	_ = fs
	p := mustPaper(t, svc, inst, sess, "Maths P1", 120)
	_, err := svc.AddPaperGroup(context.Background(), inst, p.ID, &AddPaperGroupRequest{})
	assert.ErrorIs(t, err, ErrInvalidGroup)
}

func TestAddPaperGroup_RejectsUnknownClass(t *testing.T) {
	_, svc, inst, sess := testSetup()
	p := mustPaper(t, svc, inst, sess, "Maths P1", 120)
	unknown := uuid.New()
	_, err := svc.AddPaperGroup(context.Background(), inst, p.ID, &AddPaperGroupRequest{ClassID: &unknown})
	assert.ErrorIs(t, err, ErrClassNotFound)
}

func TestCreateSitting_InvalidTimes(t *testing.T) {
	_, svc, inst, sess := testSetup()
	p := mustPaper(t, svc, inst, sess, "Maths P1", 120)
	label := "Form 4"
	mustGroup(t, svc, inst, p, nil, &label)

	_, err := svc.CreateSitting(context.Background(), inst, sess.ID, &CreateSittingRequest{
		ExamPaperID: p.ID, ExamDate: "2026-07-02", StartTime: "10:00", EndTime: "09:00",
	})
	assert.Error(t, err)

	_, err = svc.CreateSitting(context.Background(), inst, sess.ID, &CreateSittingRequest{
		ExamPaperID: p.ID, ExamDate: "", StartTime: "08:00", EndTime: "10:00",
	})
	assert.Error(t, err)

	_, err = svc.CreateSitting(context.Background(), inst, sess.ID, &CreateSittingRequest{
		ExamPaperID: p.ID, ExamDate: "2026-07-02", StartTime: "08:00", EndTime: "08:00",
	})
	assert.Error(t, err)
}

func TestCreateSitting_RejectsForeignPaper(t *testing.T) {
	_, svc, inst, sess := testSetup()
	other, err := svc.CreateSession(context.Background(), inst, nil, &CreateSessionRequest{
		Name: "Other", ExamType: "mock", StartDate: "2026-08-01", EndDate: "2026-08-05",
	})
	require.NoError(t, err)
	p := mustPaper(t, svc, inst, other, "Physics P1", 90)
	_, err = svc.CreateSitting(context.Background(), inst, sess.ID, &CreateSittingRequest{
		ExamPaperID: p.ID, ExamDate: "2026-07-02", StartTime: "08:00", EndTime: "09:30",
	})
	assert.Error(t, err)
}

func TestValidate_DetectsCandidateConflict(t *testing.T) {
	fs, svc, inst, sess := testSetup()
	classA := uuid.New()
	fs.classes[classA] = "Form 4 East"
	p1 := mustPaper(t, svc, inst, sess, "Maths P1", 120)
	p2 := mustPaper(t, svc, inst, sess, "English P1", 120)
	mustGroup(t, svc, inst, p1, &classA, nil)
	mustGroup(t, svc, inst, p2, &classA, nil)
	mustSitting(t, svc, inst, sess, p1, "2026-07-02", "08:00", "10:00", nil)
	mustSitting(t, svc, inst, sess, p2, "2026-07-02", "09:00", "11:00", nil)

	res, err := svc.Validate(context.Background(), inst, sess.ID)
	require.NoError(t, err)
	assert.False(t, res.Valid)
	found := false
	for _, b := range res.Blocking {
		if b.Code == "CANDIDATE_OVERLAP" {
			found = true
		}
	}
	assert.True(t, found, "expected CANDIDATE_OVERLAP, got %+v", res.Blocking)
}

func TestValidate_DetectsRoomConflict(t *testing.T) {
	fs, svc, inst, sess := testSetup()
	classA, classB := uuid.New(), uuid.New()
	fs.classes[classA] = "Form 4 East"
	fs.classes[classB] = "Form 4 West"
	p1 := mustPaper(t, svc, inst, sess, "Maths P1", 120)
	p2 := mustPaper(t, svc, inst, sess, "English P1", 120)
	mustGroup(t, svc, inst, p1, &classA, nil)
	mustGroup(t, svc, inst, p2, &classB, nil)
	room := "Hall A"
	mustSitting(t, svc, inst, sess, p1, "2026-07-02", "08:00", "10:00", &room)
	mustSitting(t, svc, inst, sess, p2, "2026-07-02", "09:00", "11:00", &room)

	res, err := svc.Validate(context.Background(), inst, sess.ID)
	require.NoError(t, err)
	assert.False(t, res.Valid)
	found := false
	for _, b := range res.Blocking {
		if b.Code == "ROOM_OVERLAP" {
			found = true
		}
	}
	assert.True(t, found)
}

func TestValidate_DetectsInvigilatorConflict(t *testing.T) {
	fs, svc, inst, sess := testSetup()
	classA, classB := uuid.New(), uuid.New()
	fs.classes[classA] = "Form 4 East"
	fs.classes[classB] = "Form 4 West"
	staff := uuid.New()
	fs.staff[staff] = true
	p1 := mustPaper(t, svc, inst, sess, "Maths P1", 120)
	p2 := mustPaper(t, svc, inst, sess, "English P1", 120)
	mustGroup(t, svc, inst, p1, &classA, nil)
	mustGroup(t, svc, inst, p2, &classB, nil)
	r1, r2 := "Hall A", "Hall B"
	s1 := mustSitting(t, svc, inst, sess, p1, "2026-07-02", "08:00", "10:00", &r1)
	s2 := mustSitting(t, svc, inst, sess, p2, "2026-07-02", "09:00", "11:00", &r2)
	_, err := svc.AssignInvigilator(context.Background(), inst, s1.ID, &AssignInvigilatorRequest{StaffID: staff, Role: "invigilator"})
	require.NoError(t, err)
	_, err = svc.AssignInvigilator(context.Background(), inst, s2.ID, &AssignInvigilatorRequest{StaffID: staff, Role: "invigilator"})
	require.NoError(t, err)

	res, err := svc.Validate(context.Background(), inst, sess.ID)
	require.NoError(t, err)
	assert.False(t, res.Valid)
	found := false
	for _, b := range res.Blocking {
		if b.Code == "INVIGILATOR_OVERLAP" {
			found = true
		}
	}
	assert.True(t, found)
}

func TestValidate_CapacityExceededAndUnavailable(t *testing.T) {
	fs, svc, inst, sess := testSetup()
	classA := uuid.New()
	fs.classes[classA] = "Form 4 East"
	p1 := mustPaper(t, svc, inst, sess, "Maths P1", 120)
	p2 := mustPaper(t, svc, inst, sess, "English P1", 60)
	mustGroup(t, svc, inst, p1, &classA, nil)
	mustGroup(t, svc, inst, p2, &classA, nil)
	// Exceeded: 50 candidates in capacity-30 room.
	_, err := svc.CreateSitting(context.Background(), inst, sess.ID, &CreateSittingRequest{
		ExamPaperID: p1.ID, ExamDate: "2026-07-02", StartTime: "08:00", EndTime: "10:00",
		RoomName: strp("Hall A"), RoomCapacity: intp(30), CandidateCount: intp(50),
	})
	require.NoError(t, err)
	// Unavailable: no capacity data at all.
	_, err = svc.CreateSitting(context.Background(), inst, sess.ID, &CreateSittingRequest{
		ExamPaperID: p2.ID, ExamDate: "2026-07-03", StartTime: "08:00", EndTime: "09:00",
	})
	require.NoError(t, err)

	res, err := svc.Validate(context.Background(), inst, sess.ID)
	require.NoError(t, err)
	assert.False(t, res.Valid)
	foundBlock, foundWarn := false, false
	for _, b := range res.Blocking {
		if b.Code == "ROOM_CAPACITY_EXCEEDED" {
			foundBlock = true
		}
	}
	for _, w := range res.Warnings {
		if w.Code == "ROOM_CAPACITY_UNAVAILABLE" {
			foundWarn = true
		}
	}
	assert.True(t, foundBlock, "expected capacity blocking error")
	assert.True(t, foundWarn, "expected capacity-unavailable warning, got %+v", res.Warnings)
}

func TestPublish_RejectedWithBlockingErrors(t *testing.T) {
	fs, svc, inst, sess := testSetup()
	classA := uuid.New()
	fs.classes[classA] = "Form 4 East"
	p1 := mustPaper(t, svc, inst, sess, "Maths P1", 120)
	p2 := mustPaper(t, svc, inst, sess, "English P1", 120)
	mustGroup(t, svc, inst, p1, &classA, nil)
	mustGroup(t, svc, inst, p2, &classA, nil)
	mustSitting(t, svc, inst, sess, p1, "2026-07-02", "08:00", "10:00", nil)
	mustSitting(t, svc, inst, sess, p2, "2026-07-02", "09:00", "11:00", nil)

	_, err := svc.Publish(context.Background(), inst, sess.ID)
	assert.ErrorIs(t, err, ErrBlockingConflicts)
	after, _ := svc.GetSession(context.Background(), inst, sess.ID)
	assert.Equal(t, "draft", after.Status)
}

func TestPublish_ValidScheduleAndImmutability(t *testing.T) {
	fs, svc, inst, sess := testSetup()
	_ = fs
	classA := uuid.New()
	fs.classes[classA] = "Form 4 East"
	p1 := mustPaper(t, svc, inst, sess, "Maths P1", 120)
	mustGroup(t, svc, inst, p1, &classA, nil)
	cap, cnt := 60, 45
	room := "Hall A"
	_, err := svc.CreateSitting(context.Background(), inst, sess.ID, &CreateSittingRequest{
		ExamPaperID: p1.ID, ExamDate: "2026-07-02", StartTime: "08:00", EndTime: "10:00",
		RoomName: &room, RoomCapacity: &cap, CandidateCount: &cnt,
	})
	require.NoError(t, err)

	res, err := svc.Validate(context.Background(), inst, sess.ID)
	require.NoError(t, err)
	require.True(t, res.Valid, "expected valid, got %+v", res.Blocking)

	pub, err := svc.Publish(context.Background(), inst, sess.ID)
	require.NoError(t, err)
	assert.Equal(t, "published", pub.Status)

	// Direct edits now rejected.
	_, err = svc.CreatePaper(context.Background(), inst, sess.ID, &CreatePaperRequest{
		PaperName: "Late Paper", DurationMinutes: 60,
	})
	assert.ErrorIs(t, err, ErrSessionPublished)

	sittings, _ := svc.repo.ListSittings(context.Background(), inst, sess.ID)
	require.Len(t, sittings, 1)
	_, err = svc.UpdateSitting(context.Background(), inst, sittings[0].ID, &UpdateSittingRequest{SeatingNote: strp("x")})
	assert.ErrorIs(t, err, ErrSessionPublished)
	assert.ErrorIs(t, svc.DeleteSitting(context.Background(), inst, sittings[0].ID), ErrSessionPublished)

	// Controlled revision works.
	rev, err := svc.ReopenToDraft(context.Background(), inst, sess.ID)
	require.NoError(t, err)
	assert.Equal(t, "draft", rev.Status)
	assert.Equal(t, 2, rev.Version)
}

func TestCanonicalViewConsistency(t *testing.T) {
	fs, svc, inst, sess := testSetup()
	classA := uuid.New()
	fs.classes[classA] = "Form 4 East"
	staff := uuid.New()
	fs.staff[staff] = true
	p1 := mustPaper(t, svc, inst, sess, "Mathematics Paper 1", 120)
	mustGroup(t, svc, inst, p1, &classA, nil)
	room, cap, cnt := "Hall A", 60, 45
	s1 := mustSitting(t, svc, inst, sess, p1, "2026-07-02", "08:00", "10:00", &room)
	// enrich sitting with capacity + invigilator
	_, err := svc.UpdateSitting(context.Background(), inst, s1.ID, &UpdateSittingRequest{
		RoomCapacity: &cap, CandidateCount: &cnt,
	})
	require.NoError(t, err)
	_, err = svc.AssignInvigilator(context.Background(), inst, s1.ID, &AssignInvigilatorRequest{StaffID: staff, Role: "chief"})
	require.NoError(t, err)
	_, err = svc.Publish(context.Background(), inst, sess.ID)
	require.NoError(t, err)

	master, err := svc.MasterTimetable(context.Background(), inst, sess.ID, true)
	require.NoError(t, err)
	require.Len(t, master, 1)
	mv := master[0]
	require.NotNil(t, mv.Paper)
	assert.Equal(t, "Mathematics Paper 1", mv.Paper.PaperName)
	assert.Equal(t, "2026-07-02", mv.Sitting.ExamDate)
	assert.Equal(t, "08:00", mv.Sitting.StartTime)
	assert.Equal(t, "10:00", mv.Sitting.EndTime)
	require.NotNil(t, mv.Sitting.RoomName)
	assert.Equal(t, "Hall A", *mv.Sitting.RoomName)

	cand, err := svc.CandidateView(context.Background(), inst, sess.ID, classA, true)
	require.NoError(t, err)
	require.Len(t, cand, 1)
	assert.Equal(t, s1.ID, cand[0].Sitting.ID)
	assert.Equal(t, mv.Paper.PaperName, cand[0].Paper.PaperName)

	pv, err := svc.PaperView(context.Background(), inst, sess.ID, p1.ID)
	require.NoError(t, err)
	require.Len(t, pv, 1)
	assert.Equal(t, s1.ID, pv[0].Sitting.ID)

	rv, err := svc.RoomView(context.Background(), inst, sess.ID, "hall a", true)
	require.NoError(t, err)
	require.Len(t, rv, 1)
	assert.Equal(t, s1.ID, rv[0].Sitting.ID)

	iv, err := svc.InvigilatorView(context.Background(), inst, sess.ID, staff, true)
	require.NoError(t, err)
	require.Len(t, iv, 1)
	assert.Equal(t, s1.ID, iv[0].Sitting.ID)
	require.Len(t, iv[0].Invigilators, 1)
	assert.Equal(t, "chief", iv[0].Invigilators[0].Role)

	// Candidate view for an unrelated class is empty.
	other := uuid.New()
	empty, err := svc.CandidateView(context.Background(), inst, sess.ID, other, true)
	require.NoError(t, err)
	assert.Empty(t, empty)
}

func TestGenerateDraft_DeterministicAndReportsUnscheduled(t *testing.T) {
	fs, svc, inst, sess := testSetup()
	classA, classB := uuid.New(), uuid.New()
	fs.classes[classA] = "Form 4 East"
	fs.classes[classB] = "Form 4 West"
	p1 := mustPaper(t, svc, inst, sess, "Maths P1", 120)
	p2 := mustPaper(t, svc, inst, sess, "English P1", 90)
	p3 := mustPaper(t, svc, inst, sess, "Orphan Paper", 60) // no groups
	mustGroup(t, svc, inst, p1, &classA, nil)
	mustGroup(t, svc, inst, p2, &classB, nil)
	_ = p3

	res, err := svc.GenerateDraft(context.Background(), inst, sess.ID, &GenerateDraftRequest{
		StartDate: "2026-07-01", EndDate: "2026-07-03",
		DailyStartTime: "08:00", DailyEndTime: "12:00", GapMinutes: 15,
	})
	require.NoError(t, err)
	assert.Len(t, res.Scheduled, 2)
	// Deterministic: longest/candidate-heaviest first → Maths first at 08:00.
	assert.Equal(t, "2026-07-01", res.Scheduled[0].ExamDate)
	assert.Equal(t, "08:00", res.Scheduled[0].StartTime)
	require.Len(t, res.Unscheduled, 1)
	assert.Equal(t, p3.ID, res.Unscheduled[0].PaperID)
	assert.Contains(t, res.Unscheduled[0].Reason, "no candidate groups")

	// Session still draft — generator never publishes.
	after, _ := svc.GetSession(context.Background(), inst, sess.ID)
	assert.Equal(t, "draft", after.Status)
}

func strp(s string) *string { return &s }
func intp(i int) *int       { return &i }
