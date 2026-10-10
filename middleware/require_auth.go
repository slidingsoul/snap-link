package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

// TokenValidator is implemented by *auth.Verifier; interface keeps
// middleware testable without a real JWKS endpoint.
type TokenValidator interface {
	Validate(token string) (jwt.MapClaims, error)
}

// RequireAuth enforces Authorization: Bearer <Neon JWT>.
// On success sets c.Locals("userID", sub). Public routes skip this.
func RequireAuth(v TokenValidator) fiber.Handler {
	return func(c *fiber.Ctx) error {
		header := c.Get("Authorization")
		if header == "" || !strings.HasPrefix(strings.ToLower(header), "bearer ") {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Missing or invalid Authorization header",
			})
		}
		token := strings.TrimSpace(header[len("Bearer "):])
		claims, err := v.Validate(token)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid or expired token",
			})
		}
		sub, _ := claims["sub"].(string)
		if sub == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid or expired token",
			})
		}
		c.Locals("userID", sub)
		return c.Next()
	}
}
