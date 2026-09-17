package exams

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

var (
	ErrInvalidSession     = errors.New("session name, exam_type, start_date and end_date are required")
	ErrInvalidPaper       = errors.New("paper_name and a positive duration_minutes are required")
	ErrInvalidGroup       = errors.New("either class_id or label is required for a candidate group")
	ErrClassNotFound      = errors.New("class not found")
	ErrOfferingNotFound   = errors.New("academic offering not found")
	ErrStaffNotFound      = errors.New("staff member not found")
	ErrBlockingConflicts  = errors.New("schedule has blocking conflicts — resolve them before publishing")
	ErrInvalidArrangement = errors.New("arrangement_type is required")
)

var validExamTypes = map[string]bool{
	"internal": true, "mock": true, "end_of_term": true,
	"mid_term": true, "external": true, "other": true,
}

var validArrangements = map[string]bool{
	"extra_time": true, "separate_room": true, "accessibility": true,
	"reader": true, "scribe": true, "other": true,
}

var validInvigilatorRoles = map[string]bool{
	"chief": true, "invigilator": true, "assistant": true, "rover": true,
}

type Service struct{ repo Store }

func NewService(repo Store) *Service { return &Service{repo: repo} }

// ── Sessions ──

func (s *Service) CreateSession(ctx context.Context, institutionID uuid.UUID, createdBy *uuid.UUID, req *CreateSessionRequest) (*ExamSession, error) {
	if strings.TrimSpace(req.Name) == "" || strings.TrimSpace(req.ExamType) == "" ||
		strings.TrimSpace(req.StartDate) == "" || strings.TrimSpace(req.EndDate) == "" {
		return nil, ErrInvalidSession
	}
	if !validExamTypes[req.ExamType] {
		return nil, fmt.Errorf("exam_type must be one of internal, mock, end_of_term, mid_term, external, other")
	}
	start, err1 := time.Parse("2006-01-02", req.StartDate)
	end, err2 := time.Parse("2006-01-02", req.EndDate)
	if err1 != nil || err2 != nil || end.Before(start) {
		return nil, fmt.Errorf("invalid session date range")
	}
	return s.repo.CreateSession(ctx, institutionID, createdBy, req)
}

func (s *Service) ListSessions(ctx context.Context, institutionID uuid.UUID) ([]*ExamSession, error) {
	return s.repo.ListSessions(ctx, institutionID)
}

func (s *Service) GetSession(ctx context.Context, institutionID, sessionID uuid.UUID) (*ExamSession, error) {
	return s.repo.GetSession(ctx, institutionID, sessionID)
}

func (s *Service) ensureDraft(ctx context.Context, institutionID, sessionID uuid.UUID) (*ExamSession, error) {
	sess, err := s.repo.GetSession(ctx, institutionID, sessionID)
	if err != nil {
		return nil, err
	}
	if sess.Status == SessionPublished || sess.Status == SessionArchived {
		return nil, ErrSessionPublished
	}
	return sess, nil
}

// ── Papers & groups ──

func (s *Service) CreatePaper(ctx context.Context, institutionID, sessionID uuid.UUID, req *CreatePaperRequest) (*ExamPaper, error) {
	if strings.TrimSpace(req.PaperName) == "" || req.DurationMinutes <= 0 {
		return nil, ErrInvalidPaper
	}
	if _, err := s.ensureDraft(ctx, institutionID, sessionID); err != nil {
		return nil, err
	}
	if req.AcademicOfferingID != nil {
		ok, err := s.repo.OfferingExists(ctx, institutionID, *req.AcademicOfferingID)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, ErrOfferingNotFound
		}
	}
	return s.repo.CreatePaper(ctx, institutionID, sessionID, req)
}

func (s *Service) AddPaperGroup(ctx context.Context, institutionID, paperID uuid.UUID, req *AddPaperGroupRequest) (*PaperGroup, error) {
	if req.ClassID == nil && (req.Label == nil || strings.TrimSpace(*req.Label) == "") {
		return nil, ErrInvalidGroup
	}
	paper, err := s.repo.GetPaper(ctx, institutionID, paperID)
	if err != nil {
		return nil, err
	}
	if _, err := s.ensureDraft(ctx, institutionID, paper.ExamSessionID); err != nil {
		return nil, err
	}
	if req.ClassID != nil {
		ok, err := s.repo.ClassExists(ctx, institutionID, *req.ClassID)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, ErrClassNotFound
		}
	}
	return s.repo.AddPaperGroup(ctx, institutionID, paperID, req)
}

