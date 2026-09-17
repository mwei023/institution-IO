package academics

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"institution-os/pkg/db"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rs/zerolog/log" 
)

var (
	ErrClassNotFound   = errors.New("class not found")
	ErrSubjectNotFound = errors.New("subject not found")
	ErrSessionNotFound = errors.New("attendance session not found")
	ErrSessionLocked   = errors.New("attendance session is locked")
	ErrReportCardNotFound = errors.New("report card not found")
	ErrReportCardPublished = errors.New("report card already published — unpublish before regenerating")
	ErrGradeComponentNotFound = errors.New("grade component not found")  
	ErrTimetablePeriodNotFound = errors.New("timetable period not found")
)

// Repository handles database access for the academics domain.
type Repository struct {
	db *db.Pool
}

// NewRepository creates a new academics repository.
func NewRepository(db *db.Pool) *Repository {
	return &Repository{db: db}
}

// IsChildOfParent reports whether studentID is linked to parentUserID.
// Same SQL as student.Repository — duplicated so parent handlers in this
// domain can enforce ownership without cross-package repo coupling. P0-4.
func (r *Repository) IsChildOfParent(ctx context.Context, institutionID, parentUserID, studentID uuid.UUID) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM student_parent_links spl
			JOIN parent_guardians pg ON pg.id = spl.parent_id
			WHERE spl.institution_id = $1 AND pg.user_id = $2 AND spl.student_id = $3
		)`, institutionID, parentUserID, studentID).Scan(&exists)
	return exists, err
}

// GetStudentCurrentClass resolves a student's class (pilot P0-4: parent
// attendance must follow the child, never a caller-supplied class_id).
func (r *Repository) GetStudentCurrentClass(ctx context.Context, institutionID, studentID uuid.UUID) (uuid.UUID, error) {
	var classID uuid.UUID
	err := r.db.QueryRow(ctx, `
		SELECT current_class_id FROM students
		WHERE id = $1 AND institution_id = $2 AND deleted_at IS NULL`,
		studentID, institutionID,
	).Scan(&classID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return uuid.Nil, ErrClassNotFound
		}
		return uuid.Nil, err
	}
	return classID, nil
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

func (r *Repository) GetTimetableForClass(ctx context.Context, institutionID, classID, termID uuid.UUID) ([]*TimetablePeriod, error) {
	rows, err := r.db.Query(ctx, `
		SELECT tp.id, tp.institution_id, tp.term_id, tp.class_id,
		       tp.teaching_assignment_id,
		       ta.teacher_id,
		       u.first_name || ' ' || u.last_name,
		       COALESCE(sub.name, la.name),
		       CASE WHEN sub.id IS NOT NULL THEN 'subject'
		            WHEN la.id IS NOT NULL THEN 'learning_area' END,
		       tp.day_of_week, tp.period_number,
		       to_char(tp.start_time, 'HH24:MI'), to_char(tp.end_time, 'HH24:MI'),
		       tp.room, tp.is_published
		FROM timetable_periods tp
		LEFT JOIN teaching_assignments ta ON ta.id = tp.teaching_assignment_id
		LEFT JOIN academic_offerings ao ON ao.id = ta.academic_offering_id
		LEFT JOIN subjects sub ON sub.id = ao.subject_id
		LEFT JOIN learning_areas la ON la.id = ao.learning_area_id
		LEFT JOIN users u ON u.id = ta.teacher_id
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
			&p.TeachingAssignmentID, &p.TeacherID, &p.TeacherName,
			&p.OfferingName, &p.OfferingType,
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
 
func (r *Repository) GetTimetableForTeacher(ctx context.Context, institutionID, teacherID, termID uuid.UUID) ([]*TimetablePeriod, error) {
	rows, err := r.db.Query(ctx, `
		SELECT tp.id, tp.institution_id, tp.term_id, tp.class_id,
		       tp.teaching_assignment_id,
		       ta.teacher_id,
		       u.first_name || ' ' || u.last_name,
		       COALESCE(sub.name, la.name),
		       CASE WHEN sub.id IS NOT NULL THEN 'subject'
		            WHEN la.id IS NOT NULL THEN 'learning_area' END,
		       tp.day_of_week, tp.period_number,
		       to_char(tp.start_time, 'HH24:MI'), to_char(tp.end_time, 'HH24:MI'),
		       tp.room, tp.is_published
		FROM timetable_periods tp
		JOIN teaching_assignments ta ON ta.id = tp.teaching_assignment_id
		LEFT JOIN academic_offerings ao ON ao.id = ta.academic_offering_id
		LEFT JOIN subjects sub ON sub.id = ao.subject_id
		LEFT JOIN learning_areas la ON la.id = ao.learning_area_id
		LEFT JOIN users u ON u.id = ta.teacher_id
		WHERE tp.institution_id = $1
		  AND ta.teacher_id = $2
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
			&p.TeachingAssignmentID, &p.TeacherID, &p.TeacherName,
			&p.OfferingName, &p.OfferingType,
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

func (r *Repository) GetOrCreateSession(ctx context.Context, institutionID, classID, teacherID uuid.UUID, sessionDate time.Time, sessionType string, periodID *uuid.UUID) (*AttendanceSession, error) {
	var session AttendanceSession
	err := r.db.QueryRow(ctx, `
		SELECT s.id, s.institution_id, s.class_id, c.name, s.teacher_id, s.period_id,
		       s.session_date, s.session_type, s.status, s.submitted_at, s.created_at
		FROM attendance_sessions s
		JOIN classes c ON c.id = s.class_id
		WHERE s.institution_id = $1
		  AND s.class_id = $2
		  AND s.session_date = $3
		  AND s.session_type = $4
		  AND s.period_id IS NOT DISTINCT FROM $5`,
		institutionID, classID, sessionDate, sessionType, periodID,
	).Scan(
		&session.ID, &session.InstitutionID, &session.ClassID, &session.ClassName, &session.TeacherID, &session.PeriodID,
		&session.SessionDate, &session.SessionType, &session.Status, &session.SubmittedAt, &session.CreatedAt,
	)

	if err == nil {
		return &session, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	newID := uuid.New()
	_, err = r.db.Exec(ctx, `
		INSERT INTO attendance_sessions
			(id, institution_id, class_id, teacher_id, period_id, session_date, session_type, status, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,'open',NOW())`,
		newID, institutionID, classID, teacherID, periodID, sessionDate, sessionType,
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
		PeriodID:      periodID,
		SessionDate:   sessionDate,
		SessionType:   sessionType,
		Status:        "open",
		CreatedAt:     time.Now(),
	}, nil
}

// GetSessionStudents returns all active students in a class for attendance
// taking, along with their actual saved status/remarks if a record already
// exists for this session — otherwise defaults to "present" for a fresh sheet.
//
// FIX (session 3): was `var records []*AttendanceRecord` (nil when a class
// has zero active students), which combined with the AttendanceSession
// model's `records` json tag to make the API response omit the "records"
// key entirely for empty classes — the frontend then crashed on
// `sheet.records.length` because `sheet.records` was `undefined`, not `[]`.
// Now always returns a non-nil (possibly empty) slice, and the model's
// json tag no longer has `omitempty`, so empty classes serialize as
// `"records": []` instead of omitting the key.
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

	records := []*AttendanceRecord{}
	for rows.Next() {
		var rec AttendanceRecord
		if err := rows.Scan(&rec.StudentID, &rec.Name, &rec.ID, &rec.Status, &rec.Remarks); err != nil {
			return nil, err
		}
		records = append(records, &rec)
	}
	return records, rows.Err()
}

func (r *Repository) SaveAttendanceRecords(ctx context.Context, institutionID, sessionID uuid.UUID, entries []AttendanceEntry) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

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

	_, err = tx.Exec(ctx, `
		UPDATE attendance_sessions
		SET status = 'submitted', submitted_at = NOW()
		WHERE id = $1`, sessionID)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

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

// GetGradeComponentByID returns the class, subject, and term a grade
// component belongs to, so callers can authorize a teacher against it.
func (r *Repository) GetGradeComponentByID(ctx context.Context, institutionID, componentID uuid.UUID) (*GradeComponent, error) {
	var gc GradeComponent
	err := r.db.QueryRow(ctx, `
		SELECT id, institution_id, class_id, subject_id, term_id, name, weight, max_marks
		FROM grade_components
		WHERE institution_id = $1 AND id = $2`,
		institutionID, componentID,
	).Scan(&gc.ID, &gc.InstitutionID, &gc.ClassID, &gc.SubjectID, &gc.TermID, &gc.Name, &gc.Weight, &gc.MaxMarks)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrGradeComponentNotFound
		}
		return nil, err
	}
	return &gc, nil
}

// TeacherIsTimetabledForSubjectInClass checks whether a teacher has at
// least one timetable_periods row for this exact class+subject+term.
// 8-4-4 has no dedicated teacher-subject assignment table (known issue
// #9) — this derives authorization from what's actually scheduled,
// which is more correct than a separately maintained list that could
// drift from the real timetable.
func (r *Repository) TeacherIsTimetabledForSubjectInClass(ctx context.Context, institutionID, teacherID, classID, subjectID, termID uuid.UUID) (bool, error) {
	var count int
	err := r.db.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM teaching_assignments ta
		JOIN academic_offerings ao ON ao.id = ta.academic_offering_id
		WHERE ta.institution_id = $1
		  AND ta.teacher_id = $2
		  AND ta.class_id = $3
		  AND ao.subject_id = $4
		  AND ta.term_id = $5
		  AND ta.deleted_at IS NULL`,
		institutionID, teacherID, classID, subjectID, termID,
	).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("check teaching assignment: %w", err)
	}
	return count > 0, nil
}

