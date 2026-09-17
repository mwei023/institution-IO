package exams

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

// validate.go holds the canonical exam timetable conflict rules as pure
// functions over in-memory slices, so they are testable without a
// database and shared by the service (draft validation) and the draft
// generator (placement checks).

// scheduleInput is everything validation needs for one session.
type scheduleInput struct {
	Papers       []*ExamPaper
	GroupsByPaper map[uuid.UUID][]*PaperGroup
	Sittings     []*ExamSitting
	InvigBySitting map[uuid.UUID][]*InvigilatorAssignment
}

// ValidateSchedule checks a full draft schedule and separates blocking
// errors (must fix before publish) from warnings.
//
// Rules:
//   - date/time validity: missing date, bad times, end <= start
//   - paper/candidate validity: sitting references known paper; paper has ≥1 group
//   - candidate conflict: two sittings sharing a class_id with overlapping times
//   - room conflict: same room_name (case-insensitive) with overlapping times
//   - invigilator conflict: same staff_id with overlapping times
//   - capacity: exceeded → blocking; missing data → warning (never silent)
func ValidateSchedule(in scheduleInput) *ValidationResult {
	blocking := []*ValidationIssue{}
	warnings := []*ValidationIssue{}

	paperByID := map[uuid.UUID]*ExamPaper{}
	for _, p := range in.Papers {
		paperByID[p.ID] = p
	}

	// Paper-level: every paper needs at least one candidate group.
	for _, p := range in.Papers {
		if len(in.GroupsByPaper[p.ID]) == 0 {
			blocking = append(blocking, &ValidationIssue{
				Code: "blocking", Severity: "blocking",
				Message: fmt.Sprintf("paper %q has no candidate groups", p.PaperName),
			})
		}
	}

	parsed := map[uuid.UUID]parsedSitting{}
	for _, s := range in.Sittings {
		ps, issue := parseSitting(s)
		if issue != nil {
			blocking = append(blocking, issue)
			continue
		}
		parsed[s.ID] = ps

		if _, ok := paperByID[s.ExamPaperID]; !ok {
			blocking = append(blocking, &ValidationIssue{
				Code: "INVALID_PAPER", Severity: "blocking",
				Message:   fmt.Sprintf("sitting %s references unknown paper", s.ID),
				SittingID: strptr(s.ID.String()),
			})
		}

		// Capacity tri-state.
		switch capacityStatus(s.RoomCapacity, s.CandidateCount) {
		case "exceeded":
			blocking = append(blocking, &ValidationIssue{
				Code: "ROOM_CAPACITY_EXCEEDED", Severity: "blocking",
				Message: fmt.Sprintf("sitting %s seats %d candidates in room %q (capacity %d)",
					s.ID, derefInt(s.CandidateCount), derefStr(s.RoomName), derefInt(s.RoomCapacity)),
				SittingID: strptr(s.ID.String()),
			})
		case "unavailable":
			warnings = append(warnings, &ValidationIssue{
				Code: "ROOM_CAPACITY_UNAVAILABLE", Severity: "warning",
				Message:   fmt.Sprintf("sitting %s has no room capacity data — verify seating manually", s.ID),
				SittingID: strptr(s.ID.String()),
			})
		}
	}

	// Pairwise overlap checks (only between sittings that parsed OK).
	ids := make([]uuid.UUID, 0, len(parsed))
	for id := range parsed {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i].String() < ids[j].String() })
	for i := 0; i < len(ids); i++ {
		for j := i + 1; j < len(ids); j++ {
			a := parsed[ids[i]]
			b := parsed[ids[j]]
			if !timesOverlap(a, b) {
				continue
			}
			// Candidate conflict via shared class_id.
			if shared := sharedClass(a.Groups, b.Groups); len(shared) > 0 {
				blocking = append(blocking, &ValidationIssue{
					Code: "CANDIDATE_OVERLAP", Severity: "blocking",
					Message: fmt.Sprintf("sittings %s and %s overlap for candidate class %s",
						shortID(ids[i]), shortID(ids[j]), shared[0]),
					SittingID: strptr(ids[i].String()),
				})
			}
			// Room conflict.
			if sameRoom(a.Raw.RoomName, b.Raw.RoomName) {
				blocking = append(blocking, &ValidationIssue{
					Code: "ROOM_OVERLAP", Severity: "blocking",
					Message: fmt.Sprintf("room %q double-booked by sittings %s and %s",
						derefStr(a.Raw.RoomName), shortID(ids[i]), shortID(ids[j])),
					SittingID: strptr(ids[i].String()),
				})
			}
			// Invigilator conflict.
			if shared := sharedStaff(a.Invigilators, b.Invigilators); len(shared) > 0 {
				blocking = append(blocking, &ValidationIssue{
					Code: "INVIGILATOR_OVERLAP", Severity: "blocking",
					Message: fmt.Sprintf("invigilator %s assigned to overlapping sittings %s and %s",
						shared[0], shortID(ids[i]), shortID(ids[j])),
					SittingID: strptr(ids[i].String()),
				})
			}
		}
	}

	return &ValidationResult{
		Valid:         len(blocking) == 0,
		Blocking:      blocking,
		Warnings:      warnings,
		CheckedAt:     time.Now(),
		SittingsCount: len(in.Sittings),
	}
}

