package handlers

import (
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v2"

	"github.com/foxxcyber/price-feed/internal/database"
	"github.com/foxxcyber/price-feed/internal/middleware"
	"github.com/foxxcyber/price-feed/internal/models"
)

// GetUser returns a user by ID
func (h *Handler) GetUser(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return Error(c, fiber.StatusBadRequest, "invalid user id")
	}

	user, err := h.db.GetUserByID(c.Context(), id)
	if err != nil {
		if errors.Is(err, database.ErrUserNotFound) {
			return Error(c, fiber.StatusNotFound, "user not found")
		}
		return Error(c, fiber.StatusInternalServerError, "failed to get user")
	}

	// Check if requesting own profile or has elevated permissions
	currentUserID := middleware.GetUserID(c)
	currentRole := middleware.GetUserRole(c)

	// If not own profile and not admin/mod, return public info only
	if currentUserID != id && currentRole == models.RoleUser {
		return Success(c, user.ToPublic())
	}

	return Success(c, user)
}

// UpdateUser updates a user's profile
func (h *Handler) UpdateUser(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return Error(c, fiber.StatusBadRequest, "invalid user id")
	}

	// Check authorization - users can only update their own profile
	currentUserID := middleware.GetUserID(c)
	if currentUserID != id {
		return Error(c, fiber.StatusForbidden, "cannot update another user's profile")
	}

	var req models.UpdateUserRequest
	if err := c.BodyParser(&req); err != nil {
		return Error(c, fiber.StatusBadRequest, "invalid request body")
	}

	// Validate username if provided
	if req.Username != nil {
		if len(*req.Username) < 3 || len(*req.Username) > 50 {
			return Error(c, fiber.StatusBadRequest, "username must be between 3 and 50 characters")
		}
	}

	user, err := h.db.UpdateUser(c.Context(), id, &req)
	if err != nil {
		if errors.Is(err, database.ErrUserNotFound) {
			return Error(c, fiber.StatusNotFound, "user not found")
		}
		if errors.Is(err, database.ErrUsernameExists) {
			return Error(c, fiber.StatusConflict, "username already taken")
		}
		return Error(c, fiber.StatusInternalServerError, "failed to update user")
	}

	return Success(c, user)
}

// GetUserStats returns statistics for a user
func (h *Handler) GetUserStats(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return Error(c, fiber.StatusBadRequest, "invalid user id")
	}

	stats, err := h.db.GetUserStats(c.Context(), id)
	if err != nil {
		return Error(c, fiber.StatusInternalServerError, "failed to get user stats")
	}

	return Success(c, stats)
}