func (r *Repository) GetGradebook(ctx context.Context, institutionID, componentID uuid.UUID) ([]*GradeEntry, error) {
	rows, err := r.db.Query(ctx, `
		SELECT s.id,
		       $2::uuid,
		       s.id,
		       s.first_name || ' ' || s.last_name,
		       ge.marks, 
		       COALESCE(ge.is_absent, false),
		       ge.entered_by,
		       COALESCE(ge.updated_at, NOW())
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

func (r *Repository) SaveGradeEntries(ctx context.Context, institutionID, componentID, enteredBy uuid.UUID, entries []GradeInput) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	for _, entry := range entries {
		_, err := tx.Exec(ctx, `
			INSERT INTO grade_entries
				(id, institution_id, component_id, student_id, marks, is_absent, entered_by, created_at, updated_at)
			VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, $6, NOW(), NOW())
			ON CONFLICT (institution_id, component_id, student_id)
			DO UPDATE SET marks = EXCLUDED.marks, is_absent = EXCLUDED.is_absent,
			              entered_by = EXCLUDED.entered_by, updated_at = NOW()`,
			institutionID, componentID, entry.StudentID, entry.Marks, entry.IsAbsent, enteredBy,
		)
		if err != nil {
			return fmt.Errorf("upsert grade entry for student %s: %w", entry.StudentID, err)
		}
	}
	return tx.Commit(ctx)
}

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

type PrimaryContactInfo struct {
	ParentPhone string
	ParentName  string
	StudentName string
}

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

func (r *Repository) CreateGradeComponent(ctx context.Context, institutionID uuid.UUID, req *CreateGradeComponentRequest) (*GradeComponent, error) {
	gc := &GradeComponent{
		ID:            uuid.New(),
		InstitutionID: institutionID,
		ClassID:       req.ClassID,
		SubjectID:     req.SubjectID,
		TermID:        req.TermID,
		Name:          req.Name,
		Weight:        req.Weight,
		MaxMarks:      req.MaxMarks,
	}
	_, err := r.db.Exec(ctx, `
		INSERT INTO grade_components
			(id, institution_id, class_id, subject_id, term_id, name, weight, max_marks, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,NOW())`,
		gc.ID, gc.InstitutionID, gc.ClassID, gc.SubjectID, gc.TermID,
		gc.Name, gc.Weight, gc.MaxMarks,
	)
	if err != nil {
		return nil, fmt.Errorf("create grade component: %w", err)
	}
	return gc, nil
}

type SubjectScore struct {
	SubjectID   uuid.UUID `json:"subject_id"`
	SubjectName string    `json:"subject_name"`
	Percentage  float64   `json:"percentage"`
}

func (r *Repository) GetSubjectScoresForStudent(ctx context.Context, institutionID, studentID, termID uuid.UUID) ([]*SubjectScore, error) {
	rows, err := r.db.Query(ctx, `
		SELECT
			sub.id,
			sub.name,
			ROUND(SUM((ge.marks / gc.max_marks) * gc.weight) / SUM(gc.weight) * 100, 2) AS percentage
		FROM grade_entries ge
		JOIN grade_components gc ON gc.id = ge.component_id
		JOIN subjects sub ON sub.id = gc.subject_id
		WHERE gc.institution_id = $1
		  AND gc.term_id = $2
		  AND ge.student_id = $3
		  AND ge.is_absent = false
		GROUP BY sub.id, sub.name
		ORDER BY sub.name`,
		institutionID, termID, studentID,
	)
	if err != nil {
		return nil, fmt.Errorf("get subject scores: %w", err)
	}
	defer rows.Close()

	var scores []*SubjectScore
	for rows.Next() {
		var s SubjectScore
		if err := rows.Scan(&s.SubjectID, &s.SubjectName, &s.Percentage); err != nil {
			return nil, err
		}
		scores = append(scores, &s)
	}
	return scores, rows.Err()
}

func (r *Repository) GetAggregateMarksForStudent(ctx context.Context, institutionID, studentID, termID uuid.UUID) (float64, int, error) {
	var aggregate float64
	var subjectCount int
	err := r.db.QueryRow(ctx, `
    SELECT COALESCE(ROUND(AVG(percentage), 2), 0), COUNT(*)
    FROM (
        SELECT
            ROUND(SUM((ge.marks / gc.max_marks) * gc.weight) / SUM(gc.weight) * 100, 2) AS percentage
        FROM grade_entries ge
        JOIN grade_components gc ON gc.id = ge.component_id
        WHERE gc.institution_id = $1
          AND gc.term_id = $2
          AND ge.student_id = $3
          AND ge.is_absent = false
        GROUP BY gc.subject_id
    ) per_subject`,
    institutionID, termID, studentID,
	).Scan(&aggregate, &subjectCount)
	if err != nil {
		return 0, 0, fmt.Errorf("compute aggregate marks: %w", err)
	}
	
	return aggregate, subjectCount, nil
}

type StudentRank struct {
	StudentID uuid.UUID
	Aggregate float64
	Rank      int
}

func (r *Repository) GetClassRankings(ctx context.Context, institutionID, classID, termID uuid.UUID) ([]*StudentRank, error) {
	rows, err := r.db.Query(ctx, `
		WITH subject_percentages AS (
			SELECT
				ge.student_id,
				gc.subject_id,
				SUM((ge.marks / gc.max_marks) * gc.weight) / SUM(gc.weight) * 100 AS percentage
			FROM grade_entries ge
			JOIN grade_components gc ON gc.id = ge.component_id
			JOIN students s ON s.id = ge.student_id
			WHERE gc.institution_id = $1
			  AND gc.term_id = $2
			  AND s.current_class_id = $3
			  AND s.deleted_at IS NULL
			  AND ge.is_absent = false
			GROUP BY ge.student_id, gc.subject_id
		),
		student_aggregates AS (
			SELECT student_id, ROUND(AVG(percentage), 2) AS aggregate
			FROM subject_percentages
			GROUP BY student_id
		)
		SELECT student_id, aggregate,
		       RANK() OVER (ORDER BY aggregate DESC) AS class_rank
		FROM student_aggregates
		ORDER BY aggregate DESC`,
		institutionID, termID, classID,
	)
	if err != nil {
		return nil, fmt.Errorf("compute class rankings: %w", err)
	}
	defer rows.Close()

	var rankings []*StudentRank
	for rows.Next() {
		var sr StudentRank
		if err := rows.Scan(&sr.StudentID, &sr.Aggregate, &sr.Rank); err != nil {
			return nil, err
		}
		rankings = append(rankings, &sr)
	}
	return rankings, rows.Err()
}

func (r *Repository) GetTermDates(ctx context.Context, institutionID, termID uuid.UUID) (time.Time, time.Time, error) {
	var start, end time.Time
	err := r.db.QueryRow(ctx, `
		SELECT start_date, end_date FROM terms
		WHERE institution_id = $1 AND id = $2`,
		institutionID, termID,
	).Scan(&start, &end)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("get term dates: %w", err)
	}
	return start, end, nil
}

func (r *Repository) UpsertReportCard(ctx context.Context, institutionID, classID, termID uuid.UUID, rc *ReportCard) error {
	var id uuid.UUID
	err := r.db.QueryRow(ctx, `
		INSERT INTO report_cards (
			id, institution_id, student_id, class_id, term_id,
			aggregate_marks, grade_points, class_position, stream_position,
			attendance_days_present, attendance_days_total,
			teacher_comment, principal_comment, status,
			created_at, updated_at
		) VALUES (
			gen_random_uuid(), $1, $2, $3, $4,
			$5, $6, $7, $8,
			$9, $10,
			$11, $12, $13,
			NOW(), NOW()
		)
		ON CONFLICT (institution_id, student_id, term_id)
		DO UPDATE SET
			class_id = EXCLUDED.class_id,
			aggregate_marks = EXCLUDED.aggregate_marks,
			grade_points = EXCLUDED.grade_points,
			class_position = EXCLUDED.class_position,
			stream_position = EXCLUDED.stream_position,
			attendance_days_present = EXCLUDED.attendance_days_present,
			attendance_days_total = EXCLUDED.attendance_days_total,
			status = EXCLUDED.status,
			updated_at = NOW()
		WHERE report_cards.status <> 'published'
		RETURNING id`,
		institutionID, rc.StudentID, classID, termID,
		rc.AggregateMarks, rc.GradePoints, rc.ClassPosition, rc.StreamPosition,
		rc.AttendanceDaysPresent, rc.AttendanceDaysTotal,
		rc.TeacherComment, rc.PrincipalComment, rc.Status,
	).Scan(&id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrReportCardPublished
		}
		return fmt.Errorf("upsert report card: %w", err)
	}
	rc.ID = id
	return nil
}

func (r *Repository) UnpublishReportCard(ctx context.Context, institutionID, studentID, termID uuid.UUID) (*ReportCard, error) {
	var rc ReportCard
	err := r.db.QueryRow(ctx, `
		UPDATE report_cards
		SET status = 'draft', published_at = NULL, updated_at = NOW()
		WHERE institution_id = $1 AND student_id = $2 AND term_id = $3
		  AND status = 'published'
		RETURNING id, student_id, aggregate_marks, grade_points, class_position,
		          stream_position, attendance_days_present, attendance_days_total,
		          teacher_comment, principal_comment, status, published_at`,
		institutionID, studentID, termID,
	).Scan(
		&rc.ID, &rc.StudentID, &rc.AggregateMarks, &rc.GradePoints, &rc.ClassPosition,
		&rc.StreamPosition, &rc.AttendanceDaysPresent, &rc.AttendanceDaysTotal,
		&rc.TeacherComment, &rc.PrincipalComment, &rc.Status, &rc.PublishedAt,
	)
	if err == nil {
		return &rc, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("unpublish report card: %w", err)
	}
	existing, ferr := r.getReportCardRaw(ctx, institutionID, studentID, termID)
	if ferr != nil {
		return nil, ferr
	}
	return existing, nil
}

func (r *Repository) getReportCardRaw(ctx context.Context, institutionID, studentID, termID uuid.UUID) (*ReportCard, error) {
	var rc ReportCard
	err := r.db.QueryRow(ctx, `
		SELECT id, student_id, aggregate_marks, grade_points, class_position,
		       stream_position, attendance_days_present, attendance_days_total,
		       teacher_comment, principal_comment, status, published_at
		FROM report_cards
		WHERE institution_id = $1 AND student_id = $2 AND term_id = $3`,
		institutionID, studentID, termID,
	).Scan(
		&rc.ID, &rc.StudentID, &rc.AggregateMarks, &rc.GradePoints, &rc.ClassPosition,
		&rc.StreamPosition, &rc.AttendanceDaysPresent, &rc.AttendanceDaysTotal,
		&rc.TeacherComment, &rc.PrincipalComment, &rc.Status, &rc.PublishedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrReportCardNotFound
		}
		return nil, err
	}
	return &rc, nil
}

func (r *Repository) PublishReportCard(ctx context.Context, institutionID, studentID, termID uuid.UUID) (*ReportCard, error) {
	var rc ReportCard
	err := r.db.QueryRow(ctx, `
		UPDATE report_cards
		SET status = 'published', published_at = NOW(), updated_at = NOW()
		WHERE institution_id = $1 AND student_id = $2 AND term_id = $3
		  AND status = 'draft'
		RETURNING id, student_id, aggregate_marks, grade_points, class_position,
		          stream_position, attendance_days_present, attendance_days_total,
		          teacher_comment, principal_comment, status, published_at`,
		institutionID, studentID, termID,
	).Scan(
		&rc.ID, &rc.StudentID, &rc.AggregateMarks, &rc.GradePoints, &rc.ClassPosition,
		&rc.StreamPosition, &rc.AttendanceDaysPresent, &rc.AttendanceDaysTotal,
		&rc.TeacherComment, &rc.PrincipalComment, &rc.Status, &rc.PublishedAt,
	)
	if err == nil {
		return &rc, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("publish report card: %w", err)
	}
	existing, ferr := r.getReportCardRaw(ctx, institutionID, studentID, termID)
	if ferr != nil {
		return nil, ferr
	}
	return existing, nil
}

func (s *Service) GenerateReportCard(ctx context.Context, institutionID, studentID, classID, termID uuid.UUID) (*ReportCard, error) {
	rankings, err := s.repo.GetClassRankings(ctx, institutionID, classID, termID)
	if err != nil {
		return nil, fmt.Errorf("get class rankings: %w", err)
	}

	termStart, termEnd, err := s.repo.GetTermDates(ctx, institutionID, termID)
	if err != nil {
		return nil, fmt.Errorf("get term dates: %w", err)
	}

	attendance, err := s.repo.GetAttendanceSummary(ctx, institutionID, classID, termStart, termEnd)
	if err != nil {
		return nil, fmt.Errorf("get attendance summary: %w", err)
	}

	return s.generateReportCardForStudent(ctx, institutionID, studentID, classID, termID, rankings, attendance)
}

func (s *Service) generateReportCardForStudent(
	ctx context.Context,
	institutionID, studentID, classID, termID uuid.UUID,
	rankings []*StudentRank,
	attendance []*AttendanceSummary,
) (*ReportCard, error) {
	aggregate, subjectCount, err := s.repo.GetAggregateMarksForStudent(ctx, institutionID, studentID, termID)
	if err != nil {
		return nil, fmt.Errorf("get aggregate marks: %w", err)
	}
	if subjectCount == 0 {
		return nil, fmt.Errorf("no graded subjects found for this student in this term")
	}

	var classPosition *int
	for _, r := range rankings {
		if r.StudentID == studentID {
			pos := r.Rank
			classPosition = &pos
			break
		}
	}

	var present, total int
	for _, a := range attendance {
		if a.StudentID == studentID {
			present = a.Present + a.Late
			total = a.Total
			break
		}
	}

	_, gradePoints := gradeFromPercentage(aggregate)

	rc := &ReportCard{
		StudentID:             studentID,
		AggregateMarks:        aggregate,
		GradePoints:           gradePoints,
		ClassPosition:         classPosition,
		AttendanceDaysPresent: present,
		AttendanceDaysTotal:   total,
		Status:                "draft",
	}

	if err := s.repo.UpsertReportCard(ctx, institutionID, classID, termID, rc); err != nil {
		return nil, fmt.Errorf("save report card: %w", err)
	}

	return rc, nil
}

func (s *Service) GenerateReportCardsForClass(ctx context.Context, institutionID, classID, termID uuid.UUID) (*BulkReportCardResult, error) {
	if _, err := s.repo.GetClassByID(ctx, institutionID, classID); err != nil {
		return nil, err
	}

	studentIDs, err := s.repo.GetActiveStudentIDsForClass(ctx, institutionID, classID)
	if err != nil {
		return nil, fmt.Errorf("get class roster: %w", err)
	}

	rankings, err := s.repo.GetClassRankings(ctx, institutionID, classID, termID)
	if err != nil {
		return nil, fmt.Errorf("get class rankings: %w", err)
	}

	termStart, termEnd, err := s.repo.GetTermDates(ctx, institutionID, termID)
	if err != nil {
		return nil, fmt.Errorf("get term dates: %w", err)
	}

	attendance, err := s.repo.GetAttendanceSummary(ctx, institutionID, classID, termStart, termEnd)
	if err != nil {
		return nil, fmt.Errorf("get attendance summary: %w", err)
	}

	result := &BulkReportCardResult{
		ClassID:       classID,
		TermID:        termID,
		TotalStudents: len(studentIDs),
		Succeeded:     make([]*ReportCard, 0, len(studentIDs)),
		Failed:        make([]BulkReportCardFailure, 0),
	}

	for _, studentID := range studentIDs {
		rc, err := s.generateReportCardForStudent(ctx, institutionID, studentID, classID, termID, rankings, attendance)
		if err != nil {
			log.Warn().Err(err).
				Str("student_id", studentID.String()).
				Str("class_id", classID.String()).
				Msg("skipping report card generation for student")
			result.Failed = append(result.Failed, BulkReportCardFailure{
				StudentID: studentID,
				Reason:    err.Error(),
			})
			continue
		}
		result.Succeeded = append(result.Succeeded, rc)
	}

	result.SucceededCount = len(result.Succeeded)
	result.FailedCount = len(result.Failed)

	return result, nil
}

func (s *Service) PublishReportCard(ctx context.Context, institutionID, studentID, termID uuid.UUID) (*ReportCard, error) {
	return s.repo.PublishReportCard(ctx, institutionID, studentID, termID)
}

func (r *Repository) GetPublishedReportCardForStudent(ctx context.Context, institutionID, studentID uuid.UUID) (*ReportCard, error) {
	var rc ReportCard
	err := r.db.QueryRow(ctx, `
		SELECT rc.id, rc.student_id,
		       s.first_name || ' ' || s.last_name,
		       c.name,
		       t.name,
		       rc.aggregate_marks, rc.grade_points, rc.class_position,
		       rc.stream_position, rc.attendance_days_present, rc.attendance_days_total,
		       rc.teacher_comment, rc.principal_comment, rc.status, rc.published_at
		FROM report_cards rc
		JOIN students s ON s.id = rc.student_id
		JOIN classes c ON c.id = rc.class_id
		JOIN terms t ON t.id = rc.term_id
		WHERE rc.institution_id = $1
		  AND rc.student_id = $2
		  AND rc.status = 'published'
		ORDER BY rc.published_at DESC
		LIMIT 1`,
		institutionID, studentID,
	).Scan(
		&rc.ID, &rc.StudentID,
		&rc.StudentName, &rc.ClassName, &rc.Term,
		&rc.AggregateMarks, &rc.GradePoints, &rc.ClassPosition,
		&rc.StreamPosition, &rc.AttendanceDaysPresent, &rc.AttendanceDaysTotal,
		&rc.TeacherComment, &rc.PrincipalComment, &rc.Status, &rc.PublishedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrReportCardNotFound
		}
		return nil, fmt.Errorf("get published report card: %w", err)
	}
	return &rc, nil
}

type Term struct {
	ID            uuid.UUID `json:"id"`
	InstitutionID uuid.UUID `json:"institution_id"`
	Name          string    `json:"name"`
	StartDate     time.Time `json:"start_date"`
	EndDate       time.Time `json:"end_date"`
	IsCurrent     bool      `json:"is_current"`
}

func (r *Repository) ListTerms(ctx context.Context, institutionID uuid.UUID) ([]*Term, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, institution_id, name, start_date, end_date, is_current
		FROM terms
		WHERE institution_id = $1
		ORDER BY start_date DESC`,
		institutionID,
	)
	if err != nil {
		return nil, fmt.Errorf("list terms: %w", err)
	}
	defer rows.Close()

	var terms []*Term
	for rows.Next() {
		var t Term
		if err := rows.Scan(
			&t.ID, &t.InstitutionID, &t.Name, &t.StartDate, &t.EndDate, &t.IsCurrent,
		); err != nil {
			return nil, err
		}
		terms = append(terms, &t)
	}
	return terms, rows.Err()
}

