package queue

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

type ClickEvent struct {
	ShortCode string    `json:"short_code"`
	Timestamp time.Time `json:"timestamp"`
}

type ClickQueue struct {
	client    *redis.Client
	queueKey  string
	batchSize int
	ctx       context.Context
}

func NewClickQueue(client *redis.Client, batchSize int) *ClickQueue {
	return &ClickQueue{
		client:    client,
		queueKey:  "url:click:queue",
		batchSize: batchSize,
		ctx:       context.Background(),
	}
}

// EnqueueClick adds a click event to the queue (non-blocking, async)
func (q *ClickQueue) EnqueueClick(shortCode string) error {
	event := ClickEvent{
		ShortCode: shortCode,
		Timestamp: time.Now(),
	}

	data, err := json.Marshal(event)
	if err != nil {
		log.Printf("Error marshaling click event: %v", err)
		return err
	}

	// Push to Redis list (LPUSH - O(1) operation)
	err = q.client.LPush(q.ctx, q.queueKey, string(data)).Err()
	if err != nil {
		log.Printf("Error enqueuing click: %v", err)
		return err
	}

	return nil
}

// GetQueueLength returns current queue length
func (q *ClickQueue) GetQueueLength() (int64, error) {
	length, err := q.client.LLen(q.ctx, q.queueKey).Result()
	if err != nil {
		return 0, err
	}
	return length, nil
}

// DequeueBatch retrieves up to batchSize events from the queue
func (q *ClickQueue) DequeueBatch() ([]ClickEvent, error) {
	ctx, cancel := context.WithTimeout(q.ctx, 5*time.Second)
	defer cancel()

	// Get the batch
	items, err := q.client.LRange(ctx, q.queueKey, 0, int64(q.batchSize-1)).Result()
	if err != nil {
		return nil, err
	}

	if len(items) == 0 {
		return []ClickEvent{}, nil
	}

	// Convert to events
	events := make([]ClickEvent, 0, len(items))
	for _, item := range items {
		var event ClickEvent
		if err := json.Unmarshal([]byte(item), &event); err != nil {
			log.Printf("Error unmarshaling click event: %v", err)
			continue
		}
		events = append(events, event)
	}

	// Remove processed items from queue
	if len(events) > 0 {
		err = q.client.LTrim(ctx, q.queueKey, int64(len(events)), -1).Err()
		if err != nil {
			log.Printf("Warning: Error trimming queue: %v", err)
			// Don't return error - events might have been processed anyway
		}
	}

	return events, nil
}

// Clear empties the entire queue (useful for debugging)
func (q *ClickQueue) Clear() error {
	return q.client.Del(q.ctx, q.queueKey).Err()
}

// GetQueueStats returns queue statistics
func (q *ClickQueue) GetQueueStats() map[string]interface{} {
	length, _ := q.GetQueueLength()
	return map[string]interface{}{
		"queue_length": length,
		"batch_size":   q.batchSize,
		"max_batches":  (length + int64(q.batchSize) - 1) / int64(q.batchSize),
	}
}
