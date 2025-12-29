package handlers

import (
	"github.com/gofiber/fiber/v2"

	"github.com/foxxcyber/price-feed/internal/config"
	"github.com/foxxcyber/price-feed/internal/database"
	"github.com/foxxcyber/price-feed/internal/models"
	"github.com/foxxcyber/price-feed/internal/services"
)

// Handler holds all handler dependencies
type Handler struct {
	db             *database.DB
	cfg            *config.Config
	captchaService *services.CaptchaService
	emailService   *services.EmailService
}

// New creates a new Handler instance
func New(db *database.DB, cfg *config.Config) *Handler {
	return &Handler{
		db:             db,
		cfg:            cfg,
		captchaService: services.NewCaptchaService(db, cfg),
		emailService:   services.NewEmailService(db, cfg),
	}
}

// ErrorHandler is a custom error handler for Fiber
func ErrorHandler(c *fiber.Ctx, err error) error {
	// Default to 500
	code := fiber.StatusInternalServerError
	message := "Internal Server Error"

	// Check if it's a Fiber error
	if e, ok := err.(*fiber.Error); ok {
		code = e.Code
		message = e.Message
	}

	return c.Status(code).JSON(fiber.Map{
		"error": message,
	})
}

// APIResponse is a standard API response structure
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
	Meta    *Meta       `json:"meta,omitempty"`
}

// Meta contains pagination metadata
type Meta struct {
	Total  int `json:"total"`
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

// Success returns a successful response
func Success(c *fiber.Ctx, data interface{}) error {
	return c.JSON(APIResponse{
		Success: true,
		Data:    data,
	})
}

// SuccessWithMeta returns a successful response with pagination
func SuccessWithMeta(c *fiber.Ctx, data interface{}, total, limit, offset int) error {
	return c.JSON(APIResponse{
		Success: true,
		Data:    data,
		Meta: &Meta{
			Total:  total,
			Limit:  limit,
			Offset: offset,
		},
	})
}

// Error returns an error response
func Error(c *fiber.Ctx, status int, message string) error {
	return c.Status(status).JSON(APIResponse{
		Success: false,
		Error:   message,
	})
}

// CreateEmailVerificationChecker creates a function for checking email verification status
// This can be used with the EmailVerifiedRequiredFunc middleware
func (h *Handler) CreateEmailVerificationChecker() func(c *fiber.Ctx) (required bool, verified bool, isAdmin bool, err error) {
	return func(c *fiber.Ctx) (bool, bool, bool, error) {
		userID, ok := c.Locals("user_id").(int)
		if !ok || userID == 0 {
			return false, false, false, nil
		}

		role, _ := c.Locals("user_role").(models.Role)

		// Admins are always exempt
		if role == models.RoleAdmin {
			return false, true, true, nil
		}

		// Check if verification is required from settings
		required := h.db.GetSettingBool(c.Context(), "require_email_verify", false, DeriveEncryptionKey(h.cfg.JWTSecret))

		// If not required, don't need to check further
		if !required {
			return false, true, false, nil
		}

		// Get user to check verification status
		user, err := h.db.GetUserByID(c.Context(), userID)
		if err != nil {
			return true, false, false, err
		}

		return true, user.EmailVerified, false, nil
	}
}
