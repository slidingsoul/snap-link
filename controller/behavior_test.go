package controller

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/slidingsoul/snap-link/models"
	"github.com/slidingsoul/snap-link/repository"
	"github.com/slidingsoul/snap-link/service"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupBehaviorDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}
	if err := db.AutoMigrate(&models.Link{}); err != nil {
		t.Fatalf("Failed to auto-migrate: %v", err)
	}
	return db
}

func setupBehaviorApp(t *testing.T, apiKey string) *fiber.App {
	t.Helper()
	db := setupBehaviorDB(t)
	linkSvc := service.NewLinkService(repository.NewLinkRepository(db))
	linkCtl := NewLinkController(linkSvc, apiKey)
	app := fiber.New()
	app.Get("/health", linkCtl.HealthCheck)
	app.Post("/snap", linkCtl.CreateShortLink)
	app.Get("/:shortCode", linkCtl.GetOriginalURL)
	return app
}

func doReq(t *testing.T, app *fiber.App, method, path, apiKey string, body any) (int, map[string]any) {
	t.Helper()
	var r *bytes.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		r = bytes.NewReader(b)
	} else {
		r = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, r)
	req.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		req.Header.Set("X-API-KEY", apiKey)
	}
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("app.Test error: %v", err)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	out := map[string]any{}
	_ = json.Unmarshal(data, &out)
	return resp.StatusCode, out
}

func TestSnapRequiresAuth(t *testing.T) {
	app := setupBehaviorApp(t, "secret")
	if code, _ := doReq(t, app, "POST", "/snap", "", map[string]string{"link": "https://example.com"}); code != 401 {
		t.Errorf("missing key: got %d, want 401", code)
	}
	if code, _ := doReq(t, app, "POST", "/snap", "wrong", map[string]string{"link": "https://example.com"}); code != 401 {
		t.Errorf("wrong key: got %d, want 401", code)
	}
}

func TestSnapValidationAndLeakage(t *testing.T) {
	app := setupBehaviorApp(t, "secret")
	// Invalid URL -> 400 with generic message (no DB internals).
	code, body := doReq(t, app, "POST", "/snap", "secret", map[string]string{"link": "ftp://example.com"})
	if code != 400 {
		t.Errorf("ftp scheme: got %d, want 400", code)
	}
	if body["error"] != "Invalid URL format" {
		t.Errorf("expected generic error, got %v", body["error"])
	}
	// Missing field -> 400.
	if code, _ := doReq(t, app, "POST", "/snap", "secret", map[string]string{"link": ""}); code != 400 {
		t.Errorf("empty link: got %d, want 400", code)
	}
	// Happy path -> 201.
	code, body = doReq(t, app, "POST", "/snap", "secret", map[string]string{"link": "https://example.com/a"})
	if code != 201 || body["link"] == "" {
		t.Errorf("happy path: got %d %v, want 201 with link", code, body)
	}
}

func TestGetByShortCodeStatuses(t *testing.T) {
	app := setupBehaviorApp(t, "secret")
	_, created := doReq(t, app, "POST", "/snap", "secret", map[string]string{"link": "https://example.com/b"})
	code := created["link"].(string)
	if sc, _ := doReq(t, app, "GET", "/"+code, "", nil); sc != 200 {
		t.Errorf("existing code: got %d, want 200", sc)
	}
	if sc, _ := doReq(t, app, "GET", "/abcdef", "", nil); sc != 404 {
		t.Errorf("missing code: got %d, want 404", sc)
	}
	if sc, _ := doReq(t, app, "GET", "/ab", "", nil); sc != 400 {
		t.Errorf("malformed code: got %d, want 400", sc)
	}
}

func TestHealth(t *testing.T) {
	app := setupBehaviorApp(t, "secret")
	if code, body := doReq(t, app, "GET", "/health", "", nil); code != 200 || body["status"] != "healthy" {
		t.Errorf("health: got %d %v, want 200 healthy", code, body)
	}
}
