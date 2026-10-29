package middleware

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
)

// Timeout attaches a deadline to the request context. Handlers and the DB
// layer must honor it (GORM via db.WithContext). Fiber's WriteTimeout does
// not cancel an in-flight handler on its own, so a slow query would otherwise
// keep holding a connection after the client has given up.
func Timeout(d time.Duration) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx, cancel := context.WithTimeout(c.UserContext(), d)
		defer cancel()
		c.SetUserContext(ctx)
		return c.Next()
	}
}
