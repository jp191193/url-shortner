package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jay-ponkia/go-url-shortener/internal/cache"
	"github.com/jay-ponkia/go-url-shortener/internal/config"
	"github.com/jay-ponkia/go-url-shortener/internal/db"
	"github.com/jay-ponkia/go-url-shortener/internal/handler"
	"github.com/jay-ponkia/go-url-shortener/internal/listener"
	"github.com/jay-ponkia/go-url-shortener/internal/queue"
	"github.com/jay-ponkia/go-url-shortener/internal/repository"
	"github.com/jay-ponkia/go-url-shortener/internal/service"
	"github.com/jay-ponkia/go-url-shortener/internal/worker"
	"github.com/joho/godotenv"
)

func init() {
	godotenv.Load()
}

func main() {
	cfg := config.LoadConfig()

	// Initialize database connection
	if err := db.Init(cfg.GetDSN()); err != nil {
		log.Fatal("Failed to initialize database:", err)
	}
	defer db.Close()

	// Initialize Redis cache
	redisCache, err := cache.NewRedisCache(cfg.RedisAddr)
	if err != nil {
		log.Fatalf("Failed to initialize Redis cache: %v. Make sure Redis is running on %s", err, cfg.RedisAddr)
	}
	defer redisCache.Close()

	// Parse cache TTL
	cacheTTLSeconds, _ := strconv.Atoi(cfg.CacheTTL)
	fmt.Printf("CacheTTLSeconds from Config %d", cacheTTLSeconds)
	cacheTTL := time.Duration(cacheTTLSeconds) * time.Second

	// Start listening for database changes
	listener.ListenForDBChanges(db.GetDB(), redisCache.Client, cfg.GetDSN())

	// Initialize click queue and batch processor
	// Configuration: batch_size=50, processing_delay=5 seconds
	// Adjust these values based on your load requirements
	clickQueue := queue.NewClickQueue(redisCache.Client, 50) // batch size of 50
	repo := repository.NewURLRepo()
	batchProcessor := worker.NewBatchProcessor(clickQueue, repo, 50, 5*time.Second)
	batchProcessor.Start()

	app := fiber.New()
	svc := service.NewURLService(repo, redisCache, cacheTTL)
	svc.SetClickQueue(clickQueue)

	// Register all routes
	handler.RegisterRoutes(app, svc, cfg)

	// Add route for queue stats (monitoring)
	app.Get("/admin/stats", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"processor": batchProcessor.GetStats(),
		})
	})

	// Graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan

		log.Println("Shutting down gracefully...")

		// Process remaining clicks before shutdown
		batchProcessor.ProcessAllPending()
		batchProcessor.Stop()

		if err := app.Shutdown(); err != nil {
			log.Printf("Server shutdown error: %v", err)
		}
	}()

	log.Printf("Starting server on %s", cfg.Port)
	if err := app.Listen(cfg.Port); err != nil {
		log.Fatal("Server error:", err)
	}
}
