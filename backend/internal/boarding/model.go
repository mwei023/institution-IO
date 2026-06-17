// Package boarding provides stub types for the Boarding module.
// Full implementation is planned for Phase 2.
package boarding

import (
	"time"

	"github.com/google/uuid"
)

type Block struct {
	ID            uuid.UUID `json:"id"`
	InstitutionID uuid.UUID `json:"institution_id"`
	Name          string    `json:"name"`
	Gender        string    `json:"gender"`
	MatronID      *uuid.UUID `json:"matron_id,omitempty"`
}

type Room struct {
	ID            uuid.UUID `json:"id"`
	InstitutionID uuid.UUID `json:"institution_id"`
	DormitoryID   uuid.UUID `json:"dormitory_id"`
	RoomNumber    string    `json:"room_number"`
	Capacity      int       `json:"capacity"`
	OccupiedBeds  int       `json:"occupied_beds"`
}

type Bed struct {
	ID            uuid.UUID  `json:"id"`
	InstitutionID uuid.UUID  `json:"institution_id"`
	RoomID        uuid.UUID  `json:"room_id"`
	BedLabel      string     `json:"bed_label"`
	IsOccupied    bool       `json:"is_occupied"`
	StudentID     *uuid.UUID `json:"student_id,omitempty"`
}

type NightAttendance struct {
	ID            uuid.UUID `json:"id"`
	InstitutionID uuid.UUID `json:"institution_id"`
	DormitoryID   uuid.UUID `json:"dormitory_id"`
	Date          time.Time `json:"date"`
	TakenBy       uuid.UUID `json:"taken_by"`
}

const StubNotice = "Boarding module is stubbed. Full implementation in Phase 2."
