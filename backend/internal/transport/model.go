// Package transport provides stub types for the Transport module.
// Full implementation planned for Phase 2.
package transport

import (
	"time"

	"github.com/google/uuid"
)

type Route struct {
	ID                    uuid.UUID `json:"id"`
	InstitutionID         uuid.UUID `json:"institution_id"`
	Name                  string    `json:"name"`
	EstimatedDurationMin  int       `json:"estimated_duration_min"`
	IsActive              bool      `json:"is_active"`
}

type Vehicle struct {
	ID                 uuid.UUID  `json:"id"`
	InstitutionID      uuid.UUID  `json:"institution_id"`
	RegistrationPlate  string     `json:"registration_plate"`
	Make               string     `json:"make"`
	Model              string     `json:"model"`
	Capacity           int        `json:"capacity"`
	InsuranceExpiry    *time.Time `json:"insurance_expiry,omitempty"`
	IsActive           bool       `json:"is_active"`
}

type Trip struct {
	ID            uuid.UUID  `json:"id"`
	InstitutionID uuid.UUID  `json:"institution_id"`
	RouteID       uuid.UUID  `json:"route_id"`
	VehicleID     uuid.UUID  `json:"vehicle_id"`
	DriverID      uuid.UUID  `json:"driver_id"`
	TripDate      time.Time  `json:"trip_date"`
	TripType      string     `json:"trip_type"` // AM|PM
	DepartureTime *time.Time `json:"departure_time,omitempty"`
	ArrivalTime   *time.Time `json:"arrival_time,omitempty"`
}

const StubNotice = "Transport module is stubbed. Full implementation in Phase 2."

// ─────────────────────────────────────────────────────────────────────────────

// Package library provides stub types for the Library module.
// (Separate file in real implementation, combined here for brevity)

type Book struct {
	ID            uuid.UUID `json:"id"`
	InstitutionID uuid.UUID `json:"institution_id"`
	ISBN          *string   `json:"isbn,omitempty"`
	Title         string    `json:"title"`
	Authors       string    `json:"authors"`
	Publisher     *string   `json:"publisher,omitempty"`
	PublishYear   *int      `json:"publish_year,omitempty"`
	CallNumber    *string   `json:"call_number,omitempty"`
	TotalCopies   int       `json:"total_copies"`
	AvailableCopies int     `json:"available_copies"`
	IsDigital     bool      `json:"is_digital"`
	DigitalURL    *string   `json:"digital_url,omitempty"`
}

type Borrowing struct {
	ID            uuid.UUID  `json:"id"`
	InstitutionID uuid.UUID  `json:"institution_id"`
	CopyID        uuid.UUID  `json:"copy_id"`
	BorrowerID    uuid.UUID  `json:"borrower_id"`
	IssuedAt      time.Time  `json:"issued_at"`
	DueDate       time.Time  `json:"due_date"`
	ReturnedAt    *time.Time `json:"returned_at,omitempty"`
	FineAmount    float64    `json:"fine_amount"`
	FinePaid      bool       `json:"fine_paid"`
}
