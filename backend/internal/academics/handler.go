package academics

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"institution-os/internal/auth"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"institution-os/pkg/response"

	"github.com/rs/zerolog/log"
)

// Handler handles academics HTTP endpoints.
type Handler struct {
	svc *Service
}

// NewHandler creates a new academics handler.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// ListClasses handles GET /api/v1/academics/classes
func (h *Handler) ListClasses(w http.ResponseWriter, r *http.Request) {
	institutionID := auth.InstitutionIDFromContext(r.Context())

	f := &ClassFilter{Page: 1, PageSize: 50}
	if levelStr := r.URL.Query().Get("level"); levelStr != "" {
		var level int
		if _, err := fmt.Sscanf(levelStr, "%d", &level); err == nil {
			f.Level = &level
		}
	}

	if ayID := r.URL.Query().Get("academic_year_id"); ayID != "" {
		if id, err := uuid.Parse(ayID); err == nil {
			f.AcademicYearID = &id
		}
	}

	classes, err := h.svc.ListClasses(r.Context(), institutionID, f)
	if err != nil {
		log.Error().Err(err).Msg("list classes")
		response.InternalError(w)
		return
	}
	response.Ok(w, classes)
}

// GetClass handles GET /api/v1/academics/classes/{id}
func (h *Handler) GetClass(w http.ResponseWriter, r *http.Request) {
	institutionID := auth.InstitutionIDFromContext(r.Context())
	classID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid class ID")
		return
	}

	class, err := h.svc.GetClass(r.Context(), institutionID, classID)
	if err != nil {
		if errors.Is(err, ErrClassNotFound) {
			response.NotFound(w, "CLASS")
			return
		}
		response.InternalError(w)
		return
	}
	response.Ok(w, class)
}

// ListSubjects handles GET /api/v1/academics/subjects
func (h *Handler) ListSubjects(w http.ResponseWriter, r *http.Request) {
	institutionID := auth.InstitutionIDFromContext(r.Context())
	subjects, err := h.svc.ListSubjects(r.Context(), institutionID)
	if err != nil {
		log.Error().Err(err).Msg("list subjects")
		response.InternalError(w)
		return
	}
	response.Ok(w, subjects)
}

// GetClassTimetable handles GET /api/v1/academics/classes/{id}/timetable
func (h *Handler) GetClassTimetable(w http.ResponseWriter, r *http.Request) {
	institutionID := auth.InstitutionIDFromContext(r.Context())
	classID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid class ID")
		return
	}

	var termID *uuid.UUID
	if termIDStr := r.URL.Query().Get("term_id"); termIDStr != "" {
		if id, err := uuid.Parse(termIDStr); err == nil {
			termID = &id
		}
	}

	periods, err := h.svc.GetClassTimetable(r.Context(), institutionID, classID, termID)
	if err != nil {
		response.InternalError(w)
		return
	}
	response.Ok(w, periods)
}

// GetMyTimetable handles GET /api/v1/academics/my-timetable (teacher)
func (h *Handler) GetMyTimetable(w http.ResponseWriter, r *http.Request) {
	institutionID := auth.InstitutionIDFromContext(r.Context())
	claims := auth.ClaimsFromContext(r.Context())
	teacherID, err := uuid.Parse(claims.UserID)
	if err != nil {
		response.InternalError(w)
		return
	}

	var termID *uuid.UUID
	if termIDStr := r.URL.Query().Get("term_id"); termIDStr != "" {
		if id, err := uuid.Parse(termIDStr); err == nil {
			termID = &id
		}
	}

	periods, err := h.svc.GetTeacherTimetable(r.Context(), institutionID, teacherID, termID)
	if err != nil {
		response.InternalError(w)
		return
	}
	response.Ok(w, periods)
}

// GetAttendanceSheet handles GET /api/v1/academics/attendance/sheet?class_id=...&date=...
func (h *Handler) GetAttendanceSheet(w http.ResponseWriter, r *http.Request) {
	institutionID := auth.InstitutionIDFromContext(r.Context())
	claims := auth.ClaimsFromContext(r.Context())
	teacherID, _ := uuid.Parse(claims.UserID)

	classIDStr := r.URL.Query().Get("class_id")
	if classIDStr == "" {
		response.Error(w, http.StatusBadRequest, "MISSING_CLASS_ID", "class_id is required")
		return
	}
	classID, err := uuid.Parse(classIDStr)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_CLASS_ID", "Invalid class ID")
		return
	}

	dateStr := r.URL.Query().Get("date")

	session, err := h.svc.GetAttendanceSheet(r.Context(), institutionID, classID, teacherID, dateStr)
	if err != nil {
		log.Error().Err(err).Msg("get attendance sheet")
		response.Error(w, http.StatusBadRequest, "ATTENDANCE_ERROR", err.Error())
		return
	}
	response.Ok(w, session)
}

