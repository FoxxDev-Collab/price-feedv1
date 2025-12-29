package handlers

import (
	"errors"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/foxxcyber/price-feed/internal/database"
	"github.com/foxxcyber/price-feed/internal/middleware"
	"github.com/foxxcyber/price-feed/internal/models"
)

// ListPrices returns a paginated list of prices
func (h *Handler) ListPrices(c *fiber.Ctx) error {
	params := &models.PriceListParams{
		Limit:  c.QueryInt("limit", 50),
		Offset: c.QueryInt("offset", 0),
		Search: c.Query("search"),
	}

	if storeID := c.Query("store_id"); storeID != "" {
		if id, err := strconv.Atoi(storeID); err == nil {
			params.StoreID = &id
		}
	}

	if itemID := c.Query("item_id"); itemID != "" {
		if id, err := strconv.Atoi(itemID); err == nil {
			params.ItemID = &id
		}
	}

	if regionID := c.Query("region_id"); regionID != "" {
		if id, err := strconv.Atoi(regionID); err == nil {
			params.RegionID = &id
		}
	}

	if verified := c.Query("verified"); verified != "" {
		v := verified == "true"
		params.Verified = &v
	}

	// Date filter
	if dateFilter := c.Query("date"); dateFilter != "" {
		now := time.Now()
		switch dateFilter {
		case "today":
			start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
			params.DateFrom = &start
		case "week":
			start := now.AddDate(0, 0, -7)
			params.DateFrom = &start
		case "month":
			start := now.AddDate(0, -1, 0)
			params.DateFrom = &start
		}
	}

	// Validate limits
	if params.Limit < 1 || params.Limit > 100 {
		params.Limit = 50
	}
	if params.Offset < 0 {
		params.Offset = 0
	}

	prices, total, err := h.db.ListPrices(c.Context(), params)
	if err != nil {
		return Error(c, fiber.StatusInternalServerError, "failed to list prices")
	}

	return SuccessWithMeta(c, prices, total, params.Limit, params.Offset)
}

// GetPrice returns a single price by ID
func (h *Handler) GetPrice(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return Error(c, fiber.StatusBadRequest, "invalid price id")
	}

	price, err := h.db.GetPriceByID(c.Context(), id)
	if err != nil {
		if errors.Is(err, database.ErrPriceNotFound) {
			return Error(c, fiber.StatusNotFound, "price not found")
		}
		return Error(c, fiber.StatusInternalServerError, "failed to get price")
	}

	return Success(c, price)
}

// CreatePrice creates a new price
func (h *Handler) CreatePrice(c *fiber.Ctx) error {
	var req models.CreatePriceRequest
	if err := c.BodyParser(&req); err != nil {
		return Error(c, fiber.StatusBadRequest, "invalid request body")
	}

	// Validate required fields
	if req.StoreID == 0 {
		return Error(c, fiber.StatusBadRequest, "store_id is required")
	}
	if req.ItemID == 0 {
		return Error(c, fiber.StatusBadRequest, "item_id is required")
	}
	if req.Price <= 0 {
		return Error(c, fiber.StatusBadRequest, "price must be greater than 0")
	}

	// Get user ID from context if available
	var userID *int
	if user := c.Locals("user"); user != nil {
		if u, ok := user.(*models.User); ok {
			userID = &u.ID
		}
	}

	// Check if there's an existing price for this item/store to get previous price
	var previousPrice *float64
	existingPrice, err := h.db.GetPriceForItemStore(c.Context(), req.ItemID, req.StoreID)
	if err == nil {
		previousPrice = &existingPrice.Price
	}

	price, err := h.db.CreatePrice(c.Context(), &req, userID)
	if err != nil {
		return Error(c, fiber.StatusInternalServerError, "failed to create price")
	}

	// Record price history
	if err := h.db.RecordPriceHistory(c.Context(), req.StoreID, req.ItemID, req.Price, previousPrice, userID); err != nil {
		// Log but don't fail the request
		// The price was created successfully
	}

	return c.Status(fiber.StatusCreated).JSON(APIResponse{
		Success: true,
		Data:    price,
	})
}

// UpdatePrice updates an existing price (admin only)
func (h *Handler) UpdatePrice(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return Error(c, fiber.StatusBadRequest, "invalid price id")
	}

	var req models.UpdatePriceRequest
	if err := c.BodyParser(&req); err != nil {
		return Error(c, fiber.StatusBadRequest, "invalid request body")
	}

	// Validate price if provided
	if req.Price != nil && *req.Price <= 0 {
		return Error(c, fiber.StatusBadRequest, "price must be greater than 0")
	}

	price, err := h.db.UpdatePrice(c.Context(), id, &req)
	if err != nil {
		if errors.Is(err, database.ErrPriceNotFound) {
			return Error(c, fiber.StatusNotFound, "price not found")
		}
		return Error(c, fiber.StatusInternalServerError, "failed to update price")
	}

	return Success(c, price)
}

