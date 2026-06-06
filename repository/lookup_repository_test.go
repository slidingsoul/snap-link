package repository

import (
	"testing"

	"github.com/slidingsoul/snap-link/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupLookupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}

	if err := db.AutoMigrate(&models.Lookup{}); err != nil {
		t.Fatalf("Failed to auto-migrate: %v", err)
	}

	return db
}

func TestNewLookupRepository(t *testing.T) {
	db := setupLookupTestDB(t)
	repo := NewLookupRepository(db)

	if repo == nil {
		t.Errorf("NewLookupRepository() returned nil")
	}
}

func TestLookupUpdateOrCreate(t *testing.T) {
	db := setupLookupTestDB(t)
	repo := NewLookupRepository(db)

	key := "TEST_KEY"
	value := "test_value"

	lookup, err := repo.UpdateOrCreate(key, value)
	if err != nil {
		t.Errorf("UpdateOrCreate() error = %v, want nil", err)
	}

	if lookup == nil {
		t.Errorf("UpdateOrCreate() returned nil lookup")
		return
	}

	if lookup.Key != key {
		t.Errorf("UpdateOrCreate() Key = %s, want %s", lookup.Key, key)
	}

	if lookup.Value != value {
		t.Errorf("UpdateOrCreate() Value = %s, want %s", lookup.Value, value)
	}
}

func TestLookupGetByKey(t *testing.T) {
	db := setupLookupTestDB(t)
	repo := NewLookupRepository(db)

	key := "TEST_KEY"
	value := "test_value"

	// Create a lookup first
	created, err := repo.UpdateOrCreate(key, value)
	if err != nil {
		t.Fatalf("UpdateOrCreate() error = %v", err)
	}

	// Retrieve it
	retrieved, err := repo.GetByKey(key)
	if err != nil {
		t.Errorf("GetByKey() error = %v, want nil", err)
	}

	if retrieved == nil {
		t.Errorf("GetByKey() returned nil")
		return
	}

	if retrieved.Key != created.Key {
		t.Errorf("GetByKey() Key = %s, want %s", retrieved.Key, created.Key)
	}

	if retrieved.Value != created.Value {
		t.Errorf("GetByKey() Value = %s, want %s", retrieved.Value, created.Value)
	}
}

func TestLookupGetByKeyNotFound(t *testing.T) {
	db := setupLookupTestDB(t)
	repo := NewLookupRepository(db)

	_, err := repo.GetByKey("nonexistent")
	if err == nil {
		t.Errorf("GetByKey() error = nil, want error for nonexistent key")
	}
}

func TestLookupUpdate(t *testing.T) {
	db := setupLookupTestDB(t)
	repo := NewLookupRepository(db)

	key := "TEST_KEY"
	value1 := "value1"
	value2 := "value2_updated"

	// Create initial lookup
	_, err := repo.UpdateOrCreate(key, value1)
	if err != nil {
		t.Fatalf("UpdateOrCreate() error = %v", err)
	}

	// Update the value
	updated, err := repo.UpdateOrCreate(key, value2)
	if err != nil {
		t.Errorf("UpdateOrCreate() error on update = %v, want nil", err)
	}

	if updated.Value != value2 {
		t.Errorf("UpdateOrCreate() Value after update = %s, want %s", updated.Value, value2)
	}

	// Verify the update persisted
	retrieved, err := repo.GetByKey(key)
	if err != nil {
		t.Errorf("GetByKey() error = %v", err)
	}

	if retrieved.Value != value2 {
		t.Errorf("GetByKey() Value after update = %s, want %s", retrieved.Value, value2)
	}
}
