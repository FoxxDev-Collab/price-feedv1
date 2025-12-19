package handlers

import (
	"errors"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/foxxcyber/price-feed/internal/database"
	"github.com/foxxcyber/price-feed/internal/models"
)

// getUserID extracts user ID from context using the middleware helper
func getUserID(c *fiber.Ctx) (int, error) {
	userID, ok := c.Locals("user_id").(int)
	if !ok || userID == 0 {
		return 0, errors.New("user not authenticated")
	}
	return userID, nil
}

// ListShoppingLists returns all shopping lists for the current user
func (h *Handler) ListShoppingLists(c *fiber.Ctx) error {
	userID, err := getUserID(c)
	if err != nil {
		return Error(c, fiber.StatusUnauthorized, err.Error())
	}

	params := &models.ListListParams{
		Limit:  c.QueryInt("limit", 50),
		Offset: c.QueryInt("offset", 0),
		UserID: userID,
		Status: models.ListStatus(c.Query("status")), // Optional: "active" or "completed"
	}

	// Validate limits
	if params.Limit < 1 || params.Limit > 100 {
		params.Limit = 50
	}
	if params.Offset < 0 {
		params.Offset = 0
	}

	lists, total, err := h.db.ListShoppingLists(c.Context(), params)
	if err != nil {
		return Error(c, fiber.StatusInternalServerError, "failed to list shopping lists")
	}

	return SuccessWithMeta(c, lists, total, params.Limit, params.Offset)
}

// GetShoppingList returns a single shopping list with items
func (h *Handler) GetShoppingList(c *fiber.Ctx) error {
	userID, err := getUserID(c)
	if err != nil {
		return Error(c, fiber.StatusUnauthorized, err.Error())
	}

	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return Error(c, fiber.StatusBadRequest, "invalid list id")
	}

	list, err := h.db.GetShoppingListByID(c.Context(), id, userID)
	if err != nil {
		if errors.Is(err, database.ErrListNotFound) {
			return Error(c, fiber.StatusNotFound, "shopping list not found")
		}
		if errors.Is(err, database.ErrNotListOwner) {
			return Error(c, fiber.StatusForbidden, "you do not own this list")
		}
		return Error(c, fiber.StatusInternalServerError, "failed to get shopping list")
	}

	return Success(c, list)
}

// CreateShoppingList creates a new shopping list
func (h *Handler) CreateShoppingList(c *fiber.Ctx) error {
	userID, err := getUserID(c)
	if err != nil {
		return Error(c, fiber.StatusUnauthorized, err.Error())
	}

	var req models.CreateListRequest
	if err := c.BodyParser(&req); err != nil {
		return Error(c, fiber.StatusBadRequest, "invalid request body")
	}

	// Validate required fields
	if req.Name == "" {
		return Error(c, fiber.StatusBadRequest, "name is required")
	}

	list, err := h.db.CreateShoppingList(c.Context(), &req, userID)
	if err != nil {
		return Error(c, fiber.StatusInternalServerError, "failed to create shopping list")
	}

	return c.Status(fiber.StatusCreated).JSON(APIResponse{
		Success: true,
		Data:    list,
	})
}

// UpdateShoppingList updates a shopping list
func (h *Handler) UpdateShoppingList(c *fiber.Ctx) error {
	userID, err := getUserID(c)
	if err != nil {
		return Error(c, fiber.StatusUnauthorized, err.Error())
	}

	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return Error(c, fiber.StatusBadRequest, "invalid list id")
	}

	var req models.UpdateListRequest
	if err := c.BodyParser(&req); err != nil {
		return Error(c, fiber.StatusBadRequest, "invalid request body")
	}

	list, err := h.db.UpdateShoppingList(c.Context(), id, userID, &req)
	if err != nil {
		if errors.Is(err, database.ErrListNotFound) {
			return Error(c, fiber.StatusNotFound, "shopping list not found")
		}
		return Error(c, fiber.StatusInternalServerError, "failed to update shopping list")
	}

	return Success(c, list)
}

// DeleteShoppingList deletes a shopping list
func (h *Handler) DeleteShoppingList(c *fiber.Ctx) error {
	userID, err := getUserID(c)
	if err != nil {
		return Error(c, fiber.StatusUnauthorized, err.Error())
	}

	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return Error(c, fiber.StatusBadRequest, "invalid list id")
	}

	if err := h.db.DeleteShoppingList(c.Context(), id, userID); err != nil {
		if errors.Is(err, database.ErrListNotFound) {
			return Error(c, fiber.StatusNotFound, "shopping list not found")
		}
		return Error(c, fiber.StatusInternalServerError, "failed to delete shopping list")
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "shopping list deleted successfully",
	})
}

