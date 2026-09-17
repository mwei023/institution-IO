package exams

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"institution-os/pkg/db"
)

var (
	ErrSessionNotFound   = errors.New("exam session not found")
	ErrPaperNotFound     = errors.New("exam paper not found")
	ErrSittingNotFound   = errors.New("exam sitting not found")
	ErrSessionPublished  = errors.New("published schedule cannot be edited directly — reopen it to draft first")
)

// Repository is the postgres-backed Store.
type Repository struct{ db *db.Pool }

func NewRepository(pool *db.Pool) *Repository { return &Repository{db: pool} }

func (r *Repository) Now() time.Time { return time.Now() }

func (r *Repository) CreateSession(ctx context.Context, institutionID uuid.UUID, createdBy *uuid.UUID, req *CreateSessionRequest) (*ExamSession, error) {
	var s ExamSession
	err := r.db.QueryRow(ctx, `
		INSERT INTO exam_sessions
			(id, institution_id, academic_year_id, term_id, name, exam_type, start_date, end_date, status, version, created_by, created_at, updated_at)
		VALUES (gen_random_uuid(), $1,$2,$3,$4,$5,$6,$7,'draft',1,$8,NOW(),NOW())
		RETURNING id, institution_id, academic_year_id, term_id, name, exam_type,
		          to_char(start_date,'YYYY-MM-DD'), to_char(end_date,'YYYY-MM-DD'),
		          status, version, published_at, created_at, updated_at`,
		institutionID, req.AcademicYearID, req.TermID, req.Name, req.ExamType, req.StartDate, req.EndDate, createdBy,
	).Scan(&s.ID, &s.InstitutionID, &s.AcademicYearID, &s.TermID, &s.Name, &s.ExamType,
		&s.StartDate, &s.EndDate, &s.Status, &s.Version, &s.PublishedAt, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("create exam session: %w", err)
	}
	return &s, nil
}

func (r *Repository) GetSession(ctx context.Context, institutionID, sessionID uuid.UUID) (*ExamSession, error) {
	var s ExamSession
	err := r.db.QueryRow(ctx, `
		SELECT id, institution_id, academic_year_id, term_id, name, exam_type,
		       to_char(start_date,'YYYY-MM-DD'), to_char(end_date,'YYYY-MM-DD'),
		       status, version, published_at, created_at, updated_at
		FROM exam_sessions WHERE institution_id=$1 AND id=$2 AND deleted_at IS NULL`,
		institutionID, sessionID,
	).Scan(&s.ID, &s.InstitutionID, &s.AcademicYearID, &s.TermID, &s.Name, &s.ExamType,
		&s.StartDate, &s.EndDate, &s.Status, &s.Version, &s.PublishedAt, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrSessionNotFound
		}
		return nil, err
	}
	return &s, nil
}

