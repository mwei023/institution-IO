package communication

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"institution-os/internal/auth"
	"institution-os/pkg/events"
	"institution-os/pkg/response"
	"github.com/rs/zerolog/log"
)

// ─── Service ──────────────────────────────────────────────────────

// Service handles communication business logic.
type Service struct {
	repo      *Repository
	publisher events.Publisher
}

// NewService creates a new communication service.
func NewService(repo *Repository, publisher events.Publisher) *Service {
	return &Service{repo: repo, publisher: publisher}
}

// CreateAnnouncement creates and optionally publishes a new announcement.
func (s *Service) CreateAnnouncement(ctx context.Context, institutionID, authorID uuid.UUID, req *CreateAnnouncementRequest) (*Announcement, error) {
	a := &Announcement{
		ID:            uuid.New(),
		InstitutionID: institutionID,
		Title:         req.Title,
		Body:          req.Body,
		Type:          req.Type,
		Audience:      req.Audience,
		SendSMS:       req.SendSMS,
		SendPush:      req.SendPush,
		IsPublished:   req.Publish,
		CreatedAt:     time.Now(),
	}

	if req.Publish {
		now := time.Now()
		a.PublishedAt = &now
	}

	if req.ExpiresAt != "" {
		if t, err := time.Parse("2006-01-02", req.ExpiresAt); err == nil {
			a.ExpiresAt = &t
		}
	}

	if err := s.repo.CreateAnnouncement(ctx, a); err != nil {
		return nil, err
	}

	if req.Publish {
		evt := events.New(events.EventAnnouncementPublished, institutionID.String())
		evt.EntityType = "announcement"
		evt.EntityID = a.ID.String()
		evt.ActorID = authorID.String()
		_ = s.publisher.Publish(evt)
	}

	return a, nil
}

// ListAnnouncements returns announcements visible to the current user.
func (s *Service) ListAnnouncements(ctx context.Context, institutionID, userID uuid.UUID, f *AnnouncementFilter) ([]*Announcement, int64, error) {
	return s.repo.ListAnnouncements(ctx, institutionID, userID, f)
}

// MarkRead records a read receipt.
func (s *Service) MarkRead(ctx context.Context, institutionID, announcementID, userID uuid.UUID) error {
	return s.repo.MarkAnnouncementRead(ctx, institutionID, announcementID, userID)
}

// SendMessage sends a direct message between users.
func (s *Service) SendMessage(ctx context.Context, institutionID, senderID uuid.UUID, req *SendMessageRequest) (*Message, error) {
	m := &Message{
		ID:            uuid.New(),
		InstitutionID: institutionID,
		SenderID:      senderID,
		RecipientID:   req.RecipientID,
		Body:          req.Body,
		CreatedAt:     time.Now(),
	}

	if err := s.repo.SendMessage(ctx, m); err != nil {
		return nil, err
	}

	evt := events.New(events.EventMessageReceived, institutionID.String())
	evt.EntityType = "message"
	evt.EntityID = m.ID.String()
	evt.ActorID = senderID.String()
	_ = s.publisher.Publish(evt)

	return m, nil
}

// GetThreads returns conversation threads for a user.
func (s *Service) GetThreads(ctx context.Context, institutionID, userID uuid.UUID) ([]*Thread, error) {
	return s.repo.GetThreads(ctx, institutionID, userID)
}

// GetMessages returns messages between two users.
func (s *Service) GetMessages(ctx context.Context, institutionID, userID, otherUserID uuid.UUID, limit int) ([]*Message, error) {
	return s.repo.GetMessages(ctx, institutionID, userID, otherUserID, limit)
}

// ─── Handler ──────────────────────────────────────────────────────

// Handler handles communication HTTP endpoints.
type Handler struct {
	svc *Service
}

