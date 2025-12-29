package models

import (
	"time"
)

// InventoryItem represents an item in a user's inventory/pantry
type InventoryItem struct {
	ID     int `json:"id"`
	UserID int `json:"user_id"`

	// Reference to catalog item (optional)
	ItemID *int `json:"item_id,omitempty"`

	// Custom item fields (used when ItemID is nil)
	CustomName  *string  `json:"custom_name,omitempty"`
	CustomBrand *string  `json:"custom_brand,omitempty"`
	CustomSize  *float64 `json:"custom_size,omitempty"`
	CustomUnit  *string  `json:"custom_unit,omitempty"`

	// Inventory tracking
	Quantity float64 `json:"quantity"`
	Unit     *string `json:"unit,omitempty"`

	// Stock management
	LowStockThreshold    float64 `json:"low_stock_threshold"`
	LowStockAlertEnabled bool    `json:"low_stock_alert_enabled"`

	// Dates
	PurchaseDate   *time.Time `json:"purchase_date,omitempty"`
	ExpirationDate *time.Time `json:"expiration_date,omitempty"`

	// Organization
	Location *string `json:"location,omitempty"`
	Notes    *string `json:"notes,omitempty"`

	// Timestamps
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// InventoryItemWithDetails includes joined item data for display
type InventoryItemWithDetails struct {
	InventoryItem

	// From joined items table (when ItemID is not null)
	ItemName  *string  `json:"item_name,omitempty"`
	ItemBrand *string  `json:"item_brand,omitempty"`
	ItemSize  *float64 `json:"item_size,omitempty"`
	ItemUnit  *string  `json:"item_unit,omitempty"`

	// Computed display fields
	DisplayName  string  `json:"display_name"`
	DisplayBrand *string `json:"display_brand,omitempty"`

	// Status flags (computed)
	IsLowStock      bool `json:"is_low_stock"`
	IsExpired       bool `json:"is_expired"`
	ExpiresSoon     bool `json:"expires_soon"` // Within 7 days
	DaysUntilExpiry *int `json:"days_until_expiry,omitempty"`
}

// InventorySummary provides aggregate stats for the inventory dashboard
type InventorySummary struct {
	TotalItems        int      `json:"total_items"`
	LowStockCount     int      `json:"low_stock_count"`
	ExpiredCount      int      `json:"expired_count"`
	ExpiringSoonCount int      `json:"expiring_soon_count"`
	UniqueLocations   []string `json:"unique_locations"`
}

// CreateInventoryItemRequest is the request body for adding inventory items
type CreateInventoryItemRequest struct {
	// Option 1: Reference existing catalog item
	ItemID *int `json:"item_id,omitempty"`

	// Option 2: Create custom inventory-only item
	CustomName  *string  `json:"custom_name,omitempty"`
	CustomBrand *string  `json:"custom_brand,omitempty"`
	CustomSize  *float64 `json:"custom_size,omitempty"`
	CustomUnit  *string  `json:"custom_unit,omitempty"`

	// Inventory fields
	Quantity             float64    `json:"quantity"`
	Unit                 *string    `json:"unit,omitempty"`
	LowStockThreshold    *float64   `json:"low_stock_threshold,omitempty"`
	LowStockAlertEnabled *bool      `json:"low_stock_alert_enabled,omitempty"`
	PurchaseDate         *time.Time `json:"purchase_date,omitempty"`
	ExpirationDate       *time.Time `json:"expiration_date,omitempty"`
	Location             *string    `json:"location,omitempty"`
	Notes                *string    `json:"notes,omitempty"`
}

// UpdateInventoryItemRequest is the request body for updating inventory items
type UpdateInventoryItemRequest struct {
	Quantity             *float64   `json:"quantity,omitempty"`
	Unit                 *string    `json:"unit,omitempty"`
	LowStockThreshold    *float64   `json:"low_stock_threshold,omitempty"`
	LowStockAlertEnabled *bool      `json:"low_stock_alert_enabled,omitempty"`
	PurchaseDate         *time.Time `json:"purchase_date,omitempty"`
	ExpirationDate       *time.Time `json:"expiration_date,omitempty"`
	Location             *string    `json:"location,omitempty"`
	Notes                *string    `json:"notes,omitempty"`

	// Allow updating custom item details
	CustomName  *string  `json:"custom_name,omitempty"`
	CustomBrand *string  `json:"custom_brand,omitempty"`
	CustomSize  *float64 `json:"custom_size,omitempty"`
	CustomUnit  *string  `json:"custom_unit,omitempty"`
}

// InventoryListParams contains parameters for listing inventory
type InventoryListParams struct {
	Limit        int
	Offset       int
	UserID       int
	Location     string // Filter by location
	Search       string // Search by name
	LowStock     *bool  // Filter for low stock items only
	Expired      *bool  // Filter for expired items only
	ExpiringSoon *bool  // Filter for items expiring within 7 days
	SortBy       string // "name", "expiration", "quantity", "updated"
	SortOrder    string // "asc" or "desc"
}

// AdjustInventoryQuantityRequest for adjusting item quantity
type AdjustInventoryQuantityRequest struct {
	Adjustment float64 `json:"adjustment"`
}

// AddInventoryToListRequest for quick add to shopping list
type AddInventoryToListRequest struct {
	ListID   int `json:"list_id"`
	Quantity int `json:"quantity"`
}
