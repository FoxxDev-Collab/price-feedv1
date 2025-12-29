package handlers

import (
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v2"

	"github.com/foxxcyber/price-feed/internal/database"
	"github.com/foxxcyber/price-feed/internal/models"
)

// ListInventoryItems returns all inventory items for the current user
func (h *Handler) ListInventoryItems(c *fiber.Ctx) error {
	userID, err := getUserID(c)
	if err != nil {
		return Error(c, fiber.StatusUnauthorized, err.Error())
	}

	// Parse boolean filters
	var lowStock, expired, expiringSoon *bool
	if ls := c.Query("low_stock"); ls != "" {
		v := ls == "true"
		lowStock = &v
	}
	if exp := c.Query("expired"); exp != "" {
		v := exp == "true"
		expired = &v
	}
	if es := c.Query("expiring_soon"); es != "" {
		v := es == "true"
		expiringSoon = &v
	}

	params := &models.InventoryListParams{
		Limit:        c.QueryInt("limit", 50),
		Offset:       c.QueryInt("offset", 0),
		UserID:       userID,
		Location:     c.Query("location"),
		Search:       c.Query("search"),
		LowStock:     lowStock,
		Expired:      expired,
		ExpiringSoon: expiringSoon,
		SortBy:       c.Query("sort_by", "updated"),
		SortOrder:    c.Query("sort_order", "desc"),
	}

	// Validate limits
	if params.Limit < 1 || params.Limit > 100 {
		params.Limit = 50
	}
	if params.Offset < 0 {
		params.Offset = 0
	}

	items, total, err := h.db.ListInventoryItems(c.Context(), params)
	if err != nil {
		return Error(c, fiber.StatusInternalServerError, "failed to list inventory items")
	}

	return SuccessWithMeta(c, items, total, params.Limit, params.Offset)
}

// GetInventoryItem returns a single inventory item
func (h *Handler) GetInventoryItem(c *fiber.Ctx) error {
	userID, err := getUserID(c)
	if err != nil {
		return Error(c, fiber.StatusUnauthorized, err.Error())
	}

	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return Error(c, fiber.StatusBadRequest, "invalid inventory item id")
	}

	item, err := h.db.GetInventoryItemByID(c.Context(), id, userID)
	if err != nil {
		if errors.Is(err, database.ErrInventoryItemNotFound) {
			return Error(c, fiber.StatusNotFound, "inventory item not found")
		}
		if errors.Is(err, database.ErrNotInventoryOwner) {
			return Error(c, fiber.StatusForbidden, "you do not own this inventory item")
		}
		return Error(c, fiber.StatusInternalServerError, "failed to get inventory item")
	}

	return Success(c, item)
}

// CreateInventoryItem creates a new inventory item
func (h *Handler) CreateInventoryItem(c *fiber.Ctx) error {
	userID, err := getUserID(c)
	if err != nil {
		return Error(c, fiber.StatusUnauthorized, err.Error())
	}

	var req models.CreateInventoryItemRequest
	if err := c.BodyParser(&req); err != nil {
		return Error(c, fiber.StatusBadRequest, "invalid request body")
	}

	// Validate: must have either item_id OR custom_name
	if req.ItemID == nil && (req.CustomName == nil || *req.CustomName == "") {
		return Error(c, fiber.StatusBadRequest, "either item_id or custom_name is required")
	}

	// Validate quantity
	if req.Quantity < 0 {
		return Error(c, fiber.StatusBadRequest, "quantity cannot be negative")
	}

	item, err := h.db.CreateInventoryItem(c.Context(), &req, userID)
	if err != nil {
		return Error(c, fiber.StatusInternalServerError, "failed to create inventory item")
	}

	return c.Status(fiber.StatusCreated).JSON(APIResponse{
		Success: true,
		Data:    item,
	})
}

// UpdateInventoryItem updates an inventory item
func (h *Handler) UpdateInventoryItem(c *fiber.Ctx) error {
	userID, err := getUserID(c)
	if err != nil {
		return Error(c, fiber.StatusUnauthorized, err.Error())
	}

	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return Error(c, fiber.StatusBadRequest, "invalid inventory item id")
	}

	var req models.UpdateInventoryItemRequest
	if err := c.BodyParser(&req); err != nil {
		return Error(c, fiber.StatusBadRequest, "invalid request body")
	}

	// Validate quantity if provided
	if req.Quantity != nil && *req.Quantity < 0 {
		return Error(c, fiber.StatusBadRequest, "quantity cannot be negative")
	}

	item, err := h.db.UpdateInventoryItem(c.Context(), id, userID, &req)
	if err != nil {
		if errors.Is(err, database.ErrInventoryItemNotFound) {
			return Error(c, fiber.StatusNotFound, "inventory item not found")
		}
		if errors.Is(err, database.ErrNotInventoryOwner) {
			return Error(c, fiber.StatusForbidden, "you do not own this inventory item")
		}
		return Error(c, fiber.StatusInternalServerError, "failed to update inventory item")
	}

	return Success(c, item)
}

// DeleteInventoryItem deletes an inventory item
func (h *Handler) DeleteInventoryItem(c *fiber.Ctx) error {
	userID, err := getUserID(c)
	if err != nil {
		return Error(c, fiber.StatusUnauthorized, err.Error())
	}

	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return Error(c, fiber.StatusBadRequest, "invalid inventory item id")
	}

	if err := h.db.DeleteInventoryItem(c.Context(), id, userID); err != nil {
		if errors.Is(err, database.ErrInventoryItemNotFound) {
			return Error(c, fiber.StatusNotFound, "inventory item not found")
		}
		return Error(c, fiber.StatusInternalServerError, "failed to delete inventory item")
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "inventory item deleted successfully",
	})
}

