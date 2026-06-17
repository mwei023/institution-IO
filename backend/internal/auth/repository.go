package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"institution-os/pkg/db"
	"github.com/jackc/pgx/v5"
)

// ErrUserNotFound is returned when no user matches the query.
var ErrUserNotFound = errors.New("user not found")

// ErrInvalidCredentials is returned when credentials don't match.
var ErrInvalidCredentials = errors.New("invalid credentials")

// Repository handles all database access for the auth domain.
type Repository struct {
	db *db.Pool
}

// NewRepository creates a new auth repository.
func NewRepository(db *db.Pool) *Repository {
	return &Repository{db: db}
}

// GetUserByEmail retrieves a user and their password hash by email within an institution.
func (r *Repository) GetUserByEmail(ctx context.Context, institutionID uuid.UUID, email string) (*User, string, error) {
	query := `
		SELECT u.id, u.institution_id, u.role_id, ro.slug AS role_slug,
		       u.email, u.phone, u.first_name, u.last_name, u.other_names,
		       u.profile_photo_url, u.is_active, u.mfa_enabled,
		       u.login_attempts, u.locked_until, u.last_login_at,
		       u.must_change_password, u.created_at,
		       u.password_hash
		FROM users u
		JOIN roles ro ON ro.id = u.role_id
		WHERE u.institution_id = $1
		  AND u.email = $2
		  AND u.deleted_at IS NULL
		LIMIT 1`

	var user User
	var passwordHash string

	err := r.db.QueryRow(ctx, query, institutionID, email).Scan(
		&user.ID, &user.InstitutionID, &user.RoleID, &user.RoleSlug,
		&user.Email, &user.Phone, &user.FirstName, &user.LastName, &user.OtherNames,
		&user.ProfilePhotoURL, &user.IsActive, &user.MFAEnabled,
		&user.LoginAttempts, &user.LockedUntil, &user.LastLoginAt,
		&user.MustChangePassword, &user.CreatedAt,
		&passwordHash,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, "", ErrUserNotFound
		}
		return nil, "", fmt.Errorf("get user by email: %w", err)
	}

	return &user, passwordHash, nil
}

// GetUserByPhone retrieves a user by phone number.
func (r *Repository) GetUserByPhone(ctx context.Context, institutionID uuid.UUID, phone string) (*User, string, error) {
	query := `
		SELECT u.id, u.institution_id, u.role_id, ro.slug AS role_slug,
		       u.email, u.phone, u.first_name, u.last_name, u.other_names,
		       u.profile_photo_url, u.is_active, u.mfa_enabled,
		       u.login_attempts, u.locked_until, u.last_login_at,
		       u.must_change_password, u.created_at,
		       u.password_hash
		FROM users u
		JOIN roles ro ON ro.id = u.role_id
		WHERE u.institution_id = $1
		  AND u.phone = $2
		  AND u.deleted_at IS NULL
		LIMIT 1`

	var user User
	var passwordHash string

	err := r.db.QueryRow(ctx, query, institutionID, phone).Scan(
		&user.ID, &user.InstitutionID, &user.RoleID, &user.RoleSlug,
		&user.Email, &user.Phone, &user.FirstName, &user.LastName, &user.OtherNames,
		&user.ProfilePhotoURL, &user.IsActive, &user.MFAEnabled,
		&user.LoginAttempts, &user.LockedUntil, &user.LastLoginAt,
		&user.MustChangePassword, &user.CreatedAt,
		&passwordHash,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, "", ErrUserNotFound
		}
		return nil, "", fmt.Errorf("get user by phone: %w", err)
	}

	return &user, passwordHash, nil
}

// GetUserByID retrieves a user by ID.
func (r *Repository) GetUserByID(ctx context.Context, institutionID, userID uuid.UUID) (*User, error) {
	query := `
		SELECT u.id, u.institution_id, u.role_id, ro.slug AS role_slug,
		       u.email, u.phone, u.first_name, u.last_name, u.other_names,
		       u.profile_photo_url, u.is_active, u.mfa_enabled,
		       u.login_attempts, u.locked_until, u.last_login_at,
		       u.must_change_password, u.created_at
		FROM users u
		JOIN roles ro ON ro.id = u.role_id
		WHERE u.institution_id = $1
		  AND u.id = $2
		  AND u.deleted_at IS NULL`

	var user User
	err := r.db.QueryRow(ctx, query, institutionID, userID).Scan(
		&user.ID, &user.InstitutionID, &user.RoleID, &user.RoleSlug,
		&user.Email, &user.Phone, &user.FirstName, &user.LastName, &user.OtherNames,
		&user.ProfilePhotoURL, &user.IsActive, &user.MFAEnabled,
		&user.LoginAttempts, &user.LockedUntil, &user.LastLoginAt,
		&user.MustChangePassword, &user.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("get user by id: %w", err)
	}

	return &user, nil
}

