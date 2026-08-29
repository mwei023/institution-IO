package cbc

import (
	"context"

	"github.com/google/uuid"
)

// Store is everything cbc.Service needs from a repository. The real,
// database-backed Repository struct in repository.go already has every
// one of these methods — so *Repository satisfies Store automatically,
// with no changes needed to repository.go. This interface exists only
// so tests can swap in a fake/mock implementation instead of a real
// database connection.
type Store interface {
	GetInstitutionCurriculumType(ctx context.Context, institutionID uuid.UUID) (string, error)

	CreateLearningArea(ctx context.Context, institutionID uuid.UUID, req *CreateLearningAreaRequest) (*LearningArea, error)
	ListLearningAreas(ctx context.Context, institutionID uuid.UUID) ([]*LearningArea, error)

	CreateComponent(ctx context.Context, institutionID uuid.UUID, req *CreateComponentRequest) (*Component, error)
	ListComponents(ctx context.Context, institutionID, classID, learningAreaID, termID uuid.UUID) ([]*Component, error)
	GetComponentClassID(ctx context.Context, institutionID, componentID uuid.UUID) (uuid.UUID, error)

	GetEntries(ctx context.Context, institutionID, componentID uuid.UUID) ([]*Entry, error)
	SaveEntries(ctx context.Context, institutionID, componentID, enteredBy uuid.UUID, entries []EntryInput) error

	GetAllLearningAreaPercentagesForStudent(ctx context.Context, institutionID, studentID, termID uuid.UUID) (map[uuid.UUID]float64, error)

	CreateTeacherAssignment(ctx context.Context, institutionID uuid.UUID, classID, learningAreaID, teacherID, termID uuid.UUID) (*TeacherAssignment, error)
	TeacherHasLearningAreaAssignment(ctx context.Context, institutionID, teacherID, classID uuid.UUID) (bool, error)

	GetRatingBands(ctx context.Context, institutionID uuid.UUID, curriculumType string, gradeBand *string) ([]*RatingBand, error)
	GetStudentNames(ctx context.Context, institutionID uuid.UUID, studentIDs []uuid.UUID) (map[uuid.UUID]string, error)

	UpsertReportCard(ctx context.Context, institutionID, classID, termID uuid.UUID, rc *ReportCard) error
	PublishReportCard(ctx context.Context, institutionID, studentID, termID uuid.UUID) (*ReportCard, error)
	UnpublishReportCard(ctx context.Context, institutionID, studentID, termID uuid.UUID) (*ReportCard, error)
	GetPublishedReportCardForStudent(ctx context.Context, institutionID, studentID uuid.UUID) (*ReportCard, error)
}