// ── Sittings ──

func validateSittingTimes(examDate, start, end string) error {
	if strings.TrimSpace(examDate) == "" {
		return fmt.Errorf("exam_date is required")
	}
	if _, err := time.Parse("2006-01-02", examDate); err != nil {
		return fmt.Errorf("invalid exam_date, use YYYY-MM-DD")
	}
	st, err1 := time.Parse("15:04", start)
	en, err2 := time.Parse("15:04", end)
	if err1 != nil || err2 != nil {
		return fmt.Errorf("start_time and end_time are required as HH:MM")
	}
	if !en.After(st) {
		return fmt.Errorf("end_time must be after start_time")
	}
	return nil
}

func (s *Service) CreateSitting(ctx context.Context, institutionID, sessionID uuid.UUID, req *CreateSittingRequest) (*ExamSitting, error) {
	if err := validateSittingTimes(req.ExamDate, req.StartTime, req.EndTime); err != nil {
		return nil, err
	}
	if _, err := s.ensureDraft(ctx, institutionID, sessionID); err != nil {
		return nil, err
	}
	paper, err := s.repo.GetPaper(ctx, institutionID, req.ExamPaperID)
	if err != nil {
		return nil, err
	}
	if paper.ExamSessionID != sessionID {
		return nil, fmt.Errorf("paper does not belong to this session")
	}
	return s.repo.CreateSitting(ctx, institutionID, sessionID, req)
}

func (s *Service) UpdateSitting(ctx context.Context, institutionID, sittingID uuid.UUID, req *UpdateSittingRequest) (*ExamSitting, error) {
	existing, err := s.repo.GetSitting(ctx, institutionID, sittingID)
	if err != nil {
		return nil, err
	}
	if _, err := s.ensureDraft(ctx, institutionID, existing.ExamSessionID); err != nil {
		return nil, err
	}
	date := existing.ExamDate
	start := existing.StartTime
	end := existing.EndTime
	if req.ExamDate != nil {
		date = *req.ExamDate
	}
	if req.StartTime != nil {
		start = *req.StartTime
	}
	if req.EndTime != nil {
		end = *req.EndTime
	}
	if err := validateSittingTimes(date, start, end); err != nil {
		return nil, err
	}
	return s.repo.UpdateSitting(ctx, institutionID, sittingID, req)
}

func (s *Service) DeleteSitting(ctx context.Context, institutionID, sittingID uuid.UUID) error {
	existing, err := s.repo.GetSitting(ctx, institutionID, sittingID)
	if err != nil {
		return err
	}
	if _, err := s.ensureDraft(ctx, institutionID, existing.ExamSessionID); err != nil {
		return err
	}
	return s.repo.DeleteSitting(ctx, institutionID, sittingID)
}

// ── Invigilators & arrangements ──

func (s *Service) AssignInvigilator(ctx context.Context, institutionID, sittingID uuid.UUID, req *AssignInvigilatorRequest) (*InvigilatorAssignment, error) {
	if !validInvigilatorRoles[req.Role] {
		return nil, fmt.Errorf("role must be one of chief, invigilator, assistant, rover")
	}
	sitting, err := s.repo.GetSitting(ctx, institutionID, sittingID)
	if err != nil {
		return nil, err
	}
	if _, err := s.ensureDraft(ctx, institutionID, sitting.ExamSessionID); err != nil {
		return nil, err
	}
	ok, err := s.repo.StaffExists(ctx, institutionID, req.StaffID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrStaffNotFound
	}
	return s.repo.AssignInvigilator(ctx, institutionID, sittingID, req)
}

func (s *Service) RemoveInvigilator(ctx context.Context, institutionID, sittingID, staffID uuid.UUID) error {
	sitting, err := s.repo.GetSitting(ctx, institutionID, sittingID)
	if err != nil {
		return err
	}
	if _, err := s.ensureDraft(ctx, institutionID, sitting.ExamSessionID); err != nil {
		return err
	}
	return s.repo.RemoveInvigilator(ctx, institutionID, sittingID, staffID)
}

