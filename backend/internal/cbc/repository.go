package cbc

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"institution-os/pkg/db"
)

var (
	ErrLearningAreaNotFound = errors.New("learning area not found")
	ErrReportCardNotFound   = errors.New("cbc report card not found")
	ErrReportCardPublished  = errors.New("report card already published — unpublish before regenerating")
	ErrNoRatingBandsConfigured = errors.New("no rating bands configured for this curriculum/grade band")
	ErrComponentNotFound = errors.New("learning area component not found")
)

// Repository handles database access for the CBC/CBE academic layer.
type Repository struct {
	db *db.Pool
}

func NewRepository(db *db.Pool) *Repository {
	return &Repository{db: db}
}

// GetInstitutionCurriculumType returns the institution's configured
// curriculum ('844' or 'cbe') — the gate that decides which academic
// engine's screens/endpoints an institution should be using at all.
func (r *Repository) GetInstitutionCurriculumType(ctx context.Context, institutionID uuid.UUID) (string, error) {
	var curriculumType string
	err := r.db.QueryRow(ctx, `SELECT curriculum_type FROM institutions WHERE id = $1`, institutionID).Scan(&curriculumType)
	if err != nil {
		return "", fmt.Errorf("get institution curriculum type: %w", err)
	}
	return curriculumType, nil
}

// ─── Learning Areas ─────────────────────────────────────────────────

func (r *Repository) CreateLearningArea(ctx context.Context, institutionID uuid.UUID, req *CreateLearningAreaRequest) (*LearningArea, error) {
	la := &LearningArea{
		ID:            uuid.New(),
		InstitutionID: institutionID,
		Name:          req.Name,
		Code:          req.Code,
		IsCompulsory:  req.IsCompulsory,
		Source:        "institution",
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin create learning area: %w", err)
	}
	defer tx.Rollback(ctx)

	err = tx.QueryRow(ctx, `
		INSERT INTO learning_areas (id, institution_id, name, code, is_compulsory, source, created_at)
		VALUES ($1,$2,$3,$4,$5,'institution',NOW())
		RETURNING created_at`,
		la.ID, la.InstitutionID, la.Name, la.Code, la.IsCompulsory,
	).Scan(&la.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("create learning area: %w", err)
	}

	// teaching_assignments (migration 000014) references
	// academic_offerings, not learning_areas directly — a learning area
	// with no offering can never be assigned to a teacher. Every
	// learning area now gets a matching offering from birth, in the
	// same transaction, so this gap can't reopen for future rows.
	if _, err := tx.Exec(ctx, `
		INSERT INTO academic_offerings (institution_id, learning_area_id)
		VALUES ($1, $2)`,
		institutionID, la.ID,
	); err != nil {
		return nil, fmt.Errorf("create academic offering for learning area: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit create learning area: %w", err)
	}
	return la, nil
}

