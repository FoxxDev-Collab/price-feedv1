package handlers

import (
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v2"

	"github.com/foxxcyber/price-feed/internal/database"
	"github.com/foxxcyber/price-feed/internal/models"
)

// ListRegions returns a paginated list of regions
func (h *Handler) ListRegions(c *fiber.Ctx) error {
	params := &models.RegionListParams{
		Limit:  c.QueryInt("limit", 50),
		Offset: c.QueryInt("offset", 0),
		Search: c.Query("search"),
		State:  c.Query("state"),
	}

	// Validate limits
	if params.Limit < 1 || params.Limit > 100 {
		params.Limit = 50
	}
	if params.Offset < 0 {
		params.Offset = 0
	}

	regions, total, err := h.db.ListRegions(c.Context(), params)
	if err != nil {
		return Error(c, fiber.StatusInternalServerError, "failed to list regions")
	}

	return SuccessWithMeta(c, regions, total, params.Limit, params.Offset)
}

// GetRegion returns a single region by ID
func (h *Handler) GetRegion(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return Error(c, fiber.StatusBadRequest, "invalid region id")
	}

	region, err := h.db.GetRegionByID(c.Context(), id)
	if err != nil {
		if errors.Is(err, database.ErrRegionNotFound) {
			return Error(c, fiber.StatusNotFound, "region not found")
		}
		return Error(c, fiber.StatusInternalServerError, "failed to get region")
	}

	return Success(c, region)
}

// CreateRegion creates a new region (admin only)
func (h *Handler) CreateRegion(c *fiber.Ctx) error {
	var req models.CreateRegionRequest
	if err := c.BodyParser(&req); err != nil {
		return Error(c, fiber.StatusBadRequest, "invalid request body")
	}

	// Validate required fields
	if req.Name == "" {
		return Error(c, fiber.StatusBadRequest, "name is required")
	}
	if req.State == "" {
		return Error(c, fiber.StatusBadRequest, "state is required")
	}
	if len(req.State) != 2 {
		return Error(c, fiber.StatusBadRequest, "state must be a 2-letter code")
	}

	// Initialize zip_codes as empty array if nil
	if req.ZipCodes == nil {
		req.ZipCodes = []string{}
	}

	region, err := h.db.CreateRegion(c.Context(), &req)
	if err != nil {
		return Error(c, fiber.StatusInternalServerError, "failed to create region")
	}

	return c.Status(fiber.StatusCreated).JSON(APIResponse{
		Success: true,
		Data:    region,
	})
}

// UpdateRegion updates an existing region (admin only)
func (h *Handler) UpdateRegion(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return Error(c, fiber.StatusBadRequest, "invalid region id")
	}

	var req models.UpdateRegionRequest
	if err := c.BodyParser(&req); err != nil {
		return Error(c, fiber.StatusBadRequest, "invalid request body")
	}

	// Validate state if provided
	if req.State != nil && len(*req.State) != 2 {
		return Error(c, fiber.StatusBadRequest, "state must be a 2-letter code")
	}

	region, err := h.db.UpdateRegion(c.Context(), id, &req)
	if err != nil {
		if errors.Is(err, database.ErrRegionNotFound) {
			return Error(c, fiber.StatusNotFound, "region not found")
		}
		return Error(c, fiber.StatusInternalServerError, "failed to update region")
	}

	return Success(c, region)
}

// DeleteRegion deletes a region (admin only)
func (h *Handler) DeleteRegion(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return Error(c, fiber.StatusBadRequest, "invalid region id")
	}

	if err := h.db.DeleteRegion(c.Context(), id); err != nil {
		if errors.Is(err, database.ErrRegionNotFound) {
			return Error(c, fiber.StatusNotFound, "region not found")
		}
		return Error(c, fiber.StatusInternalServerError, "failed to delete region")
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "region deleted successfully",
	})
}

// GetRegionStates returns list of distinct states
func (h *Handler) GetRegionStates(c *fiber.Ctx) error {
	states, err := h.db.GetDistinctStates(c.Context())
	if err != nil {
		return Error(c, fiber.StatusInternalServerError, "failed to get states")
	}

	return Success(c, states)
}

// GetRegionStats returns aggregate region statistics
func (h *Handler) GetRegionStats(c *fiber.Ctx) error {
	stats, err := h.db.GetRegionStats(c.Context())
	if err != nil {
		return Error(c, fiber.StatusInternalServerError, "failed to get region stats")
	}

	return Success(c, stats)
}

// SearchRegions performs a search on regions
func (h *Handler) SearchRegions(c *fiber.Ctx) error {
	query := c.Query("q")
	if query == "" {
		return Error(c, fiber.StatusBadRequest, "search query is required")
	}

	limit := c.QueryInt("limit", 20)
	if limit < 1 || limit > 100 {
		limit = 20
	}

	regions, err := h.db.SearchRegions(c.Context(), query, limit)
	if err != nil {
		return Error(c, fiber.StatusInternalServerError, "failed to search regions")
	}

	return Success(c, regions)
}
