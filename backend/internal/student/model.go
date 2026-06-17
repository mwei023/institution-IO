package student

import (
	"time"

	"github.com/google/uuid"
)

// Student is the full student profile.
type Student struct {
	ID                      uuid.UUID  `json:"id"`
	InstitutionID           uuid.UUID  `json:"institution_id"`
	AdmissionNumber         string     `json:"admission_number"`
	FirstName               string     `json:"first_name"`
	LastName                string     `json:"last_name"`
	OtherNames              *string    `json:"other_names,omitempty"`
	DateOfBirth             time.Time  `json:"date_of_birth"`
	Gender                  string     `json:"gender"`
	BirthCertificateNumber  *string    `json:"birth_certificate_number,omitempty"`
	Nationality             string     `json:"nationality"`
	BloodGroup              *string    `json:"blood_group,omitempty"`
	HomeAddress             *string    `json:"home_address,omitempty"`
	PhotoURL                *string    `json:"photo_url,omitempty"`
	CurrentClassID          *uuid.UUID `json:"current_class_id,omitempty"`
	AdmissionDate           time.Time  `json:"admission_date"`
	KCPEIndexNumber         *string    `json:"kcpe_index_number,omitempty"`
	KCPEMarks               *int       `json:"kcpe_marks,omitempty"`
	KNECRegistrationNumber  *string    `json:"knec_registration_number,omitempty"`
	Status                  string     `json:"status"`
	IsBoarder               bool       `json:"is_boarder"`
	UsesTransport           bool       `json:"uses_transport"`
	MedicalConditions       *string    `json:"medical_conditions,omitempty"`
	Allergies               *string    `json:"allergies,omitempty"`
	CreatedAt               time.Time  `json:"created_at"`
	UpdatedAt               time.Time  `json:"updated_at"`
}

// StudentListItem is a compact student representation for list views.
type StudentListItem struct {
	ID              uuid.UUID `json:"id"`
	AdmissionNumber string    `json:"admission_number"`
	FullName        string    `json:"full_name"`
	Gender          string    `json:"gender"`
	ClassName       *string   `json:"class_name,omitempty"`
	Status          string    `json:"status"`
	PhotoURL        *string   `json:"photo_url,omitempty"`
	IsBoarder       bool      `json:"is_boarder"`
}

// Admission is a pending admission application.
type Admission struct {
	ID                uuid.UUID  `json:"id"`
	InstitutionID     uuid.UUID  `json:"institution_id"`
	ApplicationNumber string     `json:"application_number"`
	AcademicYear      string     `json:"academic_year"`
	IntendedClass     string     `json:"intended_class"`
	FirstName         string     `json:"first_name"`
	LastName          string     `json:"last_name"`
	DateOfBirth       time.Time  `json:"date_of_birth"`
	Gender            string     `json:"gender"`
	Status            string     `json:"status"`
	ReviewedBy        *uuid.UUID `json:"reviewed_by,omitempty"`
	ReviewedAt        *time.Time `json:"reviewed_at,omitempty"`
	ReviewNotes       *string    `json:"review_notes,omitempty"`
	StudentID         *uuid.UUID `json:"student_id,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
}

// CreateAdmissionRequest is the request body to create a new admission application.
type CreateAdmissionRequest struct {
	AcademicYear           string `json:"academic_year" validate:"required"`
	IntendedClass          string `json:"intended_class" validate:"required"`
	FirstName              string `json:"first_name" validate:"required,max=100"`
	LastName               string `json:"last_name" validate:"required,max=100"`
	OtherNames             string `json:"other_names"`
	DateOfBirth            string `json:"date_of_birth" validate:"required"` // YYYY-MM-DD
	Gender                 string `json:"gender" validate:"required,oneof=M F O"`
	BirthCertificateNumber string `json:"birth_certificate_number"`
	Nationality            string `json:"nationality"`
	PreviousSchool         string `json:"previous_school"`
	BloodGroup             string `json:"blood_group"`
	MedicalConditions      string `json:"medical_conditions"`
	Allergies              string `json:"allergies"`
	SpecialNeeds           bool   `json:"special_needs"`
	SpecialNeedsDetails    string `json:"special_needs_details"`
	BoardingRequired       bool   `json:"boarding_required"`
	TransportRequired      bool   `json:"transport_required"`
	// Parent details
	ParentName         string `json:"parent_name" validate:"required"`
	ParentRelationship string `json:"parent_relationship" validate:"required"`
	ParentPhone        string `json:"parent_phone" validate:"required"`
	ParentEmail        string `json:"parent_email"`
}

// ApproveAdmissionRequest is the body to approve/reject an admission.
type ApproveAdmissionRequest struct {
	Action string `json:"action" validate:"required,oneof=approve reject"`
	Notes  string `json:"notes"`
}

// EnrollStudentRequest triggers enrollment after admission fee is paid.
type EnrollStudentRequest struct {
	ClassID   uuid.UUID `json:"class_id" validate:"required"`
	BedID     *uuid.UUID `json:"bed_id,omitempty"`
	RouteID   *uuid.UUID `json:"route_id,omitempty"`
}

// StudentFilter defines query parameters for listing students.
type StudentFilter struct {
	Status   string
	ClassID  *uuid.UUID
	Search   string
	Page     int
	PageSize int
}

// ParentGuardian is a parent/guardian record.
type ParentGuardian struct {
	ID               uuid.UUID `json:"id"`
	InstitutionID    uuid.UUID `json:"institution_id"`
	FirstName        string    `json:"first_name"`
	LastName         string    `json:"last_name"`
	Relationship     string    `json:"relationship"`
	Phone            string    `json:"phone"`
	Email            *string   `json:"email,omitempty"`
	Occupation       *string   `json:"occupation,omitempty"`
	IsPrimaryContact bool      `json:"is_primary_contact"`
	IsFeePayer       bool      `json:"is_fee_payer"`
}

// TimelineEvent is a student journey timeline event.
type TimelineEvent struct {
	ID          uuid.UUID   `json:"id"`
	EventType   string      `json:"event_type"`
	Title       string      `json:"title"`
	Description *string     `json:"description,omitempty"`
	Metadata    interface{} `json:"metadata,omitempty"`
	EventDate   time.Time   `json:"event_date"`
	CreatedAt   time.Time   `json:"created_at"`
}