func (r *Repository) ListLearningAreas(ctx context.Context, institutionID uuid.UUID) ([]*LearningArea, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, institution_id, name, code, is_compulsory, source, created_at
		FROM learning_areas
		WHERE institution_id = $1 AND deleted_at IS NULL
		ORDER BY name`, institutionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	areas := []*LearningArea{}
	for rows.Next() {
		var la LearningArea
		if err := rows.Scan(&la.ID, &la.InstitutionID, &la.Name, &la.Code, &la.IsCompulsory, &la.Source, &la.CreatedAt); err != nil {
			return nil, err
		}
		areas = append(areas, &la)
	}
	return areas, rows.Err()
}

func (r *Repository) GetLearningAreaByID(ctx context.Context, institutionID, id uuid.UUID) (*LearningArea, error) {
	var la LearningArea
	err := r.db.QueryRow(ctx, `
		SELECT id, institution_id, name, code, is_compulsory, source, created_at
		FROM learning_areas
		WHERE institution_id = $1 AND id = $2 AND deleted_at IS NULL`,
		institutionID, id,
	).Scan(&la.ID, &la.InstitutionID, &la.Name, &la.Code, &la.IsCompulsory, &la.Source, &la.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrLearningAreaNotFound
		}
		return nil, err
	}
	return &la, nil
}

// ─── Rating bands ───────────────────────────────────────────────────

// GetRatingBands returns the bands to use for this institution's
// curriculum+grade_band: the institution's own override rows if any
// exist, otherwise the official default rows (institution_id IS NULL).
func (r *Repository) GetRatingBands(ctx context.Context, institutionID uuid.UUID, curriculumType string, gradeBand *string) ([]*RatingBand, error) {
	// Try institution-specific override first.
	rows, err := r.queryRatingBands(ctx, curriculumType, gradeBand, &institutionID)
	if err != nil {
		return nil, err
	}
	if len(rows) > 0 {
		return rows, nil
	}
	// Fall back to the official default.
	rows, err = r.queryRatingBands(ctx, curriculumType, gradeBand, nil)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, ErrNoRatingBandsConfigured
	}
	return rows, nil
}

func (r *Repository) queryRatingBands(ctx context.Context, curriculumType string, gradeBand *string, institutionID *uuid.UUID) ([]*RatingBand, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, curriculum_type, grade_band, institution_id, code, label,
		       min_pct, max_pct, points, default_statement, sort_order, created_at
		FROM curriculum_rating_bands
		WHERE curriculum_type = $1
		  AND ($2::text IS NULL OR grade_band = $2)
		  AND ($3::uuid IS NULL AND institution_id IS NULL OR institution_id = $3)
		ORDER BY sort_order`,
		curriculumType, gradeBand, institutionID,
	)
	if err != nil {
		return nil, fmt.Errorf("query rating bands: %w", err)
	}
	defer rows.Close()

	bands := []*RatingBand{}
	for rows.Next() {
		var b RatingBand
		if err := rows.Scan(&b.ID, &b.CurriculumType, &b.GradeBand, &b.InstitutionID, &b.Code, &b.Label,
			&b.MinPct, &b.MaxPct, &b.Points, &b.DefaultStatement, &b.SortOrder, &b.CreatedAt); err != nil {
			return nil, err
		}
		bands = append(bands, &b)
	}
	return bands, rows.Err()
}

// bandForPercentage picks the matching band for a percentage from an
// already-fetched band list (avoids a query per student per learning area).
func bandForPercentage(bands []*RatingBand, pct float64) (*RatingBand, error) {
	for _, b := range bands {
		if pct >= b.MinPct && pct <= b.MaxPct {
			return b, nil
		}
	}
	return nil, fmt.Errorf("no rating band matches percentage %.2f", pct)
}

// ─── Assessment components & entries ────────────────────────────────

func (r *Repository) CreateComponent(ctx context.Context, institutionID uuid.UUID, req *CreateComponentRequest) (*Component, error) {
	c := &Component{
		ID:             uuid.New(),
		InstitutionID:  institutionID,
		ClassID:        req.ClassID,
		LearningAreaID: req.LearningAreaID,
		TermID:         req.TermID,
		Name:           req.Name,
		Weight:         req.Weight,
		MaxMarks:       req.MaxMarks,
	}
	_, err := r.db.Exec(ctx, `
		INSERT INTO learning_area_components
			(id, institution_id, class_id, learning_area_id, term_id, name, weight, max_marks, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,NOW())`,
		c.ID, c.InstitutionID, c.ClassID, c.LearningAreaID, c.TermID, c.Name, c.Weight, c.MaxMarks,
	)
	if err != nil {
		return nil, fmt.Errorf("create learning area component: %w", err)
	}
	return c, nil
}

