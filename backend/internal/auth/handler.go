package auth

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"
	"institution-os/pkg/response"
	"github.com/rs/zerolog/log"
)

// Handler handles all auth HTTP endpoints.
type Handler struct {
	svc *Service
}

// NewHandler creates a new auth handler.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// Login handles POST /api/v1/auth/login
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_BODY", "Invalid request body")
		return
	}

	slug := r.Header.Get("X-Institution-Slug")
	if slug == "" {
		response.Error(w, http.StatusBadRequest, "MISSING_INSTITUTION", "X-Institution-Slug header is required")
		return
	}

	institutionID, err := h.svc.repo.GetInstitutionBySlug(r.Context(), slug)
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "INVALID_INSTITUTION", "Institution not found")
		return
	}

	resp, err := h.svc.Login(r.Context(), *institutionID, &req, r.UserAgent(), r.RemoteAddr)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidCredentials):
			response.Error(w, http.StatusUnauthorized, "INVALID_CREDENTIALS", "Invalid email or password")
		case errors.Is(err, ErrAccountLocked):
			response.Error(w, http.StatusTooManyRequests, "ACCOUNT_LOCKED",
				"Account temporarily locked. Try again in 30 minutes.")
		case errors.Is(err, ErrAccountInactive):
			response.Error(w, http.StatusForbidden, "ACCOUNT_INACTIVE",
				"Your account has been deactivated. Contact your administrator.")
		default:
			log.Error().Err(err).Msg("login error")
			response.InternalError(w)
		}
		return
	}

	if resp.MFARequired {
		response.Ok(w, resp)
		return
	}

	response.Ok(w, resp)
}

// RefreshToken handles POST /api/v1/auth/refresh
func (h *Handler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	response.Error(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "Token refresh not yet implemented")
}

// Logout handles POST /api/v1/auth/logout
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    "",
		Path:     "/api/v1/auth",
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	})
	response.NoContent(w)
}

// VerifyMFA handles POST /api/v1/auth/mfa/verify
func (h *Handler) VerifyMFA(w http.ResponseWriter, r *http.Request) {
	var req MFAVerifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_BODY", "Invalid request body")
		return
	}

	slug := r.Header.Get("X-Institution-Slug")
	institutionID, err := h.svc.repo.GetInstitutionBySlug(r.Context(), slug)
	if err != nil {
		response.Unauthorized(w)
		return
	}

	resp, err := h.svc.VerifyMFA(r.Context(), *institutionID, &req, r.UserAgent(), r.RemoteAddr)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidMFACode), errors.Is(err, ErrInvalidToken):
			response.Error(w, http.StatusUnauthorized, "INVALID_MFA_CODE", "Invalid or expired MFA code")
		default:
			log.Error().Err(err).Msg("mfa verify error")
			response.InternalError(w)
		}
		return
	}

	response.Ok(w, resp)
}

// Me handles GET /api/v1/auth/me
func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	claims := ClaimsFromContext(r.Context())
	if claims == nil {
		response.Unauthorized(w)
		return
	}
	response.Ok(w, claims)
}

// ChangePassword handles POST /api/v1/auth/change-password
func (h *Handler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	claims := ClaimsFromContext(r.Context())
	if claims == nil {
		response.Unauthorized(w)
		return
	}

	var req ChangePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_BODY", "Invalid request body")
		return
	}

	institutionID := InstitutionIDFromContext(r.Context())
	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		response.InternalError(w)
		return
	}

	if err := h.svc.ChangePassword(r.Context(), institutionID, userID, &req); err != nil {
		switch {
		case errors.Is(err, ErrInvalidCredentials):
			response.Error(w, http.StatusUnauthorized, "WRONG_PASSWORD", "Current password is incorrect")
		case errors.Is(err, ErrPasswordReused):
			response.Error(w, http.StatusUnprocessableEntity, "PASSWORD_REUSED",
				"You cannot reuse a recent password")
		default:
			log.Error().Err(err).Msg("change password error")
			response.InternalError(w)
		}
		return
	}

	response.NoContent(w)
}