// AddItemToList adds an item to a shopping list
func (h *Handler) AddItemToList(c *fiber.Ctx) error {
	userID, err := getUserID(c)
	if err != nil {
		return Error(c, fiber.StatusUnauthorized, err.Error())
	}

	listID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return Error(c, fiber.StatusBadRequest, "invalid list id")
	}

	var req models.AddListItemRequest
	if err := c.BodyParser(&req); err != nil {
		return Error(c, fiber.StatusBadRequest, "invalid request body")
	}

	// Validate required fields
	if req.ItemID == 0 {
		return Error(c, fiber.StatusBadRequest, "item_id is required")
	}
	if req.Quantity < 1 {
		req.Quantity = 1
	}

	item, err := h.db.AddItemToList(c.Context(), listID, userID, &req)
	if err != nil {
		if errors.Is(err, database.ErrListNotFound) {
			return Error(c, fiber.StatusNotFound, "shopping list not found")
		}
		if errors.Is(err, database.ErrNotListOwner) {
			return Error(c, fiber.StatusForbidden, "you do not own this list")
		}
		return Error(c, fiber.StatusInternalServerError, "failed to add item to list")
	}

	return c.Status(fiber.StatusCreated).JSON(APIResponse{
		Success: true,
		Data:    item,
	})
}

// UpdateListItem updates the quantity of an item in a list
func (h *Handler) UpdateListItem(c *fiber.Ctx) error {
	userID, err := getUserID(c)
	if err != nil {
		return Error(c, fiber.StatusUnauthorized, err.Error())
	}

	listID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return Error(c, fiber.StatusBadRequest, "invalid list id")
	}

	itemID, err := strconv.Atoi(c.Params("item_id"))
	if err != nil {
		return Error(c, fiber.StatusBadRequest, "invalid item id")
	}

	var req models.UpdateListItemRequest
	if err := c.BodyParser(&req); err != nil {
		return Error(c, fiber.StatusBadRequest, "invalid request body")
	}

	if req.Quantity < 1 {
		return Error(c, fiber.StatusBadRequest, "quantity must be at least 1")
	}

	item, err := h.db.UpdateListItem(c.Context(), listID, itemID, userID, &req)
	if err != nil {
		if errors.Is(err, database.ErrListNotFound) {
			return Error(c, fiber.StatusNotFound, "shopping list not found")
		}
		if errors.Is(err, database.ErrListItemNotFound) {
			return Error(c, fiber.StatusNotFound, "item not found in list")
		}
		if errors.Is(err, database.ErrNotListOwner) {
			return Error(c, fiber.StatusForbidden, "you do not own this list")
		}
		return Error(c, fiber.StatusInternalServerError, "failed to update list item")
	}

	return Success(c, item)
}

// RemoveItemFromList removes an item from a shopping list
func (h *Handler) RemoveItemFromList(c *fiber.Ctx) error {
	userID, err := getUserID(c)
	if err != nil {
		return Error(c, fiber.StatusUnauthorized, err.Error())
	}

	listID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return Error(c, fiber.StatusBadRequest, "invalid list id")
	}

	itemID, err := strconv.Atoi(c.Params("item_id"))
	if err != nil {
		return Error(c, fiber.StatusBadRequest, "invalid item id")
	}

	if err := h.db.RemoveItemFromList(c.Context(), listID, itemID, userID); err != nil {
		if errors.Is(err, database.ErrListNotFound) {
			return Error(c, fiber.StatusNotFound, "shopping list not found")
		}
		if errors.Is(err, database.ErrListItemNotFound) {
			return Error(c, fiber.StatusNotFound, "item not found in list")
		}
		if errors.Is(err, database.ErrNotListOwner) {
			return Error(c, fiber.StatusForbidden, "you do not own this list")
		}
		return Error(c, fiber.StatusInternalServerError, "failed to remove item from list")
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "item removed from list successfully",
	})
}

