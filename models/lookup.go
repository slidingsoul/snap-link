package models

import "time"

// Lookup represents a configuration key-value pair stored in the database
type Lookup struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Key       string    `gorm:"type:varchar(100);not null;unique;index" json:"key"`
	Value     string    `gorm:"type:text;not null" json:"value"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

// TableName specifies the table name for the Lookup model
func (Lookup) TableName() string {
	return "lookups"
}

// UpdateLookupRequest represents the request body for updating a lookup value
type UpdateLookupRequest struct {
	Key   string `json:"key" validate:"required"`
	Value string `json:"value" validate:"required"`
}
