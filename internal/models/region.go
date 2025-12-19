package models

import (
	"time"
)

// Region represents a geographic region for price tracking
type Region struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	State     string    `json:"state"`
	ZipCodes  []string  `json:"zip_codes"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// RegionWithStats includes aggregated statistics
type RegionWithStats struct {
	Region
	StoreCount int `json:"store_count"`
	UserCount  int `json:"user_count"`
	PriceCount int `json:"price_count"`
}

// CreateRegionRequest is the request body for creating a region
type CreateRegionRequest struct {
	Name     string   `json:"name"`
	State    string   `json:"state"`
	ZipCodes []string `json:"zip_codes"`
}

// UpdateRegionRequest is the request body for updating a region
type UpdateRegionRequest struct {
	Name     *string   `json:"name,omitempty"`
	State    *string   `json:"state,omitempty"`
	ZipCodes *[]string `json:"zip_codes,omitempty"`
}

// RegionListParams contains parameters for listing regions
type RegionListParams struct {
	Limit  int
	Offset int
	Search string
	State  string
}
