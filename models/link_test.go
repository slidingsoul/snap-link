package models

import (
	"testing"
	"time"
)

func TestLinkTableName(t *testing.T) {
	link := Link{}
	if link.TableName() != "links" {
		t.Errorf("Expected table name 'links', got '%s'", link.TableName())
	}
}

func TestCreateLinkRequest(t *testing.T) {
	tests := []struct {
		name    string
		link    string
		isValid bool
	}{
		{"Valid HTTP URL", "http://example.com", true},
		{"Valid HTTPS URL", "https://example.com/path", true},
		{"Empty URL", "", true}, // struct allows this, validation happens in service
		{"URL with query params", "https://example.com?foo=bar", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := CreateLinkRequest{Link: tt.link}
			if req.Link != tt.link {
				t.Errorf("Expected link '%s', got '%s'", tt.link, req.Link)
			}
		})
	}
}

func TestCreateLinkResponse(t *testing.T) {
	shortCode := "abc123"
	resp := CreateLinkResponse{Link: shortCode}

	if resp.Link != shortCode {
		t.Errorf("Expected link '%s', got '%s'", shortCode, resp.Link)
	}
}

func TestLinkStruct(t *testing.T) {
	now := time.Now()
	link := Link{
		ID:          1,
		OriginalURL: "https://example.com/very/long/url",
		ShortCode:   "abc123",
		CreatedAt:   now,
	}

	if link.ID != 1 {
		t.Errorf("Expected ID 1, got %d", link.ID)
	}

	if link.OriginalURL != "https://example.com/very/long/url" {
		t.Errorf("Expected original URL 'https://example.com/very/long/url', got '%s'", link.OriginalURL)
	}

	if link.ShortCode != "abc123" {
		t.Errorf("Expected short code 'abc123', got '%s'", link.ShortCode)
	}

	if link.CreatedAt != now {
		t.Errorf("Expected created_at %v, got %v", now, link.CreatedAt)
	}
}
