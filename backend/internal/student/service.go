package student

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"institution-os/pkg/events"
)

// Service implements all student domain business logic.
type Service struct {
	repo      Store
	publisher events.Publisher
}

// NewService creates a new student service.
func NewService(repo Store, publisher events.Publisher) *Service {
	return &Service{repo: repo, publisher: publisher}
}

// CreateAdmission creates a new admission application.
func (s *Service) CreateAdmission(ctx context.Context, institutionID uuid.UUID, req *CreateAdmissionRequest) (*Admission, error) {
	// Parse date of birth
	dob, err := time.Parse("2006-01-02", req.DateOfBirth)
	if err != nil {
		return nil, fmt.Errorf("invalid date_of_birth format: use YYYY-MM-DD")
	}

	year := fmt.Sprintf("%d", time.Now().Year())
	appNumber, err := s.repo.GetNextApplicationNumber(ctx, institutionID, year)
	if err != nil {
		return nil, fmt.Errorf("generate application number: %w", err)
	}

	// Build parent JSON
	parentInfo := []map[string]interface{}{
		{
			"name":         req.ParentName,
			"relationship": req.ParentRelationship,
			"phone":        req.ParentPhone,
			"email":        req.ParentEmail,
		},
	}
	parentJSON, err := json.Marshal(parentInfo)
	if err != nil {
		return nil, err
	}

	nationality := req.Nationality
	if nationality == "" {
		nationality = "Kenyan"
	}

	admission := &Admission{
		ID:                uuid.New(),
		InstitutionID:     institutionID,
		ApplicationNumber: appNumber,
		AcademicYear:      year,
		IntendedClass:     req.IntendedClass,
		FirstName:         req.FirstName,
		LastName:          req.LastName,
		DateOfBirth:       dob,
		Gender:            req.Gender,
		Status:            "submitted",
		CreatedAt:         time.Now(),
	}

	if err := s.repo.CreateAdmission(ctx, admission, parentJSON); err != nil {
		return nil, fmt.Errorf("create admission: %w", err)
	}

	// Publish domain event
	evt := events.New(events.EventAdmissionCreated, institutionID.String())
	evt.EntityType = "admission"
	evt.EntityID = admission.ID.String()
	_ = s.publisher.Publish(evt)

	return admission, nil
}

// GetAdmission retrieves a single admission.
func (s *Service) GetAdmission(ctx context.Context, institutionID, admissionID uuid.UUID) (*Admission, error) {
	return s.repo.GetAdmissionByID(ctx, institutionID, admissionID)
}

// ListAdmissions returns a paginated list of admissions.
func (s *Service) ListAdmissions(ctx context.Context, institutionID uuid.UUID, status string, page, pageSize int) ([]*Admission, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 50
	}
	return s.repo.ListAdmissions(ctx, institutionID, status, page, pageSize)
}

// ReviewAdmission approves or rejects an admission.
func (s *Service) ReviewAdmission(ctx context.Context, institutionID, admissionID, reviewerID uuid.UUID, action, notes string) (*Admission, error) {
	admission, err := s.repo.GetAdmissionByID(ctx, institutionID, admissionID)
	if err != nil {
		return nil, err
	}

	if admission.Status != "submitted" && admission.Status != "under_review" {
		return nil, fmt.Errorf("admission cannot be reviewed in status: %s", admission.Status)
	}

	var newStatus string
	var eventType events.EventType
	switch action {
	case "approve":
		newStatus = "approved"
		eventType = events.EventAdmissionApproved
	case "reject":
		newStatus = "rejected"
		eventType = events.EventAdmissionRejected
	default:
		return nil, fmt.Errorf("invalid action: %s", action)
	}

	if err := s.repo.UpdateAdmissionStatus(ctx, institutionID, admissionID, reviewerID, newStatus, notes); err != nil {
		return nil, fmt.Errorf("update admission status: %w", err)
	}

	admission.Status = newStatus

	evt := events.New(eventType, institutionID.String())
	evt.EntityType = "admission"
	evt.EntityID = admissionID.String()
	evt.ActorID = reviewerID.String()
	_ = s.publisher.Publish(evt)

	return admission, nil
}

