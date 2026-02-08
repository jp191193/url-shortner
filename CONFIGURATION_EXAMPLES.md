# Example Environment Configurations

## Default Configuration (.env)
```
# Redis connection
REDIS_ADDR=localhost:6379

# Database (adjust for your DB)
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=password
DB_NAME=urlshortener
DB_SSL=disable

# Server
PORT=:3000
BASE_URL=http://localhost:3000
CACHE_TTL=3600

# Batch Processing - Recommended for most use cases
BATCH_SIZE=50
BATCH_DELAY_SECONDS=5
MAX_QUEUE_LENGTH=10000
ENABLE_BATCHING=true
```

## High-Traffic Configuration (Production)
For applications with 1000+ concurrent users or 10,000+ clicks/second:

```
BATCH_SIZE=200
BATCH_DELAY_SECONDS=2
MAX_QUEUE_LENGTH=50000
ENABLE_BATCHING=true
```

**Why:**
- Larger batch (200) = fewer database round-trips
- Shorter delay (2s) = faster processing
- Higher queue limit = buffer for traffic spikes

## Standard Configuration (Medium Traffic)
For typical applications with 100-500 concurrent users:

```
BATCH_SIZE=50
BATCH_DELAY_SECONDS=5
MAX_QUEUE_LENGTH=10000
ENABLE_BATCHING=true
```

**Characteristics:**
- Balanced batch size
- Reasonable processing delay
- Good for most scenarios

## Real-Time Configuration (Lower Throughput)
For applications where click accuracy matters more than throughput:

```
BATCH_SIZE=10
BATCH_DELAY_SECONDS=1
MAX_QUEUE_LENGTH=5000
ENABLE_BATCHING=true
```

**Characteristics:**
- Smaller batches = more frequent updates
- Shorter delay = near real-time metrics
- More database activity
- Use when you need accurate click counts immediately

## Low-Traffic Configuration (Development)
For testing and development environments:

```
BATCH_SIZE=5
BATCH_DELAY_SECONDS=10
MAX_QUEUE_LENGTH=1000
ENABLE_BATCHING=true
```

## Disabled Batching (Fallback)
If you want to disable batching (not recommended):

```
ENABLE_BATCHING=false
```

**This will:**
- Update database on every click (immediate)
- Increase database load
- Potential for deadlocks under high load
- Only use for debugging

## Performance Comparison

| Scenario | Batch Size | Delay | Throughput | Latency | DB Load |
|----------|-----------|-------|-----------|---------|---------|
| Disabled | N/A | N/A | Low | 0ms | Very High |
| Real-Time | 10 | 1s | Medium | ~500ms | High |
| Standard | 50 | 5s | High | ~2.5s | Medium |
| High-Traffic | 200 | 2s | Very High | ~1s | Low |

## AWS/Cloud Configuration

### AWS RDS PostgreSQL + ElastiCache Redis
```
# Adjust endpoints to your RDS/ElastiCache addresses
DB_HOST=mydb.xxxxx.us-east-1.rds.amazonaws.com
REDIS_ADDR=myredis.xxxxx.ng.0001.use1.cache.amazonaws.com:6379

# High-traffic cloud config
BATCH_SIZE=150
BATCH_DELAY_SECONDS=3
MAX_QUEUE_LENGTH=30000
ENABLE_BATCHING=true
```

### Google Cloud SQL + Memorystore Redis
```
DB_HOST=cloudsql.c.myproject.internal
REDIS_ADDR=10.0.0.3:6379

# Cloud config
BATCH_SIZE=100
BATCH_DELAY_SECONDS=4
MAX_QUEUE_LENGTH=20000
ENABLE_BATCHING=true
```

## Environment Variable Validation

The system will use defaults if values are not provided:

| Variable | Default | Min | Max | Type |
|----------|---------|-----|-----|------|
| BATCH_SIZE | 50 | 1 | 10000 | int |
| BATCH_DELAY_SECONDS | 5 | 1 | 3600 | int |
| MAX_QUEUE_LENGTH | 10000 | 1 | unlimited | int |
| ENABLE_BATCHING | true | N/A | N/A | bool |

## Testing Different Configurations

### Test with Load Generator
```bash
# Test configuration in separate terminal
export BATCH_SIZE=100
export BATCH_DELAY_SECONDS=2
go run main.go

# In another terminal - generate load
ab -n 10000 -c 100 http://localhost:3000/mycode

# Monitor stats
watch -n 1 'curl -s http://localhost:3000/admin/stats | jq .'
```

### Monitor Metrics
```bash
# Watch queue depth in real-time
while true; do
  curl -s http://localhost:3000/admin/stats | jq '.processor | {queue_length, batch_size, is_running}'
  sleep 1
done
```

## Recommended Starting Point

1. **Start with Standard config** - 50 batch size, 5-second delay
2. **Monitor `/admin/stats`** for 24 hours
3. **If queue_length stays high:** Increase BATCH_SIZE or decrease BATCH_DELAY_SECONDS
4. **If database is slow:** Increase BATCH_DELAY_SECONDS (less frequent writes)
5. **If you need real-time metrics:** Switch to Real-Time config

## Monitoring & Alerting

### Alert Conditions

```
# Queue growing faster than processing
queue_length > 5000 for 5 minutes

# Processor stopped
is_running == false

# Database updates failing
Check server logs for "Error incrementing click count"
```

## Database Preparation

Before running with batching, ensure your table is ready:

```sql
-- Add click_count column if it doesn't exist
ALTER TABLE urls ADD COLUMN click_count INT DEFAULT 0;

-- Create index for better performance when multiple updates happen
CREATE INDEX idx_urls_short_code ON urls(short_code);

-- Verify column exists
SELECT column_name, data_type 
FROM information_schema.columns 
WHERE table_name = 'urls';
```

## Scaling Beyond Defaults

If you're processing 100,000+ clicks per second:

```
BATCH_SIZE=500
BATCH_DELAY_SECONDS=1
MAX_QUEUE_LENGTH=100000
ENABLE_BATCHING=true
```

**Advanced:** Consider using Redis Streams instead of Lists for even better throughput, or implement multiple batch processors (sharding by URL prefix).
