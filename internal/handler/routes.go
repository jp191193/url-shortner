package handler

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jay-ponkia/go-url-shortener/internal/config"
	"github.com/jay-ponkia/go-url-shortener/internal/service"
	"github.com/jay-ponkia/go-url-shortener/internal/validation"
)

func RegisterRoutes(app *fiber.App, svc *service.URLService, cfg *config.Config) {
	// POST /shorten - Create a shortened URL
	app.Post("/shorten", shortenURL(svc, cfg))

	// GET /urls - Get all URLs (cache-first, returns timing info)
	app.Get("/urls", getAllURLs(svc, cfg))

	// GET /:code - Redirect to original URL
	app.Get("/:code", getURL(svc, cfg))
}

func shortenURL(svc *service.URLService, cfg *config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req struct {
			URL   string `json:"url"`
			Alias string `json:"alias,omitempty"`
		}

		if err := c.BodyParser(&req); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
		}

		if req.URL == "" {
			return c.Status(400).JSON(fiber.Map{"error": "URL is required"})
		}

		isValidAlias, err := validation.ValidateAlias(req.Alias)
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"error": err.Error()})
		}

		// Validate URL
		if !isValidAlias {
			return c.Status(400).JSON(fiber.Map{"error": "Invalid alias"})
		}

		shortCode, err := svc.CreateShortURL(req.URL, req.Alias)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Failed to create short URL"})
		}

		return c.Status(201).JSON(fiber.Map{
			"short_code":   shortCode,
			"short_url":    cfg.BaseURL + "/" + shortCode,
			"original_url": req.URL,
		})
	}
}

func getURL(svc *service.URLService, cfg *config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		code := c.Params("code")

		if code == "" {
			return c.Status(400).JSON(fiber.Map{"error": "code is required"})
		}

		originalURL, err := svc.GetURL(code)
		if err != nil {
			return c.Status(404).JSON(fiber.Map{"error": "Short URL not found"})
		}

		// Redirect to the original URL (cache handled in service)
		return c.Status(201).JSON(fiber.Map{
			"original_url": originalURL,
			"short_url":    cfg.BaseURL + "/" + code,
		})
	}
}

func getAllURLs(svc *service.URLService, cfg *config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()

		records, source, err := svc.GetAllURLs()
		elapsed := time.Since(start).Milliseconds()
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "failed to fetch urls", "details": err.Error()})
		}

		return c.Status(200).JSON(fiber.Map{
			"source":     source,
			"elapsed_ms": elapsed,
			"data":       records,
		})
	}
}