func (r *Repository) CreateClass(ctx context.Context, institutionID uuid.UUID, req *CreateClassRequest) (*Class, error) {
	cl := &Class{
		ID: uuid.New(), InstitutionID: institutionID, AcademicYearID: req.AcademicYearID,
		Name: req.Name, Stream: req.Stream, Level: req.Level,
		Capacity: req.Capacity, ClassTeacherID: req.ClassTeacherID,
	}
	err := r.db.QueryRow(ctx, `
		INSERT INTO classes (id, institution_id, academic_year_id, name, stream, level, capacity, class_teacher_id, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,NOW())
		RETURNING created_at`,
		cl.ID, cl.InstitutionID, cl.AcademicYearID, cl.Name, cl.Stream, cl.Level, cl.Capacity, cl.ClassTeacherID,
	).Scan(&cl.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("create class: %w", err)
	}
	return cl, nil
}

// AssignClassTeacher is deliberately a separate, auditable action from
// CreateClass — matches the lifecycle where a class may be created
// before a teacher is assigned, and reassignment happens as its own
// admin decision later, not as a side effect of a generic "edit class" call.
func (r *Repository) AssignClassTeacher(ctx context.Context, institutionID, classID, teacherID uuid.UUID) error {
	cmd, err := r.db.Exec(ctx, `
		UPDATE classes SET class_teacher_id = $3
		WHERE institution_id = $1 AND id = $2 AND deleted_at IS NULL`,
		institutionID, classID, teacherID,
	)
	if err != nil {
		return fmt.Errorf("assign class teacher: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return ErrClassNotFound
	}
	return nil
}

// ListTeachers is a narrow, single-purpose lookup for the class-teacher
// assignment dropdown — NOT a Staff module endpoint. It exists only to
// answer "which users have the teacher role", nothing more.
func (r *Repository) ListTeachers(ctx context.Context, institutionID uuid.UUID) ([]*TeacherOption, error) {
	rows, err := r.db.Query(ctx, `
		SELECT u.id, u.first_name || ' ' || u.last_name
		FROM users u
		JOIN roles ro ON ro.id = u.role_id
		WHERE u.institution_id = $1 AND ro.slug = 'teacher' AND u.is_active = true
		ORDER BY u.first_name, u.last_name`,
		institutionID,
	)
	if err != nil {
		return nil, fmt.Errorf("list teachers: %w", err)
	}
	defer rows.Close()

	teachers := []*TeacherOption{}
	for rows.Next() {
		var t TeacherOption
		if err := rows.Scan(&t.ID, &t.FullName); err != nil {
			return nil, err
		}
		teachers = append(teachers, &t)
	}
	return teachers, rows.Err()
}

func (r *Repository) ListAcademicYears(ctx context.Context, institutionID uuid.UUID) ([]*AcademicYear, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, institution_id, name, start_date, end_date, is_current
		FROM academic_years
		WHERE institution_id = $1
		ORDER BY start_date DESC`,
		institutionID,
	)
	if err != nil {
		return nil, fmt.Errorf("list academic years: %w", err)
	}
	defer rows.Close()

	years := []*AcademicYear{}
	for rows.Next() {
		var ay AcademicYear
		if err := rows.Scan(&ay.ID, &ay.InstitutionID, &ay.Name, &ay.StartDate, &ay.EndDate, &ay.IsCurrent); err != nil {
			return nil, err
		}
		years = append(years, &ay)
	}
	return years, rows.Err()
}
// academics/repository.go — new method
// CreateTeachingAssignment resolves (or self-heals) the subject's
// academic_offerings row, then creates or updates the teaching
// assignment. Subjects are seed-only today (no CreateSubject handler
// exists), so migration 014's one-time backfill should already cover
// every subject — but this doesn't assume that stays true forever,
// same defensive pattern used in cbc.Repository.CreateTeacherAssignment
// for the identical gap on the learning-area side.
func (r *Repository) CreateTeachingAssignment(ctx context.Context, institutionID uuid.UUID, req *CreateTeachingAssignmentRequest) (*TeachingAssignment, error) {
	var offeringID uuid.UUID
	err := r.db.QueryRow(ctx, `
		INSERT INTO academic_offerings (institution_id, subject_id)
		VALUES ($1, $2)
		ON CONFLICT (subject_id) DO UPDATE SET subject_id = EXCLUDED.subject_id
		RETURNING id`,
		institutionID, req.SubjectID,
	).Scan(&offeringID)
	if err != nil {
		return nil, fmt.Errorf("resolve academic offering for subject %s: %w", req.SubjectID, err)
	}

	ta := &TeachingAssignment{
		ID: uuid.New(), InstitutionID: institutionID, ClassID: req.ClassID,
		AcademicOfferingID: offeringID, TeacherID: req.TeacherID, TermID: req.TermID,
	}
	err = r.db.QueryRow(ctx, `
		INSERT INTO teaching_assignments
			(id, institution_id, class_id, academic_offering_id, teacher_id, term_id, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,NOW())
		ON CONFLICT (institution_id, class_id, academic_offering_id, term_id)
		DO UPDATE SET teacher_id = EXCLUDED.teacher_id
		RETURNING created_at`,
		ta.ID, ta.InstitutionID, ta.ClassID, ta.AcademicOfferingID, ta.TeacherID, ta.TermID,
	).Scan(&ta.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("create teaching assignment: %w", err)
	}
	return ta, nil
}

// academics/repository.go — additions

var (
	ErrSlotAlreadyBooked   = errors.New("this class already has a lesson scheduled in this slot")
	ErrTeacherDoubleBooked = errors.New("this teacher is already scheduled elsewhere at this day and period")
)

// ListTeachingAssignmentsForClass returns every teaching assignment for
// a class+term, resolved to a display-ready offering name and teacher
// name — powers the "which lesson is this period?" picker when
// scheduling. Curriculum-neutral: works identically whether the
// offering behind the assignment is a subject or a learning area.
func (r *Repository) ListTeachingAssignmentsForClass(ctx context.Context, institutionID, classID, termID uuid.UUID) ([]*TeachingAssignmentOption, error) {
	rows, err := r.db.Query(ctx, `
		SELECT ta.id,
		       ta.academic_offering_id,
		       COALESCE(sub.name, la.name)                          AS offering_name,
		       CASE WHEN ao.subject_id IS NOT NULL THEN 'subject' ELSE 'learning_area' END AS offering_type,
		       ta.teacher_id,
		       u.first_name || ' ' || u.last_name                   AS teacher_name
		FROM teaching_assignments ta
		JOIN academic_offerings ao ON ao.id = ta.academic_offering_id
		LEFT JOIN subjects sub      ON sub.id = ao.subject_id
		LEFT JOIN learning_areas la ON la.id = ao.learning_area_id
		JOIN users u ON u.id = ta.teacher_id
		WHERE ta.institution_id = $1 AND ta.class_id = $2 AND ta.term_id = $3
		  AND ta.deleted_at IS NULL
		ORDER BY offering_name`,
		institutionID, classID, termID,
	)
	if err != nil {
		return nil, fmt.Errorf("list teaching assignments for class: %w", err)
	}
	defer rows.Close()

	options := []*TeachingAssignmentOption{}
	for rows.Next() {
		var o TeachingAssignmentOption
		if err := rows.Scan(&o.ID, &o.AcademicOfferingID, &o.OfferingName, &o.OfferingType, &o.TeacherID, &o.TeacherName); err != nil {
			return nil, err
		}
		options = append(options, &o)
	}
	return options, rows.Err()
}

// CreateTimetablePeriod schedules a teaching assignment into a specific
// day/period slot. Two conflicts are checked inside one transaction,
// before the insert, since timetable_periods has NO unique constraint
// preventing either at the database level (confirmed against the live
// schema — see checkpoint follow-up):
//
//  1. Slot conflict: this exact class already has a different lesson
//     scheduled in this term/day/period.
//  2. Teacher conflict: the teacher behind teaching_assignment_id is
//     already scheduled somewhere else (any class) at this exact
//     term/day/period.
//
// SELECT ... FOR UPDATE serializes concurrent create calls against the
// same slot so two simultaneous requests can't both pass the check and
// both insert — the DB has no constraint to catch that race, so the
// code has to.
func (r *Repository) CreateTimetablePeriod(ctx context.Context, institutionID uuid.UUID, req *CreateTimetablePeriodRequest) (*TimetablePeriod, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin create timetable period: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Resolve the teacher behind this assignment — needed for the
	// cross-class double-booking check below.
	var teacherID uuid.UUID
	err = tx.QueryRow(ctx, `
		SELECT teacher_id FROM teaching_assignments
		WHERE id = $1 AND institution_id = $2 AND deleted_at IS NULL`,
		req.TeachingAssignmentID, institutionID,
	).Scan(&teacherID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("teaching assignment not found")
		}
		return nil, fmt.Errorf("resolve teaching assignment: %w", err)
	}

	// Lock and check: is this class's slot already taken?
	// NOTE: Postgres does not allow FOR UPDATE with aggregate functions
	// (SELECT COUNT(*) ... FOR UPDATE errors), so select the row ids and
	// count in Go instead.
	rows, err := tx.Query(ctx, `
		SELECT id FROM timetable_periods
		WHERE institution_id = $1 AND class_id = $2 AND term_id = $3
		  AND day_of_week = $4 AND period_number = $5 AND deleted_at IS NULL
		FOR UPDATE`,
		institutionID, req.ClassID, req.TermID, req.DayOfWeek, req.PeriodNumber,
	)
	if err != nil {
		return nil, fmt.Errorf("check slot conflict: %w", err)
	}
	slotCount := 0
	for rows.Next() {
		slotCount++
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("check slot conflict: %w", err)
	}
	if slotCount > 0 {
		return nil, ErrSlotAlreadyBooked
	}

	// Is this teacher already teaching a different class at this exact
	// day/period, this term?
	var teacherCount int
	err = tx.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM timetable_periods tp
		JOIN teaching_assignments ta ON ta.id = tp.teaching_assignment_id
		WHERE tp.institution_id = $1 AND tp.term_id = $2
		  AND tp.day_of_week = $3 AND tp.period_number = $4
		  AND tp.deleted_at IS NULL AND ta.deleted_at IS NULL
		  AND ta.teacher_id = $5`,
		institutionID, req.TermID, req.DayOfWeek, req.PeriodNumber, teacherID,
	).Scan(&teacherCount)
	if err != nil {
		return nil, fmt.Errorf("check teacher conflict: %w", err)
	}
	if teacherCount > 0 {
		return nil, ErrTeacherDoubleBooked
	}

	p := &TimetablePeriod{
		ID: uuid.New(), InstitutionID: institutionID, TermID: req.TermID, ClassID: req.ClassID,
		TeachingAssignmentID: &req.TeachingAssignmentID,
		DayOfWeek: req.DayOfWeek, PeriodNumber: req.PeriodNumber,
		StartTime: req.StartTime, EndTime: req.EndTime, Room: req.Room,
	}
	err = tx.QueryRow(ctx, `
		INSERT INTO timetable_periods
			(id, institution_id, term_id, class_id, teaching_assignment_id,
			 day_of_week, period_number, start_time, end_time, room, is_published, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,false,NOW(),NOW())
		RETURNING is_published`,
		p.ID, p.InstitutionID, p.TermID, p.ClassID, p.TeachingAssignmentID,
		p.DayOfWeek, p.PeriodNumber, p.StartTime, p.EndTime, p.Room,
	).Scan(&p.IsPublished)
	if err != nil {
		return nil, fmt.Errorf("create timetable period: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit create timetable period: %w", err)
	}
	return p, nil
}

