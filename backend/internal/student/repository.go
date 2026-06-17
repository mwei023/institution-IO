package student

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"institution-os/pkg/db"
	"github.com/jackc/pgx/v5"
)

var ErrStudentNotFound = errors.New("student not found")
var ErrAdmissionNotFound = errors.New("admission not found")
var ErrDuplicateAdmissionNumber = errors.New("admission number already exists")

// Repository handles database access for the student domain.
type Repository struct {
	db *db.Pool
}

// NewRepository creates a new student repository.
func NewRepository(db *db.Pool) *Repository {
	return &Repository{db: db}
}

// CreateAdmission inserts a new admission application.
func (r *Repository) CreateAdmission(ctx context.Context, a *Admission, parentJSON []byte) error {
	query := `
		INSERT INTO admissions (
			id, institution_id, application_number, academic_year, intended_class,
			first_name, last_name, date_of_birth, gender, status, parents_json,
			created_by, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8, $9, $10, $11,
			$12, NOW(), NOW()
		)`
	_, err := r.db.Exec(ctx, query,
		a.ID, a.InstitutionID, a.ApplicationNumber, a.AcademicYear, a.IntendedClass,
		a.FirstName, a.LastName, a.DateOfBirth, a.Gender, a.Status, parentJSON,
		nil, // created_by — TODO: pass from context
	)
	return err
}

// GetAdmissionByID retrieves a single admission.
func (r *Repository) GetAdmissionByID(ctx context.Context, institutionID, admissionID uuid.UUID) (*Admission, error) {
	query := `
		SELECT id, institution_id, application_number, academic_year, intended_class,
		       first_name, last_name, date_of_birth, gender, status,
		       reviewed_by, reviewed_at, review_notes, student_id, created_at
		FROM admissions
		WHERE institution_id = $1 AND id = $2 AND deleted_at IS NULL`

	var a Admission
	err := r.db.QueryRow(ctx, query, institutionID, admissionID).Scan(
		&a.ID, &a.InstitutionID, &a.ApplicationNumber, &a.AcademicYear, &a.IntendedClass,
		&a.FirstName, &a.LastName, &a.DateOfBirth, &a.Gender, &a.Status,
		&a.ReviewedBy, &a.ReviewedAt, &a.ReviewNotes, &a.StudentID, &a.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrAdmissionNotFound
		}
		return nil, fmt.Errorf("get admission: %w", err)
	}
	return &a, nil
}

