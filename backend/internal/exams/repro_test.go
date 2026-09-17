package exams

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"institution-os/pkg/db"
	"institution-os/pkg/config"
)

func TestReproUpdateSitting(t *testing.T) {
	cfg, err := config.Load("../../.env")
	if err != nil {
		t.Fatalf("config: %v", err)
	}
	ctx := context.Background()
	pool, err := db.Connect(ctx, cfg)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer pool.Close()
	r := NewRepository(pool)
	instID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	_ = instID
	// find any sitting
	var sitID uuid.UUID
	var institutionID uuid.UUID
	err = pool.QueryRow(ctx, `SELECT id, institution_id FROM exam_sittings WHERE deleted_at IS NULL LIMIT 1`).Scan(&sitID, &institutionID)
	if err != nil {
		t.Fatalf("no sitting: %v", err)
	}
	cap := 40
	cand := 5
	// case 1: capacity-only update (what the frontend Save sends)
	_, err = r.UpdateSitting(ctx, institutionID, sitID, &UpdateSittingRequest{RoomCapacity: &cap, CandidateCount: &cand})
	t.Logf("capacity-only update err: %v", err)
	// case 2: date update
	d := "2026-09-20"
	_, err = r.UpdateSitting(ctx, institutionID, sitID, &UpdateSittingRequest{ExamDate: &d})
	t.Logf("date update err: %v", err)
}