func (r *Repository) ListComponents(ctx context.Context, institutionID, classID, learningAreaID, termID uuid.UUID) ([]*Component, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, institution_id, class_id, learning_area_id, term_id, name, weight, max_marks
		FROM learning_area_components
		WHERE institution_id = $1 AND class_id = $2 AND learning_area_id = $3 AND term_id = $4
		ORDER BY name`, institutionID, classID, learningAreaID, termID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	comps := []*Component{}
	for rows.Next() {
		var c Component
		if err := rows.Scan(&c.ID, &c.InstitutionID, &c.ClassID, &c.LearningAreaID, &c.TermID, &c.Name, &c.Weight, &c.MaxMarks); err != nil {
			return nil, err
		}
		comps = append(comps, &c)
	}
	return comps, rows.Err()
}

func (r *Repository) GetEntries(ctx context.Context, institutionID, componentID uuid.UUID) ([]*Entry, error) {
	rows, err := r.db.Query(ctx, `
		SELECT s.id, $2::uuid, s.id, s.first_name || ' ' || s.last_name,
		       lae.marks, COALESCE(lae.is_absent, false), lae.entered_by, COALESCE(lae.updated_at, NOW())
		FROM students s
		LEFT JOIN learning_area_entries lae ON lae.student_id = s.id AND lae.component_id = $2
		JOIN learning_area_components lac ON lac.id = $2
		WHERE s.institution_id = $1
		  AND s.current_class_id = lac.class_id
		  AND s.status = 'active'
		  AND s.deleted_at IS NULL
		ORDER BY s.last_name, s.first_name`,
		institutionID, componentID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	entries := []*Entry{}
	for rows.Next() {
		var e Entry
		if err := rows.Scan(&e.ID, &e.ComponentID, &e.StudentID, &e.StudentName, &e.Marks, &e.IsAbsent, &e.EnteredBy, &e.UpdatedAt); err != nil {
			return nil, err
		}
		entries = append(entries, &e)
	}
	return entries, rows.Err()
}

func (r *Repository) SaveEntries(ctx context.Context, institutionID, componentID, enteredBy uuid.UUID, entries []EntryInput) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	for _, entry := range entries {
		_, err := tx.Exec(ctx, `
			INSERT INTO learning_area_entries
				(id, institution_id, component_id, student_id, marks, is_absent, entered_by, created_at, updated_at)
			VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, $6, NOW(), NOW())
			ON CONFLICT (institution_id, component_id, student_id)
			DO UPDATE SET marks = EXCLUDED.marks, is_absent = EXCLUDED.is_absent,
			              entered_by = EXCLUDED.entered_by, updated_at = NOW()`,
			institutionID, componentID, entry.StudentID, entry.Marks, entry.IsAbsent, enteredBy,
		)
		if err != nil {
			return fmt.Errorf("upsert learning area entry for student %s: %w", entry.StudentID, err)
		}
	}
	return tx.Commit(ctx)
}

// GetLearningAreaPercentage computes one student's weighted percentage
// for one learning area in one term, same weighting formula as 8-4-4's
// GetSubjectScoresForStudent — just against the parallel tables.
func (r *Repository) GetLearningAreaPercentage(ctx context.Context, institutionID, studentID, learningAreaID, termID uuid.UUID) (*float64, error) {
	var pct float64
	err := r.db.QueryRow(ctx, `
		SELECT ROUND(SUM((lae.marks / lac.max_marks) * lac.weight) / SUM(lac.weight) * 100, 2)
		FROM learning_area_entries lae
		JOIN learning_area_components lac ON lac.id = lae.component_id
		WHERE lac.institution_id = $1
		  AND lac.term_id = $2
		  AND lac.learning_area_id = $3
		  AND lae.student_id = $4
		  AND lae.is_absent = false`,
		institutionID, termID, learningAreaID, studentID,
	).Scan(&pct)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil // no entries yet for this learning area
		}
		return nil, fmt.Errorf("get learning area percentage: %w", err)
	}
	return &pct, nil
}

// GetAllLearningAreaPercentagesForStudent returns every learning area
// this student has entries for in a term, with computed percentage.
func (r *Repository) GetAllLearningAreaPercentagesForStudent(ctx context.Context, institutionID, studentID, termID uuid.UUID) (map[uuid.UUID]float64, error) {
	rows, err := r.db.Query(ctx, `
		SELECT lac.learning_area_id,
		       ROUND(SUM((lae.marks / lac.max_marks) * lac.weight) / SUM(lac.weight) * 100, 2) AS pct
		FROM learning_area_entries lae
		JOIN learning_area_components lac ON lac.id = lae.component_id
		WHERE lac.institution_id = $1
		  AND lac.term_id = $2
		  AND lae.student_id = $3
		  AND lae.is_absent = false
		GROUP BY lac.learning_area_id`,
		institutionID, termID, studentID,
	)
	if err != nil {
		return nil, fmt.Errorf("get all learning area percentages: %w", err)
	}
	defer rows.Close()

	result := map[uuid.UUID]float64{}
	for rows.Next() {
		var laID uuid.UUID
		var pct float64
		if err := rows.Scan(&laID, &pct); err != nil {
			return nil, err
		}
		result[laID] = pct
	}
	return result, rows.Err()
}

// ─── Teacher assignments ─────────────────────────────────────────────

func (r *Repository) CreateTeacherAssignment(ctx context.Context, institutionID uuid.UUID, classID, learningAreaID, teacherID, termID uuid.UUID) (*TeacherAssignment, error) {
	ta := &TeacherAssignment{
		ID: uuid.New(), InstitutionID: institutionID, ClassID: classID,
		LearningAreaID: learningAreaID, TeacherID: teacherID, TermID: termID,
	}

	// Resolve (or self-heal) the academic_offerings row for this
	// learning area. CreateLearningArea now creates this at birth, but
	// this upsert covers any learning area that predates that fix —
	// ON CONFLICT ... DO UPDATE (a no-op assignment) forces RETURNING
	// to fire on conflict too, so this is one round trip either way,
	// no race between a concurrent check-then-insert.
	var offeringID uuid.UUID
	err := r.db.QueryRow(ctx, `
		INSERT INTO academic_offerings (institution_id, learning_area_id)
		VALUES ($1, $2)
		ON CONFLICT (learning_area_id) DO UPDATE SET learning_area_id = EXCLUDED.learning_area_id
		RETURNING id`,
		institutionID, learningAreaID,
	).Scan(&offeringID)
	if err != nil {
		return nil, fmt.Errorf("resolve academic offering for learning area %s: %w", learningAreaID, err)
	}

	// Was: INSERT INTO learning_area_teacher_assignments (...). Now
	// writes to teaching_assignments — the one real assignment
	// mechanism shared with 8-4-4 (see checkpoint #32). The returned
	// TeacherAssignment struct's shape is unchanged; only the storage
	// underneath it moved.
	err = r.db.QueryRow(ctx, `
		INSERT INTO teaching_assignments
			(id, institution_id, class_id, academic_offering_id, teacher_id, term_id, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,NOW())
		ON CONFLICT (institution_id, class_id, academic_offering_id, term_id)
		DO UPDATE SET teacher_id = EXCLUDED.teacher_id
		RETURNING created_at`,
		ta.ID, ta.InstitutionID, ta.ClassID, offeringID, ta.TeacherID, ta.TermID,
	).Scan(&ta.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("create teaching assignment: %w", err)
	}
	return ta, nil
}

// TeacherHasLearningAreaAssignment checks whether a teacher is assigned
// to teach ANY learning area in this class — same coarse-guard
// philosophy as academics.TeacherHasAssignmentInClass.
func (r *Repository) TeacherHasLearningAreaAssignment(ctx context.Context, institutionID, teacherID, classID uuid.UUID) (bool, error) {
	var count int
	err := r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM learning_area_teacher_assignments
		WHERE institution_id = $1 AND teacher_id = $2 AND class_id = $3`,
		institutionID, teacherID, classID,
	).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("check learning area assignment: %w", err)
	}
	return count > 0, nil
}

