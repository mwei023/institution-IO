package institution

import (
	"net/http"

	"institution-os/internal/auth"
	"institution-os/pkg/response"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// GetContext returns the requesting user's institution curriculum context.
// Deliberately no permission gate: every authenticated user (any role)
// needs this to render correct navigation and screens, same as /auth/me.
func (h *Handler) GetContext(w http.ResponseWriter, r *http.Request) {
	institutionID := auth.InstitutionIDFromContext(r.Context())

	ctx, err := h.svc.GetContext(r.Context(), institutionID)
	if err != nil {
		response.InternalError(w)
		return
	}

	response.Ok(w, ctx)
}