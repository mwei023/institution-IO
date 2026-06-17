// Package hr provides stub types for the Human Resources module.
// Full implementation is planned for Phase 2.
// All database tables are already defined in migrations/000006+.
package hr

import (
	"time"

	"github.com/google/uuid"
)

// StaffRecord is the full HR profile for a staff member.
type StaffRecord struct {
	ID              uuid.UUID  `json:"id"`
	InstitutionID   uuid.UUID  `json:"institution_id"`
	UserID          uuid.UUID  `json:"user_id"`
	StaffNumber     string     `json:"staff_number"`
	TSCNumber       *string    `json:"tsc_number,omitempty"`
	DepartmentID    *uuid.UUID `json:"department_id,omitempty"`
	JobTitle        string     `json:"job_title"`
	EmploymentType  string     `json:"employment_type"`
	EmploymentStart time.Time  `json:"employment_start"`
	ProbationEnd    *time.Time `json:"probation_end,omitempty"`
	ExitDate        *time.Time `json:"exit_date,omitempty"`
	ExitType        *string    `json:"exit_type,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
}

// LeaveRequest is a staff leave application.
type LeaveRequest struct {
	ID            uuid.UUID  `json:"id"`
	InstitutionID uuid.UUID  `json:"institution_id"`
	StaffID       uuid.UUID  `json:"staff_id"`
	LeaveType     string     `json:"leave_type"`
	StartDate     time.Time  `json:"start_date"`
	EndDate       time.Time  `json:"end_date"`
	DaysRequested float64    `json:"days_requested"`
	Reason        string     `json:"reason"`
	Status        string     `json:"status"` // pending|approved|rejected|cancelled
	ReviewedBy    *uuid.UUID `json:"reviewed_by,omitempty"`
	ReviewedAt    *time.Time `json:"reviewed_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
}

// PerformanceReview is a staff appraisal record.
type PerformanceReview struct {
	ID            uuid.UUID `json:"id"`
	InstitutionID uuid.UUID `json:"institution_id"`
	StaffID       uuid.UUID `json:"staff_id"`
	ReviewerID    uuid.UUID `json:"reviewer_id"`
	ReviewPeriod  string    `json:"review_period"` // e.g. 2024-annual
	ReviewType    string    `json:"review_type"`   // mid_year|annual
	SelfScore     *float64  `json:"self_score,omitempty"`
	ReviewerScore *float64  `json:"reviewer_score,omitempty"`
	FinalScore    *float64  `json:"final_score,omitempty"`
	Comments      string    `json:"comments"`
	Status        string    `json:"status"` // draft|submitted|completed
	CreatedAt     time.Time `json:"created_at"`
}

// StubNotice marks this package as a Phase 2 stub.
const StubNotice = "HR module is stubbed. Full implementation in Phase 2."
