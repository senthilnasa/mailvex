package queue

import (
	"context"
	"testing"
	"time"

	"github.com/senthilnasa/mailvex/core/domain"
	"github.com/senthilnasa/mailvex/core/repository"
)

func TestQueueAndOutboxRelay(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewMemoryRepository()
	q := NewMemoryQueue()
	relay := NewOutboxRelay(repo, q)

	// Create pending outbox item
	item := &domain.OutboxItem{
		ID:          "out_123",
		TenantID:    "tenant_1",
		Stream:      "mvx:delivery:p1",
		Payload:     `{"message_id":"msg_123"}`,
		AvailableAt: time.Now().UTC().Add(-1 * time.Minute), // Available now
		CreatedAt:   time.Now().UTC(),
	}
	_ = repo.SaveOutboxItem(ctx, item)

	// Verify pending in repo
	pending, err := repo.ListPendingOutbox(ctx, 10)
	if err != nil || len(pending) != 1 {
		t.Fatalf("expected 1 pending outbox item, got %d err=%v", len(pending), err)
	}

	// Run relay pass
	count, err := relay.RelayOnce(ctx)
	if err != nil {
		t.Fatalf("relay failed: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 item relayed, got %d", count)
	}

	// Verify outbox marked published
	pendingAfter, _ := repo.ListPendingOutbox(ctx, 10)
	if len(pendingAfter) != 0 {
		t.Fatalf("expected 0 pending outbox items after relay, got %d", len(pendingAfter))
	}

	// Verify message in queue
	msgs, err := q.Consume(ctx, "mvx:delivery:p1", "workers", "worker_1", 10)
	if err != nil || len(msgs) != 1 {
		t.Fatalf("expected 1 message in queue stream, got %d err=%v", len(msgs), err)
	}
	if string(msgs[0].Payload) != `{"message_id":"msg_123"}` {
		t.Fatalf("queue payload mismatch: %s", string(msgs[0].Payload))
	}
}
