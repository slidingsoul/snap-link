package service

import (
	"testing"

	"github.com/slidingsoul/snap-link/models"
	"github.com/slidingsoul/snap-link/repository"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestValidateURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantURL string
		wantErr bool
	}{
		{"Valid HTTPS URL", "https://example.com", "https://example.com", false},
		{"Valid HTTP URL", "http://example.com", "http://example.com", false},
		{"Valid URL with path", "https://example.com/path/to/resource", "https://example.com/path/to/resource", false},
		{"Valid URL with query", "https://example.com?key=value", "https://example.com?key=value", false},
		{"Empty URL", "", "", true},
		{"URL without scheme", "example.com", "https://example.com", false}, // Should add https:// prefix
		{"Invalid URL with spaces", "https://example .com", "", true},
		{"URL localhost", "http://localhost:8080", "http://localhost:8080", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotURL, err := validateURL(tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && gotURL != tt.wantURL {
				t.Errorf("validateURL() gotURL = %s, wantURL %s", gotURL, tt.wantURL)
			}
		})
	}
}

func TestNewLinkService(t *testing.T) {
	// This test verifies that the validateURL helper function works
	t.Run("URL validation", func(t *testing.T) {
		got, err := validateURL("https://example.com")
		if err != nil {
			t.Errorf("validateURL() should not error for valid URL: %v", err)
		}
		if got != "https://example.com" {
			t.Errorf("validateURL() got %s, want https://example.com", got)
		}

		_, err = validateURL("")
		if err == nil {
			t.Errorf("validateURL() should error for empty URL")
		}
	})
}

func TestCreateShortLink(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}

	if err := db.AutoMigrate(&models.Link{}); err != nil {
		t.Fatalf("Failed to auto-migrate database: %v", err)
	}

	repo := repository.NewLinkRepository(db)
	svc := NewLinkService(repo)

	t.Run("Create short link with scheme added", func(t *testing.T) {
		link, err := svc.CreateShortLink("example.com")
		if err != nil {
			t.Fatalf("CreateShortLink() error = %v", err)
		}
		if link == nil {
			t.Fatalf("CreateShortLink() returned nil link")
		}
		if link.OriginalURL != "https://example.com" {
			t.Errorf("Expected OriginalURL to be 'https://example.com', got '%s'", link.OriginalURL)
		}
	})
}
