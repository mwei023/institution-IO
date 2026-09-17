package auth

import (
	"context"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"institution-os/pkg/response"
)

type contextKey string

const (
	claimsKey      contextKey = "claims"
	institutionKey contextKey = "institution_id"
)

// JWTMiddleware validates the Bearer token in the Authorization header,
// falling back to the httpOnly access_token cookie (pilot P0-4: lets the
// frontend drop localStorage without a backend change).
func JWTMiddleware(svc *Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := ""
			if h := r.Header.Get("Authorization"); h != "" {
				parts := strings.SplitN(h, " ", 2)
				if len(parts) == 2 && strings.EqualFold(parts[0], "bearer") {
					token = parts[1]
				}
			}
			if token == "" {
				if c, err := r.Cookie("access_token"); err == nil {
					token = c.Value
				}
			}
			if token == "" {
				response.Unauthorized(w)
				return
			}

			claims, err := svc.ParseAccessToken(token)
			if err != nil {
				response.Unauthorized(w)
				return
			}

			// Inject claims and institution_id into context
			institutionID, err := uuid.Parse(claims.InstitutionID)
			if err != nil {
				response.Unauthorized(w)
				return
			}

			ctx := context.WithValue(r.Context(), claimsKey, claims)
			ctx = context.WithValue(ctx, institutionKey, institutionID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequirePermission returns middleware that enforces a specific permission.
func RequirePermission(perm string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := ClaimsFromContext(r.Context())
			if claims == nil || !claims.HasPermission(perm) {
				response.Forbidden(w)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RequireRole returns middleware that enforces a specific role.
func RequireRole(roles ...string) func(http.Handler) http.Handler {
	allowed := make(map[string]bool)
	for _, r := range roles {
		allowed[r] = true
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := ClaimsFromContext(r.Context())
			if claims == nil || !allowed[claims.Role] {
				response.Forbidden(w)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// ClaimsFromContext extracts JWT claims from the request context.
func ClaimsFromContext(ctx context.Context) *Claims {
	claims, _ := ctx.Value(claimsKey).(*Claims)
	return claims
}

// InstitutionIDFromContext extracts the institution ID from context.
func InstitutionIDFromContext(ctx context.Context) uuid.UUID {
	id, _ := ctx.Value(institutionKey).(uuid.UUID)
	return id
}

// UserIDFromContext extracts the user ID string from context.
func UserIDFromContext(ctx context.Context) string {
	claims := ClaimsFromContext(ctx)
	if claims == nil {
		return ""
	}
	return claims.UserID
}
