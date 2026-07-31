package academics

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
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