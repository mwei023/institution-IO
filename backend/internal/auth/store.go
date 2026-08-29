package auth

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Store is everything auth.Service needs from a repository. service.go
// already expected a type with this name (NewService(repo Store, ...))
// but the interface itself was never defined — this file completes
// that. The real, database-backed Repository struct already has every
// one of these methods (MockAuthRepo in auth_test.go implements the
// same set), so *Repository satisfies Store automatically.
type Store interface {
	GetUserByEmail(ctx context.Context, institutionID uuid.UUID, email string) (*User, string, error)
	GetUserByPhone(ctx context.Context, institutionID uuid.UUID, phone string) (*User, string, error)
	GetUserByID(ctx context.Context, institutionID, userID uuid.UUID) (*User, error)
	GetPermissionsForRole(ctx context.Context, roleID uuid.UUID) ([]string, error)

	IncrementLoginAttempts(ctx context.Context, userID uuid.UUID) error
	ResetLoginAttempts(ctx context.Context, userID uuid.UUID) error

	UpdatePasswordHash(ctx context.Context, userID uuid.UUID, hash string) error
	GetPasswordHash(ctx context.Context, userID uuid.UUID) (string, error)
	GetPasswordHistory(ctx context.Context, userID uuid.UUID, limit int) ([]string, error)
	SavePasswordHistory(ctx context.Context, institutionID, userID uuid.UUID, hash string) error

	GetMFASecret(ctx context.Context, userID uuid.UUID) (string, error)

	StoreRefreshToken(ctx context.Context, institutionID, userID uuid.UUID, tokenHash, userAgent, ipAddress string, expiresAt time.Time) error
	RevokeRefreshToken(ctx context.Context, tokenHash string) error
	GetRefreshToken(ctx context.Context, tokenHash string) (*uuid.UUID, *uuid.UUID, *time.Time, error)

	GetInstitutionBySlug(ctx context.Context, slug string) (*uuid.UUID, error)
}