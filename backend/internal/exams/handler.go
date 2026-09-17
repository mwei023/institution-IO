package exams

import (
	"encoding/json"
	"errors"
	"net/http"

	"institution-os/internal/auth"
	"institution-os/pkg/response"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func instID(r *http.Request) uuid.UUID { return auth.InstitutionIDFromContext(r.Context()) }

func userID(r *http.Request) *uuid.UUID {
	claims := auth.ClaimsFromContext(r.Context())
	if claims == nil {
		return nil
	}
	id, err := uuid.Parse(claims.UserID)
	if err != nil {
		return nil
	}
	return &id
}

func urlID(r *http.Request, key string) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, key))
	if err != nil {
		return uuid.Nil, false
	}
	return id, true
}

func decode(w http.ResponseWriter, r *http.Request, v interface{}) bool {
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_BODY", "Invalid request body")
		return false
	}
	return true
}

func examErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrSessionNotFound):
		response.NotFound(w, "EXAM_SESSION")
	case errors.Is(err, ErrPaperNotFound):
		response.NotFound(w, "EXAM_PAPER")
	case errors.Is(err, ErrSittingNotFound):
		response.NotFound(w, "EXAM_SITTING")
	case errors.Is(err, ErrSessionPublished):
		response.Error(w, http.StatusConflict, "PUBLISHED_IMMUTABLE", err.Error())
	case errors.Is(err, ErrBlockingConflicts):
		response.Error(w, http.StatusConflict, "BLOCKING_CONFLICTS", err.Error())
	case errors.Is(err, ErrInvalidSession), errors.Is(err, ErrInvalidPaper),
		errors.Is(err, ErrInvalidGroup), errors.Is(err, ErrClassNotFound),
		errors.Is(err, ErrOfferingNotFound), errors.Is(err, ErrStaffNotFound),
		errors.Is(err, ErrInvalidArrangement):
		response.Error(w, http.StatusBadRequest, "EXAM_ERROR", err.Error())
	default:
		log.Error().Err(err).Msg("exams handler")
		response.Error(w, http.StatusBadRequest, "EXAM_ERROR", err.Error())
	}
}

// ── Sessions ──

func (h *Handler) CreateSession(w http.ResponseWriter, r *http.Request) {
	var req CreateSessionRequest
	if !decode(w, r, &req) {
		return
	}
	s, err := h.svc.CreateSession(r.Context(), instID(r), userID(r), &req)
	if err != nil {
		examErr(w, err)
		return
	}
	response.Created(w, s)
}

func (h *Handler) ListSessions(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.ListSessions(r.Context(), instID(r))
	if err != nil {
		response.InternalError(w)
		return
	}
	response.Ok(w, list)
}

func (h *Handler) GetSession(w http.ResponseWriter, r *http.Request) {
	id, ok := urlID(r, "id")
	if !ok {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid session ID")
		return
	}
	s, err := h.svc.GetSession(r.Context(), instID(r), id)
	if err != nil {
		examErr(w, err)
		return
	}
	response.Ok(w, s)
}

func (h *Handler) ReopenSession(w http.ResponseWriter, r *http.Request) {
	id, ok := urlID(r, "id")
	if !ok {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid session ID")
		return
	}
	s, err := h.svc.ReopenToDraft(r.Context(), instID(r), id)
	if err != nil {
		examErr(w, err)
		return
	}
	response.Ok(w, s)
}

// ── Papers & groups ──

func (h *Handler) CreatePaper(w http.ResponseWriter, r *http.Request) {
	sid, ok := urlID(r, "id")
	if !ok {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid session ID")
		return
	}
	var req CreatePaperRequest
	if !decode(w, r, &req) {
		return
	}
	p, err := h.svc.CreatePaper(r.Context(), instID(r), sid, &req)
	if err != nil {
		examErr(w, err)
		return
	}
	response.Created(w, p)
}

