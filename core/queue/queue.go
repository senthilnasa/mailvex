package queue

import (
	"context"
	"fmt"
	"sync"

	"github.com/mailvex/mailvex/core/crypto"
	"github.com/mailvex/mailvex/core/repository"
)

type QueueMessage struct {
	ID      string
	Stream  string
	Payload []byte
}

type Queue interface {
	Publish(ctx context.Context, stream string, payload []byte) (string, error)
	Consume(ctx context.Context, stream, group, consumer string, count int) ([]QueueMessage, error)
	Ack(ctx context.Context, stream, group string, ids ...string) error
}

// MemoryQueue is a thread-safe in-memory stream queue
type MemoryQueue struct {
	mu      sync.Mutex
	streams map[string][]QueueMessage
}

func NewMemoryQueue() *MemoryQueue {
	return &MemoryQueue{
		streams: make(map[string][]QueueMessage),
	}
}

func (q *MemoryQueue) Publish(ctx context.Context, stream string, payload []byte) (string, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	msgID := crypto.NewULID("qmsg_")
	msg := QueueMessage{
		ID:      msgID,
		Stream:  stream,
		Payload: payload,
	}
	q.streams[stream] = append(q.streams[stream], msg)
	return msgID, nil
}

func (q *MemoryQueue) Consume(ctx context.Context, stream, group, consumer string, count int) ([]QueueMessage, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	items, ok := q.streams[stream]
	if !ok || len(items) == 0 {
		return []QueueMessage{}, nil
	}

	n := count
	if n > len(items) {
		n = len(items)
	}

	result := make([]QueueMessage, n)
	copy(result, items[:n])
	q.streams[stream] = items[n:]
	return result, nil
}

func (q *MemoryQueue) Ack(ctx context.Context, stream, group string, ids ...string) error {
	return nil
}

func (q *MemoryQueue) Depth(stream string) int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.streams[stream])
}

// OutboxRelay polls pending outbox records and publishes them to the queue
type OutboxRelay struct {
	repo  repository.Repository
	queue Queue
}

func NewOutboxRelay(repo repository.Repository, q Queue) *OutboxRelay {
	return &OutboxRelay{repo: repo, queue: q}
}

// RelayOnce processes pending outbox rows in a single pass
func (r *OutboxRelay) RelayOnce(ctx context.Context) (int, error) {
	items, err := r.repo.ListPendingOutbox(ctx, 100)
	if err != nil {
		return 0, err
	}

	publishedCount := 0
	for _, item := range items {
		_, err := r.queue.Publish(ctx, item.Stream, []byte(item.Payload))
		if err != nil {
			return publishedCount, fmt.Errorf("failed to publish outbox item %s: %w", item.ID, err)
		}
		_ = r.repo.MarkOutboxPublished(ctx, item.ID)
		publishedCount++
	}

	return publishedCount, nil
}
