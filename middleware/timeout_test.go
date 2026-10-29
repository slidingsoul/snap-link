package middleware

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
)

func TestTimeoutSetsDeadline(t *testing.T) {
	app := fiber.New()
	app.Use(Timeout(50 * time.Millisecond))
	app.Get("/", func(c *fiber.Ctx) error {
		if _, ok := c.UserContext().Deadline(); !ok {
			return c.Status(500).SendString("no deadline")
		}
		return c.SendString("ok")
	})

	resp, err := app.Test(httptest.NewRequest("GET", "/", nil), -1)
	if err != nil {
		t.Fatalf("app.Test error: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Errorf("status = %d, want 200 (deadline missing)", resp.StatusCode)
	}
}

func TestTimeoutExpires(t *testing.T) {
	app := fiber.New()
	app.Use(Timeout(10 * time.Millisecond))
	app.Get("/", func(c *fiber.Ctx) error {
		<-c.UserContext().Done()
		return c.SendString("expired")
	})

	resp, err := app.Test(httptest.NewRequest("GET", "/", nil), -1)
	if err != nil {
		t.Fatalf("app.Test error: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
}
