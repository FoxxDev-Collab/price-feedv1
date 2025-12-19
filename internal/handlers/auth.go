package handlers

import (
	"errors"
	"regexp"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/foxxcyber/price-feed/internal/database"
	"github.com/foxxcyber/price-feed/internal/middleware"
	"github.com/foxxcyber/price-feed/internal/models"
)

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

// Register handles user registration
func (h *Handler) Register(c *fiber.Ctx) error {
	var req models.RegisterRequest
	if err := c.BodyParser(&req); err != nil {
		return Error(c, fiber.StatusBadRequest, "invalid request body")
	}

	// Validate email
	if !emailRegex.MatchString(req.Email) {
		return Error(c, fiber.StatusBadRequest, "invalid email format")
	}

	// Validate password
	if len(req.Password) < 8 {
		return Error(c, fiber.StatusBadRequest, "password must be at least 8 characters")
	}

	// Validate username if provided
	if req.Username != nil {
		if len(*req.Username) < 3 || len(*req.Username) > 50 {
			return Error(c, fiber.StatusBadRequest, "username must be between 3 and 50 characters")
		}
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return Error(c, fiber.StatusInternalServerError, "failed to process password")
	}

	// Create user (pass full request to include location fields)
	user, err := h.db.CreateUser(c.Context(), req.Email, string(hashedPassword), req.Username, req.RegionID, &req)
	if err != nil {
		if errors.Is(err, database.ErrEmailExists) {
			return Error(c, fiber.StatusConflict, "email already registered")
		}
		if errors.Is(err, database.ErrUsernameExists) {
			return Error(c, fiber.StatusConflict, "username already taken")
		}
		return Error(c, fiber.StatusInternalServerError, "failed to create user")
	}

	// Generate JWT token
	token, err := h.generateToken(user)
	if err != nil {
		return Error(c, fiber.StatusInternalServerError, "failed to generate token")
	}

	return c.Status(fiber.StatusCreated).JSON(models.AuthResponse{
		Token: token,
		User:  user,
	})
}

// Login handles user authentication
func (h *Handler) Login(c *fiber.Ctx) error {
	var req models.LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return Error(c, fiber.StatusBadRequest, "invalid request body")
	}

	// Validate input
	if req.Email == "" || req.Password == "" {
		return Error(c, fiber.StatusBadRequest, "email and password are required")
	}

	// Get user by email
	user, err := h.db.GetUserByEmail(c.Context(), req.Email)
	if err != nil {
		if errors.Is(err, database.ErrUserNotFound) {
			return Error(c, fiber.StatusUnauthorized, "invalid credentials")
		}
		return Error(c, fiber.StatusInternalServerError, "authentication failed")
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return Error(c, fiber.StatusUnauthorized, "invalid credentials")
	}

	// Update last login
	h.db.UpdateUserLastLogin(c.Context(), user.ID)

	// Generate JWT token
	token, err := h.generateToken(user)
	if err != nil {
		return Error(c, fiber.StatusInternalServerError, "failed to generate token")
	}

	return c.JSON(models.AuthResponse{
		Token: token,
		User:  user,
	})
}

// Logout handles user logout
func (h *Handler) Logout(c *fiber.Ctx) error {
	// In a JWT-based system, logout is typically handled client-side
	// by removing the token. However, we can also invalidate sessions
	// if we're tracking them.

	// For now, just return success
	return c.JSON(fiber.Map{
		"message": "logged out successfully",
	})
}

// GetCurrentUser returns the currently authenticated user
func (h *Handler) GetCurrentUser(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		return Error(c, fiber.StatusUnauthorized, "unauthorized")
	}

	user, err := h.db.GetUserByID(c.Context(), userID)
	if err != nil {
		if errors.Is(err, database.ErrUserNotFound) {
			return Error(c, fiber.StatusNotFound, "user not found")
		}
		return Error(c, fiber.StatusInternalServerError, "failed to get user")
	}

	return Success(c, user)
}

// RefreshToken generates a new JWT token
func (h *Handler) RefreshToken(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		return Error(c, fiber.StatusUnauthorized, "unauthorized")
	}

	user, err := h.db.GetUserByID(c.Context(), userID)
	if err != nil {
		return Error(c, fiber.StatusUnauthorized, "unauthorized")
	}

	token, err := h.generateToken(user)
	if err != nil {
		return Error(c, fiber.StatusInternalServerError, "failed to generate token")
	}

	return c.JSON(fiber.Map{
		"token": token,
	})
}

// generateToken creates a new JWT token for a user
func (h *Handler) generateToken(user *models.User) (string, error) {
	claims := &middleware.JWTClaims{
		UserID: user.ID,
		Email:  user.Email,
		Role:   user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(h.cfg.JWTExpiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   user.Email,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(h.cfg.JWTSecret))
}
