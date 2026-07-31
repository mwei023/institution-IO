package cbc

import (
	"context"
	"fmt"
	"sort"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"institution-os/internal/academics"
	"institution-os/pkg/events"
)

// Service implements CBC/CBE academic layer business logic.
//
// Depends directly on *academics.Repository for attendance and class
// data — deliberate, not an oversight: attendance is curriculum-
// agnostic and already correct/tested there. See package doc comment
// in model.go for the full separation rationale (assessment/report
// cards are NOT shared; attendance/class lookups ARE).
type Service struct {
	repo          *Repository
	academicsRepo *academics.Repository
	publisher     events.Publisher
}

func NewService(repo *Repository, academicsRepo *academics.Repository, publisher events.Publisher) *Service {
	return &Service{repo: repo, academicsRepo: academicsRepo, publisher: publisher}
}

// ─── Learning Areas ─────────────────────────────────────────────────

func (s *Service) CreateLearningArea(ctx context.Context, institutionID uuid.UUID, req *CreateLearningAreaRequest) (*LearningArea, error) {
	return s.repo.CreateLearningArea(ctx, institutionID, req)
}

func (s *Service) ListLearningAreas(ctx context.Context, institutionID uuid.UUID) ([]*LearningArea, error) {
	return s.repo.ListLearningAreas(ctx, institutionID)
}

// ─── Assessment components & entries ────────────────────────────────

func (s *Service) CreateComponent(ctx context.Context, institutionID uuid.UUID, req *CreateComponentRequest) (*Component, error) {
	if req.Weight <= 0 || req.Weight > 100 {
		return nil, fmt.Errorf("weight must be between 1 and 100")
	}
	if req.MaxMarks <= 0 {
		return nil, fmt.Errorf("max_marks must be greater than 0")
	}
	return s.repo.CreateComponent(ctx, institutionID, req)
}

func (s *Service) ListComponents(ctx context.Context, institutionID, classID, learningAreaID, termID uuid.UUID) ([]*Component, error) {
	return s.repo.ListComponents(ctx, institutionID, classID, learningAreaID, termID)
}

func (s *Service) GetEntries(ctx context.Context, institutionID, componentID uuid.UUID) ([]*Entry, error) {
	return s.repo.GetEntries(ctx, institutionID, componentID)
}

func (s *Service) SaveEntries(ctx context.Context, institutionID, teacherID uuid.UUID, req *BulkEntryRequest) error {
	if len(req.Entries) == 0 {
		return fmt.Errorf("no entries provided")
	}
	return s.repo.SaveEntries(ctx, institutionID, req.ComponentID, teacherID, req.Entries)
}

// ─── Teacher assignments ─────────────────────────────────────────────

func (s *Service) CreateTeacherAssignment(ctx context.Context, institutionID, classID, learningAreaID, teacherID, termID uuid.UUID) (*TeacherAssignment, error) {
	return s.repo.CreateTeacherAssignment(ctx, institutionID, classID, learningAreaID, teacherID, termID)
}

func (s *Service) TeacherHasLearningAreaAssignment(ctx context.Context, institutionID, teacherID, classID uuid.UUID) (bool, error) {
	return s.repo.TeacherHasLearningAreaAssignment(ctx, institutionID, teacherID, classID)
}

// ─── Report card generation ──────────────────────────────────────────

