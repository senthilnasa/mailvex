package warmup

import (
	"context"
	"testing"

	"github.com/senthilnasa/mailvex/core/repository"
)

func TestWarmupManagerLifecycle(t *testing.T) {
	repo := repository.NewMemoryRepository()
	manager := NewManager(repo)
	ctx := context.Background()

	// 1. Enroll IP on standard default plan (Day 1: max 50, google: 20)
	state, err := manager.EnrollIP(ctx, "ip_test_1", "192.0.2.10", "wplan_default")
	if err != nil {
		t.Fatalf("EnrollIP failed: %v", err)
	}
	if state.CurrentDay != 1 || state.Status != "warming" {
		t.Errorf("unexpected initial state: %+v", state)
	}

	// 2. Can send initially
	canSend, reason, err := manager.CanSend(ctx, "ip_test_1", "google")
	if err != nil || !canSend {
		t.Fatalf("expected canSend=true, got %v (%s)", canSend, reason)
	}

	// 3. Record 20 sends to Google -> reaches provider cap
	for i := 0; i < 20; i++ {
		if err := manager.RecordSend(ctx, "ip_test_1", "google"); err != nil {
			t.Fatalf("RecordSend failed: %v", err)
		}
	}

	// Now Google should be capped
	canSendGoogle, reasonGoogle, err := manager.CanSend(ctx, "ip_test_1", "google")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if canSendGoogle {
		t.Errorf("expected Google to be capped, but canSend=true")
	}
	if reasonGoogle == "" {
		t.Errorf("expected reason for Google cap rejection")
	}

	// Microsoft should still be allowed (under provider cap of 20, and under daily cap of 50)
	canSendMs, _, _ := manager.CanSend(ctx, "ip_test_1", "microsoft")
	if !canSendMs {
		t.Errorf("expected Microsoft to be allowed")
	}

	// 4. Test automated hold when bounce rate exceeds 5%
	err = manager.AdvanceDay(ctx, "ip_test_1", 0.08, 0.000, false)
	if err != nil {
		t.Fatalf("AdvanceDay failed: %v", err)
	}
	stHeld, _ := repo.GetIPWarmupState(ctx, "ip_test_1")
	if stHeld.Status != "held" {
		t.Errorf("expected status 'held', got %s", stHeld.Status)
	}

	// While held, CanSend must be false
	canSendHeld, _, _ := manager.CanSend(ctx, "ip_test_1", "microsoft")
	if canSendHeld {
		t.Errorf("expected CanSend=false while held")
	}

	// 5. Resume Warmup
	if err := manager.ResumeWarmup(ctx, "ip_test_1"); err != nil {
		t.Fatalf("ResumeWarmup failed: %v", err)
	}

	// 6. Force advance to next day
	if err := manager.AdvanceDay(ctx, "ip_test_1", 0.01, 0.000, true); err != nil {
		t.Fatalf("force AdvanceDay failed: %v", err)
	}
	stDay2, _ := repo.GetIPWarmupState(ctx, "ip_test_1")
	if stDay2.CurrentDay != 2 || stDay2.CurrentDayVolume != 0 {
		t.Errorf("expected day 2 with reset volume, got %+v", stDay2)
	}

	// 7. Test completion
	stDay2.CurrentDay = 31
	_ = repo.SaveIPWarmupState(ctx, stDay2)
	_ = manager.AdvanceDay(ctx, "ip_test_1", 0.01, 0.000, true)
	stComp, _ := repo.GetIPWarmupState(ctx, "ip_test_1")
	if stComp.Status != "completed" {
		t.Errorf("expected completed status, got %s", stComp.Status)
	}
}
