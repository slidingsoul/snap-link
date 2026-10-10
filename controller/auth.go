package controller

import (
	"crypto/subtle"
	"strings"

	"github.com/gofiber/fiber/v2"
)

// checkAPIKey compares in constant time to avoid timing attacks.
// Returns false when the server key is empty (misconfigured) or mismatch.
func checkAPIKey(provided, expected string) bool {
	if provided == "" || expected == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(provided), []byte(expected)) == 1
}

// unauthorized is the shared 401 response (no detail leakage).
func unauthorized(c *fiber.Ctx) error {
	return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
		"error": "Missing or invalid X-API-KEY header",
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
