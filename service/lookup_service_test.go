package service

import (
	"testing"

	"github.com/slidingsoul/snap-link/models"
	"github.com/slidingsoul/snap-link/repository"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupLookupServiceTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}

	if err := db.AutoMigrate(&models.Lookup{}); err != nil {
		t.Fatalf("Failed to auto-migrate: %v", err)
	}

	return db
}

func TestNewLookupService(t *testing.T) {
	db := setupLookupServiceTestDB(t)
	repo := repository.NewLookupRepository(db)
	svc := NewLookupService(repo)

	if svc == nil {
		t.Errorf("NewLookupService() returned nil")
	}
}

func TestGetLookupValue(t *testing.T) {
	db := setupLookupServiceTestDB(t)
	repo := repository.NewLookupRepository(db)
	svc := NewLookupService(repo)

	key := "TEST_KEY"
	expectedValue := "test_value"

	// Create a lookup
	_, err := svc.UpdateLookupValue(key, expectedValue)
	if err != nil {
		t.Fatalf("UpdateLookupValue() error = %v", err)
	}

	// Retrieve the value
	value, err := svc.GetLookupValue(key)
	if err != nil {
		t.Errorf("GetLookupValue() error = %v, want nil", err)
	}

	if value != expectedValue {
		t.Errorf("GetLookupValue() = %s, want %s", value, expectedValue)
	}
}

func TestGetLookupValueAsInt(t *testing.T) {
	db := setupLookupServiceTestDB(t)
	repo := repository.NewLookupRepository(db)
	svc := NewLookupService(repo)

	key := "RATE_LIMIT"
	expectedValue := 5

	// Create a lookup with integer value
	_, err := svc.UpdateLookupValue(key, "5")
	if err != nil {
		t.Fatalf("UpdateLookupValue() error = %v", err)
	}

	// Retrieve as int
	value, err := svc.GetLookupValueAsInt(key)
	if err != nil {
		t.Errorf("GetLookupValueAsInt() error = %v, want nil", err)
	}

	if value != expectedValue {
		t.Errorf("GetLookupValueAsInt() = %d, want %d", value, expectedValue)
	}
}

func TestGetLookupValueAsIntInvalid(t *testing.T) {
	db := setupLookupServiceTestDB(t)
	repo := repository.NewLookupRepository(db)
	svc := NewLookupService(repo)

	key := "INVALID_INT"

	// Create a lookup with non-integer value
	_, err := svc.UpdateLookupValue(key, "not_a_number")
	if err != nil {
		t.Fatalf("UpdateLookupValue() error = %v", err)
	}

	// Try to retrieve as int - should error
	_, err = svc.GetLookupValueAsInt(key)
	if err == nil {
		t.Errorf("GetLookupValueAsInt() error = nil, want error for invalid integer")
	}
}

func TestUpdateLookupValue(t *testing.T) {
	db := setupLookupServiceTestDB(t)
	repo := repository.NewLookupRepository(db)
	svc := NewLookupService(repo)

	key := "TEST_KEY"
	value := "test_value"

	lookup, err := svc.UpdateLookupValue(key, value)
	if err != nil {
		t.Errorf("UpdateLookupValue() error = %v, want nil", err)
	}

	if lookup == nil {
		t.Errorf("UpdateLookupValue() returned nil lookup")
		return
	}

	if lookup.Key != key {
		t.Errorf("UpdateLookupValue() Key = %s, want %s", lookup.Key, key)
	}

	if lookup.Value != value {
		t.Errorf("UpdateLookupValue() Value = %s, want %s", lookup.Value, value)
	}
}

func TestGetLookupValueNotFound(t *testing.T) {
	db := setupLookupServiceTestDB(t)
	repo := repository.NewLookupRepository(db)
	svc := NewLookupService(repo)

	_, err := svc.GetLookupValue("nonexistent")
	if err == nil {
		t.Errorf("GetLookupValue() error = nil, want error for nonexistent key")
	}
}
