package tests

// Integration probe for the nationality persistence fix: the request's
// nationality must survive CreateAdmission's INSERT and come back through
// GetAdmissionByID, so enrollment can carry it onto the student record.
// A column/placeholder mismatch in the INSERT would only ever surface
// against a real database — mocks can't catch it.

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"institution-os/internal/student"
	"institution-os/pkg/db"
)

func TestIntegration_AdmissionNationality_RoundTrip(t *testing.T) {
	pool := intDB(t)
	ctx := context.Background()

	instID := uuid.New()
	_, err := pool.Exec(ctx, `
		INSERT INTO institutions (id, name, school_slug, settings)
		VALUES ($1, $2, $3, '{}')`,
		instID, "Admission Nationality Test "+instID.String()[:8], "anat-"+instID.String()[:13])
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM admissions WHERE institution_id = $1`, instID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM institutions WHERE id = $1`, instID)
	})

	repo := student.NewRepository(&db.Pool{Pool: pool})

	// Explicit non-default nationality must round-trip.
	admission := &student.Admission{
		ID:                uuid.New(),
		InstitutionID:     instID,
		ApplicationNumber: "APP-IT-" + instID.String()[:8],
		AcademicYear:      "2026",
		IntendedClass:     "Form 1",
		FirstName:         "Amara",
		LastName:          "Odhiambo",
		DateOfBirth:       time.Date(2009, 3, 15, 0, 0, 0, 0, time.UTC),
		Gender:            "F",
		Nationality:       "Tanzanian",
		Status:            "submitted",
	}
	require.NoError(t, repo.CreateAdmission(ctx, admission, []byte(`[]`)))

	got, err := repo.GetAdmissionByID(ctx, instID, admission.ID)
	require.NoError(t, err)
	assert.Equal(t, "Tanzanian", got.Nationality, "explicit nationality must survive the INSERT")

	// Blank value must fall back to the column default ('Kenyan'), never
	// fail the NOT NULL constraint or store an empty string.
	admission2 := &student.Admission{
		ID:                uuid.New(),
		InstitutionID:     instID,
		ApplicationNumber: "APP-IT2-" + instID.String()[:8],
		AcademicYear:      "2026",
		IntendedClass:     "Form 1",
		FirstName:         "Blank",
		LastName:          "Nationality",
		DateOfBirth:       time.Date(2009, 3, 15, 0, 0, 0, 0, time.UTC),
		Gender:            "M",
		Nationality:       "",
		Status:            "submitted",
	}
	require.NoError(t, repo.CreateAdmission(ctx, admission2, []byte(`[]`)))

	got2, err := repo.GetAdmissionByID(ctx, instID, admission2.ID)
	require.NoError(t, err)
	assert.Equal(t, "Kenyan", got2.Nationality, "blank application value takes the DB column default")

	// List must expose it too (the service layer feeds the admin UI from here).
	list, total, err := repo.ListAdmissions(ctx, instID, "", 1, 50)
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	found := map[uuid.UUID]string{}
	for _, a := range list {
		found[a.ID] = a.Nationality
	}
	assert.Equal(t, "Tanzanian", found[admission.ID])
	assert.Equal(t, "Kenyan", found[admission2.ID])
}
