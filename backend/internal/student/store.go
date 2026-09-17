package student

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Store is everything student.Service needs from a repository — same
// idea as academics.Store. The real, database-backed Repository struct
// already has every one of these methods, so *Repository satisfies
// Store automatically. This interface exists only so tests can swap in
// a fake implementation instead of a real database connection.
type Store interface {
	CreateAdmission(ctx context.Context, a *Admission, parentJSON []byte) error
	GetAdmissionByID(ctx context.Context, institutionID, admissionID uuid.UUID) (*Admission, error)
	ListAdmissions(ctx context.Context, institutionID uuid.UUID, status string, page, pageSize int) ([]*Admission, int64, error)
	UpdateAdmissionStatus(ctx context.Context, institutionID, admissionID, reviewerID uuid.UUID, status, notes string) error
	CreateStudent(ctx context.Context, s *Student) error
	GetStudentByID(ctx context.Context, institutionID, studentID uuid.UUID) (*Student, error)
	ListStudents(ctx context.Context, institutionID uuid.UUID, f *StudentFilter) ([]*StudentListItem, int64, error)
	GetTimeline(ctx context.Context, institutionID, studentID uuid.UUID) ([]*TimelineEvent, error)
	AddTimelineEvent(ctx context.Context, institutionID, studentID uuid.UUID,
		eventType, title, description string, eventDate time.Time) error
	GetNextApplicationNumber(ctx context.Context, institutionID uuid.UUID, year string) (string, error)
	GetNextAdmissionNumber(ctx context.Context, institutionID uuid.UUID, year, classCode string) (string, error)
	GetStudentParents(ctx context.Context, institutionID, studentID uuid.UUID) ([]*ParentGuardian, error)
	GetChildrenForParent(ctx context.Context, institutionID, parentUserID uuid.UUID) ([]*ChildSummary, error)
}
