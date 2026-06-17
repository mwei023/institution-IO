// Package clinic provides stub types for the Clinic & Health module.
// Full implementation planned for Phase 2.
package clinic

import (
	"time"

	"github.com/google/uuid"
)

type ClinicVisit struct {
	ID              uuid.UUID  `json:"id"`
	InstitutionID   uuid.UUID  `json:"institution_id"`
	StudentID       uuid.UUID  `json:"student_id"`
	NurseID         uuid.UUID  `json:"nurse_id"`
	VisitDate       time.Time  `json:"visit_date"`
	Complaint       string     `json:"complaint"`
	Diagnosis       *string    `json:"diagnosis,omitempty"`
	TreatmentGiven  *string    `json:"treatment_given,omitempty"`
	IsEmergency     bool       `json:"is_emergency"`
	ReferredOut     bool       `json:"referred_out"`
	FollowUpDate    *time.Time `json:"follow_up_date,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
}

type IsolationAdmission struct {
	ID              uuid.UUID  `json:"id"`
	InstitutionID   uuid.UUID  `json:"institution_id"`
	StudentID       uuid.UUID  `json:"student_id"`
	AdmissionDate   time.Time  `json:"admission_date"`
	AdmissionReason string     `json:"admission_reason"`
	DischargeDate   *time.Time `json:"discharge_date,omitempty"`
	DischargeNotes  *string    `json:"discharge_notes,omitempty"`
}

const StubNotice = "Clinic module is stubbed. Full implementation in Phase 2."