// NewHandler creates a new communication handler.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// ListAnnouncements handles GET /api/v1/communications/announcements
func (h *Handler) ListAnnouncements(w http.ResponseWriter, r *http.Request) {
	institutionID := auth.InstitutionIDFromContext(r.Context())
	claims := auth.ClaimsFromContext(r.Context())
	userID, _ := uuid.Parse(claims.UserID)

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))

	f := &AnnouncementFilter{
		Type:     r.URL.Query().Get("type"),
		Page:     page,
		PageSize: pageSize,
	}

	announcements, total, err := h.svc.ListAnnouncements(r.Context(), institutionID, userID, f)
	if err != nil {
		log.Error().Err(err).Msg("list announcements")
		response.InternalError(w)
		return
	}

	response.OkList(w, announcements, &response.Pagination{
		Total:    total,
		Page:     f.Page,
		PageSize: f.PageSize,
		HasNext:  int64(f.Page*f.PageSize) < total,
	})
}

// CreateAnnouncement handles POST /api/v1/communications/announcements
func (h *Handler) CreateAnnouncement(w http.ResponseWriter, r *http.Request) {
	institutionID := auth.InstitutionIDFromContext(r.Context())
	claims := auth.ClaimsFromContext(r.Context())
	authorID, _ := uuid.Parse(claims.UserID)

	var req CreateAnnouncementRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_BODY", "Invalid request body")
		return
	}

	announcement, err := h.svc.CreateAnnouncement(r.Context(), institutionID, authorID, &req)
	if err != nil {
		log.Error().Err(err).Msg("create announcement")
		response.Error(w, http.StatusBadRequest, "CREATE_FAILED", err.Error())
		return
	}

	response.Created(w, announcement)
}

// MarkAnnouncementRead handles POST /api/v1/communications/announcements/{id}/read
func (h *Handler) MarkAnnouncementRead(w http.ResponseWriter, r *http.Request) {
	institutionID := auth.InstitutionIDFromContext(r.Context())
	claims := auth.ClaimsFromContext(r.Context())
	userID, _ := uuid.Parse(claims.UserID)

	announcementID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid announcement ID")
		return
	}

	if err := h.svc.MarkRead(r.Context(), institutionID, announcementID, userID); err != nil {
		response.InternalError(w)
		return
	}
	response.NoContent(w)
}

// GetThreads handles GET /api/v1/communications/messages
func (h *Handler) GetThreads(w http.ResponseWriter, r *http.Request) {
	institutionID := auth.InstitutionIDFromContext(r.Context())
	claims := auth.ClaimsFromContext(r.Context())
	userID, _ := uuid.Parse(claims.UserID)

	threads, err := h.svc.GetThreads(r.Context(), institutionID, userID)
	if err != nil {
		log.Error().Err(err).Msg("get threads")
		response.InternalError(w)
		return
	}
	response.Ok(w, threads)
}

// GetMessages handles GET /api/v1/communications/messages/{user_id}
func (h *Handler) GetMessages(w http.ResponseWriter, r *http.Request) {
	institutionID := auth.InstitutionIDFromContext(r.Context())
	claims := auth.ClaimsFromContext(r.Context())
	myID, _ := uuid.Parse(claims.UserID)

	otherUserID, err := uuid.Parse(chi.URLParam(r, "user_id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid user ID")
		return
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

	messages, err := h.svc.GetMessages(r.Context(), institutionID, myID, otherUserID, limit)
	if err != nil && !errors.Is(err, nil) {
		log.Error().Err(err).Msg("get messages")
		response.InternalError(w)
		return
	}
	response.Ok(w, messages)
}

// SendMessage handles POST /api/v1/communications/messages
func (h *Handler) SendMessage(w http.ResponseWriter, r *http.Request) {
	institutionID := auth.InstitutionIDFromContext(r.Context())
	claims := auth.ClaimsFromContext(r.Context())
	senderID, _ := uuid.Parse(claims.UserID)

	var req SendMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_BODY", "Invalid request body")
		return
	}

	message, err := h.svc.SendMessage(r.Context(), institutionID, senderID, &req)
	if err != nil {
		log.Error().Err(err).Msg("send message")
		response.Error(w, http.StatusBadRequest, "SEND_FAILED", err.Error())
		return
	}
	response.Created(w, message)
}
