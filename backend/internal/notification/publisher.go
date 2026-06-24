package notification

import (
	"context"

	"institution-os/pkg/events"
)

// EventPublisher wraps a notification Service so it can be used directly
// as an events.Publisher. Publish() calls HandleEvent synchronously.
//
// This is intentionally simple for Phase 1 — single process, no queue,
// no retry, no durability. If notification dispatch fails or is slow,
// it happens inline with whatever request published the event. This is
// acceptable for current volume but should be replaced with a real
// queue (Redis Streams, etc.) before this matters at scale.
type EventPublisher struct {
	notificationSvc *Service
}

// NewEventPublisher creates a publisher that dispatches events to the
// notification service synchronously.
func NewEventPublisher(notificationSvc *Service) *EventPublisher {
	return &EventPublisher{notificationSvc: notificationSvc}
}

// Publish implements events.Publisher.
func (p *EventPublisher) Publish(event events.Event) error {
	ctx := context.Background()
	p.notificationSvc.HandleEvent(ctx, event)
	return nil
}

var _ events.Publisher = (*EventPublisher)(nil)