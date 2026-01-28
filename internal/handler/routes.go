package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/jay-ponkia/go-url-shortener/internal/config"
	"github.com/jay-ponkia/go-url-shortener/internal/service"
)

func RegisterRoutes(app *fiber.App, svc *service.URLService, cfg *config.Config) {
	// POST /shorten - Create a shortened URL
	app.Post("/shorten", shortenURL(svc, cfg))

	// GET /:code - Redirect to original URL
	app.Get("/:code", getURL(svc, cfg))
}

func shortenURL(svc *service.URLService, cfg *config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req struct {
			URL string `json:"url"`
		}

		if err := c.BodyParser(&req); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
		}

		if req.URL == "" {
			return c.Status(400).JSON(fiber.Map{"error": "URL is required"})
		}

		shortCode, err := svc.CreateShortURL(req.URL)
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

		url, err := svc.GetURL(code)
		if err != nil {
			return c.Status(404).JSON(fiber.Map{"error": "Short URL not found"})
		}

		return c.Status(200).JSON(fiber.Map{
			"short_url":    cfg.BaseURL + "/" + code,
			"original_url": url,
		})
	}
}
