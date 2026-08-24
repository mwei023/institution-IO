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

// TeacherHasAssignmentInClass checks whether a teacher has any timetable
// period in this class at all, active or historical. This is a coarse
// guard — it confirms the teacher teaches *something* in this class, not
// that they're timetabled for the specific subject/period being taken
// right now. Tightening that further needs the period being submitted
// against to be identified explicitly in the request (it isn't yet —
// TakeAttendanceRequest has no period_id/subject_id field), which is
// part of the same subject_enrollments gap noted in TODO.md.

// ErrClassNeedsTeacherAssignment is returned when daily attendance is
// attempted for a class with no assigned class teacher, by anyone
// lacking attendance:approve. The absence of a class teacher is a setup
// gap to be resolved by an administrator — it must never be treated as
// "any teacher may submit," which would be a standing authorization
// hole for as long as the class stays unassigned.
var ErrClassNeedsTeacherAssignment = errors.New("this class has no assigned class teacher — an administrator must assign one before daily attendance can be taken")

func (r *Repository) TeacherHasAssignmentInClass(ctx context.Context, institutionID, teacherID, classID uuid.UUID) (bool, error) {
	var count int
	err := r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM timetable_periods
		WHERE institution_id = $1 AND teacher_id = $2 AND class_id = $3
		  AND deleted_at IS NULL`,
		institutionID, teacherID, classID,
	).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("check teacher assignment: %w", err)
	}
	return count > 0, nil
}

// TimetablePeriodInfo is the minimal shape SubmitAttendance needs to
// validate a period-attendance request against the actual timetable.
type TimetablePeriodInfo struct {
	ID        uuid.UUID
	ClassID   uuid.UUID
	TeacherID uuid.UUID
	SubjectID uuid.UUID
}

// GetTimetablePeriodByID fetches the class/teacher/subject a specific
// timetable period belongs to, so period-attendance can be validated
// against the exact lesson being taken — not just "teaches something
// in this class at some point," which TeacherHasAssignmentInClass checks.
func (r *Repository) GetTimetablePeriodByID(ctx context.Context, institutionID, periodID uuid.UUID) (*TimetablePeriodInfo, error) {
	var p TimetablePeriodInfo
	err := r.db.QueryRow(ctx, `
		SELECT id, class_id, teacher_id, subject_id
		FROM timetable_periods
		WHERE institution_id = $1 AND id = $2 AND deleted_at IS NULL`,
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