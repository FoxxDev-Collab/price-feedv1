package handlers

import (
	"fmt"

	"github.com/gofiber/fiber/v2"

	"github.com/foxxcyber/price-feed/internal/middleware"
	"github.com/foxxcyber/price-feed/internal/models"
	"github.com/foxxcyber/price-feed/internal/services"
)

// ParseShoppingList parses markdown content and matches items
// POST /api/import/shopping-list
func (h *Handler) ParseShoppingList(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		return Error(c, fiber.StatusUnauthorized, "unauthorized")
	}

	var req models.ShoppingListImportRequest
	if err := c.BodyParser(&req); err != nil {
		return Error(c, fiber.StatusBadRequest, "invalid request body")
	}

	if req.Content == "" {
		return Error(c, fiber.StatusBadRequest, "content is required")
	}

	// Parse the shopping list
	parser := services.NewShoppingListParser()
	parsedItems, err := parser.Parse(req.Content)
	if err != nil {
		return Error(c, fiber.StatusBadRequest, "failed to parse shopping list")
	}

	if len(parsedItems) == 0 {
		return Error(c, fiber.StatusBadRequest, "no items found in shopping list")
	}

	// Match each item against the database
	matcher := services.NewItemMatcher(h.db)
	var matchedItems []models.MatchedShoppingItem
	matchedCount := 0

	for _, parsed := range parsedItems {
		matched := models.MatchedShoppingItem{
			ParsedItem: parsed,
			IsMatched:  false,
		}

		// Find matches using existing item matcher
		suggestions, err := matcher.FindMatches(c.Context(), parsed.Name, 5)
		if err == nil && len(suggestions) > 0 {
			// Convert to ItemMatchResult
			for _, s := range suggestions {
				matched.Suggestions = append(matched.Suggestions, models.ItemMatchResult{
					ItemID:     s.ItemID,
					Name:       s.Name,
					Brand:      s.Brand,
					Confidence: s.Confidence,
					MatchType:  s.MatchType,
				})
			}

			// Use best match if confidence is high enough (0.6 threshold)
			if suggestions[0].Confidence >= 0.6 {
				matched.BestMatch = &models.ItemMatchResult{
					ItemID:     suggestions[0].ItemID,
					Name:       suggestions[0].Name,
					Brand:      suggestions[0].Brand,
					Confidence: suggestions[0].Confidence,
					MatchType:  suggestions[0].MatchType,
				}
				matched.IsMatched = true
				matchedCount++
			}
		}

		matchedItems = append(matchedItems, matched)
	}

	response := models.ShoppingListImportResponse{
		Items:          matchedItems,
		TotalParsed:    len(parsedItems),
		MatchedCount:   matchedCount,
		UnmatchedCount: len(parsedItems) - matchedCount,
	}

	return Success(c, response)
}

// BulkCreateItems creates multiple items from import
// POST /api/import/create-items
func (h *Handler) BulkCreateItems(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		return Error(c, fiber.StatusUnauthorized, "unauthorized")
	}

	var req models.BulkCreateItemsRequest
	if err := c.BodyParser(&req); err != nil {
		return Error(c, fiber.StatusBadRequest, "invalid request body")
	}

	if len(req.Items) == 0 {
		return Error(c, fiber.StatusBadRequest, "no items to create")
	}

	// Limit bulk creation to 50 items at once
	if len(req.Items) > 50 {
		return Error(c, fiber.StatusBadRequest, "maximum 50 items per request")
	}

	var created []models.Item
	var errors []string

	for i, item := range req.Items {
		if item.Name == "" {
			errors = append(errors, fmt.Sprintf("item %d: name is required", i+1))
			continue
		}

		// Default to private if not specified
		isPrivate := true
		if !item.IsPrivate {
			isPrivate = item.IsPrivate
		}

		// Create item request
		createReq := &models.CreateItemRequest{
			Name:        item.Name,
			Brand:       item.Brand,
			Size:        item.Size,
			Unit:        item.Unit,
			Description: item.Description,
			Tags:        item.Tags,
			IsPrivate:   &isPrivate,
		}

		newItem, err := h.db.CreateItem(c.Context(), createReq, &userID)
		if err != nil {
			errors = append(errors, fmt.Sprintf("item %d (%s): %v", i+1, item.Name, err))
			continue
		}

		created = append(created, *newItem)
	}

	return Success(c, models.BulkCreateItemsResponse{
		Created: created,
		Errors:  errors,
	})
}
