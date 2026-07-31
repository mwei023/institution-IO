package academics

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"institution-os/internal/auth"
	"institution-os/pkg/response"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

// ─── Repository ───────────────────────────────────────────────────

// GetInstitutionAttendanceToday summarizes daily attendance for a given
// date across every active student in the institution, for the principal
// dashboard. Only counts "daily" sessions (not period-level ones) — this
// codebase seeds exclusively 'daily' sessions today, but the filter is
// kept explicit in case period-level daily rollups are added later.
// A student with no session recorded yet is not counted as absent — see
// RecordsTaken, which lets the UI distinguish "no one's taken attendance
// yet this morning" from "a genuinely bad attendance day" instead of
// collapsing both into a 0% rate.
func (r *Repository) GetInstitutionAttendanceToday(ctx context.Context, institutionID uuid.UUID, date time.Time) (*InstitutionAttendanceToday, error) {
	var result InstitutionAttendanceToday
	err := r.db.QueryRow(ctx, `
		SELECT
			(SELECT COUNT(*) FROM students
			   WHERE institution_id = $1 AND status = 'active' AND deleted_at IS NULL) AS total_active,
			COUNT(ar.id) AS records_taken,
			COUNT(*) FILTER (WHERE ar.status = 'present') AS present,
			COUNT(*) FILTER (WHERE ar.status = 'absent')  AS absent,
			COUNT(*) FILTER (WHERE ar.status = 'late')    AS late,
			COUNT(*) FILTER (WHERE ar.status = 'excused') AS excused
		FROM attendance_records ar
		JOIN attendance_sessions sess ON sess.id = ar.session_id
		WHERE sess.institution_id = $1
		  AND sess.session_date = $2
		  AND sess.session_type = 'daily'`,
		institutionID, date,
	).Scan(
		&result.TotalActiveStudents, &result.RecordsTaken,
		&result.Present, &result.Absent, &result.Late, &result.Excused,
	)
	if err != nil {
		return nil, fmt.Errorf("get institution attendance today: %w", err)
	}
	if result.RecordsTaken > 0 {
		result.AttendanceRate = float64(result.Present+result.Late) / float64(result.RecordsTaken) * 100
	}
	return &result, nil
}

// ─── Service ──────────────────────────────────────────────────────

func (s *Service) GetInstitutionAttendanceToday(ctx context.Context, institutionID uuid.UUID, date time.Time) (*InstitutionAttendanceToday, error) {
	return s.repo.GetInstitutionAttendanceToday(ctx, institutionID, date)
}

// ─── Handler ──────────────────────────────────────────────────────

// GetInstitutionAttendanceToday handles GET /api/v1/academics/attendance/today
// Defaults to CURRENT_DATE. Accepts an optional ?date=YYYY-MM-DD override —
// same convention as GetAttendanceSheet — so this stays testable against
// seeded data without changing what "today" means in production.
func (h *Handler) GetInstitutionAttendanceToday(w http.ResponseWriter, r *http.Request) {
	institutionID := auth.InstitutionIDFromContext(r.Context())

	date := time.Now()
	if dateStr := r.URL.Query().Get("date"); dateStr != "" {
		if t, err := time.Parse("2006-01-02", dateStr); err == nil {
			date = t
		} else {
			response.Error(w, http.StatusBadRequest, "INVALID_DATE", "date must be YYYY-MM-DD")
			return
		}
	}

	summary, err := h.svc.GetInstitutionAttendanceToday(r.Context(), institutionID, date)
	if err != nil {
		log.Error().Err(err).Msg("get institution attendance today")
		response.InternalError(w)
		return
	}
	response.Ok(w, summary)
}