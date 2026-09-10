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
	sessionType := r.URL.Query().Get("session_type")
	var periodID *uuid.UUID
	if pidStr := r.URL.Query().Get("period_id"); pidStr != "" {
		if pid, err := uuid.Parse(pidStr); err == nil {
			periodID = &pid
		}
	}

	session, err := h.svc.GetAttendanceSheet(r.Context(), institutionID, classID, teacherID, dateStr, sessionType, periodID)
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

	// attendance:approve is the existing permission for "not the class
	// teacher, but allowed to submit daily attendance anyway" — held by
	// Super Admin today, and by anyone else a school later chooses to
	// grant it to (e.g. a deputy covering for an absent class teacher).
	canOverride := claims.HasPermission("attendance:approve")

	var req TakeAttendanceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_BODY", "Invalid request body")
		return
	}

	session, err := h.svc.SubmitAttendance(r.Context(), institutionID, teacherID, canOverride, &req)
	if err != nil {
		switch {
		case errors.Is(err, ErrSessionLocked):
			response.Error(w, http.StatusConflict, "SESSION_LOCKED",
				"This attendance session has been locked and cannot be modified")
		case errors.Is(err, ErrNotClassTeacher):
			response.Error(w, http.StatusForbidden, "NOT_CLASS_TEACHER",
				"Only this class's assigned class teacher can take daily attendance")
		case errors.Is(err, ErrNotAssignedToClass):
			response.Error(w, http.StatusForbidden, "NOT_ASSIGNED_TO_CLASS",
				"You are not timetabled to teach this class")
		case errors.Is(err, ErrClassNeedsTeacherAssignment):
			response.Error(w, http.StatusConflict, 
				"CLASS_NEEDS_TEACHER_ASSIGNMENT",
				"This class has no assigned class teacher yet. Ask an administrator to assign one before taking daily attendance.")
		case errors.Is(err, ErrClassNotFound):
			response.NotFound(w, "CLASS")
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
	claims := auth.ClaimsFromContext(r.Context())
	teacherID, _ := uuid.Parse(claims.UserID)

	componentID, err := uuid.Parse(r.URL.Query().Get("component_id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_COMPONENT_ID", "Valid component_id is required")
		return
	}

	entries, err := h.svc.GetGradebook(r.Context(), institutionID, teacherID, claims.Role == "teacher", componentID)
	if err != nil {
		if errors.Is(err, ErrNotTimetabledForSubject) {
			response.Error(w, http.StatusForbidden, "NOT_ASSIGNED", "You are not timetabled to teach this subject in this class")
			return
		}
		response.InternalError(w)
		return
	}
	response.Ok(w, entries)
}

func (h *Handler) SaveGrades(w http.ResponseWriter, r *http.Request) {
	institutionID := auth.InstitutionIDFromContext(r.Context())
	claims := auth.ClaimsFromContext(r.Context())
	teacherID, _ := uuid.Parse(claims.UserID)

	var req BulkGradeEntryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_BODY", "Invalid request body")
		return
	}

	if err := h.svc.SaveGrades(r.Context(), institutionID, teacherID, claims.Role == "teacher", &req); err != nil {
		if errors.Is(err, ErrNotTimetabledForSubject) {
			response.Error(w, http.StatusForbidden, "NOT_ASSIGNED", "You are not timetabled to teach this subject in this class")
			return
		}
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
			response.Ok(w, nil)
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

// ListTerms handles GET /api/v1/academics/terms
func (h *Handler) ListTerms(w http.ResponseWriter, r *http.Request) {
	institutionID := auth.InstitutionIDFromContext(r.Context())
	terms, err := h.svc.ListTerms(r.Context(), institutionID)
	if err != nil {
		log.Error().Err(err).Msg("list terms")
		response.InternalError(w)
		return
	}
	response.Ok(w, terms)
}

func (h *Handler) CreateClass(w http.ResponseWriter, r *http.Request) {
	institutionID := auth.InstitutionIDFromContext(r.Context())
	var req CreateClassRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_BODY", "Invalid request body")
		return
	}
	class, err := h.svc.CreateClass(r.Context(), institutionID, &req)
	if err != nil {
		log.Error().Err(err).Msg("create class")
		response.Error(w, http.StatusBadRequest, "CLASS_ERROR", err.Error())
		return
	}
	response.Created(w, class)
}

func (h *Handler) AssignClassTeacher(w http.ResponseWriter, r *http.Request) {
	institutionID := auth.InstitutionIDFromContext(r.Context())
	classID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid class ID")
		return
	}
	var req AssignClassTeacherRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_BODY", "Invalid request body")
		return
	}
	if err := h.svc.AssignClassTeacher(r.Context(), institutionID, classID, req.ClassTeacherID); err != nil {
		if errors.Is(err, ErrClassNotFound) {
			response.NotFound(w, "CLASS")
			return
		}
		log.Error().Err(err).Msg("assign class teacher")
		response.InternalError(w)
		return
	}
	response.NoContent(w)
}

