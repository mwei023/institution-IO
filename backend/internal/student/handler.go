package student

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"institution-os/internal/auth"
	"institution-os/pkg/response"
	"github.com/rs/zerolog/log"
)

// Handler handles student HTTP endpoints.
type Handler struct {
	svc *Service
}

// NewHandler creates a new student handler.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// --- Admissions ---

// CreateAdmission handles POST /api/v1/admissions
func (h *Handler) CreateAdmission(w http.ResponseWriter, r *http.Request) {
	institutionID := auth.InstitutionIDFromContext(r.Context())

	var req CreateAdmissionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_BODY", "Invalid request body")
		return
	}

	admission, err := h.svc.CreateAdmission(r.Context(), institutionID, &req)
	if err != nil {
		log.Error().Err(err).Msg("create admission")
		response.Error(w, http.StatusBadRequest, "CREATE_ADMISSION_FAILED", err.Error())
		return
	}

	response.Created(w, admission)
}

// ListAdmissions handles GET /api/v1/admissions
func (h *Handler) ListAdmissions(w http.ResponseWriter, r *http.Request) {
	institutionID := auth.InstitutionIDFromContext(r.Context())
	status := r.URL.Query().Get("status")
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))

	admissions, total, err := h.svc.ListAdmissions(r.Context(), institutionID, status, page, pageSize)
	if err != nil {
		log.Error().Err(err).Msg("list admissions")
		response.InternalError(w)
		return
	}

	response.OkList(w, admissions, &response.Pagination{
		Total:    total,
		Page:     page,
		PageSize: pageSize,
		HasNext:  int64(page*pageSize) < total,
	})
}

// GetAdmission handles GET /api/v1/admissions/{id}
func (h *Handler) GetAdmission(w http.ResponseWriter, r *http.Request) {
	institutionID := auth.InstitutionIDFromContext(r.Context())
	admissionID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid admission ID")
		return
	}

	admission, err := h.svc.GetAdmission(r.Context(), institutionID, admissionID)
	if err != nil {
		if errors.Is(err, ErrAdmissionNotFound) {
			response.NotFound(w, "ADMISSION")
			return
		}
		log.Error().Err(err).Msg("get admission")
		response.InternalError(w)
		return
	}

	response.Ok(w, admission)
}

// ReviewAdmission handles POST /api/v1/admissions/{id}/review
func (h *Handler) ReviewAdmission(w http.ResponseWriter, r *http.Request) {
	institutionID := auth.InstitutionIDFromContext(r.Context())
	claims := auth.ClaimsFromContext(r.Context())

	admissionID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid admission ID")
		return
	}

	reviewerID, err := uuid.Parse(claims.UserID)
	if err != nil {
		response.InternalError(w)
		return
	}

	var req ApproveAdmissionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_BODY", "Invalid request body")
		return
	}

	admission, err := h.svc.ReviewAdmission(r.Context(), institutionID, admissionID, reviewerID, req.Action, req.Notes)
	if err != nil {
		log.Error().Err(err).Msg("review admission")
		response.Error(w, http.StatusBadRequest, "REVIEW_FAILED", err.Error())
		return
	}

	response.Ok(w, admission)
}

// EnrollStudent handles POST /api/v1/admissions/{id}/enroll
func (h *Handler) EnrollStudent(w http.ResponseWriter, r *http.Request) {
	institutionID := auth.InstitutionIDFromContext(r.Context())

	admissionID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid admission ID")
		return
	}

	var req EnrollStudentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_BODY", "Invalid request body")
		return
	}

	student, err := h.svc.EnrollStudent(r.Context(), institutionID, admissionID, &req)
	if err != nil {
		log.Error().Err(err).Msg("enroll student")
		response.Error(w, http.StatusBadRequest, "ENROLL_FAILED", err.Error())
		return
	}

	response.Created(w, student)
}

// --- Students ---

// ListStudents handles GET /api/v1/students
func (h *Handler) ListStudents(w http.ResponseWriter, r *http.Request) {
	institutionID := auth.InstitutionIDFromContext(r.Context())

	f := &StudentFilter{
		Status:   r.URL.Query().Get("status"),
		Search:   r.URL.Query().Get("search"),
	}

	if classIDStr := r.URL.Query().Get("class_id"); classIDStr != "" {
		id, err := uuid.Parse(classIDStr)
		if err == nil {
			f.ClassID = &id
		}
	}

	f.Page, _ = strconv.Atoi(r.URL.Query().Get("page"))
	f.PageSize, _ = strconv.Atoi(r.URL.Query().Get("page_size"))

	students, total, err := h.svc.ListStudents(r.Context(), institutionID, f)
	if err != nil {
		log.Error().Err(err).Msg("list students")
		response.InternalError(w)
		return
	}

	response.OkList(w, students, &response.Pagination{
		Total:    total,
		Page:     f.Page,
		PageSize: f.PageSize,
		HasNext:  int64(f.Page*f.PageSize) < total,
	})
}

// GetStudent handles GET /api/v1/students/{id}
func (h *Handler) GetStudent(w http.ResponseWriter, r *http.Request) {
	institutionID := auth.InstitutionIDFromContext(r.Context())
	studentID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid student ID")
		return
	}

	student, err := h.svc.GetStudent(r.Context(), institutionID, studentID)
	if err != nil {
		if errors.Is(err, ErrStudentNotFound) {
			response.NotFound(w, "STUDENT")
			return
		}
		log.Error().Err(err).Msg("get student")
		response.InternalError(w)
		return
	}

	response.Ok(w, student)
}

// GetStudentTimeline handles GET /api/v1/students/{id}/timeline
func (h *Handler) GetStudentTimeline(w http.ResponseWriter, r *http.Request) {
	institutionID := auth.InstitutionIDFromContext(r.Context())
	studentID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid student ID")
		return
	}

	events, err := h.svc.GetStudentTimeline(r.Context(), institutionID, studentID)
	if err != nil {
		log.Error().Err(err).Msg("get timeline")
		response.InternalError(w)
		return
	}

	response.Ok(w, events)
}

// GetStudentParents handles GET /api/v1/students/{id}/parents
func (h *Handler) GetStudentParents(w http.ResponseWriter, r *http.Request) {
	institutionID := auth.InstitutionIDFromContext(r.Context())
	studentID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid student ID")
		return
	}

	parents, err := h.svc.GetStudentParents(r.Context(), institutionID, studentID)
	if err != nil {
		log.Error().Err(err).Msg("get student parents")
		response.InternalError(w)
		return
	}

	response.Ok(w, parents)
}
