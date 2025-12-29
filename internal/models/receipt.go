package models

import (
	"time"
)

// ReceiptStatus represents the processing status of a receipt
type ReceiptStatus string

const (
	ReceiptStatusPending    ReceiptStatus = "pending"
	ReceiptStatusProcessing ReceiptStatus = "processing"
	ReceiptStatusCompleted  ReceiptStatus = "completed"
	ReceiptStatusFailed     ReceiptStatus = "failed"
	ReceiptStatusConfirmed  ReceiptStatus = "confirmed"
)

// MatchStatus represents the matching status of a receipt item
type MatchStatus string

const (
	MatchStatusPending  MatchStatus = "pending"
	MatchStatusMatched  MatchStatus = "matched"
	MatchStatusNewItem  MatchStatus = "new_item"
	MatchStatusSkipped  MatchStatus = "skipped"
)

// Receipt represents an uploaded receipt image
type Receipt struct {
	ID               int            `json:"id"`
	UserID           int            `json:"user_id"`
	StoreID          *int           `json:"store_id,omitempty"`
	S3Bucket         string         `json:"s3_bucket"`
	S3Key            string         `json:"s3_key"`
	OriginalFilename *string        `json:"original_filename,omitempty"`
	ContentType      *string        `json:"content_type,omitempty"`
	FileSizeBytes    *int64         `json:"file_size_bytes,omitempty"`
	Status           ReceiptStatus  `json:"status"`
	OCRText          *string        `json:"ocr_text,omitempty"`
	ErrorMessage     *string        `json:"error_message,omitempty"`
	ReceiptDate      *time.Time     `json:"receipt_date,omitempty"`
	ReceiptTotal     *float64       `json:"receipt_total,omitempty"`
	UploadedAt       time.Time      `json:"uploaded_at"`
	ProcessedAt      *time.Time     `json:"processed_at,omitempty"`
	ConfirmedAt      *time.Time     `json:"confirmed_at,omitempty"`
	ExpiresAt        time.Time      `json:"expires_at"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
}

// ReceiptWithItems includes the parsed items
type ReceiptWithItems struct {
	Receipt
	Items     []ReceiptItemWithSuggestions `json:"items"`
	StoreName *string                      `json:"store_name,omitempty"`
	ImageURL  *string                      `json:"image_url,omitempty"`
}

// ReceiptItem represents a parsed line item from a receipt
type ReceiptItem struct {
	ID               int         `json:"id"`
	ReceiptID        int         `json:"receipt_id"`
	RawText          string      `json:"raw_text"`
	ExtractedName    *string     `json:"extracted_name,omitempty"`
	ExtractedPrice   *float64    `json:"extracted_price,omitempty"`
	ExtractedQuantity int        `json:"extracted_quantity"`
	MatchedItemID    *int        `json:"matched_item_id,omitempty"`
	MatchConfidence  *float64    `json:"match_confidence,omitempty"`
	MatchStatus      MatchStatus `json:"match_status"`
	ConfirmedItemID  *int        `json:"confirmed_item_id,omitempty"`
	ConfirmedPrice   *float64    `json:"confirmed_price,omitempty"`
	IsConfirmed      bool        `json:"is_confirmed"`
	CreatedItemID    *int        `json:"created_item_id,omitempty"`
	LineNumber       *int        `json:"line_number,omitempty"`
	CreatedAt        time.Time   `json:"created_at"`
	UpdatedAt        time.Time   `json:"updated_at"`
}

// ReceiptItemWithSuggestions includes match suggestions
type ReceiptItemWithSuggestions struct {
	ReceiptItem
	MatchedItemName *string          `json:"matched_item_name,omitempty"`
	Suggestions     []ItemSuggestion `json:"suggestions,omitempty"`
}

// ItemSuggestion represents a suggested item match
type ItemSuggestion struct {
	ItemID     int     `json:"item_id"`
	Name       string  `json:"name"`
	Brand      *string `json:"brand,omitempty"`
	Confidence float64 `json:"confidence"`
	MatchType  string  `json:"match_type"`
}

// CreateReceiptRequest is used when uploading a receipt
type CreateReceiptRequest struct {
	UserID           int
	StoreID          *int
	S3Bucket         string
	S3Key            string
	OriginalFilename string
	ContentType      string
	FileSizeBytes    int64
}

// CreateReceiptItemRequest is used when creating parsed items
type CreateReceiptItemRequest struct {
	ReceiptID        int
	RawText          string
	ExtractedName    *string
	ExtractedPrice   *float64
	ExtractedQuantity int
	MatchedItemID    *int
	MatchConfidence  *float64
	MatchStatus      MatchStatus
	LineNumber       int
}

// UpdateReceiptItemRequest is used when user confirms/updates an item
type UpdateReceiptItemRequest struct {
	ConfirmedItemID *int     `json:"confirmed_item_id,omitempty"`
	ConfirmedPrice  *float64 `json:"confirmed_price,omitempty"`
	MatchStatus     *string  `json:"match_status,omitempty"`
	IsConfirmed     *bool    `json:"is_confirmed,omitempty"`
	CreateNewItem   bool     `json:"create_new_item,omitempty"`
	NewItemName     *string  `json:"new_item_name,omitempty"`
}

// ConfirmReceiptRequest is used when confirming all items
type ConfirmReceiptRequest struct {
	StoreID int                      `json:"store_id"`
	Items   []ConfirmReceiptItemData `json:"items"`
}

// ConfirmReceiptItemData represents a single item confirmation
type ConfirmReceiptItemData struct {
	ReceiptItemID int      `json:"receipt_item_id"`
	ItemID        *int     `json:"item_id,omitempty"`
	Price         *float64 `json:"price,omitempty"`
	Skip          bool     `json:"skip,omitempty"`
	CreateNewItem bool     `json:"create_new_item,omitempty"`
	NewItemName   *string  `json:"new_item_name,omitempty"`
}

// ReceiptListParams contains parameters for listing receipts
type ReceiptListParams struct {
	Limit  int
	Offset int
	Status *string
	UserID int
}

// ParsedItem represents an item parsed from OCR text
type ParsedItem struct {
	RawText    string
	Name       string
	Price      float64
	Quantity   int
	LineNumber int
}

// ParsedReceipt represents the parsed result from receipt OCR
type ParsedReceipt struct {
	Items     []ParsedItem
	Total     *float64
	Date      *time.Time
	StoreName *string
}

// MatchResult represents a fuzzy match result
type MatchResult struct {
	ItemID      int
	Name        string
	Brand       *string
	Confidence  float64
	MatchType   string
}

// SpendingSummary represents monthly spending aggregations
type SpendingSummary struct {
	Months         []MonthlySpending `json:"months"`
	GrandTotal     float64           `json:"grand_total"`
	AverageMonthly float64           `json:"average_monthly"`
}

// MonthlySpending represents spending for a single month
type MonthlySpending struct {
	Month        string         `json:"month"`
	Total        float64        `json:"total"`
	ReceiptCount int            `json:"receipt_count"`
	Stores       []StoreSpend   `json:"stores"`
}

// StoreSpend represents spending at a specific store
type StoreSpend struct {
	StoreID      int     `json:"store_id"`
	StoreName    string  `json:"store_name"`
	Total        float64 `json:"total"`
	ReceiptCount int     `json:"receipt_count"`
}