func (r *Repository) ListSessions(ctx context.Context, institutionID uuid.UUID) ([]*ExamSession, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, institution_id, academic_year_id, term_id, name, exam_type,
		       to_char(start_date,'YYYY-MM-DD'), to_char(end_date,'YYYY-MM-DD'),
		       status, version, published_at, created_at, updated_at
		FROM exam_sessions WHERE institution_id=$1 AND deleted_at IS NULL
		ORDER BY start_date DESC`, institutionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*ExamSession{}
	for rows.Next() {
		var s ExamSession
		if err := rows.Scan(&s.ID, &s.InstitutionID, &s.AcademicYearID, &s.TermID, &s.Name, &s.ExamType,
			&s.StartDate, &s.EndDate, &s.Status, &s.Version, &s.PublishedAt, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, &s)
	}
	return out, rows.Err()
}

func (r *Repository) SetSessionStatus(ctx context.Context, institutionID, sessionID uuid.UUID, status string, version *int) (*ExamSession, error) {
	var s ExamSession
	var q string
	if version != nil {
		q = `UPDATE exam_sessions SET status=$3, version=$4, published_at=CASE WHEN $3::varchar='published' THEN NOW() ELSE published_at END,
		     updated_at=NOW() WHERE institution_id=$1 AND id=$2 AND deleted_at IS NULL
		     RETURNING id, institution_id, academic_year_id, term_id, name, exam_type,
		     to_char(start_date,'YYYY-MM-DD'), to_char(end_date,'YYYY-MM-DD'), status, version, published_at, created_at, updated_at`
		err := r.db.QueryRow(ctx, q, institutionID, sessionID, status, *version).Scan(
			&s.ID, &s.InstitutionID, &s.AcademicYearID, &s.TermID, &s.Name, &s.ExamType,
			&s.StartDate, &s.EndDate, &s.Status, &s.Version, &s.PublishedAt, &s.CreatedAt, &s.UpdatedAt)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, ErrSessionNotFound
			}
			return nil, err
		}
		return &s, nil
	}
	q = `UPDATE exam_sessions SET status=$3, published_at=CASE WHEN $3::varchar='published' THEN NOW() ELSE published_at END,
	     updated_at=NOW() WHERE institution_id=$1 AND id=$2 AND deleted_at IS NULL
	     RETURNING id, institution_id, academic_year_id, term_id, name, exam_type,
	     to_char(start_date,'YYYY-MM-DD'), to_char(end_date,'YYYY-MM-DD'), status, version, published_at, created_at, updated_at`
	err := r.db.QueryRow(ctx, q, institutionID, sessionID, status).Scan(
		&s.ID, &s.InstitutionID, &s.AcademicYearID, &s.TermID, &s.Name, &s.ExamType,
		&s.StartDate, &s.EndDate, &s.Status, &s.Version, &s.PublishedAt, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrSessionNotFound
		}
		return nil, err
	}
	return &s, nil
}

func (r *Repository) CreatePaper(ctx context.Context, institutionID, sessionID uuid.UUID, req *CreatePaperRequest) (*ExamPaper, error) {
	var p ExamPaper
	err := r.db.QueryRow(ctx, `
		INSERT INTO exam_papers
			(id, institution_id, exam_session_id, academic_offering_id, paper_name, paper_code, paper_number,
			 duration_minutes, max_marks, instructions, is_shared, created_at, updated_at)
		VALUES (gen_random_uuid(), $1,$2,$3,$4,$5,$6,$7,$8,$9,$10,NOW(),NOW())
		RETURNING id, institution_id, exam_session_id, academic_offering_id, paper_name, paper_code,
		          paper_number, duration_minutes, max_marks, instructions, is_shared`,
		institutionID, sessionID, req.AcademicOfferingID, req.PaperName, req.PaperCode, req.PaperNumber,
		req.DurationMinutes, req.MaxMarks, req.Instructions, req.IsShared,
	).Scan(&p.ID, &p.InstitutionID, &p.ExamSessionID, &p.AcademicOfferingID, &p.PaperName, &p.PaperCode,
		&p.PaperNumber, &p.DurationMinutes, &p.MaxMarks, &p.Instructions, &p.IsShared)
	if err != nil {
		return nil, fmt.Errorf("create exam paper: %w", err)
	}
	return &p, nil
}

func (r *Repository) ListPapers(ctx context.Context, institutionID, sessionID uuid.UUID) ([]*ExamPaper, error) {
	rows, err := r.db.Query(ctx, `
		SELECT p.id, p.institution_id, p.exam_session_id, p.academic_offering_id, p.paper_name, p.paper_code,
		       p.paper_number, p.duration_minutes, p.max_marks, p.instructions, p.is_shared,
		       COALESCE(sub.name, la.name),
		       CASE WHEN sub.id IS NOT NULL THEN 'subject' WHEN la.id IS NOT NULL THEN 'learning_area' ELSE NULL END
		FROM exam_papers p
		LEFT JOIN academic_offerings ao ON ao.id = p.academic_offering_id
		LEFT JOIN subjects sub ON sub.id = ao.subject_id
		LEFT JOIN learning_areas la ON la.id = ao.learning_area_id
		WHERE p.institution_id=$1 AND p.exam_session_id=$2 AND p.deleted_at IS NULL
		ORDER BY p.paper_name`, institutionID, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*ExamPaper{}
	for rows.Next() {
		var p ExamPaper
		if err := rows.Scan(&p.ID, &p.InstitutionID, &p.ExamSessionID, &p.AcademicOfferingID, &p.PaperName,
			&p.PaperCode, &p.PaperNumber, &p.DurationMinutes, &p.MaxMarks, &p.Instructions, &p.IsShared,
			&p.OfferingName, &p.OfferingType); err != nil {
			return nil, err
		}
		out = append(out, &p)
	}
	return out, rows.Err()
}

func (r *Repository) GetPaper(ctx context.Context, institutionID, paperID uuid.UUID) (*ExamPaper, error) {
	var p ExamPaper
	err := r.db.QueryRow(ctx, `
		SELECT p.id, p.institution_id, p.exam_session_id, p.academic_offering_id, p.paper_name, p.paper_code,
		       p.paper_number, p.duration_minutes, p.max_marks, p.instructions, p.is_shared
		FROM exam_papers p WHERE p.institution_id=$1 AND p.id=$2 AND p.deleted_at IS NULL`,
		institutionID, paperID,
	).Scan(&p.ID, &p.InstitutionID, &p.ExamSessionID, &p.AcademicOfferingID, &p.PaperName, &p.PaperCode,
		&p.PaperNumber, &p.DurationMinutes, &p.MaxMarks, &p.Instructions, &p.IsShared)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrPaperNotFound
		}
		return nil, err
	}
	return &p, nil
}

func (r *Repository) AddPaperGroup(ctx context.Context, institutionID, paperID uuid.UUID, req *AddPaperGroupRequest) (*PaperGroup, error) {
	var g PaperGroup
	err := r.db.QueryRow(ctx, `
		INSERT INTO exam_paper_groups (id, institution_id, exam_paper_id, class_id, label, candidate_count, created_at)
		VALUES (gen_random_uuid(), $1,$2,$3,$4,$5,NOW())
		ON CONFLICT (exam_paper_id, class_id, label) DO UPDATE SET candidate_count=EXCLUDED.candidate_count
		RETURNING id, institution_id, exam_paper_id, class_id, label, candidate_count`,
		institutionID, paperID, req.ClassID, req.Label, req.CandidateCount,
	).Scan(&g.ID, &g.InstitutionID, &g.ExamPaperID, &g.ClassID, &g.Label, &g.CandidateCount)
	if err != nil {
		return nil, fmt.Errorf("add paper group: %w", err)
	}
	return &g, nil
}

func (r *Repository) ListGroupsForPapers(ctx context.Context, institutionID uuid.UUID, paperIDs []uuid.UUID) (map[uuid.UUID][]*PaperGroup, error) {
	out := map[uuid.UUID][]*PaperGroup{}
	if len(paperIDs) == 0 {
		return out, nil
	}
	rows, err := r.db.Query(ctx, `
		SELECT g.id, g.institution_id, g.exam_paper_id, g.class_id, c.name, g.label, g.candidate_count
		FROM exam_paper_groups g LEFT JOIN classes c ON c.id = g.class_id
		WHERE g.institution_id=$1 AND g.exam_paper_id = ANY($2)`, institutionID, paperIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var g PaperGroup
		if err := rows.Scan(&g.ID, &g.InstitutionID, &g.ExamPaperID, &g.ClassID, &g.ClassName, &g.Label, &g.CandidateCount); err != nil {
			return nil, err
		}
		out[g.ExamPaperID] = append(out[g.ExamPaperID], &g)
	}
	return out, rows.Err()
}

func (r *Repository) CreateSitting(ctx context.Context, institutionID, sessionID uuid.UUID, req *CreateSittingRequest) (*ExamSitting, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	// Serialize concurrent creates for this session so overlapping
	// check-then-insert pairs cannot both succeed.
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext($1::text))`, sessionID.String()); err != nil {
		return nil, err
	}
	var s ExamSitting
	err = tx.QueryRow(ctx, `
		INSERT INTO exam_sittings
			(id, institution_id, exam_session_id, exam_paper_id, exam_date, start_time, end_time,
			 room_name, room_capacity, candidate_count, seating_note, status, created_at, updated_at)
		VALUES (gen_random_uuid(), $1,$2,$3,$4,$5,$6,$7,$8,$9,$10,'draft',NOW(),NOW())
		RETURNING id, institution_id, exam_session_id, exam_paper_id,
		          to_char(exam_date,'YYYY-MM-DD'), to_char(start_time,'HH24:MI'), to_char(end_time,'HH24:MI'),
		          room_name, room_capacity, candidate_count, seating_note, status`,
		institutionID, sessionID, req.ExamPaperID, req.ExamDate, req.StartTime, req.EndTime,
		req.RoomName, req.RoomCapacity, req.CandidateCount, req.SeatingNote,
	).Scan(&s.ID, &s.InstitutionID, &s.ExamSessionID, &s.ExamPaperID, &s.ExamDate, &s.StartTime,
		&s.EndTime, &s.RoomName, &s.RoomCapacity, &s.CandidateCount, &s.SeatingNote, &s.Status)
	if err != nil {
		return nil, fmt.Errorf("create sitting: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *Repository) UpdateSitting(ctx context.Context, institutionID, sittingID uuid.UUID, req *UpdateSittingRequest) (*ExamSitting, error) {
	var s ExamSitting
	err := r.db.QueryRow(ctx, `
		UPDATE exam_sittings SET
			exam_date = COALESCE($3, exam_date),
			start_time = COALESCE($4, start_time),
			end_time = COALESCE($5, end_time),
			room_name = COALESCE($6, room_name),
			room_capacity = COALESCE($7, room_capacity),
			candidate_count = COALESCE($8, candidate_count),
			seating_note = COALESCE($9, seating_note),
			updated_at = NOW()
		WHERE institution_id=$1 AND id=$2 AND deleted_at IS NULL
		RETURNING id, institution_id, exam_session_id, exam_paper_id,
		          to_char(exam_date,'YYYY-MM-DD'), to_char(start_time,'HH24:MI'), to_char(end_time,'HH24:MI'),
		          room_name, room_capacity, candidate_count, seating_note, status`,
		institutionID, sittingID, nullDate(req.ExamDate), nullTime(req.StartTime), nullTime(req.EndTime),
		req.RoomName, req.RoomCapacity, req.CandidateCount, req.SeatingNote,
	).Scan(&s.ID, &s.InstitutionID, &s.ExamSessionID, &s.ExamPaperID, &s.ExamDate, &s.StartTime,
		&s.EndTime, &s.RoomName, &s.RoomCapacity, &s.CandidateCount, &s.SeatingNote, &s.Status)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrSittingNotFound
		}
		return nil, err
	}
	return &s, nil
}