// EnrollStudent creates a student record from an approved admission.
func (s *Service) EnrollStudent(ctx context.Context, institutionID, admissionID uuid.UUID, req *EnrollStudentRequest) (*Student, error) {
	admission, err := s.repo.GetAdmissionByID(ctx, institutionID, admissionID)
	if err != nil {
		return nil, err
	}

	if admission.Status != "approved" && admission.Status != "awaiting_enrollment" {
		return nil, fmt.Errorf("admission must be approved before enrollment, current status: %s", admission.Status)
	}

	if admission.StudentID != nil {
		return nil, fmt.Errorf("admission already enrolled, student ID: %s", admission.StudentID)
	}

	year := fmt.Sprintf("%d", time.Now().Year())
	admNumber, err := s.repo.GetNextAdmissionNumber(ctx, institutionID, year, "S")
	if err != nil {
		return nil, fmt.Errorf("generate admission number: %w", err)
	}

	student := &Student{
		ID:              uuid.New(),
		InstitutionID:   institutionID,
		AdmissionNumber: admNumber,
		FirstName:       admission.FirstName,
		LastName:        admission.LastName,
		DateOfBirth:     admission.DateOfBirth,
		Gender:          admission.Gender,
		Nationality:     "Kenyan",
		Status:          "active",
		AdmissionDate:   time.Now(),
		CurrentClassID:  &req.ClassID,
		IsBoarder:       req.BedID != nil,
		UsesTransport:   req.RouteID != nil,
	}

	if err := s.repo.CreateStudent(ctx, student); err != nil {
		return nil, fmt.Errorf("create student: %w", err)
	}

	// Update admission with student_id and status
	studentID := student.ID
	_ = studentID

	// Add enrollment timeline event
	_ = s.repo.AddTimelineEvent(ctx, institutionID, student.ID,
		"enrolled", "Enrolled at school",
		fmt.Sprintf("Enrolled in %s for academic year %s", req.ClassID, year),
		time.Now(),
	)

	evt := events.New(events.EventStudentEnrolled, institutionID.String())
	evt.EntityType = "student"
	evt.EntityID = student.ID.String()
	_ = s.publisher.Publish(evt)

	return student, nil
}

// GetStudent retrieves a student by ID.
func (s *Service) GetStudent(ctx context.Context, institutionID, studentID uuid.UUID) (*Student, error) {
	return s.repo.GetStudentByID(ctx, institutionID, studentID)
}

// ListStudents returns a paginated list of students.
func (s *Service) ListStudents(ctx context.Context, institutionID uuid.UUID, f *StudentFilter) ([]*StudentListItem, int64, error) {
	if f.Page < 1 {
		f.Page = 1
	}
	if f.PageSize < 1 || f.PageSize > 100 {
		f.PageSize = 50
	}
	if f.Status == "" {
		f.Status = "active"
	}
	return s.repo.ListStudents(ctx, institutionID, f)
}

// GetStudentTimeline returns the student journey timeline.
func (s *Service) GetStudentTimeline(ctx context.Context, institutionID, studentID uuid.UUID) ([]*TimelineEvent, error) {
	return s.repo.GetTimeline(ctx, institutionID, studentID)
}

// GetStudentParents returns the parents linked to a student.
func (s *Service) GetStudentParents(ctx context.Context, institutionID, studentID uuid.UUID) ([]*ParentGuardian, error) {
	return s.repo.GetStudentParents(ctx, institutionID, studentID)
}

// GetMyChildren returns all students linked to the logged-in parent user.
func (s *Service) GetMyChildren(ctx context.Context, institutionID, parentUserID uuid.UUID) ([]*ChildSummary, error) {
	return s.repo.GetChildrenForParent(ctx, institutionID, parentUserID)
}