func (h *Handler) ListTeachers(w http.ResponseWriter, r *http.Request) {
	institutionID := auth.InstitutionIDFromContext(r.Context())
	teachers, err := h.svc.ListTeachers(r.Context(), institutionID)
	if err != nil {
		log.Error().Err(err).Msg("list teachers")
		response.InternalError(w)
		return
	}
	response.Ok(w, teachers)
}

func (h *Handler) ListAcademicYears(w http.ResponseWriter, r *http.Request) {
	institutionID := auth.InstitutionIDFromContext(r.Context())
	years, err := h.svc.ListAcademicYears(r.Context(), institutionID)
	if err != nil {
		log.Error().Err(err).Msg("list academic years")
		response.InternalError(w)
		return
	}
	response.Ok(w, years)
}
// academics/handler.go
func (h *Handler) CreateTeachingAssignment(w http.ResponseWriter, r *http.Request) {
	institutionID := auth.InstitutionIDFromContext(r.Context())
	var req CreateTeachingAssignmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_BODY", "Invalid request body")
		return
	}
	ta, err := h.svc.CreateTeachingAssignment(r.Context(), institutionID, &req)
	if err != nil {
		log.Error().Err(err).Msg("create teaching assignment")
		response.Error(w, http.StatusBadRequest, "ASSIGNMENT_ERROR", err.Error())
		return
	}
	response.Created(w, ta)
}

// academics/handler.go — additions
func (h *Handler) ListTeachingAssignmentsForClass(w http.ResponseWriter, r *http.Request) {
	institutionID := auth.InstitutionIDFromContext(r.Context())
	classID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid class ID")
		return
	}
	termID, err := uuid.Parse(r.URL.Query().Get("term_id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_TERM_ID", "Valid term_id is required")
		return
	}
	options, err := h.svc.ListTeachingAssignmentsForClass(r.Context(), institutionID, classID, termID)
	if err != nil {
		response.InternalError(w)
		return
	}
	response.Ok(w, options)
}

func (h *Handler) CreateTimetablePeriod(w http.ResponseWriter, r *http.Request) {
	institutionID := auth.InstitutionIDFromContext(r.Context())
	var req CreateTimetablePeriodRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_BODY", "Invalid request body")
		return
	}
	p, err := h.svc.CreateTimetablePeriod(r.Context(), institutionID, &req)
	if err != nil {
		switch {
		case errors.Is(err, ErrSlotAlreadyBooked):
			response.Error(w, http.StatusConflict, "SLOT_BOOKED", err.Error())
		case errors.Is(err, ErrTeacherDoubleBooked):
			response.Error(w, http.StatusConflict, "TEACHER_DOUBLE_BOOKED", err.Error())
		default:
			log.Error().Err(err).Msg("create timetable period")
			response.Error(w, http.StatusBadRequest, "PERIOD_ERROR", err.Error())
		}
		return
	}
	response.Created(w, p)
}