// SubmitAttendance handles POST /api/v1/academics/attendance
func (h *Handler) SubmitAttendance(w http.ResponseWriter, r *http.Request) {
	institutionID := auth.InstitutionIDFromContext(r.Context())
	claims := auth.ClaimsFromContext(r.Context())
	teacherID, _ := uuid.Parse(claims.UserID)

	var req TakeAttendanceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_BODY", "Invalid request body")
		return
	}

	session, err := h.svc.SubmitAttendance(r.Context(), institutionID, teacherID, &req)
	if err != nil {
		switch {
		case errors.Is(err, ErrSessionLocked):
			response.Error(w, http.StatusConflict, "SESSION_LOCKED",
				"This attendance session has been locked and cannot be modified")
		default:
			log.Error().Err(err).Msg("submit attendance")
			response.Error(w, http.StatusBadRequest, "ATTENDANCE_ERROR", err.Error())
		}
		return
	}
	response.Ok(w, session)
}

// GetAttendanceSummary handles GET /api/v1/academics/attendance/summary
func (h *Handler) GetAttendanceSummary(w http.ResponseWriter, r *http.Request) {
	institutionID := auth.InstitutionIDFromContext(r.Context())

	classIDStr := r.URL.Query().Get("class_id")
	if classIDStr == "" {
		response.Error(w, http.StatusBadRequest, "MISSING_CLASS_ID", "class_id is required")
		return
	}
	classID, err := uuid.Parse(classIDStr)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_CLASS_ID", "Invalid class ID")
		return
	}

	startDate := time.Now().AddDate(0, -3, 0)
	endDate := time.Now()
	if s := r.URL.Query().Get("start_date"); s != "" {
		if t, err := time.Parse("2006-01-02", s); err == nil {
			startDate = t
		}
	}
	if e := r.URL.Query().Get("end_date"); e != "" {
		if t, err := time.Parse("2006-01-02", e); err == nil {
			endDate = t
		}
	}

	summary, err := h.svc.GetAttendanceSummary(r.Context(), institutionID, classID, startDate, endDate)
	if err != nil {
		log.Error().Err(err).Msg("get attendance summary")
		response.InternalError(w)
		return
	}
	response.Ok(w, summary)
}

// GetGradeComponents handles GET /api/v1/academics/grades/components
func (h *Handler) GetGradeComponents(w http.ResponseWriter, r *http.Request) {
	institutionID := auth.InstitutionIDFromContext(r.Context())

	parse := func(key string) (uuid.UUID, bool) {
		id, err := uuid.Parse(r.URL.Query().Get(key))
		return id, err == nil
	}

	classID, ok1 := parse("class_id")
	subjectID, ok2 := parse("subject_id")
	termID, ok3 := parse("term_id")

	if !ok1 || !ok2 || !ok3 {
		response.Error(w, http.StatusBadRequest, "MISSING_PARAMS",
			"class_id, subject_id, and term_id are required")
		return
	}

	components, err := h.svc.GetGradeComponents(r.Context(), institutionID, classID, subjectID, termID)
	if err != nil {
		response.InternalError(w)
		return
	}
	response.Ok(w, components)
}

// GetGradebook handles GET /api/v1/academics/grades/entries?component_id=...
func (h *Handler) GetGradebook(w http.ResponseWriter, r *http.Request) {
	institutionID := auth.InstitutionIDFromContext(r.Context())
	componentID, err := uuid.Parse(r.URL.Query().Get("component_id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_COMPONENT_ID", "Valid component_id is required")
		return
	}

	entries, err := h.svc.GetGradebook(r.Context(), institutionID, componentID)
	if err != nil {
		response.InternalError(w)
		return
	}
	response.Ok(w, entries)
}

// SaveGrades handles POST /api/v1/academics/grades/entries
func (h *Handler) SaveGrades(w http.ResponseWriter, r *http.Request) {
	institutionID := auth.InstitutionIDFromContext(r.Context())
	claims := auth.ClaimsFromContext(r.Context())
	teacherID, _ := uuid.Parse(claims.UserID)

	var req BulkGradeEntryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_BODY", "Invalid request body")
		return
	}

	if err := h.svc.SaveGrades(r.Context(), institutionID, teacherID, &req); err != nil {
		log.Error().Err(err).Msg("save grades")
		response.Error(w, http.StatusBadRequest, "GRADES_ERROR", err.Error())
		return
	}
	response.NoContent(w)
}

