package middleware

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/slidingsoul/snap-link/internal/ratelimit"
)

func setupLimiter(t *testing.T, max int) *fiber.App {
	t.Helper()
	lim := NewLimiter(max)
	app := fiber.New(fiber.Config{ProxyHeader: "X-Forwarded-For"})
	// Mimic RequireAuth: set userID from a test header when present.
	app.Get("/snap",
		func(c *fiber.Ctx) error {
			if u := c.Get("X-Test-User"); u != "" {
				c.Locals("userID", u)
			}
			return c.Next()
		},
		lim.Handler(),
		func(c *fiber.Ctx) error {
			return c.SendString("ok")
		})
	return app
}

// doHit issues one request as ip (and optionally as an authenticated user).
func doHit(t *testing.T, app *fiber.App, ip, user string) *http.Response {
	t.Helper()
	req := httptest.NewRequest("GET", "/snap", nil)
	req.RemoteAddr = ip + ":1234"
	// Fiber's c.IP() honors X-Forwarded-For when configured; set directly:
	req.Header.Set("X-Forwarded-For", ip)
	if user != "" {
		req.Header.Set("X-Test-User", user)
	}
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("test req: %v", err)
	}
	defer resp.Body.Close()
	return resp
}

func hitStatus(t *testing.T, app *fiber.App, ip, user string) int {
	t.Helper()
	return doHit(t, app, ip, user).StatusCode
}

func TestLimiterBlocksOverLimit(t *testing.T) {
	app := setupLimiter(t, 3)
	ip := "10.0.0.1"
	for i := 0; i < 3; i++ {
		if code := hitStatus(t, app, ip, ""); code != 200 {
			t.Fatalf("request %d: got %d, want 200", i+1, code)
		}
	}
	if code := hitStatus(t, app, ip, ""); code != 429 {
		t.Errorf("4th request: got %d, want 429", code)
	}
	// Different IP unaffected.
	if code := hitStatus(t, app, "10.0.0.2", ""); code != 200 {
		t.Errorf("other IP: got %d, want 200", code)
	}
}

func TestLimiterKeyedByUser(t *testing.T) {
	app := setupLimiter(t, 2)
	ip := "10.0.9.9"
	for i := 0; i < 2; i++ {
		if code := hitStatus(t, app, ip, "user-a"); code != 200 {
			t.Fatalf("user-a request %d: got %d, want 200", i+1, code)
		}
	}
	if code := hitStatus(t, app, ip, "user-a"); code != 429 {
		t.Errorf("user-a over limit: got %d, want 429", code)
	}
	// Same IP, different JWT subject -> independent quota. Rotating
	// X-Forwarded-For must not help; rotating identities does, by design.
	if code := hitStatus(t, app, ip, "user-b"); code != 200 {
		t.Errorf("user-b first request: got %d, want 200", code)
	}
}

func TestLimiterRetryAfterHeader(t *testing.T) {
	app := setupLimiter(t, 1)
	ip := "10.0.7.7"
	if code := hitStatus(t, app, ip, ""); code != 200 {
		t.Fatalf("first request: got %d, want 200", code)
	}
	resp := doHit(t, app, ip, "")
	if resp.StatusCode != 429 {
		t.Fatalf("second request: got %d, want 429", resp.StatusCode)
	}
	ra := resp.Header.Get("Retry-After")
	if ra == "" {
		t.Fatalf("missing Retry-After header")
	}
	secs, err := strconv.Atoi(ra)
	if err != nil || secs < 1 || secs > 60 {
		t.Errorf("Retry-After = %q, want 1-60 seconds", ra)
	}
}

func TestLimiterDefaults(t *testing.T) {
	if ratelimit.DefaultPerMinute != 5 {
		t.Errorf("DefaultPerMinute = %d, want 5", ratelimit.DefaultPerMinute)
	}
	// max < 1 falls back to default.
	app := setupLimiter(t, 0)
	for i := 0; i < ratelimit.DefaultPerMinute; i++ {
		if code := hitStatus(t, app, "10.0.5.1", ""); code != 200 {
			t.Fatalf("request %d: got %d, want 200", i+1, code)
		}
	}
	if code := hitStatus(t, app, "10.0.5.1", ""); code != 429 {
		t.Errorf("over default: got %d, want 429", code)
	}
}
