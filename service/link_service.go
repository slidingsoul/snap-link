package service

import (
	"fmt"
	"log"
	"net/url"

	"github.com/slidingsoul/snap-link/models"
	"github.com/slidingsoul/snap-link/repository"
)

// LinkService handles business logic for links
type LinkService struct {
	repo *repository.LinkRepository
}

// NewLinkService creates a new instance of LinkService
func NewLinkService(repo *repository.LinkRepository) *LinkService {
	return &LinkService{repo: repo}
}

// CreateShortLink creates a new short link with validation
func (s *LinkService) CreateShortLink(originalURL string) (*models.Link, error) {
	// Validate the input URL
	cleanedURL, err := validateURL(originalURL)
	if err != nil {
		log.Printf("Invalid URL: %v", err)
		return nil, err
	}

	// Create the link via repository
	link, err := s.repo.Create(cleanedURL)
	if err != nil {
		log.Printf("Error creating short link: %v", err)
		return nil, fmt.Errorf("failed to create short link: %w", err)
	}

	return link, nil
}

// GetLinkByShortCode retrieves the original URL for a short code
func (s *LinkService) GetLinkByShortCode(shortCode string) (*models.Link, error) {
	if shortCode == "" {
		return nil, fmt.Errorf("short code cannot be empty")
	}

	link, err := s.repo.GetByShortCode(shortCode)
	if err != nil {
		log.Printf("Error retrieving link by short code: %v", err)
		return nil, err
	}

	return link, nil
}

// validateURL validates that the provided URL is properly formatted
func validateURL(urlStr string) (string, error) {
	if urlStr == "" {
		return "", fmt.Errorf("URL cannot be empty")
	}

	// Add scheme if missing
	if len(urlStr) < 7 || (urlStr[:7] != "http://" && urlStr[:8] != "https://") {
		urlStr = "https://" + urlStr
	}

	_, err := url.ParseRequestURI(urlStr)
	if err != nil {
		return "", fmt.Errorf("invalid URL format: %w", err)
	}

	return urlStr, nil
}
