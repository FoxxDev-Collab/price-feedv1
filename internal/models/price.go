package models

import (
	"time"
)

// StorePrice represents a price for an item at a specific store
type StorePrice struct {
	ID            int        `json:"id"`
	StoreID       int        `json:"store_id"`
	ItemID        int        `json:"item_id"`
	Price         float64    `json:"price"`
	UserID        *int       `json:"user_id,omitempty"`
	IsShared      bool       `json:"is_shared"` // If true, price is visible to community
	VerifiedCount int        `json:"verified_count"`
	LastVerified  *time.Time `json:"last_verified,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// StorePriceWithDetails includes item, store, and user info
type StorePriceWithDetails struct {
	StorePrice
	ItemName      string  `json:"item_name"`
	ItemBrand     *string `json:"item_brand,omitempty"`
	StoreName     string  `json:"store_name"`
	StoreAddress  string  `json:"store_address"`
	StoreCity     string  `json:"store_city"`
	StoreState    string  `json:"store_state"`
	StoreZipCode  string  `json:"store_zip_code"`
	RegionID      *int    `json:"region_id,omitempty"`
	RegionName    *string `json:"region_name,omitempty"`
	UserName      *string `json:"user_name,omitempty"`
	UserEmail     *string `json:"user_email,omitempty"`
}

// CreatePriceRequest is the request body for creating a price
type CreatePriceRequest struct {
	StoreID  int     `json:"store_id"`
	ItemID   int     `json:"item_id"`
	Price    float64 `json:"price"`
	IsShared bool    `json:"is_shared"` // If true, price is shared with community (default true)
}

// UpdatePriceRequest is the request body for updating a price
type UpdatePriceRequest struct {
	Price *float64 `json:"price,omitempty"`
}

// PriceListParams contains parameters for listing prices
type PriceListParams struct {
	Limit    int
	Offset   int
	Search   string
	StoreID  *int
	ItemID   *int
	RegionID *int
	Verified *bool
	DateFrom *time.Time
	DateTo   *time.Time
	IsShared *bool // Filter by shared/private prices
	UserID   *int  // Filter by submitter (for private prices)
}

// PriceStats contains aggregate statistics for prices
type PriceStats struct {
	TotalPrices   int `json:"total_prices"`
	TodayCount    int `json:"today_count"`
	WeekCount     int `json:"week_count"`
	VerifiedCount int `json:"verified_count"`
	FlaggedCount  int `json:"flagged_count"`
}

// PriceVerification represents a user's verification of a price
type PriceVerification struct {
	ID         int       `json:"id"`
	PriceID    int       `json:"price_id"`
	UserID     int       `json:"user_id"`
	IsAccurate bool      `json:"is_accurate"`
	CreatedAt  time.Time `json:"created_at"`
}

// PriceHistoryEntry represents a single historical price record
type PriceHistoryEntry struct {
	ID            int        `json:"id"`
	StoreID       int        `json:"store_id"`
	ItemID        int        `json:"item_id"`
	Price         float64    `json:"price"`
	PreviousPrice *float64   `json:"previous_price,omitempty"`
	UserID        *int       `json:"user_id,omitempty"`
	RecordedAt    time.Time  `json:"recorded_at"`
	// Joined fields
	StoreName     string     `json:"store_name,omitempty"`
	UserName      *string    `json:"user_name,omitempty"`
	ChangePercent *float64   `json:"change_percent,omitempty"`
}

// PriceTrend represents the trend direction and magnitude for a price
type PriceTrend struct {
	Direction     string  `json:"direction"`      // "up", "down", or "stable"
	ChangeAmount  float64 `json:"change_amount"`  // Absolute change in price
	ChangePercent float64 `json:"change_percent"` // Percentage change
	PeriodDays    int     `json:"period_days"`    // Period over which trend is calculated
}

// PriceHistoryResponse is the response for price history endpoint
type PriceHistoryResponse struct {
	Item    PriceHistoryItem    `json:"item"`
	Trend   *PriceTrend         `json:"trend,omitempty"`
	History []PriceHistoryEntry `json:"history"`
}

// PriceHistoryItem contains item details for history response
type PriceHistoryItem struct {
	ID           int     `json:"id"`
	Name         string  `json:"name"`
	Brand        *string `json:"brand,omitempty"`
	CurrentPrice float64 `json:"current_price"`
}

// PriceHistoryParams contains parameters for querying price history
type PriceHistoryParams struct {
	ItemID  int
	StoreID *int
	Limit   int
}