// ── Timetable engine: schedule profiles, slots, requirements ────

var (
	ErrScheduleProfileNotFound    = errors.New("schedule profile not found")
	ErrScheduleSlotNotFound       = errors.New("schedule slot not found")
	ErrSlotSequenceTaken          = errors.New("this day already has a slot at that sequence")
	ErrSlotOverlaps               = errors.New("this slot overlaps an existing active slot on the same day")
	ErrLessonRequirementNotFound  = errors.New("lesson requirement not found")
)

func (r *Repository) ListScheduleProfiles(ctx context.Context, institutionID uuid.UUID) ([]*ScheduleProfile, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, institution_id, term_id, name, is_default, created_at, updated_at
		FROM schedule_profiles
		WHERE institution_id = $1 AND deleted_at IS NULL
		ORDER BY is_default DESC, name`,
		institutionID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*ScheduleProfile
	for rows.Next() {
		var p ScheduleProfile
		if err := rows.Scan(
			&p.ID, &p.InstitutionID, &p.TermID, &p.Name, &p.IsDefault,
			&p.CreatedAt, &p.UpdatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, &p)
	}
	return out, rows.Err()
}

func (r *Repository) CreateScheduleProfile(ctx context.Context, institutionID uuid.UUID, req *CreateScheduleProfileRequest) (*ScheduleProfile, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin create schedule profile: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if req.IsDefault {
		if _, err := tx.Exec(ctx, `
			UPDATE schedule_profiles SET is_default = false, updated_at = NOW()
			WHERE institution_id = $1 AND is_default AND deleted_at IS NULL`,
			institutionID,
		); err != nil {
			return nil, fmt.Errorf("clear default profile: %w", err)
		}
	}

	var p ScheduleProfile
	err = tx.QueryRow(ctx, `
		INSERT INTO schedule_profiles
			(id, institution_id, term_id, name, is_default, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,NOW(),NOW())
		RETURNING id, institution_id, term_id, name, is_default, created_at, updated_at`,
		uuid.New(), institutionID, req.TermID, req.Name, req.IsDefault,
	).Scan(&p.ID, &p.InstitutionID, &p.TermID, &p.Name, &p.IsDefault, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("create schedule profile: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit create schedule profile: %w", err)
	}
	return &p, nil
}

func (r *Repository) ListScheduleSlots(ctx context.Context, institutionID, profileID uuid.UUID) ([]*ScheduleSlot, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, institution_id, profile_id, day_of_week, sequence,
		       to_char(start_time, 'HH24:MI'), to_char(end_time, 'HH24:MI'),
		       slot_type, is_instructional, is_active, created_at, updated_at
		FROM schedule_slots
		WHERE institution_id = $1 AND profile_id = $2 AND deleted_at IS NULL
		ORDER BY day_of_week, sequence`,
		institutionID, profileID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*ScheduleSlot
	for rows.Next() {
		var s ScheduleSlot
		if err := rows.Scan(
			&s.ID, &s.InstitutionID, &s.ProfileID, &s.DayOfWeek, &s.Sequence,
			&s.StartTime, &s.EndTime, &s.SlotType, &s.IsInstructional, &s.IsActive,
			&s.CreatedAt, &s.UpdatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, &s)
	}
	return out, rows.Err()
}