// BuildShoppingPlan generates an optimized shopping plan for a list
func (h *Handler) BuildShoppingPlan(c *fiber.Ctx) error {
	userID, err := getUserID(c)
	if err != nil {
		return Error(c, fiber.StatusUnauthorized, err.Error())
	}

	listID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return Error(c, fiber.StatusBadRequest, "invalid list id")
	}

	// Get user's region if available
	var regionID *int
	if user, err := h.db.GetUserByID(c.Context(), userID); err == nil && user.RegionID != nil {
		regionID = user.RegionID
	}

	plan, err := h.db.BuildShoppingPlan(c.Context(), listID, userID, regionID)
	if err != nil {
		if errors.Is(err, database.ErrListNotFound) {
			return Error(c, fiber.StatusNotFound, "shopping list not found")
		}
		if errors.Is(err, database.ErrNotListOwner) {
			return Error(c, fiber.StatusForbidden, "you do not own this list")
		}
		if err.Error() == "shopping list is empty" {
			return Error(c, fiber.StatusBadRequest, "shopping list is empty")
		}
		return Error(c, fiber.StatusInternalServerError, "failed to build shopping plan")
	}

	return Success(c, plan)
}

// GetPriceComparison returns a price comparison grid
func (h *Handler) GetPriceComparison(c *fiber.Ctx) error {
	userID, err := getUserID(c)
	if err != nil {
		return Error(c, fiber.StatusUnauthorized, err.Error())
	}

	// Parse store IDs (required)
	storeIDsParam := c.Query("store_ids")
	if storeIDsParam == "" {
		return Error(c, fiber.StatusBadRequest, "store_ids is required")
	}

	var storeIDs []int
	for _, idStr := range strings.Split(storeIDsParam, ",") {
		id, err := strconv.Atoi(strings.TrimSpace(idStr))
		if err != nil {
			return Error(c, fiber.StatusBadRequest, "invalid store_ids format")
		}
		storeIDs = append(storeIDs, id)
	}

	if len(storeIDs) < 1 || len(storeIDs) > 5 {
		return Error(c, fiber.StatusBadRequest, "select 1-5 stores to compare")
	}

	// Parse item IDs (optional)
	var itemIDs []int
	if itemIDsParam := c.Query("item_ids"); itemIDsParam != "" {
		for _, idStr := range strings.Split(itemIDsParam, ",") {
			id, err := strconv.Atoi(strings.TrimSpace(idStr))
			if err != nil {
				return Error(c, fiber.StatusBadRequest, "invalid item_ids format")
			}
			itemIDs = append(itemIDs, id)
		}
	}

	// Get user's region if available
	var regionID *int
	if user, err := h.db.GetUserByID(c.Context(), userID); err == nil && user.RegionID != nil {
		regionID = user.RegionID
	}

	params := &models.CompareParams{
		StoreIDs: storeIDs,
		ItemIDs:  itemIDs,
		RegionID: regionID,
		UserID:   &userID,
	}

	comparison, err := h.db.GetPriceComparison(c.Context(), params)
	if err != nil {
		return Error(c, fiber.StatusInternalServerError, "failed to get price comparison")
	}

	return Success(c, comparison)
}

// CompleteShoppingList marks a shopping list as completed with optional price confirmations
func (h *Handler) CompleteShoppingList(c *fiber.Ctx) error {
	userID, err := getUserID(c)
	if err != nil {
		return Error(c, fiber.StatusUnauthorized, err.Error())
	}

	listID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return Error(c, fiber.StatusBadRequest, "invalid list id")
	}

	var req models.CompleteListRequest
	if err := c.BodyParser(&req); err != nil {
		// Body is optional, so just use empty request
		req = models.CompleteListRequest{}
	}

	list, err := h.db.CompleteShoppingList(c.Context(), listID, userID, &req)
	if err != nil {
		if errors.Is(err, database.ErrListNotFound) {
			return Error(c, fiber.StatusNotFound, "shopping list not found")
		}
		if errors.Is(err, database.ErrNotListOwner) {
			return Error(c, fiber.StatusForbidden, "you do not own this list")
		}
		if err.Error() == "list is already completed" {
			return Error(c, fiber.StatusBadRequest, "list is already completed")
		}
		return Error(c, fiber.StatusInternalServerError, "failed to complete shopping list")
	}

	return Success(c, list)
}

// ReopenShoppingList marks a completed list as active again
func (h *Handler) ReopenShoppingList(c *fiber.Ctx) error {
	userID, err := getUserID(c)
	if err != nil {
		return Error(c, fiber.StatusUnauthorized, err.Error())
	}

	listID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return Error(c, fiber.StatusBadRequest, "invalid list id")
	}

	list, err := h.db.ReopenShoppingList(c.Context(), listID, userID)
	if err != nil {
		if errors.Is(err, database.ErrListNotFound) {
			return Error(c, fiber.StatusNotFound, "shopping list not found")
		}
		if errors.Is(err, database.ErrNotListOwner) {
			return Error(c, fiber.StatusForbidden, "you do not own this list")
		}
		return Error(c, fiber.StatusInternalServerError, "failed to reopen shopping list")
	}

	return Success(c, list)
}
