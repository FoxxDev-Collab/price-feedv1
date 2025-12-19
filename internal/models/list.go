package models

import (
	"time"
)

// ListStatus represents the status of a shopping list
type ListStatus string

const (
	ListStatusActive    ListStatus = "active"
	ListStatusCompleted ListStatus = "completed"
)

// ShoppingList represents a user's shopping list
type ShoppingList struct {
	ID          int        `json:"id"`
	UserID      int        `json:"user_id"`
	Name        string     `json:"name"`
	Status      ListStatus `json:"status"`
	TargetDate  *time.Time `json:"target_date,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// ShoppingListItem represents an item in a shopping list
type ShoppingListItem struct {
	ID        int       `json:"id"`
	ListID    int       `json:"list_id"`
	ItemID    int       `json:"item_id"`
	Quantity  int       `json:"quantity"`
	CreatedAt time.Time `json:"created_at"`
}

// ShoppingListItemWithDetails includes item info
type ShoppingListItemWithDetails struct {
	ShoppingListItem
	ItemName  string   `json:"item_name"`
	ItemBrand *string  `json:"item_brand,omitempty"`
	ItemSize  *float64 `json:"item_size,omitempty"`
	ItemUnit  *string  `json:"item_unit,omitempty"`
	BestPrice *float64 `json:"best_price,omitempty"` // Best available price for this item
	BestStore *string  `json:"best_store,omitempty"` // Store with best price
}

// ShoppingListWithItems includes the list and all its items
type ShoppingListWithItems struct {
	ShoppingList
	Items          []ShoppingListItemWithDetails `json:"items"`
	ItemCount      int                           `json:"item_count"`
	EstimatedTotal float64                       `json:"estimated_total"` // Sum of best prices * quantities
}

// ShoppingListSummary is a compact representation for list views
type ShoppingListSummary struct {
	ID             int        `json:"id"`
	Name           string     `json:"name"`
	Status         ListStatus `json:"status"`
	TargetDate     *time.Time `json:"target_date,omitempty"`
	CompletedAt    *time.Time `json:"completed_at,omitempty"`
	ItemCount      int        `json:"item_count"`
	EstimatedTotal float64    `json:"estimated_total"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// StorePlan represents a generated shopping optimization plan
type StorePlan struct {
	ID                  int       `json:"id"`
	ListID              int       `json:"list_id"`
	TotalSavings        float64   `json:"total_savings"`
	RecommendedStrategy string    `json:"recommended_strategy"` // "single_store" or "multi_store"
	GeneratedAt         time.Time `json:"generated_at"`
}

// StorePlanItem represents an item in a store plan
type StorePlanItem struct {
	ID        int       `json:"id"`
	PlanID    int       `json:"plan_id"`
	StoreID   int       `json:"store_id"`
	ItemID    int       `json:"item_id"`
	Quantity  int       `json:"quantity"`
	Price     float64   `json:"price"`
	CreatedAt time.Time `json:"created_at"`
}

// StorePlanItemWithDetails includes store and item info
type StorePlanItemWithDetails struct {
	StorePlanItem
	StoreName string  `json:"store_name"`
	ItemName  string  `json:"item_name"`
	ItemBrand *string `json:"item_brand,omitempty"`
}

// SingleStoreOption represents the best single-store shopping option
type SingleStoreOption struct {
	StoreID      int      `json:"store_id"`
	StoreName    string   `json:"store_name"`
	TotalCost    float64  `json:"total_cost"`
	ItemsFound   int      `json:"items_found"`
	ItemsMissing []string `json:"items_missing,omitempty"`
}

// MultiStoreBreakdown represents items to buy at a specific store
type MultiStoreBreakdown struct {
	StoreID   int                        `json:"store_id"`
	StoreName string                     `json:"store_name"`
	Items     []StorePlanItemWithDetails `json:"items"`
	Subtotal  float64                    `json:"subtotal"`
}

// MultiStoreOption represents the optimal multi-store shopping option
type MultiStoreOption struct {
	Stores       []MultiStoreBreakdown `json:"stores"`
	TotalCost    float64               `json:"total_cost"`
	TotalSavings float64               `json:"total_savings"` // Savings vs best single store
	TripCount    int                   `json:"trip_count"`
}

// ShoppingPlanResult is the complete optimization result
type ShoppingPlanResult struct {
	ListID         int                `json:"list_id"`
	SingleStore    *SingleStoreOption `json:"single_store,omitempty"`
	MultiStore     *MultiStoreOption  `json:"multi_store,omitempty"`
	Recommendation string             `json:"recommendation"` // "single_store" or "multi_store"
	GeneratedAt    time.Time          `json:"generated_at"`
}

// PriceComparisonCell represents a single cell in the comparison grid
type PriceComparisonCell struct {
	Price         *float64 `json:"price,omitempty"` // nil if no price data
	VerifiedCount int      `json:"verified_count"`
	SubmittedBy   *string  `json:"submitted_by,omitempty"`
	UpdatedAt     *string  `json:"updated_at,omitempty"`
	IsBest        bool     `json:"is_best"` // True if this is the lowest price for the item
}

// PriceComparisonRow represents a row (item) in the comparison grid
type PriceComparisonRow struct {
	ItemID    int                        `json:"item_id"`
	ItemName  string                     `json:"item_name"`
	ItemBrand *string                    `json:"item_brand,omitempty"`
	ItemSize  *float64                   `json:"item_size,omitempty"`
	ItemUnit  *string                    `json:"item_unit,omitempty"`
	Prices    map[int]PriceComparisonCell `json:"prices"` // Key is store_id
	BestPrice *float64                   `json:"best_price,omitempty"`
	BestStore *int                       `json:"best_store,omitempty"`
}

// PriceComparisonResult is the full comparison grid
type PriceComparisonResult struct {
	Stores []StoreBasic         `json:"stores"` // Column headers
	Items  []PriceComparisonRow `json:"items"`  // Rows
}

// StoreBasic is minimal store info for headers
type StoreBasic struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// Request types

// CreateListRequest is the request body for creating a shopping list
type CreateListRequest struct {
	Name       string     `json:"name"`
	TargetDate *time.Time `json:"target_date,omitempty"`
}

// UpdateListRequest is the request body for updating a shopping list
type UpdateListRequest struct {
	Name       *string    `json:"name,omitempty"`
	TargetDate *time.Time `json:"target_date,omitempty"`
}

// AddListItemRequest is the request body for adding an item to a list
type AddListItemRequest struct {
	ItemID   int `json:"item_id"`
	Quantity int `json:"quantity"`
}

// UpdateListItemRequest is the request body for updating a list item
type UpdateListItemRequest struct {
	Quantity int `json:"quantity"`
}

// ListListParams contains parameters for listing shopping lists
type ListListParams struct {
	Limit  int
	Offset int
	UserID int        // Required - lists are always scoped to a user
	Status ListStatus // Optional - filter by status (active, completed)
}

// CompareParams contains parameters for price comparison
type CompareParams struct {
	StoreIDs []int // Stores to compare
	ItemIDs  []int // Items to compare (optional, if empty compare all items with prices)
	RegionID *int  // Filter by region
	UserID   *int  // Include user's private prices
}

// PriceConfirmation represents a price confirmation during checkout
type PriceConfirmation struct {
	ItemID     int      `json:"item_id"`
	StoreID    int      `json:"store_id"`
	IsAccurate bool     `json:"is_accurate"`
	NewPrice   *float64 `json:"new_price,omitempty"` // If not accurate, user can provide new price
}

// CompleteListRequest is the request body for completing a shopping list
type CompleteListRequest struct {
	PriceConfirmations []PriceConfirmation `json:"price_confirmations,omitempty"`
}
