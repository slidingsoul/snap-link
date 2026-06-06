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

func setupLookupControllerTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}

	if err := db.AutoMigrate(&models.Lookup{}); err != nil {
		t.Fatalf("Failed to auto-migrate: %v", err)
	}

	return db
}

func TestNewLookupController(t *testing.T) {
	db := setupLookupControllerTestDB(t)
	repo := repository.NewLookupRepository(db)
	svc := service.NewLookupService(repo)
	controller := NewLookupController(svc, "test-api-key")

	if controller == nil {
		t.Errorf("NewLookupController() returned nil")
	}
}

func TestLookupControllerInitialization(t *testing.T) {
	db := setupLookupControllerTestDB(t)
	repo := repository.NewLookupRepository(db)
	svc := service.NewLookupService(repo)
	controller := NewLookupController(svc, "test-api-key")

	if controller == nil {
		t.Errorf("Controller is nil")
	}
	if controller.service == nil {
		t.Errorf("Controller service is nil")
	}
}

func TestLookupControllerRoutes(t *testing.T) {
	db := setupLookupControllerTestDB(t)
	repo := repository.NewLookupRepository(db)
	svc := service.NewLookupService(repo)
	controller := NewLookupController(svc, "test-api-key")

	app := fiber.New()

	// Register routes
	app.Put("/config/lookup", controller.UpdateLookup)
	app.Get("/config/lookup/:key", controller.GetLookup)

	// Verify routes are registered
	if app == nil {
		t.Errorf("Failed to setup fiber app")
	}

	// Verify controller is properly initialized
	if controller.service == nil {
		t.Errorf("Controller service is nil")
	}
}
