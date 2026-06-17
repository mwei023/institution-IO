package events

import (
	"time"

	"github.com/google/uuid"
)

// EventType is a domain event type constant.
type EventType string

const (
	// Student events
	EventStudentEnrolled    EventType = "student.enrolled"
	EventStudentExited      EventType = "student.exited"
	EventStudentPromoted    EventType = "student.promoted"
	EventStudentAbsent      EventType = "student.absent"

	// Admission events
	EventAdmissionCreated  EventType = "admission.created"
	EventAdmissionApproved EventType = "admission.approved"
	EventAdmissionRejected EventType = "admission.rejected"

	// Finance events
	EventPaymentReceived   EventType = "payment.received"
	EventInvoiceGenerated  EventType = "invoice.generated"
	EventFeeOverdue        EventType = "fee.overdue"

	// Academic events
	EventReportCardPublished EventType = "report_card.published"
	EventAssignmentDue       EventType = "assignment.due"
	EventTimetablePublished  EventType = "timetable.published"

	// Health events
	EventClinicVisit         EventType = "clinic.visit"
	EventMedicalEmergency    EventType = "clinic.emergency"
	EventIsolationAdmission  EventType = "clinic.isolation_admitted"

	// Communication events
	EventAnnouncementPublished EventType = "announcement.published"
	EventMessageReceived       EventType = "message.received"

	// Security events
	EventEmergencyBroadcast EventType = "security.emergency"
	EventUnauthorizedExit   EventType = "security.unauthorized_exit"

	// User events
	EventUserLoggedIn  EventType = "user.login"
	EventUserLoggedOut EventType = "user.logout"
)

// Event is a domain event published by any module.
type Event struct {
	ID            string      `json:"id"`
	Type          EventType   `json:"type"`
	InstitutionID string      `json:"institution_id"`
	ActorID       string      `json:"actor_id,omitempty"` // user who triggered it
	EntityType    string      `json:"entity_type,omitempty"`
	EntityID      string      `json:"entity_id,omitempty"`
	Payload       interface{} `json:"payload,omitempty"`
	OccurredAt    time.Time   `json:"occurred_at"`
}

// New creates a new Event with a generated ID and current timestamp.
func New(eventType EventType, institutionID string) Event {
	return Event{
		ID:            uuid.New().String(),
		Type:          eventType,
		InstitutionID: institutionID,
		OccurredAt:    time.Now().UTC(),
	}
}

// Publisher is the interface for publishing domain events.
type Publisher interface {
	Publish(event Event) error
}

// NoopPublisher discards all events (used in tests).
type NoopPublisher struct{}

func (n *NoopPublisher) Publish(_ Event) error { return nil }
