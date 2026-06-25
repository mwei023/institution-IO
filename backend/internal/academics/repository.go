package academics

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"institution-os/pkg/db"
	"github.com/jackc/pgx/v5"
)

var (
	ErrClassNotFound   = errors.New("class not found")
	ErrSubjectNotFound = errors.New("subject not found")
	ErrSessionNotFound = errors.New("attendance session not found")
	ErrSessionLocked   = errors.New("attendance session is locked")
)

// Repository handles database access for the academics domain.
type Repository struct {
	db *db.Pool
}

// NewRepository creates a new academics repository.
func NewRepository(db *db.Pool) *Repository {
	return &Repository{db: db}
}

// ─── Classes ──────────────────────────────────────────────────────

// ListClasses returns all classes for an institution in a given academic year.
func (r *Repository) ListClasses(ctx context.Context, institutionID uuid.UUID, f *ClassFilter) ([]*Class, error) {
	query := `
		SELECT c.id, c.institution_id, c.academic_year_id, c.name, c.stream,
		       c.level, c.class_teacher_id,
		       u.first_name || ' ' || u.last_name AS class_teacher,
		       c.capacity,
		       COUNT(s.id) AS enrollment
		FROM classes c
		LEFT JOIN users u ON u.id = c.class_teacher_id
		LEFT JOIN students s ON s.current_class_id = c.id AND s.deleted_at IS NULL AND s.status = 'active'
		WHERE c.institution_id = $1
		  AND c.deleted_at IS NULL
		  AND ($2::uuid IS NULL OR c.academic_year_id = $2)
		  AND ($3::int IS NULL OR c.level = $3)
		GROUP BY c.id, u.first_name, u.last_name
		ORDER BY c.level, c.stream`

	rows, err := r.db.Query(ctx, query, institutionID, f.AcademicYearID, f.Level)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var classes []*Class
	for rows.Next() {
		var cl Class
		if err := rows.Scan(
			&cl.ID, &cl.InstitutionID, &cl.AcademicYearID, &cl.Name, &cl.Stream,
			&cl.Level, &cl.ClassTeacherID, &cl.ClassTeacher,
			&cl.Capacity, &cl.Enrollment,
		); err != nil {
			return nil, err
		}
		classes = append(classes, &cl)
	}
	return classes, rows.Err()
}

// GetClassByID retrieves a single class.
func (r *Repository) GetClassByID(ctx context.Context, institutionID, classID uuid.UUID) (*Class, error) {
	var cl Class
	err := r.db.QueryRow(ctx, `
		SELECT c.id, c.institution_id, c.academic_year_id, c.name, c.stream,
		       c.level, c.class_teacher_id,
		       u.first_name || ' ' || u.last_name,
		       c.capacity, 0
		FROM classes c
		LEFT JOIN users u ON u.id = c.class_teacher_id
		WHERE c.institution_id = $1 AND c.id = $2 AND c.deleted_at IS NULL`,
		institutionID, classID,
	).Scan(
		&cl.ID, &cl.InstitutionID, &cl.AcademicYearID, &cl.Name, &cl.Stream,
		&cl.Level, &cl.ClassTeacherID, &cl.ClassTeacher,
		&cl.Capacity, &cl.Enrollment,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrClassNotFound
		}
		return nil, err
	}
	return &cl, nil
}