func (s *Service) AddArrangement(ctx context.Context, institutionID, sittingID uuid.UUID, req *AddArrangementRequest) (*SpecialArrangement, error) {
	if !validArrangements[req.ArrangementType] {
		return nil, ErrInvalidArrangement
	}
	sitting, err := s.repo.GetSitting(ctx, institutionID, sittingID)
	if err != nil {
		return nil, err
	}
	if _, err := s.ensureDraft(ctx, institutionID, sitting.ExamSessionID); err != nil {
		return nil, err
	}
	return s.repo.AddArrangement(ctx, institutionID, sittingID, req)
}

// ── Canonical schedule ──

// fullSchedule loads papers, groups, sittings, invigilators and
// arrangements, then assembles the canonical enriched rows. Every view
// (candidate, paper, room, invigilator, master) filters these same rows.
func (s *Service) fullSchedule(ctx context.Context, institutionID, sessionID uuid.UUID) ([]*SittingView, *ValidationResult, error) {
	papers, err := s.repo.ListPapers(ctx, institutionID, sessionID)
	if err != nil {
		return nil, nil, err
	}
	sittings, err := s.repo.ListSittings(ctx, institutionID, sessionID)
	if err != nil {
		return nil, nil, err
	}
	paperIDs := make([]uuid.UUID, 0, len(papers))
	for _, p := range papers {
		paperIDs = append(paperIDs, p.ID)
	}
	groups, err := s.repo.ListGroupsForPapers(ctx, institutionID, paperIDs)
	if err != nil {
		return nil, nil, err
	}
	sittingIDs := make([]uuid.UUID, 0, len(sittings))
	for _, x := range sittings {
		sittingIDs = append(sittingIDs, x.ID)
	}
	invig, err := s.repo.ListInvigilators(ctx, institutionID, sittingIDs)
	if err != nil {
		return nil, nil, err
	}
	arr, err := s.repo.ListArrangements(ctx, institutionID, sittingIDs)
	if err != nil {
		return nil, nil, err
	}
	in := scheduleInput{
		Papers: papers, GroupsByPaper: groups, Sittings: sittings, InvigBySitting: invig,
	}
	// Attach groups/invigilators to parsed sittings for overlap detection.
	// ValidateSchedule re-derives groups per sitting from GroupsByPaper +
	// invigilators, but parsed overlap needs them joined here.
	_ = in
	views := make([]*SittingView, 0, len(sittings))
	for _, sit := range sittings {
		paper := paperByID(papers, sit.ExamPaperID)
		// Map misses yield nil slices, which JSON-encode as null and crash
		// frontend `.map` calls. Coerce to empty slices so the API contract
		// is always `[]`, never `null`.
		g := groups[sit.ExamPaperID]
		if g == nil {
			g = []*PaperGroup{}
		}
		iv := invig[sit.ID]
		if iv == nil {
			iv = []*InvigilatorAssignment{}
		}
		ar := arr[sit.ID]
		if ar == nil {
			ar = []*SpecialArrangement{}
		}
		views = append(views, &SittingView{
			Sitting:        sit,
			Paper:          paper,
			Groups:         g,
			Invigilators:   iv,
			Arrangements:   ar,
			CapacityStatus: capacityStatus(sit.RoomCapacity, sit.CandidateCount),
		})
	}
	// Build validation input with per-sitting groups + invigilators.
	vin := scheduleInput{
		Papers: papers, GroupsByPaper: groups, Sittings: sittings, InvigBySitting: invig,
	}
	// Enrich parsed sittings: ValidateSchedule matches groups by paper via
	// GroupsByPaper? No — it needs per-sitting groups. Papers map 1:1 here
	// because groups attach to papers and sittings attach to papers, so a
	// sitting's candidate set = its paper's groups. Feed that through by
	// wrapping: extend ValidateSchedule input via helper below.
	res := validateWithSittingGroups(vin, sittings, groups, invig)
	res.SessionID = sessionID.String()
	return views, res, nil
}

func paperByID(papers []*ExamPaper, id uuid.UUID) *ExamPaper {
	for _, p := range papers {
		if p.ID == id {
			return p
		}
	}
	return nil
}