// ─── CBC report cards ────────────────────────────────────────────────

// UpsertReportCard follows the exact same published-guard pattern as
// academics.UpsertReportCard: a published card is never silently
// overwritten by regeneration.
func (r *Repository) UpsertReportCard(ctx context.Context, institutionID, classID, termID uuid.UUID, rc *ReportCard) error {
	var id uuid.UUID
	err := r.db.QueryRow(ctx, `
		INSERT INTO learning_area_report_cards (
			id, institution_id, student_id, class_id, term_id,
			total_points, mean_points, class_position, stream_position,
			attendance_days_present, attendance_days_total,
			class_teacher_remark, principal_remark, status,
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
			total_points = EXCLUDED.total_points,
			mean_points = EXCLUDED.mean_points,
			class_position = EXCLUDED.class_position,
			stream_position = EXCLUDED.stream_position,
			attendance_days_present = EXCLUDED.attendance_days_present,
			attendance_days_total = EXCLUDED.attendance_days_total,
			status = EXCLUDED.status,
			updated_at = NOW()
		WHERE learning_area_report_cards.status <> 'published'
		RETURNING id`,
		institutionID, rc.StudentID, classID, termID,
		rc.TotalPoints, rc.MeanPoints, rc.ClassPosition, rc.StreamPosition,
		rc.AttendanceDaysPresent, rc.AttendanceDaysTotal,
		rc.ClassTeacherRemark, rc.PrincipalRemark, rc.Status,
	).Scan(&id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrReportCardPublished
		}
		return fmt.Errorf("upsert cbc report card: %w", err)
	}
	rc.ID = id

	// Replace line items wholesale — simpler and safer than diffing,
	// and this whole upsert already refuses to run against a published
	// card, so there's no risk of clobbering something a parent has seen.
	if _, err := r.db.Exec(ctx, `DELETE FROM learning_area_report_card_lines WHERE report_card_id = $1`, id); err != nil {
		return fmt.Errorf("clear existing report card lines: %w", err)
	}
	for _, line := range rc.Lines {
		_, err := r.db.Exec(ctx, `
			INSERT INTO learning_area_report_card_lines
				(id, report_card_id, learning_area_id, percentage, rating_band_id,
				 rating_code, rating_label, rating_points, statement, teacher_id)
			VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, $6, $7, $8, $9)`,
			id, line.LearningAreaID, line.Percentage, line.RatingBandID,
			line.RatingCode, line.RatingLabel, line.RatingPoints, line.Statement, line.TeacherID,
		)
		if err != nil {
			return fmt.Errorf("insert report card line for learning area %s: %w", line.LearningAreaID, err)
		}
	}
	return nil
}

