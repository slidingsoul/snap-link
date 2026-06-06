package controller

import (
	"log"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/slidingsoul/snap-link/models"
	"github.com/slidingsoul/snap-link/service"
)

// LookupController handles HTTP requests for lookup configuration operations
type LookupController struct {
	service *service.LookupService
	apiKey  string
}

// NewLookupController creates a new instance of LookupController
func NewLookupController(svc *service.LookupService, apiKey string) *LookupController {
	return &LookupController{service: svc, apiKey: apiKey}
}

// UpdateLookup handles PUT /config/lookup endpoint
// Requires X-API-KEY header with valid API key
// Request body: { "key": "LINK_PER_MINUTE", "value": "10" }
// Response: { "key": "LINK_PER_MINUTE", "value": "10" }
func (lc *LookupController) UpdateLookup(c *fiber.Ctx) error {
	// Validate API key from X-API-KEY header
	apiKey := c.Get("X-API-KEY")
	if apiKey == "" || apiKey != lc.apiKey {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
			"error": "Missing or invalid X-API-KEY header",
		})
	}

	var req models.UpdateLookupRequest

	// Parse request body
	if err := c.BodyParser(&req); err != nil {
		log.Printf("Error parsing request body: %v", err)
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate request
	if req.Key == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Key field is required",
		})
	}

	if req.Value == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Value field is required",
		})
	}

	// Update lookup via service
	lookup, err := lc.service.UpdateLookupValue(req.Key, req.Value)
	if err != nil {
		log.Printf("Error updating lookup: %v", err)
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Return response with updated lookup
	return c.Status(http.StatusOK).JSON(fiber.Map{
		"key":   lookup.Key,
		"value": lookup.Value,
	})
}

// GetLookup handles GET /config/lookup/:key endpoint
// Returns the value for a given lookup key
func (lc *LookupController) GetLookup(c *fiber.Ctx) error {
	key := c.Params("key")

	value, err := lc.service.GetLookupValue(key)
	if err != nil {
		log.Printf("Error retrieving lookup: %v", err)
		return c.Status(http.StatusNotFound).JSON(fiber.Map{
			"error": "Lookup key not found",
		})
	}

	return c.Status(http.StatusOK).JSON(fiber.Map{
		"key":   key,
		"value": value,
	})
}