// ListSubjects returns all subjects for an institution.
func (r *Repository) ListSubjects(ctx context.Context, institutionID uuid.UUID) ([]*Subject, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, institution_id, department_id, code, name, knec_code, is_compulsory, credit_hours_pw
		FROM subjects
		WHERE institution_id = $1 AND deleted_at IS NULL
		ORDER BY name`, institutionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subjects []*Subject
	for rows.Next() {
		var s Subject
		if err := rows.Scan(
			&s.ID, &s.InstitutionID, &s.DepartmentID, &s.Code, &s.Name,
			&s.KNECCode, &s.IsCompulsory, &s.CreditHoursPW,
		); err != nil {
			return nil, err
		}
		subjects = append(subjects, &s)
	}
	return subjects, rows.Err()
}

// ─── Timetable ────────────────────────────────────────────────────

// GetTimetableForClass returns all published timetable periods for a class in a term.
func (r *Repository) GetTimetableForClass(ctx context.Context, institutionID, classID, termID uuid.UUID) ([]*TimetablePeriod, error) {
	rows, err := r.db.Query(ctx, `
		SELECT tp.id, tp.institution_id, tp.term_id, tp.class_id,
		       tp.subject_id, tp.teacher_id,
		       sub.name AS subject_name,
		       u.first_name || ' ' || u.last_name AS teacher_name,
		       tp.day_of_week, tp.period_number,
		       to_char(tp.start_time, 'HH24:MI'), to_char(tp.end_time, 'HH24:MI'),
		       tp.room, tp.is_published
		FROM timetable_periods tp
		JOIN subjects sub ON sub.id = tp.subject_id
		JOIN users u ON u.id = tp.teacher_id
		WHERE tp.institution_id = $1
		  AND tp.class_id = $2
		  AND tp.term_id = $3
		  AND tp.deleted_at IS NULL
		ORDER BY tp.day_of_week, tp.period_number`,
		institutionID, classID, termID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var periods []*TimetablePeriod
	for rows.Next() {
		var p TimetablePeriod
		if err := rows.Scan(
			&p.ID, &p.InstitutionID, &p.TermID, &p.ClassID,
			&p.SubjectID, &p.TeacherID,
			&p.SubjectName, &p.TeacherName,
			&p.DayOfWeek, &p.PeriodNumber,
			&p.StartTime, &p.EndTime,
			&p.Room, &p.IsPublished,
		); err != nil {
			return nil, err
		}
		periods = append(periods, &p)
	}
	return periods, rows.Err()
}

// GetTimetableForTeacher returns a teacher's timetable for a term.
func (r *Repository) GetTimetableForTeacher(ctx context.Context, institutionID, teacherID, termID uuid.UUID) ([]*TimetablePeriod, error) {
	rows, err := r.db.Query(ctx, `
		SELECT tp.id, tp.institution_id, tp.term_id, tp.class_id,
		       tp.subject_id, tp.teacher_id,
		       sub.name AS subject_name,
		       u.first_name || ' ' || u.last_name AS teacher_name,
		       tp.day_of_week, tp.period_number,
		       to_char(tp.start_time, 'HH24:MI'), to_char(tp.end_time, 'HH24:MI'),
		       tp.room, tp.is_published
		FROM timetable_periods tp
		JOIN subjects sub ON sub.id = tp.subject_id
		JOIN users u ON u.id = tp.teacher_id
		WHERE tp.institution_id = $1
		  AND tp.teacher_id = $2
		  AND tp.term_id = $3
		  AND tp.deleted_at IS NULL
		ORDER BY tp.day_of_week, tp.period_number`,
		institutionID, teacherID, termID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var periods []*TimetablePeriod
	for rows.Next() {
		var p TimetablePeriod
		if err := rows.Scan(
			&p.ID, &p.InstitutionID, &p.TermID, &p.ClassID,
			&p.SubjectID, &p.TeacherID,
			&p.SubjectName, &p.TeacherName,
			&p.DayOfWeek, &p.PeriodNumber,
			&p.StartTime, &p.EndTime,
			&p.Room, &p.IsPublished,
		); err != nil {
			return nil, err
		}
		periods = append(periods, &p)
	}
	return periods, rows.Err()
}

// ─── Attendance ───────────────────────────────────────────────────

// GetOrCreateSession finds or creates today's attendance session for a class.
func (r *Repository) GetOrCreateSession(ctx context.Context, institutionID, classID, teacherID uuid.UUID, sessionDate time.Time, sessionType string) (*AttendanceSession, error) {
	var session AttendanceSession
	err := r.db.QueryRow(ctx, `
		SELECT s.id, s.institution_id, s.class_id, c.name, s.teacher_id,
		       s.session_date, s.session_type, s.status, s.submitted_at, s.created_at
		FROM attendance_sessions s
		JOIN classes c ON c.id = s.class_id
		WHERE s.institution_id = $1
		  AND s.class_id = $2
		  AND s.session_date = $3
		  AND s.session_type = $4
		  AND s.period_id IS NULL`,
		institutionID, classID, sessionDate, sessionType,
	).Scan(
		&session.ID, &session.InstitutionID, &session.ClassID, &session.ClassName, &session.TeacherID,
		&session.SessionDate, &session.SessionType, &session.Status, &session.SubmittedAt, &session.CreatedAt,
	)

	if err == nil {
		return &session, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	// Create new session
	newID := uuid.New()
	_, err = r.db.Exec(ctx, `
		INSERT INTO attendance_sessions
			(id, institution_id, class_id, teacher_id, session_date, session_type, status, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,'open',NOW())`,
		newID, institutionID, classID, teacherID, sessionDate, sessionType,
	)
	if err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}

	var className string
	if err := r.db.QueryRow(ctx, `SELECT name FROM classes WHERE id = $1`, classID).Scan(&className); err != nil {
		className = ""
	}

	return &AttendanceSession{
		ID:            newID,
		InstitutionID: institutionID,
		ClassID:       classID,
		ClassName:     className,
		TeacherID:     teacherID,
		SessionDate:   sessionDate,
		SessionType:   sessionType,
		Status:        "open",
		CreatedAt:     time.Now(),
	}, nil
}

// GetSessionStudents returns all active students in a class for attendance taking.
// GetSessionStudents returns all active students in a class for attendance
// taking, along with their actual saved status/remarks if a record already
// exists for this session — otherwise defaults to "present" for a fresh sheet.
func (r *Repository) GetSessionStudents(ctx context.Context, institutionID, classID, sessionID uuid.UUID) ([]*AttendanceRecord, error) {
	rows, err := r.db.Query(ctx, `
		SELECT s.id, s.first_name || ' ' || s.last_name,
		       COALESCE(ar.id, '00000000-0000-0000-0000-000000000000'),
		       COALESCE(ar.status, 'present'),
		       ar.remarks
		FROM students s
		LEFT JOIN attendance_records ar
		       ON ar.student_id = s.id AND ar.session_id = $3
		WHERE s.institution_id = $1
		  AND s.current_class_id = $2
		  AND s.status = 'active'
		  AND s.deleted_at IS NULL
		ORDER BY s.last_name, s.first_name`,
		institutionID, classID, sessionID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []*AttendanceRecord
	for rows.Next() {
		var rec AttendanceRecord
		if err := rows.Scan(&rec.StudentID, &rec.Name, &rec.ID, &rec.Status, &rec.Remarks); err != nil {
			return nil, err
		}
		records = append(records, &rec)
	}
	return records, rows.Err()
}

// SaveAttendanceRecords upserts attendance records for a session.
func (r *Repository) SaveAttendanceRecords(ctx context.Context, institutionID, sessionID uuid.UUID, entries []AttendanceEntry) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	for _, entry := range entries {
		var remarks interface{} = nil
		if entry.Remarks != "" {
			remarks = entry.Remarks
		}
		_, err := tx.Exec(ctx, `
			INSERT INTO attendance_records
				(id, institution_id, session_id, student_id, status, remarks, created_at, updated_at)
			VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, NOW(), NOW())
			ON CONFLICT (institution_id, session_id, student_id)
			DO UPDATE SET status = EXCLUDED.status, remarks = EXCLUDED.remarks, updated_at = NOW()`,
			institutionID, sessionID, entry.StudentID, entry.Status, remarks,
		)
		if err != nil {
			return fmt.Errorf("upsert attendance record for student %s: %w", entry.StudentID, err)
		}
	}

	// Mark session submitted
	_, err = tx.Exec(ctx, `
		UPDATE attendance_sessions
		SET status = 'submitted', submitted_at = NOW()
		WHERE id = $1`, sessionID)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// GetAttendanceSummary returns per-student attendance stats for a class/term.
func (r *Repository) GetAttendanceSummary(ctx context.Context, institutionID, classID uuid.UUID, startDate, endDate time.Time) ([]*AttendanceSummary, error) {
	rows, err := r.db.Query(ctx, `
		SELECT
			s.id,
			s.first_name || ' ' || s.last_name AS student_name,
			COUNT(CASE WHEN ar.status = 'present' THEN 1 END) AS present,
			COUNT(CASE WHEN ar.status = 'absent'  THEN 1 END) AS absent,
			COUNT(CASE WHEN ar.status = 'late'    THEN 1 END) AS late,
			COUNT(CASE WHEN ar.status = 'excused' THEN 1 END) AS excused,
			COUNT(ar.id) AS total
		FROM students s
		LEFT JOIN attendance_records ar ON ar.student_id = s.id
		LEFT JOIN attendance_sessions sess ON sess.id = ar.session_id
			AND sess.session_date BETWEEN $3 AND $4
		WHERE s.institution_id = $1
		  AND s.current_class_id = $2
		  AND s.status = 'active'
		  AND s.deleted_at IS NULL
		GROUP BY s.id, s.first_name, s.last_name
		ORDER BY s.last_name, s.first_name`,
		institutionID, classID, startDate, endDate,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var summaries []*AttendanceSummary
	for rows.Next() {
		var s AttendanceSummary
		if err := rows.Scan(
			&s.StudentID, &s.StudentName,
			&s.Present, &s.Absent, &s.Late, &s.Excused, &s.Total,
		); err != nil {
			return nil, err
		}
		if s.Total > 0 {
			s.Percentage = float64(s.Present+s.Late) / float64(s.Total) * 100
		}
		summaries = append(summaries, &s)
	}
	return summaries, rows.Err()
}

// ─── Grades ───────────────────────────────────────────────────────

// ListGradeComponents returns all grade components for a class/subject/term.
func (r *Repository) ListGradeComponents(ctx context.Context, institutionID, classID, subjectID, termID uuid.UUID) ([]*GradeComponent, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, institution_id, class_id, subject_id, term_id, name, weight, max_marks
		FROM grade_components
		WHERE institution_id = $1 AND class_id = $2 AND subject_id = $3 AND term_id = $4
		ORDER BY name`,
		institutionID, classID, subjectID, termID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var components []*GradeComponent
	for rows.Next() {
		var gc GradeComponent
		if err := rows.Scan(
			&gc.ID, &gc.InstitutionID, &gc.ClassID, &gc.SubjectID, &gc.TermID,
			&gc.Name, &gc.Weight, &gc.MaxMarks,
		); err != nil {
			return nil, err
		}
		components = append(components, &gc)
	}
	return components, rows.Err()
}

