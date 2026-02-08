# Click Batching System - Architecture Diagram

## System Flow Diagram

```
┌─────────────────────────────────────────────────────────────────────────┐
│                         HTTP REQUESTS (User Clicks)                      │
│                   GET /abc123, GET /def456, GET /ghi789...               │
└────────────────────────┬────────────────────────────────────────────────┘
                         │
                         ▼
        ┌────────────────────────────────────────┐
        │     Fiber HTTP Handler (getURL)        │
        │                                        │
        │  1. Extract short code from URL       │
        │  2. Retrieve original URL (Redis)     │
        │  3. Background: Enqueue click event   │
        │  4. Return 301 redirect immediately   │
        └────────────────────┬───────────────────┘
                             │
                ┌────────────┴────────────┐
                │                         │
                ▼                         ▼
    ┌─────────────────────┐   ┌──────────────────────┐
    │   Redis Cache       │   │   Click Event Queue  │
    │  (URL Retrieval)    │   │   (Redis List)       │
    │                     │   │                      │
    │  GET short_code     │   │  LPUSH event         │
    │  → original_url     │   │  (0.1ms)             │
    └─────────────────────┘   └──────┬───────────────┘
                                      │
                                      │ (FIFO Queue)
                                      │ Grows as clicks arrive
                                      │
                    ┌─────────────────┴─────────────────┐
                    │                                   │
        ┌───────────────────────────┐      ┌────────────────────────┐
        │  Queue Contents           │      │  Every 5 Seconds       │
        │  [Event 1]                │      │  (Configurable)        │
        │  [Event 2]                │      │                        │
        │  [Event 3]                │      │  Batch Processor       │
        │  ...                      │      │  Wakes Up              │
        │  [Event N]                │      │                        │
        └───────────────────────────┘      └────────────┬───────────┘
                                                         │
                                                         ▼
                        ┌────────────────────────────────────────────┐
                        │     Batch Processor (Worker)              │
                        │                                            │
                        │  1. Dequeue up to 50 events              │
                        │  2. Group by short_code                  │
                        │  3. Count clicks per code                │
                        │     {                                    │
                        │       "abc123": 17,                      │
                        │       "def456": 23,                      │
                        │       "ghi789": 10                       │
                        │     }                                    │
                        │  4. Prepare batch updates                │
                        └────────────────────┬──────────────────────┘
                                             │
                                             ▼
                        ┌────────────────────────────────────────────┐
                        │     Repository (URL Repo)                 │
                        │                                            │
                        │  Execute Batch Updates:                  │
                        │  UPDATE urls SET click_count += 17       │
                        │  WHERE short_code = 'abc123'             │
                        │                                            │
                        │  UPDATE urls SET click_count += 23       │
                        │  WHERE short_code = 'def456'             │
                        │                                            │
                        │  UPDATE urls SET click_count += 10       │
                        │  WHERE short_code = 'ghi789'             │
                        └────────────────────┬──────────────────────┘
                                             │
                                             ▼
                        ┌────────────────────────────────────────────┐
                        │        PostgreSQL Database                │
                        │                                            │
                        │  urls table:                              │
                        │  short_code | original_url | click_count │
                        │  ─────────────────────────────────────── │
                        │  abc123     | https://... | 17           │
                        │  def456     | https://... | 23           │
                        │  ghi789     | https://... | 10           │
                        └────────────────────────────────────────────┘
```

## Request Timing Comparison

### Without Batching (❌ Problem)
```
100 concurrent clicks:

Click 1  →  Update DB (5ms)  →  Redirect (5ms total)
Click 2  →  Update DB (5ms)  →  Redirect (5ms total)
Click 3  →  Update DB (5ms)  →  Redirect (5ms total)
...
Click 100 → Update DB (5ms) → Redirect (5ms+ due to queue) ❌

Total DB load: 100 writes in rapid succession
Risk: Deadlocks, timeouts, inconsistency
```

