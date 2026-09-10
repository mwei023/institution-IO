package academics

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// ErrNotClassTeacher is returned when a teacher who isn't the class's
// assigned ClassTeacherID (and lacks attendance:approve) tries to submit
// daily whole-class attendance.
var ErrNotClassTeacher = errors.New("only this class's assigned class teacher can take daily attendance")

// ErrNotAssignedToClass is returned when a teacher with no timetabled
// period in this class (and lacking attendance:approve) tries to submit
// period-type attendance for it.
var ErrNotAssignedToClass = errors.New("you are not timetabled to teach this class")

// ErrClassNeedsTeacherAssignment is returned when daily attendance is
// attempted for a class with no assigned class teacher, by anyone
// lacking attendance:approve. The absence of a class teacher is a setup
// gap to be resolved by an administrator — it must never be treated as
// "any teacher may submit," which would be a standing authorization
// hole for as long as the class stays unassigned.
var ErrClassNeedsTeacherAssignment = errors.New("this class has no assigned class teacher — an administrator must assign one before daily attendance can be taken")

// TeacherHasAssignmentInClass checks whether a teacher has any teaching
// assignment in this class at all. Previously checked
// timetable_periods.teacher_id directly — that column no longer exists;
// teaching_assignments is now the actual source of truth for "who
// teaches what," independent of whether a specific timetable slot has
// been scheduled yet.
func (r *Repository) TeacherHasAssignmentInClass(ctx context.Context, institutionID, teacherID, classID uuid.UUID) (bool, error) {
	var count int
	err := r.db.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM teaching_assignments ta
		WHERE ta.institution_id = $1
		  AND ta.teacher_id = $2
		  AND ta.class_id = $3
		  AND ta.deleted_at IS NULL`,
		institutionID, teacherID, classID,
	).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("check teacher assignment: %w", err)
	}
	return count > 0, nil
}

// TimetablePeriodInfo is the minimal shape SubmitAttendance needs to
// validate a period-attendance request against the actual timetable.
// TeacherID and SubjectID are now nullable — they're resolved through
// teaching_assignment_id, which every existing period currently lacks
// following this session's migration recovery (see checkpoint). A nil
// TeacherID means "this period isn't linked to an assignment yet," not
// "no teacher" — callers must treat that as unknown, not as a pass.
type TimetablePeriodInfo struct {
	ID        uuid.UUID
	ClassID   uuid.UUID
	TeacherID *uuid.UUID
	SubjectID *uuid.UUID
}

// GetTimetablePeriodByID fetches the class/teacher/subject a specific
// timetable period belongs to, so period-attendance can be validated
// against the exact lesson being taken. SubjectID stays nil for a CBC
// (learning-area-based) period — this struct only ever represents the
// 8-4-4/subject case when populated, since nothing currently calling
// this needs to distinguish the two; if that changes, this needs an
// OfferingType field the same way the display-facing TimetablePeriod
// (in model.go) already has.
func (r *Repository) GetTimetablePeriodByID(ctx context.Context, institutionID, periodID uuid.UUID) (*TimetablePeriodInfo, error) {
	var p TimetablePeriodInfo
	err := r.db.QueryRow(ctx, `
		SELECT tp.id, tp.class_id, ta.teacher_id, ao.subject_id
		FROM timetable_periods tp
		LEFT JOIN teaching_assignments ta ON ta.id = tp.teaching_assignment_id
		LEFT JOIN academic_offerings ao ON ao.id = ta.academic_offering_id
		WHERE tp.institution_id = $1 AND tp.id = $2 AND tp.deleted_at IS NULL`,
		institutionID, periodID,
	).Scan(&p.ID, &p.ClassID, &p.TeacherID, &p.SubjectID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("timetable period not found")
		}
		return nil, fmt.Errorf("get timetable period: %w", err)
	}
	return &p, nil
}