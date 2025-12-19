package models

import (
	"time"
)

// Store represents a physical store location
type Store struct {
	ID                int        `json:"id"`
	Name              string     `json:"name"`
	StreetAddress     string     `json:"street_address"`
	City              string     `json:"city"`
	State             string     `json:"state"`
	ZipCode           string     `json:"zip_code"`
	RegionID          *int       `json:"region_id,omitempty"`
	StoreType         *string    `json:"store_type,omitempty"`
	Chain             *string    `json:"chain,omitempty"`
	Latitude          *float64   `json:"latitude,omitempty"`
	Longitude         *float64   `json:"longitude,omitempty"`
	Verified          bool       `json:"verified"`
	VerificationCount int        `json:"verification_count"`
	IsPrivate         bool       `json:"is_private"`
	CreatedBy         *int       `json:"created_by,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

// StoreWithStats includes aggregated statistics and region info
type StoreWithStats struct {
	Store
	RegionName       *string `json:"region_name,omitempty"`
	PriceCount       int     `json:"price_count"`
	ContributorCount int     `json:"contributor_count"` // Number of unique users who added prices
}

// CreateStoreRequest is the request body for creating a store
type CreateStoreRequest struct {
	Name          string   `json:"name"`
	StreetAddress string   `json:"street_address"`
	City          string   `json:"city"`
	State         string   `json:"state"`
	ZipCode       string   `json:"zip_code"`
	RegionID      *int     `json:"region_id,omitempty"`
	StoreType     *string  `json:"store_type,omitempty"`
	Chain         *string  `json:"chain,omitempty"`
	Latitude      *float64 `json:"latitude,omitempty"`
	Longitude     *float64 `json:"longitude,omitempty"`
	Verified      bool     `json:"verified"`
	IsPrivate     bool     `json:"is_private"` // If true, store is only visible to creator
}

// UpdateStoreRequest is the request body for updating a store
type UpdateStoreRequest struct {
	Name          *string  `json:"name,omitempty"`
	StreetAddress *string  `json:"street_address,omitempty"`
	City          *string  `json:"city,omitempty"`
	State         *string  `json:"state,omitempty"`
	ZipCode       *string  `json:"zip_code,omitempty"`
	RegionID      *int     `json:"region_id,omitempty"`
	StoreType     *string  `json:"store_type,omitempty"`
	Chain         *string  `json:"chain,omitempty"`
	Latitude      *float64 `json:"latitude,omitempty"`
	Longitude     *float64 `json:"longitude,omitempty"`
	Verified      *bool    `json:"verified,omitempty"`
}

// StoreListParams contains parameters for listing stores
type StoreListParams struct {
	Limit     int
	Offset    int
	Search    string
	RegionID  *int
	State     string
	Verified  *bool
	IsPrivate *bool // Filter by private/community stores
	UserID    *int  // Filter by creator (for private stores)
}

// StoreStats contains aggregate statistics for stores
type StoreStats struct {
	TotalStores   int `json:"total_stores"`
	VerifiedCount int `json:"verified_count"`
	PendingCount  int `json:"pending_count"`
	TotalPrices   int `json:"total_prices"`
}
