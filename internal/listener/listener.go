package listener

import (
	"context"
	"database/sql"
	"log"
	"time"

	"github.com/lib/pq"
	"github.com/redis/go-redis/v9"
)

// ListenForDBChanges sets up a PostgreSQL listener for database change notifications
// and invalidates the Redis cache when changes occur
func ListenForDBChanges(db *sql.DB, redisClient *redis.Client, dbConnString string) {
	listener := pq.NewListener(
		dbConnString,
		10*time.Second,
		time.Minute,
		nil,
	)

	err := listener.Listen("url_update")
	if err != nil {
		log.Fatal("Listen error:", err)
	}

	go func() {
		ctx := context.Background()
		for {
			select {
			case n := <-listener.Notify:
				if n != nil {
					log.Println("DB changed → invalidating all_urls cache")
					if err := redisClient.Del(ctx, "all_urls").Err(); err != nil {
						log.Printf("Warning: Failed to invalidate cache: %v", err)
					}
				}
			case <-time.After(90 * time.Second):
				// Keep connection alive
				go listener.Ping()
			}
		}
	}()
}
