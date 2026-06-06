package repository

import (
	"fmt"
	"log"

	"github.com/slidingsoul/snap-link/models"
	"gorm.io/gorm"
)

// LookupRepository handles database operations for lookup configuration
type LookupRepository struct {
	db *gorm.DB
}

// NewLookupRepository creates a new instance of LookupRepository
func NewLookupRepository(db *gorm.DB) *LookupRepository {
	return &LookupRepository{db: db}
}

// GetByKey retrieves a lookup value by its key
func (r *LookupRepository) GetByKey(key string) (*models.Lookup, error) {
	var lookup models.Lookup
	result := r.db.Where("key = ?", key).First(&lookup)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("lookup key not found: %s", key)
		}
		log.Printf("Error retrieving lookup: %v", result.Error)
		return nil, fmt.Errorf("failed to retrieve lookup: %w", result.Error)
	}
	return &lookup, nil
}

// UpdateOrCreate updates an existing lookup or creates a new one if it doesn't exist
func (r *LookupRepository) UpdateOrCreate(key, value string) (*models.Lookup, error) {
	// Try to find existing lookup
	existing := &models.Lookup{}
	result := r.db.Where("key = ?", key).First(existing)

	if result.Error == nil {
		// Lookup exists, update it
		result := r.db.Model(existing).Update("value", value)
		if result.Error != nil {
			log.Printf("Error updating lookup: %v", result.Error)
			return nil, fmt.Errorf("failed to update lookup: %w", result.Error)
		}
		return existing, nil
	} else if result.Error == gorm.ErrRecordNotFound {
		// Lookup doesn't exist, create it
		lookup := &models.Lookup{
			Key:   key,
			Value: value,
		}
		result := r.db.Create(lookup)
		if result.Error != nil {
			log.Printf("Error creating lookup: %v", result.Error)
			return nil, fmt.Errorf("failed to create lookup: %w", result.Error)
		}
		return lookup, nil
	} else {
		// Other error
		log.Printf("Error checking for existing lookup: %v", result.Error)
		return nil, fmt.Errorf("failed to update/create lookup: %w", result.Error)
	}
}