func (r *Repository) DeleteSitting(ctx context.Context, institutionID, sittingID uuid.UUID) error {
	cmd, err := r.db.Exec(ctx, `
		UPDATE exam_sittings SET deleted_at=NOW() WHERE institution_id=$1 AND id=$2 AND deleted_at IS NULL`,
		institutionID, sittingID)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return ErrSittingNotFound
	}
	return nil
}

func (r *Repository) ListSittings(ctx context.Context, institutionID, sessionID uuid.UUID) ([]*ExamSitting, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, institution_id, exam_session_id, exam_paper_id,
		       to_char(exam_date,'YYYY-MM-DD'), to_char(start_time,'HH24:MI'), to_char(end_time,'HH24:MI'),
		       room_name, room_capacity, candidate_count, seating_note, status
		FROM exam_sittings
		WHERE institution_id=$1 AND exam_session_id=$2 AND deleted_at IS NULL
		ORDER BY exam_date, start_time`, institutionID, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*ExamSitting{}
	for rows.Next() {
		var s ExamSitting
		if err := rows.Scan(&s.ID, &s.InstitutionID, &s.ExamSessionID, &s.ExamPaperID, &s.ExamDate,
			&s.StartTime, &s.EndTime, &s.RoomName, &s.RoomCapacity, &s.CandidateCount, &s.SeatingNote, &s.Status); err != nil {
			return nil, err
		}
		out = append(out, &s)
	}
	return out, rows.Err()
}

func (r *Repository) GetSitting(ctx context.Context, institutionID, sittingID uuid.UUID) (*ExamSitting, error) {
	var s ExamSitting
	err := r.db.QueryRow(ctx, `
		SELECT id, institution_id, exam_session_id, exam_paper_id,
		       to_char(exam_date,'YYYY-MM-DD'), to_char(start_time,'HH24:MI'), to_char(end_time,'HH24:MI'),
		       room_name, room_capacity, candidate_count, seating_note, status
		FROM exam_sittings WHERE institution_id=$1 AND id=$2 AND deleted_at IS NULL`,
		institutionID, sittingID,
	).Scan(&s.ID, &s.InstitutionID, &s.ExamSessionID, &s.ExamPaperID, &s.ExamDate,
		&s.StartTime, &s.EndTime, &s.RoomName, &s.RoomCapacity, &s.CandidateCount, &s.SeatingNote, &s.Status)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrSittingNotFound
		}
		return nil, err
	}
	return &s, nil
}

func (r *Repository) AssignInvigilator(ctx context.Context, institutionID, sittingID uuid.UUID, req *AssignInvigilatorRequest) (*InvigilatorAssignment, error) {
	var a InvigilatorAssignment
	err := r.db.QueryRow(ctx, `
		INSERT INTO exam_invigilators (id, institution_id, exam_sitting_id, staff_id, role, notes, created_at)
		VALUES (gen_random_uuid(), $1,$2,$3,$4,$5,NOW())
		ON CONFLICT (exam_sitting_id, staff_id) DO UPDATE SET role=EXCLUDED.role, notes=EXCLUDED.notes
		RETURNING id, institution_id, exam_sitting_id, staff_id, role, notes`,
		institutionID, sittingID, req.StaffID, req.Role, req.Notes,
	).Scan(&a.ID, &a.InstitutionID, &a.ExamSittingID, &a.StaffID, &a.Role, &a.Notes)
	if err != nil {
		return nil, fmt.Errorf("assign invigilator: %w", err)
	}
	return &a, nil
}

func (r *Repository) RemoveInvigilator(ctx context.Context, institutionID, sittingID, staffID uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM exam_invigilators WHERE institution_id=$1 AND exam_sitting_id=$2 AND staff_id=$3`,
		institutionID, sittingID, staffID)
	return err
}

