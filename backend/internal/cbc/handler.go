package cbc

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"institution-os/internal/auth"
	"institution-os/pkg/response"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// CreateLearningArea handles POST /api/v1/cbc/learning-areas
func (h *Handler) CreateLearningArea(w http.ResponseWriter, r *http.Request) {
	institutionID := auth.InstitutionIDFromContext(r.Context())
	var req CreateLearningAreaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_BODY", "Invalid request body")
		return
	}
	la, err := h.svc.CreateLearningArea(r.Context(), institutionID, &req)
	if err != nil {
		log.Error().Err(err).Msg("create learning area")
		response.Error(w, http.StatusBadRequest, "LEARNING_AREA_ERROR", err.Error())
		return
	}
	response.Created(w, la)
}

// ListLearningAreas handles GET /api/v1/cbc/learning-areas
func (h *Handler) ListLearningAreas(w http.ResponseWriter, r *http.Request) {
	institutionID := auth.InstitutionIDFromContext(r.Context())
	areas, err := h.svc.ListLearningAreas(r.Context(), institutionID)
	if err != nil {
		log.Error().Err(err).Msg("list learning areas")
		response.InternalError(w)
		return
	}
	response.Ok(w, areas)
}

// CreateComponent handles POST /api/v1/cbc/components
func (h *Handler) CreateComponent(w http.ResponseWriter, r *http.Request) {
	institutionID := auth.InstitutionIDFromContext(r.Context())
	var req CreateComponentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_BODY", "Invalid request body")
		return
	}
	c, err := h.svc.CreateComponent(r.Context(), institutionID, &req)
	if err != nil {
		log.Error().Err(err).Msg("create cbc component")
		response.Error(w, http.StatusBadRequest, "COMPONENT_ERROR", err.Error())
		return
	}
	response.Created(w, c)
}

// GetComponents handles GET /api/v1/cbc/components?class_id=&learning_area_id=&term_id=
func (h *Handler) GetComponents(w http.ResponseWriter, r *http.Request) {
	institutionID := auth.InstitutionIDFromContext(r.Context())
	parse := func(key string) (uuid.UUID, bool) {
		id, err := uuid.Parse(r.URL.Query().Get(key))
		return id, err == nil
	}
	classID, ok1 := parse("class_id")
	learningAreaID, ok2 := parse("learning_area_id")
	termID, ok3 := parse("term_id")
	if !ok1 || !ok2 || !ok3 {
		response.Error(w, http.StatusBadRequest, "MISSING_PARAMS", "class_id, learning_area_id, and term_id are required")
		return
	}
	comps, err := h.svc.ListComponents(r.Context(), institutionID, classID, learningAreaID, termID)
	if err != nil {
		response.InternalError(w)
		return
	}
	response.Ok(w, comps)
}

// GetEntries handles GET /api/v1/cbc/entries?component_id=
func (h *Handler) GetEntries(w http.ResponseWriter, r *http.Request) {
	institutionID := auth.InstitutionIDFromContext(r.Context())
	claims := auth.ClaimsFromContext(r.Context())
	teacherID, _ := uuid.Parse(claims.UserID)

	componentID, err := uuid.Parse(r.URL.Query().Get("component_id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_COMPONENT_ID", "Valid component_id is required")
		return
	}
	entries, err := h.svc.GetEntries(r.Context(), institutionID, teacherID, claims.Role == "teacher", componentID)
	if err != nil {
		if errors.Is(err, ErrNotAssignedToLearningArea) {
			response.Error(w, http.StatusForbidden, "NOT_ASSIGNED", "You are not assigned to teach this learning area for this class")
			return
		}
		response.InternalError(w)
		return
	}
	response.Ok(w, entries)
}

// SaveEntries handles POST /api/v1/cbc/entries
func (h *Handler) SaveEntries(w http.ResponseWriter, r *http.Request) {
	institutionID := auth.InstitutionIDFromContext(r.Context())
	claims := auth.ClaimsFromContext(r.Context())
	teacherID, _ := uuid.Parse(claims.UserID)

	var req BulkEntryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_BODY", "Invalid request body")
		return
	}
	if err := h.svc.SaveEntries(r.Context(), institutionID, teacherID, &req); err != nil {
		if errors.Is(err, ErrNotAssignedToLearningArea) {
			response.Error(w, http.StatusForbidden, "NOT_ASSIGNED", "You are not assigned to teach this learning area for this class")
			return
		}
		log.Error().Err(err).Msg("save cbc entries")
		response.Error(w, http.StatusBadRequest, "ENTRIES_ERROR", err.Error())
		return
	}
	response.NoContent(w)
}