// GetPermissionsForRole retrieves all permissions assigned to a role.
func (r *Repository) GetPermissionsForRole(ctx context.Context, roleID uuid.UUID) ([]string, error) {
	query := `
		SELECT p.resource || ':' || p.action AS permission_key
		FROM role_permissions rp
		JOIN permissions p ON p.id = rp.permission_id
		WHERE rp.role_id = $1`

	rows, err := r.db.Query(ctx, query, roleID)
	if err != nil {
		return nil, fmt.Errorf("get permissions: %w", err)
	}
	defer rows.Close()

	var perms []string
	for rows.Next() {
		var perm string
		if err := rows.Scan(&perm); err != nil {
			return nil, fmt.Errorf("scan permission: %w", err)
		}
		perms = append(perms, perm)
	}

	return perms, rows.Err()
}

// IncrementLoginAttempts increments the failed login counter.
func (r *Repository) IncrementLoginAttempts(ctx context.Context, userID uuid.UUID) error {
	_, err := r.db.Exec(ctx, `
		UPDATE users
		SET login_attempts = login_attempts + 1,
		    locked_until = CASE
		        WHEN login_attempts + 1 >= 5
		        THEN NOW() + INTERVAL '30 minutes'
		        ELSE locked_until
		    END
		WHERE id = $1`, userID)
	return err
}

// ResetLoginAttempts clears the failed login counter after successful login.
func (r *Repository) ResetLoginAttempts(ctx context.Context, userID uuid.UUID) error {
	_, err := r.db.Exec(ctx, `
		UPDATE users
		SET login_attempts = 0,
		    locked_until = NULL,
		    last_login_at = NOW()
		WHERE id = $1`, userID)
	return err
}

// UpdatePasswordHash updates the user's password hash and resets the must_change_password flag.
func (r *Repository) UpdatePasswordHash(ctx context.Context, userID uuid.UUID, hash string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE users
		SET password_hash = $1,
		    must_change_password = false,
		    password_changed_at = NOW(),
		    updated_at = NOW()
		WHERE id = $2`, hash, userID)
	return err
}

// GetPasswordHash retrieves just the password hash for a user.
func (r *Repository) GetPasswordHash(ctx context.Context, userID uuid.UUID) (string, error) {
	var hash string
	err := r.db.QueryRow(ctx,
		`SELECT password_hash FROM users WHERE id = $1 AND deleted_at IS NULL`,
		userID,
	).Scan(&hash)
	if err != nil {
		return "", fmt.Errorf("get password hash: %w", err)
	}
	return hash, nil
}

// GetPasswordHistory returns the last N password hashes for a user.
func (r *Repository) GetPasswordHistory(ctx context.Context, userID uuid.UUID, limit int) ([]string, error) {
	rows, err := r.db.Query(ctx, `
		SELECT password_hash FROM password_history
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var hashes []string
	for rows.Next() {
		var h string
		if err := rows.Scan(&h); err != nil {
			return nil, err
		}
		hashes = append(hashes, h)
	}
	return hashes, rows.Err()
}

// SavePasswordHistory records a password hash in the history table.
func (r *Repository) SavePasswordHistory(ctx context.Context, institutionID, userID uuid.UUID, hash string) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO password_history (institution_id, user_id, password_hash)
		VALUES ($1, $2, $3)`, institutionID, userID, hash)
	return err
}

// GetMFASecret retrieves the encrypted MFA secret for a user.
func (r *Repository) GetMFASecret(ctx context.Context, userID uuid.UUID) (string, error) {
	var secret string
	err := r.db.QueryRow(ctx,
		`SELECT mfa_secret FROM users WHERE id = $1 AND mfa_enabled = true`,
		userID,
	).Scan(&secret)
	if err != nil {
		return "", err
	}
	return secret, nil
}

// StoreRefreshToken saves a refresh token record.
func (r *Repository) StoreRefreshToken(ctx context.Context, institutionID, userID uuid.UUID,
	tokenHash, userAgent, ipAddress string, expiresAt time.Time) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO refresh_tokens (institution_id, user_id, token_hash, expires_at, user_agent, ip_address)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		institutionID, userID, tokenHash, expiresAt, userAgent, ipAddress)
	return err
}

// RevokeRefreshToken marks a refresh token as revoked.
func (r *Repository) RevokeRefreshToken(ctx context.Context, tokenHash string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE refresh_tokens SET revoked_at = NOW() WHERE token_hash = $1`,
		tokenHash)
	return err
}

// GetRefreshToken retrieves a refresh token record by hash.
func (r *Repository) GetRefreshToken(ctx context.Context, tokenHash string) (*uuid.UUID, *uuid.UUID, *time.Time, error) {
	var userID, institutionID uuid.UUID
	var expiresAt time.Time
	err := r.db.QueryRow(ctx, `
		SELECT user_id, institution_id, expires_at
		FROM refresh_tokens
		WHERE token_hash = $1 AND revoked_at IS NULL`,
		tokenHash,
	).Scan(&userID, &institutionID, &expiresAt)
	if err != nil {
		return nil, nil, nil, err
	}
	return &userID, &institutionID, &expiresAt, nil
}

// GetInstitutionBySlug retrieves an institution by its URL slug.
func (r *Repository) GetInstitutionBySlug(ctx context.Context, slug string) (*uuid.UUID, error) {
	var id uuid.UUID
	err := r.db.QueryRow(ctx,
		`SELECT id FROM institutions WHERE school_slug = $1 AND is_active = true`,
		slug,
	).Scan(&id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &id, nil
}
