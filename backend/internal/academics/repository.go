package academics

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"institution-os/pkg/db"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog/log" 
)

var (
	ErrClassNotFound   = errors.New("class not found")
	ErrSubjectNotFound = errors.New("subject not found")
	ErrSessionNotFound = errors.New("attendance session not found")
	ErrSessionLocked   = errors.New("attendance session is locked")
	ErrReportCardNotFound = errors.New("report card not found")
	ErrReportCardPublished = errors.New("report card already published — unpublish before regenerating")  
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
// SaveGradeEntries bulk-upserts grade entries for a single component.
func (r *Repository) SaveGradeEntries(ctx context.Context, institutionID, componentID, enteredBy uuid.UUID, entries []GradeInput) error {
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
			institutionID, componentID, entry.StudentID, entry.Marks, entry.IsAbsent, enteredBy,
		)
		if err != nil {
			return fmt.Errorf("upsert grade entry for student %s: %w", entry.StudentID, err)
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

// CreateGradeComponent inserts a new assessment component (e.g. "CAT 1", "Midterm").
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

// SubjectScore is one subject's weighted percentage for a student/term.
type SubjectScore struct {
	SubjectID   uuid.UUID `json:"subject_id"`
	SubjectName string    `json:"subject_name"`
	Percentage  float64   `json:"percentage"`
}

// GetSubjectScoresForStudent computes each subject's weighted percentage
// for a student in a term, based on real grade_entries and grade_components.
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

// GetAggregateMarksForStudent computes the term aggregate as the simple
// average of each subject's weighted percentage. Each subject contributes
// equally regardless of its internal component count, since percentages
// are already normalized to a 0-100 scale per subject.
func (r *Repository) GetAggregateMarksForStudent(ctx context.Context, institutionID, studentID, termID uuid.UUID) (float64, int, error) {
	var aggregate float64
	var subjectCount int
	// AFTER:
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

// StudentRank holds a student's aggregate and computed rank within a class or stream.
type StudentRank struct {
	StudentID uuid.UUID
	Aggregate float64
	Rank      int
}

// GetClassRankings computes aggregate marks and rank position for every
// active student in a class for a given term, ordered best-to-worst.
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

// GetTermDates returns a term's start and end dates.
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

// UpsertReportCard creates or updates a student's report card for a term.
// Uses the table's existing UNIQUE(institution_id, student_id, term_id)
// constraint to safely re-generate a report without creating duplicates.
//
// Guard: if this student's report card for this term is already
// published, the write is refused entirely — it does NOT silently
// overwrite the published numbers and does NOT downgrade status back
// to draft. A published report is something a parent may already have
// seen; changing it invisibly on a routine bulk re-run is worse than
// returning an error. Callers get ErrReportCardPublished and must call
// UnpublishReportCard first if a correction is genuinely needed.
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
			// WHERE excluded the conflicting row — it exists and is published.
			return ErrReportCardPublished
		}
		return fmt.Errorf("upsert report card: %w", err)
	}
	rc.ID = id
	return nil
}

// UnpublishReportCard reverts a published report card back to draft and
// clears published_at. This is the only sanctioned way to make a
// published report editable again — a deliberate action an admin has
// to explicitly take, rather than something that happens as a side
// effect of re-running generation. Idempotent: unpublishing an
// already-draft card just returns it unchanged.
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
	return existing, nil // already draft — no-op
}

// getReportCardRaw fetches a report card by its natural key. Used to
// disambiguate "not found" from "already in target state" in the
// idempotent publish/unpublish paths above.
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

// PublishReportCard transitions a report card from draft to published,
// stamping published_at. Idempotent: publishing an already-published
// card is a no-op that returns the existing record, so a retried or
// double-clicked request never overwrites the original published_at.
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
	return existing, nil // already published — idempotent return
}


// GenerateReportCard computes and saves a report card for a single student.
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

// generateReportCardForStudent computes one student's report card using
// class-wide data (rankings, attendance) the caller already fetched.
// GenerateReportCard fetches that data for a single call; bulk generation
// fetches it ONCE per class and calls this directly for every student,
// instead of re-running the rankings/attendance queries per student.
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

// GenerateReportCardsForClass generates (or regenerates) a report card for
// every active student in a class for the given term. Each student is
// processed independently — one student's failure (no grades entered yet,
// absent from every assessment, etc.) is recorded and skipped; it never
// aborts the rest of the batch. Class rankings and attendance summaries
// are computed once for the whole class and reused for every student.
func (s *Service) GenerateReportCardsForClass(ctx context.Context, institutionID, classID, termID uuid.UUID) (*BulkReportCardResult, error) {
	if _, err := s.repo.GetClassByID(ctx, institutionID, classID); err != nil {
		return nil, err // propagates ErrClassNotFound
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

// PublishReportCard transitions a single student's report card from
// draft to published. See Repository.PublishReportCard for idempotency
// semantics.
func (s *Service) PublishReportCard(ctx context.Context, institutionID, studentID, termID uuid.UUID) (*ReportCard, error) {
	return s.repo.PublishReportCard(ctx, institutionID, studentID, termID)
}

// GetPublishedReportCardForStudent returns the most recently published
// report card for a student, with display fields joined in.
// Returns ErrReportCardNotFound if no published card exists yet.
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