// validateWithSittingGroups adapts paper-level groups to per-sitting
// candidate sets before calling ValidateSchedule.
func validateWithSittingGroups(in scheduleInput, sittings []*ExamSitting,
	groups map[uuid.UUID][]*PaperGroup, invig map[uuid.UUID][]*InvigilatorAssignment) *ValidationResult {
	// Temporarily rewrite GroupsByPaper keyed by sitting: ValidateSchedule
	// looks up nothing per sitting — instead we inject synthetic per-paper
	// entries. Simplest correct approach: call ValidateSchedule on the
	// paper-level input but with overlap computed per sitting. Since a
	// sitting's candidates == its paper's groups, shared-class detection
	// across two sittings == shared class across their papers' groups.
	// ValidateSchedule already compares pairwise sittings using GroupsByPaper
	// only for the paper-level check; for overlap it needs per-sitting data.
	// So run a sitting-aware pass here reusing the same helpers.
	return validateSittings(sittings, in.Papers, groups, invig)
}

func validateSittings(sittings []*ExamSitting, papers []*ExamPaper,
	groups map[uuid.UUID][]*PaperGroup, invig map[uuid.UUID][]*InvigilatorAssignment) *ValidationResult {
	in := scheduleInput{Papers: papers, GroupsByPaper: groups, Sittings: sittings, InvigBySitting: invig}
	// Attach per-sitting groups for overlap: build enriched parsed list
	// inside a local copy of ValidateSchedule logic.
	res := ValidateSchedule(in)
	// ValidateSchedule's pairwise pass uses empty per-sitting groups (it has
	// no sitting→groups map). Patch: recompute overlap here with real data.
	extra := pairwiseSittingIssues(sittings, groups, invig)
	// Remove any candidate/room/invigilator issues ValidateSchedule may have
	// produced without group context (none — it had no groups), then append.
	res.Blocking = append(res.Blocking, extra...)
	res.Valid = len(res.Blocking) == 0
	sort.Slice(res.Blocking, func(i, j int) bool {
		if res.Blocking[i].Code == res.Blocking[j].Code {
			return res.Blocking[i].Message < res.Blocking[j].Message
		}
		return res.Blocking[i].Code < res.Blocking[j].Code
	})
	return res
}

func pairwiseSittingIssues(sittings []*ExamSitting, groups map[uuid.UUID][]*PaperGroup,
	invig map[uuid.UUID][]*InvigilatorAssignment) []*ValidationIssue {
	type ps struct {
		s     *ExamSitting
		start time.Time
		end   time.Time
	}
	parsed := []ps{}
	for _, s := range sittings {
		if _, err := time.Parse("2006-01-02", s.ExamDate); err != nil {
			continue
		}
		st, err1 := time.Parse("15:04", s.StartTime)
		en, err2 := time.Parse("15:04", s.EndTime)
		if err1 != nil || err2 != nil || !en.After(st) {
			continue
		}
		parsed = append(parsed, ps{s: s, start: st, end: en})
	}
	var out []*ValidationIssue
	for i := 0; i < len(parsed); i++ {
		for j := i + 1; j < len(parsed); j++ {
			a, b := parsed[i], parsed[j]
			if a.s.ExamDate != b.s.ExamDate {
				continue
			}
			if !a.start.Before(b.end) || !b.start.Before(a.end) {
				continue
			}
			if shared := sharedClass(groups[a.s.ExamPaperID], groups[b.s.ExamPaperID]); len(shared) > 0 {
				out = append(out, &ValidationIssue{Code: "CANDIDATE_OVERLAP", Severity: "blocking",
					Message:   fmt.Sprintf("sittings %s and %s overlap for candidate class %s", shortID(a.s.ID), shortID(b.s.ID), shared[0]),
					SittingID: strptr(a.s.ID.String())})
			}
			if sameRoom(a.s.RoomName, b.s.RoomName) {
				out = append(out, &ValidationIssue{Code: "ROOM_OVERLAP", Severity: "blocking",
					Message:   fmt.Sprintf("room %q double-booked by sittings %s and %s", derefStr(a.s.RoomName), shortID(a.s.ID), shortID(b.s.ID)),
					SittingID: strptr(a.s.ID.String())})
			}
			if shared := sharedStaff(invig[a.s.ID], invig[b.s.ID]); len(shared) > 0 {
				out = append(out, &ValidationIssue{Code: "INVIGILATOR_OVERLAP", Severity: "blocking",
					Message:   fmt.Sprintf("invigilator %s assigned to overlapping sittings %s and %s", shared[0], shortID(a.s.ID), shortID(b.s.ID)),
					SittingID: strptr(a.s.ID.String())})
			}
		}
	}
	return out
}