func (r *Repository) getReportCardRaw(ctx context.Context, institutionID, studentID, termID uuid.UUID) (*ReportCard, error) {
	var rc ReportCard
	err := r.db.QueryRow(ctx, `
		SELECT id, student_id, total_points, mean_points, class_position, stream_position,
		       attendance_days_present, attendance_days_total,
		       class_teacher_remark, principal_remark, status, published_at
		FROM learning_area_report_cards
		WHERE institution_id = $1 AND student_id = $2 AND term_id = $3`,
		institutionID, studentID, termID,
	).Scan(&rc.ID, &rc.StudentID, &rc.TotalPoints, &rc.MeanPoints, &rc.ClassPosition, &rc.StreamPosition,
		&rc.AttendanceDaysPresent, &rc.AttendanceDaysTotal,
		&rc.ClassTeacherRemark, &rc.PrincipalRemark, &rc.Status, &rc.PublishedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrReportCardNotFound
		}
		return nil, err
	}
	return &rc, nil
}

// PublishReportCard and UnpublishReportCard mirror the exact
// idempotency semantics of their academics package counterparts:
// publishing an already-published card is a no-op returning the
// existing record (never resets published_at on a double-click);
// unpublishing an already-draft card is a no-op too.
func (r *Repository) PublishReportCard(ctx context.Context, institutionID, studentID, termID uuid.UUID) (*ReportCard, error) {
	var rc ReportCard
	err := r.db.QueryRow(ctx, `
		UPDATE learning_area_report_cards
		SET status = 'published', published_at = NOW(), updated_at = NOW()
		WHERE institution_id = $1 AND student_id = $2 AND term_id = $3 AND status = 'draft'
		RETURNING id, student_id, total_points, mean_points, class_position, stream_position,
		          attendance_days_present, attendance_days_total,
		          class_teacher_remark, principal_remark, status, published_at`,
		institutionID, studentID, termID,
	).Scan(&rc.ID, &rc.StudentID, &rc.TotalPoints, &rc.MeanPoints, &rc.ClassPosition, &rc.StreamPosition,
		&rc.AttendanceDaysPresent, &rc.AttendanceDaysTotal,
		&rc.ClassTeacherRemark, &rc.PrincipalRemark, &rc.Status, &rc.PublishedAt)
	if err == nil {
		return &rc, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("publish cbc report card: %w", err)
	}
	return r.getReportCardRaw(ctx, institutionID, studentID, termID)
}

func (r *Repository) UnpublishReportCard(ctx context.Context, institutionID, studentID, termID uuid.UUID) (*ReportCard, error) {
	var rc ReportCard
	err := r.db.QueryRow(ctx, `
		UPDATE learning_area_report_cards
		SET status = 'draft', published_at = NULL, updated_at = NOW()
		WHERE institution_id = $1 AND student_id = $2 AND term_id = $3 AND status = 'published'
		RETURNING id, student_id, total_points, mean_points, class_position, stream_position,
		          attendance_days_present, attendance_days_total,
		          class_teacher_remark, principal_remark, status, published_at`,
		institutionID, studentID, termID,
	).Scan(&rc.ID, &rc.StudentID, &rc.TotalPoints, &rc.MeanPoints, &rc.ClassPosition, &rc.StreamPosition,
		&rc.AttendanceDaysPresent, &rc.AttendanceDaysTotal,
		&rc.ClassTeacherRemark, &rc.PrincipalRemark, &rc.Status, &rc.PublishedAt)
	if err == nil {
		return &rc, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("unpublish cbc report card: %w", err)
	}
	return r.getReportCardRaw(ctx, institutionID, studentID, termID)
}