func (h *Handler) ListPapers(w http.ResponseWriter, r *http.Request) {
	sid, ok := urlID(r, "id")
	if !ok {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid session ID")
		return
	}
	papers, err := h.svc.repo.ListPapers(r.Context(), instID(r), sid)
	if err != nil {
		response.InternalError(w)
		return
	}
	response.Ok(w, papers)
}

func (h *Handler) AddPaperGroup(w http.ResponseWriter, r *http.Request) {
	pid, ok := urlID(r, "paperId")
	if !ok {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid paper ID")
		return
	}
	var req AddPaperGroupRequest
	if !decode(w, r, &req) {
		return
	}
	g, err := h.svc.AddPaperGroup(r.Context(), instID(r), pid, &req)
	if err != nil {
		examErr(w, err)
		return
	}
	response.Created(w, g)
}

// ── Sittings ──

func (h *Handler) CreateSitting(w http.ResponseWriter, r *http.Request) {
	sid, ok := urlID(r, "id")
	if !ok {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid session ID")
		return
	}
	var req CreateSittingRequest
	if !decode(w, r, &req) {
		return
	}
	s, err := h.svc.CreateSitting(r.Context(), instID(r), sid, &req)
	if err != nil {
		examErr(w, err)
		return
	}
	response.Created(w, s)
}

func (h *Handler) UpdateSitting(w http.ResponseWriter, r *http.Request) {
	sitID, ok := urlID(r, "sittingId")
	if !ok {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid sitting ID")
		return
	}
	var req UpdateSittingRequest
	if !decode(w, r, &req) {
		return
	}
	s, err := h.svc.UpdateSitting(r.Context(), instID(r), sitID, &req)
	if err != nil {
		examErr(w, err)
		return
	}
	response.Ok(w, s)
}

func (h *Handler) DeleteSitting(w http.ResponseWriter, r *http.Request) {
	sitID, ok := urlID(r, "sittingId")
	if !ok {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid sitting ID")
		return
	}
	if err := h.svc.DeleteSitting(r.Context(), instID(r), sitID); err != nil {
		examErr(w, err)
		return
	}
	response.Ok(w, map[string]bool{"deleted": true})
}

// ── Invigilators & arrangements ──

func (h *Handler) AssignInvigilator(w http.ResponseWriter, r *http.Request) {
	sitID, ok := urlID(r, "sittingId")
	if !ok {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid sitting ID")
		return
	}
	var req AssignInvigilatorRequest
	if !decode(w, r, &req) {
		return
	}
	a, err := h.svc.AssignInvigilator(r.Context(), instID(r), sitID, &req)
	if err != nil {
		examErr(w, err)
		return
	}
	response.Created(w, a)
}

func (h *Handler) RemoveInvigilator(w http.ResponseWriter, r *http.Request) {
	sitID, ok := urlID(r, "sittingId")
	if !ok {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid sitting ID")
		return
	}
	staffID, err := uuid.Parse(r.URL.Query().Get("staff_id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_STAFF_ID", "Valid staff_id query param is required")
		return
	}
	if err := h.svc.RemoveInvigilator(r.Context(), instID(r), sitID, staffID); err != nil {
		examErr(w, err)
		return
	}
	response.Ok(w, map[string]bool{"deleted": true})
}

func (h *Handler) AddArrangement(w http.ResponseWriter, r *http.Request) {
	sitID, ok := urlID(r, "sittingId")
	if !ok {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid sitting ID")
		return
	}
	var req AddArrangementRequest
	if !decode(w, r, &req) {
		return
	}
	a, err := h.svc.AddArrangement(r.Context(), instID(r), sitID, &req)
	if err != nil {
		examErr(w, err)
		return
	}
	response.Created(w, a)
}

// ── Validate / publish / generate ──

