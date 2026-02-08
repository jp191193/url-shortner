# Click Batching System - Quick Start Guide

## What This Solves

When 100+ users click different short URLs at the same time, you don't want 100+ concurrent database writes. This system:

✅ Queues clicks in Redis (instant, non-blocking)
✅ Processes them in batches every few seconds
✅ Reduces database load by up to 95%
✅ No impact on user experience (redirects return immediately)

## Installation

The batching system is already integrated into your codebase. Just ensure you have:

1. **Redis running** (required for the queue)
2. **click_count column in your urls table** (if it doesn't exist, run):
   ```sql
   ALTER TABLE urls ADD COLUMN click_count INT DEFAULT 0;
   ```

## Configuration

Set these environment variables before running your server:

```bash
# .env file
BATCH_SIZE=50                    # clicks per batch
BATCH_DELAY_SECONDS=5            # seconds between batches
MAX_QUEUE_LENGTH=10000           # warning threshold
ENABLE_BATCHING=true             # enable/disable
```

## How to Run

```bash
go run main.go
```

You'll see logs like:
```
Batch processor started (batch_size=50, delay=5s)
Processing batch of 47 click events
Successfully processed 47 clicks out of 47 events
```

## Monitor the System

Check queue status:
```bash
curl http://localhost:3000/admin/stats
```

Response shows:
```json
{
  "processor": {
    "is_running": true,
    "batch_size": 50,
    "processing_delay": "5s",
    "queue_length": 23,
    "max_batches": 1
  }
}
```

## Tuning for Your Load

### High Traffic (1000+ clicks/sec)
```env
BATCH_SIZE=100
BATCH_DELAY_SECONDS=2
```

### Low Traffic (< 10 clicks/sec)
```env
BATCH_SIZE=10
BATCH_DELAY_SECONDS=10
```

### Real-time Updates (sacrifice throughput)
```env
BATCH_SIZE=5
BATCH_DELAY_SECONDS=1
```

## Code Flow

```
User clicks short URL
          ↓
Handler calls svc.GetURL(code)
          ↓
Service returns URL from cache/DB
          ↓
Service enqueues click in background (non-blocking)
          ↓
User gets 301 redirect immediately
          ↓
Meanwhile: Batch processor wakes up every 5 seconds
          ↓
Processes 50 clicks in a single transaction
          ↓
Updates database once per unique URL
```

## Testing the System

### Generate test load:
```bash
# Terminal 1: Start server
go run main.go

# Terminal 2: Generate 100 clicks (replace with your URL)
for i in {1..100}; do curl -s http://localhost:3000/abc123 > /dev/null & done
wait

# Terminal 3: Check stats
curl http://localhost:3000/admin/stats | jq .
```

You'll see the queue_length increase, then decrease as the processor runs.

## Key Files

- [internal/queue/click_queue.go](../internal/queue/click_queue.go) - Redis queue
- [internal/worker/batch_processor.go](../internal/worker/batch_processor.go) - Batch processor
- [main.go](../main.go) - Initialization
- [internal/service/url_service.go](../internal/service/url_service.go) - Service integration
- [internal/repository/url_repo.go](../internal/repository/url_repo.go) - Database updates

## Graceful Shutdown

When you stop the server (Ctrl+C), it will:
1. ✅ Process all remaining queued clicks
2. ✅ Close database connection
3. ✅ Close Redis connection

No clicks are lost during shutdown.

## Troubleshooting

### Queue not processing?
- Check if batch processor is running: `curl http://localhost:3000/admin/stats`
- Check server logs for errors

### Click counts not updating?
- Ensure `click_count` column exists in urls table
- Check `/admin/stats` - if queue_length is 0, processor is working

### Memory usage too high?
- Increase BATCH_SIZE (process more at once)
- Decrease BATCH_DELAY_SECONDS (process more frequently)

## Production Checklist

- [ ] Verify Redis is configured for persistence (RDB or AOF)
- [ ] Set appropriate BATCH_SIZE for your traffic
- [ ] Monitor `/admin/stats` endpoint regularly
- [ ] Set up alerting if queue_length exceeds MAX_QUEUE_LENGTH
- [ ] Test graceful shutdown works correctly
- [ ] Load test with expected concurrent users

## Next Steps

See [BATCHING_SYSTEM.md](../BATCHING_SYSTEM.md) for detailed documentation and architecture.