// GetGradebook returns all grade entries for a class/subject/term/component.
func (r *Repository) GetGradebook(ctx context.Context, institutionID, componentID uuid.UUID) ([]*GradeEntry, error) {
	rows, err := r.db.Query(ctx, `
		SELECT ge.id, ge.component_id, ge.student_id,
		       s.first_name || ' ' || s.last_name,
		       ge.marks, ge.is_absent, ge.entered_by, ge.updated_at
		FROM students s
		LEFT JOIN grade_entries ge ON ge.student_id = s.id AND ge.component_id = $2
		JOIN grade_components gc ON gc.id = $2
		WHERE s.institution_id = $1
		  AND s.current_class_id = gc.class_id
		  AND s.status = 'active'
		  AND s.deleted_at IS NULL
		ORDER BY s.last_name, s.first_name`,
		institutionID, componentID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []*GradeEntry
	for rows.Next() {
		var ge GradeEntry
		if err := rows.Scan(
			&ge.ID, &ge.ComponentID, &ge.StudentID,
			&ge.StudentName, &ge.Marks, &ge.IsAbsent, &ge.EnteredBy, &ge.UpdatedAt,
		); err != nil {
			return nil, err
		}
		entries = append(entries, &ge)
	}
	return entries, rows.Err()
}

// SaveGradeEntries bulk-upserts grade entries.
func (r *Repository) SaveGradeEntries(ctx context.Context, institutionID uuid.UUID, enteredBy uuid.UUID, entries []GradeInput) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	for _, entry := range entries {
		_, err := tx.Exec(ctx, `
			INSERT INTO grade_entries
				(id, institution_id, component_id, student_id, marks, is_absent, entered_by, created_at, updated_at)
			VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, $6, NOW(), NOW())
			ON CONFLICT (institution_id, component_id, student_id)
			DO UPDATE SET marks = EXCLUDED.marks, is_absent = EXCLUDED.is_absent,
			              entered_by = EXCLUDED.entered_by, updated_at = NOW()`,
			institutionID, entry.StudentID, entry.StudentID, entry.Marks, entry.IsAbsent, enteredBy,
		)
		if err != nil {
			return fmt.Errorf("upsert grade entry: %w", err)
		}
	}
	return tx.Commit(ctx)
}