func (r *Repository) ListInvigilators(ctx context.Context, institutionID uuid.UUID, sittingIDs []uuid.UUID) (map[uuid.UUID][]*InvigilatorAssignment, error) {
	out := map[uuid.UUID][]*InvigilatorAssignment{}
	if len(sittingIDs) == 0 {
		return out, nil
	}
	rows, err := r.db.Query(ctx, `
		SELECT e.id, e.institution_id, e.exam_sitting_id, e.staff_id,
		       u.first_name || ' ' || u.last_name, e.role, e.notes
		FROM exam_invigilators e LEFT JOIN users u ON u.id = e.staff_id
		WHERE e.institution_id=$1 AND e.exam_sitting_id = ANY($2)`, institutionID, sittingIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var a InvigilatorAssignment
		if err := rows.Scan(&a.ID, &a.InstitutionID, &a.ExamSittingID, &a.StaffID, &a.StaffName, &a.Role, &a.Notes); err != nil {
			return nil, err
		}
		out[a.ExamSittingID] = append(out[a.ExamSittingID], &a)
	}
	return out, rows.Err()
}

func (r *Repository) AddArrangement(ctx context.Context, institutionID, sittingID uuid.UUID, req *AddArrangementRequest) (*SpecialArrangement, error) {
	var a SpecialArrangement
	err := r.db.QueryRow(ctx, `
		INSERT INTO exam_special_arrangements
			(id, institution_id, exam_sitting_id, student_id, class_id, arrangement_type, details, extra_minutes, room_name, created_at)
		VALUES (gen_random_uuid(), $1,$2,$3,$4,$5,$6,$7,$8,NOW())
		RETURNING id, institution_id, exam_sitting_id, student_id, class_id, arrangement_type, details, extra_minutes, room_name`,
		institutionID, sittingID, req.StudentID, req.ClassID, req.ArrangementType, req.Details, req.ExtraMinutes, req.RoomName,
	).Scan(&a.ID, &a.InstitutionID, &a.ExamSittingID, &a.StudentID, &a.ClassID, &a.ArrangementType, &a.Details, &a.ExtraMinutes, &a.RoomName)
	if err != nil {
		return nil, fmt.Errorf("add arrangement: %w", err)
	}
	return &a, nil
}

