package repository

import (
	"testing"

	"github.com/slidingsoul/snap-link/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// SetupTestDB creates an in-memory SQLite database for testing
func SetupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}

	// Auto-migrate the Link model
	if err := db.AutoMigrate(&models.Link{}); err != nil {
		t.Fatalf("Failed to auto-migrate: %v", err)
	}

	return db
}

func TestNewLinkRepository(t *testing.T) {
	db := SetupTestDB(t)
	repo := NewLinkRepository(db)

	if repo == nil {
		t.Errorf("NewLinkRepository() returned nil")
	}
}

func TestRepositoryCreate(t *testing.T) {
	db := SetupTestDB(t)
	repo := NewLinkRepository(db)

	originalURL := "https://github.com/golang/go"
	link, err := repo.Create(originalURL)

	if err != nil {
		t.Errorf("Create() error = %v, want nil", err)
	}

	if link == nil {
		t.Errorf("Create() returned nil link")
		return
	}

	if link.OriginalURL != originalURL {
		t.Errorf("Create() OriginalURL = %s, want %s", link.OriginalURL, originalURL)
	}

	if link.ShortCode == "" {
		t.Errorf("Create() ShortCode is empty")
	}

	if link.ID == 0 {
		t.Errorf("Create() ID is 0")
	}
}

func TestRepositoryGetByShortCode(t *testing.T) {
	db := SetupTestDB(t)
	repo := NewLinkRepository(db)

	originalURL := "https://example.com"
	created, err := repo.Create(originalURL)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	retrieved, err := repo.GetByShortCode(created.ShortCode)
	if err != nil {
		t.Errorf("GetByShortCode() error = %v, want nil", err)
	}

	if retrieved == nil {
		t.Errorf("GetByShortCode() returned nil")
		return
	}

	if retrieved.ShortCode != created.ShortCode {
		t.Errorf("GetByShortCode() ShortCode = %s, want %s", retrieved.ShortCode, created.ShortCode)
	}

	if retrieved.OriginalURL != originalURL {
		t.Errorf("GetByShortCode() OriginalURL = %s, want %s", retrieved.OriginalURL, originalURL)
	}
}

func TestRepositoryGetByShortCodeNotFound(t *testing.T) {
	db := SetupTestDB(t)
	repo := NewLinkRepository(db)

	_, err := repo.GetByShortCode("nonexistent")
	if err == nil {
		t.Errorf("GetByShortCode() error = nil, want error for nonexistent code")
	}
}

func TestRepositoryMultipleCreates(t *testing.T) {
	db := SetupTestDB(t)
	repo := NewLinkRepository(db)

	urls := []string{
		"https://github.com/golang/go",
		"https://golang.org",
		"https://pkg.go.dev",
	}

	links := make([]*models.Link, len(urls))
	for i, url := range urls {
		link, err := repo.Create(url)
		if err != nil {
			t.Errorf("Create() error for %s = %v", url, err)
			return
		}
		links[i] = link
	}

	// Verify all links were created with unique short codes
	shortCodes := make(map[string]bool)
	for i, link := range links {
		if shortCodes[link.ShortCode] {
			t.Errorf("Duplicate short code generated: %s", link.ShortCode)
		}
		shortCodes[link.ShortCode] = true

		// Verify we can retrieve each link
		retrieved, err := repo.GetByShortCode(link.ShortCode)
		if err != nil {
			t.Errorf("GetByShortCode() error for %s = %v", link.ShortCode, err)
			return
		}

		if retrieved.OriginalURL != urls[i] {
			t.Errorf("Mismatch for link %d: got %s, want %s", i, retrieved.OriginalURL, urls[i])
		}
	}
}
