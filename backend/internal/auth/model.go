package auth

import (
	"time"

	"github.com/google/uuid"
)

// User is the full user record from the database.
type User struct {
	ID                   uuid.UUID  `json:"id" db:"id"`
	InstitutionID        uuid.UUID  `json:"institution_id" db:"institution_id"`
	RoleID               uuid.UUID  `json:"role_id" db:"role_id"`
	RoleSlug             string     `json:"role" db:"role_slug"`
	Email                *string    `json:"email,omitempty" db:"email"`
	Phone                *string    `json:"phone,omitempty" db:"phone"`
	FirstName            string     `json:"first_name" db:"first_name"`
	LastName             string     `json:"last_name" db:"last_name"`
	OtherNames           *string    `json:"other_names,omitempty" db:"other_names"`
	ProfilePhotoURL      *string    `json:"profile_photo_url,omitempty" db:"profile_photo_url"`
	IsActive             bool       `json:"is_active" db:"is_active"`
	MFAEnabled           bool       `json:"mfa_enabled" db:"mfa_enabled"`
	LoginAttempts        int        `json:"-" db:"login_attempts"`
	LockedUntil          *time.Time `json:"-" db:"locked_until"`
	LastLoginAt          *time.Time `json:"last_login_at,omitempty" db:"last_login_at"`
	MustChangePassword   bool       `json:"-" db:"must_change_password"`
	CreatedAt            time.Time  `json:"created_at" db:"created_at"`
}

// Role is a role record.
type Role struct {
	ID            uuid.UUID `json:"id" db:"id"`
	InstitutionID uuid.UUID `json:"institution_id" db:"institution_id"`
	Name          string    `json:"name" db:"name"`
	Slug          string    `json:"slug" db:"slug"`
	IsSystem      bool      `json:"is_system" db:"is_system"`
}

// Permission represents a single resource:action permission.
type Permission struct {
	ID       uuid.UUID `db:"id"`
	Resource string    `db:"resource"`
	Action   string    `db:"action"`
	Key      string    // computed: resource:action
}

// LoginRequest is the request body for the login endpoint.
type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
}

// LoginResponse is returned on successful authentication.
type LoginResponse struct {
	AccessToken      string `json:"access_token"`
	TokenType        string `json:"token_type"`
	ExpiresIn        int    `json:"expires_in"` // seconds
	MFARequired      bool   `json:"mfa_required,omitempty"`
	MFAToken         string `json:"mfa_token,omitempty"` // short-lived, for MFA flow only
	MustChangePassword bool  `json:"must_change_password,omitempty"`
	User             *UserProfile `json:"user,omitempty"`
}

// UserProfile is the safe user representation (no password hash).
type UserProfile struct {
	ID              uuid.UUID `json:"id"`
	InstitutionID   uuid.UUID `json:"institution_id"`
	Role            string    `json:"role"`
	FullName        string    `json:"full_name"`
	Email           *string   `json:"email,omitempty"`
	Phone           *string   `json:"phone,omitempty"`
	ProfilePhotoURL *string   `json:"profile_photo_url,omitempty"`
	Permissions     []string  `json:"permissions"`
}

// RefreshTokenRequest is the body (or cookie) for token refresh.
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// ChangePasswordRequest is the body for password change.
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" validate:"required"`
	NewPassword     string `json:"new_password" validate:"required,min=8"`
}

// ResetPasswordRequest is the body to initiate a password reset.
type ResetPasswordRequest struct {
	Email string `json:"email" validate:"required,email"`
}

// ConfirmResetPasswordRequest is the body to confirm a password reset.
type ConfirmResetPasswordRequest struct {
	Token       string `json:"token" validate:"required"`
	NewPassword string `json:"new_password" validate:"required,min=8"`
}

// MFAVerifyRequest is the body to verify a TOTP code.
type MFAVerifyRequest struct {
	MFAToken string `json:"mfa_token" validate:"required"`
	TOTPCode string `json:"totp_code" validate:"required,len=6"`
}

// Claims are the JWT payload claims.
type Claims struct {
	UserID        string   `json:"uid"`
	InstitutionID string   `json:"iid"`
	Role          string   `json:"role"`
	Permissions   []string `json:"perms"`
}

// HasPermission returns true if the claims include the given permission.
func (c *Claims) HasPermission(perm string) bool {
	for _, p := range c.Permissions {
		if p == perm || p == "*" {
			return true
		}
	}
	return false
}