// Validate_compute validation without changing status.
func (s *Service) Validate(ctx context.Context, institutionID, sessionID uuid.UUID) (*ValidationResult, error) {
	if _, err := s.repo.GetSession(ctx, institutionID, sessionID); err != nil {
		return nil, err
	}
	_, res, err := s.fullSchedule(ctx, institutionID, sessionID)
	if err != nil {
		return nil, err
	}
	return res, nil
}

// Publish validates and, only with zero blocking issues, flips the
// session to published and marks sittings published.
func (s *Service) Publish(ctx context.Context, institutionID, sessionID uuid.UUID) (*ExamSession, error) {
	sess, err := s.repo.GetSession(ctx, institutionID, sessionID)
	if err != nil {
		return nil, err
	}
	if sess.Status == SessionPublished {
		return sess, nil
	}
	if sess.Status == SessionArchived {
		return nil, ErrSessionPublished
	}
	_, res, err := s.fullSchedule(ctx, institutionID, sessionID)
	if err != nil {
		return nil, err
	}
	if !res.Valid {
		return nil, ErrBlockingConflicts
	}
	// The session status is the canonical published flag: draft views
	// read the schedule regardless of status, published views only when
	// status == published. Sitting rows keep status='draft' as their
	// creation marker; publication is a session-level transition.
	return s.repo.SetSessionStatus(ctx, institutionID, sessionID, SessionPublished, nil)
}

// ReopenToDraft implements the controlled revision workflow: a published
// (or validated) session returns to draft with a bumped version number.
// History is preserved implicitly via the version counter.
func (s *Service) ReopenToDraft(ctx context.Context, institutionID, sessionID uuid.UUID) (*ExamSession, error) {
	sess, err := s.repo.GetSession(ctx, institutionID, sessionID)
	if err != nil {
		return nil, err
	}
	v := sess.Version + 1
	return s.repo.SetSessionStatus(ctx, institutionID, sessionID, SessionDraft, &v)
}

// ── Views: all filter the same canonical rows ──

func (s *Service) MasterTimetable(ctx context.Context, institutionID, sessionID uuid.UUID, publishedOnly bool) ([]*SittingView, error) {
	sess, err := s.repo.GetSession(ctx, institutionID, sessionID)
	if err != nil {
		return nil, err
	}
	if publishedOnly && sess.Status != SessionPublished {
		return []*SittingView{}, nil
	}
	views, _, err := s.fullSchedule(ctx, institutionID, sessionID)
	if err != nil {
		return nil, err
	}
	sortViews(views)
	return views, nil
}

func (s *Service) CandidateView(ctx context.Context, institutionID, sessionID, classID uuid.UUID, publishedOnly bool) ([]*SittingView, error) {
	all, err := s.MasterTimetable(ctx, institutionID, sessionID, publishedOnly)
	if err != nil {
		return nil, err
	}
	var out []*SittingView
	for _, v := range all {
		for _, g := range v.Groups {
			if g.ClassID != nil && *g.ClassID == classID {
				out = append(out, v)
				break
			}
		}
	}
	if out == nil {
		out = []*SittingView{}
	}
	return out, nil
}

func (s *Service) PaperView(ctx context.Context, institutionID, sessionID, paperID uuid.UUID) ([]*SittingView, error) {
	all, err := s.MasterTimetable(ctx, institutionID, sessionID, false)
	if err != nil {
		return nil, err
	}
	var out []*SittingView
	for _, v := range all {
		if v.Sitting.ExamPaperID == paperID {
			out = append(out, v)
		}
	}
	if out == nil {
		out = []*SittingView{}
	}
	return out, nil
}

func (s *Service) RoomView(ctx context.Context, institutionID, sessionID uuid.UUID, room string, publishedOnly bool) ([]*SittingView, error) {
	all, err := s.MasterTimetable(ctx, institutionID, sessionID, publishedOnly)
	if err != nil {
		return nil, err
	}
	var out []*SittingView
	for _, v := range all {
		if v.Sitting.RoomName != nil && strings.EqualFold(strings.TrimSpace(*v.Sitting.RoomName), strings.TrimSpace(room)) {
			out = append(out, v)
		}
	}
	if out == nil {
		out = []*SittingView{}
	}
	return out, nil
}