### With Batching (✅ Solution)
```
100 concurrent clicks:

Click 1    →  Queue event (0.1ms)  →  Redirect (0.1ms) ✅
Click 2    →  Queue event (0.1ms)  →  Redirect (0.1ms) ✅
Click 3    →  Queue event (0.1ms)  →  Redirect (0.1ms) ✅
...
Click 100  →  Queue event (0.1ms)  →  Redirect (0.1ms) ✅

Meanwhile (at 5s, 10s, 15s, etc.):
Batch Processor wakes up
→ Dequeue 50 events
→ Group by URL: {url1: 15, url2: 18, url3: 17}
→ Execute 3 SQL UPDATE statements (not 100!)
→ Back to sleep

Total DB load: ~3 writes spread over 10+ seconds
Result: Predictable, low-impact database activity
```

## Data Flow

```
┌────────────────────────────────────────────────────────┐
│            INCOMING CLICK EVENT                        │
│  {shortCode: "abc123", timestamp: 2026-02-08T...}      │
└─────────────────────┬──────────────────────────────────┘
                      │
                      ▼
         ┌──────────────────────────┐
         │  JSON Serialize Event    │
         │  → String                │
         └──────────────────────────┘
                      │
                      ▼
         ┌──────────────────────────┐
         │  Redis LPUSH             │
         │  key: "url:click:queue"  │
         │  value: JSON string      │
         └──────────────────────────┘
                      │
        ┌─────────────┴─────────────┐
        │                           │
        ▼                           ▼
    [Event] ← [Event] ← [Event] ← [Event]
    (oldest)                       (newest)
        │
        │ (5 seconds later)
        │
        ▼
    ┌────────────────────────────────┐
    │  LRANGE 0 to 49 (batch size)   │
    │  → Retrieve 50 events          │
    └────────────────────────────────┘
        │
        ├─→ Deserialize JSON
        ├─→ Extract short_code
        └─→ Count occurrences
            {
              "abc123": 15,
              "def456": 23,
              ...
            }
        │
        ▼
    ┌────────────────────────────────┐
    │  Build SQL: UPDATE urls SET    │
    │  click_count = click_count + ? │
    │  WHERE short_code = ?          │
    └────────────────────────────────┘
        │
        ▼
    ┌────────────────────────────────┐
    │  Execute via Repository        │
    │  → Actual DB writes            │
    └────────────────────────────────┘
        │
        ▼
    ┌────────────────────────────────┐
    │  LTRIM queue (remove processed)│
    │  → Clean up queue              │
    └────────────────────────────────┘
```

## Component Interaction

```
                    ┌──────────────┐
                    │  main.go     │
                    │ Application  │
                    │  Entry Point │
                    └──────┬───────┘
                           │
              ┌────────────┼────────────┐
              │            │            │
              ▼            ▼            ▼
    ┌─────────────┐ ┌────────────┐ ┌──────────────┐
    │   Handler   │ │  Service   │ │   Processor  │
    │             │ │            │ │              │
    │ getURL()    │ │ GetURL()   │ │ Batch        │
    │             │ │  ├─ Verify │ │ Processor    │
    │ 1. Extract  │ │  │ Cache   │ │              │
    │ 2. Call svc │─→├─ Get from │ │ 1. Dequeue  │
    │ 3. Redirect │  │  │  DB    │─┼─ 2. Group   │
    └─────────────┘  │  └─ Queue │  │ 3. Update DB│
                    │  │ Click   │  └──────────────┘
                    │  │ (async) │
                    │  └─────────┘
                    │
                    └──────┬────────┘
                           │
        ┌──────────────────┼──────────────────┐
        │                  │                  │
        ▼                  ▼                  ▼
    ┌────────────┐  ┌────────────┐  ┌──────────────┐
    │   Queue    │  │  Cache     │  │  Repository  │
    │            │  │            │  │              │
    │ Redis List │  │ Redis KV   │  │ Database Ops │
    │            │  │            │  │              │
    │ LPUSH      │  │ GET/SET    │  │ Get()        │
    │ LRANGE     │  │ TTL        │  │ Save()       │
    │ LTRIM      │  │            │  │ Increment()  │
    └────────────┘  └────────────┘  └──────────────┘
         │                │               │
         └────────────────┼───────────────┘
                          │
                          ▼
                  ┌──────────────────┐
                  │   PostgreSQL DB  │
                  │                  │
                  │  urls table      │
                  │  ├─ short_code   │
                  │  ├─ original_url │
                  │  └─ click_count  │
                  └──────────────────┘
```

