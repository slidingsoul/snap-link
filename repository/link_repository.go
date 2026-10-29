package repository

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/slidingsoul/snap-link/internal/shortcode"
	"github.com/slidingsoul/snap-link/models"
	"gorm.io/gorm"
)

// ErrNotFound is returned when a short code does not exist.
// Controllers use errors.Is to map it to 404 without leaking internals.
var ErrNotFound = errors.New("link not found")

// LinkRepository handles database operations for links
type LinkRepository struct {
	db *gorm.DB
}

// NewLinkRepository creates a new instance of LinkRepository
func NewLinkRepository(db *gorm.DB) *LinkRepository {
	return &LinkRepository{db: db}
}

// Create saves a new link to the database with a generated short code.
// The caller's context bounds the query (and is cancelled on request timeout).
func (r *LinkRepository) Create(ctx context.Context, originalURL string) (*models.Link, error) {
	shortCode, err := shortcode.Generate()
	if err != nil {
		log.Printf("Error generating short code: %v", err)
		return nil, fmt.Errorf("failed to generate short code: %w", err)
	}

	// Check for collision and retry if necessary
	maxRetries := 5
	for i := 0; i < maxRetries; i++ {
		link := &models.Link{
			OriginalURL: originalURL,
			ShortCode:   shortCode,
		}

		result := r.db.WithContext(ctx).Create(link)
		if result.Error == nil {
			return link, nil
		}

		// Check if error is due to unique constraint violation
		errStr := result.Error.Error()
		if strings.Contains(errStr, "UNIQUE constraint failed") ||
			strings.Contains(errStr, "duplicate key") ||
			strings.Contains(errStr, "23505") {
			// Generate a new short code and retry
			shortCode, err = shortcode.Generate()
			if err != nil {
				log.Printf("Error generating new short code on retry %d: %v", i, err)
				return nil, fmt.Errorf("failed to generate short code after retries: %w", err)
			}
			continue
		}

		// For other errors, return immediately
		log.Printf("Error creating link: %v", result.Error)
		return nil, fmt.Errorf("failed to create link: %w", result.Error)
	}

	return nil, fmt.Errorf("failed to create link after %d retries due to collision", maxRetries)
}

// Ping verifies the database connection is alive (for /health).
func (r *LinkRepository) Ping(ctx context.Context) error {
	sqlDB, err := r.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.PingContext(ctx)
}

// GetByShortCode retrieves a link by its short code
func (r *LinkRepository) GetByShortCode(ctx context.Context, shortCode string) (*models.Link, error) {
	var link models.Link
	result := r.db.WithContext(ctx).Where("short_code = ?", shortCode).First(&link)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		log.Printf("Error retrieving link: %v", result.Error)
		return nil, fmt.Errorf("failed to retrieve link: %w", result.Error)
	}
	return &link, nil
}
