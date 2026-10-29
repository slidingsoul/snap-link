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
	service       *service.LinkService
	publicBaseURL string
}

// NewLinkController creates a new instance of LinkController.
// publicBaseURL is the externally reachable origin (e.g. https://snap.link);
// leave it empty to derive the origin from the request.
func NewLinkController(svc *service.LinkService, publicBaseURL string) *LinkController {
	return &LinkController{
		service:       svc,
		publicBaseURL: strings.TrimSuffix(publicBaseURL, "/"),
	}
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
	link, err := lc.service.CreateShortLink(c.UserContext(), req.Link)
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

	// Return the full shareable URL plus the bare code.
	return c.Status(http.StatusCreated).JSON(models.CreateLinkResponse{
		Link:      lc.shortLinkURL(c, link.ShortCode),
		ShortCode: link.ShortCode,
	})
}

// shortLinkURL builds the public URL for a code. Falls back to the request's
// origin when PUBLIC_BASE_URL is not configured (correct behind a TLS proxy
// only if X-Forwarded-Proto is set).
func (lc *LinkController) shortLinkURL(c *fiber.Ctx, code string) string {
	base := lc.publicBaseURL
	if base == "" {
		scheme := c.Protocol()
		if c.Get("X-Forwarded-Proto") == "https" {
			scheme = "https"
		}
		base = scheme + "://" + string(c.Request().Header.Host())
	}
	return base + "/" + code
}

// GetOriginalURL handles GET /:shortCode endpoint.
// Issues a 302 Found redirect to the original URL (classic shortener behavior).
func (lc *LinkController) GetOriginalURL(c *fiber.Ctx) error {
	shortCode := c.Params("shortCode")

	link, err := lc.service.GetLinkByShortCode(c.UserContext(), shortCode)
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

	return c.Redirect(link.OriginalURL, fiber.StatusFound)
}

// Liveness handles GET /livez: process is up, no dependency checks.
// Used by container runtimes so a DB blip doesn't restart the process.
func (lc *LinkController) Liveness(c *fiber.Ctx) error {
	return c.Status(http.StatusOK).JSON(fiber.Map{
		"status": "alive",
	})
}

// HealthCheck handles GET /health endpoint for deployment monitoring.
// Pings the DB so orchestrators don't send traffic to a dead instance.
func (lc *LinkController) HealthCheck(c *fiber.Ctx) error {
	if err := lc.service.Ping(c.UserContext()); err != nil {
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