func (r *Repository) ListArrangements(ctx context.Context, institutionID uuid.UUID, sittingIDs []uuid.UUID) (map[uuid.UUID][]*SpecialArrangement, error) {
	out := map[uuid.UUID][]*SpecialArrangement{}
	if len(sittingIDs) == 0 {
		return out, nil
	}
	rows, err := r.db.Query(ctx, `
		SELECT id, institution_id, exam_sitting_id, student_id, class_id, arrangement_type, details, extra_minutes, room_name
		FROM exam_special_arrangements WHERE institution_id=$1 AND exam_sitting_id = ANY($2)`,
		institutionID, sittingIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var a SpecialArrangement
		if err := rows.Scan(&a.ID, &a.InstitutionID, &a.ExamSittingID, &a.StudentID, &a.ClassID,
			&a.ArrangementType, &a.Details, &a.ExtraMinutes, &a.RoomName); err != nil {
			return nil, err
		}
		out[a.ExamSittingID] = append(out[a.ExamSittingID], &a)
	}
	return out, rows.Err()
}

func (r *Repository) ClassExists(ctx context.Context, institutionID, classID uuid.UUID) (bool, error) {
	var n int
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM classes WHERE institution_id=$1 AND id=$2 AND deleted_at IS NULL`,
		institutionID, classID).Scan(&n); err != nil {
		return false, err
	}
	return n > 0, nil
}

func (r *Repository) OfferingExists(ctx context.Context, institutionID, offeringID uuid.UUID) (bool, error) {
	var n int
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM academic_offerings WHERE institution_id=$1 AND id=$2`,
		institutionID, offeringID).Scan(&n); err != nil {
		return false, err
	}
	return n > 0, nil
}