## Configuration Tuning

```
BATCH_SIZE: 50 (default)        BATCH_DELAY_SECONDS: 5 (default)

                           Scenario 1: High Traffic
                           BATCH_SIZE: 200
                           BATCH_DELAY_SECONDS: 2
                                    │
                    ┌───────────────┼───────────────┐
                    │               │               │
        ┌───────────▼─────────┐     │     ┌─────────▼──────────┐
        │  Process 200 clicks  │     │     │ Process every 2s    │
        │  per batch           │     │     │ (more frequent)     │
        │                      │     │     │                     │
        │  ✓ Higher throughput │     │     │ ✓ Lower latency     │
        │  ✗ Larger memory     │     │     │ ✗ More DB activity  │
        └──────────────────────┘     │     └─────────────────────┘
                                     │
                           Scenario 2: Standard (Default)
                           BATCH_SIZE: 50
                           BATCH_DELAY_SECONDS: 5
                                     │
                                     ▼
                        ✓ Balanced for most use cases
                        ✓ Good throughput
                        ✓ Reasonable latency

                           Scenario 3: Real-Time
                           BATCH_SIZE: 10
                           BATCH_DELAY_SECONDS: 1
                                    │
                    ┌───────────────┼───────────────┐
                    │               │               │
        ┌───────────▼─────────┐     │     ┌─────────▼──────────┐
        │  Process 10 clicks   │     │     │ Process every 1s    │
        │  per batch           │     │     │ (very frequent)     │
        │                      │     │     │                     │
        │ ✓ Low memory usage   │     │     │ ✓ Near real-time    │
        │ ✗ Lower throughput   │     │     │ ✗ High DB activity  │
        └──────────────────────┘     │     └─────────────────────┘
```

## Graceful Shutdown

```
                    ┌──────────────┐
                    │ SIGINT/SIGTERM│
                    │  Signal       │
                    └──────┬────────┘
                           │
                           ▼
                    ┌──────────────────┐
                    │  Shutdown Handler│
                    └──────┬───────────┘
                           │
            ┌──────────────┼──────────────┐
            │              │              │
            ▼              ▼              ▼
    ┌────────────┐ ┌───────────┐ ┌──────────────┐
    │  Stop      │ │ Process   │ │ Close All    │
    │ Processor  │ │ Remaining │ │ Connections  │
    │            │ │ Queue     │ │              │
    │ Stop ticker│ │           │ │ Shutdown DB  │
    │ Set flag   │ │ Dequeue & │ │ Close Redis  │
    │            │ │ Update DB │ │              │
    └────────────┘ └───────────┘ └──────────────┘
            │              │              │
            └──────────────┼──────────────┘
                           │
                           ▼
                    ┌──────────────────┐
                    │  Server Shutdown │
                    │  Complete ✅      │
                    │  No Data Lost     │
                    └──────────────────┘
```

## Monitor Endpoint

```
GET /admin/stats

Response:
┌─────────────────────────────────────┐
│  processor                          │
├─────────────────────────────────────┤
│  is_running: true                  │
│  batch_size: 50                    │
│  processing_delay: "5s"            │
│  queue_length: 127                 │
│  max_batches: 3                    │
└─────────────────────────────────────┘

Interpretation:
• queue_length: 127 events waiting
• max_batches: 127 / 50 = 3 batches needed
• is_running: Processor is active ✅
```