// CreateGradeComponent handles POST /api/v1/academics/grades/components
func (h *Handler) CreateGradeComponent(w http.ResponseWriter, r *http.Request) {
	institutionID := auth.InstitutionIDFromContext(r.Context())

	var req CreateGradeComponentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_BODY", "Invalid request body")
		return
	}

	component, err := h.svc.CreateGradeComponent(r.Context(), institutionID, &req)
	if err != nil {
		log.Error().Err(err).Msg("create grade component")
		response.Error(w, http.StatusBadRequest, "COMPONENT_ERROR", err.Error())
		return
	}
	response.Created(w, component)
}

// GenerateReportCard handles POST /api/v1/academics/report-cards/generate
func (h *Handler) GenerateReportCard(w http.ResponseWriter, r *http.Request) {
	institutionID := auth.InstitutionIDFromContext(r.Context())

	var req GenerateReportCardRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_BODY", "Invalid request body")
		return
	}

	rc, err := h.svc.GenerateReportCard(r.Context(), institutionID, req.StudentID, req.ClassID, req.TermID)
	if err != nil {
		switch {
		case errors.Is(err, ErrReportCardPublished):
			response.Conflict(w, "This report card has already been published. Unpublish it first to regenerate.")
		default:
			log.Error().Err(err).Msg("generate report card")
			response.Error(w, http.StatusBadRequest, "REPORT_CARD_ERROR", err.Error())
		}
		return
	}
	response.Ok(w, rc)
}

// UnpublishReportCard handles POST /api/v1/academics/report-cards/unpublish
func (h *Handler) UnpublishReportCard(w http.ResponseWriter, r *http.Request) {
	institutionID := auth.InstitutionIDFromContext(r.Context())

	var req PublishReportCardRequest // same shape: student_id + term_id
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_BODY", "Invalid request body")
		return
	}

	rc, err := h.svc.UnpublishReportCard(r.Context(), institutionID, req.StudentID, req.TermID)
	if err != nil {
		switch {
		case errors.Is(err, ErrReportCardNotFound):
			response.NotFound(w, "REPORT_CARD")
		default:
			log.Error().Err(err).Msg("unpublish report card")
			response.InternalError(w)
		}
		return
	}
	response.Ok(w, rc)
}

// GenerateReportCardsForClass handles POST /api/v1/academics/report-cards/generate-class
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
		case errors.Is(err, ErrClassNotFound):
			response.NotFound(w, "CLASS")
		default:
			log.Error().Err(err).Msg("generate report cards for class")
			response.Error(w, http.StatusBadRequest, "REPORT_CARD_ERROR", err.Error())
		}
		return
	}
	// 200, not 201 — this is a bulk compute/upsert that may partially
	// succeed, not the creation of a single resource.
	response.Ok(w, result)
}

// PublishReportCard handles POST /api/v1/academics/report-cards/publish
func (h *Handler) PublishReportCard(w http.ResponseWriter, r *http.Request) {
	institutionID := auth.InstitutionIDFromContext(r.Context())

	var req PublishReportCardRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_BODY", "Invalid request body")
		return
	}

	rc, err := h.svc.PublishReportCard(r.Context(), institutionID, req.StudentID, req.TermID)
	if err != nil {
		switch {
		case errors.Is(err, ErrReportCardNotFound):
			response.NotFound(w, "REPORT_CARD")
		default:
			log.Error().Err(err).Msg("publish report card")
			response.InternalError(w)
		}
		return
	}
	response.Ok(w, rc)
}

// GetChildReportCard handles GET /api/v1/parent/children/{id}/report-card
func (h *Handler) GetChildReportCard(w http.ResponseWriter, r *http.Request) {
	institutionID := auth.InstitutionIDFromContext(r.Context())
	studentID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid student ID")
		return
	}

	rc, err := h.svc.GetChildReportCard(r.Context(), institutionID, studentID)
	if err != nil {
		if errors.Is(err, ErrReportCardNotFound) {
			response.Ok(w, nil) // no published card yet — not an error for the parent
			return
		}
		log.Error().Err(err).Msg("get child report card")
		response.InternalError(w)
		return
	}
	response.Ok(w, rc)
}

// GetChildSubjectScores handles GET /api/v1/parent/children/{id}/subject-scores
func (h *Handler) GetChildSubjectScores(w http.ResponseWriter, r *http.Request) {
	institutionID := auth.InstitutionIDFromContext(r.Context())
	studentID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid student ID")
		return
	}

	scores, err := h.svc.GetChildSubjectScores(r.Context(), institutionID, studentID)
	if err != nil {
		log.Error().Err(err).Msg("get child subject scores")
		response.InternalError(w)
		return
	}
	response.Ok(w, scores)
}