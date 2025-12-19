package handlers

import (
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"golang.org/x/crypto/bcrypt"

	"github.com/foxxcyber/price-feed/internal/database"
	"github.com/foxxcyber/price-feed/internal/models"
)

// AdminCreateUser creates a new user (admin only)
func (h *Handler) AdminCreateUser(c *fiber.Ctx) error {
	var req models.AdminCreateUserRequest
	if err := c.BodyParser(&req); err != nil {
		return Error(c, fiber.StatusBadRequest, "invalid request body")
	}

	// Validate required fields
	if req.Email == "" {
		return Error(c, fiber.StatusBadRequest, "email is required")
	}
	if req.Password == "" {
		return Error(c, fiber.StatusBadRequest, "password is required")
	}
	if len(req.Password) < 8 {
		return Error(c, fiber.StatusBadRequest, "password must be at least 8 characters")
	}

	// Validate role
	validRoles := map[models.Role]bool{
		models.RoleUser:      true,
		models.RoleAdmin:     true,
		models.RoleModerator: true,
	}
	if req.Role == "" {
		req.Role = models.RoleUser
	}
	if !validRoles[req.Role] {
		return Error(c, fiber.StatusBadRequest, "invalid role")
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return Error(c, fiber.StatusInternalServerError, "failed to hash password")
	}

	// Create user (no location fields for admin-created users)
	user, err := h.db.CreateUser(c.Context(), req.Email, string(hashedPassword), req.Username, req.RegionID, nil)
	if err != nil {
		if errors.Is(err, database.ErrEmailExists) {
			return Error(c, fiber.StatusConflict, "email already in use")
		}
		if errors.Is(err, database.ErrUsernameExists) {
			return Error(c, fiber.StatusConflict, "username already taken")
		}
		return Error(c, fiber.StatusInternalServerError, "failed to create user")
	}

	// Update role and email_verified if needed
	if req.Role != models.RoleUser || req.EmailVerified {
		updateReq := &models.AdminUpdateUserRequest{
			Role:          &req.Role,
			EmailVerified: &req.EmailVerified,
		}
		user, err = h.db.AdminUpdateUser(c.Context(), user.ID, updateReq)
		if err != nil {
			return Error(c, fiber.StatusInternalServerError, "user created but failed to set role/verified status")
		}
	}

	return Success(c, user)
}

// AdminListUsers returns a paginated list of all users
func (h *Handler) AdminListUsers(c *fiber.Ctx) error {
	// Parse pagination params
	limit := c.QueryInt("limit", 20)
	offset := c.QueryInt("offset", 0)

	// Validate limits
	if limit < 1 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	users, total, err := h.db.ListUsers(c.Context(), limit, offset)
	if err != nil {
		return Error(c, fiber.StatusInternalServerError, "failed to list users")
	}

	return SuccessWithMeta(c, users, total, limit, offset)
}

// AdminGetUser returns a user by ID (admin view with full details)
func (h *Handler) AdminGetUser(c *fiber.Ctx) error {
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

	// Get user stats as well
	stats, _ := h.db.GetUserStats(c.Context(), id)

	return Success(c, fiber.Map{
		"user":  user,
		"stats": stats,
	})
}

// AdminUpdateUser updates a user with admin privileges
func (h *Handler) AdminUpdateUser(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return Error(c, fiber.StatusBadRequest, "invalid user id")
	}

	var req models.AdminUpdateUserRequest
	if err := c.BodyParser(&req); err != nil {
		return Error(c, fiber.StatusBadRequest, "invalid request body")
	}

	// Validate role if provided
	if req.Role != nil {
		validRoles := map[models.Role]bool{
			models.RoleUser:      true,
			models.RoleAdmin:     true,
			models.RoleModerator: true,
		}
		if !validRoles[*req.Role] {
			return Error(c, fiber.StatusBadRequest, "invalid role")
		}
	}

	user, err := h.db.AdminUpdateUser(c.Context(), id, &req)
	if err != nil {
		if errors.Is(err, database.ErrUserNotFound) {
			return Error(c, fiber.StatusNotFound, "user not found")
		}
		if errors.Is(err, database.ErrEmailExists) {
			return Error(c, fiber.StatusConflict, "email already in use")
		}
		if errors.Is(err, database.ErrUsernameExists) {
			return Error(c, fiber.StatusConflict, "username already taken")
		}
		return Error(c, fiber.StatusInternalServerError, "failed to update user")
	}

	return Success(c, user)
}

// AdminDeleteUser deletes a user
func (h *Handler) AdminDeleteUser(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return Error(c, fiber.StatusBadRequest, "invalid user id")
	}

	if err := h.db.DeleteUser(c.Context(), id); err != nil {
		if errors.Is(err, database.ErrUserNotFound) {
			return Error(c, fiber.StatusNotFound, "user not found")
		}
		return Error(c, fiber.StatusInternalServerError, "failed to delete user")
	}

	return c.JSON(fiber.Map{
		"message": "user deleted successfully",
	})
}

// AdminGetStats returns system-wide statistics
func (h *Handler) AdminGetStats(c *fiber.Ctx) error {
	stats, err := h.db.GetAdminStats(c.Context())
	if err != nil {
		return Error(c, fiber.StatusInternalServerError, "failed to get stats")
	}

	return Success(c, stats)
}
