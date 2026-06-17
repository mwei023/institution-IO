package communication

import (
	"time"

	"github.com/google/uuid"
)

// Announcement is a school-wide or targeted notice.
type Announcement struct {
	ID            uuid.UUID  `json:"id"`
	InstitutionID uuid.UUID  `json:"institution_id"`
	Title         string     `json:"title"`
	Body          string     `json:"body"`
	Type          string     `json:"type"`     // general|academic|event|emergency|fee_notice
	Audience      string     `json:"audience"` // all|parents|teachers|students|staff|class:{id}
	SendSMS       bool       `json:"send_sms"`
	SendPush      bool       `json:"send_push"`
	IsPublished   bool       `json:"is_published"`
	PublishedAt   *time.Time `json:"published_at,omitempty"`
	ExpiresAt     *time.Time `json:"expires_at,omitempty"`
	ReadCount     int        `json:"read_count"`
	AuthorName    string     `json:"author_name"` // joined
	IsRead        bool       `json:"is_read"`     // per-user, computed in list
	CreatedAt     time.Time  `json:"created_at"`
}

// CreateAnnouncementRequest is the body to create a new announcement.
type CreateAnnouncementRequest struct {
	Title     string `json:"title" validate:"required,max=255"`
	Body      string `json:"body" validate:"required"`
	Type      string `json:"type" validate:"required,oneof=general academic event emergency fee_notice"`
	Audience  string `json:"audience" validate:"required"`
	SendSMS   bool   `json:"send_sms"`
	SendPush  bool   `json:"send_push"`
	Publish   bool   `json:"publish"` // publish immediately vs save as draft
	ExpiresAt string `json:"expires_at,omitempty"` // YYYY-MM-DD
}

// Message is a direct message between two users.
type Message struct {
	ID          uuid.UUID  `json:"id"`
	InstitutionID uuid.UUID `json:"institution_id"`
	SenderID    uuid.UUID  `json:"sender_id"`
	SenderName  string     `json:"sender_name"`  // joined
	RecipientID uuid.UUID  `json:"recipient_id"`
	Body        string     `json:"body"`
	IsRead      bool       `json:"is_read"`
	ReadAt      *time.Time `json:"read_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

// SendMessageRequest is the body to send a direct message.
type SendMessageRequest struct {
	RecipientID uuid.UUID `json:"recipient_id" validate:"required"`
	Body        string    `json:"body" validate:"required,min=1,max=2000"`
}

// Thread is a conversation between two users (grouped messages).
type Thread struct {
	OtherUserID   uuid.UUID `json:"other_user_id"`
	OtherUserName string    `json:"other_user_name"`
	OtherUserRole string    `json:"other_user_role"`
	LastMessage   string    `json:"last_message"`
	LastMessageAt time.Time `json:"last_message_at"`
	UnreadCount   int       `json:"unread_count"`
}

// AnnouncementFilter holds query params for listing announcements.
type AnnouncementFilter struct {
	Type     string
	Audience string
	Page     int
	PageSize int
}