// slotsOverlap reports whether [newStart,newEnd) intersects any active
// same-day slot. Times are "HH:MM".
func slotsOverlap(newStart, newEnd time.Time, rows []struct{ Start, End time.Time }) bool {
	for _, r := range rows {
		if newStart.Before(r.End) && r.Start.Before(newEnd) {
			return true
		}
	}
	return false
}

func (r *Repository) CreateScheduleSlot(ctx context.Context, institutionID, profileID uuid.UUID, req *CreateScheduleSlotRequest) (*ScheduleSlot, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin create schedule slot: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var profileIDCheck uuid.UUID
	err = tx.QueryRow(ctx, `
		SELECT id FROM schedule_profiles
		WHERE id = $1 AND institution_id = $2 AND deleted_at IS NULL`,
		profileID, institutionID,
	).Scan(&profileIDCheck)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrScheduleProfileNotFound
		}
		return nil, fmt.Errorf("resolve schedule profile: %w", err)
	}

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}
	isInstructional := true
	if req.IsInstructional != nil {
		isInstructional = *req.IsInstructional
	}

	s, err := insertScheduleSlotTx(ctx, tx, institutionID, profileID,
		req.DayOfWeek, req.Sequence, req.StartTime, req.EndTime,
		req.SlotType, isInstructional, isActive)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit create schedule slot: %w", err)
	}
	return s, nil
}

