package middleware

import (
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/slidingsoul/snap-link/internal/ratelimit"
)

func setupLimiter(t *testing.T, max int) *fiber.App {
	t.Helper()
	lim := NewLimiter(max)
	app := fiber.New(fiber.Config{ProxyHeader: "X-Forwarded-For"})
	app.Get("/snap", lim.Handler(), func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})
	return app
}

func hit(t *testing.T, app *fiber.App, ip string) int {
	t.Helper()
	req := httptest.NewRequest("GET", "/snap", nil)
	req.RemoteAddr = ip + ":1234"
	// Fiber's c.IP() honors X-Forwarded-For when configured; set directly:
	req.Header.Set("X-Forwarded-For", ip)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("test req: %v", err)
	}
	defer resp.Body.Close()
	return resp.StatusCode
}

func TestLimiterBlocksOverLimit(t *testing.T) {
	app := setupLimiter(t, 3)
	ip := "10.0.0.1"
	for i := 0; i < 3; i++ {
		if code := hit(t, app, ip); code != 200 {
			t.Fatalf("request %d: got %d, want 200", i+1, code)
		}
	}
	if code := hit(t, app, ip); code != 429 {
		t.Errorf("4th request: got %d, want 429", code)
	}
	// Different IP unaffected.
	if code := hit(t, app, "10.0.0.2"); code != 200 {
		t.Errorf("other IP: got %d, want 200", code)
	}
}

func TestLimiterDefaults(t *testing.T) {
	if ratelimit.DefaultPerMinute != 5 {
		t.Errorf("DefaultPerMinute = %d, want 5", ratelimit.DefaultPerMinute)
	}
	// max < 1 falls back to default.
	app := setupLimiter(t, 0)
	for i := 0; i < ratelimit.DefaultPerMinute; i++ {
		if code := hit(t, app, "10.0.5.1"); code != 200 {
			t.Fatalf("request %d: got %d, want 200", i+1, code)
		}
	}
	if code := hit(t, app, "10.0.5.1"); code != 429 {
		t.Errorf("over default: got %d, want 429", code)
	}
}
