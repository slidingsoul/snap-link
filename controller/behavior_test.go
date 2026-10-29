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

func setupBehaviorApp(t *testing.T) *fiber.App {
	t.Helper()
	db := setupBehaviorDB(t)
	linkSvc := service.NewLinkService(repository.NewLinkRepository(db))
	linkCtl := NewLinkController(linkSvc, "https://snap.test")
	app := fiber.New()
	app.Get("/livez", linkCtl.Liveness)
	app.Get("/health", linkCtl.HealthCheck)
	app.Post("/snap", linkCtl.CreateShortLink)
	app.Get("/:shortCode", linkCtl.GetOriginalURL)
	return app
}

func doReq(t *testing.T, app *fiber.App, method, path string, body any) (int, map[string]any) {
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

func TestSnapIsPublic(t *testing.T) {
	app := setupBehaviorApp(t)
	// Controller has no auth check (enforced by RequireAuth middleware in
	// main.go); posting directly succeeds.
	code, body := doReq(t, app, "POST", "/snap", map[string]string{"link": "https://example.com/a"})
	if code != 201 || body["link"] == "" {
		t.Errorf("public create: got %d %v, want 201 with link", code, body)
	}
}

func TestSnapValidationAndLeakage(t *testing.T) {
	app := setupBehaviorApp(t)
	// Invalid URL -> 400 with generic message (no DB internals).
	code, body := doReq(t, app, "POST", "/snap", map[string]string{"link": "ftp://example.com"})
	if code != 400 {
		t.Errorf("ftp scheme: got %d, want 400", code)
	}
	if body["error"] != "Invalid URL format" {
		t.Errorf("expected generic error, got %v", body["error"])
	}
	// Missing field -> 400.
	if code, _ := doReq(t, app, "POST", "/snap", map[string]string{"link": ""}); code != 400 {
		t.Errorf("empty link: got %d, want 400", code)
	}
	// Happy path -> 201.
	code, body = doReq(t, app, "POST", "/snap", map[string]string{"link": "https://example.com/a"})
	if code != 201 || body["link"] == "" {
		t.Errorf("happy path: got %d %v, want 201 with link", code, body)
	}
}

func TestGetByShortCodeStatuses(t *testing.T) {
	app := setupBehaviorApp(t)
	_, created := doReq(t, app, "POST", "/snap", map[string]string{"link": "https://example.com/b"})
	code := created["short_code"].(string)
	if want := "https://snap.test/" + code; created["link"] != want {
		t.Errorf("link = %v, want %q", created["link"], want)
	}
	// Existing code -> 302 redirect with Location header (no redirect follow).
	req := httptest.NewRequest("GET", "/"+code, nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("app.Test error: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 302 {
		t.Errorf("existing code: got %d, want 302", resp.StatusCode)
	}
	if loc := resp.Header.Get("Location"); loc != "https://example.com/b" {
		t.Errorf("Location = %q, want %q", loc, "https://example.com/b")
	}
	if sc, _ := doReq(t, app, "GET", "/abcdef", nil); sc != 404 {
		t.Errorf("missing code: got %d, want 404", sc)
	}
	if sc, _ := doReq(t, app, "GET", "/ab", nil); sc != 400 {
		t.Errorf("malformed code: got %d, want 400", sc)
	}
}

func TestHealth(t *testing.T) {
	app := setupBehaviorApp(t)
	if code, body := doReq(t, app, "GET", "/health", nil); code != 200 || body["status"] != "healthy" {
		t.Errorf("health: got %d %v, want 200 healthy", code, body)
	}
	if code, body := doReq(t, app, "GET", "/livez", nil); code != 200 || body["status"] != "alive" {
		t.Errorf("livez: got %d %v, want 200 alive", code, body)
	}
}
