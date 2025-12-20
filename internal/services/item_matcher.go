package services

import (
	"context"
	"strings"

	"github.com/foxxcyber/price-feed/internal/database"
	"github.com/foxxcyber/price-feed/internal/models"
)

// ItemMatcher handles fuzzy matching of items
type ItemMatcher struct {
	db *database.DB
}

// NewItemMatcher creates a new item matcher
func NewItemMatcher(db *database.DB) *ItemMatcher {
	return &ItemMatcher{
		db: db,
	}
}

// MatchedReceiptItem represents a parsed item with match results
type MatchedReceiptItem struct {
	ParsedItem  models.ParsedItem
	BestMatch   *models.MatchResult
	Suggestions []models.MatchResult
}

// FindMatches finds items similar to the given name
func (m *ItemMatcher) FindMatches(ctx context.Context, itemName string, limit int) ([]models.MatchResult, error) {
	// Normalize the item name
	normalized := m.normalizeItemName(itemName)

	// Use database fuzzy matching
	return m.db.FindSimilarItems(ctx, normalized, limit)
}

// MatchReceiptItems matches a list of parsed items against the database
func (m *ItemMatcher) MatchReceiptItems(ctx context.Context, items []models.ParsedItem) ([]MatchedReceiptItem, error) {
	var results []MatchedReceiptItem

	for _, item := range items {
		matched := MatchedReceiptItem{
			ParsedItem: item,
		}

		// Find similar items
		suggestions, err := m.FindMatches(ctx, item.Name, 5)
		if err != nil {
			// Log error but continue processing
			results = append(results, matched)
			continue
		}

		matched.Suggestions = suggestions

		// Use the best match if confidence is high enough
		if len(suggestions) > 0 && suggestions[0].Confidence >= 0.5 {
			matched.BestMatch = &suggestions[0]
		}

		results = append(results, matched)
	}

	return results, nil
}

// normalizeItemName cleans up an item name for better matching
func (m *ItemMatcher) normalizeItemName(name string) string {
	name = strings.ToLower(name)

	// Common abbreviation expansions
	replacements := map[string]string{
		"org ":    "organic ",
		"whl ":    "whole ",
		"chkn":    "chicken",
		"brst":    "breast",
		"bnls":    "boneless",
		"sknls":   "skinless",
		"gal":     "gallon",
		"qt":      "quart",
		"pt":      "pint",
		"oz":      "ounce",
		"lb":      "pound",
		"lbs":     "pounds",
		"pkg":     "package",
		"btl":     "bottle",
		"cn":      "can",
		"bx":      "box",
		"bg":      "bag",
		"ea":      "each",
		"ct":      "count",
		"pc":      "piece",
		"pcs":     "pieces",
		"lrg":     "large",
		"med":     "medium",
		"sml":     "small",
		"frsh":    "fresh",
		"frzn":    "frozen",
		"slf":     "self",
		"rsg":     "rising",
		"flr":     "flour",
		"veg":     "vegetable",
		"vegs":    "vegetables",
		"frt":     "fruit",
		"jce":     "juice",
		"mlk":     "milk",
		"chse":    "cheese",
		"brd":     "bread",
		"wht":     "white",
		"brn":     "brown",
		"grn":     "green",
		"red":     "red",
		"yel":     "yellow",
		"blu":     "blue",
		"blk":     "black",
	}

	for abbrev, full := range replacements {
		name = strings.ReplaceAll(name, abbrev, full)
	}

	// Remove common price suffixes that might be left
	suffixes := []string{" f", " t", " n", " @"}
	for _, suffix := range suffixes {
		if strings.HasSuffix(name, suffix) {
			name = strings.TrimSuffix(name, suffix)
		}
	}

	// Remove extra whitespace
	name = strings.Join(strings.Fields(name), " ")

	return strings.TrimSpace(name)
}

// GetMatchConfidenceLevel returns a human-readable confidence level
func GetMatchConfidenceLevel(confidence float64) string {
	switch {
	case confidence >= 0.9:
		return "high"
	case confidence >= 0.7:
		return "medium"
	case confidence >= 0.5:
		return "low"
	default:
		return "none"
	}
}
