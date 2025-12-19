package models

import (
	"time"
)

// Item represents a product that can be priced
type Item struct {
	ID                int        `json:"id"`
	Name              string     `json:"name"`
	Brand             *string    `json:"brand,omitempty"`
	Size              *float64   `json:"size,omitempty"`
	Unit              *string    `json:"unit,omitempty"`
	Description       *string    `json:"description,omitempty"`
	Verified          bool       `json:"verified"`
	VerificationCount int        `json:"verification_count"`
	CreatedBy         *int       `json:"created_by,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

// ItemWithStats includes aggregated statistics
type ItemWithStats struct {
	Item
	PriceCount int      `json:"price_count"`
	AvgPrice   *float64 `json:"avg_price,omitempty"`
	MinPrice   *float64 `json:"min_price,omitempty"`
	MaxPrice   *float64 `json:"max_price,omitempty"`
	Tags       []string `json:"tags"`
}

// CreateItemRequest is the request body for creating an item
type CreateItemRequest struct {
	Name        string   `json:"name"`
	Brand       *string  `json:"brand,omitempty"`
	Size        *float64 `json:"size,omitempty"`
	Unit        *string  `json:"unit,omitempty"`
	Description *string  `json:"description,omitempty"`
	Tags        []string `json:"tags,omitempty"`
}

// UpdateItemRequest is the request body for updating an item
type UpdateItemRequest struct {
	Name        *string  `json:"name,omitempty"`
	Brand       *string  `json:"brand,omitempty"`
	Size        *float64 `json:"size,omitempty"`
	Unit        *string  `json:"unit,omitempty"`
	Description *string  `json:"description,omitempty"`
	Verified    *bool    `json:"verified,omitempty"`
	Tags        []string `json:"tags,omitempty"`
}

// ItemListParams contains parameters for listing items
type ItemListParams struct {
	Limit  int
	Offset int
	Search string
	Tag    string
}

// ItemStats contains aggregate statistics for items
type ItemStats struct {
	TotalItems     int `json:"total_items"`
	VerifiedCount  int `json:"verified_count"`
	WithPrices     int `json:"with_prices"`
	TotalTags      int `json:"total_tags"`
}

// Tag represents a product tag/category
type Tag struct {
	ID         int       `json:"id"`
	Name       string    `json:"name"`
	Slug       string    `json:"slug"`
	UsageCount int       `json:"usage_count"`
	CreatedAt  time.Time `json:"created_at"`
}
