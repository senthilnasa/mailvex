package retention_test

import (
	"context"
	"testing"
	"time"

	"github.com/senthilnasa/mailvex/core/domain"
	"github.com/senthilnasa/mailvex/core/repository"
	"github.com/senthilnasa/mailvex/core/retention"
)

func TestRetentionPolicyAndPurge(t *testing.T) {
	repo := repository.NewMemoryRepository()
	svc := retention.NewRetentionService(repo)
	ctx := context.Background()
	tenantID := "tenant_retention_test"

	// 1. Get default policy
	pol, err := svc.GetPolicy(ctx, tenantID)
	if err != nil {
		t.Fatalf("failed to get policy: %v", err)
	}
	if pol.MessageRetentionDays != 30 || pol.EventsRetentionDays != 90 || pol.RawMIMERetentionDays != 7 {
		t.Errorf("unexpected default policy values: %+v", pol)
	}

	// 2. Update policy to custom days (e.g. 5 days messages, 10 days events, 2 days raw MIME)
	pol.MessageRetentionDays = 5
	pol.EventsRetentionDays = 10
	pol.RawMIMERetentionDays = 2
	err = svc.UpdatePolicy(ctx, pol)
	if err != nil {
		t.Fatalf("failed to update policy: %v", err)
	}

	// 3. Seed stale and fresh records
	now := time.Now().UTC()
	staleDate := now.AddDate(0, 0, -15) // 15 days ago (older than 5 and 10 days)
	freshDate := now.AddDate(0, 0, -1)  // 1 day ago

	staleMIME := "blob://stale_mime"
	freshMIME := "blob://fresh_mime"

	staleMsg := &domain.Message{
		ID:         "msg_stale_1",
		TenantID:   tenantID,
		StreamID:   "stream_tx",
		RawMIMERef: &staleMIME,
		CreatedAt:  staleDate,
	}
	freshMsg := &domain.Message{
		ID:         "msg_fresh_1",
		TenantID:   tenantID,
		StreamID:   "stream_tx",
		RawMIMERef: &freshMIME,
		CreatedAt:  freshDate,
	}

	staleRcpt := &domain.MessageRecipient{
		ID:        "rcpt_stale_1",
		MessageID: "msg_stale_1",
		TenantID:  tenantID,
		Address:   "stale@example.com",
		CreatedAt: staleDate,
	}
	freshRcpt := &domain.MessageRecipient{
		ID:        "rcpt_fresh_1",
		MessageID: "msg_fresh_1",
		TenantID:  tenantID,
		Address:   "fresh@example.com",
		CreatedAt: freshDate,
	}

	staleEvt := &domain.MessageEvent{
		ID:        "evt_stale_1",
		TenantID:  tenantID,
		MessageID: "msg_stale_1",
		Type:      "message.sent",
		CreatedAt: staleDate,
	}
	freshEvt := &domain.MessageEvent{
		ID:        "evt_fresh_1",
		TenantID:  tenantID,
		MessageID: "msg_fresh_1",
		Type:      "message.sent",
		CreatedAt: freshDate,
	}

	repo.SaveMessage(ctx, staleMsg, []*domain.MessageRecipient{staleRcpt}, []*domain.MessageEvent{staleEvt}, nil)
	repo.SaveMessage(ctx, freshMsg, []*domain.MessageRecipient{freshRcpt}, []*domain.MessageEvent{freshEvt}, nil)

	// 4. Execute Purge
	summary, err := svc.ExecutePurge(ctx, tenantID)
	if err != nil {
		t.Fatalf("failed to execute purge: %v", err)
	}

	if summary.PurgedMessagesCount != 1 {
		t.Errorf("expected 1 purged message, got %d", summary.PurgedMessagesCount)
	}
	if summary.PurgedEventsCount != 1 {
		t.Errorf("expected 1 purged event, got %d", summary.PurgedEventsCount)
	}

	// 5. Verify fresh message still exists
	keptMsg, recs, _, err := repo.GetMessage(ctx, tenantID, "msg_fresh_1")
	if err != nil {
		t.Fatalf("fresh message was erroneously purged: %v", err)
	}
	if keptMsg == nil || len(recs) != 1 {
		t.Errorf("expected fresh message to be kept intact")
	}

	// Verify stale message is gone
	_, _, _, err = repo.GetMessage(ctx, tenantID, "msg_stale_1")
	if err == nil {
		t.Errorf("expected stale message to be deleted, but was found")
	}
}
