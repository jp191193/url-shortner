package main

import (
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/jay-ponkia/go-url-shortener/internal/config"
	"github.com/jay-ponkia/go-url-shortener/internal/db"
	"github.com/jay-ponkia/go-url-shortener/internal/handler"
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

	app := fiber.New()
	repo := repository.NewURLRepo()
	svc := service.NewURLService(repo)

	// Register all routes
	handler.RegisterRoutes(app, svc, cfg)

	log.Printf("Starting server on %s", cfg.Port)
	if err := app.Listen(cfg.Port); err != nil {
		log.Fatal("Server error:", err)
	}
}