// insertScheduleSlotTx inserts one slot inside the caller's transaction,
// enforcing the per-day sequence guard and the overlap guard. The caller
// must have verified the profile. Times are "HH:MM".
func insertScheduleSlotTx(ctx context.Context, tx pgx.Tx, institutionID, profileID uuid.UUID, day, sequence int, startTime, endTime, slotType string, isInstructional, isActive bool) (*ScheduleSlot, error) {
	newStart, err := time.Parse("15:04", startTime)
	if err != nil {
		return nil, fmt.Errorf("invalid start_time: %w", err)
	}
	newEnd, err := time.Parse("15:04", endTime)
	if err != nil {
		return nil, fmt.Errorf("invalid end_time: %w", err)
	}
	if !newEnd.After(newStart) {
		return nil, fmt.Errorf("end_time must be after start_time")
	}

	var taken uuid.UUID
	err = tx.QueryRow(ctx, `
		SELECT id FROM schedule_slots
		WHERE profile_id = $1 AND day_of_week = $2 AND sequence = $3 AND deleted_at IS NULL
		FOR UPDATE`,
		profileID, day, sequence,
	).Scan(&taken)
	if err == nil {
		return nil, ErrSlotSequenceTaken
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("check slot sequence: %w", err)
	}

	if isActive {
		if err := checkSlotOverlap(ctx, tx, profileID, day, nil, newStart, newEnd); err != nil {
			return nil, err
		}
	}

	var s ScheduleSlot
	err = tx.QueryRow(ctx, `
		INSERT INTO schedule_slots
			(id, institution_id, profile_id, day_of_week, sequence,
			 start_time, end_time, slot_type, is_instructional, is_active,
			 created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,NOW(),NOW())
		RETURNING id, institution_id, profile_id, day_of_week, sequence,
			to_char(start_time, 'HH24:MI'), to_char(end_time, 'HH24:MI'),
			slot_type, is_instructional, is_active, created_at, updated_at`,
		uuid.New(), institutionID, profileID, day, sequence,
		startTime, endTime, slotType, isInstructional, isActive,
	).Scan(
		&s.ID, &s.InstitutionID, &s.ProfileID, &s.DayOfWeek, &s.Sequence,
		&s.StartTime, &s.EndTime, &s.SlotType, &s.IsInstructional, &s.IsActive,
		&s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("create schedule slot: %w", err)
	}
	return &s, nil
}

// CreateScheduleSlotsBulk creates the same time block on several days in one
// transaction. Conflicting days are skipped and reported; the rest commit.
func (r *Repository) CreateScheduleSlotsBulk(ctx context.Context, institutionID, profileID uuid.UUID, req *BulkCreateScheduleSlotsRequest) (*BulkCreateScheduleSlotsResult, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin bulk create schedule slots: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var profileIDCheck uuid.UUID
	err = tx.QueryRow(ctx, `
		SELECT id FROM schedule_profiles
		WHERE id = $1 AND institution_id = $2 AND deleted_at IS NULL`,
		profileID, institutionID,
	).Scan(&profileIDCheck)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrScheduleProfileNotFound
		}
		return nil, fmt.Errorf("resolve schedule profile: %w", err)
	}

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}
	isInstructional := true
	if req.IsInstructional != nil {
		isInstructional = *req.IsInstructional
	}

	result := &BulkCreateScheduleSlotsResult{}
	seen := map[int]bool{}
	for _, day := range req.Days {
		if seen[day] {
			continue
		}
		seen[day] = true
		s, err := insertScheduleSlotTx(ctx, tx, institutionID, profileID,
			day, req.Sequence, req.StartTime, req.EndTime,
			req.SlotType, isInstructional, isActive)
		if err != nil {
			if errors.Is(err, ErrSlotSequenceTaken) || errors.Is(err, ErrSlotOverlaps) {
				result.Failed = append(result.Failed, BulkSlotFailure{DayOfWeek: day, Sequence: req.Sequence, Error: err.Error()})
				continue
			}
			return nil, err
		}
		result.Created = append(result.Created, s)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit bulk create schedule slots: %w", err)
	}
	return result, nil
}

// CopyScheduleDay copies every live slot from one day to several others.
// With overwrite=false conflicts are skipped per slot; with overwrite=true
// each target day is cleared first.
func (r *Repository) CopyScheduleDay(ctx context.Context, institutionID, profileID uuid.UUID, req *CopyScheduleDayRequest) (*BulkCreateScheduleSlotsResult, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin copy schedule day: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var profileIDCheck uuid.UUID
	err = tx.QueryRow(ctx, `
		SELECT id FROM schedule_profiles
		WHERE id = $1 AND institution_id = $2 AND deleted_at IS NULL`,
		profileID, institutionID,
	).Scan(&profileIDCheck)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrScheduleProfileNotFound
		}
		return nil, fmt.Errorf("resolve schedule profile: %w", err)
	}

	rows, err := tx.Query(ctx, `
		SELECT sequence,
		       to_char(start_time, 'HH24:MI'), to_char(end_time, 'HH24:MI'),
		       slot_type, is_instructional, is_active
		FROM schedule_slots
		WHERE institution_id = $1 AND profile_id = $2 AND day_of_week = $3 AND deleted_at IS NULL
		ORDER BY sequence`,
		institutionID, profileID, req.FromDay,
	)
	if err != nil {
		return nil, fmt.Errorf("load source day slots: %w", err)
	}
	type sourceSlot struct {
		sequence        int
		start, end      string
		slotType        string
		isInstructional bool
		isActive        bool
	}
	var sources []sourceSlot
	for rows.Next() {
		var ss sourceSlot
		if err := rows.Scan(&ss.sequence, &ss.start, &ss.end, &ss.slotType, &ss.isInstructional, &ss.isActive); err != nil {
			rows.Close()
			return nil, fmt.Errorf("load source day slots: %w", err)
		}
		sources = append(sources, ss)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("load source day slots: %w", err)
	}
	if len(sources) == 0 {
		return nil, fmt.Errorf("no live slots found on day %d to copy", req.FromDay)
	}

	result := &BulkCreateScheduleSlotsResult{}
	seen := map[int]bool{}
	for _, day := range req.ToDays {
		if seen[day] || day == req.FromDay {
			continue
		}
		seen[day] = true
		if req.Overwrite {
			if _, err := tx.Exec(ctx, `
				UPDATE schedule_slots SET deleted_at = NOW(), updated_at = NOW()
				WHERE profile_id = $1 AND day_of_week = $2 AND deleted_at IS NULL`,
				profileID, day,
			); err != nil {
				return nil, fmt.Errorf("clear target day slots: %w", err)
			}
		}
		for _, ss := range sources {
			s, err := insertScheduleSlotTx(ctx, tx, institutionID, profileID,
				day, ss.sequence, ss.start, ss.end, ss.slotType, ss.isInstructional, ss.isActive)
			if err != nil {
				if errors.Is(err, ErrSlotSequenceTaken) || errors.Is(err, ErrSlotOverlaps) {
					result.Failed = append(result.Failed, BulkSlotFailure{DayOfWeek: day, Sequence: ss.sequence, Error: err.Error()})
					continue
				}
				return nil, err
			}
			result.Created = append(result.Created, s)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit copy schedule day: %w", err)
	}
	return result, nil
}