// ListAdmissions returns a filtered page of admissions.
func (r *Repository) ListAdmissions(ctx context.Context, institutionID uuid.UUID, status string, page, pageSize int) ([]*Admission, int64, error) {
	offset := (page - 1) * pageSize

	countQuery := `
		SELECT COUNT(*) FROM admissions
		WHERE institution_id = $1 AND deleted_at IS NULL
		AND ($2 = '' OR status = $2)`
	var total int64
	if err := r.db.QueryRow(ctx, countQuery, institutionID, status).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := `
		SELECT id, institution_id, application_number, academic_year, intended_class,
		       first_name, last_name, date_of_birth, gender, status,
		       reviewed_by, reviewed_at, review_notes, student_id, created_at
		FROM admissions
		WHERE institution_id = $1 AND deleted_at IS NULL
		AND ($2 = '' OR status = $2)
		ORDER BY created_at DESC
		LIMIT $3 OFFSET $4`

	rows, err := r.db.Query(ctx, query, institutionID, status, pageSize, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var admissions []*Admission
	for rows.Next() {
		var a Admission
		if err := rows.Scan(
			&a.ID, &a.InstitutionID, &a.ApplicationNumber, &a.AcademicYear, &a.IntendedClass,
			&a.FirstName, &a.LastName, &a.DateOfBirth, &a.Gender, &a.Status,
			&a.ReviewedBy, &a.ReviewedAt, &a.ReviewNotes, &a.StudentID, &a.CreatedAt,
		); err != nil {
			return nil, 0, err
		}
		admissions = append(admissions, &a)
	}
	return admissions, total, rows.Err()
}

// UpdateAdmissionStatus updates the status, reviewer, and notes of an admission.
func (r *Repository) UpdateAdmissionStatus(ctx context.Context, institutionID, admissionID, reviewerID uuid.UUID, status, notes string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE admissions
		SET status = $1, reviewed_by = $2, reviewed_at = NOW(), review_notes = $3, updated_at = NOW()
		WHERE institution_id = $4 AND id = $5 AND deleted_at IS NULL`,
		status, reviewerID, notes, institutionID, admissionID,
	)
	return err
}

// CreateStudent inserts a new student record.
func (r *Repository) CreateStudent(ctx context.Context, s *Student) error {
	query := `
		INSERT INTO students (
			id, institution_id, admission_id, admission_number,
			first_name, last_name, date_of_birth, gender,
			nationality, is_boarder, uses_transport, status,
			admission_date, current_class_id,
			medical_conditions, allergies,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4,
			$5, $6, $7, $8,
			$9, $10, $11, $12,
			$13, $14,
			$15, $16,
			NOW(), NOW()
		)`
	_, err := r.db.Exec(ctx, query,
		s.ID, s.InstitutionID, nil, s.AdmissionNumber,
		s.FirstName, s.LastName, s.DateOfBirth, s.Gender,
		s.Nationality, s.IsBoarder, s.UsesTransport, s.Status,
		s.AdmissionDate, s.CurrentClassID,
		s.MedicalConditions, s.Allergies,
	)
	return err
}

// GetStudentByID retrieves a student by ID.
func (r *Repository) GetStudentByID(ctx context.Context, institutionID, studentID uuid.UUID) (*Student, error) {
	query := `
		SELECT id, institution_id, admission_number,
		       first_name, last_name, other_names, date_of_birth, gender,
		       birth_certificate_number, nationality, blood_group,
		       home_address, photo_url, current_class_id, admission_date,
		       kcpe_index_number, kcpe_marks, knec_registration_number,
		       status, is_boarder, uses_transport,
		       medical_conditions, allergies, created_at, updated_at
		FROM students
		WHERE institution_id = $1 AND id = $2 AND deleted_at IS NULL`

	var s Student
	err := r.db.QueryRow(ctx, query, institutionID, studentID).Scan(
		&s.ID, &s.InstitutionID, &s.AdmissionNumber,
		&s.FirstName, &s.LastName, &s.OtherNames, &s.DateOfBirth, &s.Gender,
		&s.BirthCertificateNumber, &s.Nationality, &s.BloodGroup,
		&s.HomeAddress, &s.PhotoURL, &s.CurrentClassID, &s.AdmissionDate,
		&s.KCPEIndexNumber, &s.KCPEMarks, &s.KNECRegistrationNumber,
		&s.Status, &s.IsBoarder, &s.UsesTransport,
		&s.MedicalConditions, &s.Allergies, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrStudentNotFound
		}
		return nil, fmt.Errorf("get student: %w", err)
	}
	return &s, nil
}

// ListStudents returns a filtered, paginated list of students.
func (r *Repository) ListStudents(ctx context.Context, institutionID uuid.UUID, f *StudentFilter) ([]*StudentListItem, int64, error) {
	offset := (f.Page - 1) * f.PageSize
	if offset < 0 {
		offset = 0
	}

	countQuery := `
		SELECT COUNT(*) FROM students s
		WHERE s.institution_id = $1
		  AND s.deleted_at IS NULL
		  AND ($2 = '' OR s.status = $2)
		  AND ($3::uuid IS NULL OR s.current_class_id = $3)
		  AND ($4 = '' OR (
		       lower(s.first_name || ' ' || s.last_name) LIKE lower('%' || $4 || '%')
		       OR s.admission_number ILIKE '%' || $4 || '%'
		  ))`

	var total int64
	if err := r.db.QueryRow(ctx, countQuery,
		institutionID, f.Status, f.ClassID, f.Search,
	).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := `
		SELECT s.id, s.admission_number,
		       s.first_name || ' ' || s.last_name AS full_name,
		       s.gender,
		       c.name || COALESCE(' ' || c.stream, '') AS class_name,
		       s.status, s.photo_url, s.is_boarder
		FROM students s
		LEFT JOIN classes c ON c.id = s.current_class_id
		WHERE s.institution_id = $1
		  AND s.deleted_at IS NULL
		  AND ($2 = '' OR s.status = $2)
		  AND ($3::uuid IS NULL OR s.current_class_id = $3)
		  AND ($4 = '' OR (
		       lower(s.first_name || ' ' || s.last_name) LIKE lower('%' || $4 || '%')
		       OR s.admission_number ILIKE '%' || $4 || '%'
		  ))
		ORDER BY s.last_name, s.first_name
		LIMIT $5 OFFSET $6`

	rows, err := r.db.Query(ctx, query,
		institutionID, f.Status, f.ClassID, f.Search, f.PageSize, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var students []*StudentListItem
	for rows.Next() {
		var s StudentListItem
		if err := rows.Scan(
			&s.ID, &s.AdmissionNumber, &s.FullName, &s.Gender,
			&s.ClassName, &s.Status, &s.PhotoURL, &s.IsBoarder,
		); err != nil {
			return nil, 0, err
		}
		students = append(students, &s)
	}
	return students, total, rows.Err()
}

// GetTimeline retrieves the student journey timeline events.
func (r *Repository) GetTimeline(ctx context.Context, institutionID, studentID uuid.UUID) ([]*TimelineEvent, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, event_type, title, description, metadata, event_date, created_at
		FROM student_timeline_events
		WHERE institution_id = $1 AND student_id = $2
		ORDER BY event_date DESC, created_at DESC
		LIMIT 100`,
		institutionID, studentID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []*TimelineEvent
	for rows.Next() {
		var e TimelineEvent
		if err := rows.Scan(
			&e.ID, &e.EventType, &e.Title, &e.Description, &e.Metadata, &e.EventDate, &e.CreatedAt,
		); err != nil {
			return nil, err
		}
		events = append(events, &e)
	}
	return events, rows.Err()
}

// AddTimelineEvent appends an event to the student timeline.
func (r *Repository) AddTimelineEvent(ctx context.Context, institutionID, studentID uuid.UUID,
	eventType, title, description string, eventDate time.Time) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO student_timeline_events
			(id, institution_id, student_id, event_type, title, description, event_date, created_at)
		VALUES
			(gen_random_uuid(), $1, $2, $3, $4, $5, $6, NOW())`,
		institutionID, studentID, eventType, title, description, eventDate,
	)
	return err
}

// GetNextApplicationNumber generates the next sequential application number.
func (r *Repository) GetNextApplicationNumber(ctx context.Context, institutionID uuid.UUID, year string) (string, error) {
	var count int
	err := r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM admissions
		WHERE institution_id = $1 AND academic_year = $2`,
		institutionID, year,
	).Scan(&count)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("APP/%s/%04d", year, count+1), nil
}

