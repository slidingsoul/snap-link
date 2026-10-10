package middleware

import (
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

type stubValidator struct {
	claims jwt.MapClaims
	err    error
}

func (s stubValidator) Validate(string) (jwt.MapClaims, error) {
	return s.claims, s.err
}

func setupAuthApp(v TokenValidator) *fiber.App {
	app := fiber.New()
	app.Get("/snap", RequireAuth(v), func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"user": c.Locals("userID")})
	})
	return app
}

func TestRequireAuth(t *testing.T) {
	good := setupAuthApp(stubValidator{claims: jwt.MapClaims{"sub": "user-1"}})

	// Missing header -> 401.
	req := httptest.NewRequest("GET", "/snap", nil)
	resp, _ := good.Test(req, -1)
	if resp.StatusCode != 401 {
		t.Errorf("missing header: got %d, want 401", resp.StatusCode)
	}
	resp.Body.Close()

	// Valid token -> 200 + userID.
	req = httptest.NewRequest("GET", "/snap", nil)
	req.Header.Set("Authorization", "Bearer tok")
	resp, _ = good.Test(req, -1)
	if resp.StatusCode != 200 {
		t.Errorf("valid: got %d, want 200", resp.StatusCode)
	}
	resp.Body.Close()

	// Invalid token -> 401 generic (no internals).
	bad := setupAuthApp(stubValidator{err: errors.New("boom: connection refused")})
	req = httptest.NewRequest("GET", "/snap", nil)
	req.Header.Set("Authorization", "Bearer bad")
	resp, _ = bad.Test(req, -1)
	if resp.StatusCode != 401 {
		t.Errorf("invalid: got %d, want 401", resp.StatusCode)
	}
	resp.Body.Close()

	// Missing sub -> 401.
	nosub := setupAuthApp(stubValidator{claims: jwt.MapClaims{}})
	req = httptest.NewRequest("GET", "/snap", nil)
	req.Header.Set("Authorization", "Bearer tok")
	resp, _ = nosub.Test(req, -1)
	if resp.StatusCode != 401 {
		t.Errorf("no sub: got %d, want 401", resp.StatusCode)
	}
	resp.Body.Close()
}
