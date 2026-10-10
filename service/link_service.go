package service

import (
	"errors"
	"fmt"
	"log"
	"net/url"
	"strings"

	"github.com/slidingsoul/snap-link/internal/shortcode"
	"github.com/slidingsoul/snap-link/models"
	"github.com/slidingsoul/snap-link/repository"
)

// Sentinel errors so controllers can map to correct HTTP status codes
// without leaking internal details to clients.
var (
	ErrInvalidURL       = errors.New("invalid URL format")
	ErrInvalidShortCode = errors.New("invalid short code")
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

// Ping verifies the database connection is alive (for /health).
func (s *LinkService) Ping() error {
	return s.repo.Ping()
}

// GetLinkByShortCode retrieves the original URL for a short code
func (s *LinkService) GetLinkByShortCode(shortCode string) (*models.Link, error) {
	if err := ValidateShortCode(shortCode); err != nil {
		return nil, err
	}

	link, err := s.repo.GetByShortCode(shortCode)
	if err != nil {
		log.Printf("Error retrieving link by short code: %v", err)
		return nil, err
	}

	return link, nil
}

// ValidateShortCode delegates to the shared shortcode package so
// generation and validation can never drift apart.
// It wraps shortcode.ErrInvalid with ErrInvalidShortCode to keep
// existing errors.Is(err, service.ErrInvalidShortCode) call sites working.
func ValidateShortCode(code string) error {
	if err := shortcode.Validate(code); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidShortCode, err)
	}
	return nil
}

// validateURL validates that the provided URL is properly formatted
func validateURL(urlStr string) (string, error) {
	urlStr = strings.TrimSpace(urlStr)
	if urlStr == "" {
		return "", fmt.Errorf("%w: URL cannot be empty", ErrInvalidURL)
	}
	if len(urlStr) > 2048 {
		return "", fmt.Errorf("%w: URL too long", ErrInvalidURL)
	}

	lowered := strings.ToLower(urlStr)
	hasHTTP := strings.HasPrefix(lowered, "http://")
	hasHTTPS := strings.HasPrefix(lowered, "https://")
	if !hasHTTP && !hasHTTPS {
		// Reject non-http(s) schemes explicitly instead of blindly prefixing.
		// e.g. "javascript:alert(1)" or "data:..." should not become https://...
		if strings.Contains(urlStr, "://") {
			return "", fmt.Errorf("%w: only http and https schemes are allowed", ErrInvalidURL)
		}
		urlStr = "https://" + urlStr
	}

	parsed, err := url.Parse(urlStr)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrInvalidURL, err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", fmt.Errorf("%w: only http and https schemes are allowed", ErrInvalidURL)
	}
	if parsed.Host == "" {
		return "", fmt.Errorf("%w: missing host", ErrInvalidURL)
	}
	if strings.Contains(urlStr, " ") {
		return "", fmt.Errorf("%w: URL must not contain spaces", ErrInvalidURL)
	}

	return urlStr, nil
}
