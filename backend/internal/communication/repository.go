package communication

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"institution-os/pkg/db"
	"github.com/jackc/pgx/v5"
)

var ErrAnnouncementNotFound = errors.New("announcement not found")

// Repository handles database access for communication.
type Repository struct {
	db *db.Pool
}

// NewRepository creates a new communication repository.
func NewRepository(db *db.Pool) *Repository {
	return &Repository{db: db}
}

// CreateAnnouncement inserts a new announcement.
func (r *Repository) CreateAnnouncement(ctx context.Context, a *Announcement) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO announcements (
			id, institution_id, title, body, type, audience,
			send_sms, send_push, is_published, published_at, expires_at,
			read_count, created_by, created_at, updated_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,0,$12,NOW(),NOW())`,
		a.ID, a.InstitutionID, a.Title, a.Body, a.Type, a.Audience,
		a.SendSMS, a.SendPush, a.IsPublished, a.PublishedAt, a.ExpiresAt,
		nil, // created_by set from context in service
	)
	return err
}

// GetAnnouncementByID retrieves a single announcement.
func (r *Repository) GetAnnouncementByID(ctx context.Context, institutionID, announcementID uuid.UUID) (*Announcement, error) {
	var a Announcement
	err := r.db.QueryRow(ctx, `
		SELECT a.id, a.institution_id, a.title, a.body, a.type, a.audience,
		       a.send_sms, a.send_push, a.is_published, a.published_at, a.expires_at,
		       a.read_count, u.first_name || ' ' || u.last_name, a.created_at
		FROM announcements a
		LEFT JOIN users u ON u.id = a.created_by
		WHERE a.institution_id = $1 AND a.id = $2 AND a.deleted_at IS NULL`,
		institutionID, announcementID,
	).Scan(
		&a.ID, &a.InstitutionID, &a.Title, &a.Body, &a.Type, &a.Audience,
		&a.SendSMS, &a.SendPush, &a.IsPublished, &a.PublishedAt, &a.ExpiresAt,
		&a.ReadCount, &a.AuthorName, &a.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrAnnouncementNotFound
		}
		return nil, err
	}
	return &a, nil
}

// ListAnnouncements returns a paginated list of announcements for a user.
func (r *Repository) ListAnnouncements(ctx context.Context, institutionID, userID uuid.UUID, f *AnnouncementFilter) ([]*Announcement, int64, error) {
	if f.Page < 1 {
		f.Page = 1
	}
	if f.PageSize < 1 {
		f.PageSize = 25
	}
	offset := (f.Page - 1) * f.PageSize

	var total int64
	err := r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM announcements
		WHERE institution_id = $1
		  AND deleted_at IS NULL
		  AND is_published = true
		  AND ($2 = '' OR type = $2)`,
		institutionID, f.Type,
	).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	rows, err := r.db.Query(ctx, `
		SELECT a.id, a.institution_id, a.title, a.body, a.type, a.audience,
		       a.send_sms, a.send_push, a.is_published, a.published_at, a.expires_at,
		       a.read_count,
		       COALESCE(u.first_name || ' ' || u.last_name, 'System'),
		       a.created_at,
		       (SELECT COUNT(*) FROM announcement_reads ar
		        WHERE ar.announcement_id = a.id AND ar.user_id = $2) > 0 AS is_read
		FROM announcements a
		LEFT JOIN users u ON u.id = a.created_by
		WHERE a.institution_id = $1
		  AND a.deleted_at IS NULL
		  AND a.is_published = true
		  AND ($3 = '' OR a.type = $3)
		ORDER BY a.published_at DESC
		LIMIT $4 OFFSET $5`,
		institutionID, userID, f.Type, f.PageSize, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var announcements []*Announcement
	for rows.Next() {
		var a Announcement
		if err := rows.Scan(
			&a.ID, &a.InstitutionID, &a.Title, &a.Body, &a.Type, &a.Audience,
			&a.SendSMS, &a.SendPush, &a.IsPublished, &a.PublishedAt, &a.ExpiresAt,
			&a.ReadCount, &a.AuthorName, &a.CreatedAt, &a.IsRead,
		); err != nil {
			return nil, 0, err
		}
		announcements = append(announcements, &a)
	}
	return announcements, total, rows.Err()
}