// PropagateScheduleSlot applies a correction made on one slot to the
// same-sequence slots on the target days. Days with no matching slot are
// reported in Skipped; conflicting days keep their old values and are
// reported in Skipped as well.
func (r *Repository) PropagateScheduleSlot(ctx context.Context, institutionID, slotID uuid.UUID, req *PropagateScheduleSlotRequest) (*PropagateScheduleSlotResult, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin propagate schedule slot: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var src ScheduleSlot
	err = tx.QueryRow(ctx, `
		SELECT id, institution_id, profile_id, day_of_week, sequence,
		       to_char(start_time, 'HH24:MI'), to_char(end_time, 'HH24:MI'),
		       slot_type, is_instructional, is_active, created_at, updated_at
		FROM schedule_slots
		WHERE id = $1 AND institution_id = $2 AND deleted_at IS NULL
		FOR UPDATE`,
		slotID, institutionID,
	).Scan(
		&src.ID, &src.InstitutionID, &src.ProfileID, &src.DayOfWeek, &src.Sequence,
		&src.StartTime, &src.EndTime, &src.SlotType, &src.IsInstructional, &src.IsActive,
		&src.CreatedAt, &src.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrScheduleSlotNotFound
		}
		return nil, fmt.Errorf("resolve schedule slot: %w", err)
	}

	applyPatch := func(s *ScheduleSlot) {
		if req.StartTime != nil {
			s.StartTime = *req.StartTime
		}
		if req.EndTime != nil {
			s.EndTime = *req.EndTime
		}
		if req.SlotType != nil {
			s.SlotType = *req.SlotType
		}
		if req.IsInstructional != nil {
			s.IsInstructional = *req.IsInstructional
		}
		if req.IsActive != nil {
			s.IsActive = *req.IsActive
		}
	}
	validateTimes := func(s *ScheduleSlot) (time.Time, time.Time, error) {
		newStart, err := time.Parse("15:04", s.StartTime)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid start_time: %w", err)
		}
		newEnd, err := time.Parse("15:04", s.EndTime)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid end_time: %w", err)
		}
		if !newEnd.After(newStart) {
			return time.Time{}, time.Time{}, fmt.Errorf("end_time must be after start_time")
		}
		return newStart, newEnd, nil
	}
	persistSlot := func(s *ScheduleSlot) error {
		return tx.QueryRow(ctx, `
			UPDATE schedule_slots
			SET start_time = $3, end_time = $4, slot_type = $5,
			    is_instructional = $6, is_active = $7, updated_at = NOW()
			WHERE id = $1 AND institution_id = $2 AND deleted_at IS NULL
			RETURNING updated_at`,
			s.ID, institutionID, s.StartTime, s.EndTime, s.SlotType,
			s.IsInstructional, s.IsActive,
		).Scan(&s.UpdatedAt)
	}

	applyPatch(&src)
	srcStart, srcEnd, err := validateTimes(&src)
	if err != nil {
		return nil, err
	}
	if src.IsActive {
		if err := checkSlotOverlap(ctx, tx, src.ProfileID, src.DayOfWeek, &src.ID, srcStart, srcEnd); err != nil {
			return nil, err
		}
	}
	if err := persistSlot(&src); err != nil {
		return nil, fmt.Errorf("update source schedule slot: %w", err)
	}

	result := &PropagateScheduleSlotResult{Updated: []*ScheduleSlot{&src}}
	seen := map[int]bool{}
	for _, day := range req.TargetDays {
		if seen[day] || day == src.DayOfWeek {
			continue
		}
		seen[day] = true
		var tgt ScheduleSlot
		err := tx.QueryRow(ctx, `
			SELECT id, institution_id, profile_id, day_of_week, sequence,
			       to_char(start_time, 'HH24:MI'), to_char(end_time, 'HH24:MI'),
			       slot_type, is_instructional, is_active, created_at, updated_at
			FROM schedule_slots
			WHERE profile_id = $1 AND day_of_week = $2 AND sequence = $3 AND deleted_at IS NULL
			FOR UPDATE`,
			src.ProfileID, day, src.Sequence,
		).Scan(
			&tgt.ID, &tgt.InstitutionID, &tgt.ProfileID, &tgt.DayOfWeek, &tgt.Sequence,
			&tgt.StartTime, &tgt.EndTime, &tgt.SlotType, &tgt.IsInstructional, &tgt.IsActive,
			&tgt.CreatedAt, &tgt.UpdatedAt,
		)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				result.Skipped = append(result.Skipped, BulkSlotFailure{DayOfWeek: day, Sequence: src.Sequence, Error: "no slot at this sequence on that day"})
				continue
			}
			return nil, fmt.Errorf("resolve target schedule slot: %w", err)
		}
		applyPatch(&tgt)
		newStart, newEnd, err := validateTimes(&tgt)
		if err != nil {
			result.Skipped = append(result.Skipped, BulkSlotFailure{DayOfWeek: day, Sequence: src.Sequence, Error: err.Error()})
			continue
		}
		if tgt.IsActive {
			if err := checkSlotOverlap(ctx, tx, tgt.ProfileID, tgt.DayOfWeek, &tgt.ID, newStart, newEnd); err != nil {
				result.Skipped = append(result.Skipped, BulkSlotFailure{DayOfWeek: day, Sequence: src.Sequence, Error: err.Error()})
				continue
			}
		}
		if err := persistSlot(&tgt); err != nil {
			return nil, fmt.Errorf("update target schedule slot: %w", err)
		}
		tgtCopy := tgt
		result.Updated = append(result.Updated, &tgtCopy)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit propagate schedule slot: %w", err)
	}
	return result, nil
}

// checkSlotOverlap errors with ErrSlotOverlaps when [newStart,newEnd)
// intersects an active same-day slot other than excludeID (nil = none).
func checkSlotOverlap(ctx context.Context, tx pgx.Tx, profileID uuid.UUID, day int, excludeID *uuid.UUID, newStart, newEnd time.Time) error {
	rows, err := tx.Query(ctx, `
		SELECT to_char(start_time, 'HH24:MI'), to_char(end_time, 'HH24:MI')
		FROM schedule_slots
		WHERE profile_id = $1 AND day_of_week = $2
		  AND is_active AND deleted_at IS NULL
		  AND ($3::uuid IS NULL OR id <> $3)`,
		profileID, day, excludeID,
	)
	if err != nil {
		return fmt.Errorf("check slot overlap: %w", err)
	}
	defer rows.Close()

	var existing []struct{ Start, End time.Time }
	for rows.Next() {
		var s, e string
		if err := rows.Scan(&s, &e); err != nil {
			return fmt.Errorf("check slot overlap: %w", err)
		}
		start, err := time.Parse("15:04", s)
		if err != nil {
			return fmt.Errorf("check slot overlap: %w", err)
		}
		end, err := time.Parse("15:04", e)
		if err != nil {
			return fmt.Errorf("check slot overlap: %w", err)
		}
		existing = append(existing, struct{ Start, End time.Time }{start, end})
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("check slot overlap: %w", err)
	}
	if slotsOverlap(newStart, newEnd, existing) {
		return ErrSlotOverlaps
	}
	return nil
}

