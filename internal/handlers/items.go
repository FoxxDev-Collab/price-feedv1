package handlers

import (
	"errors"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/foxxcyber/price-feed/internal/database"
	"github.com/foxxcyber/price-feed/internal/middleware"
	"github.com/foxxcyber/price-feed/internal/models"
)

// ListItems returns a paginated list of items
func (h *Handler) ListItems(c *fiber.Ctx) error {
	params := &models.ItemListParams{
		Limit:  c.QueryInt("limit", 50),
		Offset: c.QueryInt("offset", 0),
		Search: c.Query("search"),
		Tag:    c.Query("tag"),
	}

	// Validate limits
	if params.Limit < 1 || params.Limit > 100 {
		params.Limit = 50
	}
	if params.Offset < 0 {
		params.Offset = 0
	}

	items, total, err := h.db.ListItems(c.Context(), params)
	if err != nil {
		return Error(c, fiber.StatusInternalServerError, "failed to list items")
	}

	return SuccessWithMeta(c, items, total, params.Limit, params.Offset)
}

// GetItem returns a single item by ID
func (h *Handler) GetItem(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return Error(c, fiber.StatusBadRequest, "invalid item id")
	}

	item, err := h.db.GetItemByID(c.Context(), id)
	if err != nil {
		if errors.Is(err, database.ErrItemNotFound) {
			return Error(c, fiber.StatusNotFound, "item not found")
		}
		return Error(c, fiber.StatusInternalServerError, "failed to get item")
	}

	return Success(c, item)
}

// CreateItem creates a new item (admin only)
func (h *Handler) CreateItem(c *fiber.Ctx) error {
	var req models.CreateItemRequest
	if err := c.BodyParser(&req); err != nil {
		return Error(c, fiber.StatusBadRequest, "invalid request body")
	}

	// Validate required fields
	if req.Name == "" {
		return Error(c, fiber.StatusBadRequest, "name is required")
	}

	// Parse tags from comma-separated string if needed
	if len(req.Tags) == 1 && strings.Contains(req.Tags[0], ",") {
		parts := strings.Split(req.Tags[0], ",")
		req.Tags = make([]string, 0, len(parts))
		for _, p := range parts {
			if t := strings.TrimSpace(p); t != "" {
				req.Tags = append(req.Tags, t)
			}
		}
	}

	// Get user ID from context if available
	var createdBy *int
	if user := c.Locals("user"); user != nil {
		if u, ok := user.(*models.User); ok {
			createdBy = &u.ID
		}
	}

	item, err := h.db.CreateItem(c.Context(), &req, createdBy)
	if err != nil {
		return Error(c, fiber.StatusInternalServerError, "failed to create item")
	}

	return c.Status(fiber.StatusCreated).JSON(APIResponse{
		Success: true,
		Data:    item,
	})
}

// UpdateItem updates an existing item (admin only)
func (h *Handler) UpdateItem(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return Error(c, fiber.StatusBadRequest, "invalid item id")
	}

	var req models.UpdateItemRequest
	if err := c.BodyParser(&req); err != nil {
		return Error(c, fiber.StatusBadRequest, "invalid request body")
	}

	// Parse tags from comma-separated string if needed
	if req.Tags != nil && len(req.Tags) == 1 && strings.Contains(req.Tags[0], ",") {
		parts := strings.Split(req.Tags[0], ",")
		req.Tags = make([]string, 0, len(parts))
		for _, p := range parts {
			if t := strings.TrimSpace(p); t != "" {
				req.Tags = append(req.Tags, t)
			}
		}
	}

	item, err := h.db.UpdateItem(c.Context(), id, &req)
	if err != nil {
		if errors.Is(err, database.ErrItemNotFound) {
			return Error(c, fiber.StatusNotFound, "item not found")
		}
		return Error(c, fiber.StatusInternalServerError, "failed to update item")
	}

	return Success(c, item)
}

// DeleteItem deletes an item (admin only)
func (h *Handler) DeleteItem(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return Error(c, fiber.StatusBadRequest, "invalid item id")
	}

	if err := h.db.DeleteItem(c.Context(), id); err != nil {
		if errors.Is(err, database.ErrItemNotFound) {
			return Error(c, fiber.StatusNotFound, "item not found")
		}
		return Error(c, fiber.StatusInternalServerError, "failed to delete item")
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "item deleted successfully",
	})
}

// GetItemStats returns aggregate item statistics
func (h *Handler) GetItemStats(c *fiber.Ctx) error {
	stats, err := h.db.GetItemStats(c.Context())
	if err != nil {
		return Error(c, fiber.StatusInternalServerError, "failed to get item stats")
	}

	return Success(c, stats)
}