// GenerateReportCardsForClass generates (or regenerates) a CBC report
// card for every active student in a class/term. Mirrors
// academics.GenerateReportCardsForClass's partial-success semantics:
// every student is processed independently, one student's failure
// (no entries for any learning area yet) is recorded and skipped, it
// never aborts the rest of the batch.
//
// curriculumType and gradeBand are derived internally rather than
// passed by the caller — curriculumType from the institution's own
// setting, gradeBand from the class's Level (7-9 = Junior School,
// 10-12 = Senior School). Deliberate: the handler having to correctly
// pass these every time was a real source of drift; deriving them
// here means there's exactly one place this logic can be wrong.
func (s *Service) GenerateReportCardsForClass(ctx context.Context, institutionID, classID, termID uuid.UUID) (*BulkReportCardResult, error) {
	class, err := s.academicsRepo.GetClassByID(ctx, institutionID, classID)
	if err != nil {
		return nil, err
	}

	curriculumType, err := s.repo.GetInstitutionCurriculumType(ctx, institutionID)
	if err != nil {
		return nil, err
	}

	var gradeBand *string
	switch {
	case class.Level >= 7 && class.Level <= 9:
		jb := "junior_school"
		gradeBand = &jb
	case class.Level >= 10 && class.Level <= 12:
		sb := "senior_school"
		gradeBand = &sb
	}
	// class.Level outside 7-12 (e.g. a CBE primary class, PP1-Grade 6)
	// leaves gradeBand nil — GetRatingBands will look for bands with
	// grade_band IS NULL, which isn't seeded yet in the v1.0 migration.
	// That's a real, known gap: only Junior/Senior School bands exist
	// today, matching the two report cards actually seen so far.

	studentIDs, err := s.academicsRepo.GetActiveStudentIDsForClass(ctx, institutionID, classID)
	if err != nil {
		return nil, fmt.Errorf("get class roster: %w", err)
	}

	bands, err := s.repo.GetRatingBands(ctx, institutionID, curriculumType, gradeBand)
	if err != nil {
		return nil, fmt.Errorf("get rating bands: %w", err)
	}

	learningAreas, err := s.repo.ListLearningAreas(ctx, institutionID)
	if err != nil {
		return nil, fmt.Errorf("list learning areas: %w", err)
	}
	learningAreaNames := make(map[uuid.UUID]string, len(learningAreas))
	for _, la := range learningAreas {
		learningAreaNames[la.ID] = la.Name
	}

	termStart, termEnd, err := s.academicsRepo.GetTermDates(ctx, institutionID, termID)
	if err != nil {
		return nil, fmt.Errorf("get term dates: %w", err)
	}
	attendance, err := s.academicsRepo.GetAttendanceSummary(ctx, institutionID, classID, termStart, termEnd)
	if err != nil {
		return nil, fmt.Errorf("get attendance summary: %w", err)
	}

	// Compute every student's mean points once, up front, so ranking
	// doesn't require re-querying per student — same pattern as
	// academics.GetClassRankings being computed once and reused.
	type studentComputation struct {
		studentID  uuid.UUID
		meanPoints float64
		lines      []*ReportCardLine
	}
	var computations []*studentComputation
	failures := []BulkFailure{}

	for _, studentID := range studentIDs {
		percentages, err := s.repo.GetAllLearningAreaPercentagesForStudent(ctx, institutionID, studentID, termID)
		if err != nil {
			failures = append(failures, BulkFailure{StudentID: studentID, Reason: err.Error()})
			continue
		}
		if len(percentages) == 0 {
			failures = append(failures, BulkFailure{StudentID: studentID, Reason: "no graded learning areas found for this student in this term"})
			continue
		}

		var lines []*ReportCardLine
		var totalPoints float64
		for laID, pct := range percentages {
			band, err := bandForPercentage(bands, pct)
			if err != nil {
				log.Warn().Err(err).Str("student_id", studentID.String()).Str("learning_area_id", laID.String()).
					Msg("skipping learning area line — no matching band")
				continue
			}
			statement := band.DefaultStatement
			lines = append(lines, &ReportCardLine{
				LearningAreaID:   laID,
				LearningAreaName: learningAreaNames[laID],
				Percentage:       pct,
				RatingBandID:     &band.ID,
				RatingCode:       band.Code,
				RatingLabel:      band.Label,
				RatingPoints:     band.Points,
				Statement:        statement,
			})
			totalPoints += band.Points
		}
		if len(lines) == 0 {
			failures = append(failures, BulkFailure{StudentID: studentID, Reason: "no learning area matched a configured rating band"})
			continue
		}

		computations = append(computations, &studentComputation{
			studentID:  studentID,
			meanPoints: totalPoints / float64(len(lines)),
			lines:      lines,
		})
	}

	// Rank by mean points, descending — ties share the same rank
	// (standard "competition ranking", same convention as
	// academics.GetClassRankings' RANK() OVER (ORDER BY ... DESC)).
	sort.Slice(computations, func(i, j int) bool {
		return computations[i].meanPoints > computations[j].meanPoints
	})
	ranks := make(map[uuid.UUID]int, len(computations))
	for i, c := range computations {
		rank := i + 1
		if i > 0 && computations[i-1].meanPoints == c.meanPoints {
			rank = ranks[computations[i-1].studentID]
		}
		ranks[c.studentID] = rank
	}

	result := &BulkReportCardResult{
		ClassID:       classID,
		TermID:        termID,
		TotalStudents: len(studentIDs),
		Succeeded:     make([]*ReportCard, 0, len(computations)),
		Failed:        failures,
	}

	for _, c := range computations {
		var present, total int
		for _, a := range attendance {
			if a.StudentID == c.studentID {
				present = a.Present + a.Late
				total = a.Total
				break
			}
		}

		meanPoints := c.meanPoints
		totalPointsVal := c.meanPoints * float64(len(c.lines))
		rank := ranks[c.studentID]

		rc := &ReportCard{
			StudentID:             c.studentID,
			TotalPoints:           &totalPointsVal,
			MeanPoints:            &meanPoints,
			ClassPosition:         &rank,
			AttendanceDaysPresent: present,
			AttendanceDaysTotal:   total,
			Status:                "draft",
			Lines:                 c.lines,
		}

		if err := s.repo.UpsertReportCard(ctx, institutionID, classID, termID, rc); err != nil {
			log.Warn().Err(err).Str("student_id", c.studentID.String()).Msg("skipping cbc report card save")
			result.Failed = append(result.Failed, BulkFailure{StudentID: c.studentID, Reason: err.Error()})
			continue
		}
		result.Succeeded = append(result.Succeeded, rc)
	}

	result.SucceededCount = len(result.Succeeded)
	result.FailedCount = len(result.Failed)
	return result, nil
}

func (s *Service) PublishReportCard(ctx context.Context, institutionID, studentID, termID uuid.UUID) (*ReportCard, error) {
	return s.repo.PublishReportCard(ctx, institutionID, studentID, termID)
}

func (s *Service) UnpublishReportCard(ctx context.Context, institutionID, studentID, termID uuid.UUID) (*ReportCard, error) {
	return s.repo.UnpublishReportCard(ctx, institutionID, studentID, termID)
}

func (s *Service) GetChildReportCard(ctx context.Context, institutionID, studentID uuid.UUID) (*ReportCard, error) {
	return s.repo.GetPublishedReportCardForStudent(ctx, institutionID, studentID)
}