func (r *Repository) OfferingName(ctx context.Context, institutionID, offeringID uuid.UUID) (string, string, error) {
	var name, typ *string
	err := r.db.QueryRow(ctx, `
		SELECT COALESCE(sub.name, la.name),
		       CASE WHEN sub.id IS NOT NULL THEN 'subject' WHEN la.id IS NOT NULL THEN 'learning_area' ELSE NULL END
		FROM academic_offerings ao
		LEFT JOIN subjects sub ON sub.id = ao.subject_id
		LEFT JOIN learning_areas la ON la.id = ao.learning_area_id
		WHERE ao.institution_id=$1 AND ao.id=$2`, institutionID, offeringID).Scan(&name, &typ)
	if err != nil {
		return "", "", err
	}
	n, t := "", ""
	if name != nil {
		n = *name
	}
	if typ != nil {
		t = *typ
	}
	return n, t, nil
}

func (r *Repository) StaffExists(ctx context.Context, institutionID, staffID uuid.UUID) (bool, error) {
	var n int
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM users WHERE institution_id=$1 AND id=$2 AND deleted_at IS NULL`,
		institutionID, staffID).Scan(&n); err != nil {
		return false, err
	}
	return n > 0, nil
}

func (r *Repository) ClassName(ctx context.Context, institutionID, classID uuid.UUID) (string, error) {
	var n string
	if err := r.db.QueryRow(ctx, `SELECT name FROM classes WHERE institution_id=$1 AND id=$2`,
		institutionID, classID).Scan(&n); err != nil {
		return "", err
	}
	return n, nil
}

func (r *Repository) StudentCountForClass(ctx context.Context, institutionID, classID uuid.UUID) (int, error) {
	var n int
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM students WHERE institution_id=$1 AND current_class_id=$2 AND status='active' AND deleted_at IS NULL`,
		institutionID, classID).Scan(&n); err != nil {
		return 0, err
	}
	return n, nil
}

func nullDate(s *string) interface{} {
	if s == nil {
		return nil
	}
	return *s
}

func nullTime(s *string) interface{} {
	if s == nil {
		return nil
	}
	return *s
}
