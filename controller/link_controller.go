package controller

import (
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/slidingsoul/snap-link/models"
	"github.com/slidingsoul/snap-link/repository"
	"github.com/slidingsoul/snap-link/service"
)

// LinkController handles HTTP requests for link operations
type LinkController struct {
	service *service.LinkService
}

// NewLinkController creates a new instance of LinkController
func NewLinkController(svc *service.LinkService) *LinkController {
	return &LinkController{service: svc}
}

// CreateShortLink handles POST /snap endpoint (requires Neon JWT via middleware)
// Request body: { "link": "https://example.com/very/long/url" }
// Response: { "link": "abc123" } (short code) or full URL path
func (lc *LinkController) CreateShortLink(c *fiber.Ctx) error {
	var req models.CreateLinkRequest

	// Parse request body
	if err := c.BodyParser(&req); err != nil {
		log.Printf("Error parsing request body: %v", err)
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate request
	if req.Link == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Link field is required",
		})
	}

	// Create short link via service
	link, err := lc.service.CreateShortLink(req.Link)
	if err != nil {
		log.Printf("Error creating short link: %v", err)
		// Validation errors -> 400 with safe message; DB/infra -> 500 generic.
		if errors.Is(err, service.ErrInvalidURL) || isValidationError(err) {
			return c.Status(http.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid URL format",
			})
		}
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create short link",
		})
	}

	// Return response with short code
	return c.Status(http.StatusCreated).JSON(models.CreateLinkResponse{
		Link: link.ShortCode,
	})
}

// GetOriginalURL handles GET /:shortCode endpoint
// Returns the original URL metadata for a given short code.
// NOTE: this API returns JSON by design; use a 302 redirect here
// if you want classic shortener behavior (see README).
func (lc *LinkController) GetOriginalURL(c *fiber.Ctx) error {
	shortCode := c.Params("shortCode")

	link, err := lc.service.GetLinkByShortCode(shortCode)
	if err != nil {
		log.Printf("Error retrieving link: %v", err)
		if errors.Is(err, repository.ErrNotFound) {
			return c.Status(http.StatusNotFound).JSON(fiber.Map{
				"error": "Short link not found",
			})
		}
		if errors.Is(err, service.ErrInvalidShortCode) || isValidationError(err) {
			return c.Status(http.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid short code",
			})
		}
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve short link",
		})
	}

	return c.Status(http.StatusOK).JSON(fiber.Map{
		"original_url": link.OriginalURL,
		"short_code":   link.ShortCode,
		"created_at":   link.CreatedAt,
	})
}

// HealthCheck handles GET /health endpoint for deployment monitoring.
// Pings the DB so orchestrators don't send traffic to a dead instance.
func (lc *LinkController) HealthCheck(c *fiber.Ctx) error {
	if err := lc.service.Ping(); err != nil {
		log.Printf("Health check DB ping failed: %v", err)
		return c.Status(http.StatusServiceUnavailable).JSON(fiber.Map{
			"status": "unhealthy",
		})
	}
	return c.Status(http.StatusOK).JSON(fiber.Map{
		"status": "healthy",
	})
}

// isValidationError reports client-caused errors that map to 400.
func isValidationError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "invalid") ||
		strings.Contains(msg, "required") ||
		strings.Contains(msg, "cannot be empty") ||
		strings.Contains(msg, "too long") ||
		strings.Contains(msg, "must be") ||
		strings.Contains(msg, "missing host") ||
		strings.Contains(msg, "only http")
}
