package institution

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"institution-os/pkg/db"
)

var ErrNotFound = errors.New("institution not found")

type Repository struct {
	db *db.Pool
}

func NewRepository(db *db.Pool) *Repository {
	return &Repository{db: db}
}

// GetCurriculumType returns the institution's curriculum_type ('844' or 'cbc').
func (r *Repository) GetCurriculumType(ctx context.Context, institutionID uuid.UUID) (string, error) {
	var curriculumType string
	err := r.db.QueryRow(ctx,
		`SELECT curriculum_type FROM institutions WHERE id = $1`,
		institutionID,
	).Scan(&curriculumType)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrNotFound
		}
		return "", fmt.Errorf("get curriculum type: %w", err)
	}
	return curriculumType, nil
}