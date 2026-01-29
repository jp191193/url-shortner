package cache

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisCache struct {
	client *redis.Client
}

// NewRedisCache initializes a new Redis cache instance
func NewRedisCache(addr string) (*RedisCache, error) {
	client := redis.NewClient(&redis.Options{
		Addr: addr,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := client.Ping(ctx).Err()
	if err != nil {
		return nil, err
	}

	return &RedisCache{client: client}, nil
}

// Get retrieves a value from cache
func (rc *RedisCache) Get(ctx context.Context, key string) (string, error) {
	return rc.client.Get(ctx, key).Result()
}

// Set stores a value in cache with TTL
func (rc *RedisCache) Set(ctx context.Context, key string, value string, ttl time.Duration) error {
	return rc.client.Set(ctx, key, value, ttl).Err()
}

// Delete removes a key from cache
func (rc *RedisCache) Delete(ctx context.Context, key string) error {
	return rc.client.Del(ctx, key).Err()
}

// Close closes the Redis connection
func (rc *RedisCache) Close() error {
	return rc.client.Close()
}

// Exists checks if a key exists
func (rc *RedisCache) Exists(ctx context.Context, key string) (bool, error) {
	result, err := rc.client.Exists(ctx, key).Result()
	return result > 0, err
}