// GetNextAdmissionNumber generates the next sequential admission number.
func (r *Repository) GetNextAdmissionNumber(ctx context.Context, institutionID uuid.UUID, year, classCode string) (string, error) {
	var count int
	err := r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM students
		WHERE institution_id = $1
		AND admission_number LIKE $2`,
		institutionID, year+"/"+classCode+"/%",
	).Scan(&count)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s/%s/%04d", year, classCode, count+1), nil
}

// GetStudentParents returns the parents linked to a student.
func (r *Repository) GetStudentParents(ctx context.Context, institutionID, studentID uuid.UUID) ([]*ParentGuardian, error) {
	rows, err := r.db.Query(ctx, `
		SELECT pg.id, pg.institution_id, pg.first_name, pg.last_name,
		       pg.relationship, pg.phone, pg.email, pg.occupation,
		       pg.is_primary_contact, pg.is_fee_payer
		FROM parent_guardians pg
		JOIN student_parent_links spl ON spl.parent_id = pg.id
		WHERE spl.institution_id = $1 AND spl.student_id = $2
		AND pg.deleted_at IS NULL`,
		institutionID, studentID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var parents []*ParentGuardian
	for rows.Next() {
		var p ParentGuardian
		if err := rows.Scan(
			&p.ID, &p.InstitutionID, &p.FirstName, &p.LastName,
			&p.Relationship, &p.Phone, &p.Email, &p.Occupation,
			&p.IsPrimaryContact, &p.IsFeePayer,
		); err != nil {
			return nil, err
		}
		parents = append(parents, &p)
	}
	return parents, rows.Err()
}
