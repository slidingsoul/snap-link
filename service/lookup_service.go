package service

import (
	"fmt"
	"log"
	"strconv"

	"github.com/slidingsoul/snap-link/models"
	"github.com/slidingsoul/snap-link/repository"
)

// LookupService handles business logic for lookup configuration
type LookupService struct {
	repo *repository.LookupRepository
}

// NewLookupService creates a new instance of LookupService
func NewLookupService(repo *repository.LookupRepository) *LookupService {
	return &LookupService{repo: repo}
}

// GetLookupValue retrieves a lookup value by key
func (s *LookupService) GetLookupValue(key string) (string, error) {
	if key == "" {
		return "", fmt.Errorf("lookup key cannot be empty")
	}

	lookup, err := s.repo.GetByKey(key)
	if err != nil {
		log.Printf("Error retrieving lookup value: %v", err)
		return "", err
	}

	return lookup.Value, nil
}

// GetLookupValueAsInt retrieves a lookup value as an integer
func (s *LookupService) GetLookupValueAsInt(key string) (int, error) {
	value, err := s.GetLookupValue(key)
	if err != nil {
		return 0, err
	}

	intValue, err := strconv.Atoi(value)
	if err != nil {
		log.Printf("Error converting lookup value to int: %v", err)
		return 0, fmt.Errorf("lookup value is not a valid integer: %w", err)
	}

	return intValue, nil
}

// UpdateLookupValue updates or creates a lookup key-value pair
func (s *LookupService) UpdateLookupValue(key, value string) (*models.Lookup, error) {
	if key == "" || value == "" {
		return nil, fmt.Errorf("lookup key and value cannot be empty")
	}

	lookup, err := s.repo.UpdateOrCreate(key, value)
	if err != nil {
		log.Printf("Error updating lookup value: %v", err)
		return nil, err
	}

	return lookup, nil
}