func (h *Handler) Validate(w http.ResponseWriter, r *http.Request) {
	sid, ok := urlID(r, "id")
	if !ok {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid session ID")
		return
	}
	res, err := h.svc.Validate(r.Context(), instID(r), sid)
	if err != nil {
		examErr(w, err)
		return
	}
	response.Ok(w, res)
}

func (h *Handler) Publish(w http.ResponseWriter, r *http.Request) {
	sid, ok := urlID(r, "id")
	if !ok {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid session ID")
		return
	}
	s, err := h.svc.Publish(r.Context(), instID(r), sid)
	if err != nil {
		examErr(w, err)
		return
	}
	response.Ok(w, s)
}

func (h *Handler) GenerateDraft(w http.ResponseWriter, r *http.Request) {
	sid, ok := urlID(r, "id")
	if !ok {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid session ID")
		return
	}
	var req GenerateDraftRequest
	if !decode(w, r, &req) {
		return
	}
	res, err := h.svc.GenerateDraft(r.Context(), instID(r), sid, &req)
	if err != nil {
		examErr(w, err)
		return
	}
	response.Ok(w, res)
}

// ── Views (all derive from the canonical schedule) ──

func (h *Handler) MasterView(w http.ResponseWriter, r *http.Request) {
	sid, ok := urlID(r, "id")
	if !ok {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid session ID")
		return
	}
	views, err := h.svc.MasterTimetable(r.Context(), instID(r), sid, false)
	if err != nil {
		examErr(w, err)
		return
	}
	response.Ok(w, views)
}

func (h *Handler) PublishedMasterView(w http.ResponseWriter, r *http.Request) {
	sid, ok := urlID(r, "id")
	if !ok {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid session ID")
		return
	}
	views, err := h.svc.MasterTimetable(r.Context(), instID(r), sid, true)
	if err != nil {
		examErr(w, err)
		return
	}
	response.Ok(w, views)
}

func (h *Handler) CandidateView(w http.ResponseWriter, r *http.Request) {
	sid, ok := urlID(r, "id")
	if !ok {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid session ID")
		return
	}
	classID, err := uuid.Parse(r.URL.Query().Get("class_id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_CLASS_ID", "Valid class_id query param is required")
		return
	}
	views, err := h.svc.CandidateView(r.Context(), instID(r), sid, classID, false)
	if err != nil {
		examErr(w, err)
		return
	}
	response.Ok(w, views)
}

func (h *Handler) PaperTimetableView(w http.ResponseWriter, r *http.Request) {
	sid, ok := urlID(r, "id")
	if !ok {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid session ID")
		return
	}
	paperID, err := uuid.Parse(r.URL.Query().Get("paper_id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_PAPER_ID", "Valid paper_id query param is required")
		return
	}
	views, err := h.svc.PaperView(r.Context(), instID(r), sid, paperID)
	if err != nil {
		examErr(w, err)
		return
	}
	response.Ok(w, views)
}

func (h *Handler) RoomView(w http.ResponseWriter, r *http.Request) {
	sid, ok := urlID(r, "id")
	if !ok {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid session ID")
		return
	}
	room := r.URL.Query().Get("room")
	if room == "" {
		response.Error(w, http.StatusBadRequest, "MISSING_ROOM", "room query param is required")
		return
	}
	views, err := h.svc.RoomView(r.Context(), instID(r), sid, room, false)
	if err != nil {
		examErr(w, err)
		return
	}
	response.Ok(w, views)
}

func (h *Handler) InvigilatorView(w http.ResponseWriter, r *http.Request) {
	sid, ok := urlID(r, "id")
	if !ok {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid session ID")
		return
	}
	staffID, err := uuid.Parse(r.URL.Query().Get("staff_id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_STAFF_ID", "Valid staff_id query param is required")
		return
	}
	views, err := h.svc.InvigilatorView(r.Context(), instID(r), sid, staffID, false)
	if err != nil {
		examErr(w, err)
		return
	}
	response.Ok(w, views)
}