// SearchItems performs a search on items
func (h *Handler) SearchItems(c *fiber.Ctx) error {
	query := c.Query("q")
	if query == "" {
		return Error(c, fiber.StatusBadRequest, "search query is required")
	}

	limit := c.QueryInt("limit", 20)
	if limit < 1 || limit > 100 {
		limit = 20
	}

	items, err := h.db.SearchItems(c.Context(), query, limit)
	if err != nil {
		return Error(c, fiber.StatusInternalServerError, "failed to search items")
	}

	return Success(c, items)
}

// ListTags returns all tags
func (h *Handler) ListTags(c *fiber.Ctx) error {
	tags, err := h.db.ListTags(c.Context())
	if err != nil {
		return Error(c, fiber.StatusInternalServerError, "failed to list tags")
	}

	return Success(c, tags)
}

// UserCreateItem allows authenticated users to create items
func (h *Handler) UserCreateItem(c *fiber.Ctx) error {
	var req models.CreateItemRequest
	if err := c.BodyParser(&req); err != nil {
		return Error(c, fiber.StatusBadRequest, "invalid request body")
	}

	// Validate required fields
	if req.Name == "" {
		return Error(c, fiber.StatusBadRequest, "name is required")
	}

	// Parse tags from comma-separated string if needed
	if len(req.Tags) == 1 && strings.Contains(req.Tags[0], ",") {
		parts := strings.Split(req.Tags[0], ",")
		req.Tags = make([]string, 0, len(parts))
		for _, p := range parts {
			if t := strings.TrimSpace(p); t != "" {
				req.Tags = append(req.Tags, t)
			}
		}
	}

	// Get user ID from context
	userID := middleware.GetUserID(c)
	if userID == 0 {
		return Error(c, fiber.StatusUnauthorized, "unauthorized")
	}

	item, err := h.db.CreateItem(c.Context(), &req, &userID)
	if err != nil {
		return Error(c, fiber.StatusInternalServerError, "failed to create item")
	}

	return c.Status(fiber.StatusCreated).JSON(APIResponse{
		Success: true,
		Data:    item,
	})
}

// UserUpdateItem allows users to update their own items
func (h *Handler) UserUpdateItem(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return Error(c, fiber.StatusBadRequest, "invalid item id")
	}

	// Get user ID from context
	userID := middleware.GetUserID(c)
	if userID == 0 {
		return Error(c, fiber.StatusUnauthorized, "unauthorized")
	}

	// Get the item to verify ownership
	item, err := h.db.GetItemByID(c.Context(), id)
	if err != nil {
		if errors.Is(err, database.ErrItemNotFound) {
			return Error(c, fiber.StatusNotFound, "item not found")
		}
		return Error(c, fiber.StatusInternalServerError, "failed to get item")
	}

	// Verify user owns this item
	if item.CreatedBy == nil || *item.CreatedBy != userID {
		return Error(c, fiber.StatusForbidden, "cannot update others' items")
	}

	var req models.UpdateItemRequest
	if err := c.BodyParser(&req); err != nil {
		return Error(c, fiber.StatusBadRequest, "invalid request body")
	}

	// Parse tags from comma-separated string if needed
	if req.Tags != nil && len(req.Tags) == 1 && strings.Contains(req.Tags[0], ",") {
		parts := strings.Split(req.Tags[0], ",")
		req.Tags = make([]string, 0, len(parts))
		for _, p := range parts {
			if t := strings.TrimSpace(p); t != "" {
				req.Tags = append(req.Tags, t)
			}
		}
	}

	updatedItem, err := h.db.UpdateItem(c.Context(), id, &req)
	if err != nil {
		if errors.Is(err, database.ErrItemNotFound) {
			return Error(c, fiber.StatusNotFound, "item not found")
		}
		return Error(c, fiber.StatusInternalServerError, "failed to update item")
	}

	return Success(c, updatedItem)
}

// UserDeleteItem allows users to delete their own items
func (h *Handler) UserDeleteItem(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return Error(c, fiber.StatusBadRequest, "invalid item id")
	}

	// Get user ID from context
	userID := middleware.GetUserID(c)
	if userID == 0 {
		return Error(c, fiber.StatusUnauthorized, "unauthorized")
	}

	// Get the item to verify ownership
	item, err := h.db.GetItemByID(c.Context(), id)
	if err != nil {
		if errors.Is(err, database.ErrItemNotFound) {
			return Error(c, fiber.StatusNotFound, "item not found")
		}
		return Error(c, fiber.StatusInternalServerError, "failed to get item")
	}

	// Verify user owns this item
	if item.CreatedBy == nil || *item.CreatedBy != userID {
		return Error(c, fiber.StatusForbidden, "cannot delete others' items")
	}

	if err := h.db.DeleteItem(c.Context(), id); err != nil {
		if errors.Is(err, database.ErrItemNotFound) {
			return Error(c, fiber.StatusNotFound, "item not found")
		}
		return Error(c, fiber.StatusInternalServerError, "failed to delete item")
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "item deleted successfully",
	})
}
