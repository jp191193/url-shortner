package config

import (
	"os"
	"strconv"
	"time"
)

// BatchConfig holds configuration for click batch processing
type BatchConfig struct {
	BatchSize       int
	ProcessingDelay time.Duration
	MaxQueueLength  int
	EnableBatching  bool
}

// LoadBatchConfig loads batch processing configuration from environment variables
func (c *Config) LoadBatchConfig() BatchConfig {
	batchSize := 50 // default
	if val := os.Getenv("BATCH_SIZE"); val != "" {
		if size, err := strconv.Atoi(val); err == nil && size > 0 {
			batchSize = size
		}
	}

	delaySeconds := 5 // default
	if val := os.Getenv("BATCH_DELAY_SECONDS"); val != "" {
		if delay, err := strconv.Atoi(val); err == nil && delay > 0 {
			delaySeconds = delay
		}
	}

	maxQueue := 10000 // default
	if val := os.Getenv("MAX_QUEUE_LENGTH"); val != "" {
		if max, err := strconv.Atoi(val); err == nil && max > 0 {
			maxQueue = max
		}
	}

	enableBatching := true
	if val := os.Getenv("ENABLE_BATCHING"); val != "" {
		enableBatching = val != "false" && val != "0"
	}

	return BatchConfig{
		BatchSize:       batchSize,
		ProcessingDelay: time.Duration(delaySeconds) * time.Second,
		MaxQueueLength:  maxQueue,
		EnableBatching:  enableBatching,
	}
}