// CreateTeacherAssignment handles POST /api/v1/cbc/teacher-assignments
func (h *Handler) CreateTeacherAssignment(w http.ResponseWriter, r *http.Request) {
	institutionID := auth.InstitutionIDFromContext(r.Context())
	var req struct {
		ClassID        uuid.UUID `json:"class_id" validate:"required"`
		LearningAreaID uuid.UUID `json:"learning_area_id" validate:"required"`
		TeacherID      uuid.UUID `json:"teacher_id" validate:"required"`
		TermID         uuid.UUID `json:"term_id" validate:"required"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_BODY", "Invalid request body")
		return
	}
	ta, err := h.svc.CreateTeacherAssignment(r.Context(), institutionID, req.ClassID, req.LearningAreaID, req.TeacherID, req.TermID)
	if err != nil {
		log.Error().Err(err).Msg("create cbc teacher assignment")
		response.Error(w, http.StatusBadRequest, "ASSIGNMENT_ERROR", err.Error())
		return
	}
	response.Created(w, ta)
}

// GenerateReportCardsForClass handles POST /api/v1/cbc/report-cards/generate-class
func (h *Handler) GenerateReportCardsForClass(w http.ResponseWriter, r *http.Request) {
	institutionID := auth.InstitutionIDFromContext(r.Context())
	var req GenerateReportCardsForClassRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_BODY", "Invalid request body")
		return
	}
	result, err := h.svc.GenerateReportCardsForClass(r.Context(), institutionID, req.ClassID, req.TermID)
	if err != nil {
		switch {
		case errors.Is(err, ErrLearningAreaNotFound):
			response.NotFound(w, "LEARNING_AREA")
		case errors.Is(err, ErrNoRatingBandsConfigured):
			response.Error(w, http.StatusUnprocessableEntity, "NO_RATING_BANDS",
				"No rating bands are configured for this curriculum/grade band yet")
		default:
			log.Error().Err(err).Msg("generate cbc report cards for class")
			response.Error(w, http.StatusBadRequest, "REPORT_CARD_ERROR", err.Error())
		}
		return
	}
	response.Ok(w, result)
}

// PublishReportCard handles POST /api/v1/cbc/report-cards/publish
func (h *Handler) PublishReportCard(w http.ResponseWriter, r *http.Request) {
	institutionID := auth.InstitutionIDFromContext(r.Context())
	var req PublishReportCardRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_BODY", "Invalid request body")
		return
	}
	rc, err := h.svc.PublishReportCard(r.Context(), institutionID, req.StudentID, req.TermID)
	if err != nil {
		if errors.Is(err, ErrReportCardNotFound) {
			response.NotFound(w, "REPORT_CARD")
			return
		}
		log.Error().Err(err).Msg("publish cbc report card")
		response.InternalError(w)
		return
	}
	response.Ok(w, rc)
}

// UnpublishReportCard handles POST /api/v1/cbc/report-cards/unpublish
func (h *Handler) UnpublishReportCard(w http.ResponseWriter, r *http.Request) {
	institutionID := auth.InstitutionIDFromContext(r.Context())
	var req PublishReportCardRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_BODY", "Invalid request body")
		return
	}
	rc, err := h.svc.UnpublishReportCard(r.Context(), institutionID, req.StudentID, req.TermID)
	if err != nil {
		if errors.Is(err, ErrReportCardNotFound) {
			response.NotFound(w, "REPORT_CARD")
			return
		}
		log.Error().Err(err).Msg("unpublish cbc report card")
		response.InternalError(w)
		return
	}
	response.Ok(w, rc)
}

// GetChildReportCard handles GET /api/v1/parent/children/{id}/cbc-report-card
func (h *Handler) GetChildReportCard(w http.ResponseWriter, r *http.Request) {
	institutionID := auth.InstitutionIDFromContext(r.Context())
	claims := auth.ClaimsFromContext(r.Context())
	studentID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid student ID")
		return
	}
	if claims != nil && claims.Role == "parent" {
		parentID, perr := uuid.Parse(claims.UserID)
		if perr != nil {
			response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid user")
			return
		}
		if verr := h.svc.VerifyParentChild(r.Context(), institutionID, parentID, studentID); verr != nil {
			response.Error(w, http.StatusForbidden, "FORBIDDEN", "Not linked to student")
			return
		}
	}
	rc, err := h.svc.GetChildReportCard(r.Context(), institutionID, studentID)
	if err != nil {
		if errors.Is(err, ErrReportCardNotFound) {
			response.Ok(w, nil) // no published card yet — not an error for the parent
			return
		}
		log.Error().Err(err).Msg("get child cbc report card")
		response.InternalError(w)
		return
	}
	response.Ok(w, rc)
}