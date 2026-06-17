package notification

import (
	"context"
	"fmt"

	"institution-os/pkg/events"
	"github.com/rs/zerolog/log"
)

// Channel represents a notification delivery channel.
type Channel string

const (
	ChannelInApp    Channel = "in_app"
	ChannelPush     Channel = "push"
	ChannelSMS      Channel = "sms"
	ChannelEmail    Channel = "email"
	ChannelWhatsApp Channel = "whatsapp"
)

// Message is a notification to be delivered.
type Message struct {
	RecipientID   string
	InstitutionID string
	Channel       Channel
	Title         string
	Body          string
	Data          map[string]string
}

// SMSProvider is the interface for SMS delivery.
type SMSProvider interface {
	Send(ctx context.Context, to, message string) error
}

// EmailProvider is the interface for email delivery.
type EmailProvider interface {
	Send(ctx context.Context, to, subject, body string) error
}

// ConsoleSMSProvider logs SMS to stdout (Phase 1 stub).
type ConsoleSMSProvider struct{}

func (p *ConsoleSMSProvider) Send(ctx context.Context, to, message string) error {
	log.Info().
		Str("channel", "sms").
		Str("to", to).
		Str("message", message).
		Msg("SMS_STUB: would send SMS")
	return nil
}

// ConsoleEmailProvider logs emails to stdout (Phase 1 stub).
type ConsoleEmailProvider struct{}

func (p *ConsoleEmailProvider) Send(ctx context.Context, to, subject, body string) error {
	log.Info().
		Str("channel", "email").
		Str("to", to).
		Str("subject", subject).
		Msg("EMAIL_STUB: would send email")
	return nil
}

// Service dispatches notifications via configured providers.
type Service struct {
	sms   SMSProvider
	email EmailProvider
}

// NewService creates a new notification service with stub providers.
func NewService() *Service {
	return &Service{
		sms:   &ConsoleSMSProvider{},
		email: &ConsoleEmailProvider{},
	}
}

// NewServiceWithProviders creates a notification service with real providers.
func NewServiceWithProviders(sms SMSProvider, email EmailProvider) *Service {
	return &Service{sms: sms, email: email}
}

// Send dispatches a notification via the specified channel.
func (s *Service) Send(ctx context.Context, msg *Message) error {
	switch msg.Channel {
	case ChannelSMS:
		return s.sms.Send(ctx, msg.RecipientID, msg.Body)
	case ChannelEmail:
		return s.email.Send(ctx, msg.RecipientID, msg.Title, msg.Body)
	case ChannelInApp, ChannelPush:
		// Phase 1: log to console; Phase 2: real push via Firebase / web push
		log.Info().
			Str("channel", string(msg.Channel)).
			Str("recipient_id", msg.RecipientID).
			Str("title", msg.Title).
			Str("body", msg.Body).
			Msg("PUSH_STUB: would send push notification")
		return nil
	default:
		return fmt.Errorf("unknown notification channel: %s", msg.Channel)
	}
}

// HandleEvent processes a domain event and dispatches relevant notifications.
// This is the event → notification mapping layer.
func (s *Service) HandleEvent(ctx context.Context, evt events.Event) {
	switch evt.Type {
	case events.EventStudentAbsent:
		// SMS to parent: "Your child was absent today"
		if payload, ok := evt.Payload.(map[string]interface{}); ok {
			parentPhone, _ := payload["parent_phone"].(string)
			studentName, _ := payload["student_name"].(string)
			if parentPhone != "" {
				_ = s.sms.Send(ctx, parentPhone,
					fmt.Sprintf("Dear Parent, %s was absent from school today. Please contact the school if this is unexpected.", studentName))
			}
		}

	case events.EventPaymentReceived:
		// SMS to parent: "Payment received, receipt number..."
		if payload, ok := evt.Payload.(map[string]interface{}); ok {
			parentPhone, _ := payload["parent_phone"].(string)
			amount, _ := payload["amount"].(float64)
			receiptNumber, _ := payload["receipt_number"].(string)
			if parentPhone != "" {
				_ = s.sms.Send(ctx, parentPhone,
					fmt.Sprintf("Payment of KES %.0f received. Receipt: %s. Thank you.", amount, receiptNumber))
			}
		}

	case events.EventMedicalEmergency:
		// SMS to parent + push to principal
		if payload, ok := evt.Payload.(map[string]interface{}); ok {
			parentPhone, _ := payload["parent_phone"].(string)
			studentName, _ := payload["student_name"].(string)
			condition, _ := payload["condition"].(string)
			if parentPhone != "" {
				_ = s.sms.Send(ctx, parentPhone,
					fmt.Sprintf("URGENT: %s requires medical attention at school. Condition: %s. Please call the school immediately.", studentName, condition))
			}
		}

	case events.EventAdmissionApproved:
		// Email to parent: "Application approved"
		if payload, ok := evt.Payload.(map[string]interface{}); ok {
			parentEmail, _ := payload["parent_email"].(string)
			studentName, _ := payload["student_name"].(string)
			if parentEmail != "" {
				_ = s.email.Send(ctx, parentEmail,
					"Application Approved",
					fmt.Sprintf("Dear Parent, we are pleased to inform you that the application for %s has been approved. Please log in to your portal to complete enrollment.", studentName))
			}
		}

	case events.EventReportCardPublished:
		// Push notification to parent
		if payload, ok := evt.Payload.(map[string]interface{}); ok {
			parentID, _ := payload["parent_id"].(string)
			studentName, _ := payload["student_name"].(string)
			if parentID != "" {
				_ = s.Send(ctx, &Message{
					RecipientID:   parentID,
					InstitutionID: evt.InstitutionID,
					Channel:       ChannelPush,
					Title:         "Report Card Available",
					Body:          fmt.Sprintf("%s's report card is now available.", studentName),
				})
			}
		}

	default:
		// Log unhandled events in debug mode
		log.Debug().
			Str("event_type", string(evt.Type)).
			Msg("notification: no handler for event type")
	}
}
