package exams

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Store is everything Service needs. *Repository satisfies it; tests use
// an in-memory fake.
type Store interface {
	CreateSession(ctx context.Context, institutionID uuid.UUID, createdBy *uuid.UUID, req *CreateSessionRequest) (*ExamSession, error)
	GetSession(ctx context.Context, institutionID, sessionID uuid.UUID) (*ExamSession, error)
	ListSessions(ctx context.Context, institutionID uuid.UUID) ([]*ExamSession, error)
	SetSessionStatus(ctx context.Context, institutionID, sessionID uuid.UUID, status string, version *int) (*ExamSession, error)

	CreatePaper(ctx context.Context, institutionID, sessionID uuid.UUID, req *CreatePaperRequest) (*ExamPaper, error)
	ListPapers(ctx context.Context, institutionID, sessionID uuid.UUID) ([]*ExamPaper, error)
	GetPaper(ctx context.Context, institutionID, paperID uuid.UUID) (*ExamPaper, error)

	AddPaperGroup(ctx context.Context, institutionID, paperID uuid.UUID, req *AddPaperGroupRequest) (*PaperGroup, error)
	ListGroupsForPapers(ctx context.Context, institutionID uuid.UUID, paperIDs []uuid.UUID) (map[uuid.UUID][]*PaperGroup, error)

	CreateSitting(ctx context.Context, institutionID, sessionID uuid.UUID, req *CreateSittingRequest) (*ExamSitting, error)
	UpdateSitting(ctx context.Context, institutionID, sittingID uuid.UUID, req *UpdateSittingRequest) (*ExamSitting, error)
	DeleteSitting(ctx context.Context, institutionID, sittingID uuid.UUID) error
	ListSittings(ctx context.Context, institutionID, sessionID uuid.UUID) ([]*ExamSitting, error)
	GetSitting(ctx context.Context, institutionID, sittingID uuid.UUID) (*ExamSitting, error)

	AssignInvigilator(ctx context.Context, institutionID, sittingID uuid.UUID, req *AssignInvigilatorRequest) (*InvigilatorAssignment, error)
	RemoveInvigilator(ctx context.Context, institutionID, sittingID, staffID uuid.UUID) error
	ListInvigilators(ctx context.Context, institutionID uuid.UUID, sittingIDs []uuid.UUID) (map[uuid.UUID][]*InvigilatorAssignment, error)

	AddArrangement(ctx context.Context, institutionID, sittingID uuid.UUID, req *AddArrangementRequest) (*SpecialArrangement, error)
	ListArrangements(ctx context.Context, institutionID uuid.UUID, sittingIDs []uuid.UUID) (map[uuid.UUID][]*SpecialArrangement, error)

	ClassExists(ctx context.Context, institutionID, classID uuid.UUID) (bool, error)
	OfferingExists(ctx context.Context, institutionID, offeringID uuid.UUID) (bool, error)
	OfferingName(ctx context.Context, institutionID, offeringID uuid.UUID) (name, typ string, err error)
	StaffExists(ctx context.Context, institutionID, staffID uuid.UUID) (bool, error)
	ClassName(ctx context.Context, institutionID, classID uuid.UUID) (string, error)
	StudentCountForClass(ctx context.Context, institutionID, classID uuid.UUID) (int, error)
	Now() time.Time
}
