package institution

import (
	"context"

	"github.com/google/uuid"
)

// Context is the response payload for GET /institution/context. The
// sidebar, gradebook, and report-card frontend code should all derive
// their curriculum-specific behavior from this single call rather than
// checking curriculum_type ad hoc in 50 places.
//
// v1.0: grading_scheme/report_card/ranking_allowed are derived purely
// from curriculum_type below. When v1.1's Curriculum Engine introduces
// per-institution or per-grade-band overrides, this is the seam where
// that logic replaces the switch statement — the response shape doesn't
// need to change, only how it's computed.
type Context struct {
	InstitutionID  string `json:"institution_id"`
	CurriculumType string `json:"curriculum_type"`
	GradingScheme  string `json:"grading_scheme"`
	ReportCard     string `json:"report_card"`
	RankingAllowed bool   `json:"ranking_allowed"`
}

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) GetContext(ctx context.Context, institutionID uuid.UUID) (*Context, error) {
	curriculumType, err := s.repo.GetCurriculumType(ctx, institutionID)
	if err != nil {
		return nil, err
	}

	c := &Context{
		InstitutionID:  institutionID.String(),
		CurriculumType: curriculumType,
	}

	switch curriculumType {
	case "cbc":
		c.GradingScheme = "ME/AE/BE"
		c.ReportCard = "cbc_v1"
		c.RankingAllowed = false
	default: // "844"
		c.GradingScheme = "percentage"
		c.ReportCard = "844_v1"
		c.RankingAllowed = true
	}

	return c, nil
}