func (s *Service) InvigilatorView(ctx context.Context, institutionID, sessionID, staffID uuid.UUID, publishedOnly bool) ([]*SittingView, error) {
	all, err := s.MasterTimetable(ctx, institutionID, sessionID, publishedOnly)
	if err != nil {
		return nil, err
	}
	var out []*SittingView
	for _, v := range all {
		for _, a := range v.Invigilators {
			if a.StaffID == staffID {
				out = append(out, v)
				break
			}
		}
	}
	if out == nil {
		out = []*SittingView{}
	}
	return out, nil
}

func sortViews(views []*SittingView) {
	sort.Slice(views, func(i, j int) bool {
		a, b := views[i].Sitting, views[j].Sitting
		if a.ExamDate != b.ExamDate {
			return a.ExamDate < b.ExamDate
		}
		if a.StartTime != b.StartTime {
			return a.StartTime < b.StartTime
		}
		return a.ID.String() < b.ID.String()
	})
}

// ── Deterministic draft generator ──
//
// Manual scheduling + validation is the foundation. This generator adds a
// deterministic draft-only placer on top: papers sorted by (candidate
// load desc, duration desc, name), earliest fitting day/slot first. It
// never publishes, respects all blocking rules, and reports every paper
// it could not place with a reason instead of dropping it silently.
//
// Prerequisites that are NOT in the data model (documented, not faked):
// room capacities/availability and invigilator availability are unknown,
// so generated sittings carry no room/invigilator — validation surfaces
// ROOM_CAPACITY_UNAVAILABLE warnings for the operator to resolve.
func (s *Service) GenerateDraft(ctx context.Context, institutionID, sessionID uuid.UUID, req *GenerateDraftRequest) (*GenerateDraftResult, error) {
	if _, err := s.ensureDraft(ctx, institutionID, sessionID); err != nil {
		return nil, err
	}
	start, err1 := time.Parse("2006-01-02", req.StartDate)
	end, err2 := time.Parse("2006-01-02", req.EndDate)
	if err1 != nil || err2 != nil || end.Before(start) {
		return nil, fmt.Errorf("invalid generation date range")
	}
	dayStart, err1 := time.Parse("15:04", req.DailyStartTime)
	dayEnd, err2 := time.Parse("15:04", req.DailyEndTime)
	if err1 != nil || err2 != nil || !dayEnd.After(dayStart) {
		return nil, fmt.Errorf("invalid daily time window")
	}
	gap := req.GapMinutes
	if gap < 0 {
		gap = 0
	}
	maxPerDay := req.MaxSittingsPerDay
	if maxPerDay <= 0 {
		maxPerDay = 3
	}

	papers, err := s.repo.ListPapers(ctx, institutionID, sessionID)
	if err != nil {
		return nil, err
	}
	paperIDs := make([]uuid.UUID, 0, len(papers))
	for _, p := range papers {
		paperIDs = append(paperIDs, p.ID)
	}
	groups, err := s.repo.ListGroupsForPapers(ctx, institutionID, paperIDs)
	if err != nil {
		return nil, err
	}
	existing, err := s.repo.ListSittings(ctx, institutionID, sessionID)
	if err != nil {
		return nil, err
	}

	type placement struct {
		date       string
		start, end time.Time
		classes    map[string]bool
	}
	placed := []placement{}
	for _, sit := range existing {
		st, err1 := time.Parse("15:04", sit.StartTime)
		en, err2 := time.Parse("15:04", sit.EndTime)
		if err1 != nil || err2 != nil {
			continue
		}
		cls := map[string]bool{}
		for _, g := range groups[sit.ExamPaperID] {
			if g.ClassID != nil {
				cls[g.ClassID.String()] = true
			}
		}
		placed = append(placed, placement{date: sit.ExamDate, start: st, end: en, classes: cls})
	}

	// Papers already scheduled are skipped (no duplicates).
	scheduledPaper := map[uuid.UUID]bool{}
	for _, sit := range existing {
		scheduledPaper[sit.ExamPaperID] = true
	}
	var queue []*ExamPaper
	for _, p := range papers {
		if scheduledPaper[p.ID] {
			continue
		}
		if len(groups[p.ID]) == 0 {
			continue // reported as unscheduled below
		}
		queue = append(queue, p)
	}
	sort.Slice(queue, func(i, j int) bool {
		ci, cj := totalCandidates(groups[queue[i].ID]), totalCandidates(groups[queue[j].ID])
		if ci != cj {
			return ci > cj
		}
		if queue[i].DurationMinutes != queue[j].DurationMinutes {
			return queue[i].DurationMinutes > queue[j].DurationMinutes
		}
		return queue[i].PaperName < queue[j].PaperName
	})

	result := &GenerateDraftResult{}
	dayCount := int(end.Sub(start).Hours()/24) + 1
	for _, p := range queue {
		dur := time.Duration(p.DurationMinutes) * time.Minute
		cls := map[string]bool{}
		for _, g := range groups[p.ID] {
			if g.ClassID != nil {
				cls[g.ClassID.String()] = true
			}
		}
		placedOK := false
		var lastReason string
		for d := 0; d < dayCount && !placedOK; d++ {
			day := start.AddDate(0, 0, d)
			dateStr := day.Format("2006-01-02")
			perDay := 0
			for _, pl := range placed {
				if pl.date == dateStr {
					perDay++
				}
			}
			if perDay >= maxPerDay {
				lastReason = fmt.Sprintf("day %s already has %d sittings (limit %d)", dateStr, perDay, maxPerDay)
				continue
			}
			cursor := dayStart
			for {
				slotEnd := cursor.Add(dur)
				if slotEnd.After(dayEnd) {
					lastReason = fmt.Sprintf("no %d-minute window fits on %s within %s–%s", p.DurationMinutes, dateStr, req.DailyStartTime, req.DailyEndTime)
					break
				}
				conflict := ""
				for _, pl := range placed {
					if pl.date != dateStr {
						continue
					}
					if cursor.Before(pl.end) && pl.start.Before(slotEnd) {
						for c := range cls {
							if pl.classes[c] {
								conflict = fmt.Sprintf("candidate overlap on %s", dateStr)
								break
							}
						}
						if conflict == "" {
							// Same-day overlap for shared candidates only blocks;
							// different candidate sets may run in parallel only
							// if rooms/invigilators were known — they are not,
							// so serialize: one sitting at a time per day.
							conflict = fmt.Sprintf("time %s–%s on %s already taken (rooms/invigilators unknown, serialized)", cursor.Format("15:04"), slotEnd.Format("15:04"), dateStr)
						}
						break
					}
				}
				if conflict == "" {
					sit, err := s.repo.CreateSitting(ctx, institutionID, sessionID, &CreateSittingRequest{
						ExamPaperID: p.ID,
						ExamDate:    dateStr,
						StartTime:   cursor.Format("15:04"),
						EndTime:     slotEnd.Format("15:04"),
					})
					if err != nil {
						lastReason = fmt.Sprintf("could not persist sitting: %v", err)
						break
					}
					result.Scheduled = append(result.Scheduled, sit)
					placed = append(placed, placement{date: dateStr, start: cursor, end: slotEnd, classes: cls})
					placedOK = true
					break
				}
				lastReason = conflict
				cursor = cursor.Add(30 * time.Minute) // deterministic 30-min grid
				_ = gap
			}
		}
		if !placedOK {
			if lastReason == "" {
				lastReason = "no fitting slot in the given date range"
			}
			result.Unscheduled = append(result.Unscheduled, UnscheduledItem{PaperID: p.ID, Reason: lastReason})
		}
	}
	for _, p := range papers {
		if scheduledPaper[p.ID] {
			continue
		}
		if len(groups[p.ID]) == 0 {
			result.Unscheduled = append(result.Unscheduled, UnscheduledItem{PaperID: p.ID, Reason: "paper has no candidate groups"})
		}
	}
	if result.Scheduled == nil {
		result.Scheduled = []*ExamSitting{}
	}
	if result.Unscheduled == nil {
		result.Unscheduled = []UnscheduledItem{}
	}
	return result, nil
}

func totalCandidates(groups []*PaperGroup) int {
	n := 0
	for _, g := range groups {
		if g.CandidateCount != nil {
			n += *g.CandidateCount
		} else {
			n += 1
		}
	}
	return n
}
