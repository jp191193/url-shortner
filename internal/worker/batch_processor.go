package worker

import (
	"log"
	"sync"
	"time"

	"github.com/jay-ponkia/go-url-shortener/internal/queue"
	"github.com/jay-ponkia/go-url-shortener/internal/repository"
)

type BatchProcessor struct {
	queue           *queue.ClickQueue
	repo            *repository.URLRepo
	batchSize       int
	processingDelay time.Duration
	ticker          *time.Ticker
	done            chan bool
	isRunning       bool
	mu              sync.Mutex
}

func NewBatchProcessor(q *queue.ClickQueue, repo *repository.URLRepo, batchSize int, processingDelay time.Duration) *BatchProcessor {
	return &BatchProcessor{
		queue:           q,
		repo:            repo,
		batchSize:       batchSize,
		processingDelay: processingDelay,
		done:            make(chan bool),
		isRunning:       false,
	}
}

// Start begins the background batch processing worker
func (bp *BatchProcessor) Start() {
	bp.mu.Lock()
	if bp.isRunning {
		bp.mu.Unlock()
		return
	}
	bp.isRunning = true
	bp.mu.Unlock()

	go bp.processWorker()
	log.Printf("Batch processor started (batch_size=%d, delay=%v)", bp.batchSize, bp.processingDelay)
}

// Stop gracefully stops the batch processor
func (bp *BatchProcessor) Stop() {
	bp.mu.Lock()
	defer bp.mu.Unlock()

	if !bp.isRunning {
		return
	}

	bp.isRunning = false
	bp.done <- true
	log.Println("Batch processor stopped")
}

// processWorker is the main loop that processes batches
// processWorker runs a continuous loop that processes batches at regular intervals.
// It uses a ticker to trigger batch processing based on the configured processingDelay.
// The worker will exit when a signal is received on the done channel.
// ticker.C is a channel that receives a time.Time value at each tick interval,
// signaling that a new batch should be processed.
func (bp *BatchProcessor) processWorker() {
	ticker := time.NewTicker(bp.processingDelay)
	defer ticker.Stop()

	for {
		select {
		case <-bp.done:
			return
		case <-ticker.C:
			bp.processBatch()
		}
	}
}

// processBatch retrieves and processes a single batch from the queue
func (bp *BatchProcessor) processBatch() {
	events, err := bp.queue.DequeueBatch()
	if err != nil {
		log.Printf("Error dequeuing batch: %v", err)
		return
	}

	if len(events) == 0 {
		return // No events to process
	}

	log.Printf("Processing batch of %d click events", len(events))

	// Group clicks by short code
	clickMap := make(map[string]int)
	for _, event := range events {
		clickMap[event.ShortCode]++
	}

	// Process each short code's click count
	successCount := 0
	for shortCode, count := range clickMap {
		if err := bp.repo.IncrementClickCount(shortCode, count); err != nil {
			log.Printf("Error incrementing click count for %s: %v", shortCode, err)
		} else {
			successCount += count
		}
	}

	log.Printf("Successfully processed %d clicks out of %d events", successCount, len(events))
}

// ProcessAllPending processes all remaining events in the queue at once (useful for shutdown)
func (bp *BatchProcessor) ProcessAllPending() {
	log.Println("Processing all pending click events...")
	processed := 0

	for {
		events, err := bp.queue.DequeueBatch()
		if err != nil {
			log.Printf("Error dequeuing pending batch: %v", err)
			break
		}

		if len(events) == 0 {
			break
		}

		clickMap := make(map[string]int)
		for _, event := range events {
			clickMap[event.ShortCode]++
		}

		for shortCode, count := range clickMap {
			if err := bp.repo.IncrementClickCount(shortCode, count); err != nil {
				log.Printf("Error incrementing click count for %s: %v", shortCode, err)
			} else {
				processed += count
			}
		}
	}

	log.Printf("Finished processing all pending events. Total: %d", processed)
}

// GetStats returns current processor statistics
func (bp *BatchProcessor) GetStats() map[string]interface{} {
	queueStats := bp.queue.GetQueueStats()
	return map[string]interface{}{
		"is_running":       bp.isRunning,
		"batch_size":       bp.batchSize,
		"processing_delay": bp.processingDelay.String(),
		"queue_length":     queueStats["queue_length"],
		"max_batches":      queueStats["max_batches"],
	}
}
