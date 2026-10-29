package models

import (
	"time"
)

// Link represents a shortened URL record in the database
type Link struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	OriginalURL string    `gorm:"type:text;not null;index" json:"original_url"`
	ShortCode   string    `gorm:"type:varchar(10);not null;unique;index" json:"short_code"`
	CreatedAt   time.Time `gorm:"autoCreateTime" json:"created_at"`
}

// TableName specifies the table name for the Link model
func (Link) TableName() string {
	return "links"
}

// CreateLinkRequest represents the incoming request body for creating a short link
type CreateLinkRequest struct {
	Link string `json:"link" validate:"required,url"`
}

// CreateLinkResponse represents the response body for creating a short link.
// Link is the full shareable URL (PUBLIC_BASE_URL + code); ShortCode is the
// bare code for clients that build their own links.
type CreateLinkResponse struct {
	Link      string `json:"link"`
	ShortCode string `json:"short_code"`
}
