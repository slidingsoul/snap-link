package repository

import (
	"crypto/rand"
	"fmt"
	"log"
	"strings"

	"github.com/slidingsoul/snap-link/models"
	"gorm.io/gorm"
)

// LinkRepository handles database operations for links
type LinkRepository struct {
	db *gorm.DB
}

// NewLinkRepository creates a new instance of LinkRepository
func NewLinkRepository(db *gorm.DB) *LinkRepository {
	return &LinkRepository{db: db}
}

// Create saves a new link to the database with a generated short code
func (r *LinkRepository) Create(originalURL string) (*models.Link, error) {
	shortCode, err := generateShortCode()
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

		result := r.db.Create(link)
		if result.Error == nil {
			return link, nil
		}

		// Check if error is due to unique constraint violation
		errStr := result.Error.Error()
		if strings.Contains(errStr, "UNIQUE constraint failed") ||
			strings.Contains(errStr, "duplicate key") ||
			strings.Contains(errStr, "23505") {
			// Generate a new short code and retry
			shortCode, err = generateShortCode()
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

// GetByShortCode retrieves a link by its short code
func (r *LinkRepository) GetByShortCode(shortCode string) (*models.Link, error) {
	var link models.Link
	result := r.db.Where("short_code = ?", shortCode).First(&link)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("link not found")
		}
		log.Printf("Error retrieving link: %v", result.Error)
		return nil, fmt.Errorf("failed to retrieve link: %w", result.Error)
	}
	return &link, nil
}

// generateShortCode generates a collision-resistant random alphanumeric short code
// Uses 6-8 characters from a 62-character alphabet (A-Z, a-z, 0-9)
func generateShortCode() (string, error) {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
	const codeLength = 6

	code := make([]byte, codeLength)
	for i := 0; i < codeLength; i++ {
		randomIndex := make([]byte, 1)
		_, err := rand.Read(randomIndex)
		if err != nil {
			return "", err
		}
		code[i] = charset[int(randomIndex[0])%len(charset)]
	}

	return string(code), nil
}