// UserUpdatePrice allows any authenticated user to update prices
// Prices are community data - anyone can report/update the current price
func (h *Handler) UserUpdatePrice(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return Error(c, fiber.StatusBadRequest, "invalid price id")
	}

	// Get user ID from context
	userID := middleware.GetUserID(c)
	if userID == 0 {
		return Error(c, fiber.StatusUnauthorized, "unauthorized")
	}

	// Get existing price to record history
	existingPrice, err := h.db.GetPriceByID(c.Context(), id)
	if err != nil {
		if errors.Is(err, database.ErrPriceNotFound) {
			return Error(c, fiber.StatusNotFound, "price not found")
		}
		return Error(c, fiber.StatusInternalServerError, "failed to get price")
	}

	var req models.UpdatePriceRequest
	if err := c.BodyParser(&req); err != nil {
		return Error(c, fiber.StatusBadRequest, "invalid request body")
	}

	// Validate price if provided
	if req.Price != nil && *req.Price <= 0 {
		return Error(c, fiber.StatusBadRequest, "price must be greater than 0")
	}

	updatedPrice, err := h.db.UpdatePrice(c.Context(), id, &req)
	if err != nil {
		if errors.Is(err, database.ErrPriceNotFound) {
			return Error(c, fiber.StatusNotFound, "price not found")
		}
		return Error(c, fiber.StatusInternalServerError, "failed to update price")
	}

	// Record price history if price actually changed
	if req.Price != nil && *req.Price != existingPrice.Price {
		previousPrice := existingPrice.Price
		if err := h.db.RecordPriceHistory(c.Context(), existingPrice.StoreID, existingPrice.ItemID, *req.Price, &previousPrice, &userID); err != nil {
			// Log but don't fail the request
		}
	}

	return Success(c, updatedPrice)
}

// UserDeletePrice allows any authenticated user to delete prices
// Prices are community data - users can remove outdated prices
func (h *Handler) UserDeletePrice(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return Error(c, fiber.StatusBadRequest, "invalid price id")
	}

	// Get user ID from context
	userID := middleware.GetUserID(c)
	if userID == 0 {
		return Error(c, fiber.StatusUnauthorized, "unauthorized")
	}

	if err := h.db.DeletePrice(c.Context(), id); err != nil {
		if errors.Is(err, database.ErrPriceNotFound) {
			return Error(c, fiber.StatusNotFound, "price not found")
		}
		return Error(c, fiber.StatusInternalServerError, "failed to delete price")
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "price deleted successfully",
	})
}

// DeletePrice deletes a price (admin only)
func (h *Handler) DeletePrice(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return Error(c, fiber.StatusBadRequest, "invalid price id")
	}

	if err := h.db.DeletePrice(c.Context(), id); err != nil {
		if errors.Is(err, database.ErrPriceNotFound) {
			return Error(c, fiber.StatusNotFound, "price not found")
		}
		return Error(c, fiber.StatusInternalServerError, "failed to delete price")
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "price deleted successfully",
	})
}

// VerifyPrice allows a user to verify a price
func (h *Handler) VerifyPrice(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return Error(c, fiber.StatusBadRequest, "invalid price id")
	}

	var req struct {
		IsAccurate bool `json:"is_accurate"`
	}
	if err := c.BodyParser(&req); err != nil {
		return Error(c, fiber.StatusBadRequest, "invalid request body")
	}

	// Get user ID from context
	user := c.Locals("user")
	if user == nil {
		return Error(c, fiber.StatusUnauthorized, "authentication required")
	}
	u, ok := user.(*models.User)
	if !ok {
		return Error(c, fiber.StatusInternalServerError, "invalid user context")
	}

	if err := h.db.VerifyPrice(c.Context(), id, u.ID, req.IsAccurate); err != nil {
		return Error(c, fiber.StatusInternalServerError, "failed to verify price")
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "price verification recorded",
	})
}

// GetPriceStats returns aggregate price statistics
func (h *Handler) GetPriceStats(c *fiber.Ctx) error {
	stats, err := h.db.GetPriceStats(c.Context())
	if err != nil {
		return Error(c, fiber.StatusInternalServerError, "failed to get price stats")
	}

	return Success(c, stats)
}

// GetPricesByStore returns all prices for a store
func (h *Handler) GetPricesByStore(c *fiber.Ctx) error {
	storeID, err := strconv.Atoi(c.Params("store_id"))
	if err != nil {
		return Error(c, fiber.StatusBadRequest, "invalid store id")
	}

	prices, err := h.db.GetPricesByStore(c.Context(), storeID)
	if err != nil {
		return Error(c, fiber.StatusInternalServerError, "failed to get prices")
	}

	return Success(c, prices)
}

// GetPricesByItem returns all prices for an item
func (h *Handler) GetPricesByItem(c *fiber.Ctx) error {
	itemID, err := strconv.Atoi(c.Params("item_id"))
	if err != nil {
		return Error(c, fiber.StatusBadRequest, "invalid item id")
	}

	prices, err := h.db.GetPricesByItem(c.Context(), itemID)
	if err != nil {
		return Error(c, fiber.StatusInternalServerError, "failed to get prices")
	}

	return Success(c, prices)
}

// GetPriceHistory returns the price history for an item
func (h *Handler) GetPriceHistory(c *fiber.Ctx) error {
	itemID, err := strconv.Atoi(c.Params("item_id"))
	if err != nil {
		return Error(c, fiber.StatusBadRequest, "invalid item id")
	}

	params := &models.PriceHistoryParams{
		ItemID: itemID,
		Limit:  c.QueryInt("limit", 50),
	}

	// Optional store filter
	if storeID := c.Query("store_id"); storeID != "" {
		if id, err := strconv.Atoi(storeID); err == nil {
			params.StoreID = &id
		}
	}

	// Validate limit
	if params.Limit < 1 || params.Limit > 100 {
		params.Limit = 50
	}

	history, err := h.db.GetPriceHistory(c.Context(), params)
	if err != nil {
		if err.Error() == "item not found" {
			return Error(c, fiber.StatusNotFound, "item not found")
		}
		return Error(c, fiber.StatusInternalServerError, "failed to get price history")
	}

	return Success(c, history)
}