type parsedSitting struct {
	Raw          *ExamSitting
	Date         string
	Start        time.Time
	End          time.Time
	Groups       []*PaperGroup
	Invigilators []*InvigilatorAssignment
}

func parseSitting(s *ExamSitting) (parsedSitting, *ValidationIssue) {
	sid := strptr(s.ID.String())
	if strings.TrimSpace(s.ExamDate) == "" {
		return parsedSitting{}, &ValidationIssue{Code: "MISSING_DATE", Severity: "blocking",
			Message: fmt.Sprintf("sitting %s has no date", s.ID), SittingID: sid}
	}
	if _, err := time.Parse("2006-01-02", s.ExamDate); err != nil {
		return parsedSitting{}, &ValidationIssue{Code: "INVALID_DATE", Severity: "blocking",
			Message: fmt.Sprintf("sitting %s has invalid date %q", s.ID, s.ExamDate), SittingID: sid}
	}
	start, err1 := time.Parse("15:04", s.StartTime)
	end, err2 := time.Parse("15:04", s.EndTime)
	if err1 != nil || err2 != nil {
		return parsedSitting{}, &ValidationIssue{Code: "INVALID_TIME", Severity: "blocking",
			Message: fmt.Sprintf("sitting %s has invalid start/end time", s.ID), SittingID: sid}
	}
	if !end.After(start) {
		return parsedSitting{}, &ValidationIssue{Code: "INVALID_TIME_RANGE", Severity: "blocking",
			Message: fmt.Sprintf("sitting %s end time must be after start time", s.ID), SittingID: sid}
	}
	return parsedSitting{Raw: s, Date: s.ExamDate, Start: start, End: end}, nil
}

func timesOverlap(a, b parsedSitting) bool {
	if a.Date != b.Date {
		return false
	}
	return a.Start.Before(b.End) && b.Start.Before(a.End)
}

func sharedClass(a, b []*PaperGroup) []string {
	set := map[string]bool{}
	for _, g := range a {
		if g.ClassID != nil {
			set[g.ClassID.String()] = true
		}
	}
	var out []string
	for _, g := range b {
		if g.ClassID != nil && set[g.ClassID.String()] {
			name := g.ClassID.String()
			if g.ClassName != nil && *g.ClassName != "" {
				name = *g.ClassName
			}
			out = append(out, name)
		}
	}
	return out
}

func sharedStaff(a, b []*InvigilatorAssignment) []string {
	set := map[uuid.UUID]bool{}
	for _, x := range a {
		set[x.StaffID] = true
	}
	var out []string
	for _, x := range b {
		if set[x.StaffID] {
			out = append(out, x.StaffID.String())
		}
	}
	return out
}

func sameRoom(a, b *string) bool {
	if a == nil || b == nil {
		return false
	}
	ra := strings.TrimSpace(strings.ToLower(*a))
	rb := strings.TrimSpace(strings.ToLower(*b))
	return ra != "" && ra == rb
}

// capacityStatus returns "verified" | "unavailable" | "exceeded".
// Missing data is never silent — it returns "unavailable" so callers
// surface a warning instead of assuming the room fits.
func capacityStatus(roomCap, candCount *int) string {
	if roomCap == nil || candCount == nil {
		return "unavailable"
	}
	if *candCount > *roomCap {
		return "exceeded"
	}
	return "verified"
}

func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func derefInt(i *int) int {
	if i == nil {
		return 0
	}
	return *i
}

func strptr(s string) *string { return &s }

func shortID(id uuid.UUID) string { return id.String()[:8] }