func (r *Repository) GetPublishedReportCardForStudent(ctx context.Context, institutionID, studentID uuid.UUID) (*ReportCard, error) {
	var rc ReportCard
	err := r.db.QueryRow(ctx, `
		SELECT rc.id, rc.student_id, s.first_name || ' ' || s.last_name, c.name, t.name,
		       rc.total_points, rc.mean_points, rc.class_position, rc.stream_position,
		       rc.attendance_days_present, rc.attendance_days_total,
		       rc.class_teacher_remark, rc.principal_remark, rc.status, rc.published_at
		FROM learning_area_report_cards rc
		JOIN students s ON s.id = rc.student_id
		JOIN classes c ON c.id = rc.class_id
		JOIN terms t ON t.id = rc.term_id
		WHERE rc.institution_id = $1 AND rc.student_id = $2 AND rc.status = 'published'
		ORDER BY rc.published_at DESC LIMIT 1`,
		institutionID, studentID,
	).Scan(&rc.ID, &rc.StudentID, &rc.StudentName, &rc.ClassName, &rc.Term,
		&rc.TotalPoints, &rc.MeanPoints, &rc.ClassPosition, &rc.StreamPosition,
		&rc.AttendanceDaysPresent, &rc.AttendanceDaysTotal,
		&rc.ClassTeacherRemark, &rc.PrincipalRemark, &rc.Status, &rc.PublishedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrReportCardNotFound
		}
		return nil, fmt.Errorf("get published cbc report card: %w", err)
	}

	lineRows, err := r.db.Query(ctx, `
		SELECT l.id, l.report_card_id, l.learning_area_id, la.name, l.percentage,
		       l.rating_band_id, l.rating_code, l.rating_label, l.rating_points, l.statement, l.teacher_id
		FROM learning_area_report_card_lines l
		JOIN learning_areas la ON la.id = l.learning_area_id
		WHERE l.report_card_id = $1
		ORDER BY la.name`, rc.ID)
	if err != nil {
		return nil, fmt.Errorf("get report card lines: %w", err)
	}
	defer lineRows.Close()
	for lineRows.Next() {
		var l ReportCardLine
		if err := lineRows.Scan(&l.ID, &l.ReportCardID, &l.LearningAreaID, &l.LearningAreaName, &l.Percentage,
			&l.RatingBandID, &l.RatingCode, &l.RatingLabel, &l.RatingPoints, &l.Statement, &l.TeacherID); err != nil {
			return nil, err
		}
		rc.Lines = append(rc.Lines, &l)
	}
	return &rc, lineRows.Err()
}

// GetStudentNames returns display names for a set of student IDs —
// used by bulk report card generation, which otherwise never joins
// against students at all.
func (r *Repository) GetStudentNames(ctx context.Context, institutionID uuid.UUID, studentIDs []uuid.UUID) (map[uuid.UUID]string, error) {
	if len(studentIDs) == 0 {
		return map[uuid.UUID]string{}, nil
	}
	rows, err := r.db.Query(ctx, `
		SELECT id, first_name || ' ' || last_name
		FROM students
		WHERE institution_id = $1 AND id = ANY($2)`,
		institutionID, studentIDs,
	)
	if err != nil {
		return nil, fmt.Errorf("get student names: %w", err)
	}
	defer rows.Close()

	names := map[uuid.UUID]string{}
	for rows.Next() {
		var id uuid.UUID
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			return nil, err
		}
		names[id] = name
	}
	return names, rows.Err()
}

func (r *Repository) GetComponentClassID(ctx context.Context, institutionID, componentID uuid.UUID) (uuid.UUID, error) {
	var classID uuid.UUID
	err := r.db.QueryRow(ctx, `
		SELECT class_id FROM learning_area_components
		WHERE institution_id = $1 AND id = $2`,
		institutionID, componentID,
	).Scan(&classID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return uuid.Nil, ErrComponentNotFound
		}
		return uuid.Nil, fmt.Errorf("get component class id: %w", err)
	}
	return classID, nil
}