// MarkAnnouncementRead records that a user has read an announcement.
func (r *Repository) MarkAnnouncementRead(ctx context.Context, institutionID, announcementID, userID uuid.UUID) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO announcement_reads (id, institution_id, announcement_id, user_id, read_at)
		VALUES (gen_random_uuid(), $1, $2, $3, NOW())
		ON CONFLICT (institution_id, announcement_id, user_id) DO NOTHING`,
		institutionID, announcementID, userID,
	)
	if err != nil {
		return err
	}
	// Increment read count
	_, err = r.db.Exec(ctx, `
		UPDATE announcements SET read_count = read_count + 1
		WHERE id = $1 AND institution_id = $2`,
		announcementID, institutionID,
	)
	return err
}

// PublishAnnouncement sets is_published = true.
func (r *Repository) PublishAnnouncement(ctx context.Context, institutionID, announcementID uuid.UUID) error {
	now := time.Now()
	_, err := r.db.Exec(ctx, `
		UPDATE announcements
		SET is_published = true, published_at = $1, updated_at = NOW()
		WHERE institution_id = $2 AND id = $3 AND deleted_at IS NULL`,
		now, institutionID, announcementID,
	)
	return err
}

// ─── Messages ─────────────────────────────────────────────────────

// SendMessage inserts a new direct message.
func (r *Repository) SendMessage(ctx context.Context, m *Message) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO messages (id, institution_id, sender_id, recipient_id, body, is_read, created_at)
		VALUES ($1,$2,$3,$4,$5,false,NOW())`,
		m.ID, m.InstitutionID, m.SenderID, m.RecipientID, m.Body,
	)
	return err
}

// GetThreads returns the list of conversation threads for a user.
func (r *Repository) GetThreads(ctx context.Context, institutionID, userID uuid.UUID) ([]*Thread, error) {
	rows, err := r.db.Query(ctx, `
		WITH latest AS (
			SELECT DISTINCT ON (
				LEAST(sender_id::text, recipient_id::text) || '|' || GREATEST(sender_id::text, recipient_id::text)
			)
			id, sender_id, recipient_id, body, created_at,
			LEAST(sender_id::text, recipient_id::text) || '|' || GREATEST(sender_id::text, recipient_id::text) AS thread_key
			FROM messages
			WHERE institution_id = $1 AND (sender_id = $2 OR recipient_id = $2) AND deleted_at IS NULL
			ORDER BY thread_key, created_at DESC
		)
		SELECT
			CASE WHEN l.sender_id = $2 THEN l.recipient_id ELSE l.sender_id END AS other_user_id,
			u.first_name || ' ' || u.last_name AS other_user_name,
			ro.slug AS other_user_role,
			l.body AS last_message,
			l.created_at AS last_message_at,
			COUNT(m2.id) AS unread_count
		FROM latest l
		JOIN users u ON u.id = CASE WHEN l.sender_id = $2 THEN l.recipient_id ELSE l.sender_id END
		JOIN roles ro ON ro.id = u.role_id
		LEFT JOIN messages m2 ON m2.institution_id = $1
			AND m2.recipient_id = $2
			AND m2.sender_id = CASE WHEN l.sender_id = $2 THEN l.recipient_id ELSE l.sender_id END
			AND m2.is_read = false
			AND m2.deleted_at IS NULL
		GROUP BY other_user_id, other_user_name, other_user_role, l.body, l.created_at
		ORDER BY l.created_at DESC`,
		institutionID, userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var threads []*Thread
	for rows.Next() {
		var t Thread
		if err := rows.Scan(
			&t.OtherUserID, &t.OtherUserName, &t.OtherUserRole,
			&t.LastMessage, &t.LastMessageAt, &t.UnreadCount,
		); err != nil {
			return nil, err
		}
		threads = append(threads, &t)
	}
	return threads, rows.Err()
}

// GetMessages returns messages between two users.
func (r *Repository) GetMessages(ctx context.Context, institutionID, userID, otherUserID uuid.UUID, limit int) ([]*Message, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	rows, err := r.db.Query(ctx, `
		SELECT m.id, m.institution_id, m.sender_id,
		       u.first_name || ' ' || u.last_name,
		       m.recipient_id, m.body, m.is_read, m.read_at, m.created_at
		FROM messages m
		JOIN users u ON u.id = m.sender_id
		WHERE m.institution_id = $1
		  AND ((m.sender_id = $2 AND m.recipient_id = $3) OR (m.sender_id = $3 AND m.recipient_id = $2))
		  AND m.deleted_at IS NULL
		ORDER BY m.created_at DESC
		LIMIT $4`,
		institutionID, userID, otherUserID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []*Message
	for rows.Next() {
		var m Message
		if err := rows.Scan(
			&m.ID, &m.InstitutionID, &m.SenderID, &m.SenderName,
			&m.RecipientID, &m.Body, &m.IsRead, &m.ReadAt, &m.CreatedAt,
		); err != nil {
			return nil, err
		}
		messages = append(messages, &m)
	}

	// Mark messages to this user as read
	_, _ = r.db.Exec(ctx, `
		UPDATE messages SET is_read = true, read_at = NOW()
		WHERE institution_id = $1 AND sender_id = $2 AND recipient_id = $3
		  AND is_read = false AND deleted_at IS NULL`,
		institutionID, otherUserID, userID,
	)

	if err := rows.Err(); err != nil {
    return nil, fmt.Errorf("iterate messages: %w", err)
}
return messages, nil
}
