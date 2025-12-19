package handlers

import (
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v2"

	"github.com/foxxcyber/price-feed/internal/database"
	"github.com/foxxcyber/price-feed/internal/middleware"
	"github.com/foxxcyber/price-feed/internal/models"
)

// ListStores returns a paginated list of stores
func (h *Handler) ListStores(c *fiber.Ctx) error {
	params := &models.StoreListParams{
		Limit:  c.QueryInt("limit", 50),
		Offset: c.QueryInt("offset", 0),
		Search: c.Query("search"),
		State:  c.Query("state"),
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

	// Validate limits
	if params.Limit < 1 || params.Limit > 100 {
		params.Limit = 50
	}
	if params.Offset < 0 {
		params.Offset = 0
	}

	stores, total, err := h.db.ListStores(c.Context(), params)
	if err != nil {
		return Error(c, fiber.StatusInternalServerError, "failed to list stores")
	}

	return SuccessWithMeta(c, stores, total, params.Limit, params.Offset)
}

// GetStore returns a single store by ID
func (h *Handler) GetStore(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return Error(c, fiber.StatusBadRequest, "invalid store id")
	}

	store, err := h.db.GetStoreByID(c.Context(), id)
	if err != nil {
		if errors.Is(err, database.ErrStoreNotFound) {
			return Error(c, fiber.StatusNotFound, "store not found")
		}
		return Error(c, fiber.StatusInternalServerError, "failed to get store")
	}

	return Success(c, store)
}

// CreateStore creates a new store (admin only)
func (h *Handler) CreateStore(c *fiber.Ctx) error {
	var req models.CreateStoreRequest
	if err := c.BodyParser(&req); err != nil {
		return Error(c, fiber.StatusBadRequest, "invalid request body")
	}

	// Validate required fields
	if req.Name == "" {
		return Error(c, fiber.StatusBadRequest, "name is required")
	}
	if req.StreetAddress == "" {
		return Error(c, fiber.StatusBadRequest, "street_address is required")
	}
	if req.City == "" {
		return Error(c, fiber.StatusBadRequest, "city is required")
	}
	if req.State == "" {
		return Error(c, fiber.StatusBadRequest, "state is required")
	}
	if len(req.State) != 2 {
		return Error(c, fiber.StatusBadRequest, "state must be a 2-letter code")
	}
	if req.ZipCode == "" {
		return Error(c, fiber.StatusBadRequest, "zip_code is required")
	}

	// Get user ID from context if available
	var createdBy *int
	if user := c.Locals("user"); user != nil {
		if u, ok := user.(*models.User); ok {
			createdBy = &u.ID
		}
	}

	store, err := h.db.CreateStore(c.Context(), &req, createdBy)
	if err != nil {
		return Error(c, fiber.StatusInternalServerError, "failed to create store")
	}

	return c.Status(fiber.StatusCreated).JSON(APIResponse{
		Success: true,
		Data:    store,
	})
}

// UpdateStore updates an existing store (admin only)
func (h *Handler) UpdateStore(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return Error(c, fiber.StatusBadRequest, "invalid store id")
	}

	var req models.UpdateStoreRequest
	if err := c.BodyParser(&req); err != nil {
		return Error(c, fiber.StatusBadRequest, "invalid request body")
	}

	// Validate state if provided
	if req.State != nil && len(*req.State) != 2 {
		return Error(c, fiber.StatusBadRequest, "state must be a 2-letter code")
	}

	store, err := h.db.UpdateStore(c.Context(), id, &req)
	if err != nil {
		if errors.Is(err, database.ErrStoreNotFound) {
			return Error(c, fiber.StatusNotFound, "store not found")
		}
		return Error(c, fiber.StatusInternalServerError, "failed to update store")
	}

	return Success(c, store)
}

// DeleteStore deletes a store (admin only)
func (h *Handler) DeleteStore(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return Error(c, fiber.StatusBadRequest, "invalid store id")
	}

	if err := h.db.DeleteStore(c.Context(), id); err != nil {
		if errors.Is(err, database.ErrStoreNotFound) {
			return Error(c, fiber.StatusNotFound, "store not found")
		}
		return Error(c, fiber.StatusInternalServerError, "failed to delete store")
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "store deleted successfully",
	})
}

// VerifyStore marks a store as verified (admin only)
func (h *Handler) VerifyStore(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return Error(c, fiber.StatusBadRequest, "invalid store id")
	}

	if err := h.db.VerifyStore(c.Context(), id); err != nil {
		if errors.Is(err, database.ErrStoreNotFound) {
			return Error(c, fiber.StatusNotFound, "store not found")
		}
		return Error(c, fiber.StatusInternalServerError, "failed to verify store")
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "store verified successfully",
	})
}

