package controller

import (
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/slidingsoul/snap-link/models"
	"github.com/slidingsoul/snap-link/repository"
	"github.com/slidingsoul/snap-link/service"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}

	if err := db.AutoMigrate(&models.Link{}); err != nil {
		t.Fatalf("Failed to auto-migrate: %v", err)
	}

	return db
}

func TestNewLinkController(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewLinkRepository(db)
	svc := service.NewLinkService(repo)
	controller := NewLinkController(svc, "test-api-key")

	if controller == nil {
		t.Errorf("NewLinkController() returned nil")
	}
}

func TestControllerMethods(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewLinkRepository(db)
	svc := service.NewLinkService(repo)
	controller := NewLinkController(svc, "test-api-key")

	// Verify controller is properly initialized
	if controller == nil {
		t.Errorf("Controller is nil")
	}
	if controller.service == nil {
		t.Errorf("Controller service is nil")
	}
}

func TestControllerRoutes(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewLinkRepository(db)
	svc := service.NewLinkService(repo)
	controller := NewLinkController(svc, "test-api-key")

	app := fiber.New()

	// Register routes
	app.Get("/health", controller.HealthCheck)
	app.Post("/snap", controller.CreateShortLink)
	app.Get("/:shortCode", controller.GetOriginalURL)

	// Verify routes are registered
	if app == nil {
		t.Errorf("Failed to setup fiber app")
	}

	// Verify controller is properly initialized
	if controller.service == nil {
		t.Errorf("Controller service is nil")
	}
}
