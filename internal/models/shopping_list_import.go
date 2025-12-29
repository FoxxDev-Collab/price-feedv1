package models

// ParsedShoppingItem represents a single parsed line from a shopping list
type ParsedShoppingItem struct {
	RawText    string  `json:"raw_text"`
	Quantity   float64 `json:"quantity"`
	Unit       string  `json:"unit"`
	Name       string  `json:"name"`
	Notes      string  `json:"notes"`
	LineNumber int     `json:"line_number"`
}

// ShoppingListImportRequest is the API request body
type ShoppingListImportRequest struct {
	Content string `json:"content"`
}

// MatchedShoppingItem represents a parsed item with match results
type MatchedShoppingItem struct {
	ParsedItem  ParsedShoppingItem `json:"parsed_item"`
	BestMatch   *ItemMatchResult   `json:"best_match,omitempty"`
	Suggestions []ItemMatchResult  `json:"suggestions"`
	IsMatched   bool               `json:"is_matched"`
}

// ItemMatchResult represents a single match suggestion
type ItemMatchResult struct {
	ItemID     int      `json:"item_id"`
	Name       string   `json:"name"`
	Brand      *string  `json:"brand,omitempty"`
	Size       *float64 `json:"size,omitempty"`
	Unit       *string  `json:"unit,omitempty"`
	Confidence float64  `json:"confidence"`
	MatchType  string   `json:"match_type"`
}

// ShoppingListImportResponse is the API response
type ShoppingListImportResponse struct {
	Items          []MatchedShoppingItem `json:"items"`
	TotalParsed    int                   `json:"total_parsed"`
	MatchedCount   int                   `json:"matched_count"`
	UnmatchedCount int                   `json:"unmatched_count"`
}

// BulkCreateItemsRequest for creating multiple items at once
type BulkCreateItemsRequest struct {
	Items []CreateItemFromImport `json:"items"`
}

// CreateItemFromImport is a single item to create from import
type CreateItemFromImport struct {
	Name        string   `json:"name"`
	Brand       *string  `json:"brand,omitempty"`
	Size        *float64 `json:"size,omitempty"`
	Unit        *string  `json:"unit,omitempty"`
	Description *string  `json:"description,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	IsPrivate   bool     `json:"is_private"`
}

// BulkCreateItemsResponse for created items
type BulkCreateItemsResponse struct {
	Created []Item   `json:"created"`
	Errors  []string `json:"errors,omitempty"`
}