// UserCreateStore allows authenticated users to add stores they discover
func (h *Handler) UserCreateStore(c *fiber.Ctx) error {
	var req models.CreateStoreRequest
	if err := c.BodyParser(&req); err != nil {
		return Error(c, fiber.StatusBadRequest, "invalid request body")
	}

	// Validate required fields
	if req.Name == "" {
		return Error(c, fiber.StatusBadRequest, "name is required")
	}
	if req.StreetAddress == "" {
		return Error(c, fiber.StatusBadRequest, "street_address is required")
	}
	if req.City == "" {
		return Error(c, fiber.StatusBadRequest, "city is required")
	}
	if req.State == "" {
		return Error(c, fiber.StatusBadRequest, "state is required")
	}
	if len(req.State) != 2 {
		return Error(c, fiber.StatusBadRequest, "state must be a 2-letter code")
	}
	if req.ZipCode == "" {
		return Error(c, fiber.StatusBadRequest, "zip_code is required")
	}

	// Get user ID from context (required for user-created stores)
	userID := middleware.GetUserID(c)
	if userID == 0 {
		return Error(c, fiber.StatusUnauthorized, "unauthorized")
	}

	store, err := h.db.CreateStore(c.Context(), &req, &userID)
	if err != nil {
		return Error(c, fiber.StatusInternalServerError, "failed to create store")
	}

	return c.Status(fiber.StatusCreated).JSON(APIResponse{
		Success: true,
		Data:    store,
	})
}

// UserUpdateStore allows users to update their own stores
func (h *Handler) UserUpdateStore(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return Error(c, fiber.StatusBadRequest, "invalid store id")
	}

	// Get user ID from context
	userID := middleware.GetUserID(c)
	if userID == 0 {
		return Error(c, fiber.StatusUnauthorized, "unauthorized")
	}

	// Get the store to verify ownership
	store, err := h.db.GetStoreByID(c.Context(), id)
	if err != nil {
		if errors.Is(err, database.ErrStoreNotFound) {
			return Error(c, fiber.StatusNotFound, "store not found")
		}
		return Error(c, fiber.StatusInternalServerError, "failed to get store")
	}

	// Verify user owns this store
	if store.CreatedBy == nil || *store.CreatedBy != userID {
		return Error(c, fiber.StatusForbidden, "cannot update others' stores")
	}

	var req models.UpdateStoreRequest
	if err := c.BodyParser(&req); err != nil {
		return Error(c, fiber.StatusBadRequest, "invalid request body")
	}

	// Validate state if provided
	if req.State != nil && len(*req.State) != 2 {
		return Error(c, fiber.StatusBadRequest, "state must be a 2-letter code")
	}

	updatedStore, err := h.db.UpdateStore(c.Context(), id, &req)
	if err != nil {
		if errors.Is(err, database.ErrStoreNotFound) {
			return Error(c, fiber.StatusNotFound, "store not found")
		}
		return Error(c, fiber.StatusInternalServerError, "failed to update store")
	}

	return Success(c, updatedStore)
}

// UserDeleteStore allows users to delete their own stores
func (h *Handler) UserDeleteStore(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return Error(c, fiber.StatusBadRequest, "invalid store id")
	}

	// Get user ID from context
	userID := middleware.GetUserID(c)
	if userID == 0 {
		return Error(c, fiber.StatusUnauthorized, "unauthorized")
	}

	// Get the store to verify ownership
	store, err := h.db.GetStoreByID(c.Context(), id)
	if err != nil {
		if errors.Is(err, database.ErrStoreNotFound) {
			return Error(c, fiber.StatusNotFound, "store not found")
		}
		return Error(c, fiber.StatusInternalServerError, "failed to get store")
	}

	// Verify user owns this store
	if store.CreatedBy == nil || *store.CreatedBy != userID {
		return Error(c, fiber.StatusForbidden, "cannot delete others' stores")
	}

	if err := h.db.DeleteStore(c.Context(), id); err != nil {
		if errors.Is(err, database.ErrStoreNotFound) {
			return Error(c, fiber.StatusNotFound, "store not found")
		}
		return Error(c, fiber.StatusInternalServerError, "failed to delete store")
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "store deleted successfully",
	})
}

// GetStoreStats returns aggregate store statistics
func (h *Handler) GetStoreStats(c *fiber.Ctx) error {
	stats, err := h.db.GetStoreStats(c.Context())
	if err != nil {
		return Error(c, fiber.StatusInternalServerError, "failed to get store stats")
	}

	return Success(c, stats)
}

// SearchStores performs a search on stores
func (h *Handler) SearchStores(c *fiber.Ctx) error {
	query := c.Query("q")
	if query == "" {
		return Error(c, fiber.StatusBadRequest, "search query is required")
	}

	limit := c.QueryInt("limit", 20)
	if limit < 1 || limit > 100 {
		limit = 20
	}

	stores, err := h.db.SearchStores(c.Context(), query, limit)
	if err != nil {
		return Error(c, fiber.StatusInternalServerError, "failed to search stores")
	}

	return Success(c, stores)
}
