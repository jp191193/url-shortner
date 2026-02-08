package main

import (
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jay-ponkia/go-url-shortener/internal/cache"
	"github.com/jay-ponkia/go-url-shortener/internal/config"
	"github.com/jay-ponkia/go-url-shortener/internal/db"
	"github.com/jay-ponkia/go-url-shortener/internal/handler"
	"github.com/jay-ponkia/go-url-shortener/internal/listener"
	"github.com/jay-ponkia/go-url-shortener/internal/repository"
	"github.com/jay-ponkia/go-url-shortener/internal/service"
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

	app := fiber.New()
	repo := repository.NewURLRepo()
	svc := service.NewURLService(repo, redisCache, cacheTTL)

	// Register all routes
	handler.RegisterRoutes(app, svc, cfg)

	log.Printf("Starting server on %s", cfg.Port)
	if err := app.Listen(cfg.Port); err != nil {
		log.Fatal("Server error:", err)
	}
}
