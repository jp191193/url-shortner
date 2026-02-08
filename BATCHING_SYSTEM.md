# Click Batching System Documentation

## Overview

This system prevents database overload by batching URL click events instead of updating the database on every single click. When many users click short URLs simultaneously, clicks are queued in Redis and processed in controlled batches.

## How It Works

### Without Batching (Previous Approach)
- User clicks short URL → Handler retrieves URL → Database updates click_count
- 100 concurrent users = 100 direct database writes = potential deadlocks/bottlenecks

### With Batching (New Approach)
- User clicks short URL → Handler retrieves URL → Click event queued in Redis (O(1) operation)
- Background worker processes queued clicks in batches every 5 seconds
- Reduces database load significantly

## Architecture

```
┌─────────────────────────────────────────┐
│         User Requests (100 clicks)      │
└──────────────────┬──────────────────────┘
                   │
                   ▼
        ┌──────────────────────┐
        │  Fiber HTTP Handler  │
        │  - Get URL (Redis)   │
        │  - Return 301        │
        │  - Enqueue Click     │
        └──────────┬───────────┘
                   │
                   ▼
        ┌──────────────────────┐
        │    Redis Queue       │
        │  (FIFO List)         │
        │  - O(1) Operations   │
        │  - No DB Writes      │
        └──────────┬───────────┘
                   │
      (Every 5 seconds or on shutdown)
                   │
                   ▼
        ┌──────────────────────┐
        │  Batch Processor     │
        │  - Dequeue 50 items  │
        │  - Group by URL      │
        │  - Bulk Update DB    │
        └──────────┬───────────┘
                   │
                   ▼
        ┌──────────────────────┐
        │    Database          │
        │  - Single UPDATE     │
        │  - Per URL           │
        └──────────────────────┘
```

## Configuration

Set environment variables to customize the batching behavior:

```env
# Number of clicks to process per batch (default: 50)
BATCH_SIZE=50

# Delay between batch processing runs in seconds (default: 5)
BATCH_DELAY_SECONDS=5

# Maximum number of queued items before warnings (default: 10000)
MAX_QUEUE_LENGTH=10000

# Enable/disable batching (default: true)
ENABLE_BATCHING=true
```

### Example Configurations

**High-Traffic Setup (1000+ concurrent users):**
```env
BATCH_SIZE=100
BATCH_DELAY_SECONDS=2
MAX_QUEUE_LENGTH=50000
```

**Low-Traffic Setup (< 100 concurrent users):**
```env
BATCH_SIZE=10
BATCH_DELAY_SECONDS=10
MAX_QUEUE_LENGTH=1000
```

**Real-Time Updates (sacrifice throughput for latency):**
```env
BATCH_SIZE=5
BATCH_DELAY_SECONDS=1
MAX_QUEUE_LENGTH=5000
```

## API Endpoints

### Retrieve Queue Statistics
```bash
GET /admin/stats
```

Response:
```json
{
  "processor": {
    "is_running": true,
    "batch_size": 50,
    "processing_delay": "5s",
    "queue_length": 127,
    "max_batches": 3
  }
}
```

## Key Features

### 1. **Non-Blocking Clicks**
- Click tracking doesn't delay the redirect response
- User experience not affected by database load

### 2. **Automatic Batching**
- Clicks grouped by URL code
- Only one database write per unique URL per batch
- Reduces write operations by up to 95%

### 3. **Graceful Shutdown**
- Server captures SIGINT/SIGTERM signals
- Processes all remaining queued clicks before stopping
- No data loss on shutdown

### 4. **Redis-Backed Queue**
- Uses Redis LPUSH/LRANGE/LTRIM for O(1) operations
- Persists across application restarts (if Redis is configured)
- Decouples request handling from database operations

### 5. **Monitoring**
- `/admin/stats` endpoint shows queue depth and processing stats
- Logs indicate batch processing status
- Easy to identify performance bottlenecks

## Performance Example

**Scenario:** 1000 users hit different short URLs over 5 seconds

### Without Batching:
- 1000 individual UPDATE queries to database
- Each query takes ~5ms
- Total database time: 5000ms
- Potential timeouts and deadlocks

### With Batching (batch_size=50, delay=5s):
- Single batch processes in ~1 second
- 20 batches total = 20 seconds total processing
- Database load distributed over time
- No request timeouts

## Code Integration

### In Your Handler
```go
func getURL(svc *service.URLService, cfg *config.Config) fiber.Handler {
    return func(c *fiber.Ctx) error {
        code := c.Params("code")
        originalURL, err := svc.GetURL(code) // Enqueues click automatically
        if err != nil {
            return c.Status(404).JSON(fiber.Map{"error": "Not found"})
        }
        return c.Status(301).Redirect(originalURL)
    }
}
```

The click tracking is **completely transparent** - the service handles it automatically.

## Monitoring & Debugging

### Check queue length:
```bash
curl http://localhost:3000/admin/stats
```

### Clear the queue (if needed):
```go
// In a temporary admin endpoint
clickQueue.Clear()
```

### Manual processing:
```go
// Process all remaining events immediately
batchProcessor.ProcessAllPending()
```

## Database Schema Requirement

Ensure your `urls` table has a `click_count` column:

```sql
ALTER TABLE urls ADD COLUMN click_count INT DEFAULT 0;
```

If not already present, the migration is in `scripts/alter-table/`.

## Potential Issues & Solutions

### Problem: Queue keeps growing
- **Cause:** Batch processor stopped or very slow database
- **Solution:** Check logs, increase BATCH_SIZE, or check database performance

### Problem: Click counts not updating
- **Cause:** `click_count` column missing or batch processor not running
- **Solution:** Verify column exists, check `/admin/stats` endpoint

### Problem: Memory usage spike
- **Cause:** Queue length exceeded max capacity
- **Solution:** Increase BATCH_SIZE or reduce BATCH_DELAY_SECONDS to process faster

## Best Practices

1. **Start Conservative:** Begin with default settings, adjust based on metrics
2. **Monitor Queue Depth:** Check `/admin/stats` regularly
3. **Set Appropriate Batch Size:** 
   - Too small = more database round-trips
   - Too large = more memory usage
4. **Adjust Delay Based on Traffic:**
   - High traffic: shorter delay (1-2 seconds)
   - Low traffic: longer delay (10+ seconds)
5. **Test Before Production:** Load test with your expected concurrent user count

## Future Enhancements

- [ ] Adaptive batch sizing based on queue depth
- [ ] Circuit breaker if database is down
- [ ] Metrics export (Prometheus)
- [ ] Per-URL click breakdown in batch processor
- [ ] Web UI for monitoring queue