// GetCurrentTerm retrieves the currently active term for an institution.
func (r *Repository) GetCurrentTerm(ctx context.Context, institutionID uuid.UUID) (*uuid.UUID, error) {
	var termID uuid.UUID
	err := r.db.QueryRow(ctx, `
		SELECT id FROM terms
		WHERE institution_id = $1 AND is_current = true
		LIMIT 1`, institutionID,
	).Scan(&termID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &termID, nil
}

// PrimaryContactInfo holds what's needed to notify a student's primary parent.
type PrimaryContactInfo struct {
	ParentPhone string
	ParentName  string
	StudentName string
}

// GetPrimaryContactForStudent returns the primary parent/guardian contact
// details for a student, for use in notification payloads.
func (r *Repository) GetPrimaryContactForStudent(ctx context.Context, institutionID, studentID uuid.UUID) (*PrimaryContactInfo, error) {
	var info PrimaryContactInfo
	err := r.db.QueryRow(ctx, `
		SELECT s.first_name || ' ' || s.last_name AS student_name,
		       pg.first_name || ' ' || pg.last_name AS parent_name,
		       pg.phone
		FROM students s
		JOIN student_parent_links spl ON spl.student_id = s.id
		JOIN parent_guardians pg ON pg.id = spl.parent_id
		WHERE s.institution_id = $1 AND s.id = $2
		  AND pg.is_primary_contact = true
		  AND pg.deleted_at IS NULL
		LIMIT 1`,
		institutionID, studentID,
	).Scan(&info.StudentName, &info.ParentName, &info.ParentPhone)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("no primary contact found for student %s", studentID)
		}
		return nil, err
	}
	return &info, nil
}

// GetActiveStudentIDsForClass returns the IDs of all active students
// currently enrolled in a class, for roster validation.
func (r *Repository) GetActiveStudentIDsForClass(ctx context.Context, institutionID, classID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id FROM students
		WHERE institution_id = $1 AND current_class_id = $2
		  AND status = 'active' AND deleted_at IS NULL`,
		institutionID, classID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}