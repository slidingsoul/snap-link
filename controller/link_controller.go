package controller

import (
	"log"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/slidingsoul/snap-link/models"
	"github.com/slidingsoul/snap-link/service"
)

// LinkController handles HTTP requests for link operations
type LinkController struct {
	service *service.LinkService
	apiKey  string
}

// NewLinkController creates a new instance of LinkController
func NewLinkController(svc *service.LinkService, apiKey string) *LinkController {
	return &LinkController{service: svc, apiKey: apiKey}
}

// CreateShortLink handles POST /snap endpoint
// Requires X-API-KEY header with valid API key
// Request body: { "link": "https://example.com/very/long/url" }
// Response: { "link": "abc123" } (short code) or full URL path
func (lc *LinkController) CreateShortLink(c *fiber.Ctx) error {
	// Validate API key from X-API-KEY header
	apiKey := c.Get("X-API-KEY")
	if apiKey == "" || apiKey != lc.apiKey {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
			"error": "Missing or invalid X-API-KEY header",
		})
	}

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
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Return response with short code
	return c.Status(http.StatusCreated).JSON(models.CreateLinkResponse{
		Link: link.ShortCode,
	})
}

// GetOriginalURL handles GET /:shortCode endpoint (optional)
// Returns the original URL for a given short code
func (lc *LinkController) GetOriginalURL(c *fiber.Ctx) error {
	shortCode := c.Params("shortCode")

	link, err := lc.service.GetLinkByShortCode(shortCode)
	if err != nil {
		log.Printf("Error retrieving link: %v", err)
		return c.Status(http.StatusNotFound).JSON(fiber.Map{
			"error": "Short link not found",
		})
	}

	// Optionally redirect to original URL
	return c.Status(http.StatusOK).JSON(fiber.Map{
		"original_url": link.OriginalURL,
		"short_code":   link.ShortCode,
		"created_at":   link.CreatedAt,
	})
}

// HealthCheck handles GET /health endpoint for deployment monitoring
func (lc *LinkController) HealthCheck(c *fiber.Ctx) error {
	return c.Status(http.StatusOK).JSON(fiber.Map{
		"status": "healthy",
	})
}