// AdjustInventoryQuantity adjusts the quantity of an inventory item
func (h *Handler) AdjustInventoryQuantity(c *fiber.Ctx) error {
	userID, err := getUserID(c)
	if err != nil {
		return Error(c, fiber.StatusUnauthorized, err.Error())
	}

	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return Error(c, fiber.StatusBadRequest, "invalid inventory item id")
	}

	var req models.AdjustInventoryQuantityRequest
	if err := c.BodyParser(&req); err != nil {
		return Error(c, fiber.StatusBadRequest, "invalid request body")
	}

	item, err := h.db.AdjustInventoryQuantity(c.Context(), id, userID, req.Adjustment)
	if err != nil {
		if errors.Is(err, database.ErrInventoryItemNotFound) {
			return Error(c, fiber.StatusNotFound, "inventory item not found")
		}
		if errors.Is(err, database.ErrNotInventoryOwner) {
			return Error(c, fiber.StatusForbidden, "you do not own this inventory item")
		}
		return Error(c, fiber.StatusInternalServerError, "failed to adjust inventory quantity")
	}

	return Success(c, item)
}

// GetInventorySummary returns aggregate stats for user's inventory
func (h *Handler) GetInventorySummary(c *fiber.Ctx) error {
	userID, err := getUserID(c)
	if err != nil {
		return Error(c, fiber.StatusUnauthorized, err.Error())
	}

	summary, err := h.db.GetInventorySummary(c.Context(), userID)
	if err != nil {
		return Error(c, fiber.StatusInternalServerError, "failed to get inventory summary")
	}

	return Success(c, summary)
}

// GetLowStockItems returns items below their threshold
func (h *Handler) GetLowStockItems(c *fiber.Ctx) error {
	userID, err := getUserID(c)
	if err != nil {
		return Error(c, fiber.StatusUnauthorized, err.Error())
	}

	items, err := h.db.GetLowStockItems(c.Context(), userID)
	if err != nil {
		return Error(c, fiber.StatusInternalServerError, "failed to get low stock items")
	}

	return Success(c, items)
}

// GetExpiringItems returns items expiring within specified days
func (h *Handler) GetExpiringItems(c *fiber.Ctx) error {
	userID, err := getUserID(c)
	if err != nil {
		return Error(c, fiber.StatusUnauthorized, err.Error())
	}

	days := c.QueryInt("days", 7)
	if days < 1 {
		days = 7
	}
	if days > 365 {
		days = 365
	}

	items, err := h.db.GetExpiringItems(c.Context(), userID, days)
	if err != nil {
		return Error(c, fiber.StatusInternalServerError, "failed to get expiring items")
	}

	return Success(c, items)
}

// GetInventoryLocations returns unique locations for a user's inventory
func (h *Handler) GetInventoryLocations(c *fiber.Ctx) error {
	userID, err := getUserID(c)
	if err != nil {
		return Error(c, fiber.StatusUnauthorized, err.Error())
	}

	locations, err := h.db.GetInventoryLocations(c.Context(), userID)
	if err != nil {
		return Error(c, fiber.StatusInternalServerError, "failed to get inventory locations")
	}

	return Success(c, locations)
}

// AddInventoryToShoppingList adds an inventory item to a shopping list
func (h *Handler) AddInventoryToShoppingList(c *fiber.Ctx) error {
	userID, err := getUserID(c)
	if err != nil {
		return Error(c, fiber.StatusUnauthorized, err.Error())
	}

	inventoryID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return Error(c, fiber.StatusBadRequest, "invalid inventory item id")
	}

	var req models.AddInventoryToListRequest
	if err := c.BodyParser(&req); err != nil {
		return Error(c, fiber.StatusBadRequest, "invalid request body")
	}

	// Validate required fields
	if req.ListID == 0 {
		return Error(c, fiber.StatusBadRequest, "list_id is required")
	}
	if req.Quantity < 1 {
		req.Quantity = 1
	}

	err = h.db.AddInventoryItemToShoppingList(c.Context(), inventoryID, userID, req.ListID, req.Quantity)
	if err != nil {
		if errors.Is(err, database.ErrInventoryItemNotFound) {
			return Error(c, fiber.StatusNotFound, "inventory item not found")
		}
		if errors.Is(err, database.ErrNotInventoryOwner) {
			return Error(c, fiber.StatusForbidden, "you do not own this inventory item")
		}
		if errors.Is(err, database.ErrListNotFound) {
			return Error(c, fiber.StatusNotFound, "shopping list not found")
		}
		if errors.Is(err, database.ErrNotListOwner) {
			return Error(c, fiber.StatusForbidden, "you do not own this shopping list")
		}
		// Check for custom item error
		if err.Error() == "cannot add custom inventory items to shopping list (no catalog item linked)" {
			return Error(c, fiber.StatusBadRequest, err.Error())
		}
		return Error(c, fiber.StatusInternalServerError, "failed to add item to shopping list")
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "item added to shopping list successfully",
	})
}

// GetActiveShoppingListsForInventory returns user's active shopping lists (for quick-add dropdown)
func (h *Handler) GetActiveShoppingListsForInventory(c *fiber.Ctx) error {
	userID, err := getUserID(c)
	if err != nil {
		return Error(c, fiber.StatusUnauthorized, err.Error())
	}

	lists, err := h.db.GetActiveShoppingLists(c.Context(), userID)
	if err != nil {
		return Error(c, fiber.StatusInternalServerError, "failed to get active shopping lists")
	}

	return Success(c, lists)
}