func (r *Repository) UpdateScheduleSlot(ctx context.Context, institutionID, slotID uuid.UUID, req *UpdateScheduleSlotRequest) (*ScheduleSlot, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin update schedule slot: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var s ScheduleSlot
	err = tx.QueryRow(ctx, `
		SELECT id, institution_id, profile_id, day_of_week, sequence,
		       to_char(start_time, 'HH24:MI'), to_char(end_time, 'HH24:MI'),
		       slot_type, is_instructional, is_active, created_at, updated_at
		FROM schedule_slots
		WHERE id = $1 AND institution_id = $2 AND deleted_at IS NULL
		FOR UPDATE`,
		slotID, institutionID,
	).Scan(
		&s.ID, &s.InstitutionID, &s.ProfileID, &s.DayOfWeek, &s.Sequence,
		&s.StartTime, &s.EndTime, &s.SlotType, &s.IsInstructional, &s.IsActive,
		&s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrScheduleSlotNotFound
		}
		return nil, fmt.Errorf("resolve schedule slot: %w", err)
	}

	if req.StartTime != nil {
		s.StartTime = *req.StartTime
	}
	if req.EndTime != nil {
		s.EndTime = *req.EndTime
	}
	if req.SlotType != nil {
		s.SlotType = *req.SlotType
	}
	if req.IsInstructional != nil {
		s.IsInstructional = *req.IsInstructional
	}
	if req.IsActive != nil {
		s.IsActive = *req.IsActive
	}

	newStart, err := time.Parse("15:04", s.StartTime)
	if err != nil {
		return nil, fmt.Errorf("invalid start_time: %w", err)
	}
	newEnd, err := time.Parse("15:04", s.EndTime)
	if err != nil {
		return nil, fmt.Errorf("invalid end_time: %w", err)
	}
	if !newEnd.After(newStart) {
		return nil, fmt.Errorf("end_time must be after start_time")
	}
	if s.IsActive {
		if err := checkSlotOverlap(ctx, tx, s.ProfileID, s.DayOfWeek, &s.ID, newStart, newEnd); err != nil {
			return nil, err
		}
	}

	err = tx.QueryRow(ctx, `
		UPDATE schedule_slots
		SET start_time = $3, end_time = $4, slot_type = $5,
		    is_instructional = $6, is_active = $7, updated_at = NOW()
		WHERE id = $1 AND institution_id = $2 AND deleted_at IS NULL
		RETURNING updated_at`,
		s.ID, institutionID, s.StartTime, s.EndTime, s.SlotType,
		s.IsInstructional, s.IsActive,
	).Scan(&s.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("update schedule slot: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit update schedule slot: %w", err)
	}
	return &s, nil
}

func (r *Repository) DeleteScheduleSlot(ctx context.Context, institutionID, slotID uuid.UUID) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE schedule_slots SET deleted_at = NOW(), updated_at = NOW()
		WHERE id = $1 AND institution_id = $2 AND deleted_at IS NULL`,
		slotID, institutionID,
	)
	if err != nil {
		return fmt.Errorf("delete schedule slot: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrScheduleSlotNotFound
	}
	return nil
}

func scanLessonRequirement(rows pgx.Rows) (*LessonRequirement, error) {
	var lr LessonRequirement
	var maxPerDay pgtype.Int4
	if err := rows.Scan(
		&lr.ID, &lr.InstitutionID, &lr.ClassID, &lr.AcademicOfferingID,
		&lr.OfferingName, &lr.OfferingType,
		&lr.TermID, &lr.SessionsPerWeek,
		&lr.AllowSingle, &lr.AllowDouble, &lr.AllowTriple,
		&lr.SpreadAcrossDays, &maxPerDay,
		&lr.CreatedAt, &lr.UpdatedAt,
	); err != nil {
		return nil, err
	}
	if maxPerDay.Valid {
		v := int(maxPerDay.Int32)
		lr.MaxPerDay = &v
	}
	return &lr, nil
}

const lessonRequirementColumns = `
	lr.id, lr.institution_id, lr.class_id, lr.academic_offering_id,
	COALESCE(sub.name, la.name),
	CASE WHEN sub.id IS NOT NULL THEN 'subject'
	     WHEN la.id IS NOT NULL THEN 'learning_area' END,
	lr.term_id, lr.sessions_per_week,
	lr.allow_single, lr.allow_double, lr.allow_triple,
	lr.spread_across_days, lr.max_per_day,
	lr.created_at, lr.updated_at
	FROM lesson_requirements lr
	JOIN academic_offerings ao ON ao.id = lr.academic_offering_id
	LEFT JOIN subjects sub ON sub.id = ao.subject_id
	LEFT JOIN learning_areas la ON la.id = ao.learning_area_id`

func (r *Repository) ListLessonRequirements(ctx context.Context, institutionID, classID, termID uuid.UUID) ([]*LessonRequirement, error) {
	rows, err := r.db.Query(ctx, `
		SELECT `+lessonRequirementColumns+`
		WHERE lr.institution_id = $1 AND lr.class_id = $2 AND lr.term_id = $3
		  AND lr.deleted_at IS NULL
		ORDER BY 5`,
		institutionID, classID, termID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*LessonRequirement
	for rows.Next() {
		lr, err := scanLessonRequirement(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, lr)
	}
	return out, rows.Err()
}

func (r *Repository) UpsertLessonRequirement(ctx context.Context, institutionID, classID uuid.UUID, req *UpsertLessonRequirementRequest) (*LessonRequirement, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin upsert lesson requirement: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var id uuid.UUID
	err = tx.QueryRow(ctx, `
		SELECT id FROM lesson_requirements
		WHERE institution_id = $1 AND class_id = $2
		  AND academic_offering_id = $3 AND term_id = $4 AND deleted_at IS NULL
		FOR UPDATE`,
		institutionID, classID, req.AcademicOfferingID, req.TermID,
	).Scan(&id)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("resolve lesson requirement: %w", err)
	}

	if err == nil {
		_, err = tx.Exec(ctx, `
			UPDATE lesson_requirements
			SET sessions_per_week = $2, allow_single = $3, allow_double = $4,
			    allow_triple = $5, spread_across_days = $6, max_per_day = $7,
			    updated_at = NOW()
			WHERE id = $1`,
			id, req.SessionsPerWeek, req.AllowSingle, req.AllowDouble,
			req.AllowTriple, req.SpreadAcrossDays, req.MaxPerDay,
		)
		if err != nil {
			return nil, fmt.Errorf("update lesson requirement: %w", err)
		}
	} else {
		err = tx.QueryRow(ctx, `
			INSERT INTO lesson_requirements
				(id, institution_id, class_id, academic_offering_id, term_id,
				 sessions_per_week, allow_single, allow_double, allow_triple,
				 spread_across_days, max_per_day, created_at, updated_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,NOW(),NOW())
			RETURNING id`,
			uuid.New(), institutionID, classID, req.AcademicOfferingID, req.TermID,
			req.SessionsPerWeek, req.AllowSingle, req.AllowDouble, req.AllowTriple,
			req.SpreadAcrossDays, req.MaxPerDay,
		).Scan(&id)
		if err != nil {
			return nil, fmt.Errorf("create lesson requirement: %w", err)
		}
	}

	rows, err := tx.Query(ctx, `
		SELECT `+lessonRequirementColumns+`
		WHERE lr.id = $1`,
		id,
	)
	if err != nil {
		return nil, fmt.Errorf("read lesson requirement: %w", err)
	}
	defer rows.Close()
	if !rows.Next() {
		return nil, fmt.Errorf("read lesson requirement: %w", rows.Err())
	}
	lr, err := scanLessonRequirement(rows)
	if err != nil {
		return nil, err
	}
	rows.Close()
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit upsert lesson requirement: %w", err)
	}
	return lr, nil
}

func (r *Repository) DeleteLessonRequirement(ctx context.Context, institutionID, requirementID uuid.UUID) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE lesson_requirements SET deleted_at = NOW(), updated_at = NOW()
		WHERE id = $1 AND institution_id = $2 AND deleted_at IS NULL`,
		requirementID, institutionID,
	)
	if err != nil {
		return fmt.Errorf("delete lesson requirement: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrLessonRequirementNotFound
	}
	return nil
}